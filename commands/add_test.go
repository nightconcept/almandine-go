package commands_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/nightconcept/almandine-go/commands"
	"github.com/nightconcept/almandine-go/internal/project"
	"github.com/nightconcept/almandine-go/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func init() {
	// Enable host validation bypass for testing with mock server
	source.SetTestModeBypassHostValidation(true)
}

// setupAddTestEnvironment creates a temporary directory for testing and optionally
// initializes a project.toml file within it.
// It returns the path to the temporary directory.
func setupAddTestEnvironment(t *testing.T, initialProjectTomlContent string) (tempDir string) {
	t.Helper()
	tempDir = t.TempDir()

	if initialProjectTomlContent != "" {
		projectTomlPath := filepath.Join(tempDir, "project.toml")
		err := os.WriteFile(projectTomlPath, []byte(initialProjectTomlContent), 0644)
		require.NoError(t, err, "Failed to write initial project.toml")
	}
	return tempDir
}

// runAddCommand executes the 'add' command within a specific working directory.
// It changes the current working directory to workDir for the duration of the command execution.
// addCmdArgs should be the arguments for the 'add' command itself (e.g., URL, flags).
func runAddCommand(t *testing.T, workDir string, addCmdArgs ...string) error {
	t.Helper()

	originalWd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")
	err = os.Chdir(workDir)
	require.NoError(t, err, "Failed to change to working directory: %s", workDir)
	defer func() {
		require.NoError(t, os.Chdir(originalWd), "Failed to restore original working directory")
	}()

	app := &cli.App{
		Name: "almd-test-add",
		Commands: []*cli.Command{
			commands.AddCommand,
		},
		// Suppress help printer during tests unless specifically testing help output
		Writer:    os.Stderr, // Default, or io.Discard for cleaner test logs
		ErrWriter: os.Stderr, // Default, or io.Discard
		ExitErrHandler: func(context *cli.Context, err error) {
			// Do nothing by default, let the test assertions handle errors from app.Run()
			// This prevents os.Exit(1) from urfave/cli from stopping the test run
		},
	}

	cliArgs := []string{"almd-test-add", "add"}
	cliArgs = append(cliArgs, addCmdArgs...)

	return app.Run(cliArgs)
}

// startMockServer starts an httptest.Server that serves specific responses
// for a map of expected paths.
// Other paths will result in a 404.
func startMockServer(t *testing.T, pathResponses map[string]struct {
	Body string
	Code int
}) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Construct path with query for matching, as GitHub API calls include queries.
		requestPathWithQuery := r.URL.Path
		if r.URL.RawQuery != "" {
			requestPathWithQuery += "?" + r.URL.RawQuery
		}

		for path, response := range pathResponses {
			// Allow simple path match or path with query match
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
	t.Cleanup(server.Close) // Ensure server is closed after the test
	return server
}

// readProjectToml reads and unmarshals a project.toml file into a project.Project struct.
func readProjectToml(t *testing.T, tomlPath string) project.Project {
	t.Helper()
	bytes, err := os.ReadFile(tomlPath)
	require.NoError(t, err, "Failed to read project.toml: %s", tomlPath)

	var projCfg project.Project
	err = toml.Unmarshal(bytes, &projCfg)
	require.NoError(t, err, "Failed to unmarshal project.toml: %s", tomlPath)
	return projCfg
}

// readAlmdLockToml reads and unmarshals an almd-lock.toml file into a project.LockFile struct.
func readAlmdLockToml(t *testing.T, lockPath string) project.LockFile {
	t.Helper()
	bytes, err := os.ReadFile(lockPath)
	require.NoError(t, err, "Failed to read almd-lock.toml: %s", lockPath)

	var lockCfg project.LockFile
	err = toml.Unmarshal(bytes, &lockCfg)
	require.NoError(t, err, "Failed to unmarshal almd-lock.toml: %s", lockPath)
	return lockCfg
}

func TestAddCommand_Success_ExplicitNameAndDir(t *testing.T) {
	// --- Test Setup ---
	// This test implements Task 3.4.2
	initialTomlContent := `
[package]
name = "test-project"
version = "0.1.0"
`
	tempDir := setupAddTestEnvironment(t, initialTomlContent)

	mockContent := "// This is a mock lua library content\nlocal lib = {}\nfunction lib.hello() print('hello from lua lib') end\nreturn lib\n"
	// Adjust mockServerPath to fit the expected /<owner>/<repo>/<ref>/<file...> structure
	// and use .lua extension as requested.
	mockFileURLPath := "/testowner/testrepo/v1.0.0/mylib_script.lua"
	mockCommitSHA := "fixedmockshaforexplicittest1234567890"
	// Path for the GetLatestCommitSHAForFile call (matches what GetLatestCommitSHAForFile constructs)
	mockAPIPathForCommits := fmt.Sprintf("/repos/%s/%s/commits?path=%s&sha=%s&per_page=1", "testowner", "testrepo", "mylib_script.lua", "v1.0.0")
	mockAPIResponseBody := fmt.Sprintf(`[{"sha": "%s"}]`, mockCommitSHA)

	pathResps := map[string]struct {
		Body string
		Code int
	}{
		mockFileURLPath:       {Body: mockContent, Code: http.StatusOK},
		mockAPIPathForCommits: {Body: mockAPIResponseBody, Code: http.StatusOK},
	}
	mockServer := startMockServer(t, pathResps)

	// IMPORTANT: Override GithubAPIBaseURL to point to our mock server for this test
	originalGHAPIBaseURL := source.GithubAPIBaseURL
	source.GithubAPIBaseURL = mockServer.URL
	defer func() { source.GithubAPIBaseURL = originalGHAPIBaseURL }()

	dependencyURL := mockServer.URL + mockFileURLPath
	dependencyName := "mylib"        // As per Task 3.4.2
	dependencyDir := "vendor/custom" // As per Task 3.4.2

	// --- Run Command ---
	err := runAddCommand(t, tempDir,
		"-n", dependencyName,
		"-d", dependencyDir,
		dependencyURL,
	)
	require.NoError(t, err, "almd add command failed")

	// --- Assertions ---

	// 1. Verify downloaded file content and path
	// The filename should be the explicit name + extension from source URL path,
	// based on the observed behavior of the `add` command.
	extractedSourceFileExtension := filepath.Ext(mockFileURLPath)           // .lua
	expectedFileNameOnDisk := dependencyName + extractedSourceFileExtension // mylib.lua

	downloadedFilePath := filepath.Join(tempDir, dependencyDir, expectedFileNameOnDisk)
	require.FileExists(t, downloadedFilePath, "Downloaded file does not exist at expected path: %s", downloadedFilePath)

	contentBytes, readErr := os.ReadFile(downloadedFilePath)
	require.NoError(t, readErr, "Failed to read downloaded file: %s", downloadedFilePath)
	assert.Equal(t, mockContent, string(contentBytes), "Downloaded file content mismatch")

	// 2. Verify project.toml was updated correctly
	projectTomlPath := filepath.Join(tempDir, "project.toml")
	projCfg := readProjectToml(t, projectTomlPath)

	require.NotNil(t, projCfg.Dependencies, "Dependencies map in project.toml is nil")
	depEntry, ok := projCfg.Dependencies[dependencyName]
	require.True(t, ok, "Dependency entry not found in project.toml for: %s", dependencyName)

	// Expected canonical source based on the new mockServerPath structure
	// Format: github:<owner>/<repo>/<path_to_file_in_repo>@<ref>
	expectedCanonicalSource := "github:testowner/testrepo/mylib_script.lua@v1.0.0"
	assert.Equal(t, expectedCanonicalSource, depEntry.Source, "Dependency source mismatch in project.toml")
	assert.Equal(t, filepath.ToSlash(filepath.Join(dependencyDir, expectedFileNameOnDisk)), depEntry.Path, "Dependency path mismatch in project.toml")

	// 3. Verify almd-lock.toml was created/updated correctly
	lockFilePath := filepath.Join(tempDir, "almd-lock.toml")
	require.FileExists(t, lockFilePath, "almd-lock.toml was not created")
	lockCfg := readAlmdLockToml(t, lockFilePath)

	assert.Equal(t, "1", lockCfg.APIVersion, "API version in almd-lock.toml mismatch")
	require.NotNil(t, lockCfg.Package, "Packages map in almd-lock.toml is nil")
	lockPkgEntry, ok := lockCfg.Package[dependencyName]
	require.True(t, ok, "Package entry not found in almd-lock.toml for: %s", dependencyName)

	assert.Equal(t, dependencyURL, lockPkgEntry.Source, "Package source mismatch in almd-lock.toml (raw URL)")
	assert.Equal(t, filepath.ToSlash(filepath.Join(dependencyDir, expectedFileNameOnDisk)), lockPkgEntry.Path, "Package path mismatch in almd-lock.toml")

	// Hash should now reflect the commit SHA from the mocked API call.
	expectedHash := "commit:" + mockCommitSHA
	assert.Equal(t, expectedHash, lockPkgEntry.Hash, "Package hash mismatch in almd-lock.toml")
}

func TestAddCommand_Success_InferredName_DefaultDir(t *testing.T) {
	// --- Test Setup ---
	// This test implements Task 3.4.3
	initialTomlContent := `
[package]
name = "test-project-inferred"
version = "0.1.0"
`
	tempDir := setupAddTestEnvironment(t, initialTomlContent)

	mockContent := "// This is a mock lua library for inferred name test\nlocal lib = {}\nreturn lib\n"
	// Adjust mockServerPath to fit the expected /<owner>/<repo>/<ref>/<file...> structure
	// for the test mode URL parser.
	mockFileURLPath_Inferred := "/inferredowner/inferredrepo/mainbranch/test_dependency_file.lua"
	mockCommitSHA_Inferred := "fixedmockshaforinferredtest1234567890"
	mockAPIPathForCommits_Inferred := fmt.Sprintf("/repos/%s/%s/commits?path=%s&sha=%s&per_page=1", "inferredowner", "inferredrepo", "test_dependency_file.lua", "mainbranch")
	mockAPIResponseBody_Inferred := fmt.Sprintf(`[{"sha": "%s"}]`, mockCommitSHA_Inferred)

	pathResps_Inferred := map[string]struct {
		Body string
		Code int
	}{
		mockFileURLPath_Inferred:       {Body: mockContent, Code: http.StatusOK},
		mockAPIPathForCommits_Inferred: {Body: mockAPIResponseBody_Inferred, Code: http.StatusOK},
	}
	mockServer := startMockServer(t, pathResps_Inferred)

	// IMPORTANT: Override GithubAPIBaseURL to point to our mock server for this test
	originalGHAPIBaseURL_Inferred := source.GithubAPIBaseURL
	source.GithubAPIBaseURL = mockServer.URL
	defer func() { source.GithubAPIBaseURL = originalGHAPIBaseURL_Inferred }()

	dependencyURL := mockServer.URL + mockFileURLPath_Inferred

	// --- Run Command ---
	// No -n (name) or -d (directory) flags, testing inference and defaults
	err := runAddCommand(t, tempDir, dependencyURL)
	require.NoError(t, err, "almd add command failed")

	// --- Assertions ---

	// 1. Verify downloaded file content and path (inferred name, default directory)
	sourceFileName := filepath.Base(mockFileURLPath_Inferred)                           // "test_dependency_file.lua"
	inferredDepName := strings.TrimSuffix(sourceFileName, filepath.Ext(sourceFileName)) // "test_dependency_file"

	expectedDiskFileName := sourceFileName // "test_dependency_file.lua"
	// The add command defaults to "src/lib" when -d is not specified.
	expectedDirOnDisk := "src/lib"
	downloadedFilePath := filepath.Join(tempDir, expectedDirOnDisk, expectedDiskFileName)

	require.FileExists(t, downloadedFilePath, "Downloaded file does not exist at expected path: %s", downloadedFilePath)
	contentBytes, readErr := os.ReadFile(downloadedFilePath)
	require.NoError(t, readErr, "Failed to read downloaded file: %s", downloadedFilePath)
	assert.Equal(t, mockContent, string(contentBytes), "Downloaded file content mismatch")

	// 2. Verify project.toml was updated correctly
	projectTomlPath := filepath.Join(tempDir, "project.toml")
	projCfg := readProjectToml(t, projectTomlPath)

	require.NotNil(t, projCfg.Dependencies, "Dependencies map in project.toml is nil")
	depEntry, ok := projCfg.Dependencies[inferredDepName]
	require.True(t, ok, "Dependency entry not found in project.toml for inferred name: %s", inferredDepName)

	// Because the mockServerPath now conforms to the /<owner>/<repo>/<ref>/<file...> structure,
	// the canonical source identifier will be a GitHub-like string.
	expectedCanonicalSource := "github:inferredowner/inferredrepo/test_dependency_file.lua@mainbranch"
	assert.Equal(t, expectedCanonicalSource, depEntry.Source, "Dependency source mismatch in project.toml")

	expectedPathInToml := filepath.ToSlash(filepath.Join(expectedDirOnDisk, expectedDiskFileName))
	assert.Equal(t, expectedPathInToml, depEntry.Path, "Dependency path mismatch in project.toml")

	// 3. Verify almd-lock.toml was created/updated correctly
	lockFilePath := filepath.Join(tempDir, "almd-lock.toml")
	require.FileExists(t, lockFilePath, "almd-lock.toml was not created")
	lockCfg := readAlmdLockToml(t, lockFilePath)

	assert.Equal(t, "1", lockCfg.APIVersion, "API version in almd-lock.toml mismatch")
	require.NotNil(t, lockCfg.Package, "Packages map in almd-lock.toml is nil")
	lockPkgEntry, ok := lockCfg.Package[inferredDepName]
	require.True(t, ok, "Package entry not found in almd-lock.toml for inferred name: %s", inferredDepName)

	assert.Equal(t, dependencyURL, lockPkgEntry.Source, "Package source mismatch in almd-lock.toml (raw URL)")
	assert.Equal(t, expectedPathInToml, lockPkgEntry.Path, "Package path mismatch in almd-lock.toml")

	// Hash should now reflect the commit SHA from the mocked API call.
	expectedHash := "commit:" + mockCommitSHA_Inferred
	assert.Equal(t, expectedHash, lockPkgEntry.Hash, "Package hash mismatch in almd-lock.toml")
}
