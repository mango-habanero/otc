// Package runtime provides OCI/CRI runtime detection and management.
package runtime

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// Type represents the category of container runtime.
type Type string

const (
	// TypeOCI represents direct OCI runtimes (runc, crun, youki)
	TypeOCI Type = "oci"

	// TypeCRI represents Container Runtime Interface implementations (containerd, CRI-O)
	TypeCRI Type = "cri"

	// TypePodman represents Podman runtime
	TypePodman Type = "podman"

	// TypeDocker represents Docker runtime (backward compatibility)
	TypeDocker Type = "docker"
)

// Runtime contains information about a detected container runtime.
type Runtime struct {
	// Name is the runtime identifier (e.g., "runc", "containerd", "crio")
	Name string

	// Type is the category of runtime
	Type Type

	// Version is the runtime version string
	Version string

	// Path is the filesystem path to the runtime
	// For binaries: executable path (e.g., "/usr/bin/runc")
	// For socket-based runtimes: socket path (e.g., "unix:///run/containerd/containerd.sock")
	Path string

	// Priority determines selection order when multiple runtimes are available.
	// Higher values indicate higher priority.
	Priority int
}

// Priority constants for runtime selection.
const (
	PriorityCRI    = 100 // Production Kubernetes (containerd, CRI-O)
	PriorityOCI    = 70  // Direct OCI runtimes (runc, crun, youki)
	PriorityPodman = 50  // Podman
	PriorityDocker = 30  // Docker (backward compatibility)
)

// Runtime name constants for OTC_RUNTIME environment variable.
const (
	RNRunc       = "runc"
	RNCrun       = "crun"
	RNYouki      = "youki"
	RNContainerd = "containerd"
	RNCRIO       = "crio"
	RNPodman     = "podman"
	RNDocker     = "docker"
)

// Result contains the results of runtime detection.
type Result struct {
	// Runtimes is the list of all detected runtimes, ordered by priority (highest first)
	Runtimes []Runtime

	// Selected is the highest priority runtime (nil if no runtimes detected)
	Selected *Runtime

	// Warnings contains non-fatal errors from individual detectors.
	// Detection continues even if some detectors fail.
	// Empty if all detectors succeeded.
	Warnings []error
}

// HasWarnings returns true if any detector encountered non-fatal errors.
func (r *Result) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// OCIDetector finds OCI-compliant runtime binaries (runc, crun, youki).
// Implementations search system PATH for runtime executables.
type OCIDetector interface {
	// Detect finds all available OCI runtime binaries.
	Detect() ([]Runtime, error)
}

// CRIDetector finds CRI socket-based runtimes (containerd, CRI-O).
// Implementations check standard socket locations.
type CRIDetector interface {
	// Detect finds all available CRI runtimes.
	// Context is used for socket connection timeouts.
	Detect(ctx context.Context) ([]Runtime, error)
}

// PodmanDetector finds Podman installations (rootful and rootless).
type PodmanDetector interface {
	// Detect finds available Podman runtimes.
	// Context is used for socket connection timeouts.
	Detect(ctx context.Context) ([]Runtime, error)
}

// Detector orchestrates runtime detection across all types.
type Detector struct {
	oci      OCIDetector
	cri      CRIDetector
	podman   PodmanDetector
	override string // If set, only detect this specific runtime
}

// NewDetector creates a new runtime detector with the provided implementations.
// Pass nil for any detector type not needed.
//
// The detector automatically reads the OTC_RUNTIME environment variable.
// If set, only the specified runtime will be detected.
// Valid values: runc, crun, youki, containerd, crio, podman, docker
func NewDetector(oci OCIDetector, cri CRIDetector, podman PodmanDetector) *Detector {
	return &Detector{
		oci:      oci,
		cri:      cri,
		podman:   podman,
		override: getOverrideFromEnv(),
	}
}

// Detect finds all available container runtimes on the system.
// It aggregates results from all configured detectors and selects the highest priority runtime.
// If individual detectors fail, detection continues and errors are returned in Result.Warnings.
// Only returns error if all detectors fail or a fatal error occurs.
//
// If OTC_RUNTIME environment variable is set, only the specified runtime is detected.
// Returns error if the specified runtime is not found.
func (d *Detector) Detect(ctx context.Context) (*Result, error) {
	// If override is set, only detect that runtime
	if d.override != "" {
		return d.detectOverride(ctx)
	}

	var runtimes []Runtime
	var warnings []error

	// Detect OCI runtimes (no context needed for PATH lookups)
	if d.oci != nil {
		oci, err := d.oci.Detect()
		if err != nil {
			warnings = append(warnings, err)
		} else {
			runtimes = append(runtimes, oci...)
		}
	}

	// Detect CRI runtimes (context for socket operations)
	if d.cri != nil {
		cri, err := d.cri.Detect(ctx)
		if err != nil {
			warnings = append(warnings, err)
		} else {
			runtimes = append(runtimes, cri...)
		}
	}

	// Detect Podman (context for socket operations)
	if d.podman != nil {
		podman, err := d.podman.Detect(ctx)
		if err != nil {
			warnings = append(warnings, err)
		} else {
			runtimes = append(runtimes, podman...)
		}
	}

	// If no runtimes found and we have warnings, return the first error
	if len(runtimes) == 0 && len(warnings) > 0 {
		return nil, warnings[0]
	}

	// Sort by priority (highest first)
	sortByPriority(runtimes)

	result := &Result{
		Runtimes: runtimes,
		Warnings: warnings,
	}

	// Select highest priority runtime
	if len(runtimes) > 0 {
		result.Selected = &runtimes[0]
	}

	return result, nil
}

// detectOverride detects only the runtime specified in OTC_RUNTIME.
func (d *Detector) detectOverride(ctx context.Context) (*Result, error) {
	var runtimes []Runtime
	var err error

	// Determine which detector to use based on override value
	switch d.override {
	case RNRunc, RNCrun, RNYouki:
		if d.oci == nil {
			return nil, fmt.Errorf("OTC_RUNTIME=%s but OCI detector not configured", d.override)
		}
		runtimes, err = d.oci.Detect()

	case RNContainerd, RNCRIO:
		if d.cri == nil {
			return nil, fmt.Errorf("OTC_RUNTIME=%s but CRI detector not configured", d.override)
		}
		runtimes, err = d.cri.Detect(ctx)

	case RNPodman:
		if d.podman == nil {
			return nil, fmt.Errorf("OTC_RUNTIME=%s but Podman detector not configured", d.override)
		}
		runtimes, err = d.podman.Detect(ctx)

	case RNDocker:
		return nil, fmt.Errorf("docker runtime not yet supported")

	default:
		return nil, fmt.Errorf("invalid OTC_RUNTIME value: %s (valid: runc, crun, youki, containerd, crio, podman)", d.override)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to detect runtime %s: %w", d.override, err)
	}

	// Filter to only the requested runtime
	var filtered []Runtime
	for _, rt := range runtimes {
		if rt.Name == d.override {
			filtered = append(filtered, rt)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("runtime %s not found on system", d.override)
	}

	result := &Result{
		Runtimes: filtered,
		Selected: &filtered[0],
	}

	return result, nil
}

// getOverrideFromEnv reads the OTC_RUNTIME environment variable.
// Returns empty string if not set or if value is empty after trimming whitespace.
func getOverrideFromEnv() string {
	return strings.TrimSpace(os.Getenv("OTC_RUNTIME"))
}
