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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	infrav1 "github.com/wx-one/cluster-api-provider-wxone/api/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// WXOneClusterReconciler reconciles a WXOneCluster object
type WXOneClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=wxoneclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=wxoneclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=wxoneclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WXOneCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *WXOneClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	wxoneCluster := &infrav1.WXOneCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, wxoneCluster); err != nil {
		// 	import apierrors "k8s.io/apimachinery/pkg/api/errors"
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	cluster, err := util.GetOwnerCluster(ctx, r.Client, wxoneCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err

	}
	if cluster == nil {
		log.Info("Waiting for Cluster Controller to set OwnerRef on WxoneCluster")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("Cluster", klog.KObj(cluster))
	ctx = ctrl.LoggerInto(ctx, log)

	// Handle deleted clusters
	if !wxoneCluster.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.reconcileDelete(wxoneCluster, log, ctx)
	}

	// Handle non-deleted clusters
	return ctrl.Result{}, r.reconcileNormal(wxoneCluster, log, ctx)
}

func (r *WXOneClusterReconciler) reconcileNormal(wxoneCluster *infrav1.WXOneCluster, log logr.Logger, ctx context.Context) error {
	log.Info("starting reconcile normal")

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(wxoneCluster, r.Client)
	if err != nil {
		return err
	}

	controllerutil.AddFinalizer(wxoneCluster, infrav1.ClusterFinalizer)
	log.Info("patch cluster")
	if err := patchHelper.Patch(ctx, wxoneCluster); err != nil {
		log.Error(err, "failed to patch WxoneCluster")
		return err
	}
	log.Info("patched without errors")

	wxOneClients, err := NewWXOneClients(log, ctx)
	if err != nil {
		log.Error(err, "failed to create WXOneClients")
		return err
	}

	if wxoneCluster.Status.Project.ResourceID == "" {
		defaultProject, err := getDefaultProject(ctx, wxOneClients.graphqlClient)
		if err != nil {
			log.Error(err, "failed to retrieve default Project")
			return err
		}

		wxoneCluster.Status.Project.ResourceID = defaultProject.GetDefaultProject.Msg.Id
	}

	subnetInput := make([]SubnetInput, len(wxoneCluster.Spec.Network.Subnets))
	for i, item := range wxoneCluster.Spec.Network.Subnets {
		subnetInput[i] = SubnetInput{
			Name:      item.Name,
			IpVersion: item.IPVersion,
			Cidr:      item.CIDR,
		}
	}

	if wxoneCluster.Status.Network.SubnetReference.ResourceID == "" {
		log.Info("creating Cluster Network")
		network, err := createNetwork(ctx, wxOneClients.graphqlClient, wxoneCluster.Spec.Network.Name, AvailabilityZone(wxoneCluster.Spec.AvailabilityZone), wxoneCluster.Status.Project.ResourceID, subnetInput)

		if err != nil {
			log.Error(err, "failed to create Cluster Network")
			return err
		}

		wxoneCluster.Status.Network.ResourceID = network.CreateNetwork.Msg.Id
		wxoneCluster.Status.Network.SubnetReference.ResourceID = network.CreateNetwork.Msg.Subnets[0].Id
		log.Info("patch cluster")
		if err := patchHelper.Patch(ctx, wxoneCluster); err != nil {
			log.Error(err, "failed to patch WxoneCluster")
			return err
		}
		log.Info("patched without errors")
	}

	if wxoneCluster.Status.FloatingIP.ResourceID == "" {
		log.Info("creating Floating IP")
		network, err := createFloatingIP(ctx, wxOneClients.graphqlClient, wxoneCluster.Status.Project.ResourceID)

		if err != nil {
			log.Error(err, "failed to create Floating IP")
			return err
		}

		wxoneCluster.Status.FloatingIP.ResourceID = network.CreateFloatingIP.Msg.Id
		wxoneCluster.Status.FloatingIP.IP = network.CreateFloatingIP.Msg.Ip
		log.Info("patch cluster")
		if err := patchHelper.Patch(ctx, wxoneCluster); err != nil {
			log.Error(err, "failed to patch WxoneCluster")
			return err
		}
		log.Info("patched without errors")
	}

	if wxoneCluster.Status.SSHKey.ResourceID == "" {
		log.Info("creating SSH Key")
		sshKey, err := createKey(ctx, wxOneClients.graphqlClient, wxoneCluster.Spec.Network.Name, wxoneCluster.Spec.SSHKey.PublicKey, wxoneCluster.Status.Project.ResourceID, wxoneCluster.Spec.SSHKey.ProjectWide)

		if err != nil {
			log.Error(err, "failed to create SSH Key for Cluster Machine access")
			return err
		}

		wxoneCluster.Status.SSHKey.ResourceID = sshKey.CreateKey.Msg.Id
		log.Info("patch cluster")
		if err := patchHelper.Patch(ctx, wxoneCluster); err != nil {
			log.Error(err, "failed to patch WxoneCluster")
			return err
		}
		log.Info("patched without errors")
	}

	if wxoneCluster.Status.Network.SubnetReference.ResourceID != "" &&
		wxoneCluster.Status.FloatingIP.ResourceID != "" &&
		wxoneCluster.Status.SSHKey.ResourceID != "" {

		log.Info("set control plane endpoint")
		wxoneCluster.Spec.ControlPlaneEndpoint.Host = wxoneCluster.Status.FloatingIP.IP
		wxoneCluster.Spec.ControlPlaneEndpoint.Port = 443

		// Mark the wxoneCluster ready
		log.Info("mark cluster ready")
		wxoneCluster.Status.Ready = true
		log.Info("patch cluster")
		if err := patchHelper.Patch(ctx, wxoneCluster); err != nil {
			log.Error(err, "failed to patch WxoneCluster")
			return err
		}
		log.Info("patched without errors")
	}

	return nil
}

func (r *WXOneClusterReconciler) reconcileDelete(wxoneCluster *infrav1.WXOneCluster, log logr.Logger, ctx context.Context) error {
	log.Info("starting reconcile delete")
	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(wxoneCluster, r.Client)
	if err != nil {
		return err
	}

	wxOneClients, err := NewWXOneClients(log, ctx)
	if err != nil {
		log.Error(err, "failed to create WXOneClients")
		return err
	}

	if wxoneCluster.Status.Network.SubnetReference.ResourceID != "" {
		log.Info("deleting network", "network", wxoneCluster.Status.Network.SubnetReference.ResourceID)
		_, err := deleteNetwork(ctx, wxOneClients.graphqlClient, wxoneCluster.Status.Network.ResourceID, wxoneCluster.Status.Project.ResourceID)

		if err != nil {
			log.Error(err, "failed to delete cluster network")
			return err
		}

		wxoneCluster.Status.Network.SubnetReference.ResourceID = ""
		wxoneCluster.Status.Network.ResourceID = ""
		log.Info("patch cluster")
		if err := patchHelper.Patch(ctx, wxoneCluster); err != nil {
			log.Error(err, "failed to patch WxoneCluster")
			return err
		}
		log.Info("patched without errors")
	}

	if wxoneCluster.Status.FloatingIP.ResourceID != "" {
		log.Info("deleting floating ip", "floating ip", wxoneCluster.Status.FloatingIP.ResourceID)
		_, err := deleteFloatingIP(ctx, wxOneClients.graphqlClient, wxoneCluster.Status.FloatingIP.ResourceID, wxoneCluster.Status.Project.ResourceID)

		if err != nil {
			log.Error(err, "failed to delete floating ip")
			return err
		}

		wxoneCluster.Status.FloatingIP.ResourceID = ""
		log.Info("patch cluster")
		if err := patchHelper.Patch(ctx, wxoneCluster); err != nil {
			log.Error(err, "failed to patch WxoneCluster")
			return err
		}
		log.Info("patched without errors")
	}

	if wxoneCluster.Status.SSHKey.ResourceID != "" {
		log.Info("deleting ssh key", "ssh key", wxoneCluster.Status.SSHKey.ResourceID)
		_, err := deleteKey(ctx, wxOneClients.graphqlClient, wxoneCluster.Status.SSHKey.ResourceID, wxoneCluster.Status.Project.ResourceID)

		if err != nil {
			log.Error(err, "failed to delete ssh key")
			return err
		}

		wxoneCluster.Status.SSHKey.ResourceID = ""
		log.Info("patch cluster")
		if err := patchHelper.Patch(ctx, wxoneCluster); err != nil {
			log.Error(err, "failed to patch WxoneCluster")
			return err
		}
		log.Info("patched without errors")
	}

	if wxoneCluster.Status.SSHKey.ResourceID == "" && wxoneCluster.Status.Network.SubnetReference.ResourceID == "" && wxoneCluster.Status.FloatingIP.ResourceID == "" {
		log.Info("removing finalizer")
		// Cluster is deleted so remove the finalizer.
		controllerutil.RemoveFinalizer(wxoneCluster, infrav1.ClusterFinalizer)
		log.Info("patch cluster")
		if err := patchHelper.Patch(ctx, wxoneCluster); err != nil {
			log.Error(err, "failed to patch WxoneCluster")
			return err
		}
		log.Info("patched without errors")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WXOneClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.WXOneCluster{}).
		Named("wxonecluster").
		Complete(r)
}
