/*
Copyright 2019 The Kubernetes Authors.

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

package options

import (
	"flag"
	"time"
)

var (
	TokenTTL              time.Duration
	UserDataFolder        string
	UserDataPostprocessor string
)

func init() {
	flag.DurationVar(&TokenTTL, "token_ttl", 60*time.Minute, "TTL for kubeadm bootstrap token of the target Kubernetes cluster")
	flag.StringVar(&UserDataFolder, "user-data-folder", "", "if set, user data files are retrieved from <user-data-folder>/<machine-name>.yaml")
	flag.StringVar(&UserDataPostprocessor, "user-data-postprocessor", "", "postprocessor to user for the user data")
}
