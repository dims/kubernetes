/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package diff

import (
	"regexp"
	"strings"
	"testing"
)

// normalizeDiff trims and collapses whitespace for robust comparison.
func normalizeDiff(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func checkExpected(t *testing.T, result string, expected interface{}) {
	normResult := normalizeDiff(result)
	if expected == "" && normResult != "" {
		t.Errorf("Expected empty diff, got: %q", result)
	} else if s, ok := expected.(string); ok && s != "" {
		// Allow for type information and context in the output
		if !strings.Contains(result, s) {
			t.Errorf("Expected diff to contain %q, got: %q", s, result)
		}
	} else if strs, ok := expected.([]string); ok {
		for _, s := range strs {
			if !strings.Contains(result, s) {
				t.Errorf("Expected diff to contain %q, got: %q", s, result)
			}
		}
	}

	// Additional check for proper formatting
	if expected != "" && result != "" {
		// Check for proper line prefixes
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			// Allow for context lines without prefixes
			if !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "+ ") && !strings.HasPrefix(line, "  ") {
				t.Errorf("Line doesn't have proper prefix: %q", line)
			}
		}

		// Check for proper path formatting in diff lines
		for _, line := range lines {
			if line == "" || strings.HasPrefix(line, "  ") {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				continue
			}
			path := strings.TrimSpace(parts[0][2:]) // Remove prefix (- or +) and trim
			if path != "." && !strings.HasPrefix(path, ".") {
				t.Errorf("Path doesn't start with dot: %q", path)
			}
		}
	}
}

// removeTypeInfo removes type information from the diff output for comparison
func removeTypeInfo(s string) string {
	// Replace patterns like "int(42)" with just "42"
	re := regexp.MustCompile(`\w+\(([^)]+)\)`)
	s = re.ReplaceAllString(s, "$1")

	// Handle pointer types like "*int(42)" -> "42"
	re = regexp.MustCompile(`\*\w+\(([^)]+)\)`)
	return re.ReplaceAllString(s, "$1")
}
