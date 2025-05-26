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

func TestPrepareContinueToken(t *testing.T) {
	emptyPredicate := SelectionPredicate{Limit: 5}
	nonEmptyPredicate := SelectionPredicate{
		Limit: 5,
		Field: fields.SelectorFromSet(fields.Set{"metadata.name": "test"}),
	}

	tests := []struct {
		name                 string
		keyLastItem          string
		keyPrefix            string
		resourceVersion      int64
		itemsCount           int64
		hasMoreItems         bool
		opts                 ListOptions
		expectedContinueVal  bool // just check if empty or not
		expectedRemainingCnt bool // just check if nil or not
		expectedErr          bool
	}{
		{
			name:                 "no more items",
			keyLastItem:          "lastKey",
			keyPrefix:            "lastKey",
			resourceVersion:      1,
			itemsCount:           10,
			hasMoreItems:         false,
			opts:                 ListOptions{},
			expectedContinueVal:  false,
			expectedRemainingCnt: false,
			expectedErr:          false,
		},
		{
			name:                 "has more items with empty predicate",
			keyLastItem:          "lastKey",
			keyPrefix:            "lastKey",
			resourceVersion:      1,
			itemsCount:           10,
			hasMoreItems:         true,
			opts:                 ListOptions{Predicate: emptyPredicate},
			expectedContinueVal:  true,
			expectedRemainingCnt: true,
			expectedErr:          false,
		},
		{
			name:                 "has more items with non-empty predicate",
			keyLastItem:          "lastKey",
			keyPrefix:            "lastKey",
			resourceVersion:      1,
			itemsCount:           10,
			hasMoreItems:         true,
			opts:                 ListOptions{Predicate: nonEmptyPredicate},
			expectedContinueVal:  true,
			expectedRemainingCnt: false,
			expectedErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			continueVal, remainingCnt, err := PrepareContinueToken(tt.keyLastItem, tt.keyPrefix, tt.resourceVersion, tt.itemsCount, tt.hasMoreItems, tt.opts)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.expectedContinueVal {
					assert.NotEmpty(t, continueVal)
				} else {
					assert.Empty(t, continueVal)
				}

				if tt.expectedRemainingCnt {
					assert.NotNil(t, remainingCnt)
				} else {
					assert.Nil(t, remainingCnt)
				}
			}
		})
	}
}
