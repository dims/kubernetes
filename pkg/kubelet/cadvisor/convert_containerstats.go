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
	cadvisorapiv1 "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"

	"k8s.io/kubernetes/pkg/kubelet/containerstats"
)

// RequestOptionsToCadvisor converts kubelet-owned RequestOptions into the
// cAdvisor v2 form accepted by the cAdvisor manager.
func RequestOptionsToCadvisor(o containerstats.RequestOptions) cadvisorapiv2.RequestOptions {
	return cadvisorapiv2.RequestOptions{
		IdType:    o.IdType,
		Count:     o.Count,
		Recursive: o.Recursive,
		MaxAge:    o.MaxAge,
	}
}

// ToContainerInfoMap converts a map of cAdvisor v2 ContainerInfo into the
// kubelet-owned containerstats.ContainerInfo map.
func ToContainerInfoMap(in map[string]cadvisorapiv2.ContainerInfo) map[string]containerstats.ContainerInfo {
	if in == nil {
		return nil
	}
	out := make(map[string]containerstats.ContainerInfo, len(in))
	for k, v := range in {
		out[k] = toContainerInfo(v)
	}
	return out
}

func toContainerInfo(in cadvisorapiv2.ContainerInfo) containerstats.ContainerInfo {
	out := containerstats.ContainerInfo{Spec: toContainerSpec(in.Spec)}
	if in.Stats != nil {
		out.Stats = make([]*containerstats.ContainerStats, len(in.Stats))
		for i, s := range in.Stats {
			out.Stats[i] = toContainerStats(s)
		}
	}
	return out
}

func toContainerSpec(in cadvisorapiv2.ContainerSpec) containerstats.ContainerSpec {
	return containerstats.ContainerSpec{
		CreationTime:     in.CreationTime,
		HasCpu:           in.HasCpu,
		HasMemory:        in.HasMemory,
		HasNetwork:       in.HasNetwork,
		HasDiskIo:        in.HasDiskIo,
		HasCustomMetrics: in.HasCustomMetrics,
		Memory: containerstats.MemorySpec{
			Limit:     in.Memory.Limit,
			SwapLimit: in.Memory.SwapLimit,
		},
		CustomMetrics: toMetricSpecs(in.CustomMetrics),
		Labels:        in.Labels,
	}
}

func toContainerStats(in *cadvisorapiv2.ContainerStats) *containerstats.ContainerStats {
	if in == nil {
		return nil
	}
	out := &containerstats.ContainerStats{
		Timestamp:     in.Timestamp,
		Accelerators:  toAccelerators(in.Accelerators),
		CustomMetrics: toMetricVals(in.CustomMetrics),
	}
	if in.Cpu != nil {
		out.Cpu = &containerstats.CpuStats{
			Usage: containerstats.CpuUsage{Total: in.Cpu.Usage.Total},
			PSI:   toPSI(in.Cpu.PSI),
		}
	}
	if in.CpuInst != nil {
		out.CpuInst = &containerstats.CpuInstStats{Usage: containerstats.CpuInstUsage{Total: in.CpuInst.Usage.Total}}
	}
	if in.Memory != nil {
		out.Memory = &containerstats.MemoryStats{
			Usage:      in.Memory.Usage,
			WorkingSet: in.Memory.WorkingSet,
			RSS:        in.Memory.RSS,
			Swap:       in.Memory.Swap,
			ContainerData: containerstats.MemoryStatsMemoryData{
				Pgfault:    in.Memory.ContainerData.Pgfault,
				Pgmajfault: in.Memory.ContainerData.Pgmajfault,
			},
			PSI: toPSI(in.Memory.PSI),
		}
	}
	if in.Network != nil {
		out.Network = &containerstats.NetworkStats{Interfaces: toInterfaces(in.Network.Interfaces)}
	}
	if in.Filesystem != nil {
		out.Filesystem = &containerstats.FilesystemStats{
			TotalUsageBytes: in.Filesystem.TotalUsageBytes,
			BaseUsageBytes:  in.Filesystem.BaseUsageBytes,
			InodeUsage:      in.Filesystem.InodeUsage,
		}
	}
	if in.DiskIo != nil {
		out.DiskIo = &containerstats.DiskIoStats{PSI: toPSI(in.DiskIo.PSI)}
	}
	if in.Processes != nil {
		out.Processes = &containerstats.ProcessStats{ProcessCount: in.Processes.ProcessCount}
	}
	return out
}

func toPSI(in cadvisorapiv1.PSIStats) containerstats.PSIStats {
	return containerstats.PSIStats{
		Full: containerstats.PSIData{Total: in.Full.Total, Avg10: in.Full.Avg10, Avg60: in.Full.Avg60, Avg300: in.Full.Avg300},
		Some: containerstats.PSIData{Total: in.Some.Total, Avg10: in.Some.Avg10, Avg60: in.Some.Avg60, Avg300: in.Some.Avg300},
	}
}

func toInterfaces(in []cadvisorapiv1.InterfaceStats) []containerstats.InterfaceStats {
	if in == nil {
		return nil
	}
	out := make([]containerstats.InterfaceStats, len(in))
	for i, n := range in {
		out[i] = containerstats.InterfaceStats{
			Name:     n.Name,
			RxBytes:  n.RxBytes,
			RxErrors: n.RxErrors,
			TxBytes:  n.TxBytes,
			TxErrors: n.TxErrors,
		}
	}
	return out
}

func toAccelerators(in []cadvisorapiv1.AcceleratorStats) []containerstats.AcceleratorStats {
	if in == nil {
		return nil
	}
	out := make([]containerstats.AcceleratorStats, len(in))
	for i, a := range in {
		out[i] = containerstats.AcceleratorStats{
			Make:        a.Make,
			Model:       a.Model,
			ID:          a.ID,
			MemoryTotal: a.MemoryTotal,
			MemoryUsed:  a.MemoryUsed,
			DutyCycle:   a.DutyCycle,
		}
	}
	return out
}

func toMetricSpecs(in []cadvisorapiv1.MetricSpec) []containerstats.MetricSpec {
	if in == nil {
		return nil
	}
	out := make([]containerstats.MetricSpec, len(in))
	for i, m := range in {
		out[i] = containerstats.MetricSpec{
			Name:   m.Name,
			Type:   containerstats.MetricType(m.Type),
			Format: containerstats.DataType(m.Format),
			Units:  m.Units,
		}
	}
	return out
}

func toMetricVals(in map[string][]cadvisorapiv1.MetricVal) map[string][]containerstats.MetricVal {
	if in == nil {
		return nil
	}
	out := make(map[string][]containerstats.MetricVal, len(in))
	for k, vals := range in {
		cv := make([]containerstats.MetricVal, len(vals))
		for i, v := range vals {
			cv[i] = containerstats.MetricVal{Timestamp: v.Timestamp, IntValue: v.IntValue, FloatValue: v.FloatValue}
		}
		out[k] = cv
	}
	return out
}
