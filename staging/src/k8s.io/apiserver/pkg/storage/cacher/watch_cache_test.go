/*
Copyright 2014 The Kubernetes Authors.

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

package cacher

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/cacher/metrics"
	"k8s.io/apiserver/pkg/storage/cacher/progress"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/tools/cache"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	k8smetrics "k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/testutil"
	"k8s.io/utils/clock"
	testingclock "k8s.io/utils/clock/testing"
)

func makeTestPod(name string, resourceVersion uint64) *v1.Pod {
	return makeTestPodDetails(name, resourceVersion, "some-node", map[string]string{"k8s-app": "my-app"})
}

func makeTestPodDetails(name string, resourceVersion uint64, nodeName string, labels map[string]string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "ns",
			Name:            name,
			ResourceVersion: strconv.FormatUint(resourceVersion, 10),
			Labels:          labels,
		},
		Spec: v1.PodSpec{
			NodeName: nodeName,
		},
	}
}

func makeTestStoreElement(pod *v1.Pod) *storeElement {
	return &storeElement{
		Key:    "prefix/ns/" + pod.Name,
		Object: pod,
		Labels: labels.Set(pod.Labels),
		Fields: fields.Set{"spec.nodeName": pod.Spec.NodeName},
	}
}

type testWatchCache struct {
	*watchCache

	bookmarkRevision chan int64
	stopCh           chan struct{}
}

func (w *testWatchCache) getAllEventsSince(resourceVersion uint64, opts storage.ListOptions) ([]*watchCacheEvent, error) {
	cacheInterval, err := w.getCacheIntervalForEvents(resourceVersion, opts)
	if err != nil {
		return nil, err
	}

	result := []*watchCacheEvent{}
	for {
		event, err := cacheInterval.Next()
		if err != nil {
			return nil, err
		}
		if event == nil {
			break
		}
		result = append(result, event)
	}

	return result, nil
}

func (w *testWatchCache) getCacheIntervalForEvents(resourceVersion uint64, opts storage.ListOptions) (*watchCacheInterval, error) {
	w.RLock()
	defer w.RUnlock()

	return w.getAllEventsSinceLocked(resourceVersion, "", opts)
}

// newTestWatchCache just adds a fake clock.
func newTestWatchCache(capacity int, eventFreshDuration time.Duration, indexers *cache.Indexers) *testWatchCache {
	keyFunc := func(obj runtime.Object) (string, error) {
		return storage.NamespaceKeyFunc("prefix", obj)
	}
	getAttrsFunc := func(obj runtime.Object) (labels.Set, fields.Set, error) {
		pod, ok := obj.(*v1.Pod)
		if !ok {
			return nil, nil, fmt.Errorf("not a pod")
		}
		return labels.Set(pod.Labels), fields.Set{"spec.nodeName": pod.Spec.NodeName}, nil
	}
	versioner := storage.APIObjectVersioner{}
	mockHandler := func(*watchCacheEvent) {}
	wc := &testWatchCache{}
	wc.bookmarkRevision = make(chan int64, 1)
	wc.stopCh = make(chan struct{})
	pr := progress.NewConditionalProgressRequester(wc.RequestWatchProgress, &immediateTickerFactory{}, nil)
	go pr.Run(wc.stopCh)
	wc.watchCache = newWatchCache(keyFunc, mockHandler, getAttrsFunc, versioner, indexers, testingclock.NewFakeClock(time.Now()), eventFreshDuration, schema.GroupResource{Resource: "pods"}, pr)
	// To preserve behavior of tests that assume a given capacity,
	// resize it to th expected size.
	wc.capacity = capacity
	wc.cache = make([]*watchCacheEvent, capacity)
	wc.lowerBoundCapacity = min(capacity, defaultLowerBoundCapacity)
	wc.upperBoundCapacity = max(capacity, defaultUpperBoundCapacity)

	return wc
}

type immediateTickerFactory struct{}

func (t *immediateTickerFactory) NewTimer(d time.Duration) clock.Timer {
	timer := immediateTicker{
		c: make(chan time.Time),
	}
	timer.Reset(d)
	return &timer
}

type immediateTicker struct {
	c chan time.Time
}

func (t *immediateTicker) Reset(d time.Duration) (active bool) {
	select {
	case <-t.c:
		active = true
	default:
	}
	go func() {
		t.c <- time.Now()
	}()
	return active
}

func (t *immediateTicker) C() <-chan time.Time {
	return t.c
}

func (t *immediateTicker) Stop() bool {
	select {
	case <-t.c:
		return true
	default:
		return false
	}
}

func (w *testWatchCache) RequestWatchProgress(ctx context.Context) error {
	go func() {
		select {
		case rev := <-w.bookmarkRevision:
			w.UpdateResourceVersion(fmt.Sprintf("%d", rev))
		case <-ctx.Done():
			return
		}
	}()
	return nil
}

func (w *testWatchCache) Stop() {
	close(w.stopCh)
}

func TestWatchCacheBasic(t *testing.T) {
	store := newTestWatchCache(2, DefaultEventFreshDuration, &cache.Indexers{})
	defer store.Stop()

	// Test Add/Update/Delete.
	pod1 := makeTestPod("pod", 1)
	if err := store.Add(pod1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if item, ok, _ := store.Get(pod1); !ok {
		t.Errorf("didn't find pod")
	} else {
		expected := makeTestStoreElement(makeTestPod("pod", 1))
		if !apiequality.Semantic.DeepEqual(expected, item) {
			t.Errorf("expected %v, got %v", expected, item)
		}
	}
	pod2 := makeTestPod("pod", 2)
	if err := store.Update(pod2); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if item, ok, _ := store.Get(pod2); !ok {
		t.Errorf("didn't find pod")
	} else {
		expected := makeTestStoreElement(makeTestPod("pod", 2))
		if !apiequality.Semantic.DeepEqual(expected, item) {
			t.Errorf("expected %v, got %v", expected, item)
		}
	}
	pod3 := makeTestPod("pod", 3)
	if err := store.Delete(pod3); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if _, ok, _ := store.Get(pod3); ok {
		t.Errorf("found pod")
	}

	// Test List.
	store.Add(makeTestPod("pod1", 4))
	store.Add(makeTestPod("pod2", 5))
	store.Add(makeTestPod("pod3", 6))
	{
		expected := map[string]storeElement{
			"prefix/ns/pod1": *makeTestStoreElement(makeTestPod("pod1", 4)),
			"prefix/ns/pod2": *makeTestStoreElement(makeTestPod("pod2", 5)),
			"prefix/ns/pod3": *makeTestStoreElement(makeTestPod("pod3", 6)),
		}
		items := make(map[string]storeElement)
		for _, item := range store.List() {
			elem := item.(*storeElement)
			items[elem.Key] = *elem
		}
		if !apiequality.Semantic.DeepEqual(expected, items) {
			t.Errorf("expected %v, got %v", expected, items)
		}
	}

	// Test Replace.
	store.Replace([]interface{}{
		makeTestPod("pod4", 7),
		makeTestPod("pod5", 8),
	}, "8")
	{
		expected := map[string]storeElement{
			"prefix/ns/pod4": *makeTestStoreElement(makeTestPod("pod4", 7)),
			"prefix/ns/pod5": *makeTestStoreElement(makeTestPod("pod5", 8)),
		}
		items := make(map[string]storeElement)
		for _, item := range store.List() {
			elem := item.(*storeElement)
			items[elem.Key] = *elem
		}
		if !apiequality.Semantic.DeepEqual(expected, items) {
			t.Errorf("expected %v, got %v", expected, items)
		}
	}
}

func TestEvents(t *testing.T) {
	store := newTestWatchCache(5, DefaultEventFreshDuration, &cache.Indexers{})
	defer store.Stop()

	// no dynamic-size cache to fit old tests.
	store.lowerBoundCapacity = 5
	store.upperBoundCapacity = 5

	store.Add(makeTestPod("pod", 3))

	// Test for Added event.
	{
		_, err := store.getAllEventsSince(1, storage.ListOptions{Predicate: storage.Everything})
		if err == nil {
			t.Errorf("expected error too old")
		}
		if _, ok := err.(*errors.StatusError); !ok {
			t.Errorf("expected error to be of type StatusError")
		}
	}
	{
		result, err := store.getAllEventsSince(2, storage.ListOptions{Predicate: storage.Everything})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("unexpected events: %v", result)
		}
		if result[0].Type != watch.Added {
			t.Errorf("unexpected event type: %v", result[0].Type)
		}
		pod := makeTestPod("pod", uint64(3))
		if !apiequality.Semantic.DeepEqual(pod, result[0].Object) {
			t.Errorf("unexpected item: %v, expected: %v", result[0].Object, pod)
		}
		if result[0].PrevObject != nil {
			t.Errorf("unexpected item: %v", result[0].PrevObject)
		}
	}

	store.Update(makeTestPod("pod", 4))
	store.Update(makeTestPod("pod", 5))

	// Test with not full cache.
	{
		_, err := store.getAllEventsSince(1, storage.ListOptions{Predicate: storage.Everything})
		if err == nil {
			t.Errorf("expected error too old")
		}
	}
	{
		result, err := store.getAllEventsSince(3, storage.ListOptions{Predicate: storage.Everything})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("unexpected events: %v", result)
		}
		for i := 0; i < 2; i++ {
			if result[i].Type != watch.Modified {
				t.Errorf("unexpected event type: %v", result[i].Type)
			}
			pod := makeTestPod("pod", uint64(i+4))
			if !apiequality.Semantic.DeepEqual(pod, result[i].Object) {
				t.Errorf("unexpected item: %v, expected: %v", result[i].Object, pod)
			}
			prevPod := makeTestPod("pod", uint64(i+3))
			if !apiequality.Semantic.DeepEqual(prevPod, result[i].PrevObject) {
				t.Errorf("unexpected item: %v, expected: %v", result[i].PrevObject, prevPod)
			}
		}
	}

	for i := 6; i < 10; i++ {
		store.Update(makeTestPod("pod", uint64(i)))
	}

	// Test with full cache - there should be elements from 5 to 9.
	{
		_, err := store.getAllEventsSince(3, storage.ListOptions{Predicate: storage.Everything})
		if err == nil {
			t.Errorf("expected error too old")
		}
	}
	{
		result, err := store.getAllEventsSince(4, storage.ListOptions{Predicate: storage.Everything})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 5 {
			t.Fatalf("unexpected events: %v", result)
		}
		for i := 0; i < 5; i++ {
			pod := makeTestPod("pod", uint64(i+5))
			if !apiequality.Semantic.DeepEqual(pod, result[i].Object) {
				t.Errorf("unexpected item: %v, expected: %v", result[i].Object, pod)
			}
		}
	}

	// Test for delete event.
	store.Delete(makeTestPod("pod", uint64(10)))

	{
		result, err := store.getAllEventsSince(9, storage.ListOptions{Predicate: storage.Everything})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("unexpected events: %v", result)
		}
		if result[0].Type != watch.Deleted {
			t.Errorf("unexpected event type: %v", result[0].Type)
		}
		pod := makeTestPod("pod", uint64(10))
		if !apiequality.Semantic.DeepEqual(pod, result[0].Object) {
			t.Errorf("unexpected item: %v, expected: %v", result[0].Object, pod)
		}
		prevPod := makeTestPod("pod", uint64(9))
		if !apiequality.Semantic.DeepEqual(prevPod, result[0].PrevObject) {
			t.Errorf("unexpected item: %v, expected: %v", result[0].PrevObject, prevPod)
		}
	}
}

func TestMarker(t *testing.T) {
	store := newTestWatchCache(3, DefaultEventFreshDuration, &cache.Indexers{})
	defer store.Stop()

	// First thing that is called when propagated from storage is Replace.
	store.Replace([]interface{}{
		makeTestPod("pod1", 5),
		makeTestPod("pod2", 9),
	}, "9")

	_, err := store.getAllEventsSince(8, storage.ListOptions{Predicate: storage.Everything})
	if err == nil || !strings.Contains(err.Error(), "too old resource version") {
		t.Errorf("unexpected error: %v", err)
	}
	// Getting events from 8 should return no events,
	// even though there is a marker there.
	result, err := store.getAllEventsSince(9, storage.ListOptions{Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("unexpected result: %#v, expected no events", result)
	}

	pod := makeTestPod("pods", 12)
	store.Add(pod)
	// Getting events from 8 should still work and return one event.
	result, err = store.getAllEventsSince(9, storage.ListOptions{Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || !apiequality.Semantic.DeepEqual(result[0].Object, pod) {
		t.Errorf("unexpected result: %#v, expected %v", result, pod)
	}
}

func TestWaitUntilFreshAndList(t *testing.T) {
	ctx := context.Background()
	store := newTestWatchCache(3, DefaultEventFreshDuration, &cache.Indexers{
		"l:label": func(obj interface{}) ([]string, error) {
			pod, ok := obj.(*v1.Pod)
			if !ok {
				return nil, fmt.Errorf("not a pod %#v", obj)
			}
			if value, ok := pod.Labels["label"]; ok {
				return []string{value}, nil
			}
			return nil, nil
		},
		"f:spec.nodeName": func(obj interface{}) ([]string, error) {
			pod, ok := obj.(*v1.Pod)
			if !ok {
				return nil, fmt.Errorf("not a pod %#v", obj)
			}
			return []string{pod.Spec.NodeName}, nil
		},
	})
	defer store.Stop()
	// In background, update the store.
	go func() {
		store.Add(makeTestPodDetails("pod1", 2, "node1", map[string]string{"label": "value1"}))
		store.Add(makeTestPodDetails("pod2", 3, "node1", map[string]string{"label": "value1"}))
		store.Add(makeTestPodDetails("pod3", 5, "node2", map[string]string{"label": "value2"}))
	}()

	// list by empty MatchValues.
	resp, indexUsed, err := store.WaitUntilFreshAndList(ctx, 5, "prefix/", storage.ListOptions{Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ResourceVersion != 5 {
		t.Errorf("unexpected resourceVersion: %v, expected: 5", resp.ResourceVersion)
	}
	if len(resp.Items) != 3 {
		t.Errorf("unexpected list returned: %#v", resp)
	}
	if indexUsed != "" {
		t.Errorf("Used index %q but expected none to be used", indexUsed)
	}

	// list by label index.
	resp, indexUsed, err = store.WaitUntilFreshAndList(ctx, 5, "prefix/", storage.ListOptions{Predicate: storage.SelectionPredicate{
		Label: labels.SelectorFromSet(map[string]string{
			"label": "value1",
		}),
		Field: fields.SelectorFromSet(map[string]string{
			"spec.nodeName": "node2",
		}),
		IndexLabels: []string{"label"},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ResourceVersion != 5 {
		t.Errorf("unexpected resourceVersion: %v, expected: 5", resp.ResourceVersion)
	}
	if len(resp.Items) != 2 {
		t.Errorf("unexpected list returned: %#v", resp)
	}
	if indexUsed != "l:label" {
		t.Errorf("Used index %q but expected %q", indexUsed, "l:label")
	}

	// list with spec.nodeName index.
	resp, indexUsed, err = store.WaitUntilFreshAndList(ctx, 5, "prefix/", storage.ListOptions{Predicate: storage.SelectionPredicate{
		Label: labels.SelectorFromSet(map[string]string{
			"not-exist-label": "whatever",
		}),
		Field: fields.SelectorFromSet(map[string]string{
			"spec.nodeName": "node2",
		}),
		IndexFields: []string{"spec.nodeName"},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ResourceVersion != 5 {
		t.Errorf("unexpected resourceVersion: %v, expected: 5", resp.ResourceVersion)
	}
	if len(resp.Items) != 1 {
		t.Errorf("unexpected list returned: %#v", resp)
	}
	if indexUsed != "f:spec.nodeName" {
		t.Errorf("Used index %q but expected %q", indexUsed, "f:spec.nodeName")
	}

	// list with index not exists.
	resp, indexUsed, err = store.WaitUntilFreshAndList(ctx, 5, "prefix/", storage.ListOptions{Predicate: storage.SelectionPredicate{
		Label: labels.SelectorFromSet(map[string]string{
			"not-exist-label": "whatever",
		}),
		Field:       fields.Everything(),
		IndexLabels: []string{"label"},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ResourceVersion != 5 {
		t.Errorf("unexpected resourceVersion: %v, expected: 5", resp.ResourceVersion)
	}
	if len(resp.Items) != 3 {
		t.Errorf("unexpected list returned: %#v", resp)
	}
	if indexUsed != "" {
		t.Errorf("Used index %q but expected none to be used", indexUsed)
	}
}

func TestWaitUntilFreshAndListFromCache(t *testing.T) {
	featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.ConsistentListFromCache, true)
	forceRequestWatchProgressSupport(t)
	ctx := context.Background()
	store := newTestWatchCache(3, DefaultEventFreshDuration, &cache.Indexers{})
	defer store.Stop()
	// In background, update the store.
	go func() {
		store.Add(makeTestPod("pod1", 2))
		store.bookmarkRevision <- 3
	}()

	// list from future revision. Requires watch cache to request bookmark to get it.
	resp, indexUsed, err := store.WaitUntilFreshAndList(ctx, 3, "prefix/", storage.ListOptions{Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ResourceVersion != 3 {
		t.Errorf("unexpected resourceVersion: %v, expected: 6", resp.ResourceVersion)
	}
	if len(resp.Items) != 1 {
		t.Errorf("unexpected list returned: %#v", resp)
	}
	if indexUsed != "" {
		t.Errorf("Used index %q but expected none to be used", indexUsed)
	}
}

func TestWaitUntilFreshAndGet(t *testing.T) {
	ctx := context.Background()
	store := newTestWatchCache(3, DefaultEventFreshDuration, &cache.Indexers{})
	defer store.Stop()

	// In background, update the store.
	go func() {
		store.Add(makeTestPod("foo", 2))
		store.Add(makeTestPod("bar", 5))
	}()

	obj, exists, resourceVersion, err := store.WaitUntilFreshAndGet(ctx, 5, "prefix/ns/bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resourceVersion != 5 {
		t.Errorf("unexpected resourceVersion: %v, expected: 5", resourceVersion)
	}
	if !exists {
		t.Fatalf("no results returned: %#v", obj)
	}
	expected := makeTestStoreElement(makeTestPod("bar", 5))
	if !apiequality.Semantic.DeepEqual(expected, obj) {
		t.Errorf("expected %v, got %v", expected, obj)
	}
}

func TestWaitUntilFreshAndListTimeout(t *testing.T) {
	tcs := []struct {
		name                    string
		ConsistentListFromCache bool
	}{
		{
			name:                    "FromStorage",
			ConsistentListFromCache: false,
		},
		{
			name:                    "FromCache",
			ConsistentListFromCache: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.ConsistentListFromCache, tc.ConsistentListFromCache)
			ctx := context.Background()
			store := newTestWatchCache(3, DefaultEventFreshDuration, &cache.Indexers{})
			defer store.Stop()
			fc := store.clock.(*testingclock.FakeClock)

			// In background, step clock after the below call starts the timer.
			go func() {
				for !fc.HasWaiters() {
					time.Sleep(time.Millisecond)
				}
				store.Add(makeTestPod("foo", 2))
				store.bookmarkRevision <- 3
				fc.Step(blockTimeout)

				// Add an object to make sure the test would
				// eventually fail instead of just waiting
				// forever.
				time.Sleep(30 * time.Second)
				store.Add(makeTestPod("bar", 4))
			}()

			_, _, err := store.WaitUntilFreshAndList(ctx, 4, "", storage.ListOptions{Predicate: storage.Everything})
			if !errors.IsTimeout(err) {
				t.Errorf("expected timeout error but got: %v", err)
			}
			if !storage.IsTooLargeResourceVersion(err) {
				t.Errorf("expected 'Too large resource version' cause in error but got: %v", err)
			}
		})
	}
}
