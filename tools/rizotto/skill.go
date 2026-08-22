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
)

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
