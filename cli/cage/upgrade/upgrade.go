package upgrade

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/apex/log"
	"github.com/google/go-github/v62/github"
	"github.com/minio/selfupdate"
	"golang.org/x/xerrors"
)

type Input struct {
	CurrentVersion string
	PreRelease     bool
	TargetPath     string
}

func Upgrade(p *Input) error {
	cont := context.Background()
	client := github.NewClient(nil)
	log.Infof("checking for updates...")
	releases, _, err := client.Repositories.ListReleases(cont, "loilo-inc", "canarycage", nil)
	if err != nil {
		return err
	}
	var latestRelease *github.RepositoryRelease
	for _, release := range releases {
		_, err := semver.NewVersion(release.GetTagName())
		if err != nil {
			continue
		}
		if release.GetPrerelease() && p.PreRelease {
			latestRelease = release
			break
		} else if !release.GetPrerelease() {
			latestRelease = release
			break
		}
	}
	if latestRelease == nil {
		return xerrors.Errorf("failed to find latest release")
	}
	log.Infof("latest release: %s", latestRelease.GetTagName())
	currVer, currVerErr := semver.NewVersion(p.CurrentVersion)
	latestVer := semver.MustParse(latestRelease.GetTagName())
	if currVerErr == nil {
		if currVer.Equal(latestVer) || currVer.GreaterThan(latestVer) {
			log.Info("no updates available")
			return nil
		}
	}
	// ignore current version if it's not a valid semver
	log.Infof("upgrading from %s to %s", p.CurrentVersion, latestRelease.GetTagName())
	var version = latestRelease.GetTagName()
	var checksumAsset *github.ReleaseAsset
	var binaryAsset *github.ReleaseAsset
	checksumAssetName := fmt.Sprintf("canarycage_%s_checksums.txt", version)
	binariAssetName := fmt.Sprintf("canarycage_%s_%s.zip", runtime.GOOS, runtime.GOARCH)
	for _, asset := range latestRelease.Assets {
		if asset.GetName() == checksumAssetName {
			checksumAsset = asset
		}
		if asset.GetName() == binariAssetName {
			binaryAsset = asset
		}
	}
	if checksumAsset == nil || binaryAsset == nil {
		return xerrors.Errorf("failed to find assets for version %s", version)
	}
	log.Info("downloading checksums...")
	checksums, err := parseChecksums(checksumAsset.GetBrowserDownloadURL())
	if err != nil {
		return err
	}
	checksum, ok := checksums[binaryAsset.GetName()]
	if !ok {
		return xerrors.Errorf("failed to find checksum for %s", binaryAsset.GetName())
	}
	bin, err := hex.DecodeString(checksum)
	if err != nil {
		return err
	}
	log.Infof("downloading binary %s...", binaryAsset.GetName())
	resp, err := http.DefaultClient.Get(binaryAsset.GetBrowserDownloadURL())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	log.Infof("upgrading to %s", version)
	return selfupdate.Apply(resp.Body, selfupdate.Options{
		Checksum:   bin,
		TargetPath: p.TargetPath,
	})
}

func parseChecksums(url string) (map[string]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	str := string(b)
	sums := make(map[string]string)
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "  ")
		if len(parts) != 2 {
			return nil, xerrors.Errorf("invalid checksum line: %s", line)
		}
		sums[parts[1]] = parts[0]
	}
	return sums, nil
}
