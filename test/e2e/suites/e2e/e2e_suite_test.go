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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

var e2eCtx *shared.E2EContext

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
	RunSpecs(t, "capo-e2e")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	data := shared.Node1BeforeSuite(e2eCtx)
	createTestVolumeType(e2eCtx)
	return data
}, func(data []byte) {
	shared.AllNodesBeforeSuite(e2eCtx, data)
})

var _ = SynchronizedAfterSuite(func() {
	shared.AllNodesAfterSuite(e2eCtx)
}, func() {
	shared.Node1AfterSuite(e2eCtx)
})
