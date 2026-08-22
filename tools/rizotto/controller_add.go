package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const controllerUsage = `Usage:
	rizotto controller add [flags]
	rizotto controller skill

Adds an HTTP controller to an existing project, asking which service it exposes,
which routes it serves, whether they need an authenticated user and under which
prefix they live. Every answer can be given upfront with a flag (-service, -name,
-methods, -auth, -prefix) and the matching question is then skipped.

"rizotto controller skill" prints how a controller works in detail.
`

var errControllerExist = errors.New("the controller directory already exists, pass -force to add files to it")

type controllerFlags struct {
	name     string
	service  string
	project  string
	methods  string
	auth     string
	prefix   string
	force    bool
	skipTidy bool
}

func bindControllerFlags(fs *flag.FlagSet) *controllerFlags {
	f := &controllerFlags{} //nolint:exhaustruct_v5 // filled by the flag package

	fs.StringVar(&f.name, "name", "", "controller name (default: the name of the service it exposes)")
	fs.StringVar(&f.service, "service", "", "service the controller exposes, e.g. order")
	fs.StringVar(&f.project, "project", "", `path inside the project the controller is added to (default ".")`)
	fs.StringVar(&f.methods, "methods", "",
		`comma separated routes: create, read, list, update, delete, or "all" (default: what the service implements)`)
	fs.StringVar(&f.auth, "auth", "",
		`"yes" when the routes need an authenticated user (PrivateArea), "no" for PublicArea`)
	fs.StringVar(&f.prefix, "prefix", "", "route prefix (default: /api/v1/<plural name>)")
	fs.BoolVar(&f.force, "force", false, "write into an existing controller directory")
	fs.BoolVar(&f.skipTidy, "skip-tidy", false, "do not run go mod tidy in the project")

	return f
}

// runController dispatches the subcommands of "rizotto controller".
func runController(args []string, in io.Reader, out io.Writer) error {
	return dispatchAdd(args, controllerUsage, "controller", out, func(rest []string) error {
		return runControllerAdd(rest, in, out)
	})
}

func runControllerAdd(args []string, in io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("controller add", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.Usage = func() {
		_, _ = fmt.Fprint(out, controllerUsage+"\nFlags:\n")
		fs.PrintDefaults()
	}

	flags := bindControllerFlags(fs)

	positional, err := parseInterleaved(fs, args)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	if len(positional) > 0 {
		return fmt.Errorf("%w: unexpected arguments %v, use -name and -service", errUsage, positional)
	}

	projectRoot, module, err := findProject(flags.project)
	if err != nil {
		return err
	}

	services, err := discoverServices(projectRoot, module)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		return fmt.Errorf("%w: %s", errNoServices, projectRoot)
	}

	spec, err := askController(flags, module, services, newPrompter(in, out))
	if err != nil {
		return err
	}

	err = ensureControllerDir(filepath.Join(projectRoot, spec.Dir()), flags.force)
	if err != nil {
		return err
	}

	files, err := generateController(spec, projectRoot)
	if err != nil {
		return err
	}

	if !flags.skipTidy {
		tidy(out, projectRoot)
	}

	reportController(out, spec, projectRoot, files)

	return nil
}

// askController collects the answers, skipping the steps already given as flags.
func askController(
	flags *controllerFlags,
	module string,
	services []serviceInfo,
	pr *prompter,
) (controllerSpec, error) {
	service, err := askServiceChoice(flags.service, services, pr)
	if err != nil {
		return controllerSpec{}, err
	}

	name, err := answer(pr, flags.name, "Controller name", service.Entity, normalizeEntityName)
	if err != nil {
		return controllerSpec{}, err
	}

	methods, err := answer(pr, flags.methods,
		"Routes (create, read, list, update, delete)", defaultMethods(service), normalizeMethods)
	if err != nil {
		return controllerSpec{}, err
	}

	private, err := resolveAuth(flags.auth, pr)
	if err != nil {
		return controllerSpec{}, err
	}

	prefix, err := answer(pr, flags.prefix, "Route prefix", defaultPrefix(strings.ToLower(name)), normalizePrefix)
	if err != nil {
		return controllerSpec{}, err
	}

	return newControllerSpec(module, name, service, methods, prefix, private)
}

// askServiceChoice asks which of the services of the project the controller exposes.
func askServiceChoice(flagValue string, services []serviceInfo, pr *prompter) (serviceInfo, error) {
	pick := func(raw string) (string, error) {
		service, err := findService(services, raw)
		if err != nil {
			return "", err
		}

		return service.Lower, nil
	}

	if flagValue != "" {
		name, err := pick(flagValue)
		if err != nil {
			return serviceInfo{}, fmt.Errorf("%w: %w", errUsage, err)
		}

		return findService(services, name)
	}

	name, err := pr.askValid("Service to expose ("+serviceNames(services)+")", services[0].Lower, pick)
	if err != nil {
		return serviceInfo{}, err
	}

	return findService(services, name)
}

func resolveAuth(flagValue string, pr *prompter) (bool, error) {
	if flagValue == "" {
		return pr.askYesNo("Do the routes need an authenticated user (PrivateArea)?", true)
	}

	private, ok := parseYesNo(flagValue)
	if !ok {
		return false, fmt.Errorf(`%w: -auth expects "yes" or "no", got %q`, errUsage, flagValue)
	}

	return private, nil
}

func ensureControllerDir(dir string, force bool) error {
	_, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("read controller directory: %w", err)
	}

	if !force {
		return fmt.Errorf("%w: %s", errControllerExist, dir)
	}

	return nil
}

func reportController(out io.Writer, spec controllerSpec, projectRoot string, files []string) {
	_, _ = fmt.Fprintf(out, "Added controller %q (%s) to %s\n", spec.Name, spec.Area, projectRoot)

	for _, file := range files {
		_, _ = fmt.Fprintln(out, "  "+file)
	}

	_, _ = fmt.Fprintln(out, "\nRoutes:")

	for _, route := range spec.Routes {
		_, _ = fmt.Fprintf(out, "  %-42s -> %s\n", route.Route, route.RPC.Name)
	}

	_, _ = fmt.Fprintf(out, `
Next steps:
  1. join the route table in server.go:

         import "%s/%s"

         gt.Routing(
             gateway.JoinRouteTables(
                 %s,
             ),
         )

  2. task test        # runs the route tests and writes %s/openapi.yml
`,
		spec.Module, spec.Dir(), spec.RouteTable(), spec.Dir())
}
