/*
Copyright 2026 The Kubernetes Authors.

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

package json

import (
	"encoding/json"
	"io"

	"sigs.k8s.io/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// streamYAMLEncodeCollections writes obj as YAML, streaming list items so the whole collection is
// never buffered. Output matches yaml.Marshal(obj). Returns false if obj isn't a streamable list.
//
// Only typed lists stream; unstructured lists fall back (yaml.v2's numeric-aware key sort makes
// splicing items into the envelope hard). TODO: stream unstructured lists too.
func streamYAMLEncodeCollections(obj runtime.Object, w io.Writer) (bool, error) {
	if _, ok := obj.(*unstructured.UnstructuredList); ok {
		return false, nil
	}
	// json.Marshaler types are encoded whole by yaml.Marshal (json.Marshal + JSONToYAML); don't stream.
	if _, ok := obj.(json.Marshaler); ok {
		return false, nil
	}
	// getListMeta rejects []runtime.RawExtension lists (raw content type would be lost) -> fall back.
	typeMeta, listMeta, items, err := getListMeta(obj)
	if err != nil {
		return false, nil
	}
	return true, streamYAMLList(typeMeta, listMeta, items, w)
}

// streamYAMLList writes a typed list as YAML. yaml.Marshal sorts keys: apiVersion, items, kind,
// metadata; kind and apiVersion are omitted when empty (json omitempty).
func streamYAMLList(typeMeta metav1.TypeMeta, listMeta metav1.ListMeta, items []runtime.Object, w io.Writer) error {
	if typeMeta.APIVersion != "" {
		if err := writeYAMLMapEntry(w, "apiVersion", typeMeta.APIVersion); err != nil {
			return err
		}
	}
	if err := writeYAMLItems(w, items); err != nil {
		return err
	}
	if typeMeta.Kind != "" {
		if err := writeYAMLMapEntry(w, "kind", typeMeta.Kind); err != nil {
			return err
		}
	}
	return writeYAMLMapEntry(w, "metadata", listMeta)
}

// writeYAMLItems writes "items": "null" for a nil slice, "[]" for empty, else each item as a
// one-element sequence so it renders at the same indentation and wrapping it has in the full list.
func writeYAMLItems(w io.Writer, items []runtime.Object) error {
	switch {
	case items == nil:
		_, err := io.WriteString(w, "items: null\n")
		return err
	case len(items) == 0:
		_, err := io.WriteString(w, "items: []\n")
		return err
	}
	if _, err := io.WriteString(w, "items:\n"); err != nil {
		return err
	}
	for _, item := range items {
		b, err := yaml.Marshal([]interface{}{item})
		if err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	return nil
}

// writeYAMLMapEntry marshals one key; a single-pair map yields that key's bytes in the full doc.
func writeYAMLMapEntry(w io.Writer, key string, value interface{}) error {
	b, err := yaml.Marshal(map[string]interface{}{key: value})
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
