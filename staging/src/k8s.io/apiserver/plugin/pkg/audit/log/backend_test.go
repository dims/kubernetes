/*
Copyright 2017 The Kubernetes Authors.

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

package log

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/pborman/uuid"

	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	auditinternal "k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/apiserver/pkg/apis/audit/install"
	auditv1beta1 "k8s.io/apiserver/pkg/apis/audit/v1beta1"
	"k8s.io/apiserver/pkg/audit"
)

// NOTE: Copied from webhook backend to register auditv1beta1 to scheme
var (
	groupFactoryRegistry = make(announced.APIGroupFactoryRegistry)
	registry             = registered.NewOrDie("")
	payload = `{
	"kind": "Event",
	"apiVersion": "audit.k8s.io/v1beta1",
	"metadata": {
		"creationTimestamp": "2017-09-25T00:31:09Z"
	},
	"level": "RequestResponse",
	"timestamp": "2017-09-25T00:31:09Z",
	"auditID": "36384c8a-1394-4c92-8726-904e4c442b7d",
	"stage": "ResponseComplete",
	"requestURI": "/api/v1/namespaces/kubemark/pods",
	"verb": "deletecollection",
	"user": {
		"username": "system:serviceaccount:kube-system:namespace-controller",
		"uid": "84ebc427-a187-11e7-902b-42010a800002",
		"groups": ["system:serviceaccounts", "system:serviceaccounts:kube-system", "system:authenticated"]
	},
	"sourceIPs": ["::1"],
	"objectRef": {
		"resource": "pods",
		"namespace": "kubemark",
		"apiVersion": "v1"
	},
	"responseStatus": {
		"metadata": {},
		"code": 200
	},
	"requestObject": {
		"kind": "DeleteOptions",
		"apiVersion": "v1",
		"propagationPolicy": "Background"
	},
	"responseObject": {
		"kind": "PodList",
		"apiVersion": "v1",
		"metadata": {
			"selfLink": "/api/v1/namespaces/kubemark/pods",
			"resourceVersion": "1227"
		},
		"items": [{
			"metadata": {
				"name": "heapster-v1.3.0-hgjwm",
				"generateName": "heapster-v1.3.0-",
				"namespace": "kubemark",
				"selfLink": "/api/v1/namespaces/kubemark/pods/heapster-v1.3.0-hgjwm",
				"uid": "20cc10d6-a188-11e7-bb13-42010a800002",
				"resourceVersion": "1219",
				"creationTimestamp": "2017-09-25T00:26:09Z",
				"deletionTimestamp": "2017-09-25T00:31:34Z",
				"deletionGracePeriodSeconds": 30,
				"labels": {
					"k8s-app": "heapster",
					"version": "v1.3.0"
				},
				"annotations": {
					"kubernetes.io/created-by": "{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicationController\",\"namespace\":\"kubemark\",\"name\":\"heapster-v1.3.0\",\"uid\":\"20ca4411-a188-11e7-bb13-42010a800002\",\"apiVersion\":\"v1\",\"resourceVersion\":\"806\"}}\n"
				},
				"ownerReferences": [{
					"apiVersion": "v1",
					"kind": "ReplicationController",
					"name": "heapster-v1.3.0",
					"uid": "20ca4411-a188-11e7-bb13-42010a800002",
					"controller": true,
					"blockOwnerDeletion": true
				}]
			},
			"spec": {
				"volumes": [{
					"name": "kubeconfig-volume",
					"secret": {
						"secretName": "kubeconfig",
						"defaultMode": 420
					}
				}, {
					"name": "default-token-6552z",
					"secret": {
						"secretName": "default-token-6552z",
						"defaultMode": 420
					}
				}],
				"containers": [{
					"name": "heapster",
					"image": "gcr.io/google_containers/heapster:v1.3.0",
					"command": ["/heapster"],
					"args": ["--source=kubernetes:https://35.192.172.191:443?inClusterConfig=0\u0026useServiceAccount=0\u0026auth=/kubeconfig/heapster.kubeconfig"],
					"resources": {
						"requests": {
							"cpu": "82m",
							"memory": "220Mi"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"mountPath": "/kubeconfig"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "IfNotPresent"
				}, {
					"name": "eventer",
					"image": "gcr.io/google_containers/heapster:v1.3.0",
					"command": ["/eventer"],
					"args": ["--source=kubernetes:https://35.192.172.191:443?inClusterConfig=0\u0026useServiceAccount=0\u0026auth=/kubeconfig/heapster.kubeconfig"],
					"resources": {
						"requests": {
							"memory": "207300Ki"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"mountPath": "/kubeconfig"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "IfNotPresent"
				}],
				"restartPolicy": "Always",
				"terminationGracePeriodSeconds": 30,
				"dnsPolicy": "ClusterFirst",
				"serviceAccountName": "default",
				"serviceAccount": "default",
				"nodeName": "e2e-53532-minion-group-q6xr",
				"securityContext": {},
				"schedulerName": "default-scheduler",
				"tolerations": [{
					"key": "node.alpha.kubernetes.io/notReady",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}, {
					"key": "node.alpha.kubernetes.io/unreachable",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}]
			},
			"status": {
				"phase": "Running",
				"conditions": [{
					"type": "Initialized",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:09Z"
				}, {
					"type": "Ready",
					"status": "False",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:31:05Z",
					"reason": "ContainersNotReady",
					"message": "containers with unready status: [heapster eventer]"
				}, {
					"type": "PodScheduled",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:09Z"
				}],
				"hostIP": "10.128.0.3",
				"podIP": "10.64.1.13",
				"startTime": "2017-09-25T00:26:09Z",
				"containerStatuses": [{
					"name": "eventer",
					"state": {
						"terminated": {
							"exitCode": 2,
							"reason": "Error",
							"startedAt": "2017-09-25T00:26:16Z",
							"finishedAt": "2017-09-25T00:31:04Z",
							"containerID": "docker://e5b480d37b17f50dd56366ca14ea2c01a13dd2ec3f408b0e71e3ba79f7c05442"
						}
					},
					"lastState": {},
					"ready": false,
					"restartCount": 0,
					"image": "gcr.io/google_containers/heapster:v1.3.0",
					"imageID": "docker-pullable://gcr.io/google_containers/heapster@sha256:3dff9b2425a196aa51df0cebde0f8b427388425ba84568721acf416fa003cd5c",
					"containerID": "docker://e5b480d37b17f50dd56366ca14ea2c01a13dd2ec3f408b0e71e3ba79f7c05442"
				}, {
					"name": "heapster",
					"state": {
						"terminated": {
							"exitCode": 2,
							"reason": "Error",
							"startedAt": "2017-09-25T00:26:15Z",
							"finishedAt": "2017-09-25T00:31:04Z",
							"containerID": "docker://b03938c75e91f55be4d534fbf68b5b652139c17a3bcb1b2476fb5e48c7908f76"
						}
					},
					"lastState": {},
					"ready": false,
					"restartCount": 0,
					"image": "gcr.io/google_containers/heapster:v1.3.0",
					"imageID": "docker-pullable://gcr.io/google_containers/heapster@sha256:3dff9b2425a196aa51df0cebde0f8b427388425ba84568721acf416fa003cd5c",
					"containerID": "docker://b03938c75e91f55be4d534fbf68b5b652139c17a3bcb1b2476fb5e48c7908f76"
				}],
				"qosClass": "Burstable"
			}
		}, {
			"metadata": {
				"name": "hollow-node-8bj52",
				"generateName": "hollow-node-",
				"namespace": "kubemark",
				"selfLink": "/api/v1/namespaces/kubemark/pods/hollow-node-8bj52",
				"uid": "210232d7-a188-11e7-bb13-42010a800002",
				"resourceVersion": "1215",
				"creationTimestamp": "2017-09-25T00:26:09Z",
				"deletionTimestamp": "2017-09-25T00:31:34Z",
				"deletionGracePeriodSeconds": 30,
				"labels": {
					"name": "hollow-node"
				},
				"annotations": {
					"kubernetes.io/created-by": "{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicationController\",\"namespace\":\"kubemark\",\"name\":\"hollow-node\",\"uid\":\"20f362b5-a188-11e7-bb13-42010a800002\",\"apiVersion\":\"v1\",\"resourceVersion\":\"813\"}}\n"
				},
				"ownerReferences": [{
					"apiVersion": "v1",
					"kind": "ReplicationController",
					"name": "hollow-node",
					"uid": "20f362b5-a188-11e7-bb13-42010a800002",
					"controller": true,
					"blockOwnerDeletion": true
				}]
			},
			"spec": {
				"volumes": [{
					"name": "kubeconfig-volume",
					"secret": {
						"secretName": "kubeconfig",
						"defaultMode": 420
					}
				}, {
					"name": "kernelmonitorconfig-volume",
					"configMap": {
						"name": "node-configmap",
						"defaultMode": 420
					}
				}, {
					"name": "logs-volume",
					"hostPath": {
						"path": "/var/log",
						"type": ""
					}
				}, {
					"name": "no-serviceaccount-access-to-real-master",
					"emptyDir": {}
				}, {
					"name": "default-token-6552z",
					"secret": {
						"secretName": "default-token-6552z",
						"defaultMode": 420
					}
				}],
				"initContainers": [{
					"name": "init-inotify-limit",
					"image": "busybox",
					"command": ["sysctl", "-w", "fs.inotify.max_user_instances=200"],
					"resources": {},
					"volumeMounts": [{
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always",
					"securityContext": {
						"privileged": true
					}
				}],
				"containers": [{
					"name": "hollow-kubelet",
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"command": ["/bin/sh", "-c", "/kubemark --morph=kubelet --name=$(NODE_NAME) --kubeconfig=/kubeconfig/kubelet.kubeconfig $(CONTENT_TYPE) --alsologtostderr --v=2 1\u003e\u003e/var/log/kubelet-$(NODE_NAME).log 2\u003e\u00261"],
					"ports": [{
						"containerPort": 4194,
						"protocol": "TCP"
					}, {
						"containerPort": 10250,
						"protocol": "TCP"
					}, {
						"containerPort": 10255,
						"protocol": "TCP"
					}],
					"env": [{
						"name": "CONTENT_TYPE",
						"valueFrom": {
							"configMapKeyRef": {
								"name": "node-configmap",
								"key": "content.type"
							}
						}
					}, {
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "40m",
							"memory": "100M"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always",
					"securityContext": {
						"privileged": true
					}
				}, {
					"name": "hollow-proxy",
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"command": ["/bin/sh", "-c", "/kubemark --morph=proxy --name=$(NODE_NAME) --use-real-proxier=true --kubeconfig=/kubeconfig/kubeproxy.kubeconfig $(CONTENT_TYPE) --alsologtostderr --v=2 1\u003e\u003e/var/log/kubeproxy-$(NODE_NAME).log 2\u003e\u00261"],
					"env": [{
						"name": "CONTENT_TYPE",
						"valueFrom": {
							"configMapKeyRef": {
								"name": "node-configmap",
								"key": "content.type"
							}
						}
					}, {
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "20m",
							"memory": "102650Ki"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always"
				}, {
					"name": "hollow-node-problem-detector",
					"image": "gcr.io/google_containers/node-problem-detector:v0.4.1",
					"command": ["/bin/sh", "-c", "/node-problem-detector --system-log-monitors=/config/kernel.monitor --apiserver-override=\"https://35.192.172.191:443?inClusterConfig=false\u0026auth=/kubeconfig/npd.kubeconfig\" --alsologtostderr 1\u003e\u003e/var/log/npd-$(NODE_NAME).log 2\u003e\u00261"],
					"env": [{
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "20m",
							"memory": "20Mi"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "kernelmonitorconfig-volume",
						"readOnly": true,
						"mountPath": "/config"
					}, {
						"name": "no-serviceaccount-access-to-real-master",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "IfNotPresent",
					"securityContext": {
						"privileged": true
					}
				}],
				"restartPolicy": "Always",
				"terminationGracePeriodSeconds": 30,
				"dnsPolicy": "ClusterFirst",
				"serviceAccountName": "default",
				"serviceAccount": "default",
				"nodeName": "e2e-53532-minion-group-q6xr",
				"securityContext": {},
				"schedulerName": "default-scheduler",
				"tolerations": [{
					"key": "node.alpha.kubernetes.io/notReady",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}, {
					"key": "node.alpha.kubernetes.io/unreachable",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}]
			},
			"status": {
				"phase": "Running",
				"conditions": [{
					"type": "Initialized",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:17Z"
				}, {
					"type": "Ready",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:42Z"
				}, {
					"type": "PodScheduled",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:10Z"
				}],
				"hostIP": "10.128.0.3",
				"podIP": "10.64.1.18",
				"startTime": "2017-09-25T00:26:10Z",
				"initContainerStatuses": [{
					"name": "init-inotify-limit",
					"state": {
						"terminated": {
							"exitCode": 0,
							"reason": "Completed",
							"startedAt": "2017-09-25T00:26:16Z",
							"finishedAt": "2017-09-25T00:26:16Z",
							"containerID": "docker://99804733c499122f5ed354447e4e5162261bd4b696afad7358ff6f147727ed54"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "busybox:latest",
					"imageID": "docker-pullable://busybox@sha256:030fcb92e1487b18c974784dcc110a93147c9fc402188370fbfd17efabffc6af",
					"containerID": "docker://99804733c499122f5ed354447e4e5162261bd4b696afad7358ff6f147727ed54"
				}],
				"containerStatuses": [{
					"name": "hollow-kubelet",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:25Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"imageID": "docker-pullable://gcr.io/k8s-jkns-pr-kubemark/kubemark@sha256:8f87f47834bcb9dfcbe58a2769708ce082dfd768c449f51caedf564d7e37fb92",
					"containerID": "docker://89ac79222821a25bcd9c8107fb271954583e87b95d0e3f481b526445ca5be464"
				}, {
					"name": "hollow-node-problem-detector",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:42Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/google_containers/node-problem-detector:v0.4.1",
					"imageID": "docker-pullable://gcr.io/google_containers/node-problem-detector@sha256:f95cab985c26b2f46e9bd43283e0bfa88860c14e0fb0649266babe8b65e9eb2b",
					"containerID": "docker://afd87ed0ee06700721dbec7de25a436bc6d0589b09a7112fec30d37c1e5aef35"
				}, {
					"name": "hollow-proxy",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:27Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"imageID": "docker-pullable://gcr.io/k8s-jkns-pr-kubemark/kubemark@sha256:8f87f47834bcb9dfcbe58a2769708ce082dfd768c449f51caedf564d7e37fb92",
					"containerID": "docker://22c4d787ca3cbe16008666fc57fc8b32c0f03ebc55bcbd6f7e7008ac7fb33518"
				}],
				"qosClass": "Burstable"
			}
		}, {
			"metadata": {
				"name": "hollow-node-bbsgc",
				"generateName": "hollow-node-",
				"namespace": "kubemark",
				"selfLink": "/api/v1/namespaces/kubemark/pods/hollow-node-bbsgc",
				"uid": "20f72909-a188-11e7-bb13-42010a800002",
				"resourceVersion": "1214",
				"creationTimestamp": "2017-09-25T00:26:09Z",
				"deletionTimestamp": "2017-09-25T00:31:34Z",
				"deletionGracePeriodSeconds": 30,
				"labels": {
					"name": "hollow-node"
				},
				"annotations": {
					"kubernetes.io/created-by": "{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicationController\",\"namespace\":\"kubemark\",\"name\":\"hollow-node\",\"uid\":\"20f362b5-a188-11e7-bb13-42010a800002\",\"apiVersion\":\"v1\",\"resourceVersion\":\"813\"}}\n"
				},
				"ownerReferences": [{
					"apiVersion": "v1",
					"kind": "ReplicationController",
					"name": "hollow-node",
					"uid": "20f362b5-a188-11e7-bb13-42010a800002",
					"controller": true,
					"blockOwnerDeletion": true
				}]
			},
			"spec": {
				"volumes": [{
					"name": "kubeconfig-volume",
					"secret": {
						"secretName": "kubeconfig",
						"defaultMode": 420
					}
				}, {
					"name": "kernelmonitorconfig-volume",
					"configMap": {
						"name": "node-configmap",
						"defaultMode": 420
					}
				}, {
					"name": "logs-volume",
					"hostPath": {
						"path": "/var/log",
						"type": ""
					}
				}, {
					"name": "no-serviceaccount-access-to-real-master",
					"emptyDir": {}
				}, {
					"name": "default-token-6552z",
					"secret": {
						"secretName": "default-token-6552z",
						"defaultMode": 420
					}
				}],
				"initContainers": [{
					"name": "init-inotify-limit",
					"image": "busybox",
					"command": ["sysctl", "-w", "fs.inotify.max_user_instances=200"],
					"resources": {},
					"volumeMounts": [{
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always",
					"securityContext": {
						"privileged": true
					}
				}],
				"containers": [{
					"name": "hollow-kubelet",
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"command": ["/bin/sh", "-c", "/kubemark --morph=kubelet --name=$(NODE_NAME) --kubeconfig=/kubeconfig/kubelet.kubeconfig $(CONTENT_TYPE) --alsologtostderr --v=2 1\u003e\u003e/var/log/kubelet-$(NODE_NAME).log 2\u003e\u00261"],
					"ports": [{
						"containerPort": 4194,
						"protocol": "TCP"
					}, {
						"containerPort": 10250,
						"protocol": "TCP"
					}, {
						"containerPort": 10255,
						"protocol": "TCP"
					}],
					"env": [{
						"name": "CONTENT_TYPE",
						"valueFrom": {
							"configMapKeyRef": {
								"name": "node-configmap",
								"key": "content.type"
							}
						}
					}, {
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "40m",
							"memory": "100M"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always",
					"securityContext": {
						"privileged": true
					}
				}, {
					"name": "hollow-proxy",
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"command": ["/bin/sh", "-c", "/kubemark --morph=proxy --name=$(NODE_NAME) --use-real-proxier=true --kubeconfig=/kubeconfig/kubeproxy.kubeconfig $(CONTENT_TYPE) --alsologtostderr --v=2 1\u003e\u003e/var/log/kubeproxy-$(NODE_NAME).log 2\u003e\u00261"],
					"env": [{
						"name": "CONTENT_TYPE",
						"valueFrom": {
							"configMapKeyRef": {
								"name": "node-configmap",
								"key": "content.type"
							}
						}
					}, {
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "20m",
							"memory": "102650Ki"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always"
				}, {
					"name": "hollow-node-problem-detector",
					"image": "gcr.io/google_containers/node-problem-detector:v0.4.1",
					"command": ["/bin/sh", "-c", "/node-problem-detector --system-log-monitors=/config/kernel.monitor --apiserver-override=\"https://35.192.172.191:443?inClusterConfig=false\u0026auth=/kubeconfig/npd.kubeconfig\" --alsologtostderr 1\u003e\u003e/var/log/npd-$(NODE_NAME).log 2\u003e\u00261"],
					"env": [{
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "20m",
							"memory": "20Mi"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "kernelmonitorconfig-volume",
						"readOnly": true,
						"mountPath": "/config"
					}, {
						"name": "no-serviceaccount-access-to-real-master",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "IfNotPresent",
					"securityContext": {
						"privileged": true
					}
				}],
				"restartPolicy": "Always",
				"terminationGracePeriodSeconds": 30,
				"dnsPolicy": "ClusterFirst",
				"serviceAccountName": "default",
				"serviceAccount": "default",
				"nodeName": "e2e-53532-minion-group-q6xr",
				"securityContext": {},
				"schedulerName": "default-scheduler",
				"tolerations": [{
					"key": "node.alpha.kubernetes.io/notReady",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}, {
					"key": "node.alpha.kubernetes.io/unreachable",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}]
			},
			"status": {
				"phase": "Running",
				"conditions": [{
					"type": "Initialized",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:17Z"
				}, {
					"type": "Ready",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:42Z"
				}, {
					"type": "PodScheduled",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:09Z"
				}],
				"hostIP": "10.128.0.3",
				"podIP": "10.64.1.16",
				"startTime": "2017-09-25T00:26:09Z",
				"initContainerStatuses": [{
					"name": "init-inotify-limit",
					"state": {
						"terminated": {
							"exitCode": 0,
							"reason": "Completed",
							"startedAt": "2017-09-25T00:26:15Z",
							"finishedAt": "2017-09-25T00:26:16Z",
							"containerID": "docker://e0981e0bf2b48a077633b0f94fdeea67ea522500251d0541c24d8c48e702fb2f"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "busybox:latest",
					"imageID": "docker-pullable://busybox@sha256:030fcb92e1487b18c974784dcc110a93147c9fc402188370fbfd17efabffc6af",
					"containerID": "docker://e0981e0bf2b48a077633b0f94fdeea67ea522500251d0541c24d8c48e702fb2f"
				}],
				"containerStatuses": [{
					"name": "hollow-kubelet",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:26Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"imageID": "docker-pullable://gcr.io/k8s-jkns-pr-kubemark/kubemark@sha256:8f87f47834bcb9dfcbe58a2769708ce082dfd768c449f51caedf564d7e37fb92",
					"containerID": "docker://419d46dffc8126127b57b824cccb910174f0b4b5c1351a31298439ad64835427"
				}, {
					"name": "hollow-node-problem-detector",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:42Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/google_containers/node-problem-detector:v0.4.1",
					"imageID": "docker-pullable://gcr.io/google_containers/node-problem-detector@sha256:f95cab985c26b2f46e9bd43283e0bfa88860c14e0fb0649266babe8b65e9eb2b",
					"containerID": "docker://bbb862811d05c32bcfad5fb74b2f85569a297305e842af06489706bf62fdb451"
				}, {
					"name": "hollow-proxy",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:27Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"imageID": "docker-pullable://gcr.io/k8s-jkns-pr-kubemark/kubemark@sha256:8f87f47834bcb9dfcbe58a2769708ce082dfd768c449f51caedf564d7e37fb92",
					"containerID": "docker://c7b0241a994efd295e5fc9cfc54dc267a4f3f927ae349a10dc0d05153f5dab62"
				}],
				"qosClass": "Burstable"
			}
		}, {
			"metadata": {
				"name": "hollow-node-bhsnj",
				"generateName": "hollow-node-",
				"namespace": "kubemark",
				"selfLink": "/api/v1/namespaces/kubemark/pods/hollow-node-bhsnj",
				"uid": "21088c4a-a188-11e7-bb13-42010a800002",
				"resourceVersion": "1216",
				"creationTimestamp": "2017-09-25T00:26:09Z",
				"deletionTimestamp": "2017-09-25T00:31:34Z",
				"deletionGracePeriodSeconds": 30,
				"labels": {
					"name": "hollow-node"
				},
				"annotations": {
					"kubernetes.io/created-by": "{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicationController\",\"namespace\":\"kubemark\",\"name\":\"hollow-node\",\"uid\":\"20f362b5-a188-11e7-bb13-42010a800002\",\"apiVersion\":\"v1\",\"resourceVersion\":\"813\"}}\n"
				},
				"ownerReferences": [{
					"apiVersion": "v1",
					"kind": "ReplicationController",
					"name": "hollow-node",
					"uid": "20f362b5-a188-11e7-bb13-42010a800002",
					"controller": true,
					"blockOwnerDeletion": true
				}]
			},
			"spec": {
				"volumes": [{
					"name": "kubeconfig-volume",
					"secret": {
						"secretName": "kubeconfig",
						"defaultMode": 420
					}
				}, {
					"name": "kernelmonitorconfig-volume",
					"configMap": {
						"name": "node-configmap",
						"defaultMode": 420
					}
				}, {
					"name": "logs-volume",
					"hostPath": {
						"path": "/var/log",
						"type": ""
					}
				}, {
					"name": "no-serviceaccount-access-to-real-master",
					"emptyDir": {}
				}, {
					"name": "default-token-6552z",
					"secret": {
						"secretName": "default-token-6552z",
						"defaultMode": 420
					}
				}],
				"initContainers": [{
					"name": "init-inotify-limit",
					"image": "busybox",
					"command": ["sysctl", "-w", "fs.inotify.max_user_instances=200"],
					"resources": {},
					"volumeMounts": [{
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always",
					"securityContext": {
						"privileged": true
					}
				}],
				"containers": [{
					"name": "hollow-kubelet",
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"command": ["/bin/sh", "-c", "/kubemark --morph=kubelet --name=$(NODE_NAME) --kubeconfig=/kubeconfig/kubelet.kubeconfig $(CONTENT_TYPE) --alsologtostderr --v=2 1\u003e\u003e/var/log/kubelet-$(NODE_NAME).log 2\u003e\u00261"],
					"ports": [{
						"containerPort": 4194,
						"protocol": "TCP"
					}, {
						"containerPort": 10250,
						"protocol": "TCP"
					}, {
						"containerPort": 10255,
						"protocol": "TCP"
					}],
					"env": [{
						"name": "CONTENT_TYPE",
						"valueFrom": {
							"configMapKeyRef": {
								"name": "node-configmap",
								"key": "content.type"
							}
						}
					}, {
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "40m",
							"memory": "100M"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always",
					"securityContext": {
						"privileged": true
					}
				}, {
					"name": "hollow-proxy",
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"command": ["/bin/sh", "-c", "/kubemark --morph=proxy --name=$(NODE_NAME) --use-real-proxier=true --kubeconfig=/kubeconfig/kubeproxy.kubeconfig $(CONTENT_TYPE) --alsologtostderr --v=2 1\u003e\u003e/var/log/kubeproxy-$(NODE_NAME).log 2\u003e\u00261"],
					"env": [{
						"name": "CONTENT_TYPE",
						"valueFrom": {
							"configMapKeyRef": {
								"name": "node-configmap",
								"key": "content.type"
							}
						}
					}, {
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "20m",
							"memory": "102650Ki"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always"
				}, {
					"name": "hollow-node-problem-detector",
					"image": "gcr.io/google_containers/node-problem-detector:v0.4.1",
					"command": ["/bin/sh", "-c", "/node-problem-detector --system-log-monitors=/config/kernel.monitor --apiserver-override=\"https://35.192.172.191:443?inClusterConfig=false\u0026auth=/kubeconfig/npd.kubeconfig\" --alsologtostderr 1\u003e\u003e/var/log/npd-$(NODE_NAME).log 2\u003e\u00261"],
					"env": [{
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "20m",
							"memory": "20Mi"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "kernelmonitorconfig-volume",
						"readOnly": true,
						"mountPath": "/config"
					}, {
						"name": "no-serviceaccount-access-to-real-master",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "IfNotPresent",
					"securityContext": {
						"privileged": true
					}
				}],
				"restartPolicy": "Always",
				"terminationGracePeriodSeconds": 30,
				"dnsPolicy": "ClusterFirst",
				"serviceAccountName": "default",
				"serviceAccount": "default",
				"nodeName": "e2e-53532-minion-group-q6xr",
				"securityContext": {},
				"schedulerName": "default-scheduler",
				"tolerations": [{
					"key": "node.alpha.kubernetes.io/notReady",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}, {
					"key": "node.alpha.kubernetes.io/unreachable",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}]
			},
			"status": {
				"phase": "Running",
				"conditions": [{
					"type": "Initialized",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:16Z"
				}, {
					"type": "Ready",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:42Z"
				}, {
					"type": "PodScheduled",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:10Z"
				}],
				"hostIP": "10.128.0.3",
				"podIP": "10.64.1.15",
				"startTime": "2017-09-25T00:26:10Z",
				"initContainerStatuses": [{
					"name": "init-inotify-limit",
					"state": {
						"terminated": {
							"exitCode": 0,
							"reason": "Completed",
							"startedAt": "2017-09-25T00:26:15Z",
							"finishedAt": "2017-09-25T00:26:15Z",
							"containerID": "docker://785b830554af6f15af183b6ed388e4241137e25ced10274e9255a5548dc3f569"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "busybox:latest",
					"imageID": "docker-pullable://busybox@sha256:030fcb92e1487b18c974784dcc110a93147c9fc402188370fbfd17efabffc6af",
					"containerID": "docker://785b830554af6f15af183b6ed388e4241137e25ced10274e9255a5548dc3f569"
				}],
				"containerStatuses": [{
					"name": "hollow-kubelet",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:26Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"imageID": "docker-pullable://gcr.io/k8s-jkns-pr-kubemark/kubemark@sha256:8f87f47834bcb9dfcbe58a2769708ce082dfd768c449f51caedf564d7e37fb92",
					"containerID": "docker://e98b741031e9c358394bb39ee211fb07d657744931ad407fdb1f9f8f85c9ef18"
				}, {
					"name": "hollow-node-problem-detector",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:42Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/google_containers/node-problem-detector:v0.4.1",
					"imageID": "docker-pullable://gcr.io/google_containers/node-problem-detector@sha256:f95cab985c26b2f46e9bd43283e0bfa88860c14e0fb0649266babe8b65e9eb2b",
					"containerID": "docker://ee6dfa70992ef10c33fb8d765b7c6314f69d62bad6fab252e6de78f59d1c2d43"
				}, {
					"name": "hollow-proxy",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:27Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"imageID": "docker-pullable://gcr.io/k8s-jkns-pr-kubemark/kubemark@sha256:8f87f47834bcb9dfcbe58a2769708ce082dfd768c449f51caedf564d7e37fb92",
					"containerID": "docker://f62130542dab7e1c4692eb1450c0ba61dda287f8e52e39250aafdba1a51a491a"
				}],
				"qosClass": "Burstable"
			}
		}, {
			"metadata": {
				"name": "hollow-node-stlpj",
				"generateName": "hollow-node-",
				"namespace": "kubemark",
				"selfLink": "/api/v1/namespaces/kubemark/pods/hollow-node-stlpj",
				"uid": "20ffdad7-a188-11e7-bb13-42010a800002",
				"resourceVersion": "1213",
				"creationTimestamp": "2017-09-25T00:26:09Z",
				"deletionTimestamp": "2017-09-25T00:31:34Z",
				"deletionGracePeriodSeconds": 30,
				"labels": {
					"name": "hollow-node"
				},
				"annotations": {
					"kubernetes.io/created-by": "{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicationController\",\"namespace\":\"kubemark\",\"name\":\"hollow-node\",\"uid\":\"20f362b5-a188-11e7-bb13-42010a800002\",\"apiVersion\":\"v1\",\"resourceVersion\":\"813\"}}\n"
				},
				"ownerReferences": [{
					"apiVersion": "v1",
					"kind": "ReplicationController",
					"name": "hollow-node",
					"uid": "20f362b5-a188-11e7-bb13-42010a800002",
					"controller": true,
					"blockOwnerDeletion": true
				}]
			},
			"spec": {
				"volumes": [{
					"name": "kubeconfig-volume",
					"secret": {
						"secretName": "kubeconfig",
						"defaultMode": 420
					}
				}, {
					"name": "kernelmonitorconfig-volume",
					"configMap": {
						"name": "node-configmap",
						"defaultMode": 420
					}
				}, {
					"name": "logs-volume",
					"hostPath": {
						"path": "/var/log",
						"type": ""
					}
				}, {
					"name": "no-serviceaccount-access-to-real-master",
					"emptyDir": {}
				}, {
					"name": "default-token-6552z",
					"secret": {
						"secretName": "default-token-6552z",
						"defaultMode": 420
					}
				}],
				"initContainers": [{
					"name": "init-inotify-limit",
					"image": "busybox",
					"command": ["sysctl", "-w", "fs.inotify.max_user_instances=200"],
					"resources": {},
					"volumeMounts": [{
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always",
					"securityContext": {
						"privileged": true
					}
				}],
				"containers": [{
					"name": "hollow-kubelet",
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"command": ["/bin/sh", "-c", "/kubemark --morph=kubelet --name=$(NODE_NAME) --kubeconfig=/kubeconfig/kubelet.kubeconfig $(CONTENT_TYPE) --alsologtostderr --v=2 1\u003e\u003e/var/log/kubelet-$(NODE_NAME).log 2\u003e\u00261"],
					"ports": [{
						"containerPort": 4194,
						"protocol": "TCP"
					}, {
						"containerPort": 10250,
						"protocol": "TCP"
					}, {
						"containerPort": 10255,
						"protocol": "TCP"
					}],
					"env": [{
						"name": "CONTENT_TYPE",
						"valueFrom": {
							"configMapKeyRef": {
								"name": "node-configmap",
								"key": "content.type"
							}
						}
					}, {
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "40m",
							"memory": "100M"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always",
					"securityContext": {
						"privileged": true
					}
				}, {
					"name": "hollow-proxy",
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"command": ["/bin/sh", "-c", "/kubemark --morph=proxy --name=$(NODE_NAME) --use-real-proxier=true --kubeconfig=/kubeconfig/kubeproxy.kubeconfig $(CONTENT_TYPE) --alsologtostderr --v=2 1\u003e\u003e/var/log/kubeproxy-$(NODE_NAME).log 2\u003e\u00261"],
					"env": [{
						"name": "CONTENT_TYPE",
						"valueFrom": {
							"configMapKeyRef": {
								"name": "node-configmap",
								"key": "content.type"
							}
						}
					}, {
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "20m",
							"memory": "102650Ki"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always"
				}, {
					"name": "hollow-node-problem-detector",
					"image": "gcr.io/google_containers/node-problem-detector:v0.4.1",
					"command": ["/bin/sh", "-c", "/node-problem-detector --system-log-monitors=/config/kernel.monitor --apiserver-override=\"https://35.192.172.191:443?inClusterConfig=false\u0026auth=/kubeconfig/npd.kubeconfig\" --alsologtostderr 1\u003e\u003e/var/log/npd-$(NODE_NAME).log 2\u003e\u00261"],
					"env": [{
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "20m",
							"memory": "20Mi"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "kernelmonitorconfig-volume",
						"readOnly": true,
						"mountPath": "/config"
					}, {
						"name": "no-serviceaccount-access-to-real-master",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "IfNotPresent",
					"securityContext": {
						"privileged": true
					}
				}],
				"restartPolicy": "Always",
				"terminationGracePeriodSeconds": 30,
				"dnsPolicy": "ClusterFirst",
				"serviceAccountName": "default",
				"serviceAccount": "default",
				"nodeName": "e2e-53532-minion-group-q6xr",
				"securityContext": {},
				"schedulerName": "default-scheduler",
				"tolerations": [{
					"key": "node.alpha.kubernetes.io/notReady",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}, {
					"key": "node.alpha.kubernetes.io/unreachable",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}]
			},
			"status": {
				"phase": "Running",
				"conditions": [{
					"type": "Initialized",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:17Z"
				}, {
					"type": "Ready",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:42Z"
				}, {
					"type": "PodScheduled",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:09Z"
				}],
				"hostIP": "10.128.0.3",
				"podIP": "10.64.1.17",
				"startTime": "2017-09-25T00:26:10Z",
				"initContainerStatuses": [{
					"name": "init-inotify-limit",
					"state": {
						"terminated": {
							"exitCode": 0,
							"reason": "Completed",
							"startedAt": "2017-09-25T00:26:16Z",
							"finishedAt": "2017-09-25T00:26:16Z",
							"containerID": "docker://5b1bcb5bfe5f4f61158889eefb9d91233c9f1f491954d65b6d712504d441e759"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "busybox:latest",
					"imageID": "docker-pullable://busybox@sha256:030fcb92e1487b18c974784dcc110a93147c9fc402188370fbfd17efabffc6af",
					"containerID": "docker://5b1bcb5bfe5f4f61158889eefb9d91233c9f1f491954d65b6d712504d441e759"
				}],
				"containerStatuses": [{
					"name": "hollow-kubelet",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:25Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"imageID": "docker-pullable://gcr.io/k8s-jkns-pr-kubemark/kubemark@sha256:8f87f47834bcb9dfcbe58a2769708ce082dfd768c449f51caedf564d7e37fb92",
					"containerID": "docker://70a214b2ed897c4f27ea82a34ee93f466f6b2e856173d74d49b6f4eaa2fea21b"
				}, {
					"name": "hollow-node-problem-detector",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:42Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/google_containers/node-problem-detector:v0.4.1",
					"imageID": "docker-pullable://gcr.io/google_containers/node-problem-detector@sha256:f95cab985c26b2f46e9bd43283e0bfa88860c14e0fb0649266babe8b65e9eb2b",
					"containerID": "docker://53347050891dab25f564103f76e1e30426ec94f6ae5872104b0d81dca636527d"
				}, {
					"name": "hollow-proxy",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:27Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"imageID": "docker-pullable://gcr.io/k8s-jkns-pr-kubemark/kubemark@sha256:8f87f47834bcb9dfcbe58a2769708ce082dfd768c449f51caedf564d7e37fb92",
					"containerID": "docker://d27815a224e8c57a7cc25c0a6ff4605b513e87aa65072569303a2b25b94d3d2f"
				}],
				"qosClass": "Burstable"
			}
		}, {
			"metadata": {
				"name": "hollow-node-wv9q6",
				"generateName": "hollow-node-",
				"namespace": "kubemark",
				"selfLink": "/api/v1/namespaces/kubemark/pods/hollow-node-wv9q6",
				"uid": "2107f419-a188-11e7-bb13-42010a800002",
				"resourceVersion": "1217",
				"creationTimestamp": "2017-09-25T00:26:09Z",
				"deletionTimestamp": "2017-09-25T00:31:34Z",
				"deletionGracePeriodSeconds": 30,
				"labels": {
					"name": "hollow-node"
				},
				"annotations": {
					"kubernetes.io/created-by": "{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicationController\",\"namespace\":\"kubemark\",\"name\":\"hollow-node\",\"uid\":\"20f362b5-a188-11e7-bb13-42010a800002\",\"apiVersion\":\"v1\",\"resourceVersion\":\"813\"}}\n"
				},
				"ownerReferences": [{
					"apiVersion": "v1",
					"kind": "ReplicationController",
					"name": "hollow-node",
					"uid": "20f362b5-a188-11e7-bb13-42010a800002",
					"controller": true,
					"blockOwnerDeletion": true
				}]
			},
			"spec": {
				"volumes": [{
					"name": "kubeconfig-volume",
					"secret": {
						"secretName": "kubeconfig",
						"defaultMode": 420
					}
				}, {
					"name": "kernelmonitorconfig-volume",
					"configMap": {
						"name": "node-configmap",
						"defaultMode": 420
					}
				}, {
					"name": "logs-volume",
					"hostPath": {
						"path": "/var/log",
						"type": ""
					}
				}, {
					"name": "no-serviceaccount-access-to-real-master",
					"emptyDir": {}
				}, {
					"name": "default-token-6552z",
					"secret": {
						"secretName": "default-token-6552z",
						"defaultMode": 420
					}
				}],
				"initContainers": [{
					"name": "init-inotify-limit",
					"image": "busybox",
					"command": ["sysctl", "-w", "fs.inotify.max_user_instances=200"],
					"resources": {},
					"volumeMounts": [{
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always",
					"securityContext": {
						"privileged": true
					}
				}],
				"containers": [{
					"name": "hollow-kubelet",
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"command": ["/bin/sh", "-c", "/kubemark --morph=kubelet --name=$(NODE_NAME) --kubeconfig=/kubeconfig/kubelet.kubeconfig $(CONTENT_TYPE) --alsologtostderr --v=2 1\u003e\u003e/var/log/kubelet-$(NODE_NAME).log 2\u003e\u00261"],
					"ports": [{
						"containerPort": 4194,
						"protocol": "TCP"
					}, {
						"containerPort": 10250,
						"protocol": "TCP"
					}, {
						"containerPort": 10255,
						"protocol": "TCP"
					}],
					"env": [{
						"name": "CONTENT_TYPE",
						"valueFrom": {
							"configMapKeyRef": {
								"name": "node-configmap",
								"key": "content.type"
							}
						}
					}, {
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "40m",
							"memory": "100M"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always",
					"securityContext": {
						"privileged": true
					}
				}, {
					"name": "hollow-proxy",
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"command": ["/bin/sh", "-c", "/kubemark --morph=proxy --name=$(NODE_NAME) --use-real-proxier=true --kubeconfig=/kubeconfig/kubeproxy.kubeconfig $(CONTENT_TYPE) --alsologtostderr --v=2 1\u003e\u003e/var/log/kubeproxy-$(NODE_NAME).log 2\u003e\u00261"],
					"env": [{
						"name": "CONTENT_TYPE",
						"valueFrom": {
							"configMapKeyRef": {
								"name": "node-configmap",
								"key": "content.type"
							}
						}
					}, {
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "20m",
							"memory": "102650Ki"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}, {
						"name": "default-token-6552z",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "Always"
				}, {
					"name": "hollow-node-problem-detector",
					"image": "gcr.io/google_containers/node-problem-detector:v0.4.1",
					"command": ["/bin/sh", "-c", "/node-problem-detector --system-log-monitors=/config/kernel.monitor --apiserver-override=\"https://35.192.172.191:443?inClusterConfig=false\u0026auth=/kubeconfig/npd.kubeconfig\" --alsologtostderr 1\u003e\u003e/var/log/npd-$(NODE_NAME).log 2\u003e\u00261"],
					"env": [{
						"name": "NODE_NAME",
						"valueFrom": {
							"fieldRef": {
								"apiVersion": "v1",
								"fieldPath": "metadata.name"
							}
						}
					}],
					"resources": {
						"requests": {
							"cpu": "20m",
							"memory": "20Mi"
						}
					},
					"volumeMounts": [{
						"name": "kubeconfig-volume",
						"readOnly": true,
						"mountPath": "/kubeconfig"
					}, {
						"name": "kernelmonitorconfig-volume",
						"readOnly": true,
						"mountPath": "/config"
					}, {
						"name": "no-serviceaccount-access-to-real-master",
						"readOnly": true,
						"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
					}, {
						"name": "logs-volume",
						"mountPath": "/var/log"
					}],
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"imagePullPolicy": "IfNotPresent",
					"securityContext": {
						"privileged": true
					}
				}],
				"restartPolicy": "Always",
				"terminationGracePeriodSeconds": 30,
				"dnsPolicy": "ClusterFirst",
				"serviceAccountName": "default",
				"serviceAccount": "default",
				"nodeName": "e2e-53532-minion-group-q6xr",
				"securityContext": {},
				"schedulerName": "default-scheduler",
				"tolerations": [{
					"key": "node.alpha.kubernetes.io/notReady",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}, {
					"key": "node.alpha.kubernetes.io/unreachable",
					"operator": "Exists",
					"effect": "NoExecute",
					"tolerationSeconds": 300
				}]
			},
			"status": {
				"phase": "Running",
				"conditions": [{
					"type": "Initialized",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:14Z"
				}, {
					"type": "Ready",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:42Z"
				}, {
					"type": "PodScheduled",
					"status": "True",
					"lastProbeTime": null,
					"lastTransitionTime": "2017-09-25T00:26:10Z"
				}],
				"hostIP": "10.128.0.3",
				"podIP": "10.64.1.14",
				"startTime": "2017-09-25T00:26:10Z",
				"initContainerStatuses": [{
					"name": "init-inotify-limit",
					"state": {
						"terminated": {
							"exitCode": 0,
							"reason": "Completed",
							"startedAt": "2017-09-25T00:26:12Z",
							"finishedAt": "2017-09-25T00:26:12Z",
							"containerID": "docker://cefcc7121dd70e0fe3cda5cca69d632cdeafe413bee12f584a661876ceb4bb3c"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "busybox:latest",
					"imageID": "docker-pullable://busybox@sha256:030fcb92e1487b18c974784dcc110a93147c9fc402188370fbfd17efabffc6af",
					"containerID": "docker://cefcc7121dd70e0fe3cda5cca69d632cdeafe413bee12f584a661876ceb4bb3c"
				}],
				"containerStatuses": [{
					"name": "hollow-kubelet",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:25Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"imageID": "docker-pullable://gcr.io/k8s-jkns-pr-kubemark/kubemark@sha256:8f87f47834bcb9dfcbe58a2769708ce082dfd768c449f51caedf564d7e37fb92",
					"containerID": "docker://1f26e0161984ca664b8f3e9f3299a5350cc27880682b0e36c69ff9f00770cd44"
				}, {
					"name": "hollow-node-problem-detector",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:42Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/google_containers/node-problem-detector:v0.4.1",
					"imageID": "docker-pullable://gcr.io/google_containers/node-problem-detector@sha256:f95cab985c26b2f46e9bd43283e0bfa88860c14e0fb0649266babe8b65e9eb2b",
					"containerID": "docker://69aeabd5eba7f00d0730a265400a9f0fa6122e47b24a272f714a786974a95143"
				}, {
					"name": "hollow-proxy",
					"state": {
						"running": {
							"startedAt": "2017-09-25T00:26:27Z"
						}
					},
					"lastState": {},
					"ready": true,
					"restartCount": 0,
					"image": "gcr.io/k8s-jkns-pr-kubemark/kubemark:latest",
					"imageID": "docker-pullable://gcr.io/k8s-jkns-pr-kubemark/kubemark@sha256:8f87f47834bcb9dfcbe58a2769708ce082dfd768c449f51caedf564d7e37fb92",
					"containerID": "docker://1b6da3cfd6e5e33a1a4e9fa75f86d42289b470948a04ed0c21e185a7b047d612"
				}],
				"qosClass": "Burstable"
			}
		}]
	}
}`
)

func init() {
	allGVs := []schema.GroupVersion{auditv1beta1.SchemeGroupVersion}
	registry.RegisterVersions(allGVs)
	if err := registry.EnableVersions(allGVs...); err != nil {
		panic(fmt.Sprintf("failed to enable version %v", allGVs))
	}
	install.Install(groupFactoryRegistry, registry, audit.Scheme)
}

func TestLogEventsLegacy(t *testing.T) {
	for _, test := range []struct {
		event    *auditinternal.Event
		expected string
	}{
		{
			&auditinternal.Event{
				AuditID: types.UID(uuid.NewRandom().String()),
			},
			`[\d\:\-\.\+TZ]+ AUDIT: id="[\w-]+" stage="" ip="<unknown>" method="" user="<none>" groups="<none>" as="<self>" asgroups="<lookup>" namespace="<none>" uri="" response="<deferred>"`,
		},
		{
			&auditinternal.Event{
				ResponseStatus: &metav1.Status{
					Code: 200,
				},
				ResponseObject: &runtime.Unknown{
					TypeMeta:    runtime.TypeMeta{APIVersion: "", Kind: ""},
					Raw:         []byte(payload),
					ContentType: runtime.ContentTypeJSON,
				},
				RequestURI: "/apis/rbac.authorization.k8s.io/v1/roles",
				SourceIPs: []string{
					"127.0.0.1",
				},
				Timestamp: metav1.NewTime(time.Now()),
				AuditID:   types.UID(uuid.NewRandom().String()),
				Stage:     auditinternal.StageRequestReceived,
				Verb:      "get",
				User: auditinternal.UserInfo{
					Username: "admin",
					Groups: []string{
						"system:masters",
						"system:authenticated",
					},
				},
				ObjectRef: &auditinternal.ObjectReference{
					Namespace: "default",
				},
			},
			`[\d\:\-\.\+TZ]+ AUDIT: id="[\w-]+" stage="RequestReceived" ip="127.0.0.1" method="get" user="admin" groups="\\"system:masters\\",\\"system:authenticated\\"" as="<self>" asgroups="<lookup>" namespace="default" uri="/apis/rbac.authorization.k8s.io/v1/roles" response="200"`,
		},
		{
			&auditinternal.Event{
				AuditID: types.UID(uuid.NewRandom().String()),
				Level:   auditinternal.LevelMetadata,
				ObjectRef: &auditinternal.ObjectReference{
					Resource:    "foo",
					APIVersion:  "v1",
					Subresource: "bar",
				},
			},
			`[\d\:\-\.\+TZ]+ AUDIT: id="[\w-]+" stage="" ip="<unknown>" method="" user="<none>" groups="<none>" as="<self>" asgroups="<lookup>" namespace="<none>" uri="" response="<deferred>"`,
		},
	} {
		var buf bytes.Buffer
		backend := NewBackend(&buf, FormatLegacy, auditv1beta1.SchemeGroupVersion)
		backend.ProcessEvents(test.event)
		match, err := regexp.MatchString(test.expected, buf.String())
		if err != nil {
			t.Errorf("Unexpected error matching line %v", err)
			continue
		}
		if !match {
			t.Errorf("Unexpected line of audit: %s", buf.String())
		}
	}
}

func TestLogEventsJson(t *testing.T) {
	for _, event := range []*auditinternal.Event{
		{
			AuditID: types.UID(uuid.NewRandom().String()),
		},
		{
			ResponseStatus: &metav1.Status{
				Code: 200,
			},
			ResponseObject: &runtime.Unknown{
				TypeMeta:    runtime.TypeMeta{APIVersion: "", Kind: ""},
				Raw:         []byte(payload),
				ContentType: runtime.ContentTypeJSON,
			},
			RequestURI: "/apis/rbac.authorization.k8s.io/v1/roles",
			SourceIPs: []string{
				"127.0.0.1",
			},
			// When encoding to json format, the nanosecond part of timestamp is
			// lost and it will become zero when we decode event back, so we rounding
			// timestamp down to a multiple of second.
			Timestamp: metav1.NewTime(time.Now().Truncate(time.Second)),
			AuditID:   types.UID(uuid.NewRandom().String()),
			Stage:     auditinternal.StageRequestReceived,
			Verb:      "get",
			User: auditinternal.UserInfo{
				Username: "admin",
				Groups: []string{
					"system:masters",
					"system:authenticated",
				},
			},
			ObjectRef: &auditinternal.ObjectReference{
				Namespace: "default",
			},
		},
		{
			AuditID: types.UID(uuid.NewRandom().String()),
			Level:   auditinternal.LevelMetadata,
			ObjectRef: &auditinternal.ObjectReference{
				Resource:    "foo",
				APIVersion:  "v1",
				Subresource: "bar",
			},
		},
	} {
		var buf bytes.Buffer
		backend := NewBackend(&buf, FormatJson, auditv1beta1.SchemeGroupVersion)
		backend.ProcessEvents(event)
		// decode events back and compare with the original one.
		result := &auditinternal.Event{}
		decoder := audit.Codecs.UniversalDecoder(auditv1beta1.SchemeGroupVersion)
		if err := runtime.DecodeInto(decoder, buf.Bytes(), result); err != nil {
			t.Errorf("failed decoding buf: %s", buf.String())
			continue
		}
		if !reflect.DeepEqual(event, result) {
			t.Errorf("The result event should be the same with the original one, \noriginal: \n%#v\n result: \n%#v", event, result)
		}
	}
}
