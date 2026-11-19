package content

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"

	"github.com/BurntSushi/toml"
)

// ConfigsSchemaVersion is the supported schema version of configs.toml.
const ConfigsSchemaVersion = 1

var rubySymbolRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// FontConfig is one Font.default_* group.
type FontConfig struct {
	Name    string `toml:"name"`
	Outline bool   `toml:"outline"`
	Shadow  bool   `toml:"shadow"`
	Bold    bool   `toml:"bold"`
	Size    int    `toml:"size"`
}

// FontsConfig groups the default and chat fonts.
type FontsConfig struct {
	Default FontConfig `toml:"default"`
	Chat    FontConfig `toml:"chat"`
}

// NetworkConfig contains public network settings.
type NetworkConfig struct {
	Host        string `toml:"host"`
	Port        int    `toml:"port"`
	GameVersion int    `toml:"game_version"`
	ShopWebsite string `toml:"shop_website"`
}

// Resolution is one supported window size.
type Resolution struct {
	Width  int `toml:"width"`
	Height int `toml:"height"`
}

// LimitsConfig groups numeric limits used by the client.
type LimitsConfig struct {
	MinCharacters      int   `toml:"min_characters"`
	MaxCharacters      int   `toml:"max_characters"`
	MinLevelPvP        int   `toml:"min_level_pvp"`
	MaxActors          int   `toml:"max_actors"`
	MaxHotbar          int   `toml:"max_hotbar"`
	MaxDefaultClasses  int   `toml:"max_default_classes"`
	MaxVIPClasses      int   `toml:"max_vip_classes"`
	MaxItems           int   `toml:"max_items"`
	MaxGold            int64 `toml:"max_gold"`
	MaxParams          int   `toml:"max_params"`
	MaxLevel           int   `toml:"max_level"`
	MaxPlayerSwitches  int   `toml:"max_player_switches"`
	MaxPlayerVariables int   `toml:"max_player_variables"`
	MaxFriends         int   `toml:"max_friends"`
	MaxGuildMembers    int   `toml:"max_guild_members"`
	MaxPlayerItems     int   `toml:"max_player_items"`
	MaxTradeItems      int   `toml:"max_trade_items"`
	MaxBankItems       int   `toml:"max_bank_items"`
	MaxChatLines       int   `toml:"max_chat_lines"`
	MaxPartyMembers    int   `toml:"max_party_members"`
	MaxEquips          int   `toml:"max_equips"`
	MaxMapDrops        int   `toml:"max_map_drops"`
}

// PaperdollConfig groups equipment draw order per direction.
type PaperdollConfig struct {
	Down  []int `toml:"down"`
	Left  []int `toml:"left"`
	Right []int `toml:"right"`
	Up    []int `toml:"up"`
}

// ChatColors is the chat color table.
type ChatColors struct {
	Normal  int `toml:"normal"`
	Global  int `toml:"global"`
	Success int `toml:"success"`
	Error   int `toml:"error"`
	Alert   int `toml:"alert"`
	AdmMsg  int `toml:"adm_msg"`
}

// NameColors is the player/enemy/guild name color table.
type NameColors struct {
	Default int `toml:"default"`
	Monitor int `toml:"monitor"`
	Admin   int `toml:"admin"`
	Enemy   int `toml:"enemy"`
	Boss    int `toml:"boss"`
	Guild   int `toml:"guild"`
}

// ColorsConfig groups both color tables.
type ColorsConfig struct {
	Chat ChatColors `toml:"chat"`
	Name NameColors `toml:"name"`
}

// MenuIcons is the main menu icon table.
type MenuIcons struct {
	Item   int `toml:"item"`
	Skill  int `toml:"skill"`
	Status int `toml:"status"`
	Quest  int `toml:"quest"`
	Friend int `toml:"friend"`
	Guild  int `toml:"guild"`
	Menu   int `toml:"menu"`
}

// InteractionIcons is the interaction menu icon table.
type InteractionIcons struct {
	Private int `toml:"private"`
	Block   int `toml:"block"`
	Unlock  int `toml:"unlock"`
	Trade   int `toml:"trade"`
	Party   int `toml:"party"`
}

// MiscIcons is the remaining icon table.
type MiscIcons struct {
	Config          int `toml:"config"`
	Gold            int `toml:"gold"`
	Exp             int `toml:"exp"`
	AddGold         int `toml:"add_gold"`
	RemoveGold      int `toml:"remove_gold"`
	MapPvP          int `toml:"map_pvp"`
	MapNonPvP       int `toml:"map_nonpvp"`
	PlayerOn        int `toml:"player_on"`
	PlayerOff       int `toml:"player_off"`
	QuestNotStarted int `toml:"quest_not_started"`
	QuestFinished   int `toml:"quest_finished"`
	QuestInProgress int `toml:"quest_in_progress"`
	LeaveParty      int `toml:"leave_party"`
	Emoji           int `toml:"emoji"`
}

// IconsConfig groups menu, interaction, misc and buff icon ranges.
type IconsConfig struct {
	Menu        MenuIcons        `toml:"menu"`
	Interaction InteractionIcons `toml:"interaction"`
	Misc        MiscIcons        `toml:"misc"`
	BuffStart   int              `toml:"buff_start"`
	DebuffStart int              `toml:"debuff_start"`
}

// TeleportStop is one destination in a teleport group.
type TeleportStop struct {
	MapID int `toml:"map_id"`
	X     int `toml:"x"`
	Y     int `toml:"y"`
	Gold  int `toml:"gold"`
}

// RangeWeapon is one long-range weapon projectile definition.
type RangeWeapon struct {
	ProjectileName string `toml:"projectile_name"`
	Range          int    `toml:"range"`
	ItemID         int    `toml:"item_id"`
	StepAnime      bool   `toml:"step_anime"`
	MPCost         int    `toml:"mp_cost"`
}

// RangeSkill is one long-range skill projectile definition.
type RangeSkill struct {
	ProjectileName string `toml:"projectile_name"`
	StepAnime      bool   `toml:"step_anime"`
}

// TimingConfig groups attack/cooldown timings.
type TimingConfig struct {
	AttackTime          float64 `toml:"attack_time"`
	CooldownSkillTime   float64 `toml:"cooldown_skill_time"`
	GlobalAntispamTime  float64 `toml:"global_antispam_time"`
	AttackAnimationTime int     `toml:"attack_animation_time"`
	LevelUpAnimationID  int     `toml:"level_up_animation_id"`
}

// KeysConfig groups key bindings and hotkey lists.
type KeysConfig struct {
	Attack      string   `toml:"attack"`
	GetDrop     string   `toml:"get_drop"`
	SelectEnemy string   `toml:"select_enemy"`
	Item        string   `toml:"item"`
	Skill       string   `toml:"skill"`
	Status      string   `toml:"status"`
	Quest       string   `toml:"quest"`
	Friend      string   `toml:"friend"`
	Guild       string   `toml:"guild"`
	Menu        string   `toml:"menu"`
	Hotkeys     []string `toml:"hotkeys"`
	Balloons    []string `toml:"balloons"`
}

// ConfigsFile is the decoded content/configs.toml.
type ConfigsFile struct {
	SchemaVersion      int                    `toml:"schema_version"`
	Fonts              FontsConfig            `toml:"fonts"`
	Network            NetworkConfig          `toml:"network"`
	Resolutions        []Resolution           `toml:"resolution"`
	LoadingTime        int                    `toml:"loading_time"`
	LoadingTitles      []string               `toml:"loading_titles"`
	Limits             LimitsConfig           `toml:"limits"`
	Paperdoll          PaperdollConfig        `toml:"paperdoll"`
	Colors             ColorsConfig           `toml:"colors"`
	Icons              IconsConfig            `toml:"icons"`
	StartPoints        int                    `toml:"start_points"`
	TitleBarHeight     int                    `toml:"title_bar_height"`
	DropDespawnSeconds int                    `toml:"drop_despawn_seconds"`
	DropPickupSeconds  int                    `toml:"drop_pickup_seconds"`
	Teleports          [][]TeleportStop       `toml:"teleports"`
	RangeWeapons       map[string]RangeWeapon `toml:"range_weapons"`
	RangeSkills        map[string]RangeSkill  `toml:"range_skills"`
	ForbiddenNames     []string               `toml:"forbidden_names"`
	Timing             TimingConfig           `toml:"timing"`
	Keys               KeysConfig             `toml:"keys"`
}

// LoadConfigs reads and validates content/configs.toml.
func LoadConfigs(path string) (*ConfigsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file ConfigsFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("content: %s: %w", path, err)
	}
	if err := file.Validate(); err != nil {
		return nil, fmt.Errorf("content: %s: %w", path, err)
	}
	return &file, nil
}

// Validate checks the values used by the Ruby generator.
func (f *ConfigsFile) Validate() error {
	if f.SchemaVersion != ConfigsSchemaVersion {
		return fmt.Errorf("unsupported configs schema version %d (want %d)", f.SchemaVersion, ConfigsSchemaVersion)
	}
	if f.Network.Port < 1 || f.Network.Port > 65535 {
		return fmt.Errorf("network.port %d out of range", f.Network.Port)
	}
	if len(f.Resolutions) == 0 {
		return fmt.Errorf("at least one resolution is required")
	}
	if f.DropDespawnSeconds <= 0 || f.DropPickupSeconds <= 0 {
		return fmt.Errorf("drop timers must be positive")
	}
	for _, res := range f.Resolutions {
		if res.Width <= 0 || res.Height <= 0 {
			return fmt.Errorf("invalid resolution %dx%d", res.Width, res.Height)
		}
	}
	if f.Limits.MaxEquips <= 0 {
		return fmt.Errorf("limits.max_equips must be positive")
	}
	for name, list := range map[string][]int{
		"down": f.Paperdoll.Down, "left": f.Paperdoll.Left,
		"right": f.Paperdoll.Right, "up": f.Paperdoll.Up,
	} {
		if len(list) != f.Limits.MaxEquips {
			return fmt.Errorf("paperdoll.%s has %d entries, max_equips is %d", name, len(list), f.Limits.MaxEquips)
		}
	}
	if len(f.Keys.Hotkeys) != f.Limits.MaxHotbar {
		return fmt.Errorf("keys.hotkeys has %d entries, max_hotbar is %d", len(f.Keys.Hotkeys), f.Limits.MaxHotbar)
	}
	for name, key := range map[string]string{
		"attack": f.Keys.Attack, "get_drop": f.Keys.GetDrop, "select_enemy": f.Keys.SelectEnemy,
		"item": f.Keys.Item, "skill": f.Keys.Skill, "status": f.Keys.Status, "quest": f.Keys.Quest,
		"friend": f.Keys.Friend, "guild": f.Keys.Guild, "menu": f.Keys.Menu,
	} {
		if !rubySymbolRE.MatchString(key) {
			return fmt.Errorf("keys.%s has invalid key %q", name, key)
		}
	}
	for _, key := range append(append([]string{}, f.Keys.Hotkeys...), f.Keys.Balloons...) {
		if !rubySymbolRE.MatchString(key) {
			return fmt.Errorf("invalid key name %q", key)
		}
	}
	for id, weapon := range f.RangeWeapons {
		if _, err := strconv.Atoi(id); err != nil {
			return fmt.Errorf("range_weapons key %q is not a numeric id", id)
		}
		if weapon.ProjectileName == "" || weapon.Range < 1 {
			return fmt.Errorf("range_weapons.%s is incomplete", id)
		}
	}
	for id, skill := range f.RangeSkills {
		if _, err := strconv.Atoi(id); err != nil {
			return fmt.Errorf("range_skills key %q is not a numeric id", id)
		}
		if skill.ProjectileName == "" {
			return fmt.Errorf("range_skills.%s is incomplete", id)
		}
	}
	return nil
}

// ProjectileSkillIDs returns the skill ids with a client projectile definition.
func (f *ConfigsFile) ProjectileSkillIDs() []int {
	ids := make([]int, 0, len(f.RangeSkills))
	for key := range f.RangeSkills {
		id, err := strconv.Atoi(key)
		if err == nil {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}

// RenderConfigs produces the [VS] Configs Ruby module.
func RenderConfigs(file *ConfigsFile) ([]byte, error) {
	if err := file.Validate(); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteString("# Generated by vxaos-content from configs.toml. Do not edit.\n")
	b.WriteString("module Configs\n\n")

	writeRubyString := func(name, value string) error {
		s, err := rubyString(value)
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "  %s = %s\n", name, s)
		return nil
	}
	writeInt := func(name string, value int) {
		fmt.Fprintf(&b, "  %s = %d\n", name, value)
	}
	writeInt64 := func(name string, value int64) {
		fmt.Fprintf(&b, "  %s = %d\n", name, value)
	}
	writeBool := func(name string, value bool) {
		fmt.Fprintf(&b, "  %s = %t\n", name, value)
	}
	writeSymbol := func(name, value string) error {
		if !rubySymbolRE.MatchString(value) {
			return fmt.Errorf("invalid symbol %q", value)
		}
		fmt.Fprintf(&b, "  %s = :%s\n", name, value)
		return nil
	}
	writeIntArray := func(name string, values []int) {
		b.WriteString("  " + name + " = [")
		for i, value := range values {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%d", value)
		}
		b.WriteString("]\n")
	}
	writeStringArray := func(name string, values []string) error {
		b.WriteString("  " + name + " = [")
		for i, value := range values {
			if i > 0 {
				b.WriteString(", ")
			}
			s, err := rubyString(value)
			if err != nil {
				return err
			}
			b.WriteString(s)
		}
		b.WriteString("]\n")
		return nil
	}

	// Fonts.
	if err := writeRubyString("CHAT_FONT_NAME", file.Fonts.Chat.Name); err != nil {
		return nil, err
	}
	writeBool("CHAT_FONT_OUTLINE", file.Fonts.Chat.Outline)
	writeBool("CHAT_FONT_SHADOW", file.Fonts.Chat.Shadow)
	writeBool("CHAT_FONT_BOLD", file.Fonts.Chat.Bold)
	writeInt("CHAT_FONT_SIZE", file.Fonts.Chat.Size)
	if err := writeRubyString("HOST", file.Network.Host); err != nil {
		return nil, err
	}
	writeInt("PORT", file.Network.Port)
	writeInt("GAME_VERSION", file.Network.GameVersion)
	if err := writeRubyString("SHOP_WEBSITE", file.Network.ShopWebsite); err != nil {
		return nil, err
	}

	b.WriteString("  RESOLUTIONS = [\n")
	for i, res := range file.Resolutions {
		comma := ","
		if i == len(file.Resolutions)-1 {
			comma = ""
		}
		fmt.Fprintf(&b, "    {:width => %d, :height => %d}%s\n", res.Width, res.Height, comma)
	}
	b.WriteString("  ]\n")

	writeInt("LOADING_TIME", file.LoadingTime)
	if err := writeStringArray("LOADING_TITLES", file.LoadingTitles); err != nil {
		return nil, err
	}

	l := file.Limits
	writeInt("MIN_CHARACTERS", l.MinCharacters)
	writeInt("MAX_CHARACTERS", l.MaxCharacters)
	writeInt("MIN_LEVEL_PVP", l.MinLevelPvP)
	writeInt("MAX_ACTORS", l.MaxActors)
	writeInt("MAX_HOTBAR", l.MaxHotbar)
	writeInt("MAX_DEFAULT_CLASSES", l.MaxDefaultClasses)
	writeInt("MAX_VIP_CLASSES", l.MaxVIPClasses)
	writeInt("MAX_ITEMS", l.MaxItems)
	writeInt64("MAX_GOLD", l.MaxGold)
	writeInt("MAX_PARAMS", l.MaxParams)
	writeInt("MAX_LEVEL", l.MaxLevel)
	writeInt("MAX_PLAYER_SWITCHES", l.MaxPlayerSwitches)
	writeInt("MAX_PLAYER_VARIABLES", l.MaxPlayerVariables)
	writeInt("MAX_FRIENDS", l.MaxFriends)
	writeInt("MAX_GUILD_MEMBERS", l.MaxGuildMembers)
	writeInt("MAX_PLAYER_ITEMS", l.MaxPlayerItems)
	writeInt("MAX_TRADE_ITEMS", l.MaxTradeItems)
	writeInt("MAX_BANK_ITEMS", l.MaxBankItems)
	writeInt("MAX_CHAT_LINES", l.MaxChatLines)
	writeInt("MAX_PARTY_MEMBERS", l.MaxPartyMembers)
	writeInt("MAX_EQUIPS", l.MaxEquips)

	writeIntArray("PAPERDOLL_DOWN_DIR", file.Paperdoll.Down)
	writeIntArray("PAPERDOLL_LEFT_DIR", file.Paperdoll.Left)
	writeIntArray("PAPERDOLL_RIGHT_DIR", file.Paperdoll.Right)
	writeIntArray("PAPERDOLL_UP_DIR", file.Paperdoll.Up)

	writeInt("NORMAL_COLOR", file.Colors.Chat.Normal)
	writeInt("GLOBAL_COLOR", file.Colors.Chat.Global)
	writeInt("SUCCESS_COLOR", file.Colors.Chat.Success)
	writeInt("ERROR_COLOR", file.Colors.Chat.Error)
	writeInt("ALERT_COLOR", file.Colors.Chat.Alert)
	writeInt("ADM_MSG_COLOR", file.Colors.Chat.AdmMsg)
	writeInt("DEFAULT_COLOR", file.Colors.Name.Default)
	writeInt("MONITOR_COLOR", file.Colors.Name.Monitor)
	writeInt("ADMIN_COLOR", file.Colors.Name.Admin)
	writeInt("ENEMY_COLOR", file.Colors.Name.Enemy)
	writeInt("BOSS_COLOR", file.Colors.Name.Boss)
	writeInt("GUILD_COLOR", file.Colors.Name.Guild)

	writeInt("ITEM_ICON", file.Icons.Menu.Item)
	writeInt("SKILL_ICON", file.Icons.Menu.Skill)
	writeInt("STATUS_ICON", file.Icons.Menu.Status)
	writeInt("QUEST_ICON", file.Icons.Menu.Quest)
	writeInt("FRIEND_ICON", file.Icons.Menu.Friend)
	writeInt("GUILD_ICON", file.Icons.Menu.Guild)
	writeInt("MENU_ICON", file.Icons.Menu.Menu)
	writeInt("PRIVATE_ICON", file.Icons.Interaction.Private)
	writeInt("BLOCK_ICON", file.Icons.Interaction.Block)
	writeInt("UNLOCK_ICON", file.Icons.Interaction.Unlock)
	writeInt("TRADE_ICON", file.Icons.Interaction.Trade)
	writeInt("PARTY_ICON", file.Icons.Interaction.Party)
	writeInt("CONFIG_ICON", file.Icons.Misc.Config)
	writeInt("GOLD_ICON", file.Icons.Misc.Gold)
	writeInt("EXP_ICON", file.Icons.Misc.Exp)
	writeInt("ADD_GOLD_ICON", file.Icons.Misc.AddGold)
	writeInt("REMOVE_GOLD_ICON", file.Icons.Misc.RemoveGold)
	writeInt("MAP_PVP_ICON", file.Icons.Misc.MapPvP)
	writeInt("MAP_NONPVP_ICON", file.Icons.Misc.MapNonPvP)
	writeInt("PLAYER_ON_ICON", file.Icons.Misc.PlayerOn)
	writeInt("PLAYER_OFF_ICON", file.Icons.Misc.PlayerOff)
	writeInt("QUEST_NOT_STARTED_ICON", file.Icons.Misc.QuestNotStarted)
	writeInt("QUEST_FINISHED_ICON", file.Icons.Misc.QuestFinished)
	writeInt("QUEST_IN_PROGRESS_ICON", file.Icons.Misc.QuestInProgress)
	writeInt("LEAVE_PARTY_ICON", file.Icons.Misc.LeaveParty)
	writeInt("EMOJI_ICON", file.Icons.Misc.Emoji)
	writeInt("ICON_BUFF_START", file.Icons.BuffStart)
	writeInt("ICON_DEBUFF_START", file.Icons.DebuffStart)

	writeInt("START_POINTS", file.StartPoints)
	writeInt("TITLE_BAR_HEIGHT", file.TitleBarHeight)

	fmt.Fprintf(&b, "  TELEPORTS = %s\n", renderTeleports(file.Teleports))

	b.WriteString("  RANGE_WEAPONS = {}\n")
	for _, id := range sortedRangeWeaponKeys(file.RangeWeapons) {
		weapon := file.RangeWeapons[id]
		fmt.Fprintf(&b, "  RANGE_WEAPONS[%s] = {\n", id)
		name, _ := rubyString(weapon.ProjectileName)
		fmt.Fprintf(&b, "    :projectile_name => %s,\n", name)
		fmt.Fprintf(&b, "    :range           => %d,\n", weapon.Range)
		fmt.Fprintf(&b, "    :item_id         => %d", weapon.ItemID)
		if weapon.StepAnime {
			b.WriteString(",\n    :step_anime      => true")
		}
		if weapon.MPCost > 0 {
			fmt.Fprintf(&b, ",\n    :mp_cost         => %d", weapon.MPCost)
		}
		b.WriteString("\n  }\n")
	}

	b.WriteString("  RANGE_SKILLS = {}\n")
	for _, id := range sortedStringIntKeys(file.RangeSkills) {
		skill := file.RangeSkills[id]
		fmt.Fprintf(&b, "  RANGE_SKILLS[%s] = {\n", id)
		name, _ := rubyString(skill.ProjectileName)
		fmt.Fprintf(&b, "    :projectile_name => %s", name)
		if skill.StepAnime {
			b.WriteString(",\n    :step_anime      => true")
		}
		b.WriteString("\n  }\n")
	}

	if err := writeStringArray("FORBIDDEN_NAMES", file.ForbiddenNames); err != nil {
		return nil, err
	}

	fmt.Fprintf(&b, "  ATTACK_TIME = %s\n", formatFloat(file.Timing.AttackTime))
	fmt.Fprintf(&b, "  COOLDOWN_SKILL_TIME = %s\n", formatFloat(file.Timing.CooldownSkillTime))
	fmt.Fprintf(&b, "  GLOBAL_ANTISPAM_TIME = %s\n", formatFloat(file.Timing.GlobalAntispamTime))
	writeInt("ATTACK_ANIMATION_TIME", file.Timing.AttackAnimationTime)
	writeInt("LEVEL_UP_ANIMATION_ID", file.Timing.LevelUpAnimationID)
	writeInt("MAX_MAP_DROPS", l.MaxMapDrops)

	if err := writeSymbol("ATTACK_KEY", file.Keys.Attack); err != nil {
		return nil, err
	}
	if err := writeSymbol("GET_DROP_KEY", file.Keys.GetDrop); err != nil {
		return nil, err
	}
	if err := writeSymbol("SELECT_ENEMY_KEY", file.Keys.SelectEnemy); err != nil {
		return nil, err
	}
	for _, pair := range []struct {
		name  string
		value string
	}{
		{"ITEM_KEY", file.Keys.Item}, {"SKILL_KEY", file.Keys.Skill}, {"STATUS_KEY", file.Keys.Status},
		{"QUEST_KEY", file.Keys.Quest}, {"FRIEND_KEY", file.Keys.Friend}, {"GUILD_KEY", file.Keys.Guild},
		{"MENU_KEY", file.Keys.Menu},
	} {
		if err := writeSymbol(pair.name, pair.value); err != nil {
			return nil, err
		}
	}
	b.WriteString("  HOTKEYS = [")
	for i, key := range file.Keys.Hotkeys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(":" + key)
	}
	b.WriteString("]\n")
	b.WriteString("  BALLOONS_KEYS = [")
	for i, key := range file.Keys.Balloons {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(":" + key)
	}
	b.WriteString("]\n")

	b.WriteString("\nend\n")

	// The default font is set through Font.default_* before the module body.
	module := b.String()
	var out bytes.Buffer
	out.WriteString("# Generated by vxaos-content from configs.toml. Do not edit.\n")
	fmt.Fprintf(&out, "Font.default_name = %s\n", mustRubyString(file.Fonts.Default.Name))
	fmt.Fprintf(&out, "Font.default_outline = %t\n", file.Fonts.Default.Outline)
	fmt.Fprintf(&out, "Font.default_shadow = %t\n", file.Fonts.Default.Shadow)
	fmt.Fprintf(&out, "Font.default_bold = %t\n", file.Fonts.Default.Bold)
	fmt.Fprintf(&out, "Font.default_size = %d\n\n", file.Fonts.Default.Size)
	out.WriteString(module[bytes.IndexByte([]byte(module), '\n')+1:])
	return out.Bytes(), nil
}

func renderTeleports(groups [][]TeleportStop) string {
	var b bytes.Buffer
	b.WriteString("[\n")
	for gi, group := range groups {
		b.WriteString("    [\n")
		for i, stop := range group {
			comma := ","
			if i == len(group)-1 {
				comma = ""
			}
			fmt.Fprintf(&b, "      {:map_id => %d, :x => %d, :y => %d, :gold => %d}%s\n", stop.MapID, stop.X, stop.Y, stop.Gold, comma)
		}
		comma := ","
		if gi == len(groups)-1 {
			comma = ""
		}
		b.WriteString("    ]" + comma + "\n")
	}
	b.WriteString("  ]")
	return b.String()
}

func sortedRangeWeaponKeys(values map[string]RangeWeapon) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return numericLess(keys[i], keys[j]) })
	return keys
}

func sortedStringIntKeys(values map[string]RangeSkill) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return numericLess(keys[i], keys[j]) })
	return keys
}

func numericLess(a, b string) bool {
	ai, aerr := strconv.Atoi(a)
	bi, berr := strconv.Atoi(b)
	if aerr == nil && berr == nil {
		return ai < bi
	}
	return a < b
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func mustRubyString(s string) string {
	out, _ := rubyString(s)
	return out
}
