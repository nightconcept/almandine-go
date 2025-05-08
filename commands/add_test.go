package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	addcmd "github.com/nightconcept/almandine-go/internal/cli/add"
	"github.com/nightconcept/almandine-go/internal/core/config"
	"github.com/nightconcept/almandine-go/internal/core/lockfile"
	"github.com/nightconcept/almandine-go/internal/core/project"
	"github.com/nightconcept/almandine-go/internal/core/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// Copied init from internal/cli/add/add_test.go
func init() {
	// Enable host validation bypass for testing with mock server
	source.SetTestModeBypassHostValidation(true)
}

// Copied from internal/cli/add/add_test.go
// setupAddTestEnvironment creates a temporary directory for testing and optionally
// initializes a project.toml file within it.
// It returns the path to the temporary directory.
func setupAddTestEnvironment(t *testing.T, initialProjectTomlContent string) (tempDir string) {
	t.Helper()
	tempDir = t.TempDir()

	if initialProjectTomlContent != "" {
		projectTomlPath := filepath.Join(tempDir, config.ProjectTomlName)
		err := os.WriteFile(projectTomlPath, []byte(initialProjectTomlContent), 0644)
		require.NoError(t, err, "Failed to write initial project.toml")
	}
	return tempDir
}

// Copied from internal/cli/add/add_test.go and adjusted
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
			addcmd.AddCommand,
		},
		Writer:    os.Stderr,
		ErrWriter: os.Stderr,
		ExitErrHandler: func(context *cli.Context, err error) {
			// Prevents os.Exit(1) from urfave/cli from stopping the test run
		},
	}

	cliArgs := []string{"almd-test-add", "add"}
	cliArgs = append(cliArgs, addCmdArgs...)

	return app.Run(cliArgs)
}

// Copied from internal/cli/add/add_test.go
// startMockServer starts an httptest.Server that serves specific responses
// for a map of expected paths.
// Other paths will result in a 404.
func startMockServer(t *testing.T, pathResponses map[string]struct {
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

// Copied from internal/cli/add/add_test.go
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

// func readAlmdLockToml(t *testing.T, lockPath string) project.LockFile { // This function is unused
// 	t.Helper()
// 	bytes, err := os.ReadFile(lockPath)
// 	require.NoError(t, err, "Failed to read almd-lock.toml: %s", lockPath)
//
// 	var lockCfg project.LockFile
// 	err = toml.Unmarshal(bytes, &lockCfg)
// 	require.NoError(t, err, "Failed to unmarshal almd-lock.toml: %s", lockPath)
// 	return lockCfg
// }

func TestAddCommand_CleanupOnFailure_LockfileWriteError(t *testing.T) {
	// This test implements Task 3.4.7

	// --- Test Setup ---
	initialTomlContent := `
[package]
name = "test-cleanup-project"
version = "0.1.0"
`
	tempDir := setupAddTestEnvironment(t, initialTomlContent)
	projectTomlPath := filepath.Join(tempDir, config.ProjectTomlName)

	mockContent := "// Mock library content for cleanup test\\nlocal m = {}\\nfunction m.do() return 'ok' end\\nreturn m"

	// Adjusted mockFileURLPath to conform to the test mode parser's expected GitHub-like format
	mockOwner := "testowner"
	mockRepo := "testrepo"
	mockRef := "main"
	mockFileName := "mocklib.lua"
	mockFileURLPath := fmt.Sprintf("/%s/%s/%s/%s", mockOwner, mockRepo, mockRef, mockFileName)

	mockCommitSHA := "mockcleanupcommitsha1234567890"
	// Mock for GitHub API call to get commit SHA
	mockAPIPathForCommits := fmt.Sprintf("/repos/%s/%s/commits?path=%s&sha=%s&per_page=1", mockOwner, mockRepo, mockFileName, mockRef)
	mockAPIResponseBody := fmt.Sprintf(`[{"sha": "%s"}]`, mockCommitSHA)

	pathResps := map[string]struct {
		Body string
		Code int
	}{
		mockFileURLPath:       {Body: mockContent, Code: http.StatusOK},
		mockAPIPathForCommits: {Body: mockAPIResponseBody, Code: http.StatusOK}, // Added mock for commit API
	}
	mockServer := startMockServer(t, pathResps)

	// IMPORTANT: Override GithubAPIBaseURL to point to our mock server for this test to ensure commit API mock is hit.
	// This was seen in other tests in internal/cli/add/add_test.go
	originalGHAPIBaseURL := source.GithubAPIBaseURL
	source.GithubAPIBaseURL = mockServer.URL
	defer func() { source.GithubAPIBaseURL = originalGHAPIBaseURL }()

	dependencyURL := mockServer.URL + mockFileURLPath

	sourceFileName := mockFileName // Use the defined mockFileName
	expectedDepName := strings.TrimSuffix(sourceFileName, filepath.Ext(sourceFileName))
	defaultLibsDir := "src/lib"
	expectedDownloadedFilePath := filepath.Join(tempDir, defaultLibsDir, sourceFileName)

	lockFilePath := filepath.Join(tempDir, lockfile.LockfileName)
	err := os.Mkdir(lockFilePath, 0755)
	require.NoError(t, err, "Test setup: Failed to create %s as a directory", lockfile.LockfileName)

	// --- Run Command ---
	cmdErr := runAddCommand(t, tempDir, dependencyURL)

	// --- Assertions ---
	require.Error(t, cmdErr, "almd add command should return an error due to lockfile write failure")

	if exitErr, ok := cmdErr.(cli.ExitCoder); ok {
		errorOutput := strings.ToLower(exitErr.Error())
		assert.Contains(t, errorOutput, "lockfile", "Error message should mention 'lockfile'")
		assert.Contains(t, errorOutput, lockfile.LockfileName, "Error message should mention the lockfile name")
		// Check for OS-specific parts of the error like "is a directory" or a TOML encoding error for the lockfile
		// This makes the test more robust if the exact wording changes slightly.
		assert.Condition(t, func() bool {
			return strings.Contains(errorOutput, "is a directory") ||
				strings.Contains(errorOutput, "toml") || // If TOML marshal fails due to directory
				strings.Contains(errorOutput, "permission denied") // Another possible OS error
		}, "Error message details should indicate a write/type issue with the lockfile path: %s", errorOutput)
	} else {
		// Handle cases where the error might not be a cli.ExitCoder but a direct error
		// (though urfave/cli usually wraps errors in ExitCoder for command actions).
		lowerCmdErr := strings.ToLower(cmdErr.Error())
		assert.Contains(t, lowerCmdErr, "lockfile", "Direct error message should mention 'lockfile': %v", cmdErr)
		assert.Fail(t, "Expected command error to be a cli.ExitCoder, got %T: %v", cmdErr, cmdErr)
	}

	_, statErr := os.Stat(expectedDownloadedFilePath)
	assert.True(t, os.IsNotExist(statErr),
		"Downloaded dependency file '%s' should have been removed after lockfile write failure.", expectedDownloadedFilePath)

	projCfg := readProjectToml(t, projectTomlPath)
	depEntry, ok := projCfg.Dependencies[expectedDepName]
	require.True(t, ok, "Dependency '%s' should still be listed in project.toml. Current dependencies: %v", expectedDepName, projCfg.Dependencies)

	// The canonical source will now be a GitHub-like source string
	expectedCanonicalSource := fmt.Sprintf("github:%s/%s/%s@%s", mockOwner, mockRepo, mockFileName, mockRef)
	assert.Equal(t, expectedCanonicalSource, depEntry.Source, "Dependency source in project.toml for '%s' is incorrect", expectedDepName)
	assert.Equal(t, filepath.ToSlash(filepath.Join(defaultLibsDir, sourceFileName)), depEntry.Path,
		"Dependency path in project.toml is incorrect for '%s'", expectedDepName)

	lockFileStat, statLockErr := os.Stat(lockFilePath)
	require.NoError(t, statLockErr, "Should be able to stat the %s path (which is a directory)", lockfile.LockfileName)
	assert.True(t, lockFileStat.IsDir(), "%s should remain a directory", lockfile.LockfileName)

	_, err = os.ReadFile(lockFilePath)
	require.Error(t, err, "Attempting to read %s (which is a dir) as a file should fail", lockfile.LockfileName)
}
