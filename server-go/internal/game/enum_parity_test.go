package game

import (
	"testing"

	"github.com/kelaia/vxaos/server/internal/content"
)

func TestAdminCommandEnumParity(t *testing.T) {
	file, err := content.LoadEnums("../../content/enums.toml")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]uint8{
		"KICK": AdminKick, "TELEPORT": AdminTeleport, "GO": AdminGo, "PULL": AdminPull,
		"ITEM": AdminItem, "WEAPON": AdminWeapon, "ARMOR": AdminArmor, "GOLD": AdminGold,
		"BAN_IP": AdminBanIP, "BAN_ACC": AdminBanAcc, "UNBAN": AdminUnban, "SWITCH": AdminSwitch,
		"MOTD": AdminMotd, "MUTE": AdminMute, "MSG": AdminMessage,
	}
	found := false
	for _, group := range file.Enums {
		if group.Name != "Command" {
			continue
		}
		found = true
		for _, value := range group.Values {
			expected, ok := want[value.Name]
			if !ok {
				t.Fatalf("unexpected admin command %s", value.Name)
			}
			if uint8(value.Value) != expected {
				t.Fatalf("Command::%s = %d, Go = %d", value.Name, value.Value, expected)
			}
			delete(want, value.Name)
		}
	}
	if !found {
		t.Fatal("Command enum missing")
	}
	if len(want) != 0 {
		t.Fatalf("Go admin commands missing from enums.toml: %v", want)
	}
}

func TestSharedConfigServerValues(t *testing.T) {
	file, err := content.LoadConfigs("../../content/configs.toml")
	if err != nil {
		t.Fatal(err)
	}
	ids := file.ProjectileSkillIDs()
	if len(ids) != 2 || ids[0] != 51 || ids[1] != 70 {
		t.Fatalf("projectile ids = %v", ids)
	}
	if file.Limits.MaxEquips != 9 || file.Limits.MaxHotbar != 9 || file.Limits.MaxPartyMembers != 4 {
		t.Fatalf("limits = %+v", file.Limits)
	}
	if file.DropDespawnSeconds != 60 || file.DropPickupSeconds != 10 {
		t.Fatalf("drop timers = %d/%d", file.DropDespawnSeconds, file.DropPickupSeconds)
	}
}
