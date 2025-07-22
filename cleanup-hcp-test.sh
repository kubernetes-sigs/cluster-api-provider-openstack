#!/bin/bash
# Cleanup script for failed HCP test resources

# Set OpenStack environment
export OS_CLOUD=openstack
export OS_CLOUD_YAML_FILE=/Users/bnr/work/openstack/clouds.yaml

echo "🧹 Cleaning up HCP test resources..."

# Delete the specific instances from your screenshot
echo "Deleting OpenStack instances..."
openstack server delete hcp-mgmt-hcp-mgmt-hcp-1752250951-al4m23-bastion
openstack server delete hcp-mgmt-hcp-mgmt-hcp-1752250951-al4m23-control-plane-sinpp5-bastion

# Clean up any floating IPs that might be allocated
echo "Cleaning up floating IPs..."
openstack floating ip list --status DOWN -f value -c ID | xargs -r openstack floating ip delete

# Clean up security groups (if any were created)
echo "Cleaning up security groups..."
openstack security group list --project $(openstack token issue -c project_id -f value) | grep "hcp-mgmt\|cluster-api" | awk '{print $2}' | xargs -r openstack security group delete

# Clean up keypairs (if any were created)
echo "Cleaning up keypairs..."
openstack keypair list | grep "cluster-api-provider-openstack-sigs-k8s-io" | awk '{print $2}' | xargs -r openstack keypair delete

# Clean up networks and subnets (if any were created)
echo "Cleaning up networks..."
openstack network list | grep "hcp-mgmt\|cluster-api" | awk '{print $2}' | xargs -r openstack network delete

# Clean up any volumes
echo "Cleaning up volumes..."
openstack volume list --status available | grep "hcp-mgmt\|cluster-api" | awk '{print $2}' | xargs -r openstack volume delete

echo "✅ Cleanup completed!"
echo ""
echo "Manual verification commands:"
echo "openstack server list"
echo "openstack floating ip list"
echo "openstack security group list"
echo "openstack network list" 