package otc

import "testing"

func TestVersion(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "version is set",
			want: "0.1.0-dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if Version != tt.want {
				t.Errorf("Version() got = %v, want %v", Version, tt.want)
			}
		})
	}
}
