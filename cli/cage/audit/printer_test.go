package audit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
)

func makeScanResult(
	list ...ecrtypes.FindingSeverity) []*ScanResult {
	return []*ScanResult{
		{
			ImageInfo: ImageInfo{
				ContainerName: "test-container",
				Registry:      "test-registry",
				Repository:    "test-repo",
				Tag:           "latest",
			},
			ImageScanFindings: &ecrtypes.ImageScanFindings{
				Findings: makeVuln(list),
			},
		},
	}
}

func makeVuln(severities []ecrtypes.FindingSeverity) []ecrtypes.ImageScanFinding {
	findings := make([]ecrtypes.ImageScanFinding, len(severities))
	for i, sev := range severities {
		findings[i] = ecrtypes.ImageScanFinding{
			Severity:    sev,
			Name:        aws.String(fmt.Sprintf("CVE-2023-%04d", i+1)),
			Uri:         aws.String("http://example.com"),
			Description: aws.String("Test vulnerability description"),
		}
	}
	return findings
}

func TestPrinter_Print(t *testing.T) {
	setup := func(t *testing.T) (*di.D, *test.MockLogger) {
		t.Helper()
		l := &test.MockLogger{}
		d := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, l)
			b.Set(key.Time, test.NewNeverTimer())
		})
		return d, l
	}
	t.Run("prints no CVEs message when no findings", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false)

		result := makeScanResult()
		printer.Print(result)

		// Check that "No CVEs found" message is present
		found := false
		for _, log := range logger.Logs {
			if log == "No CVEs found\n" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'No CVEs found' message")
		}
	})

	t.Run("prints table header", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false)

		result := makeScanResult(ecrtypes.FindingSeverityCritical)
		printer.Print(result)

		// Check that header contains expected columns
		if len(logger.Logs) == 0 {
			t.Fatal("Expected logs to be generated")
		}
		header := logger.Logs[0]
		expectedCols := []string{"CONTAINER", "STATUS", "CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO", "IMAGE"}
		for _, col := range expectedCols {
			if !strings.Contains(header, col) {
				t.Errorf("Header missing column: %s", col)
			}
		}
	})

	t.Run("prints findings by severity", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false)

		result := makeScanResult(
			ecrtypes.FindingSeverityCritical,
			ecrtypes.FindingSeverityHigh,
			ecrtypes.FindingSeverityMedium,
		)
		printer.Print(result)

		// Should have CRITICAL, HIGH, MEDIUM sections
		criticalFound := false
		highFound := false
		mediumFound := false
		for _, log := range logger.Logs {
			if strings.Contains(log, "CRITICAL") && strings.Contains(log, "===") {
				criticalFound = true
			}
			if strings.Contains(log, "HIGH") && strings.Contains(log, "===") {
				highFound = true
			}
			if strings.Contains(log, "MEDIUM") && strings.Contains(log, "===") {
				mediumFound = true
			}
		}
		if !criticalFound {
			t.Error("Expected CRITICAL section")
		}
		if !highFound {
			t.Error("Expected HIGH section")
		}
		if !mediumFound {
			t.Error("Expected MEDIUM section")
		}
	})

	t.Run("prints total summary with counts", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false)

		result := makeScanResult(
			ecrtypes.FindingSeverityCritical,
			ecrtypes.FindingSeverityHigh,
		)
		printer.Print(result)

		// Check for total line
		totalFound := false
		for _, log := range logger.Logs {
			if strings.Contains(log, "Total:") {
				totalFound = true
				break
			}
		}
		if !totalFound {
			t.Error("Expected Total summary line")
		}
	})
}

func TestPrinter_logVuln(t *testing.T) {
	setup := func(t *testing.T) (*di.D, *test.MockLogger) {
		t.Helper()
		l := &test.MockLogger{}
		d := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, l)
			b.Set(key.Time, test.NewNeverTimer())
		})
		return d, l
	}
	t.Run("returns early when no findings", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false)
		printer.logVuln(ecrtypes.FindingSeverityCritical, []*Vuln{})

		if len(logger.Logs) != 0 {
			t.Errorf("Expected no logs, got %d", len(logger.Logs))
		}
	})

	t.Run("prints severity header", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false)

		findings := []*Vuln{
			{
				CVE: CVE{
					Name:        "CVE-2023-0001",
					Uri:         "http://example.com",
					Description: "Test description",
				},
			},
		}

		printer.logVuln(ecrtypes.FindingSeverityCritical, findings)

		headerFound := false
		for _, log := range logger.Logs {
			if strings.Contains(log, "CRITICAL") && strings.Contains(log, "===") {
				headerFound = true
				break
			}
		}
		if !headerFound {
			t.Error("Expected severity header with CRITICAL")
		}
	})

	t.Run("prints CVE name and URI", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false)

		findings := []*Vuln{
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

		assert := assert.New(t)
		assert.Contains(logger.Logs[0], "=== MEDIUM ===")
		assert.Contains(logger.Stdout[1], "- CVE-2023-0001 container-1, container-2 \n")
		assert.Contains(logger.Stdout[2], "test-package::1.2.3 (http://example.com)\n")
	})

	t.Run("uses unknown for missing package info", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false)

		findings := []*Vuln{
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

		assert := assert.New(t)
		assert.Contains(logger.Logs[0], "=== LOW ===")
		assert.Contains(logger.Logs[1], "- CVE-2023-0001 container-1 \n")
		assert.Contains(logger.Logs[2], "unknown::unknown (http://example.com)\n")
	})

	t.Run("prints description when logDetail is true", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, true) // logDetail = true

		findings := []*Vuln{
			{
				CVE: CVE{
					Name:        "CVE-2023-0001",
					Uri:         "http://example.com",
					Description: "Detailed vulnerability description",
				},
			},
		}

		printer.logVuln(ecrtypes.FindingSeverityHigh, findings)

		descFound := false
		for _, log := range logger.Logs {
			if strings.Contains(log, "Detailed vulnerability description") {
				descFound = true
				break
			}
		}
		if !descFound {
			t.Error("Expected description in output when logDetail is true")
		}
	})

	t.Run("does not print description when logDetail is false", func(t *testing.T) {
		d, logger := setup(t)
		printer := NewPrinter(d, true, false) // logDetail = false

		findings := []*Vuln{
			{
				CVE: CVE{
					Name:        "CVE-2023-0001",
					Uri:         "http://example.com",
					Description: "Detailed vulnerability description",
				},
			},
		}

		printer.logVuln(ecrtypes.FindingSeverityHigh, findings)

		descFound := false
		for _, log := range logger.Logs {
			if strings.Contains(log, "Detailed vulnerability description") {
				descFound = true
				break
			}
		}
		if descFound {
			t.Error("Expected no description in output when logDetail is false")
		}
	})
}
