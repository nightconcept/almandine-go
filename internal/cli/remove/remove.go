package remove

import (
	"fmt"
	"os"

	"github.com/nightconcept/almandine-go/internal/core/config"
	"github.com/nightconcept/almandine-go/internal/core/lockfile"
	"github.com/urfave/cli/v2"
)

// RemoveCommand defines the structure for the 'remove' CLI command.
func RemoveCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Usage:     "Removes a dependency from the project",
		ArgsUsage: "<dependency_name>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return cli.Exit("Error: Missing dependency name argument.", 1)
			}
			dependencyName := c.Args().First()

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

			dep, ok := proj.Dependencies[dependencyName]
			if !ok {
				return cli.Exit(fmt.Sprintf("Error: Dependency '%s' not found in %s.", dependencyName, projectFilePath), 1)
			}

			dependencyPath := dep.Path
			// Remove the dependency from the manifest
			delete(proj.Dependencies, dependencyName)

			// Save the updated manifest
			if err := config.WriteProjectToml(projectFilePath, proj); err != nil {
				return cli.Exit(fmt.Sprintf("Error: Failed to update %s: %v", projectFilePath, err), 1)
			}
			fmt.Printf("Successfully removed dependency '%s' from %s.\n", dependencyName, projectFilePath)

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
			}

			// Update lockfile
			lf, err := lockfile.Load(".") // Load from current directory
			if err != nil {
				// If lockfile loading fails, it's not a critical error that should stop the command,
				// as manifest and file are already handled. Report as warning.
				fmt.Printf("Warning: Failed to load %s: %v. Manifest and file processed.\n", lockfile.LockfileName, err)
			} else {
				if lf.Package != nil {
					if _, ok := lf.Package[dependencyName]; ok {
						delete(lf.Package, dependencyName)
						if err := lockfile.Save(".", lf); err != nil {
							fmt.Printf("Warning: Failed to update %s: %v. Manifest and file processed.\n", lockfile.LockfileName, err)
						} else {
							fmt.Printf("Successfully removed dependency '%s' from %s.\n", dependencyName, lockfile.LockfileName)
						}
					} else {
						fmt.Printf("Info: Dependency '%s' not found in %s. No changes made to lockfile.\n", dependencyName, lockfile.LockfileName)
					}
				} else {
					fmt.Printf("Info: No 'package' section found in %s. No changes made to lockfile.\n", lockfile.LockfileName)
				}
			}

			fmt.Printf("Successfully removed dependency '%s'.\n", dependencyName)
			return nil
		},
	}
}