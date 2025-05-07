package commands_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// startMockServer starts an httptest.Server that serves a specific responseBody
// for a given expectedPath, with a defined statusCode.
// Other paths will result in a 404.
func startMockServer(t *testing.T, expectedPath string, responseBody string, statusCode int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == expectedPath {
			w.WriteHeader(statusCode)
			_, err := w.Write([]byte(responseBody))
			assert.NoError(t, err, "Mock server failed to write response body")
		} else {
			// t.Logf("Mock server: unexpected request: Method %s, Path %s (expected GET %s)", r.Method, r.URL.Path, expectedPath)
			http.NotFound(w, r)
		}
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
	mockServerPath := "/testowner/testrepo/v1.0.0/mylib_script.lua"
	mockServer := startMockServer(t, mockServerPath, mockContent, http.StatusOK)
	// server.Close() is handled by t.Cleanup in startMockServer

	dependencyURL := mockServer.URL + mockServerPath
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
	extractedSourceFileExtension := filepath.Ext(mockServerPath)            // .lua
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

	// Hash should now reflect the extracted git reference from the mockServerPath.
	expectedHash := "github:v1.0.0" // Extracted from "/testowner/testrepo/v1.0.0/mylib_script.lua"
	assert.Equal(t, expectedHash, lockPkgEntry.Hash, "Package hash mismatch in almd-lock.toml")
}
