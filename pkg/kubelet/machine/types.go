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

// Package machine defines kubelet-owned types describing the host machine
// CPU/NUMA/memory topology, consumed by the node resource managers. It imports
// no cAdvisor package; pkg/kubelet/cadvisor converts cAdvisor's MachineInfo into
// these types.
package machine

// MachineInfo describes the topology-relevant properties of a machine that the
// node resource managers need. The fields mirror the corresponding fields of
// github.com/google/cadvisor/info/v1.MachineInfo.
type MachineInfo struct {
	// NumCores is the number of logical cores in this machine.
	NumCores int
	// NumPhysicalCores is the number of physical cores in this machine.
	NumPhysicalCores int
	// NumSockets is the number of cpu sockets in this machine.
	NumSockets int
	// MemoryCapacity is the amount of memory (in bytes) in this machine.
	MemoryCapacity uint64
	// SwapCapacity is the amount of swap (in bytes) in this machine.
	SwapCapacity uint64
	// HugePages on this machine.
	HugePages []HugePagesInfo
	// Topology describes the cpu/memory layout and hierarchy.
	Topology []Node
}

// Node describes a single NUMA node.
type Node struct {
	Id        int
	Memory    uint64
	HugePages []HugePagesInfo
	Cores     []Core
	Caches    []Cache
	Distances []uint64
}

// Core describes a single physical core and its logical threads.
type Core struct {
	Id           int
	Threads      []int
	Caches       []Cache
	UncoreCaches []Cache
	SocketID     int
	BookID       string
	DrawerID     string
}

// Cache describes a single CPU cache.
type Cache struct {
	Id    int
	Size  uint64
	Type  string
	Level int
}

// HugePagesInfo describes a hugepage size and count.
type HugePagesInfo struct {
	PageSize uint64
	NumPages uint64
}
