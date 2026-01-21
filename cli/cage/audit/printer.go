package audit

import (
	"fmt"
	"strings"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/logger"
)

type printer struct {
	logger    logger.Logger
	color     logger.Color
	logDetail bool
}

func NewPrinter(l logger.Logger, noColor, logDetail bool) *printer {
	return &printer{
		logger:    l,
		color:     logger.Color{NoColor: noColor},
		logDetail: logDetail,
	}
}

func (p *printer) Print(result []*ScanResult) {
	containerMax, imageMax := MaxHeaderWidth(result)
	// |container|status|critical|high|medium|low|info|image|
	headerFmt := fmt.Sprintf("|%%-%ds|%%-10s|%%-8s|%%-5s|%%-6s|%%-4s|%%-4s|%%-%ds|\n", containerMax, imageMax)
	p.logger.Printf(headerFmt, "CONTAINER", "STATUS", "CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO", "IMAGE")
	bodyFmt := fmt.Sprintf("|%%-%ds|%%-10s|%%-8d|%%-5d|%%-6d|%%-4d|%%-4d|%%-%ds|\n", containerMax, imageMax)
	agg := NewAggregater()
	for _, r := range result {
		agg.Add(r)
	}
	for _, summaries := range agg.summaries {
		for _, summary := range summaries {
			p.logger.Printf(
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
	p.logImageScanFindings("CRITICAL", agg.CriticalCves(), agg)
	p.logImageScanFindings("HIGH", agg.HighCves(), agg)
	p.logImageScanFindings("MEDIUM", agg.MediumCves(), agg)
	total := agg.TotalCVECount()
	color := p.color
	if total == 0 {
		p.logger.Printf("%s\n", color.Greenf("No CVEs found"))
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

	p.logger.Printf(
		"\nTotal: %s (%s)\n",
		highest.BSprintf("%d", summary.TotalCount),
		strings.Join(list, ", "),
	)
}

func (p *printer) logImageScanFindings(
	serverity ecrtypes.FindingSeverity,
	findings []ecrtypes.ImageScanFinding,
	aggregater *aggregater,
) {
	if len(findings) == 0 {
		return
	}
	sp := &severityPrinter{severity: serverity, color: p.color}
	color := p.color
	p.logger.Printf("\n=== %s ===\n", sp.BSprintf("%s", serverity))
	for _, cve := range findings {
		containers := aggregater.GetVulnContainers(*cve.Name)
		var containerList []string
		for _, c := range containers {
			containerList = append(containerList, color.Boldf("%s", c))
		}
		p.logger.Printf("- %s \n", strings.Join(containerList, ", "))
		p.logger.Printf("  %s (%s)\n", *cve.Name, *cve.Uri)
		if p.logDetail {
			p.logger.Printf("\n%s\n", *cve.Description)
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
