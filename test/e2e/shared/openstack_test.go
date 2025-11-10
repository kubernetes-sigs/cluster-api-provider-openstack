//go:build e2e

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
	"context"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
)

// Test_dumpOpenStackClusters can be used to develop and test
// dumpOpenStackClusters locally.
func Test_dumpOpenStackClusters(t *testing.T) {
	t.Skip()
	RegisterFailHandler(Fail)

	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	dumpOpenStack(context.TODO(), &E2EContext{
		E2EConfig: &clusterctl.E2EConfig{
			Variables: map[string]string{
				OpenStackCloudYAMLFile: fmt.Sprintf("%s/../../../clouds.yaml", currentDir),
				OpenStackCloud:         "capo-e2e",
			},
		},
		Settings: Settings{
			ArtifactFolder: "/tmp",
		},
	}, "bootstrap")
}
