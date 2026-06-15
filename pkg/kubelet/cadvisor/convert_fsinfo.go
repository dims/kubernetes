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

package cadvisor

import (
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"

	"k8s.io/kubernetes/pkg/kubelet/machine"
)

// ToFsInfo converts a cAdvisor v2 FsInfo into the kubelet-owned machine.FsInfo.
func ToFsInfo(fs cadvisorapiv2.FsInfo) machine.FsInfo {
	return machine.FsInfo{
		Timestamp:  fs.Timestamp,
		Device:     fs.Device,
		Mountpoint: fs.Mountpoint,
		Capacity:   fs.Capacity,
		Available:  fs.Available,
		Usage:      fs.Usage,
		Labels:     fs.Labels,
		Inodes:     fs.Inodes,
		InodesFree: fs.InodesFree,
	}
}
