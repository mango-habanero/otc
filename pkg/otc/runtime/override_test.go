package runtime

import (
	"context"
	"os"
	"testing"
)

func TestGetOverrideFromEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		envValue string
		want     string
	}{
		{
			name:     "not set",
			envValue: "",
			want:     "",
		},
		{
			name:     "runc",
			envValue: "runc",
			want:     "runc",
		},
		{
			name:     "with whitespace",
			envValue: "  runc  ",
			want:     "runc",
		},
		{
			name:     "containerd",
			envValue: "containerd",
			want:     "containerd",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Can't run parallel here because we're modifying env

			// Set env var
			if tt.envValue != "" {
				if err := os.Setenv("OTC_RUNTIME", tt.envValue); err != nil {
					t.Fatalf("failed to set OTC_RUNTIME: %v", err)
				}
			} else {
				if err := os.Unsetenv("OTC_RUNTIME"); err != nil {
					t.Fatalf("failed to unset OTC_RUNTIME: %v", err)
				}
			}
			defer func() {
				if err := os.Unsetenv("OTC_RUNTIME"); err != nil {
					t.Errorf("failed to cleanup OTC_RUNTIME: %v", err)
				}
			}()

			got := getOverrideFromEnv()
			if got != tt.want {
				t.Errorf("getOverrideFromEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetector_Detect_WithOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		override  string
		oci       OCIDetector
		cri       CRIDetector
		podman    PodmanDetector
		wantErr   bool
		errMsg    string
		checkFunc func(t *testing.T, result *Result)
	}{
		{
			name:     "override runc - found",
			override: "runc",
			oci:      NewOCIDetector(),
			cri:      nil,
			podman:   nil,
			wantErr:  false,
			checkFunc: func(t *testing.T, result *Result) {
				if result == nil {
					t.Fatal("expected result, got nil")
				}
				// We can't guarantee runc is installed, but if result
				// is returned, it should have the right structure
				if len(result.Runtimes) > 0 {
					if result.Runtimes[0].Name != "runc" {
						t.Errorf("expected runc, got %s", result.Runtimes[0].Name)
					}
					if result.Selected == nil {
						t.Error("expected Selected to be set")
					}
					if result.Selected.Name != "runc" {
						t.Errorf("expected Selected to be runc, got %s", result.Selected.Name)
					}
				}
			},
		},
		{
			name:     "override invalid runtime name",
			override: "invalid-runtime",
			oci:      NewOCIDetector(),
			cri:      nil,
			podman:   nil,
			wantErr:  true,
			errMsg:   "invalid OTC_RUNTIME value",
		},
		{
			name:     "override containerd - detector not configured",
			override: "containerd",
			oci:      NewOCIDetector(),
			cri:      nil, // CRI detector not provided
			podman:   nil,
			wantErr:  true,
			errMsg:   "CRI detector not configured",
		},
		{
			name:     "override podman - detector not configured",
			override: "podman",
			oci:      NewOCIDetector(),
			cri:      nil,
			podman:   nil, // Podman detector not provided
			wantErr:  true,
			errMsg:   "Podman detector not configured",
		},
		{
			name:     "override docker - not supported",
			override: "docker",
			oci:      NewOCIDetector(),
			cri:      nil,
			podman:   nil,
			wantErr:  true,
			errMsg:   "Docker runtime not yet supported",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create detector with override
			detector := &Detector{
				oci:      tt.oci,
				cri:      tt.cri,
				podman:   tt.podman,
				override: tt.override,
			}

			result, err := detector.Detect(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Detect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !contains(err.Error(), tt.errMsg) {
					t.Errorf("error message %q does not contain %q", err.Error(), tt.errMsg)
				}
			}

			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestDetector_Detect_WithoutOverride(t *testing.T) {
	t.Parallel()

	// Test that normal detection still works when no override is set
	detector := &Detector{
		oci:      NewOCIDetector(),
		cri:      nil,
		podman:   nil,
		override: "", // No override
	}

	result, err := detector.Detect(context.Background())
	if err != nil {
		// It's OK if no runtimes found, just shouldn't panic
		return
	}

	// If runtimes found, they should be from OCI detector only
	for _, rt := range result.Runtimes {
		if rt.Type != TypeOCI {
			t.Errorf("expected TypeOCI, got %v", rt.Type)
		}
	}
}

func TestNewDetector_ReadsEnv(t *testing.T) {
	// This test modifies env, so can't run parallel

	// Set environment variable
	if err := os.Setenv("OTC_RUNTIME", "runc"); err != nil {
		t.Fatalf("failed to set OTC_RUNTIME: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("OTC_RUNTIME"); err != nil {
			t.Errorf("failed to cleanup OTC_RUNTIME: %v", err)
		}
	}()

	detector := NewDetector(NewOCIDetector(), nil, nil)

	// Check that detector has override set
	if detector.override != "runc" {
		t.Errorf("expected override to be 'runc', got %q", detector.override)
	}
}

// contains checks if string s contains substring substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			(len(s) > len(substr) && indexString(s, substr) >= 0))
}

// indexString returns the index of substr in s, or -1 if not found.
func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
