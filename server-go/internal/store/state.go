package store

import (
	"encoding/json"
	"fmt"
	"strings"
)

// StateVersion is the current JSON aggregate version.
const StateVersion = 1

// Account is one row of accounts.
type Account struct {
	ID                 int64
	Username           string
	UsernameKey        string
	Email              string
	PasswordAlgo       string
	PasswordSalt       []byte
	PasswordIterations int
	PasswordVerifier   []byte
	GroupID            int
	VIPUntil           int64
	CreatedAt          int64
	LastLoginAt        int64
}

// Character is one row of characters plus its aggregate state.
type Character struct {
	ID             int64
	AccountID      int64
	Slot           int
	Name           string
	NameKey        string
	ClassID        int
	Sex            int
	Level          int
	Exp            int
	HP             int
	MP             int
	Points         int
	MHP            int
	MMP            int
	ATK            int
	DEF            int
	MAT            int
	MDF            int
	AGI            int
	LUK            int
	CharacterName  string
	CharacterIndex int
	FaceName       string
	FaceIndex      int
	MapID          int
	X              int
	Y              int
	Direction      int
	ReviveMapID    int
	ReviveX        int
	ReviveY        int
	Gold           int64
	GuildID        int64
	CreatedAt      int64
	LastLoginAt    int64
}

// ItemStack is one inventory/bank entry. Kind is 1 item, 2 weapon, 3 armor.
type ItemStack struct {
	Kind   int `json:"kind"`
	ItemID int `json:"item_id"`
	Amount int `json:"amount"`
}

// QuestState is one quest entry.
type QuestState struct {
	QuestID int `json:"quest_id"`
	State   int `json:"state"`
	Kills   int `json:"kills"`
}

// HotbarSlot is one hotbar entry.
type HotbarSlot struct {
	Type   int `json:"type"`
	ItemID int `json:"item_id"`
}

// SelfSwitchState is one self switch.
type SelfSwitchState struct {
	MapID   int    `json:"map_id"`
	EventID int    `json:"event_id"`
	Ch      string `json:"ch"`
	Value   bool   `json:"value"`
}

// CharacterState is the aggregate stored in character_state.data.
type CharacterState struct {
	Version      int               `json:"version"`
	Equips       []int             `json:"equips,omitempty"`
	Inventory    []ItemStack       `json:"inventory,omitempty"`
	Skills       []int             `json:"skills,omitempty"`
	Quests       []QuestState      `json:"quests,omitempty"`
	Hotbar       []HotbarSlot      `json:"hotbar,omitempty"`
	Switches     []bool            `json:"switches,omitempty"`
	Variables    []int32           `json:"variables,omitempty"`
	SelfSwitches []SelfSwitchState `json:"self_switches,omitempty"`
}

// BankState is the aggregate stored in bank_state.data.
type BankState struct {
	Version int         `json:"version"`
	Gold    int64       `json:"gold"`
	Items   []ItemStack `json:"items,omitempty"`
}

// DefaultCharacterState returns an empty versioned aggregate.
func DefaultCharacterState() CharacterState {
	return CharacterState{Version: StateVersion}
}

// DefaultBankState returns an empty versioned aggregate.
func DefaultBankState() BankState {
	return BankState{Version: StateVersion}
}

// EncodeState marshals a character aggregate.
func EncodeState(st CharacterState) (string, error) {
	if st.Version == 0 {
		st.Version = StateVersion
	}
	b, err := json.Marshal(st)
	if err != nil {
		return "", fmt.Errorf("store: encode character state: %w", err)
	}
	return string(b), nil
}

// DecodeState unmarshals a character aggregate. An empty payload yields the
// default state.
func DecodeState(data string) (CharacterState, error) {
	if strings.TrimSpace(data) == "" {
		return DefaultCharacterState(), nil
	}
	var st CharacterState
	if err := json.Unmarshal([]byte(data), &st); err != nil {
		return CharacterState{}, fmt.Errorf("store: decode character state: %w", err)
	}
	if st.Version == 0 {
		st.Version = StateVersion
	}
	if st.Version > StateVersion {
		return CharacterState{}, fmt.Errorf("store: unsupported character state version %d", st.Version)
	}
	return st, nil
}

// EncodeBankState marshals a bank aggregate.
func EncodeBankState(st BankState) (string, error) {
	if st.Version == 0 {
		st.Version = StateVersion
	}
	b, err := json.Marshal(st)
	if err != nil {
		return "", fmt.Errorf("store: encode bank state: %w", err)
	}
	return string(b), nil
}

// DecodeBankState unmarshals a bank aggregate.
func DecodeBankState(data string) (BankState, error) {
	if strings.TrimSpace(data) == "" {
		return DefaultBankState(), nil
	}
	var st BankState
	if err := json.Unmarshal([]byte(data), &st); err != nil {
		return BankState{}, fmt.Errorf("store: decode bank state: %w", err)
	}
	if st.Version == 0 {
		st.Version = StateVersion
	}
	if st.Version > StateVersion {
		return BankState{}, fmt.Errorf("store: unsupported bank state version %d", st.Version)
	}
	return st, nil
}

// UsernameKey normalizes a username for case-insensitive lookup.
func UsernameKey(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// NameKey normalizes a character/guild name for case-insensitive uniqueness.
func NameKey(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
