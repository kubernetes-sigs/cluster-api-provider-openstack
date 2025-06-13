<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [External Cloud Provider](#external-cloud-provider)
  - [Steps of using external cloud provider template](#steps-of-using-external-cloud-provider-template)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# External Cloud Provider

All [cluster templates](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/blob/main/templates) are meant to be used with the external cloud provider for OpenStack.
Refer to the [external cloud provider repository](https://github.com/kubernetes/cloud-provider-openstack) or the [helm chart](https://github.com/kubernetes/cloud-provider-openstack/tree/master/charts/openstack-cloud-controller-manager) for more details.

## Steps of using external cloud provider template

- After control plane is up and running, retrieve the workload cluster Kubeconfig:

    ```shell
    clusterctl get kubeconfig ${CLUSTER_NAME} --namespace default > ./${CLUSTER_NAME}.kubeconfig
    ```

- Deploy a CNI solution (using Calico now)

    Note: choose desired version by replace <v3.23> below

    ```shell
    kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f https://docs.projectcalico.org/archive/v3.23/manifests/calico.yaml
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
