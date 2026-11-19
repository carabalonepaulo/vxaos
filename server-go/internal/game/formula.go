package game

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/kelaia/vxaos/server/internal/script"
)

// Damage formulas are stored as Lisp in the .rvdata2 data (the client guard
// makes the field server-only). They are compiled once and cached as bytecode.
var (
	formulaMu       sync.Mutex
	formulaVM       = script.NewVM()
	formulaPrograms = make(map[string]*script.Program)
	formulaFailed   = make(map[string]bool)
)

// evalFormula evaluates a server-side Lisp damage formula. Empty and zero
// formulas evaluate to zero without compiling.
func evalFormula(source string, a, b battleStats, vars []int32) float64 {
	source = strings.TrimSpace(source)
	if source == "" || source == "0" {
		return 0
	}
	formulaMu.Lock()
	defer formulaMu.Unlock()
	program, compiled := formulaPrograms[source]
	if !compiled {
		if formulaFailed[source] {
			return 0
		}
		var err error
		program, err = script.Compile("damage", source)
		if err != nil {
			formulaFailed[source] = true
			return 0
		}
		formulaPrograms[source] = program
	}
	value, err := formulaVM.Eval(program, map[string]any{
		"a": formulaBattler{stats: a},
		"b": formulaBattler{stats: b},
		"v": formulaVariables{vars: vars},
	})
	if err != nil {
		return 0
	}
	switch n := value.(type) {
	case int64:
		return float64(n)
	case float64:
		return n
	case int:
		return float64(n)
	}
	return 0
}

type formulaBattler struct {
	stats battleStats
}

func (b formulaBattler) Get(name string) (any, bool) {
	switch name {
	case "hp":
		return int64(b.stats.HP), true
	case "mp":
		return int64(b.stats.MP), true
	case "mhp":
		return int64(b.stats.MHP), true
	case "mmp":
		return int64(b.stats.MMP), true
	case "atk":
		return int64(b.stats.Params[2]), true
	case "def":
		return int64(b.stats.Params[3]), true
	case "mat":
		return int64(b.stats.Params[4]), true
	case "mdf":
		return int64(b.stats.Params[5]), true
	case "agi":
		return int64(b.stats.Params[6]), true
	case "luk":
		return int64(b.stats.Params[7]), true
	case "level":
		return int64(b.stats.Level), true
	case "id":
		return int64(0), true
	}
	return int64(0), true
}

func (formulaBattler) Call(string, []any) (any, error) {
	return nil, fmt.Errorf("formula: battler methods are not available")
}

func (formulaBattler) Set(string, any) error {
	return fmt.Errorf("formula: battlers are read-only")
}

type formulaVariables struct {
	vars []int32
}

func (v formulaVariables) Get(name string) (any, bool) {
	index, err := strconv.Atoi(name)
	if err != nil || index < 0 || index >= len(v.vars) {
		return int64(0), true
	}
	return int64(v.vars[index]), true
}

func (formulaVariables) Call(string, []any) (any, error) {
	return nil, fmt.Errorf("formula: variables are not callable")
}

func (formulaVariables) Set(string, any) error {
	return fmt.Errorf("formula: variables are read-only")
}
