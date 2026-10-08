#!/usr/bin/env bash

# Copyright The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Builds hack/etcd-testserver into _output/local/bin so that unit tests do not
# build it themselves. The source hash must match sourceHash() in
# staging/src/k8s.io/apiserver/pkg/storage/etcd3/testserver/binary.go.

set -o errexit
set -o nounset
set -o pipefail

KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
source "${KUBE_ROOT}/hack/lib/init.sh"

kube::golang::setup_env

src="${KUBE_ROOT}/hack/etcd-testserver"
out="${KUBE_ROOT}/_output/local/bin/etcd-testserver$(go env GOEXE)"
hash=$( { cat "${src}/go.mod" "${src}/go.sum" "${src}/main.go"; printf '%s/%s\n%s\n' "$(go env GOOS)" "$(go env GOARCH)" "$(go env GOVERSION)"; } | sha256sum | cut -d' ' -f1)

if [[ -x "${out}" && "$(cat "${out}.srchash" 2>/dev/null)" == "${hash}" ]]; then
  exit 0
fi
mkdir -p "$(dirname "${out}")"
(cd "${src}" && GOWORK=off GOFLAGS=-mod=mod CGO_ENABLED=0 go build -o "${out}.tmp" .)
mv "${out}.tmp" "${out}"
echo -n "${hash}" > "${out}.srchash"
kube::log::status "built ${out}"
