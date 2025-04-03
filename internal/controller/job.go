package controller

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	schedulev1 "github.com/erfan-272758/eifa-replica-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *EifaReplicaReconciler) runJob(ctx context.Context, req ctrl.Request, eifaReplica *schedulev1.EifaReplica) (int32, error) {
	// 1. init job obj

	// set defaults
	defActiveSec := int64(15)
	defBackoffLim := int32(1)
	completions := int32(1)
	parallelism := int32(1)

	if eifaReplica.Spec.JobTemplate.Spec.ActiveDeadlineSeconds == nil {
		eifaReplica.Spec.JobTemplate.Spec.ActiveDeadlineSeconds = &defActiveSec
	}
	if eifaReplica.Spec.JobTemplate.Spec.BackoffLimit == nil {
		eifaReplica.Spec.JobTemplate.Spec.BackoffLimit = &defBackoffLim
	}

	eifaReplica.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	eifaReplica.Spec.JobTemplate.Spec.Completions = &completions
	eifaReplica.Spec.JobTemplate.Spec.Parallelism = &parallelism

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-job-%s", req.Name, time.Now().Format(time.DateOnly)),
			Namespace:    req.Namespace,
		},
		Spec: eifaReplica.Spec.JobTemplate.Spec,
	}
	// 2. set owner ref
	if err := ctrl.SetControllerReference(eifaReplica, job, r.Scheme); err != nil {
		return 0, fmt.Errorf("can not set owner ref, %s", err)
	}

	// 3. create job
	if err := r.Client.Create(ctx, job); err != nil {
		return 0, fmt.Errorf("can not create job, %s", err)
	}

	// 4. wait for completion
	var compJob batchv1.Job
	interval := 1 * time.Second
	jobKey := client.ObjectKey{Name: job.Name, Namespace: job.Namespace}

	for {
		err := r.Get(ctx, jobKey, &compJob)
		if err != nil {
			return 0, fmt.Errorf("can not get created job, %s", err)
		}
		jobStatus := r.checkJobStatus(&compJob)

		// success pods
		if jobStatus == schedulev1.JOB_SUCCESS {
			break
		}

		// job ends without any success pods
		if jobStatus == schedulev1.JOB_FAILED {
			return 0, fmt.Errorf("job ends without any success pods")
		}

		time.Sleep(interval)
	}

	// 5. read logs to find desired replica
	desiredReplica, err := r.parseJobLogs(ctx, jobKey)
	if err != nil {
		return 0, fmt.Errorf("[parse-job-logs] %s", err)
	}

	return max(eifaReplica.Spec.MinReplicas, min(eifaReplica.Spec.MaxReplicas, desiredReplica)), nil

}

func (r *EifaReplicaReconciler) parseJobLogs(ctx context.Context, jobKey types.NamespacedName) (int32, error) {
	// Step 1: List Pods associated with the Job
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(jobKey.Namespace), client.MatchingLabels{"job-name": jobKey.Name}); err != nil {
		return 0, fmt.Errorf("can not get list of pods: %w", err)
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
		return 0, fmt.Errorf("can not find success pod")
	}

	// Step 3: Read Logs

	clientset, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		return 0, fmt.Errorf("failed to create clientset: %s", err)
	}

	tail := int64(1)
	req := clientset.CoreV1().Pods(jobKey.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{TailLines: &tail})
	// Step 3: Stream logs from the pod
	logs, err := req.Stream(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to stream logs: %s", err)
	}
	defer logs.Close()

	// Step 4: Read logs
	logContent, err := io.ReadAll(logs)
	if err != nil {
		return 0, fmt.Errorf("failed to read logs: %s", err)
	}

	desiredReplica, err := strconv.ParseInt(strings.TrimSpace((string(logContent))), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("can not parse log to int, %s", err)
	}

	return int32(desiredReplica), nil

}

func (r *EifaReplicaReconciler) checkJobStatus(job *batchv1.Job) string {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionStatus(metav1.ConditionTrue) {
			return schedulev1.JOB_SUCCESS // Job completed successfully
		}
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionStatus(metav1.ConditionTrue) {
			return schedulev1.JOB_FAILED // Job failed
		}
	}
	return schedulev1.JOB_RUNNING // Job is still running
}
