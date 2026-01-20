package audit

import (
	"testing"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
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
			i := &ImageInfo{
				Registry: tt.registry,
			}
			if got := i.IsECRImage(); got != tt.want {
				t.Errorf("ImageInfo.IsECRImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
func Test_summaryScanResult(t *testing.T) {
	tests := []struct {
		name   string
		result *ScanResult
		want   *ScanResultSummary
	}{
		{
			name: "no findings - NONE status",
			result: &ScanResult{
				ImageInfo: &ImageInfo{
					ContainerName: "test-container",
					Registry:      "123456789012.dkr.ecr.us-east-1.amazonaws.com",
					Repository:    "test-repo",
					Tag:           "latest",
				},
				ImageScanFindings: &ecrtypes.ImageScanFindings{
					Findings: []ecrtypes.ImageScanFinding{},
				},
			},
			want: &ScanResultSummary{
				ContainerName: "test-container",
				Status:        "NONE",
				CriticalCount: 0,
				HighCount:     0,
				MediumCount:   0,
				LowCount:      0,
				InfoCount:     0,
				ImageURI: formatImageLabel(&ImageInfo{
					ContainerName: "test-container",
					Registry:      "123456789012.dkr.ecr.us-east-1.amazonaws.com",
					Repository:    "test-repo",
					Tag:           "latest",
				}),
			},
		},
		{
			name: "critical findings - VULNERABLE status",
			result: &ScanResult{
				ImageInfo: &ImageInfo{
					ContainerName: "test-container",
				},
				ImageScanFindings: &ecrtypes.ImageScanFindings{
					Findings: []ecrtypes.ImageScanFinding{
						{Severity: ecrtypes.FindingSeverityCritical},
						{Severity: ecrtypes.FindingSeverityHigh},
					},
				},
			},
			want: &ScanResultSummary{
				ContainerName: "test-container",
				Status:        "VULNERABLE",
				CriticalCount: 1,
				HighCount:     1,
				MediumCount:   0,
				LowCount:      0,
				InfoCount:     0,
			},
		},
		{
			name: "high findings - VULNERABLE status",
			result: &ScanResult{
				ImageInfo: &ImageInfo{
					ContainerName: "test-container",
				},
				ImageScanFindings: &ecrtypes.ImageScanFindings{
					Findings: []ecrtypes.ImageScanFinding{
						{Severity: ecrtypes.FindingSeverityHigh},
						{Severity: ecrtypes.FindingSeverityHigh},
					},
				},
			},
			want: &ScanResultSummary{
				ContainerName: "test-container",
				Status:        "VULNERABLE",
				CriticalCount: 0,
				HighCount:     2,
				MediumCount:   0,
				LowCount:      0,
				InfoCount:     0,
			},
		},
		{
			name: "medium findings - WARNING status",
			result: &ScanResult{
				ImageInfo: &ImageInfo{
					ContainerName: "test-container",
				},
				ImageScanFindings: &ecrtypes.ImageScanFindings{
					Findings: []ecrtypes.ImageScanFinding{
						{Severity: ecrtypes.FindingSeverityMedium},
						{Severity: ecrtypes.FindingSeverityLow},
					},
				},
			},
			want: &ScanResultSummary{
				ContainerName: "test-container",
				Status:        "WARNING",
				CriticalCount: 0,
				HighCount:     0,
				MediumCount:   1,
				LowCount:      1,
				InfoCount:     0,
			},
		},
		{
			name: "low and informational findings - empty status",
			result: &ScanResult{
				ImageInfo: &ImageInfo{
					ContainerName: "test-container",
				},
				ImageScanFindings: &ecrtypes.ImageScanFindings{
					Findings: []ecrtypes.ImageScanFinding{
						{Severity: ecrtypes.FindingSeverityLow},
						{Severity: ecrtypes.FindingSeverityInformational},
					},
				},
			},
			want: &ScanResultSummary{
				ContainerName: "test-container",
				Status:        "",
				CriticalCount: 0,
				HighCount:     0,
				MediumCount:   0,
				LowCount:      1,
				InfoCount:     1,
			},
		},
		{
			name: "mixed severity findings",
			result: &ScanResult{
				ImageInfo: &ImageInfo{
					ContainerName: "mixed-container",
				},
				ImageScanFindings: &ecrtypes.ImageScanFindings{
					Findings: []ecrtypes.ImageScanFinding{
						{Severity: ecrtypes.FindingSeverityCritical},
						{Severity: ecrtypes.FindingSeverityCritical},
						{Severity: ecrtypes.FindingSeverityHigh},
						{Severity: ecrtypes.FindingSeverityMedium},
						{Severity: ecrtypes.FindingSeverityLow},
						{Severity: ecrtypes.FindingSeverityInformational},
					},
				},
			},
			want: &ScanResultSummary{
				ContainerName: "mixed-container",
				Status:        "VULNERABLE",
				CriticalCount: 2,
				HighCount:     1,
				MediumCount:   1,
				LowCount:      1,
				InfoCount:     1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summaryScanResult(tt.result)
			if got.ContainerName != tt.want.ContainerName {
				t.Errorf("ContainerName = %v, want %v", got.ContainerName, tt.want.ContainerName)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %v, want %v", got.Status, tt.want.Status)
			}
			if got.CriticalCount != tt.want.CriticalCount {
				t.Errorf("CriticalCount = %v, want %v", got.CriticalCount, tt.want.CriticalCount)
			}
			if got.HighCount != tt.want.HighCount {
				t.Errorf("HighCount = %v, want %v", got.HighCount, tt.want.HighCount)
			}
			if got.MediumCount != tt.want.MediumCount {
				t.Errorf("MediumCount = %v, want %v", got.MediumCount, tt.want.MediumCount)
			}
			if got.LowCount != tt.want.LowCount {
				t.Errorf("LowCount = %v, want %v", got.LowCount, tt.want.LowCount)
			}
			if got.InfoCount != tt.want.InfoCount {
				t.Errorf("InfoCount = %v, want %v", got.InfoCount, tt.want.InfoCount)
			}
		})
	}
}
