package audit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	smithy "github.com/aws/smithy-go"
	"github.com/loilo-inc/canarycage/v5/awsiface"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/logos/v2/di"
)

type scanner struct {
	di *di.D
}

type ScanRequest struct {
	Cluster string
	Service string
	InPlace bool
}

type Scanner interface {
	Scan(ctx context.Context, req ScanRequest) ([]ScanResult, error)
}

var _ Scanner = (*scanner)(nil)

func NewScanner(di *di.D) Scanner {
	return &scanner{di: di}
}

func (s *scanner) Scan(
	ctx context.Context,
	req ScanRequest,
) (results []ScanResult, err error) {
	ecsTool := newEcsTool(s.di.Get(key.EcsCli).(awsiface.EcsClient))
	ecrTool := newEcrTool(s.di.Get(key.EcrCli).(awsiface.EcrClient), &waiter{di: s.di})
	var imageInfos []ImageInfo
	if imageInfos, err = ecsTool.GetServiceImageInfos(ctx, req.Cluster, req.Service); err != nil {
		return nil, err
	}
	findingsList := make([]ScanResult, len(imageInfos))
	for i, info := range imageInfos {
		if info.IsECRImage() {
			findingsList[i] = s.scanImage(ctx, ecrTool, info, req.InPlace)
		} else {
			findingsList[i] = ScanResult{ImageInfo: info, Err: ErrNonEcrImage}
		}
	}
	return findingsList, nil
}

var ErrNonEcrImage = fmt.Errorf("non-ECR image")
var ErrScanNotFound = fmt.Errorf("scan not found")

func (s *scanner) scanImage(ctx context.Context, ecrTool EcrTool, info ImageInfo, inPlace bool) ScanResult {
	imageID, err := ecrTool.GetActualImageIdentifier(ctx, &info)
	if err != nil {
		return ScanResult{ImageInfo: info, Err: err}
	}
	if imageID.ImageDigest != nil {
		info.Digest = *imageID.ImageDigest
	}

	var findings *ecrtypes.ImageScanFindings
	if inPlace {
		findings, err = ecrTool.ExecImageScanFindings(ctx, &info, imageID)
	} else {
		findings, err = ecrTool.GetImageScanFindings(ctx, &info, imageID)
	}

	if err != nil {
		return ScanResult{ImageInfo: info, Err: parseError(err)}
	}

	cves := scanFindingsToCVEs(findings)
	return ScanResult{ImageInfo: info, Cves: cves}
}

func parseError(err error) error {
	var awserr smithy.APIError
	if errors.As(err, &awserr) && awserr.ErrorCode() == "ScanNotFoundException" {
		return ErrScanNotFound
	}
	return err
}

func scanFindingsToCVEs(findings *ecrtypes.ImageScanFindings) []CVE {
	var cves []CVE
	if len(findings.Findings) > 0 {
		for _, f := range findings.Findings {
			cves = append(cves, findingToCVE(f))
		}
	} else if len(findings.EnhancedFindings) > 0 {
		for _, f := range findings.EnhancedFindings {
			cves = append(cves, enhancedFindingToCVE(f))
		}
	}
	return cves
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

func enhancedFindingToCVE(finding ecrtypes.EnhancedImageScanFinding) CVE {
	analysis := &EnhancedAnalysis{Score: finding.Score}
	cve := CVE{
		Name:             "unknown",
		PackageName:      "unknown",
		PackageVersion:   "unknown",
		Severity:         ecrtypes.FindingSeverityUndefined,
		EnhancedAnalysis: analysis,
	}
	if finding.Description != nil {
		cve.Description = *finding.Description
	}
	if finding.Severity != nil {
		cve.Severity = ecrtypes.FindingSeverity(*finding.Severity)
	}
	if finding.Status != nil {
		analysis.Status = *finding.Status
	}
	if finding.ExploitAvailable != nil {
		analysis.ExploitAvailable = *finding.ExploitAvailable
	}
	if finding.FixAvailable != nil {
		analysis.FixAvailable = *finding.FixAvailable
	}
	if finding.PackageVulnerabilityDetails == nil {
		return cve
	}
	details := finding.PackageVulnerabilityDetails
	if details.VulnerabilityId != nil {
		cve.Name = *details.VulnerabilityId
	}
	if cve.Severity == ecrtypes.FindingSeverityUndefined && details.VendorSeverity != nil {
		cve.Severity = ecrtypes.FindingSeverity(*details.VendorSeverity)
	}
	if details.SourceUrl != nil {
		cve.Uri = *details.SourceUrl
	} else if len(details.ReferenceUrls) > 0 {
		cve.Uri = details.ReferenceUrls[0]
	}
	cve.PackageName, cve.PackageVersion = vulnerablePackagesToNameVersion(details.VulnerablePackages)
	analysis.FixedInVersion = strings.Join(uniquePackageValues(details.VulnerablePackages, func(pkg ecrtypes.VulnerablePackage) *string {
		return pkg.FixedInVersion
	}), ", ")
	return cve
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

func vulnerablePackagesToNameVersion(packages []ecrtypes.VulnerablePackage) (string, string) {
	names := uniquePackageValues(packages, func(pkg ecrtypes.VulnerablePackage) *string {
		return pkg.Name
	})
	versions := uniquePackageValues(packages, func(pkg ecrtypes.VulnerablePackage) *string {
		return pkg.Version
	})
	return joinOrUnknown(names), joinOrUnknown(versions)
}

func uniquePackageValues(packages []ecrtypes.VulnerablePackage, value func(ecrtypes.VulnerablePackage) *string) []string {
	seen := make(map[string]struct{})
	var values []string
	for _, pkg := range packages {
		v := value(pkg)
		if v == nil || *v == "" {
			continue
		}
		if _, ok := seen[*v]; ok {
			continue
		}
		seen[*v] = struct{}{}
		values = append(values, *v)
	}
	return values
}

func joinOrUnknown(values []string) string {
	if len(values) == 0 {
		return "unknown"
	}
	return strings.Join(values, ", ")
}
