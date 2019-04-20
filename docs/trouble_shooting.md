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
