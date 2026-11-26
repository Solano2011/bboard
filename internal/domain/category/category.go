package category

import "strings"

const (
	Electric   = "electric"
	Plumbing   = "plumbing"
	Earthworks = "earthworks"
	Repair     = "repair"
	Cargo      = "cargo"
	Other      = "other"
)

var allowed = map[string]struct{}{
	Electric:   {},
	Plumbing:   {},
	Earthworks: {},
	Repair:     {},
	Cargo:      {},
	Other:      {},
}

// Normalize очищает код категории от пробелов и приводит к нижнему регистру.
func Normalize(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// IsValid проверяет, входит ли категория в список допустимых значений.
func IsValid(cat string) bool {
	_, ok := allowed[Normalize(cat)]
	return ok
}
