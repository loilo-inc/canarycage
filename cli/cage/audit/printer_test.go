package audit

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
)

func makeScanResult(
	list ...ecrtypes.FindingSeverity) []ScanResult {
	return []ScanResult{
		{
			ImageInfo: ImageInfo{
				ContainerName: "test-container",
				Registry:      "test-registry",
				Repository:    "test-repo",
				Tag:           "latest",
			},
			Cves: func() []CVE {
				var cves []CVE
				for i, sev := range list {
					cves = append(cves, CVE{
						Name:     fmt.Sprintf("CVE-2023-000%d", i+1),
						Severity: sev,
					})
				}
				return cves
			}(),
		},
	}
}

func TestPrinter_Print(t *testing.T) {
	setup := func(t *testing.T) (*di.D, *test.MockPrinter) {
		t.Helper()
		p := &test.MockPrinter{}
		d := di.NewDomain(func(b *di.B) {
			b.Set(key.Printer, p)
			b.Set(key.Time, test.NewNeverTimer())
		})
		return d, p
	}
	t.Run("prints no CVEs message when no findings", func(t *testing.T) {
		d, p := setup(t)
		printer := NewPrinter(d, true, false)

		result := makeScanResult()
		printer.Print(result)

		assert.Equal(t, p.Logs, []string{"No CVEs found\n"})
		assert.Len(t, p.Stderr, 0)
	})

	t.Run("prints table header", func(t *testing.T) {
		d, p := setup(t)
		printer := NewPrinter(d, true, false)

		result := makeScanResult(ecrtypes.FindingSeverityCritical)
		printer.Print(result)

		assert.NotEmpty(t, p.Logs, "Expected logs to be generated")
		header := p.Logs[0]
		expectedCols := []string{"CONTAINER", "STATUS", "CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO", "IMAGE"}
		for _, col := range expectedCols {
			assert.Contains(t, header, col)
		}
	})

	t.Run("prints findings by severity", func(t *testing.T) {
		d, p := setup(t)
		printer := NewPrinter(d, true, false)

		result := makeScanResult(
			ecrtypes.FindingSeverityCritical,
			ecrtypes.FindingSeverityHigh,
			ecrtypes.FindingSeverityMedium,
		)
		printer.Print(result)

		criticalFound := slices.ContainsFunc(p.Logs, func(log string) bool {
			return strings.Contains(log, "CRITICAL") && strings.Contains(log, "===")
		})
		highFound := slices.ContainsFunc(p.Logs, func(log string) bool {
			return strings.Contains(log, "HIGH") && strings.Contains(log, "===")
		})
		mediumFound := slices.ContainsFunc(p.Logs, func(log string) bool {
			return strings.Contains(log, "MEDIUM") && strings.Contains(log, "===")
		})

		assert.True(t, criticalFound, "Expected CRITICAL section")
		assert.True(t, highFound, "Expected HIGH section")
		assert.True(t, mediumFound, "Expected MEDIUM section")
	})

	t.Run("prints total summary with counts", func(t *testing.T) {
		d, p := setup(t)
		printer := NewPrinter(d, true, false)

		result := makeScanResult(
			ecrtypes.FindingSeverityCritical,
			ecrtypes.FindingSeverityHigh,
		)
		printer.Print(result)

		totalFound := slices.ContainsFunc(p.Logs, func(log string) bool {
			return strings.Contains(log, "Total:")
		})
		assert.True(t, totalFound, "Expected Total summary line")
	})
}

func TestPrinter_logVuln(t *testing.T) {
	setup := func(t *testing.T) (*di.D, *test.MockPrinter) {
		t.Helper()
		p := &test.MockPrinter{}
		d := di.NewDomain(func(b *di.B) {
			b.Set(key.Printer, p)
			b.Set(key.Time, test.NewNeverTimer())
		})
		return d, p
	}
	t.Run("returns early when no findings", func(t *testing.T) {
		d, p := setup(t)
		printer := NewPrinter(d, true, false)
		printer.logVuln(ecrtypes.FindingSeverityCritical, []Vuln{})

		assert.Empty(t, p.Logs)
	})

	t.Run("prints severity header", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false)

		findings := []Vuln{
			{
				CVE: CVE{
					Name:        "CVE-2023-0001",
					Uri:         "http://example.com",
					Description: "Test description",
				},
			},
		}

		printer.logVuln(ecrtypes.FindingSeverityCritical, findings)

		headerFound := slices.ContainsFunc(logger.Logs, func(log string) bool {
			return strings.Contains(log, "CRITICAL") && strings.Contains(log, "===")
		})
		assert.True(t, headerFound, "Expected severity header with CRITICAL")
	})

	t.Run("prints CVE name and URI", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false)

		findings := []Vuln{
			{
				Containers: []string{"container-1", "container-2"},
				CVE: CVE{
					Name:           "CVE-2023-0001",
					Uri:            "http://example.com",
					Description:    "Test description",
					PackageName:    "test-package",
					PackageVersion: "1.2.3",
				},
			},
		}

		printer.logVuln(ecrtypes.FindingSeverityMedium, findings)

		assert.Contains(t, logger.Logs[0], "=== MEDIUM ===")
		assert.Contains(t, logger.Stdout[1], "- CVE-2023-0001 container-1, container-2 \n")
		assert.Contains(t, logger.Stdout[2], "test-package::1.2.3 (http://example.com)\n")
	})

	t.Run("uses unknown for missing package info", func(t *testing.T) {
		d, p := setup(t)
		printer := NewPrinter(d, true, false)

		findings := []Vuln{
			{
				Containers: []string{"container-1"},
				CVE: CVE{
					Name:           "CVE-2023-0001",
					Uri:            "http://example.com",
					Description:    "Test description",
					PackageName:    "unknown",
					PackageVersion: "unknown",
				},
			},
		}

		printer.logVuln(ecrtypes.FindingSeverityLow, findings)

		assert.Contains(t, p.Logs[0], "=== LOW ===")
		assert.Contains(t, p.Logs[1], "- CVE-2023-0001 container-1 \n")
		assert.Contains(t, p.Logs[2], "unknown::unknown (http://example.com)\n")
	})

	t.Run("prints description when logDetail is true", func(t *testing.T) {
		d, p := setup(t)
		printer := NewPrinter(d, true, true) // logDetail = true

		findings := []Vuln{
			{
				CVE: CVE{
					Name:        "CVE-2023-0001",
					Uri:         "http://example.com",
					Description: "Detailed vulnerability description",
				},
			},
		}

		printer.logVuln(ecrtypes.FindingSeverityHigh, findings)

		descFound := slices.ContainsFunc(p.Logs, func(log string) bool {
			return strings.Contains(log, "Detailed vulnerability description")
		})
		assert.True(t, descFound, "Expected description in output when logDetail is true")
	})

	t.Run("does not print description when logDetail is false", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false) // logDetail = false

		findings := []Vuln{
			{
				CVE: CVE{
					Name:        "CVE-2023-0001",
					Uri:         "http://example.com",
					Description: "Detailed vulnerability description",
				},
			},
		}

		printer.logVuln(ecrtypes.FindingSeverityHigh, findings)

		descFound := slices.ContainsFunc(logger.Logs, func(log string) bool {
			return strings.Contains(log, "Detailed vulnerability description")
		})
		assert.False(t, descFound, "Expected no description in output when logDetail is false")
	})
}

func TestPrinter_PrintJSON(t *testing.T) {
	setup := func(t *testing.T) (*di.D, *test.MockPrinter) {
		t.Helper()
		p := &test.MockPrinter{}
		d := di.NewDomain(func(b *di.B) {
			b.Set(key.Printer, p)
			b.Set(key.Time, test.NewNeverTimer())
		})
		return d, p
	}

	t.Run("prints valid JSON output", func(t *testing.T) {
		d, p := setup(t)
		printer := NewPrinter(d, true, false)

		metadata := Target{
			Region:  "ap-northeast-1",
			Cluster: "test-cluster",
			Service: "test-service",
		}
		result := makeScanResult(ecrtypes.FindingSeverityCritical)

		printer.PrintJSON(metadata, result)

		assert.NotEmpty(t, p.Stdout)

		var parsed FinalResult
		err := json.Unmarshal([]byte(p.Stdout[0]), &parsed)
		assert.NoError(t, err)
	})

	t.Run("handles empty scan results", func(t *testing.T) {
		d, p := setup(t)
		printer := NewPrinter(d, true, false)

		metadata := Target{
			Region:  "ap-northeast-1",
			Cluster: "test-cluster",
			Service: "test-service",
		}
		result := []ScanResult{}

		printer.PrintJSON(metadata, result)

		assert.NotEmpty(t, p.Stdout)

		var parsed FinalResult
		err := json.Unmarshal([]byte(p.Stdout[0]), &parsed)
		assert.NoError(t, err)
	})
}
