package script

import "fmt"

type compiler struct {
	name   string
	source string
	code   []Instruction
	consts []any
	lines  []int
	gensym int
}

func (c *compiler) proto() *Proto {
	return &Proto{Name: c.name, Code: c.code, Consts: c.consts, Lines: c.lines}
}

func (c *compiler) emit(op OpCode, x int) int {
	c.code = append(c.code, Instruction{Op: op, X: x})
	return len(c.code) - 1
}

func (c *compiler) constIndex(v any) int {
	c.consts = append(c.consts, v)
	return len(c.consts) - 1
}

func (c *compiler) patch(at, target int) {
	c.code[at].X = target
}

func (c *compiler) compileBody(forms []any) error {
	if len(forms) == 0 {
		c.emit(OpConst, c.constIndex(nil))
		return nil
	}
	for i, form := range forms {
		if err := c.compileExpr(form); err != nil {
			return err
		}
		if i != len(forms)-1 {
			c.emit(OpPop, 0)
		}
	}
	return nil
}

func (c *compiler) compileExpr(form any) error {
	switch x := form.(type) {
	case nil:
		c.emit(OpConst, c.constIndex(nil))
		return nil
	case bool:
		c.emit(OpConst, c.constIndex(x))
		return nil
	case int64:
		c.emit(OpConst, c.constIndex(x))
		return nil
	case float64:
		c.emit(OpConst, c.constIndex(x))
		return nil
	case string:
		c.emit(OpConst, c.constIndex(x))
		return nil
	case Symbol:
		c.emit(OpLoad, c.constIndex(x))
		return nil
	case *List:
		return c.compileList(x)
	default:
		return fmt.Errorf("script: %s: cannot compile value %T", c.name, form)
	}
}

func (c *compiler) compileList(list *List) error {
	if list == nil || len(list.Elems) == 0 {
		c.emit(OpConst, c.constIndex(nil))
		return nil
	}
	if head, ok := list.Elems[0].(Symbol); ok {
		switch head {
		case "quote":
			if len(list.Elems) != 2 {
				return c.formErr(list, "quote expects one argument")
			}
			c.emit(OpConst, c.constIndex(list.Elems[1]))
			return nil
		case "if":
			if len(list.Elems) < 3 || len(list.Elems) > 4 {
				return c.formErr(list, "if expects 2 or 3 arguments")
			}
			if err := c.compileExpr(list.Elems[1]); err != nil {
				return err
			}
			jumpElse := c.emit(OpJumpIfFalse, 0)
			if err := c.compileExpr(list.Elems[2]); err != nil {
				return err
			}
			jumpEnd := c.emit(OpJump, 0)
			c.patch(jumpElse, len(c.code))
			if len(list.Elems) == 4 {
				if err := c.compileExpr(list.Elems[3]); err != nil {
					return err
				}
			} else {
				c.emit(OpConst, c.constIndex(nil))
			}
			c.patch(jumpEnd, len(c.code))
			return nil
		case "begin", "do":
			return c.compileBody(list.Elems[1:])
		case "lambda":
			return c.compileLambda(list)
		case "define":
			if len(list.Elems) != 3 {
				return c.formErr(list, "define expects a name and a value")
			}
			if sig, ok := list.Elems[1].(*List); ok {
				if len(sig.Elems) == 0 {
					return c.formErr(list, "define function name is missing")
				}
				name, ok := sig.Elems[0].(Symbol)
				if !ok {
					return c.formErr(list, "define function name must be a symbol")
				}
				params := &List{Elems: append([]any(nil), sig.Elems[1:]...)}
				body := &List{Elems: append([]any{Symbol("begin")}, list.Elems[2:]...)}
				lambda := &List{Elems: []any{Symbol("lambda"), params, body}}
				return c.compileExpr(&List{Elems: []any{Symbol("define"), name, lambda}})
			}
			name, ok := list.Elems[1].(Symbol)
			if !ok {
				return c.formErr(list, "define name must be a symbol")
			}
			if err := c.compileExpr(list.Elems[2]); err != nil {
				return err
			}
			c.emit(OpStoreGlobal, c.constIndex(name))
			return nil
		case "set!":
			if len(list.Elems) != 3 {
				return c.formErr(list, "set! expects a name and a value")
			}
			name, ok := list.Elems[1].(Symbol)
			if !ok {
				return c.formErr(list, "set! name must be a symbol")
			}
			if err := c.compileExpr(list.Elems[2]); err != nil {
				return err
			}
			c.emit(OpStore, c.constIndex(name))
			return nil
		case "let":
			return c.compileLet(list)
		case "cond":
			return c.compileCond(list.Elems[1:])
		case "when":
			if len(list.Elems) < 2 {
				return c.formErr(list, "when expects a condition")
			}
			if len(list.Elems) == 2 {
				if err := c.compileExpr(list.Elems[1]); err != nil {
					return err
				}
				c.emit(OpPop, 0)
				c.emit(OpConst, c.constIndex(nil))
				return nil
			}
			body := &List{Elems: append([]any{Symbol("begin")}, list.Elems[2:]...)}
			return c.compileExpr(&List{Elems: []any{Symbol("if"), list.Elems[1], body, nil}})
		case "unless":
			if len(list.Elems) < 2 {
				return c.formErr(list, "unless expects a condition")
			}
			if len(list.Elems) == 2 {
				if err := c.compileExpr(list.Elems[1]); err != nil {
					return err
				}
				c.emit(OpPop, 0)
				c.emit(OpConst, c.constIndex(nil))
				return nil
			}
			body := &List{Elems: append([]any{Symbol("begin")}, list.Elems[2:]...)}
			return c.compileExpr(&List{Elems: []any{Symbol("if"), list.Elems[1], nil, body}})
		case "and":
			return c.compileAnd(list.Elems[1:])
		case "or":
			return c.compileOr(list.Elems[1:])
		}
	}
	// Ordinary function call.
	if err := c.compileExpr(list.Elems[0]); err != nil {
		return err
	}
	for _, arg := range list.Elems[1:] {
		if err := c.compileExpr(arg); err != nil {
			return err
		}
	}
	c.emit(OpCall, len(list.Elems)-1)
	return nil
}

func (c *compiler) compileLambda(list *List) error {
	if len(list.Elems) < 2 {
		return c.formErr(list, "lambda expects parameters and a body")
	}
	paramsList, ok := list.Elems[1].(*List)
	if !ok {
		return c.formErr(list, "lambda parameters must be a list")
	}
	var params []string
	variadic := false
	rest := ""
	for i := 0; i < len(paramsList.Elems); i++ {
		sym, ok := paramsList.Elems[i].(Symbol)
		if !ok {
			return c.formErr(list, "lambda parameter must be a symbol")
		}
		if sym == "&rest" {
			if i+1 >= len(paramsList.Elems) {
				return c.formErr(list, "&rest requires a parameter name")
			}
			restSym, ok := paramsList.Elems[i+1].(Symbol)
			if !ok {
				return c.formErr(list, "&rest parameter must be a symbol")
			}
			variadic = true
			rest = string(restSym)
			if i+2 != len(paramsList.Elems) {
				return c.formErr(list, "&rest must be the last parameter")
			}
			break
		}
		params = append(params, string(sym))
	}
	child := &compiler{name: fmt.Sprintf("%s:<lambda@%d>", c.name, len(c.code)), source: c.source, gensym: c.gensym}
	if err := child.compileBody(list.Elems[2:]); err != nil {
		return err
	}
	child.emit(OpReturn, 0)
	proto := child.proto()
	proto.Params = params
	proto.Variadic = variadic
	proto.RestParam = rest
	c.emit(OpMakeClosure, c.constIndex(proto))
	return nil
}

func (c *compiler) compileLet(list *List) error {
	if len(list.Elems) < 2 {
		return c.formErr(list, "let expects bindings and a body")
	}
	bindings, ok := list.Elems[1].(*List)
	if !ok {
		return c.formErr(list, "let bindings must be a list")
	}
	names := &List{}
	args := make([]any, 0, len(bindings.Elems))
	for _, b := range bindings.Elems {
		pair, ok := b.(*List)
		if !ok || len(pair.Elems) != 2 {
			return c.formErr(list, "let binding must be (name expression)")
		}
		name, ok := pair.Elems[0].(Symbol)
		if !ok {
			return c.formErr(list, "let binding name must be a symbol")
		}
		names.Elems = append(names.Elems, name)
		args = append(args, pair.Elems[1])
	}
	body := &List{Elems: append([]any{Symbol("begin")}, list.Elems[2:]...)}
	lambda := &List{Elems: []any{Symbol("lambda"), names, body}}
	call := &List{Elems: append([]any{lambda}, args...)}
	return c.compileExpr(call)
}

func (c *compiler) compileCond(clauses []any) error {
	if len(clauses) == 0 {
		c.emit(OpConst, c.constIndex(nil))
		return nil
	}
	clause, ok := clauses[0].(*List)
	if !ok || len(clause.Elems) == 0 {
		return fmt.Errorf("script: %s: invalid cond clause", c.name)
	}
	if test, ok := clause.Elems[0].(Symbol); ok && test == "else" {
		if len(clause.Elems) == 1 {
			c.emit(OpConst, c.constIndex(nil))
			return nil
		}
		return c.compileBody(clause.Elems[1:])
	}
	test := clause.Elems[0]
	var thenForm any
	if len(clause.Elems) == 1 {
		thenForm = test
	} else if len(clause.Elems) == 2 {
		thenForm = clause.Elems[1]
	} else {
		thenForm = &List{Elems: append([]any{Symbol("begin")}, clause.Elems[1:]...)}
	}
	rest := &List{Elems: append([]any{Symbol("cond")}, clauses[1:]...)}
	return c.compileExpr(&List{Elems: []any{Symbol("if"), test, thenForm, rest}})
}

func (c *compiler) compileAnd(args []any) error {
	switch len(args) {
	case 0:
		c.emit(OpConst, c.constIndex(true))
		return nil
	case 1:
		return c.compileExpr(args[0])
	default:
		tmp := Symbol(fmt.Sprintf("__and_%d", c.gensym))
		c.gensym++
		rest := &List{Elems: append([]any{Symbol("and")}, args[1:]...)}
		ifForm := &List{Elems: []any{Symbol("if"), tmp, rest, tmp}}
		letForm := &List{Elems: []any{Symbol("let"), &List{Elems: []any{&List{Elems: []any{tmp, args[0]}}}}, ifForm}}
		return c.compileExpr(letForm)
	}
}

func (c *compiler) compileOr(args []any) error {
	switch len(args) {
	case 0:
		c.emit(OpConst, c.constIndex(nil))
		return nil
	case 1:
		return c.compileExpr(args[0])
	default:
		tmp := Symbol(fmt.Sprintf("__or_%d", c.gensym))
		c.gensym++
		rest := &List{Elems: append([]any{Symbol("or")}, args[1:]...)}
		ifForm := &List{Elems: []any{Symbol("if"), tmp, tmp, rest}}
		letForm := &List{Elems: []any{Symbol("let"), &List{Elems: []any{&List{Elems: []any{tmp, args[0]}}}}, ifForm}}
		return c.compileExpr(letForm)
	}
}

func (c *compiler) formErr(list *List, msg string) error {
	return fmt.Errorf("script: %s: %s in %s", c.name, msg, list.String())
}
