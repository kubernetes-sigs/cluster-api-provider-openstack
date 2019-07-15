<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Trouble shooting](#trouble-shooting)
  - [Get log of clusterapi-controllers containers](#get-log-of-clusterapi-controllers-containers)
  - [Master failed to start with error: node xxxx not found](#master-failed-to-start-with-error-node-xxxx-not-found)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Trouble shooting

This guide (based on minikube and others should be similar) explains general info on how to debug issues if cluster failed to create.

## Get log of clusterapi-controllers containers

1. Get openstack container name, the output depends on the system you are running.
   the `minikube.kubeconfig` which is bootstrap cluster's kubeconfig by default locates at `cmd/clusterctl` folder.

   ```
   # kubectl --kubeconfig minikube.kubeconfig get pods -n openstack-provider-system
   NAMESPACE                   NAME                                     READY   STATUS    RESTARTS   AGE
   openstack-provider-system   clusterapi-controllers-xxxxxxxxx-xxxxx   1/1     Running   0          27m
   ```

2. Get log of clusterapi-controllers-xxxxxxxx-xxxxx

   ```
   # kubectl --kubeconfig minikube.kubeconfig log clusterapi-controllers-xxxxxxxxx-xxxxx -n openstack-provider-system
   ```

## Master failed to start with error: node xxxx not found

Sometimes the master machine is created but failed to startup, take ubuntu as example, open `/var/log/messages`
and if you see something like
```
Jul 10 00:07:58 openstack-master-5wgrw kubelet: E0710 00:07:58.444950 4340 kubelet.go:2248] node "openstack-master-5wgrw" not found
Jul 10 00:07:58 openstack-master-5wgrw kubelet: I0710 00:07:58.526091 4340 kubelet_node_status.go:72] Attempting to register node openstack-master-5wgrw
Jul 10 00:07:58 openstack-master-5wgrw kubelet: E0710 00:07:58.527398 4340 kubelet_node_status.go:94] Unable to register node "openstack-master-5wgrw" with API server: nodes "openstack-master-5wgrw" is forbidden: node "openstack-master-5wgrw.novalocal" is not allowed to modify node "openstack-master-5wgrw"
```

This might be caused by [This issue](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/issues/391), try the method proposed there.
