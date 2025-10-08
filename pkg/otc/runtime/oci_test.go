package runtime

import (
	"testing"
)

func TestOCIDetector_Detect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr bool
		// We can't test specific runtimes as they may not be installed
		// Instead we test that the method runs without panic
	}{
		{
			name:    "detect OCI runtimes",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			detector := NewOCIDetector()
			runtimes, err := detector.Detect()

			if (err != nil) != tt.wantErr {
				t.Errorf("Detect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify structure of returned runtimes (if any found)
			for _, rt := range runtimes {
				if rt.Name == "" {
					t.Error("Runtime has empty Name")
				}
				if rt.Type != TypeOCI {
					t.Errorf("Runtime %s has wrong Type: got %v, want %v",
						rt.Name, rt.Type, TypeOCI)
				}
				if rt.Version == "" {
					t.Errorf("Runtime %s has empty Version", rt.Name)
				}
				if rt.Path == "" {
					t.Errorf("Runtime %s has empty Path", rt.Name)
				}
				if rt.Priority != PriorityOCI {
					t.Errorf("Runtime %s has wrong Priority: got %d, want %d",
						rt.Name, rt.Priority, PriorityOCI)
				}
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "runc version output",
			output: "runc version 1.1.12\ncommit: v1.1.12-0-g51d5e94",
			want:   "1.1.12",
		},
		{
			name:   "crun version output",
			output: "crun version 1.8.7\ncommit: 53b5c4915d472830b5c7f3890ba1c77c0b37fb87",
			want:   "1.8.7",
		},
		{
			name:   "youki version output",
			output: "youki version 0.3.3\ncommit: 4f3c8307",
			want:   "0.3.3",
		},
		{
			name:   "case insensitive version keyword",
			output: "runtime Version 1.2.3",
			want:   "1.2.3",
		},
		{
			name:   "version with extra text",
			output: "runtime version 1.2.3-rc1+git.abcdef",
			want:   "1.2.3-rc1+git.abcdef",
		},
		{
			name:   "no version keyword",
			output: "runtime 1.2.3",
			want:   "",
		},
		{
			name:   "version keyword at end",
			output: "runtime version",
			want:   "",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseVersion(tt.output)
			if got != tt.want {
				t.Errorf("parseVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOCIDetector_DetectRuntime_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	detector := &ociDetector{}

	// Test with a runtime that definitely doesn't exist
	_, err := detector.detectRuntime("nonexistent-runtime-xyz123")
	if err == nil {
		t.Error("detectRuntime() expected error for nonexistent runtime, got nil")
	}
}
