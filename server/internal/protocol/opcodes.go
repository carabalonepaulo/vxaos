package protocol

// ClientOp identifies a message sent by the client. Values are independent of
// the server opcodes: the legacy shared enum is gone.
type ClientOp uint16

const (
	ClientPing ClientOp = 1 + iota
	ClientAuthBegin
	ClientAuthProof
	ClientRegisterBegin
	ClientRegisterProof
	ClientSelectCharacter
	ClientCreateCharacter
	ClientDeleteCharacter
	ClientMove
	ClientChat
	ClientAttack
	ClientUseItem
	ClientUseSkill
	ClientUseHotbar
	ClientBalloon
	ClientDropItem
	ClientPickupDrop
	ClientSpendPoint
	ClientEquip
	ClientSetHotbar
	ClientSetTarget
	ClientOpenFriends
	ClientRemoveFriend
	ClientGuildCreate
	ClientGuildOpen
	ClientGuildSetLeader
	ClientGuildSetNotice
	ClientGuildKick
	ClientGuildInvite
	ClientGuildLeave
	ClientPartyLeave
	ClientChoice
	ClientBankItem
	ClientBankGold
	ClientCloseWindow
	ClientShopBuy
	ClientShopSell
	ClientChoiceTeleport
	ClientEventNext
	ClientEventChoice
	ClientRequest
	ClientRequestAccept
	ClientRequestDecline
	ClientTradeItem
	ClientTradeGold
	ClientLogout
	ClientAdminCommand
)

// ServerOp identifies a message sent by the server.
type ServerOp uint16

const (
	ServerPong ServerOp = 1 + iota
	ServerAuthChallenge
	ServerAuthOK
	ServerAuthError
	ServerRegisterChallenge
	ServerRegisterOK
	ServerRegisterError
	ServerCharacterCreated
	ServerCharacterDeleted
	ServerCharacterError
	ServerEnterWorld
	ServerPlayerData
	ServerPlayerLeft
	ServerPlayerMove
	ServerMapChat
	ServerChat
	ServerAlert
	ServerAttackPlayer
	ServerAttackEnemy
	ServerAnimation
	ServerBalloon
	ServerEnemyRevive
	ServerEventData
	ServerEventMove
	ServerDropAdd
	ServerDropRemove
	ServerProjectile
	ServerVitals
	ServerExp
	ServerState
	ServerBuff
	ServerItem
	ServerGold
	ServerParam
	ServerEquip
	ServerSkill
	ServerClass
	ServerSex
	ServerGraphic
	ServerPoints
	ServerHotbar
	ServerTarget
	ServerTransfer
	ServerFriends
	ServerFriendAdd
	ServerFriendRemove
	ServerGuildOpen
	ServerGuildName
	ServerGuildLeader
	ServerGuildNotice
	ServerGuildMemberRemove
	ServerPartyJoin
	ServerPartyLeave
	ServerPartyDissolve
	ServerBankOpen
	ServerBankItem
	ServerBankGold
	ServerWindowClose
	ServerShopOpen
	ServerTeleportOpen
	ServerEventCommand
	ServerRequest
	ServerRequestAccept
	ServerTradeItem
	ServerTradeGold
	ServerQuestAdd
	ServerQuestFinish
	ServerVIPDays
	ServerLogout
	ServerAdminCommand
	ServerSwitch
	ServerVariable
	ServerSelfSwitch
	ServerGlobalSwitches
	ServerError
	ServerDisconnect
	ServerGuildCreateOpen
	ServerMapReset
	ServerEventRemove
	ServerMapPlayerEnter
	ServerMapPlayerLeave
)

// Auth and error codes are stable protocol values.
const (
	AuthInvalidCredentials uint16 = 1
	AuthAccountBanned      uint16 = 2
	AuthAlreadyOnline      uint16 = 3
	AuthServerFull         uint16 = 4
	AuthInactivity         uint16 = 5
	AuthOldVersion         uint16 = 6
	AuthInternal           uint16 = 7
	AuthAccountExists      uint16 = 8
)

// Error codes for ServerError.
const (
	CodeBadRequest       uint16 = 1
	CodeUnknownOpcode    uint16 = 2
	CodeNotAuthenticated uint16 = 3
	CodeNotInGame        uint16 = 4
	CodeInvalidState     uint16 = 5
	CodeRateLimited      uint16 = 6
	CodeInvalidTarget    uint16 = 7
	CodeFullInventory    uint16 = 8
	CodeNotEnoughGold    uint16 = 9
	CodeInternal         uint16 = 10
	CodeCharacterName    uint16 = 11
	CodeCharacterSlot    uint16 = 12
	CodeCharacterInvalid uint16 = 13
)
