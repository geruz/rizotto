package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const serviceUsage = `Usage:
	rizotto service add [flags]
	rizotto service skill

Adds a service to an existing project, asking for its name, whether it works with
a database and which CRUD methods it needs. Every answer can be given upfront with
a flag (-name, -db, -methods) and the matching question is then skipped.

"rizotto service skill" prints how a service works in detail.
`

var (
	errNotAModule   = errors.New("go.mod has no module directive")
	errDBAnswer     = errors.New(`-db expects "yes" or "no"`)
	errServiceExist = errors.New("the service directory already exists, pass -force to add files to it")
)

//nolint:gochecknoglobals // compiled once
var moduleRe = regexp.MustCompile(`(?m)^module\s+(\S+)\s*$`)

type serviceFlags struct {
	name     string
	project  string
	db       string
	methods  string
	force    bool
	skipTidy bool
}

func bindServiceFlags(fs *flag.FlagSet) *serviceFlags {
	f := &serviceFlags{} //nolint:exhaustruct_v5 // filled by the flag package

	fs.StringVar(&f.name, "name", "", "service name: singular, english letters and digits, e.g. Order")
	fs.StringVar(&f.project, "project", "", `path inside the project the service is added to (default ".")`)
	fs.StringVar(&f.db, "db", "", `"yes" when the service owns a database table, "no" otherwise`)
	fs.StringVar(&f.methods, "methods", "",
		`comma separated CRUD methods: create, read, list, update, delete, or "all"`)
	fs.BoolVar(&f.force, "force", false, "write into an existing service directory")
	fs.BoolVar(&f.skipTidy, "skip-tidy", false, "do not run go mod tidy in the project")

	return f
}

// runService dispatches the subcommands of "rizotto service".
func runService(args []string, in io.Reader, out io.Writer) error {
	return dispatchAdd(args, serviceUsage, "service", out, func(rest []string) error {
		return runServiceAdd(rest, in, out)
	})
}

func runServiceAdd(args []string, in io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("service add", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.Usage = func() {
		_, _ = fmt.Fprint(out, serviceUsage+"\nFlags:\n")
		fs.PrintDefaults()
	}

	flags := bindServiceFlags(fs)

	positional, err := parseInterleaved(fs, args)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	if len(positional) > 0 {
		return fmt.Errorf("%w: unexpected arguments %v, use -name and -project", errUsage, positional)
	}

	projectRoot, module, err := findProject(flags.project)
	if err != nil {
		return err
	}

	spec, err := askService(flags, module, newPrompter(in, out))
	if err != nil {
		return err
	}

	err = ensureServiceDir(filepath.Join(projectRoot, spec.Dir()), flags.force)
	if err != nil {
		return err
	}

	files, err := generateService(spec, projectRoot)
	if err != nil {
		return err
	}

	if !flags.skipTidy {
		tidy(out, projectRoot)
	}

	reportService(out, spec, projectRoot, files)

	return nil
}

// askService collects the answers, skipping the steps already given as flags.
func askService(flags *serviceFlags, module string, pr *prompter) (serviceSpec, error) {
	name, err := answer(pr, flags.name,
		"Service name (english letters and digits, singular, e.g. Order)", "", normalizeEntityName)
	if err != nil {
		return serviceSpec{}, err
	}

	withDB, err := resolveDB(flags.db, pr)
	if err != nil {
		return serviceSpec{}, err
	}

	methods, err := answer(pr, flags.methods,
		"CRUD methods (create, read, list, update, delete)", "all", normalizeMethods)
	if err != nil {
		return serviceSpec{}, err
	}

	spec, err := newServiceSpec(module, name, methods, withDB)
	if err != nil {
		return serviceSpec{}, err
	}

	return spec, nil
}

func resolveDB(flagValue string, pr *prompter) (bool, error) {
	if flagValue == "" {
		return pr.askYesNo("Does the service work with the database (PostgreSQL)?", false)
	}

	withDB, ok := parseYesNo(flagValue)
	if !ok {
		return false, fmt.Errorf("%w: %w, got %q", errUsage, errDBAnswer, flagValue)
	}

	return withDB, nil
}

// findProject walks up from the given directory looking for the go.mod of the
// project and returns its root and module path.
func findProject(start string) (string, string, error) {
	dir, err := normalizePath(start)
	if err != nil {
		return "", "", err
	}

	for {
		content, err := os.ReadFile(filepath.Join(dir, "go.mod")) //nolint:gosec // the path is chosen by the author
		if err == nil {
			module := moduleRe.FindSubmatch(content)
			if module == nil {
				return "", "", fmt.Errorf("%w: %s", errNotAModule, filepath.Join(dir, "go.mod"))
			}

			return dir, string(module[1]), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", fmt.Errorf("%w", errNoService)
		}

		dir = parent
	}
}

func ensureServiceDir(dir string, force bool) error {
	_, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("read service directory: %w", err)
	}

	if !force {
		return fmt.Errorf("%w: %s", errServiceExist, dir)
	}

	return nil
}

func reportService(out io.Writer, spec serviceSpec, projectRoot string, files []string) {
	storage := "in memory"
	if spec.DB {
		storage = "with the " + spec.LowerPlural + " table"
	}

	methods := make([]string, 0, len(spec.Methods))
	for _, method := range spec.Methods {
		methods = append(methods, method.Name)
	}

	_, _ = fmt.Fprintf(out, "Added service %q (%s) to %s\n", spec.Name, storage, projectRoot)

	for _, file := range files {
		_, _ = fmt.Fprintln(out, "  "+file)
	}

	_, _ = fmt.Fprintln(out, "\nMethods: "+strings.Join(methods, ", "))
	printServiceNextSteps(out, spec)
}

func printServiceNextSteps(out io.Writer, spec serviceSpec) {
	_, _ = fmt.Fprintf(out, `
Next steps:
  1. register the service in server.go:

         import (
             %s "%s/%s"
             "%s/%s/%s-client"
         )

         %s.RegisterServer(%s)

  2. task gen%s
`,
		spec.Package, spec.Module, spec.Dir(),
		spec.Module, spec.Dir(), spec.Lower,
		spec.Lower, spec.Constructor(),
		sqlNextSteps(spec),
	)
}

func sqlNextSteps(spec serviceSpec) string {
	if !spec.DB {
		return "        # regenerate bind-gen.go"
	}

	return "        # regenerate bind-gen.go and the sqlc queries\n  3. task migrate-up  # create the " +
		spec.LowerPlural + " table"
}
