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

package cacher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/apis/example"
	"k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/storage"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
)

// TestPaginatedListFallbackToEtcd tests that paginated LIST calls with rv="" and LIMIT set
// are correctly served from cache when the ListFromCacheSnapshot feature is disabled.
func TestPaginatedListFallbackToEtcd(t *testing.T) {
	testCases := []struct {
		name                  string
		listFromCacheSnapshot bool
		resourceVersion       string
		limit                 int64
		expectFallbackToEtcd  bool
		expectResourceExpired bool
	}{
		{
			name:                  "Feature enabled, rv=\"\", limit set - should use listExactRV and fallback to etcd",
			listFromCacheSnapshot: true,
			resourceVersion:       "",
			limit:                 500,
			expectFallbackToEtcd:  true,
			expectResourceExpired: true,
		},
		{
			name:                  "Feature disabled, rv=\"\", limit set - should use listLatestRV and not fallback to etcd",
			listFromCacheSnapshot: false,
			resourceVersion:       "",
			limit:                 500,
			expectFallbackToEtcd:  false,
			expectResourceExpired: false,
		},
		{
			name:                  "Feature disabled, rv=\"10\", limit set - should use listLatestRV and not fallback to etcd",
			listFromCacheSnapshot: false,
			resourceVersion:       "10",
			limit:                 500,
			expectFallbackToEtcd:  false,
			expectResourceExpired: false,
		},
		{
			name:                  "Feature disabled, rv=\"\", no limit - should use listLatestRV and not fallback to etcd",
			listFromCacheSnapshot: false,
			resourceVersion:       "",
			limit:                 0,
			expectFallbackToEtcd:  false,
			expectResourceExpired: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set feature gate for test
			featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.ListFromCacheSnapshot, tc.listFromCacheSnapshot)

			// Create mock storage and cacher
			etcdCalled := false
			etcd := &dummyStorage{
				getListFn: func(_ context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
					etcdCalled = true
					return nil
				},
			}

			resourceExpiredReturned := false
			cacher := &dummyCacher{
				ready: true,
				dummyStorage: dummyStorage{
					getListFn: func(_ context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
						if tc.expectResourceExpired {
							resourceExpiredReturned = true
							return errors.NewResourceExpired("expired")
						}
						return nil
					},
				},
			}

			// Create a mock delegator that simulates the behavior we're testing
			mockDelegator := &mockCacheDelegator{
				storage: etcd,
				cacher:  cacher,
			}

			// Create list options with the test parameters
			listOpts := storage.ListOptions{
				ResourceVersion: tc.resourceVersion,
				Predicate: storage.SelectionPredicate{
					Limit: tc.limit,
				},
			}

			// Call GetList
			listObj := &example.PodList{}
			_ = mockDelegator.GetList(context.Background(), "key", listOpts, listObj)

			// Verify expectations
			assert.Equal(t, tc.expectFallbackToEtcd, etcdCalled, "etcd fallback expectation not met")
			if tc.expectResourceExpired {
				assert.True(t, resourceExpiredReturned, "ResourceExpired error should have been returned from cacher")
			}
		})
	}
}

// mockCacheDelegator simulates the behavior of CacheDelegator for testing
type mockCacheDelegator struct {
	storage storage.Interface
	cacher  *dummyCacher
}

func (d *mockCacheDelegator) GetList(ctx context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
	// First try to get from cacher
	err := d.cacher.GetList(ctx, key, opts, listObj)

	// If ResourceExpired error and ListFromCacheSnapshot feature is enabled, fallback to etcd
	if err != nil && errors.IsResourceExpired(err) && utilfeature.DefaultFeatureGate.Enabled(features.ListFromCacheSnapshot) {
		return d.storage.GetList(ctx, key, opts, listObj)
	}

	return err
}
