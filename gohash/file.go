package gohash

import (
	"fmt"
	"golang.org/x/mod/sumdb/dirhash"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type H1 struct {
	Path          string
	Version       string
	DirectoryHash string
	GoModHash     string
	Directory     string
}

func FileDirHash(path string, dependency string) error {
	directory := filepath.Join(path, dependency)

	directoryHash, err := dirhash.HashDir(directory, dependency, dirhash.Hash1)
	if err != nil {
		return fmt.Errorf("failed to hash directory: %w", err)
	}

	modHash, err := fileGoModHash(directory)
	if err != nil {
		return fmt.Errorf("failed to hash go.mod: %w", err)
	}

	name, version := split(dependency)
	fmt.Printf("%s %s %s\n", name, version, directoryHash)
	fmt.Printf("%s %s/go.mod %s\n", name, version, modHash)

	return nil
}

func fileGoModHash(directory string) (string, error) {
	osOpen := func(name string) (io.ReadCloser, error) {
		return os.Open(filepath.Join(directory, name))
	}

	// TODO can be more than one go.mod file?
	goModPath := "go.mod"
	modHash, err := dirhash.Hash1([]string{goModPath}, osOpen)
	if err != nil {
		return "", fmt.Errorf("failed to get hash of go.mod: %w", err)
	}

	return modHash, err
}

func filteredHashDir(directory string, dependency string) (string, error) {
	files, err := dirhash.DirFiles(directory, dependency)
	if err != nil {
		return "", fmt.Errorf("failed to get hash %w", err)
	}

	filteredFiles := make([]string, 0)
	for _, file := range files {
		if !strings.HasPrefix(file, dependency+"/.git/") && !strings.HasPrefix(file, dependency+"/.idea/") {
			filteredFiles = append(filteredFiles, file)
		}
	}

	osOpen := func(name string) (io.ReadCloser, error) {
		return os.Open(filepath.Join(directory, strings.TrimPrefix(name, dependency)))
	}

	return dirhash.Hash1(filteredFiles, osOpen)
}

func split(dependency string) (string, string) {
	parts := strings.Split(dependency, "@")
	return parts[0], parts[1]
}
