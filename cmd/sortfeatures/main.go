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

// Package main implements a tool that fixes feature gates sorting order.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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

// FeatureEntry represents a feature gate with its associated comments
type FeatureEntry struct {
	Name       string   // Name of the feature gate
	Comments   []string // Comments associated with the feature gate
	Definition string   // The feature gate definition line
}

// FixFile sorts feature gates in const and var blocks in the given file
func FixFile(filename string) error {
	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Split the content into lines
	lines := strings.Split(string(content), "\n")

	// Find const and var blocks
	constVarRegex := regexp.MustCompile(`^(const|var)\s+\($`)
	closingParenRegex := regexp.MustCompile(`^\)$`)

	modified := false
	inBlock := false
	blockStart := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check if this is the start of a const or var block
		if constVarRegex.FindStringSubmatch(line) != nil {
			inBlock = true
			blockStart = i
			continue
		}

		// Check if this is the end of a block
		if inBlock && closingParenRegex.MatchString(strings.TrimSpace(line)) {
			// Process the block
			if newBlock, changed := processBlock(lines[blockStart : i+1]); changed {
				modified = true

				// Replace the block in the lines
				newLines := make([]string, 0, len(lines)-(i+1-blockStart)+len(newBlock))
				newLines = append(newLines, lines[:blockStart]...)
				newLines = append(newLines, newBlock...)
				newLines = append(newLines, lines[i+1:]...)

				lines = newLines
				i = blockStart + len(newBlock) - 1 // Adjust the index
			}

			inBlock = false
		}
	}

	// If no changes were made, we're done
	if !modified {
		fmt.Printf("No changes needed for %s\n", filename)
		return nil
	}

	// Write the modified file
	if err := os.WriteFile(filename, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Successfully fixed sorting in %s\n", filename)
	return nil
}

// processBlock processes a const or var block and returns a sorted version if needed
func processBlock(block []string) ([]string, bool) {
	if len(block) < 3 {
		return block, false // Not enough lines for a block with features
	}

	// Extract feature entries
	var features []FeatureEntry
	var blockHeaderComments []string
	var currentComments []string
	var inBlockHeader = true
	var lastFeatureIndex = -1

	featureRegex := regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s*(?:=\s*.*Feature|.*Feature\s*=)`)

	// First pass: identify block header comments and feature gates
	for i := 1; i < len(block)-1; i++ {
		line := block[i]
		trimmedLine := strings.TrimSpace(line)

		// Check if this is a feature gate
		if matches := featureRegex.FindStringSubmatch(line); matches != nil {
			inBlockHeader = false
			lastFeatureIndex = i
			// We'll process feature gates in the second pass
			continue
		}

		// Check if this is a comment
		if strings.HasPrefix(trimmedLine, "//") {
			if inBlockHeader {
				// Only add to block header comments if we haven't seen a feature gate yet
				blockHeaderComments = append(blockHeaderComments, line)
			}
			continue
		}

		// If we encounter a non-comment, non-feature gate line, we're no longer in the block header
		if trimmedLine != "" {
			inBlockHeader = false
		}
	}

	// Reset for second pass
	inBlockHeader = true
	currentComments = nil
	seenFeatures := make(map[string]bool)

	// Second pass: associate comments with feature gates
	for i := 1; i < len(block)-1; i++ {
		line := block[i]
		trimmedLine := strings.TrimSpace(line)

		// Skip block header comments
		if inBlockHeader && i <= lastFeatureIndex && len(blockHeaderComments) > 0 {
			// Check if this line is a block header comment
			isBlockHeaderComment := false
			for _, headerComment := range blockHeaderComments {
				if line == headerComment {
					isBlockHeaderComment = true
					break
				}
			}
			if isBlockHeaderComment {
				continue
			}
			inBlockHeader = false
		}

		// Check if this is an empty line
		if trimmedLine == "" {
			// Don't reset comments on empty lines
			continue
		}

		// Check if this is a comment
		if strings.HasPrefix(trimmedLine, "//") {
			currentComments = append(currentComments, line)
			continue
		}

		// Check if this is a feature gate
		if matches := featureRegex.FindStringSubmatch(line); matches != nil {
			featureName := matches[1]

			// Skip duplicate feature gates
			if seenFeatures[featureName] {
				currentComments = nil
				continue
			}
			seenFeatures[featureName] = true

			// Create a new feature entry
			feature := FeatureEntry{
				Name:       featureName,
				Comments:   make([]string, len(currentComments)),
				Definition: line,
			}

			// Copy the comments
			copy(feature.Comments, currentComments)

			features = append(features, feature)
			currentComments = nil
		} else {
			// If this is not a feature gate or comment, reset comments
			currentComments = nil
		}
	}

	// If we have less than 2 features, no need to sort
	if len(features) < 2 {
		return block, false
	}

	// Check if the features are already sorted
	isSorted := true
	for i := 1; i < len(features); i++ {
		if strings.Compare(features[i-1].Name, features[i].Name) > 0 {
			isSorted = false
			break
		}
	}

	// Check if there are blank lines between features in the original block
	hasBlankLines := true // Assume there are blank lines by default
	featureIndices := []int{}

	// Find all feature indices
	for i := 1; i < len(block)-1; i++ {
		line := block[i]
		if matches := featureRegex.FindStringSubmatch(line); matches != nil {
			featureIndices = append(featureIndices, i)
		}
	}

	// Check if there's at least one blank line between consecutive features
	for i := 0; i < len(featureIndices)-1; i++ {
		hasBlankLineBetween := false
		for j := featureIndices[i] + 1; j < featureIndices[i+1]; j++ {
			if strings.TrimSpace(block[j]) == "" {
				hasBlankLineBetween = true
				break
			}
		}
		if !hasBlankLineBetween {
			hasBlankLines = false
			break
		}
	}

	// If features are already sorted and have blank lines, no changes needed
	if isSorted && hasBlankLines {
		return block, false
	}

	// Sort the features
	sort.Slice(features, func(i, j int) bool {
		return strings.Compare(features[i].Name, features[j].Name) < 0
	})

	// Reconstruct the block
	var newBlock []string

	// Add the const/var line
	newBlock = append(newBlock, block[0])

	// Process block header comments to remove duplicates and ensure proper formatting
	if len(blockHeaderComments) > 0 {
		// Remove duplicate comments
		uniqueComments := make(map[string]bool)
		var cleanedHeaderComments []string

		for _, comment := range blockHeaderComments {
			if !uniqueComments[comment] {
				uniqueComments[comment] = true
				cleanedHeaderComments = append(cleanedHeaderComments, comment)
			}
		}

		// Add cleaned header comments
		newBlock = append(newBlock, cleanedHeaderComments...)

		// Add a blank line after block header comments
		newBlock = append(newBlock, "")
	}

	for i, feature := range features {
		// Add a blank line before each feature except the first one
		if i > 0 {
			newBlock = append(newBlock, "")
		}

		// Process feature comments to remove duplicates
		if len(feature.Comments) > 0 {
			uniqueComments := make(map[string]bool)
			var cleanedComments []string

			for _, comment := range feature.Comments {
				if !uniqueComments[comment] {
					uniqueComments[comment] = true
					cleanedComments = append(cleanedComments, comment)
				}
			}

			// Add cleaned comments
			newBlock = append(newBlock, cleanedComments...)
		}

		// Add the feature definition
		newBlock = append(newBlock, feature.Definition)
	}

	newBlock = append(newBlock, block[len(block)-1]) // Add the closing parenthesis

	return newBlock, true
}

// FixFiles fixes sorting in the specified files or default target files
func FixFiles(files []string) error {
	// Determine which files to check
	var targetFiles []string
	if len(files) > 0 {
		// If specific files are provided, only check those
		targetFiles = files
	} else {
		// Otherwise use the default target files
		targetFiles = defaultTargetFiles
	}

	// Process each file
	for _, target := range targetFiles {
		// Handle both absolute paths and relative paths
		var filesToProcess []string
		if filepath.IsAbs(target) {
			filesToProcess = []string{target}
		} else {
			// Find files that match the target pattern
			matches, err := filepath.Glob(target)
			if err != nil {
				return fmt.Errorf("failed to glob pattern %s: %w", target, err)
			}

			// If no matches, try to find the file relative to the current directory
			if len(matches) == 0 {
				// Try to find the file in the current directory or subdirectories
				err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() && strings.HasSuffix(path, filepath.Base(target)) {
						filesToProcess = append(filesToProcess, path)
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("failed to walk directory: %w", err)
				}
			} else {
				filesToProcess = matches
			}
		}

		// Process each file
		for _, file := range filesToProcess {
			if err := FixFile(file); err != nil {
				return err
			}
		}
	}

	return nil
}

// Command line tool entry point
func main() {
	var files []string

	// If no arguments are provided, use default target files
	if len(os.Args) < 2 {
		fmt.Println("No files specified, using default target files")
	} else {
		files = os.Args[1:]
	}

	if err := FixFiles(files); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
