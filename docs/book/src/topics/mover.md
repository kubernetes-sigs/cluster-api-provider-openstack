<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Pre-condition](#pre-condition)
- [Install OpenStack Cluster API provider into target cluster](#install-openstack-cluster-api-provider-into-target-cluster)
- [Move objects from `bootstrap` cluster into `target` cluster.](#move-objects-from-bootstrap-cluster-into-target-cluster)
- [Check cluster status](#check-cluster-status)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

This documentation describes how to move `Cluster API` related objects from `bootstrap` cluster to `target` cluster. Check [clusterctl move](https://cluster-api.sigs.k8s.io/clusterctl/commands/move.html) for further information.

# Pre-condition

Bootstrap cluster
```
# kubectl get pods --all-namespaces
NAMESPACE                           NAME                                                             READY   STATUS    RESTARTS   AGE
capi-kubeadm-bootstrap-system       capi-kubeadm-bootstrap-controller-manager-68cfd4c5b8-mjq75       2/2     Running   0          27m
capi-kubeadm-control-plane-system   capi-kubeadm-control-plane-controller-manager-848575ccb7-m672j   2/2     Running   0          27m
capi-system                         capi-controller-manager-564d97d59c-2t7sl                         2/2     Running   0          27m
capi-webhook-system                 capi-controller-manager-9c8b5d6d4-49czx                          2/2     Running   0          28m
capi-webhook-system                 capi-kubeadm-bootstrap-controller-manager-7dff4b8c7-8w9sq        2/2     Running   1          27m
capi-webhook-system                 capi-kubeadm-control-plane-controller-manager-74c99998d-bftbn    2/2     Running   0          27m
capi-webhook-system                 capo-controller-manager-7d7bfc856b-5ttw6                         2/2     Running   0          24m
capo-system                         capo-controller-manager-5fb48fcb4c-ttkpv                         2/2     Running   0          25m
cert-manager                        cert-manager-544d659678-l9pjb                                    1/1     Running   0          29m
cert-manager                        cert-manager-cainjector-64c9f978d7-bjxkg                         1/1     Running   0          29m
cert-manager                        cert-manager-webhook-5855bb8c8c-8hb9w                            1/1     Running   0          29m
kube-system                         coredns-66bff467f8-ggn54                                         1/1     Running   0          40m
kube-system                         coredns-66bff467f8-t4bqr                                         1/1     Running   0          40m
kube-system                         etcd-kind-control-plane                                          1/1     Running   1          40m
kube-system                         kindnet-ng2gf                                                    1/1     Running   0          40m
kube-system                         kube-apiserver-kind-control-plane                                1/1     Running   1          40m
kube-system                         kube-controller-manager-kind-control-plane                       1/1     Running   1          40m
kube-system                         kube-proxy-h6rmz                                                 1/1     Running   0          40m
kube-system                         kube-scheduler-kind-control-plane                                1/1     Running   1          40m
local-path-storage                  local-path-provisioner-bd4bb6b75-ft7wh                           1/1     Running   0          40m
```

Target cluster (Below is an example of external cloud provider)
```
# kubectl get pods --kubeconfig capi-openstack-3.kubeconfig --all-namespaces
NAMESPACE     NAME                                                         READY   STATUS    RESTARTS   AGE
kube-system   calico-kube-controllers-59b699859f-djqd7                     1/1     Running   0          6m2s
kube-system   calico-node-szp44                                            1/1     Running   0          6m2s
kube-system   calico-node-xhgzr                                            1/1     Running   0          6m2s
kube-system   coredns-6955765f44-wk2vq                                     1/1     Running   0          21m
kube-system   coredns-6955765f44-zhps9                                     1/1     Running   0          21m
kube-system   etcd-capi-openstack-control-plane-82xck                      1/1     Running   0          22m
kube-system   kube-apiserver-capi-openstack-control-plane-82xck            1/1     Running   0          22m
kube-system   kube-controller-manager-capi-openstack-control-plane-82xck   1/1     Running   2          22m
kube-system   kube-proxy-4f9k8                                             1/1     Running   0          21m
kube-system   kube-proxy-gjd55                                             1/1     Running   0          21m
kube-system   kube-scheduler-capi-openstack-control-plane-82xck            1/1     Running   2          22m
kube-system   openstack-cloud-controller-manager-z9jtc                     1/1     Running   1          4m9s
```

# Install OpenStack Cluster API provider into target cluster

You need install OpenStack cluster api providers into `target` cluster first.
```
# clusterctl --kubeconfig capi-openstack-3.kubeconfig  init --infrastructure openstack
Fetching providers
Installing cert-manager
Waiting for cert-manager to be available...
Installing Provider="cluster-api" Version="v0.3.8" TargetNamespace="capi-system"
Installing Provider="bootstrap-kubeadm" Version="v0.3.8" TargetNamespace="capi-kubeadm-bootstrap-system"
Installing Provider="control-plane-kubeadm" Version="v0.3.8" TargetNamespace="capi-kubeadm-control-plane-system"
Installing Provider="infrastructure-openstack" Version="v0.3.1" TargetNamespace="capo-system"

Your management cluster has been initialized successfully!

You can now create your first workload cluster by running the following:

  clusterctl generate cluster [name] --kubernetes-version [version] | kubectl apply -f -
```

# Move objects from `bootstrap` cluster into `target` cluster.

CRD, objects such as `OpenStackCluster`, `OpenStackMachine` etc need to be moved.
```
# clusterctl move --to-kubeconfig capi-openstack-3.kubeconfig -v 10
No default config file available
Performing move...
Discovering Cluster API objects
Cluster Count=1
KubeadmConfig Count=2
KubeadmConfigTemplate Count=1
KubeadmControlPlane Count=1
MachineDeployment Count=1
Machine Count=2
MachineSet Count=1
OpenStackCluster Count=1
OpenStackMachine Count=2
OpenStackMachineTemplate Count=2
Secret Count=9
Total objects Count=23
Moving Cluster API objects Clusters=1
Pausing the source cluster
Set Cluster.Spec.Paused Paused=true Cluster="capi-openstack-3" Namespace="default"
Creating target namespaces, if missing
Creating objects in the target cluster
Creating Cluster="capi-openstack-3" Namespace="default"
Creating OpenStackMachineTemplate="capi-openstack-control-plane" Namespace="default"
Creating OpenStackMachineTemplate="capi-openstack-md-0" Namespace="default"
Creating OpenStackCluster="capi-openstack-3" Namespace="default"
Creating KubeadmConfigTemplate="capi-openstack-md-0" Namespace="default"
Creating MachineDeployment="capi-openstack-md-0" Namespace="default"
Creating KubeadmControlPlane="capi-openstack-control-plane" Namespace="default"
Creating Secret="capi-openstack-3-etcd" Namespace="default"
Creating Machine="capi-openstack-control-plane-n2kdq" Namespace="default"
Creating Secret="capi-openstack-3-sa" Namespace="default"
Creating Secret="capi-openstack-3-kubeconfig" Namespace="default"
Creating Secret="capi-openstack-3-proxy" Namespace="default"
Creating MachineSet="capi-openstack-md-0-dfdf94979" Namespace="default"
Creating Secret="capi-openstack-3-ca" Namespace="default"
Creating Machine="capi-openstack-md-0-dfdf94979-656zq" Namespace="default"
Creating KubeadmConfig="capi-openstack-control-plane-xzj7x" Namespace="default"
Creating OpenStackMachine="capi-openstack-control-plane-82xck" Namespace="default"
Creating Secret="capi-openstack-control-plane-xzj7x" Namespace="default"
Creating OpenStackMachine="capi-openstack-md-0-bkwhh" Namespace="default"
Creating KubeadmConfig="capi-openstack-md-0-t44gj" Namespace="default"
Creating Secret="capi-openstack-md-0-t44gj" Namespace="default"
Deleting objects from the source cluster
Deleting Secret="capi-openstack-md-0-t44gj" Namespace="default"
Deleting Secret="capi-openstack-control-plane-xzj7x" Namespace="default"
Deleting OpenStackMachine="capi-openstack-md-0-bkwhh" Namespace="default"
Deleting KubeadmConfig="capi-openstack-md-0-t44gj" Namespace="default"
Deleting Machine="capi-openstack-md-0-dfdf94979-656zq" Namespace="default"
Deleting KubeadmConfig="capi-openstack-control-plane-xzj7x" Namespace="default"
Deleting OpenStackMachine="capi-openstack-control-plane-82xck" Namespace="default"
Deleting Secret="capi-openstack-3-etcd" Namespace="default"
Deleting Machine="capi-openstack-control-plane-n2kdq" Namespace="default"
Deleting Secret="capi-openstack-3-sa" Namespace="default"
Deleting Secret="capi-openstack-3-kubeconfig" Namespace="default"
Deleting Secret="capi-openstack-3-proxy" Namespace="default"
Deleting MachineSet="capi-openstack-md-0-dfdf94979" Namespace="default"
Deleting Secret="capi-openstack-3-ca" Namespace="default"
Deleting OpenStackMachineTemplate="capi-openstack-control-plane" Namespace="default"
Deleting OpenStackMachineTemplate="capi-openstack-md-0" Namespace="default"
Deleting OpenStackCluster="capi-openstack-3" Namespace="default"
Deleting KubeadmConfigTemplate="capi-openstack-md-0" Namespace="default"
Deleting MachineDeployment="capi-openstack-md-0" Namespace="default"
Deleting KubeadmControlPlane="capi-openstack-control-plane" Namespace="default"
Deleting Cluster="capi-openstack-3" Namespace="default"
Resuming the target cluster
Set Cluster.Spec.Paused Paused=false Cluster="capi-openstack-3" Namespace="default"
```
# Check cluster status

```
# kubectl get openstackcluster --kubeconfig capi-openstack-3.kubeconfig --all-namespaces
NAMESPACE   NAME               CLUSTER            READY   NETWORK                                SUBNET
default     capi-openstack-3   capi-openstack-3   true    4a6f2d57-bb3d-44f4-a28a-4c94a92e41d0   1a1a1d9d-5258-42cb-8756-fa4c648af72b

# kubectl get openstackmachines --kubeconfig capi-openstack-3.kubeconfig --all-namespaces
NAMESPACE   NAME                                 CLUSTER            STATE    READY   INSTANCEID                                         MACHINE
default     capi-openstack-control-plane-82xck   capi-openstack-3   ACTIVE   true    openstack:///f29007c5-f672-4214-a508-b7cf4a17b3ed   capi-openstack-control-plane-n2kdq
default     capi-openstack-md-0-bkwhh            capi-openstack-3   ACTIVE   true    openstack:///6e23324d-315a-4d75-85a9-350fd1705ab6   capi-openstack-md-0-dfdf94979-656zq
```
