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

// Package containerstats defines kubelet-owned types describing a container's
// resource usage. The fields mirror the subset of github.com/google/cadvisor's
// info v1/v2 stats types consumed by the kubelet stats providers, using the same
// field names so callers port without changing their bodies. It imports no
// cAdvisor package; pkg/kubelet/cadvisor converts cAdvisor's types into these.
package containerstats

import "time"

// Container identifier types for RequestOptions.IdType.
const (
	TypeName   = "name"
	TypeDocker = "docker"
)

// RequestOptions mirrors github.com/google/cadvisor/info/v2.RequestOptions.
type RequestOptions struct {
	IdType    string
	Count     int
	Recursive bool
	MaxAge    *time.Duration
}

// ContainerInfo mirrors github.com/google/cadvisor/info/v2.ContainerInfo.
type ContainerInfo struct {
	Spec  ContainerSpec
	Stats []*ContainerStats
}

// ContainerSpec mirrors the consumed fields of v2.ContainerSpec.
type ContainerSpec struct {
	CreationTime     time.Time
	HasCpu           bool
	HasMemory        bool
	HasNetwork       bool
	HasDiskIo        bool
	HasCustomMetrics bool
	Memory           MemorySpec
	CustomMetrics    []MetricSpec
	// Labels are the metadata labels associated with the container.
	Labels map[string]string
}

// MemorySpec mirrors the consumed fields of v2.MemorySpec.
type MemorySpec struct {
	Limit     uint64
	SwapLimit uint64
}

// ContainerStats mirrors the consumed fields of v2.ContainerStats.
type ContainerStats struct {
	Timestamp     time.Time
	Cpu           *CpuStats
	CpuInst       *CpuInstStats
	Memory        *MemoryStats
	Network       *NetworkStats
	Filesystem    *FilesystemStats
	DiskIo        *DiskIoStats
	Accelerators  []AcceleratorStats
	Processes     *ProcessStats
	CustomMetrics map[string][]MetricVal
}

// CpuStats mirrors the consumed fields of v1.CpuStats.
type CpuStats struct {
	Usage CpuUsage
	PSI   PSIStats
}

// CpuUsage mirrors the consumed fields of v1.CpuUsage.
type CpuUsage struct {
	Total uint64
}

// CpuInstStats mirrors v2.CpuInstStats.
type CpuInstStats struct {
	Usage CpuInstUsage
}

// CpuInstUsage mirrors the consumed fields of v2.CpuInstUsage.
type CpuInstUsage struct {
	Total uint64
}

// MemoryStats mirrors the consumed fields of v1.MemoryStats.
type MemoryStats struct {
	Usage         uint64
	WorkingSet    uint64
	RSS           uint64
	Swap          uint64
	ContainerData MemoryStatsMemoryData
	PSI           PSIStats
}

// MemoryStatsMemoryData mirrors the consumed fields of v1.MemoryStatsMemoryData.
type MemoryStatsMemoryData struct {
	Pgfault    uint64
	Pgmajfault uint64
}

// NetworkStats mirrors the consumed fields of v2.NetworkStats.
type NetworkStats struct {
	Interfaces []InterfaceStats
}

// InterfaceStats mirrors the consumed fields of v1.InterfaceStats.
type InterfaceStats struct {
	Name     string
	RxBytes  uint64
	RxErrors uint64
	TxBytes  uint64
	TxErrors uint64
}

// FilesystemStats mirrors v2.FilesystemStats.
type FilesystemStats struct {
	TotalUsageBytes *uint64
	BaseUsageBytes  *uint64
	InodeUsage      *uint64
}

// DiskIoStats mirrors the consumed fields of v1.DiskIoStats.
type DiskIoStats struct {
	PSI PSIStats
}

// AcceleratorStats mirrors v1.AcceleratorStats.
type AcceleratorStats struct {
	Make        string
	Model       string
	ID          string
	MemoryTotal uint64
	MemoryUsed  uint64
	DutyCycle   uint64
}

// ProcessStats mirrors the consumed fields of v1.ProcessStats.
type ProcessStats struct {
	ProcessCount uint64
}

// PSIStats mirrors v1.PSIStats.
type PSIStats struct {
	Full PSIData
	Some PSIData
}

// PSIData mirrors v1.PSIData.
type PSIData struct {
	Total  uint64
	Avg10  float64
	Avg60  float64
	Avg300 float64
}

// MetricType mirrors v1.MetricType.
type MetricType string

const (
	MetricGauge      MetricType = "gauge"
	MetricCumulative MetricType = "cumulative"
)

// DataType mirrors v1.DataType.
type DataType string

const (
	IntType   DataType = "int"
	FloatType DataType = "float"
)

// MetricSpec mirrors the consumed fields of v1.MetricSpec.
type MetricSpec struct {
	Name   string
	Type   MetricType
	Format DataType
	Units  string
}

// MetricVal mirrors the consumed fields of v1.MetricVal.
type MetricVal struct {
	Timestamp  time.Time
	IntValue   int64
	FloatValue float64
}
