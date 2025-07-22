#!/bin/bash
# Proper OpenStack cleanup script - deletes in correct dependency order

export OS_CLOUD=openstack
export OS_CLOUD_YAML_FILE=/Users/bnr/work/openstack/clouds.yaml

echo "🧹 Starting proper OpenStack cleanup (dependency order)..."

# 1. First, delete servers (instances)
echo "=== Step 1: Deleting Servers ==="
openstack server list -f value -c ID -c Name | grep -E "(hcp|e2e|cluster-api)" | while read id name; do
    echo "Deleting server: $name ($id)"
    openstack server delete "$id" || true
done

# Wait a bit for servers to be deleted
echo "Waiting for servers to be deleted..."
sleep 10

# 2. Delete floating IPs
echo "=== Step 2: Deleting Floating IPs ==="
openstack floating ip list -f value -c ID | xargs -r -I {} bash -c 'echo "Deleting floating IP: {}"; openstack floating ip delete {} || true'

# 3. Delete load balancers (if any)
echo "=== Step 3: Deleting Load Balancers ==="
openstack loadbalancer list -f value -c id -c name | grep -E "(hcp|e2e|cluster-api)" | while read id name; do
    echo "Deleting loadbalancer: $name ($id)"
    openstack loadbalancer delete "$id" --cascade || true
done

# Wait for load balancers to be deleted
echo "Waiting for load balancers to be deleted..."
sleep 15

# 4. Delete router interfaces and routers
echo "=== Step 4: Deleting Routers ==="
openstack router list -f value -c ID -c Name | grep -E "(hcp|e2e|cluster-api)" | while read id name; do
    echo "Processing router: $name ($id)"
    
    # First remove all interfaces from the router
    echo "  Removing interfaces from router $name"
    openstack port list --router "$id" -f value -c ID | while read port_id; do
        echo "    Removing interface $port_id"
        openstack router remove port "$id" "$port_id" || true
    done
    
    # Then delete the router
    echo "  Deleting router $name"
    openstack router delete "$id" || true
done

# 5. Delete ports
echo "=== Step 5: Deleting Ports ==="
openstack port list -f value -c ID -c Name | grep -E "(hcp|e2e|cluster-api)" | while read id name; do
    echo "Deleting port: $name ($id)"
    openstack port delete "$id" || true
done

# 6. Delete subnets
echo "=== Step 6: Deleting Subnets ==="
openstack subnet list -f value -c ID -c Name | grep -E "(hcp|e2e|cluster-api)" | while read id name; do
    echo "Deleting subnet: $name ($id)"
    openstack subnet delete "$id" || true
done

# 7. Finally, delete networks
echo "=== Step 7: Deleting Networks ==="
openstack network list -f value -c ID -c Name | grep -E "(hcp|e2e|cluster-api)" | while read id name; do
    echo "Deleting network: $name ($id)"
    openstack network delete "$id" || true
done

# 8. Delete security groups
echo "=== Step 8: Deleting Security Groups ==="
openstack security group list -f value -c ID -c Name | grep -E "(hcp|e2e|cluster-api)" | while read id name; do
    echo "Deleting security group: $name ($id)"
    openstack security group delete "$id" || true
done

# 9. Delete keypairs
echo "=== Step 9: Deleting Keypairs ==="
openstack keypair list -f value -c Name | grep -E "(hcp|e2e|cluster-api)" | while read name; do
    echo "Deleting keypair: $name"
    openstack keypair delete "$name" || true
done

# 10. Delete volumes
echo "=== Step 10: Deleting Volumes ==="
openstack volume list --status available -f value -c ID -c Name | grep -E "(hcp|e2e|cluster-api)" | while read id name; do
    echo "Deleting volume: $name ($id)"
    openstack volume delete "$id" || true
done

echo "✅ Cleanup completed!"
echo ""
echo "Verification commands:"
echo "openstack server list"
echo "openstack network list | grep -E '(hcp|e2e|cluster-api)'"
echo "openstack router list | grep -E '(hcp|e2e|cluster-api)'"
echo "openstack security group list | grep -E '(hcp|e2e|cluster-api)'" 