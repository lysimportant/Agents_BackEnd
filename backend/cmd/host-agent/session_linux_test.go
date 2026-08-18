//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWriteLocalFileFollowsSymlink 验证部署机编辑器可以保存 Nginx sites-enabled 等符号链接目标文件。
func TestWriteLocalFileFollowsSymlink(t *testing.T) {
	temporaryDirectory := t.TempDir()
	targetPath := filepath.Join(temporaryDirectory, "sites-available", "default")
	linkPath := filepath.Join(temporaryDirectory, "sites-enabled", "default")
	if mkdirErr := os.MkdirAll(filepath.Dir(targetPath), 0o755); mkdirErr != nil {
		t.Fatalf("create target directory: %v", mkdirErr)
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(linkPath), 0o755); mkdirErr != nil {
		t.Fatalf("create link directory: %v", mkdirErr)
	}
	if writeErr := os.WriteFile(targetPath, []byte("old\n"), 0o644); writeErr != nil {
		t.Fatalf("write target file: %v", writeErr)
	}
	if linkErr := os.Symlink(targetPath, linkPath); linkErr != nil {
		t.Fatalf("create symlink: %v", linkErr)
	}

	response, writeErr := writeLocalFile(linkPath, "", "updated\n")
	if writeErr != nil {
		t.Fatalf("write through symlink: %v", writeErr)
	}
	if response.Type != "file_saved" {
		t.Fatalf("unexpected response type: %q", response.Type)
	}
	content, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("read target file: %v", readErr)
	}
	if string(content) != "updated\n" {
		t.Fatalf("target content = %q", content)
	}
}
