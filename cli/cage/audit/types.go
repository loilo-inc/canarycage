package audit

import (
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

type ScanResultSummary struct {
	ContainerName string `json:"container_name"`
	Status        string `json:"status"`
	CriticalCount int32  `json:"critical_count"`
	HighCount     int32  `json:"high_count"`
	MediumCount   int32  `json:"medium_count"`
	LowCount      int32  `json:"low_count"`
	InfoCount     int32  `json:"info_count"`
	ImageURI      string `json:"image_uri"`
}

func findingToCVE(finding ecrtypes.ImageScanFinding) CVE {
	var packageName string
	var packageVersion string
	for _, attr := range finding.Attributes {
		switch *attr.Key {
		case "package_name":
			packageName = *attr.Value
		case "package_version":
			packageVersion = *attr.Value
		}
	}
	var uri string
	var description string
	if finding.Uri != nil {
		uri = *finding.Uri
	}
	if finding.Description != nil {
		description = *finding.Description
	}
	return CVE{
		Name:           *finding.Name,
		Severity:       finding.Severity,
		Description:    description,
		Uri:            uri,
		PackageName:    packageName,
		PackageVersion: packageVersion,
	}
}

func summaryScanResult(result *ScanResult) *ScanResultSummary {
	var status = "OK"
	var critical, high, medium, low, info int32
	cves := result.Cves
	for _, f := range cves {
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
	if len(result.Cves) == 0 {
		status = "NONE"
	} else if critical > 0 || high > 0 {
		status = "VULNERABLE"
	} else if medium > 0 {
		status = "WARNING"
	}
	return &ScanResultSummary{
		ContainerName: result.ContainerName,
		Status:        status,
		CriticalCount: critical,
		HighCount:     high,
		MediumCount:   medium,
		LowCount:      low,
		InfoCount:     info,
		ImageURI:      result.formatImageLabel(),
	}
}

type FinalResult struct {
	*Resource
	*Result
	ScannedAt string `json:"scanned_at"`
}

type Resource struct {
	Region  string `json:"region"`
	Cluster string `json:"cluster"`
	Service string `json:"service"`
}

type Result struct {
	Summary *AggregateResult `json:"summary"`
	Vulns   []*Vuln          `json:"vulns"`
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

func (a *Result) CriticalCves() []*Vuln {
	return a.filterCvesBySeverity(ecrtypes.FindingSeverityCritical)
}

func (a *Result) HighCves() []*Vuln {
	return a.filterCvesBySeverity(ecrtypes.FindingSeverityHigh)
}

func (a *Result) MediumCves() []*Vuln {
	return a.filterCvesBySeverity(ecrtypes.FindingSeverityMedium)
}

func (a *Result) filterCvesBySeverity(severity ecrtypes.FindingSeverity) []*Vuln {
	var vulns []*Vuln
	for _, v := range a.Vulns {
		if v.CVE.Severity == severity {
			vulns = append(vulns, v)
		}
	}
	return vulns
}
