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
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

// BackupReconciler reconciles a Backup object
type BackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Clientset     *kubernetes.Clientset
	DynamicClient *dynamic.DynamicClient
}

// +kubebuilder:rbac:groups=gobackup.gobackup.io,resources=backups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gobackup.gobackup.io,resources=backups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gobackup.gobackup.io,resources=backups/finalizers,verbs=update
func (r *BackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Define a Backup object
	backup := &backupv1.Backup{}

	// Fetch the Backup instance
	if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ensure Storage and Database CRDs existence
	// TODO: Extend this by checking every storage and database
	if len(backup.StorageRefs) == 0 || len(backup.DatabaseRefs) == 0 {
		return ctrl.Result{}, client.IgnoreNotFound(nil)
	}

	apiversionSplited := strings.Split(backup.APIVersion, "/")
	if len(apiversionSplited) == 0 {
		return ctrl.Result{}, fmt.Errorf("failed to parse APIVersion: %s", backup.APIVersion)
	}

	_, err := k8sutil.GetCRD(ctx, r.DynamicClient, apiversionSplited[0], apiversionSplited[1], "backupmodels", backup.Namespace, backup.BackupModelRef.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Log.Error(err, "BackupModel not found")
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	err = k8sutil.CreateSecret(ctx, backup.Model, r.Clientset, r.DynamicClient, backup.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create job with the given BackupModel to run 'gobackup perform'
	_, err = r.createBackupJob(ctx, backup)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.Backup{}).
		Complete(r)
}

// createBackupJob creates a job to run the 'gobackup perform'
func (r *BackupReconciler) createBackupJob(ctx context.Context, backup *backupv1.Backup) (*batchv1.Job, error) {
	_ = log.FromContext(ctx)

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
									SecretName: backup.BackupModelRef.Name,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	// Create the Job
	_, err := r.Clientset.BatchV1().Jobs(backup.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return job, nil
}
