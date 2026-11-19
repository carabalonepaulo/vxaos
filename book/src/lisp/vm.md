# The Lisp VM

The server embeds a small Scheme-like Lisp in `internal/script`. It is used for:

- damage formulas stored in `Skills.rvdata2`/`Items.rvdata2`;
- event command 355 blocks that run on the server;
- the optional server prelude (`content/scripts/*.lisp`) with user functions.

The client never parses or executes this Lisp. Client-only visual scripts remain
RGSS Ruby and are marked `[NG]` in event data.

## Why Lisp instead of Ruby

- A small, auditable evaluator with no file/network/eval primitives.
- Deterministic integer/float behavior that can mirror Ruby formulas.
- One VM reused for formulas and event scripts.
- No Ruby runtime on the server.
- Easy migration: formulas are stored as data, not executed at startup.

## Values

`nil`, booleans, integers, floats, strings, symbols and lists.

```lisp
'foo
'(1 2 3)
```

`nil` and `false` are the only false values. `0` is true.

## Special forms

```lisp
(if cond then else?)
(begin expr...)
(define name value)
(define (name args...) body...)
(set! name value)
(lambda (args...) body...)
(let ((name value)...) body...)
(cond (test body...)... (else body...))
(when test body...)
(unless test body...)
(and expr...)
(or expr...)
(quote value)
```

`and`/`or` return the actual value, not a coerced boolean.

## Built-ins

- Arithmetic: `+ - * / % **`, `abs`, `min`, `max`, `floor`, `ceil`, `round`,
  `rand`.
- Comparison: `= != < <= > >=`, `not`.
- Predicates: `null?`, `number?`, `string?`, `symbol?`, `list?`, `boolean?`,
  `procedure?`.
- Lists: `list`, `car`, `cdr`, `cons`, `length`, `nth`, `append`, `apply`.
- Strings: `str`, `format` (Go `fmt.Sprintf` verbs), `print`.

Integer `/` uses Ruby-style floor division; `%` follows Ruby's sign rules. This
matches integer formulas from the original data.

## Host objects

Go values can expose fields and methods:

```go
type HostObject interface {
    Get(name string) (any, bool)
    Call(name string, args []any) (any, error)
    Set(name string, value any) error
}
```

Access from Lisp:

```lisp
(get client 'level)
(call client 'quest-in-progress? 1)
(set-field client 'level 10)
```

## Bindings

Damage formulas bind:

- `a` — attacker stats;
- `b` — target stats;
- `v` — player variables.

Event scripts bind host objects such as `client`, `event`, `map`,
`interpreter`, `config`, `data` and `network`, depending on the context, plus
native helpers (`start-quest`, `check-point`, `open-bank`, `open-teleport`,
`open-create-guild`, `chat-add`, `send-player-graphic`, ...).

## Limits and security

- Every evaluation has an instruction budget and a call-depth limit.
- There is no `eval`, file, process or network primitive.
- Only registered natives are reachable.
- Errors abort the event/formula and are logged with the script name.
- Scripts are data: a malformed formula fails validation/compilation rather than
  crashing the server.

## Prelude

`content/scripts/*.lisp` is optional server prelude code. It is loaded once and
can define reusable functions for formulas/events. Keep it small and validate
with the server tests; a prelude error can prevent startup.

## Where to go next

- [Damage formulas](damage-formulas.md)
- [Event scripts](event-scripts.md)
- [Language reference](reference.md)
- Low-level ABI: `server/docs/lisp.md`
