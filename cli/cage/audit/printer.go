package audit

import (
	"fmt"
	"strings"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/cli/color"
	"github.com/loilo-inc/canarycage/logger"
)

type Printer struct {
	Logger    logger.Logger
	LogDetail bool
	NoColor   bool
}

func (p *Printer) Print(result []*ScanResult) {
	containerMax, imageMax := MaxHeaderWidth(result)
	// |container|status|critical|high|medium|low|info|image|
	headerFmt := fmt.Sprintf("|%%-%ds|%%-10s|%%-8s|%%-5s|%%-6s|%%-4s|%%-4s|%%-%ds|\n", containerMax, imageMax)
	p.Logger.Printf(headerFmt, "CONTAINER", "STATUS", "CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO", "IMAGE")
	bodyFmt := fmt.Sprintf("|%%-%ds|%%-10s|%%-8d|%%-5d|%%-6d|%%-4d|%%-4d|%%-%ds|\n", containerMax, imageMax)
	agg := NewAggregater()
	for _, r := range result {
		agg.Add(r)
	}
	for _, summaries := range agg.summaries {
		for _, summary := range summaries {
			p.Logger.Printf(
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
	p.logImageScanFindings("CRITICAL", agg.CriticalCves())
	p.logImageScanFindings("HIGH", agg.HighCves())
	p.logImageScanFindings("MEDIUM", agg.MediumCves())
	total := agg.TotalCVECount()
	chalk := color.Color{NoColor: p.NoColor}
	if total == 0 {
		p.Logger.Printf("%s\n", chalk.Greenf("No CVEs found"))
		return
	}
	summary := agg.SummarizeTotal()
	highest := &severityPrinter{
		severity: summary.HighestSeverity,
	}
	var list []string
	for _, v := range summary.SeverityCounts() {
		if v.Count == 0 {
			continue
		}
		sp := &severityPrinter{severity: v.Severity}
		list = append(list, fmt.Sprintf("%d %s", v.Count, sp.BSprintf("%s", v.Severity)))
	}

	p.Logger.Printf(
		"Total: %s (%s)\n",
		highest.BSprintf("%d", summary.TotalCount),
		strings.Join(list, ", "),
	)
}

func (p *Printer) logImageScanFindings(serverity ecrtypes.FindingSeverity, findings []ecrtypes.ImageScanFinding) {
	if len(findings) == 0 {
		return
	}
	sp := &severityPrinter{severity: serverity}
	p.Logger.Printf("=== %s ===\n", sp.BSprintf("%s", serverity))
	for _, cve := range findings {
		p.Logger.Printf("- %s (%s)\n", *cve.Name, *cve.Uri)
		if p.LogDetail && cve.Description != nil {
			p.Logger.Printf("%s\n", *cve.Description)
		}
	}
}

func formatImageLabel(info *ImageInfo) string {
	return fmt.Sprintf("%s/%s:%s", info.Registry, info.Repository, info.Tag)
}

func MaxHeaderWidth(imageInfos []*ScanResult) (int, int) {
	containerMax := len("CONTAINER")
	imageMax := len("IMAGE")
	for _, info := range imageInfos {
		if l := len(info.ImageInfo.ContainerName); l > containerMax {
			containerMax = l
		}
		imageLabel := formatImageLabel(info.ImageInfo)
		if l := len(imageLabel); l > imageMax {
			imageMax = l
		}
	}
	return containerMax, imageMax
}
