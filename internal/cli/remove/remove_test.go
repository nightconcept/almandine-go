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

	// Run the remove command
	err := runRemoveCommand(t, tempDir, "testlib")
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

	originalWd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")
	err = os.Chdir(workDir)
	require.NoError(t, err, "Failed to change to working directory: %s", workDir)
	defer func() {
		require.NoError(t, os.Chdir(originalWd), "Failed to restore original working directory")
	}()

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
