package remove

import (
	"fmt"
	"os"

	"github.com/nightconcept/almandine-go/internal/core/config"
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
			fmt.Printf("Found dependency '%s' at path: %s\n", dependencyName, dependencyPath)
			fmt.Printf("Manifest loaded. Dependency path for '%s' is '%s'. Next steps: update manifest, delete file, update lockfile.\n", dependencyName, dependencyPath)
			// Further implementation for manifest update, file deletion, and lockfile update will go here.
			return nil
		},
	}
}