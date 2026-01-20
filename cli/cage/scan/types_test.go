package scan_test

import (
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/scan"
)

func TestImageInfo_IsECRImage(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		want     bool
	}{
		{
			name:     "public ECR registry",
			registry: "public.ecr.aws",
			want:     true,
		},
		{
			name:     "private ECR registry with standard suffix",
			registry: "123456789012.dkr.ecr.us-east-1.amazonaws.com",
			want:     true,
		},
		{
			name:     "private ECR registry with different region",
			registry: "123456789012.dkr.ecr.eu-west-1.amazonaws.com",
			want:     true,
		},
		{
			name:     "Docker Hub registry",
			registry: "docker.io",
			want:     false,
		},
		{
			name:     "empty registry",
			registry: "",
			want:     false,
		},
		{
			name:     "non-ECR AWS registry",
			registry: "amazonaws.com",
			want:     false,
		},
		{
			name:     "registry with partial ECR suffix",
			registry: "example.com",
			want:     false,
		},
		{
			name:     "registry with ECR substring but not suffix",
			registry: ".dkr.ecr.amazonaws.com.example.com",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &scan.ImageInfo{
				Registry: tt.registry,
			}
			if got := i.IsECRImage(); got != tt.want {
				t.Errorf("ImageInfo.IsECRImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
