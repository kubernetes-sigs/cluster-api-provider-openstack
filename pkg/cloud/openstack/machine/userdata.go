package machine

import (
	"fmt"

	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/machine/userdata"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/machine/userdata/cloudinit"
)

// NewUserData returns a new instance of type implementing userdata.UserData
func NewUserData(distri string) (userdata.UserData, error) {
	switch distri {
	case "ubuntu", "centos":
		return &cloudinit.CloudInit{
			Distribution: distri,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported distribution: %s", distri)
	}
}
