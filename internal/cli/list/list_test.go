package list

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/urfave/cli/v2"

	"github.com/nightconcept/almandine-go/internal/core/config"
	// "github.com/nightconcept/almandine-go/internal/core/project" // Will be needed when other tests are implemented
)

// setupListTestEnvironment creates a temporary directory with project.toml,
// almd-lock.toml, and optional dummy dependency files.
// It returns the path to the temporary directory.
func setupListTestEnvironment(t *testing.T, projectTomlContent string, lockfileContent string, depFiles map[string]string) string {
	t.Helper()
	tempDir := t.TempDir()

	if projectTomlContent != "" {
		projectTomlPath := filepath.Join(tempDir, config.ProjectTomlName)
		err := os.WriteFile(projectTomlPath, []byte(projectTomlContent), 0644)
		require.NoError(t, err, "Failed to write project.toml")
	}

	if lockfileContent != "" {
		lockfilePath := filepath.Join(tempDir, config.LockfileName)
		err := os.WriteFile(lockfilePath, []byte(lockfileContent), 0644)
		require.NoError(t, err, "Failed to write almd-lock.toml")
	}

	for relPath, content := range depFiles {
		absPath := filepath.Join(tempDir, relPath)
		err := os.MkdirAll(filepath.Dir(absPath), 0755)
		require.NoError(t, err, "Failed to create parent directory for dep file")
		err = os.WriteFile(absPath, []byte(content), 0644)
		require.NoError(t, err, "Failed to write dependency file")
	}

	return tempDir
}

// runListCommand executes the list command in the given testDir and captures its stdout.
// It changes the CWD to testDir for the duration of the command execution.
func runListCommand(t *testing.T, testDir string, appArgs ...string) (string, error) {
	t.Helper()

	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	originalWD, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	err = os.Chdir(testDir)
	require.NoError(t, err, "Failed to change working directory to testDir")

	defer func() {
		os.Stdout = originalStdout
		err := os.Chdir(originalWD)
		if err != nil {
			// Log or handle error if changing back directory fails
			fmt.Fprintf(os.Stderr, "Error changing back to original directory: %v\n", err)
		}
		_ = r.Close() // Close read end of pipe
		_ = w.Close() // Close write end of pipe
	}()

	app := &cli.App{
		Commands: []*cli.Command{
			ListCmd, // Assumes ListCmd is defined in the current 'list' package
		},
		// Prevent os.Exit from being called by urfave/cli during tests
		ExitErrHandler: func(context *cli.Context, err error) {
			if err != nil {
				// This handler is primarily to prevent os.Exit.
				// Actual errors from app.Run are caught by cmdErr.
				fmt.Fprintf(os.Stderr, "Note: cli.ExitErrHandler caught error (expected for tests): %v\n", err)
			}
		},
	}
	fullArgs := []string{"almd"}
	fullArgs = append(fullArgs, appArgs...)

	// Disable color output for consistent test results
	t.Setenv("NO_COLOR", "1")

	cmdErr := app.Run(fullArgs)

	err = w.Close() // Close writer to flush buffer before reading
	if err != nil {
		// It's possible the pipe is already closed by app.Run, especially on error.
		// This is often a "write on closed pipe" or similar, which is expected in some error cases.
		// We log it for debugging but don't fail the test solely on this.
		fmt.Fprintf(os.Stderr, "Note: Error closing pipe writer (often expected on app error): %v\n", err)
	}

	var outBuf bytes.Buffer
	_, readErr := outBuf.ReadFrom(r)
	if readErr != nil && readErr.Error() != "io: read/write on closed pipe" {
		// Only assert if it's not the expected pipe closed error
		require.NoError(t, readErr, "Failed to read from stdout pipe")
	}

	return outBuf.String(), cmdErr
}

// TestListCommand_NoDependencies tests the `almd list` command when there are no dependencies.
func TestListCommand_NoDependencies(t *testing.T) {
	t.Run("project.toml exists but is empty", func(t *testing.T) {
		projectTomlContent := `
[package]
name = "test-project"
version = "0.1.0"
description = "A test project."
license = "MIT"
`
		tempDir := setupListTestEnvironment(t, projectTomlContent, "", nil)
		output, err := runListCommand(t, tempDir, "list")

		require.NoError(t, err)
		assert.Contains(t, output, "test-project@0.1.0")
		assert.Contains(t, output, tempDir) // Project path
		assert.Contains(t, output, "dependencies:")
		// Check that there are no dependency lines after "dependencies:"
		// This is a bit fragile, depends on exact output format.
		// A more robust check might parse the output or look for absence of typical dep lines.
		lines := strings.Split(strings.TrimSpace(output), "\n")
		depHeaderIndex := -1
		for i, line := range lines {
			if strings.Contains(line, "dependencies:") {
				depHeaderIndex = i
				break
			}
		}
		require.NotEqual(t, -1, depHeaderIndex, "Dependencies header not found")
		// Ensure no lines follow that look like dependency entries
		// For now, we assume if "No dependencies found" is NOT there, and header is, it's an empty list.
		// The actual list.go prints project info then "dependencies:", then items.
		// If no items, it just prints the header.
		// With the changes in list.go, this case (project.toml exists, package info present, but no [dependencies] table or it's empty)
		// should now print "No dependencies found in project.toml."
		assert.Contains(t, output, "No dependencies found in project.toml.", "Expected 'No dependencies found' message")
	})

	t.Run("project.toml with empty dependencies table", func(t *testing.T) {
		projectTomlContent := `
[package]
name = "test-project-empty-deps"
version = "0.1.0"
description = "A test project."
license = "MIT"

[dependencies]
`
		tempDir := setupListTestEnvironment(t, projectTomlContent, "", nil)
		output, err := runListCommand(t, tempDir, "list")

		require.NoError(t, err)
		assert.Contains(t, output, "test-project-empty-deps@0.1.0")
		assert.Contains(t, output, tempDir)
		assert.Contains(t, output, "dependencies:")
		// The "No dependencies found in project.toml." message is shown by printNoDependenciesMessage
		// which is called if proj.Dependencies is nil or len(proj.Dependencies) == 0
		// This seems to be the expected output from list.go's current logic.
		assert.Contains(t, output, "No dependencies found in project.toml.")
	})

	t.Run("project.toml with no dependencies table", func(t *testing.T) {
		// This is effectively the same as the first sub-test "project.toml exists but is empty"
		// if "empty" means no [dependencies] table.
		// The list command loads the project, and if project.Dependencies is nil, it triggers the "No dependencies" message.
		projectTomlContent := `
[package]
name = "test-project-no-deps-table"
version = "0.1.0"
`
		tempDir := setupListTestEnvironment(t, projectTomlContent, "", nil)
		output, err := runListCommand(t, tempDir, "list")

		require.NoError(t, err)
		assert.Contains(t, output, "test-project-no-deps-table@0.1.0")
		assert.Contains(t, output, tempDir)
		assert.Contains(t, output, "dependencies:") // Header is always printed
		assert.Contains(t, output, "No dependencies found in project.toml.")
	})
}

// TestListCommand_ProjectTomlNotFound tests `almd list` when project.toml is missing.
// This is a separate test as per typical Go test structure.
func TestListCommand_ProjectTomlNotFound(t *testing.T) {
	tempDir := t.TempDir() // Create an empty temp directory

	// Ensure NO_COLOR is set for consistent error message format if colors are used there too
	t.Setenv("NO_COLOR", "1")

	_, err := runListCommand(t, tempDir, "list")

	require.Error(t, err, "Expected an error when project.toml is not found")
	// Error message comes from internal/cli/list/list.go, from loadProjectAndLockfile
	// It should be something like "Error loading project.toml: open project.toml: no such file or directory"
	// The exact error message might be wrapped by urfave/cli.
	// Let's check the output for the core part of the error.
	// The actual error returned by app.Run might be a cli.ExitCoder.
	// The error message from cli.Exit is returned as an error that implements cli.ExitCoder.
	// Its Error() method gives the message.
	// urfave/cli prints this message to os.Stderr, not os.Stdout (which `output` captures).
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Error: project.toml not found. No project configuration loaded.")
}

// Placeholder for future tests from Task 9.2.x
func TestListCommand_SingleDependency(t *testing.T) {
	// TODO: Implement Sub-Task 9.2.2
	t.Skip("TODO: Implement Sub-Task 9.2.2: Test almd list - Single dependency (fully installed and locked)")
}

func TestListCommand_MultipleDependenciesVariedStates(t *testing.T) {
	// TODO: Implement Sub-Task 9.2.3
	t.Skip("TODO: Implement Sub-Task 9.2.3: Test almd list - Multiple dependencies with varied states")
}

func TestListCommand_AliasLs(t *testing.T) {
	// TODO: Implement Sub-Task 9.2.4
	t.Skip("TODO: Implement Sub-Task 9.2.4: Test almd ls (alias) - Verify alias works")
}

// Note: Task 9.2.5 "project.toml not found" is covered by TestListCommand_ProjectTomlNotFound

// Helper to get project details from project.toml for assertions
/*
func getProjectDetails(t *testing.T, projectTomlPath string) (name, version string) {
	t.Helper()
	proj, err := config.LoadProjectToml(filepath.Dir(projectTomlPath))
	require.NoError(t, err, "Failed to load project.toml for details")
	require.NotNil(t, proj.Package, "Project package section is nil")
	return proj.Package.Name, proj.Package.Version
}
*/
