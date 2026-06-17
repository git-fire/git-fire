// Package updatecheck compares the running binary version to the latest GitHub release.
package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const defaultAPIURL = "https://api.github.com/repos/git-fire/git-fire/releases/latest"

// releaseAPIResponse is the subset of the GitHub releases API we need.
type releaseAPIResponse struct {
	TagName string `json:"tag_name"`
}

// LatestReleaseNewerThan fetches the latest release tag and reports whether it is
// semantically newer than current. current should match --version output (e.g. v1.2.3).
// When current cannot be parsed as a release version, newer is always false.
func LatestReleaseNewerThan(ctx context.Context, current string) (latestTag string, newer bool, err error) {
	return latestReleaseNewerThan(ctx, current, defaultAPIURL, nil)
}

func latestReleaseNewerThan(ctx context.Context, current, apiURL string, client *http.Client) (latestTag string, newer bool, err error) {
	cur := canonicalSemverTag(current)
	if cur == "" {
		return "", false, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "git-fire/update-check")

	c := client
	if c == nil {
		c = &http.Client{Timeout: 12 * time.Second}
	}

	resp, err := c.Do(req)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return "", false, readErr
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("github releases API: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var rel releaseAPIResponse
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", false, err
	}
	rel.TagName = strings.TrimSpace(rel.TagName)
	if rel.TagName == "" {
		return "", false, errors.New("empty tag_name in release response")
	}

	latest := canonicalSemverTag(rel.TagName)
	if latest == "" || semver.Compare(latest, cur) <= 0 {
		return rel.TagName, false, nil
	}
	return rel.TagName, true, nil
}

func canonicalSemverTag(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "dev") {
		return ""
	}
	// Strip git-describe suffixes (e.g. v1.2.3-5-gabc, v1.2.3-dirty) for comparison.
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return ""
	}
	candidate := "v" + v
	c := semver.Canonical(candidate)
	if c == "" || !semver.IsValid(c) {
		return ""
	}
	return c
}
