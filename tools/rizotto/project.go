package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const (
	fallbackGoVersion = "1.25.0"
	rizottoModule     = "github.com/geruz/rizotto"
)

// project is the data every template is rendered with.
type project struct {
	// Name is the project name: english letters and digits only, so that it can be
	// used as an identifier in the generated code.
	Name string
	// Slug is the lowercased name, used for hosts, env values and the database name.
	Slug string
	// Repo is the git repository of the project, normalised to a go module path.
	Repo string
	// Module is the go module path of the generated project; it follows the repository.
	Module string
	// GoVersion goes into the go directive of the generated go.mod.
	GoVersion string
	// RizottoPath, when set, produces a replace directive pointing to a local checkout.
	RizottoPath string
	// RizottoModule is the import path of the framework.
	RizottoModule string
	// SQL tells whether the project needs PostgreSQL.
	SQL bool
}

var (
	// repoRe matches a host with a domain suffix followed by at least one path element.
	repoRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*(\.[a-zA-Z0-9-]+)*\.[a-zA-Z]{2,}(/[a-zA-Z0-9._~-]+)+$`)

	errInvalidRepo = errors.New(
		"expected a repository like github.com/acme/shop, https://github.com/acme/shop.git or git@github.com:acme/shop.git",
	)
	errRepoPort = errors.New("a repository with a port cannot be used as a go module path, remove it")
)

// normalizePath turns the answer of the directory question into an absolute path.
func normalizePath(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(expandHome(path))
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", path, err)
	}

	return absPath, nil
}

func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}

// normalizeRepo turns any of the usual git repository spellings into a go module
// path: git@github.com:acme/shop.git and https://github.com/acme/shop both become
// github.com/acme/shop.
func normalizeRepo(raw string) (string, error) {
	repo := strings.TrimSpace(raw)

	for _, scheme := range []string{"https://", "http://", "ssh://", "git://"} {
		repo = strings.TrimPrefix(repo, scheme)
	}

	if user := strings.Index(repo, "@"); user >= 0 {
		repo = repo[user+1:]
	}

	repo = replaceHostSeparator(repo)
	repo = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(repo, "/"), ".git"), "/")

	if strings.Contains(repo, ":") {
		return "", fmt.Errorf("%w: %q", errRepoPort, raw)
	}

	if !repoRe.MatchString(repo) || !hasOwnerAndName(repo) {
		return "", fmt.Errorf("%w, got %q", errInvalidRepo, raw)
	}

	return repo, nil
}

//nolint:gochecknoglobals // the host set is static
var hostsWithOwner = map[string]bool{
	"github.com": true, "gitlab.com": true, "bitbucket.org": true,
}

// hasOwnerAndName rejects github.com/acme and the like, where the repository name
// is missing.
func hasOwnerAndName(repo string) bool {
	const ownerAndName = 3

	parts := strings.Split(repo, "/")
	if hostsWithOwner[parts[0]] {
		return len(parts) >= ownerAndName
	}

	return true
}

// replaceHostSeparator rewrites the scp-like "host:owner/repo" into "host/owner/repo",
// leaving "host:port/..." alone so that it can be reported as an error.
func replaceHostSeparator(repo string) string {
	host, rest, found := strings.Cut(repo, ":")
	if !found || rest == "" {
		return repo
	}

	port, _, _ := strings.Cut(rest, "/")

	_, err := strconv.Atoi(port)
	if err == nil {
		return repo // the colon introduces a port, keep it so that it can be rejected
	}

	return host + "/" + rest
}

func newProject(name, repo, goVersion, rizottoPath string, withSQL bool) (project, error) {
	name, err := normalizeName(name)
	if err != nil {
		return project{}, err
	}

	module, err := normalizeRepo(repo)
	if err != nil {
		return project{}, err
	}

	if goVersion == "" {
		goVersion = toolchainGoVersion()
	}

	if rizottoPath != "" {
		rizottoPath, err = normalizePath(rizottoPath)
		if err != nil {
			return project{}, err
		}
	}

	return project{
		Name:          name,
		Slug:          strings.ToLower(name),
		Repo:          module,
		Module:        module,
		GoVersion:     goVersion,
		RizottoPath:   rizottoPath,
		RizottoModule: rizottoModule,
		SQL:           withSQL,
	}, nil
}

func toolchainGoVersion() string {
	version := regexp.MustCompile(`^go(\d+\.\d+(\.\d+)?)`).FindStringSubmatch(runtime.Version())
	if version == nil {
		return fallbackGoVersion
	}

	return version[1]
}
