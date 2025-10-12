package runtime

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// Standard containerd socket paths in order of preference
var containerdSocketPaths = []string{
	"/run/containerd/containerd.sock",     // Primary - canonical location
	"/var/run/containerd/containerd.sock", // Alternative - symlink on modern systems
	"/run/k3s/containerd/containerd.sock", // K3s/RKE2
}

// ContainerdDetector detects containerd via CRI socket
type ContainerdDetector struct {
	socketPaths []string
	timeout     time.Duration
}

// NewContainerdDetector creates a new containerd detector with default settings
func NewContainerdDetector() *ContainerdDetector {
	return &ContainerdDetector{
		socketPaths: containerdSocketPaths,
		timeout:     5 * time.Second, // Default timeout for CRI calls
	}
}

// Detect attempts to detect containerd via CRI socket
func (d *ContainerdDetector) Detect(ctx context.Context) ([]Runtime, error) {
	// Find first accessible socket
	socket, err := d.findSocket()
	if err != nil {
		return nil, fmt.Errorf("containerd socket not found: %w", err)
	}

	// Get version via CRI API
	version, err := d.getVersion(ctx, socket)
	if err != nil {
		return nil, fmt.Errorf("failed to get containerd version from CRI: %w", err)
	}

	return []Runtime{
		{
			Name:     Containerd,
			Type:     TypeCRI,
			Version:  version,
			Path:     socket,
			Priority: PriorityCRI,
		},
	}, nil
}

// findSocket searches for the first accessible containerd socket
func (d *ContainerdDetector) findSocket() (string, error) {
	for _, path := range d.socketPaths {
		// Check if path exists
		info, err := os.Stat(path)
		if err != nil {
			continue // Socket doesn't exist, try next
		}

		// Verify it's actually a socket
		if info.Mode()&os.ModeSocket == 0 {
			continue // Not a socket, try next
		}

		return path, nil
	}

	return "", fmt.Errorf("no accessible socket found in: %v", d.socketPaths)
}

// getVersion connects to containerd via CRI and retrieves version information
func (d *ContainerdDetector) getVersion(ctx context.Context, socketPath string) (string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	// Establish gRPC connection to containerd socket using NewClient
	conn, err := grpc.NewClient(
		"unix://"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			// Log or handle close error if needed
			// In detection context, we can ignore close errors
			_ = closeErr
		}
	}()

	// Create CRI runtime service client
	client := runtimeapi.NewRuntimeServiceClient(conn)

	// Call Version API
	resp, err := client.Version(ctx, &runtimeapi.VersionRequest{
		Version: "v1", // CRI API version
	})
	if err != nil {
		return "", fmt.Errorf("CRI Version call failed: %w", err)
	}

	return resp.RuntimeVersion, nil
}
