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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1 "github.com/gobackup/gobackup-operator/api/v1alpha1"
)

func strptr(s string) *string { return &s }

// These specs exercise the E3 CEL one-of rules on DatabaseSpec/StorageSpec:
// exactly the config sub-object matching spec.type may be set.
var _ = Describe("Config sub-object CEL validation", func() {
	ctx := context.Background()

	Context("Database", func() {
		It("accepts a matching config sub-object", func() {
			db := &backupv1.Database{
				ObjectMeta: metav1.ObjectMeta{Name: "cel-db-ok", Namespace: "default"},
				Spec: backupv1.DatabaseSpec{
					Type: "postgresql",
					Config: backupv1.DatabaseConfig{
						PostgreSQL: &backupv1.PostgreSQLConfig{Database: strptr("appdb")},
					},
				},
			}
			Expect(k8sClient.Create(ctx, db)).To(Succeed())
			Expect(k8sClient.Delete(ctx, db)).To(Succeed())
		})

		It("rejects a config sub-object that does not match spec.type", func() {
			db := &backupv1.Database{
				ObjectMeta: metav1.ObjectMeta{Name: "cel-db-mismatch", Namespace: "default"},
				Spec: backupv1.DatabaseSpec{
					Type: "postgresql",
					Config: backupv1.DatabaseConfig{
						MySQL: &backupv1.MySQLConfig{Database: strptr("appdb")},
					},
				},
			}
			Expect(k8sClient.Create(ctx, db)).NotTo(Succeed())
		})

		It("rejects when no config sub-object is set", func() {
			db := &backupv1.Database{
				ObjectMeta: metav1.ObjectMeta{Name: "cel-db-empty", Namespace: "default"},
				Spec: backupv1.DatabaseSpec{
					Type:   "postgresql",
					Config: backupv1.DatabaseConfig{},
				},
			}
			Expect(k8sClient.Create(ctx, db)).NotTo(Succeed())
		})

		It("rejects when two config sub-objects are set", func() {
			db := &backupv1.Database{
				ObjectMeta: metav1.ObjectMeta{Name: "cel-db-two", Namespace: "default"},
				Spec: backupv1.DatabaseSpec{
					Type: "postgresql",
					Config: backupv1.DatabaseConfig{
						PostgreSQL: &backupv1.PostgreSQLConfig{Database: strptr("a")},
						MySQL:      &backupv1.MySQLConfig{Database: strptr("b")},
					},
				},
			}
			Expect(k8sClient.Create(ctx, db)).NotTo(Succeed())
		})
	})

	Context("Storage", func() {
		It("accepts a matching config sub-object", func() {
			st := &backupv1.Storage{
				ObjectMeta: metav1.ObjectMeta{Name: "cel-st-ok", Namespace: "default"},
				Spec: backupv1.StorageSpec{
					Type: "s3",
					Config: backupv1.StorageConfig{
						S3: &backupv1.S3CompatibleConfig{Bucket: strptr("b")},
					},
				},
			}
			Expect(k8sClient.Create(ctx, st)).To(Succeed())
			Expect(k8sClient.Delete(ctx, st)).To(Succeed())
		})

		It("rejects a config sub-object that does not match spec.type", func() {
			st := &backupv1.Storage{
				ObjectMeta: metav1.ObjectMeta{Name: "cel-st-mismatch", Namespace: "default"},
				Spec: backupv1.StorageSpec{
					Type: "s3",
					Config: backupv1.StorageConfig{
						Local: &backupv1.LocalConfig{Path: strptr("/backups")},
					},
				},
			}
			Expect(k8sClient.Create(ctx, st)).NotTo(Succeed())
		})
	})
})
