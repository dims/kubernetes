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

package api

// MachineInfo mirrors the topology-relevant fields of
// github.com/google/cadvisor/info/v1.MachineInfo that the kubelet node resource
// managers consume. Field names match the cAdvisor type so callers port without
// touching their bodies.
type MachineInfo struct {
	// The number of cores in this machine.
	NumCores int
	// The number of physical cores in this machine.
	NumPhysicalCores int
	// The number of cpu sockets in this machine.
	NumSockets int
	// The amount of memory (in bytes) in this machine.
	MemoryCapacity uint64
	// The amount of swap (in bytes) in this machine.
	SwapCapacity uint64
	// HugePages on this machine.
	HugePages []HugePagesInfo
	// Machine topology: describes the cpu/memory layout and hierarchy.
	Topology []Node
}

// Node mirrors github.com/google/cadvisor/info/v1.Node (a NUMA node).
type Node struct {
	Id        int
	Memory    uint64
	HugePages []HugePagesInfo
	Cores     []Core
	Caches    []Cache
	Distances []uint64
}

// Core mirrors github.com/google/cadvisor/info/v1.Core.
type Core struct {
	Id           int
	Threads      []int
	Caches       []Cache
	UncoreCaches []Cache
	SocketID     int
	BookID       string
	DrawerID     string
}

// Cache mirrors github.com/google/cadvisor/info/v1.Cache.
type Cache struct {
	Id    int
	Size  uint64
	Type  string
	Level int
}

// HugePagesInfo mirrors github.com/google/cadvisor/info/v1.HugePagesInfo.
type HugePagesInfo struct {
	PageSize uint64
	NumPages uint64
}
