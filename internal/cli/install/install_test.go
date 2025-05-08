// Title: Almandine CLI Install Command Tests
// Purpose: Contains test cases for the 'install' command of the Almandine CLI.
package install_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	installcmd "github.com/nightconcept/almandine-go/internal/cli/install" // Import the package being tested
	"github.com/nightconcept/almandine-go/internal/core/config"
	"github.com/nightconcept/almandine-go/internal/core/lockfile"
	"github.com/nightconcept/almandine-go/internal/core/project"
	"github.com/nightconcept/almandine-go/internal/core/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func init() {
	// Enable host validation bypass for testing with mock server
	source.SetTestModeBypassHostValidation(true)
}

// startMockHTTPServer starts an httptest.Server that serves specific responses
// for a map of expected paths. Other paths will result in a 404.
// Adapted from add_test.go
func startMockHTTPServer(t *testing.T, pathResponses map[string]struct {
	Body string
	Code int
}) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPathWithQuery := r.URL.Path
		if r.URL.RawQuery != "" {
			requestPathWithQuery += "?" + r.URL.RawQuery
		}

		for path, response := range pathResponses {
			if r.Method == http.MethodGet && (r.URL.Path == path || requestPathWithQuery == path) {
				w.WriteHeader(response.Code)
				_, err := w.Write([]byte(response.Body))
				assert.NoError(t, err, "Mock server failed to write response body for path: %s", path)
				return
			}
		}
		t.Logf("Mock server: unexpected request: Method %s, Path %s, Query %s", r.Method, r.URL.Path, r.URL.RawQuery)
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

// setupInstallTestEnvironment creates a temporary directory for testing and initializes
// project.toml, almd-lock.toml, and optionally mock dependency files.
// Returns the path to the temporary directory.
func setupInstallTestEnvironment(t *testing.T, initialProjectTomlContent string, initialLockfileContent string, mockDepFiles map[string]string) (tempDir string) {
	t.Helper()
	tempDir = t.TempDir()

	if initialProjectTomlContent != "" {
		projectTomlPath := filepath.Join(tempDir, config.ProjectTomlName)
		err := os.WriteFile(projectTomlPath, []byte(initialProjectTomlContent), 0644)
		require.NoError(t, err, "Failed to write initial project.toml")
	}

	if initialLockfileContent != "" {
		lockfilePath := filepath.Join(tempDir, lockfile.LockfileName)
		err := os.WriteFile(lockfilePath, []byte(initialLockfileContent), 0644)
		require.NoError(t, err, "Failed to write initial almd-lock.toml")
	}

	for relPath, content := range mockDepFiles {
		absPath := filepath.Join(tempDir, relPath)
		err := os.MkdirAll(filepath.Dir(absPath), 0755)
		require.NoError(t, err, "Failed to create directory for mock dep file: %s", filepath.Dir(absPath))
		err = os.WriteFile(absPath, []byte(content), 0644)
		require.NoError(t, err, "Failed to write mock dependency file: %s", absPath)
	}

	return tempDir
}

// runInstallCommand executes the 'install' command within a specific working directory.
func runInstallCommand(t *testing.T, workDir string, installCmdArgs ...string) error {
	t.Helper()

	originalWd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")
	err = os.Chdir(workDir)
	require.NoError(t, err, "Failed to change to working directory: %s", workDir)
	defer func() {
		require.NoError(t, os.Chdir(originalWd), "Failed to restore original working directory")
	}()

	app := &cli.App{
		Name: "almd-test-install",
		Commands: []*cli.Command{
			installcmd.NewInstallCommand(),
		},
		Writer:    os.Stderr,
		ErrWriter: os.Stderr,
		ExitErrHandler: func(context *cli.Context, err error) {
			// Do nothing, let test assertions handle errors
		},
	}

	cliArgs := []string{"almd-test-install", "install"}
	cliArgs = append(cliArgs, installCmdArgs...)

	return app.Run(cliArgs)
}

// Helper to read project.toml, adapted from add_test.go
func readProjectToml(t *testing.T, tomlPath string) project.Project {
	t.Helper()
	bytes, err := os.ReadFile(tomlPath)
	require.NoError(t, err, "Failed to read project.toml: %s", tomlPath)

	var projCfg project.Project
	err = toml.Unmarshal(bytes, &projCfg)
	require.NoError(t, err, "Failed to unmarshal project.toml: %s", tomlPath)
	return projCfg
}

// Helper to read almd-lock.toml, adapted from add_test.go
func readAlmdLockToml(t *testing.T, lockPath string) lockfile.Lockfile {
	t.Helper()
	bytes, err := os.ReadFile(lockPath)
	require.NoError(t, err, "Failed to read almd-lock.toml: %s", lockPath)

	var lockCfg lockfile.Lockfile
	err = toml.Unmarshal(bytes, &lockCfg)
	require.NoError(t, err, "Failed to unmarshal almd-lock.toml: %s", lockPath)
	return lockCfg
}

// Task 7.2.1: Test `almd install` - All dependencies, one needs install (commit hash change)
func TestInstallCommand_OneDepNeedsUpdate_CommitHashChange(t *testing.T) {
	depAName := "depA"
	depAPath := "libs/depA.lua"
	depAOriginalContent := "local depA_v1 = true"
	depANewContent := "local depA_v2 = true; print('updated')"

	initialProjectToml := fmt.Sprintf(`
[package]
name = "test-install-project"
version = "0.1.0"

[dependencies.%s]
source = "github:testowner/testrepo/%s@main"
path = "%s"
`, depAName, depAPath, depAPath) // Assuming file name in repo is same as path for simplicity

	initialLockfile := fmt.Sprintf(`
api_version = "1"

[package.%s]
source = "https://raw.githubusercontent.com/testowner/testrepo/commit1_sha_abcdef1234567890/%s"
path = "%s"
hash = "commit:commit1_sha_abcdef1234567890"
`, depAName, depAPath, depAPath)

	mockFiles := map[string]string{
		depAPath: depAOriginalContent,
	}

	tempDir := setupInstallTestEnvironment(t, initialProjectToml, initialLockfile, mockFiles)

	// Mock server setup
	// 1. GitHub API call to resolve 'main' for depA to a valid hex commit SHA
	// 2. Raw content download for depA using that commit SHA
	commit2SHA := "fedcba0987654321abcdef1234567890" // Valid hex SHA
	// Path for GetLatestCommitSHAForFile: /repos/<owner>/<repo>/commits?path=<file_path_in_repo>&sha=<ref>&per_page=1
	// The <file_path_in_repo> is extracted from the canonical source string.
	// Canonical source: "github:testowner/testrepo/libs/depA.lua@main" -> file_path_in_repo: "libs/depA.lua"
	githubAPIPathForDepA := fmt.Sprintf("/repos/testowner/testrepo/commits?path=%s&sha=main&per_page=1", depAPath)
	githubAPIResponseForDepA := fmt.Sprintf(`[{"sha": "%s"}]`, commit2SHA)

	// Raw download URL path for depA with new commit
	// source.go constructs this as: /<owner>/<repo>/<commit_sha>/<file_path_in_repo>
	// when testModeBypassHostValidation is true.
	rawDownloadPathDepA := fmt.Sprintf("/testowner/testrepo/%s/%s", commit2SHA, depAPath)

	pathResps := map[string]struct {
		Body string
		Code int
	}{
		githubAPIPathForDepA: {Body: githubAPIResponseForDepA, Code: http.StatusOK},
		rawDownloadPathDepA:  {Body: depANewContent, Code: http.StatusOK},
	}
	mockServer := startMockHTTPServer(t, pathResps)

	// Override GitHub API base URL to point to mock server
	originalGHAPIBaseURL := source.GithubAPIBaseURL
	source.GithubAPIBaseURL = mockServer.URL
	defer func() { source.GithubAPIBaseURL = originalGHAPIBaseURL }()

	// Override Raw GitHub Content base URL (used by downloader if source.ParseSourceURL returns a raw URL directly)
	// For test mode, source.ParseSourceURL constructs a full mock server URL if it's a GitHub source.
	// So, this override might not be strictly necessary if GithubAPIBaseURL is correctly used by ParseSourceURL
	// to build the *raw* download URL for the mock server.
	// Let's assume source.ParseSourceURL will use the mockServer.URL correctly for raw downloads in test mode.

	// --- Run Command ---
	err := runInstallCommand(t, tempDir) // No specific args, should install all from project.toml
	require.NoError(t, err, "almd install command failed")

	// --- Assertions ---
	// 1. Verify depA file content is updated
	depAFilePath := filepath.Join(tempDir, depAPath)
	updatedContentBytes, readErr := os.ReadFile(depAFilePath)
	require.NoError(t, readErr, "Failed to read updated depA file: %s", depAFilePath)
	assert.Equal(t, depANewContent, string(updatedContentBytes), "depA file content mismatch after install")

	// 2. Verify almd-lock.toml is updated for depA
	lockFilePath := filepath.Join(tempDir, lockfile.LockfileName)
	updatedLockCfg := readAlmdLockToml(t, lockFilePath)

	require.NotNil(t, updatedLockCfg.Package, "Packages map in almd-lock.toml is nil after install")
	depALockEntry, ok := updatedLockCfg.Package[depAName]
	require.True(t, ok, "depA entry not found in almd-lock.toml after install")

	// Expected raw source URL in lockfile should point to the new commit on the mock server
	expectedLockSourceURL := mockServer.URL + rawDownloadPathDepA
	assert.Equal(t, expectedLockSourceURL, depALockEntry.Source, "depA lockfile source URL mismatch")
	assert.Equal(t, depAPath, depALockEntry.Path, "depA lockfile path mismatch")
	assert.Equal(t, "commit:"+commit2SHA, depALockEntry.Hash, "depA lockfile hash mismatch")

	// 3. Verify project.toml remains unchanged (install doesn't modify project.toml sources)
	projTomlPath := filepath.Join(tempDir, config.ProjectTomlName)
	currentProjCfg := readProjectToml(t, projTomlPath)
	depAProjEntry, ok := currentProjCfg.Dependencies[depAName]
	require.True(t, ok, "depA entry not found in project.toml")
	assert.Equal(t, fmt.Sprintf("github:testowner/testrepo/%s@main", depAPath), depAProjEntry.Source, "project.toml source for depA should not change")
}
