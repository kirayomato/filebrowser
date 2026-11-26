package fbhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/filebrowser/filebrowser/v2/settings"
	"github.com/filebrowser/filebrowser/v2/users"
)

func recursiveTestHandler(t *testing.T, userScope string) (http.Handler, string) {
	t.Helper()

	key := []byte("test-signing-key")
	perm := users.Permissions{Create: true, Modify: true, Download: true}
	st := scopedUserStorage(t, userScope, perm, key)

	return handle(resourceGetRecursiveHandler, "/api/resources/recursive", st, &settings.Server{}), signToken(t, perm, key)
}

func TestResourceRecursiveListsTree(t *testing.T) {
	userScope := t.TempDir()
	if err := os.MkdirAll(filepath.Join(userScope, "target", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userScope, "target", "nested", "file.txt"), []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}

	handler, token := recursiveTestHandler(t, userScope)

	req, _ := http.NewRequest(http.MethodGet, "/api/resources/recursive/target", http.NoBody)
	req.Header.Set("X-Auth", token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %q", rec.Code, rec.Body.String())
	}

	var entries []RecursiveEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(entries), entries)
	}
}

// The flat listing asks for a single level below the current directory. A
// trailing slash in the root must not turn that into two levels, otherwise
// every path reached through the breadcrumbs scans one level too deep.
func TestResourceRecursiveDepthIgnoresTrailingSlash(t *testing.T) {
	userScope := t.TempDir()
	if err := os.MkdirAll(filepath.Join(userScope, "target", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userScope, "target", "nested", "file.txt"), []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}

	handler, token := recursiveTestHandler(t, userScope)

	for _, url := range []string{
		"/api/resources/recursive/target?depth=1",
		"/api/resources/recursive/target/?depth=1",
	} {
		req, _ := http.NewRequest(http.MethodGet, url, http.NoBody)
		req.Header.Set("X-Auth", token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d body = %q", url, rec.Code, rec.Body.String())
		}

		var entries []RecursiveEntry
		if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
			t.Fatalf("%s: decode response: %v", url, err)
		}
		if len(entries) != 1 || entries[0].Path != "/target/nested" {
			t.Fatalf("%s: got %+v, want only /target/nested", url, entries)
		}
	}
}

// Walking the tree stats every entry through the scoped filesystem, which is
// expensive on a large destination. When the client has already hung up there
// is nothing to answer, so the walk must stop instead of running to completion
// and then reporting a write failure as a 500.
func TestResourceRecursiveStopsWhenClientDisconnects(t *testing.T) {
	userScope := t.TempDir()
	if err := os.MkdirAll(filepath.Join(userScope, "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userScope, "target", "file.txt"), []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}

	handler, token := recursiveTestHandler(t, userScope)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/api/resources/recursive/target", http.NoBody)
	req.Header.Set("X-Auth", token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if body := rec.Body.String(); body != "" {
		t.Fatalf("wrote a response for a disconnected client: %q", body)
	}
}
