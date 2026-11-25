/*
Copyright 2026 The Kubernetes Authors.

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

package webhooks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"

	infrav1alpha1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha1"
	infrav1beta1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	infrav1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta2"
)

// TestConversionWebhookRegistration exercises the registered converters through
// the HTTP handler, without starting a Kubernetes API server or the manager.
func TestConversionWebhookRegistration(t *testing.T) {
	g := NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(infrav1alpha1.AddToScheme(scheme)).To(Succeed())
	g.Expect(infrav1beta1.AddToScheme(scheme)).To(Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(Succeed())
	mgr, err := manager.New(&rest.Config{Host: "https://localhost:6443"}, manager.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(RegisterAllWithManager(mgr)).To(BeEmpty())
	handler := conversion.NewWebhookHandler(scheme, mgr.GetConverterRegistry())

	for _, kind := range []string{"OpenStackCluster", "OpenStackMachine", "OpenStackClusterTemplate", "OpenStackMachineTemplate"} {
		for _, from := range []string{"v1beta1", "v1beta2"} {
			t.Run(kind+"/"+from, func(t *testing.T) {
				g := NewWithT(t)
				to := "v1beta2"
				if from == to {
					to = "v1beta1"
				}
				gvk := infrav1.SchemeGroupVersion.WithKind(kind)
				_, registered := mgr.GetConverterRegistry().GetConverter(gvk.GroupKind())
				g.Expect(registered).To(BeTrue())
				gvk.Version = from
				obj, err := scheme.New(gvk)
				g.Expect(err).NotTo(HaveOccurred())
				obj.GetObjectKind().SetGroupVersionKind(gvk)
				obj.(metav1.Object).SetName("test-conversion")
				request := apiextensionsv1.ConversionReview{
					TypeMeta: metav1.TypeMeta{APIVersion: apiextensionsv1.SchemeGroupVersion.String(), Kind: "ConversionReview"},
					Request: &apiextensionsv1.ConversionRequest{
						UID:               "test-conversion",
						DesiredAPIVersion: infrav1.GroupName + "/" + to,
						Objects:           []runtime.RawExtension{{Object: obj}},
					},
				}
				body, err := json.Marshal(request)
				g.Expect(err).NotTo(HaveOccurred())
				req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/convert", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, req)
				g.Expect(response.Code).To(Equal(http.StatusOK))
				var review apiextensionsv1.ConversionReview
				g.Expect(json.Unmarshal(response.Body.Bytes(), &review)).To(Succeed())
				g.Expect(review.Response).NotTo(BeNil())
				g.Expect(review.Response.UID).To(Equal(request.Request.UID))
				g.Expect(review.Response.Result.Status).To(Equal(metav1.StatusSuccess))
				g.Expect(review.Response.ConvertedObjects).To(HaveLen(1))
				var converted struct {
					metav1.TypeMeta   `json:",inline"`
					metav1.ObjectMeta `json:"metadata"`
				}
				g.Expect(json.Unmarshal(review.Response.ConvertedObjects[0].Raw, &converted)).To(Succeed())
				g.Expect(converted.APIVersion).To(Equal(request.Request.DesiredAPIVersion))
				g.Expect(converted.Kind).To(Equal(kind))
				g.Expect(converted.Name).To(Equal("test-conversion"))
			})
		}
	}
}
