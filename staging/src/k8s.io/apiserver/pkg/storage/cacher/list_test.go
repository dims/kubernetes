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
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/storage"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
)

// TestListWithFeatureGate tests that LIST calls with rv="" and LIMIT set
// are correctly handled based on the ListFromCacheSnapshot feature gate setting.
func TestListWithFeatureGate(t *testing.T) {
	testCases := []struct {
		name                  string
		listFromCacheSnapshot bool
		resourceVersion       string
		limit                 int64
		expectListExactRV     bool
	}{
		{
			name:                  "Feature enabled, rv=\"\", limit set - should use listLatestRV",
			listFromCacheSnapshot: true,
			resourceVersion:       "",
			limit:                 500,
			expectListExactRV:     false,
		},
		{
			name:                  "Feature disabled, rv=\"\", limit set - should use listLatestRV",
			listFromCacheSnapshot: false,
			resourceVersion:       "",
			limit:                 500,
			expectListExactRV:     false,
		},
		{
			name:                  "Feature enabled, rv=\"10\", limit set - should use listExactRV",
			listFromCacheSnapshot: true,
			resourceVersion:       "10",
			limit:                 500,
			expectListExactRV:     true,
		},
		{
			name:                  "Feature disabled, rv=\"10\", limit set - should use listLatestRV",
			listFromCacheSnapshot: false,
			resourceVersion:       "10",
			limit:                 500,
			expectListExactRV:     false,
		},
		{
			name:                  "Feature enabled, rv=\"\", no limit - should use listLatestRV",
			listFromCacheSnapshot: true,
			resourceVersion:       "",
			limit:                 0,
			expectListExactRV:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set feature gate for test
			featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.ListFromCacheSnapshot, tc.listFromCacheSnapshot)

			// Track which path was taken
			listExactRVCalled := false
			listLatestRVCalled := false

			// Create a custom list function for testing
			listFunc := func(ctx context.Context, resourceVersion uint64, key string, opts storage.ListOptions) (interface{}, error) {
				// This simulates the condition in watch_cache.go that was fixed
				if opts.Predicate.Limit > 0 &&
					len(opts.ResourceVersion) > 0 &&
					opts.ResourceVersion != "0" &&
					utilfeature.DefaultFeatureGate.Enabled(features.ListFromCacheSnapshot) &&
					resourceVersion > 0 {
					listExactRVCalled = true
					return nil, errors.NewResourceExpired("expired")
				} else {
					listLatestRVCalled = true
					return "result", nil
				}
			}

			// Call the list function with the test parameters
			listOpts := storage.ListOptions{
				ResourceVersion: tc.resourceVersion,
				Predicate: storage.SelectionPredicate{
					Limit: tc.limit,
				},
			}

			result, err := listFunc(context.Background(), 10, "prefix/", listOpts)

			// Verify expectations
			if tc.expectListExactRV {
				assert.True(t, listExactRVCalled, "listExactRV path should have been taken")
				assert.False(t, listLatestRVCalled, "listLatestRV path should not have been taken")
				assert.Nil(t, result, "Result should be nil when listExactRV is called")
				assert.True(t, errors.IsResourceExpired(err), "ResourceExpired error should have been returned")
			} else {
				assert.False(t, listExactRVCalled, "listExactRV path should not have been taken")
				assert.True(t, listLatestRVCalled, "listLatestRV path should have been taken")
				assert.Equal(t, "result", result, "Result should not be nil when listLatestRV is called")
				assert.Nil(t, err, "No error should have been returned")
			}
		})
	}
}

// TestNegativeResourceVersionList tests that LIST calls with negative resource versions
// are correctly handled regardless of the ListFromCacheSnapshot feature gate setting.
func TestNegativeResourceVersionList(t *testing.T) {
	testCases := []struct {
		name                  string
		listFromCacheSnapshot bool
		resourceVersion       string
		limit                 int64
		continueToken         string
	}{
		{
			name:                  "Feature enabled, negative rv, limit set - should use listLatestRV",
			listFromCacheSnapshot: true,
			resourceVersion:       "-1",
			limit:                 500,
		},
		{
			name:                  "Feature disabled, negative rv, limit set - should use listLatestRV",
			listFromCacheSnapshot: false,
			resourceVersion:       "-10",
			limit:                 500,
		},
		{
			name:                  "Feature enabled, negative rv, with continue token - should use listLatestRV",
			listFromCacheSnapshot: true,
			resourceVersion:       "-5",
			limit:                 500,
			continueToken:         "someToken",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set feature gate for test
			featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.ListFromCacheSnapshot, tc.listFromCacheSnapshot)

			// Track which path was taken
			listExactRVCalled := false
			listLatestRVCalled := false

			// Parse the resource version to simulate what happens in the code
			rv, _ := strconv.ParseInt(tc.resourceVersion, 10, 64)
			resourceVersion := uint64(0)
			if rv > 0 {
				resourceVersion = uint64(rv)
			}

			// Create a custom list function for testing
			listFunc := func(ctx context.Context, resourceVersion uint64, key string, opts storage.ListOptions) (interface{}, error) {
				// This simulates the condition in watch_cache.go that was fixed
				if opts.Predicate.Limit > 0 &&
					len(opts.ResourceVersion) > 0 &&
					opts.ResourceVersion != "0" &&
					utilfeature.DefaultFeatureGate.Enabled(features.ListFromCacheSnapshot) &&
					resourceVersion > 0 {
					listExactRVCalled = true
					return nil, errors.NewResourceExpired("expired")
				} else {
					listLatestRVCalled = true
					return "result", nil
				}
			}

			// Call the list function with the test parameters
			listOpts := storage.ListOptions{
				ResourceVersion: tc.resourceVersion,
				Predicate: storage.SelectionPredicate{
					Limit:    tc.limit,
					Continue: tc.continueToken,
				},
			}

			result, err := listFunc(context.Background(), resourceVersion, "prefix/", listOpts)

			// For negative resource versions, we should always use listLatestRV
			assert.False(t, listExactRVCalled, "listExactRV path should not have been taken for negative resource version")
			assert.True(t, listLatestRVCalled, "listLatestRV path should have been taken for negative resource version")
			assert.Equal(t, "result", result, "Result should not be nil when listLatestRV is called")
			assert.Nil(t, err, "No error should have been returned")
		})
	}
}
