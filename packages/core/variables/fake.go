package variables

import (
	"fmt"
	"strings"
)

// A small embedded set, enough for realistic-looking test data without a dependency. Every
// generated address uses example.com and every phone number a fictional 555-01xx line.
var (
	firstNames = []string{"Alex", "Sam", "Jordan", "Taylor", "Morgan", "Casey", "Riley", "Jamie", "Avery", "Quinn", "Maria", "Chen", "Aisha", "Omar", "Elena", "Ivan", "Priya", "Lucas", "Nora", "Yusuf"}
	lastNames  = []string{"Smith", "Johnson", "Garcia", "Kim", "Nguyen", "Brown", "Martin", "Lopez", "Khan", "Rossi", "Silva", "Novak", "Ito", "Haddad", "Larsen", "Okafor", "Petrov", "Dubois", "Cohen", "Moreau"}
	words      = []string{"river", "maple", "signal", "orbit", "lantern", "copper", "meadow", "harbor", "quartz", "ember", "willow", "summit", "breeze", "anchor", "falcon", "pebble", "cinder", "violet", "thunder", "marble"}
)

func pick(e *Engine, list []string) string { return list[e.draw().IntN(len(list))] }

func fnFirstName(e *Engine, _ call) (string, error) { return pick(e, firstNames), nil }
func fnLastName(e *Engine, _ call) (string, error)  { return pick(e, lastNames), nil }
func fnWord(e *Engine, _ call) (string, error)      { return pick(e, words), nil }

func fnFullName(e *Engine, _ call) (string, error) {
	return pick(e, firstNames) + " " + pick(e, lastNames), nil
}

func fnEmail(e *Engine, _ call) (string, error) {
	return fmt.Sprintf("%s.%s%d@example.com", strings.ToLower(pick(e, firstNames)), strings.ToLower(pick(e, lastNames)), e.draw().IntN(9000)+1000), nil
}

func fnPhone(e *Engine, _ call) (string, error) {
	return fmt.Sprintf("+1-555-01%02d", e.draw().IntN(100)), nil
}
