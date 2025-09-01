<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Hosted Control Plane on OpenStack with Cluster API](#hosted-control-plane-on-openstack-with-cluster-api)
  - [Overview](#overview)
    - [Advantages of Hosted Control Planes](#advantages-of-hosted-control-planes)
  - [Prerequisites](#prerequisites)
    - [Management Cluster Requirements](#management-cluster-requirements)
    - [Architecture Considerations](#architecture-considerations)
  - [Setup Instructions](#setup-instructions)
    - [0. Create the hcp-system namespace](#0-create-the-hcp-system-namespace)
    - [1. Prepare OpenStack Credentials](#1-prepare-openstack-credentials)
    - [2. Ensure Storage is Available (Required for etcd)](#2-ensure-storage-is-available-required-for-etcd)
    - [3. Configure and Deploy the Cluster](#3-configure-and-deploy-the-cluster)
      - [Hosted Control Plane Cluster Manifests (`openstack-hcp-cluster.yaml`)](#hosted-control-plane-cluster-manifests-openstack-hcp-clusteryaml)
  - [Deployment and Monitoring](#deployment-and-monitoring)
    - [Deploy the Cluster](#deploy-the-cluster)
    - [Monitor Cluster Creation](#monitor-cluster-creation)
  - [Post-Deployment Configuration](#post-deployment-configuration)
    - [Retrieve Workload Cluster Access](#retrieve-workload-cluster-access)
    - [Configure OpenStack Integration](#configure-openstack-integration)
    - [Verify Workload Cluster](#verify-workload-cluster)
  - [Troubleshooting](#troubleshooting)
    - [Common Issues and Solutions](#common-issues-and-solutions)
  - [Conclusion](#conclusion)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Hosted Control Plane on OpenStack with Cluster API

## Overview 

This guide demonstrates how to deploy a Kubernetes cluster on OpenStack using Cluster API Provider OpenStack (CAPO) with a hosted control plane architecture. The guide uses k0smotron as the hosted control plane provider to illustrate a fully CAPI-native approach.

### Advantages of Hosted Control Planes

- **Resource Efficiency**: Control plane components run as pods on the management cluster, reducing the infrastructure footprint
- **Simplified Management**: Centralized control plane management across multiple workload clusters
- **High Availability**: Leverages the management cluster's infrastructure for control plane resilience
- **Cost Optimization**: Eliminates the need for dedicated control plane nodes in each workload cluster

> **Note**: CAPO is not opinionated about which control plane provider you use. This guide uses k0smotron as an example. For alternative hosted control plane solutions like Kamaji, see the [Kamaji documentation](https://kamaji.clastix.io/getting-started/kamaji-generic/).

This architecture consists of:
- A management cluster that hosts the control plane pods
- A workload cluster with worker nodes created by CAPI/CAPO

## Prerequisites

Before proceeding, ensure your management cluster meets the following requirements:

### Management Cluster Requirements

- A healthy Kubernetes cluster with cluster-admin access
- Cluster API initialized with the OpenStack infrastructure provider:
  ```bash
  clusterctl init --infrastructure openstack
  ```
- k0smotron installed (docs: https://docs.k0smotron.io/stable/install/#software-prerequisites)
- A storage solution for etcd (required), e.g. OpenStack Cinder CSI (recommended), local-path-provisioner, Ceph CSI, or any CSI driver with a default StorageClass
- A LoadBalancer implementation for the hosted control plane service (recommended), e.g. OpenStack CCM/Octavia or MetalLB

### Architecture Considerations

The hosted control plane etcd persistent volume resides on the management cluster, requiring functional storage. The workload cluster receives its own Cloud Controller Manager (CCM) and Container Storage Interface (CSI) drivers through k0s configuration extensions.

## Setup Instructions

### 0. Create the hcp-system namespace

```bash
kubectl create namespace hcp-system
```
*Note: Throughout the guide, we will use the `hcp-system` namespace to deploy all the components. Ideally, you can choose a different namespace, but you will need to adjust the manifests accordingly.*

### 1. Prepare OpenStack Credentials

Create a secret with credentials for the workload cluster external cloud provider. The cluster manifest expects a secret named `openstack-cloud-config` in the `hcp-system` namespace with a `clouds.yaml` key.

Create the secret directly from your `clouds.yaml` file like this:
```bash
kubectl -n hcp-system create secret generic openstack-cloud-config \
  --from-file=clouds.yaml=./clouds.yaml
```

### 2. Ensure Storage is Available (Required for etcd)

The hosted control plane requires persistent storage for etcd. Ensure your management cluster has:
- A functional CSI driver (e.g., OpenStack Cinder CSI, local-path-provisioner, Ceph CSI)
- A default StorageClass configured

Verify storage is ready:
```bash
kubectl get storageclass
```

### 3. Configure and Deploy the Cluster

Before applying the manifests, update the following fields:

- Update network, router, and subnet names in `OpenStackCluster` and `OpenStackMachineTemplate` to match your OpenStack environment

#### Hosted Control Plane Cluster Manifests (`openstack-hcp-cluster.yaml`)

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: openstack-hcp-cluster
  namespace: hcp-system
spec:
  clusterNetwork:
    pods:
      cidrBlocks: [10.244.0.0/16] # Adjust accordingly
    serviceDomain: cluster.local
    services:
      cidrBlocks: [10.96.0.0/12] # Adjust accordingly
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: K0smotronControlPlane
    name: openstack-hcp-cluster-cp
    namespace: hcp-system
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: OpenStackCluster
    name: openstack-hcp-cluster
    namespace: hcp-system
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackCluster
metadata:
  name: openstack-hcp-cluster
  namespace: hcp-system
spec:
  externalNetwork:
    filter:
      name: public
  identityRef:
    cloudName: openstack
    name: openstack-cloud-config
    region: RegionOne
  network:
    filter:
      name: k8s-clusterapi-cluster-default-capo-test
  router:
    filter:
      name: k8s-clusterapi-cluster-default-capo-test
  subnets:
  - filter:
      name: k8s-clusterapi-cluster-default-capo-test
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: openstack-hcp-cluster-md
  namespace: hcp-system
spec:
  clusterName: openstack-hcp-cluster
  replicas: 1
  selector:
    matchLabels:
      cluster.x-k8s.io/cluster-name: openstack-hcp-cluster
  template:
    metadata:
      labels:
        cluster.x-k8s.io/cluster-name: openstack-hcp-cluster
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: K0sWorkerConfigTemplate
          name: openstack-hcp-cluster-machine-config
          namespace: hcp-system
      clusterName: openstack-hcp-cluster
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
        kind: OpenStackMachineTemplate
        name: openstack-hcp-cluster-mt
        namespace: hcp-system
      version: v1.32.6
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackMachineTemplate
metadata:
  name: openstack-hcp-cluster-mt
  namespace: hcp-system
spec:
  template:
    spec:
      flavor: m1.medium
      identityRef:
        cloudName: openstack
        name: openstack-cloud-config
        region: RegionOne
      image:
        filter:
          name: ubuntu-22.04-x86_64
      ports:
      - network:
          filter:
            name: k8s-clusterapi-cluster-default-capo-test
      securityGroups:
      - filter:
          name: default
---
apiVersion: controlplane.cluster.x-k8s.io/v1beta1
kind: K0smotronControlPlane
metadata:
  name: openstack-hcp-cluster-cp
  namespace: hcp-system
spec:
  controllerPlaneFlags:
  - --enable-cloud-provider=true
  - --debug=true
  etcd:
    autoDeletePVCs: false
    image: quay.io/k0sproject/etcd:v3.5.13
    persistence:
      size: 1Gi
  image: ghcr.io/k0sproject/k0s:v1.32.6-k0s.0  # pinned GHCR tag to avoid rate limits with docker hub
  k0sConfig:
    apiVersion: k0s.k0sproject.io/v1beta1
    kind: ClusterConfig
    metadata:
      name: k0s
    spec:
      extensions:
        helm:
          charts:
          - chartname: openstack/openstack-cloud-controller-manager
            name: openstack-ccm
            namespace: kube-system
            order: 1
            values: |
              secret:
                enabled: true
                name: openstack-cloud-config
                create: false
              nodeSelector: null
              tolerations:
                - key: node.cloudprovider.kubernetes.io/uninitialized
                  value: "true"
                  effect: NoSchedule
                - key: node-role.kubernetes.io/control-plane
                  effect: NoSchedule
                - key: node-role.kubernetes.io/master
                  effect: NoSchedule
              extraEnv:
                - name: OS_CCM_REGIONAL
                  value: "true"
              extraVolumes:
                - name: flexvolume-dir
                  hostPath:
                    path: /usr/libexec/kubernetes/kubelet-plugins/volume/exec
                - name: k8s-certs
                  hostPath:
                    path: /etc/kubernetes/pki
              extraVolumeMounts:
                - name: flexvolume-dir
                  mountPath: /usr/libexec/kubernetes/kubelet-plugins/volume/exec
                  readOnly: true
                - name: k8s-certs
                  mountPath: /etc/kubernetes/pki
                  readOnly: true
            version: 2.31.1
          - chartname: openstack/openstack-cinder-csi
            name: openstack-csi
            namespace: kube-system
            order: 2
            values: |
              storageClass:
                enabled: true
                delete:
                  isDefault: true
                  allowVolumeExpansion: true
                retain:
                  isDefault: false
                  allowVolumeExpansion: false
              secret:
                enabled: true
                name: openstack-cloud-config
                create: false   # set to true if you want the chart to create the Secret in workload cluster
              csi:
                plugin:
                  nodePlugin:
                    kubeletDir: /var/lib/k0s/kubelet   # workload cluster nodes run k0s
            version: 2.31.2
          repositories:
          - name: openstack
            url: https://kubernetes.github.io/cloud-provider-openstack/
      network:
        calico:
          mode: vxlan
        clusterDomain: cluster.local
        podCIDR: 10.244.0.0/16
        provider: calico
        serviceCIDR: 10.96.0.0/12
  replicas: 1
  service:
    apiPort: 6443
    konnectivityPort: 8132
    type: LoadBalancer
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: K0sWorkerConfigTemplate
metadata:
  name: openstack-hcp-cluster-machine-config
  namespace: hcp-system
spec:
  template:
    spec:
      args:
      - --enable-cloud-provider
      - --kubelet-extra-args="--cloud-provider=external"
      - --debug=true
      version: v1.32.6+k0s.0
```

## Deployment and Monitoring

### Deploy the Cluster

Apply the cluster manifest:

```bash
kubectl apply -f openstack-hcp-cluster.yaml
```

### Monitor Cluster Creation

Monitor the hosted control plane deployment:

```bash
# Watch etcd PVC binding and pod startup on the management cluster
kubectl -n hcp-system get pvc
kubectl -n hcp-system get pods -w

# Verify the LoadBalancer service receives an external IP address
kubectl -n hcp-system get svc openstack-hcp-cluster-cp -o wide
```

Expected components in the `hcp-system` namespace:
- `kmc-openstack-hcp-cluster-etcd-0` pod in Running state
- `kmc-openstack-hcp-cluster-0` (controller) pod in Running state  
- `openstack-hcp-cluster-cp` service (LoadBalancer) with an assigned EXTERNAL-IP

The control plane will become operational within a few minutes, followed by worker nodes joining the cluster.

## Post-Deployment Configuration

### Retrieve Workload Cluster Access

Obtain the workload cluster kubeconfig:

```bash
clusterctl -n hcp-system get kubeconfig openstack-hcp-cluster > workload-cluster.kubeconfig
```

### Configure OpenStack Integration

If the cluster manifest has `create: false` for secrets (as shown in the example), manually create the OpenStack credentials in the workload cluster:

```bash
kubectl --kubeconfig workload-cluster.kubeconfig -n kube-system create secret generic openstack-cloud-config \
  --from-file=clouds.yaml=./clouds.yaml
```

### Verify Workload Cluster

Validate the workload cluster components and nodes:

```bash
kubectl --kubeconfig workload-cluster.kubeconfig -n kube-system get pods | egrep -i 'openstack|cinder|calico'
kubectl --kubeconfig workload-cluster.kubeconfig get sc
kubectl --kubeconfig workload-cluster.kubeconfig get nodes
```

## Troubleshooting

### Common Issues and Solutions

**PVC Pending (Management Cluster)**
- Ensure Cinder CSI is running on the management cluster
- Verify correct kubelet directory configuration
- Check OpenStack credentials secret is properly mounted
- Confirm default StorageClass is configured

**Node Plugin Mount Errors**
- Verify kubelet directory mismatch: management cluster uses `/var/lib/kubelet`, workload cluster k0s nodes use `/var/lib/k0s/kubelet`

**Invalid Image Name Errors**
- Avoid double-tagging by pinning the image to `ghcr.io/k0sproject/k0s:<version>-k0s.0`
- Set `version: null` in the k0smotron configuration

**Image Pull Errors (429 rate limit)**
- Use GitHub Container Registry (GHCR) instead of Docker Hub
- Configure imagePullSecrets if necessary

**LoadBalancer Pending**
- Ensure the management cluster has a functional LoadBalancer implementation (OpenStack CCM/Octavia or MetalLB)

## Conclusion

This guide demonstrated how to deploy a Kubernetes cluster on OpenStack using a hosted control plane architecture with Cluster API Provider OpenStack (CAPO) and k0smotron.

For production deployments, ensure proper sizing of the management cluster to handle multiple hosted control planes and their associated workloads.
