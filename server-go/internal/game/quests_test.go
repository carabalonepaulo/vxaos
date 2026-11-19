package game

import (
	"context"
	"testing"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/store"
)

func TestQuestScriptConditions(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Conditioner", 21, 12)
	join := func() {
		if err := w.Join(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	join()
	if p.Character.Sex == 0 {
		if !w.evalScriptCondition(p, "@client.sex == Enums::Sex::MALE") {
			t.Fatal("male sex condition failed")
		}
		if w.evalScriptCondition(p, "@client.sex == Enums::Sex::FEMALE") {
			t.Fatal("female sex condition matched a male player")
		}
	}
	if !w.evalScriptCondition(p, "@client.original_character_name.empty?") {
		t.Fatal("original graphic should start empty")
	}
	p.OriginalCharacterName = "Charset02"
	if w.evalScriptCondition(p, "@client.original_character_name.empty?") {
		t.Fatal("saved original graphic still considered empty")
	}
	p.State.Inventory = []store.ItemStack{{Kind: KindItem, ItemID: 23, Amount: 10}}
	if !w.evalScriptCondition(p, "@client.item_number($data_items[23]) >= 10") {
		t.Fatal("item_number condition failed")
	}
	if w.evalScriptCondition(p, "@client.item_number($data_items[23]) > 10") {
		t.Fatal("item_number comparison ignored the count")
	}

	first := w.opts.Quests[0]
	if err := w.do(ctx, func() error {
		w.startQuest(p, 1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if w.evalScriptCondition(p, "@client.finished_quest_requirements?(1)") {
		t.Fatal("requirements matched before the item was obtained")
	}
	if first.ItemID > 0 {
		p.State.Inventory = append(p.State.Inventory, store.ItemStack{
			Kind: first.ItemKind, ItemID: first.ItemID, Amount: first.ItemAmount,
		})
	}
	if !w.evalScriptCondition(p, "@client.finished_quest_requirements?(1)") {
		t.Fatalf("requirements did not match: quest=%+v def=%+v", p.State.Quests, first)
	}
	if w.evalScriptCondition(p, "@client.quest_finished?(1)") {
		t.Fatal("quest reported finished before turn-in")
	}
}

func TestQuestFlow(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Quester", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	if len(w.opts.Quests) < 2 {
		t.Fatalf("quest content = %d entries", len(w.opts.Quests))
	}
	first := w.opts.Quests[0]
	second := w.opts.Quests[1]

	// Start quest 0. Starting it twice must not duplicate the entry.
	if err := w.do(ctx, func() error {
		w.startQuest(p, 1)
		w.startQuest(p, 1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(p.State.Quests) != 1 || p.State.Quests[0].State != QuestInProgress {
		t.Fatalf("quests after start = %+v", p.State.Quests)
	}
	client := p.Client.(*fakeClient)
	if len(client.find(protocol.ServerQuestAdd)) != 1 {
		t.Fatalf("quest add messages = %d", len(client.find(protocol.ServerQuestAdd)))
	}

	// Turning in without the required item does nothing.
	if err := w.do(ctx, func() error {
		w.finishQuest(p, 1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if p.State.Quests[0].State != QuestInProgress {
		t.Fatal("quest finished without requirements")
	}

	if first.ItemID > 0 && first.ItemAmount > 0 {
		if err := w.do(ctx, func() error {
			if err := addInventoryItem(p.State, w.opts.MaxItems, first.ItemKind, first.ItemID, first.ItemAmount); err != nil {
				return err
			}
			w.finishQuest(p, 1)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if p.State.Quests[0].State != QuestFinished {
			t.Fatalf("quest 0 state = %d", p.State.Quests[0].State)
		}
		if findInventory(p.State, first.ItemKind, first.ItemID) >= 0 {
			t.Fatal("required item was not consumed")
		}
		if len(client.find(protocol.ServerQuestFinish)) != 1 {
			t.Fatal("no QuestFinish sent")
		}
	}

	// Quest 1 is repeatable and tracks kills of its enemy.
	if err := w.do(ctx, func() error {
		w.startQuest(p, 2)
		for i := 0; i < second.EnemyAmount; i++ {
			w.addQuestKills(p, second.EnemyID)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	index := questIndex(p, 1)
	if index < 0 || p.State.Quests[index].Kills != second.EnemyAmount {
		t.Fatalf("quest 1 kills = %+v", p.State.Quests)
	}
	goldBefore := p.Character.Gold
	if err := w.do(ctx, func() error {
		w.finishQuest(p, 2)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if questIndex(p, 1) >= 0 {
		t.Fatal("repeatable quest was not removed after finishing")
	}
	if p.Character.Gold != goldBefore+int64(second.RewGold) {
		t.Fatalf("gold = %d want %d", p.Character.Gold, goldBefore+int64(second.RewGold))
	}
	if second.RewItemID > 0 && findInventory(p.State, second.RewItemKind, second.RewItemID) < 0 {
		t.Fatal("quest reward item was not granted")
	}
}
