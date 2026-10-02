/*
Copyright 2023 The Kubernetes Authors.

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

// Package identity holds the constants that identify kube-apiserver identity leases.
// It has no imports, so controllers can use them without importing the control plane.
package identity

const (
	// LeaseComponentLabelKey is the label on identity lease objects that names the component holding the lease.
	LeaseComponentLabelKey = "apiserver.kubernetes.io/identity"
	// KubeAPIServer is the component value for kube-apiserver.
	KubeAPIServer = "kube-apiserver"
)
