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

package machine

import "time"

// FsInfo describes a filesystem on the host (e.g. the root or image
// filesystem). The fields mirror github.com/google/cadvisor/info/v2.FsInfo.
type FsInfo struct {
	Timestamp  time.Time
	Device     string
	Mountpoint string
	Capacity   uint64
	Available  uint64
	Usage      uint64
	Labels     []string
	Inodes     *uint64
	InodesFree *uint64
}
