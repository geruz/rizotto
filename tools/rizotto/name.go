package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	nameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)

	errInvalidName = errors.New(
		"the name must start with an english letter and contain only english letters and digits",
	)
	errReservedName = errors.New("the name is a go keyword")
	// errPluralName rejects an entity name that is already plural.
	errPluralName = errors.New("the name must be singular")
)

//nolint:gochecknoglobals // the keyword set is static
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// knownSingulars end in "s" without being plurals, and no suffix rule tells them
// apart from one: singularize would otherwise offer "New" for "News".
//
//nolint:gochecknoglobals // the word set is static
var knownSingulars = map[string]bool{
	"alias": true, "atlas": true, "canvas": true, "gas": true, "lens": true, "news": true,
}

// normalizeName validates a name that has to be usable as a go identifier. It is
// what a project, a service and a controller all have to satisfy.
func normalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if !nameRe.MatchString(name) {
		return "", fmt.Errorf("%w: %q", errInvalidName, name)
	}

	if goKeywords[strings.ToLower(name)] {
		return "", fmt.Errorf("%w: %q", errReservedName, name)
	}

	return name, nil
}

// normalizeEntityName validates the name of a service or controller entity. On
// top of the identifier rules it must be singular: pluralize derives the table,
// the RPC names, the route and the list field from it, so a plural name lands as
// "articleses" in all of them at once and there is no later step that could
// notice. A project name is free to be plural, which is why this is not part of
// normalizeName.
func normalizeEntityName(raw string) (string, error) {
	name, err := normalizeName(raw)
	if err != nil {
		return "", err
	}

	singular, isPlural := singularize(name)
	if isPlural {
		return "", fmt.Errorf("%w: %q looks plural, try %q", errPluralName, name, singular)
	}

	return name, nil
}

// singularize reports whether word looks like the plural of something, and of
// what. A word counts as plural only when pluralize rebuilds it exactly from the
// candidate, which is what keeps "Address" (whose candidate pluralizes back to
// "Addreses") out.
func singularize(word string) (string, bool) {
	lower := strings.ToLower(word)

	if !strings.HasSuffix(lower, "s") || knownSingulars[lower] {
		return word, false
	}

	// No english plural ends in these, so a word that does is a singular like
	// Status, Address or Analysis.
	for _, suffix := range []string{"us", "ss", "is"} {
		if strings.HasSuffix(lower, suffix) {
			return word, false
		}
	}

	for _, candidate := range singularCandidates(word) {
		if candidate != "" && pluralize(candidate) == word {
			return candidate, true
		}
	}

	return word, false
}

// singularCandidates inverts the rules of pluralize, most specific first.
func singularCandidates(word string) []string {
	// One candidate per rule of pluralize: -ies, -es and the bare -s.
	const rules = 3

	lower := strings.ToLower(word)
	candidates := make([]string, 0, rules)

	if strings.HasSuffix(lower, "ies") && len(word) > len("ies") {
		candidates = append(candidates, word[:len(word)-len("ies")]+"y")
	}

	if strings.HasSuffix(lower, "es") && len(word) > len("es") {
		candidates = append(candidates, word[:len(word)-len("es")])
	}

	return append(candidates, word[:len(word)-1])
}

// pluralize is good enough for the entity names the scaffold deals with. It is
// only ever called on a name normalizeEntityName has already accepted, so it
// never has to cope with a plural input.
func pluralize(word string) string {
	lower := strings.ToLower(word)

	switch {
	case strings.HasSuffix(lower, "s"), strings.HasSuffix(lower, "x"), strings.HasSuffix(lower, "z"),
		strings.HasSuffix(lower, "ch"), strings.HasSuffix(lower, "sh"):
		return word + "es"
	case strings.HasSuffix(lower, "y") && len(word) > 1 && !isVowel(lower[len(lower)-2]):
		return word[:len(word)-1] + "ies"
	default:
		return word + "s"
	}
}

func isVowel(letter byte) bool {
	return strings.IndexByte("aeiou", letter) >= 0
}
