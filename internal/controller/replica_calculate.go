package controller

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	schedulev1 "github.com/erfan-272758/eifa-replica-operator/api/v1"
	"github.com/gorhill/cronexpr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
			StartAt:        startAt.String(),
			NextAt:         cron.Next(time.Now()).String(),
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

func (r *EifaReplicaReconciler) runJob(ctx context.Context, req ctrl.Request, eifaReplica *schedulev1.EifaReplica) (int32, error) {
	// 1. init job obj
	log := log.FromContext(ctx)
	activeSec := int64(15)
	backoffLim := int32(1)

	// set defaults
	if eifaReplica.Spec.JobTemplate.Spec.ActiveDeadlineSeconds == nil {
		eifaReplica.Spec.JobTemplate.Spec.ActiveDeadlineSeconds = &activeSec
	}
	if eifaReplica.Spec.JobTemplate.Spec.BackoffLimit == nil {
		eifaReplica.Spec.JobTemplate.Spec.BackoffLimit = &backoffLim
	}

	eifaReplica.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	eifaReplica.Spec.JobTemplate.CreationTimestamp = metav1.Now()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-job-%s", req.Name, time.Now().Format(time.RFC3339)),
			Namespace: req.Namespace,
		},
		Spec: eifaReplica.Spec.JobTemplate.Spec,
	}
	// 2. set owner ref
	if err := ctrl.SetControllerReference(eifaReplica, job, r.Scheme); err != nil {
		log.Info("can not set owner ref")
		return 0, err
	}

	// 3. create job
	if err := r.Client.Create(ctx, job); err != nil {
		log.Info("can not create job")
		return 0, err
	}

	// 4. wait for completion
	var compJob batchv1.Job
	interval := 1 * time.Second
	jobKey := client.ObjectKey{Name: job.Name, Namespace: job.Namespace}

	for {
		err := r.Get(ctx, jobKey, &compJob)
		if err != nil {
			log.Info("can not get job")
			return 0, err
		}

		// success pods
		if compJob.Status.Succeeded > 0 {
			break
		}

		// job ends without any success pods
		if compJob.Status.Active == 0 {
			log.Info("job ends without any success pods")
			return 0, fmt.Errorf("job ends without any success pods")
		}

		time.Sleep(interval)
	}

	// 5. read logs to find desired replica
	desiredReplica, err := r.parseJobLogs(ctx, jobKey)
	if err != nil {
		log.Info("can not parse pod logs")
		return 0, err
	}

	return max(eifaReplica.Spec.MinReplicas, min(eifaReplica.Spec.MaxReplicas), desiredReplica), nil

}

func (r *EifaReplicaReconciler) parseJobLogs(ctx context.Context, jobKey types.NamespacedName) (int32, error) {
	log := log.FromContext(ctx)

	// Step 1: List Pods associated with the Job
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(jobKey.Namespace), client.MatchingLabels{"job-name": jobKey.Name}); err != nil {
		log.Info("failed to list pods")
		return 0, fmt.Errorf("failed to list pods: %w", err)
	}

	// Step 2: Find Success Pod
	var pod *corev1.Pod
	for _, p := range podList.Items {
		if p.Status.Phase == corev1.PodSucceeded {
			pod = &p
			break
		}
	}

	if pod == nil {
		log.Info("can not find success pod")
		return 0, fmt.Errorf("can not find success pod")
	}

	// Step 3: Read Logs

	clientset, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		return 0, fmt.Errorf("failed to create clientset: %w", err)
	}

	tail := int64(1)
	req := clientset.CoreV1().Pods(jobKey.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{TailLines: &tail})
	// Step 3: Stream logs from the pod
	logs, err := req.Stream(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to stream logs: %w", err)
	}
	defer logs.Close()

	// Step 4: Read logs
	logContent, err := io.ReadAll(logs)
	if err != nil {
		return 0, fmt.Errorf("failed to read logs: %w", err)
	}

	desiredReplica, err := strconv.ParseInt(strings.TrimSpace((string(logContent))), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("can not parse log to int, %s", err.Error())
	}

	return int32(desiredReplica), nil

}
