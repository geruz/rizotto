// Command rizotto is the framework CLI. Its make-project command scaffolds a
// new service built on top of rizotto, using the layout of the example project.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const toolUsage = `rizotto is the command line companion of the rizotto framework.

Usage:
	rizotto make-project [flags]
	rizotto service add [flags]
	rizotto controller add [flags]
	rizotto skill
	rizotto help

Commands:
	make-project   create a new rizotto project, asking for its name, directory,
	               git repository and whether it needs SQL
	service add    scaffold a service in an existing project, asking whether it
	               works with the database and which CRUD methods it needs
	controller add scaffold an HTTP controller for one of the services, asking
	               which routes it serves and whether they need authentication
	skill          print what a rizotto project is made of; "service skill" and
	               "controller skill" print the details of one layer

Every answer can be given upfront with a flag; the remaining ones are asked step
by step.
`

const (
	helpCommand = "help"

	exitFailure = 1
	exitUsage   = 2
	// commandArg is the index of the command in os.Args.
	commandArg = 1
)

var (
	errUsage          = errors.New("wrong usage")
	errTargetNotEmpty = errors.New("target directory is not empty, pass -force to generate into it anyway")
	errSQLAnswer      = errors.New(`-sql expects "yes" or "no"`)
)

func main() {
	if len(os.Args) <= commandArg {
		_, _ = fmt.Fprint(os.Stderr, toolUsage)
		os.Exit(exitUsage)
	}

	var err error

	switch os.Args[commandArg] {
	case "make-project":
		err = runMakeProject(os.Args[commandArg+1:], os.Stdin, os.Stdout)
	case "service":
		err = runService(os.Args[commandArg+1:], os.Stdin, os.Stdout)
	case "controller":
		err = runController(os.Args[commandArg+1:], os.Stdin, os.Stdout)
	case skillCommand:
		err = runSkill(os.Args[commandArg+1:], os.Stdout)
	case helpCommand, "-h", "--help":
		_, _ = fmt.Fprint(os.Stdout, toolUsage)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[commandArg], toolUsage)
		os.Exit(exitUsage)
	}

	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "rizotto "+os.Args[commandArg]+": "+err.Error())

		if errors.Is(err, errUsage) {
			os.Exit(exitUsage)
		}

		os.Exit(exitFailure)
	}
}

type makeProjectFlags struct {
	name        string
	path        string
	repo        string
	sql         string
	rizottoPath string
	goVersion   string
	force       bool
	skipTidy    bool
}

func bindMakeProjectFlags(fs *flag.FlagSet) *makeProjectFlags {
	f := &makeProjectFlags{} //nolint:exhaustruct_v5 // filled by the flag package

	fs.StringVar(&f.name, "name", "", "project name: english letters and digits, usable as an identifier")
	fs.StringVar(&f.path, "path", "", `directory the project folder is created in (default ".")`)
	fs.StringVar(&f.repo, "repo", "",
		"git repository of the project; it becomes the go module path (github.com/acme/shop)")
	fs.StringVar(&f.sql, "sql", "",
		`"yes" to scaffold PostgreSQL support, "no" to skip it; when omitted the author is asked`)
	fs.StringVar(&f.rizottoPath, "rizotto-path", "",
		"path to a local rizotto checkout; adds a replace directive to the generated go.mod")
	fs.StringVar(&f.goVersion, "go-version", "", "go directive of the generated go.mod (default: the running toolchain)")
	fs.BoolVar(&f.force, "force", false, "generate into an existing non-empty directory")
	fs.BoolVar(&f.skipTidy, "skip-tidy", false, "do not run go mod tidy inside the created project")

	return f
}

func runMakeProject(args []string, in io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("make-project", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.Usage = func() {
		_, _ = fmt.Fprint(out, "Usage: rizotto make-project [flags]\n\nFlags:\n")
		fs.PrintDefaults()
	}

	flags := bindMakeProjectFlags(fs)

	positional, err := parseInterleaved(fs, args)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	if len(positional) > 0 {
		return fmt.Errorf("%w: unexpected arguments %v, use -name, -path and -repo", errUsage, positional)
	}

	prj, target, err := askProject(flags, newPrompter(in, out))
	if err != nil {
		return err
	}

	err = ensureTargetDir(target, flags.force)
	if err != nil {
		return err
	}

	files, err := generateProject(prj, target)
	if err != nil {
		return err
	}

	report(out, prj, target, files)

	if !flags.skipTidy {
		tidy(out, target)
	}

	printNextSteps(out, prj, target)

	return nil
}

// askProject collects the answers of the four steps, skipping the ones already
// given as flags, and returns the project together with its target directory.
func askProject(flags *makeProjectFlags, pr *prompter) (project, string, error) {
	name, err := answer(pr, flags.name, "Project name (english letters and digits, e.g. MyShop)", "", normalizeName)
	if err != nil {
		return project{}, "", err
	}

	path, err := answer(pr, flags.path, "Directory to create the project in", ".", normalizePath)
	if err != nil {
		return project{}, "", err
	}

	repo, err := answer(pr, flags.repo, "Git repository (github.com/acme/shop)", "", normalizeRepo)
	if err != nil {
		return project{}, "", err
	}

	withSQL, err := resolveSQL(flags.sql, pr)
	if err != nil {
		return project{}, "", err
	}

	prj, err := newProject(name, repo, flags.goVersion, flags.rizottoPath, withSQL)
	if err != nil {
		return project{}, "", err
	}

	return prj, filepath.Join(path, prj.Name), nil
}

// answer validates the value coming from a flag, or asks for it when the flag is empty.
func answer(
	pr *prompter,
	flagValue string,
	question string,
	defaultValue string,
	normalize func(string) (string, error),
) (string, error) {
	if flagValue == "" {
		return pr.askValid(question, defaultValue, normalize)
	}

	value, err := normalize(flagValue)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errUsage, err)
	}

	return value, nil
}

// resolveSQL reads the -sql flag and falls back to asking the author.
func resolveSQL(flagValue string, pr *prompter) (bool, error) {
	if flagValue == "" {
		return pr.askYesNo("Does the project need SQL (PostgreSQL + sqlc + dbmate)?", false)
	}

	withSQL, ok := parseYesNo(flagValue)
	if !ok {
		return false, fmt.Errorf("%w: %w, got %q", errUsage, errSQLAnswer, flagValue)
	}

	return withSQL, nil
}

// dispatchAdd routes the "add", "skill" and "help" subcommands of a command.
func dispatchAdd(args []string, usage string, skill string, out io.Writer, add func([]string) error) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: missing subcommand\n\n%s", errUsage, usage)
	}

	switch args[0] {
	case "add":
		return add(args[1:])
	case skillCommand:
		return printSkill(out, skill)
	case helpCommand, "-h", "--help":
		_, _ = fmt.Fprint(out, usage)

		return nil
	default:
		return fmt.Errorf("%w: unknown subcommand %q\n\n%s", errUsage, args[0], usage)
	}
}

// parseInterleaved parses the flag set and returns the positional arguments, so
// that flags may be written before, after or between them.
func parseInterleaved(fs *flag.FlagSet, args []string) ([]string, error) {
	positional := []string{}
	rest := args

	for {
		err := fs.Parse(rest)
		if err != nil {
			return nil, err
		}

		rest = fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}

		positional = append(positional, rest[0])
		rest = rest[1:]
	}
}

func ensureTargetDir(target string, force bool) error {
	entries, err := os.ReadDir(target)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(target, dirPerm)
	}

	if err != nil {
		return fmt.Errorf("read target directory: %w", err)
	}

	if len(entries) > 0 && !force {
		return fmt.Errorf("%w: %s", errTargetNotEmpty, target)
	}

	return nil
}

func report(out io.Writer, prj project, target string, files []string) {
	sql := "without SQL"
	if prj.SQL {
		sql = "with SQL"
	}

	_, _ = fmt.Fprintf(out, "Created project %q (%s) in %s\n", prj.Name, sql, target)

	for _, file := range files {
		_, _ = fmt.Fprintln(out, "  "+file)
	}
}

func tidy(out io.Writer, target string) {
	_, err := exec.LookPath("go")
	if err != nil {
		return
	}

	cmd := exec.CommandContext(context.Background(), "go", "mod", "tidy")
	cmd.Dir = target

	output, err := cmd.CombinedOutput()
	if err != nil {
		_, _ = fmt.Fprintf(out, "\ngo mod tidy failed, run it yourself:\n%s\n", output)
		_, _ = fmt.Fprintf(out,
			"If %s cannot be downloaded, set GOPRIVATE=%s or generate with -rizotto-path <local rizotto checkout>.\n",
			rizottoModule, rizottoModule)
	}
}

func printNextSteps(out io.Writer, prj project, target string) {
	steps := []string{"cd " + target, "task deps", "task start"}
	if prj.SQL {
		steps = []string{"cd " + target, "task deps", "task migrate-up", "task start"}
	}

	_, _ = fmt.Fprintln(out, "\nNext steps:")

	for _, step := range steps {
		_, _ = fmt.Fprintln(out, "  "+step)
	}
}
