package script

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"unicode/utf8"
)

func defaultRand(n int64) int64 {
	if n <= 0 {
		return 0
	}
	return rand.Int63n(n)
}

func native(vm *VM, name string, minArgs, maxArgs int, fn func(*VM, []any) (any, error)) {
	vm.Globals[name] = &NativeFunc{Name: name, MinArgs: minArgs, MaxArgs: maxArgs, Fn: fn}
}

func registerBuiltins(vm *VM) {
	native(vm, "+", 0, -1, func(vm *VM, args []any) (any, error) { return add(args) })
	native(vm, "-", 1, -1, func(vm *VM, args []any) (any, error) { return sub(args) })
	native(vm, "*", 0, -1, func(vm *VM, args []any) (any, error) { return mul(args) })
	native(vm, "/", 1, -1, func(vm *VM, args []any) (any, error) { return div(args) })
	native(vm, "%", 2, 2, func(vm *VM, args []any) (any, error) { return mod(args[0], args[1]) })
	native(vm, "**", 2, 2, func(vm *VM, args []any) (any, error) { return pow(args[0], args[1]) })
	native(vm, "=", 2, -1, func(vm *VM, args []any) (any, error) {
		for i := 1; i < len(args); i++ {
			if !Equal(args[i-1], args[i]) {
				return false, nil
			}
		}
		return true, nil
	})
	native(vm, "!=", 2, 2, func(vm *VM, args []any) (any, error) { return !Equal(args[0], args[1]), nil })
	native(vm, "<", 2, -1, func(vm *VM, args []any) (any, error) { return compareChain(args, func(c int) bool { return c < 0 }) })
	native(vm, "<=", 2, -1, func(vm *VM, args []any) (any, error) { return compareChain(args, func(c int) bool { return c <= 0 }) })
	native(vm, ">", 2, -1, func(vm *VM, args []any) (any, error) { return compareChain(args, func(c int) bool { return c > 0 }) })
	native(vm, ">=", 2, -1, func(vm *VM, args []any) (any, error) { return compareChain(args, func(c int) bool { return c >= 0 }) })
	native(vm, "not", 1, 1, func(vm *VM, args []any) (any, error) { return !Truthy(args[0]), nil })

	native(vm, "null?", 1, 1, func(vm *VM, args []any) (any, error) { return args[0] == nil, nil })
	native(vm, "number?", 1, 1, func(vm *VM, args []any) (any, error) {
		_, ok := asNumber(args[0])
		return ok, nil
	})
	native(vm, "string?", 1, 1, func(vm *VM, args []any) (any, error) {
		_, ok := args[0].(string)
		return ok, nil
	})
	native(vm, "symbol?", 1, 1, func(vm *VM, args []any) (any, error) {
		_, ok := args[0].(Symbol)
		return ok, nil
	})
	native(vm, "list?", 1, 1, func(vm *VM, args []any) (any, error) {
		_, ok := args[0].(*List)
		return ok, nil
	})
	native(vm, "boolean?", 1, 1, func(vm *VM, args []any) (any, error) {
		_, ok := args[0].(bool)
		return ok, nil
	})
	native(vm, "procedure?", 1, 1, func(vm *VM, args []any) (any, error) {
		switch args[0].(type) {
		case *Closure, *NativeFunc:
			return true, nil
		}
		return false, nil
	})

	native(vm, "abs", 1, 1, func(vm *VM, args []any) (any, error) {
		if i, ok := args[0].(int64); ok {
			if i < 0 {
				return -i, nil
			}
			return i, nil
		}
		f, ok := asNumber(args[0])
		if !ok {
			return nil, typeErr("abs", args[0])
		}
		return math.Abs(f), nil
	})
	native(vm, "min", 1, -1, func(vm *VM, args []any) (any, error) { return minMax(args, true) })
	native(vm, "max", 1, -1, func(vm *VM, args []any) (any, error) { return minMax(args, false) })
	native(vm, "floor", 1, 1, func(vm *VM, args []any) (any, error) {
		f, ok := asNumber(args[0])
		if !ok {
			return nil, typeErr("floor", args[0])
		}
		return int64(math.Floor(f)), nil
	})
	native(vm, "sqrt", 1, 1, func(vm *VM, args []any) (any, error) {
		f, ok := asNumber(args[0])
		if !ok {
			return nil, typeErr("sqrt", args[0])
		}
		return math.Sqrt(f), nil
	})
	native(vm, "ceil", 1, 1, func(vm *VM, args []any) (any, error) {
		f, ok := asNumber(args[0])
		if !ok {
			return nil, typeErr("ceil", args[0])
		}
		return int64(math.Ceil(f)), nil
	})
	native(vm, "round", 1, 1, func(vm *VM, args []any) (any, error) {
		f, ok := asNumber(args[0])
		if !ok {
			return nil, typeErr("round", args[0])
		}
		return int64(math.Round(f)), nil
	})
	native(vm, "rand", 0, 1, func(vm *VM, args []any) (any, error) {
		if len(args) == 0 {
			return float64(vm.Rand(1<<53)) / float64(int64(1)<<53), nil
		}
		n, ok := args[0].(int64)
		if !ok || n <= 0 {
			return nil, typeErr("rand", args[0])
		}
		return vm.Rand(n), nil
	})

	native(vm, "list", 0, -1, func(vm *VM, args []any) (any, error) {
		return &List{Elems: append([]any(nil), args...)}, nil
	})
	native(vm, "car", 1, 1, func(vm *VM, args []any) (any, error) {
		l, ok := args[0].(*List)
		if !ok {
			return nil, typeErr("car", args[0])
		}
		if len(l.Elems) == 0 {
			return nil, nil
		}
		return l.Elems[0], nil
	})
	native(vm, "cdr", 1, 1, func(vm *VM, args []any) (any, error) {
		l, ok := args[0].(*List)
		if !ok {
			return nil, typeErr("cdr", args[0])
		}
		if len(l.Elems) <= 1 {
			return &List{}, nil
		}
		return &List{Elems: append([]any(nil), l.Elems[1:]...)}, nil
	})
	native(vm, "cons", 2, 2, func(vm *VM, args []any) (any, error) {
		out := &List{}
		out.Elems = append(out.Elems, args[0])
		switch tail := args[1].(type) {
		case *List:
			out.Elems = append(out.Elems, tail.Elems...)
		case nil:
		default:
			out.Elems = append(out.Elems, tail)
		}
		return out, nil
	})
	native(vm, "length", 1, 1, func(vm *VM, args []any) (any, error) {
		switch x := args[0].(type) {
		case *List:
			return int64(len(x.Elems)), nil
		case string:
			return int64(utf8.RuneCountInString(x)), nil
		case nil:
			return int64(0), nil
		default:
			return nil, typeErr("length", args[0])
		}
	})
	native(vm, "nth", 2, 2, func(vm *VM, args []any) (any, error) {
		l, ok := args[0].(*List)
		if !ok {
			return nil, typeErr("nth", args[0])
		}
		n, ok := args[1].(int64)
		if !ok || n < 0 || int(n) >= len(l.Elems) {
			return nil, nil
		}
		return l.Elems[n], nil
	})
	native(vm, "append", 0, -1, func(vm *VM, args []any) (any, error) {
		out := &List{}
		for _, arg := range args {
			switch x := arg.(type) {
			case *List:
				out.Elems = append(out.Elems, x.Elems...)
			case nil:
			default:
				return nil, typeErr("append", arg)
			}
		}
		return out, nil
	})
	native(vm, "apply", 2, 2, func(vm *VM, args []any) (any, error) {
		list, ok := args[1].(*List)
		if !ok {
			return nil, typeErr("apply", args[1])
		}
		return vm.call(args[0], append([]any(nil), list.Elems...))
	})

	native(vm, "str", 0, -1, func(vm *VM, args []any) (any, error) {
		var b strings.Builder
		for _, arg := range args {
			b.WriteString(display(arg))
		}
		return b.String(), nil
	})
	native(vm, "format", 1, -1, func(vm *VM, args []any) (any, error) {
		format, ok := args[0].(string)
		if !ok {
			return nil, typeErr("format", args[0])
		}
		values := make([]any, len(args)-1)
		for i, v := range args[1:] {
			switch x := v.(type) {
			case Symbol:
				values[i] = string(x)
			case *List:
				values[i] = x.String()
			default:
				values[i] = v
			}
		}
		return fmt.Sprintf(format, values...), nil
	})
	native(vm, "print", 0, -1, func(vm *VM, args []any) (any, error) {
		parts := make([]string, len(args))
		for i, arg := range args {
			parts[i] = display(arg)
		}
		if vm.Stdout != nil {
			vm.Stdout(strings.Join(parts, " ") + "\n")
		}
		return nil, nil
	})
	native(vm, "error", 1, -1, func(vm *VM, args []any) (any, error) {
		parts := make([]string, len(args))
		for i, arg := range args {
			parts[i] = display(arg)
		}
		return nil, fmt.Errorf("script: %s", strings.Join(parts, " "))
	})

	native(vm, "get", 2, 2, func(vm *VM, args []any) (any, error) {
		host, ok := args[0].(HostObject)
		if !ok {
			return nil, typeErr("get", args[0])
		}
		name, ok := keyName(args[1])
		if !ok {
			return nil, typeErr("get", args[1])
		}
		v, _ := host.Get(name)
		return v, nil
	})
	native(vm, "call", 2, -1, func(vm *VM, args []any) (any, error) {
		host, ok := args[0].(HostObject)
		if !ok {
			return nil, typeErr("call", args[0])
		}
		name, ok := keyName(args[1])
		if !ok {
			return nil, typeErr("call", args[1])
		}
		return host.Call(name, args[2:])
	})
	native(vm, "set-field", 3, 3, func(vm *VM, args []any) (any, error) {
		host, ok := args[0].(HostObject)
		if !ok {
			return nil, typeErr("set-field", args[0])
		}
		name, ok := keyName(args[1])
		if !ok {
			return nil, typeErr("set-field", args[1])
		}
		if err := host.Set(name, args[2]); err != nil {
			return nil, err
		}
		return nil, nil
	})
}

func add(args []any) (any, error) {
	var i int64
	var f float64
	useFloat := false
	for _, v := range args {
		if iv, ok := v.(int64); ok && !useFloat {
			i += iv
			continue
		}
		fv, ok := asNumber(v)
		if !ok {
			return nil, typeErr("+", v)
		}
		if !useFloat {
			f = float64(i)
			useFloat = true
		}
		f += fv
	}
	if useFloat {
		return f, nil
	}
	return i, nil
}

func sub(args []any) (any, error) {
	if len(args) == 1 {
		switch x := args[0].(type) {
		case int64:
			return -x, nil
		}
		f, ok := asNumber(args[0])
		if !ok {
			return nil, typeErr("-", args[0])
		}
		return -f, nil
	}
	var i int64
	var f float64
	useFloat := false
	for idx, v := range args {
		if iv, ok := v.(int64); ok && !useFloat {
			if idx == 0 {
				i = iv
			} else {
				i -= iv
			}
			continue
		}
		fv, ok := asNumber(v)
		if !ok {
			return nil, typeErr("-", v)
		}
		if !useFloat {
			f = float64(i)
			useFloat = true
		}
		if idx == 0 {
			f = fv
		} else {
			f -= fv
		}
	}
	if useFloat {
		return f, nil
	}
	return i, nil
}

func mul(args []any) (any, error) {
	var i int64 = 1
	var f float64 = 1
	useFloat := false
	for _, v := range args {
		if iv, ok := v.(int64); ok && !useFloat {
			i *= iv
			continue
		}
		fv, ok := asNumber(v)
		if !ok {
			return nil, typeErr("*", v)
		}
		if !useFloat {
			f = float64(i)
			useFloat = true
		}
		f *= fv
	}
	if useFloat {
		return f, nil
	}
	return i, nil
}

func div(args []any) (any, error) {
	if len(args) == 1 {
		f, ok := asNumber(args[0])
		if !ok {
			return nil, typeErr("/", args[0])
		}
		return 1 / f, nil
	}
	allInt := true
	for _, v := range args {
		if _, ok := v.(int64); !ok {
			allInt = false
			break
		}
	}
	if allInt {
		result := args[0].(int64)
		for _, v := range args[1:] {
			d := v.(int64)
			if d == 0 {
				return nil, fmt.Errorf("script: division by zero")
			}
			result = floorDiv(result, d)
		}
		return result, nil
	}
	f, ok := asNumber(args[0])
	if !ok {
		return nil, typeErr("/", args[0])
	}
	for _, v := range args[1:] {
		fv, ok := asNumber(v)
		if !ok {
			return nil, typeErr("/", v)
		}
		if fv == 0 {
			return nil, fmt.Errorf("script: division by zero")
		}
		f /= fv
	}
	return f, nil
}

func mod(a, b any) (any, error) {
	ai, aok := a.(int64)
	bi, bok := b.(int64)
	if aok && bok {
		if bi == 0 {
			return nil, fmt.Errorf("script: division by zero")
		}
		r := ai % bi
		if r != 0 && (r < 0) != (bi < 0) {
			r += bi
		}
		return r, nil
	}
	af, ok := asNumber(a)
	if !ok {
		return nil, typeErr("%", a)
	}
	bf, ok := asNumber(b)
	if !ok {
		return nil, typeErr("%", b)
	}
	if bf == 0 {
		return nil, fmt.Errorf("script: division by zero")
	}
	return math.Mod(af, bf), nil
}

func pow(a, b any) (any, error) {
	if ai, ok := a.(int64); ok {
		if bi, ok := b.(int64); ok && bi >= 0 {
			result := int64(1)
			base := ai
			exp := bi
			for exp > 0 {
				if exp&1 == 1 {
					result *= base
				}
				exp >>= 1
				if exp > 0 {
					base *= base
				}
			}
			return result, nil
		}
	}
	af, ok := asNumber(a)
	if !ok {
		return nil, typeErr("**", a)
	}
	bf, ok := asNumber(b)
	if !ok {
		return nil, typeErr("**", b)
	}
	return math.Pow(af, bf), nil
}

func floorDiv(a, b int64) int64 {
	q := a / b
	r := a % b
	if r != 0 && (r < 0) != (b < 0) {
		q--
	}
	return q
}

func minMax(args []any, min bool) (any, error) {
	best := args[0]
	for _, v := range args[1:] {
		c, err := compare(best, v)
		if err != nil {
			return nil, err
		}
		if (min && c > 0) || (!min && c < 0) {
			best = v
		}
	}
	return best, nil
}

func compare(a, b any) (int, error) {
	if as, ok := a.(string); ok {
		bs, ok := b.(string)
		if !ok {
			return 0, fmt.Errorf("script: cannot compare string with %s", Repr(b))
		}
		return strings.Compare(as, bs), nil
	}
	af, ok := asNumber(a)
	if !ok {
		return 0, fmt.Errorf("script: cannot compare %s", Repr(a))
	}
	bf, ok := asNumber(b)
	if !ok {
		return 0, fmt.Errorf("script: cannot compare %s", Repr(b))
	}
	switch {
	case af < bf:
		return -1, nil
	case af > bf:
		return 1, nil
	default:
		return 0, nil
	}
}

func compareChain(args []any, pred func(int) bool) (any, error) {
	for i := 1; i < len(args); i++ {
		c, err := compare(args[i-1], args[i])
		if err != nil {
			return nil, err
		}
		if !pred(c) {
			return false, nil
		}
	}
	return true, nil
}

func typeErr(name string, v any) error {
	return fmt.Errorf("script: %s: invalid argument %s", name, Repr(v))
}

func keyName(v any) (string, bool) {
	switch x := v.(type) {
	case Symbol:
		return string(x), true
	case string:
		return x, true
	case int64:
		return strconv.FormatInt(x, 10), true
	case int:
		return strconv.Itoa(x), true
	default:
		return "", false
	}
}

func display(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case Symbol:
		return string(x)
	case bool:
		return strconv.FormatBool(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case *List:
		return x.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
