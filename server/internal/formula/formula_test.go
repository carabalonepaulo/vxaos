package formula

import "testing"

func TestToLisp(t *testing.T) {
	cases := map[string]string{
		"a.atk * 4 - b.def * 2":             `(- (* (get a "atk") 4) (* (get b "def") 2))`,
		"a.atk * 4 + a.def * 2 - b.def * 2": `(- (+ (* (get a "atk") 4) (* (get a "def") 2)) (* (get b "def") 2))`,
		"(a.atk + 2) / 2":                   `(/ (+ (get a "atk") 2) 2)`,
		"Math.sqrt(a.agi) + v[2]":           `(+ (sqrt (get a "agi")) (get v 2))`,
		"-a.mat":                            `(- (get a "mat"))`,
	}
	for source, want := range cases {
		got, ok := ToLisp(source)
		if !ok {
			t.Fatalf("ToLisp(%q) failed", source)
		}
		if got != want {
			t.Errorf("ToLisp(%q) = %q, want %q", source, got, want)
		}
	}
	if _, ok := ToLisp("a.atk ? 1 : 2"); ok {
		t.Fatal("unsupported conditional translated")
	}
}

func TestIsLisp(t *testing.T) {
	for _, source := range []string{"0", "12", "-5", `(- (* (get a "atk") 4) 2)`, ""} {
		if !IsLisp(source) {
			t.Errorf("IsLisp(%q) = false", source)
		}
	}
	if IsLisp("a.atk * 4") {
		t.Error("Ruby formula reported as Lisp")
	}
}
