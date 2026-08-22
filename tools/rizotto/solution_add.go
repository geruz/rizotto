package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

const solutionUsage = `Usage:
	rizotto solution list
	rizotto solution add <name> [flags]
	rizotto solution skill [<name>]

A solution is a whole feature rather than a layer: it brings the services, the
repositories, the migrations, the controllers and the contexts it is made of, and
prints the few lines that have to be wired into server.go by hand.

"rizotto solution list" prints what can be installed, "rizotto solution skill"
explains how solutions work and "rizotto solution skill <name>" describes one.

Common flags:
	-project     path inside the project the solution is added to (default ".")
	-force       write the files even when the project already has some of them
	-skip-tidy   do not run go mod tidy in the project
`

type solutionFlags struct {
	project  string
	force    bool
	skipTidy bool
}

func bindSolutionFlags(fs *flag.FlagSet) *solutionFlags {
	f := &solutionFlags{} //nolint:exhaustruct_v5 // filled by the flag package

	fs.StringVar(&f.project, "project", "", `path inside the project the solution is added to (default ".")`)
	fs.BoolVar(&f.force, "force", false, "write the files even when the project already has some of them")
	fs.BoolVar(&f.skipTidy, "skip-tidy", false, "do not run go mod tidy in the project")

	return f
}

// runSolution dispatches the subcommands of "rizotto solution". It has its own
// router because, unlike a layer, a solution is addressed by name.
func runSolution(args []string, in io.Reader, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: missing subcommand\n\n%s", errUsage, solutionUsage)
	}

	switch args[0] {
	case listCommand:
		return runSolutionList(args[1:], out)
	case addCommand:
		return runSolutionAdd(args[1:], in, out)
	case skillCommand:
		return runSolutionSkill(args[1:], out)
	case helpCommand, helpFlagShort, helpFlagLong:
		_, _ = fmt.Fprint(out, solutionUsage)

		return nil
	default:
		return fmt.Errorf("%w: unknown subcommand %q\n\n%s", errUsage, args[0], solutionUsage)
	}
}

// runSolutionList prints the registry.
func runSolutionList(args []string, out io.Writer) error {
	if len(args) > 0 {
		return fmt.Errorf("%w: list takes no arguments (got %q)", errUsage, args[0])
	}

	_, _ = fmt.Fprintln(out, "Solutions:")

	for _, sol := range solutions {
		_, _ = fmt.Fprintf(out, "  %-10s %s\n", sol.Name(), sol.Summary())
	}

	_, _ = fmt.Fprintf(out, "\nInstall one with: rizotto solution add %s\n", solutions[0].Name())

	return nil
}

// runSolutionSkill prints the general document, or the one of a solution.
func runSolutionSkill(args []string, out io.Writer) error {
	if len(args) == 0 {
		return printSkill(out, solutionSkill)
	}

	if len(args) > 1 {
		return fmt.Errorf("%w: skill takes at most one solution name (got %q)", errUsage, args[1])
	}

	sol, err := findSolution(args[0])
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	return printSkill(out, sol.Skill())
}

func runSolutionAdd(args []string, in io.Reader, out io.Writer) error {
	name, rest, err := splitSolutionName(args)
	if err != nil {
		return err
	}

	sol, err := findSolution(name)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	fs := flag.NewFlagSet("solution add "+sol.Name(), flag.ContinueOnError)
	fs.SetOutput(out)
	fs.Usage = func() {
		_, _ = fmt.Fprint(out, sol.Usage()+"\nFlags:\n")
		fs.PrintDefaults()
	}

	flags := bindSolutionFlags(fs)
	own := sol.BindFlags(fs)

	positional, err := parseInterleaved(fs, rest)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	if len(positional) > 0 {
		return fmt.Errorf("%w: unexpected arguments %v", errUsage, positional)
	}

	return addSolution(sol, flags, own, in, out)
}

// splitSolutionName takes the solution name out of the arguments, wherever the
// flags were written around it.
func splitSolutionName(args []string) (string, []string, error) {
	for index, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}

		rest := append(append([]string{}, args[:index]...), args[index+1:]...)

		return arg, rest, nil
	}

	return "", nil, fmt.Errorf("%w: which solution? one of: %s\n\n%s", errUsage, solutionNames(), solutionUsage)
}

// addSolution installs the solution and whatever it is built on. The flags only
// ever reach the solution that was asked for: a dependency is installed with its
// defaults, so choosing differently means installing it first, by name.
func addSolution(sol solution, flags *solutionFlags, own any, in io.Reader, out io.Writer) error {
	prj, err := inspectProject(flags.project)
	if err != nil {
		return err
	}

	order, err := installOrder(sol, prj)
	if err != nil {
		return err
	}

	asked := newPrompter(in, out)
	silent := newSilentPrompter(out)

	for _, current := range order {
		// Each solution sees the project as the previous one left it, which is how
		// a dependency writing a service becomes visible to the next.
		prj, err = inspectProject(flags.project)
		if err != nil {
			return err
		}

		requiredBy := ""
		pr := asked

		if current.Name() != sol.Name() {
			requiredBy = sol.Name()
			pr = silent
		}

		err = installOne(current, ownFlags(current, own, requiredBy), prj, flags.force, requiredBy, pr, out)
		if err != nil {
			return err
		}
	}

	if !flags.skipTidy {
		tidy(out, prj.Root)
	}

	return nil
}

// ownFlags hands the parsed flags to the solution they were declared by, and an
// empty set to a dependency.
func ownFlags(current solution, own any, requiredBy string) any {
	if requiredBy == "" {
		return own
	}

	return current.BindFlags(flag.NewFlagSet(current.Name(), flag.ContinueOnError))
}

// installOne installs a single solution. requiredBy names the solution that
// pulled it in, and is empty for the one that was asked for.
func installOne(
	sol solution,
	own any,
	prj projectInfo,
	force bool,
	requiredBy string,
	pr *prompter,
	out io.Writer,
) error {
	if requiredBy != "" {
		_, _ = fmt.Fprintf(out,
			"%q requires %q, which the project does not have yet. Installing it with its defaults;\n"+
				"install it by name first if you want to answer its questions yourself.\n\n",
			requiredBy, sol.Name())
	}

	plan, err := sol.Ask(own, pr, prj)
	if err != nil {
		return err
	}

	err = checkSolutionConflicts(plan, prj, force)
	if err != nil {
		return err
	}

	written, err := installSolution(plan, prj)
	if err != nil {
		return err
	}

	reportSolution(out, sol, plan, prj, written)

	return nil
}

// checkSolutionConflicts refuses to overwrite what the project already has.
func checkSolutionConflicts(plan solutionPlan, prj projectInfo, force bool) error {
	if force {
		return nil
	}

	files, err := planFiles(plan, prj)
	if err != nil {
		return err
	}

	existing := conflicts(prj.Root, files)
	if len(existing) == 0 {
		return nil
	}

	return fmt.Errorf("%w:\n  %s", errSolutionConflict, strings.Join(existing, "\n  "))
}

func reportSolution(out io.Writer, sol solution, plan solutionPlan, prj projectInfo, written []string) {
	_, _ = fmt.Fprintf(out, "Added solution %q to %s\n", sol.Name(), prj.Root)

	for _, file := range written {
		_, _ = fmt.Fprintln(out, "  "+file)
	}

	if len(plan.Env) > 0 {
		_, _ = fmt.Fprintln(out, "\nAdd to .env.example:")

		for _, variable := range plan.Env {
			if variable.Comment != "" {
				_, _ = fmt.Fprintln(out, "  # "+variable.Comment)
			}

			_, _ = fmt.Fprintf(out, "  %s=%s\n", variable.Key, variable.Value)
		}
	}

	if len(plan.Steps) == 0 {
		return
	}

	_, _ = fmt.Fprintln(out, "\nNext steps:")

	for index, step := range plan.Steps {
		_, _ = fmt.Fprintf(out, "  %d. %s\n", index+1, step)
	}
}
