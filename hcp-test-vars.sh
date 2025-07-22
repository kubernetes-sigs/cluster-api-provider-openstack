#!/bin/bash
# Complete environment variables for CAPO HCP testing
# Usage: source hcp-test-vars.sh && make test-hcp

# CRITICAL: Kubernetes version must match your image
# Check your ubuntu-22.04 image and set accordingly (probably v1.28.x, v1.29.x, or v1.30.x)
export KUBERNETES_VERSION="v1.30.2"  # CHANGE THIS to match your image's k8s version

# OpenStack Configuration
export OPENSTACK_CLOUD="openstack"
export OPENSTACK_CLOUD_ADMIN="openstack"  # Same as OPENSTACK_CLOUD for most setups
export OPENSTACK_CLOUD_YAML_FILE="/Users/bnr/work/openstack/clouds.yaml"

# Image Configuration - Use existing image with dummy URLs to pass validation
export OPENSTACK_IMAGE_NAME="ubuntu-2404-kube-v1.33.1"
export OPENSTACK_IMAGE_URL="file:///dev/null"  # Dummy URL - image already exists
export OPENSTACK_BASTION_IMAGE_NAME="ubuntu-2404-kube-v1.33.1"  # Use same image for bastion
export OPENSTACK_BASTION_IMAGE_URL="file:///dev/null"  # Dummy URL - image already exists

# Flavor Configuration - ADJUST based on your OpenStack
export OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR="m1.medium"
export OPENSTACK_NODE_MACHINE_FLAVOR="m1.small"
export OPENSTACK_BASTION_MACHINE_FLAVOR="m1.small"

# Network Configuration - ADJUST based on your OpenStack
export OPENSTACK_EXTERNAL_NETWORK_NAME="public"  # Check: openstack network list --external
export OPENSTACK_DNS_NAMESERVERS="8.8.8.8"  # or your preferred DNS
export OPENSTACK_FAILURE_DOMAIN="nova"  # Check: openstack availability zone list

# SSH Key - CRITICAL
export OPENSTACK_SSH_KEY_NAME="cluster-api-provider-openstack-sigs-k8s-io"  # Must exist in OpenStack

# HCP Specific Configuration
export KAMAJI_VERSION="v0.15.3"  # Stable version instead of edge
export KAMAJI_NAMESPACE="kamaji-system"
export CLUSTER_DATASTORE="default"
export HCP_SERVICE_TYPE="LoadBalancer"
export HCP_CPU_LIMIT="1000m"
export HCP_MEMORY_LIMIT="1Gi"
export HCP_CPU_REQUEST="100m"
export HCP_MEMORY_REQUEST="300Mi"

# Test Configuration
export E2E_GINKGO_FOCUS="Management cluster verification"

# Timeout adjustments for slower environments
export GINKGO_ARGS="-v --progress --timeout=45m"

echo "✅ Environment variables set for HCP testing"
echo "🔍 Key settings:"
echo "   Kubernetes Version: $KUBERNETES_VERSION"
echo "   Image Name: $OPENSTACK_IMAGE_NAME"
echo "   Control Plane Flavor: $OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR"
echo "   External Network: $OPENSTACK_EXTERNAL_NETWORK_NAME"
echo "   SSH Key: $OPENSTACK_SSH_KEY_NAME"
echo ""
echo "⚠️  VERIFY these match your OpenStack environment:"
echo "   1. Check image exists: openstack image show $OPENSTACK_IMAGE_NAME"
echo "   2. Check flavors exist: openstack flavor show $OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR"
echo "   3. Check SSH key exists: openstack keypair show $OPENSTACK_SSH_KEY_NAME"
echo "   4. Check external network: openstack network show $OPENSTACK_EXTERNAL_NETWORK_NAME"
echo ""
echo "🚀 Run: make test-hcp" 