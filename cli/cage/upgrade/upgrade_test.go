package upgrade

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-github/v62/github"
	"github.com/jarcoal/httpmock"
	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
)

func TestUpgrade(t *testing.T) {
	logDI := di.NewDomain(func(b *di.B) {
		b.Set(key.Logger, test.NewLogger())
	})
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
		u := NewUpgrader(logDI, &cageapp.UpgradeCmdInput{
			CurrentVersion: "0.1.0",
			TargetPath:     tmpDir + "/cage",
		})
		err := u.Upgrade(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		assertUpgraded(t, tmpDir, "0.2.0")
	})
	t.Run("no updates", func(t *testing.T) {
		registerResponses(t, "0.1.0")
		tmpDir := setupCurrent(t, "0.1.0")
		u := NewUpgrader(logDI, &cageapp.UpgradeCmdInput{
			CurrentVersion: "0.1.0",
			TargetPath:     tmpDir + "/cage",
		})
		err := u.Upgrade(t.Context())
		assert.NoError(t, err)
		assertUpgraded(t, tmpDir, "0.1.0")
	})
	t.Run("pre-release", func(t *testing.T) {
		registerResponses(t, "0.1.0", "0.2.0", "0.2.1-rc1")
		tmpDir := setupCurrent(t, "0.1.0")
		u := NewUpgrader(logDI, &cageapp.UpgradeCmdInput{
			CurrentVersion: "0.1.0",
			TargetPath:     tmpDir + "/cage",
			PreRelease:     true,
		})
		err := u.Upgrade(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		assertUpgraded(t, tmpDir, "0.2.1-rc1")
	})
	t.Run("should upgrade if current version is not a valid semver", func(t *testing.T) {
		registerResponses(t, "0.1.0", "0.2.0")
		tmpDir := setupCurrent(t, "dev")
		u := NewUpgrader(logDI, &cageapp.UpgradeCmdInput{
			CurrentVersion: "dev",
			TargetPath:     tmpDir + "/cage",
		})
		err := u.Upgrade(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		assertUpgraded(t, tmpDir, "0.2.0")
	})
	t.Run("should error if no assets found", func(t *testing.T) {
		httpmock.Activate(t)
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/loilo-inc/canarycage/releases",
			httpmock.NewJsonResponderOrPanic(200, []*github.RepositoryRelease{
				{
					TagName: github.String("0.2.0"),
					Assets: []*github.ReleaseAsset{
						{Name: github.String("some_other_file.txt")},
					},
				},
			}))
		u := NewUpgrader(logDI, &cageapp.UpgradeCmdInput{
			CurrentVersion: "0.1.0",
		})
		err := u.Upgrade(t.Context())
		assert.EqualError(t, err, "failed to find assets for version 0.2.0")
	})
	t.Run("parse checksum error", func(t *testing.T) {
		httpmock.Activate(t)
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/loilo-inc/canarycage/releases",
			httpmock.NewJsonResponderOrPanic(200, makeReleases("0.1.0", "0.2.0")))
		httpmock.RegisterResponder("GET", "https://localhost/0.2.0/canarycage_0.2.0_checksums.txt",
			httpmock.NewStringResponder(200, "invalid"))
		u := NewUpgrader(logDI, &cageapp.UpgradeCmdInput{
			CurrentVersion: "0.1.0",
		})
		err := u.Upgrade(t.Context())
		assert.EqualError(t, err, "invalid checksum line: invalid")
	})
}

func Test_findAssets(t *testing.T) {
	t.Run("should return assets", func(t *testing.T) {
		release := &github.RepositoryRelease{
			TagName: github.String("0.2.0"),
			Assets: []*github.ReleaseAsset{
				makeAsset("0.2.0", "canarycage_0.2.0_checksums.txt"),
				makeAsset("0.2.0", binaryAssetName),
			},
		}
		checksumAsset, binaryAsset, err := findAssets(release)
		assert.NoError(t, err)
		assert.Equal(t, "canarycage_0.2.0_checksums.txt", checksumAsset.GetName())
		assert.Equal(t, binaryAssetName, binaryAsset.GetName())
	})
	t.Run("should trim v from tag name", func(t *testing.T) {
		release := &github.RepositoryRelease{
			TagName: github.String("v0.2.0"),
			Assets: []*github.ReleaseAsset{
				makeAsset("v0.2.0", "canarycage_0.2.0_checksums.txt"),
				makeAsset("v0.2.0", binaryAssetName),
			},
		}
		checksumAsset, binaryAsset, err := findAssets(release)
		assert.NoError(t, err)
		assert.Equal(t, "canarycage_0.2.0_checksums.txt", checksumAsset.GetName())
		assert.Equal(t, binaryAssetName, binaryAsset.GetName())
	})
	t.Run("should return error if assets not found", func(t *testing.T) {
		release := &github.RepositoryRelease{
			TagName: github.String("0.2.0"),
			Assets: []*github.ReleaseAsset{
				makeAsset("0.2.0", "some_other_file.txt"),
			},
		}
		checksumAsset, binaryAsset, err := findAssets(release)
		assert.Nil(t, checksumAsset)
		assert.Nil(t, binaryAsset)
		assert.EqualError(t, err, "failed to find assets for version 0.2.0")
	})

}

func Test_findLatestRelease(t *testing.T) {
	t.Run("should return latest release", func(t *testing.T) {
		registerResponses(t, "0.1.0", "0.2.0", "0.2.1-rc1")
		release, err := findLatestRelease(t.Context(), false)
		assert.NoError(t, err)
		assert.Equal(t, "0.2.0", release.GetTagName())
	})
	t.Run("should return latest pre-release", func(t *testing.T) {
		registerResponses(t, "0.1.0", "0.2.0", "0.2.1-rc1")
		release, err := findLatestRelease(t.Context(), true)
		assert.NoError(t, err)
		assert.Equal(t, "0.2.1-rc1", release.GetTagName())
	})
	t.Run("should return nil if no releases", func(t *testing.T) {
		registerResponses(t, "not-a-release")
		release, err := findLatestRelease(t.Context(), false)
		assert.Nil(t, release)
		assert.EqualError(t, err, "no releases found")
	})
	t.Run("should return nil if no releases with pre-release == false", func(t *testing.T) {
		registerResponses(t, "0.1.0-rc1")
		release, err := findLatestRelease(t.Context(), false)
		assert.Nil(t, release)
		assert.EqualError(t, err, "no releases found")
	})
	t.Run("should return error if ListReleases failed", func(t *testing.T) {
		httpmock.Activate(t)
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/loilo-inc/canarycage/releases",
			httpmock.NewErrorResponder(fmt.Errorf("error")))
		release, err := findLatestRelease(t.Context(), false)
		assert.Nil(t, release)
		assert.ErrorContains(t, err, "failed to list releases")
	})
}

func makeAsset(tag, name string) *github.ReleaseAsset {
	return &github.ReleaseAsset{
		Name:               github.String(name),
		BrowserDownloadURL: github.String(fmt.Sprintf("https://localhost/%s/%s", tag, name)),
	}
}

var binaryAssetName = fmt.Sprintf("canarycage_%s_%s.zip", runtime.GOOS, runtime.GOARCH)

func makeReleases(tags ...string) []*github.RepositoryRelease {
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
func registerResponses(
	t *testing.T,
	candidates ...string) {
	httpmock.Activate(t)
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
