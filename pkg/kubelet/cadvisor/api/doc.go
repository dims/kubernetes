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

// Package api holds kubelet-owned types that mirror the subset of the cAdvisor
// info types consumed across the kubelet. It intentionally imports no cAdvisor
// package, so that callers can depend on these types without taking a direct
// dependency on cAdvisor. Conversions from the cAdvisor types into these mirror
// types live in the parent package, pkg/kubelet/cadvisor, which is the single
// place in the kubelet allowed to import cAdvisor.
package api
