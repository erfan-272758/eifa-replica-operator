package controller

import (
	"context"

	schedulev1 "github.com/erfan-272758/eifa-replica-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *EifaReplicaReconciler) GetDesiredReplica(ctx context.Context, req ctrl.Request, eifaReplica *schedulev1.EifaReplica) (int32, error) {
	log := log.FromContext(ctx)
	min := eifaReplica.Spec.MinReplicas
	log.Info("calculate expected replica to min value")
	return int32(min), nil
}
