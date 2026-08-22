package main

import (
	"testing"

	"github.com/shoenig/test/must"
)

func Test_normalizeName(t *testing.T) {
	t.Parallel()

	valid := []string{testName, bareWord, "Shop2", "a"}
	for _, name := range valid {
		normalized, err := normalizeName(" " + name + " ")
		must.NoError(t, err, must.Sprintf("name %q", name))
		must.Eq(t, name, normalized)
	}

	invalid := []string{"", "2shop", "my-shop", "my_shop", "my shop", "my.shop", "магазин", "shop!", "type"}
	for _, name := range invalid {
		_, err := normalizeName(name)
		must.Error(t, err, must.Sprintf("name %q must be rejected", name))
	}
}

func Test_pluralize(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		testService + "": "Orders",
		"order":          "orders",
		otherService:     "Categories",
		addressName:      "Addresses",
		boxName:          "Boxes",
		"Dish":           "Dishes",
		"Day":            "Days",
	}

	for word, expected := range cases {
		must.Eq(t, expected, pluralize(word), must.Sprintf("word %q", word))
	}
}

// Test_singularize_DetectsPlurals covers the names pluralize would mangle: it
// happily turns "Articles" into "articleses", so the plural has to be caught
// before it reaches the table, the RPCs, the route and the list field.
func Test_singularize_DetectsPlurals(t *testing.T) {
	t.Parallel()

	plurals := map[string]string{
		pluralName:   singularName,
		"Orders":     testService,
		"Users":      "User",
		"Categories": otherService,
		"Companies":  "Company",
		"Boxes":      boxName,
		"Dishes":     "Dish",
		"Matches":    "Match",
		"Buses":      "Bus",
		"Statuses":   "Status",
		"Addresses":  addressName,
		"articles":   "article",
	}

	for word, expected := range plurals {
		singular, isPlural := singularize(word)
		must.True(t, isPlural, must.Sprintf("word %q should look plural", word))
		must.Eq(t, expected, singular, must.Sprintf("word %q", word))
	}
}

// Test_singularize_KeepsSingularsEndingInS is the other half: a singular that
// happens to end in "s" must survive, or the check would reject perfectly good
// entity names like Status or Address.
func Test_singularize_KeepsSingularsEndingInS(t *testing.T) {
	t.Parallel()

	singulars := []string{
		"Status", addressName, "Class", "Business", "Process", "Access",
		"Analysis", "Basis", "Crisis", "Axis", "Thesis",
		"Campus", "Bus", "Focus", "Bonus", "Virus",
		"Lens", "News", "Alias", "Atlas", "Canvas", "Gas",
		singularName, testService, otherService, boxName, "Day",
	}

	for _, word := range singulars {
		singular, isPlural := singularize(word)
		must.False(t, isPlural, must.Sprintf("word %q should not look plural", word))
		must.Eq(t, word, singular)
	}
}

func Test_normalizeEntityName_RejectsAPluralWithASuggestion(t *testing.T) {
	t.Parallel()

	_, err := normalizeEntityName(pluralName)
	must.ErrorIs(t, err, errPluralName)
	must.StrContains(t, err.Error(), `"`+singularName+`"`)

	// The bare name rules still apply.
	_, err = normalizeEntityName("9lives")
	must.ErrorIs(t, err, errInvalidName)

	name, err := normalizeEntityName(testService)
	must.NoError(t, err)
	must.Eq(t, testService, name)
}
