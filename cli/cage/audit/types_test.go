package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
			i := ImageInfo{
				Registry: tt.registry,
			}
			assert.Equal(t, tt.want, i.IsECRImage())
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
				ImageInfo: ImageInfo{
					ContainerName: "test-container",
					Registry:      "123456789012.dkr.ecr.us-east-1.amazonaws.com",
					Repository:    "test-repo",
					Tag:           "latest",
				},
				Cves: []CVE{},
			},
			want: &ScanResultSummary{
				ContainerName: "test-container",
				Status:        "NONE",
				CriticalCount: 0,
				HighCount:     0,
				MediumCount:   0,
				LowCount:      0,
				InfoCount:     0,
				ImageURI:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/test-repo:latest",
			},
		},
		{
			name: "critical findings - VULNERABLE status",
			result: &ScanResult{
				ImageInfo: ImageInfo{
					ContainerName: "test-container",
				},
				Cves: []CVE{
					{Severity: ecrtypes.FindingSeverityCritical},
					{Severity: ecrtypes.FindingSeverityHigh},
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
				ImageInfo: ImageInfo{
					ContainerName: "test-container",
				},
				Cves: []CVE{
					{Severity: ecrtypes.FindingSeverityHigh},
					{Severity: ecrtypes.FindingSeverityHigh},
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
				ImageInfo: ImageInfo{
					ContainerName: "test-container",
				},
				Cves: []CVE{
					{Severity: ecrtypes.FindingSeverityMedium},
					{Severity: ecrtypes.FindingSeverityLow},
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
				ImageInfo: ImageInfo{
					ContainerName: "test-container",
				},
				Cves: []CVE{
					{Severity: ecrtypes.FindingSeverityLow},
					{Severity: ecrtypes.FindingSeverityInformational},
				},
			},
			want: &ScanResultSummary{
				ContainerName: "test-container",
				Status:        "OK",
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
				ImageInfo: ImageInfo{
					ContainerName: "mixed-container",
				},
				Cves: []CVE{
					{Severity: ecrtypes.FindingSeverityCritical},
					{Severity: ecrtypes.FindingSeverityCritical},
					{Severity: ecrtypes.FindingSeverityHigh},
					{Severity: ecrtypes.FindingSeverityMedium},
					{Severity: ecrtypes.FindingSeverityLow},
					{Severity: ecrtypes.FindingSeverityInformational},
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
			assert.Equal(t, tt.want.ContainerName, got.ContainerName)
			assert.Equal(t, tt.want.Status, got.Status)
			assert.Equal(t, tt.want.CriticalCount, got.CriticalCount)
			assert.Equal(t, tt.want.HighCount, got.HighCount)
			assert.Equal(t, tt.want.MediumCount, got.MediumCount)
			assert.Equal(t, tt.want.LowCount, got.LowCount)
			assert.Equal(t, tt.want.InfoCount, got.InfoCount)
		})
	}
}

func TestResult_CriticalCves(t *testing.T) {
	tests := []struct {
		name   string
		result *Result
		want   int
	}{
		{
			name: "no vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{},
			},
			want: 0,
		},
		{
			name: "only critical vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{
					{CVE: CVE{Severity: ecrtypes.FindingSeverityCritical}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityCritical}},
				},
			},
			want: 2,
		},
		{
			name: "mixed severity vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{
					{CVE: CVE{Severity: ecrtypes.FindingSeverityCritical}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityHigh}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityMedium}},
				},
			},
			want: 1,
		},
		{
			name: "no critical vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{
					{CVE: CVE{Severity: ecrtypes.FindingSeverityHigh}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityMedium}},
				},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.CriticalCves()
			assert.Len(t, got, tt.want)
		})
	}
}

func TestResult_HighCves(t *testing.T) {
	tests := []struct {
		name   string
		result *Result
		want   int
	}{
		{
			name: "no vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{},
			},
			want: 0,
		},
		{
			name: "only high vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{
					{CVE: CVE{Severity: ecrtypes.FindingSeverityHigh}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityHigh}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityHigh}},
				},
			},
			want: 3,
		},
		{
			name: "mixed severity vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{
					{CVE: CVE{Severity: ecrtypes.FindingSeverityCritical}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityHigh}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityHigh}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityMedium}},
				},
			},
			want: 2,
		},
		{
			name: "no high vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{
					{CVE: CVE{Severity: ecrtypes.FindingSeverityCritical}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityMedium}},
				},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.HighCves()
			assert.Len(t, got, tt.want)
		})
	}
}

func TestResult_MediumCves(t *testing.T) {
	tests := []struct {
		name   string
		result *Result
		want   int
	}{
		{
			name: "no vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{},
			},
			want: 0,
		},
		{
			name: "only medium vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{
					{CVE: CVE{Severity: ecrtypes.FindingSeverityMedium}},
				},
			},
			want: 1,
		},
		{
			name: "mixed severity vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{
					{CVE: CVE{Severity: ecrtypes.FindingSeverityCritical}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityHigh}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityMedium}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityMedium}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityLow}},
				},
			},
			want: 2,
		},
		{
			name: "no medium vulnerabilities",
			result: &Result{
				Vulns: []*Vuln{
					{CVE: CVE{Severity: ecrtypes.FindingSeverityCritical}},
					{CVE: CVE{Severity: ecrtypes.FindingSeverityLow}},
				},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.MediumCves()
			assert.Len(t, got, tt.want)
		})
	}
}
func Test_unwrapAttributes(t *testing.T) {
	tests := []struct {
		name  string
		attrs []ecrtypes.Attribute
		want  map[string]string
	}{
		{
			name:  "empty attributes",
			attrs: []ecrtypes.Attribute{},
			want:  map[string]string{},
		},
		{
			name: "valid attributes",
			attrs: []ecrtypes.Attribute{
				{Key: stringPtr("package_name"), Value: stringPtr("curl")},
				{Key: stringPtr("package_version"), Value: stringPtr("7.68.0")},
			},
			want: map[string]string{
				"package_name":    "curl",
				"package_version": "7.68.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unwrapAttributes(tt.attrs)
			assert.Equal(t, tt.want, got)
		})
	}
}
func Test_findingToCVE(t *testing.T) {
	tests := []struct {
		name    string
		finding ecrtypes.ImageScanFinding
		want    CVE
	}{
		{
			name: "complete finding with all fields",
			finding: ecrtypes.ImageScanFinding{
				Name:        stringPtr("CVE-2021-12345"),
				Severity:    ecrtypes.FindingSeverityCritical,
				Uri:         stringPtr("https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-12345"),
				Description: stringPtr("Critical vulnerability in package"),
				Attributes: []ecrtypes.Attribute{
					{Key: stringPtr("package_name"), Value: stringPtr("curl")},
					{Key: stringPtr("package_version"), Value: stringPtr("7.68.0-1ubuntu2.7")},
				},
			},
			want: CVE{
				Name:           "CVE-2021-12345",
				PackageName:    "curl",
				PackageVersion: "7.68.0-1ubuntu2.7",
				Uri:            "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-12345",
				Description:    "Critical vulnerability in package",
				Severity:       ecrtypes.FindingSeverityCritical,
			},
		},
		{
			name: "finding with missing fields",
			finding: ecrtypes.ImageScanFinding{
				Name:     nil,
				Severity: ecrtypes.FindingSeverityHigh,
				Attributes: []ecrtypes.Attribute{
					{Key: stringPtr("other_field"), Value: stringPtr("value")},
				},
			},
			want: CVE{
				Name:           "unknown",
				PackageName:    "unknown",
				PackageVersion: "unknown",
				Uri:            "",
				Description:    "",
				Severity:       ecrtypes.FindingSeverityHigh,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findingToCVE(tt.finding)
			assert.Equal(t, tt.want, got)
		})
	}
}
func stringPtr(s string) *string {
	return &s
}
