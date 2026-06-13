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

package cbor

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sort"

	"github.com/fxamacker/cbor/v2"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/cbor/internal/modes"
)

// CBOR initial-byte major type bits (RFC 8949 Section 3.1).
const (
	cborMajorArray byte = 0x80
	cborMajorMap   byte = 0xa0
)

// cborNull is the CBOR "null" simple value (how modes.Encode renders a nil slice).
var cborNull = []byte{0xf6}

// runtime.RawExtension item lists fall back to non-streaming; see getListMeta.
var rawExtensionType = reflect.TypeOf(runtime.RawExtension{})

var (
	errListFieldCount    = errors.New("expected list type to have exactly 3 fields")
	errListTypeMeta      = errors.New("expected first list field to be an embedded TypeMeta")
	errListTypeMetaTag   = errors.New(`expected TypeMeta json field tag to be "" or ",inline"`)
	errListMeta          = errors.New("expected second list field to be a ListMeta")
	errListMetaTag       = errors.New(`expected ListMeta json field tag to be "metadata,omitempty"`)
	errListItemsTag      = errors.New(`expected Items json field tag to be "items"`)
	errRawExtensionItems = errors.New("list Items are runtime.RawExtension, whose content type meta.ExtractList drops")
)

// streamEncodeCollections writes obj as a CBOR list, streaming items one at a time so the whole
// encoded collection is never buffered. Output is identical to encoding obj with modes.Encode.
// Returns (true, err) for a streamable list, or (false, nil) if obj must use the normal path.
func streamEncodeCollections(obj runtime.Object, w io.Writer) (bool, error) {
	if list, ok := obj.(*unstructured.UnstructuredList); ok {
		return true, streamEncodeUnstructuredList(list, w)
	}
	// Types with custom CBOR/JSON marshaling are encoded whole by modes.Encode, not streamed.
	if _, ok := obj.(cbor.Marshaler); ok {
		return false, nil
	}
	if _, ok := obj.(json.Marshaler); ok {
		return false, nil
	}
	typeMeta, listMeta, items, err := getListMeta(obj)
	if err != nil {
		return false, nil
	}
	return true, streamEncodeList(typeMeta, listMeta, items, w)
}

// streamEncodeList writes a typed list as a CBOR map. Keys are emitted in SortBytewiseLexical
// order, which for these byte-string keys is by encoded length: kind(4) < items(5) <
// metadata(8) < apiVersion(10). items and metadata are always present; kind and apiVersion are
// omitted when empty (json omitempty, honored by modes.Encode).
func streamEncodeList(typeMeta metav1.TypeMeta, listMeta metav1.ListMeta, items []runtime.Object, w io.Writer) error {
	entries := 2
	if typeMeta.Kind != "" {
		entries++
	}
	if typeMeta.APIVersion != "" {
		entries++
	}
	if err := encodeHead(w, cborMajorMap, uint64(entries)); err != nil {
		return err
	}
	if typeMeta.Kind != "" {
		if err := encodeString(w, "kind"); err != nil {
			return err
		}
		if err := encodeString(w, typeMeta.Kind); err != nil {
			return err
		}
	}
	if err := encodeString(w, "items"); err != nil {
		return err
	}
	if err := encodeListItems(w, items); err != nil {
		return err
	}
	if err := encodeString(w, "metadata"); err != nil {
		return err
	}
	if err := modes.Encode.MarshalTo(listMeta, w); err != nil {
		return err
	}
	if typeMeta.APIVersion != "" {
		if err := encodeString(w, "apiVersion"); err != nil {
			return err
		}
		if err := encodeString(w, typeMeta.APIVersion); err != nil {
			return err
		}
	}
	return nil
}

// encodeListItems writes "items": null for a nil slice, else an array (NilContainerAsNull).
func encodeListItems(w io.Writer, items []runtime.Object) error {
	if items == nil {
		_, err := w.Write(cborNull)
		return err
	}
	if err := encodeHead(w, cborMajorArray, uint64(len(items))); err != nil {
		return err
	}
	for _, item := range items {
		if err := modes.Encode.MarshalTo(item, w); err != nil {
			return err
		}
	}
	return nil
}

// streamEncodeUnstructuredList reproduces UnstructuredContent: list.Object with "items" replaced
// by a never-nil array from list.Items, keys in SortBytewiseLexical order.
func streamEncodeUnstructuredList(list *unstructured.UnstructuredList, w io.Writer) error {
	keys := make([]string, 0, len(list.Object)+1)
	for k := range list.Object {
		if k == "items" {
			continue
		}
		keys = append(keys, k)
	}
	keys = append(keys, "items")
	sortStringsByEncoding(keys)

	if err := encodeHead(w, cborMajorMap, uint64(len(keys))); err != nil {
		return err
	}
	for _, key := range keys {
		if err := encodeString(w, key); err != nil {
			return err
		}
		if key == "items" {
			if err := encodeUnstructuredItems(w, list.Items); err != nil {
				return err
			}
			continue
		}
		if err := modes.Encode.MarshalTo(list.Object[key], w); err != nil {
			return err
		}
	}
	return nil
}

// encodeUnstructuredItems writes "items" for an unstructured list: always an array, never null.
func encodeUnstructuredItems(w io.Writer, items []unstructured.Unstructured) error {
	if err := encodeHead(w, cborMajorArray, uint64(len(items))); err != nil {
		return err
	}
	for i := range items {
		if err := modes.Encode.MarshalTo(items[i].UnstructuredContent(), w); err != nil {
			return err
		}
	}
	return nil
}

// encodeString writes s via modes.Encode; keys and string values share one byte-string encoding.
func encodeString(w io.Writer, s string) error {
	return modes.Encode.MarshalTo(s, w)
}

// sortStringsByEncoding sorts keys by their encoded bytes, as modes.Encode does (SortBytewiseLexical).
func sortStringsByEncoding(keys []string) {
	if len(keys) < 2 {
		return
	}
	encoded := make(map[string][]byte, len(keys))
	for _, k := range keys {
		b, _ := modes.Encode.Marshal(k)
		encoded[k] = b
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(encoded[keys[i]], encoded[keys[j]]) < 0
	})
}

// encodeHead writes the shortest-form CBOR head for majorType, matching modes.Encode (definite length).
func encodeHead(w io.Writer, majorType byte, n uint64) error {
	var buf [9]byte
	switch {
	case n < 24:
		buf[0] = majorType | byte(n)
		_, err := w.Write(buf[:1])
		return err
	case n < 1<<8:
		buf[0] = majorType | 24
		buf[1] = byte(n)
		_, err := w.Write(buf[:2])
		return err
	case n < 1<<16:
		buf[0] = majorType | 25
		binary.BigEndian.PutUint16(buf[1:3], uint16(n))
		_, err := w.Write(buf[:3])
		return err
	case n < 1<<32:
		buf[0] = majorType | 26
		binary.BigEndian.PutUint32(buf[1:5], uint32(n))
		_, err := w.Write(buf[:5])
		return err
	default:
		buf[0] = majorType | 27
		binary.BigEndian.PutUint64(buf[1:9], n)
		_, err := w.Write(buf[:9])
		return err
	}
}

// getListMeta returns a typed list's TypeMeta, ListMeta, and items, erroring unless its layout and
// json tags match what streaming requires (modes.Encode derives CBOR field names from json tags).
func getListMeta(list runtime.Object) (metav1.TypeMeta, metav1.ListMeta, []runtime.Object, error) {
	listValue, err := conversion.EnforcePtr(list)
	if err != nil {
		return metav1.TypeMeta{}, metav1.ListMeta{}, nil, err
	}
	listType := listValue.Type()
	if listType.NumField() != 3 {
		return metav1.TypeMeta{}, metav1.ListMeta{}, nil, errListFieldCount
	}
	// TypeMeta
	typeMeta, ok := listValue.Field(0).Interface().(metav1.TypeMeta)
	if !ok {
		return metav1.TypeMeta{}, metav1.ListMeta{}, nil, errListTypeMeta
	}
	if !listType.Field(0).Anonymous {
		return metav1.TypeMeta{}, metav1.ListMeta{}, nil, errListTypeMetaTag
	}
	if jsonTag, ok := listType.Field(0).Tag.Lookup("json"); !ok {
		return metav1.TypeMeta{}, metav1.ListMeta{}, nil, errListTypeMetaTag
	} else if jsonTag != "" && jsonTag != ",inline" {
		return metav1.TypeMeta{}, metav1.ListMeta{}, nil, errListTypeMetaTag
	}
	// ListMeta
	listMeta, ok := listValue.Field(1).Interface().(metav1.ListMeta)
	if !ok {
		return metav1.TypeMeta{}, metav1.ListMeta{}, nil, errListMeta
	}
	if listType.Field(1).Tag.Get("json") != "metadata,omitempty" {
		return metav1.TypeMeta{}, metav1.ListMeta{}, nil, errListMetaTag
	}
	// RawExtension items carry a content type ExtractList drops, so fall back to keep the raw bytes.
	if f := listType.Field(2).Type; f.Kind() == reflect.Slice && f.Elem() == rawExtensionType {
		return metav1.TypeMeta{}, metav1.ListMeta{}, nil, errRawExtensionItems
	}
	// Items
	items, err := meta.ExtractList(list)
	if err != nil {
		return metav1.TypeMeta{}, metav1.ListMeta{}, nil, err
	}
	if listType.Field(2).Tag.Get("json") != "items" {
		return metav1.TypeMeta{}, metav1.ListMeta{}, nil, errListItemsTag
	}
	return typeMeta, listMeta, items, nil
}
