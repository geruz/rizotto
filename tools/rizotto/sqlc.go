package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	sqlcConfigFile = "sqlc.yaml"
	sqlcVersion    = "2"
	yamlIndent     = 4
)

var errSqlcConfig = errors.New("sqlc.yaml has an unexpected shape, add the entry by hand")

type sqlcEntry struct {
	Engine  string   `yaml:"engine"`
	Queries string   `yaml:"queries"`
	Schema  []string `yaml:"schema"`
	Gen     sqlcGen  `yaml:"gen"`
}

type sqlcGen struct {
	Go sqlcGo `yaml:"go"`
}

// sqlcGo mirrors the go generator options of sqlc.
//
//nolint:tagliatelle // the keys are defined by sqlc
type sqlcGo struct {
	SQLPackage               string `yaml:"sql_package"`
	Out                      string `yaml:"out"`
	EmitPointersForNullTypes bool   `yaml:"emit_pointers_for_null_types"`
	EmitEnumValidMethod      bool   `yaml:"emit_enum_valid_method"`
}

func newSqlcEntry(spec serviceSpec) sqlcEntry {
	dir := "./" + spec.Dir()

	return sqlcEntry{
		Engine:  "postgresql",
		Queries: dir + "/repository/sql/" + spec.Lower + "-queries.sql",
		Schema:  []string{dir + "/repository/sql/schema.sql"},
		Gen: sqlcGen{
			Go: sqlcGo{
				SQLPackage:               "pgx/v5",
				Out:                      dir + "/repository/db",
				EmitPointersForNullTypes: true,
				EmitEnumValidMethod:      true,
			},
		},
	}
}

// addSqlcEntry registers the queries of the service in sqlc.yaml, creating the file
// when it does not exist yet. The rest of the file, comments included, is kept as is.
// It reports whether the file was written.
func addSqlcEntry(projectRoot string, spec serviceSpec) (bool, error) {
	path := filepath.Join(projectRoot, sqlcConfigFile)
	entry := newSqlcEntry(spec)

	content, err := os.ReadFile(path) //nolint:gosec // the project directory is chosen by the author
	if errors.Is(err, os.ErrNotExist) {
		return true, writeSqlcConfig(path, entry)
	}

	if err != nil {
		return false, fmt.Errorf("read %s: %w", sqlcConfigFile, err)
	}

	var doc yaml.Node

	err = yaml.Unmarshal(content, &doc)
	if err != nil || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return false, fmt.Errorf("%w: %s", errSqlcConfig, path)
	}

	added, err := appendSqlcEntry(doc.Content[0], entry)
	if err != nil || !added {
		return false, err
	}

	return true, writeYAML(path, &doc)
}

// appendSqlcEntry adds the entry to the "sql" sequence of the document, unless the
// same queries file is already registered. It reports whether anything changed.
func appendSqlcEntry(root *yaml.Node, entry sqlcEntry) (bool, error) {
	sequence := findMapValue(root, "sql")
	if sequence == nil {
		sequence = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"} //nolint:exhaustruct_v5 // the encoder fills the rest
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "sql"}, //nolint:exhaustruct_v5 // key node
			sequence,
		)
	}

	if sequence.Kind != yaml.SequenceNode {
		return false, errSqlcConfig
	}

	for _, existing := range sequence.Content {
		queries := findMapValue(existing, "queries")
		if queries != nil && queries.Value == entry.Queries {
			return false, nil
		}
	}

	node := &yaml.Node{} //nolint:exhaustruct_v5 // filled by Encode

	err := node.Encode(entry)
	if err != nil {
		return false, fmt.Errorf("encode sqlc entry: %w", err)
	}

	sequence.Content = append(sequence.Content, node)

	return true, nil
}

func findMapValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}

	return nil
}

func writeSqlcConfig(path string, entries ...sqlcEntry) error {
	config := struct {
		Version string      `yaml:"version"`
		SQL     []sqlcEntry `yaml:"sql"`
	}{Version: sqlcVersion, SQL: entries}

	var doc yaml.Node

	err := doc.Encode(config)
	if err != nil {
		return fmt.Errorf("encode %s: %w", sqlcConfigFile, err)
	}

	return writeYAML(path, &doc)
}

func writeYAML(path string, doc *yaml.Node) error {
	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(yamlIndent)

	err := encoder.Encode(doc)
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}

	err = encoder.Close()
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}

	err = os.WriteFile(path, buf.Bytes(), filePerm)
	if err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}

	return nil
}
