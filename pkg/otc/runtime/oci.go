package runtime

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ociDetector implements OCIDetector for finding OCI runtime binaries.
type ociDetector struct{}

// NewOCIDetector creates a new OCI runtime detector.
func NewOCIDetector() OCIDetector {
	return &ociDetector{}
}

// Detect finds all available OCI runtime binaries in system PATH.
// It searches for runc, crun, and youki executables.
func (d *ociDetector) Detect() ([]Runtime, error) {
	runtimeNames := []string{"runc", "crun", "youki"}
	var found []Runtime

	for _, name := range runtimeNames {
		runtime, err := d.detectRuntime(name)
		if err != nil {
			// Binary not found or not accessible - this is normal, continue
			continue
		}
		found = append(found, runtime)
	}

	return found, nil
}

// detectRuntime attempts to find and query a specific OCI runtime.
func (d *ociDetector) detectRuntime(name string) (Runtime, error) {
	// Find binary in PATH
	path, err := exec.LookPath(name)
	if err != nil {
		return Runtime{}, fmt.Errorf("runtime %s not found in PATH: %w", name, err)
	}

	// Extract version
	version, err := d.extractVersion(name, path)
	if err != nil {
		return Runtime{}, fmt.Errorf("failed to get version for %s: %w", name, err)
	}

	return Runtime{
		Name:     name,
		Type:     TypeOCI,
		Version:  version,
		Path:     path,
		Priority: PriorityOCI,
	}, nil
}

// extractVersion executes `<runtime> --version` and parses the output.
func (d *ociDetector) extractVersion(name, path string) (string, error) {
	cmd := exec.Command(path, "--version")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute %s --version: %w (stderr: %s)",
			name, err, stderr.String())
	}

	// Parse version from output
	output := stdout.String()
	version := parseVersion(output)
	if version == "" {
		return "", fmt.Errorf("failed to parse version from output: %s", output)
	}

	return version, nil
}

// parseVersion extracts version string from runtime --version output.
// All OCI runtimes (runc, crun, youki) output format: "<name> version <version> ..."
func parseVersion(output string) string {
	// Split by whitespace and find "version" keyword
	fields := strings.Fields(output)
	for i, field := range fields {
		if strings.EqualFold(field, "version") && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}
