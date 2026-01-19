package scan

import "fmt"

type printer struct {
	logger Logger
}
type Logger interface {
	Printf(format string, args ...any)
}
type Printer interface {
	Print(result []*ScanResult)
}

func DefaultLogger() Logger {
	return &defaultLogger{}
}

type defaultLogger struct{}

func (l *defaultLogger) Printf(format string, args ...any) {
	fmt.Printf(format, args...)
}

func NewPrinter(logger Logger) Printer {
	return &printer{logger: logger}
}

func (p *printer) Print(result []*ScanResult) {
	// |image|status|critical|high|medium|low|info|error|
	fmtStr := "|%-40s|%-10s|%-8d|%-5d|%-6d|%-4d|%-4d|%-5d|\n"
	p.logger.Printf(fmtStr, "IMAGE", "STATUS", "CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO", "UNKNOWN")
	for _, r := range result {
		if r.Err != nil {
			p.logger.Printf(fmtStr, formatImageLabel(r.ImageInfo), "ERROR", 0, 0, 0, 0, 0, 1)
			continue
		}
		findings := r.ImageScanFindings
		var critical, high, medium, low, info, unclassified int32
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
			case "UNCLASSIFIED":
				unclassified++
			}
		}
		status := "OK"
		if len(findings.Findings) == 0 {
			status = "NONE"
		} else if critical > 0 || high > 0 {
			status = "VULNERABLE"
		}
		p.logger.Printf(
			fmtStr,
			formatImageLabel(r.ImageInfo),
			status,
			critical,
			high,
			medium,
			low,
			info,
			unclassified,
		)
	}
}

func formatImageLabel(info *ImageInfo) string {
	return fmt.Sprintf("%s (%s:%s)", info.Registry, info.Repository, info.Tag)
}
