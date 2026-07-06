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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCRDShape asserts the regenerated CRD bases carry the E4 markers: printer
// columns on the Backup CRD and an explicit storage version on all three CRDs.
func TestCRDShape(t *testing.T) {
	base := filepath.Join("..", "..", "config", "crd", "bases")

	backups := readCRD(t, filepath.Join(base, "gobackup.io_backups.yaml"))
	if !strings.Contains(backups, "additionalPrinterColumns") {
		t.Errorf("backups CRD missing additionalPrinterColumns")
	}
	for _, col := range []string{".status.phase", ".spec.schedule.cron", ".status.lastSuccessfulBackupTime", ".status.failureCount"} {
		if !strings.Contains(backups, col) {
			t.Errorf("backups CRD missing printer column JSONPath %q", col)
		}
	}

	for _, name := range []string{
		"gobackup.io_backups.yaml",
		"gobackup.io_databases.yaml",
		"gobackup.io_storages.yaml",
	} {
		content := readCRD(t, filepath.Join(base, name))
		if !strings.Contains(content, "storage: true") {
			t.Errorf("%s missing 'storage: true' (explicit storage version)", name)
		}
	}
}

func readCRD(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(data)
}
