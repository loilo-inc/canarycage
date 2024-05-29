package upgrade_test

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-github/v62/github"
	"github.com/jarcoal/httpmock"
	"github.com/loilo-inc/canarycage/cli/cage/upgrade"
	"github.com/stretchr/testify/assert"
)

func TestUpgrade(t *testing.T) {
	makeAsset := func(tag, name string) *github.ReleaseAsset {
		return &github.ReleaseAsset{
			Name:               github.String(name),
			BrowserDownloadURL: github.String(fmt.Sprintf("https://localhost/%s/%s", tag, name)),
		}
	}
	binaryAssetName := fmt.Sprintf("canarycage_%s_%s.zip", runtime.GOOS, runtime.GOARCH)
	makeReleases := func(tags ...string) []*github.RepositoryRelease {
		var releases []*github.RepositoryRelease
		for _, tag := range tags {
			releases = append(releases, &github.RepositoryRelease{
				TagName:    github.String(tag),
				Prerelease: github.Bool(strings.HasSuffix(tag, "-pre")),
				Assets: []*github.ReleaseAsset{
					makeAsset(tag, "canarycage_"+tag+"_checksums.txt"),
					makeAsset(tag, binaryAssetName)},
			})
		}
		sort.Slice(releases, func(i, j int) bool {
			return strings.Compare(releases[i].GetTagName(), releases[j].GetTagName()) > 0
		})
		return releases
	}
	t.Run("basic", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "https://api.github.com/repos/loilo-inc/canarycage/releases",
			httpmock.NewJsonResponderOrPanic(200, makeReleases("0.1.0", "0.2.0")))
		httpmock.RegisterResponder("GET", "https://localhost/0.2.0/canarycage_0.2.0_checksums.txt",
			httpmock.NewStringResponder(200, "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b  "+binaryAssetName))
		httpmock.RegisterResponder("GET", "https://localhost/0.2.0/"+binaryAssetName,
			httpmock.NewStringResponder(200, "1"))
		tmpDir, err := os.MkdirTemp("", "canarycage")
		assert.NoError(t, err)
		err = os.WriteFile(tmpDir+"/cage", []byte("0.1.0"), 0644)
		assert.NoError(t, err)
		err = upgrade.Upgrade(&upgrade.Input{
			CurrentVersion: "0.1.0",
			TargetPath:     tmpDir + "/cage"})
		assert.NoError(t, err)
		if content, err := os.ReadFile(tmpDir + "/cage"); err != nil {
			t.Fatal(err)
		} else {
			assert.Equal(t, "1", string(content))
		}
	})
	t.Run("no updates", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "https://api.github.com/repos/loilo-inc/canarycage/releases",
			httpmock.NewJsonResponderOrPanic(200, makeReleases("0.1.0")))
		err := upgrade.Upgrade(&upgrade.Input{
			CurrentVersion: "0.1.0"})
		assert.NoError(t, err)
	})
	t.Run("pre-release", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "https://api.github.com/repos/loilo-inc/canarycage/releases",
			httpmock.NewJsonResponderOrPanic(200, makeReleases("0.1.0", "0.2.0-pre")))
		httpmock.RegisterResponder("GET", "https://localhost/0.2.0-pre/canarycage_0.2.0-pre_checksums.txt",
			httpmock.NewStringResponder(200, "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b  "+binaryAssetName))
		httpmock.RegisterResponder("GET", "https://localhost/0.2.0-pre/"+binaryAssetName,
			httpmock.NewStringResponder(200, "1"))
		tmpDir, err := os.MkdirTemp("", "canarycage")
		assert.NoError(t, err)
		err = os.WriteFile(tmpDir+"/cage", []byte("0.1.0"), 0644)
		assert.NoError(t, err)
		err = upgrade.Upgrade(&upgrade.Input{
			CurrentVersion: "0.1.0",
			PreRelease:     true,
			TargetPath:     tmpDir + "/cage"})
		assert.NoError(t, err)
		if content, err := os.ReadFile(tmpDir + "/cage"); err != nil {
			t.Fatal(err)
		} else {
			assert.Equal(t, "1", string(content))
		}
	})
	t.Run("parse checksum error", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "https://api.github.com/repos/loilo-inc/canarycage/releases",
			httpmock.NewJsonResponderOrPanic(200, makeReleases("0.1.0", "0.2.0")))
		httpmock.RegisterResponder("GET", "https://localhost/0.2.0/canarycage_0.2.0_checksums.txt",
			httpmock.NewStringResponder(200, "invalid"))
		err := upgrade.Upgrade(&upgrade.Input{
			CurrentVersion: "0.1.0"})
		assert.EqualError(t, err, "invalid checksum line: invalid")
	})
}
