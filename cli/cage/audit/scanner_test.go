package audit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	smithy "github.com/aws/smithy-go"
	"github.com/loilo-inc/canarycage/v5/mocks/mock_awsiface"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type stubEcrTool struct {
	imageID  *ecrtypes.ImageIdentifier
	findings *ecrtypes.ImageScanFindings
	errID    error
	errScan  error
}

func (s *stubEcrTool) GetActualImageIdentifier(ctx context.Context, info *ImageInfo) (*ecrtypes.ImageIdentifier, error) {
	if s.errID != nil {
		return nil, s.errID
	}
	return s.imageID, nil
}

func (s *stubEcrTool) GetImageScanFindings(ctx context.Context, info *ImageInfo, imageID *ecrtypes.ImageIdentifier) (*ecrtypes.ImageScanFindings, error) {
	if s.errScan != nil {
		return nil, s.errScan
	}
	return s.findings, nil
}

func TestScanner_Scan(t *testing.T) {
	ctx := context.Background()

	t.Run("returns scan results for each image", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockEcs := mock_awsiface.NewMockEcsClient(ctrl)
		mockEcr := mock_awsiface.NewMockEcrClient(ctrl)
		scanner := NewScanner(mockEcs, mockEcr)

		mockEcs.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(&ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{{TaskDefinition: aws.String("td:1")}},
			}, nil)

		mockEcs.EXPECT().DescribeTaskDefinition(ctx, gomock.AssignableToTypeOf(&ecs.DescribeTaskDefinitionInput{})).
			Return(&ecs.DescribeTaskDefinitionOutput{
				TaskDefinition: &ecstypes.TaskDefinition{
					ContainerDefinitions: []ecstypes.ContainerDefinition{
						{
							Name:  aws.String("app"),
							Image: aws.String("123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo:1.2.3"),
						},
						{
							Name:  aws.String("sidecar"),
							Image: aws.String("nginx:latest"),
						},
					},
				},
			}, nil)

		manifestJSON, _ := json.Marshal(dockerSchema{
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		})
		manifestStr := string(manifestJSON)

		mockEcr.EXPECT().BatchGetImage(ctx, gomock.AssignableToTypeOf(&ecr.BatchGetImageInput{})).
			DoAndReturn(func(ctx context.Context, input *ecr.BatchGetImageInput, opts ...func(*ecr.Options)) (*ecr.BatchGetImageOutput, error) {
				assert.Equal(t, "my-repo", *input.RepositoryName)
				assert.Equal(t, "1.2.3", *input.ImageIds[0].ImageTag)
				return &ecr.BatchGetImageOutput{
					Images: []ecrtypes.Image{{ImageManifest: &manifestStr}},
				}, nil
			})

		mockEcr.EXPECT().DescribeImageScanFindings(ctx, gomock.AssignableToTypeOf(&ecr.DescribeImageScanFindingsInput{})).
			DoAndReturn(func(ctx context.Context, input *ecr.DescribeImageScanFindingsInput, opts ...func(*ecr.Options)) (*ecr.DescribeImageScanFindingsOutput, error) {
				assert.Equal(t, "my-repo", *input.RepositoryName)
				assert.Equal(t, "1.2.3", *input.ImageId.ImageTag)
				return &ecr.DescribeImageScanFindingsOutput{
					ImageScanFindings: &ecrtypes.ImageScanFindings{},
				}, nil
			})

		results, err := scanner.Scan(ctx, "cluster-a", "service-a")

		assert.NoError(t, err)
		if assert.Len(t, results, 2) {
			assert.Equal(t, "app", results[0].ImageInfo.ContainerName)
			assert.NoError(t, results[0].Err)
			assert.Equal(t, "sidecar", results[1].ImageInfo.ContainerName)
			assert.EqualError(t, results[1].Err, "non-ECR image")
		}
	})

	t.Run("ecs error returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockEcs := mock_awsiface.NewMockEcsClient(ctrl)
		mockEcr := mock_awsiface.NewMockEcrClient(ctrl)
		scanner := NewScanner(mockEcs, mockEcr)

		mockEcs.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(nil, errors.New("ecs error"))

		results, err := scanner.Scan(ctx, "cluster-a", "service-a")

		assert.EqualError(t, err, "ecs error")
		assert.Nil(t, results)
	})
}

func TestScanImage(t *testing.T) {
	ctx := context.Background()
	imageInfo := ImageInfo{
		ContainerName: "test-container",
		Repository:    "test-repo",
		Tag:           "test-tag",
	}

	t.Run("returns scan result with CVEs when scan succeeds", func(t *testing.T) {
		imageID := &ecrtypes.ImageIdentifier{
			ImageTag: aws.String("test-tag"),
		}
		findings := &ecrtypes.ImageScanFindings{
			Findings: []ecrtypes.ImageScanFinding{
				{
					Name:        aws.String("CVE-2021-1234"),
					Severity:    ecrtypes.FindingSeverityHigh,
					Description: aws.String("Test vulnerability"),
				},
			},
		}
		stub := &stubEcrTool{
			imageID:  imageID,
			findings: findings,
		}

		result := scanImage(ctx, stub, imageInfo)

		assert.NoError(t, result.Err)
		assert.Equal(t, imageInfo, result.ImageInfo)
		assert.Len(t, result.Cves, 1)
	})

	t.Run("returns error when GetActualImageIdentifier fails", func(t *testing.T) {
		expectedErr := errors.New("image identifier error")
		stub := &stubEcrTool{
			errID: expectedErr,
		}

		result := scanImage(ctx, stub, imageInfo)

		assert.Equal(t, expectedErr, result.Err)
		assert.Equal(t, imageInfo, result.ImageInfo)
		assert.Nil(t, result.Cves)
	})

	t.Run("returns parsed error when GetImageScanFindings fails", func(t *testing.T) {
		imageID := &ecrtypes.ImageIdentifier{
			ImageTag: aws.String("test-tag"),
		}
		scanErr := errors.New("scan error")
		stub := &stubEcrTool{
			imageID: imageID,
			errScan: scanErr,
		}

		result := scanImage(ctx, stub, imageInfo)

		assert.Equal(t, scanErr, result.Err)
		assert.Equal(t, imageInfo, result.ImageInfo)
		assert.Nil(t, result.Cves)
	})

	t.Run("returns ErrScanNotFound when scan not found", func(t *testing.T) {
		imageID := &ecrtypes.ImageIdentifier{
			ImageTag: aws.String("test-tag"),
		}
		scanErr := &smithy.GenericAPIError{
			Code:    "ScanNotFoundException",
			Message: "Scan not found",
		}
		stub := &stubEcrTool{
			imageID: imageID,
			errScan: scanErr,
		}

		result := scanImage(ctx, stub, imageInfo)

		assert.Equal(t, ErrScanNotFound, result.Err)
		assert.Equal(t, imageInfo, result.ImageInfo)
		assert.Nil(t, result.Cves)
	})

	t.Run("returns empty CVE list when no findings", func(t *testing.T) {
		imageID := &ecrtypes.ImageIdentifier{
			ImageTag: aws.String("test-tag"),
		}
		findings := &ecrtypes.ImageScanFindings{
			Findings: []ecrtypes.ImageScanFinding{},
		}
		stub := &stubEcrTool{
			imageID:  imageID,
			findings: findings,
		}

		result := scanImage(ctx, stub, imageInfo)

		assert.NoError(t, result.Err)
		assert.Equal(t, imageInfo, result.ImageInfo)
		assert.Empty(t, result.Cves)
	})
}
func TestScanFindingsToCVEs(t *testing.T) {
	t.Run("uses basic Findings when present", func(t *testing.T) {
		cves := scanFindingsToCVEs(&ecrtypes.ImageScanFindings{
			Findings: []ecrtypes.ImageScanFinding{
				{Name: aws.String("CVE-BASIC"), Severity: ecrtypes.FindingSeverityHigh},
			},
		})
		assert.Len(t, cves, 1)
		assert.Equal(t, "CVE-BASIC", cves[0].Name)
		assert.Nil(t, cves[0].EnhancedAnalysis)
	})

	t.Run("uses EnhancedFindings when basic Findings is nil", func(t *testing.T) {
		cves := scanFindingsToCVEs(&ecrtypes.ImageScanFindings{
			EnhancedFindings: []ecrtypes.EnhancedImageScanFinding{
				{
					Severity: aws.String(string(ecrtypes.FindingSeverityMedium)),
					PackageVulnerabilityDetails: &ecrtypes.PackageVulnerabilityDetails{
						VulnerabilityId: aws.String("CVE-ENHANCED"),
					},
				},
			},
		})
		assert.Len(t, cves, 1)
		assert.Equal(t, "CVE-ENHANCED", cves[0].Name)
		assert.NotNil(t, cves[0].EnhancedAnalysis)
	})

	t.Run("returns nil when neither Findings nor EnhancedFindings is set", func(t *testing.T) {
		cves := scanFindingsToCVEs(&ecrtypes.ImageScanFindings{})
		assert.Nil(t, cves)
	})
}

func TestEnhancedFindingToCVE(t *testing.T) {
	t.Run("populates all fields from a fully-specified finding", func(t *testing.T) {
		got := enhancedFindingToCVE(ecrtypes.EnhancedImageScanFinding{
			Description:      aws.String("Enhanced vulnerability"),
			Severity:         aws.String(string(ecrtypes.FindingSeverityHigh)),
			Status:           aws.String("ACTIVE"),
			ExploitAvailable: aws.String("NO"),
			FixAvailable:     aws.String("YES"),
			Score:            7.5,
			PackageVulnerabilityDetails: &ecrtypes.PackageVulnerabilityDetails{
				SourceUrl:       aws.String("https://example.com/CVE-2026-1234"),
				VulnerabilityId: aws.String("CVE-2026-1234"),
				VulnerablePackages: []ecrtypes.VulnerablePackage{
					{
						Name:           aws.String("openssl"),
						Version:        aws.String("1.0.0"),
						FixedInVersion: aws.String("1.0.1"),
					},
				},
			},
		})
		assert.Equal(t, CVE{
			Name:           "CVE-2026-1234",
			Description:    "Enhanced vulnerability",
			PackageName:    "openssl",
			PackageVersion: "1.0.0",
			Uri:            "https://example.com/CVE-2026-1234",
			Severity:       ecrtypes.FindingSeverityHigh,
			EnhancedAnalysis: &EnhancedAnalysis{
				Status:           "ACTIVE",
				ExploitAvailable: "NO",
				FixAvailable:     "YES",
				FixedInVersion:   "1.0.1",
				Score:            7.5,
			},
		}, got)
	})

	t.Run("nil PackageVulnerabilityDetails leaves defaults and skips package fields", func(t *testing.T) {
		got := enhancedFindingToCVE(ecrtypes.EnhancedImageScanFinding{
			Severity: aws.String(string(ecrtypes.FindingSeverityHigh)),
			Score:    4.2,
		})
		assert.Equal(t, CVE{
			Name:           "unknown",
			PackageName:    "unknown",
			PackageVersion: "unknown",
			Severity:       ecrtypes.FindingSeverityHigh,
			EnhancedAnalysis: &EnhancedAnalysis{
				Score: 4.2,
			},
		}, got)
	})

	t.Run("falls back to VendorSeverity when top-level severity is missing", func(t *testing.T) {
		got := enhancedFindingToCVE(ecrtypes.EnhancedImageScanFinding{
			PackageVulnerabilityDetails: &ecrtypes.PackageVulnerabilityDetails{
				VulnerabilityId: aws.String("CVE-VENDOR"),
				VendorSeverity:  aws.String(string(ecrtypes.FindingSeverityMedium)),
			},
		})
		assert.Equal(t, ecrtypes.FindingSeverityMedium, got.Severity)
		assert.Equal(t, "CVE-VENDOR", got.Name)
	})

	t.Run("falls back to first ReferenceUrl when SourceUrl is missing", func(t *testing.T) {
		got := enhancedFindingToCVE(ecrtypes.EnhancedImageScanFinding{
			PackageVulnerabilityDetails: &ecrtypes.PackageVulnerabilityDetails{
				ReferenceUrls: []string{"https://ref.example/a", "https://ref.example/b"},
			},
		})
		assert.Equal(t, "https://ref.example/a", got.Uri)
	})

	t.Run("uniquePackageValues skips nil/empty and dedupes", func(t *testing.T) {
		got := enhancedFindingToCVE(ecrtypes.EnhancedImageScanFinding{
			PackageVulnerabilityDetails: &ecrtypes.PackageVulnerabilityDetails{
				VulnerablePackages: []ecrtypes.VulnerablePackage{
					{Name: aws.String("openssl"), Version: aws.String("1.0.0"), FixedInVersion: aws.String("1.0.1")},
					{Name: nil, Version: aws.String(""), FixedInVersion: nil},
					{Name: aws.String("openssl"), Version: aws.String("1.0.0"), FixedInVersion: aws.String("1.0.1")},
					{Name: aws.String("curl"), Version: aws.String("8.0"), FixedInVersion: aws.String("")},
				},
			},
		})
		assert.Equal(t, "openssl, curl", got.PackageName)
		assert.Equal(t, "1.0.0, 8.0", got.PackageVersion)
		assert.Equal(t, "1.0.1", got.EnhancedAnalysis.FixedInVersion)
	})

	t.Run("returns 'unknown' for missing package fields when VulnerablePackages is empty", func(t *testing.T) {
		got := enhancedFindingToCVE(ecrtypes.EnhancedImageScanFinding{
			PackageVulnerabilityDetails: &ecrtypes.PackageVulnerabilityDetails{
				VulnerabilityId: aws.String("CVE-EMPTY"),
			},
		})
		assert.Equal(t, "unknown", got.PackageName)
		assert.Equal(t, "unknown", got.PackageVersion)
		assert.Empty(t, got.EnhancedAnalysis.FixedInVersion)
	})
}

func TestParseError(t *testing.T) {
	t.Run("returns ErrScanNotFound when error code is ScanNotFoundException", func(t *testing.T) {
		scanErr := &smithy.GenericAPIError{
			Code:    "ScanNotFoundException",
			Message: "Scan not found",
		}

		result := parseError(scanErr)

		assert.Equal(t, ErrScanNotFound, result)
	})

	t.Run("returns original error when error code is not ScanNotFoundException", func(t *testing.T) {
		otherErr := &smithy.GenericAPIError{
			Code:    "OtherException",
			Message: "Some other error",
		}

		result := parseError(otherErr)

		assert.Equal(t, otherErr, result)
	})

	t.Run("returns original error when error is not smithy.APIError", func(t *testing.T) {
		standardErr := errors.New("standard error")

		result := parseError(standardErr)

		assert.Equal(t, standardErr, result)
	})

	t.Run("returns original error when error is nil", func(t *testing.T) {
		result := parseError(nil)

		assert.Nil(t, result)
	})
}
