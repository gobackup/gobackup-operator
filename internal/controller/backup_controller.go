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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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

// +kubebuilder:rbac:groups=gobackup.io,resources=backups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gobackup.io,resources=backups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gobackup.io,resources=backups/finalizers,verbs=update
// +kubebuilder:rbac:groups=gobackup.io,resources=postgresqls,verbs=get;list;watch
// +kubebuilder:rbac:groups=gobackup.io,resources=s3s,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
func (r *BackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Backup", "namespace", req.Namespace, "name", req.Name)

	// Define a Backup object
	backup := &backupv1.Backup{}

	// Fetch the Backup instance
	if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Reconcile PVC if persistence is enabled
	if backup.Spec.Persistence != nil && backup.Spec.Persistence.Enabled {
		if err := r.reconcilePVC(ctx, backup); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check if this is a scheduled backup
	if backup.Spec.Schedule == nil || backup.Spec.Schedule.Cron == "" {
		// We only support scheduled backups now as per design decision
		logger.Info("Backup has no schedule defined, ignoring", "name", backup.Name)
		return ctrl.Result{}, nil
	}

	return r.reconcileScheduledBackup(ctx, backup)
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

		_, err = r.createCronJob(ctx, backup)
		if err != nil {
			logger.Error(err, "Failed to create CronJob")
			return ctrl.Result{}, err
		}
	} else {
		// Update the existing CronJob if needed
		logger.Info("Updating existing CronJob", "namespace", backup.Namespace, "name", backup.Name)

		// Check for changes
		var needsUpdate bool

		if cronJob.Spec.Schedule != backup.Spec.Schedule.Cron {
			cronJob.Spec.Schedule = backup.Spec.Schedule.Cron
			needsUpdate = true
		}

		if backup.Spec.Schedule.StartingDeadlineSeconds != nil &&
			(cronJob.Spec.StartingDeadlineSeconds == nil || *cronJob.Spec.StartingDeadlineSeconds != *backup.Spec.Schedule.StartingDeadlineSeconds) {
			cronJob.Spec.StartingDeadlineSeconds = backup.Spec.Schedule.StartingDeadlineSeconds
			needsUpdate = true
		}

		if backup.Spec.Schedule.SuccessfulJobsHistoryLimit != nil &&
			(cronJob.Spec.SuccessfulJobsHistoryLimit == nil || *cronJob.Spec.SuccessfulJobsHistoryLimit != *backup.Spec.Schedule.SuccessfulJobsHistoryLimit) {
			cronJob.Spec.SuccessfulJobsHistoryLimit = backup.Spec.Schedule.SuccessfulJobsHistoryLimit
			needsUpdate = true
		}

		if backup.Spec.Schedule.FailedJobsHistoryLimit != nil &&
			(cronJob.Spec.FailedJobsHistoryLimit == nil || *cronJob.Spec.FailedJobsHistoryLimit != *backup.Spec.Schedule.FailedJobsHistoryLimit) {
			cronJob.Spec.FailedJobsHistoryLimit = backup.Spec.Schedule.FailedJobsHistoryLimit
			needsUpdate = true
		}

		if backup.Spec.Schedule.Suspend != nil &&
			(cronJob.Spec.Suspend == nil || *cronJob.Spec.Suspend != *backup.Spec.Schedule.Suspend) {
			cronJob.Spec.Suspend = backup.Spec.Schedule.Suspend
			needsUpdate = true
		}

		if needsUpdate {
			if err := r.Update(ctx, cronJob); err != nil {
				logger.Error(err, "Failed to update CronJob")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// reconcilePVC ensures a PVC exists for the backup
func (r *BackupReconciler) reconcilePVC(ctx context.Context, backup *backupv1.Backup) error {
	logger := log.FromContext(ctx)
	pvcName := backup.Name

	// Check if PVC exists
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: backup.Namespace}, pvc)
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "Failed to get PVC")
		return err
	}

	if errors.IsNotFound(err) {
		logger.Info("Creating PVC for backup", "namespace", backup.Namespace, "name", pvcName)

		storageClass := backup.Spec.Persistence.StorageClass
		accessMode := corev1.PersistentVolumeAccessMode(backup.Spec.Persistence.AccessMode)
		if accessMode == "" {
			accessMode = corev1.ReadWriteOnce
		}

		size := backup.Spec.Persistence.Size
		if size == "" {
			size = "100Mi"
		}
		storageSize := resource.MustParse(size)

		pvc = &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: backup.Namespace,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{accessMode},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: storageSize,
					},
				},
				StorageClassName: storageClass,
			},
		}

		// Set controller reference
		if err := controllerutil.SetControllerReference(backup, pvc, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, pvc); err != nil {
			logger.Error(err, "Failed to create PVC")
			return err
		}
	}

	return nil
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

// createCronJob creates a CronJob for scheduled backups
func (r *BackupReconciler) createCronJob(ctx context.Context, backup *backupv1.Backup) (*batchv1.CronJob, error) {
	logger := log.FromContext(ctx)
	logger.Info("Creating CronJob for scheduled backup", "namespace", backup.Namespace, "name", backup.Name)

	// Default configuration (no persistence)
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

	// Adjust for persistence
	if backup.Spec.Persistence != nil && backup.Spec.Persistence.Enabled {
		configMountPath = "/etc/gobackup"
		command = []string{"/bin/sh", "-c", "gobackup perform -c /etc/gobackup/gobackup.yml"}

		// Update config mount path
		volumeMounts[0].MountPath = configMountPath

		// Add persistence volume
		volumes = append(volumes, corev1.Volume{
			Name: "persistence",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: backup.Name,
				},
			},
		})

		// Add persistence mount
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "persistence",
			MountPath: "/root/.gobackup",
		})
	}

	// Prepare the job template
	jobTemplate := batchv1.JobTemplateSpec{
		Spec: batchv1.JobSpec{
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
	// Set ConcurrencyPolicy to Forbid to prevent multiple backup jobs from running at the same time
	cronJob.Spec.ConcurrencyPolicy = batchv1.ForbidConcurrent

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
