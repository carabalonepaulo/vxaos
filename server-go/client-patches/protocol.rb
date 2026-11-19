#==============================================================================
# ** Protocol
#------------------------------------------------------------------------------
#  Constantes e framing do protocolo v2. Mantido em sincronia com
#  internal/protocol/opcodes.go.
#==============================================================================

module Protocol

  VERSION   = 1
  MAGIC     = "VXAOS2\r\n"
  MAX_FRAME = 1 << 20

  module Client
    PING             = 1
    AUTH_BEGIN       = 2
    AUTH_PROOF       = 3
    REGISTER_BEGIN   = 4
    REGISTER_PROOF   = 5
    SELECT_CHARACTER = 6
    CREATE_CHARACTER = 7
    DELETE_CHARACTER = 8
    MOVE             = 9
    CHAT             = 10
    ATTACK           = 11
    USE_ITEM         = 12
    USE_SKILL        = 13
    USE_HOTBAR       = 14
    BALLOON          = 15
    DROP_ITEM        = 16
    PICKUP_DROP      = 17
    SPEND_POINT      = 18
    EQUIP            = 19
    SET_HOTBAR       = 20
    SET_TARGET       = 21
    OPEN_FRIENDS     = 22
    REMOVE_FRIEND    = 23
    GUILD_CREATE     = 24
    GUILD_OPEN       = 25
    GUILD_SET_LEADER = 26
    GUILD_SET_NOTICE = 27
    GUILD_KICK       = 28
    GUILD_INVITE     = 29
    GUILD_LEAVE      = 30
    PARTY_LEAVE      = 31
    CHOICE           = 32
    BANK_ITEM        = 33
    BANK_GOLD        = 34
    CLOSE_WINDOW     = 35
    SHOP_BUY         = 36
    SHOP_SELL        = 37
    CHOICE_TELEPORT  = 38
    EVENT_NEXT       = 39
    EVENT_CHOICE     = 40
    REQUEST          = 41
    REQUEST_ACCEPT   = 42
    REQUEST_DECLINE  = 43
    TRADE_ITEM       = 44
    TRADE_GOLD       = 45
    LOGOUT           = 46
    ADMIN_COMMAND    = 47
  end

  module Server
    PONG                = 1
    AUTH_CHALLENGE      = 2
    AUTH_OK             = 3
    AUTH_ERROR          = 4
    REGISTER_CHALLENGE  = 5
    REGISTER_OK         = 6
    REGISTER_ERROR      = 7
    CHARACTER_CREATED   = 8
    CHARACTER_DELETED   = 9
    CHARACTER_ERROR     = 10
    ENTER_WORLD         = 11
    PLAYER_DATA         = 12
    PLAYER_LEFT         = 13
    PLAYER_MOVE         = 14
    MAP_CHAT            = 15
    CHAT                = 16
    ALERT               = 17
    ATTACK_PLAYER       = 18
    ATTACK_ENEMY        = 19
    ANIMATION           = 20
    BALLOON             = 21
    ENEMY_REVIVE        = 22
    EVENT_DATA          = 23
    EVENT_MOVE          = 24
    DROP_ADD            = 25
    DROP_REMOVE         = 26
    PROJECTILE          = 27
    VITALS              = 28
    EXP                 = 29
    STATE               = 30
    BUFF                = 31
    ITEM                = 32
    GOLD                = 33
    PARAM               = 34
    EQUIP               = 35
    SKILL               = 36
    CLASS               = 37
    SEX                 = 38
    GRAPHIC             = 39
    POINTS              = 40
    HOTBAR              = 41
    TARGET              = 42
    TRANSFER            = 43
    FRIENDS             = 44
    FRIEND_ADD          = 45
    FRIEND_REMOVE       = 46
    GUILD_OPEN          = 47
    GUILD_NAME          = 48
    GUILD_LEADER        = 49
    GUILD_NOTICE        = 50
    GUILD_MEMBER_REMOVE = 51
    PARTY_JOIN          = 52
    PARTY_LEAVE         = 53
    PARTY_DISSOLVE      = 54
    BANK_OPEN           = 55
    BANK_ITEM           = 56
    BANK_GOLD           = 57
    WINDOW_CLOSE        = 58
    SHOP_OPEN           = 59
    TELEPORT_OPEN       = 60
    EVENT_COMMAND       = 61
    REQUEST             = 62
    REQUEST_ACCEPT      = 63
    TRADE_ITEM          = 64
    TRADE_GOLD          = 65
    QUEST_ADD           = 66
    QUEST_FINISH        = 67
    VIP_DAYS            = 68
    LOGOUT              = 69
    ADMIN_COMMAND       = 70
    SWITCH              = 71
    VARIABLE            = 72
    SELF_SWITCH         = 73
    GLOBAL_SWITCHES     = 74
    ERROR               = 75
    DISCONNECT          = 76
    GUILD_CREATE_OPEN   = 77
  end

end
