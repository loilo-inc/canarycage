package scan_test

import (
	"fmt"
	"strings"
	"testing"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/cli/cage/scan"
)

type mockLogger struct {
	logs []string
}

func (m *mockLogger) Printf(format string, args ...any) {
	m.logs = append(m.logs, fmt.Sprintf(format, args...))
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
			name: "single result with no findings",
			results: []*scan.ScanResult{
				{
					ImageInfo: &scan.ImageInfo{
						ContainerName: "container1",
						Registry:      "registry.io",
						Repository:    "myapp",
						Tag:           "v1.0",
					},
					ImageScanFindings: &ecrtypes.ImageScanFindings{
						Findings: []ecrtypes.ImageScanFinding{},
					},
				},
			},
			expectedLines:  2, // header + 1 body
			expectedStatus: []string{"NONE"},
			expectedCounts: [][]int32{{0, 0, 0, 0, 0}},
		},
		{
			name: "single result with vulnerabilities",
			results: []*scan.ScanResult{
				{
					ImageInfo: &scan.ImageInfo{
						ContainerName: "container2",
						Registry:      "registry.io",
						Repository:    "myapp",
						Tag:           "v2.0",
					},
					ImageScanFindings: &ecrtypes.ImageScanFindings{
						Findings: []ecrtypes.ImageScanFinding{
							{Severity: "CRITICAL"},
							{Severity: "HIGH"},
							{Severity: "MEDIUM"},
							{Severity: "LOW"},
							{Severity: "INFORMATIONAL"},
						},
					},
				},
			},
			expectedLines:  2,
			expectedStatus: []string{"VULNERABLE"},
			expectedCounts: [][]int32{{1, 1, 1, 1, 1}},
		},
		{
			name: "result with error",
			results: []*scan.ScanResult{
				{
					ImageInfo: &scan.ImageInfo{
						ContainerName: "container3",
						Registry:      "registry.io",
						Repository:    "myapp",
						Tag:           "v3.0",
					},
					Err: fmt.Errorf("scan failed"),
				},
			},
			expectedLines:  2,
			expectedStatus: []string{"ERROR"},
			expectedCounts: [][]int32{{0, 0, 0, 0, 0}},
		},
		{
			name: "multiple results mixed",
			results: []*scan.ScanResult{
				{
					ImageInfo: &scan.ImageInfo{
						ContainerName: "container4",
						Registry:      "registry.io",
						Repository:    "app1",
						Tag:           "v1",
					},
					ImageScanFindings: &ecrtypes.ImageScanFindings{
						Findings: []ecrtypes.ImageScanFinding{
							{Severity: "CRITICAL"},
							{Severity: "CRITICAL"},
						},
					},
				},
				{
					ImageInfo: &scan.ImageInfo{
						ContainerName: "container5",
						Registry:      "registry.io",
						Repository:    "app2",
						Tag:           "v2",
					},
					ImageScanFindings: &ecrtypes.ImageScanFindings{
						Findings: []ecrtypes.ImageScanFinding{},
					},
				},
				{
					ImageInfo: &scan.ImageInfo{
						ContainerName: "container6",
						Registry:      "registry.io",
						Repository:    "app3",
						Tag:           "v3",
					},
					Err: fmt.Errorf("error"),
				},
			},
			expectedLines:  4, // header + 3 bodies
			expectedStatus: []string{"VULNERABLE", "NONE", "ERROR"},
			expectedCounts: [][]int32{{2, 0, 0, 0, 0}, {0, 0, 0, 0, 0}, {0, 0, 0, 0, 0}},
		},
		{
			name: "result with only low severity",
			results: []*scan.ScanResult{
				{
					ImageInfo: &scan.ImageInfo{
						ContainerName: "container7",
						Registry:      "registry.io",
						Repository:    "app",
						Tag:           "v1",
					},
					ImageScanFindings: &ecrtypes.ImageScanFindings{
						Findings: []ecrtypes.ImageScanFinding{
							{Severity: "LOW"},
							{Severity: "INFORMATIONAL"},
						},
					},
				},
			},
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

			if len(logger.logs) != tt.expectedLines {
				t.Errorf("expected %d log lines, got %d", tt.expectedLines, len(logger.logs))
			}

			// Check header is present
			if len(logger.logs) > 0 {
				header := logger.logs[0]
				if !strings.Contains(header, "CONTAINER") || !strings.Contains(header, "STATUS") {
					t.Errorf("expected header to contain CONTAINER and STATUS, got: %s", header)
				}
			}

			// Check statuses
			for i, expectedStatus := range tt.expectedStatus {
				bodyLine := logger.logs[i+1]
				if !strings.Contains(bodyLine, expectedStatus) {
					t.Errorf("expected line %d to contain status %s, got: %s", i+1, expectedStatus, bodyLine)
				}
			}

			// Check container names
			for i, result := range tt.results {
				bodyLine := logger.logs[i+1]
				if !strings.Contains(bodyLine, result.ImageInfo.ContainerName) {
					t.Errorf("expected line %d to contain container name %s, got: %s", i+1, result.ImageInfo.ContainerName, bodyLine)
				}
			}
		})
	}
}

func TestPrinter_Print_EmptyResults(t *testing.T) {
	logger := &mockLogger{logs: []string{}}
	printer := scan.NewPrinter(logger)

	printer.Print([]*scan.ScanResult{})

	if len(logger.logs) != 1 {
		t.Errorf("expected 1 log line (header only), got %d", len(logger.logs))
	}
}
