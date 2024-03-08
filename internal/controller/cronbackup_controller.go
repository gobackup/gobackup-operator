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
	"os"

	"gopkg.in/yaml.v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1"
	"github.com/gobackup/gobackup-operator/pkg/utils"
)

// CronBackupReconciler reconciles a CronBackup object
type CronBackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// BackupConfig represents the configuration for backups
type BackupConfig struct {
	Models Models `yaml:"models,omitempty"`
}

// Models represents the different models for backup configuration
type Models struct {
	MyBackup MyBackup `yaml:"my_backup,omitempty"`
}

// MyBackup represents the configuration for "my_backup" model
type MyBackup struct {
	Databases Databases `yaml:"databases"`
	Storages  Storages  `yaml:"storages"`
	backupv1.BackupModelSpecConfig
}

// Databases represents the database configurations
type Databases struct {
	Postgres backupv1.PostgreSQLSpecConfig `yaml:"postgres"`
}

// Storages represents the storage configurations
type Storages struct {
	S3 backupv1.S3SpecConfig `yaml:"s3"`
}

//+kubebuilder:rbac:groups=gobackup.io,resources=cronbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gobackup.io,resources=cronbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gobackup.io,resources=cronbackups/finalizers,verbs=update

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

	// Ensure Storage and Database CRDs existence
	// TODO: Extend this by checking every storage and database
	if len(cronBackup.StorageRefs) == 0 || len(cronBackup.DatabaseRefs) == 0 {
		return ctrl.Result{}, client.IgnoreNotFound(nil)
	}

	config, err := clientcmd.BuildConfigFromFlags("", "/Users/payam/.kube/config")
	if err != nil {
		return ctrl.Result{}, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return ctrl.Result{}, err
	}

	examplepsql, err := utils.GetCRD(ctx, dynamicClient, "database.gobackup.io", "v1", "postgresqls", "gobackup-operator-test", "example-postgresql")
	if err != nil {
		return ctrl.Result{}, err
	}

	examples3, err := utils.GetCRD(ctx, dynamicClient, "storage.gobackup.io", "v1", "s3s", "gobackup-operator-test", "example-s3")
	if err != nil {
		return ctrl.Result{}, err
	}

	var postgreSQLSpec backupv1.PostgreSQLSpec

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(examplepsql.Object["spec"].(map[string]interface{}), &postgreSQLSpec); err != nil {
		return ctrl.Result{}, err
	}
	postgreSQLSpec.Type = "postgresql"

	var s3Spec backupv1.S3Spec

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(examples3.Object["spec"].(map[string]interface{}), &s3Spec); err != nil {
		return ctrl.Result{}, err
	}
	s3Spec.Type = "s3"
	backupConfig := BackupConfig{
		Models: Models{
			MyBackup: MyBackup{
				Databases: Databases{
					backupv1.PostgreSQLSpecConfig(postgreSQLSpec),
				},
				Storages: Storages{
					backupv1.S3SpecConfig(s3Spec),
				},
			},
		},
	}

	yamlData, err := yaml.Marshal(&backupConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Write to gobackup.yaml
	err = os.WriteFile("gobackup.yaml", yamlData, 0644)
	if err != nil {
		return ctrl.Result{}, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gobackup-secret",
		},
		StringData: map[string]string{
			"gobackup.yml": string(yamlData),
		},
	}

	// Create a clientset from the configuration
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create the Secret in the specified namespace
	_, err = clientset.CoreV1().Secrets("gobackup-operator-test").Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("-----model--->:\n%s\n", string(yamlData))

	// secretData, _ := yaml.Marshal(backupConfig)
	// secret := &corev1.Secret{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      secretName,
	// 		Namespace: ns,
	// 	},
	// 	Data: map[string][]byte{
	// 		"gobackup.yaml": secretData,
	// 	},
	// }

	// TODO: Create a secret from goabckup config
	// for _, database := range cronBackup.DatabaseRefs {
	// TODO: Fetch the database type instance for example: example-postgres
	// and add it to the gobackup config file
	// }
	// for _, storage := range cronBackup.StorageRefs {
	// TODO: Fetch the storage type instance for example: example-s3
	// and add it to the gobackup config file
	// }

	// Create job with the given BackupModel to run 'gobackup perform'
	_, err = r.createBackupJob(ctx, config, "gobackup-operator-test")
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

// createBackupJob creates a job to run the 'gobackup perform'
func (r *CronBackupReconciler) createBackupJob(ctx context.Context, config *rest.Config, namespace string) (*batchv1.Job, error) {
	_ = log.FromContext(ctx)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gobackup-job",
			Namespace: namespace,
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

	// Create a clientset from the configuration
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Create the Job
	_, err = clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return job, nil
}

// nolint
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
	_, err = clientset.BatchV1().CronJobs(namespace).Create(ctx, cronJob, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}

	return cronJob, nil
}
