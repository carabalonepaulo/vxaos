# Server Lisp

The server-side script language is a small Scheme-like Lisp. It is used only by
the server:

- event command 355 blocks that run on the server;
- damage formulas in `Skills.rvdata2` / `Items.rvdata2`;
- the optional server prelude (`content/scripts/*.lisp`) with user functions.

Client-only visual scripts (`[NG]` blocks) stay RGSS Ruby. The client does not
parse or execute the server Lisp.

## Values

`nil`, booleans, integers, floats, strings, symbols and lists. Symbols are
written bare (`client`, `atk`). Quoted symbols and lists use `'`:

```lisp
'foo
'(1 2 3)
```

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

`and` and `or` return the actual value, not a coerced boolean. `nil` and
`false` are the only false values; `0` is true.

## Built-ins

- Arithmetic: `+ - * / % **`, `abs`, `min`, `max`, `floor`, `ceil`, `round`,
  `rand`.
- Comparison: `= != < <= > >=`, `not`.
- Predicates: `null?`, `number?`, `string?`, `symbol?`, `list?`, `boolean?`,
  `procedure?`.
- Lists: `list`, `car`, `cdr`, `cons`, `length`, `nth`, `append`, `apply`.
- Strings: `str`, `format` (Go `fmt.Sprintf` verbs), `print`.

Integer `/` uses Ruby-style floor division; `%` follows Ruby's sign rules. This
keeps the formulas' integer semantics when both operands are integers.

## Host objects

Go values implement:

```go
type HostObject interface {
    Get(name string) (any, bool)
    Call(name string, args []any) (any, error)
    Set(name string, value any) error
}
```

Scripts access them with:

```lisp
(get client 'level)
(call client 'quest-in-progress? 1)
(set-field client 'level 10)
```

For a damage formula the server binds `a`, `b` and `v`:

```lisp
(- (* (get a 'atk) 4) (* (get b 'def) 2))
```

For an event script the server binds `client`, `event`, `map`, `interpreter`,
`config`, `data` and `network` as appropriate, plus helper natives such as
`check-point`, `start-quest`, `finish-quest`, `open-bank`, `open-teleport`,
`open-create-guild`, `chat-add` and `send-player-graphic`.

## Limits

Every evaluation has an instruction budget and a call-depth limit. The VM has
no `eval`, file or network primitives; only registered natives are reachable.
Errors abort the event/formula and are logged with the script name.
