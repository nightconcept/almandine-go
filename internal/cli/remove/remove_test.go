package remove

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/nightconcept/almandine-go/internal/core/config"
	"github.com/nightconcept/almandine-go/internal/core/lockfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestRemoveCommand_SuccessfulRemoval(t *testing.T) {
	// Store original working directory and restore it after test
	originalWd, err := os.Getwd()
	t.Logf("Test starting in directory: %s", originalWd)
	require.NoError(t, err, "Failed to get current working directory")
	defer func() {
		t.Logf("Test cleanup: restoring directory to %s", originalWd)
		require.NoError(t, os.Chdir(originalWd), "Failed to restore original working directory")
	}()

	// Create initial project.toml content
	projectToml := `
[package]
name = "test-project"
version = "0.1.0"

[dependencies]
testlib = { source = "github:user/repo/file.lua@abc123", path = "libs/testlib.lua" }
`

	// Create initial lockfile content
	lockToml := `
api_version = "1"

[package.testlib]
source = "https://raw.githubusercontent.com/user/repo/abc123/file.lua"
path = "libs/testlib.lua"
hash = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
`

	// Create map of dependency files to create
	depFiles := map[string]string{
		"libs/testlib.lua": "-- Test dependency content",
	}

	// Set up test environment
	tempDir := setupRemoveTestEnvironment(t, projectToml, lockToml, depFiles)

	// After setup, verify files exist
	if _, err := os.Stat(filepath.Join(tempDir, "project.toml")); err != nil {
		t.Logf("After setup - project.toml status: %v", err)
	} else {
		t.Log("After setup - project.toml exists")
	}

	// Change to temp directory before running the test
	err = os.Chdir(tempDir)
	t.Logf("Changed to temp directory: %s", tempDir)
	require.NoError(t, err, "Failed to change to temporary directory")

	// Run the remove command
	err = runRemoveCommand(t, tempDir, "testlib")
	require.NoError(t, err)

	// Verify project.toml no longer contains the dependency
	projContent, err := os.ReadFile(filepath.Join(tempDir, "project.toml"))
	require.NoError(t, err)
	assert.NotContains(t, string(projContent), "testlib")

	// Parse project.toml to verify dependency is gone
	var proj struct {
		Dependencies map[string]interface{} `toml:"dependencies"`
	}
	err = toml.Unmarshal(projContent, &proj)
	require.NoError(t, err)
	assert.NotContains(t, proj.Dependencies, "testlib")

	// Verify almd-lock.toml no longer contains the dependency
	lockContent, err := os.ReadFile(filepath.Join(tempDir, "almd-lock.toml"))
	require.NoError(t, err)
	assert.NotContains(t, string(lockContent), "testlib")

	// Parse lock file to verify dependency is gone
	var lock struct {
		Package map[string]interface{} `toml:"package"`
	}
	err = toml.Unmarshal(lockContent, &lock)
	require.NoError(t, err)
	assert.NotContains(t, lock.Package, "testlib")

	// Verify dependency file is deleted
	_, err = os.Stat(filepath.Join(tempDir, "libs", "testlib.lua"))
	assert.True(t, os.IsNotExist(err), "Dependency file should be deleted")

	// Verify libs directory is removed (since it should be empty now)
	_, err = os.Stat(filepath.Join(tempDir, "libs"))
	assert.True(t, os.IsNotExist(err), "Empty libs directory should be removed")
}

func TestRemove_DependencyNotFound(t *testing.T) {
	// Store original working directory
	originalWd, err := os.Getwd()
	t.Logf("Test starting in directory: %s", originalWd)
	require.NoError(t, err, "Failed to get current working directory")
	defer func() {
		t.Logf("Test cleanup: restoring directory to %s", originalWd)
		require.NoError(t, os.Chdir(originalWd), "Failed to restore original working directory")
	}()

	// Create a temp dir for the test
	tempDir := t.TempDir()

	// Create a project.toml without the dependency we'll try to remove
	projectToml := `
[package]
name = "test-project"
version = "0.1.0"

[dependencies]
existing-dep = { source = "github:user/repo/file.lua", path = "libs/existing-dep.lua" }
`
	err = os.WriteFile(filepath.Join(tempDir, "project.toml"), []byte(projectToml), 0644)
	require.NoError(t, err)

	// Create a lockfile that matches project.toml
	lockfileToml := `
api_version = "1"

[package.existing-dep]
source = "https://raw.githubusercontent.com/user/repo/main/file.lua"
path = "libs/existing-dep.lua"
hash = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
`
	err = os.WriteFile(filepath.Join(tempDir, "almd-lock.toml"), []byte(lockfileToml), 0644)
	require.NoError(t, err)

	// Create the existing dependency file to ensure we don't accidentally delete it
	existingDepDir := filepath.Join(tempDir, "libs")
	err = os.MkdirAll(existingDepDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(existingDepDir, "existing-dep.lua"), []byte("-- test content"), 0644)
	require.NoError(t, err)

	// Change to temp directory before running command
	err = os.Chdir(tempDir)
	t.Logf("Changed to temp directory: %s", tempDir)
	require.NoError(t, err, "Failed to change to temporary directory")

	// Execute the remove command
	err = runRemoveCommand(t, tempDir, "non-existent-dep")

	// Verify expectations
	assert.Error(t, err)
	assert.Equal(t, "Error: Dependency 'non-existent-dep' not found in project.toml.", err.Error())
	assert.Equal(t, 1, err.(cli.ExitCoder).ExitCode())

	// Verify project.toml and almd-lock.toml remain unchanged
	currentProjectToml, err := os.ReadFile(filepath.Join(tempDir, "project.toml"))
	require.NoError(t, err)
	assert.Equal(t, string(projectToml), string(currentProjectToml))

	currentLockfileToml, err := os.ReadFile(filepath.Join(tempDir, "almd-lock.toml"))
	require.NoError(t, err)
	assert.Equal(t, string(lockfileToml), string(currentLockfileToml))

	// Verify the existing dependency file was not touched
	_, err = os.Stat(filepath.Join(existingDepDir, "existing-dep.lua"))
	assert.NoError(t, err, "existing dependency file should not be deleted")
}

// Helper Functions
func setupRemoveTestEnvironment(t *testing.T, initialProjectTomlContent string, initialLockfileContent string, depFiles map[string]string) (tempDir string) {
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

	for relPath, content := range depFiles {
		absPath := filepath.Join(tempDir, relPath)
		err := os.MkdirAll(filepath.Dir(absPath), 0755)
		require.NoError(t, err, "Failed to create directory for dependency file: %s", filepath.Dir(absPath))
		err = os.WriteFile(absPath, []byte(content), 0644)
		require.NoError(t, err, "Failed to write dependency file: %s", absPath)
	}

	return tempDir
}

func runRemoveCommand(t *testing.T, workDir string, removeCmdArgs ...string) error {
	t.Helper()

	// Remove working directory handling from here since it's now handled in the test
	app := &cli.App{
		Name: "almd-test-remove",
		Commands: []*cli.Command{
			RemoveCommand(),
		},
		Writer:         os.Stderr,
		ErrWriter:      os.Stderr,
		ExitErrHandler: func(context *cli.Context, err error) {},
	}

	cliArgs := []string{"almd-test-remove", "remove"}
	cliArgs = append(cliArgs, removeCmdArgs...)

	return app.Run(cliArgs)
}
