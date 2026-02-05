package audit

import (
	"errors"
	"regexp"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type ImageInfo struct {
	ContainerName string
	Registry      string
	Repository    string
	Tag           string
	PlatFormOS    ecstypes.OSFamily
	PlatformArch  ecstypes.CPUArchitecture
}

func (i *ImageInfo) IsECRImage() bool {
	return i.Registry == "public.ecr.aws" || i.registryHasECRSuffix()
}

var ecrURLPattern = regexp.MustCompile(`^[0-9]{12}\.dkr\.ecr\.[a-zA-Z0-9-]+\.amazonaws\.com$`)

func (i *ImageInfo) registryHasECRSuffix() bool {
	return ecrURLPattern.MatchString(i.Registry)
}

type ScanResult struct {
	ImageInfo
	Cves []CVE
	Err  error
}

type ScanStatus string

const (
	ScanStatusOK         ScanStatus = "OK"
	ScanStatusWarning    ScanStatus = "WARNING"
	ScanStatusVulnerable ScanStatus = "VULNERABLE"
	ScanStatusError      ScanStatus = "ERROR"
	ScanStatusNA         ScanStatus = "N/A"
)

func (r ScanResult) Summary() ScanResultSummary {
	var status ScanStatus = ScanStatusOK
	var critical, high, medium, low, info int32
	cves := r.Cves
	for _, f := range cves {
		switch f.Severity {
		case ecrtypes.FindingSeverityCritical:
			critical++
		case ecrtypes.FindingSeverityHigh:
			high++
		case ecrtypes.FindingSeverityMedium:
			medium++
		case ecrtypes.FindingSeverityLow:
			low++
		case ecrtypes.FindingSeverityInformational:
			info++
		}
	}
	if errors.Is(r.Err, ErrScanNotFound) {
		status = ScanStatusNA
	} else if r.Err != nil {
		status = ScanStatusError
	} else if critical > 0 || high > 0 {
		status = ScanStatusVulnerable
	} else if medium > 0 {
		status = ScanStatusWarning
	}
	return ScanResultSummary{
		ContainerName: r.ContainerName,
		Status:        status,
		CriticalCount: critical,
		HighCount:     high,
		MediumCount:   medium,
		LowCount:      low,
		InfoCount:     info,
		ImageURI:      r.formatImageLabel(),
	}
}

type ScanResultSummary struct {
	ContainerName string     `json:"container_name"`
	Status        ScanStatus `json:"status"`
	CriticalCount int32      `json:"critical_count"`
	HighCount     int32      `json:"high_count"`
	MediumCount   int32      `json:"medium_count"`
	LowCount      int32      `json:"low_count"`
	InfoCount     int32      `json:"info_count"`
	ImageURI      string     `json:"image_uri"`
}

func unwrapAttributes(attrs []ecrtypes.Attribute) map[string]string {
	m := make(map[string]string)
	for _, attr := range attrs {
		if attr.Key != nil && attr.Value != nil {
			m[*attr.Key] = *attr.Value
		}
	}
	return m
}

func findingToCVE(finding ecrtypes.ImageScanFinding) CVE {
	cve := CVE{
		Name:           "unknown",
		PackageName:    "unknown",
		PackageVersion: "unknown",
		Severity:       finding.Severity,
	}
	if finding.Name != nil {
		cve.Name = *finding.Name
	}
	attrs := unwrapAttributes(finding.Attributes)
	if val, ok := attrs["package_name"]; ok {
		cve.PackageName = val
	}
	if val, ok := attrs["package_version"]; ok {
		cve.PackageVersion = val
	}
	if finding.Uri != nil {
		cve.Uri = *finding.Uri
	}
	if finding.Description != nil {
		cve.Description = *finding.Description
	}
	return cve
}

type FinalResult struct {
	Target
	Result
	ScannedAt string `json:"scanned_at"`
}

type Target struct {
	Region  string `json:"region"`
	Cluster string `json:"cluster"`
	Service string `json:"service"`
}

type Result struct {
	Summary *AggregateResult `json:"summary"`
	Vulns   []Vuln           `json:"vulns"`
}

type Vuln struct {
	CVE        CVE      `json:"cve"`
	Containers []string `json:"containers"`
}

type CVE struct {
	Name           string                   `json:"name"`
	Description    string                   `json:"description"`
	PackageName    string                   `json:"package_name"`
	PackageVersion string                   `json:"package_version"`
	Uri            string                   `json:"uri"`
	Severity       ecrtypes.FindingSeverity `json:"severity"`
}

func (a *Result) CriticalCves() []Vuln {
	return a.filterCvesBySeverity(ecrtypes.FindingSeverityCritical)
}

func (a *Result) HighCves() []Vuln {
	return a.filterCvesBySeverity(ecrtypes.FindingSeverityHigh)
}

func (a *Result) MediumCves() []Vuln {
	return a.filterCvesBySeverity(ecrtypes.FindingSeverityMedium)
}

func (a *Result) filterCvesBySeverity(severity ecrtypes.FindingSeverity) []Vuln {
	var vulns []Vuln
	for _, v := range a.Vulns {
		if v.CVE.Severity == severity {
			vulns = append(vulns, v)
		}
	}
	return vulns
}
