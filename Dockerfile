#
# Copyright 2022 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build the manager binary
FROM golang:1.19 as builder
ARG TARGETOS
ARG TARGETARCH

### Please remove comment-out if you encounter "Get "https://proxy.golang.org/cloud.google.com/go/@v/v0.99.0.mod": x509: certificate signed by unknown authority"
#   https://stackoverflow.com/questions/64462922/docker-multi-stage-build-go-image-x509-certificate-signed-by-unknown-authorit

# RUN apt-get update && apt-get install -y ca-certificates openssl
# ARG cert_location=/usr/local/share/ca-certificates
# # Get certificate from "github.com"
# RUN openssl s_client -showcerts -connect github.com:443 </dev/null 2>/dev/null|openssl x509 -outform PEM > ${cert_location}/github.crt
# # Get certificate from "proxy.golang.org"
# RUN openssl s_client -showcerts -connect proxy.golang.org:443 </dev/null 2>/dev/null|openssl x509 -outform PEM >  ${cert_location}/proxy.golang.crt
# # Update certificates
# RUN update-ca-certificates

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY resources/ resources/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go

FROM busybox:1.35.0-uclibc as busybox
# Use kcp plugins as of Oct. 22 for now
FROM ghcr.io/kcp-dev/kcp:bb0332e as kcp

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=busybox /bin/sh /bin/sh
COPY --from=busybox /bin/mkdir /bin/mkdir
COPY --from=busybox /bin/tar /bin/tar
COPY --from=busybox /bin/cat /bin/cat
COPY --from=busybox /bin/ls /bin/ls
COPY --from=busybox /bin/cp /bin/cp
COPY --from=kcp /usr/local/bin/kubectl /bin/kubectl
COPY --from=kcp /usr/local/bin/kubectl-kcp /bin/kubectl-kcp
COPY --from=kcp /usr/local/bin/kubectl-workspace /bin/kubectl-workspace

RUN mkdir -p /tmp/kyverno-manifests /tmp/kcp
# Manifests for Kyverno installation
COPY ./config/kyverno/*.yaml /tmp/kyverno-manifests
# Manifest for API binding to bind k8s basic resource in the target workspace to which Kyverno will be installed 
COPY ./config/kcp/apibindings.yaml /tmp/kcp/apibindings.yaml
# Set Kyverno manifests directory and APIBiinding file path to environment variables
ENV WORKSPACE_KYVERNO_INSTALL_MANIFESTS_DIR=/tmp/kyverno-manifests WORKSPACE_APIBINDINGS_MANIFEST=/tmp/kcp/apibindings.yaml

USER 65532:65532

ENTRYPOINT ["/manager"]
