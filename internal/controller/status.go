package controller

import (
	"context"
	"time"

	schedulev1 "github.com/erfan-272758/eifa-replica-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *EifaReplicaReconciler) UpdateStatus(ctx context.Context, eifaReplica *schedulev1.EifaReplica, cond *metav1.Condition, next *time.Time) error {
	if cond == nil && next == nil {
		// nothing to do
		return nil
	}
	if cond != nil {
		// append
		eifaReplica.Status.Conditions = append(eifaReplica.Status.Conditions, *cond)

		// store only last 10 conditions
		if len(eifaReplica.Status.Conditions) > 10 {
			eifaReplica.Status.Conditions = eifaReplica.Status.Conditions[len(eifaReplica.Status.Conditions)-10:]
		}
	}

	if next != nil {
		eifaReplica.Status.NextTransitionTime = next.Format(time.RFC3339)
	}

	return r.Status().Update(ctx, eifaReplica)
}
