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

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Command line flags
	forceUpdate bool
	files       []string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sortfeatures",
		Short: "Sort feature declarations in Kubernetes feature files",
		Long: `Sort feature declarations in Kubernetes feature files.
This tool parses specified files, finds var/const blocks containing feature declarations,
sorts them alphabetically (case-sensitive), and updates the files if the order has changed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no files are specified via the --files flag, use positional args
			files = append(files, args...)

			if len(files) == 0 {
				return fmt.Errorf("no files specified, use --files flag or provide file paths as arguments")
			}

			for _, filePath := range files {
				if err := processFile(filePath); err != nil {
					return err
				}
			}
			return nil
		},
	}

	rootCmd.Flags().BoolVarP(&forceUpdate, "force", "f", false, "Force update even if the file is already sorted")
	rootCmd.Flags().StringSliceVarP(&files, "files", "", nil, "One or more file paths to process")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// processFile processes a single file
func processFile(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	fmt.Printf("Processing %s\n", filePath)

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Track if any changes were made
	fileChanged := false
	newContent := string(content)

	// Process each declaration in the file
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		// Only process var and const blocks
		if genDecl.Tok != token.VAR && genDecl.Tok != token.CONST {
			continue
		}

		// Skip if there's only one spec (not a block)
		if len(genDecl.Specs) <= 1 {
			continue
		}

		// Extract features with their comments
		features := extractFeatures(genDecl, file.Comments, fset, string(content))

		// Sort features
		sortedFeatures := sortFeatures(features)

		// Check if the order has changed
		orderChanged := hasOrderChanged(features, sortedFeatures)

		// Update the file if the order has changed or force update is enabled
		if orderChanged || forceUpdate {
			// Create a new file with sorted features
			newContent = updateFile(newContent, genDecl, sortedFeatures, fset)

			fileChanged = true
			fmt.Printf("  Reordered %d features in %s block\n", len(sortedFeatures), tokenToString(genDecl.Tok))
		}
	}

	// Write the updated file if changes were made
	if fileChanged {
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Updated %s\n", filePath)
	} else {
		fmt.Printf("No changes needed for %s\n", filePath)
	}

	return nil
}

// tokenToString converts a token to its string representation
func tokenToString(tok token.Token) string {
	switch tok {
	case token.VAR:
		return "var"
	case token.CONST:
		return "const"
	default:
		return tok.String()
	}
}

// Feature represents a feature declaration with its associated comments
type Feature struct {
	Name     string   // Name of the feature
	Comments []string // Comments associated with the feature
	Line     string   // The entire line of the feature declaration
}

// extractFeatures extracts features from a GenDecl
func extractFeatures(decl *ast.GenDecl, comments []*ast.CommentGroup, fset *token.FileSet, content string) []Feature {
	var features []Feature

	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok || len(valueSpec.Names) == 0 {
			continue
		}

		// Get the name of the feature
		name := valueSpec.Names[0].Name

		// Get comments for this feature
		var featureComments []string

		// Check for doc comments directly on the value spec
		if valueSpec.Doc != nil {
			for _, comment := range valueSpec.Doc.List {
				featureComments = append(featureComments, comment.Text)
			}
		} else {
			// Look for comments before this spec
			for _, cg := range comments {
				if cg.End()+1 == valueSpec.Pos() {
					for _, comment := range cg.List {
						featureComments = append(featureComments, comment.Text)
					}
				}
			}
		}

		// Get the entire line of the feature declaration
		specStart := fset.Position(valueSpec.Pos())
		specEnd := fset.Position(valueSpec.End())

		// Find the start of the line
		lineStart := specStart.Offset
		for lineStart > 0 && content[lineStart-1] != '\n' && content[lineStart-1] != '\t' {
			lineStart--
		}

		// Get the entire line
		line := content[lineStart:specEnd.Offset]

		features = append(features, Feature{
			Name:     name,
			Comments: featureComments,
			Line:     line,
		})
	}

	return features
}

// sortFeatures sorts features alphabetically by name
func sortFeatures(features []Feature) []Feature {
	sorted := make([]Feature, len(features))
	copy(sorted, features)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}

// hasOrderChanged checks if the order of features has changed
func hasOrderChanged(original, sorted []Feature) bool {
	if len(original) != len(sorted) {
		return true
	}

	for i := range original {
		if original[i].Name != sorted[i].Name {
			return true
		}
	}

	return false
}

// updateFile creates a new file content with sorted features
func updateFile(content string, decl *ast.GenDecl, sortedFeatures []Feature, fset *token.FileSet) string {
	// Create a buffer for the new content
	var buf strings.Builder

	// Get the position of the declaration in the file
	declPos := fset.Position(decl.Pos())
	declEnd := fset.Position(decl.End())

	// Find the start of the line containing the declaration
	lineStart := declPos.Offset
	for lineStart > 0 && content[lineStart-1] != '\n' {
		lineStart--
	}

	// Write the content up to the start of the line containing the declaration
	buf.WriteString(content[:lineStart])

	// Write the declaration token (var or const)
	buf.WriteString(tokenToString(decl.Tok))
	buf.WriteString(" (")

	// Write each feature in sorted order
	for i, feature := range sortedFeatures {
		buf.WriteString("\n")

		// Write comments
		for _, comment := range feature.Comments {
			buf.WriteString("\t")
			buf.WriteString(comment)
			buf.WriteString("\n")
		}

		// Write the feature declaration
		buf.WriteString("\t")
		buf.WriteString(feature.Line)

		// Add a newline between features
		if i < len(sortedFeatures)-1 {
			buf.WriteString("\n")
		}
	}

	// Close the declaration block
	buf.WriteString("\n)")

	// Write the rest of the file
	buf.WriteString(content[declEnd.Offset:])

	return buf.String()
}
