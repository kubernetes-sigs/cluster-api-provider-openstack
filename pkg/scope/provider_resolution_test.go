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
	"errors"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

const (
	testNSA          = "ns-a"
	testNSAllowed    = "allowed-team"
	testNSTeamY      = "team-y"
	testNSTeamZ      = "team-z"
	testNSTeamW      = "team-w"
	testNSTeamA      = "team-a"
	testNSCapo       = "capo-system"
	resTestCloudName = "mycloud"
	testProdCloud    = "prodcloud"
)

var (
	testValidCloudsYAML = []byte(`clouds:
  mycloud:
    auth:
      auth_url: https://keystone.example.com/
      application_credential_id: id
      application_credential_secret: secret
    region_name: RegionOne
`)
	testProdCloudsYAML = []byte(`clouds:
  prodcloud:
    auth:
      auth_url: https://keystone.prod.com/
      application_credential_id: prod-id
      application_credential_secret: prod-secret
    region_name: RegionOne
`)
	testEmptyCloudsYAML   = []byte("clouds: {}\n")
	testDefaultCloudsYAML = []byte("clouds: { default: {} }\n")
)

// ensureSchemes creates a runtime scheme with all required API types for testing.
func ensureSchemes(t *testing.T) *runtime.Scheme {
	t.Helper()
	local := runtime.NewScheme()
	if err := scheme.AddToScheme(local); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}
	if err := infrav1.AddToScheme(local); err != nil {
		t.Fatalf("failed to add v1beta1 scheme: %v", err)
	}
	if err := infrav1alpha1.AddToScheme(local); err != nil {
		t.Fatalf("failed to add v1alpha1 scheme: %v", err)
	}
	return local
}

// createResTestSecret creates a test Secret with the given namespace, name, and data.
func createResTestSecret(namespace, name string, data map[string][]byte) *corev1.Secret {
	secret := &corev1.Secret{}
	secret.Namespace = namespace
	secret.Name = name
	secret.Data = data
	return secret
}

// createTestNamespace creates a test Namespace with the given name and labels.
func createTestNamespace(name string, labels map[string]string) *corev1.Namespace {
	ns := &corev1.Namespace{}
	ns.Name = name
	ns.Labels = labels
	return ns
}

// createTestClusterIdentity creates a test OpenStackClusterIdentity with the given name and namespace selector.
func createTestClusterIdentity(name string, selector *metav1.LabelSelector) *infrav1alpha1.OpenStackClusterIdentity {
	identity := &infrav1alpha1.OpenStackClusterIdentity{}
	identity.Name = name
	identity.Spec.SecretRef = infrav1alpha1.OpenStackCredentialSecretReference{
		Name:      "creds",
		Namespace: testNSCapo,
	}
	identity.Spec.NamespaceSelector = selector
	return identity
}

// newFakeClient creates a fake Kubernetes client with the provided scheme and objects.
func newFakeClient(sch *runtime.Scheme, objs ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(sch).WithObjects(objs...).Build()
}

// assertResolutionReached ensures the error is not caused by credential resolution (missing secret/namespace or access denied).
// We still expect an error (typically from OpenStack auth) when running full scope creation with the real factory.
func assertResolutionReached(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected OpenStack auth error, got success")
	}
	if strings.Contains(err.Error(), "secret") && strings.Contains(err.Error(), "not found") {
		t.Fatalf("credential resolution failed: %v", err)
	}
	var denied *IdentityAccessDeniedError
	if errors.As(err, &denied) {
		t.Fatalf("credential resolution failed: %v", err)
	}
}

// assertDenied verifies that the error is an IdentityAccessDeniedError.
func assertDenied(t *testing.T, err error) {
	t.Helper()
	var denied *IdentityAccessDeniedError
	if err == nil || !errors.As(err, &denied) {
		t.Fatalf("expected IdentityAccessDeniedError, got %T %v", err, err)
	}
}

// assertNotDenied verifies that the error is NOT an IdentityAccessDeniedError.
func assertNotDenied(t *testing.T, err error) {
	t.Helper()
	var denied *IdentityAccessDeniedError
	if errors.As(err, &denied) {
		t.Fatalf("did not expect IdentityAccessDeniedError, got %v", err)
	}
}

// TestNewClientScopeFromObject_Resolution tests credential resolution logic for both Secret and ClusterIdentity paths.
func TestNewClientScopeFromObject_Resolution(t *testing.T) {
	t.Parallel()
	localScheme := ensureSchemes(t)
	type testCase struct {
		name      string
		objects   []client.Object
		namespace string
		identity  infrav1.OpenStackIdentityReference
		assertErr func(*testing.T, error)
	}

	cases := []testCase{
		{
			name: "secret path returns scope",
			objects: []client.Object{
				createResTestSecret(testNSA, "valid-creds", map[string][]byte{CloudsSecretKey: testValidCloudsYAML}),
			},
			namespace: testNSA,
			identity:  infrav1.OpenStackIdentityReference{Name: "valid-creds", CloudName: resTestCloudName},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				assertResolutionReached(t, err)
			},
		},
		{
			name: "clusteridentity returns scope when selector allows",
			objects: []client.Object{
				createTestNamespace(testNSAllowed, map[string]string{"env": "prod"}),
				createTestClusterIdentity("prod-id", &metav1.LabelSelector{MatchLabels: map[string]string{"env": "prod"}}),
				createResTestSecret(testNSCapo, "creds", map[string][]byte{CloudsSecretKey: testProdCloudsYAML}),
			},
			namespace: testNSAllowed,
			identity:  infrav1.OpenStackIdentityReference{Type: "ClusterIdentity", Name: "prod-id", CloudName: testProdCloud},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				assertResolutionReached(t, err)
			},
		},
		{
			name:      "secret path: missing secret returns error",
			objects:   []client.Object{},
			namespace: testNSA,
			identity:  infrav1.OpenStackIdentityReference{Name: "missing", CloudName: "cloudA"},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if err == nil {
					t.Fatalf("expected error")
				}
			},
		},
		{
			name:      "secret path: empty cloudName returns error",
			objects:   []client.Object{createResTestSecret(testNSA, "creds", map[string][]byte{CloudsSecretKey: testEmptyCloudsYAML})},
			namespace: testNSA,
			identity:  infrav1.OpenStackIdentityReference{Name: "creds", CloudName: ""},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if err == nil {
					t.Fatalf("expected error")
				}
			},
		},
		{
			name:      "clusteridentity: identity not found",
			objects:   []client.Object{},
			namespace: "team-x",
			identity:  infrav1.OpenStackIdentityReference{Type: "ClusterIdentity", Name: "missing-id", CloudName: "cloudA"},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if err == nil {
					t.Fatalf("expected error")
				}
			},
		},
		{
			name: "clusteridentity: selector denies -> access denied",
			objects: []client.Object{
				createTestNamespace(testNSTeamY, nil),
				createTestClusterIdentity("prod-id", &metav1.LabelSelector{MatchLabels: map[string]string{"allowed": "true"}}),
				createResTestSecret(testNSCapo, "creds", map[string][]byte{CloudsSecretKey: testEmptyCloudsYAML}),
			},
			namespace: testNSTeamY,
			identity:  infrav1.OpenStackIdentityReference{Type: "ClusterIdentity", Name: "prod-id", CloudName: "cloudA"},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				assertDenied(t, err)
			},
		},
		{
			name: "clusteridentity: selector nil allows (not denied)",
			objects: []client.Object{
				createTestNamespace(testNSTeamZ, nil),
				createTestClusterIdentity("any-id", nil),
				createResTestSecret(testNSCapo, "creds", map[string][]byte{CloudsSecretKey: testDefaultCloudsYAML}),
			},
			namespace: testNSTeamZ,
			identity:  infrav1.OpenStackIdentityReference{Type: "ClusterIdentity", Name: "any-id", CloudName: "default"},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				assertNotDenied(t, err)
			},
		},
		{
			name: "clusteridentity: empty selector matches all (not denied)",
			objects: []client.Object{
				createTestNamespace(testNSTeamW, nil),
				createTestClusterIdentity("empty-selector-id", &metav1.LabelSelector{}),
				createResTestSecret(testNSCapo, "creds", map[string][]byte{CloudsSecretKey: testDefaultCloudsYAML}),
			},
			namespace: testNSTeamW,
			identity:  infrav1.OpenStackIdentityReference{Type: "ClusterIdentity", Name: "empty-selector-id", CloudName: "default"},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				assertNotDenied(t, err)
			},
		},
		{
			name: "clusteridentity: cross-namespace secret allowed (not denied)",
			objects: []client.Object{
				createTestNamespace(testNSTeamA, nil),
				createTestNamespace(testNSCapo, nil),
				createTestClusterIdentity("cross-ns-id", nil),
				createResTestSecret(testNSCapo, "creds", map[string][]byte{CloudsSecretKey: testDefaultCloudsYAML}),
			},
			namespace: testNSTeamA,
			identity:  infrav1.OpenStackIdentityReference{Type: "ClusterIdentity", Name: "cross-ns-id", CloudName: "default"},
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				assertNotDenied(t, err)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			c := newFakeClient(localScheme, tc.objects...)
			// Always use real factory with fake k8s client - this tests credential resolution
			// without making OpenStack API calls
			factory := &providerScopeFactory{}

			srv := &infrav1alpha1.OpenStackServer{}
			srv.Namespace = tc.namespace
			srv.Spec.IdentityRef = tc.identity

			_, err := factory.NewClientScopeFromObject(ctx, c, nil, logr.Discard(), srv)
			tc.assertErr(t, err)
		})
	}
}
