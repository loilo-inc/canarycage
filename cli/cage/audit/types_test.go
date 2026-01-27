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
			i := ImageInfo{
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
			if len(got) != tt.want {
				t.Errorf("Result.CriticalCves() returned %d vulnerabilities, want %d", len(got), tt.want)
			}
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
			if len(got) != tt.want {
				t.Errorf("Result.HighCves() returned %d vulnerabilities, want %d", len(got), tt.want)
			}
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
			if len(got) != tt.want {
				t.Errorf("Result.MediumCves() returned %d vulnerabilities, want %d", len(got), tt.want)
			}
		})
	}
}

func Test_findingToCVE(t *testing.T) {
	stringPtr := func(s string) *string { return &s }

	tests := []struct {
		name    string
		finding ecrtypes.ImageScanFinding
		want    CVE
	}{
		{
			name: "complete finding with all fields",
			finding: ecrtypes.ImageScanFinding{
				Name:        stringPtr("CVE-2021-1234"),
				Severity:    ecrtypes.FindingSeverityCritical,
				Description: stringPtr("A critical security vulnerability"),
				Uri:         stringPtr("https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-1234"),
				Attributes: []ecrtypes.Attribute{
					{Key: stringPtr("package_name"), Value: stringPtr("openssl")},
					{Key: stringPtr("package_version"), Value: stringPtr("1.0.2k")},
				},
			},
			want: CVE{
				Name:           "CVE-2021-1234",
				Severity:       ecrtypes.FindingSeverityCritical,
				Description:    "A critical security vulnerability",
				Uri:            "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-1234",
				PackageName:    "openssl",
				PackageVersion: "1.0.2k",
			},
		},
		{
			name: "finding with nil description and uri",
			finding: ecrtypes.ImageScanFinding{
				Name:        stringPtr("CVE-2022-5678"),
				Severity:    ecrtypes.FindingSeverityHigh,
				Description: nil,
				Uri:         nil,
				Attributes: []ecrtypes.Attribute{
					{Key: stringPtr("package_name"), Value: stringPtr("curl")},
					{Key: stringPtr("package_version"), Value: stringPtr("7.64.0")},
				},
			},
			want: CVE{
				Name:           "CVE-2022-5678",
				Severity:       ecrtypes.FindingSeverityHigh,
				Description:    "",
				Uri:            "",
				PackageName:    "curl",
				PackageVersion: "7.64.0",
			},
		},
		{
			name: "finding without package attributes",
			finding: ecrtypes.ImageScanFinding{
				Name:        stringPtr("CVE-2023-9999"),
				Severity:    ecrtypes.FindingSeverityMedium,
				Description: stringPtr("Medium severity issue"),
				Uri:         stringPtr("https://example.com/cve"),
				Attributes:  []ecrtypes.Attribute{},
			},
			want: CVE{
				Name:           "CVE-2023-9999",
				Severity:       ecrtypes.FindingSeverityMedium,
				Description:    "Medium severity issue",
				Uri:            "https://example.com/cve",
				PackageName:    "unknown",
				PackageVersion: "unknown",
			},
		},
		{
			name: "finding with only package_name",
			finding: ecrtypes.ImageScanFinding{
				Name:        stringPtr("CVE-2020-1111"),
				Severity:    ecrtypes.FindingSeverityLow,
				Description: stringPtr("Low severity issue"),
				Uri:         stringPtr("https://example.com"),
				Attributes: []ecrtypes.Attribute{
					{Key: stringPtr("package_name"), Value: stringPtr("nginx")},
				},
			},
			want: CVE{
				Name:           "CVE-2020-1111",
				Severity:       ecrtypes.FindingSeverityLow,
				Description:    "Low severity issue",
				Uri:            "https://example.com",
				PackageName:    "nginx",
				PackageVersion: "unknown",
			},
		},
		{
			name: "finding with only package_version",
			finding: ecrtypes.ImageScanFinding{
				Name:        stringPtr("CVE-2019-2222"),
				Severity:    ecrtypes.FindingSeverityInformational,
				Description: stringPtr("Informational issue"),
				Uri:         stringPtr("https://example.com"),
				Attributes: []ecrtypes.Attribute{
					{Key: stringPtr("package_version"), Value: stringPtr("2.4.6")},
				},
			},
			want: CVE{
				Name:           "CVE-2019-2222",
				Severity:       ecrtypes.FindingSeverityInformational,
				Description:    "Informational issue",
				Uri:            "https://example.com",
				PackageName:    "unknown",
				PackageVersion: "2.4.6",
			},
		},
		{
			name: "finding with extra attributes",
			finding: ecrtypes.ImageScanFinding{
				Name:        stringPtr("CVE-2024-3333"),
				Severity:    ecrtypes.FindingSeverityHigh,
				Description: stringPtr("Test description"),
				Uri:         stringPtr("https://test.com"),
				Attributes: []ecrtypes.Attribute{
					{Key: stringPtr("package_name"), Value: stringPtr("testpkg")},
					{Key: stringPtr("package_version"), Value: stringPtr("1.2.3")},
					{Key: stringPtr("other_field"), Value: stringPtr("ignored")},
				},
			},
			want: CVE{
				Name:           "CVE-2024-3333",
				Severity:       ecrtypes.FindingSeverityHigh,
				Description:    "Test description",
				Uri:            "https://test.com",
				PackageName:    "testpkg",
				PackageVersion: "1.2.3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findingToCVE(tt.finding)
			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Severity != tt.want.Severity {
				t.Errorf("Severity = %v, want %v", got.Severity, tt.want.Severity)
			}
			if got.Description != tt.want.Description {
				t.Errorf("Description = %v, want %v", got.Description, tt.want.Description)
			}
			if got.Uri != tt.want.Uri {
				t.Errorf("Uri = %v, want %v", got.Uri, tt.want.Uri)
			}
			if got.PackageName != tt.want.PackageName {
				t.Errorf("PackageName = %v, want %v", got.PackageName, tt.want.PackageName)
			}
			if got.PackageVersion != tt.want.PackageVersion {
				t.Errorf("PackageVersion = %v, want %v", got.PackageVersion, tt.want.PackageVersion)
			}
		})
	}
}
