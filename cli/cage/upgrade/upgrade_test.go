package upgrade_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-github/v62/github"
	"github.com/jarcoal/httpmock"
	"github.com/loilo-inc/canarycage/cli/cage/upgrade"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/logger"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
)

func TestUpgrade(t *testing.T) {
	logDI := di.NewDomain(func(b *di.B) {
		b.Set(key.Logger, logger.DefaultLogger(io.Discard, io.Discard))
	})
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
				Prerelease: github.Bool(regexp.MustCompile(`-rc\d+$`).MatchString(tag)),
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
	registerResponses := func(
		t *testing.T,
		candidates ...string) {

		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)

		respond := func(req *http.Request) (*http.Response, error) {
			f, err := os.Open("testdata" + req.URL.Path)
			if err != nil {
				return nil, err
			}
			return &http.Response{
				StatusCode: 200,
				Body:       f,
			}, nil
		}

		httpmock.RegisterResponder("GET", "https://api.github.com/repos/loilo-inc/canarycage/releases",
			httpmock.NewJsonResponderOrPanic(200, makeReleases(candidates...)))

		for _, release := range candidates {
			httpmock.RegisterResponder("GET",
				fmt.Sprintf("https://localhost/%s/canarycage_%s_checksums.txt", release, release),
				respond,
			)
			httpmock.RegisterResponder("GET",
				fmt.Sprintf("https://localhost/%s/%s", release, binaryAssetName),
				respond,
			)
		}
	}
	setupCurrent := func(t *testing.T, version string) string {
		var err error
		tmpDir, err := os.MkdirTemp("", "canarycage")
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(tmpDir+"/cage", []byte(version+"\n"), 0755)
		if err != nil {
			t.Fatal(err)
		}
		return tmpDir
	}

	assertUpgraded := func(t *testing.T, tmpDir, version string) {
		if content, err := os.ReadFile(tmpDir + "/cage"); err != nil {
			t.Fatal(err)
		} else {
			assert.Equal(t, version+"\n", string(content))
		}
		_, err := os.Stat(tmpDir + "/cage.new")
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(tmpDir + "/cage.old")
		assert.True(t, os.IsNotExist(err))
		stat, _ := os.Stat(tmpDir + "/cage")
		assert.Equal(t, 0755, int(stat.Mode().Perm()))
	}

	t.Run("basic", func(t *testing.T) {
		registerResponses(t, "0.1.0", "0.2.0", "0.2.1-rc1")
		tmpDir := setupCurrent(t, "0.1.0")
		u := upgrade.NewUpgrader(logDI, "0.1.0")
		err := u.Upgrade(&upgrade.Input{
			TargetPath: tmpDir + "/cage"})
		if err != nil {
			t.Fatal(err)
		}
		assertUpgraded(t, tmpDir, "0.2.0")
	})
	t.Run("no updates", func(t *testing.T) {
		registerResponses(t, "0.1.0")
		tmpDir := setupCurrent(t, "0.1.0")
		u := upgrade.NewUpgrader(logDI, "0.1.0")
		err := u.Upgrade(&upgrade.Input{
			TargetPath: tmpDir + "/cage",
		})
		assert.NoError(t, err)
		assertUpgraded(t, tmpDir, "0.1.0")
	})
	t.Run("pre-release", func(t *testing.T) {
		registerResponses(t, "0.1.0", "0.2.0", "0.2.1-rc1")
		tmpDir := setupCurrent(t, "0.1.0")
		u := upgrade.NewUpgrader(logDI, "0.1.0")
		err := u.Upgrade(&upgrade.Input{
			PreRelease: true,
			TargetPath: tmpDir + "/cage"})
		if err != nil {
			t.Fatal(err)
		}
		assertUpgraded(t, tmpDir, "0.2.1-rc1")
	})
	t.Run("should upgrade if current version is not a valid semver", func(t *testing.T) {
		registerResponses(t, "0.1.0", "0.2.0")
		tmpDir := setupCurrent(t, "dev")
		u := upgrade.NewUpgrader(logDI, "dev")
		err := u.Upgrade(&upgrade.Input{
			TargetPath: tmpDir + "/cage"})
		if err != nil {
			t.Fatal(err)
		}
		assertUpgraded(t, tmpDir, "0.2.0")
	})
	t.Run("parse checksum error", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "https://api.github.com/repos/loilo-inc/canarycage/releases",
			httpmock.NewJsonResponderOrPanic(200, makeReleases("0.1.0", "0.2.0")))
		httpmock.RegisterResponder("GET", "https://localhost/0.2.0/canarycage_0.2.0_checksums.txt",
			httpmock.NewStringResponder(200, "invalid"))
		u := upgrade.NewUpgrader(logDI, "0.1.0")
		err := u.Upgrade(&upgrade.Input{})
		assert.EqualError(t, err, "invalid checksum line: invalid")
	})
	t.Run("FindLatestRelease", func(t *testing.T) {
		t.Run("should return latest release", func(t *testing.T) {
			registerResponses(t, "0.1.0", "0.2.0", "0.2.1-rc1")
			u := upgrade.ExportedUpgrader{}
			release, err := u.FindLatestRelease(false)
			assert.NoError(t, err)
			assert.Equal(t, "0.2.0", release.GetTagName())
		})
		t.Run("should return latest pre-release", func(t *testing.T) {
			registerResponses(t, "0.1.0", "0.2.0", "0.2.1-rc1")
			u := upgrade.ExportedUpgrader{}
			release, err := u.FindLatestRelease(true)
			assert.NoError(t, err)
			assert.Equal(t, "0.2.1-rc1", release.GetTagName())
		})
		t.Run("should return nil if no releases", func(t *testing.T) {
			registerResponses(t, "not-a-release")
			u := &upgrade.ExportedUpgrader{}
			release, err := u.FindLatestRelease(false)
			assert.Nil(t, release)
			assert.EqualError(t, err, "no releases found")
		})
		t.Run("should return nil if no releases with pre-release", func(t *testing.T) {
			registerResponses(t, "0.1.0-rc1")
			u := &upgrade.ExportedUpgrader{}
			release, err := u.FindLatestRelease(false)
			assert.Nil(t, release)
			assert.EqualError(t, err, "no releases found")
		})
		t.Run("should return error if ListReleases failed", func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			httpmock.RegisterResponder("GET", "https://api.github.com/repos/loilo-inc/canarycage/releases",
				httpmock.NewErrorResponder(fmt.Errorf("error")))
			u := &upgrade.ExportedUpgrader{}
			release, err := u.FindLatestRelease(false)
			assert.Nil(t, release)
			assert.ErrorContains(t, err, "failed to list releases")
		})
	})
}
