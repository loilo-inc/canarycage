package audit

import (
	"fmt"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/logger"
)

type aggregater struct {
	cves            map[string]CVE
	cveToSeverity   map[string]ecrtypes.FindingSeverity
	cveToContainers map[string][]string
	// container name to summaries
	summaries map[string][]*ScanResultSummary
}

func NewAggregater() *aggregater {
	return &aggregater{
		cves:            make(map[string]CVE),
		cveToSeverity:   make(map[string]ecrtypes.FindingSeverity),
		cveToContainers: make(map[string][]string),
		summaries:       make(map[string][]*ScanResultSummary)}
}

func (a *aggregater) Add(r *ScanResult) {
	container := r.ContainerName
	if r.Err != nil {
		a.summaries[container] = append(a.summaries[container], &ScanResultSummary{
			ContainerName: container,
			Status:        "ERROR",
		})
		return
	} else if len(r.Cves) == 0 {
		a.summaries[container] = append(a.summaries[container], &ScanResultSummary{
			ContainerName: container,
			Status:        "N/A",
		})
		return
	}
	summary := summaryScanResult(r)
	a.summaries[container] = append(a.summaries[container], summary)
	for _, f := range r.Cves {
		if _, exists := a.cves[f.Name]; !exists {
			a.cves[f.Name] = f
			a.cveToSeverity[f.Name] = f.Severity
			a.cveToContainers[f.Name] = append(a.cveToContainers[f.Name], container)
		}
	}
}

type AggregateResult struct {
	CriticalCount   int
	HighCount       int
	MediumCount     int
	LowCount        int
	InfoCount       int
	TotalCount      int
	HighestSeverity ecrtypes.FindingSeverity
}

func (a *aggregater) SummarizeTotal() *AggregateResult {
	result := &AggregateResult{}
	highest := ecrtypes.FindingSeverityInformational
	for cve := range a.cves {
		severity := a.cveToSeverity[cve]
		switch severity {
		case ecrtypes.FindingSeverityCritical:
			result.CriticalCount++
		case ecrtypes.FindingSeverityHigh:
			result.HighCount++
		case ecrtypes.FindingSeverityMedium:
			result.MediumCount++
		case ecrtypes.FindingSeverityLow:
			result.LowCount++
		case ecrtypes.FindingSeverityInformational:
			result.InfoCount++
		}
	}
	if result.CriticalCount > 0 {
		highest = ecrtypes.FindingSeverityCritical
	} else if result.HighCount > 0 {
		highest = ecrtypes.FindingSeverityHigh
	} else if result.MediumCount > 0 {
		highest = ecrtypes.FindingSeverityMedium
	} else if result.LowCount > 0 {
		highest = ecrtypes.FindingSeverityLow
	} else {
		highest = ecrtypes.FindingSeverityInformational
	}
	result.HighestSeverity = highest
	result.TotalCount = len(a.cves)
	return result
}

type SeverityCount struct {
	Severity ecrtypes.FindingSeverity
	Count    int
}

func (a *AggregateResult) SeverityCounts() []SeverityCount {
	return []SeverityCount{
		{Severity: ecrtypes.FindingSeverityInformational, Count: a.InfoCount},
		{Severity: ecrtypes.FindingSeverityLow, Count: a.LowCount},
		{Severity: ecrtypes.FindingSeverityMedium, Count: a.MediumCount},
		{Severity: ecrtypes.FindingSeverityHigh, Count: a.HighCount},
		{Severity: ecrtypes.FindingSeverityCritical, Count: a.CriticalCount},
	}
}

func (a *aggregater) TotalCVECount() int {
	return len(a.cves)
}

func (a *aggregater) GetVulnContainers(cveName string) []string {
	containersSet := a.cveToContainers[cveName]
	return containersSet
}

func (a *aggregater) Result() *Result {
	summary := a.SummarizeTotal()
	var vulns []*Vuln
	for _, cve := range a.cves {
		vuln := &Vuln{
			Containers: a.GetVulnContainers(cve.Name),
			CVE:        cve,
		}
		vulns = append(vulns, vuln)
	}
	return &Result{Summary: summary, Vulns: vulns}
}

type severityPrinter struct {
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
	return s.color.Bold(s.Sprintf(format, a...))
}
