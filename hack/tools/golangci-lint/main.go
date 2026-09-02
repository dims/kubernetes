/*
Copyright The Kubernetes Authors.

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

// Command golangci-lint is golangci-lint with the Kubernetes linters
// logcheck, kubeapilinter and sorted compiled in as module plugins.
//
// Go plugins (-buildmode=plugin) don't work with Go 1.27. The runtime adds
// the itabs of a plugin to its itab table a second time, and type switches
// in golangci-lint then fail. See https://github.com/golang/go/issues/48532.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/golangci/golangci-lint/v2/pkg/commands"
	"github.com/golangci/golangci-lint/v2/pkg/exitcodes"

	// These imports register the custom linters.
	_ "k8s.io/kubernetes/hack/tools/golangci-lint/sorted/plugin"
	_ "sigs.k8s.io/kube-api-linter"
	_ "sigs.k8s.io/logtools/logcheck/gclplugin"
)

func main() {
	if err := commands.Execute(commands.BuildInfo{GoVersion: runtime.Version()}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcodes.Failure)
	}
}
