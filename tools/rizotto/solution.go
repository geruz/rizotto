package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// solution is one recurring vertical rizotto knows how to install: not a layer
// like a service or a controller, but the whole set of them that a feature such
// as authentication is made of.
type solution interface {
	// Name is what is typed on the command line, e.g. "oauth".
	Name() string
	// Summary is the single line "solution list" prints.
	Summary() string
	// Usage is the help of "solution add <name>".
	Usage() string
	// Skill is the name of the skills/<name>.md document describing it.
	Skill() string
	// Requires names the solutions this one is built on. They are installed
	// first, with their defaults, unless the project already has them.
	Requires() []string
	// Marker is a file that exists exactly when the solution is installed. It is
	// how a dependency already satisfied is told from a missing one.
	Marker() string
	// BindFlags declares the flags the solution answers its own questions with
	// and returns the struct they are parsed into, which Ask is handed back. The
	// struct is created per command run, so nothing is shared between runs.
	BindFlags(fs *flag.FlagSet) any
	// Ask turns the flags and the answers of the author into what to install.
	// Every question must have a default, because a solution installed as a
	// dependency is given no flags.
	Ask(flags any, pr *prompter, prj projectInfo) (solutionPlan, error)
}

// The names of the solutions, which are also how they refer to each other.
const (
	userSolutionName  = "user"
	oauthSolutionName = "oauth"
	filesSolutionName = "files"
)

// solutions is the registry. Adding a solution means adding a file next to this
// one and one entry here; nothing else in the tool has to know about it.
//
//nolint:gochecknoglobals // the solution registry is static
var solutions = []solution{
	userSolution{},
	oauthSolution{},
	filesSolution{},
}

// solutionPlan describes an installation instead of performing it, so that every
// solution shares the same runner: rendering, conflict checks and the report.
type solutionPlan struct {
	// Data is what the templates listed in Files are rendered with.
	Data any
	// Files are the templates the solution brings itself.
	Files []fileSpec
	// Services are generated the usual way, so they also reach sqlc.yaml.
	Services []serviceSpec
	// Controllers are resolved once their services exist on disk.
	Controllers []controllerIntent
	// Repositories are the sqlc entries of the repositories the solution brings
	// itself, i.e. the ones not belonging to a service of Services.
	Repositories []sqlcEntry
	// Env are the variables the solution needs, printed for .env.example.
	Env []envVar
	// Steps are the wiring lines the author has to apply by hand.
	Steps []string
}

// The steps every solution ends with, in the order they have to be applied.
const (
	stepEnvExample = "add the variables above to .env.example"
	stepGen        = "task gen        # regenerates bind-gen.go from the contract and runs sqlc"
)

// envVar is one configuration value a solution reads at startup.
type envVar struct {
	Key     string
	Value   string
	Comment string
}

// controllerIntent is a controller to build once the service it exposes has been
// written: newControllerSpec needs the serviceInfo that only discoverServices
// can produce, and it produces it by parsing the client.go of the service.
type controllerIntent struct {
	Service string
	Name    string
	Methods string
	Prefix  string
	Private bool
}

// projectInfo is what a solution may know about the project it is installed into.
type projectInfo struct {
	// Root is the directory holding go.mod.
	Root string
	// Module is the go module path, which every generated import follows.
	Module string
	// SQL tells whether the project has sqlc.yaml. make-project always writes
	// it, but a project may have lost it or predate that.
	SQL bool
	// Services are the services already present.
	Services []serviceInfo
}

var (
	errUnknownSolution = errors.New("unknown solution")
	// errWrongFlags guards the handover from BindFlags to Ask, which the compiler
	// cannot check because every solution declares its own flags.
	errWrongFlags       = errors.New("the solution was handed the flags of another one")
	errSolutionNeedsSQL = errors.New(
		"this solution owns database tables, but the project has no sqlc.yaml; " +
			"projects made by make-project have one, restore it or add a service with -db yes first")
	errSolutionConflict = errors.New("the solution would overwrite existing files, pass -force to write them anyway")
	errSolutionCycle    = errors.New("the solutions require each other in a circle")
)

// inspectProject finds the project and reads back what it supports, so that a
// solution can refuse to install rather than generate code that cannot compile.
func inspectProject(start string) (projectInfo, error) {
	root, module, err := findProject(start)
	if err != nil {
		return projectInfo{}, err
	}

	_, err = os.Stat(filepath.Join(root, sqlcConfigFile))
	withSQL := err == nil

	// A solution may well be the first thing added to a fresh project, so having
	// no service yet is not an error here.
	services, err := discoverServices(root, module)
	if err != nil && !errors.Is(err, errNoServices) {
		return projectInfo{}, err
	}

	return projectInfo{
		Root:     root,
		Module:   module,
		SQL:      withSQL,
		Services: services,
	}, nil
}

// findSolution returns the solution of that name.
func findSolution(name string) (solution, error) {
	wanted := strings.ToLower(strings.TrimSpace(name))

	for _, sol := range solutions {
		if sol.Name() == wanted {
			return sol, nil
		}
	}

	return nil, fmt.Errorf("%w %q, known solutions: %s", errUnknownSolution, name, solutionNames())
}

// isInstalled reports whether the project already has the solution.
func isInstalled(prj projectInfo, sol solution) bool {
	_, err := os.Stat(filepath.Join(prj.Root, sol.Marker()))

	return err == nil
}

// installOrder lists what has to be installed for sol to work, dependencies
// first, leaving out the ones the project already has. sol itself is always last
// and always included, so re-installing it stays possible with -force.
func installOrder(sol solution, prj projectInfo) ([]solution, error) {
	order := []solution{}
	// visiting carries the current path, which is what turns a cycle into an
	// error naming the two solutions instead of a stack overflow.
	visiting := map[string]bool{}
	done := map[string]bool{}

	var walk func(current solution, root bool) error

	walk = func(current solution, root bool) error {
		if done[current.Name()] {
			return nil
		}

		if visiting[current.Name()] {
			return fmt.Errorf("%w: %s", errSolutionCycle, current.Name())
		}

		visiting[current.Name()] = true

		for _, name := range current.Requires() {
			required, err := findSolution(name)
			if err != nil {
				return fmt.Errorf("%s requires %w", current.Name(), err)
			}

			err = walk(required, false)
			if err != nil {
				return err
			}
		}

		visiting[current.Name()] = false
		done[current.Name()] = true

		if root || !isInstalled(prj, current) {
			order = append(order, current)
		}

		return nil
	}

	err := walk(sol, true)
	if err != nil {
		return nil, err
	}

	return order, nil
}

// solutionNames lists the registry for the error messages and the usage.
func solutionNames() string {
	names := make([]string, 0, len(solutions))
	for _, sol := range solutions {
		names = append(names, sol.Name())
	}

	return strings.Join(names, ", ")
}

// conflicts returns the files of the plan that the project already has, so that a
// solution never silently overwrites hand written code.
func conflicts(root string, files []fileSpec) []string {
	found := []string{}

	for _, spec := range files {
		_, err := os.Stat(filepath.Join(root, spec.out))
		if err == nil {
			found = append(found, spec.out)
		}
	}

	sort.Strings(found)

	return found
}

// planFiles lists every file the plan writes, the ones coming from the services
// and the controllers included.
func planFiles(plan solutionPlan, prj projectInfo) ([]fileSpec, error) {
	files := append([]fileSpec{}, plan.Files...)

	for _, spec := range plan.Services {
		files = append(files, spec.files()...)
	}

	specs, err := resolveControllers(plan.Controllers, prj)
	if err != nil {
		return nil, err
	}

	for _, spec := range specs {
		files = append(files, spec.files()...)
	}

	return files, nil
}

// resolveControllers turns the intents into specs against the services the
// project has right now. It answers an empty list before the services of the
// plan are written, which is why installSolution calls it again afterwards.
func resolveControllers(intents []controllerIntent, prj projectInfo) ([]controllerSpec, error) {
	specs := make([]controllerSpec, 0, len(intents))

	for _, intent := range intents {
		service, err := findService(prj.Services, intent.Service)
		if err != nil {
			return nil, err
		}

		spec, err := newControllerSpec(prj.Module, intent.Name, service, intent.Methods, intent.Prefix, intent.Private)
		if err != nil {
			return nil, err
		}

		specs = append(specs, spec)
	}

	return specs, nil
}

// installSolution writes the plan into the project and returns what it wrote.
func installSolution(plan solutionPlan, prj projectInfo) ([]string, error) {
	written, err := generate(plan.Data, plan.Files, prj.Root)
	if err != nil {
		return nil, err
	}

	for _, spec := range plan.Services {
		service, err := generateService(spec, prj.Root)
		if err != nil {
			return nil, err
		}

		written = append(written, service...)
	}

	for _, entry := range plan.Repositories {
		changed, err := addSqlcEntry(prj.Root, entry)
		if err != nil {
			return nil, err
		}

		if changed {
			written = append(written, sqlcConfigFile)
		}
	}

	files, err := installControllers(plan, prj)
	if err != nil {
		return nil, err
	}

	written = append(written, files...)

	sort.Strings(written)

	return slices.Compact(written), nil
}

// installControllers generates the controllers of the plan. The services of the
// plan are on disk by now, so the discovery sees them.
func installControllers(plan solutionPlan, prj projectInfo) ([]string, error) {
	if len(plan.Controllers) == 0 {
		return nil, nil
	}

	services, err := discoverServices(prj.Root, prj.Module)
	if err != nil {
		return nil, err
	}

	prj.Services = services

	specs, err := resolveControllers(plan.Controllers, prj)
	if err != nil {
		return nil, err
	}

	written := []string{}

	for _, spec := range specs {
		files, err := generateController(spec, prj.Root)
		if err != nil {
			return nil, err
		}

		written = append(written, files...)
	}

	return written, nil
}
