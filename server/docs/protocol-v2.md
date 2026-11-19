# Protocol v2

The server and the patched client use a new, symmetric binary protocol. There is
no compatibility layer for the old v1 wire format.

## Why v2

The old protocol had:

- a length prefix on server→client messages and no length prefix on
  client→server messages;
- a client receive loop that assumed the two-byte length arrived in one read;
- a four-byte padding hack in `SocketLib#recv_non_block`;
- one shared opcode enum for both directions;
- `int16` variables that truncated values accepted by the event editor.

v2 frames both directions, separates opcodes by direction and uses `i32` for
game quantities.

## Connection handshake

The handshake is raw, before framing:

```
client -> server:
  magic          8 bytes  "VXAOS2\r\n"
  protocol       u16      ProtocolVersion (1)
  game version   u16      Configs::GAME_VERSION
  build          string   u16 length + UTF-8

server -> client:
  magic          8 bytes  "VXAOS2\r\n"
  protocol       u16
  game version   u16
  server time    i64      Unix seconds
  max frame      u32      server limit (default 1 MiB)
  status         u8       0 ok, 1 version, 2 full, 3 maintenance, 4 banned
  message        string
```

A wrong magic is rejected immediately. A version mismatch returns a status
message and closes the connection.

## Frames

After the handshake, both directions use the same frame:

```
u32 length       includes the two-byte opcode
u16 opcode       direction-specific
... payload
```

`length` must be at least 2 and at most `max frame` (1 MiB by default). The
reader uses `io.ReadFull` for the header and the payload, so TCP fragmentation
is handled correctly.

## Field encoding

All integers are little-endian.

| Type | Encoding |
|---|---|
| `u8` | one byte |
| `bool` | one byte, 0/1 |
| `u16` | two bytes |
| `u32` | four bytes |
| `i32` | four bytes, two's complement |
| `i64` | eight bytes, two's complement |
| `string` | `u16` byte length + UTF-8; invalid UTF-8 is rejected |
| `blob` | `u16` byte length + bytes |
| `array` | `u16` count followed by items |

`i32` is used for HP, MP, EXP, gold, parameters, variables and points, which
removes the `short` overflow of the old protocol.

## Authentication

PBKDF2-HMAC-SHA256 challenge-response. No TLS in this phase.

```
client -> AUTH_BEGIN { username }
server -> AUTH_CHALLENGE { salt, kdf=1, iterations, nonce, server_time }
client -> AUTH_PROOF { client_nonce, proof }
server -> AUTH_OK { group, vip_until, actors[] }
       or AUTH_ERROR { code, message }
```

Definitions:

```
verifier = PBKDF2-HMAC-SHA256(password, salt, iterations)
proof    = HMAC-SHA256(verifier, "login" || username || nonce || client_nonce)
```

- `salt`, `nonce` and `client_nonce` are random byte strings.
- The server stores `salt`, `kdf`, `iterations` and `verifier`; it never stores
  the password.
- Unknown usernames receive a random fake challenge and fail after the proof, so
  the protocol does not enumerate accounts.
- The stored verifier is password-equivalent for authentication. A leaked DB
  lets an attacker authenticate without cracking the password, but does not
  reveal the password. SRP/OPAQUE or TLS is the future fix.
- Registration needs to send the verifier to the server. Over plaintext this
  exposes it to a passive eavesdropper; treat registration as trusted-network
  only until SRP/TLS, or create accounts from the server CLI.

## Direction-specific opcodes

The old shared `Enums::Packet` is gone. `ClientOp` and `ServerOp` are separate
types with separate numeric values. The full enum is in
`internal/protocol/opcodes.go`.

Implemented now:

| Client | Server |
|---|---|
| `PING` | `PONG` |
| `AUTH_BEGIN` | `AUTH_CHALLENGE` |
| `AUTH_PROOF` | `AUTH_OK`, `AUTH_ERROR` |
| `REGISTER_BEGIN` | `REGISTER_CHALLENGE` |
| `REGISTER_PROOF` | `REGISTER_OK`, `REGISTER_ERROR` |
| `SELECT_CHARACTER` | `ENTER_WORLD` (reserved) |
| `MOVE` | `PLAYER_MOVE` (reserved) |
| `CHAT` | `MAP_CHAT`, `CHAT` (reserved) |
| `ATTACK`, `USE_ITEM`, `USE_SKILL`, ... | `ATTACK_PLAYER`, `ATTACK_ENEMY`, ... (reserved) |

Reserved opcodes are declared so both sides can code against the same names;
their payloads will be added as the corresponding game features are ported.

## Client patch checklist

1. `[VS] Buffer`: replace with the v2 codec (or a generated equivalent).
2. `[VS] Socket`: send raw handshake bytes; fix `recv_non_block` to return
   exactly `r_len` bytes; remove the `maxlen+4` padding path.
3. `[VS] Network`: buffered frame reader, handshake state machine, no
   `message.size - 4` heuristics.
4. `[VS] Send_Data` / `[VS] Handle_Data`: direction-specific opcodes and field
   widths; no `write_short` for variables.
5. `[VS] Configs` / `[VS] Quests` / `[VS] Vocab` / `[VS] Enums`: generated from
   `content/`.
6. Auth: PBKDF2/HMAC in `VXAOS.dll` (or a pure-Ruby fallback) and the
   challenge-response flow in `Network`.
7. Damage formulas: override `RPG::UsableItem::Damage#eval` to return 0; remove
   the cosmetic “Dano base” line. The server is the only damage authority.
