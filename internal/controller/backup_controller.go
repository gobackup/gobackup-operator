/*
Copyright 2024.

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
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1"
	"github.com/gobackup/gobackup-operator/pkg/k8sutil"
)

// BackupReconciler reconciles a Backup object
type BackupReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	K8s       *k8sutil.K8s
	Clientset *kubernetes.Clientset
}

const (
	// MaxRecentRuns is the maximum number of recent runs to keep in status
	MaxRecentRuns = 5
	// MaxLogSize is the maximum size of logs to store in status (4KB)
	MaxLogSize = 4096
	// MaxMessageSize is the maximum size of message to store in status (1KB)
	MaxMessageSize = 1024
)

// +kubebuilder:rbac:groups=gobackup.io,resources=backups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gobackup.io,resources=backups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gobackup.io,resources=backups/finalizers,verbs=update
// +kubebuilder:rbac:groups=gobackup.io,resources=databases,verbs=get;list;watch
// +kubebuilder:rbac:groups=gobackup.io,resources=storages,verbs=get;list;watch
// +kubebuilder:rbac:groups=gobackup.io,resources=postgresqls,verbs=get;list;watch
// +kubebuilder:rbac:groups=gobackup.io,resources=s3s,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods/log,verbs=get

// Reconcile is the main reconciliation loop for Backup resources.
// It handles the creation and management of CronJobs for scheduled backups.
// It separates create and update operations for better control and logging.
func (r *BackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Backup", "namespace", req.Namespace, "name", req.Name)

	// Fetch the Backup instance
	backup := &backupv1.Backup{}
	if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if Backup is being deleted
	if !backup.DeletionTimestamp.IsZero() {
		logger.Info("Backup is being deleted, skipping reconciliation", "name", backup.Name)
		return ctrl.Result{}, nil
	}

	// Determine if this is a create or update operation
	// Check if a CronJob already exists for this backup
	cronJob := &batchv1.CronJob{}
	err := r.Get(ctx, types.NamespacedName{Name: backup.Name, Namespace: backup.Namespace}, cronJob)
	isCreate := errors.IsNotFound(err)

	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "Failed to check if CronJob exists")
		return ctrl.Result{}, err
	}

	// Route to appropriate handler based on operation type
	if isCreate {
		logger.Info("Handling Backup CREATE operation", "namespace", backup.Namespace, "name", backup.Name)
		return r.handleBackupCreate(ctx, backup)
	} else {
		logger.Info("Handling Backup UPDATE operation", "namespace", backup.Namespace, "name", backup.Name)
		return r.handleBackupUpdate(ctx, backup, cronJob)
	}
}

// handleBackupCreate handles the creation of a new Backup resource.
// This method is called when a Backup CRD is first created.
func (r *BackupReconciler) handleBackupCreate(ctx context.Context, backup *backupv1.Backup) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Processing Backup creation", "namespace", backup.Namespace, "name", backup.Name)

	// Validate that schedule is defined (we only support scheduled backups)
	if backup.Spec.Schedule == nil || strings.TrimSpace(backup.Spec.Schedule.Cron) == "" {
		logger.Info("Backup has no schedule defined, ignoring", "name", backup.Name)
		return ctrl.Result{}, nil
	}

	// Validate cron expression format
	if err := r.validateCronExpression(backup.Spec.Schedule.Cron); err != nil {
		logger.Error(err, "Invalid cron expression during create", "cron", backup.Spec.Schedule.Cron)
		return ctrl.Result{}, err
	}

	// Validate the backup spec
	if err := r.validateBackupSpec(backup); err != nil {
		logger.Error(err, "Invalid backup specification during create")
		return ctrl.Result{}, err
	}

	// Create the secret that will be used by the CronJob
	if err := r.K8s.CreateSecret(ctx, backup.Spec, backup.Namespace, backup.Name); err != nil {
		logger.Error(err, "Failed to create secret for scheduled backup")
		return ctrl.Result{}, err
	}

	// Create a new CronJob
	logger.Info("Creating a new CronJob for Backup", "namespace", backup.Namespace, "name", backup.Name)
	if _, err := r.createCronJob(ctx, backup); err != nil {
		logger.Error(err, "Failed to create CronJob during Backup create")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully created Backup and associated resources", "namespace", backup.Namespace, "name", backup.Name)
	return ctrl.Result{}, nil
}

// handleBackupUpdate handles the update of an existing Backup resource.
// This method is called when a Backup CRD is updated.
func (r *BackupReconciler) handleBackupUpdate(ctx context.Context, backup *backupv1.Backup, existingCronJob *batchv1.CronJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Processing Backup update", "namespace", backup.Namespace, "name", backup.Name)

	// Validate that schedule is defined (we only support scheduled backups)
	if backup.Spec.Schedule == nil || strings.TrimSpace(backup.Spec.Schedule.Cron) == "" {
		logger.Info("Backup has no schedule defined, ignoring update", "name", backup.Name)
		return ctrl.Result{}, nil
	}

	// Validate cron expression format
	if err := r.validateCronExpression(backup.Spec.Schedule.Cron); err != nil {
		logger.Error(err, "Invalid cron expression during update", "cron", backup.Spec.Schedule.Cron)
		return ctrl.Result{}, err
	}

	// Validate the backup spec
	if err := r.validateBackupSpec(backup); err != nil {
		logger.Error(err, "Invalid backup specification during update")
		return ctrl.Result{}, err
	}

	// Update the secret that will be used by the CronJob
	if err := r.K8s.CreateSecret(ctx, backup.Spec, backup.Namespace, backup.Name); err != nil {
		logger.Error(err, "Failed to update secret for scheduled backup")
		return ctrl.Result{}, err
	}

	// Update the existing CronJob if needed
	if updated, err := r.updateCronJobIfNeeded(ctx, existingCronJob, backup); err != nil {
		logger.Error(err, "Failed to update CronJob during Backup update")
		return ctrl.Result{}, err
	} else if updated {
		logger.Info("Successfully updated CronJob for Backup", "namespace", backup.Namespace, "name", backup.Name)
	} else {
		logger.V(1).Info("No changes detected in CronJob for Backup", "namespace", backup.Namespace, "name", backup.Name)
	}

	logger.Info("Successfully processed Backup update", "namespace", backup.Namespace, "name", backup.Name)
	return ctrl.Result{}, nil
}

// validateBackupSpec validates that the backup spec is correctly configured.
// It ensures that at least one storage and one database reference is specified.
func (r *BackupReconciler) validateBackupSpec(backup *backupv1.Backup) error {
	if len(backup.Spec.StorageRefs) == 0 {
		return fmt.Errorf("no storage references specified in backup spec")
	}

	if len(backup.Spec.DatabaseRefs) == 0 {
		return fmt.Errorf("no database references specified in backup spec")
	}

	return nil
}

// validateCronExpression performs basic validation on the cron expression.
// This is a simple validation - Kubernetes CronJob will perform more thorough validation.
func (r *BackupReconciler) validateCronExpression(cron string) error {
	if cron == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}

	// Basic validation: cron expression should have 5 fields (minute hour day month weekday)
	parts := strings.Fields(cron)
	if len(parts) != 5 {
		return fmt.Errorf("invalid cron expression: expected 5 fields, got %d", len(parts))
	}

	return nil
}

// updateCronJobIfNeeded checks if the CronJob needs to be updated based on the Backup spec
// and updates it if necessary. Returns true if an update was performed.
func (r *BackupReconciler) updateCronJobIfNeeded(ctx context.Context, cronJob *batchv1.CronJob, backup *backupv1.Backup) (bool, error) {
	logger := log.FromContext(ctx)
	needsUpdate := false

	// Check and update schedule
	if cronJob.Spec.Schedule != backup.Spec.Schedule.Cron {
		cronJob.Spec.Schedule = backup.Spec.Schedule.Cron
		needsUpdate = true
		logger.V(1).Info("CronJob schedule changed", "old", cronJob.Spec.Schedule, "new", backup.Spec.Schedule.Cron)
	}

	// Check and update StartingDeadlineSeconds
	if !equalInt64Ptr(cronJob.Spec.StartingDeadlineSeconds, backup.Spec.Schedule.StartingDeadlineSeconds) {
		cronJob.Spec.StartingDeadlineSeconds = backup.Spec.Schedule.StartingDeadlineSeconds
		needsUpdate = true
	}

	// Check and update SuccessfulJobsHistoryLimit
	if !equalInt32Ptr(cronJob.Spec.SuccessfulJobsHistoryLimit, backup.Spec.Schedule.SuccessfulJobsHistoryLimit) {
		cronJob.Spec.SuccessfulJobsHistoryLimit = backup.Spec.Schedule.SuccessfulJobsHistoryLimit
		needsUpdate = true
	}

	// Check and update FailedJobsHistoryLimit
	if !equalInt32Ptr(cronJob.Spec.FailedJobsHistoryLimit, backup.Spec.Schedule.FailedJobsHistoryLimit) {
		cronJob.Spec.FailedJobsHistoryLimit = backup.Spec.Schedule.FailedJobsHistoryLimit
		needsUpdate = true
	}

	// Check and update Suspend
	if !equalBoolPtr(cronJob.Spec.Suspend, backup.Spec.Schedule.Suspend) {
		cronJob.Spec.Suspend = backup.Spec.Schedule.Suspend
		needsUpdate = true
	}

	// Check if job template needs update (this is a simplified check)
	// In a production system, you might want to do a deep comparison
	expectedTemplate := r.buildJobTemplate(backup)
	if !jobTemplatesEqual(&cronJob.Spec.JobTemplate, &expectedTemplate) {
		cronJob.Spec.JobTemplate = expectedTemplate
		needsUpdate = true
		logger.V(1).Info("CronJob job template changed")
	}

	if needsUpdate {
		if err := r.Update(ctx, cronJob); err != nil {
			return false, fmt.Errorf("failed to update CronJob: %w", err)
		}
		return true, nil
	}

	return false, nil
}

// equalInt64Ptr compares two *int64 pointers for equality, handling nil cases.
func equalInt64Ptr(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// equalInt32Ptr compares two *int32 pointers for equality, handling nil cases.
func equalInt32Ptr(a, b *int32) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// equalBoolPtr compares two *bool pointers for equality, handling nil cases.
func equalBoolPtr(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// jobTemplatesEqual performs a basic comparison of two JobTemplateSpecs.
// This is a simplified comparison - in production you might want a deeper comparison.
func jobTemplatesEqual(a, b *batchv1.JobTemplateSpec) bool {
	// Compare container image and command
	if len(a.Spec.Template.Spec.Containers) != len(b.Spec.Template.Spec.Containers) {
		return false
	}
	if len(a.Spec.Template.Spec.Containers) > 0 {
		containerA := a.Spec.Template.Spec.Containers[0]
		containerB := b.Spec.Template.Spec.Containers[0]
		if containerA.Image != containerB.Image {
			return false
		}
		if !stringSlicesEqual(containerA.Command, containerB.Command) {
			return false
		}
	}
	return true
}

// stringSlicesEqual compares two string slices for equality.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// buildJobTemplate creates a JobTemplateSpec from the Backup spec.
func (r *BackupReconciler) buildJobTemplate(backup *backupv1.Backup) batchv1.JobTemplateSpec {
	imageName := "huacnlee/gobackup:latest"
	command := []string{"/bin/sh", "-c", "gobackup perform"}
	configMountPath := "/root/.gobackup"

	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: backup.Name,
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: configMountPath,
		},
	}

	// Set TTLSecondsAfterFinished to automatically clean up completed/failed Jobs
	// Hardcoded to 1 second (1 second)
	ttlSecondsAfterFinished := int32(1)

	return batchv1.JobTemplateSpec{
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "gobackup",
							Image:           imageName,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         command,
							VolumeMounts:    volumeMounts,
						},
					},
					Volumes:       volumes,
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.Backup{}).
		Owns(&batchv1.CronJob{}).
		Watches(&batchv1.Job{}, handler.EnqueueRequestsFromMapFunc(r.findBackupForJob)).
		Complete(r)
}

// findBackupForJob maps a Job to the Backup that owns it (via CronJob)
func (r *BackupReconciler) findBackupForJob(ctx context.Context, obj client.Object) []ctrl.Request {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return nil
	}

	// Find the CronJob owner
	var cronJobName string
	for _, ref := range job.OwnerReferences {
		if ref.Kind == "CronJob" {
			cronJobName = ref.Name
			break
		}
	}

	if cronJobName == "" {
		return nil
	}

	// The CronJob name is the same as the Backup name
	return []ctrl.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      cronJobName,
				Namespace: job.Namespace,
			},
		},
	}
}

// createCronJob creates a CronJob for scheduled backups.
// It sets up the job template, schedule, and other CronJob-specific configurations.
func (r *BackupReconciler) createCronJob(ctx context.Context, backup *backupv1.Backup) (*batchv1.CronJob, error) {
	logger := log.FromContext(ctx)
	logger.Info("Creating CronJob for scheduled backup", "namespace", backup.Namespace, "name", backup.Name)

	// Build the job template
	jobTemplate := r.buildJobTemplate(backup)

	// Set default values for optional fields
	var successfulLimit int32 = 3
	var failedLimit int32 = 1

	if backup.Spec.Schedule.SuccessfulJobsHistoryLimit != nil {
		successfulLimit = *backup.Spec.Schedule.SuccessfulJobsHistoryLimit
	}
	if backup.Spec.Schedule.FailedJobsHistoryLimit != nil {
		failedLimit = *backup.Spec.Schedule.FailedJobsHistoryLimit
	}

	// Create the CronJob
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.Name,
			Namespace: backup.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                   backup.Spec.Schedule.Cron,
			JobTemplate:                jobTemplate,
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			StartingDeadlineSeconds:    backup.Spec.Schedule.StartingDeadlineSeconds,
			SuccessfulJobsHistoryLimit: &successfulLimit,
			FailedJobsHistoryLimit:     &failedLimit,
			Suspend:                    backup.Spec.Schedule.Suspend,
		},
	}

	// Set the Backup instance as the owner of the CronJob
	if err := controllerutil.SetControllerReference(backup, cronJob, r.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference for CronJob: %w", err)
	}

	// Create the CronJob
	if err := r.Create(ctx, cronJob); err != nil {
		return nil, fmt.Errorf("failed to create CronJob: %w", err)
	}

	return cronJob, nil
}

// reconcileJobStatus updates the Backup status based on the state of related Jobs
func (r *BackupReconciler) reconcileJobStatus(ctx context.Context, backup *backupv1.Backup) error {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling job status", "backup", backup.Name)

	// List all Jobs owned by the CronJob (which is owned by the Backup)
	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList, client.InNamespace(backup.Namespace)); err != nil {
		return fmt.Errorf("failed to list jobs: %w", err)
	}

	logger.Info("Found jobs in namespace", "count", len(jobList.Items), "namespace", backup.Namespace)

	// Filter jobs that belong to this backup's CronJob
	var relatedJobs []batchv1.Job
	for _, job := range jobList.Items {
		// Jobs created by CronJob have the CronJob name as a prefix
		if strings.HasPrefix(job.Name, backup.Name+"-") {
			relatedJobs = append(relatedJobs, job)
			logger.V(1).Info("Found related job", "job", job.Name, "phase", r.getJobPhase(&job))
		}
	}

	logger.Info("Found related jobs for backup", "backup", backup.Name, "count", len(relatedJobs))

	if len(relatedJobs) == 0 {
		logger.Info("No related jobs found, skipping status update")
		return nil
	}

	// Find the most recent job
	var latestJob *batchv1.Job
	for i := range relatedJobs {
		job := &relatedJobs[i]
		if latestJob == nil || job.CreationTimestamp.After(latestJob.CreationTimestamp.Time) {
			latestJob = job
		}
	}

	if latestJob == nil {
		return nil
	}

	logger.Info("Latest job found", "job", latestJob.Name, "phase", r.getJobPhase(latestJob))

	// Get the current phase
	currentPhase := r.getJobPhase(latestJob)

	// Check if this job has already been processed with the same phase
	if backup.Status.LastRun != nil && backup.Status.LastRun.JobName == latestJob.Name {
		if backup.Status.LastRun.Phase == currentPhase {
			logger.V(1).Info("Job already processed with same phase, skipping", "job", latestJob.Name, "phase", currentPhase)
			return nil // Already up to date
		}
		logger.Info("Job phase changed, updating status", "job", latestJob.Name, "oldPhase", backup.Status.LastRun.Phase, "newPhase", currentPhase)
	} else {
		logger.Info("New job detected, updating status", "job", latestJob.Name, "phase", currentPhase)
	}

	// Build the run status
	runStatus := r.buildRunStatus(ctx, latestJob)

	// Determine if we should increment counters (only when transitioning to a terminal state)
	shouldIncrementCounters := false
	if backup.Status.LastRun == nil || backup.Status.LastRun.JobName != latestJob.Name {
		// This is a new job we haven't seen before
		shouldIncrementCounters = (currentPhase == "Succeeded" || currentPhase == "Failed")
	} else if backup.Status.LastRun.Phase != currentPhase {
		// Phase changed - only increment if transitioning TO a terminal state
		shouldIncrementCounters = (currentPhase == "Succeeded" || currentPhase == "Failed")
	}

	// Update the backup status
	statusCopy := backup.Status.DeepCopy()
	now := metav1.Now()

	statusCopy.LastBackupTime = &now
	statusCopy.Phase = runStatus.Phase
	statusCopy.LastRun = &runStatus

	// Update counters and timestamps based on phase (only once per job completion)
	if shouldIncrementCounters {
		if runStatus.Phase == "Succeeded" {
			statusCopy.LastSuccessfulBackupTime = &now
			statusCopy.SuccessCount++
			statusCopy.FailureCount = 0 // Reset consecutive failures
			logger.Info("Incrementing success count", "newCount", statusCopy.SuccessCount)
		} else if runStatus.Phase == "Failed" {
			statusCopy.FailureCount++
			logger.Info("Incrementing failure count", "newCount", statusCopy.FailureCount)
		}
	}

	// Add to recent runs (sliding window)
	statusCopy.RecentRuns = r.addToRecentRuns(statusCopy.RecentRuns, runStatus)

	// Update the status
	backup.Status = *statusCopy
	if err := r.Status().Update(ctx, backup); err != nil {
		if errors.IsConflict(err) {
			logger.Info("Conflict updating backup status, will retry on next reconciliation")
			return nil // Will be retried on next reconciliation
		}
		logger.Error(err, "Failed to update backup status")
		return fmt.Errorf("failed to update backup status: %w", err)
	}

	logger.Info("Successfully updated backup status", "phase", runStatus.Phase, "job", latestJob.Name)
	return nil
}

// getJobPhase returns the phase of a Job
func (r *BackupReconciler) getJobPhase(job *batchv1.Job) string {
	if job.Status.Succeeded > 0 {
		return "Succeeded"
	}
	if job.Status.Failed > 0 {
		return "Failed"
	}
	if job.Status.Active > 0 {
		return "Running"
	}
	return "Pending"
}

// buildRunStatus creates a BackupRunStatus from a Job
func (r *BackupReconciler) buildRunStatus(ctx context.Context, job *batchv1.Job) backupv1.BackupRunStatus {
	logger := log.FromContext(ctx)

	runStatus := backupv1.BackupRunStatus{
		JobName:   job.Name,
		StartTime: job.Status.StartTime,
		Phase:     r.getJobPhase(job),
	}

	if job.Status.CompletionTime != nil {
		runStatus.CompletionTime = job.Status.CompletionTime
	}

	// Set message based on job conditions
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			runStatus.Message = truncateString("Backup completed successfully", MaxMessageSize)
		} else if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			runStatus.Message = truncateString(fmt.Sprintf("Backup failed: %s", condition.Message), MaxMessageSize)
		}
	}

	// Collect logs only on failure to save space
	if runStatus.Phase == "Failed" && r.Clientset != nil {
		logs, err := r.collectPodLogs(ctx, job)
		if err != nil {
			logger.V(1).Info("Failed to collect pod logs", "error", err)
			runStatus.Logs = truncateString(fmt.Sprintf("Failed to collect logs: %v", err), MaxLogSize)
		} else {
			runStatus.Logs = logs
		}
	}

	return runStatus
}

// collectPodLogs collects logs from pods belonging to a Job
func (r *BackupReconciler) collectPodLogs(ctx context.Context, job *batchv1.Job) (string, error) {
	if r.Clientset == nil {
		return "", fmt.Errorf("clientset not available")
	}

	// List pods for this job
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList,
		client.InNamespace(job.Namespace),
		client.MatchingLabels{"job-name": job.Name}); err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	if len(podList.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s", job.Name)
	}

	// Get logs from the first (usually only) pod
	pod := &podList.Items[0]

	// Get container logs
	tailLines := int64(100) // Last 100 lines
	podLogOpts := &corev1.PodLogOptions{
		Container: "gobackup",
		TailLines: &tailLines,
	}

	req := r.Clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("failed to read pod logs: %w", err)
	}

	return truncateString(buf.String(), MaxLogSize), nil
}

// addToRecentRuns adds a run to the recent runs list, maintaining the max size
func (r *BackupReconciler) addToRecentRuns(recentRuns []backupv1.BackupRunStatus, newRun backupv1.BackupRunStatus) []backupv1.BackupRunStatus {
	// Check if this run already exists (by job name)
	for i, run := range recentRuns {
		if run.JobName == newRun.JobName {
			// Update existing entry
			recentRuns[i] = newRun
			return recentRuns
		}
	}

	// Add new run at the beginning
	recentRuns = append([]backupv1.BackupRunStatus{newRun}, recentRuns...)

	// Trim to max size
	if len(recentRuns) > MaxRecentRuns {
		recentRuns = recentRuns[:MaxRecentRuns]
	}

	return recentRuns
}

// truncateString truncates a string to maxLen, adding a prefix indicator if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	prefix := "...[truncated]...\n"
	return prefix + s[len(s)-maxLen+len(prefix):]
}
