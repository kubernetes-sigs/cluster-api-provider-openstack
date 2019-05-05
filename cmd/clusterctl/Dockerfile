# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build the cluster binary
FROM golang:1.10.3 as builder

# Copy in the go src
WORKDIR /go/src/sigs.k8s.io/cluster-api-provider-openstack
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o clusterctl sigs.k8s.io/cluster-api-provider-openstack/cmd/clusterctl

# Copy clusterctl into a thin image
FROM debian:stretch-slim
WORKDIR /

RUN apt-get update && apt-get install -y ca-certificates openssh-client

COPY --from=builder /go/src/sigs.k8s.io/cluster-api-provider-openstack/clusterctl .
ENTRYPOINT ["/clusterctl"]
