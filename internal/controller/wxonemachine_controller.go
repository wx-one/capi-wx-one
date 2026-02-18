/*
Copyright 2025.

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

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/clustercache"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1 "github.com/wx-one/cluster-api-provider-wxone/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clog "sigs.k8s.io/cluster-api/util/log"
)

// WXOneMachineReconciler reconciles a WXOneMachine object
type WXOneMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=wxonemachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=wxonemachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=wxonemachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets;configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WXOneMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *WXOneMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the DockerMachine instance.
	wxOneMachine := &infrav1.WXOneMachine{}
	if err := r.Client.Get(ctx, req.NamespacedName, wxOneMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// AddOwners adds the owners of DockerMachine as k/v pairs to the logger.
	// Specifically, it will add KubeadmControlPlane, MachineSet and MachineDeployment.
	ctx, log, err := clog.AddOwners(ctx, r.Client, wxOneMachine)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, wxOneMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Waiting for Machine Controller to set OwnerRef on DockerMachine")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("Machine", klog.KObj(machine))
	ctx = ctrl.LoggerInto(ctx, log)

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("DockerMachine owner Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, err
	}
	if cluster == nil {
		log.Info(fmt.Sprintf("Please associate this machine with a cluster using the label %s: <name of cluster>", clusterv1.ClusterNameLabel))
		return ctrl.Result{}, nil
	}

	log = log.WithValues("Cluster", klog.KObj(cluster))
	ctx = ctrl.LoggerInto(ctx, log)

	if cluster.Spec.InfrastructureRef == nil {
		log.Info("Cluster infrastructureRef is not available yet")
		return ctrl.Result{}, nil
	}

	// Fetch the WXOne Cluster.
	wxOneCluster := &infrav1.WXOneCluster{}
	wxOneClusterName := client.ObjectKey{
		Namespace: wxOneMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, wxOneClusterName, wxOneCluster); err != nil {
		log.Info("WXOneCluster is not available yet")
		return ctrl.Result{}, nil
	}

	// Handle deleted machines
	if !wxOneMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, cluster, wxOneCluster, machine, wxOneMachine)
	}

	// Handle non-deleted machines
	res, err := r.reconcileNormal(ctx, cluster, wxOneCluster, machine, wxOneMachine)
	// Requeue if the reconcile failed because the ClusterCacheTracker was locked for
	// the current cluster because of concurrent access.
	if errors.Is(err, clustercache.ErrClusterNotConnected) {
		log.V(5).Info("Requeuing because connection to the workload cluster is down")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	return res, err
}

func (r *WXOneMachineReconciler) reconcileNormal(ctx context.Context, cluster *clusterv1.Cluster, wxOneCluster *infrav1.WXOneCluster, machine *clusterv1.Machine, wxOneMachine *infrav1.WXOneMachine) (res ctrl.Result, retErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("starting reconcile normal")
	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(wxOneMachine, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.AddFinalizer(wxOneMachine, infrav1.MachineFinalizer)
	log.Info("patch machine")
	if err := patchHelper.Patch(ctx, wxOneMachine); err != nil {
		log.Error(err, "failed to patch WxoneCluster")
		return ctrl.Result{}, err
	}
	log.Info("patched without errors")
	var host string
	var user string
	var pass string

	if wxOneCluster.Spec.CredentialsSecretRef != nil {
		var s corev1.Secret
		key := types.NamespacedName{
			Namespace: wxOneCluster.Namespace,
			Name:      wxOneCluster.Spec.CredentialsSecretRef.Name,
		}
		if err := r.Client.Get(ctx, key, &s); err != nil {
			return ctrl.Result{}, err
		}

		host = string(s.Data["host"])
		user = string(s.Data["username"])
		pass = string(s.Data["password"])
	}
	wxOneClients, err := NewWXOneClients(log, ctx, host, user, pass)

	if err != nil {
		log.Error(err, "failed to create WXOneClients")
		return ctrl.Result{}, err
	}

	// Check if the infrastructure is ready, otherwise return and wait for the cluster object to be updated
	if !cluster.Status.InfrastructureReady {
		log.Info("Waiting for DockerCluster Controller to create cluster infrastructure")
		return ctrl.Result{}, nil
	}

	if machine.Spec.Bootstrap.DataSecretName == nil {
		log.Info("Bootstrap data secret reference is not yet available")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if wxOneMachine.Status.Flavor.ResourceID == "" {
		log.Info("retrieving Flavor")
		flavor, err := getFlavorByName(ctx, wxOneClients.graphqlClient, wxOneMachine.Spec.Flavor.Name)

		if err != nil {
			log.Error(err, "failed to retrieve Flavor for Cluster Machines")
			return ctrl.Result{}, err
		}

		wxOneMachine.Status.Flavor.ResourceID = flavor.GetFlavorByName.Msg.Id
		log.Info("patch cluster")
		if err := patchHelper.Patch(ctx, wxOneMachine); err != nil {
			log.Error(err, "failed to patch WxoneCluster")
			return ctrl.Result{}, err
		}
		log.Info("patched without errors")
	}

	if wxOneMachine.Status.Image.ResourceID == "" {
		log.Info("retrieving Image")
		imageList, err := getImageList(ctx, wxOneClients.graphqlClient, wxOneCluster.Status.Project.ResourceID)

		if err != nil {
			log.Error(err, "failed to retrieve image list")
			return ctrl.Result{}, err
		}

		imageFound := false
		for _, image := range imageList.GetImageList.Msg {
			if image.Name == wxOneMachine.Spec.Image.Name {
				imageFound = true
				wxOneMachine.Status.Image.ResourceID = image.Id
				log.Info("patch cluster")
				if err := patchHelper.Patch(ctx, wxOneMachine); err != nil {
					log.Error(err, "failed to patch WxoneCluster")
					return ctrl.Result{}, err
				}
				log.Info("patched without errors")
			}
		}

		if imageFound == false {
			err := errors.New("image with name does not exist")
			log.Error(err, "failed to retrieve Image for Cluster Machines")
			return ctrl.Result{}, err
		}
	}

	if wxOneMachine.Status.Instance.ResourceID == "" && wxOneMachine.Status.Image.ResourceID != "" && wxOneMachine.Status.Flavor.ResourceID != "" {
		log.Info("fetch bootstrap data")
		secret := &corev1.Secret{}
		key := types.NamespacedName{Namespace: wxOneMachine.Namespace, Name: *machine.Spec.Bootstrap.DataSecretName}
		if err := r.Client.Get(ctx, key, secret); err != nil {
			return ctrl.Result{}, err
		}

		// get the bootstrap data. This contains the cloud init script which needs to be passed somehow to instance creation
		_, ok := secret.Data["value"]
		if !ok {
			return ctrl.Result{}, errors.New("error retrieving bootstrap data: secret value key is missing")
		}

		var additional InstanceAdditionalInput
		var userData UserDataInput

		userData = UserDataInput{}
		raw := json.RawMessage([]byte(string(secret.Data["value"])))
		userData.Content = raw
		userData.Mode = "override"
		additional.UserData = userData

		log.Info("creating instance")
		instance, err := createInstance(ctx, wxOneClients.graphqlClient, wxOneCluster.Status.Network.SubnetReference.ResourceID, wxOneMachine.Status.Flavor.ResourceID, wxOneMachine.Status.Image.ResourceID, wxOneCluster.Status.Project.ResourceID, machine.Name, []string{wxOneCluster.Status.SSHKey.ResourceID}, AvailabilityZone(wxOneCluster.Spec.AvailabilityZone), false, additional)
		if err != nil {
			log.Error(err, "Failed to create instance")
			return ctrl.Result{}, err
		}
		wxOneMachine.Spec.ProviderID = ptr.To[string](fmt.Sprintf("wxone://%s", instance.CreateInstance.Msg.Id))
		wxOneMachine.Status.Instance.ResourceID = instance.CreateInstance.Msg.Id

		addresses := make([]v1beta1.MachineAddress, len(instance.CreateInstance.Msg.Addresses))
		for i, str := range instance.CreateInstance.Msg.Addresses {
			addresses[i] = v1beta1.MachineAddress{
				Type:    v1beta1.MachineInternalIP,
				Address: str,
			}
		}

		wxOneMachine.Status.Addresses = addresses

		log.Info("patch machine")
		if err := patchHelper.Patch(ctx, wxOneMachine); err != nil {
			log.Error(err, "failed to patch WxoneCluster")
			return ctrl.Result{}, err
		}
		log.Info("patched without errors")
	}

	if util.IsControlPlaneMachine(machine) && wxOneMachine.Status.Instance.ResourceID != "" && wxOneMachine.Status.FloatingIPAttachement.ResourceID == "" {
		floatingIPAttachement, err := createFloatingGroup(ctx, wxOneClients.graphqlClient, wxOneCluster.Status.FloatingIP.ResourceID, wxOneCluster.Status.Project.ResourceID, []FloatingGroupVmInput{{Priority: 0, Vm: wxOneMachine.Status.Instance.ResourceID}}, true)
		if err != nil {
			log.Error(err, "Failed to create floating ip attachement")
			return ctrl.Result{}, err
		}
		wxOneMachine.Status.FloatingIPAttachement.ResourceID = floatingIPAttachement.CreateFloatingGroup.Msg[0].Id

		log.Info("patch machine")
		if err := patchHelper.Patch(ctx, wxOneMachine); err != nil {
			log.Error(err, "failed to patch WxoneCluster")
			return ctrl.Result{}, err
		}
		log.Info("patched without errors")
	}

	if (util.IsControlPlaneMachine(machine) && wxOneMachine.Status.FloatingIPAttachement.ResourceID != "" || !util.IsControlPlaneMachine(machine)) &&
		wxOneMachine.Status.Instance.ResourceID != "" {
		instance, err := getInstance(ctx, wxOneClients.graphqlClient, wxOneMachine.Status.Instance.ResourceID, wxOneCluster.Status.Project.ResourceID)
		if err != nil {
			log.Error(err, "Failed to retrieve instance", wxOneMachine.Status.Instance.ResourceID)
			return ctrl.Result{}, err
		}
		if instance.GetInstance.Msg.Status == "RUNNING" {
			log.Info("Instance", wxOneMachine.Status.Instance.ResourceID, "has entered running state, mark ready")
			wxOneMachine.Status.Ready = true
			log.Info("patch machine")
			if err := patchHelper.Patch(ctx, wxOneMachine); err != nil {
				log.Error(err, "failed to patch WxoneCluster")
				return ctrl.Result{}, err
			}
			log.Info("patched without errors")
		} else {
			log.Info("Instance", wxOneMachine.Status.Instance.ResourceID, "has not entered running state, instance is in state", string(instance.GetInstance.Msg.Status), "requeueing")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	return res, retErr
}

func (r *WXOneMachineReconciler) reconcileDelete(ctx context.Context, cluster *clusterv1.Cluster, wxOneCluster *infrav1.WXOneCluster, machine *clusterv1.Machine, wxOneMachine *infrav1.WXOneMachine) (res ctrl.Result, retErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("starting reconcile delete")
	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(wxOneMachine, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	var host string
	var user string
	var pass string

	if wxOneCluster.Spec.CredentialsSecretRef != nil {
		var s corev1.Secret
		key := types.NamespacedName{
			Namespace: wxOneCluster.Namespace,
			Name:      wxOneCluster.Spec.CredentialsSecretRef.Name,
		}
		if err := r.Client.Get(ctx, key, &s); err != nil {
			return ctrl.Result{}, err
		}

		host = string(s.Data["host"])
		user = string(s.Data["username"])
		pass = string(s.Data["password"])
	}
	wxOneClients, err := NewWXOneClients(log, ctx, host, user, pass)
	if err != nil {
		log.Error(err, "failed to create WXOneClients")
		return ctrl.Result{}, err
	}

	// if the deleted machine is a control-plane node, remove it from the load balancer configuration;
	if util.IsControlPlaneMachine(machine) && wxOneMachine.Status.FloatingIPAttachement.ResourceID != "" {
		_, err = deleteFloatingGroupByFloatingIpIdAndInstanceId(ctx, wxOneClients.graphqlClient, wxOneCluster.Status.Project.ResourceID, wxOneCluster.Status.FloatingIP.ResourceID, wxOneMachine.Status.Instance.ResourceID)
		if err != nil {
			log.Error(err, "failed to delete floating group")
			return ctrl.Result{}, err
		}

		wxOneMachine.Status.FloatingIPAttachement.ResourceID = ""
		log.Info("patch machine")
		if err := patchHelper.Patch(ctx, wxOneMachine); err != nil {
			log.Error(err, "failed to patch WxoneMachine")
			return ctrl.Result{}, err
		}
		log.Info("patched without errors")
	}

	if wxOneMachine.Status.Instance.ResourceID != "" && (util.IsControlPlaneMachine(machine) && wxOneMachine.Status.FloatingIPAttachement.ResourceID == "" || !util.IsControlPlaneMachine(machine)) {
		log.Info("deleting instance", "instance", wxOneMachine.Status.Instance.ResourceID)
		_, err := deleteInstance(ctx, wxOneClients.graphqlClient, wxOneMachine.Status.Instance.ResourceID, wxOneCluster.Status.Project.ResourceID)

		if err != nil {
			log.Error(err, "failed to delete instance")
			return ctrl.Result{}, err
		}

		wxOneMachine.Status.Instance.ResourceID = ""
		log.Info("patch machine")
		if err := patchHelper.Patch(ctx, wxOneMachine); err != nil {
			log.Error(err, "failed to patch WxoneMachine")
			return ctrl.Result{}, err
		}
		log.Info("patched without errors")
	}

	if wxOneMachine.Status.Instance.ResourceID == "" {
		log.Info("removing finalizer")
		// Machine is deleted so remove the finalizer.
		controllerutil.RemoveFinalizer(wxOneMachine, infrav1.MachineFinalizer)
		log.Info("patch machine")
		if err := patchHelper.Patch(ctx, wxOneMachine); err != nil {
			log.Error(err, "failed to patch WxoneMachine")
			return ctrl.Result{}, err
		}
		log.Info("patched without errors")
		return ctrl.Result{}, nil
	} else {
		log.Info("requeueing")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
}

// WXOneClusterToWxOneMachines is a handler.ToRequestsFunc to be used to enqueue
// requests for reconciliation of WXOneMachines.
func (r *WXOneMachineReconciler) WXOneClusterToWXOneMachines(ctx context.Context, o client.Object) []ctrl.Request {
	result := []ctrl.Request{}
	c, ok := o.(*infrav1.WXOneCluster)
	if !ok {
		panic(fmt.Sprintf("Expected a WxOneCluster but got a %T", o))
	}

	cluster, err := util.GetOwnerCluster(ctx, r.Client, c.ObjectMeta)
	switch {
	case apierrors.IsNotFound(err) || cluster == nil:
		return result
	case err != nil:
		return result
	}

	labels := map[string]string{clusterv1.ClusterNameLabel: cluster.Name}
	machineList := &clusterv1.MachineList{}
	if err := r.Client.List(ctx, machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
		return nil
	}
	for _, m := range machineList.Items {
		if m.Spec.InfrastructureRef.Name == "" {
			continue
		}
		name := client.ObjectKey{Namespace: m.Namespace, Name: m.Name}
		result = append(result, ctrl.Request{NamespacedName: name})
	}

	return result
}

// SetupWithManager sets up the controller with the Manager.
func (r *WXOneMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	predicateLog := ctrl.LoggerFrom(ctx).WithValues("controller", "dockermachine")
	clusterToWxOneMachines, err := util.ClusterToTypedObjectsMapper(mgr.GetClient(), &infrav1.WXOneMachineList{}, mgr.GetScheme())
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.WXOneMachine{}).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("DockerMachine"))),
			builder.WithPredicates(predicates.ResourceIsChanged(mgr.GetScheme(), predicateLog)),
		).
		Watches(
			&infrav1.WXOneCluster{},
			handler.EnqueueRequestsFromMapFunc(r.WXOneClusterToWXOneMachines),
			builder.WithPredicates(predicates.ResourceIsChanged(mgr.GetScheme(), predicateLog)),
		).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(clusterToWxOneMachines),
			builder.WithPredicates(predicates.All(mgr.GetScheme(), predicateLog,
				predicates.ResourceIsChanged(mgr.GetScheme(), predicateLog),
				predicates.ClusterPausedTransitionsOrInfrastructureReady(mgr.GetScheme(), predicateLog),
			)),
		).
		Named("wxonemachine").
		Complete(r)
}
