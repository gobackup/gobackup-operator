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
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1"
	"github.com/gobackup/gobackup-operator/pkg/k8sutil"
)

// BackupReconciler reconciles a Backup object
type BackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	K8s    *k8sutil.K8s
}

// +kubebuilder:rbac:groups=gobackup.gobackup.io,resources=backups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gobackup.gobackup.io,resources=backups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gobackup.gobackup.io,resources=backups/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
func (r *BackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Backup", "namespace", req.Namespace, "name", req.Name)

	// Define a Backup object
	backup := &backupv1.Backup{}

	// Fetch the Backup instance
	if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if this is a scheduled backup or an immediate one
	if backup.Spec.Schedule != nil && backup.Spec.Schedule.Cron != "" {
		return r.reconcileScheduledBackup(ctx, backup)
	} else {
		return r.reconcileImmediateBackup(ctx, backup)
	}
}

// reconcileScheduledBackup handles backups with a schedule defined
func (r *BackupReconciler) reconcileScheduledBackup(ctx context.Context, backup *backupv1.Backup) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling scheduled backup", "namespace", backup.Namespace, "name", backup.Name)

	// Validate the backup spec
	if err := r.validateBackupSpec(backup); err != nil {
		logger.Error(err, "Invalid backup specification")
		return ctrl.Result{}, err
	}

	// Create the secret that will be used by the CronJob
	if err := r.K8s.CreateSecret(ctx, backup.Spec, backup.Namespace, backup.Name); err != nil {
		logger.Error(err, "Failed to create secret for scheduled backup")
		return ctrl.Result{}, err
	}

	// Check if a CronJob already exists for this backup
	cronJob := &batchv1.CronJob{}
	err := r.Get(ctx, types.NamespacedName{Name: backup.Name, Namespace: backup.Namespace}, cronJob)

	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "Failed to get CronJob")
		return ctrl.Result{}, err
	}

	// Create or update the CronJob
	if errors.IsNotFound(err) {
		// Create a new CronJob
		logger.Info("Creating a new CronJob", "namespace", backup.Namespace, "name", backup.Name)

		cronJob, err = r.createCronJob(ctx, backup)
		if err != nil {
			logger.Error(err, "Failed to create CronJob")
			return ctrl.Result{}, err
		}
	} else {
		// Update the existing CronJob if needed
		logger.Info("Updating existing CronJob", "namespace", backup.Namespace, "name", backup.Name)

		if cronJob.Spec.Schedule != backup.Spec.Schedule.Cron {
			cronJob.Spec.Schedule = backup.Spec.Schedule.Cron

			// Update other fields from the backup spec
			if backup.Spec.Schedule.StartingDeadlineSeconds != nil {
				cronJob.Spec.StartingDeadlineSeconds = backup.Spec.Schedule.StartingDeadlineSeconds
			}

			if backup.Spec.Schedule.SuccessfulJobsHistoryLimit != nil {
				cronJob.Spec.SuccessfulJobsHistoryLimit = backup.Spec.Schedule.SuccessfulJobsHistoryLimit
			}

			if backup.Spec.Schedule.FailedJobsHistoryLimit != nil {
				cronJob.Spec.FailedJobsHistoryLimit = backup.Spec.Schedule.FailedJobsHistoryLimit
			}

			if backup.Spec.Schedule.Suspend != nil {
				cronJob.Spec.Suspend = backup.Spec.Schedule.Suspend
			}

			if err := r.Update(ctx, cronJob); err != nil {
				logger.Error(err, "Failed to update CronJob")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// reconcileImmediateBackup handles backups without a schedule (immediate execution)
func (r *BackupReconciler) reconcileImmediateBackup(ctx context.Context, backup *backupv1.Backup) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling immediate backup", "namespace", backup.Namespace, "name", backup.Name)

	// Validate the backup spec
	if err := r.validateBackupSpec(backup); err != nil {
		logger.Error(err, "Invalid backup specification")
		return ctrl.Result{}, err
	}

	// Check if a job already exists for this backup
	job := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: backup.Name, Namespace: backup.Namespace}, job)

	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Failed to get Job")
			return ctrl.Result{}, err
		}

		// Job doesn't exist yet, create the secret and job
		if err := r.K8s.CreateSecret(ctx, backup.Spec, backup.Namespace, backup.Name); err != nil {
			logger.Error(err, "Failed to create secret")
			return ctrl.Result{}, err
		}

		// Create the job
		_, err = r.createBackupJob(ctx, backup)
		if err != nil {
			logger.Error(err, "Failed to create job")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Job exists, check its status
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			// Job completed successfully
			logger.Info("Backup job completed successfully", "namespace", backup.Namespace, "name", backup.Name)

			// Clean up
			if err := r.deleteBackup(ctx, backup); err != nil {
				logger.Error(err, "Failed to clean up backup")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		} else if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			// Job failed
			logger.Error(fmt.Errorf("backup job failed"), "Backup job failed", "namespace", backup.Namespace, "name", backup.Name)

			// Clean up (or implement retry logic or error reporting)
			if err := r.deleteBackup(ctx, backup); err != nil {
				logger.Error(err, "Failed to clean up backup after failure")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
	}

	// Job is still running
	logger.Info("Backup job is still running", "namespace", backup.Namespace, "name", backup.Name)
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// validateBackupSpec validates that the backup spec is correctly configured
func (r *BackupReconciler) validateBackupSpec(backup *backupv1.Backup) error {
	// Ensure Storage and Database CRDs existence
	if len(backup.Spec.StorageRefs) == 0 {
		return fmt.Errorf("no storage references specified in backup spec")
	}

	if len(backup.Spec.DatabaseRefs) == 0 {
		return fmt.Errorf("no database references specified in backup spec")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.Backup{}).
		Owns(&batchv1.Job{}).
		Owns(&batchv1.CronJob{}).
		Complete(r)
}

// deleteBackup cleans up resources associated with a backup
func (r *BackupReconciler) deleteBackup(ctx context.Context, backup *backupv1.Backup) error {
	logger := log.FromContext(ctx)

	// Delete the secret
	if err := r.K8s.DeleteSecret(ctx, backup.Namespace, backup.Name); err != nil {
		logger.Error(err, "Failed to delete secret")
		return err
	}

	// Delete the job
	if err := r.K8s.DeleteJob(ctx, backup.Namespace, backup.Name); err != nil {
		logger.Error(err, "Failed to delete job")
		return err
	}

	// Delete the backup CR
	if err := r.Delete(ctx, backup); err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Failed to delete backup")
			return err
		}
	}

	return nil
}

// createBackupJob creates a job to run the 'gobackup perform'
func (r *BackupReconciler) createBackupJob(ctx context.Context, backup *backupv1.Backup) (*batchv1.Job, error) {
	logger := log.FromContext(ctx)
	logger.Info("Creating backup job", "namespace", backup.Namespace, "name", backup.Name)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.Name,
			Namespace: backup.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "gobackup",
							Image:           "huacnlee/gobackup",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c", "gobackup perform"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/root/.gobackup",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: backup.Name,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	// Set the Backup instance as the owner of the Job
	if err := controllerutil.SetControllerReference(backup, job, r.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference for job")
		return nil, err
	}

	// Create the Job
	if err := r.Create(ctx, job); err != nil {
		logger.Error(err, "Failed to create job")
		return nil, err
	}

	return job, nil
}

// createCronJob creates a CronJob for scheduled backups
func (r *BackupReconciler) createCronJob(ctx context.Context, backup *backupv1.Backup) (*batchv1.CronJob, error) {
	logger := log.FromContext(ctx)
	logger.Info("Creating CronJob for scheduled backup", "namespace", backup.Namespace, "name", backup.Name)

	// Prepare the job template
	jobTemplate := batchv1.JobTemplateSpec{
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "gobackup",
							Image:           "huacnlee/gobackup",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c", "gobackup perform"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/root/.gobackup",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: backup.Name,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	// Set up the CronJob
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
			SuccessfulJobsHistoryLimit: backup.Spec.Schedule.SuccessfulJobsHistoryLimit,
			FailedJobsHistoryLimit:     backup.Spec.Schedule.FailedJobsHistoryLimit,
			Suspend:                    backup.Spec.Schedule.Suspend,
		},
	}

	// Set default values for optional fields if not specified
	if cronJob.Spec.SuccessfulJobsHistoryLimit == nil {
		var limit int32 = 3
		cronJob.Spec.SuccessfulJobsHistoryLimit = &limit
	}

	if cronJob.Spec.FailedJobsHistoryLimit == nil {
		var limit int32 = 1
		cronJob.Spec.FailedJobsHistoryLimit = &limit
	}

	// Set the Backup instance as the owner of the CronJob
	if err := controllerutil.SetControllerReference(backup, cronJob, r.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference for CronJob")
		return nil, err
	}

	// Create the CronJob
	if err := r.Create(ctx, cronJob); err != nil {
		logger.Error(err, "Failed to create CronJob")
		return nil, err
	}

	return cronJob, nil
}
