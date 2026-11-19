// Package chargen creates a new character from the RPG Maker database.
package chargen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/carabalonepaulo/vxaos/server/internal/rpg"
	"github.com/carabalonepaulo/vxaos/server/internal/store"
)

// Options configures character creation.
type Options struct {
	Data               *rpg.Data
	MaxEquips          int
	MaxHotbar          int
	MaxPlayerSwitches  int
	MaxPlayerVariables int
	StartPoints        int
	MaxActors          int
	MinCharacters      int
	MaxCharacters      int
	MaxDefaultClasses  int
	MaxVIPClasses      int
}

// Create builds a character and its initial aggregate. It does not touch the
// database; the caller persists the result. allocated contains the eight
// creation points (HP/MP points are worth 10).
func Create(opts Options, accountID int64, slot int, name string, classID, sex, characterIndex int, allocated [8]int) (*store.Character, *store.CharacterState, error) {
	if opts.Data == nil {
		return nil, nil, fmt.Errorf("chargen: nil data")
	}
	name = titleize(name)
	if len(name) < opts.MinCharacters || len(name) > opts.MaxCharacters {
		return nil, nil, fmt.Errorf("chargen: name length out of range")
	}
	if !validName(name) {
		return nil, nil, fmt.Errorf("chargen: invalid name characters")
	}
	if classID <= 0 || classID >= len(opts.Data.Classes) || opts.Data.Classes[classID] == nil {
		return nil, nil, fmt.Errorf("chargen: class %d does not exist", classID)
	}
	if classID >= len(opts.Data.Actors) || opts.Data.Actors[classID] == nil {
		return nil, nil, fmt.Errorf("chargen: actor class %d does not exist", classID)
	}
	if sex != 0 && sex != 1 {
		return nil, nil, fmt.Errorf("chargen: invalid sex %d", sex)
	}
	if opts.MaxEquips <= 0 {
		opts.MaxEquips = 9
	}
	if opts.MaxHotbar <= 0 {
		opts.MaxHotbar = 9
	}
	if opts.MaxActors > 0 && (slot < 0 || slot >= opts.MaxActors) {
		return nil, nil, fmt.Errorf("chargen: slot %d out of range", slot)
	}
	totalPoints := 0
	for _, p := range allocated {
		if p < 0 || p > 255 {
			return nil, nil, fmt.Errorf("chargen: allocation out of range")
		}
		totalPoints += p
	}
	if totalPoints > opts.StartPoints {
		return nil, nil, fmt.Errorf("chargen: allocation exceeds %d points", opts.StartPoints)
	}
	actor := opts.Data.Actors[classID]
	class := opts.Data.Classes[classID]
	level := actor.InitialLevel
	if level < 1 {
		level = 1
	}
	params := [8]int{}
	for i := 0; i < 8; i++ {
		params[i] = class.ParamAt(i, level)
	}
	params[0] += allocated[0] * 10
	params[1] += allocated[1] * 10
	for i := 2; i < 8; i++ {
		params[i] += allocated[i]
	}
	characterName, charIndex, faceName, faceIndex := graphicFor(class, actor, sex, characterIndex)

	equips := make([]int, opts.MaxEquips)
	for i := 0; i < len(equips) && i < len(actor.Equips); i++ {
		equips[i] = actor.Equips[i]
	}
	skills := make([]int, 0, len(class.Learnings))
	seen := make(map[int]bool)
	for _, l := range class.Learnings {
		if l.Level <= level && !seen[l.SkillID] {
			seen[l.SkillID] = true
			skills = append(skills, l.SkillID)
		}
	}
	sort.Ints(skills)
	hotbar := make([]store.HotbarSlot, opts.MaxHotbar)
	switches := make([]bool, opts.MaxPlayerSwitches)
	variables := make([]int32, opts.MaxPlayerVariables)
	startMap := opts.Data.System.StartMapID
	startX := opts.Data.System.StartX
	startY := opts.Data.System.StartY
	points := opts.StartPoints - totalPoints
	c := &store.Character{
		AccountID:      accountID,
		Slot:           slot,
		Name:           name,
		NameKey:        store.NameKey(name),
		ClassID:        classID,
		Sex:            sex,
		Level:          level,
		Exp:            class.ExpForLevel(level),
		HP:             params[0],
		MP:             params[1],
		Points:         points,
		MHP:            params[0],
		MMP:            params[1],
		ATK:            params[2],
		DEF:            params[3],
		MAT:            params[4],
		MDF:            params[5],
		AGI:            params[6],
		LUK:            params[7],
		CharacterName:  characterName,
		CharacterIndex: charIndex,
		FaceName:       faceName,
		FaceIndex:      faceIndex,
		MapID:          startMap,
		X:              startX,
		Y:              startY,
		Direction:      2,
		ReviveMapID:    startMap,
		ReviveX:        startX,
		ReviveY:        startY,
		Gold:           0,
	}
	st := &store.CharacterState{
		Version:   store.StateVersion,
		Equips:    equips,
		Skills:    skills,
		Hotbar:    hotbar,
		Switches:  switches,
		Variables: variables,
	}
	return c, st, nil
}

func titleize(name string) string {
	words := strings.Fields(strings.TrimSpace(name))
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
	}
	return strings.Join(words, " ")
}

func validName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == ' ' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			continue
		}
		return false
	}
	return true
}

func graphicFor(class *rpg.Class, actor *rpg.Actor, sex, characterIndex int) (string, int, string, int) {
	sprites := class.Graphics.MaleSprites
	faces := class.Graphics.MaleFaces
	if sex == 1 {
		sprites = class.Graphics.FemaleSprites
		faces = class.Graphics.FemaleFaces
	}
	name, index := actor.CharacterName, actor.CharacterIndex
	faceName, faceIndex := actor.FaceName, actor.FaceIndex
	if characterIndex >= 0 && characterIndex < len(sprites) {
		name, index = sprites[characterIndex].Name, sprites[characterIndex].Index
	}
	if characterIndex >= 0 && characterIndex < len(faces) {
		faceName, faceIndex = faces[characterIndex].Name, faces[characterIndex].Index
	}
	return name, index, faceName, faceIndex
}
