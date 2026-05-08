package main

import (
	"crm-middleware/cmd/cli"
	"crm-middleware/logger"
	"crm-middleware/model/dataschema"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var stderr = logger.Stderr()

func main() {
	cli.Register(cli.Command("generate", "<file.json|dir/> [outdir/]", "generate JSON schemas from YAML files", generate))
	cli.Register(cli.Command("validate", "<file.json|dir/>", "validate JSON schema files", validate))

	os.Exit(cli.Exec(os.Args...))
}

func generate(args ...string) int {
	path := args[0]
	var outDir string
	if len(args) > 1 {
		outDir = args[1]
		if err := os.MkdirAll(outDir, 0755); err != nil {
			stderr(fmt.Sprintf("mkdir %s: %v", outDir, err))
			return 1
		}
	} else {
		var err error
		if outDir, err = os.Getwd(); err != nil {
			stderr("could not determine working directory:", err)
			return 1
		}
	}

	return runOnFiles(path, ".yaml", func(f string) (string, error) {
		schemaPath := filepath.Join(outDir, strings.TrimSuffix(filepath.Base(f), ".yaml")+".json")
		raw, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		m, err := parseYAML(raw)
		if err != nil {
			return "", err
		}
		if err := dataschema.ValidateModel(m); err != nil {
			return "", err
		}
		schema, err := json.MarshalIndent(m, "", "\t")
		if err != nil {
			return "", err
		}
		return schemaPath, os.WriteFile(schemaPath, append(schema, '\n'), 0644)
	})
}

func validate(args ...string) int {
	return runOnFiles(args[0], ".json", func(f string) (string, error) {
		raw, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		var m dataschema.Model
		if err := json.Unmarshal(raw, &m); err != nil {
			return "", err
		}
		return "", dataschema.ValidateModel(&m)
	})
}
