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

	"k8s.io/kubernetes/pkg/kubelet/cadvisor/api"
)

// MachineInfoToAPI converts a cAdvisor MachineInfo into the kubelet-owned
// api.MachineInfo, copying the topology-relevant fields consumed by the node
// resource managers.
func MachineInfoToAPI(mi *cadvisorapi.MachineInfo) *api.MachineInfo {
	if mi == nil {
		return nil
	}
	return &api.MachineInfo{
		NumCores:         mi.NumCores,
		NumPhysicalCores: mi.NumPhysicalCores,
		NumSockets:       mi.NumSockets,
		MemoryCapacity:   mi.MemoryCapacity,
		SwapCapacity:     mi.SwapCapacity,
		HugePages:        hugePagesToAPI(mi.HugePages),
		Topology:         NodesToAPI(mi.Topology),
	}
}

// NodesToAPI converts cAdvisor NUMA nodes into kubelet-owned api.Node values.
func NodesToAPI(nodes []cadvisorapi.Node) []api.Node {
	if nodes == nil {
		return nil
	}
	out := make([]api.Node, len(nodes))
	for i := range nodes {
		out[i] = nodeToAPI(nodes[i])
	}
	return out
}

func nodeToAPI(n cadvisorapi.Node) api.Node {
	return api.Node{
		Id:        n.Id,
		Memory:    n.Memory,
		HugePages: hugePagesToAPI(n.HugePages),
		Cores:     coresToAPI(n.Cores),
		Caches:    cachesToAPI(n.Caches),
		Distances: n.Distances,
	}
}

func coresToAPI(cores []cadvisorapi.Core) []api.Core {
	if cores == nil {
		return nil
	}
	out := make([]api.Core, len(cores))
	for i, c := range cores {
		out[i] = api.Core{
			Id:           c.Id,
			Threads:      c.Threads,
			Caches:       cachesToAPI(c.Caches),
			UncoreCaches: cachesToAPI(c.UncoreCaches),
			SocketID:     c.SocketID,
			BookID:       c.BookID,
			DrawerID:     c.DrawerID,
		}
	}
	return out
}

func cachesToAPI(caches []cadvisorapi.Cache) []api.Cache {
	if caches == nil {
		return nil
	}
	out := make([]api.Cache, len(caches))
	for i, c := range caches {
		out[i] = api.Cache{Id: c.Id, Size: c.Size, Type: c.Type, Level: c.Level}
	}
	return out
}

func hugePagesToAPI(hp []cadvisorapi.HugePagesInfo) []api.HugePagesInfo {
	if hp == nil {
		return nil
	}
	out := make([]api.HugePagesInfo, len(hp))
	for i, h := range hp {
		out[i] = api.HugePagesInfo{PageSize: h.PageSize, NumPages: h.NumPages}
	}
	return out
}
