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

// Package sortedfeatures implements a linter that checks if feature gates are sorted alphabetically.
package sortedfeatures

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/pmezard/go-difflib/difflib"
	"golang.org/x/tools/go/analysis"
)

// List of default files to check for feature gate sorting
var defaultTargetFiles = []string{
	"pkg/features/kube_features.go",
	"staging/src/k8s.io/apiserver/pkg/features/kube_features.go",
	"staging/src/k8s.io/client-go/features/known_features.go",
	"staging/src/k8s.io/controller-manager/pkg/features/kube_features.go",
	"staging/src/k8s.io/apiextensions-apiserver/pkg/features/kube_features.go",
	"test/e2e/feature/feature.go",
	"test/e2e/environment/environment.go",
}

// Config holds the configuration for the sortedfeatures analyzer
type Config struct {
	// Files contains files to check. If specified, only these files will be checked.
	Files []string
	// Debug enables debug logging
	Debug bool
}

// NewAnalyzer returns a new sortedfeatures analyzer.
func NewAnalyzer() *analysis.Analyzer {
	return NewAnalyzerWithConfig(Config{})
}

// NewAnalyzerWithConfig returns a new sortedfeatures analyzer with the given configuration.
func NewAnalyzerWithConfig(config Config) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "sortedfeatures",
		Doc:  "Checks if feature gates are sorted alphabetically in const and var blocks",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			return run(pass, config)
		},
	}
}

func run(pass *analysis.Pass, config Config) (interface{}, error) {
	// Check if there are any files to analyze
	if len(pass.Files) == 0 {
		// No files to analyze, return early
		return nil, nil
	}

	// Check if the current file is one of our target files
	filename := pass.Fset.File(pass.Files[0].Pos()).Name()
	isTargetFile := false

	// Determine which files to check
	var targetFiles []string
	if len(config.Files) > 0 {
		// If specific files are provided, only check those
		targetFiles = config.Files
	} else {
		// Otherwise use the default target files
		targetFiles = defaultTargetFiles
	}

	if config.Debug {
		fmt.Printf("Checking file: %s\n", filename)
	}

	for _, target := range targetFiles {
		if strings.HasSuffix(filename, target) || strings.HasSuffix(filename, filepath.Base(target)) {
			isTargetFile = true
			break
		}
	}

	if !isTargetFile {
		return nil, nil
	}

	for _, file := range pass.Files {
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
			features := extractFeatures(genDecl, file.Comments)
			
			// Skip if no features were found
			if len(features) <= 1 {
				continue
			}

			// Sort features
			sortedFeatures := sortFeatures(features)

			// Check if the order has changed
			orderChanged := hasOrderChanged(features, sortedFeatures)

			if orderChanged {
				// Generate a diff to show what's wrong
				reportSortingIssue(pass, genDecl, features, sortedFeatures)
			}
		}
	}
	return nil, nil
}

// Feature represents a feature declaration with its associated comments
type Feature struct {
	Name     string   // Name of the feature
	Comments []string // Comments associated with the feature
}

// extractFeatures extracts features from a GenDecl
func extractFeatures(decl *ast.GenDecl, comments []*ast.CommentGroup) []Feature {
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

		features = append(features, Feature{
			Name:     name,
			Comments: featureComments,
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

// reportSortingIssue reports a linting issue with a diff showing the correct order
func reportSortingIssue(pass *analysis.Pass, decl *ast.GenDecl, current, sorted []Feature) {
	// Configure spew for better output
	spewConfig := spew.ConfigState{
		Indent:                  "  ",
		DisablePointerAddresses: true,
		DisableCapacities:       true,
		SortKeys:                true,
	}
	
	// Generate dumps of both current and expected orders
	currentDump := spewConfig.Sdump(current)
	sortedDump := spewConfig.Sdump(sorted)
	
	// Create a unified diff between the two dumps
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(currentDump),
		B:        difflib.SplitLines(sortedDump),
		FromFile: "Current Order",
		ToFile:   "Expected Order",
		Context:  3,
	}

	diffText, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		pass.Reportf(decl.Pos(), "feature gates are not sorted alphabetically (error creating diff: %v)", err)
		return
	}

	// Report the issue with the diff
	pass.Reportf(decl.Pos(), "feature gates are not sorted alphabetically:\n%s\nRun hack/update-sortfeatures.sh to fix", diffText)
}
