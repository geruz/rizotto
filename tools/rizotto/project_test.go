package main

import (
	"testing"

	"github.com/shoenig/test/must"
)

func Test_normalizeRepo(t *testing.T) {
	t.Parallel()

	valid := map[string]string{
		"github.com/acme/shop":                   testRepo,
		"https://github.com/acme/shop":           testRepo,
		"https://github.com/acme/shop.git":       testRepo,
		"http://github.com/acme/shop/":           testRepo,
		"git@github.com:acme/shop.git":           testRepo,
		"  https://github.com/acme/shop.git  ":   testRepo,
		"ssh://git@gitlab.example.com/acme/shop": "gitlab.example.com/acme/shop",
		"git://git.sr.ht/~acme/shop":             "git.sr.ht/~acme/shop",
		"https://gitlab.com/acme/group/shop.git": "gitlab.com/acme/group/shop",
	}

	for raw, expected := range valid {
		normalized, err := normalizeRepo(raw)
		must.NoError(t, err, must.Sprintf("repo %q", raw))
		must.Eq(t, expected, normalized, must.Sprintf("repo %q", raw))
	}

	invalid := []string{
		"",
		"shop",
		"github.com",
		"github.com/acme",
		"ssh://git@git.internal:2222/acme/shop",
		"локальный/репозиторий",
	}

	for _, raw := range invalid {
		_, err := normalizeRepo(raw)
		must.Error(t, err, must.Sprintf("repo %q must be rejected", raw))
	}
}

func Test_normalizePath(t *testing.T) {
	t.Parallel()

	empty, err := normalizePath("")
	must.NoError(t, err)
	must.True(t, len(empty) > 0)

	absolute, err := normalizePath("/tmp/projects")
	must.NoError(t, err)
	must.Eq(t, "/tmp/projects", absolute)
}

func Test_newProject(t *testing.T) {
	t.Parallel()

	prj, err := newProject("MyShop", "git@github.com:acme/shop.git", "1.25.0", "", true)
	must.NoError(t, err)
	must.Eq(t, "MyShop", prj.Name)
	must.Eq(t, "myshop", prj.Slug)
	must.Eq(t, "github.com/acme/shop/server", prj.Module)
	must.Eq(t, "github.com/acme/shop", prj.Repo)
	must.Eq(t, rizottoModule, prj.RizottoModule)
	must.True(t, prj.SQL)

	_, err = newProject("my-shop", testRepo, "", "", false)
	must.Error(t, err)

	_, err = newProject(testName, "not-a-repo", "", "", false)
	must.Error(t, err)
}
