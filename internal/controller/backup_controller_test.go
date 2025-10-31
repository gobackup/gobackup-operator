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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1"
	"github.com/gobackup/gobackup-operator/pkg/k8sutil"
)

var _ = Describe("Backup Controller", func() {
	var (
		ctx           context.Context
		cancel        context.CancelFunc
		testNamespace string
		clientset     *kubernetes.Clientset
		dynamicClient *dynamic.DynamicClient
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		testNamespace = "test-" + time.Now().Format("20060102-150405")

		// Create test namespace
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

		// Setup k8s clients
		var err error
		clientset, err = kubernetes.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())

		dynamicClient, err = dynamic.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Clean up namespace
		namespace := &corev1.Namespace{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testNamespace}, namespace)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, namespace)).Should(Succeed())

		cancel()
	})

	Context("When creating an immediate backup", func() {
		It("Should create a Job and Secret", func() {
			// Create PostgreSQL resource
			postgres := createTestPostgreSQL(testNamespace, "test-postgres")
			Expect(k8sClient.Create(ctx, postgres)).Should(Succeed())

			// Create S3 resource
			s3 := createTestS3(testNamespace, "test-s3")
			Expect(k8sClient.Create(ctx, s3)).Should(Succeed())

			// Wait for resources to be ready
			time.Sleep(500 * time.Millisecond)

			// Create Backup resource
			backup := &backupv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup",
					Namespace: testNamespace,
				},
				Spec: backupv1.BackupSpec{
					DatabaseRefs: []backupv1.DatabaseRef{
						{
							APIGroup: "gobackup.io",
							Type:     "PostgreSQL",
							Name:     "test-postgres",
						},
					},
					StorageRefs: []backupv1.StorageRef{
						{
							APIGroup: "gobackup.io",
							Type:     "S3",
							Name:     "test-s3",
							Keep:     5,
							Timeout:  300,
						},
					},
					CompressWith: &backupv1.Compress{
						Type: "gzip",
					},
				},
			}

			Expect(k8sClient.Create(ctx, backup)).Should(Succeed())

			// Setup reconciler
			k8s := &k8sutil.K8s{
				Clientset:     clientset,
				DynamicClient: dynamicClient,
			}

			// Create a scheme that includes our types
			testScheme := runtime.NewScheme()
			backupv1.AddToScheme(testScheme)

			reconciler := &BackupReconciler{
				Client: k8sClient,
				Scheme: testScheme,
				K8s:    k8s,
			}

			// Reconcile
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      backup.Name,
					Namespace: backup.Namespace,
				},
			}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Secret was created
			secret := &corev1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      backup.Name,
					Namespace: testNamespace,
				}, secret)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			Expect(secret.Data).To(HaveKey("gobackup.yml"))
			Expect(secret.StringData).To(BeEmpty()) // Should be Data after creation

			// Verify Job was created
			job := &batchv1.Job{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      backup.Name,
					Namespace: testNamespace,
				}, job)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			Expect(job.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(job.Spec.Template.Spec.Containers[0].Image).To(Equal("huacnlee/gobackup"))
			Expect(job.Spec.Template.Spec.Containers[0].Command).To(ContainElement("gobackup perform"))
		})

		It("Should fail validation if no database refs", func() {
			backup := &backupv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup-invalid",
					Namespace: testNamespace,
				},
				Spec: backupv1.BackupSpec{
					StorageRefs: []backupv1.StorageRef{
						{
							APIGroup: "gobackup.io",
							Type:     "S3",
							Name:     "test-s3",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, backup)).Should(Succeed())

			k8s := &k8sutil.K8s{
				Clientset:     clientset,
				DynamicClient: dynamicClient,
			}

			testScheme := runtime.NewScheme()
			backupv1.AddToScheme(testScheme)

			reconciler := &BackupReconciler{
				Client: k8sClient,
				Scheme: testScheme,
				K8s:    k8s,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      backup.Name,
					Namespace: backup.Namespace,
				},
			}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no database references"))
		})

		It("Should fail validation if no storage refs", func() {
			backup := &backupv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup-invalid-2",
					Namespace: testNamespace,
				},
				Spec: backupv1.BackupSpec{
					DatabaseRefs: []backupv1.DatabaseRef{
						{
							APIGroup: "gobackup.io",
							Type:     "PostgreSQL",
							Name:     "test-postgres",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, backup)).Should(Succeed())

			k8s := &k8sutil.K8s{
				Clientset:     clientset,
				DynamicClient: dynamicClient,
			}

			testScheme := runtime.NewScheme()
			backupv1.AddToScheme(testScheme)

			reconciler := &BackupReconciler{
				Client: k8sClient,
				Scheme: testScheme,
				K8s:    k8s,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      backup.Name,
					Namespace: backup.Namespace,
				},
			}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no storage references"))
		})
	})

	Context("When creating a scheduled backup", func() {
		It("Should create a CronJob and Secret", func() {
			// Create PostgreSQL resource
			postgres := createTestPostgreSQL(testNamespace, "test-postgres-scheduled")
			Expect(k8sClient.Create(ctx, postgres)).Should(Succeed())

			// Create S3 resource
			s3 := createTestS3(testNamespace, "test-s3-scheduled")
			Expect(k8sClient.Create(ctx, s3)).Should(Succeed())

			time.Sleep(500 * time.Millisecond)

			// Create Backup with schedule
			successLimit := int32(3)
			failedLimit := int32(1)
			backup := &backupv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup-scheduled",
					Namespace: testNamespace,
				},
				Spec: backupv1.BackupSpec{
					DatabaseRefs: []backupv1.DatabaseRef{
						{
							APIGroup: "gobackup.io",
							Type:     "PostgreSQL",
							Name:     "test-postgres-scheduled",
						},
					},
					StorageRefs: []backupv1.StorageRef{
						{
							APIGroup: "gobackup.io",
							Type:     "S3",
							Name:     "test-s3-scheduled",
							Keep:     10,
							Timeout:  600,
						},
					},
					CompressWith: &backupv1.Compress{
						Type: "gzip",
					},
					Schedule: &backupv1.BackupSchedule{
						Cron:                       "0 2 * * *",
						SuccessfulJobsHistoryLimit: &successLimit,
						FailedJobsHistoryLimit:     &failedLimit,
					},
				},
			}

			Expect(k8sClient.Create(ctx, backup)).Should(Succeed())

			k8s := &k8sutil.K8s{
				Clientset:     clientset,
				DynamicClient: dynamicClient,
			}

			testScheme := runtime.NewScheme()
			backupv1.AddToScheme(testScheme)

			reconciler := &BackupReconciler{
				Client: k8sClient,
				Scheme: testScheme,
				K8s:    k8s,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      backup.Name,
					Namespace: backup.Namespace,
				},
			}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Secret was created
			secret := &corev1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      backup.Name,
					Namespace: testNamespace,
				}, secret)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			// Verify CronJob was created
			cronJob := &batchv1.CronJob{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      backup.Name,
					Namespace: testNamespace,
				}, cronJob)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			Expect(cronJob.Spec.Schedule).To(Equal("0 2 * * *"))
			Expect(*cronJob.Spec.SuccessfulJobsHistoryLimit).To(Equal(int32(3)))
			Expect(*cronJob.Spec.FailedJobsHistoryLimit).To(Equal(int32(1)))
			Expect(cronJob.Spec.ConcurrencyPolicy).To(Equal(batchv1.ForbidConcurrent))
		})

		It("Should update CronJob when schedule changes", func() {
			// Create resources
			postgres := createTestPostgreSQL(testNamespace, "test-postgres-update")
			Expect(k8sClient.Create(ctx, postgres)).Should(Succeed())
			s3 := createTestS3(testNamespace, "test-s3-update")
			Expect(k8sClient.Create(ctx, s3)).Should(Succeed())
			time.Sleep(500 * time.Millisecond)

			successLimit := int32(5)
			backup := &backupv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup-update",
					Namespace: testNamespace,
				},
				Spec: backupv1.BackupSpec{
					DatabaseRefs: []backupv1.DatabaseRef{
						{
							APIGroup: "gobackup.io",
							Type:     "PostgreSQL",
							Name:     "test-postgres-update",
						},
					},
					StorageRefs: []backupv1.StorageRef{
						{
							APIGroup: "gobackup.io",
							Type:     "S3",
							Name:     "test-s3-update",
						},
					},
					Schedule: &backupv1.BackupSchedule{
						Cron:                       "0 2 * * *",
						SuccessfulJobsHistoryLimit: &successLimit,
					},
				},
			}

			Expect(k8sClient.Create(ctx, backup)).Should(Succeed())

			k8s := &k8sutil.K8s{
				Clientset:     clientset,
				DynamicClient: dynamicClient,
			}

			testScheme := runtime.NewScheme()
			backupv1.AddToScheme(testScheme)

			reconciler := &BackupReconciler{
				Client: k8sClient,
				Scheme: testScheme,
				K8s:    k8s,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      backup.Name,
					Namespace: backup.Namespace,
				},
			}

			// Initial reconcile
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Wait for CronJob creation
			var cronJob *batchv1.CronJob
			Eventually(func() error {
				cronJob = &batchv1.CronJob{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      backup.Name,
					Namespace: testNamespace,
				}, cronJob)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			// Update schedule
			backup.Spec.Schedule.Cron = "0 3 * * *"
			Expect(k8sClient.Update(ctx, backup)).Should(Succeed())

			// Reconcile again
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify CronJob was updated
			Eventually(func() string {
				updatedCronJob := &batchv1.CronJob{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      backup.Name,
					Namespace: testNamespace,
				}, updatedCronJob); err != nil {
					return ""
				}
				return updatedCronJob.Spec.Schedule
			}, time.Second*5, time.Millisecond*500).Should(Equal("0 3 * * *"))
		})
	})
})

// Helper functions to create test resources
func createTestPostgreSQL(namespace, name string) *backupv1.PostgreSQL {
	return &backupv1.PostgreSQL{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gobackup.io/v1",
			Kind:       "PostgreSQL",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: backupv1.PostgreSQLSpec{
			Host:     "postgres.example.com",
			Port:     5432,
			Database: "testdb",
			Username: "testuser",
			Password: "testpass",
		},
	}
}

func createTestS3(namespace, name string) *backupv1.S3 {
	return &backupv1.S3{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gobackup.io/v1",
			Kind:       "S3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: backupv1.S3Spec{
			Bucket:          "test-bucket",
			Region:          "us-east-1",
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
		},
	}
}
