/*
Copyright 2022 The Kubernetes Authors.

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

package scope

import (
	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

// Scope is used to initialize Services from Controllers and includes the
// common objects required for this.
//
// The Gophercloud ProviderClient and ClientOpts are required to create new
// Gophercloud API Clients (e.g. for Networking/Neutron).
//
// The Logger includes context values such as the cluster name.
type Scope struct {
	ProviderClient     *gophercloud.ProviderClient
	ProviderClientOpts *clientconfig.ClientOpts

	Logger logr.Logger
}
