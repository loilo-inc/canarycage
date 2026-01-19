package scan

import (
	"fmt"

	"github.com/loilo-inc/canarycage/logger"
)

type printer struct {
	logger logger.Logger
}

type Printer interface {
	Print(result []*ScanResult)
}

func NewPrinter(logger logger.Logger) Printer {
	return &printer{logger: logger}
}

func (p *printer) Print(result []*ScanResult) {
	containerMax, imageMax := MaxHeaderWidth(result)
	// |container|status|critical|high|medium|low|info|image|
	headerFmt := fmt.Sprintf("|%%-%ds|%%-10s|%%-8s|%%-5s|%%-6s|%%-4s|%%-4s|%%-%ds|\n", containerMax, imageMax)
	p.logger.Printf(headerFmt, "CONTAINER", "STATUS", "CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO", "IMAGE")
	bodyFmt := fmt.Sprintf("|%%-%ds|%%-10s|%%-8d|%%-5d|%%-6d|%%-4d|%%-4d|%%-%ds|\n", containerMax, imageMax)
	for _, r := range result {
		if r.Err != nil {
			p.logger.Printf(bodyFmt,
				r.ImageInfo.ContainerName,
				"ERROR", 0, 0, 0, 0, 0,
				formatImageLabel(r.ImageInfo),
			)
			continue
		}
		findings := r.ImageScanFindings
		var critical, high, medium, low, info int32
		for _, f := range findings.Findings {
			switch f.Severity {
			case "CRITICAL":
				critical++
			case "HIGH":
				high++
			case "MEDIUM":
				medium++
			case "LOW":
				low++
			case "INFORMATIONAL":
				info++
			}
		}
		status := "OK"
		if len(findings.Findings) == 0 {
			status = "NONE"
		} else if critical > 0 || high > 0 {
			status = "VULNERABLE"
		} else if medium > 0 {
			status = "WARNING"
		}
		p.logger.Printf(
			bodyFmt,
			r.ImageInfo.ContainerName,
			status,
			critical,
			high,
			medium,
			low,
			info,
			formatImageLabel(r.ImageInfo),
		)
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
