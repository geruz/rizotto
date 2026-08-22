package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shoenig/test/must"
)

const (
	skipTidyFlag = "-skip-tidy"
	nameFlag     = "-name"
	pathFlag     = "-path"
	repoFlag     = "-repo"
	sqlFlag      = "-sql"
	invalidName  = "my-shop"
	// bareWord is a valid project name but not a valid repository.
	bareWord = "shop"
)

func Test_runMakeProject_AsksEveryStep(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	out := &bytes.Buffer{}
	answers := strings.Join([]string{testName, root, "git@github.com:acme/shop.git", "y", ""}, "\n")

	err := runMakeProject([]string{skipTidyFlag}, strings.NewReader(answers), out)
	must.NoError(t, err)

	printed := out.String()
	must.StrContains(t, printed, "Project name")
	must.StrContains(t, printed, "Directory to create the project in [.]")
	must.StrContains(t, printed, "Git repository")
	must.StrContains(t, printed, "Does the project need SQL")
	must.StrContains(t, printed, `Created project "MyShop" (with SQL)`)

	must.FileExists(t, filepath.Join(root, "MyShop", "server.go"))
	must.FileExists(t, filepath.Join(root, "MyShop", "sqlc.yaml"))
}

func Test_runMakeProject_RepeatsAWrongAnswer(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	out := &bytes.Buffer{}
	answers := strings.Join([]string{invalidName, "МагазинЪ", testName, root, bareWord, testRepo, "n", ""}, "\n")

	err := runMakeProject([]string{skipTidyFlag}, strings.NewReader(answers), out)
	must.NoError(t, err)

	printed := out.String()
	must.StrContains(t, printed, "english letters and digits")
	must.StrContains(t, printed, "expected a repository like github.com/acme/shop")
	must.StrContains(t, printed, `Created project "MyShop" (without SQL)`)
	must.FileExists(t, filepath.Join(root, "MyShop", "server.go"))
}

func Test_runMakeProject_FlagsSkipTheQuestions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	out := &bytes.Buffer{}
	args := []string{nameFlag, testName, pathFlag, root, repoFlag, testRepo, sqlFlag, "no", skipTidyFlag}

	err := runMakeProject(args, strings.NewReader(""), out)
	must.NoError(t, err)
	must.StrNotContains(t, out.String(), "Project name")
	must.FileExists(t, filepath.Join(root, testName, "go.mod"))

	// a second run must not silently overwrite an existing project
	err = runMakeProject(args, strings.NewReader(""), out)
	must.Error(t, err)
}

func Test_runMakeProject_RejectsBadInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	cases := map[string][]string{
		caseInvalidName:      {nameFlag, invalidName, pathFlag, root, repoFlag, testRepo, sqlFlag, "no"},
		"invalid repo":       {nameFlag, testName, pathFlag, root, repoFlag, bareWord, sqlFlag, "no"},
		"invalid sql answer": {nameFlag, testName, pathFlag, root, repoFlag, testRepo, sqlFlag, "maybe"},
		casePositional:       {testName, root},
		"unknown flag":       {"-what", "ever"},
	}

	for name, args := range cases {
		out := &bytes.Buffer{}

		err := runMakeProject(append(args, skipTidyFlag), strings.NewReader(""), out)
		must.ErrorIs(t, err, errUsage, must.Sprintf("case %q", name))
	}
}

func Test_runMakeProject_FailsWithoutAnswers(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	err := runMakeProject([]string{skipTidyFlag}, strings.NewReader(""), out)
	must.ErrorIs(t, err, errNoInput)
}

func Test_parseYesNo(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"y", "Yes", "TRUE", "1", "да"} {
		answer, ok := parseYesNo(value)
		must.True(t, ok, must.Sprintf("value %q", value))
		must.True(t, answer, must.Sprintf("value %q", value))
	}

	for _, value := range []string{"n", "No", "false", "0", "нет"} {
		answer, ok := parseYesNo(value)
		must.True(t, ok, must.Sprintf("value %q", value))
		must.False(t, answer, must.Sprintf("value %q", value))
	}

	_, ok := parseYesNo("maybe")
	must.False(t, ok)
}
