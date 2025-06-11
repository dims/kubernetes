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
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Diff returns a human-readable report of the differences between two values.
// It returns an empty string if and only if the values are equal.
//
// The output is displayed as a literal in pseudo-Go syntax.
// At the start of each line, a "-" prefix indicates an element removed from a,
// a "+" prefix indicates an element added from b, and the lack of a prefix
// indicates an element common to both a and b.
//
// This function is designed to match the output format of github.com/google/go-cmp/cmp.Diff.
func Diff(a, b any) string {
	if reflect.DeepEqual(a, b) {
		return ""
	}

	var sb strings.Builder
	diffValues(&sb, reflect.ValueOf(a), reflect.ValueOf(b), "", make(map[uintptr]bool), make(map[uintptr]bool))
	return sb.String()
}

// diffMode represents the mode of a diff line.
type diffMode int

const (
	diffIdentical diffMode = iota // Represents an identical line
	diffRemoved                   // Represents a line removed from a
	diffInserted                  // Represents a line added to b
)

// diffContext represents the context information for a diff operation
type diffContext struct {
	path     string
	visitedA map[uintptr]bool
	visitedB map[uintptr]bool
	sb       *strings.Builder
}

// diffValues compares two values and writes the differences to the string builder.
func diffValues(sb *strings.Builder, a, b reflect.Value, path string, visitedA, visitedB map[uintptr]bool) {
	ctx := &diffContext{
		path:     path,
		visitedA: visitedA,
		visitedB: visitedB,
		sb:       sb,
	}

	// Handle invalid values (nil)
	if !a.IsValid() || !b.IsValid() {
		if a.IsValid() != b.IsValid() {
			if a.IsValid() {
				ctx.writeType(a.Type(), diffRemoved)
				ctx.sb.WriteString("(\n")
				ctx.writeValue(a, diffRemoved, 1)
				ctx.sb.WriteString(")")
			} else {
				ctx.writeType(b.Type(), diffInserted)
				ctx.sb.WriteString("(\n")
				ctx.writeValue(b, diffInserted, 1)
				ctx.sb.WriteString(")")
			}
		}
		return
	}

	// Different types
	if a.Type() != b.Type() {
		ctx.writeType(a.Type(), diffRemoved)
		ctx.sb.WriteString("(\n")
		ctx.writeValue(a, diffRemoved, 1)
		ctx.sb.WriteString(")\n")
		ctx.writeType(b.Type(), diffInserted)
		ctx.sb.WriteString("(\n")
		ctx.writeValue(b, diffInserted, 1)
		ctx.sb.WriteString(")")
		return
	}

	// Handle nil slices vs empty slices and nil maps vs empty maps
	switch a.Kind() {
	case reflect.Slice, reflect.Map:
		if (a.IsNil() && !b.IsNil() && b.Len() == 0) || (!a.IsNil() && a.Len() == 0 && b.IsNil()) {
			return
		}
	}

	// Handle based on kind
	switch a.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		if a.Interface() != b.Interface() {
			ctx.writeType(a.Type(), diffIdentical)
			ctx.sb.WriteString("(\n")
			ctx.writeValue(a, diffRemoved, 1)
			ctx.writeValue(b, diffInserted, 1)
			ctx.sb.WriteString(")")
		}

	case reflect.Ptr, reflect.Interface:
		if a.IsNil() || b.IsNil() {
			if a.IsNil() != b.IsNil() {
				if a.Kind() == reflect.Ptr {
					ctx.writeType(a.Type(), diffIdentical)
				} else {
					ctx.sb.WriteString("  any")
				}
				ctx.sb.WriteString("(\n")
				if a.IsNil() {
					ctx.writeLine(diffRemoved, 1, "nil")
					ctx.writeValue(b, diffInserted, 1)
				} else {
					ctx.writeValue(a, diffRemoved, 1)
					ctx.writeLine(diffInserted, 1, "nil")
				}
				ctx.sb.WriteString(")")
			}
			return
		}

		// Check for cycles
		if a.Kind() == reflect.Ptr {
			ptrA, ptrB := a.Pointer(), b.Pointer()
			if ctx.visitedA[ptrA] && ctx.visitedB[ptrB] {
				return
			}
			ctx.visitedA[ptrA] = true
			ctx.visitedB[ptrB] = true
			defer func() {
				delete(ctx.visitedA, ptrA)
				delete(ctx.visitedB, ptrB)
			}()
		}

		// For pointers, print the type with & prefix
		if a.Kind() == reflect.Ptr {
			if !reflect.DeepEqual(a.Elem().Interface(), b.Elem().Interface()) {
				ctx.sb.WriteString("  &")
				if a.Type().Elem().Name() != "" {
					ctx.sb.WriteString(a.Type().Elem().String())
				}
				ctx.sb.WriteString("{\n")
				diffValues(ctx.sb, a.Elem(), b.Elem(), ctx.path, ctx.visitedA, ctx.visitedB)
				ctx.sb.WriteString("  }")
			}
		} else {
			diffValues(ctx.sb, a.Elem(), b.Elem(), ctx.path, ctx.visitedA, ctx.visitedB)
		}

	case reflect.Struct:
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			ctx.sb.WriteString("  " + a.Type().String() + "{\n")

			// First, collect all fields for context
			type fieldInfo struct {
				name     string
				fieldA   reflect.Value
				fieldB   reflect.Value
				exported bool
				changed  bool
			}

			fields := make([]fieldInfo, 0, a.NumField())
			for i := 0; i < a.NumField(); i++ {
				field := a.Type().Field(i)
				fieldA := a.Field(i)
				fieldB := b.Field(i)
				exported := field.IsExported()
				changed := exported && !reflect.DeepEqual(fieldA.Interface(), fieldB.Interface())

				fields = append(fields, fieldInfo{
					name:     field.Name,
					fieldA:   fieldA,
					fieldB:   fieldB,
					exported: exported,
					changed:  changed,
				})
			}

			// Show context and changes
			for _, field := range fields {
				if !field.exported {
					continue
				}

				if field.changed {
					// Field changed
					ctx.writeLine(diffRemoved, 1, fmt.Sprintf("%s: %s,", field.name, formatValue(field.fieldA)))
					ctx.writeLine(diffInserted, 1, fmt.Sprintf("%s: %s,", field.name, formatValue(field.fieldB)))
				} else if !field.fieldA.IsZero() {
					// Field unchanged and non-zero - show as context
					ctx.writeLine(diffIdentical, 1, fmt.Sprintf("%s: %s,", field.name, formatValue(field.fieldA)))
				}
			}

			ctx.sb.WriteString("  }")
		}

	case reflect.Slice, reflect.Array:
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			ctx.sb.WriteString("  " + a.Type().String() + "{\n")

			// Find the maximum length
			maxLen := a.Len()
			if b.Len() > maxLen {
				maxLen = b.Len()
			}

			// Show context and changes
			for i := 0; i < maxLen; i++ {
				if i < a.Len() && i < b.Len() {
					// Both slices have this element
					elemA := a.Index(i)
					elemB := b.Index(i)

					if !reflect.DeepEqual(elemA.Interface(), elemB.Interface()) {
						// Elements differ
						ctx.writeLine(diffRemoved, 1, formatValue(elemA)+",")
						ctx.writeLine(diffInserted, 1, formatValue(elemB)+",")
					} else {
						// Elements are the same - show as context
						ctx.writeLine(diffIdentical, 1, formatValue(elemA)+",")
					}
				} else if i < a.Len() {
					// Element only in a
					ctx.writeLine(diffRemoved, 1, formatValue(a.Index(i))+",")
				} else {
					// Element only in b
					ctx.writeLine(diffInserted, 1, formatValue(b.Index(i))+",")
				}
			}

			ctx.sb.WriteString("  }")
		}

	case reflect.Map:
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			ctx.sb.WriteString("  " + a.Type().String() + "{\n")

			// Get all keys from both maps
			keys := make(map[interface{}]bool)
			for _, k := range a.MapKeys() {
				keys[k.Interface()] = true
			}
			for _, k := range b.MapKeys() {
				keys[k.Interface()] = true
			}

			// Sort keys for deterministic output
			sortedKeys := make([]reflect.Value, 0, len(keys))
			for k := range keys {
				sortedKeys = append(sortedKeys, reflect.ValueOf(k))
			}
			sort.Slice(sortedKeys, func(i, j int) bool {
				return fmt.Sprintf("%v", sortedKeys[i].Interface()) < fmt.Sprintf("%v", sortedKeys[j].Interface())
			})

			// Show context and changes
			for _, k := range sortedKeys {
				keyStr := formatMapKey(k)

				aValue := a.MapIndex(k)
				bValue := b.MapIndex(k)

				if !aValue.IsValid() {
					// Key only in b
					ctx.writeLine(diffInserted, 1, fmt.Sprintf("%s: %s,", keyStr, formatValue(bValue)))
				} else if !bValue.IsValid() {
					// Key only in a
					ctx.writeLine(diffRemoved, 1, fmt.Sprintf("%s: %s,", keyStr, formatValue(aValue)))
				} else if !reflect.DeepEqual(aValue.Interface(), bValue.Interface()) {
					// Values differ
					ctx.writeLine(diffRemoved, 1, fmt.Sprintf("%s: %s,", keyStr, formatValue(aValue)))
					ctx.writeLine(diffInserted, 1, fmt.Sprintf("%s: %s,", keyStr, formatValue(bValue)))
				} else {
					// Values are the same - show as context
					ctx.writeLine(diffIdentical, 1, fmt.Sprintf("%s: %s,", keyStr, formatValue(aValue)))
				}
			}

			ctx.sb.WriteString("  }")
		}

	default:
		// For other types, use reflect.DeepEqual
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			ctx.writeType(a.Type(), diffIdentical)
			ctx.sb.WriteString("(\n")
			ctx.writeValue(a, diffRemoved, 1)
			ctx.writeValue(b, diffInserted, 1)
			ctx.sb.WriteString(")")
		}
	}
}

// writeType writes the type name with the appropriate diff prefix
func (ctx *diffContext) writeType(t reflect.Type, mode diffMode) {
	prefix := "  "
	switch mode {
	case diffRemoved:
		prefix = "- "
	case diffInserted:
		prefix = "+ "
	}
	ctx.sb.WriteString(prefix + t.String())
}

// writeValue writes a value with the appropriate diff prefix and indentation
func (ctx *diffContext) writeValue(v reflect.Value, mode diffMode, indent int) {
	prefix := "  "
	switch mode {
	case diffRemoved:
		prefix = "- "
	case diffInserted:
		prefix = "+ "
	}

	// Add indentation
	for i := 0; i < indent; i++ {
		prefix += "\t"
	}

	// Format the value
	if !v.IsValid() {
		ctx.sb.WriteString(prefix + "nil,\n")
		return
	}

	switch v.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		ctx.sb.WriteString(fmt.Sprintf("%s%v,\n", prefix, v.Interface()))
	case reflect.String:
		ctx.sb.WriteString(fmt.Sprintf("%s%s,\n", prefix, formatString(v.String())))
	case reflect.Ptr:
		if v.IsNil() {
			ctx.sb.WriteString(prefix + "nil,\n")
		} else {
			ctx.sb.WriteString(fmt.Sprintf("%s&%v,\n", prefix, v.Elem().Interface()))
		}
	default:
		ctx.sb.WriteString(fmt.Sprintf("%s%v,\n", prefix, v.Interface()))
	}
}

// writeLine writes a line with the appropriate diff prefix and indentation
func (ctx *diffContext) writeLine(mode diffMode, indent int, text string) {
	prefix := "  "
	switch mode {
	case diffRemoved:
		prefix = "- "
	case diffInserted:
		prefix = "+ "
	}

	// Add indentation
	for i := 0; i < indent; i++ {
		prefix += "\t"
	}

	ctx.sb.WriteString(prefix + text + "\n")
}

// formatValue returns a string representation of the value with type information
func formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return "nil"
	}

	switch v.Kind() {
	case reflect.String:
		return fmt.Sprintf("%s(%s)", v.Type(), formatString(v.String()))
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return "nil"
		}
		if v.Kind() == reflect.Ptr {
			return fmt.Sprintf("*%s(%v)", v.Type().Elem(), v.Elem().Interface())
		}
		return fmt.Sprintf("%s(%v)", v.Type(), v.Elem().Interface())
	case reflect.Struct:
		if v.Type().String() == "time.Time" {
			// Special handling for time.Time
			if m := v.MethodByName("String"); m.IsValid() {
				return fmt.Sprintf("s%v", m.Call(nil)[0])
			}
		}
		return fmt.Sprintf("%s(%v)", v.Type(), v.Interface())
	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			return fmt.Sprintf("%v{}", v.Type())
		}
		return fmt.Sprintf("%s%v", v.Type(), v.Interface())
	case reflect.Map:
		if v.Len() == 0 {
			return fmt.Sprintf("%v{}", v.Type())
		}
		return fmt.Sprintf("%s%v", v.Type(), v.Interface())
	default:
		return fmt.Sprintf("%s(%v)", v.Type(), v.Interface())
	}
}

// formatMapKey formats a map key for display.
func formatMapKey(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return formatString(v.String())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// formatString formats a string for display, using quotes or backticks as appropriate.
func formatString(s string) string {
	// Use quoted string if it the same length as a raw string literal.
	// Otherwise, attempt to use the raw string form.
	qs := strconv.Quote(s)
	if len(qs) == 1+len(s)+1 {
		return qs
	}

	// Disallow newlines to ensure output is a single line.
	// Only allow printable runes for readability purposes.
	rawInvalid := func(r rune) bool {
		return r == '`' || r == '\n' || !(unicode.IsPrint(r) || r == '\t')
	}
	if utf8.ValidString(s) && strings.IndexFunc(s, rawInvalid) < 0 {
		return "`" + s + "`"
	}
	return qs
}

// joinPath joins a parent path with a field name
func joinPath(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

// For backward compatibility, we also provide the original functions
// These functions are used by the tests and may be used by existing code

// ObjectDiff computes a diff between two objects and returns it as a string.
// This is a convenience wrapper around Diff.
func ObjectDiff(a, b interface{}) string {
	return Diff(a, b)
}

// StringDiff computes a diff between two strings and returns it as a string.
// This is a convenience wrapper around Diff.
func StringDiff(a, b string) string {
	return Diff(a, b)
}
