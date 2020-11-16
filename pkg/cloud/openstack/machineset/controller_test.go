package machineset

import (
	"encoding/json"
	"fmt"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	gtypes "github.com/onsi/gomega/types"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"math/rand"
	machineproviderv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strconv"
	"testing"
	"time"
)

var emptyFlavorName = ""
var validFlavorName = "mock.xlarge"
var invalidFlavorName = "mock.invalid"

var mockFlavor = flavors.Flavor{
	ID:         "moc-flavor-id",
	Disk:       200,
	RAM:        16000,
	Name:       validFlavorName,
	RxTxFactor: 0,
	Swap:       0,
	VCPUs:      4,
	IsPublic:   false,
	Ephemeral:  0,
}

type MockInstanceService struct {
	flavor *flavors.Flavor
}

func (mock *MockInstanceService) GetFlavorID(flavorName string) (string, error) {
	if flavorName == mock.flavor.Name {
		return mock.flavor.ID, nil
	}
	return "", fmt.Errorf("flavor %q not found", flavorName)
}

func (mock *MockInstanceService) GetFlavorInfo(flavorID string) (flavor *flavors.Flavor, err error) {
	if flavorID == mock.flavor.ID {
		return mock.flavor, nil
	}
	return &flavors.Flavor{}, fmt.Errorf("flavor ID %q not found", flavorID)
}

func RandomString(prefix string, n int) string {
	const alphanum = "0123456789abcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Seed(time.Now().UTC().UnixNano())
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return prefix + string(bytes)
}

var _ = Describe("Reconciler", func() {
	var c client.Client
	var stopMgr chan struct{}
	var fakeRecorder *record.FakeRecorder
	var namespace *corev1.Namespace
	var suiteFlavorCache = newMachineFlavorCache()
	var suiteInstanceService = &MockInstanceService{
		flavor: &mockFlavor,
	}

	BeforeEach(func() {
		namespaceName := RandomString("mhc-test-", 5)
		namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}}
		mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0", Namespace: namespace.Name})
		Expect(err).ToNot(HaveOccurred())

		r := Reconciler{
			instanceService: suiteInstanceService,
		}

		Expect(r.SetupWithManager(mgr, controller.Options{})).To(Succeed())

		fakeRecorder = record.NewFakeRecorder(4)
		r.eventRecorder = fakeRecorder
		r.flavorCache = suiteFlavorCache
		c = mgr.GetClient()
		stopMgr = StartTestManager(mgr)

		Expect(c.Create(ctx, namespace)).To(Succeed())
	})

	AfterEach(func() {
		Expect(deleteMachineSets(c, namespace.Name)).To(Succeed())
		Expect(deleteNameSpace(c, namespace)).To(Succeed())
		close(stopMgr)
	})

	type reconcileTestCase = struct {
		machineFlavor       string
		existingAnnotations map[string]string
		expectedAnnotations map[string]string
		expectedEvents      []string
	}

	DescribeTable("when reconciling MachineSets",
		func(rtc reconcileTestCase) {
			machineSet, err := newTestMachineSet(namespace.Name, rtc.machineFlavor, rtc.existingAnnotations)
			Expect(err).ToNot(HaveOccurred())

			Expect(c.Create(ctx, machineSet)).To(Succeed())

			Eventually(func() map[string]string {
				m := &machinev1.MachineSet{}
				key := client.ObjectKey{Namespace: machineSet.Namespace, Name: machineSet.Name}
				err := c.Get(ctx, key, m)
				if err != nil {
					return nil
				}
				annotations := m.GetAnnotations()
				if annotations != nil {
					return annotations
				}
				// Return an empty map to distinguish between empty annotations and errors
				return make(map[string]string)
			}, timeout).Should(Equal(rtc.expectedAnnotations))

			//  Check which event types were sent
			Eventually(fakeRecorder.Events, timeout).Should(HaveLen(len(rtc.expectedEvents)))
			var receivedEvents []string
			var eventMatchers []gtypes.GomegaMatcher
			for _, ev := range rtc.expectedEvents {
				receivedEvents = append(receivedEvents, <-fakeRecorder.Events)
				eventMatchers = append(eventMatchers, ContainSubstring(fmt.Sprintf("%s", ev)))
			}

			Eventually(receivedEvents).Should(ConsistOf(eventMatchers))
		},
		Entry("with machine flavor", reconcileTestCase{
			machineFlavor:       validFlavorName,
			existingAnnotations: make(map[string]string),
			expectedAnnotations: map[string]string{
				cpuKey:    strconv.Itoa(mockFlavor.VCPUs),
				memoryKey: strconv.Itoa(mockFlavor.RAM),
			},
			expectedEvents: []string{},
		}),
		Entry("with existing annotations", reconcileTestCase{
			machineFlavor: validFlavorName,
			existingAnnotations: map[string]string{
				"existing": "annotation",
				"annother": "existingAnnotation",
			},
			expectedAnnotations: map[string]string{
				"existing": "annotation",
				"annother": "existingAnnotation",
				cpuKey:     strconv.Itoa(mockFlavor.VCPUs),
				memoryKey:  strconv.Itoa(mockFlavor.RAM),
			},
			expectedEvents: []string{},
		}),
		Entry("with no machine flavor", reconcileTestCase{
			machineFlavor:       "",
			existingAnnotations: make(map[string]string),
			expectedAnnotations: make(map[string]string),
			expectedEvents:      []string{"Warning ReconcileError flavor name is empty"},
		}),
		Entry("with an invalid machine flavor", reconcileTestCase{
			machineFlavor: invalidFlavorName,
			existingAnnotations: map[string]string{
				"existing": "annotation",
				"annother": "existingAnnotation",
			},
			expectedAnnotations: map[string]string{
				"existing": "annotation",
				"annother": "existingAnnotation",
			},
			expectedEvents: []string{"ReconcileError"},
		}),
	)
})

func deleteNameSpace(c client.Client, ns *corev1.Namespace) error {
	return c.Delete(ctx, ns)
}

func deleteMachineSets(c client.Client, namespaceName string) error {
	machineSets := &machinev1.MachineSetList{}
	err := c.List(ctx, machineSets, client.InNamespace(namespaceName))
	if err != nil {
		return err
	}

	for _, ms := range machineSets.Items {
		err := c.Delete(ctx, &ms)
		if err != nil {
			return err
		}
	}

	Eventually(func() error {
		machineSets := &machinev1.MachineSetList{}
		err := c.List(ctx, machineSets, client.InNamespace(namespaceName))
		if err != nil {
			return err
		}
		if len(machineSets.Items) > 0 {
			return fmt.Errorf("MachineSets not deleted")
		}
		return nil
	}, timeout).Should(Succeed())

	return nil
}

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                string
		flavor              string
		existingAnnotations map[string]string
		expectedAnnotations map[string]string
		expectErr           bool
	}{
		{
			name:   "with existing annotations",
			flavor: validFlavorName,
			existingAnnotations: map[string]string{
				"existing": "annotation",
				"annother": "existingAnnotation",
			},
			expectedAnnotations: map[string]string{
				"existing": "annotation",
				"annother": "existingAnnotation",
				cpuKey:     strconv.Itoa(mockFlavor.VCPUs),
				memoryKey:  strconv.Itoa(mockFlavor.RAM),
			},
			expectErr: false,
		},
		{
			name:   "with an invalid machine flavor",
			flavor: invalidFlavorName,
			existingAnnotations: map[string]string{
				"existing": "annotation",
				"annother": "existingAnnotation",
			},
			expectedAnnotations: map[string]string{
				"existing": "annotation",
				"annother": "existingAnnotation",
			},
			expectErr: true,
		},
		{
			name:                "with no machine flavor",
			flavor:              emptyFlavorName,
			existingAnnotations: make(map[string]string),
			expectedAnnotations: make(map[string]string),
			expectErr:           true,
		},
		{
			name:                "with machine flavor",
			flavor:              validFlavorName,
			existingAnnotations: make(map[string]string),
			expectedAnnotations: map[string]string{
				cpuKey:    strconv.Itoa(mockFlavor.VCPUs),
				memoryKey: strconv.Itoa(mockFlavor.RAM),
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			g := NewWithT(tt)

			//Create reconciler
			r := Reconciler{
				instanceService: &MockInstanceService{
					flavor: &mockFlavor,
				},
				flavorCache: newMachineFlavorCache(),
			}

			//Get a machineset
			machineSet, err := newTestMachineSet("default", tc.flavor, tc.existingAnnotations)
			g.Expect(err).ToNot(HaveOccurred())

			//Use the reconciler we create to reconcile the machineset
			_, err = r.reconcile(machineSet)
			g.Expect(err != nil).To(Equal(tc.expectErr))
			g.Expect(machineSet.Annotations).To(Equal(tc.expectedAnnotations))
		})
	}
}

func newTestMachineSet(namespace string, flavor string, existingAnnotations map[string]string) (*machinev1.MachineSet, error) {
	// Copy anntotations map so we don't modify the input
	annotations := make(map[string]string)
	for k, v := range existingAnnotations {
		annotations[k] = v
	}

	machineProviderSpec := &machineproviderv1.OpenstackProviderSpec{
		Flavor:    flavor,
		CloudName: "openstack",
		CloudsSecret: &corev1.SecretReference{
			Name:      "openstack-cloud-credentials",
			Namespace: "openshift-machine-api",
		},
	}

	providerSpec, err := providerSpecFromMachine(machineProviderSpec)
	if err != nil {
		return nil, err
	}

	return &machinev1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Annotations:  annotations,
			GenerateName: "test-machineset-",
			Namespace:    namespace,
		},
		Spec: machinev1.MachineSetSpec{
			Template: machinev1.MachineTemplateSpec{
				Spec: machinev1.MachineSpec{
					ProviderSpec: providerSpec,
				},
			},
		},
	}, nil
}

func providerSpecFromMachine(in *machineproviderv1.OpenstackProviderSpec) (machinev1.ProviderSpec, error) {
	bytes, err := json.Marshal(in)
	if err != nil {
		return machinev1.ProviderSpec{}, err
	}
	return machinev1.ProviderSpec{
		Value: &runtime.RawExtension{Raw: bytes},
	}, nil
}
