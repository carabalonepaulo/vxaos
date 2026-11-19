package protocol

import (
	"errors"
	"fmt"
)

// MaxEquips is the number of equipment slots carried in actor summaries.
// It must match the client's Configs::MAX_EQUIPS.
const MaxEquips = 9

// MaxActorsInList caps the actor list accepted from the wire.
const MaxActorsInList = 64

// MaxChatBytes caps a chat message accepted by the codec.
const MaxChatBytes = 4096

// MaxStateEntries caps variable-length aggregate arrays.
const MaxStateEntries = 4096

var (
	ErrUnknownOpcode = errors.New("protocol: unknown opcode")
	ErrUnsupported   = errors.New("protocol: opcode not implemented")
)

// ClientMessage is implemented by every client-to-server message.
type ClientMessage interface {
	ClientOpcode() ClientOp
	Encode() ([]byte, error)
}

// ServerMessage is implemented by every server-to-client message.
type ServerMessage interface {
	ServerOpcode() ServerOp
	Encode() ([]byte, error)
}

// ActorSummary is the character-selection entry sent in AuthOK and
// CharacterCreated.
type ActorSummary struct {
	Slot           uint8
	Name           string
	CharacterName  string
	CharacterIndex uint8
	FaceName       string
	FaceIndex      uint8
	Sex            uint8
	Equips         [MaxEquips]uint16
}

func writeActorSummary(w *Writer, a ActorSummary) {
	w.U8(a.Slot)
	w.String(a.Name)
	w.String(a.CharacterName)
	w.U8(a.CharacterIndex)
	w.String(a.FaceName)
	w.U8(a.FaceIndex)
	w.U8(a.Sex)
	for _, e := range a.Equips {
		w.U16(e)
	}
}

func readActorSummary(r *Reader) ActorSummary {
	var a ActorSummary
	a.Slot = r.U8()
	a.Name = r.String()
	a.CharacterName = r.String()
	a.CharacterIndex = r.U8()
	a.FaceName = r.String()
	a.FaceIndex = r.U8()
	a.Sex = r.U8()
	for i := range a.Equips {
		a.Equips[i] = r.U16()
	}
	return a
}

// AuthBegin starts account authentication.
type AuthBegin struct {
	Username string
}

// ClientOpcode implements ClientMessage.
func (m AuthBegin) ClientOpcode() ClientOp { return ClientAuthBegin }

// Encode implements ClientMessage.
func (m AuthBegin) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Username)
	return w.Done()
}

// AuthChallenge is the server's salt/nonce challenge.
type AuthChallenge struct {
	Salt       []byte
	KDF        uint8 // 1 = PBKDF2-HMAC-SHA256
	Iterations uint32
	Nonce      []byte
	ServerTime int64
}

// ServerOpcode implements ServerMessage.
func (m AuthChallenge) ServerOpcode() ServerOp { return ServerAuthChallenge }

// Encode implements ServerMessage.
func (m AuthChallenge) Encode() ([]byte, error) {
	w := NewWriter()
	w.Blob(m.Salt)
	w.U8(m.KDF)
	w.U32(m.Iterations)
	w.Blob(m.Nonce)
	w.I64(m.ServerTime)
	return w.Done()
}

// AuthProof is the client's challenge response.
type AuthProof struct {
	ClientNonce []byte
	Proof       []byte
}

// ClientOpcode implements ClientMessage.
func (m AuthProof) ClientOpcode() ClientOp { return ClientAuthProof }

// Encode implements ClientMessage.
func (m AuthProof) Encode() ([]byte, error) {
	w := NewWriter()
	w.Blob(m.ClientNonce)
	w.Blob(m.Proof)
	return w.Done()
}

// AuthOK completes authentication and carries the account's characters.
type AuthOK struct {
	Group    uint8
	VIPUntil int64
	Actors   []ActorSummary
}

// ServerOpcode implements ServerMessage.
func (m AuthOK) ServerOpcode() ServerOp { return ServerAuthOK }

// Encode implements ServerMessage.
func (m AuthOK) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Group)
	w.I64(m.VIPUntil)
	w.Count(len(m.Actors))
	for _, a := range m.Actors {
		writeActorSummary(w, a)
	}
	return w.Done()
}

// AuthError rejects authentication.
type AuthError struct {
	Code    uint16
	Message string
}

// ServerOpcode implements ServerMessage.
func (m AuthError) ServerOpcode() ServerOp { return ServerAuthError }

// Encode implements ServerMessage.
func (m AuthError) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.Code)
	w.String(m.Message)
	return w.Done()
}

// RegisterBegin starts account registration.
type RegisterBegin struct {
	Username string
	Email    string
}

// ClientOpcode implements ClientMessage.
func (m RegisterBegin) ClientOpcode() ClientOp { return ClientRegisterBegin }

// Encode implements ClientMessage.
func (m RegisterBegin) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Username)
	w.String(m.Email)
	return w.Done()
}

// RegisterChallenge is the server's registration challenge.
type RegisterChallenge struct {
	Salt       []byte
	KDF        uint8
	Iterations uint32
	Nonce      []byte
	ServerTime int64
}

// ServerOpcode implements ServerMessage.
func (m RegisterChallenge) ServerOpcode() ServerOp { return ServerRegisterChallenge }

// Encode implements ServerMessage.
func (m RegisterChallenge) Encode() ([]byte, error) {
	w := NewWriter()
	w.Blob(m.Salt)
	w.U8(m.KDF)
	w.U32(m.Iterations)
	w.Blob(m.Nonce)
	w.I64(m.ServerTime)
	return w.Done()
}

// RegisterProof carries the verifier and proof. The verifier must reach the
// server for storage, so registration over a plaintext connection exposes it;
// until SRP/TLS is added this flow should be treated as trusted-network only.
type RegisterProof struct {
	ClientNonce []byte
	Verifier    []byte
	Proof       []byte
}

// ClientOpcode implements ClientMessage.
func (m RegisterProof) ClientOpcode() ClientOp { return ClientRegisterProof }

// Encode implements ClientMessage.
func (m RegisterProof) Encode() ([]byte, error) {
	w := NewWriter()
	w.Blob(m.ClientNonce)
	w.Blob(m.Verifier)
	w.Blob(m.Proof)
	return w.Done()
}

// RegisterOK confirms registration.
type RegisterOK struct{}

// ServerOpcode implements ServerMessage.
func (m RegisterOK) ServerOpcode() ServerOp { return ServerRegisterOK }

// Encode implements ServerMessage.
func (m RegisterOK) Encode() ([]byte, error) { return nil, nil }

// RegisterError rejects registration.
type RegisterError struct {
	Code    uint16
	Message string
}

// ServerOpcode implements ServerMessage.
func (m RegisterError) ServerOpcode() ServerOp { return ServerRegisterError }

// Encode implements ServerMessage.
func (m RegisterError) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.Code)
	w.String(m.Message)
	return w.Done()
}

// CreateCharacter asks the server to create a character in a slot.
type CreateCharacter struct {
	Slot           uint8
	Name           string
	CharacterIndex uint8
	ClassID        uint16
	Sex            uint8
	Params         [8]uint8
}

// ClientOpcode implements ClientMessage.
func (m CreateCharacter) ClientOpcode() ClientOp { return ClientCreateCharacter }

// Encode implements ClientMessage.
func (m CreateCharacter) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Slot)
	w.String(m.Name)
	w.U8(m.CharacterIndex)
	w.U16(m.ClassID)
	w.U8(m.Sex)
	for _, p := range m.Params {
		w.U8(p)
	}
	return w.Done()
}

// CharacterCreated confirms a character creation.
type CharacterCreated struct {
	Actor ActorSummary
}

// ServerOpcode implements ServerMessage.
func (m CharacterCreated) ServerOpcode() ServerOp { return ServerCharacterCreated }

// Encode implements ServerMessage.
func (m CharacterCreated) Encode() ([]byte, error) {
	w := NewWriter()
	writeActorSummary(w, m.Actor)
	return w.Done()
}

// CharacterError rejects a character operation.
type CharacterError struct {
	Code    uint16
	Message string
}

// ServerOpcode implements ServerMessage.
func (m CharacterError) ServerOpcode() ServerOp { return ServerCharacterError }

// Encode implements ServerMessage.
func (m CharacterError) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.Code)
	w.String(m.Message)
	return w.Done()
}

// SelectCharacter asks to enter the world with a character slot.
type SelectCharacter struct {
	Slot uint8
}

// ClientOpcode implements ClientMessage.
func (m SelectCharacter) ClientOpcode() ClientOp { return ClientSelectCharacter }

// Encode implements ClientMessage.
func (m SelectCharacter) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Slot)
	return w.Done()
}

// Move requests one tile step or a turn.
type Move struct {
	Direction uint8
}

// ClientOpcode implements ClientMessage.
func (m Move) ClientOpcode() ClientOp { return ClientMove }

// Encode implements ClientMessage.
func (m Move) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Direction)
	return w.Done()
}

// Chat sends a chat message. Channel follows the game's Enums::Chat values.
type Chat struct {
	Channel uint8
	Message string
	Target  string
}

// ClientOpcode implements ClientMessage.
func (m Chat) ClientOpcode() ClientOp { return ClientChat }

// Encode implements ClientMessage.
func (m Chat) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Channel)
	w.String(m.Message)
	w.String(m.Target)
	return w.Done()
}

// EquipItem equips or unequips an inventory item. Slot 0 is the weapon;
// itemID 0 unequips the slot.
type EquipItem struct {
	Slot   uint8
	ItemID uint16
}

// ClientOpcode implements ClientMessage.
func (m EquipItem) ClientOpcode() ClientOp { return ClientEquip }

// Encode implements ClientMessage.
func (m EquipItem) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Slot)
	w.U16(m.ItemID)
	return w.Done()
}

// SetHotbar assigns one hotbar slot. Type 0 clears it, 1 is an item and 2 is a
// skill.
type SetHotbar struct {
	Slot   uint8
	Type   uint8
	ItemID uint16
}

// ClientOpcode implements ClientMessage.
func (m SetHotbar) ClientOpcode() ClientOp { return ClientSetHotbar }

// Encode implements ClientMessage.
func (m SetHotbar) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Slot)
	w.U8(m.Type)
	w.U16(m.ItemID)
	return w.Done()
}

// SpendPoint spends one creation point on a parameter.
type SpendPoint struct {
	ParamID uint8
}

// ClientOpcode implements ClientMessage.
func (m SpendPoint) ClientOpcode() ClientOp { return ClientSpendPoint }

// Encode implements ClientMessage.
func (m SpendPoint) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.ParamID)
	return w.Done()
}

// Attack requests a normal weapon attack.
type Attack struct{}

// ClientOpcode implements ClientMessage.
func (m Attack) ClientOpcode() ClientOp { return ClientAttack }

// Encode implements ClientMessage.
func (m Attack) Encode() ([]byte, error) { return nil, nil }

// Target selects a player or enemy target. Type follows Enums::Target.
type Target struct {
	Type uint8
	ID   uint16
}

// ClientOpcode implements ClientMessage.
func (m Target) ClientOpcode() ClientOp { return ClientSetTarget }

// Encode implements ClientMessage.
func (m Target) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Type)
	w.U16(m.ID)
	return w.Done()
}

// UseItem consumes/uses an inventory item.
type UseItem struct {
	ItemID uint16
}

// ClientOpcode implements ClientMessage.
func (m UseItem) ClientOpcode() ClientOp { return ClientUseItem }

// Encode implements ClientMessage.
func (m UseItem) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ItemID)
	return w.Done()
}

// UseSkill casts a learned skill.
type UseSkill struct {
	SkillID uint16
}

// ClientOpcode implements ClientMessage.
func (m UseSkill) ClientOpcode() ClientOp { return ClientUseSkill }

// Encode implements ClientMessage.
func (m UseSkill) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.SkillID)
	return w.Done()
}

// UseHotbar activates one hotbar slot.
type UseHotbar struct {
	Slot uint8
}

// ClientOpcode implements ClientMessage.
func (m UseHotbar) ClientOpcode() ClientOp { return ClientUseHotbar }

// Encode implements ClientMessage.
func (m UseHotbar) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Slot)
	return w.Done()
}

// Choice answers a server-side event choice or number input.
type Choice struct {
	Index int32
}

// ClientOpcode implements ClientMessage.
func (m Choice) ClientOpcode() ClientOp { return ClientChoice }

// Encode implements ClientMessage.
func (m Choice) Encode() ([]byte, error) {
	w := NewWriter()
	w.I32(m.Index)
	return w.Done()
}

// EventNext acknowledges a message window and resumes the event.
type EventNext struct{}

// ClientOpcode implements ClientMessage.
func (m EventNext) ClientOpcode() ClientOp { return ClientEventNext }

// Encode implements ClientMessage.
func (m EventNext) Encode() ([]byte, error) { return nil, nil }

// ChoiceTeleport selects a destination in the open teleport window.
type ChoiceTeleport struct {
	Index uint8
}

// ClientOpcode implements ClientMessage.
func (m ChoiceTeleport) ClientOpcode() ClientOp { return ClientChoiceTeleport }

// Encode implements ClientMessage.
func (m ChoiceTeleport) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Index)
	return w.Done()
}

// ShopBuy buys a quantity of one shop good.
type ShopBuy struct {
	Index  uint8
	Amount uint16
}

// ClientOpcode implements ClientMessage.
func (m ShopBuy) ClientOpcode() ClientOp { return ClientShopBuy }

// Encode implements ClientMessage.
func (m ShopBuy) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Index)
	w.U16(m.Amount)
	return w.Done()
}

// ShopSell sells inventory items.
type ShopSell struct {
	ItemID uint16
	Kind   uint8
	Amount uint16
}

// ClientOpcode implements ClientMessage.
func (m ShopSell) ClientOpcode() ClientOp { return ClientShopSell }

// Encode implements ClientMessage.
func (m ShopSell) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ItemID)
	w.U8(m.Kind)
	w.U16(m.Amount)
	return w.Done()
}

// BankItemRequest moves a stack between the inventory and the bank. Positive
// amounts deposit, negative amounts withdraw.
type BankItemRequest struct {
	ItemID uint16
	Kind   uint8
	Amount int16
}

// ClientOpcode implements ClientMessage.
func (m BankItemRequest) ClientOpcode() ClientOp { return ClientBankItem }

// Encode implements ClientMessage.
func (m BankItemRequest) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ItemID)
	w.U8(m.Kind)
	w.I16(m.Amount)
	return w.Done()
}

// BankGoldRequest moves gold between the player and the bank. Positive amounts
// deposit, negative amounts withdraw.
type BankGoldRequest struct {
	Amount int32
}

// ClientOpcode implements ClientMessage.
func (m BankGoldRequest) ClientOpcode() ClientOp { return ClientBankGold }

// Encode implements ClientMessage.
func (m BankGoldRequest) Encode() ([]byte, error) {
	w := NewWriter()
	w.I32(m.Amount)
	return w.Done()
}

// CloseWindow closes the current server-driven window.
type CloseWindow struct{}

// ClientOpcode implements ClientMessage.
func (m CloseWindow) ClientOpcode() ClientOp { return ClientCloseWindow }

// Encode implements ClientMessage.
func (m CloseWindow) Encode() ([]byte, error) { return nil, nil }

// GuildCreate creates a guild from the create-guild window.
type GuildCreate struct {
	Name string
	Flag []byte
}

// ClientOpcode implements ClientMessage.
func (m GuildCreate) ClientOpcode() ClientOp { return ClientGuildCreate }

// Encode implements ClientMessage.
func (m GuildCreate) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Name)
	for i := 0; i < 64; i++ {
		value := byte(255)
		if i < len(m.Flag) {
			value = m.Flag[i]
		}
		w.U8(value)
	}
	return w.Done()
}

// GuildOpenRequest asks for the guild window contents.
type GuildOpenRequest struct{}

// ClientOpcode implements ClientMessage.
func (m GuildOpenRequest) ClientOpcode() ClientOp { return ClientGuildOpen }

// Encode implements ClientMessage.
func (m GuildOpenRequest) Encode() ([]byte, error) { return nil, nil }

// GuildSetLeader transfers guild leadership.
type GuildSetLeader struct {
	Name string
}

// ClientOpcode implements ClientMessage.
func (m GuildSetLeader) ClientOpcode() ClientOp { return ClientGuildSetLeader }

// Encode implements ClientMessage.
func (m GuildSetLeader) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Name)
	return w.Done()
}

// GuildSetNotice changes the guild notice.
type GuildSetNotice struct {
	Notice string
}

// ClientOpcode implements ClientMessage.
func (m GuildSetNotice) ClientOpcode() ClientOp { return ClientGuildSetNotice }

// Encode implements ClientMessage.
func (m GuildSetNotice) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Notice)
	return w.Done()
}

// GuildKick removes a member from the guild.
type GuildKick struct {
	Name string
}

// ClientOpcode implements ClientMessage.
func (m GuildKick) ClientOpcode() ClientOp { return ClientGuildKick }

// Encode implements ClientMessage.
func (m GuildKick) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Name)
	return w.Done()
}

// GuildInvite invites an online player to the guild.
type GuildInvite struct {
	Name string
}

// ClientOpcode implements ClientMessage.
func (m GuildInvite) ClientOpcode() ClientOp { return ClientGuildInvite }

// Encode implements ClientMessage.
func (m GuildInvite) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Name)
	return w.Done()
}

// GuildLeave leaves the guild (the leader deletes it).
type GuildLeave struct{}

// ClientOpcode implements ClientMessage.
func (m GuildLeave) ClientOpcode() ClientOp { return ClientGuildLeave }

// Encode implements ClientMessage.
func (m GuildLeave) Encode() ([]byte, error) { return nil, nil }

// RequestAccept accepts the pending request (guild invite, trade, party...).
type RequestAccept struct{}

// ClientOpcode implements ClientMessage.
func (m RequestAccept) ClientOpcode() ClientOp { return ClientRequestAccept }

// Encode implements ClientMessage.
func (m RequestAccept) Encode() ([]byte, error) { return nil, nil }

// RequestDecline declines the pending request.
type RequestDecline struct{}

// ClientOpcode implements ClientMessage.
func (m RequestDecline) ClientOpcode() ClientOp { return ClientRequestDecline }

// Encode implements ClientMessage.
func (m RequestDecline) Encode() ([]byte, error) { return nil, nil }

// RequestInvite asks another player for a party/guild/friend/trade.
type RequestInvite struct {
	Type     uint8
	PlayerID uint16
}

// ClientOpcode implements ClientMessage.
func (m RequestInvite) ClientOpcode() ClientOp { return ClientRequest }

// Encode implements ClientMessage.
func (m RequestInvite) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Type)
	w.U16(m.PlayerID)
	return w.Done()
}

// PartyLeaveRequest leaves the current party.
type PartyLeaveRequest struct{}

// ClientOpcode implements ClientMessage.
func (m PartyLeaveRequest) ClientOpcode() ClientOp { return ClientPartyLeave }

// Encode implements ClientMessage.
func (m PartyLeaveRequest) Encode() ([]byte, error) { return nil, nil }

// OpenFriends requests the friend list window.
type OpenFriends struct{}

// ClientOpcode implements ClientMessage.
func (m OpenFriends) ClientOpcode() ClientOp { return ClientOpenFriends }

// Encode implements ClientMessage.
func (m OpenFriends) Encode() ([]byte, error) { return nil, nil }

// RemoveFriend removes one entry from the friend list.
type RemoveFriend struct {
	Index uint8
}

// ClientOpcode implements ClientMessage.
func (m RemoveFriend) ClientOpcode() ClientOp { return ClientRemoveFriend }

// Encode implements ClientMessage.
func (m RemoveFriend) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Index)
	return w.Done()
}

// TradeItemOffer adds/removes an item in the current trade.
type TradeItemOffer struct {
	ItemID uint16
	Kind   uint8
	Amount int16
}

// ClientOpcode implements ClientMessage.
func (m TradeItemOffer) ClientOpcode() ClientOp { return ClientTradeItem }

// Encode implements ClientMessage.
func (m TradeItemOffer) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ItemID)
	w.U8(m.Kind)
	w.I16(m.Amount)
	return w.Done()
}

// TradeGoldOffer adds/removes gold in the current trade.
type TradeGoldOffer struct {
	Amount int32
}

// ClientOpcode implements ClientMessage.
func (m TradeGoldOffer) ClientOpcode() ClientOp { return ClientTradeGold }

// Encode implements ClientMessage.
func (m TradeGoldOffer) Encode() ([]byte, error) {
	w := NewWriter()
	w.I32(m.Amount)
	return w.Done()
}

// DeleteCharacter removes one character slot after proving the account
// password again.
type DeleteCharacter struct {
	Slot        uint8
	ClientNonce []byte
	Proof       []byte
}

// ClientOpcode implements ClientMessage.
func (m DeleteCharacter) ClientOpcode() ClientOp { return ClientDeleteCharacter }

// Encode implements ClientMessage.
func (m DeleteCharacter) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Slot)
	w.Blob(m.ClientNonce)
	w.Blob(m.Proof)
	return w.Done()
}

// CharacterDeleted reports a removed character slot.
type CharacterDeleted struct {
	Slot uint8
}

// ServerOpcode implements ServerMessage.
func (m CharacterDeleted) ServerOpcode() ServerOp { return ServerCharacterDeleted }

// Encode implements ServerMessage.
func (m CharacterDeleted) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Slot)
	return w.Done()
}

// DropItemRequest drops items from the inventory onto the map.
type DropItemRequest struct {
	ItemID uint16
	Kind   uint8
	Amount uint16
}

// ClientOpcode implements ClientMessage.
func (m DropItemRequest) ClientOpcode() ClientOp { return ClientDropItem }

// Encode implements ClientMessage.
func (m DropItemRequest) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ItemID)
	w.U8(m.Kind)
	w.U16(m.Amount)
	return w.Done()
}

// PickupDrop picks up one ground drop by list index.
type PickupDrop struct {
	DropID uint8
}

// ClientOpcode implements ClientMessage.
func (m PickupDrop) ClientOpcode() ClientOp { return ClientPickupDrop }

// Encode implements ClientMessage.
func (m PickupDrop) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.DropID)
	return w.Done()
}

// AdminCommand sends one administrator panel command.
type AdminCommand struct {
	Command uint8
	Text    string
	Int1    int32
	Int2    int32
	Int3    int16
}

// ClientOpcode implements ClientMessage.
func (m AdminCommand) ClientOpcode() ClientOp { return ClientAdminCommand }

// Encode implements ClientMessage.
func (m AdminCommand) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Command)
	w.String(m.Text)
	w.I32(m.Int1)
	w.I32(m.Int2)
	w.I16(m.Int3)
	return w.Done()
}

// AdminCommandNotice is the server-side admin panel notification.
type AdminCommandNotice struct {
	Command uint8
	Message string
}

// ServerOpcode implements ServerMessage.
func (m AdminCommandNotice) ServerOpcode() ServerOp { return ServerAdminCommand }

// Encode implements ServerMessage.
func (m AdminCommandNotice) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Command)
	w.String(m.Message)
	return w.Done()
}

// Ping measures latency and keeps the session alive.
type Ping struct {
	Timestamp int64
}

// ClientOpcode implements ClientMessage.
func (m Ping) ClientOpcode() ClientOp { return ClientPing }

// Encode implements ClientMessage.
func (m Ping) Encode() ([]byte, error) {
	w := NewWriter()
	w.I64(m.Timestamp)
	return w.Done()
}

// Pong answers Ping.
type Pong struct {
	Timestamp int64
}

// ServerOpcode implements ServerMessage.
func (m Pong) ServerOpcode() ServerOp { return ServerPong }

// Encode implements ServerMessage.
func (m Pong) Encode() ([]byte, error) {
	w := NewWriter()
	w.I64(m.Timestamp)
	return w.Done()
}

// ErrorMessage reports a protocol or domain error.
type ErrorMessage struct {
	Code    uint16
	Message string
}

// ServerOpcode implements ServerMessage.
func (m ErrorMessage) ServerOpcode() ServerOp { return ServerError }

// Encode implements ServerMessage.
func (m ErrorMessage) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.Code)
	w.String(m.Message)
	return w.Done()
}

// Disconnect tells the client why the server is closing the connection.
type Disconnect struct {
	Code    uint16
	Message string
}

// ServerOpcode implements ServerMessage.
func (m Disconnect) ServerOpcode() ServerOp { return ServerDisconnect }

// Encode implements ServerMessage.
func (m Disconnect) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.Code)
	w.String(m.Message)
	return w.Done()
}

// PlayerSnapshot is the compact player state sent on enter and for other
// players. The full inventory/quests/switch aggregate will be sent by later
// dedicated messages as the client is ported.
type PlayerSnapshot struct {
	ID             uint16
	Group          uint8
	Name           string
	CharacterName  string
	CharacterIndex uint8
	FaceName       string
	FaceIndex      uint8
	Sex            uint8
	ClassID        uint16
	Level          int32
	HP             int32
	MP             int32
	MHP            int32
	MMP            int32
	EXP            int32
	Points         int32
	Gold           int64
	GuildName      string
	MapID          uint16
	X              uint16
	Y              uint16
	Direction      uint8
	Equips         [MaxEquips]uint16
}

func writePlayerSnapshot(w *Writer, p PlayerSnapshot) {
	w.U16(p.ID)
	w.U8(p.Group)
	w.String(p.Name)
	w.String(p.CharacterName)
	w.U8(p.CharacterIndex)
	w.String(p.FaceName)
	w.U8(p.FaceIndex)
	w.U8(p.Sex)
	w.U16(p.ClassID)
	w.I32(p.Level)
	w.I32(p.HP)
	w.I32(p.MP)
	w.I32(p.MHP)
	w.I32(p.MMP)
	w.I32(p.EXP)
	w.I32(p.Points)
	w.I64(p.Gold)
	w.String(p.GuildName)
	w.U16(p.MapID)
	w.U16(p.X)
	w.U16(p.Y)
	w.U8(p.Direction)
	for _, e := range p.Equips {
		w.U16(e)
	}
}

func readPlayerSnapshot(r *Reader) PlayerSnapshot {
	var p PlayerSnapshot
	p.ID = r.U16()
	p.Group = r.U8()
	p.Name = r.String()
	p.CharacterName = r.String()
	p.CharacterIndex = r.U8()
	p.FaceName = r.String()
	p.FaceIndex = r.U8()
	p.Sex = r.U8()
	p.ClassID = r.U16()
	p.Level = r.I32()
	p.HP = r.I32()
	p.MP = r.I32()
	p.MHP = r.I32()
	p.MMP = r.I32()
	p.EXP = r.I32()
	p.Points = r.I32()
	p.Gold = r.I64()
	p.GuildName = r.String()
	p.MapID = r.U16()
	p.X = r.U16()
	p.Y = r.U16()
	p.Direction = r.U8()
	for i := range p.Equips {
		p.Equips[i] = r.U16()
	}
	return p
}

// ItemEntry is one inventory/bank stack in EnterWorld.
type ItemEntry struct {
	Kind   uint8
	ItemID uint16
	Amount uint16
}

// QuestEntry is one quest in EnterWorld.
type QuestEntry struct {
	QuestID uint8
	State   uint8
}

// HotbarEntry is one hotbar slot in EnterWorld.
type HotbarEntry struct {
	Type   uint8
	ItemID uint16
}

// SelfSwitchEntry is one self switch in EnterWorld.
type SelfSwitchEntry struct {
	MapID   uint16
	EventID uint16
	Ch      string
	Value   bool
}

// EnterWorld tells the client that its character is now in the world and
// carries the complete initial aggregate.
type EnterWorld struct {
	Player       PlayerSnapshot
	Params       [8]int32
	Items        []ItemEntry
	Weapons      []ItemEntry
	Armors       []ItemEntry
	Skills       []uint16
	Friends      []string
	Quests       []QuestEntry
	Hotbar       []HotbarEntry
	Switches     []bool
	Variables    []int32
	SelfSwitches []SelfSwitchEntry
}

// ServerOpcode implements ServerMessage.
func (m EnterWorld) ServerOpcode() ServerOp { return ServerEnterWorld }

// Encode implements ServerMessage.
func (m EnterWorld) Encode() ([]byte, error) {
	w := NewWriter()
	writePlayerSnapshot(w, m.Player)
	for _, p := range m.Params {
		w.I32(p)
	}
	writeItems := func(items []ItemEntry) {
		w.Count(len(items))
		for _, it := range items {
			w.U8(it.Kind)
			w.U16(it.ItemID)
			w.U16(it.Amount)
		}
	}
	writeItems(m.Items)
	writeItems(m.Weapons)
	writeItems(m.Armors)
	w.Count(len(m.Skills))
	for _, s := range m.Skills {
		w.U16(s)
	}
	w.Count(len(m.Friends))
	for _, f := range m.Friends {
		w.String(f)
	}
	w.Count(len(m.Quests))
	for _, q := range m.Quests {
		w.U8(q.QuestID)
		w.U8(q.State)
	}
	w.Count(len(m.Hotbar))
	for _, h := range m.Hotbar {
		w.U8(h.Type)
		w.U16(h.ItemID)
	}
	w.Count(len(m.Switches))
	for _, b := range m.Switches {
		w.Bool(b)
	}
	w.Count(len(m.Variables))
	for _, v := range m.Variables {
		w.I32(v)
	}
	w.Count(len(m.SelfSwitches))
	for _, s := range m.SelfSwitches {
		w.U16(s.MapID)
		w.U16(s.EventID)
		w.String(s.Ch)
		w.Bool(s.Value)
	}
	return w.Done()
}

func readItems(r *Reader) []ItemEntry {
	n := r.Count()
	if n > MaxStateEntries {
		r.fail("item count %d exceeds limit", n)
		return nil
	}
	out := make([]ItemEntry, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, ItemEntry{Kind: r.U8(), ItemID: r.U16(), Amount: r.U16()})
	}
	return out
}

func readEnterWorld(r *Reader) EnterWorld {
	var m EnterWorld
	m.Player = readPlayerSnapshot(r)
	for i := range m.Params {
		m.Params[i] = r.I32()
	}
	m.Items = readItems(r)
	m.Weapons = readItems(r)
	m.Armors = readItems(r)
	n := r.Count()
	if n > MaxStateEntries {
		r.fail("skill count %d exceeds limit", n)
		return m
	}
	m.Skills = make([]uint16, n)
	for i := range m.Skills {
		m.Skills[i] = r.U16()
	}
	n = r.Count()
	if n > MaxStateEntries {
		r.fail("friend count %d exceeds limit", n)
		return m
	}
	m.Friends = make([]string, n)
	for i := range m.Friends {
		m.Friends[i] = r.String()
	}
	n = r.Count()
	if n > MaxStateEntries {
		r.fail("quest count %d exceeds limit", n)
		return m
	}
	m.Quests = make([]QuestEntry, n)
	for i := range m.Quests {
		m.Quests[i] = QuestEntry{QuestID: r.U8(), State: r.U8()}
	}
	n = r.Count()
	if n > MaxStateEntries {
		r.fail("hotbar count %d exceeds limit", n)
		return m
	}
	m.Hotbar = make([]HotbarEntry, n)
	for i := range m.Hotbar {
		m.Hotbar[i] = HotbarEntry{Type: r.U8(), ItemID: r.U16()}
	}
	n = r.Count()
	if n > MaxStateEntries {
		r.fail("switch count %d exceeds limit", n)
		return m
	}
	m.Switches = make([]bool, n)
	for i := range m.Switches {
		m.Switches[i] = r.Bool()
	}
	n = r.Count()
	if n > MaxStateEntries {
		r.fail("variable count %d exceeds limit", n)
		return m
	}
	m.Variables = make([]int32, n)
	for i := range m.Variables {
		m.Variables[i] = r.I32()
	}
	n = r.Count()
	if n > MaxStateEntries {
		r.fail("self switch count %d exceeds limit", n)
		return m
	}
	m.SelfSwitches = make([]SelfSwitchEntry, n)
	for i := range m.SelfSwitches {
		m.SelfSwitches[i] = SelfSwitchEntry{MapID: r.U16(), EventID: r.U16(), Ch: r.String(), Value: r.Bool()}
	}
	return m
}

// PlayerData introduces another player already on the map.
type PlayerData struct {
	Player PlayerSnapshot
}

// ServerOpcode implements ServerMessage.
func (m PlayerData) ServerOpcode() ServerOp { return ServerPlayerData }

// Encode implements ServerMessage.
func (m PlayerData) Encode() ([]byte, error) {
	w := NewWriter()
	writePlayerSnapshot(w, m.Player)
	return w.Done()
}

// PlayerMove reports a player step or turn.
type PlayerMove struct {
	ID        uint16
	X         uint16
	Y         uint16
	Direction uint8
}

// ServerOpcode implements ServerMessage.
func (m PlayerMove) ServerOpcode() ServerOp { return ServerPlayerMove }

// Encode implements ServerMessage.
func (m PlayerMove) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ID)
	w.U16(m.X)
	w.U16(m.Y)
	w.U8(m.Direction)
	return w.Done()
}

// PlayerLeft removes a player from the map.
type PlayerLeft struct {
	ID uint16
}

// ServerOpcode implements ServerMessage.
func (m PlayerLeft) ServerOpcode() ServerOp { return ServerPlayerLeft }

// Encode implements ServerMessage.
func (m PlayerLeft) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ID)
	return w.Done()
}

// MapChat is a message shown to players on one map.
type MapChat struct {
	PlayerID uint16
	Color    uint8
	Message  string
}

// ServerOpcode implements ServerMessage.
func (m MapChat) ServerOpcode() ServerOp { return ServerMapChat }

// Encode implements ServerMessage.
func (m MapChat) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.PlayerID)
	w.U8(m.Color)
	w.String(m.Message)
	return w.Done()
}

// ChatMessage is a global/party/guild chat line broadcast by the server.
type ChatMessage struct {
	Color   uint8
	Message string
}

// ServerOpcode implements ServerMessage.
func (m ChatMessage) ServerOpcode() ServerOp { return ServerChat }

// Encode implements ServerMessage.
func (m ChatMessage) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Color)
	w.String(m.Message)
	return w.Done()
}

// Vitals reports current HP/MP for one player.
type Vitals struct {
	PlayerID uint16
	HP       int32
	MP       int32
}

// ServerOpcode implements ServerMessage.
func (m Vitals) ServerOpcode() ServerOp { return ServerVitals }

// Encode implements ServerMessage.
func (m Vitals) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.PlayerID)
	w.I32(m.HP)
	w.I32(m.MP)
	return w.Done()
}

// Equip reports a changed equipment slot for a player.
type Equip struct {
	PlayerID uint16
	Slot     uint8
	ItemID   uint16
}

// ServerOpcode implements ServerMessage.
func (m Equip) ServerOpcode() ServerOp { return ServerEquip }

// Encode implements ServerMessage.
func (m Equip) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.PlayerID)
	w.U8(m.Slot)
	w.U16(m.ItemID)
	return w.Done()
}

// Hotbar reports the player's own hotbar slot change.
type Hotbar struct {
	Slot   uint8
	Type   uint8
	ItemID uint16
}

// ServerOpcode implements ServerMessage.
func (m Hotbar) ServerOpcode() ServerOp { return ServerHotbar }

// Encode implements ServerMessage.
func (m Hotbar) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Slot)
	w.U8(m.Type)
	w.U16(m.ItemID)
	return w.Done()
}

// Param reports a parameter bonus change.
type Param struct {
	PlayerID uint16
	ParamID  uint8
	Value    int32
}

// ServerOpcode implements ServerMessage.
func (m Param) ServerOpcode() ServerOp { return ServerParam }

// Encode implements ServerMessage.
func (m Param) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.PlayerID)
	w.U8(m.ParamID)
	w.I32(m.Value)
	return w.Done()
}

// Points reports the remaining creation points.
type Points struct {
	Value int32
}

// ServerOpcode implements ServerMessage.
func (m Points) ServerOpcode() ServerOp { return ServerPoints }

// Encode implements ServerMessage.
func (m Points) Encode() ([]byte, error) {
	w := NewWriter()
	w.I32(m.Value)
	return w.Done()
}

// Item reports an inventory delta. Kind follows the character aggregate: 1
// item, 2 weapon, 3 armor.
type Item struct {
	ItemID    uint16
	Kind      uint8
	Amount    int32
	DropSound bool
	Popup     bool
}

// ServerOpcode implements ServerMessage.
func (m Item) ServerOpcode() ServerOp { return ServerItem }

// Encode implements ServerMessage.
func (m Item) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ItemID)
	w.U8(m.Kind)
	w.I32(m.Amount)
	w.Bool(m.DropSound)
	w.Bool(m.Popup)
	return w.Done()
}

// Gold reports a gold delta.
type Gold struct {
	Amount    int64
	ShopSound bool
	Popup     bool
}

// ServerOpcode implements ServerMessage.
func (m Gold) ServerOpcode() ServerOp { return ServerGold }

// Encode implements ServerMessage.
func (m Gold) Encode() ([]byte, error) {
	w := NewWriter()
	w.I64(m.Amount)
	w.Bool(m.ShopSound)
	w.Bool(m.Popup)
	return w.Done()
}

// Skill reports a learned/forgotten skill.
type Skill struct {
	SkillID uint16
	Learned bool
}

// ServerOpcode implements ServerMessage.
func (m Skill) ServerOpcode() ServerOp { return ServerSkill }

// Encode implements ServerMessage.
func (m Skill) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.SkillID)
	w.Bool(m.Learned)
	return w.Done()
}

// AttackEnemy reports damage to a map enemy event.
type AttackEnemy struct {
	AttackerID   uint16
	AttackerType uint8
	AniIndex     uint8
	EventID      uint16
	HPDamage     int32
	MPDamage     int32
	Critical     bool
	AnimationID  uint16
}

// ServerOpcode implements ServerMessage.
func (m AttackEnemy) ServerOpcode() ServerOp { return ServerAttackEnemy }

// Encode implements ServerMessage.
func (m AttackEnemy) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.AttackerID)
	w.U8(m.AttackerType)
	w.U8(m.AniIndex)
	w.U16(m.EventID)
	w.I32(m.HPDamage)
	w.I32(m.MPDamage)
	w.Bool(m.Critical)
	w.U16(m.AnimationID)
	return w.Done()
}

// AttackPlayer reports damage to a player.
type AttackPlayer struct {
	AttackerID    uint16
	AttackerType  uint8
	AniIndex      uint8
	PlayerID      uint16
	HPDamage      int32
	MPDamage      int32
	Critical      bool
	AnimationID   uint16
	NotShowMissed bool
}

// ServerOpcode implements ServerMessage.
func (m AttackPlayer) ServerOpcode() ServerOp { return ServerAttackPlayer }

// Encode implements ServerMessage.
func (m AttackPlayer) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.AttackerID)
	w.U8(m.AttackerType)
	w.U8(m.AniIndex)
	w.U16(m.PlayerID)
	w.I32(m.HPDamage)
	w.I32(m.MPDamage)
	w.Bool(m.Critical)
	w.U16(m.AnimationID)
	w.Bool(m.NotShowMissed)
	return w.Done()
}

// EventData synchronizes a map event's position and HP.
type EventData struct {
	EventID   uint16
	X         uint16
	Y         uint16
	Direction uint8
	HP        int32
}

// ServerOpcode implements ServerMessage.
func (m EventData) ServerOpcode() ServerOp { return ServerEventData }

// Encode implements ServerMessage.
func (m EventData) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.EventID)
	w.U16(m.X)
	w.U16(m.Y)
	w.U8(m.Direction)
	w.I32(m.HP)
	return w.Done()
}

// EventMove synchronizes a moving event.
type EventMove struct {
	EventID   uint16
	X         uint16
	Y         uint16
	Direction uint8
}

// ServerOpcode implements ServerMessage.
func (m EventMove) ServerOpcode() ServerOp { return ServerEventMove }

// Encode implements ServerMessage.
func (m EventMove) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.EventID)
	w.U16(m.X)
	w.U16(m.Y)
	w.U8(m.Direction)
	return w.Done()
}

// Projectile spawns a visual projectile on the map.
type Projectile struct {
	StartX         uint16
	StartY         uint16
	FinishX        uint16
	FinishY        uint16
	TargetX        uint16
	TargetY        uint16
	ProjectileType uint8
	ProjectileID   uint8
}

// ServerOpcode implements ServerMessage.
func (m Projectile) ServerOpcode() ServerOp { return ServerProjectile }

// Encode implements ServerMessage.
func (m Projectile) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.StartX)
	w.U16(m.StartY)
	w.U16(m.FinishX)
	w.U16(m.FinishY)
	w.U16(m.TargetX)
	w.U16(m.TargetY)
	w.U8(m.ProjectileType)
	w.U8(m.ProjectileID)
	return w.Done()
}

// DropAdd adds a ground drop to the map's drop list.
type DropAdd struct {
	ItemID uint16
	Kind   uint8
	Amount uint16
	X      uint16
	Y      uint16
}

// ServerOpcode implements ServerMessage.
func (m DropAdd) ServerOpcode() ServerOp { return ServerDropAdd }

// Encode implements ServerMessage.
func (m DropAdd) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ItemID)
	w.U8(m.Kind)
	w.U16(m.Amount)
	w.U16(m.X)
	w.U16(m.Y)
	return w.Done()
}

// DropRemove removes one ground drop by list index.
type DropRemove struct {
	DropID uint8
}

// ServerOpcode implements ServerMessage.
func (m DropRemove) ServerOpcode() ServerOp { return ServerDropRemove }

// Encode implements ServerMessage.
func (m DropRemove) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.DropID)
	return w.Done()
}

// Balloon shows an emotion balloon over a character or event.
type Balloon struct {
	CharacterID   uint16
	CharacterType uint8
	BalloonID     uint8
}

// ServerOpcode implements ServerMessage.
func (m Balloon) ServerOpcode() ServerOp { return ServerBalloon }

// Encode implements ServerMessage.
func (m Balloon) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.CharacterID)
	w.U8(m.CharacterType)
	w.U8(m.BalloonID)
	return w.Done()
}

// EnemyRevive tells clients that an enemy is alive again.
type EnemyRevive struct {
	EventID uint16
}

// ServerOpcode implements ServerMessage.
func (m EnemyRevive) ServerOpcode() ServerOp { return ServerEnemyRevive }

// Encode implements ServerMessage.
func (m EnemyRevive) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.EventID)
	return w.Done()
}

// Animation plays an animation on a character.
type Animation struct {
	AttackerID    uint16
	AttackerType  uint8
	AniIndex      uint8
	CharacterID   uint16
	CharacterType uint8
	AnimationID   uint16
}

// ServerOpcode implements ServerMessage.
func (m Animation) ServerOpcode() ServerOp { return ServerAnimation }

// Encode implements ServerMessage.
func (m Animation) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.AttackerID)
	w.U8(m.AttackerType)
	w.U8(m.AniIndex)
	w.U16(m.CharacterID)
	w.U8(m.CharacterType)
	w.U16(m.AnimationID)
	return w.Done()
}

// Exp reports an experience delta.
type Exp struct {
	PlayerID uint16
	Amount   int32
}

// ServerOpcode implements ServerMessage.
func (m Exp) ServerOpcode() ServerOp { return ServerExp }

// Encode implements ServerMessage.
func (m Exp) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.PlayerID)
	w.I32(m.Amount)
	return w.Done()
}

// Transfer moves a player to another map position.
type Transfer struct {
	MapID     uint16
	X         uint16
	Y         uint16
	Direction uint8
}

// ServerOpcode implements ServerMessage.
func (m Transfer) ServerOpcode() ServerOp { return ServerTransfer }

// Encode implements ServerMessage.
func (m Transfer) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.MapID)
	w.U16(m.X)
	w.U16(m.Y)
	w.U8(m.Direction)
	return w.Done()
}

// EventCommand asks the client to execute a slice of an event list.
type EventCommand struct {
	EventID      int32
	InitialIndex uint16
	FinalIndex   uint16
}

// ServerOpcode implements ServerMessage.
func (m EventCommand) ServerOpcode() ServerOp { return ServerEventCommand }

// Encode implements ServerMessage.
func (m EventCommand) Encode() ([]byte, error) {
	w := NewWriter()
	w.I32(m.EventID)
	w.U16(m.InitialIndex)
	w.U16(m.FinalIndex)
	return w.Done()
}

// Switch reports a switch change.
type Switch struct {
	ID    uint16
	Value bool
}

// ServerOpcode implements ServerMessage.
func (m Switch) ServerOpcode() ServerOp { return ServerSwitch }

// Encode implements ServerMessage.
func (m Switch) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ID)
	w.Bool(m.Value)
	return w.Done()
}

// Variable reports a variable change.
type Variable struct {
	ID    uint16
	Value int32
}

// ServerOpcode implements ServerMessage.
func (m Variable) ServerOpcode() ServerOp { return ServerVariable }

// Encode implements ServerMessage.
func (m Variable) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ID)
	w.I32(m.Value)
	return w.Done()
}

// SelfSwitch reports one event self switch.
type SelfSwitch struct {
	MapID   uint16
	EventID uint16
	Ch      string
	Value   bool
}

// ServerOpcode implements ServerMessage.
func (m SelfSwitch) ServerOpcode() ServerOp { return ServerSelfSwitch }

// Encode implements ServerMessage.
func (m SelfSwitch) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.MapID)
	w.U16(m.EventID)
	w.String(m.Ch)
	w.Bool(m.Value)
	return w.Done()
}

// QuestAdd tells the client that a quest started (0-based quest id).
type QuestAdd struct {
	QuestID uint8
}

// ServerOpcode implements ServerMessage.
func (m QuestAdd) ServerOpcode() ServerOp { return ServerQuestAdd }

// Encode implements ServerMessage.
func (m QuestAdd) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.QuestID)
	return w.Done()
}

// QuestFinish tells the client that a quest finished (0-based quest id).
type QuestFinish struct {
	QuestID uint8
}

// ServerOpcode implements ServerMessage.
func (m QuestFinish) ServerOpcode() ServerOp { return ServerQuestFinish }

// Encode implements ServerMessage.
func (m QuestFinish) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.QuestID)
	return w.Done()
}

// VIPDays reports the account's VIP expiry as a unix timestamp.
type VIPDays struct {
	Until int64
}

// ServerOpcode implements ServerMessage.
func (m VIPDays) ServerOpcode() ServerOp { return ServerVIPDays }

// Encode implements ServerMessage.
func (m VIPDays) Encode() ([]byte, error) {
	w := NewWriter()
	w.I64(m.Until)
	return w.Done()
}

// Sex reports a player sex change.
type Sex struct {
	PlayerID uint16
	Sex      uint8
}

// ServerOpcode implements ServerMessage.
func (m Sex) ServerOpcode() ServerOp { return ServerSex }

// Encode implements ServerMessage.
func (m Sex) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.PlayerID)
	w.U8(m.Sex)
	return w.Done()
}

// Graphic reports a player graphic change.
type Graphic struct {
	PlayerID       uint16
	CharacterName  string
	CharacterIndex uint8
	FaceName       string
	FaceIndex      uint8
}

// ServerOpcode implements ServerMessage.
func (m Graphic) ServerOpcode() ServerOp { return ServerGraphic }

// Encode implements ServerMessage.
func (m Graphic) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.PlayerID)
	w.String(m.CharacterName)
	w.U8(m.CharacterIndex)
	w.String(m.FaceName)
	w.U8(m.FaceIndex)
	return w.Done()
}

// GlobalSwitches sends the server-wide switch block (100 values).
type GlobalSwitches struct {
	Values []bool
}

// ServerOpcode implements ServerMessage.
func (m GlobalSwitches) ServerOpcode() ServerOp { return ServerGlobalSwitches }

// Encode implements ServerMessage.
func (m GlobalSwitches) Encode() ([]byte, error) {
	w := NewWriter()
	for i := 0; i < 100; i++ {
		value := false
		if i < len(m.Values) {
			value = m.Values[i]
		}
		w.Bool(value)
	}
	return w.Done()
}

// BankStack is one bank entry.
type BankStack struct {
	Kind   uint8
	ItemID uint16
	Amount uint16
}

// BankOpen sends the full bank contents and opens the bank window.
type BankOpen struct {
	Gold   int32
	Stacks []BankStack
}

// ServerOpcode implements ServerMessage.
func (m BankOpen) ServerOpcode() ServerOp { return ServerBankOpen }

// Encode implements ServerMessage.
func (m BankOpen) Encode() ([]byte, error) {
	w := NewWriter()
	w.I32(m.Gold)
	counts := [4]int{}
	for _, stack := range m.Stacks {
		if stack.Kind >= 1 && stack.Kind <= 3 {
			counts[stack.Kind]++
		}
	}
	for kind := 1; kind <= 3; kind++ {
		count := counts[kind]
		if count > 255 {
			count = 255
		}
		w.U8(uint8(count))
		for _, stack := range m.Stacks {
			if stack.Kind == uint8(kind) {
				w.U16(stack.ItemID)
				w.U16(stack.Amount)
			}
		}
	}
	return w.Done()
}

// BankItem reports a bank stack change. Positive amounts were deposited.
type BankItem struct {
	ItemID uint16
	Kind   uint8
	Amount int16
}

// ServerOpcode implements ServerMessage.
func (m BankItem) ServerOpcode() ServerOp { return ServerBankItem }

// Encode implements ServerMessage.
func (m BankItem) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.ItemID)
	w.U8(m.Kind)
	w.I16(m.Amount)
	return w.Done()
}

// BankGold reports a bank gold change. Positive amounts were deposited.
type BankGold struct {
	Amount int32
}

// ServerOpcode implements ServerMessage.
func (m BankGold) ServerOpcode() ServerOp { return ServerBankGold }

// Encode implements ServerMessage.
func (m BankGold) Encode() ([]byte, error) {
	w := NewWriter()
	w.I32(m.Amount)
	return w.Done()
}

// Friends sends the online friends first.
type Friends struct {
	Online uint8
	Names  []string
}

// ServerOpcode implements ServerMessage.
func (m Friends) ServerOpcode() ServerOp { return ServerFriends }

// Encode implements ServerMessage.
func (m Friends) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Online)
	for _, name := range m.Names {
		w.String(name)
	}
	return w.Done()
}

// FriendAdd reports a new friend.
type FriendAdd struct {
	Name string
}

// ServerOpcode implements ServerMessage.
func (m FriendAdd) ServerOpcode() ServerOp { return ServerFriendAdd }

// Encode implements ServerMessage.
func (m FriendAdd) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Name)
	return w.Done()
}

// FriendRemove removes a friend list entry by index.
type FriendRemove struct {
	Index uint8
}

// ServerOpcode implements ServerMessage.
func (m FriendRemove) ServerOpcode() ServerOp { return ServerFriendRemove }

// Encode implements ServerMessage.
func (m FriendRemove) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Index)
	return w.Done()
}

// RequestAccepted tells both sides that a request was accepted.
type RequestAccepted struct {
	Type uint8
}

// ServerOpcode implements ServerMessage.
func (m RequestAccepted) ServerOpcode() ServerOp { return ServerRequestAccept }

// Encode implements ServerMessage.
func (m RequestAccepted) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Type)
	return w.Done()
}

// WindowClose tells the client to close the current server-driven window.
type WindowClose struct{}

// ServerOpcode implements ServerMessage.
func (m WindowClose) ServerOpcode() ServerOp { return ServerWindowClose }

// Encode implements ServerMessage.
func (m WindowClose) Encode() ([]byte, error) { return nil, nil }

// TradeItem reports an item offered by one trade participant.
type TradeItem struct {
	PlayerID uint16
	ItemID   uint16
	Kind     uint8
	Amount   int16
}

// ServerOpcode implements ServerMessage.
func (m TradeItem) ServerOpcode() ServerOp { return ServerTradeItem }

// Encode implements ServerMessage.
func (m TradeItem) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.PlayerID)
	w.U16(m.ItemID)
	w.U8(m.Kind)
	w.I16(m.Amount)
	return w.Done()
}

// TradeGold reports gold offered by one trade participant.
type TradeGold struct {
	PlayerID uint16
	Amount   int32
}

// ServerOpcode implements ServerMessage.
func (m TradeGold) ServerOpcode() ServerOp { return ServerTradeGold }

// Encode implements ServerMessage.
func (m TradeGold) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.PlayerID)
	w.I32(m.Amount)
	return w.Done()
}

// PartyJoin adds a member to the client's party panel.
type PartyJoin struct {
	PlayerID       uint16
	Name           string
	CharacterName  string
	CharacterIndex uint8
	Sex            uint8
	Equips         [MaxEquips]uint16
	MHP            int32
	HP             int32
	EXP            int32
}

// ServerOpcode implements ServerMessage.
func (m PartyJoin) ServerOpcode() ServerOp { return ServerPartyJoin }

// Encode implements ServerMessage.
func (m PartyJoin) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.PlayerID)
	w.String(m.Name)
	w.String(m.CharacterName)
	w.U8(m.CharacterIndex)
	w.U8(m.Sex)
	for _, id := range m.Equips {
		w.U16(id)
	}
	w.I32(m.MHP)
	w.I32(m.HP)
	w.I32(m.EXP)
	return w.Done()
}

// PartyLeave removes a member from the party panel.
type PartyLeave struct {
	PlayerID uint16
}

// ServerOpcode implements ServerMessage.
func (m PartyLeave) ServerOpcode() ServerOp { return ServerPartyLeave }

// Encode implements ServerMessage.
func (m PartyLeave) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.PlayerID)
	return w.Done()
}

// PartyDissolve clears the party panel.
type PartyDissolve struct{}

// ServerOpcode implements ServerMessage.
func (m PartyDissolve) ServerOpcode() ServerOp { return ServerPartyDissolve }

// Encode implements ServerMessage.
func (m PartyDissolve) Encode() ([]byte, error) { return nil, nil }

// GuildCreateOpen asks the client to show the guild creation window.
type GuildCreateOpen struct{}

// ServerOpcode implements ServerMessage.
func (m GuildCreateOpen) ServerOpcode() ServerOp { return ServerGuildCreateOpen }

// Encode implements ServerMessage.
func (m GuildCreateOpen) Encode() ([]byte, error) { return nil, nil }

// GuildOpen sends the guild window contents. Members are ordered online first.
type GuildOpen struct {
	Leader  string
	Notice  string
	Flag    []byte
	Members []string
	Online  int
}

// ServerOpcode implements ServerMessage.
func (m GuildOpen) ServerOpcode() ServerOp { return ServerGuildOpen }

// Encode implements ServerMessage.
func (m GuildOpen) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Leader)
	w.String(m.Notice)
	for i := 0; i < 64; i++ {
		value := byte(255)
		if i < len(m.Flag) {
			value = m.Flag[i]
		}
		w.U8(value)
	}
	w.U8(uint8(min(len(m.Members), 255)))
	w.U8(uint8(min(m.Online, 255)))
	for _, name := range m.Members {
		w.String(name)
	}
	return w.Done()
}

// GuildName reports a player's guild name (used for name tags).
type GuildName struct {
	PlayerID uint16
	Name     string
}

// ServerOpcode implements ServerMessage.
func (m GuildName) ServerOpcode() ServerOp { return ServerGuildName }

// Encode implements ServerMessage.
func (m GuildName) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Name)
	w.U16(m.PlayerID)
	return w.Done()
}

// GuildLeader reports a leadership change.
type GuildLeader struct {
	Name string
}

// ServerOpcode implements ServerMessage.
func (m GuildLeader) ServerOpcode() ServerOp { return ServerGuildLeader }

// Encode implements ServerMessage.
func (m GuildLeader) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Name)
	return w.Done()
}

// GuildNotice reports a notice change.
type GuildNotice struct {
	Notice string
}

// ServerOpcode implements ServerMessage.
func (m GuildNotice) ServerOpcode() ServerOp { return ServerGuildNotice }

// Encode implements ServerMessage.
func (m GuildNotice) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Notice)
	return w.Done()
}

// GuildMemberRemove reports a kicked/left member.
type GuildMemberRemove struct {
	Name string
}

// ServerOpcode implements ServerMessage.
func (m GuildMemberRemove) ServerOpcode() ServerOp { return ServerGuildMemberRemove }

// Encode implements ServerMessage.
func (m GuildMemberRemove) Encode() ([]byte, error) {
	w := NewWriter()
	w.String(m.Name)
	return w.Done()
}

// State reports a state being added or removed.
type State struct {
	StateID uint16
	Add     bool
}

// ServerOpcode implements ServerMessage.
func (m State) ServerOpcode() ServerOp { return ServerState }

// Encode implements ServerMessage.
func (m State) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.StateID)
	w.Bool(m.Add)
	return w.Done()
}

// Buff reports a buff level change (1 buff, -1 debuff, 0 erase).
type Buff struct {
	ParamID uint8
	Level   int16
}

// ServerOpcode implements ServerMessage.
func (m Buff) ServerOpcode() ServerOp { return ServerBuff }

// Encode implements ServerMessage.
func (m Buff) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.ParamID)
	w.I16(m.Level)
	return w.Done()
}

// PlayerRequest is an invitation shown in the target's choice window.
type PlayerRequest struct {
	Type   uint8
	FromID uint16
	Text   string
}

// ServerOpcode implements ServerMessage.
func (m PlayerRequest) ServerOpcode() ServerOp { return ServerRequest }

// Encode implements ServerMessage.
func (m PlayerRequest) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.Type)
	w.U16(m.FromID)
	w.String(m.Text)
	return w.Done()
}

// TeleportOpen opens a teleport destination list. ID is 0-based.
type TeleportOpen struct {
	ID uint8
}

// ServerOpcode implements ServerMessage.
func (m TeleportOpen) ServerOpcode() ServerOp { return ServerTeleportOpen }

// Encode implements ServerMessage.
func (m TeleportOpen) Encode() ([]byte, error) {
	w := NewWriter()
	w.U8(m.ID)
	return w.Done()
}

// ShopOpen asks the client to show a map event's shop goods.
type ShopOpen struct {
	EventID uint16
	Index   uint16
}

// ServerOpcode implements ServerMessage.
func (m ShopOpen) ServerOpcode() ServerOp { return ServerShopOpen }

// Encode implements ServerMessage.
func (m ShopOpen) Encode() ([]byte, error) {
	w := NewWriter()
	w.U16(m.EventID)
	w.U16(m.Index)
	return w.Done()
}

// LogoutRequest asks the server to leave the world.
type LogoutRequest struct{}

// ClientOpcode implements ClientMessage.
func (m LogoutRequest) ClientOpcode() ClientOp { return ClientLogout }

// Encode implements ClientMessage.
func (m LogoutRequest) Encode() ([]byte, error) { return nil, nil }

// Logout confirms leaving the world.
type Logout struct{}

// ServerOpcode implements ServerMessage.
func (m Logout) ServerOpcode() ServerOp { return ServerLogout }

// Encode implements ServerMessage.
func (m Logout) Encode() ([]byte, error) { return nil, nil }

// DecodeClient decodes a known client message.
func DecodeClient(op ClientOp, payload []byte) (any, error) {
	r := NewReader(payload)
	var m any
	switch op {
	case ClientPing:
		m = Ping{Timestamp: r.I64()}
	case ClientAuthBegin:
		m = AuthBegin{Username: r.String()}
	case ClientAuthProof:
		m = AuthProof{ClientNonce: r.Blob(), Proof: r.Blob()}
	case ClientRegisterBegin:
		m = RegisterBegin{Username: r.String(), Email: r.String()}
	case ClientRegisterProof:
		m = RegisterProof{ClientNonce: r.Blob(), Verifier: r.Blob(), Proof: r.Blob()}
	case ClientSelectCharacter:
		m = SelectCharacter{Slot: r.U8()}
	case ClientCreateCharacter:
		var c CreateCharacter
		c.Slot = r.U8()
		c.Name = r.String()
		c.CharacterIndex = r.U8()
		c.ClassID = r.U16()
		c.Sex = r.U8()
		for i := range c.Params {
			c.Params[i] = r.U8()
		}
		m = c
	case ClientMove:
		m = Move{Direction: r.U8()}
	case ClientChat:
		m = Chat{Channel: r.U8(), Message: r.String(), Target: r.String()}
	case ClientEquip:
		m = EquipItem{Slot: r.U8(), ItemID: r.U16()}
	case ClientSetHotbar:
		m = SetHotbar{Slot: r.U8(), Type: r.U8(), ItemID: r.U16()}
	case ClientSpendPoint:
		m = SpendPoint{ParamID: r.U8()}
	case ClientAttack:
		m = Attack{}
	case ClientSetTarget:
		m = Target{Type: r.U8(), ID: r.U16()}
	case ClientUseItem:
		m = UseItem{ItemID: r.U16()}
	case ClientUseSkill:
		m = UseSkill{SkillID: r.U16()}
	case ClientUseHotbar:
		m = UseHotbar{Slot: r.U8()}
	case ClientChoice:
		m = Choice{Index: r.I32()}
	case ClientChoiceTeleport:
		m = ChoiceTeleport{Index: r.U8()}
	case ClientShopBuy:
		m = ShopBuy{Index: r.U8(), Amount: r.U16()}
	case ClientShopSell:
		m = ShopSell{ItemID: r.U16(), Kind: r.U8(), Amount: r.U16()}
	case ClientBankItem:
		m = BankItemRequest{ItemID: r.U16(), Kind: r.U8(), Amount: r.I16()}
	case ClientBankGold:
		m = BankGoldRequest{Amount: r.I32()}
	case ClientCloseWindow:
		m = CloseWindow{}
	case ClientEventNext:
		m = EventNext{}
	case ClientDropItem:
		m = DropItemRequest{ItemID: r.U16(), Kind: r.U8(), Amount: r.U16()}
	case ClientPickupDrop:
		m = PickupDrop{DropID: r.U8()}
	case ClientAdminCommand:
		m = AdminCommand{Command: r.U8(), Text: r.String(), Int1: r.I32(), Int2: r.I32(), Int3: r.I16()}
	case ClientDeleteCharacter:
		m = DeleteCharacter{Slot: r.U8(), ClientNonce: r.Blob(), Proof: r.Blob()}
	case ClientLogout:
		m = LogoutRequest{}
	case ClientGuildCreate:
		obj := GuildCreate{Name: r.String()}
		obj.Flag = make([]byte, 64)
		for i := range obj.Flag {
			obj.Flag[i] = r.U8()
		}
		m = obj
	case ClientGuildOpen:
		m = GuildOpenRequest{}
	case ClientGuildSetLeader:
		m = GuildSetLeader{Name: r.String()}
	case ClientGuildSetNotice:
		m = GuildSetNotice{Notice: r.String()}
	case ClientGuildKick:
		m = GuildKick{Name: r.String()}
	case ClientGuildInvite:
		m = GuildInvite{Name: r.String()}
	case ClientGuildLeave:
		m = GuildLeave{}
	case ClientRequestAccept:
		m = RequestAccept{}
	case ClientRequestDecline:
		m = RequestDecline{}
	case ClientRequest:
		m = RequestInvite{Type: r.U8(), PlayerID: r.U16()}
	case ClientPartyLeave:
		m = PartyLeaveRequest{}
	case ClientOpenFriends:
		m = OpenFriends{}
	case ClientRemoveFriend:
		m = RemoveFriend{Index: r.U8()}
	case ClientTradeItem:
		m = TradeItemOffer{ItemID: r.U16(), Kind: r.U8(), Amount: r.I16()}
	case ClientTradeGold:
		m = TradeGoldOffer{Amount: r.I32()}
	default:
		return nil, ErrUnknownOpcode
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	if err := validateClientMessage(m); err != nil {
		return nil, err
	}
	return m, nil
}

// DecodeServer decodes a known server message. It exists for the headless Go
// client and for tests.
func DecodeServer(op ServerOp, payload []byte) (any, error) {
	r := NewReader(payload)
	var m any
	switch op {
	case ServerPong:
		m = Pong{Timestamp: r.I64()}
	case ServerAuthChallenge:
		m = AuthChallenge{Salt: r.Blob(), KDF: r.U8(), Iterations: r.U32(), Nonce: r.Blob(), ServerTime: r.I64()}
	case ServerAuthOK:
		var ok AuthOK
		ok.Group = r.U8()
		ok.VIPUntil = r.I64()
		n := r.Count()
		if n > MaxActorsInList {
			return nil, fmt.Errorf("protocol: actor count %d exceeds limit %d", n, MaxActorsInList)
		}
		for i := 0; i < n; i++ {
			ok.Actors = append(ok.Actors, readActorSummary(r))
		}
		m = ok
	case ServerAuthError:
		m = AuthError{Code: r.U16(), Message: r.String()}
	case ServerCharacterCreated:
		m = CharacterCreated{Actor: readActorSummary(r)}
	case ServerCharacterError:
		m = CharacterError{Code: r.U16(), Message: r.String()}
	case ServerRegisterChallenge:
		m = RegisterChallenge{Salt: r.Blob(), KDF: r.U8(), Iterations: r.U32(), Nonce: r.Blob(), ServerTime: r.I64()}
	case ServerRegisterOK:
		m = RegisterOK{}
	case ServerRegisterError:
		m = RegisterError{Code: r.U16(), Message: r.String()}
	case ServerError:
		m = ErrorMessage{Code: r.U16(), Message: r.String()}
	case ServerDisconnect:
		m = Disconnect{Code: r.U16(), Message: r.String()}
	case ServerEnterWorld:
		m = readEnterWorld(r)
	case ServerPlayerData:
		m = PlayerData{Player: readPlayerSnapshot(r)}
	case ServerPlayerMove:
		m = PlayerMove{ID: r.U16(), X: r.U16(), Y: r.U16(), Direction: r.U8()}
	case ServerPlayerLeft:
		m = PlayerLeft{ID: r.U16()}
	case ServerMapChat:
		m = MapChat{PlayerID: r.U16(), Color: r.U8(), Message: r.String()}
	case ServerChat:
		m = ChatMessage{Color: r.U8(), Message: r.String()}
	case ServerEquip:
		m = Equip{PlayerID: r.U16(), Slot: r.U8(), ItemID: r.U16()}
	case ServerVitals:
		m = Vitals{PlayerID: r.U16(), HP: r.I32(), MP: r.I32()}
	case ServerHotbar:
		m = Hotbar{Slot: r.U8(), Type: r.U8(), ItemID: r.U16()}
	case ServerParam:
		m = Param{PlayerID: r.U16(), ParamID: r.U8(), Value: r.I32()}
	case ServerPoints:
		m = Points{Value: r.I32()}
	case ServerItem:
		m = Item{ItemID: r.U16(), Kind: r.U8(), Amount: r.I32(), DropSound: r.Bool(), Popup: r.Bool()}
	case ServerGold:
		m = Gold{Amount: r.I64(), ShopSound: r.Bool(), Popup: r.Bool()}
	case ServerSkill:
		m = Skill{SkillID: r.U16(), Learned: r.Bool()}
	case ServerAttackEnemy:
		m = AttackEnemy{AttackerID: r.U16(), AttackerType: r.U8(), AniIndex: r.U8(), EventID: r.U16(), HPDamage: r.I32(), MPDamage: r.I32(), Critical: r.Bool(), AnimationID: r.U16()}
	case ServerAttackPlayer:
		m = AttackPlayer{AttackerID: r.U16(), AttackerType: r.U8(), AniIndex: r.U8(), PlayerID: r.U16(), HPDamage: r.I32(), MPDamage: r.I32(), Critical: r.Bool(), AnimationID: r.U16(), NotShowMissed: r.Bool()}
	case ServerEventData:
		m = EventData{EventID: r.U16(), X: r.U16(), Y: r.U16(), Direction: r.U8(), HP: r.I32()}
	case ServerEventMove:
		m = EventMove{EventID: r.U16(), X: r.U16(), Y: r.U16(), Direction: r.U8()}
	case ServerEnemyRevive:
		m = EnemyRevive{EventID: r.U16()}
	case ServerBalloon:
		m = Balloon{CharacterID: r.U16(), CharacterType: r.U8(), BalloonID: r.U8()}
	case ServerDropAdd:
		m = DropAdd{ItemID: r.U16(), Kind: r.U8(), Amount: r.U16(), X: r.U16(), Y: r.U16()}
	case ServerDropRemove:
		m = DropRemove{DropID: r.U8()}
	case ServerAdminCommand:
		m = AdminCommandNotice{Command: r.U8(), Message: r.String()}
	case ServerProjectile:
		m = Projectile{
			StartX: r.U16(), StartY: r.U16(), FinishX: r.U16(), FinishY: r.U16(),
			TargetX: r.U16(), TargetY: r.U16(), ProjectileType: r.U8(), ProjectileID: r.U8(),
		}
	case ServerAnimation:
		m = Animation{AttackerID: r.U16(), AttackerType: r.U8(), AniIndex: r.U8(), CharacterID: r.U16(), CharacterType: r.U8(), AnimationID: r.U16()}
	case ServerExp:
		m = Exp{PlayerID: r.U16(), Amount: r.I32()}
	case ServerTransfer:
		m = Transfer{MapID: r.U16(), X: r.U16(), Y: r.U16(), Direction: r.U8()}
	case ServerEventCommand:
		m = EventCommand{EventID: r.I32(), InitialIndex: r.U16(), FinalIndex: r.U16()}
	case ServerSwitch:
		m = Switch{ID: r.U16(), Value: r.Bool()}
	case ServerVariable:
		m = Variable{ID: r.U16(), Value: r.I32()}
	case ServerSelfSwitch:
		m = SelfSwitch{MapID: r.U16(), EventID: r.U16(), Ch: r.String(), Value: r.Bool()}
	case ServerGlobalSwitches:
		values := make([]bool, 100)
		for i := range values {
			values[i] = r.Bool()
		}
		m = GlobalSwitches{Values: values}
	case ServerQuestAdd:
		m = QuestAdd{QuestID: r.U8()}
	case ServerQuestFinish:
		m = QuestFinish{QuestID: r.U8()}
	case ServerVIPDays:
		m = VIPDays{Until: r.I64()}
	case ServerSex:
		m = Sex{PlayerID: r.U16(), Sex: r.U8()}
	case ServerGraphic:
		m = Graphic{PlayerID: r.U16(), CharacterName: r.String(), CharacterIndex: r.U8(), FaceName: r.String(), FaceIndex: r.U8()}
	case ServerTeleportOpen:
		m = TeleportOpen{ID: r.U8()}
	case ServerGuildCreateOpen:
		m = GuildCreateOpen{}
	case ServerGuildOpen:
		open := GuildOpen{Leader: r.String(), Notice: r.String()}
		open.Flag = make([]byte, 64)
		for i := range open.Flag {
			open.Flag[i] = r.U8()
		}
		count := r.U8()
		open.Online = int(r.U8())
		for i := 0; i < int(count); i++ {
			open.Members = append(open.Members, r.String())
		}
		m = open
	case ServerGuildName:
		m = GuildName{Name: r.String(), PlayerID: r.U16()}
	case ServerGuildLeader:
		m = GuildLeader{Name: r.String()}
	case ServerGuildNotice:
		m = GuildNotice{Notice: r.String()}
	case ServerGuildMemberRemove:
		m = GuildMemberRemove{Name: r.String()}
	case ServerRequest:
		m = PlayerRequest{Type: r.U8(), FromID: r.U16(), Text: r.String()}
	case ServerPartyJoin:
		join := PartyJoin{PlayerID: r.U16(), Name: r.String(), CharacterName: r.String(), CharacterIndex: r.U8(), Sex: r.U8()}
		for i := range join.Equips {
			join.Equips[i] = r.U16()
		}
		join.MHP = r.I32()
		join.HP = r.I32()
		join.EXP = r.I32()
		m = join
	case ServerPartyLeave:
		m = PartyLeave{PlayerID: r.U16()}
	case ServerPartyDissolve:
		m = PartyDissolve{}
	case ServerFriends:
		friends := Friends{Online: r.U8()}
		for i := 0; i < int(friends.Online); i++ {
			friends.Names = append(friends.Names, r.String())
		}
		m = friends
	case ServerFriendAdd:
		m = FriendAdd{Name: r.String()}
	case ServerFriendRemove:
		m = FriendRemove{Index: r.U8()}
	case ServerRequestAccept:
		m = RequestAccepted{Type: r.U8()}
	case ServerWindowClose:
		m = WindowClose{}
	case ServerTradeItem:
		m = TradeItem{PlayerID: r.U16(), ItemID: r.U16(), Kind: r.U8(), Amount: r.I16()}
	case ServerTradeGold:
		m = TradeGold{PlayerID: r.U16(), Amount: r.I32()}
	case ServerCharacterDeleted:
		m = CharacterDeleted{Slot: r.U8()}
	case ServerState:
		m = State{StateID: r.U16(), Add: r.Bool()}
	case ServerBuff:
		m = Buff{ParamID: r.U8(), Level: r.I16()}
	case ServerShopOpen:
		m = ShopOpen{EventID: r.U16(), Index: r.U16()}
	case ServerBankGold:
		m = BankGold{Amount: r.I32()}
	case ServerBankItem:
		m = BankItem{ItemID: r.U16(), Kind: r.U8(), Amount: r.I16()}
	case ServerBankOpen:
		open := BankOpen{Gold: r.I32()}
		for kind := 1; kind <= 3; kind++ {
			count := r.U8()
			for i := 0; i < int(count); i++ {
				open.Stacks = append(open.Stacks, BankStack{
					Kind: uint8(kind), ItemID: r.U16(), Amount: r.U16(),
				})
			}
		}
		m = open
	case ServerLogout:
		m = Logout{}
	default:
		return nil, ErrUnknownOpcode
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

func validateClientMessage(m any) error {
	switch x := m.(type) {
	case AuthBegin:
		if len(x.Username) == 0 || len(x.Username) > 64 {
			return fmt.Errorf("protocol: username length %d out of range", len(x.Username))
		}
	case AuthProof:
		if len(x.ClientNonce) < 8 || len(x.ClientNonce) > 64 {
			return fmt.Errorf("protocol: client nonce length %d out of range", len(x.ClientNonce))
		}
		if len(x.Proof) < 16 || len(x.Proof) > 128 {
			return fmt.Errorf("protocol: proof length %d out of range", len(x.Proof))
		}
	case RegisterBegin:
		if len(x.Username) == 0 || len(x.Username) > 64 {
			return fmt.Errorf("protocol: username length %d out of range", len(x.Username))
		}
		if len(x.Email) > 255 {
			return fmt.Errorf("protocol: email too long")
		}
	case RegisterProof:
		if len(x.Verifier) < 16 || len(x.Verifier) > 128 {
			return fmt.Errorf("protocol: verifier length %d out of range", len(x.Verifier))
		}
		if len(x.Proof) < 16 || len(x.Proof) > 128 {
			return fmt.Errorf("protocol: proof length %d out of range", len(x.Proof))
		}
	case CreateCharacter:
		if len(x.Name) == 0 || len(x.Name) > 64 {
			return fmt.Errorf("protocol: character name length %d out of range", len(x.Name))
		}
		if x.Sex > 1 {
			return fmt.Errorf("protocol: invalid sex %d", x.Sex)
		}
		if x.ClassID == 0 {
			return fmt.Errorf("protocol: invalid class id")
		}
	case EquipItem:
		if x.Slot >= MaxEquips {
			return fmt.Errorf("protocol: equip slot %d out of range", x.Slot)
		}
	case SetHotbar:
		if x.Slot >= 64 {
			return fmt.Errorf("protocol: hotbar slot %d out of range", x.Slot)
		}
		if x.Type > 2 {
			return fmt.Errorf("protocol: invalid hotbar type %d", x.Type)
		}
	case SpendPoint:
		if x.ParamID > 7 {
			return fmt.Errorf("protocol: invalid param id %d", x.ParamID)
		}
	case Target:
		if x.Type > 2 {
			return fmt.Errorf("protocol: invalid target type %d", x.Type)
		}
	case UseHotbar:
		if x.Slot >= 64 {
			return fmt.Errorf("protocol: hotbar slot %d out of range", x.Slot)
		}
	case Choice:
		if x.Index < 0 || x.Index > 99999999 {
			return fmt.Errorf("protocol: choice index %d out of range", x.Index)
		}
	case Chat:
		if len(x.Message) == 0 || len(x.Message) > MaxChatBytes {
			return fmt.Errorf("protocol: chat length %d out of range", len(x.Message))
		}
		if len(x.Target) > 64 {
			return fmt.Errorf("protocol: chat target too long")
		}
	case Move:
		if x.Direction < 1 || x.Direction > 9 || x.Direction == 5 {
			return fmt.Errorf("protocol: invalid direction %d", x.Direction)
		}
	}
	return nil
}
