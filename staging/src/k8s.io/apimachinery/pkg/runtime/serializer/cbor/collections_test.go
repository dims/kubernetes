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
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"sigs.k8s.io/randfill"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	testapigroupv1 "k8s.io/apimachinery/pkg/apis/testapigroup/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// listWithExtraField has more than the three fields a streamable list may have.
type listWithExtraField struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []testapigroupv1.Carp `json:"items"`
	Extra           int                   `json:"extra"`
}

func (l *listWithExtraField) DeepCopyObject() runtime.Object { return nil }

// listWithMarshalJSON implements json.Marshaler (transcoded by modes.Encode), so it must not stream.
type listWithMarshalJSON struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []testapigroupv1.Carp `json:"items"`
}

func (l *listWithMarshalJSON) DeepCopyObject() runtime.Object { return nil }
func (l *listWithMarshalJSON) MarshalJSON() ([]byte, error)   { return []byte(`"marshalJSON"`), nil }

// listWithMarshalCBOR implements cbor.Marshaler, so it must not be streamed field-by-field.
type listWithMarshalCBOR struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []testapigroupv1.Carp `json:"items"`
}

func (l *listWithMarshalCBOR) DeepCopyObject() runtime.Object { return nil }
func (l *listWithMarshalCBOR) MarshalCBOR() ([]byte, error)   { return []byte{0xa0}, nil } // empty map

func TestStreamingCollectionsEncoding(t *testing.T) {
	nonStreaming := NewSerializer(nil, nil)
	streaming := NewSerializer(nil, nil, StreamingCollectionsEncoding(true))

	var remaining int64 = 1
	for _, tc := range []struct {
		name         string
		in           runtime.Object
		cannotStream bool
	}{
		{
			name: "carp list two elements",
			in: &testapigroupv1.CarpList{
				TypeMeta: metav1.TypeMeta{Kind: "CarpList", APIVersion: "testapigroup.k8s.io/v1"},
				ListMeta: metav1.ListMeta{ResourceVersion: "2345"},
				Items: []testapigroupv1.Carp{
					{TypeMeta: metav1.TypeMeta{Kind: "Carp", APIVersion: "v1"}, ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default"}},
					{TypeMeta: metav1.TypeMeta{Kind: "Carp", APIVersion: "v1"}, ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "default2"}},
				},
			},
		},
		{
			name: "carp list with continue and remaining",
			in: &testapigroupv1.CarpList{
				ListMeta: metav1.ListMeta{ResourceVersion: "2345", Continue: "abc", RemainingItemCount: &remaining},
				Items:    []testapigroupv1.Carp{{ObjectMeta: metav1.ObjectMeta{Name: "pod"}}},
			},
		},
		{name: "carp list nil items", in: &testapigroupv1.CarpList{Items: nil}},
		{name: "carp list empty items", in: &testapigroupv1.CarpList{Items: []testapigroupv1.Carp{}}},
		{name: "carp list just kind", in: &testapigroupv1.CarpList{TypeMeta: metav1.TypeMeta{Kind: "List"}}},
		{name: "carp list just apiVersion", in: &testapigroupv1.CarpList{TypeMeta: metav1.TypeMeta{APIVersion: "v1"}}},
		{
			name: "carp list item with nil and empty containers",
			in: &testapigroupv1.CarpList{Items: []testapigroupv1.Carp{
				{Spec: testapigroupv1.CarpSpec{NodeSelector: nil}},
				{Spec: testapigroupv1.CarpSpec{NodeSelector: map[string]string{}}},
			}},
		},
		{
			name: "unstructured list full",
			in: &unstructured.UnstructuredList{
				Object: map[string]interface{}{"kind": "List", "apiVersion": "v1", "metadata": map[string]interface{}{"resourceVersion": "2345"}},
				Items: []unstructured.Unstructured{
					{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Carp", "metadata": map[string]interface{}{"name": "pod"}}},
				},
			},
		},
		{name: "unstructured list nil items", in: &unstructured.UnstructuredList{Items: nil}},
		{name: "unstructured list empty items", in: &unstructured.UnstructuredList{Items: []unstructured.Unstructured{}}},
		{name: "unstructured list nil object", in: &unstructured.UnstructuredList{Object: nil}},
		{name: "unstructured list empty", in: &unstructured.UnstructuredList{}},
		{
			// keys of varying length to exercise SortBytewiseLexical ordering.
			name: "unstructured list extra keys",
			in: &unstructured.UnstructuredList{
				Object: map[string]interface{}{"z": int64(1), "aa": "x", "kind": "List", "apiVersion": "v1", "averylongkeyname": true, "": "empty"},
				Items:  []unstructured.Unstructured{{Object: map[string]interface{}{"a": int64(1)}}},
			},
		},
		{
			// "items" present in Object is overridden by the typed Items field.
			name: "unstructured list items override",
			in: &unstructured.UnstructuredList{
				Object: map[string]interface{}{"items": []interface{}{map[string]interface{}{"name": "ignored"}}},
				Items:  []unstructured.Unstructured{{Object: map[string]interface{}{"name": "used"}}},
			},
		},
		{
			name: "unstructured list invalid utf8 value",
			in: &unstructured.UnstructuredList{
				Object: map[string]interface{}{"key": "\x80"},
				Items:  []unstructured.Unstructured{{Object: map[string]interface{}{"key": "\x80"}}},
			},
		},
		{name: "list with extra field cannot stream", in: &listWithExtraField{Items: []testapigroupv1.Carp{}}, cannotStream: true},
		{name: "list with MarshalJSON cannot stream", in: &listWithMarshalJSON{}, cannotStream: true},
		{name: "list with MarshalCBOR cannot stream", in: &listWithMarshalCBOR{}, cannotStream: true},
		{name: "non-list cannot stream", in: &testapigroupv1.Carp{TypeMeta: metav1.TypeMeta{Kind: "Carp"}}, cannotStream: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var want, got bytes.Buffer
			if err := nonStreaming.Encode(tc.in, &want); err != nil {
				t.Fatalf("non-streaming encode: %v", err)
			}
			if err := streaming.Encode(tc.in, &got); err != nil {
				t.Fatalf("streaming encode: %v", err)
			}
			if diff := cmp.Diff(want.Bytes(), got.Bytes()); diff != "" {
				t.Errorf("streaming serializer output differs from non-streaming (-want +got):\n%s\nwant % x\ngot  % x", diff, want.Bytes(), got.Bytes())
			}

			var body bytes.Buffer
			ok, err := streamEncodeCollections(tc.in, &body)
			if err != nil {
				t.Fatalf("streamEncodeCollections: %v", err)
			}
			if ok == tc.cannotStream {
				t.Errorf("streamEncodeCollections returned ok=%v, want %v", ok, !tc.cannotStream)
			}
			if ok {
				full := append(append([]byte{}, selfDescribedCBOR...), body.Bytes()...)
				if diff := cmp.Diff(want.Bytes(), full); diff != "" {
					t.Errorf("stream body (with self-described tag) differs from full encoding (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestStreamingCollectionsEncodingGroundTruth pins exact bytes so a shared regression can't hide.
func TestStreamingCollectionsEncodingGroundTruth(t *testing.T) {
	streaming := NewSerializer(nil, nil, StreamingCollectionsEncoding(true))
	for _, tc := range []struct {
		name string
		in   runtime.Object
		want []byte
	}{
		{
			name: "nil items encodes as null",
			in:   &testapigroupv1.CarpList{Items: nil},
			// d9d9f7 a2 45"items" f6(null) 48"metadata" a0({})
			want: []byte{0xd9, 0xd9, 0xf7, 0xa2, 0x45, 'i', 't', 'e', 'm', 's', 0xf6, 0x48, 'm', 'e', 't', 'a', 'd', 'a', 't', 'a', 0xa0},
		},
		{
			name: "empty items encodes as empty array",
			in:   &testapigroupv1.CarpList{Items: []testapigroupv1.Carp{}},
			// d9d9f7 a2 45"items" 80([]) 48"metadata" a0({})
			want: []byte{0xd9, 0xd9, 0xf7, 0xa2, 0x45, 'i', 't', 'e', 'm', 's', 0x80, 0x48, 'm', 'e', 't', 'a', 'd', 'a', 't', 'a', 0xa0},
		},
		{
			name: "empty unstructured list has only an empty items array",
			in:   &unstructured.UnstructuredList{},
			// d9d9f7 a1 45"items" 80([])
			want: []byte{0xd9, 0xd9, 0xf7, 0xa1, 0x45, 'i', 't', 'e', 'm', 's', 0x80},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := streaming.Encode(tc.in, &buf); err != nil {
				t.Fatalf("encode: %v", err)
			}
			if diff := cmp.Diff(tc.want, buf.Bytes()); diff != "" {
				t.Errorf("unexpected encoding (-want +got):\n%s\nwant % x\ngot  % x", diff, tc.want, buf.Bytes())
			}
		})
	}
}

// TestStreamingCBORRawExtensionFallsBack: metav1.List has []runtime.RawExtension items, whose
// content type ExtractList drops, so these lists must fall back to non-streaming.
func TestStreamingCBORRawExtensionFallsBack(t *testing.T) {
	nonStreaming := NewSerializer(nil, nil)
	streaming := NewSerializer(nil, nil, StreamingCollectionsEncoding(true))
	cborRaw := []byte{0xd9, 0xd9, 0xf7, 0xa1, 0x41, 'a', 0x01} // self-described CBOR {"a":1}
	jsonRaw := []byte(`{"a":1}`)
	for _, tc := range []struct {
		name string
		list runtime.Object
	}{
		{"raw cbor", &metav1.List{Items: []runtime.RawExtension{{Raw: cborRaw}}}},
		{"raw json", &metav1.List{Items: []runtime.RawExtension{{Raw: jsonRaw}}}},
		{"object", &metav1.List{Items: []runtime.RawExtension{{Object: &testapigroupv1.Carp{ObjectMeta: metav1.ObjectMeta{Name: "x"}}}}}},
		{"nil item", &metav1.List{Items: []runtime.RawExtension{{}}}},
		{"empty", &metav1.List{Items: []runtime.RawExtension{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var want, got bytes.Buffer
			if err := nonStreaming.Encode(tc.list, &want); err != nil {
				t.Fatalf("non-streaming encode: %v", err)
			}
			if err := streaming.Encode(tc.list, &got); err != nil {
				t.Fatalf("streaming encode: %v", err)
			}
			if diff := cmp.Diff(want.Bytes(), got.Bytes()); diff != "" {
				t.Errorf("streaming differs from non-streaming (-want +got):\n%s", diff)
			}
			if ok, _ := streamEncodeCollections(tc.list, io.Discard); ok {
				t.Errorf("expected RawExtension list to fall back to non-streaming (ok=false)")
			}
		})
	}
}

func TestFuzzStreamingCollectionsEncoding(t *testing.T) {
	nonStreaming := NewSerializer(nil, nil)
	disableFuzzFieldsV1 := func(field *metav1.FieldsV1, c randfill.Continue) {}
	fuzzUnstructuredList := func(list *unstructured.UnstructuredList, c randfill.Continue) {
		list.Object = map[string]interface{}{
			"kind":       "List",
			"apiVersion": "v1",
			c.String(0):  c.String(0),
			c.String(0):  c.Uint64(),
			c.String(0):  c.Bool(),
			"metadata": map[string]interface{}{
				"resourceVersion": fmt.Sprintf("%d", c.Uint64()),
				c.String(0):       c.String(0),
			},
		}
		c.Fill(&list.Items)
	}
	fuzzMap := func(kvs map[string]interface{}, c randfill.Continue) {
		kvs[c.String(0)] = c.Bool()
		kvs[c.String(0)] = c.Uint64()
		kvs[c.String(0)] = c.String(0)
	}
	f := randfill.New().Funcs(disableFuzzFieldsV1, fuzzUnstructuredList, fuzzMap)

	compare := func(t *testing.T, in runtime.Object) {
		t.Helper()
		var body, normal bytes.Buffer
		ok, err := streamEncodeCollections(in, &body)
		if err != nil {
			t.Fatalf("streamEncodeCollections: %v", err)
		}
		if !ok {
			t.Fatalf("expected streaming encoder to handle %T", in)
		}
		if err := nonStreaming.Encode(in, &normal); err != nil {
			t.Fatalf("non-streaming encode: %v", err)
		}
		full := append(append([]byte{}, selfDescribedCBOR...), body.Bytes()...)
		if diff := cmp.Diff(normal.Bytes(), full); diff != "" {
			t.Errorf("streaming differs from non-streaming (-want +got):\n%s\nwant % x\ngot  % x", diff, normal.Bytes(), full)
		}
	}

	t.Run("CarpList", func(t *testing.T) {
		for range 1000 {
			list := &testapigroupv1.CarpList{}
			f.Fill(list)
			compare(t, list)
		}
	})
	t.Run("UnstructuredList", func(t *testing.T) {
		for range 1000 {
			list := &unstructured.UnstructuredList{}
			f.Fill(list)
			compare(t, list)
		}
	})
}

// BenchmarkStreamingCollectionsEncoding shows streaming allocates per-item, not per-collection.
func BenchmarkStreamingCollectionsEncoding(b *testing.B) {
	const items = 1000
	const valueSize = 10 * 1024
	build := func() *unstructured.UnstructuredList {
		list := &unstructured.UnstructuredList{
			Object: map[string]interface{}{"kind": "List", "apiVersion": "v1", "metadata": map[string]interface{}{"resourceVersion": "1"}},
			Items:  make([]unstructured.Unstructured, items),
		}
		for i := range list.Items {
			list.Items[i] = unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Carp",
				"metadata":   map[string]interface{}{"name": fmt.Sprintf("item-%d", i)},
				"data":       strings.Repeat("x", valueSize),
			}}
		}
		return list
	}
	list := build()

	b.Run("streaming", func(b *testing.B) {
		s := NewSerializer(nil, nil, StreamingCollectionsEncoding(true))
		var buf bytes.Buffer
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			if err := s.Encode(list, &buf); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("non-streaming", func(b *testing.B) {
		s := NewSerializer(nil, nil)
		var buf bytes.Buffer
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			if err := s.Encode(list, &buf); err != nil {
				b.Fatal(err)
			}
		}
	})
}
