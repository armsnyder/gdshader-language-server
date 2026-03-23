// Copyright (c) 2026 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package main_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	. "github.com/onsi/gomega"

	. "github.com/armsnyder/gdshader-language-server/internal/tools/gentables"
)

func TestGenerateTables_Basic(t *testing.T) {
	g := NewWithT(t)
	targetDir := t.TempDir()
	g.Expect(GenerateTables(os.DirFS("testdata/basic/docs"), fstest.MapFS{}, targetDir)).To(Succeed())
	assertDirectoriesEqual(t, os.DirFS("testdata/basic/expected"), os.DirFS(targetDir))
}

func assertDirectoriesEqual(t *testing.T, expectedFS, actualFS fs.FS) {
	t.Helper()
	g := NewWithT(t)

	expectedFiles := listFiles(t, expectedFS)
	actualFiles := listFiles(t, actualFS)
	g.Expect(expectedFiles).To(Equal(actualFiles), "File set mismatch")

	for _, rel := range expectedFiles {
		expectedBytes, err := fs.ReadFile(expectedFS, rel)
		g.Expect(err).ToNot(HaveOccurred(), "Read expected file %s", rel)
		actualBytes, err := fs.ReadFile(actualFS, rel)
		g.Expect(err).ToNot(HaveOccurred(), "Read actual file %s", rel)

		g.Expect(actualBytes).To(Equal(expectedBytes), "Content mismatch for file %s", rel)
	}
}

func listFiles(t *testing.T, root fs.FS) []string {
	t.Helper()

	files := []string{}
	if err := fs.WalkDir(root, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	}); err != nil {
		t.Fatalf("walk dir: %v", err)
	}

	slices.Sort(files)
	return files
}
