package controller

import (
	"context"
	"fmt"
	"time"

	schedulev1 "github.com/erfan-272758/eifa-replica-operator/api/v1"
	"github.com/gorhill/cronexpr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *EifaReplicaReconciler) GetDesiredReplica(ctx context.Context, req ctrl.Request, eifaReplica *schedulev1.EifaReplica) (*int32, *time.Time, error) {
	// check last status
	next := time.Now().Add(15 * time.Second)

	if eifaReplica.Status.NextTransitionTime != "" {
		// check time
		t, err := time.Parse(time.RFC3339, eifaReplica.Status.NextTransitionTime)
		if err != nil {
			return nil, &next, fmt.Errorf("can not parse .Status.NextTransitionTime, %s", err)
		}
		// set next time base on NextTransitionTime
		next = t

		if time.Now().Before(next) {
			return nil, &next, nil
		}

	}

	cron, err := cronexpr.Parse(eifaReplica.Spec.Schedule)
	if err != nil {
		return nil, &next, fmt.Errorf("can not parse .Spec.Schedule, %s", err)
	}

	// run job
	desiredReplica, err := r.runJob(ctx, req, eifaReplica)
	next = cron.Next(time.Now())

	// job failed
	if err != nil {
		return nil, &next, fmt.Errorf("[run-job] %s", err)
	}
	return &desiredReplica, &next, nil

}
