/*
Copyright 2025 The Kubernetes Authors.

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
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace = "test-ns"
	testCloudName = "mycloud"
	testRegion    = "RegionOne"
)

var (
	testCloudsYAML = []byte(`clouds:
  mycloud:
    auth:
      auth_url: https://keystone.example.com/
      application_credential_id: id
      application_credential_secret: secret
    region_name: RegionOne
    interface: public
    identity_api_version: 3
    auth_type: v3applicationcredential
`)
	testCACert = []byte("-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----\n")
)

// buildCoreScheme creates a runtime scheme with core Kubernetes API types for testing.
func buildCoreScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	sch := runtime.NewScheme()
	if err := scheme.AddToScheme(sch); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}
	return sch
}

// createTestSecret creates a test Secret in the test namespace with the given name and data.
func createTestSecret(name string, data map[string][]byte) *corev1.Secret {
	secret := &corev1.Secret{}
	secret.Namespace = testNamespace
	secret.Name = name
	secret.Data = data
	return secret
}

// TestGetCloudFromSecret_SuccessWithCACert tests successful cloud retrieval when CA certificate is present.
func TestGetCloudFromSecret_SuccessWithCACert(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	secretName := "os-cred"
	secret := createTestSecret(secretName, map[string][]byte{
		CloudsSecretKey: testCloudsYAML,
		CASecretKey:     testCACert,
	})

	c := fake.NewClientBuilder().WithScheme(buildCoreScheme(t)).WithObjects(secret).Build()

	cloud, gotCACert, err := getCloudFromSecret(ctx, c, testNamespace, secretName, testCloudName)
	if err != nil {
		t.Fatalf("getCloudFromSecret returned error: %v", err)
	}
	if cloud.RegionName != testRegion {
		t.Fatalf("expected %s region, got %q", testRegion, cloud.RegionName)
	}
	if len(gotCACert) == 0 {
		t.Fatalf("expected non-empty caCert")
	}
}

// TestGetCloudFromSecret_SuccessWithoutCACert tests successful cloud retrieval when CA certificate is not present.
func TestGetCloudFromSecret_SuccessWithoutCACert(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	secretName := "os-cred-no-ca" //nolint:gosec
	secret := createTestSecret(secretName, map[string][]byte{
		CloudsSecretKey: testCloudsYAML,
	})

	c := fake.NewClientBuilder().WithScheme(buildCoreScheme(t)).WithObjects(secret).Build()

	cloud, gotCACert, err := getCloudFromSecret(ctx, c, testNamespace, secretName, testCloudName)
	if err != nil {
		t.Fatalf("getCloudFromSecret returned error: %v", err)
	}
	if cloud.RegionName != testRegion {
		t.Fatalf("expected %s region, got %q", testRegion, cloud.RegionName)
	}
	if gotCACert != nil {
		t.Fatalf("expected nil caCert when not present, got %d bytes", len(gotCACert))
	}
}

// TestGetCloudFromSecret_MissingSecret tests error handling when the secret does not exist.
func TestGetCloudFromSecret_MissingSecret(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := fake.NewClientBuilder().WithScheme(buildCoreScheme(t)).Build()

	_, _, err := getCloudFromSecret(ctx, c, testNamespace, "missing", testCloudName)
	if err == nil {
		t.Fatalf("expected error for missing secret, got nil")
	}
}

// TestGetCloudFromSecret_MissingCloudsKey tests error handling when the clouds.yaml key is missing from the secret.
func TestGetCloudFromSecret_MissingCloudsKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	secretName := "no-clouds"
	secret := createTestSecret(secretName, map[string][]byte{
		// intentionally no CloudsSecretKey
		"other": []byte("x"),
	})

	c := fake.NewClientBuilder().WithScheme(buildCoreScheme(t)).WithObjects(secret).Build()

	_, _, err := getCloudFromSecret(ctx, c, testNamespace, secretName, testCloudName)
	if err == nil {
		t.Fatalf("expected error for missing clouds.yaml key, got nil")
	}
}

// TestGetCloudFromSecret_EmptyCloudName tests error handling when cloudName is empty.
func TestGetCloudFromSecret_EmptyCloudName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	secretName := "any"
	secret := createTestSecret(secretName, map[string][]byte{
		CloudsSecretKey: []byte("clouds: {}\n"),
	})

	c := fake.NewClientBuilder().WithScheme(buildCoreScheme(t)).WithObjects(secret).Build()

	_, _, err := getCloudFromSecret(ctx, c, testNamespace, secretName, "")
	if err == nil {
		t.Fatalf("expected error when cloudName is empty, got nil")
	}
}

// TestGetCloudFromSecret_InvalidCloudName tests behavior when cloudName does not exist in clouds.yaml.
func TestGetCloudFromSecret_InvalidCloudName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	secretName := "cred"
	secret := createTestSecret(secretName, map[string][]byte{
		CloudsSecretKey: testCloudsYAML,
	})

	c := fake.NewClientBuilder().WithScheme(buildCoreScheme(t)).WithObjects(secret).Build()

	cloud, ca, err := getCloudFromSecret(ctx, c, testNamespace, secretName, "missing-cloud")
	if err != nil {
		t.Fatalf("expected no error for unknown cloudName (returned zero-value), got: %v", err)
	}
	if ca != nil {
		t.Fatalf("expected nil caCert for missing key, got %d bytes", len(ca))
	}
	if cloud.RegionName != "" || cloud.AuthInfo != nil {
		t.Fatalf("expected zero-value cloud for unknown cloudName, got RegionName=%q AuthInfo-nil=%v", cloud.RegionName, cloud.AuthInfo == nil)
	}
}
