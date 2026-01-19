package scan_test

import (
	"fmt"
	"testing"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/cli/cage/scan"
	"github.com/stretchr/testify/assert"
)

type mockLogger struct {
	logs []string
}

func (m *mockLogger) Printf(format string, args ...any) {
	m.logs = append(m.logs, fmt.Sprintf(format, args...))
}

var imageInfo = &scan.ImageInfo{
	ContainerName: "test-container",
	Registry:      "test-registry",
	Repository:    "test-repo",
	Tag:           "latest",
}

func TestPrinter_Print(t *testing.T) {
	tests := []struct {
		name           string
		results        []*scan.ScanResult
		expectedLines  int
		expectedStatus []string
		expectedCounts [][]int32
	}{
		{
			name:           "single result with no findings",
			results:        makeScanResult(),
			expectedLines:  2, // header + 1 body
			expectedStatus: []string{"NONE"},
			expectedCounts: [][]int32{{0, 0, 0, 0, 0}},
		},
		{
			name: "single result with vulnerabilities",
			results: makeScanResult(
				"CRITICAL",
				"HIGH",
				"MEDIUM",
				"LOW",
				"INFORMATIONAL",
			),
			expectedLines:  2,
			expectedStatus: []string{"VULNERABLE"},
			expectedCounts: [][]int32{{1, 1, 1, 1, 1}},
		},
		{
			name:           "single result with only medium severity",
			results:        makeScanResult("MEDIUM", "MEDIUM"),
			expectedLines:  2,
			expectedStatus: []string{"WARNING"},
			expectedCounts: [][]int32{{0, 0, 2, 0, 0}},
		},
		{
			name: "result with error",
			results: []*scan.ScanResult{
				{
					ImageInfo: imageInfo,
					Err:       fmt.Errorf("scan failed"),
				},
			},
			expectedLines:  2,
			expectedStatus: []string{"ERROR"},
			expectedCounts: [][]int32{{0, 0, 0, 0, 0}},
		},
		{
			name: "multiple results mixed",
			results: []*scan.ScanResult{
				makeScanResult("CRITICAL", "HIGH")[0],
				{
					ImageInfo: imageInfo,
					ImageScanFindings: &ecrtypes.ImageScanFindings{
						Findings: []ecrtypes.ImageScanFinding{},
					},
				},
				{
					ImageInfo: imageInfo,
					Err:       fmt.Errorf("error"),
				},
			},
			expectedLines:  4, // header + 3 bodies
			expectedStatus: []string{"VULNERABLE", "NONE", "ERROR"},
		},
		{
			name:           "result with only low severity",
			results:        makeScanResult("LOW", "INFORMATIONAL"),
			expectedLines:  2,
			expectedStatus: []string{"OK"},
			expectedCounts: [][]int32{{0, 0, 0, 1, 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &mockLogger{logs: []string{}}
			printer := scan.NewPrinter(logger)

			printer.Print(tt.results)

			assert.Equal(t, tt.expectedLines, len(logger.logs), "unexpected number of log lines")

			// Check header is present
			if len(logger.logs) > 0 {
				header := logger.logs[0]
				assert.Contains(t, header, "CONTAINER", "header should contain CONTAINER")
				assert.Contains(t, header, "STATUS", "header should contain STATUS")
			}

			// Check statuses
			for i, expectedStatus := range tt.expectedStatus {
				bodyLine := logger.logs[i+1]
				assert.Contains(t, bodyLine, expectedStatus, "line should contain expected status")
			}

			// Check container names
			for i, result := range tt.results {
				bodyLine := logger.logs[i+1]
				assert.Contains(t, bodyLine, result.ImageInfo.ContainerName, "line should contain container name")
			}
		})
	}
}

func TestPrinter_Print_EmptyResults(t *testing.T) {
	logger := &mockLogger{logs: []string{}}
	printer := scan.NewPrinter(logger)

	printer.Print([]*scan.ScanResult{})

	assert.Equal(t, 1, len(logger.logs), "expected header only")
}

func makeScanResult(
	list ...ecrtypes.FindingSeverity) []*scan.ScanResult {
	return []*scan.ScanResult{
		{
			ImageInfo: &scan.ImageInfo{
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
		findings[i] = ecrtypes.ImageScanFinding{Severity: sev}
	}
	return findings
}
