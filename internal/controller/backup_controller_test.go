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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1alpha1"
	"github.com/gobackup/gobackup-operator/pkg/k8sutil"
)

func ptr[T any](v T) *T { return &v }

var _ = Describe("BackupReconciler status conditions", func() {
	const namespace = "default"

	var (
		ctx        context.Context
		reconciler *BackupReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()

		clientset, err := kubernetes.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
		dynClient, err := dynamic.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())

		reconciler = &BackupReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
			K8s: &k8sutil.K8s{
				Clientset:     clientset,
				DynamicClient: dynClient,
			},
		}
	})

	It("sets a Ready condition with the current ObservedGeneration after reconcile", func() {
		db := &backupv1.Database{
			ObjectMeta: metav1.ObjectMeta{Name: "cond-db", Namespace: namespace},
			Spec: backupv1.DatabaseSpec{
				Type: "postgresql",
				Config: backupv1.DatabaseConfig{
					Host:     ptr("localhost"),
					Database: ptr("testdb"),
					Username: ptr("user"),
					Password: ptr("pass"),
				},
			},
		}
		Expect(k8sClient.Create(ctx, db)).To(Succeed())

		storage := &backupv1.Storage{
			ObjectMeta: metav1.ObjectMeta{Name: "cond-storage", Namespace: namespace},
			Spec: backupv1.StorageSpec{
				Type: "s3",
				Config: backupv1.StorageConfig{
					Bucket:          ptr("test-bucket"),
					Region:          ptr("us-east-1"),
					AccessKeyID:     ptr("access-key"),
					SecretAccessKey: ptr("secret-key"),
				},
			},
		}
		Expect(k8sClient.Create(ctx, storage)).To(Succeed())

		backup := &backupv1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "cond-backup", Namespace: namespace},
			Spec: backupv1.BackupSpec{
				DatabaseRefs: []backupv1.DatabaseRef{{Name: "cond-db"}},
				StorageRefs:  []backupv1.StorageRef{{Name: "cond-storage"}},
				Schedule:     &backupv1.BackupSchedule{Cron: "*/5 * * * *"},
			},
		}
		Expect(k8sClient.Create(ctx, backup)).To(Succeed())

		DeferCleanup(func() {
			_ = k8sClient.Delete(ctx, backup)
			_ = k8sClient.Delete(ctx, storage)
			_ = k8sClient.Delete(ctx, db)
		})

		key := types.NamespacedName{Name: backup.Name, Namespace: namespace}
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
		Expect(err).NotTo(HaveOccurred())

		fetched := &backupv1.Backup{}
		Expect(k8sClient.Get(ctx, key, fetched)).To(Succeed())

		ready := meta.FindStatusCondition(fetched.Status.Conditions, backupv1.ConditionTypeReady)
		Expect(ready).NotTo(BeNil(), "expected a Ready condition to be set")
		Expect(ready.ObservedGeneration).To(Equal(fetched.Generation))
		Expect(fetched.Status.ObservedGeneration).To(Equal(fetched.Generation))
	})
})
