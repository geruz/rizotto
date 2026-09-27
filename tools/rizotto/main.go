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
	"time"
)

const toolUsage = `rizotto is the command line companion of the rizotto framework.

Usage:
	rizotto make-project [flags]
	rizotto service add [flags]
	rizotto controller add [flags]
	rizotto solution add <name> [flags]
	rizotto solution list
	rizotto repository skill
	rizotto skill
	rizotto help

Commands:
	make-project   create a new rizotto project, asking for its name, directory,
	               git repository and whether it needs SQL
	service add    scaffold a service in an existing project, asking whether it
	               works with the database and which CRUD methods it needs
	controller add scaffold an HTTP controller for one of the services, asking
	               which routes it serves and whether they need authentication
	solution       install a whole feature rather than a layer, with the services,
	               migrations and controllers it is made of; "solution list"
	               prints what is available
	repository     "repository skill" prints how the database layer works: the
	               column conventions, the sqlc queries and the migrations
	skill          print what a rizotto project is made of; "service skill",
	               "controller skill", "repository skill" and "solution skill"
	               print one layer

Every answer can be given upfront with a flag; the remaining ones are asked step
by step.
`

const (
	helpCommand = "help"
	// addCommand is the subcommand scaffolding a layer or installing a solution.
	addCommand = "add"
	// listCommand is the subcommand printing what can be installed.
	listCommand = "list"
	// helpFlagShort and helpFlagLong ask for the usage of any command.
	helpFlagShort = "-h"
	helpFlagLong  = "--help"

	exitFailure = 1
	exitUsage   = 2
	// commandArg is the index of the command in os.Args.
	commandArg = 1
)

var (
	errUsage          = errors.New("wrong usage")
	errUnknownCommand = errors.New("unknown command")
	errTargetNotEmpty = errors.New("target directory is not empty, pass -force to generate into it anyway")
	errSQLAnswer      = errors.New(`-sql expects "yes" or "no"`)
)

// runCommand dispatches one command and returns errUnknownCommand for a name
// no command answers to.
func runCommand(command string, args []string) error {
	switch command {
	case "make-project":
		return runMakeProject(args, os.Stdin, os.Stdout)
	case "service":
		return runService(args, os.Stdin, os.Stdout)
	case "controller":
		return runController(args, os.Stdin, os.Stdout)
	case "solution":
		return runSolution(args, os.Stdin, os.Stdout)
	case "repository":
		return runRepository(args, os.Stdout)
	case skillCommand:
		return runSkill(args, os.Stdout)
	case helpCommand, helpFlagShort, helpFlagLong:
		_, _ = fmt.Fprint(os.Stdout, toolUsage)

		return nil
	default:
		return fmt.Errorf("%w %q", errUnknownCommand, command)
	}
}

func main() {
	if len(os.Args) <= commandArg {
		_, _ = fmt.Fprint(os.Stderr, toolUsage)
		os.Exit(exitUsage)
	}

	err := runCommand(os.Args[commandArg], os.Args[commandArg+1:])

	if errors.Is(err, errUnknownCommand) {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n\n%s", err.Error(), toolUsage)
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
	skipGit     bool
	skipSkills  bool
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
	fs.BoolVar(&f.skipGit, "skip-git", false, "do not initialise a git repository in the created project")
	fs.BoolVar(&f.skipSkills, "skip-skills", false, "do not run npx skills add shadcn/ui for the web app")

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
		tidy(out, filepath.Join(target, serverDir))
	}

	if !flags.skipGit {
		initGit(out, target)
	}

	if !flags.skipSkills {
		addWebSkills(out, target)
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
	case addCommand:
		return add(args[1:])
	case skillCommand:
		return printSkill(out, skill)
	case helpCommand, helpFlagShort, helpFlagLong:
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

// initGit turns the created project into a git repository, unless git is missing
// or the project already lives inside another repository.
func initGit(out io.Writer, target string) {
	_, err := exec.LookPath("git")
	if err != nil {
		return
	}

	inside := exec.CommandContext(context.Background(), "git", "rev-parse", "--is-inside-work-tree")
	inside.Dir = target

	if inside.Run() == nil {
		_, _ = fmt.Fprintf(out, "\n%s is already inside a git repository, git init skipped\n", target)

		return
	}

	cmd := exec.CommandContext(context.Background(), "git", "init")
	cmd.Dir = target

	output, err := cmd.CombinedOutput()
	if err != nil {
		_, _ = fmt.Fprintf(out, "\ngit init failed, run it yourself:\n%s\n", output)

		return
	}

	_, _ = fmt.Fprintln(out, "\nInitialised an empty git repository in "+target)
}

// skillsTimeout bounds npx skills add, which clones a repository and may stall
// on a slow or missing network.
const skillsTimeout = 5 * time.Minute

// addWebSkills installs the shadcn/ui agent skill when the project has the web
// app. It never fails the command: the project is already there, so a missing
// npx, a network error or a timeout only print how to finish the step by hand.
func addWebSkills(out io.Writer, target string) {
	_, err := os.Stat(filepath.Join(target, webDir, "package.json"))
	if err != nil {
		return
	}

	const manual = "npx skills add shadcn/ui --skill shadcn"

	_, err = exec.LookPath("npx")
	if err != nil {
		_, _ = fmt.Fprintln(out, "\nnpx is not installed, add the shadcn/ui skill yourself: "+manual)

		return
	}

	_, _ = fmt.Fprintln(out, "\n"+manual)

	ctx, cancel := context.WithTimeout(context.Background(), skillsTimeout)
	defer cancel()

	// --yes before skills keeps npx from asking to download the package, --skill
	// picks shadcn out of the repository and the last --yes keeps skills from
	// asking which agents to install to; stdin stays empty so that nothing waits
	// for an answer.
	cmd := exec.CommandContext(ctx, "npx", "--yes", "skills", "add", "shadcn/ui", "--skill", "shadcn", "--yes")
	cmd.Dir = target
	cmd.Stdout = out
	cmd.Stderr = out

	err = cmd.Run()
	if err == nil {
		// skills exits with 0 when it gives up on a prompt, so the result is
		// checked on disk.
		_, err = os.Stat(filepath.Join(target, ".agents", "skills", "shadcn", "SKILL.md"))
	}

	if err != nil {
		_, _ = fmt.Fprintf(out, "\nthe shadcn/ui skill was not installed (%v), run it yourself in %s: %s\n",
			err, target, manual)
	}
}

func printNextSteps(out io.Writer, prj project, target string) {
	steps := []string{"cd " + target, "task deps"}
	if prj.SQL {
		steps = append(steps, "task migrate-up")
	}

	steps = append(steps, "task start      # the server on :9090", "task web:dev    # the React app on :5173")

	_, _ = fmt.Fprintln(out, "\nNext steps:")

	for _, step := range steps {
		_, _ = fmt.Fprintln(out, "  "+step)
	}
}
