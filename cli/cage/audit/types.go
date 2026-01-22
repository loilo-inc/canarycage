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
	ImageScanFindings *ecrtypes.ImageScanFindings
	Err               error
}

type ScanResultSummary struct {
	ContainerName string
	Status        string
	CriticalCount int32
	HighCount     int32
	MediumCount   int32
	LowCount      int32
	InfoCount     int32
	ImageURI      string
}

func summaryScanResult(result *ScanResult) *ScanResultSummary {
	var status = "OK"
	var critical, high, medium, low, info int32
	findings := result.ImageScanFindings
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
	if len(result.ImageScanFindings.Findings) == 0 {
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
