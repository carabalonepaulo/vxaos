package game

import (
	"fmt"

	"github.com/carabalonepaulo/vxaos/server/internal/content"
	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
	"github.com/carabalonepaulo/vxaos/server/internal/store"
)

// Quest states match the client's Enums::Quest.
const (
	QuestInProgress = 0
	QuestFinished   = 1
)

func (w *World) questDef(id int) *content.Quest {
	if id < 0 || id >= len(w.opts.Quests) {
		return nil
	}
	return &w.opts.Quests[id]
}

func questIndex(p *Player, id int) int {
	for i := range p.State.Quests {
		if p.State.Quests[i].QuestID == id {
			return i
		}
	}
	return -1
}

// startQuest adds a quest in progress. Existing entries (including finished
// non-repeatable quests) are left alone, matching the original server.
func (w *World) startQuest(p *Player, id int) {
	questID := id - 1
	if w.questDef(questID) == nil || questIndex(p, questID) >= 0 {
		return
	}
	p.State.Quests = append(p.State.Quests, store.QuestState{QuestID: questID, State: QuestInProgress})
	_ = p.Client.Send(protocol.ServerQuestAdd, protocol.QuestAdd{QuestID: uint8(clampInt(questID, 0, 255))})
	w.saveQuietly(p)
}

// finishQuest validates the turn-in requirements, consumes the required item
// and grants the rewards before marking the quest finished.
func (w *World) finishQuest(p *Player, id int) {
	questID := id - 1
	index := questIndex(p, questID)
	def := w.questDef(questID)
	if def == nil || index < 0 || p.State.Quests[index].State != QuestInProgress {
		return
	}
	quest := &p.State.Quests[index]
	if !w.questRequirementsMet(p, quest, def) {
		return
	}
	if def.ItemID > 0 && def.ItemAmount > 0 {
		if !removeInventoryItem(p.State, def.ItemKind, def.ItemID, def.ItemAmount) {
			return
		}
		w.sendItemTo(p, def.ItemKind, def.ItemID, -def.ItemAmount, false, false)
	}
	if def.RewGold > 0 {
		p.Character.Gold = clampInt64(p.Character.Gold+int64(def.RewGold), 0, w.opts.MaxGold)
		_ = p.Client.Send(protocol.ServerGold, protocol.Gold{Amount: int64(def.RewGold), Popup: true})
	}
	if def.RewEXP > 0 {
		w.gainExp(p, def.RewEXP)
	}
	if def.RewItemID > 0 && def.RewItemAmount > 0 {
		if addInventoryItem(p.State, w.opts.MaxItems, def.RewItemKind, def.RewItemID, def.RewItemAmount) == nil {
			w.sendItemTo(p, def.RewItemKind, def.RewItemID, def.RewItemAmount, false, true)
		}
	}
	quest.State = QuestFinished
	if def.Repeat {
		p.State.Quests = append(p.State.Quests[:index], p.State.Quests[index+1:]...)
	}
	_ = p.Client.Send(protocol.ServerQuestFinish, protocol.QuestFinish{QuestID: uint8(clampInt(questID, 0, 255))})
	w.saveQuietly(p)
}

func (w *World) questRequirementsMet(p *Player, quest *store.QuestState, def *content.Quest) bool {
	if def.SwitchID > 0 && !w.switchValue(p, def.SwitchID) {
		return false
	}
	if def.VariableID > 0 && w.variableValue(p, def.VariableID) < def.VariableAmount {
		return false
	}
	if def.ItemID > 0 && def.ItemAmount > 0 {
		index := findInventory(p.State, def.ItemKind, def.ItemID)
		if index < 0 || p.State.Inventory[index].Amount < def.ItemAmount {
			return false
		}
	}
	return quest.Kills >= def.EnemyAmount
}

// addQuestKills advances the first in-progress quest that tracks enemyID.
func (w *World) addQuestKills(p *Player, enemyID int) {
	if enemyID <= 0 {
		return
	}
	name := ""
	if enemyID < len(w.opts.Data.Enemies) && w.opts.Data.Enemies[enemyID] != nil {
		name = w.opts.Data.Enemies[enemyID].Name
	}
	for i := range p.State.Quests {
		quest := &p.State.Quests[i]
		if quest.State != QuestInProgress {
			continue
		}
		def := w.questDef(quest.QuestID)
		if def == nil || def.EnemyID != enemyID || def.EnemyAmount <= 0 {
			continue
		}
		if quest.Kills >= def.EnemyAmount {
			break
		}
		quest.Kills++
		w.questChat(p, fmt.Sprintf("Matou %d/%d %s.", quest.Kills, def.EnemyAmount, name))
		break
	}
}

// addQuestItemNotice reports progress for quests that require an item.
func (w *World) addQuestItemNotice(p *Player, kind, id int) {
	for i := range p.State.Quests {
		quest := &p.State.Quests[i]
		if quest.State != QuestInProgress {
			continue
		}
		def := w.questDef(quest.QuestID)
		if def == nil || def.ItemID != id || def.ItemKind != kind || def.ItemAmount <= 0 {
			continue
		}
		index := findInventory(p.State, kind, id)
		count := 0
		if index >= 0 {
			count = p.State.Inventory[index].Amount
		}
		if count > def.ItemAmount {
			break
		}
		w.questChat(p, fmt.Sprintf("Você tem %d/%d %s.", count, def.ItemAmount, w.itemName(kind, id)))
		break
	}
}

func (w *World) questChat(p *Player, message string) {
	_ = p.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: 5, Message: message})
}

func (w *World) itemName(kind, id int) string {
	switch kind {
	case KindItem:
		if item := w.item(id); item != nil {
			return item.Name
		}
	case KindWeapon:
		if weapon := w.weapon(id); weapon != nil {
			return weapon.Name
		}
	case KindArmor:
		if armor := w.armor(id); armor != nil {
			return armor.Name
		}
	}
	return ""
}
