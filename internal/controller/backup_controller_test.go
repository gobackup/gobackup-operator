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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
		testNamespace = fmt.Sprintf("test-%s-%d", time.Now().Format("20060102-150405"), time.Now().Nanosecond())

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

	It("Should update CronJob suspend state", func() {
		// Create resources
		postgres := createTestPostgreSQL(testNamespace, "test-postgres-suspend")
		Expect(k8sClient.Create(ctx, postgres)).Should(Succeed())
		s3 := createTestS3(testNamespace, "test-s3-suspend")
		Expect(k8sClient.Create(ctx, s3)).Should(Succeed())
		time.Sleep(500 * time.Millisecond)

		// Create Backup
		suspend := false
		backup := &backupv1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-backup-suspend",
				Namespace: testNamespace,
			},
			Spec: backupv1.BackupSpec{
				DatabaseRefs: []backupv1.DatabaseRef{
					{
						APIGroup: "gobackup.io",
						Type:     "PostgreSQL",
						Name:     "test-postgres-suspend",
					},
				},
				StorageRefs: []backupv1.StorageRef{
					{
						APIGroup: "gobackup.io",
						Type:     "S3",
						Name:     "test-s3-suspend",
					},
				},
				Schedule: &backupv1.BackupSchedule{
					Cron:    "0 2 * * *",
					Suspend: &suspend,
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
		cronJob := &batchv1.CronJob{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{
				Name:      backup.Name,
				Namespace: testNamespace,
			}, cronJob)
		}, time.Second*5, time.Millisecond*500).Should(Succeed())

		Expect(*cronJob.Spec.Suspend).To(BeFalse())

		// Update suspend to true
		suspendTrue := true
		backup.Spec.Schedule.Suspend = &suspendTrue
		Expect(k8sClient.Update(ctx, backup)).Should(Succeed())

		// Reconcile again
		_, err = reconciler.Reconcile(ctx, req)
		Expect(err).NotTo(HaveOccurred())

		// Verify CronJob was updated
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      backup.Name,
				Namespace: testNamespace,
			}, cronJob); err != nil {
				return false
			}
			return *cronJob.Spec.Suspend
		}, time.Second*5, time.Millisecond*500).Should(BeTrue())
	})

	Context("When creating a backup with persistence enabled", func() {
		It("Should create a PVC and Job with persistence configured", func() {
			// Create resources
			postgres := createTestPostgreSQL(testNamespace, "test-postgres-persist")
			Expect(k8sClient.Create(ctx, postgres)).Should(Succeed())
			s3 := createTestS3(testNamespace, "test-s3-persist")
			Expect(k8sClient.Create(ctx, s3)).Should(Succeed())
			time.Sleep(500 * time.Millisecond)

			// Create Backup with persistence
			backup := &backupv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup-persist",
					Namespace: testNamespace,
				},
				Spec: backupv1.BackupSpec{
					DatabaseRefs: []backupv1.DatabaseRef{
						{
							APIGroup: "gobackup.io",
							Type:     "PostgreSQL",
							Name:     "test-postgres-persist",
						},
					},
					StorageRefs: []backupv1.StorageRef{
						{
							APIGroup: "gobackup.io",
							Type:     "S3",
							Name:     "test-s3-persist",
						},
					},
					Persistence: &backupv1.Persistence{
						Enabled: true,
						Size:    "200Mi",
					},
					Schedule: &backupv1.BackupSchedule{
						Cron:                       "0 2 * * *",
						SuccessfulJobsHistoryLimit: new(int32),
					},
				},
			}
			*backup.Spec.Schedule.SuccessfulJobsHistoryLimit = 3

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

			// Verify PVC was created
			pvc := &corev1.PersistentVolumeClaim{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      backup.Name,
					Namespace: testNamespace,
				}, pvc)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("200Mi")))

			// Verify CronJob was created with correct spec
			cronJob := &batchv1.CronJob{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      backup.Name,
					Namespace: testNamespace,
				}, cronJob)
			}, time.Second*5, time.Millisecond*500).Should(Succeed())

			// Check volumes in CronJob template
			jobSpec := cronJob.Spec.JobTemplate.Spec.Template.Spec
			Expect(jobSpec.Volumes).To(HaveLen(2)) // config + persistence

			// Check container
			container := jobSpec.Containers[0]
			Expect(container.VolumeMounts).To(HaveLen(2))

			// Verify image pinned
			Expect(container.Image).To(Equal("huacnlee/gobackup:latest"))

			// Verify command updated
			Expect(container.Command).To(ContainElement("gobackup perform -c /etc/gobackup/gobackup.yml"))
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
