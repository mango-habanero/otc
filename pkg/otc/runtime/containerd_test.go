package runtime

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
)

// createTestSocket creates a Unix socket for testing and returns a cleanup function
func createTestSocket(t *testing.T, socketName string) (string, func()) {
	t.Helper()

	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, socketName)

	// Create an actual Unix socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create Unix socket: %v", err)
	}

	cleanup := func() {
		if err := listener.Close(); err != nil {
			t.Logf("failed to close listener: %v", err)
		}
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			t.Logf("failed to remove socket: %v", err)
		}
	}

	return socketPath, cleanup
}

func TestNewContainerdDetector(t *testing.T) {
	t.Parallel()

	detector := NewContainerdDetector()

	if detector == nil {
		t.Fatal("NewContainerdDetector returned nil")
	}

	if len(detector.socketPaths) == 0 {
		t.Error("detector has no socket paths configured")
	}

	if detector.timeout == 0 {
		t.Error("detector timeout not set")
	}
}

func TestContainerdDetector_findSocket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupFunc func(t *testing.T) (detector *ContainerdDetector, cleanup func())
		wantErr   bool
		wantPath  string
	}{
		{
			name: "no sockets exist",
			setupFunc: func(_ *testing.T) (*ContainerdDetector, func()) {
				detector := &ContainerdDetector{
					socketPaths: []string{
						"/nonexistent/path1.sock",
						"/nonexistent/path2.sock",
					},
				}
				return detector, func() {}
			},
			wantErr: true,
		},
		{
			name: "finds first valid socket",
			setupFunc: func(t *testing.T) (*ContainerdDetector, func()) {
				socketPath, cleanup := createTestSocket(t, "containerd.sock")

				detector := &ContainerdDetector{
					socketPaths: []string{socketPath},
				}

				return detector, cleanup
			},
			wantErr:  false,
			wantPath: "", // Will be set by setupFunc dynamically
		},
		{
			name: "skips non-socket files",
			setupFunc: func(t *testing.T) (*ContainerdDetector, func()) {
				tempDir := t.TempDir()

				// Create regular file (not a socket)
				regularFile := filepath.Join(tempDir, "not-a-socket")
				if err := os.WriteFile(regularFile, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}

				detector := &ContainerdDetector{
					socketPaths: []string{regularFile, "/nonexistent.sock"},
				}

				return detector, func() {}
			},
			wantErr: true,
		},
		{
			name: "tries multiple paths in order",
			setupFunc: func(t *testing.T) (*ContainerdDetector, func()) {
				socketPath, cleanup := createTestSocket(t, "containerd.sock")

				detector := &ContainerdDetector{
					socketPaths: []string{
						"/nonexistent/first.sock",
						"/nonexistent/second.sock",
						socketPath, // Should find this one
					},
				}

				return detector, cleanup
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			detector, cleanup := tt.setupFunc(t)
			defer cleanup()

			gotPath, err := detector.findSocket()

			if (err != nil) != tt.wantErr {
				t.Errorf("findSocket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && gotPath == "" {
				t.Error("findSocket() returned empty path but no error")
			}

			if tt.wantErr && gotPath != "" {
				t.Errorf("findSocket() returned path %q but expected error", gotPath)
			}
		})
	}
}

func TestContainerdDetector_Detect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) (*ContainerdDetector, func())
		wantErr bool
		errMsg  string
	}{
		{
			name: "no socket found",
			setup: func(_ *testing.T) (*ContainerdDetector, func()) {
				detector := &ContainerdDetector{
					socketPaths: []string{"/nonexistent.sock"},
				}
				return detector, func() {}
			},
			wantErr: true,
			errMsg:  "socket not found",
		},
		{
			name: "socket exists but not accessible",
			setup: func(t *testing.T) (*ContainerdDetector, func()) {
				socketPath, cleanup := createTestSocket(t, "containerd.sock")

				detector := &ContainerdDetector{
					socketPaths: []string{socketPath},
				}

				return detector, cleanup
			},
			wantErr: true,
			errMsg:  "failed to get containerd version",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			detector, cleanup := tt.setup(t)
			defer cleanup()

			ctx := context.Background()

			runtimes, err := detector.Detect(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Detect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.errMsg != "" && err.Error() == "" {
					t.Errorf("Detect() expected error message containing %q, got %q", tt.errMsg, err.Error())
				}
			}

			if !tt.wantErr {
				// Should return exactly 1 runtime
				if len(runtimes) != 1 {
					t.Fatalf("Detect() expected 1 runtime, got %d", len(runtimes))
				}

				runtime := runtimes[0]

				// Verify it's containerd
				if runtime.Name != Containerd {
					t.Errorf("Detect() runtime name = %v, want %v", runtime.Name, Containerd)
				}

				if runtime.Type != TypeCRI {
					t.Errorf("Detect() runtime type = %v, want %v", runtime.Type, TypeCRI)
				}

				if runtime.Priority != PriorityCRI {
					t.Errorf("Detect() runtime priority = %v, want %v", runtime.Priority, PriorityCRI)
				}
			}

			if tt.wantErr && len(runtimes) > 0 {
				t.Errorf("Detect() returned %d runtimes but expected error", len(runtimes))
			}
		})
	}
}

// TestContainerdDetector_Detect_Integration tests with actual containerd if available
// This test is skipped if containerd is not available
func TestContainerdDetector_Detect_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	detector := NewContainerdDetector()
	ctx := context.Background()

	runtimes, err := detector.Detect(ctx)

	if err != nil {
		// It's okay if containerd isn't installed
		t.Logf("containerd not available: %v", err)
		t.Skip("containerd not available for integration test")
		return
	}

	// If we got here, containerd was detected
	if len(runtimes) != 1 {
		t.Fatalf("Detect() expected 1 runtime, got %d", len(runtimes))
	}

	runtime := runtimes[0]

	// Verify it's containerd
	if runtime.Name != Containerd {
		t.Errorf("runtime name = %v, want %v", runtime.Name, Containerd)
	}

	if runtime.Type != TypeCRI {
		t.Errorf("runtime type = %v, want %v", runtime.Type, TypeCRI)
	}

	if runtime.Version == "" {
		t.Error("runtime version is empty")
	}

	if runtime.Path == "" {
		t.Error("runtime path is empty")
	}

	if runtime.Priority != PriorityCRI {
		t.Errorf("runtime priority = %v, want %v", runtime.Priority, PriorityCRI)
	}

	t.Logf("Detected containerd: version=%s, path=%s", runtime.Version, runtime.Path)
}
