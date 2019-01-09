FROM registry.svc.ci.openshift.org/openshift/release:golang-1.10 AS builder
WORKDIR /go/src/sigs.k8s.io/cluster-api-provider-openstack
COPY . .

# Needed for the cluster-api/cmd/maager build
RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure -add k8s.io/apimachinery/pkg/util/rand k8s.io/apimachinery/pkg/api/equality k8s.io/client-go/plugin/pkg/client/auth/gcp

RUN go build -o ./machine-controller-manager ./cmd/manager
RUN go build -o ./manager ./vendor/sigs.k8s.io/cluster-api/cmd/manager

FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base
RUN INSTALL_PKGS=" \
      openssh \
      " && \
    yum install -y $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all
COPY --from=builder /go/src/sigs.k8s.io/cluster-api-provider-openstack/manager /
COPY --from=builder /go/src/sigs.k8s.io/cluster-api-provider-openstack/machine-controller-manager /
