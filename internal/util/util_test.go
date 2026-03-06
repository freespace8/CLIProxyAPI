package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAuthDirReturnsAbsolutePath(t *testing.T) {
	dir, err := ResolveAuthDir("./auths")
	if err != nil {
		t.Fatalf("ResolveAuthDir returned error: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Fatalf("expected absolute path, got %q", dir)
	}
	want, err := filepath.Abs("auths")
	if err != nil {
		t.Fatalf("filepath.Abs returned error: %v", err)
	}
	if dir != want {
		t.Fatalf("expected %q, got %q", want, dir)
	}
}

func TestResolveAuthDirExpandsHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir returned error: %v", err)
	}
	dir, err := ResolveAuthDir("~/auths")
	if err != nil {
		t.Fatalf("ResolveAuthDir returned error: %v", err)
	}
	if !strings.HasPrefix(dir, home) {
		t.Fatalf("expected %q to start with home %q", dir, home)
	}
	want := filepath.Join(home, "auths")
	if dir != want {
		t.Fatalf("expected %q, got %q", want, dir)
	}
}
