// Package source_test contains tests for the source package.
package source_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nightconcept/almandine-go/internal/core/source"
)

// setupSourceTest sets up a mock server and configures the source package for testing.
// It returns the mock server's URL and a cleanup function.
func setupSourceTest(t *testing.T, handler http.HandlerFunc) (string, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	originalAPIBaseURL := source.GithubAPIBaseURL
	source.GithubAPIBaseURL = server.URL
	source.SetTestModeBypassHostValidation(true)

	cleanup := func() {
		server.Close()
		source.GithubAPIBaseURL = originalAPIBaseURL
		source.SetTestModeBypassHostValidation(false)
	}
	return server.URL, cleanup
}

func TestParseSourceURL_GitHubShorthand(t *testing.T) {
	// t.Parallel() // Removed due to global state modification by some sub-tests
	tests := []struct {
		name          string
		url           string
		mockServerURL string // Only used if testModeBypassHostValidation affects RawURL construction
		want          *source.ParsedSourceInfo
		wantErr       bool
		errContains   string
	}{
		{
			name: "valid shorthand main branch",
			url:  "github:owner/repo/path/to/file.txt@main",
			want: &source.ParsedSourceInfo{
				RawURL:            "https://raw.githubusercontent.com/owner/repo/main/path/to/file.txt",
				CanonicalURL:      "github:owner/repo/path/to/file.txt@main",
				Ref:               "main",
				Provider:          "github",
				Owner:             "owner",
				Repo:              "repo",
				PathInRepo:        "path/to/file.txt",
				SuggestedFilename: "file.txt",
			},
		},
		{
			name: "valid shorthand commit sha",
			url:  "github:owner/repo/file.lua@abcdef1234567890",
			want: &source.ParsedSourceInfo{
				RawURL:            "https://raw.githubusercontent.com/owner/repo/abcdef1234567890/file.lua",
				CanonicalURL:      "github:owner/repo/file.lua@abcdef1234567890",
				Ref:               "abcdef1234567890",
				Provider:          "github",
				Owner:             "owner",
				Repo:              "repo",
				PathInRepo:        "file.lua",
				SuggestedFilename: "file.lua",
			},
		},
		{
			name:        "invalid shorthand missing @ref",
			url:         "github:owner/repo/path/to/file.txt",
			wantErr:     true,
			errContains: "missing @ref",
		},
		{
			name:        "invalid shorthand empty ref",
			url:         "github:owner/repo/path/to/file.txt@",
			wantErr:     true,
			errContains: "ref part is empty after @",
		},
		{
			name:        "invalid shorthand not enough path components",
			url:         "github:owner/repo@main",
			wantErr:     true,
			errContains: "expected format owner/repo/path/to/file",
		},
		{
			name:        "invalid shorthand empty owner",
			url:         "github:/repo/file.txt@main",
			wantErr:     true,
			errContains: "owner, repo, or path/filename cannot be empty",
		},
		{
			name: "valid shorthand with test mode bypass for raw URL",
			url:  "github:testowner/testrepo/test/file.sh@testref",
			// mockServerURL will be dynamically set
			want: &source.ParsedSourceInfo{
				// RawURL will be constructed using mock server URL
				CanonicalURL:      "github:testowner/testrepo/test/file.sh@testref",
				Ref:               "testref",
				Provider:          "github",
				Owner:             "testowner",
				Repo:              "testrepo",
				PathInRepo:        "test/file.sh",
				SuggestedFilename: "file.sh",
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			// t.Parallel() // Removed: sub-tests here need to be serial due to differing global state expectations/setups
			var mockURL string
			var cleanup func()

			if strings.Contains(tt.name, "test mode bypass") {
				mockURL, cleanup = setupSourceTest(t, func(w http.ResponseWriter, r *http.Request) {
					// This handler is not strictly needed for ParseSourceURL for shorthand,
					// but setupSourceTest expects one.
					w.WriteHeader(http.StatusOK)
				})
				defer cleanup()
				// Dynamically set the expected RawURL for this test case
				tt.want.RawURL = fmt.Sprintf("%s/testowner/testrepo/testref/test/file.sh", mockURL)
			}

			got, err := source.ParseSourceURL(tt.url)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseSourceURL_FullGitHubURLs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		url         string
		want        *source.ParsedSourceInfo
		wantErr     bool
		errContains string
	}{
		{
			name: "github.com blob url",
			url:  "https://github.com/owner/repo/blob/main/path/to/script.sh",
			want: &source.ParsedSourceInfo{
				RawURL:            "https://raw.githubusercontent.com/owner/repo/main/path/to/script.sh",
				CanonicalURL:      "github:owner/repo/path/to/script.sh@main",
				Ref:               "main",
				Provider:          "github",
				Owner:             "owner",
				Repo:              "repo",
				PathInRepo:        "path/to/script.sh",
				SuggestedFilename: "script.sh",
			},
		},
		{
			name: "github.com raw url (less common input)",
			url:  "https://github.com/owner/repo/raw/develop/another/file.lua",
			want: &source.ParsedSourceInfo{
				RawURL:            "https://raw.githubusercontent.com/owner/repo/develop/another/file.lua",
				CanonicalURL:      "github:owner/repo/another/file.lua@develop",
				Ref:               "develop",
				Provider:          "github",
				Owner:             "owner",
				Repo:              "repo",
				PathInRepo:        "another/file.lua",
				SuggestedFilename: "file.lua",
			},
		},
		{
			name: "raw.githubusercontent.com url",
			url:  "https://raw.githubusercontent.com/user/project/v1.0/src/main.go",
			want: &source.ParsedSourceInfo{
				RawURL:            "https://raw.githubusercontent.com/user/project/v1.0/src/main.go",
				CanonicalURL:      "github:user/project/src/main.go@v1.0",
				Ref:               "v1.0",
				Provider:          "github",
				Owner:             "user",
				Repo:              "project",
				PathInRepo:        "src/main.go",
				SuggestedFilename: "main.go",
			},
		},
		{
			name: "github.com url with @ref in path",
			url:  "https://github.com/owner/repo/some/file.py@feature-branch",
			want: &source.ParsedSourceInfo{
				RawURL:            "https://raw.githubusercontent.com/owner/repo/feature-branch/some/file.py",
				CanonicalURL:      "github:owner/repo/some/file.py@feature-branch",
				Ref:               "feature-branch",
				Provider:          "github",
				Owner:             "owner",
				Repo:              "repo",
				PathInRepo:        "some/file.py",
				SuggestedFilename: "file.py",
			},
		},
		{
			name:        "github.com tree url (unsupported)",
			url:         "https://github.com/owner/repo/tree/main/path/to/dir",
			wantErr:     true,
			errContains: "direct links to GitHub trees are not supported",
		},
		{
			name:        "invalid raw.githubusercontent.com url short path",
			url:         "https://raw.githubusercontent.com/owner/repo/main", // missing filename
			wantErr:     true,
			errContains: "invalid GitHub raw content URL path",
		},
		{
			name:        "ambiguous github.com url (no ref, no blob/raw)",
			url:         "https://github.com/owner/repo/path/file.txt",
			wantErr:     true,
			errContains: "ambiguous GitHub URL",
		},
		{
			name:        "incomplete github.com blob url",
			url:         "https://github.com/owner/repo/blob/main", // missing filepath
			wantErr:     true,
			errContains: "incomplete GitHub URL path",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := source.ParseSourceURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseSourceURL_WithTestModeBypass_FullMockURL(t *testing.T) {
	// t.Parallel() // Removed due to setupSourceTest modifying global state for all sub-tests

	mockServerURL, cleanup := setupSourceTest(t, func(w http.ResponseWriter, r *http.Request) {
		// This handler is not strictly needed for ParseSourceURL when testModeBypassHostValidation is true,
		// as it directly uses the URL string, but setupSourceTest expects a handler.
		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	tests := []struct {
		name        string
		url         string // This will be the mock server's URL with a path
		want        *source.ParsedSourceInfo
		wantErr     bool
		errContains string
	}{
		{
			name: "mock server full URL resembling raw content path",
			url:  fmt.Sprintf("%s/mockowner/mockrepo/mockref/path/to/mockfile.txt", mockServerURL),
			want: &source.ParsedSourceInfo{
				RawURL:            fmt.Sprintf("%s/mockowner/mockrepo/mockref/path/to/mockfile.txt", mockServerURL),
				CanonicalURL:      "github:mockowner/mockrepo/path/to/mockfile.txt@mockref",
				Ref:               "mockref",
				Provider:          "github", // Simulates GitHub
				Owner:             "mockowner",
				Repo:              "mockrepo",
				PathInRepo:        "path/to/mockfile.txt",
				SuggestedFilename: "mockfile.txt",
			},
		},
		{
			name: "mock server full URL, file at repo root",
			url:  fmt.Sprintf("%s/anotherowner/anotherrepo/anotherref/file.lua", mockServerURL),
			want: &source.ParsedSourceInfo{
				RawURL:            fmt.Sprintf("%s/anotherowner/anotherrepo/anotherref/file.lua", mockServerURL),
				CanonicalURL:      "github:anotherowner/anotherrepo/file.lua@anotherref",
				Ref:               "anotherref",
				Provider:          "github",
				Owner:             "anotherowner",
				Repo:              "anotherrepo",
				PathInRepo:        "file.lua",
				SuggestedFilename: "file.lua",
			},
		},
		{
			name:        "mock server full URL, path too short",
			url:         fmt.Sprintf("%s/owner/repo/ref", mockServerURL), // Missing filename part
			wantErr:     true,
			errContains: "test mode URL path", // "not in expected format" or "seems to point to a directory"
		},
		{
			name:        "mock server full URL, path indicates directory",
			url:         fmt.Sprintf("%s/owner/repo/ref/", mockServerURL), // Trailing slash
			wantErr:     true,
			errContains: "test mode URL path", // "seems to point to a directory"
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := source.ParseSourceURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseSourceURL_NonGitHubURLs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		url         string
		wantErr     bool
		errContains string
	}{
		{
			name:        "unsupported http url",
			url:         "http://example.com/somefile.txt",
			wantErr:     true,
			errContains: "unsupported source URL host: example.com",
		},
		{
			name:        "unsupported gitlab url",
			url:         "https://gitlab.com/user/project/raw/main/file.lua",
			wantErr:     true,
			errContains: "unsupported source URL host: gitlab.com",
		},
		{
			name:        "invalid url format",
			url:         ":not_a_url",
			wantErr:     true,
			errContains: "failed to parse source URL",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := source.ParseSourceURL(tt.url)
			require.Error(t, err)
			if tt.errContains != "" {
				assert.Contains(t, err.Error(), tt.errContains)
			}
		})
	}
}
