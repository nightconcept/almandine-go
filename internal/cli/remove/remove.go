package remove

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nightconcept/almandine-go/internal/core/config"
	"github.com/nightconcept/almandine-go/internal/core/lockfile"
	"github.com/urfave/cli/v2"
)

// isDirEmpty checks if a directory is empty.
// It returns true if the directory has no entries, false otherwise.
// An error is returned if the directory cannot be read.
func isDirEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("failed to read directory %s: %w", path, err)
	}
	return len(entries) == 0, nil
}

// RemoveCommand defines the structure for the 'remove' CLI command.
func RemoveCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Usage:     "Remove a dependency from the project",
		ArgsUsage: "DEPENDENCY",
		Action: func(c *cli.Context) error {
			if !c.Args().Present() {
				return fmt.Errorf("dependency name is required")
			}

			depName := c.Args().First()

			// Check if project.toml exists first
			if _, err := os.Stat("project.toml"); os.IsNotExist(err) {
				return cli.Exit("dependency not found", 1)
			}

			projectFilePath := config.ProjectTomlName
			proj, err := config.LoadProjectToml(projectFilePath)
			if err != nil {
				if os.IsNotExist(err) {
					return cli.Exit(fmt.Sprintf("Error: %s not found in the current directory.", projectFilePath), 1)
				}
				return cli.Exit(fmt.Sprintf("Error: Failed to load %s: %v", projectFilePath, err), 1)
			}

			if proj.Dependencies == nil {
				// This case should ideally not happen if project.toml is valid and loaded,
				// but good for robustness.
				return cli.Exit(fmt.Sprintf("Error: No dependencies found in %s.", projectFilePath), 1)
			}

			dep, ok := proj.Dependencies[depName]
			if !ok {
				return cli.Exit(fmt.Sprintf("Error: Dependency '%s' not found in %s.", depName, projectFilePath), 1)
			}

			dependencyPath := dep.Path
			// Remove the dependency from the manifest
			delete(proj.Dependencies, depName)

			// Save the updated manifest
			if err := config.WriteProjectToml(projectFilePath, proj); err != nil {
				return cli.Exit(fmt.Sprintf("Error: Failed to update %s: %v", projectFilePath, err), 1)
			}
			fmt.Printf("Successfully removed dependency '%s' from %s.\n", depName, projectFilePath)

			// Delete the dependency file
			if err := os.Remove(dependencyPath); err != nil {
				// If the file is already gone, it's not a critical error for the remove operation's main goal (manifest update).
				// However, other errors (like permission issues) should be reported.
				if !os.IsNotExist(err) {
					return cli.Exit(fmt.Sprintf("Error: Failed to delete dependency file '%s': %v. Manifest updated.", dependencyPath, err), 1)
				}
				fmt.Printf("Warning: Dependency file '%s' not found for deletion, but manifest updated.\n", dependencyPath)
			} else {
				fmt.Printf("Successfully deleted dependency file '%s'.\n", dependencyPath)

				// Attempt to clean up empty parent directories
				currentDir := filepath.Dir(dependencyPath)
				projectRootAbs, err := filepath.Abs(".")
				if err != nil {
					fmt.Printf("Warning: Could not determine project root absolute path: %v. Skipping directory cleanup.\n", err)
				} else {
					for {
						absCurrentDir, err := filepath.Abs(currentDir)
						if err != nil {
							fmt.Printf("Warning: Could not get absolute path for '%s': %v. Stopping directory cleanup.\n", currentDir, err)
							break
						}

						// Stop conditions:
						// 1. Reached project root
						// 2. Reached a filesystem root (e.g., "/" or "C:\")
						// 3. Current directory is "." (already at project root relative)
						if absCurrentDir == projectRootAbs || filepath.Dir(absCurrentDir) == absCurrentDir || currentDir == "." {
							break
						}

						empty, err := isDirEmpty(currentDir)
						if err != nil {
							fmt.Printf("Warning: Could not check if directory '%s' is empty: %v. Stopping directory cleanup.\n", currentDir, err)
							break
						}

						if !empty {
							// Directory is not empty, so stop
							break
						}

						// Directory is empty, try to remove it
						fmt.Printf("Info: Directory '%s' is empty, attempting to remove.\n", currentDir)
						if err := os.Remove(currentDir); err != nil {
							fmt.Printf("Warning: Failed to remove empty directory '%s': %v. Stopping directory cleanup.\n", currentDir, err)
							break // Stop if removal fails (e.g., permissions)
						}
						fmt.Printf("Successfully removed empty directory '%s'.\n", currentDir)

						// Move to parent directory
						currentDir = filepath.Dir(currentDir)
					}
				}
			}

			// Update lockfile
			lf, err := lockfile.Load(".") // Load from current directory
			if err != nil {
				// If lockfile loading fails, it's not a critical error that should stop the command,
				// as manifest and file are already handled. Report as warning.
				fmt.Printf("Warning: Failed to load %s: %v. Manifest and file processed.\n", lockfile.LockfileName, err)
			} else {
				if lf.Package != nil {
					if _, ok := lf.Package[depName]; ok {
						delete(lf.Package, depName)
						if err := lockfile.Save(".", lf); err != nil {
							fmt.Printf("Warning: Failed to update %s: %v. Manifest and file processed.\n", lockfile.LockfileName, err)
						} else {
							fmt.Printf("Successfully removed dependency '%s' from %s.\n", depName, lockfile.LockfileName)
						}
					} else {
						fmt.Printf("Info: Dependency '%s' not found in %s. No changes made to lockfile.\n", depName, lockfile.LockfileName)
					}
				} else {
					fmt.Printf("Info: No 'package' section found in %s. No changes made to lockfile.\n", lockfile.LockfileName)
				}
			}

			fmt.Printf("Successfully removed dependency '%s'.\n", depName)
			return nil
		},
	}
}
