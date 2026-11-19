/*
Copyright 2016 The Kubernetes Authors.

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

package node

import (
	"context"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	admissionapi "k8s.io/pod-security-admission/api"
)

const (
	etcHostsPodName             = "test-pod"
	etcHostsHostNetworkPodName  = "test-host-network-pod"
	etcHostsPartialContent      = "# Kubernetes-managed hosts file."
	etcHostsPathLinux           = "/etc/hosts"
	etcHostsPathWindows         = `C:\Windows\System32\drivers\etc\hosts`
	etcHostsOriginalPathLinux   = "/etc/hosts-original"
	etcHostsOriginalPathWindows = `C:\Windows\System32\drivers\etc\hosts-original`
	etcHostsConfigMapName       = "etc-hosts"
	etcHostsConfigMapKey        = "hosts"
	// Must not start with etcHostsPartialContent, or a container that mounts
	// this file could not be told apart from one with a kubelet-managed file.
	etcHostsConfigMapContent = "# hosts file from a ConfigMap.\n127.0.0.1\tlocalhost\n"
)

// KubeletManagedHostConfig defines the types for running managed etc hosts test cases
type KubeletManagedHostConfig struct {
	hostNetworkPod *v1.Pod
	pod            *v1.Pod
	f              *framework.Framework
}

var _ = SIGDescribe("KubeletManagedEtcHosts", func() {
	f := framework.NewDefaultFramework("e2e-kubelet-etc-hosts")
	f.NamespacePodSecurityLevel = admissionapi.LevelPrivileged
	config := &KubeletManagedHostConfig{
		f: f,
	}

	/*
		Release: v1.9
		Testname: Kubelet, managed etc hosts
		Description: Create a Pod with containers with hostNetwork set to false, one of the containers mounts its own /etc/hosts file from a ConfigMap volume.
			1. The Pod with hostNetwork=false MUST have /etc/hosts of containers managed by the Kubelet.
			2. The Pod with hostNetwork=false but the container mounts its own /etc/hosts file. The /etc/hosts file MUST not be managed by the Kubelet.
	*/
	framework.ConformanceIt("should test kubelet managed /etc/hosts file", f.WithNodeConformance(), func(ctx context.Context) {
		ginkgo.By("Creating hostNetwork=false pod")
		config.createPodWithoutHostNetwork(ctx)

		ginkgo.By("Verifying /etc/hosts of container is kubelet-managed for pod with hostNetwork=false")
		assertManagedStatus(config, etcHostsPodName, true, "busybox-1")
		assertManagedStatus(config, etcHostsPodName, true, "busybox-2")

		ginkgo.By("Verifying /etc/hosts of container is not kubelet-managed since container specifies /etc/hosts mount")
		assertManagedStatus(config, etcHostsPodName, false, "busybox-3")
	})

	framework.It("should test kubelet managed /etc/hosts file with hostNetwork", f.WithNodeConformance(), func(ctx context.Context) {
		ginkgo.By("Creating hostNetwork=true pod")
		config.createPodWithHostNetwork(ctx)

		ginkgo.By("Verifying /etc/hosts content of container is not kubelet-managed for pod with hostNetwork=true")
		assertManagedStatus(config, etcHostsHostNetworkPodName, false, "busybox-1")
		assertManagedStatus(config, etcHostsHostNetworkPodName, false, "busybox-2")
	})
})

func (config *KubeletManagedHostConfig) createPodWithoutHostNetwork(ctx context.Context) {
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: etcHostsConfigMapName,
		},
		Data: map[string]string{
			etcHostsConfigMapKey: etcHostsConfigMapContent,
		},
	}
	_, err := config.f.ClientSet.CoreV1().ConfigMaps(config.f.Namespace.Name).Create(ctx, configMap, metav1.CreateOptions{})
	framework.ExpectNoError(err, "creating ConfigMap %s", etcHostsConfigMapName)

	podSpec := config.createPodSpec(etcHostsPodName)
	config.pod = e2epod.NewPodClient(config.f).CreateSync(ctx, podSpec)
}

func (config *KubeletManagedHostConfig) createPodWithHostNetwork(ctx context.Context) {
	podSpec := config.createPodSpecWithHostNetwork(etcHostsHostNetworkPodName)
	config.hostNetworkPod = e2epod.NewPodClient(config.f).CreateSync(ctx, podSpec)
}

func assertManagedStatus(
	config *KubeletManagedHostConfig, podName string, expectedIsManaged bool, name string) {
	// TODO: workaround for https://github.com/kubernetes/kubernetes/issues/34256
	//
	// Retry until timeout for the contents of /etc/hosts to show
	// up. Note: if /etc/hosts is properly mounted, then this will
	// succeed immediately.
	const retryTimeout = 30 * time.Second

	retryCount := 0
	etcHostsContent := ""
	etcHostsPath, etcHostsOriginalPath := getEtcHostsPath()

	for startTime := time.Now(); time.Since(startTime) < retryTimeout; {
		etcHostsContent = config.getFileContents(podName, name, etcHostsPath)
		etcHostsOriginalContent := config.getFileContents(podName, name, etcHostsOriginalPath)

		// Make sure there is some content in both files
		if len(etcHostsContent) > 0 && len(etcHostsOriginalContent) > 0 {
			// if the files match, kubernetes did not touch the file at all
			// if the file has the header, kubernetes is not using host network
			// and is constructing the file based on Pod IP
			isManaged := strings.HasPrefix(etcHostsContent, etcHostsPartialContent) &&
				etcHostsContent != etcHostsOriginalContent
			if expectedIsManaged == isManaged {
				return
			}
		}

		klog.Warningf(
			"For pod: %s, name: %s, expected %t, (/etc/hosts was %q), (/etc/hosts-original was %q), retryCount: %d",
			podName, name, expectedIsManaged, etcHostsContent, etcHostsOriginalContent, retryCount)

		retryCount++
		time.Sleep(100 * time.Millisecond)
	}

	if expectedIsManaged {
		framework.Failf(
			"/etc/hosts file should be kubelet managed (name: %s, retries: %d). /etc/hosts contains %q",
			name, retryCount, etcHostsContent)
	} else {
		framework.Failf(
			"/etc/hosts file should no be kubelet managed (name: %s, retries: %d). /etc/hosts contains %q",
			name, retryCount, etcHostsContent)
	}
}

func (config *KubeletManagedHostConfig) getFileContents(podName, containerName, path string) string {
	return e2epod.ExecCommandInContainer(config.f, podName, containerName, "cat", path)
}

func (config *KubeletManagedHostConfig) createPodSpec(podName string) *v1.Pod {
	etcHostsPath, etcHostsOriginalPath := getEtcHostsPath()
	mounts := []v1.VolumeMount{
		{
			Name:      "etc-hosts",
			MountPath: etcHostsOriginalPath,
			SubPath:   etcHostsConfigMapKey,
		},
	}
	multipleMounts := []v1.VolumeMount{
		mounts[0],
		{
			Name:      "etc-hosts",
			MountPath: etcHostsPath,
			SubPath:   etcHostsConfigMapKey,
		},
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				e2epod.NewAgnhostContainer("busybox-1", mounts, nil),
				e2epod.NewAgnhostContainer("busybox-2", mounts, nil),
				e2epod.NewAgnhostContainer("busybox-3", multipleMounts, nil),
			},
			Volumes: []v1.Volume{
				{
					Name: "etc-hosts",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{Name: etcHostsConfigMapName},
						},
					},
				},
			},
		},
	}

	return pod
}

func (config *KubeletManagedHostConfig) createPodSpecWithHostNetwork(podName string) *v1.Pod {
	hostPathType := new(v1.HostPathType)
	*hostPathType = v1.HostPathType(string(v1.HostPathFileOrCreate))
	etcHostsPath, etcHostsOriginalPath := getEtcHostsPath()
	mounts := []v1.VolumeMount{
		{
			Name:      "host-etc-hosts",
			MountPath: etcHostsOriginalPath,
		},
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: v1.PodSpec{
			HostNetwork:     true,
			SecurityContext: &v1.PodSecurityContext{},
			Containers: []v1.Container{
				e2epod.NewAgnhostContainer("busybox-1", mounts, nil),
				e2epod.NewAgnhostContainer("busybox-2", mounts, nil),
			},
			Volumes: []v1.Volume{
				{
					Name: "host-etc-hosts",
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: etcHostsPath,
							Type: hostPathType,
						},
					},
				},
			},
		},
	}
	return pod
}

func getEtcHostsPath() (string, string) {
	if framework.NodeOSDistroIs("windows") {
		return etcHostsPathWindows, etcHostsOriginalPathWindows
	}
	return etcHostsPathLinux, etcHostsOriginalPathLinux
}
