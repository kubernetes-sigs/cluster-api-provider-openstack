/*
Copyright 2018 The Kubernetes Authors.

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

package clients

import (
	"strings"
	"testing"
)

func TestMachineServiceInstance(t *testing.T) {
	_, err := NewInstanceService()
	if !(strings.Contains(err.Error(), "[auth_url]")) {
		t.Errorf("Couldn't create instance service: %v", err)
	}
}
