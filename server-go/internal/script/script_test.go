package script

import (
	"fmt"
	"testing"
)

func eval(t *testing.T, src string, bindings map[string]any) any {
	t.Helper()
	vm := NewVM()
	v, err := vm.EvalString("test", src, bindings)
	if err != nil {
		t.Fatalf("eval %q: %v", src, err)
	}
	return v
}

func evalVM(t *testing.T, vm *VM, src string, bindings map[string]any) any {
	t.Helper()
	v, err := vm.EvalString("test", src, bindings)
	if err != nil {
		t.Fatalf("eval %q: %v", src, err)
	}
	return v
}

func wantInt(t *testing.T, got any, want int64) {
	t.Helper()
	n, ok := got.(int64)
	if !ok || n != want {
		t.Fatalf("got %#v, want %d", got, want)
	}
}

func wantFloat(t *testing.T, got any, want float64) {
	t.Helper()
	f, ok := got.(float64)
	if !ok || f != want {
		t.Fatalf("got %#v, want %v", got, want)
	}
}

func TestArithmetic(t *testing.T) {
	wantInt(t, eval(t, "(+ 1 2 3)", nil), 6)
	wantInt(t, eval(t, "(- 10 3 2)", nil), 5)
	wantInt(t, eval(t, "(- 5)", nil), -5)
	wantInt(t, eval(t, "(* 2 3 4)", nil), 24)
	wantInt(t, eval(t, "(/ 7 2)", nil), 3)
	wantInt(t, eval(t, "(/ -7 2)", nil), -4)
	wantInt(t, eval(t, "(% 7 3)", nil), 1)
	wantInt(t, eval(t, "(** 2 10)", nil), 1024)
	wantFloat(t, eval(t, "(+ 1 2.5)", nil), 3.5)
	wantFloat(t, eval(t, "(/ 1 2.0)", nil), 0.5)
	wantInt(t, eval(t, "(abs -5)", nil), 5)
	wantInt(t, eval(t, "(max 1 9 3)", nil), 9)
	wantInt(t, eval(t, "(min 1 -9 3)", nil), -9)
}

func TestComparisonAndLogic(t *testing.T) {
	if eval(t, "(< 1 2 3)", nil) != true {
		t.Fatal("<")
	}
	if eval(t, "(< 1 3 2)", nil) != false {
		t.Fatal("< chain")
	}
	if eval(t, "(!= 1 2)", nil) != true {
		t.Fatal("!=")
	}
	if eval(t, "(= 'a 'a)", nil) != true {
		t.Fatal("= symbol")
	}
	if eval(t, "(and 1 2)", nil) != int64(2) {
		t.Fatal("and")
	}
	if eval(t, "(and 1 nil 3)", nil) != nil {
		t.Fatal("and nil")
	}
	if eval(t, "(or nil false 3)", nil) != int64(3) {
		t.Fatal("or")
	}
	if eval(t, "(not nil)", nil) != true {
		t.Fatal("not")
	}
	if eval(t, "(if (> 2 1) 'yes 'no)", nil) != Symbol("yes") {
		t.Fatal("if")
	}
}

func TestLetLambdaClosure(t *testing.T) {
	wantInt(t, eval(t, "(let ((x 2) (y 3)) (* x y))", nil), 6)
	wantInt(t, eval(t, "((lambda (x y) (+ x y)) 4 5)", nil), 9)
	wantInt(t, eval(t, "(begin (define (add a b) (+ a b)) (add 7 8))", nil), 15)
	wantInt(t, eval(t, "(begin (define (fact n) (if (<= n 1) 1 (* n (fact (- n 1))))) (fact 5))", nil), 120)
	wantInt(t, eval(t, "(begin (define (make-adder x) (lambda (y) (+ x y))) ((make-adder 10) 5))", nil), 15)
	wantInt(t, eval(t, "(let ((x 1)) (set! x 2) x)", nil), 2)
	wantInt(t, eval(t, "(let ((x 1)) (let ((x 2)) x))", nil), 2)
	wantInt(t, eval(t, "(let ((x 1)) (let ((x 2)) x) x)", nil), 1)
}

func TestVariadic(t *testing.T) {
	wantInt(t, eval(t, "((lambda (a &rest xs) (length xs)) 1 2 3)", nil), 2)
	wantInt(t, eval(t, "((lambda (&rest xs) (length xs)))", nil), 0)
	wantInt(t, eval(t, "((lambda (a &rest xs) (+ a (car xs))) 1 2 3)", nil), 3)
}

func TestCondWhenUnless(t *testing.T) {
	if got := eval(t, "(cond ((> 1 2) 'a) ((< 1 2) 'b) (else 'c))", nil); got != Symbol("b") {
		t.Fatalf("cond = %#v", got)
	}
	if got := eval(t, "(cond ((> 1 2) 'a) (else 'c))", nil); got != Symbol("c") {
		t.Fatalf("cond else = %#v", got)
	}
	if got := eval(t, "(when true 1 2)", nil); got != int64(2) {
		t.Fatalf("when = %#v", got)
	}
	if got := eval(t, "(unless false 1 2)", nil); got != int64(2) {
		t.Fatalf("unless = %#v", got)
	}
	if got := eval(t, "(when false 1)", nil); got != nil {
		t.Fatalf("when false = %#v", got)
	}
}

func TestListsAndQuote(t *testing.T) {
	if got := eval(t, "(car '(1 2 3))", nil); got != int64(1) {
		t.Fatalf("car = %#v", got)
	}
	if got := eval(t, "(cdr '(1 2 3))", nil); !Equal(got, &List{Elems: []any{int64(2), int64(3)}}) {
		t.Fatalf("cdr = %#v", got)
	}
	if got := eval(t, "(length '(1 2 3))", nil); got != int64(3) {
		t.Fatalf("length = %#v", got)
	}
	if got := eval(t, "(nth '(1 2 3) 1)", nil); got != int64(2) {
		t.Fatalf("nth = %#v", got)
	}
	if got := eval(t, "(append '(1 2) '(3 4))", nil); !Equal(got, &List{Elems: []any{int64(1), int64(2), int64(3), int64(4)}}) {
		t.Fatalf("append = %#v", got)
	}
	if got := eval(t, "(list 1 'a \"b\")", nil); got.(*List).Len() != 3 {
		t.Fatalf("list = %#v", got)
	}
}

func TestStringsAndFormat(t *testing.T) {
	if got := eval(t, `(str "a" 1 'b)`, nil); got != "a1b" {
		t.Fatalf("str = %#v", got)
	}
	if got := eval(t, `(format "%d %s" 7 "x")`, nil); got != "7 x" {
		t.Fatalf("format = %#v", got)
	}
}

type testHost struct {
	fields map[string]any
	calls  map[string]func([]any) (any, error)
}

func (h *testHost) Get(name string) (any, bool) {
	v, ok := h.fields[name]
	return v, ok
}

func (h *testHost) Call(name string, args []any) (any, error) {
	fn, ok := h.calls[name]
	if !ok {
		return nil, fmt.Errorf("no method %s", name)
	}
	return fn(args)
}

func (h *testHost) Set(name string, value any) error {
	h.fields[name] = value
	return nil
}

func TestHostObject(t *testing.T) {
	a := &testHost{fields: map[string]any{"atk": int64(10), "level": int64(3)}}
	b := &testHost{fields: map[string]any{"def": int64(4), "mhp": int64(100)}}
	a.calls = map[string]func([]any) (any, error){
		"quest-in-progress?": func(args []any) (any, error) { return true, nil },
	}
	bindings := map[string]any{"a": a, "b": b}
	wantInt(t, eval(t, "(+ (get a 'atk) 5)", bindings), 15)
	wantInt(t, eval(t, "(- (get b 'mhp) (* (get b 'def) 2))", bindings), 92)
	if got := eval(t, "(call a 'quest-in-progress? 1)", bindings); got != true {
		t.Fatalf("call = %#v", got)
	}
	eval(t, "(set-field a 'level 9)", bindings)
	if a.fields["level"] != int64(9) {
		t.Fatalf("set-field = %#v", a.fields)
	}
}

func TestRandDeterministic(t *testing.T) {
	vm := NewVM()
	vm.Rand = func(n int64) int64 { return n - 1 }
	if got := evalVM(t, vm, "(rand 10)", nil); got != int64(9) {
		t.Fatalf("rand = %#v", got)
	}
}

func TestLimits(t *testing.T) {
	vm := NewVM()
	vm.MaxDepth = 32
	vm.MaxSteps = 10000
	_, err := vm.EvalString("loop", "(begin (define (loop) (loop)) (loop))", nil)
	if err == nil {
		t.Fatal("expected recursion/step error")
	}
}

func TestErrors(t *testing.T) {
	vm := NewVM()
	for _, src := range []string{"(", "(+ 1", "'", ")"} {
		if _, err := vm.EvalString("bad", src, nil); err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
	if _, err := vm.EvalString("bad", "(+ 1 'x)", nil); err == nil {
		t.Error("expected type error")
	}
	if _, err := vm.EvalString("bad", "(missing 1)", nil); err == nil {
		t.Error("expected unbound error")
	}
}
