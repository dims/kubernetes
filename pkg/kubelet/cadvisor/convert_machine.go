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
	cadvisorapi "github.com/google/cadvisor/info/v1"

	"k8s.io/kubernetes/pkg/kubelet/machine"
)

// ToMachineInfo converts a cAdvisor MachineInfo into the kubelet-owned
// machine.MachineInfo, copying the topology and identity fields.
func ToMachineInfo(mi *cadvisorapi.MachineInfo) *machine.MachineInfo {
	if mi == nil {
		return nil
	}
	return &machine.MachineInfo{
		NumCores:         mi.NumCores,
		NumPhysicalCores: mi.NumPhysicalCores,
		NumSockets:       mi.NumSockets,
		MemoryCapacity:   mi.MemoryCapacity,
		SwapCapacity:     mi.SwapCapacity,
		HugePages:        hugePagesToTopology(mi.HugePages),
		Topology:         nodesToTopology(mi.Topology),
		MachineID:        mi.MachineID,
		SystemUUID:       mi.SystemUUID,
		BootID:           mi.BootID,
	}
}

// ToVersionInfo converts a cAdvisor VersionInfo into the kubelet-owned
// machine.VersionInfo.
func ToVersionInfo(vi *cadvisorapi.VersionInfo) *machine.VersionInfo {
	if vi == nil {
		return nil
	}
	return &machine.VersionInfo{
		KernelVersion:      vi.KernelVersion,
		ContainerOsVersion: vi.ContainerOsVersion,
		DockerVersion:      vi.DockerVersion,
		DockerAPIVersion:   vi.DockerAPIVersion,
		CadvisorVersion:    vi.CadvisorVersion,
		CadvisorRevision:   vi.CadvisorRevision,
	}
}

func nodesToTopology(nodes []cadvisorapi.Node) []machine.Node {
	if nodes == nil {
		return nil
	}
	out := make([]machine.Node, len(nodes))
	for i := range nodes {
		out[i] = machine.Node{
			Id:        nodes[i].Id,
			Memory:    nodes[i].Memory,
			HugePages: hugePagesToTopology(nodes[i].HugePages),
			Cores:     coresToTopology(nodes[i].Cores),
			Caches:    cachesToTopology(nodes[i].Caches),
			Distances: nodes[i].Distances,
		}
	}
	return out
}

func coresToTopology(cores []cadvisorapi.Core) []machine.Core {
	if cores == nil {
		return nil
	}
	out := make([]machine.Core, len(cores))
	for i, c := range cores {
		out[i] = machine.Core{
			Id:           c.Id,
			Threads:      c.Threads,
			Caches:       cachesToTopology(c.Caches),
			UncoreCaches: cachesToTopology(c.UncoreCaches),
			SocketID:     c.SocketID,
			BookID:       c.BookID,
			DrawerID:     c.DrawerID,
		}
	}
	return out
}

func cachesToTopology(caches []cadvisorapi.Cache) []machine.Cache {
	if caches == nil {
		return nil
	}
	out := make([]machine.Cache, len(caches))
	for i, c := range caches {
		out[i] = machine.Cache{Id: c.Id, Size: c.Size, Type: c.Type, Level: c.Level}
	}
	return out
}

func hugePagesToTopology(hp []cadvisorapi.HugePagesInfo) []machine.HugePagesInfo {
	if hp == nil {
		return nil
	}
	out := make([]machine.HugePagesInfo, len(hp))
	for i, h := range hp {
		out[i] = machine.HugePagesInfo{PageSize: h.PageSize, NumPages: h.NumPages}
	}
	return out
}
