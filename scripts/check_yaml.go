//go:build ignore

// Checked via "go run scripts/check_yaml.go": exact YAML syntax and duplicate
// mapping-key validation, with the YAML parser version pinned in go.mod.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.yaml.in/yaml/v4"
)

func duplicateMapping(n *yaml.Node) error {
	if n.Kind == yaml.MappingNode {
		seen := make(map[string]bool)
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i].Value
			if seen[key] {
				return fmt.Errorf("duplicate YAML mapping key %q at line %d", key, n.Content[i].Line)
			}
			seen[key] = true
		}
	}
	for _, child := range n.Content {
		if err := duplicateMapping(child); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	out, err := exec.Command("git", "ls-files", "-z", "--", "*.yaml", "*.yml").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "YAML FAIL: cannot enumerate tracked files:", err)
		os.Exit(1)
	}
	count := 0
	for _, raw := range bytes.Split(out, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		path := string(raw)
		if strings.HasPrefix(path, "-") {
			fmt.Fprintln(os.Stderr, "YAML FAIL: unsafe path:", path)
			os.Exit(1)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "YAML FAIL:", path, err)
			os.Exit(1)
		}
		var tree yaml.Node
		if err := yaml.Unmarshal(data, &tree); err != nil {
			fmt.Fprintln(os.Stderr, "YAML FAIL:", path, err)
			os.Exit(1)
		}
		if err := duplicateMapping(&tree); err != nil {
			fmt.Fprintln(os.Stderr, "YAML FAIL:", path, err)
			os.Exit(1)
		}
		count++
	}
	fmt.Printf("YAML PASS: %d tracked files parsed and no duplicate keys\n", count)
}
