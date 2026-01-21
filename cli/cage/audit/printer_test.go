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
			ImageInfo: ImageInfo{
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

func TestPrinter_logImageScanFindings(t *testing.T) {
	t.Run("does nothing when findings are empty", func(t *testing.T) {
		logger := &mockLogger{}
		printer := NewPrinter(logger, true, false)
		agg := NewAggregater()

		printer.logImageScanFindings(ecrtypes.FindingSeverityCritical, []ecrtypes.ImageScanFinding{}, agg)

		if len(logger.logs) != 0 {
			t.Errorf("Expected no logs, got %d", len(logger.logs))
		}
	})

	t.Run("prints severity header", func(t *testing.T) {
		logger := &mockLogger{}
		printer := NewPrinter(logger, true, false)
		agg := NewAggregater()
		result := makeScanResult(ecrtypes.FindingSeverityCritical)
		agg.Add(result[0])

		findings := result[0].ImageScanFindings.Findings
		printer.logImageScanFindings(ecrtypes.FindingSeverityCritical, findings, agg)

		headerFound := false
		for _, log := range logger.logs {
			if containsString(log, "=== CRITICAL ===") {
				headerFound = true
				break
			}
		}
		if !headerFound {
			t.Error("Expected severity header to be printed")
		}
	})

	t.Run("prints CVE name and URI", func(t *testing.T) {
		logger := &mockLogger{}
		printer := NewPrinter(logger, true, false)
		agg := NewAggregater()
		result := makeScanResult(ecrtypes.FindingSeverityHigh)
		agg.Add(result[0])

		findings := result[0].ImageScanFindings.Findings
		printer.logImageScanFindings(ecrtypes.FindingSeverityHigh, findings, agg)

		cveFound := false
		uriFound := false
		for _, log := range logger.logs {
			if containsString(log, "CVE-2023-0001") {
				cveFound = true
			}
			if containsString(log, "http://example.com") {
				uriFound = true
			}
		}
		if !cveFound {
			t.Error("Expected CVE name to be printed")
		}
		if !uriFound {
			t.Error("Expected CVE URI to be printed")
		}
	})

	t.Run("prints container names", func(t *testing.T) {
		logger := &mockLogger{}
		printer := NewPrinter(logger, true, false)
		agg := NewAggregater()
		result := makeScanResult(ecrtypes.FindingSeverityMedium)
		agg.Add(result[0])

		findings := result[0].ImageScanFindings.Findings
		printer.logImageScanFindings(ecrtypes.FindingSeverityMedium, findings, agg)

		containerFound := false
		for _, log := range logger.logs {
			if containsString(log, "test-container") {
				containerFound = true
				break
			}
		}
		if !containerFound {
			t.Error("Expected container name to be printed")
		}
	})

	t.Run("prints description when logDetail is true", func(t *testing.T) {
		logger := &mockLogger{}
		printer := NewPrinter(logger, true, true) // logDetail = true
		agg := NewAggregater()
		result := makeScanResult(ecrtypes.FindingSeverityCritical)
		agg.Add(result[0])

		findings := result[0].ImageScanFindings.Findings
		printer.logImageScanFindings(ecrtypes.FindingSeverityCritical, findings, agg)

		descriptionFound := false
		for _, log := range logger.logs {
			if containsString(log, "Test vulnerability description") {
				descriptionFound = true
				break
			}
		}
		if !descriptionFound {
			t.Error("Expected description to be printed when logDetail is true")
		}
	})

	t.Run("does not print description when logDetail is false", func(t *testing.T) {
		logger := &mockLogger{}
		printer := NewPrinter(logger, true, false) // logDetail = false
		agg := NewAggregater()
		result := makeScanResult(ecrtypes.FindingSeverityCritical)
		agg.Add(result[0])

		findings := result[0].ImageScanFindings.Findings
		printer.logImageScanFindings(ecrtypes.FindingSeverityCritical, findings, agg)

		descriptionFound := false
		for _, log := range logger.logs {
			if containsString(log, "Test vulnerability description") {
				descriptionFound = true
				break
			}
		}
		if descriptionFound {
			t.Error("Expected description NOT to be printed when logDetail is false")
		}
	})

	t.Run("handles multiple findings", func(t *testing.T) {
		logger := &mockLogger{}
		printer := NewPrinter(logger, true, false)
		agg := NewAggregater()
		result := makeScanResult(
			ecrtypes.FindingSeverityHigh,
			ecrtypes.FindingSeverityHigh,
			ecrtypes.FindingSeverityHigh,
		)
		agg.Add(result[0])

		findings := result[0].ImageScanFindings.Findings
		printer.logImageScanFindings(ecrtypes.FindingSeverityHigh, findings, agg)

		cve1Found := false
		cve2Found := false
		cve3Found := false
		for _, log := range logger.logs {
			if containsString(log, "CVE-2023-0001") {
				cve1Found = true
			}
			if containsString(log, "CVE-2023-0002") {
				cve2Found = true
			}
			if containsString(log, "CVE-2023-0003") {
				cve3Found = true
			}
		}
		if !cve1Found || !cve2Found || !cve3Found {
			t.Error("Expected all three CVEs to be printed")
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
