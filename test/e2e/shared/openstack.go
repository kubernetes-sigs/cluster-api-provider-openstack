//go:build e2e
// +build e2e

/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shared

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/utils/openstack/clientconfig"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/ini.v1"
	"sigs.k8s.io/yaml"

	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha5"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/compute"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/services/provider"
)

type ServerExtWithIP struct {
	compute.ServerExt
	ip string
}

// ensureSSHKeyPair ensures A SSH key is present under the name.
func ensureSSHKeyPair(e2eCtx *E2EContext) {
	Byf("Ensuring presence of SSH key %q in OpenStack", DefaultSSHKeyPairName)

	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	Expect(err).NotTo(HaveOccurred())

	computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: clientOpts.RegionName})
	Expect(err).NotTo(HaveOccurred())

	keyPairCreateOpts := &keypairs.CreateOpts{
		Name: DefaultSSHKeyPairName,
	}
	keypair, err := keypairs.Create(computeClient, keyPairCreateOpts).Extract()
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return
		}
		Expect(err).NotTo(HaveOccurred())
	}

	sshDir := filepath.Join(e2eCtx.Settings.ArtifactFolder, "ssh")
	Byf("Storing keypair in %q", sshDir)

	err = os.MkdirAll(sshDir, 0o750)
	Expect(err).NotTo(HaveOccurred())

	err = os.WriteFile(filepath.Join(sshDir, DefaultSSHKeyPairName), []byte(keypair.PrivateKey), 0o600)
	Expect(err).NotTo(HaveOccurred())

	err = os.WriteFile(filepath.Join(sshDir, fmt.Sprintf("%s.pub", DefaultSSHKeyPairName)), []byte(keypair.PublicKey), 0o600)
	Expect(err).NotTo(HaveOccurred())
}

func dumpOpenStack(_ context.Context, e2eCtx *E2EContext, bootstrapClusterProxyName string) {
	Byf("Running dumpOpenStack")
	logPath := filepath.Join(e2eCtx.Settings.ArtifactFolder, "clusters", bootstrapClusterProxyName, "openstack-resources")
	if err := os.MkdirAll(logPath, os.ModePerm); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating directory %s: %s\n", logPath, err)
		return
	}
	_, _ = fmt.Fprintf(GinkgoWriter, "folder created for OpenStack clusters: %s\n", logPath)

	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return
	}

	if err := dumpOpenStackImages(providerClient, clientOpts, logPath); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error dumping OpenStack images: %s\n", err)
	}
}

func dumpOpenStackImages(providerClient *gophercloud.ProviderClient, clientOpts *clientconfig.ClientOpts, logPath string) error {
	imageClient, err := openstack.NewImageServiceV2(providerClient, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return fmt.Errorf("error creating compute client: %s", err)
	}

	allPages, err := images.List(imageClient, images.ListOpts{}).AllPages()
	if err != nil {
		return fmt.Errorf("error getting images: %s", err)
	}
	imagesList, err := images.ExtractImages(allPages)
	if err != nil {
		return fmt.Errorf("error extracting images: %s", err)
	}
	imagesJSON, err := json.MarshalIndent(imagesList, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshalling images %v: %s", imagesList, err)
	}
	if err := os.WriteFile(path.Join(logPath, "images.txt"), imagesJSON, 0o600); err != nil {
		return fmt.Errorf("error writing seversJSON %s: %s", imagesJSON, err)
	}
	return nil
}

func DumpOpenStackServers(e2eCtx *E2EContext, filter servers.ListOpts) ([]servers.Server, error) {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return nil, nil
	}

	computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: clientOpts.RegionName})
	if err != nil {
		return nil, fmt.Errorf("error creating compute client: %v", err)
	}

	computeClient.Microversion = compute.NovaMinimumMicroversion
	allPages, err := servers.List(computeClient, filter).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error listing servers: %v", err)
	}

	allServers, err := servers.ExtractServers(allPages)
	if err != nil {
		return nil, fmt.Errorf("error extracting server: %v", err)
	}

	return allServers, nil
}

func DumpOpenStackNetworks(e2eCtx *E2EContext, filter networks.ListOpts) ([]networks.Network, error) {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return nil, nil
	}

	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating network client: %s", err)
	}

	allPages, err := networks.List(networkClient, filter).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error listing networks: %s", err)
	}
	networksList, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return nil, fmt.Errorf("error extracting networks: %s", err)
	}
	return networksList, nil
}

func DumpOpenStackSubnets(e2eCtx *E2EContext, filter subnets.ListOpts) ([]subnets.Subnet, error) {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return nil, nil
	}

	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating network client: %s", err)
	}

	allPages, err := subnets.List(networkClient, filter).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error listing subnets: %s", err)
	}
	subnetsList, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		return nil, fmt.Errorf("error extracting subnets: %s", err)
	}
	return subnetsList, nil
}

func DumpOpenStackRouters(e2eCtx *E2EContext, filter routers.ListOpts) ([]routers.Router, error) {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return nil, nil
	}

	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating network client: %s", err)
	}

	allPages, err := routers.List(networkClient, filter).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error listing routers: %s", err)
	}
	routersList, err := routers.ExtractRouters(allPages)
	if err != nil {
		return nil, fmt.Errorf("error extracting routers: %s", err)
	}
	return routersList, nil
}

func DumpOpenStackSecurityGroups(e2eCtx *E2EContext, filter groups.ListOpts) ([]groups.SecGroup, error) {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return nil, nil
	}

	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating network client: %s", err)
	}

	allPages, err := groups.List(networkClient, filter).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error listing security groups: %s", err)
	}
	groupsList, err := groups.ExtractGroups(allPages)
	if err != nil {
		return nil, fmt.Errorf("error extracting security groups: %s", err)
	}
	return groupsList, nil
}

func DumpOpenStackPorts(e2eCtx *E2EContext, filter ports.ListOpts) ([]ports.Port, error) {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return nil, err
	}

	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating network client: %s", err)
	}

	allPages, err := ports.List(networkClient, filter).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error getting ports: %s", err)
	}
	portsList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return nil, fmt.Errorf("error extracting ports: %s", err)
	}
	return portsList, nil
}

// DumpOpenStackTrunks trunks for a given port.
func DumpOpenStackTrunks(e2eCtx *E2EContext, portID string) (*trunks.Trunk, error) {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return nil, err
	}
	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating network client: %s", err)
	}

	allPages, err := trunks.List(networkClient, trunks.ListOpts{
		PortID: portID,
	}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error getting trunks: %s", err)
	}
	trunkList, err := trunks.ExtractTrunks(allPages)
	if err != nil {
		return nil, fmt.Errorf("searching for existing trunk for port: %v", err)
	}

	if len(trunkList) != 0 {
		return &trunkList[0], nil
	}
	return nil, nil
}

// GetOpenStackServers gets all OpenStack servers at once, to save on DescribeInstances
// calls.
func GetOpenStackServers(e2eCtx *E2EContext, openStackCluster *infrav1.OpenStackCluster) (map[string]ServerExtWithIP, error) {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return nil, nil
	}

	computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: clientOpts.RegionName})
	if err != nil {
		return nil, fmt.Errorf("error creating compute client: %v", err)
	}

	serverListOpts := &servers.ListOpts{}
	allPages, err := servers.List(computeClient, serverListOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error listing server: %v", err)
	}

	var serverList []compute.ServerExt
	err = servers.ExtractServersInto(allPages, &serverList)
	if err != nil {
		return nil, fmt.Errorf("error extracting server: %v", err)
	}

	srvs := map[string]ServerExtWithIP{}
	for i := range serverList {
		srv := &serverList[i]
		instanceStatus := compute.NewInstanceStatusFromServer(srv, logr.Discard())
		instanceNS, err := instanceStatus.NetworkStatus()
		if err != nil {
			return nil, fmt.Errorf("error getting network status for server %s: %v", srv.Name, err)
		}

		ip := instanceNS.IP(openStackCluster.Status.Network.Name)
		if ip == "" {
			_, _ = fmt.Fprintf(GinkgoWriter, "error getting internal ip for server %s: internal ip doesn't exist (yet)\n", srv.Name)
			continue
		}

		srvs[srv.Name] = ServerExtWithIP{
			ServerExt: *srv,
			ip:        ip,
		}
	}
	return srvs, nil
}

func GetTenantProviderClient(e2eCtx *E2EContext) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, *string, error) {
	openstackCloud := e2eCtx.E2EConfig.GetVariable(OpenStackCloud)
	return getProviderClient(e2eCtx, openstackCloud)
}

func GetAdminProviderClient(e2eCtx *E2EContext) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, *string, error) {
	openstackCloud := e2eCtx.E2EConfig.GetVariable(OpenStackCloudAdmin)
	return getProviderClient(e2eCtx, openstackCloud)
}

func getProviderClient(e2eCtx *E2EContext, openstackCloud string) (*gophercloud.ProviderClient, *clientconfig.ClientOpts, *string, error) {
	openStackCloudYAMLFile := e2eCtx.E2EConfig.GetVariable(OpenStackCloudYAMLFile)

	clouds := getParsedOpenStackCloudYAML(openStackCloudYAMLFile)
	cloud := clouds.Clouds[openstackCloud]

	providerClient, clientOpts, projectID, err := provider.NewClient(cloud, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	return providerClient, clientOpts, &projectID, nil
}

// Config is used to read and store information from the cloud configuration file
// Depends on: /home/sbuerin/code/src/k8s.io/cloud-provider-openstack/pkg/cloudprovider/providers/openstack/openstack.go.
type Config struct {
	Global AuthOpts
}

type AuthOpts struct {
	AuthURL    string `ini:"auth-url"`
	UserID     string `ini:"user-id"`
	Username   string `ini:"username"`
	Password   string `ini:"password"`
	TenantID   string `ini:"tenant-id"`
	TenantName string `ini:"tenant-name"`
	DomainID   string `ini:"domain-id"`
	DomainName string `ini:"domain-name"`

	// In-tree cloud provider will fail to start if these are present
	// TenantDomainID   string `ini:"tenant-domain-id"`
	// TenantDomainName string `ini:"tenant-domain-name"`
	// UserDomainID     string `ini:"user-domain-id"`
	// UserDomainName   string `ini:"user-domain-name"`
	Region string `ini:"region"`
	CAFile string `ini:"ca-file"`
	// TLSInsecure      string `ini:"tls-insecure"`

	// CloudsFile string `ini:"clouds-file"`
	// Cloud      string `ini:"cloud"`

	// ApplicationCredentialID   string `ini:"application-credential-id"`
	// ApplicationCredentialName string `ini:"application-credential-name"`
}

func getEncodedOpenStackCloudYAML(cloudYAML string) string {
	cloudYAMLContent := getOpenStackCloudYAML(cloudYAML)
	return base64.StdEncoding.EncodeToString(cloudYAMLContent)
}

func getEncodedOpenStackCloudProviderConf(cloudYAML, cloudName string) string {
	clouds := getParsedOpenStackCloudYAML(cloudYAML)
	cloud := clouds.Clouds[cloudName]

	authopts := AuthOpts{
		AuthURL:    cloud.AuthInfo.AuthURL,
		UserID:     cloud.AuthInfo.UserID,
		Username:   cloud.AuthInfo.Username,
		Password:   cloud.AuthInfo.Password,
		TenantID:   cloud.AuthInfo.ProjectID,
		TenantName: cloud.AuthInfo.ProjectName,
		Region:     cloud.RegionName,
	}

	// In-tree OpenStack cloud provider does not support
	// {Tenant,User}Domain{ID,Name}, but external cloud provider does.
	// Here we manually set Domain{ID,Name} depending on the most specific config available
	switch {
	case cloud.AuthInfo.UserDomainID != "":
		authopts.DomainID = cloud.AuthInfo.UserDomainID
	case cloud.AuthInfo.UserDomainName != "":
		authopts.DomainName = cloud.AuthInfo.UserDomainName
	case cloud.AuthInfo.ProjectDomainID != "":
		authopts.DomainID = cloud.AuthInfo.UserDomainID
	case cloud.AuthInfo.ProjectDomainName != "":
		authopts.DomainName = cloud.AuthInfo.ProjectDomainName
	case cloud.AuthInfo.DomainID != "":
		authopts.DomainID = cloud.AuthInfo.DomainID
	case cloud.AuthInfo.DomainName != "":
		authopts.DomainName = cloud.AuthInfo.DomainName
	}

	// Regardless of the path to a CA cert specified in the input
	// clouds.yaml, we will deploy the cert to /etc/certs/cacert in the
	// target cluster as specified in KubeadmControlPlane and KubeadmConfig
	// for the control plane and workers respectively in the E2E cluster templates.
	if cloud.CACertFile != "" {
		authopts.CAFile = "/etc/certs/cacert"
	}

	cloudProviderConf := &Config{
		Global: authopts,
	}

	cfg := ini.Empty()
	err := ini.ReflectFrom(cfg, cloudProviderConf)
	Expect(err).NotTo(HaveOccurred())

	var buf bytes.Buffer
	_, err = cfg.WriteTo(&buf)
	Expect(err).NotTo(HaveOccurred())

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func getParsedOpenStackCloudYAML(cloudYAML string) clientconfig.Clouds {
	cloudYAMLContent := getOpenStackCloudYAML(cloudYAML)

	var clouds clientconfig.Clouds
	err := yaml.Unmarshal(cloudYAMLContent, &clouds)
	Expect(err).NotTo(HaveOccurred())
	return clouds
}

func getOpenStackCloudYAML(cloudYAML string) []byte {
	cloudYAMLContent, err := os.ReadFile(cloudYAML)
	Expect(err).NotTo(HaveOccurred())
	return cloudYAMLContent
}

func GetComputeAvailabilityZones(e2eCtx *E2EContext) []string {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	Expect(err).NotTo(HaveOccurred())

	computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: clientOpts.RegionName})
	Expect(err).NotTo(HaveOccurred())

	allPages, err := availabilityzones.List(computeClient).AllPages()
	Expect(err).NotTo(HaveOccurred())
	availabilityZoneList, err := availabilityzones.ExtractAvailabilityZones(allPages)
	Expect(err).NotTo(HaveOccurred())

	var azs []string
	for _, az := range availabilityZoneList {
		if az.ZoneState.Available {
			azs = append(azs, az.ZoneName)
		}
	}

	return azs
}

func CreateOpenStackNetwork(e2eCtx *E2EContext, name, cidr string) (*networks.Network, error) {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return nil, err
	}

	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating network client: %s", err)
	}

	netCreateOpts := networks.CreateOpts{
		Name:         name,
		AdminStateUp: gophercloud.Enabled,
	}
	net, err := networks.Create(networkClient, netCreateOpts).Extract()
	if err != nil {
		return net, err
	}

	subnetCreateOpts := subnets.CreateOpts{
		Name:      name,
		NetworkID: net.ID,
		IPVersion: 4,
		CIDR:      cidr,
	}
	_, err = subnets.Create(networkClient, subnetCreateOpts).Extract()
	if err != nil {
		networks.Delete(networkClient, net.ID)
		return nil, err
	}
	return net, nil
}

func DeleteOpenStackNetwork(e2eCtx *E2EContext, id string) error {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return err
	}

	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return fmt.Errorf("error creating network client: %s", err)
	}

	return networks.Delete(networkClient, id).ExtractErr()
}

func GetOpenStackVolume(e2eCtx *E2EContext, name string) (*volumes.Volume, error) {
	providerClient, clientOpts, _, err := GetTenantProviderClient(e2eCtx)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "error creating provider client: %s\n", err)
		return nil, err
	}

	volumeClient, err := openstack.NewBlockStorageV3(providerClient, gophercloud.EndpointOpts{
		Region: clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating volume client: %s", err)
	}

	volume, err := volumes.Get(volumeClient, name).Extract()
	if err != nil {
		return volume, err
	}

	return volume, nil
}
