package audit

import (
	"testing"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/stretchr/testify/assert"
)

func TestSeverityPrinter_Sprintf(t *testing.T) {
	tests := []struct {
		name     string
		severity ecrtypes.FindingSeverity
		format   string
		args     []any
		want     string
	}{
		{
			name:     "critical severity formats with magenta",
			severity: ecrtypes.FindingSeverityCritical,
			format:   "test %s",
			args:     []any{"critical"},
			want:     "\x1b[35mtest critical\x1b[0m",
		},
		{
			name:     "high severity formats with red",
			severity: ecrtypes.FindingSeverityHigh,
			format:   "test %s",
			args:     []any{"high"},
			want:     "\x1b[31mtest high\x1b[0m",
		},
		{
			name:     "medium severity formats with yellow",
			severity: ecrtypes.FindingSeverityMedium,
			format:   "test %s",
			args:     []any{"medium"},
			want:     "\x1b[33mtest medium\x1b[0m",
		},
		{
			name:     "low severity formats without color",
			severity: ecrtypes.FindingSeverityLow,
			format:   "test %s",
			args:     []any{"low"},
			want:     "test low",
		},
		{
			name:     "informational severity formats without color",
			severity: ecrtypes.FindingSeverityInformational,
			format:   "test %s",
			args:     []any{"info"},
			want:     "test info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &severityPrinter{
				severity: tt.severity,
			}
			got := s.Sprintf(tt.format, tt.args...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSeverityPrinter_BSprintf(t *testing.T) {
	s := &severityPrinter{
		severity: ecrtypes.FindingSeverityCritical,
	}
	got := s.BSprintf("test %s", "critical")
	want := "\x1b[1m\x1b[35mtest critical\x1b[0m\x1b[0m"
	assert.Equal(t, want, got)
}

func TestNewAggregater(t *testing.T) {
	agg := NewAggregater()
	assert.NotNil(t, agg)
	assert.NotNil(t, agg.cves)
	assert.NotNil(t, agg.summaries)
	assert.Equal(t, 0, len(agg.cves))
	assert.Equal(t, 0, len(agg.summaries))
}

func TestAggregater_Add(t *testing.T) {
	tests := []struct {
		name         string
		scanResult   *ScanResult
		wantStatus   string
		wantCVECount int
	}{
		{
			name: "add result with error",
			scanResult: &ScanResult{
				Err: assert.AnError,
			},
			wantStatus:   "ERROR",
			wantCVECount: 0,
		},
		{
			name:         "add result with nil findings",
			scanResult:   &ScanResult{},
			wantStatus:   "N/A",
			wantCVECount: 0,
		},
		{
			name: "add result with findings",
			scanResult: &ScanResult{
				ImageInfo: ImageInfo{
					ContainerName: "test-container",
				},
				Cves: []CVE{
					{
						Name:     "CVE-2021-1234",
						Severity: ecrtypes.FindingSeverityCritical,
					},
					{
						Name:     "CVE-2021-5678",
						Severity: ecrtypes.FindingSeverityHigh,
					},
				},
			},
			wantStatus:   "VULNERABLE",
			wantCVECount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := NewAggregater()
			agg.Add(tt.scanResult)
			assert.Equal(t, 1, len(agg.summaries))
			assert.Equal(t, tt.wantStatus, agg.summaries[tt.scanResult.ImageInfo.ContainerName][0].Status)
			assert.Equal(t, tt.wantCVECount, len(agg.cves))
		})

	}
}

func TestAggregater_SummarizeTotal(t *testing.T) {
	t.Run("should summarize total counts and highest severity", func(t *testing.T) {
		agg := NewAggregater()
		agg.cves = map[string]CVE{
			"CVE-2021-1": {Name: "CVE-2021-1", Severity: ecrtypes.FindingSeverityCritical},
			"CVE-2021-2": {Name: "CVE-2021-2", Severity: ecrtypes.FindingSeverityHigh},
			"CVE-2021-3": {Name: "CVE-2021-3", Severity: ecrtypes.FindingSeverityMedium},
			"CVE-2021-4": {Name: "CVE-2021-4", Severity: ecrtypes.FindingSeverityLow},
			"CVE-2021-5": {Name: "CVE-2021-5", Severity: ecrtypes.FindingSeverityInformational},
		}
		result := agg.SummarizeTotal()
		assert.Equal(t, 1, result.CriticalCount)
		assert.Equal(t, 1, result.HighCount)
		assert.Equal(t, 1, result.MediumCount)
		assert.Equal(t, 1, result.LowCount)
		assert.Equal(t, 1, result.InfoCount)
		assert.Equal(t, 5, result.TotalCount)
		assert.Equal(t, ecrtypes.FindingSeverityCritical, result.HighestSeverity)
	})
	t.Run("highest", func(t *testing.T) {
		tests := []struct {
			severity ecrtypes.FindingSeverity
		}{
			{severity: ecrtypes.FindingSeverityHigh},
			{severity: ecrtypes.FindingSeverityMedium},
			{severity: ecrtypes.FindingSeverityLow},
			{severity: ecrtypes.FindingSeverityInformational},
		}
		for _, tt := range tests {
			t.Run(string(tt.severity), func(t *testing.T) {
				agg := NewAggregater()
				agg.cves = map[string]CVE{
					"CVE-2021-1": {Name: "CVE-2021-1", Severity: tt.severity},
				}
				result := agg.SummarizeTotal()
				assert.Equal(t, tt.severity, result.HighestSeverity)
			})
		}
	})
}

func TestAggregateResult_SeverityCounts(t *testing.T) {
	result := &AggregateResult{
		CriticalCount: 1,
		HighCount:     2,
		MediumCount:   3,
		LowCount:      4,
		InfoCount:     5,
	}

	counts := result.SeverityCounts()
	assert.Equal(t, 5, len(counts))
	assert.Equal(t, 5, counts[0].Count)
	assert.Equal(t, 4, counts[1].Count)
	assert.Equal(t, 3, counts[2].Count)
	assert.Equal(t, 2, counts[3].Count)
	assert.Equal(t, 1, counts[4].Count)
}

func TestAggregater_GetVulnContainers(t *testing.T) {
	t.Run("returns containers affected by CVE", func(t *testing.T) {
		agg := NewAggregater()
		agg.cveToContainers = map[string][]string{
			"CVE-2021-1234": {"container1", "container2"},
			"CVE-2021-5678": {"container3"},
		}

		containers := agg.GetVulnContainers("CVE-2021-1234")
		assert.Equal(t, 2, len(containers))
		assert.Contains(t, containers, "container1")
		assert.Contains(t, containers, "container2")
	})

	t.Run("returns nil for non-existent CVE", func(t *testing.T) {
		agg := NewAggregater()
		agg.cveToContainers = map[string][]string{
			"CVE-2021-1234": {"container1"},
		}

		containers := agg.GetVulnContainers("CVE-9999-9999")
		assert.Nil(t, containers)
	})
}

func TestAggregater_Result(t *testing.T) {
	t.Run("returns empty result when no CVEs", func(t *testing.T) {
		agg := NewAggregater()

		result := agg.Result()

		assert.NotNil(t, result)
		assert.NotNil(t, result.Summary)
		assert.Equal(t, 0, result.Summary.TotalCount)
		assert.Equal(t, 0, len(result.Vulns))
	})

	t.Run("returns result with vulnerabilities", func(t *testing.T) {
		agg := NewAggregater()
		agg.cves = map[string]CVE{
			"CVE-2021-1234": {
				Name:           "CVE-2021-1234",
				Severity:       ecrtypes.FindingSeverityCritical,
				Description:    "Critical vulnerability",
				Uri:            "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-1234",
				PackageName:    "openssl",
				PackageVersion: "1.0.0",
			},
			"CVE-2021-5678": {
				Name:     "CVE-2021-5678",
				Severity: ecrtypes.FindingSeverityHigh,
			},
		}
		agg.cveToContainers = map[string][]string{
			"CVE-2021-1234": {"container1", "container2"},
			"CVE-2021-5678": {"container3"},
		}

		result := agg.Result()

		assert.NotNil(t, result)
		assert.NotNil(t, result.Summary)
		assert.Equal(t, 2, result.Summary.TotalCount)
		assert.Equal(t, 1, result.Summary.CriticalCount)
		assert.Equal(t, 1, result.Summary.HighCount)
		assert.Equal(t, 2, len(result.Vulns))

		// Find CVE-2021-1234 in results
		var vuln1234 *Vuln
		for _, v := range result.Vulns {
			if v.CVE.Name == "CVE-2021-1234" {
				vuln1234 = v
				break
			}
		}
		assert.NotNil(t, vuln1234)
		assert.Equal(t, "CVE-2021-1234", vuln1234.CVE.Name)
		assert.Equal(t, ecrtypes.FindingSeverityCritical, vuln1234.CVE.Severity)
		assert.Equal(t, "Critical vulnerability", vuln1234.CVE.Description)
		assert.Equal(t, "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-1234", vuln1234.CVE.Uri)
		assert.Equal(t, "openssl", vuln1234.CVE.PackageName)
		assert.Equal(t, "1.0.0", vuln1234.CVE.PackageVersion)
		assert.Equal(t, 2, len(vuln1234.Containers))
		assert.Contains(t, vuln1234.Containers, "container1")
		assert.Contains(t, vuln1234.Containers, "container2")
	})

	t.Run("handles CVE with minimal information", func(t *testing.T) {
		agg := NewAggregater()
		agg.cves = map[string]CVE{
			"CVE-2021-9999": {
				Name:     "CVE-2021-9999",
				Severity: ecrtypes.FindingSeverityLow,
			},
		}
		agg.cveToContainers = map[string][]string{
			"CVE-2021-9999": {"container1"},
		}

		result := agg.Result()

		assert.NotNil(t, result)
		assert.Equal(t, 1, len(result.Vulns))
		assert.Equal(t, "CVE-2021-9999", result.Vulns[0].CVE.Name)
		assert.Equal(t, ecrtypes.FindingSeverityLow, result.Vulns[0].CVE.Severity)
		assert.Equal(t, "", result.Vulns[0].CVE.Description)
		assert.Equal(t, "", result.Vulns[0].CVE.Uri)
		assert.Equal(t, "", result.Vulns[0].CVE.PackageName)
		assert.Equal(t, "", result.Vulns[0].CVE.PackageVersion)
	})
}
