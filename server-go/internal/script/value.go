// Package script implements the small Lisp used by the server for event
// command scripts and damage formulas.
//
// The language is intentionally small: S-expressions, lexical closures, a
// compact stack bytecode and a registry of native functions. Content can add
// pure Lisp functions in a prelude, while Go adds or removes capabilities by
// registering natives. There is no eval, file or network access unless a host
// exposes it explicitly.
package script

import (
	"fmt"
	"strconv"
	"strings"
)

// Symbol is a Lisp identifier.
type Symbol string

// List is a Lisp list. Empty lists are represented by a non-nil List with no
// elements; nil is the empty/nil value.
type List struct {
	Elems []any
}

// Len returns the number of elements.
func (l *List) Len() int {
	if l == nil {
		return 0
	}
	return len(l.Elems)
}

// String renders a readable representation for errors.
func (l *List) String() string {
	if l == nil {
		return "()"
	}
	parts := make([]string, len(l.Elems))
	for i, e := range l.Elems {
		parts[i] = Repr(e)
	}
	return "(" + strings.Join(parts, " ") + ")"
}

// Env is a lexical environment.
type Env struct {
	Parent *Env
	Vars   map[string]any
}

// NewEnv creates a child environment.
func NewEnv(parent *Env) *Env {
	return &Env{Parent: parent, Vars: make(map[string]any)}
}

// Lookup walks the environment chain.
func (e *Env) Lookup(name string) (any, bool) {
	for env := e; env != nil; env = env.Parent {
		if v, ok := env.Vars[name]; ok {
			return v, true
		}
	}
	return nil, false
}

// Assign updates an existing binding or creates it in this environment.
func (e *Env) Assign(name string, value any) bool {
	for env := e; env != nil; env = env.Parent {
		if _, ok := env.Vars[name]; ok {
			env.Vars[name] = value
			return true
		}
	}
	if e != nil {
		e.Vars[name] = value
		return true
	}
	return false
}

// Closure is a compiled function with a captured environment.
type Closure struct {
	Proto *Proto
	Env   *Env
}

// NativeFunc is a Go function exposed to Lisp.
type NativeFunc struct {
	Name    string
	MinArgs int
	MaxArgs int // -1 means variadic
	Fn      func(*VM, []any) (any, error)
}

// HostObject is implemented by Go values exposed to scripts. Get/Call/Set are
// the only operations a script can perform on them.
type HostObject interface {
	Get(name string) (any, bool)
	Call(name string, args []any) (any, error)
	Set(name string, value any) error
}

// OpCode is a VM instruction.
type OpCode uint8

const (
	OpConst OpCode = iota
	OpLoad
	OpStore
	OpStoreGlobal
	OpPop
	OpJump
	OpJumpIfFalse
	OpMakeClosure
	OpCall
	OpReturn
)

// Instruction is one bytecode operation.
type Instruction struct {
	Op OpCode
	X  int
}

// Proto is a compiled function body.
type Proto struct {
	Name      string
	Params    []string
	Variadic  bool
	RestParam string
	Code      []Instruction
	Consts    []any
	Lines     []int
}

// Program is a compiled top-level unit.
type Program struct {
	Proto *Proto
}

// VM executes programs. A VM owns globals and resource limits; it is not safe
// for concurrent use.
type VM struct {
	Globals  map[string]any
	Rand     func(n int64) int64
	MaxSteps int
	MaxDepth int
	Stdout   func(string)

	steps int
	depth int
}

// NewVM creates a VM with the standard library.
func NewVM() *VM {
	vm := &VM{
		Globals:  make(map[string]any),
		MaxSteps: 200000,
		MaxDepth: 64,
		Stdout:   func(string) {},
	}
	vm.Rand = func(n int64) int64 {
		if n <= 0 {
			return 0
		}
		return defaultRand(n)
	}
	registerBuiltins(vm)
	return vm
}

// Eval runs a program with bindings in a fresh scope.
func (vm *VM) Eval(p *Program, bindings map[string]any) (any, error) {
	if p == nil || p.Proto == nil {
		return nil, fmt.Errorf("script: nil program")
	}
	env := NewEnv(nil)
	for k, v := range bindings {
		env.Vars[k] = v
	}
	vm.steps = 0
	vm.depth = 0
	return vm.run(p.Proto, env)
}

// EvalString compiles and evaluates source.
func (vm *VM) EvalString(name, source string, bindings map[string]any) (any, error) {
	p, err := Compile(name, source)
	if err != nil {
		return nil, err
	}
	return vm.Eval(p, bindings)
}

func (vm *VM) run(proto *Proto, env *Env) (any, error) {
	if proto == nil {
		return nil, fmt.Errorf("script: nil proto")
	}
	stack := make([]any, 0, 8)
	pc := 0
	for pc < len(proto.Code) {
		vm.steps++
		if vm.MaxSteps > 0 && vm.steps > vm.MaxSteps {
			return nil, fmt.Errorf("script: instruction limit exceeded in %s", proto.Name)
		}
		if vm.MaxDepth > 0 && vm.depth > vm.MaxDepth {
			return nil, fmt.Errorf("script: call depth exceeded in %s", proto.Name)
		}
		ins := proto.Code[pc]
		pc++
		switch ins.Op {
		case OpConst:
			stack = append(stack, proto.Consts[ins.X])
		case OpLoad:
			name, _ := proto.Consts[ins.X].(Symbol)
			if v, ok := env.Lookup(string(name)); ok {
				stack = append(stack, v)
			} else if v, ok := vm.Globals[string(name)]; ok {
				stack = append(stack, v)
			} else {
				return nil, fmt.Errorf("script: unbound symbol %q in %s", name, proto.Name)
			}
		case OpStore:
			v := pop(&stack)
			name, _ := proto.Consts[ins.X].(Symbol)
			if !env.Assign(string(name), v) {
				vm.Globals[string(name)] = v
			}
			stack = append(stack, nil)
		case OpStoreGlobal:
			v := pop(&stack)
			name, _ := proto.Consts[ins.X].(Symbol)
			vm.Globals[string(name)] = v
			stack = append(stack, nil)
		case OpPop:
			pop(&stack)
		case OpJump:
			pc = ins.X
		case OpJumpIfFalse:
			v := pop(&stack)
			if !Truthy(v) {
				pc = ins.X
			}
		case OpMakeClosure:
			closureProto, _ := proto.Consts[ins.X].(*Proto)
			stack = append(stack, &Closure{Proto: closureProto, Env: env})
		case OpCall:
			argc := ins.X
			if argc < 0 || len(stack) < argc+1 {
				return nil, fmt.Errorf("script: stack underflow in %s", proto.Name)
			}
			args := append([]any(nil), stack[len(stack)-argc:]...)
			callee := stack[len(stack)-argc-1]
			stack = stack[:len(stack)-argc-1]
			v, err := vm.call(callee, args)
			if err != nil {
				return nil, err
			}
			stack = append(stack, v)
		case OpReturn:
			if len(stack) == 0 {
				return nil, nil
			}
			return stack[len(stack)-1], nil
		default:
			return nil, fmt.Errorf("script: bad opcode %d in %s", ins.Op, proto.Name)
		}
	}
	if len(stack) == 0 {
		return nil, nil
	}
	return stack[len(stack)-1], nil
}

func (vm *VM) call(callee any, args []any) (any, error) {
	switch fn := callee.(type) {
	case *Closure:
		if fn.Proto.Variadic {
			if len(args) < len(fn.Proto.Params) {
				return nil, fmt.Errorf("script: %s expects at least %d arguments, got %d", fn.Proto.Name, len(fn.Proto.Params), len(args))
			}
		} else if len(args) != len(fn.Proto.Params) {
			return nil, fmt.Errorf("script: %s expects %d arguments, got %d", fn.Proto.Name, len(fn.Proto.Params), len(args))
		}
		env := NewEnv(fn.Env)
		for i, name := range fn.Proto.Params {
			env.Vars[name] = args[i]
		}
		if fn.Proto.Variadic {
			rest := &List{}
			if len(args) > len(fn.Proto.Params) {
				rest.Elems = append(rest.Elems, args[len(fn.Proto.Params):]...)
			}
			env.Vars[fn.Proto.RestParam] = rest
		}
		vm.depth++
		defer func() { vm.depth-- }()
		return vm.run(fn.Proto, env)
	case *NativeFunc:
		if len(args) < fn.MinArgs || (fn.MaxArgs >= 0 && len(args) > fn.MaxArgs) {
			if fn.MaxArgs < 0 {
				return nil, fmt.Errorf("script: %s expects at least %d arguments, got %d", fn.Name, fn.MinArgs, len(args))
			}
			return nil, fmt.Errorf("script: %s expects %d..%d arguments, got %d", fn.Name, fn.MinArgs, fn.MaxArgs, len(args))
		}
		return fn.Fn(vm, args)
	default:
		return nil, fmt.Errorf("script: %s is not callable", Repr(callee))
	}
}

func pop(stack *[]any) any {
	s := *stack
	if len(s) == 0 {
		return nil
	}
	v := s[len(s)-1]
	*stack = s[:len(s)-1]
	return v
}

// Truthy reports Lisp truthiness: only nil and false are false.
func Truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	default:
		return true
	}
}

// Repr renders a value for debugging and errors.
func Repr(v any) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case string:
		return strconv.Quote(x)
	case Symbol:
		return string(x)
	case *List:
		return x.String()
	case *Closure:
		return "#<closure " + x.Proto.Name + ">"
	case *NativeFunc:
		return "#<native " + x.Name + ">"
	case HostObject:
		return "#<host>"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Equal implements Lisp value equality for the value types used by the VM.
func Equal(a, b any) bool {
	if an, aok := asNumber(a); aok {
		if bn, bok := asNumber(b); bok {
			return an == bn
		}
		return false
	}
	switch x := a.(type) {
	case nil:
		return b == nil
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	case Symbol:
		y, ok := b.(Symbol)
		return ok && x == y
	case *List:
		y, ok := b.(*List)
		if !ok || x.Len() != y.Len() {
			return false
		}
		for i := range x.Elems {
			if !Equal(x.Elems[i], y.Elems[i]) {
				return false
			}
		}
		return true
	}
	return a == b
}

func asNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case int64:
		return float64(x), true
	case int:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}
