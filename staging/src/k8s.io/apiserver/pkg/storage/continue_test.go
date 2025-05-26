/*
Copyright 2022 The Kubernetes Authors.

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

package storage

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

func encodeContinueOrDie(apiVersion string, resourceVersion int64, nextKey string) string {
	out, err := json.Marshal(&continueToken{APIVersion: apiVersion, ResourceVersion: resourceVersion, StartKey: nextKey})
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(out)
}

func Test_decodeContinue(t *testing.T) {
	type args struct {
		continueValue string
		keyPrefix     string
	}
	tests := []struct {
		name        string
		args        args
		wantFromKey string
		wantRv      int64
		wantErr     error
	}{
		{
			name:        "valid",
			args:        args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "key"), keyPrefix: "/test/"},
			wantRv:      1,
			wantFromKey: "/test/key",
		},
		{
			name:        "root path",
			args:        args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "/"), keyPrefix: "/test/"},
			wantRv:      1,
			wantFromKey: "/test/",
		},
		{
			name:    "empty version",
			args:    args{continueValue: encodeContinueOrDie("", 1, "key"), keyPrefix: "/test/"},
			wantErr: ErrUnrecognizedEncodedVersion,
		},
		{
			name:    "invalid version",
			args:    args{continueValue: encodeContinueOrDie("v1", 1, "key"), keyPrefix: "/test/"},
			wantErr: ErrUnrecognizedEncodedVersion,
		},
		{
			name:    "invalid RV",
			args:    args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 0, "key"), keyPrefix: "/test/"},
			wantErr: ErrInvalidStartRV,
		},
		{
			name:    "no start Key",
			args:    args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, ""), keyPrefix: "/test/"},
			wantErr: ErrEmptyStartKey,
		},
		{
			name:    "path traversal - parent",
			args:    args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "../key"), keyPrefix: "/test/"},
			wantErr: ErrGenericInvalidKey,
		},
		{
			name:    "path traversal - local",
			args:    args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "./key"), keyPrefix: "/test/"},
			wantErr: ErrGenericInvalidKey,
		},
		{
			name:    "path traversal - double parent",
			args:    args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "./../key"), keyPrefix: "/test/"},
			wantErr: ErrGenericInvalidKey,
		},
		{
			name:    "path traversal - after parent",
			args:    args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "key/../.."), keyPrefix: "/test/"},
			wantErr: ErrGenericInvalidKey,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFromKey, gotRv, err := DecodeContinue(tt.args.continueValue, tt.args.keyPrefix)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("decodeContinue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotFromKey != tt.wantFromKey {
				t.Errorf("decodeContinue() gotFromKey = %v, want %v", gotFromKey, tt.wantFromKey)
			}
			if gotRv != tt.wantRv {
				t.Errorf("decodeContinue() gotRv = %v, want %v", gotRv, tt.wantRv)
			}
		})
	}
}

func TestPrepareContinueTokenWithLargeLimit(t *testing.T) {
	emptyPredicate := SelectionPredicate{Limit: 50000} // Large limit value
	// Initialize the Label and Field to make Empty() return true
	emptyPredicate.Label = labels.Everything()
	emptyPredicate.Field = fields.Everything()

	tests := []struct {
		name                 string
		keyLastItem          string
		keyPrefix            string
		resourceVersion      int64
		itemsCount           int64
		hasMoreItems         bool
		opts                 ListOptions
		expectedRemainingCnt int64
		expectedHasRemaining bool
	}{
		{
			name:                 "large limit exceeds item count",
			keyLastItem:          "lastKey",
			keyPrefix:            "lastKey",
			resourceVersion:      1,
			itemsCount:           1000, // Fewer items than the limit
			hasMoreItems:         true,
			opts:                 ListOptions{Predicate: emptyPredicate},
			expectedRemainingCnt: 0,
			expectedHasRemaining: true,
		},
		{
			name:                 "item count exceeds large limit",
			keyLastItem:          "lastKey",
			keyPrefix:            "lastKey",
			resourceVersion:      1,
			itemsCount:           60000, // More items than the limit
			hasMoreItems:         true,
			opts:                 ListOptions{Predicate: emptyPredicate},
			expectedRemainingCnt: 10000, // 60000 - 50000
			expectedHasRemaining: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			continueVal, remainingCnt, err := PrepareContinueToken(tt.keyLastItem, tt.keyPrefix, tt.resourceVersion, tt.itemsCount, tt.hasMoreItems, tt.opts)

			assert.NoError(t, err)
			assert.NotEmpty(t, continueVal)

			if tt.expectedHasRemaining {
				assert.NotNil(t, remainingCnt)
				assert.Equal(t, tt.expectedRemainingCnt, *remainingCnt)
			} else {
				assert.Nil(t, remainingCnt)
			}
		})
	}
}
