package main

import (
	"embed"
	"errors"
	"fmt"
	"io"
)

// skillsFS holds the documents printed by the skill commands.
//
//go:embed skills
var skillsFS embed.FS

const (
	// skillCommand is the subcommand printing a skill document.
	skillCommand = "skill"
	// projectSkill is the document behind "rizotto skill".
	projectSkill = "rizotto"
	// repositorySkill is the document behind "rizotto repository skill".
	repositorySkill = "repository"
)

const repositoryUsage = `Usage:
	rizotto repository skill

Prints how the database layer of a service works: the column conventions, the
sqlc queries, the dbmate migrations and the transaction helpers.

Repositories are not scaffolded on their own — "rizotto service add -db yes"
writes one together with the service that owns it.
`

var errUnknownSkill = errors.New("unknown skill")

// printSkill writes skills/<name>.md to out.
func printSkill(out io.Writer, name string) error {
	content, err := skillsFS.ReadFile("skills/" + name + ".md")
	if err != nil {
		return fmt.Errorf("%w %q", errUnknownSkill, name)
	}

	_, err = out.Write(content)
	if err != nil {
		return fmt.Errorf("print the %s skill: %w", name, err)
	}

	return nil
}

// runSkill serves "rizotto skill", which takes no arguments.
func runSkill(args []string, out io.Writer) error {
	if len(args) > 0 {
		return fmt.Errorf(
			`%w: skill takes no arguments (got %q); for one layer use "rizotto service skill" or "rizotto controller skill"`,
			errUsage, args[0],
		)
	}

	return printSkill(out, projectSkill)
}

// runRepository serves "rizotto repository", which only prints its skill: a
// repository belongs to the service that owns the table, so it has no "add".
func runRepository(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: missing subcommand\n\n%s", errUsage, repositoryUsage)
	}

	switch args[0] {
	case skillCommand:
		return printSkill(out, repositorySkill)
	case helpCommand, helpFlagShort, helpFlagLong:
		_, _ = fmt.Fprint(out, repositoryUsage)

		return nil
	default:
		return fmt.Errorf("%w: unknown subcommand %q\n\n%s", errUsage, args[0], repositoryUsage)
	}
}
