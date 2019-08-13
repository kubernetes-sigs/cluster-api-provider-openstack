package userdata

import (
	"fmt"
	"k8s.io/klog"
	kubeadmv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	providerv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/certificates"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/kubeadm"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	localIPV4Lookup = "${OPENSTACK_IPV4_LOCAL}"

	cloudProvider       = "openstack"
	cloudProviderConfig = "/etc/kubernetes/cloud.conf"

	nodeRole = "node-role.kubernetes.io/node="
)

func generateKubeadmConfig(isControlPlane bool, bootstrapToken string, cluster *clusterv1.Cluster, machine *clusterv1.Machine, machineProviderSpec *providerv1.OpenstackProviderSpec, clusterProviderSpec *providerv1.OpenstackClusterProviderSpec) (string, error) {

	caCertHash, err := certificates.GenerateCertificateHash(clusterProviderSpec.CAKeyPair.Cert)
	if err != nil {
		return "", err
	}

	if bootstrapToken != "" && len(cluster.Status.APIEndpoints) == 0 {
		return "", fmt.Errorf("no APIEndpoint set yet, waiting until APIEndpoints is set")
	}

	if isControlPlane {
		if bootstrapToken == "" {
			klog.Info("Machine is the first control plane machine for the cluster")

			if !clusterProviderSpec.CAKeyPair.HasCertAndKey() {
				return "", fmt.Errorf("failed to create controlplane kubeadm config, missing CAKeyPair")
			}

			clusterConfigurationCopy := clusterProviderSpec.ClusterConfiguration.DeepCopy()
			// Set default values. If they are not set, these two properties are added with an
			// empty string and the CoreOS ignition postprocesser fails with "could not find expected key"
			if clusterConfigurationCopy.CertificatesDir == "" {
				clusterConfigurationCopy.CertificatesDir = "/etc/kubernetes/pki"
			}
			if string(clusterConfigurationCopy.DNS.Type) == "" {
				clusterConfigurationCopy.DNS.Type = kubeadmv1beta1.CoreDNS
			}
			kubeadm.SetClusterConfigurationOptions(
				clusterConfigurationCopy,
				kubeadm.WithKubernetesVersion(machine.Spec.Versions.ControlPlane),
				kubeadm.WithAPIServerCertificateSANs(localIPV4Lookup),
				kubeadm.WithAPIServerExtraArgs(map[string]string{"cloud-provider": cloudProvider}),
				kubeadm.WithAPIServerExtraArgs(map[string]string{"cloud-config": cloudProviderConfig}),
				kubeadm.WithAPIServerExtraVolumes([]kubeadmv1beta1.HostPathMount{
					{
						Name:      "cloud",
						HostPath:  "/etc/kubernetes/cloud.conf",
						MountPath: "/etc/kubernetes/cloud.conf",
					},
				}),
				kubeadm.WithControllerManagerExtraArgs(map[string]string{"cloud-provider": cloudProvider}),
				kubeadm.WithControllerManagerExtraArgs(map[string]string{"cloud-config": cloudProviderConfig}),
				kubeadm.WithControllerManagerExtraVolumes([]kubeadmv1beta1.HostPathMount{
					{
						Name:      "cloud",
						HostPath:  "/etc/kubernetes/cloud.conf",
						MountPath: "/etc/kubernetes/cloud.conf",
					}, {
						Name:      "cacert",
						HostPath:  "/etc/certs/cacert",
						MountPath: "/etc/certs/cacert",
					},
				}),
				kubeadm.WithClusterNetworkFromClusterNetworkingConfig(cluster.Spec.ClusterNetwork),
				kubeadm.WithKubernetesVersion(machine.Spec.Versions.ControlPlane),
			)
			clusterConfigYAML, err := kubeadm.ConfigurationToYAML(clusterConfigurationCopy)
			if err != nil {
				return "", err
			}

			kubeadm.SetInitConfigurationOptions(
				&machineProviderSpec.KubeadmConfiguration.Init,
				kubeadm.WithNodeRegistrationOptions(
					kubeadm.NewNodeRegistration(
						kubeadm.WithTaints(machine.Spec.Taints),
						kubeadm.WithKubeletExtraArgs(map[string]string{"cloud-provider": cloudProvider}),
						kubeadm.WithKubeletExtraArgs(map[string]string{"cloud-config": cloudProviderConfig}),
					),
				),
				kubeadm.WithInitLocalAPIEndpointAndPort(localIPV4Lookup, cluster.Status.APIEndpoints[0].Port),
			)
			initConfigYAML, err := kubeadm.ConfigurationToYAML(&machineProviderSpec.KubeadmConfiguration.Init)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s\n---\n%s", clusterConfigYAML, initConfigYAML), nil

		} else {
			klog.Info("Allowing a machine to join the control plane")

			joinConfigurationCopy := machineProviderSpec.KubeadmConfiguration.Join
			// Set default values. If they are not set, these two properties are added with an
			// empty string and the CoreOS ignition postprocesser fails with "could not find expected key"
			joinConfigurationCopy.CACertPath = "/etc/kubernetes/pki/ca.crt"
			kubeadm.SetJoinConfigurationOptions(
				&joinConfigurationCopy,
				kubeadm.WithBootstrapTokenDiscovery(
					kubeadm.NewBootstrapTokenDiscovery(
						kubeadm.WithAPIServerEndpoint(clusterProviderSpec.ClusterConfiguration.ControlPlaneEndpoint),
						kubeadm.WithToken(bootstrapToken),
						kubeadm.WithCACertificateHash(caCertHash),
					),
				),
				kubeadm.WithTLSBootstrapToken(bootstrapToken),
				kubeadm.WithJoinNodeRegistrationOptions(
					kubeadm.NewNodeRegistration(
						kubeadm.WithTaints(machine.Spec.Taints),
						kubeadm.WithKubeletExtraArgs(map[string]string{"cloud-provider": cloudProvider}),
						kubeadm.WithKubeletExtraArgs(map[string]string{"cloud-config": cloudProviderConfig}),
					),
				),
				// this also creates .controlPlane
				kubeadm.WithLocalAPIEndpointAndPort(localIPV4Lookup, cluster.Status.APIEndpoints[0].Port),
			)
			joinConfigurationYAML, err := kubeadm.ConfigurationToYAML(&joinConfigurationCopy)
			if err != nil {
				return "", err
			}

			return joinConfigurationYAML, nil
		}
	} else {
		klog.Info("Joining a worker node to the cluster")

		joinConfigurationCopy := machineProviderSpec.KubeadmConfiguration.Join
		// Set default values. If they are not set, these two properties are added with an
		// empty string and the CoreOS ignition postprocesser fails with "could not find expected key"
		joinConfigurationCopy.CACertPath = "/etc/kubernetes/pki/ca.crt"
		kubeadm.SetJoinConfigurationOptions(
			&joinConfigurationCopy,
			kubeadm.WithBootstrapTokenDiscovery(
				kubeadm.NewBootstrapTokenDiscovery(
					kubeadm.WithAPIServerEndpoint(clusterProviderSpec.ClusterConfiguration.ControlPlaneEndpoint),
					kubeadm.WithToken(bootstrapToken),
					kubeadm.WithCACertificateHash(caCertHash),
				),
			),
			kubeadm.WithTLSBootstrapToken(bootstrapToken),
			kubeadm.WithJoinNodeRegistrationOptions(
				kubeadm.NewNodeRegistration(
					kubeadm.WithTaints(machine.Spec.Taints),
					kubeadm.WithKubeletExtraArgs(map[string]string{"cloud-provider": cloudProvider}),
					kubeadm.WithKubeletExtraArgs(map[string]string{"cloud-config": cloudProviderConfig}),
					kubeadm.WithKubeletExtraArgs(map[string]string{"node-labels": nodeRole}),
				),
			),
		)
		joinConfigurationYAML, err := kubeadm.ConfigurationToYAML(&joinConfigurationCopy)
		if err != nil {
			return "", err
		}

		return joinConfigurationYAML, nil
	}
}
