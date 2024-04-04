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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1"
	"github.com/gobackup/gobackup-operator/pkg/k8sutil"
)

// CronBackupReconciler reconciles a CronBackup object
type CronBackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Clientset     *kubernetes.Clientset
	DynamicClient *dynamic.DynamicClient
}

// +kubebuilder:rbac:groups=gobackup.io,resources=cronbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gobackup.io,resources=cronbackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gobackup.io,resources=cronbackups/finalizers,verbs=update
func (r *CronBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Define a CronBackup object
	cronBackup := &backupv1.CronBackup{}

	// Fetch the CronBackup instance
	if err := r.Get(ctx, req.NamespacedName, cronBackup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ensure Storage and Database CRDs existence
	// TODO: Extend this by checking every storage and database
	if len(cronBackup.StorageRefs) == 0 || len(cronBackup.DatabaseRefs) == 0 {
		return ctrl.Result{}, client.IgnoreNotFound(nil)
	}

	err := k8sutil.CreateSecret(ctx, cronBackup.Model, r.Clientset, r.DynamicClient, cronBackup.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create job with the given BackupModel to run 'gobackup perform'
	_, err = r.createBackupCronJob(ctx, cronBackup.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.CronBackup{}).
		Complete(r)
}

// createBackupCronJob creates a cronjob to run the 'gobackup perform'
func (r *CronBackupReconciler) createBackupCronJob(ctx context.Context, namespace string) (*batchv1.CronJob, error) {
	_ = log.FromContext(ctx)

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gobackup-cronjob",
			Namespace: namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/1 * * * *", // Runs every minute
			JobTemplate: batchv1.JobTemplateSpec{
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
											Name:      "gobackup-secret-volume",
											MountPath: "/root/.gobackup",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "gobackup-secret-volume",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "gobackup-secret",
										},
									},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}

	// Create the CronJob
	_, err := r.Clientset.BatchV1().CronJobs(namespace).Create(ctx, cronJob, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return cronJob, nil
}
