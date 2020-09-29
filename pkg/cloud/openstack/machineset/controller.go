package machineset

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	openstackconfigv1 "sigs.k8s.io/cluster-api-provider-openstack/pkg/apis/openstackproviderconfig/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-openstack/pkg/cloud/openstack/clients"
	ctrlRuntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"strconv"
	"time"
)

const (
	// This exposes compute information based on the providerSpec input.
	// This is needed by the autoscaler to foresee upcoming capacity when scaling from zero.
	// https://github.com/openshift/enhancements/pull/186
	cpuKey    = "machine.openshift.io/vCPU"
	memoryKey = "machine.openshift.io/memoryMb"
)

type OpenStackInstanceService interface {
	GetFlavorID(flavorName string) (string, error)
	GetFlavorInfo(flavorID string) (flavor *flavors.Flavor, err error)
}

type Reconciler struct {
	Client          client.Client
	Log             logr.Logger
	eventRecorder   record.EventRecorder
	scheme          *runtime.Scheme
	kubeClient      *kubernetes.Clientset
	instanceService OpenStackInstanceService
	flavorCache     *machineFlavorsCache
}

// Reconcile implements controller runtime Reconciler interface.
func (r *Reconciler) Reconcile(req ctrlRuntime.Request) (ctrlRuntime.Result, error) {

	logger := r.Log.WithValues("machineset", req.Name, "namespace", req.Namespace)
	logger.V(3).Info("Reconciling")

	ctx := context.Background()
	machineSet := &machinev1.MachineSet{}
	if err := r.Client.Get(ctx, req.NamespacedName, machineSet); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrlRuntime.Result{}, nil
		}
		return ctrlRuntime.Result{}, err
	}

	// Ignore deleted MachineSets, this can happen when foregroundDeletion
	// is enabled
	if !machineSet.DeletionTimestamp.IsZero() {
		return ctrlRuntime.Result{}, nil
	}

	originalMachineSetPatch := client.MergeFrom(machineSet.DeepCopy())

	if r.instanceService == nil {
		//Create the OpenStack InstanceService if we don't have one already.
		m := &machinev1.Machine{Spec: machineSet.Spec.Template.Spec}
		is, err := clients.NewInstanceServiceFromMachine(r.kubeClient, m)
		if err != nil {
			return ctrlRuntime.Result{}, fmt.Errorf("failed to get InstanceService: %v", err)
		}
		r.instanceService = is
	}
	//reconcile the machine set and patch  even if reconcile failed.
	result, err := r.reconcile(machineSet)
	if err != nil {
		logger.Error(err, "Failed to reconcile MachineSet %q", machineSet.Name)
		r.eventRecorder.Eventf(machineSet, corev1.EventTypeWarning, "ReconcileError", "%v", err)
	}

	if err := r.Client.Patch(ctx, machineSet, originalMachineSetPatch); err != nil {
		return ctrlRuntime.Result{}, fmt.Errorf("failed to patch machineSet: %v", err)
	}
	return result, err
}

func requeueTime() time.Duration {
	// Currently depends on caches refresh failure time, which is how long the cache will wait before
	// retrying to refresh the information of a failed look up.
	return RefreshFailureTime / 2
}
func (r *Reconciler) reconcile(machineSet *machinev1.MachineSet) (ctrlRuntime.Result, error) {
	pSpec, err := openstackconfigv1.MachineSpecFromProviderSpec(machineSet.Spec.Template.Spec.ProviderSpec)
	if err != nil {
		return ctrlRuntime.Result{}, fmt.Errorf("failed to get OpenStackProviderSpec from machineset: %v", err)
	}
	if pSpec.Flavor == "" {
		return ctrlRuntime.Result{}, fmt.Errorf("flavor name is empty for machineset %q in namespace %q", machineSet.Name, machineSet.Namespace)
	}

	if machineSet.Annotations == nil {
		machineSet.Annotations = make(map[string]string)
	}

	flavorInfo := r.flavorCache.getFlavorInfo(r.instanceService, pSpec.Flavor)
	if flavorInfo == nil {
		// At this time we don't have enough information to set correct annotations
		// so we inform the controller it needs to requeue the request.
		return ctrlRuntime.Result{
			Requeue:      true,
			RequeueAfter: requeueTime(),
		}, fmt.Errorf("could not find information for %q", pSpec.Flavor)
	}

	machineSet.Annotations[cpuKey] = strconv.Itoa(flavorInfo.VCPUs)
	machineSet.Annotations[memoryKey] = strconv.Itoa(flavorInfo.RAM)

	return ctrlRuntime.Result{}, nil
}

// SetupWithManager creates a new controller for a manager.
func (r *Reconciler) SetupWithManager(mgr ctrlRuntime.Manager, options controller.Options) error {
	err := ctrlRuntime.NewControllerManagedBy(mgr).
		For(&machinev1.MachineSet{}).
		WithOptions(options).
		Complete(r)
	//TODO (adduarte) evaluate if it is valuable to set number of instances of Reconcilers. (MaxConcurr3entReconciles)
	// see https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/#controller-configurations

	if err != nil {
		return errors.Wrap(err, "controller creation failed")
	}

	r.Client = mgr.GetClient()
	r.scheme = mgr.GetScheme()
	r.Log = mgr.GetLogger()
	r.eventRecorder = mgr.GetEventRecorderFor("machineset-controller")
	r.scheme = mgr.GetScheme()
	config := mgr.GetConfig()
	r.kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "could not create kubernetes client to talk to the API server")
	}
	r.flavorCache = newMachineFlavorCache()

	return nil
}
