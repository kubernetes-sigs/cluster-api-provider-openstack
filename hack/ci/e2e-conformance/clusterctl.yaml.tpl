providers:
- name: openstack
  type: InfrastructureProvider
  url: ${PWD}/out/infrastructure-openstack/${E2E_CAPO_VERSION}/infrastructure-components.yaml
- name: cluster-api
  type: CoreProvider
  url: ${PWD}/out/cluster-api/v${E2E_CAPI_VERSION}/core-components.yaml
- name: kubeadm
  type: BootstrapProvider
  url: ${PWD}/out/bootstrap-kubeadm/v${E2E_CAPI_VERSION}/bootstrap-components.yaml
- name: kubeadm
  type: ControlPlaneProvider
  url: ${PWD}/out/control-plane-kubeadm/v${E2E_CAPI_VERSION}/control-plane-components.yaml
