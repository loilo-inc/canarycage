package audit

import (
	"encoding/json"
	"fmt"
	"strings"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/logger"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
)

type printer struct {
	di        *di.D
	color     logger.Color
	logDetail bool
}

type Printer interface {
	Print(result []*ScanResult)
	PrintJSON(metadata *Target, scanResults []*ScanResult)
}

func NewPrinter(di *di.D, noColor, logDetail bool) *printer {
	return &printer{
		di:        di,
		color:     logger.Color{NoColor: noColor},
		logDetail: logDetail,
	}
}

func (p *printer) Print(scanResults []*ScanResult) {
	l := p.di.Get(key.Logger).(logger.Logger)
	containerMax, imageMax := MaxHeaderWidth(scanResults)
	// |container|status|critical|high|medium|low|info|image|
	headerFmt := fmt.Sprintf("|%%-%ds|%%-10s|%%-8s|%%-5s|%%-6s|%%-4s|%%-4s|%%-%ds|\n", containerMax, imageMax)
	l.Printf(headerFmt, "CONTAINER", "STATUS", "CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO", "IMAGE")
	bodyFmt := fmt.Sprintf("|%%-%ds|%%-10s|%%-8d|%%-5d|%%-6d|%%-4d|%%-4d|%%-%ds|\n", containerMax, imageMax)
	agg := NewAggregater()
	for _, r := range scanResults {
		agg.Add(r)
	}
	for _, summaries := range agg.summaries {
		for _, summary := range summaries {
			l.Printf(
				bodyFmt,
				summary.ContainerName,
				summary.Status,
				summary.CriticalCount,
				summary.HighCount,
				summary.MediumCount,
				summary.LowCount,
				summary.InfoCount,
				summary.ImageURI,
			)
		}
	}
	result := agg.Result()
	p.logVuln(ecrtypes.FindingSeverityCritical, result.CriticalCves())
	p.logVuln(ecrtypes.FindingSeverityHigh, result.HighCves())
	p.logVuln(ecrtypes.FindingSeverityMedium, result.MediumCves())
	total := agg.TotalCVECount()
	color := p.color
	if total == 0 {
		l.Printf("%s\n", color.Greenf("No CVEs found"))
		return
	}
	summary := agg.SummarizeTotal()
	highest := &severityPrinter{
		severity: summary.HighestSeverity,
		color:    p.color,
	}
	var list []string
	for _, v := range summary.SeverityCounts() {
		if v.Count == 0 {
			continue
		}
		sp := &severityPrinter{severity: v.Severity, color: p.color}
		list = append(list, fmt.Sprintf("%d %s", v.Count, sp.BSprintf("%s", v.Severity)))
	}

	l.Printf(
		"\nTotal: %s (%s)\n",
		highest.BSprintf("%d", summary.TotalCount),
		strings.Join(list, ", "),
	)
}

func (p *printer) PrintJSON(metadata Target, scanResults []*ScanResult) {
	l := p.di.Get(key.Logger).(logger.Logger)
	t := p.di.Get(key.Time).(types.Time)
	agg := NewAggregater()
	for _, r := range scanResults {
		agg.Add(r)
	}
	aggResult := agg.Result()
	finalResult := &FinalResult{
		Target:    metadata,
		Result:    aggResult,
		ScannedAt: t.Now().Format("2006-01-02T15:04:05Z07:00"),
	}
	if data, err := json.MarshalIndent(finalResult, "", "  "); err != nil {
		l.Errorf("failed to marshal JSON output: %v", err)
	} else {
		l.Printf("%s\n", data)
	}
}

func (p *printer) logVuln(severity ecrtypes.FindingSeverity, vulns []Vuln) {
	if len(vulns) == 0 {
		return
	}
	sp := &severityPrinter{severity: severity, color: p.color}
	l := p.di.Get(key.Logger).(logger.Logger)
	l.Printf("\n=== %s ===\n", sp.BSprintf("%s", severity))
	for _, vuln := range vulns {
		cve := vuln.CVE
		var containerList []string
		for _, c := range vuln.Containers {
			containerList = append(containerList, p.color.Bold(c))
		}
		l.Printf("- %s %s \n", cve.Name, strings.Join(containerList, ", "))
		l.Printf("  %s::%s (%s)\n", cve.PackageName, cve.PackageVersion, cve.Uri)
		if p.logDetail {
			l.Printf("\n%s\n", cve.Description)
		}
	}
}

func (i *ImageInfo) formatImageLabel() string {
	return fmt.Sprintf("%s/%s:%s", i.Registry, i.Repository, i.Tag)
}

func MaxHeaderWidth(imageInfos []*ScanResult) (int, int) {
	containerMax := len("CONTAINER")
	imageMax := len("IMAGE")
	for _, info := range imageInfos {
		if l := len(info.ImageInfo.ContainerName); l > containerMax {
			containerMax = l
		}
		imageLabel := info.formatImageLabel()
		if l := len(imageLabel); l > imageMax {
			imageMax = l
		}
	}
	return containerMax, imageMax
}
