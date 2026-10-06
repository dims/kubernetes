/*
Copyright 2019 The Kubernetes Authors.

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

package util

import (
	"crypto/x509"
	"fmt"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
	clientcertutil "k8s.io/client-go/util/cert"

	"k8s.io/kubernetes/cmd/kubeadm/app/util/errors"
	kubeconfigutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
)

// GetJoinWorkerCommand returns the kubeadm join command for a given token and
// Kubernetes cluster (the current cluster in the kubeconfig file)
func GetJoinWorkerCommand(kubeConfigFile, token string, skipTokenPrint bool) (string, error) {
	return getJoinCommand(kubeConfigFile, token, "", false, skipTokenPrint, false)
}

// GetJoinControlPlaneCommand returns the kubeadm join command for a given token and
// Kubernetes cluster (the current cluster in the kubeconfig file)
func GetJoinControlPlaneCommand(kubeConfigFile, token, key string, skipTokenPrint, skipCertificateKeyPrint bool) (string, error) {
	return getJoinCommand(kubeConfigFile, token, key, true, skipTokenPrint, skipCertificateKeyPrint)
}

func getJoinCommand(kubeConfigFile, token, key string, controlPlane, skipTokenPrint, skipCertificateKeyPrint bool) (string, error) {
	// load the kubeconfig file to get the CA certificate and endpoint
	config, err := clientcmd.LoadFromFile(kubeConfigFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to load kubeconfig")
	}

	// load the default cluster config
	_, clusterConfig, err := kubeconfigutil.GetClusterFromKubeConfig(config)
	if err != nil {
		return "", errors.Wrapf(err, "malformed kubeconfig file: %s", kubeConfigFile)
	}

	// load CA certificates from the kubeconfig (either from PEM data or by file path)
	var caCerts []*x509.Certificate
	if clusterConfig.CertificateAuthorityData != nil {
		caCerts, err = clientcertutil.ParseCertsPEM(clusterConfig.CertificateAuthorityData)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse CA certificate from kubeconfig")
		}
	} else if clusterConfig.CertificateAuthority != "" {
		caCerts, err = clientcertutil.CertsFromFile(clusterConfig.CertificateAuthority)
		if err != nil {
			return "", errors.Wrap(err, "failed to load CA certificate referenced by kubeconfig")
		}
	} else {
		return "", errors.New("no CA certificates found in kubeconfig")
	}

	// hash all the CA certs and include their public key pins as trusted values
	publicKeyPins := make([]string, 0, len(caCerts))
	for _, caCert := range caCerts {
		publicKeyPins = append(publicKeyPins, pubkeypin.Hash(caCert))
	}

	if skipTokenPrint {
		token = "<value withheld>"
	}
	if skipCertificateKeyPrint {
		key = "<value withheld>"
	}
	return joinCommand(strings.Replace(clusterConfig.Server, "https://", "", -1), token, publicKeyPins, controlPlane, key), nil
}

// joinCommand formats the kubeadm join command line.
func joinCommand(hostPort, token string, caCertHashes []string, controlPlane bool, certificateKey string) string {
	var out strings.Builder
	fmt.Fprintf(&out, "kubeadm join %s --token %s \\\n\t", hostPort, token)
	for _, hash := range caCertHashes {
		fmt.Fprintf(&out, "--discovery-token-ca-cert-hash %s ", hash)
	}
	if controlPlane {
		out.WriteString("\\\n\t--control-plane ")
		if certificateKey != "" {
			fmt.Fprintf(&out, "--certificate-key %s", certificateKey)
		}
	}
	return out.String()
}
