package updatecheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestReleaseNewerThan(t *testing.T) {
	t.Parallel()

	jsonOK := `{"tag_name":"v2.0.0"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(jsonOK))
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()

	gotTag, newer, err := latestReleaseNewerThan(ctx, "v1.0.0", srv.URL, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if gotTag != "v2.0.0" || !newer {
		t.Fatalf("got %q newer=%v want v2.0.0 newer=true", gotTag, newer)
	}

	_, newer2, err := latestReleaseNewerThan(ctx, "v2.0.0", srv.URL, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if newer2 {
		t.Fatal("same version should not be newer")
	}

	_, newer3, err := latestReleaseNewerThan(ctx, "dev", srv.URL, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if newer3 {
		t.Fatal("dev should skip comparison")
	}
}

func TestCanonicalSemverTag_gitDescribe(t *testing.T) {
	t.Parallel()
	if got := canonicalSemverTag("v1.2.3-5-gabcdef"); got != "v1.2.3" {
		t.Fatalf("got %q want v1.2.3", got)
	}
	if got := canonicalSemverTag("v1.2.3-dirty"); got != "v1.2.3" {
		t.Fatalf("got %q want v1.2.3", got)
	}
}
