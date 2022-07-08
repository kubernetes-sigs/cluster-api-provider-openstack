<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [External Cloud Provider](#external-cloud-provider)
  - [Use automatic scripts)(#Use automatic scripts)
  - [Step by step](#steps)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# External Cloud Provider

To deploy a cluster using [external cloud provider](https://github.com/kubernetes/cloud-provider-openstack), create a cluster configuration with the [external cloud provider template](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/templates/cluster-template-external-cloud-provider.yaml) or refer to [helm chart](https://github.com/kubernetes/cloud-provider-openstack/tree/master/charts/openstack-cloud-controller-manager).

## Use automatic scripts (easier way)

Use `./templates/external_cloud_setup.sh` to apply all the manifests needed. Refer to following command as reference.

```
# bash external_cloud_setup.sh capi-quickstart
##########################################
Create kubeconfig file capi-quickstart.kubeconfig for cluster capi-quickstart

New clusterctl version available: v1.1.4 -> v1.1.5
https://github.com/kubernetes-sigs/cluster-api/releases/tag/v1.1.5
##########################################
Create secret cloud-config
secret/cloud-config created
##########################################
Apply OCCM manifests
clusterrole.rbac.authorization.k8s.io/system:cloud-controller-manager created
clusterrole.rbac.authorization.k8s.io/system:cloud-node-controller created
clusterrolebinding.rbac.authorization.k8s.io/system:cloud-node-controller created
clusterrolebinding.rbac.authorization.k8s.io/system:cloud-controller-manager created
serviceaccount/cloud-controller-manager created
daemonset.apps/openstack-cloud-controller-manager created
##########################################
Install CNI, currently it's calico
configmap/calico-config created
customresourcedefinition.apiextensions.k8s.io/bgpconfigurations.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/bgppeers.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/blockaffinities.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/caliconodestatuses.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/clusterinformations.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/felixconfigurations.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/globalnetworkpolicies.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/globalnetworksets.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/hostendpoints.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/ipamblocks.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/ipamconfigs.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/ipamhandles.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/ippools.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/ipreservations.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/kubecontrollersconfigurations.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/networkpolicies.crd.projectcalico.org created
customresourcedefinition.apiextensions.k8s.io/networksets.crd.projectcalico.org created
clusterrole.rbac.authorization.k8s.io/calico-kube-controllers created
clusterrolebinding.rbac.authorization.k8s.io/calico-kube-controllers created
clusterrole.rbac.authorization.k8s.io/calico-node created
clusterrolebinding.rbac.authorization.k8s.io/calico-node created
daemonset.apps/calico-node created
serviceaccount/calico-node created
deployment.apps/calico-kube-controllers created
serviceaccount/calico-kube-controllers created
poddisruptionbudget.policy/calico-kube-controllers created
```

## Steps of using external cloud provider template (step by step)

- After control plane is up and running, retrieve the workload cluster Kubeconfig:

    ```shell
    clusterctl get kubeconfig ${CLUSTER_NAME} --namespace default > ./${CLUSTER_NAME}.kubeconfig
    ```

- Deploy a CNI solution (using Calico now)

    Note: choose desired version by replace <v3.23> below

    ```shell
    kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f https://docs.projectcalico.org/v3.23/manifests/calico.yaml
    ```

- Create a secret containing the cloud configuration

    ```shell
    templates/create_cloud_conf.sh <path/to/clouds.yaml> <cloud> > /tmp/cloud.conf
    ```

    ```shell
    kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig create secret -n kube-system generic cloud-config --from-file=/tmp/cloud.conf
    ```

    ```shell
    rm /tmp/cloud.conf
    ```

- Create RBAC resources and openstack-cloud-controller-manager deamonset

    ```shell
    kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f https://raw.githubusercontent.com/kubernetes/cloud-provider-openstack/master/manifests/controller-manager/cloud-controller-manager-roles.yaml
    kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f https://raw.githubusercontent.com/kubernetes/cloud-provider-openstack/master/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml
    kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f https://raw.githubusercontent.com/kubernetes/cloud-provider-openstack/master/manifests/controller-manager/openstack-cloud-controller-manager-ds.yaml
    ```

- Waiting for all the pods in kube-system namespace up and running

    ```shell
    $ kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig get pod -n kube-system
    NAME                                                    READY   STATUS    RESTARTS   AGE
    calico-kube-controllers-5569bdd565-ncrff                1/1     Running   0          20m
    calico-node-g5qqq                                       1/1     Running   0          20m
    calico-node-hdgxs                                       1/1     Running   0          20m
    coredns-864fccfb95-8qgp2                                1/1     Running   0          109m
    coredns-864fccfb95-b4zsf                                1/1     Running   0          109m
    etcd-mycluster-control-plane-cp2zw                      1/1     Running   0          108m
    kube-apiserver-mycluster-control-plane-cp2zw            1/1     Running   0          110m
    kube-controller-manager-mycluster-control-plane-cp2zw   1/1     Running   0          109m
    kube-proxy-mxkdp                                        1/1     Running   0          107m
    kube-proxy-rxltx                                        1/1     Running   0          109m
    kube-scheduler-mycluster-control-plane-cp2zw            1/1     Running   0          109m
    openstack-cloud-controller-manager-rbxkz                1/1     Running   8          18m
    ```
