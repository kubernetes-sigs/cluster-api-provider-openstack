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

package e2e

import (
	"os"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

var (
	e2eCtx                *shared.E2EContext
	initialServers        []servers.Server
	initialNetworks       []networks.Network
	initialSecurityGroups []groups.SecGroup
	initialLoadBalancers  []loadbalancers.LoadBalancer
	initialVolumes        []volumes.Volume
	err                   error
)

func init() {
	e2eCtx = shared.NewE2EContext()
	shared.CreateDefaultFlags(e2eCtx)

	// Gophercloud will ignore any explicitly passed configuration if
	// OS_CLOUD is set. This will always cause this test to fail, as we use
	// at least 2 cloud definitions (tenant and admin).
	// https://github.com/gophercloud/utils/issues/164
	os.Unsetenv("OS_CLOUD")
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	ctrl.SetLogger(klog.Background())
	RunSpecs(t, "capo-e2e")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	data := shared.Node1BeforeSuite(e2eCtx)
	return data
}, func(data []byte) {
	shared.AllNodesBeforeSuite(e2eCtx, data)

	initialServers, err = shared.DumpOpenStackServers(e2eCtx, servers.ListOpts{})
	Expect(err).NotTo(HaveOccurred())
	initialNetworks, err = shared.DumpOpenStackNetworks(e2eCtx, networks.ListOpts{})
	Expect(err).NotTo(HaveOccurred())
	initialSecurityGroups, err = shared.DumpOpenStackSecurityGroups(e2eCtx, groups.ListOpts{})
	Expect(err).NotTo(HaveOccurred())
	initialLoadBalancers, err = shared.DumpOpenStackLoadBalancers(e2eCtx, loadbalancers.ListOpts{})
	Expect(err).NotTo(HaveOccurred())
	initialVolumes, err = shared.DumpOpenStackVolumes(e2eCtx, volumes.ListOpts{})
	Expect(err).NotTo(HaveOccurred())
})

func CheckResourceCleanup[T any, L any](f func(*shared.E2EContext, L) ([]T, error), l L, initialResources []T) *string {
	endResources, err := f(e2eCtx, l)

	if err != nil {
		return pointer.String(err.Error())
	}

	matcher := ConsistOfIDs(initialResources)
	success, err := matcher.Match(endResources)
	if err != nil {
		return pointer.String(err.Error())
	}
	if !success {
		return pointer.String(matcher.FailureMessage(endResources))
	}

	return nil
}

var _ = SynchronizedAfterSuite(func() {
	shared.AllNodesAfterSuite(e2eCtx)
}, func() {
	failed := false
	for _, error := range []*string{
		CheckResourceCleanup(shared.DumpOpenStackServers, servers.ListOpts{}, initialServers),
		CheckResourceCleanup(shared.DumpOpenStackNetworks, networks.ListOpts{}, initialNetworks),
		CheckResourceCleanup(shared.DumpOpenStackSecurityGroups, groups.ListOpts{}, initialSecurityGroups),
		CheckResourceCleanup(shared.DumpOpenStackLoadBalancers, loadbalancers.ListOpts{}, initialLoadBalancers),
		CheckResourceCleanup(shared.DumpOpenStackVolumes, volumes.ListOpts{}, initialVolumes),
	} {
		if error != nil {
			GinkgoWriter.Println(*error)
			failed = true
		}
	}

	shared.Node1AfterSuite(e2eCtx)

	if failed {
		Fail("Not all resources were cleaned up")
	}
})
