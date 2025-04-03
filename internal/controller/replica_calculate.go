package controller

import (
	"context"
	"time"

	schedulev1 "github.com/erfan-272758/eifa-replica-operator/api/v1"
	"github.com/gorhill/cronexpr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *EifaReplicaReconciler) GetDesiredReplica(ctx context.Context, req ctrl.Request, eifaReplica *schedulev1.EifaReplica) (int32, error) {
	log := log.FromContext(ctx)
	startAt := time.Now()

	// check last status
	repStatusArr := eifaReplica.Status.ReplicationStatus
	defaultReplica := eifaReplica.Spec.MinReplicas

	if len(repStatusArr) > 0 {
		repStatus := repStatusArr[len(repStatusArr)-1]
		defaultReplica = repStatus.DesiredReplica

		// check time
		t, err := time.Parse(time.RFC3339, repStatus.NextAt)
		if err != nil {
			log.Info("can not parse next time")
			return defaultReplica, err
		}
		if time.Now().Before(t) {
			log.Info("time is not exceeded")
			return defaultReplica, nil
		}

	}

	cron, err := cronexpr.Parse(eifaReplica.Spec.Schedule)
	if err != nil {
		log.Info("schedule expression is invalid")
		return defaultReplica, err
	}

	// run job
	desiredReplica, err := r.runJob(ctx, req, eifaReplica)

	// job failed
	if err != nil {
		log.Info("job has error")
		repStatusArr = append(repStatusArr, schedulev1.ReplicationStatus{
			Status:         schedulev1.JOB_FAILED,
			Reason:         err.Error(),
			StartAt:        startAt.Format(time.RFC3339),
			NextAt:         cron.Next(time.Now()).Format(time.RFC3339),
			CurrentReplica: defaultReplica,
			DesiredReplica: defaultReplica,
		})
		eifaReplica.Status.ReplicationStatus = repStatusArr
		r.Status().Update(ctx, eifaReplica)
		return defaultReplica, err
	}

	// job success
	repStatusArr = append(repStatusArr, schedulev1.ReplicationStatus{
		Status:         schedulev1.JOB_SUCCESS,
		StartAt:        startAt.Format(time.RFC3339),
		NextAt:         cron.Next(time.Now()).Format(time.RFC3339),
		CurrentReplica: defaultReplica,
		DesiredReplica: desiredReplica,
	})
	eifaReplica.Status.ReplicationStatus = repStatusArr
	// update status
	r.Status().Update(ctx, eifaReplica)

	return desiredReplica, nil
}
