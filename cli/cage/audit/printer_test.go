package audit

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

type mockLogger struct {
	logs []string
}

func (m *mockLogger) Printf(format string, args ...any) {
	m.logs = append(m.logs, fmt.Sprintf(format, args...))
}

func makeScanResult(
	list ...ecrtypes.FindingSeverity) []*ScanResult {
	return []*ScanResult{
		{
			ImageInfo: &ImageInfo{
				ContainerName: "test-container",
				Registry:      "test-registry",
				Repository:    "test-repo",
				Tag:           "latest",
			},
			ImageScanFindings: &ecrtypes.ImageScanFindings{
				Findings: makeFindings(list),
			},
		},
	}
}

func makeFindings(severities []ecrtypes.FindingSeverity) []ecrtypes.ImageScanFinding {
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
	t.Run("prints no CVEs message when no findings", func(t *testing.T) {
		logger := &mockLogger{}
		printer := NewPrinter(logger, true, false)

		result := makeScanResult()
		printer.Print(result)

		// Check that "No CVEs found" message is present
		found := false
		for _, log := range logger.logs {
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
		logger := &mockLogger{}
		printer := NewPrinter(logger, true, false)

		result := makeScanResult(ecrtypes.FindingSeverityCritical)
		printer.Print(result)

		// Check that header contains expected columns
		if len(logger.logs) == 0 {
			t.Fatal("Expected logs to be generated")
		}
		header := logger.logs[0]
		expectedCols := []string{"CONTAINER", "STATUS", "CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO", "IMAGE"}
		for _, col := range expectedCols {
			if !containsString(header, col) {
				t.Errorf("Header missing column: %s", col)
			}
		}
	})

	t.Run("prints findings by severity", func(t *testing.T) {
		logger := &mockLogger{}
		printer := NewPrinter(logger, true, false)

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
		for _, log := range logger.logs {
			if containsString(log, "CRITICAL") && containsString(log, "===") {
				criticalFound = true
			}
			if containsString(log, "HIGH") && containsString(log, "===") {
				highFound = true
			}
			if containsString(log, "MEDIUM") && containsString(log, "===") {
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
		logger := &mockLogger{}
		printer := NewPrinter(logger, true, false)

		result := makeScanResult(
			ecrtypes.FindingSeverityCritical,
			ecrtypes.FindingSeverityHigh,
		)
		printer.Print(result)

		// Check for total line
		totalFound := false
		for _, log := range logger.logs {
			if containsString(log, "Total:") {
				totalFound = true
				break
			}
		}
		if !totalFound {
			t.Error("Expected Total summary line")
		}
	})
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
