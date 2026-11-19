// Package rpg provides typed access to the RPG Maker VX Ace data files used
// by the server. It deliberately models only the fields the server needs;
// rvdata retains the full document for tools that need to rewrite files.
package rpg

import (
	"math"

	"github.com/carabalonepaulo/vxaos/server/internal/rvdata"
)

// Data is the loaded game database.
type Data struct {
	Actors       []*Actor
	Classes      []*Class
	Skills       []*Skill
	Items        []*Item
	Weapons      []*Weapon
	Armors       []*Armor
	Enemies      []*Enemy
	States       []*State
	Animations   []*Animation
	Tilesets     []*Tileset
	CommonEvents []*CommonEvent
	System       *System
	MapInfos     []*MapInfo
	Maps         []*Map
}

// Feature is one RPG::BaseItem::Feature.
type Feature struct {
	Code   int
	DataID int
	Value  float64
}

// Damage is RPG::UsableItem::Damage.
type Damage struct {
	Type      int
	ElementID int
	Formula   string
	Variance  int
	Critical  bool
}

// Effect is RPG::UsableItem::Effect.
type Effect struct {
	Code   int
	DataID int
	Value1 float64
	Value2 float64
}

// Learning is RPG::Class::Learning.
type Learning struct {
	Level   int
	SkillID int
}

// SpriteRef is one "Graphics/Index" pair from a class note.
type SpriteRef struct {
	Name  string
	Index int
}

// ClassGraphics holds the four note lines: male sprites, female sprites,
// male faces, female faces.
type ClassGraphics struct {
	MaleSprites   []SpriteRef
	FemaleSprites []SpriteRef
	MaleFaces     []SpriteRef
	FemaleFaces   []SpriteRef
}

// Actor is RPG::Actor.
type Actor struct {
	ID             int
	Name           string
	Description    string
	ClassID        int
	InitialLevel   int
	MaxLevel       int
	CharacterName  string
	CharacterIndex int
	FaceName       string
	FaceIndex      int
	Equips         []int
	Features       []Feature
	Note           string
}

// Class is RPG::Class.
type Class struct {
	ID        int
	Name      string
	ExpParams [4]int
	Params    *rvdata.Table
	Learnings []Learning
	Features  []Feature
	Note      string
	Graphics  ClassGraphics
}

// ExpForLevel is the exact RGSS3 RPG::Class#exp_for_level formula.
func (c *Class) ExpForLevel(level int) int {
	if c == nil {
		return 0
	}
	if level < 1 {
		level = 1
	}
	lv := float64(level)
	basis := float64(c.ExpParams[0])
	extra := float64(c.ExpParams[1])
	accA := float64(c.ExpParams[2])
	accB := float64(c.ExpParams[3])
	if accB == 0 {
		accB = 1
	}
	v := basis*math.Pow(lv-1, 0.9+accA/250)*lv*(lv+1)/(6+lv*lv/50/accB) + (lv-1)*extra
	return int(math.Round(v))
}

// ParamAt returns a base parameter for a level. Parameter ids follow VX Ace:
// 0 mhp, 1 mmp, 2 atk, 3 def, 4 mat, 5 mdf, 6 agi, 7 luk.
func (c *Class) ParamAt(paramID, level int) int {
	if c == nil || c.Params == nil {
		return 0
	}
	return int(c.Params.At(int32(paramID), int32(level), 0))
}

// UsableItem contains fields shared by skills and items.
type UsableItem struct {
	ID          int
	Name        string
	Description string
	Note        string
	Scope       int
	Occasion    int
	HitType     int
	AnimationID int
	SuccessRate int
	Repeats     int
	MPCost      int
	TPCost      int
	Damage      Damage
	Effects     []Effect
	Features    []Feature
	Range       int
	AOE         int
	Level       int
	HasAniIndex bool
	AniIndex    int
	Soulbound   bool
}

// Skill is RPG::Skill.
type Skill struct {
	UsableItem
	STypeID          int
	RequiredWTypeID1 int
	RequiredWTypeID2 int
}

// Item is RPG::Item.
type Item struct {
	UsableItem
	ITypeID    int
	Price      int
	Consumable bool
	KeyItem    bool
}

// Weapon is RPG::Weapon.
type Weapon struct {
	ID             int
	Name           string
	Description    string
	Note           string
	ETypeID        int
	WTypeID        int
	AnimationID    int
	Params         [8]int
	Price          int
	Features       []Feature
	Level          int
	VIP            bool
	Soulbound      bool
	HasAniIndex    bool
	AniIndex       int
	PaperdollName  string
	PaperdollIndex int
}

// Armor is RPG::Armor.
type Armor struct {
	ID             int
	Name           string
	Description    string
	Note           string
	ETypeID        int
	ATypeID        int
	Params         [8]int
	Price          int
	Features       []Feature
	Level          int
	VIP            bool
	Soulbound      bool
	Sex            int
	PaperdollName  string
	PaperdollIndex int
}

// EnemyDrop is RPG::Enemy::DropItem.
type EnemyDrop struct {
	Kind        int
	DataID      int
	Denominator int
}

// EnemyAction is RPG::Enemy::Action.
type EnemyAction struct {
	SkillID         int
	ConditionType   int
	ConditionParam1 int
	ConditionParam2 int
	Rating          int
}

// Enemy is RPG::Enemy plus the VXA-OS note fields.
type Enemy struct {
	ID                int
	Name              string
	Description       string
	Params            [8]int
	Exp               int
	Gold              int
	DropItems         []EnemyDrop
	Actions           []EnemyAction
	Features          []Feature
	Note              string
	Sight             int
	ReviveTime        int
	DisableSwitchID   int
	DisableVariableID int
	HasAniIndex       bool
	AniIndex          int
	Escape            bool
	Boss              bool
}

// State is RPG::State.
type State struct {
	ID                  int
	Name                string
	IconIndex           int
	Restriction         int
	Priority            int
	RemoveByRestriction bool
	RemoveByDamage      bool
	ChanceByDamage      int
	AutoRemovalTiming   int
	MinTurns            int
	MaxTurns            int
	StepsToRemove       int
	Features            []Feature
	Note                string
}

// Animation stores the only field the server uses.
type Animation struct {
	ID       int
	Name     string
	FrameMax int
}

// Tileset is RPG::Tileset.
type Tileset struct {
	ID    int
	Name  string
	Mode  int
	Flags *rvdata.Table
	Note  string
}

// CommonEvent is RPG::CommonEvent.
type CommonEvent struct {
	ID       int
	Name     string
	Trigger  int
	SwitchID int
	List     []*EventCommand
}

// System is the subset of RPG::System used by the server.
type System struct {
	GameTitle     string
	StartMapID    int
	StartX        int
	StartY        int
	OptFloorDeath bool
	CurrencyUnit  string
}

// MapInfo is RPG::MapInfo.
type MapInfo struct {
	ID       int
	Name     string
	ParentID int
	Order    int
}

// Map is RPG::Map with server-relevant derived fields.
type Map struct {
	ID        int
	Name      string
	Width     int
	Height    int
	TilesetID int
	Note      string
	Data      *rvdata.Table
	Events    []*Event
	PvP       bool
}

// Event is RPG::Event.
type Event struct {
	ID    int
	Name  string
	X     int
	Y     int
	Pages []*Page
}

// Page is RPG::Event::Page.
type Page struct {
	Condition     Condition
	Graphic       Graphic
	MoveType      int
	MoveSpeed     int
	MoveFrequency int
	WalkAnime     bool
	StepAnime     bool
	DirectionFix  bool
	Through       bool
	PriorityType  int
	Trigger       int
	MoveRoute     *MoveRoute
	List          []*EventCommand
}

// Condition is RPG::Event::Page::Condition.
type Condition struct {
	Switch1Valid    bool
	Switch1ID       int
	Switch2Valid    bool
	Switch2ID       int
	VariableValid   bool
	VariableID      int
	VariableValue   int
	SelfSwitchValid bool
	SelfSwitchCh    string
	ItemValid       bool
	ItemID          int
	ActorValid      bool
	ActorID         int
}

// Graphic is RPG::Event::Page::Graphic.
type Graphic struct {
	TileID         int
	CharacterName  string
	CharacterIndex int
	Direction      int
	Pattern        int
}

// MoveRoute is RPG::MoveRoute.
type MoveRoute struct {
	Repeat    bool
	Skippable bool
	Wait      bool
	List      []MoveCommand
}

// MoveCommand is RPG::MoveCommand.
type MoveCommand struct {
	Code       int
	Parameters []any
}

// EventCommand is RPG::EventCommand.
type EventCommand struct {
	Code       int
	Indent     int
	Parameters []any
}
