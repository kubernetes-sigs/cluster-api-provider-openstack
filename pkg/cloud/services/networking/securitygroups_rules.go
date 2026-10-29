/*
Copyright 2022 The Kubernetes Authors.

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

package networking

import (
	"k8s.io/utils/net"
)

const (
	securityGroupRuleDirectionIngress            = "ingress"
	securityGroupRuleDirectionEgress             = "egress"
	securityGroupRuleEtherTypeIPv4               = "IPv4"
	securityGroupRuleEtherTypeIPv6               = "IPv6"
	securityGroupRuleProtocolTCP                 = "tcp"
	securityGroupRuleProtocolUDP                 = "udp"
	securityGroupRuleDescriptionSSH              = "SSH"
	securityGroupRuleDescriptionKubeletAPI       = "Kubelet API"
	securityGroupRuleDescriptionNodePortServices = "Node Port Services"
	securityGroupRuleDescriptionInClusterIngress = "In-cluster Ingress"
	securityGroupRuleDescriptionInClusterEgress  = "In-cluster Egress"
	securityGroupRuleDescriptionKubernetesAPI    = "Kubernetes API"
)

var defaultRules = []resolvedSecurityGroupRuleSpec{
	{
		Direction:      securityGroupRuleDirectionEgress,
		Description:    "Full open",
		EtherType:      securityGroupRuleEtherTypeIPv4,
		PortRangeMin:   0,
		PortRangeMax:   0,
		Protocol:       "",
		RemoteIPPrefix: "",
	},
	{
		Direction:      securityGroupRuleDirectionEgress,
		Description:    "Full open",
		EtherType:      securityGroupRuleEtherTypeIPv6,
		PortRangeMin:   0,
		PortRangeMax:   0,
		Protocol:       "",
		RemoteIPPrefix: "",
	},
}

// Permit traffic for etcd, kubelet.
func getSGControlPlaneCommon(remoteGroupIDSelf, secWorkerGroupID string) []resolvedSecurityGroupRuleSpec {
	return []resolvedSecurityGroupRuleSpec{
		{
			Description:   "Etcd",
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  2379,
			PortRangeMax:  2380,
			Protocol:      securityGroupRuleProtocolTCP,
			RemoteGroupID: remoteGroupIDSelf,
		},
		{
			// kubeadm says this is needed
			Description:   securityGroupRuleDescriptionKubeletAPI,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  10250,
			PortRangeMax:  10250,
			Protocol:      securityGroupRuleProtocolTCP,
			RemoteGroupID: remoteGroupIDSelf,
		},
		{
			// This is needed to support metrics-server deployments
			Description:   securityGroupRuleDescriptionKubeletAPI,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  10250,
			PortRangeMax:  10250,
			Protocol:      securityGroupRuleProtocolTCP,
			RemoteGroupID: secWorkerGroupID,
		},
	}
}

// Permit traffic for kubelet.
func getSGWorkerCommon(remoteGroupIDSelf, secControlPlaneGroupID string) []resolvedSecurityGroupRuleSpec {
	return []resolvedSecurityGroupRuleSpec{
		{
			// This is needed to support metrics-server deployments
			Description:   securityGroupRuleDescriptionKubeletAPI,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  10250,
			PortRangeMax:  10250,
			Protocol:      securityGroupRuleProtocolTCP,
			RemoteGroupID: remoteGroupIDSelf,
		},
		{
			Description:   securityGroupRuleDescriptionKubeletAPI,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  10250,
			PortRangeMax:  10250,
			Protocol:      securityGroupRuleProtocolTCP,
			RemoteGroupID: secControlPlaneGroupID,
		},
	}
}

// Permit traffic for ssh control plane.
func getSGControlPlaneSSH(secBastionGroupID string) []resolvedSecurityGroupRuleSpec {
	return []resolvedSecurityGroupRuleSpec{
		{
			Description:   securityGroupRuleDescriptionSSH,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  22,
			PortRangeMax:  22,
			Protocol:      securityGroupRuleProtocolTCP,
			RemoteGroupID: secBastionGroupID,
		},
	}
}

// Permit traffic for ssh worker.
func getSGWorkerSSH(secBastionGroupID string) []resolvedSecurityGroupRuleSpec {
	return []resolvedSecurityGroupRuleSpec{
		{
			Description:   securityGroupRuleDescriptionSSH,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  22,
			PortRangeMax:  22,
			Protocol:      securityGroupRuleProtocolTCP,
			RemoteGroupID: secBastionGroupID,
		},
	}
}

// Allow all traffic, including from outside the cluster, to access the API.
func getSGControlPlaneHTTPS() []resolvedSecurityGroupRuleSpec {
	return []resolvedSecurityGroupRuleSpec{
		{
			Description:  securityGroupRuleDescriptionKubernetesAPI,
			Direction:    securityGroupRuleDirectionIngress,
			EtherType:    securityGroupRuleEtherTypeIPv4,
			PortRangeMin: 6443,
			PortRangeMax: 6443,
			Protocol:     securityGroupRuleProtocolTCP,
		},
	}
}

// Allow all traffic, including from outside the cluster, to access node port services.
func getSGWorkerNodePort(secWorkerGroupID string, secControlPlaneGroupID string) []resolvedSecurityGroupRuleSpec {
	return []resolvedSecurityGroupRuleSpec{
		{
			Description:   securityGroupRuleDescriptionNodePortServices,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  30000,
			PortRangeMax:  32767,
			Protocol:      securityGroupRuleProtocolTCP,
			RemoteGroupID: secWorkerGroupID,
		},
		{
			Description:   securityGroupRuleDescriptionNodePortServices,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  30000,
			PortRangeMax:  32767,
			Protocol:      securityGroupRuleProtocolUDP,
			RemoteGroupID: secWorkerGroupID,
		},
		{
			Description:   securityGroupRuleDescriptionNodePortServices,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  30000,
			PortRangeMax:  32767,
			Protocol:      securityGroupRuleProtocolTCP,
			RemoteGroupID: secControlPlaneGroupID,
		},
		{
			Description:   securityGroupRuleDescriptionNodePortServices,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  30000,
			PortRangeMax:  32767,
			Protocol:      securityGroupRuleProtocolUDP,
			RemoteGroupID: secControlPlaneGroupID,
		},
	}
}

// Allow all traffic from a specific CIDR to access node port services.
func getSGWorkerNodePortCIDR(cidr string) []resolvedSecurityGroupRuleSpec {
	return []resolvedSecurityGroupRuleSpec{
		{
			Description:    securityGroupRuleDescriptionNodePortServices,
			Direction:      securityGroupRuleDirectionIngress,
			EtherType:      securityGroupRuleEtherTypeIPv4,
			PortRangeMin:   30000,
			PortRangeMax:   32767,
			Protocol:       securityGroupRuleProtocolTCP,
			RemoteIPPrefix: cidr,
		},
		{
			Description:    securityGroupRuleDescriptionNodePortServices,
			Direction:      securityGroupRuleDirectionIngress,
			EtherType:      securityGroupRuleEtherTypeIPv4,
			PortRangeMin:   30000,
			PortRangeMax:   32767,
			Protocol:       securityGroupRuleProtocolUDP,
			RemoteIPPrefix: cidr,
		},
	}
}

// Permit all ingress from the cluster security groups.
func getSGControlPlaneAllowAll(remoteGroupIDSelf, secWorkerGroupID string) []resolvedSecurityGroupRuleSpec {
	return []resolvedSecurityGroupRuleSpec{
		{
			Description:   securityGroupRuleDescriptionInClusterIngress,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  0,
			PortRangeMax:  0,
			Protocol:      "",
			RemoteGroupID: remoteGroupIDSelf,
		},
		{
			Description:   securityGroupRuleDescriptionInClusterIngress,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  0,
			PortRangeMax:  0,
			Protocol:      "",
			RemoteGroupID: secWorkerGroupID,
		},
	}
}

// Permit all ingress from the cluster security groups.
func getSGWorkerAllowAll(remoteGroupIDSelf, secControlPlaneGroupID string) []resolvedSecurityGroupRuleSpec {
	return []resolvedSecurityGroupRuleSpec{
		{
			Description:   securityGroupRuleDescriptionInClusterIngress,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  0,
			PortRangeMax:  0,
			Protocol:      "",
			RemoteGroupID: remoteGroupIDSelf,
		},
		{
			Description:   securityGroupRuleDescriptionInClusterIngress,
			Direction:     securityGroupRuleDirectionIngress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  0,
			PortRangeMax:  0,
			Protocol:      "",
			RemoteGroupID: secControlPlaneGroupID,
		},
	}
}

// Permit all egress to the given security groups, for both IPv4 and IPv6.
// Used instead of defaultRules when allow-all outbound traffic is disabled.
func getSGInClusterEgress(remoteGroupIDs ...string) []resolvedSecurityGroupRuleSpec {
	r := make([]resolvedSecurityGroupRuleSpec, 0, 2*len(remoteGroupIDs))
	for _, remoteGroupID := range remoteGroupIDs {
		for _, etherType := range []string{securityGroupRuleEtherTypeIPv4, securityGroupRuleEtherTypeIPv6} {
			r = append(r, resolvedSecurityGroupRuleSpec{
				Description:   securityGroupRuleDescriptionInClusterEgress,
				Direction:     securityGroupRuleDirectionEgress,
				EtherType:     etherType,
				PortRangeMin:  0,
				PortRangeMax:  0,
				Protocol:      "",
				RemoteGroupID: remoteGroupID,
			})
		}
	}
	return r
}

// Permit egress to the API server endpoint. The ether type matches the address family of the endpoint.
// No rule is returned when the endpoint is not an IP address.
func getSGAPIServerEgress(host string, port int) []resolvedSecurityGroupRuleSpec {
	ip := net.ParseIPSloppy(host)
	if ip == nil {
		return nil
	}

	etherType, remoteIPPrefix := securityGroupRuleEtherTypeIPv6, ip.String()+"/128"
	if ip.To4() != nil {
		etherType, remoteIPPrefix = securityGroupRuleEtherTypeIPv4, ip.String()+"/32"
	}

	return []resolvedSecurityGroupRuleSpec{
		{
			Description:    securityGroupRuleDescriptionKubernetesAPI,
			Direction:      securityGroupRuleDirectionEgress,
			EtherType:      etherType,
			PortRangeMin:   port,
			PortRangeMax:   port,
			Protocol:       securityGroupRuleProtocolTCP,
			RemoteIPPrefix: remoteIPPrefix,
		},
	}
}

// Permit ssh from the bastion to the control plane and worker nodes.
func getSGBastionSSHEgress(secControlPlaneGroupID, secWorkerGroupID string) []resolvedSecurityGroupRuleSpec {
	return []resolvedSecurityGroupRuleSpec{
		{
			Description:   securityGroupRuleDescriptionSSH,
			Direction:     securityGroupRuleDirectionEgress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  22,
			PortRangeMax:  22,
			Protocol:      securityGroupRuleProtocolTCP,
			RemoteGroupID: secControlPlaneGroupID,
		},
		{
			Description:   securityGroupRuleDescriptionSSH,
			Direction:     securityGroupRuleDirectionEgress,
			EtherType:     securityGroupRuleEtherTypeIPv4,
			PortRangeMin:  22,
			PortRangeMax:  22,
			Protocol:      securityGroupRuleProtocolTCP,
			RemoteGroupID: secWorkerGroupID,
		},
	}
}

// Permit ports that defined in openStackCluster.Spec.APIServerLoadBalancer.AdditionalPorts.
func getSGControlPlaneAdditionalPorts(ports []int32) []resolvedSecurityGroupRuleSpec {
	// Preallocate r with len(ports)
	r := make([]resolvedSecurityGroupRuleSpec, len(ports))
	for i, p := range ports {
		r[i] = resolvedSecurityGroupRuleSpec{
			Description:  "Additional port",
			Direction:    securityGroupRuleDirectionIngress,
			EtherType:    securityGroupRuleEtherTypeIPv4,
			Protocol:     securityGroupRuleProtocolTCP,
			PortRangeMin: int(p),
			PortRangeMax: int(p),
		}
	}
	return r
}

func getSGControlPlaneGeneral(remoteGroupIDSelf, secWorkerGroupID string) []resolvedSecurityGroupRuleSpec {
	return getSGControlPlaneCommon(remoteGroupIDSelf, secWorkerGroupID)
}

func getSGWorkerGeneral(remoteGroupIDSelf, secControlPlaneGroupID string) []resolvedSecurityGroupRuleSpec {
	return getSGWorkerCommon(remoteGroupIDSelf, secControlPlaneGroupID)
}
