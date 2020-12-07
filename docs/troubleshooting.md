<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Troubleshooting](#troubleshooting)
  - [Get logs of Cluster API controller containers](#get-logs-of-cluster-api-controller-containers)
  - [Master failed to start with error: node xxxx not found](#master-failed-to-start-with-error-node-xxxx-not-found)
  - [providerClient authentication err](#providerclient-authentication-err)
  - [Fails in creating floating IP during cluster creation.](#fails-in-creating-floating-ip-during-cluster-creation)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Troubleshooting

This guide (based on Minikube but others should be similar) explains general info on how to debug issues if a cluster creation fails.

## Get logs of Cluster API controller containers

```bash
kubectl --kubeconfig minikube.kubeconfig -n capo-system logs -l control-plane=capo-controller-manager -c manager
```

Similarly, the logs of the other controllers in the namespaces `capi-system` and `cabpk-system` can be retrieved.

## Master failed to start with error: node xxxx not found

Sometimes the master machine is created but fails to startup, take Ubuntu as example, open `/var/log/messages`
and if you see something like this:
```
Jul 10 00:07:58 openstack-master-5wgrw kubelet: E0710 00:07:58.444950 4340 kubelet.go:2248] node "openstack-master-5wgrw" not found
Jul 10 00:07:58 openstack-master-5wgrw kubelet: I0710 00:07:58.526091 4340 kubelet_node_status.go:72] Attempting to register node openstack-master-5wgrw
Jul 10 00:07:58 openstack-master-5wgrw kubelet: E0710 00:07:58.527398 4340 kubelet_node_status.go:94] Unable to register node "openstack-master-5wgrw" with API server: nodes "openstack-master-5wgrw" is forbidden: node "openstack-master-5wgrw.novalocal" is not allowed to modify node "openstack-master-5wgrw"
```

This might be caused by [This issue](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/issues/391), try the method proposed there.

## providerClient authentication err

If you are using https, you must specify the CA certificate in your `clouds.yaml` file, and when you encounter issue like:

```bash
kubectl --kubeconfig minikube.kubeconfig logs -n capo-system logs -l control-plane=capo-controller-manager
...
E0814 04:32:52.688514       1 machine_controller.go:204] Failed to check if machine "openstack-master-hxk9r" exists: providerClient authentication err: Post https://xxxxxxxxxxxxxxx:5000/v3/auth/tokens: x509: certificate signed by unknown authority
...
```

you can also add `verify: false` into `clouds.yaml` file to solve the problem.
```
clouds:
  openstack:
    auth:
        ....
    region_name: "RegionOne"
    interface: "public"
    identity_api_version: 3
    cacert: /etc/certs/cacert
    verify: false
```

## Fails in creating floating IP during cluster creation.

If you encounter `rule:create_floatingip and rule:create_floatingip:floating_ip_address is disallowed by policy` when create floating ip, check with your openstack administrator, you need to be authorized to perform those actions, see [issue 572](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/issues/572) for more detailed information.

Refer to [rule:create_floatingip](https://github.com/openstack/neutron/blob/master/neutron/conf/policies/floatingip.py#L26) and [rule:create_floatingip:floating_ip_address](https://github.com/openstack/neutron/blob/master/neutron/conf/policies/floatingip.py#L36) for further policy information.

An alternative is to create the floating IP before create the cluster and use it.

## calico failed to start

After cluster is running, you might see following error:

```
root@jjtest7:~/abc# kubectl --kubeconfig capi-openstack-3.kubeconfig get pods --all-namespaces
NAMESPACE     NAME                                                         READY   STATUS                  RESTARTS   AGE
kube-system   calico-kube-controllers-59b699859f-hqpbt                     0/1     Pending                 0          118m
kube-system   calico-node-gntzr                                            0/1     Init:ImagePullBackOff   0          118m
kube-system   calico-node-rfhqb                                            0/1     Init:ImagePullBackOff   0          118m
kube-system   coredns-6955765f44-k6l6j                                     0/1     Pending                 0          120m
kube-system   coredns-6955765f44-zvdml                                     0/1     Pending                 0          120m
```

This might be caused by docker pull limit, check whether you are using anonymous user and use registered user instead.

```
  ----     ------   ----                  ----                                         -------
  Normal   BackOff  20m (x432 over 120m)  kubelet, capi-openstack-control-plane-5ztrj  Back-off pulling image "calico/cni:v3.12.3"
  Warning  Failed   20s (x508 over 120m)  kubelet, capi-openstack-control-plane-5ztrj  Error: ImagePullBackOff


429 Too Many Requests - Server message: toomanyrequests: You have reached your pull rate limit. You may increase the limit by authenticating and upgrading: https://www.docker.com/increase-rate-limit
```
