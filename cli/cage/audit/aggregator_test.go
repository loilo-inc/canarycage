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
	assert.NotNil(t, agg.cveToSeverity)
	assert.NotNil(t, agg.summaries)
	assert.Equal(t, 0, len(agg.cves))
	assert.Equal(t, 0, len(agg.cveToSeverity))
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
			name: "add result with nil findings",
			scanResult: &ScanResult{
				ImageScanFindings: nil,
			},
			wantStatus:   "N/A",
			wantCVECount: 0,
		},
		{
			name: "add result with findings",
			scanResult: &ScanResult{
				ImageInfo: ImageInfo{
					ContainerName: "test-container",
				},
				ImageScanFindings: &ecrtypes.ImageScanFindings{
					Findings: []ecrtypes.ImageScanFinding{
						{
							Name:     stringPtr("CVE-2021-1234"),
							Severity: ecrtypes.FindingSeverityCritical,
						},
						{
							Name:     stringPtr("CVE-2021-5678"),
							Severity: ecrtypes.FindingSeverityHigh,
						},
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
		agg.cves = map[string]ecrtypes.ImageScanFinding{
			"CVE-2021-1": {Name: stringPtr("CVE-2021-1"), Severity: ecrtypes.FindingSeverityCritical},
			"CVE-2021-2": {Name: stringPtr("CVE-2021-2"), Severity: ecrtypes.FindingSeverityHigh},
			"CVE-2021-3": {Name: stringPtr("CVE-2021-3"), Severity: ecrtypes.FindingSeverityMedium},
			"CVE-2021-4": {Name: stringPtr("CVE-2021-4"), Severity: ecrtypes.FindingSeverityLow},
			"CVE-2021-5": {Name: stringPtr("CVE-2021-5"), Severity: ecrtypes.FindingSeverityInformational},
		}
		agg.cveToSeverity = map[string]string{
			"CVE-2021-1": string(ecrtypes.FindingSeverityCritical),
			"CVE-2021-2": string(ecrtypes.FindingSeverityHigh),
			"CVE-2021-3": string(ecrtypes.FindingSeverityMedium),
			"CVE-2021-4": string(ecrtypes.FindingSeverityLow),
			"CVE-2021-5": string(ecrtypes.FindingSeverityInformational),
		}

		result := agg.SummarizeTotal()
		assert.Equal(t, int32(1), result.CriticalCount)
		assert.Equal(t, int32(1), result.HighCount)
		assert.Equal(t, int32(1), result.MediumCount)
		assert.Equal(t, int32(1), result.LowCount)
		assert.Equal(t, int32(1), result.InfoCount)
		assert.Equal(t, int32(5), result.TotalCount)
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
				agg.cves = map[string]ecrtypes.ImageScanFinding{
					"CVE-2021-1": {Name: stringPtr("CVE-2021-1"), Severity: tt.severity},
				}
				agg.cveToSeverity = map[string]string{
					"CVE-2021-1": string(tt.severity),
				}
				result := agg.SummarizeTotal()
				assert.Equal(t, tt.severity, result.HighestSeverity)
			})
		}
	})
}

func TestAggregater_FilterCvesBySeverity(t *testing.T) {
	agg := NewAggregater()
	agg.cves = map[string]ecrtypes.ImageScanFinding{
		"CVE-2021-1": {Name: stringPtr("CVE-2021-1"), Severity: ecrtypes.FindingSeverityCritical},
		"CVE-2021-2": {Name: stringPtr("CVE-2021-2"), Severity: ecrtypes.FindingSeverityHigh},
		"CVE-2021-3": {Name: stringPtr("CVE-2021-3"), Severity: ecrtypes.FindingSeverityCritical},
	}
	agg.cveToSeverity = map[string]string{
		"CVE-2021-1": string(ecrtypes.FindingSeverityCritical),
		"CVE-2021-2": string(ecrtypes.FindingSeverityHigh),
		"CVE-2021-3": string(ecrtypes.FindingSeverityCritical),
	}

	critical := agg.CriticalCves()
	assert.Equal(t, 2, len(critical))

	high := agg.HighCves()
	assert.Equal(t, 1, len(high))

	medium := agg.MediumCves()
	assert.Equal(t, 0, len(medium))
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

func stringPtr(s string) *string {
	return &s
}
