/*
Copyright 2025 Erfan Mahvash.

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
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	schedulev1 "github.com/erfan-272758/eifa-replica-operator/api/v1"
)

// EifaReplicaReconciler reconciles a EifaReplica object
type EifaReplicaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=schedule.eifa.org,resources=eifareplicas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=schedule.eifa.org,resources=eifareplicas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=schedule.eifa.org,resources=eifareplicas/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the EifaReplica object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *EifaReplicaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("starting reconciliation")

	// Fetch EifaReplica object
	eifaReplica := &schedulev1.EifaReplica{}
	if err := r.Get(ctx, req.NamespacedName, eifaReplica); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("EifaReplica resource not found. Ignoring since object must be deleted")

			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get EifaReplica")

		return ctrl.Result{}, err

	}

	requeueAfter := 15 * time.Second

	// Calculate desired replicas based on JobTemplate and Scheduler
	desiredReplicas, err := r.GetDesiredReplica(ctx, req, eifaReplica)
	if err != nil {
		log.Error(err, "failed to calculate desired replicas")
		return ctrl.Result{RequeueAfter: requeueAfter}, err
	}

	// Fetch target
	kind := strings.ToLower(eifaReplica.Spec.ScaleTargetRef.Kind)
	if kind != "deployment" || kind == "deploy" {
		log.Info("invalid scale target kind")
		return ctrl.Result{RequeueAfter: requeueAfter}, fmt.Errorf("invalid scale target kind: %s", kind)
	}
	targetObj := &appsv1.Deployment{}

	if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: eifaReplica.Spec.ScaleTargetRef.Name}, targetObj); err != nil {
		log.Error(err, "unable to fetch target")
		return ctrl.Result{RequeueAfter: requeueAfter}, client.IgnoreNotFound(err)
	}

	// Check current replicas against desired replicas
	if *targetObj.Spec.Replicas != desiredReplicas {
		log.Info("updating deployment replicas", "desiredReplicas", desiredReplicas)
		targetObj.Spec.Replicas = &desiredReplicas
		if err := r.Update(ctx, targetObj); err != nil {
			log.Error(err, "failed to update deployment replicas")
			return ctrl.Result{RequeueAfter: requeueAfter}, err
		}
	}

	log.Info("ending reconciliation")
	return ctrl.Result{RequeueAfter: requeueAfter}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *EifaReplicaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulev1.EifaReplica{}).
		Complete(r)
}
