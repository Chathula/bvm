package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// mockTagsAPI serves paginated tags newest-first, like the GitHub API.
func mockTagsAPI(t *testing.T, tags []string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		perPage, _ := strconv.Atoi(q.Get("per_page"))
		page, _ := strconv.Atoi(q.Get("page"))
		if perPage <= 0 || page <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		start := (page - 1) * perPage
		end := start + perPage
		if end > len(tags) {
			end = len(tags)
		}
		var out []gitTag
		for _, name := range tags[start:end] {
			out = append(out, gitTag{Name: name})
		}
		json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func setReleasesAPIURL(t *testing.T, url string) {
	t.Helper()
	old := releasesAPIURL
	releasesAPIURL = url
	t.Cleanup(func() { releasesAPIURL = old })
}

func TestRemoteVersionsPaginationAndOrder(t *testing.T) {
	var tags []string // newest-first, as served by the API
	for i := 250; i >= 1; i-- {
		tags = append(tags, fmt.Sprintf("bun-v0.%d.0", i))
	}
	setReleasesAPIURL(t, mockTagsAPI(t, tags).URL)

	versions, err := RemoteVersions()
	if err != nil {
		t.Fatalf("RemoteVersions() error = %v", err)
	}
	if len(versions) != 250 {
		t.Fatalf("got %d versions, want 250", len(versions))
	}
	if versions[0] != "v0.1.0" || versions[len(versions)-1] != "v0.250.0" {
		t.Fatalf("unexpected order: first=%s last=%s", versions[0], versions[len(versions)-1])
	}
}

func TestRemoteVersionsStripsBunPrefix(t *testing.T) {
	// Fixture is newest-first, exactly as the GitHub API returns tags.
	setReleasesAPIURL(t, mockTagsAPI(t, []string{"bun-v1.2.4", "bun-v1.2.3"}).URL)

	versions, err := RemoteVersions()
	if err != nil {
		t.Fatalf("RemoteVersions() error = %v", err)
	}
	want := []string{"v1.2.3", "v1.2.4"}
	for i, w := range want {
		if versions[i] != w {
			t.Fatalf("versions[%d] = %q, want %q", i, versions[i], w)
		}
	}
}

func TestNormalizeVersion(t *testing.T) {
	cases := []struct{ in, want string }{
		{"1.2.3", "v1.2.3"},
		{"V1.2.3", "v1.2.3"},
		{"v1.2.3", "v1.2.3"},
		{" 1.10.0 ", "v1.10.0"},
		{"1.2", "v1.2.0"},
	}
	for _, c := range cases {
		got, err := NormalizeVersion(c.in)
		if err != nil {
			t.Fatalf("NormalizeVersion(%q) error = %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("NormalizeVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	for _, in := range []string{"", "abc", "1.2.3.4", "vv1.0.0"} {
		if _, err := NormalizeVersion(in); err == nil {
			t.Errorf("NormalizeVersion(%q) expected error", in)
		}
	}
}

func TestResolveTargetLatest(t *testing.T) {
	setReleasesAPIURL(t, mockTagsAPI(t, []string{"bun-v2.0.0", "bun-v1.0.0"}).URL)

	got, err := ResolveTarget("latest")
	if err != nil {
		t.Fatalf("ResolveTarget(latest) error = %v", err)
	}
	if got != "v2.0.0" {
		t.Fatalf("ResolveTarget(latest) = %q, want v2.0.0", got)
	}
}

func TestResolveTargetExplicit(t *testing.T) {
	setReleasesAPIURL(t, mockTagsAPI(t, []string{"bun-v2.0.0", "bun-v1.0.0"}).URL)

	if got, err := ResolveTarget("1.0.0"); err != nil || got != "v1.0.0" {
		t.Fatalf("ResolveTarget(1.0.0) = (%q, %v)", got, err)
	}
	if err := ValidRemoteVersion("v9.9.9"); err == nil {
		t.Fatal("ValidRemoteVersion(v9.9.9) expected error")
	}
	if _, err := ResolveTarget("9.9.9"); err == nil {
		t.Fatal("ResolveTarget(unknown version) expected error")
	}
}

func TestRemoteVersionsIgnoresNonReleaseTags(t *testing.T) {
	// Newest-first fixture mixing stable releases with channels the Bun
	// repo may publish (canaries, prereleases, junk).
	setReleasesAPIURL(t, mockTagsAPI(t, []string{
		"bun-canary-v9.9.9",
		"bun-v2.0.0-beta.1",
		"bun-v1.4.0",
		"weird-tag",
		"bun-v1.3.14",
	}).URL)

	versions, err := RemoteVersions()
	if err != nil {
		t.Fatalf("RemoteVersions() error = %v", err)
	}
	want := []string{"v1.3.14", "v1.4.0"}
	if len(versions) != len(want) {
		t.Fatalf("got %v, want %v", versions, want)
	}
	for i := range want {
		if versions[i] != want[i] {
			t.Fatalf("versions[%d] = %q, want %q", i, versions[i], want[i])
		}
	}

	// "latest" must resolve to a real stable release.
	got, err := ResolveTarget("latest")
	if err != nil || got != "v1.4.0" {
		t.Fatalf("ResolveTarget(latest) = (%q, %v), want v1.4.0", got, err)
	}
}

func TestAPIGetSendsTokenWhenPresent(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "secret-token")
	t.Setenv("GH_TOKEN", "")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, "[]")
	}))
	defer srv.Close()

	resp, err := APIGet(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("Authorization = %q, want Bearer secret-token", gotAuth)
	}
}

func TestAPIGetAnonymousWithoutToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, "[]")
	}))
	defer srv.Close()

	resp, err := APIGet(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotAuth != "" {
		t.Fatalf("Authorization = %q, want empty", gotAuth)
	}
}
