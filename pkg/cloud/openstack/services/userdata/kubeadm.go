package userdata

import (
	"fmt"
	"k8s.io/klog"
	kubeadmv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha2"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/certificates"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/services/kubeadm"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha2"
)

const (
	localIPV4Lookup = "${OPENSTACK_IPV4_LOCAL}"

	cloudProvider       = "openstack"
	cloudProviderConfig = "/etc/kubernetes/cloud.conf"

	nodeRole = "node-role.kubernetes.io/node="
)

func generateKubeadmConfig(isControlPlane bool, bootstrapToken string, machine *clusterv1.Machine, openStackMachine *infrav1.OpenStackMachine, cluster *clusterv1.Cluster, openStackCluster *infrav1.OpenStackCluster) (string, error) {

	caCertHash, err := certificates.GenerateCertificateHash(openStackCluster.Spec.CAKeyPair.Cert)
	if err != nil {
		return "", err
	}

	if bootstrapToken != "" && len(cluster.Status.APIEndpoints) == 0 {
		return "", fmt.Errorf("no APIEndpoint set yet, waiting until APIEndpoints is set")
	}

	if isControlPlane {
		if bootstrapToken == "" {
			klog.Info("Machine is the first control plane machine for the cluster")

			if !openStackCluster.Spec.CAKeyPair.HasCertAndKey() {
				return "", fmt.Errorf("failed to create controlplane kubeadm config, missing CAKeyPair")
			}

			clusterConfigurationCopy := openStackCluster.Spec.ClusterConfiguration.DeepCopy()
			// Set default values. If they are not set, these two properties are added with an
			// empty string and the CoreOS ignition postprocesser fails with "could not find expected key"
			if clusterConfigurationCopy.CertificatesDir == "" {
				clusterConfigurationCopy.CertificatesDir = kubeadmv1beta1.DefaultCertificatesDir
			}
			if clusterConfigurationCopy.ImageRepository == "" {
				clusterConfigurationCopy.ImageRepository = kubeadmv1beta1.DefaultImageRepository
			}
			if string(clusterConfigurationCopy.DNS.Type) == "" {
				clusterConfigurationCopy.DNS.Type = kubeadmv1beta1.CoreDNS
			}
			if clusterConfigurationCopy.ImageRepository == "" {
				clusterConfigurationCopy.ImageRepository = kubeadmv1beta1.DefaultImageRepository
			}
			kubeadm.SetClusterConfigurationOptions(
				clusterConfigurationCopy,
				kubeadm.WithKubernetesVersion(*machine.Spec.Version),
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
				kubeadm.WithKubernetesVersion(*machine.Spec.Version),
			)
			clusterConfigYAML, err := kubeadm.ConfigurationToYAML(clusterConfigurationCopy)
			if err != nil {
				return "", err
			}

			kubeadm.SetInitConfigurationOptions(
				&openStackMachine.Spec.KubeadmConfiguration.Init,
				kubeadm.WithNodeRegistrationOptions(
					kubeadm.NewNodeRegistration(
						kubeadm.WithKubeletExtraArgs(map[string]string{"cloud-provider": cloudProvider}),
						kubeadm.WithKubeletExtraArgs(map[string]string{"cloud-config": cloudProviderConfig}),
					),
				),
				kubeadm.WithInitLocalAPIEndpointAndPort(localIPV4Lookup, cluster.Status.APIEndpoints[0].Port),
			)
			initConfigYAML, err := kubeadm.ConfigurationToYAML(&openStackMachine.Spec.KubeadmConfiguration.Init)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s\n---\n%s", clusterConfigYAML, initConfigYAML), nil

		} else {
			klog.Info("Allowing a machine to join the control plane")

			joinConfigurationCopy := openStackMachine.Spec.KubeadmConfiguration.Join
			// Set default values. If they are not set, these two properties are added with an
			// empty string and the CoreOS ignition postprocesser fails with "could not find expected key"
			joinConfigurationCopy.CACertPath = kubeadmv1beta1.DefaultCACertPath
			kubeadm.SetJoinConfigurationOptions(
				&joinConfigurationCopy,
				kubeadm.WithBootstrapTokenDiscovery(
					kubeadm.NewBootstrapTokenDiscovery(
						kubeadm.WithAPIServerEndpoint(openStackCluster.Spec.ClusterConfiguration.ControlPlaneEndpoint),
						kubeadm.WithToken(bootstrapToken),
						kubeadm.WithCACertificateHash(caCertHash),
					),
				),
				kubeadm.WithTLSBootstrapToken(bootstrapToken),
				kubeadm.WithJoinNodeRegistrationOptions(
					kubeadm.NewNodeRegistration(
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

		joinConfigurationCopy := openStackMachine.Spec.KubeadmConfiguration.Join
		// Set default values. If they are not set, these two properties are added with an
		// empty string and the CoreOS ignition postprocesser fails with "could not find expected key"
		joinConfigurationCopy.CACertPath = kubeadmv1beta1.DefaultCACertPath
		kubeadm.SetJoinConfigurationOptions(
			&joinConfigurationCopy,
			kubeadm.WithBootstrapTokenDiscovery(
				kubeadm.NewBootstrapTokenDiscovery(
					kubeadm.WithAPIServerEndpoint(openStackCluster.Spec.ClusterConfiguration.ControlPlaneEndpoint),
					kubeadm.WithToken(bootstrapToken),
					kubeadm.WithCACertificateHash(caCertHash),
				),
			),
			kubeadm.WithTLSBootstrapToken(bootstrapToken),
			kubeadm.WithJoinNodeRegistrationOptions(
				kubeadm.NewNodeRegistration(
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
