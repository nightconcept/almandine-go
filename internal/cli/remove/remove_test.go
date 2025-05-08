package remove

// Commenting out unused functions for now, can be deleted if confirmed they are no longer needed.
// // setupRemoveTestEnvironment creates a temporary directory for testing.
// // It can optionally initialize a project.toml, almd-lock.toml, and specified dependency files.
// // It returns the path to the temporary directory.
// func setupRemoveTestEnvironment(t *testing.T, initialProjectTomlContent string, initialLockfileContent string, depFiles map[string]string) (tempDir string) {
// 	t.Helper()
// 	tempDir = t.TempDir()

// 	if initialProjectTomlContent != "" {
// 		projectTomlPath := filepath.Join(tempDir, config.ProjectTomlName)
// 		err := os.WriteFile(projectTomlPath, []byte(initialProjectTomlContent), 0644)
// 		require.NoError(t, err, "Failed to write initial project.toml")
// 	}

// 	if initialLockfileContent != "" {
// 		lockfilePath := filepath.Join(tempDir, config.LockfileName)
// 		err := os.WriteFile(lockfilePath, []byte(initialLockfileContent), 0644)
// 		require.NoError(t, err, "Failed to write initial almd-lock.toml")
// 	}

// 	// S1031: unnecessary nil check around range (staticcheck)
// 	// The range loop over a nil map is a no-op, so the nil check is redundant.
// 	for relPath, content := range depFiles {
// 		absPath := filepath.Join(tempDir, relPath)
// 		err := os.MkdirAll(filepath.Dir(absPath), 0755)
// 		require.NoError(t, err, "Failed to create directory for dependency file: %s", filepath.Dir(absPath))
// 		err = os.WriteFile(absPath, []byte(content), 0644)
// 		require.NoError(t, err, "Failed to write dependency file: %s", absPath)
// 	}

// 	return tempDir
// }

// // runRemoveCommand executes the 'remove' command within a specific working directory.
// // It changes the current working directory to workDir for the duration of the command execution.
// // removeCmdArgs should be the arguments for the 'remove' command itself (e.g., dependency name).
// func runRemoveCommand(t *testing.T, workDir string, removeCmdArgs ...string) error {
// 	t.Helper()

// 	originalWd, err := os.Getwd()
// 	require.NoError(t, err, "Failed to get current working directory")
// 	err = os.Chdir(workDir)
// 	require.NoError(t, err, "Failed to change to working directory: %s", workDir)
// 	defer func() {
// 		require.NoError(t, os.Chdir(originalWd), "Failed to restore original working directory")
// 	}()

// 	app := &cli.App{
// 		Name: "almd-test-remove",
// 		Commands: []*cli.Command{
// 			RemoveCommand(), // Assumes RemoveCommand is defined in the same package or imported
// 		},
// 		Writer:    os.Stderr, // Default, or io.Discard for cleaner test logs
// 		ErrWriter: os.Stderr, // Default, or io.Discard
// 		ExitErrHandler: func(context *cli.Context, err error) {
// 			// Do nothing by default, let the test assertions handle errors from app.Run()
// 		},
// 	}

// 	cliArgs := []string{"almd-test-remove", "remove"}
// 	cliArgs = append(cliArgs, removeCmdArgs...)

// 	return app.Run(cliArgs)
// }
