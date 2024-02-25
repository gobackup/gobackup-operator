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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1 "github.com/payamQorbanpour/backup-operator/api/v1"
)

// CronBackupReconciler reconciles a CronBackup object
type CronBackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=backup.github.com,resources=cronbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=backup.github.com,resources=cronbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=backup.github.com,resources=cronbackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CronBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *CronBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Define a CronBackup object
	cronBackup := &backupv1.CronBackup{}

	// Fetch the CronBackup instance
	if err := r.Get(ctx, req.NamespacedName, cronBackup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ensure Storage and Database CRDs existance
	if len(cronBackup.StorageRefs) <= 0 || len(cronBackup.DatabaseRefs) <= 0 {
		return ctrl.Result{}, client.IgnoreNotFound(nil)
	}

	// TODO: Create a secret from goabckup config

	// Create job with the given BackupModel to run 'gobackup perform'
	_, err := r.createBackupJob(ctx)
	if err != nil {
		fmt.Println("Err: ", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.CronBackup{}).
		Complete(r)
}

func (r *CronBackupReconciler) createBackupJob(ctx context.Context) (*batchv1.Job, error) {
	_ = log.FromContext(ctx)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gobackup-job",
			Namespace: "default",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "gobackup",
							Image:   "huacnlee/gobackup",
							Command: []string{"/bin/sh", "-c", "gobackup perform"},
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
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	config, err := clientcmd.BuildConfigFromFlags("", "/Users/payam/.kube/config")
	if err != nil {
		return nil, err
	}

	// Create a clientset from the configuration
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Create the Job
	_, err = clientset.BatchV1().Jobs("default").Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (r *CronBackupReconciler) createBackupCronJob(ctx context.Context) (*batchv1.Job, error) {
	_ = log.FromContext(ctx)

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gobackup-cronjob",
			Namespace: "default",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/1 * * * *", // Runs every minute
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    "gobackup",
									Image:   "huacnlee/gobackup",
									Command: []string{"/bin/sh", "-c", "gobackup perform"},
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

	config, err := clientcmd.BuildConfigFromFlags("", "/Users/payam/.kube/config")
	if err != nil {
		return nil, err
	}

	// Create a clientset from the configuration
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Create the CronJob
	_, err = clientset.BatchV1().CronJobs("default").Create(ctx, cronJob, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}

	return nil, nil
}
