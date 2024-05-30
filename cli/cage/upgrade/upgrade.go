package upgrade

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/apex/log"
	"github.com/google/go-github/v62/github"
	"golang.org/x/xerrors"
)

type Input struct {
	CurrentVersion string
	PreRelease     bool
	TargetPath     string
	Os             string
	Arch           string
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
	checksum, err := parseChecksums(checksumAsset.GetBrowserDownloadURL(), binariAssetName)
	if err != nil {
		return err
	}
	log.Infof("downloading binary %s...", binaryAsset.GetName())
	resp, err := http.DefaultClient.Get(binaryAsset.GetBrowserDownloadURL())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	zipdest, err := os.CreateTemp("", "cage")
	if err != nil {
		return err
	}
	defer zipdest.Close()

	sha := sha256.New()
	if _, err := io.Copy(zipdest, io.TeeReader(resp.Body, sha)); err != nil {
		return err
	}

	actChecksum := sha.Sum(nil)
	if !bytes.Equal(checksum, actChecksum) {
		return xerrors.Errorf("checksum mismatch: expected %x, got %x", checksum, actChecksum)
	}

	ziprd, err := zip.OpenReader(zipdest.Name())
	if err != nil {
		return err
	}
	defer ziprd.Close()
	cageRd, err := ziprd.Open("cage")
	if err != nil {
		return err
	}

	targetPath := p.TargetPath
	if targetPath == "" {
		exec, err := os.Executable()
		if err != nil {
			return err
		}
		targetPath = exec
	}

	oldFilepath := targetPath + ".old"
	if err := os.Rename(targetPath, oldFilepath); err != nil {
		return err
	}
	newFilepath := targetPath + ".new"
	newFile, err := os.OpenFile(newFilepath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer newFile.Close()
	if _, err := io.Copy(newFile, cageRd); err != nil {
		return err
	}
	if err := os.Rename(newFilepath, targetPath); err != nil {
		return err
	}
	if err := os.Remove(oldFilepath); err != nil {
		return err
	}
	log.Infof("upgraded to %s", version)
	return nil
}

func parseChecksums(url string, file string) ([]byte, error) {
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
	checksum, ok := sums[file]
	if !ok {
		return nil, xerrors.Errorf("failed to find checksum for %s", file)
	}
	bin, err := hex.DecodeString(checksum)
	if err != nil {
		return nil, err
	}
	return bin, nil
}
