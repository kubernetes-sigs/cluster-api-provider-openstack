#!/bin/bash
set -e

echo "=== Cleaning up Security Groups ==="
openstack security group list -f value -c ID -c Name | grep -E "(hcp|k8s-cluster.*e2e)" | awk '{print $1}' | xargs -I {} openstack security group delete {}

echo "=== Cleaning up Load Balancers ==="
openstack loadbalancer list -f value -c id -c name | grep -E "(hcp|e2e)" | awk '{print $1}' | xargs -I {} openstack loadbalancer delete {} || true

echo "=== Cleaning up Routers ==="
openstack router list -f value -c ID -c Name | grep -E "(hcp|e2e)" | while read router_id router_name; do
  echo "Cleaning router: $router_name ($router_id)"
  # Remove external gateway
  openstack router unset --external-gateway $router_id 2>/dev/null || true
  # Remove all ports
  openstack port list --router $router_id -f value -c ID | xargs -I {} openstack router remove port $router_id {} 2>/dev/null || true
  # Delete router
  openstack router delete $router_id
done

echo "=== Cleaning up Subnets ==="
openstack subnet list -f value -c ID -c Name | grep -E "(hcp|e2e)" | awk '{print $1}' | xargs -I {} openstack subnet delete {}

echo "=== Cleaning up Networks ==="
openstack network list -f value -c ID -c Name | grep -E "(hcp|e2e)" | awk '{print $1}' | xargs -I {} openstack network delete {}

echo "=== Cleaning up Floating IPs ==="
openstack floating ip list -f value -c ID -c Description | grep -E "(hcp|e2e|cluster)" | awk '{print $1}' | xargs -I {} openstack floating ip delete {} || true

echo "=== Cleanup Complete ==="
