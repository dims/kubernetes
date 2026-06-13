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
	"bytes"
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

func TestStreamingYAMLCollectionsEncoding(t *testing.T) {
	nonStreaming := NewSerializerWithOptions(DefaultMetaFactory, nil, nil, SerializerOptions{Yaml: true})
	streaming := NewSerializerWithOptions(DefaultMetaFactory, nil, nil, SerializerOptions{Yaml: true, StreamingCollectionsEncoding: true})

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
					{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "default2"}},
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
		{
			name: "carp list item with long wrapping value",
			in: &testapigroupv1.CarpList{
				Items: []testapigroupv1.Carp{{ObjectMeta: metav1.ObjectMeta{
					Name:        "pod",
					Annotations: map[string]string{"data": strings.Repeat("abcdefghij ", 16)},
				}}},
			},
		},
		{name: "carp list nil items", in: &testapigroupv1.CarpList{Items: nil}},
		{name: "carp list empty items", in: &testapigroupv1.CarpList{Items: []testapigroupv1.Carp{}}},
		{name: "carp list just kind", in: &testapigroupv1.CarpList{TypeMeta: metav1.TypeMeta{Kind: "List"}}},
		{name: "carp list just apiVersion", in: &testapigroupv1.CarpList{TypeMeta: metav1.TypeMeta{APIVersion: "v1"}}},
		// Unstructured lists fall back to the non-streaming path for now.
		{name: "unstructured list cannot stream", in: &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]interface{}{"name": "pod"}}}}, cannotStream: true},
		{name: "list with extra field cannot stream", in: &ListWithAdditionalFields{Items: []testapigroupv1.Carp{}}, cannotStream: true},
		{name: "list with MarshalJSON cannot stream", in: &ListWithMarshalJSONList{}, cannotStream: true},
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
			if diff := cmp.Diff(want.String(), got.String()); diff != "" {
				t.Errorf("streaming output differs from non-streaming (-want +got):\n%s", diff)
			}

			var body bytes.Buffer
			ok, err := streamYAMLEncodeCollections(tc.in, &body)
			if err != nil {
				t.Fatalf("streamYAMLEncodeCollections: %v", err)
			}
			if ok == tc.cannotStream {
				t.Errorf("streamYAMLEncodeCollections returned ok=%v, want %v", ok, !tc.cannotStream)
			}
			if ok {
				if diff := cmp.Diff(want.String(), body.String()); diff != "" {
					t.Errorf("stream body differs from full encoding (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestStreamingYAMLCollectionsEncodingGroundTruth(t *testing.T) {
	streaming := NewSerializerWithOptions(DefaultMetaFactory, nil, nil, SerializerOptions{Yaml: true, StreamingCollectionsEncoding: true})
	for _, tc := range []struct {
		name string
		in   runtime.Object
		want string
	}{
		{name: "nil items", in: &testapigroupv1.CarpList{Items: nil}, want: "items: null\nmetadata: {}\n"},
		{name: "empty items", in: &testapigroupv1.CarpList{Items: []testapigroupv1.Carp{}}, want: "items: []\nmetadata: {}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := streaming.Encode(tc.in, &buf); err != nil {
				t.Fatalf("encode: %v", err)
			}
			if diff := cmp.Diff(tc.want, buf.String()); diff != "" {
				t.Errorf("unexpected encoding (-want +got):\n%s", diff)
			}
		})
	}
}

// TestStreamingYAMLRawExtensionFallsBack: metav1.List has []runtime.RawExtension items, whose
// content type ExtractList drops, so these lists must fall back to non-streaming.
func TestStreamingYAMLRawExtensionFallsBack(t *testing.T) {
	nonStreaming := NewSerializerWithOptions(DefaultMetaFactory, nil, nil, SerializerOptions{Yaml: true})
	streaming := NewSerializerWithOptions(DefaultMetaFactory, nil, nil, SerializerOptions{Yaml: true, StreamingCollectionsEncoding: true})
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
			if diff := cmp.Diff(want.String(), got.String()); diff != "" {
				t.Errorf("streaming differs from non-streaming (-want +got):\n%s", diff)
			}
			if ok, _ := streamYAMLEncodeCollections(tc.list, io.Discard); ok {
				t.Errorf("expected RawExtension list to fall back to non-streaming (ok=false)")
			}
		})
	}
}

func TestFuzzStreamingYAMLCollectionsEncoding(t *testing.T) {
	nonStreaming := NewSerializerWithOptions(DefaultMetaFactory, nil, nil, SerializerOptions{Yaml: true})
	disableFuzzFieldsV1 := func(field *metav1.FieldsV1, c randfill.Continue) {}
	f := randfill.New().Funcs(disableFuzzFieldsV1)
	for range 1000 {
		list := &testapigroupv1.CarpList{}
		f.Fill(list)
		var body, normal bytes.Buffer
		ok, err := streamYAMLEncodeCollections(list, &body)
		if err != nil {
			t.Fatalf("streamYAMLEncodeCollections: %v", err)
		}
		if !ok {
			t.Fatalf("expected streaming encoder to handle %T", list)
		}
		if err := nonStreaming.Encode(list, &normal); err != nil {
			t.Fatalf("non-streaming encode: %v", err)
		}
		if diff := cmp.Diff(normal.String(), body.String()); diff != "" {
			t.Errorf("streaming differs from non-streaming (-want +got):\n%s", diff)
		}
	}
}

// maxWriteRecorder records the largest single Write and the number of writes, discarding data.
type maxWriteRecorder struct {
	max   int
	count int
}

func (r *maxWriteRecorder) Write(p []byte) (int, error) {
	if len(p) > r.max {
		r.max = len(p)
	}
	r.count++
	return len(p), nil
}

// TestStreamingYAMLBoundsBuffer shows the win: streaming buffers at most one item before writing,
// while non-streaming materializes the whole document. Bounds peak (live) memory, not total churn.
func TestStreamingYAMLBoundsBuffer(t *testing.T) {
	const items = 500
	list := &testapigroupv1.CarpList{Items: make([]testapigroupv1.Carp, items)}
	for i := range list.Items {
		list.Items[i] = testapigroupv1.Carp{ObjectMeta: metav1.ObjectMeta{
			Name:        "item",
			Annotations: map[string]string{"data": strings.Repeat("x", 4096)},
		}}
	}

	stream := &maxWriteRecorder{}
	if err := NewSerializerWithOptions(DefaultMetaFactory, nil, nil, SerializerOptions{Yaml: true, StreamingCollectionsEncoding: true}).Encode(list, stream); err != nil {
		t.Fatal(err)
	}
	whole := &maxWriteRecorder{}
	if err := NewSerializerWithOptions(DefaultMetaFactory, nil, nil, SerializerOptions{Yaml: true}).Encode(list, whole); err != nil {
		t.Fatal(err)
	}
	t.Logf("streaming: maxWrite=%d writes=%d; non-streaming: maxWrite=%d writes=%d", stream.max, stream.count, whole.max, whole.count)
	if stream.count <= whole.count {
		t.Errorf("expected streaming to use more, smaller writes (streaming=%d non-streaming=%d)", stream.count, whole.count)
	}
	if stream.max*int(items/2) > whole.max {
		t.Errorf("streaming max single write %d is not bounded well below the whole document %d", stream.max, whole.max)
	}
}
