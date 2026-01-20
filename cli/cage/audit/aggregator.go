package audit

import (
	"fmt"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/logger"
)

type aggregater struct {
	cves          map[string]ecrtypes.ImageScanFinding
	cveToSeverity map[string]string
	// container name to summaries
	summaries map[string][]*ScanResultSummary
}

func NewAggregater() *aggregater {
	return &aggregater{
		cves:          make(map[string]ecrtypes.ImageScanFinding),
		cveToSeverity: make(map[string]string),
		summaries:     make(map[string][]*ScanResultSummary)}
}

func (a *aggregater) Add(r *ScanResult) {
	container := r.ContainerName()
	if r.Err != nil {
		a.summaries[container] = append(a.summaries[container], &ScanResultSummary{
			ContainerName: container,
			Status:        "ERROR",
		})
		return
	} else if r.ImageScanFindings == nil {
		a.summaries[container] = append(a.summaries[container], &ScanResultSummary{
			ContainerName: container,
			Status:        "N/A",
		})
		return
	}
	summary := summaryScanResult(r)
	a.summaries[container] = append(a.summaries[container], summary)
	for _, f := range r.ImageScanFindings.Findings {
		if _, exists := a.cves[*f.Name]; !exists {
			a.cves[*f.Name] = f
			a.cveToSeverity[*f.Name] = string(f.Severity)
		}
	}
}

type AggregateResult struct {
	CriticalCount   int32
	HighCount       int32
	MediumCount     int32
	LowCount        int32
	InfoCount       int32
	TotalCount      int32
	HighestSeverity ecrtypes.FindingSeverity
}

func (a *aggregater) SummarizeTotal() *AggregateResult {
	result := &AggregateResult{}
	highestServity := ecrtypes.FindingSeverityInformational
	for cve := range a.cves {
		severity := a.cveToSeverity[cve]
		switch severity {
		case string(ecrtypes.FindingSeverityCritical):
			result.CriticalCount++
		case string(ecrtypes.FindingSeverityHigh):
			result.HighCount++
		case string(ecrtypes.FindingSeverityMedium):
			result.MediumCount++
		case string(ecrtypes.FindingSeverityLow):
			result.LowCount++
		case string(ecrtypes.FindingSeverityInformational):
			result.InfoCount++
		}
	}
	if result.CriticalCount > 0 {
		highestServity = ecrtypes.FindingSeverityCritical
	} else if result.HighCount > 0 {
		highestServity = ecrtypes.FindingSeverityHigh
	} else if result.MediumCount > 0 {
		highestServity = ecrtypes.FindingSeverityMedium
	} else {
		highestServity = ecrtypes.FindingSeverityLow
	}
	result.HighestSeverity = highestServity
	result.TotalCount = int32(len(a.cves))
	return result
}

type SeverityCount struct {
	Severity ecrtypes.FindingSeverity
	Count    int
}

func (a *AggregateResult) SeverityCounts() []SeverityCount {
	return []SeverityCount{
		{Severity: ecrtypes.FindingSeverityInformational, Count: int(a.InfoCount)},
		{Severity: ecrtypes.FindingSeverityLow, Count: int(a.LowCount)},
		{Severity: ecrtypes.FindingSeverityMedium, Count: int(a.MediumCount)},
		{Severity: ecrtypes.FindingSeverityHigh, Count: int(a.HighCount)},
		{Severity: ecrtypes.FindingSeverityCritical, Count: int(a.CriticalCount)},
	}
}

func (a *aggregater) TotalCVECount() int {
	return len(a.cves)
}

func (a *aggregater) CriticalCves() []ecrtypes.ImageScanFinding {
	return a.filterCvesBySeverity(ecrtypes.FindingSeverityCritical)
}

func (a *aggregater) HighCves() []ecrtypes.ImageScanFinding {
	return a.filterCvesBySeverity(ecrtypes.FindingSeverityHigh)
}

func (a *aggregater) MediumCves() []ecrtypes.ImageScanFinding {
	return a.filterCvesBySeverity(ecrtypes.FindingSeverityMedium)
}

func (a *aggregater) filterCvesBySeverity(severity ecrtypes.FindingSeverity) []ecrtypes.ImageScanFinding {
	var cves []ecrtypes.ImageScanFinding
	for cve, sev := range a.cveToSeverity {
		if sev == string(severity) {
			cves = append(cves, a.cves[cve])
		}
	}
	return cves
}

type severityPrinter struct {
	noColor  bool
	severity ecrtypes.FindingSeverity
	color    logger.Color
}

func (s *severityPrinter) Sprintf(format string, a ...any) string {
	switch s.severity {
	case ecrtypes.FindingSeverityCritical:
		return s.color.Magentaf(format, a...)
	case ecrtypes.FindingSeverityHigh:
		return s.color.Redf(format, a...)
	case ecrtypes.FindingSeverityMedium:
		return s.color.Yellowf(format, a...)
	default:
		return fmt.Sprintf(format, a...)
	}
}

func (s *severityPrinter) BSprintf(format string, a ...any) string {
	return s.color.Boldf("%s", s.Sprintf(format, a...))
}
