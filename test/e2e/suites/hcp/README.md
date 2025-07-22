# HCP (Hosted Control Plane) E2E Tests

This directory contains end-to-end tests for Hosted Control Plane (HCP) functionality using Kamaji as the control plane provider.

## Overview

The HCP tests verify that:

1. **Management Cluster**: A traditional CAPO cluster can be configured to host control planes for other clusters
2. **Workload Cluster**: Worker-only clusters can be created with external control planes managed by Kamaji
3. **Graceful Failures**: Broken configurations fail gracefully without panics or nil pointer exceptions

## Test Structure

### Test Suites

- **Management Cluster Test**: Creates a CAPO cluster with HCP hosting capabilities
- **Workload Cluster Test**: Creates worker-only clusters using KamajiControlPlane  
- **Graceful Failure Tests**: Validates error handling for broken configurations

### Test Flow

1. **Suite Setup**: Creates shared management cluster with Kamaji installed
2. **Workload Tests**: Creates multiple workload clusters using the shared management cluster
3. **Failure Tests**: Tests broken scenarios to ensure graceful error handling
4. **Suite Cleanup**: Cleans up shared resources

## Running Tests

### Prerequisites

- OpenStack environment configured
- `OPENSTACK_CLOUD_YAML_FILE` environment variable set
- Docker installed ([[memory:2673423]])

### Run HCP Tests

```bash
# Run all HCP tests
make test-hcp

# Run with specific focus
E2E_GINKGO_FOCUS="Management cluster" make test-hcp

# Run with existing cluster
E2E_ARGS="-use-existing-cluster=true" make test-hcp
```

### Test Configuration

Tests use the same configuration as other e2e tests:
- Config: `test/e2e/data/e2e_conf.yaml` 
- Templates: `test/e2e/data/kustomize/hcp-*`
- Artifacts: `_artifacts/` directory

## Template Structure

### HCP Management (`hcp-management`)
- Traditional CAPO cluster with larger worker nodes
- Additional security rules for hosting control planes
- Kamaji installation and configuration

### HCP Workload (`hcp-workload`) 
- Worker-only cluster configuration
- Uses `KamajiControlPlane` instead of `KubeadmControlPlane`
- Different network CIDRs to avoid conflicts

### HCP Broken (`hcp-broken`)
- Intentionally broken networking configuration
- Used to test graceful failure scenarios

## Test Intervals

HCP tests use dedicated intervals defined in `e2e_conf.yaml`:

```yaml
intervals:
  hcp/wait-kamaji-install: ["10m", "30s"]
  hcp/wait-kamaji-control-plane: ["15m", "30s"] 
  hcp/wait-cluster: ["25m", "10s"]
  hcp/wait-control-plane: ["30m", "10s"]
  hcp/wait-worker-nodes: ["30m", "10s"]
```

## Debugging

### Log Collection
Logs are automatically collected in `_artifacts/clusters/` for failed tests.

### Manual Debug
Use the `skip-cleanup` flag to preserve resources for investigation:

```bash
E2E_ARGS="-skip-cleanup=true" make test-hcp
```

### Kamaji Resources
Check Kamaji-specific resources:

```bash
kubectl get kamajicontrolplane -A
kubectl get datastore -A  
kubectl get -n kamaji-system pods
``` 