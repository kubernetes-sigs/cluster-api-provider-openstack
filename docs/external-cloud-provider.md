# External Cloud Provider

To deploy a cluster using [external cloud provider](https://github.com/kubernetes-sigs/cloud-provider-openstack), create a cluster configuration with the [external cloud provider template](../templates/cluster-template-external-cloud-provider.yaml).

## Steps

- After control plane is up and running, retrieve the workload cluster Kubeconfig:

    ```shell
    kubectl --namespace=default get secret/${CLUSTER_NAME}-kubeconfig -o jsonpath={.data.value} \
    | base64 --decode \
    > ./${CLUSTER_NAME}.kubeconfig
    ```

- Deploy a CNI solution

    ```shell
    curl https://docs.projectcalico.org/v3.12/manifests/calico.yaml | sed "s/veth_mtu:.*/veth_mtu: \"1430\"/g" | kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f -
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
    kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f https://raw.githubusercontent.com/kubernetes/cloud-provider-openstack/master/cluster/addons/rbac/cloud-controller-manager-roles.yaml
    kubectl --kubeconfig=./${CLUSTER_NAME}.kubeconfig apply -f https://raw.githubusercontent.com/kubernetes/cloud-provider-openstack/master/cluster/addons/rbac/cloud-controller-manager-role-bindings.yaml
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
