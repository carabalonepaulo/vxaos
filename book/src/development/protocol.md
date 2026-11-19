# Protocol v2

VXA-OS uses a symmetric binary protocol. The patched client and the server must
use the same version. There is no v1 compatibility layer.

The implementation is `server/internal/protocol`; the client equivalent is
`server/client-patches/protocol.rb` (plus buffer/socket/network patches).

## Handshake

Raw bytes, before framing:

```text
client -> server
  magic         8 bytes  "VXAOS2\r\n"
  protocol      u16      1
  game version  u16      Configs::GAME_VERSION
  build         string

server -> client
  magic         8 bytes  "VXAOS2\r\n"
  protocol      u16
  game version  u16
  server time   i64
  max frame     u32
  status        u8       0 ok, 1 version, 2 full, 3 maintenance, 4 banned
  message       string
```

Wrong magic is rejected. A version mismatch returns a status and closes.

## Frames

After the handshake:

```text
u32 length    includes the 2-byte opcode
u16 opcode    direction-specific
... payload
```

`length` must be 2..`max frame` (1 MiB by default). Both directions use
`io.ReadFull`; TCP fragmentation is handled.

## Encoding

Little-endian:

| Type | Encoding |
|---|---|
| `u8` | 1 byte |
| `bool` | 1 byte 0/1 |
| `u16` | 2 bytes |
| `u32` | 4 bytes |
| `i32` | 4 bytes |
| `i64` | 8 bytes |
| `string` | `u16` length + UTF-8 |
| `blob` | `u16` length + bytes |
| `array` | `u16` count + items |

HP/MP/EXP/gold/params/variables/points use `i32` to avoid the old `int16`
overflow.

## Auth

```text
AUTH_BEGIN { username }
AUTH_CHALLENGE { salt, kdf, iterations, nonce, server_time }
AUTH_PROOF { client_nonce, proof }
AUTH_OK { group, vip_until, actors[] } | AUTH_ERROR { code, message }
```

```text
verifier = PBKDF2-HMAC-SHA256(password, salt, iterations)
proof    = HMAC-SHA256(verifier, "login" || username || nonce || client_nonce)
```

Unknown users receive a fake challenge. There is no TLS; use a trusted network,
VPN or TLS proxy. Registration is disabled in this build; create accounts with
`vxaos-admin`.

## Opcodes

`ClientOp` and `ServerOp` are separate enums. The full list is
`internal/protocol/opcodes.go`; the client copy is
`client-patches/protocol.rb`; `content/enums.toml` has the same table for
parity tests.

Rules:

- never renumber an existing opcode in a release line;
- add at the end of the direction;
- update Go, Ruby and TOML together;
- update `content/enums_test.go` hardcoded counts when adding ops;
- keep payload decode/encode symmetric and add a round-trip test.

AOI-specific ops:

| Op | Direction | Purpose |
|---|---|---|
| `MAP_RESET` | server | clear client AOI state before a snapshot |
| `EVENT_REMOVE` | server | remove an NPC/enemy sprite |
| `MAP_PLAYER_ENTER` | server | logical map membership |
| `MAP_PLAYER_LEAVE` | server | logical map membership |

`PLAYER_DATA`/`PLAYER_LEFT` are sprite-level AOI add/remove, not logical map
membership.

Drops use stable `uint32` ids in `DROP_ADD`, `DROP_REMOVE` and `PICKUP_DROP`.

## Adding a message

1. Add the opcode at the end of `ClientOp`/`ServerOp`.
2. Add the message struct with `Encode` and the right opcode method.
3. Add the `DecodeClient`/`DecodeServer` case.
4. Add the Ruby constant and handler.
5. Update `content/enums.toml`.
6. Add a protocol round-trip test.
7. Add the server dispatch and client handler.
8. Run `go test ./...` and the client smoke test.

## Error codes

`ServerError` uses codes such as bad request, unknown opcode, not authenticated,
not in game, invalid state, rate limited, invalid target, full inventory, not
enough gold and internal. The client shows a chat/alert; failed commands do not
close the session unless the session is invalid.

## Flow control

The server session has a 1024-packet send queue. A slow client that fills it is
closed (`send queue full`), which is intentional. Map actors never block on a
session write.

## References

- `server/docs/protocol-v2.md` (historical detail)
- `server/internal/protocol/opcodes.go` (authoritative list)
- `server/client-patches/protocol.rb`
- `server/content/enums.toml`
