# Forked cAdvisor

This is a minimized fork of [github.com/google/cadvisor](https://github.com/google/cadvisor).

## Source

Forked from github.com/google/cadvisor v0.53.0 (the version vendored in Kubernetes).

## Purpose

Kubernetes uses cAdvisor for container and machine metrics collection. This fork:
1. Removes libpfm/libipmctl conditional compilation (keeps only no-op stubs)
2. Strips down to only the packages required by Kubernetes
3. Eliminates the external go.mod dependency on github.com/google/cadvisor

## What's Included

Only packages actually imported by Kubernetes code:
- `cache/memory` - In-memory stats caching
- `client/v2` - HTTP client (for e2e tests)
- `collector` - Metrics collection framework
- `container/*` - Container runtime handlers (containerd, crio, systemd, raw)
- `devicemapper` - DeviceMapper thin pool support (used by fs)
- `events` - Event handling
- `fs` - Filesystem stats
- `info/v1`, `info/v2` - Type definitions
- `machine` - Machine info
- `manager` - Core manager
- `metrics` - Prometheus metrics
- `nvm` - NVM support (no-op stub only)
- `perf` - Perf events (no-op stub only)
- `resctrl` - Resource control
- `stats` - Stats interfaces
- `storage` - Storage interfaces
- `summary` - Stats summaries
- `utils/*` - Utilities (oomparser, sysfs, sysinfo, cpuload, cloudinfo)
- `version` - Version info
- `watcher` - Container watcher

## What's Removed

- Full libpfm implementation (perf profiling with libpfm4)
- Full libipmctl implementation (Intel NVM)
- Build tag conditional files (only keeping the no-op versions)

## Maintenance

When updating cAdvisor, copy the relevant files from the upstream vendor and ensure
imports are updated to use `k8s.io/kubernetes/third_party/forked/cadvisor/...`.
