// Package commands contains the definitions for the almd CLI commands.
package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli/v2"
)

// --- Structs for project.toml ---

// PackageMeta holds the metadata for the [package] section
type PackageMeta struct {
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	Description string `toml:"description,omitempty"` // Use omitempty for optional fields
	License     string `toml:"license"`
}

// ProjectConfig represents the overall structure of project.toml
type ProjectConfig struct {
	Package      PackageMeta       `toml:"package"`
	Scripts      map[string]string `toml:"scripts"`
	Dependencies map[string]string `toml:"dependencies,omitempty"` // Use omitempty if no dependencies
}

// Helper function to prompt user and get input with a default value
func promptWithDefault(reader *bufio.Reader, promptText string, defaultValue string) (string, error) {
	// Show default if not empty
	if defaultValue != "" {
		fmt.Printf("%s (default: %s): ", promptText, defaultValue)
	} else {
		fmt.Printf("%s: ", promptText)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		// Return specific error for prompt context
		return "", fmt.Errorf("failed to read input for '%s': %w", promptText, err)
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil // Return default if input is empty
	}
	return input, nil
}

// GetInitCommand returns the definition for the "init" command.
func GetInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize a new Almandine project (creates project.toml)",
		Action: func(c *cli.Context) error {
			fmt.Println("Starting project initialization...") // Adjusted initial message

			reader := bufio.NewReader(os.Stdin)

			var packageName, version, license, description string
			var err error

			// Prompt for package name
			packageName, err = promptWithDefault(reader, "Package name", "my-almandine-project")
			if err != nil {
				// Use cli.Exit for cleaner error handling in actions
				return cli.Exit(err.Error(), 1)
			}

			// Prompt for version
			version, err = promptWithDefault(reader, "Version", "0.1.0")
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			// Prompt for license
			license, err = promptWithDefault(reader, "License", "MIT")
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			// Prompt for description (optional, default is empty)
			// Use the helper, but pass an empty default
			description, err = promptWithDefault(reader, "Description (optional)", "")
			if err != nil {
				return cli.Exit(err.Error(), 1) // Use consistent error exit
			}

			// For now, just print the collected values (Task 1.5 will write the file)
			fmt.Println("\n--- Collected Metadata ---")
			fmt.Printf("Package Name: %s\n", packageName)
			fmt.Printf("Version:      %s\n", version)
			fmt.Printf("License:      %s\n", license)
			fmt.Printf("Description:  %s\n", description)
			fmt.Println("--------------------------")

			// --- Task 1.3: Implement Interactive Prompts for Scripts ---
			scripts := make(map[string]string)
			fmt.Println("\nEnter scripts (leave script name empty to finish):")

			for {
				scriptName, err := promptWithDefault(reader, "Script name", "")
				if err != nil {
					return cli.Exit(fmt.Sprintf("Error reading script name: %v", err), 1)
				}

				if scriptName == "" {
					break // Exit loop if script name is empty
				}

				scriptCmd, err := promptWithDefault(reader, fmt.Sprintf("Command for script '%s'", scriptName), "")
				if err != nil {
					return cli.Exit(fmt.Sprintf("Error reading command for script '%s': %v", scriptName, err), 1)
				}

				// Add or overwrite script
				scripts[scriptName] = scriptCmd
			}

			// Add default 'run' script if not provided by the user
			if _, exists := scripts["run"]; !exists {
				scripts["run"] = "lua src/main.lua"
			}

			// For now, just print the collected scripts
			fmt.Println("\n--- Collected Scripts ---")
			if len(scripts) > 0 {
				for name, cmd := range scripts {
					fmt.Printf("%s = \"%s\"\n", name, cmd)
				}
			} else {
				fmt.Println("(No scripts defined)") // Should not happen due to default 'run'
			}
			fmt.Println("-------------------------")

			// --- Task 1.4: Implement Interactive Prompts for Dependencies (Placeholders) ---
			dependencies := make(map[string]string)
			fmt.Println("\nEnter dependencies (leave dependency name empty to finish):")

			for {
				depName, err := promptWithDefault(reader, "Dependency name", "")
				if err != nil {
					return cli.Exit(fmt.Sprintf("Error reading dependency name: %v", err), 1)
				}

				if depName == "" {
					break // Exit loop if dependency name is empty
				}

				// Simple source/version string for now, as per PRD placeholder
				depSource, err := promptWithDefault(reader, fmt.Sprintf("Source/Version for dependency '%s'", depName), "")
				if err != nil {
					return cli.Exit(fmt.Sprintf("Error reading source for dependency '%s': %v", depName, err), 1)
				}

				dependencies[depName] = depSource
			}

			config := ProjectConfig{
				Package: PackageMeta{
					Name:        packageName,
					Version:     version,
					Description: description, // Will be omitted if empty due to tag
					License:     license,
				},
				Scripts:      scripts,
				Dependencies: dependencies, // Will be omitted if empty due to tag
			}

			// Create or truncate the project.toml file
			file, err := os.Create("project.toml")
			if err != nil {
				return cli.Exit(fmt.Sprintf("Error creating project.toml: %v", err), 1)
			}
			defer file.Close() // Ensure file is closed

			// Encode the config struct to TOML and write to the file
			encoder := toml.NewEncoder(file)
			// Indentation settings for readability (optional but nice)
			encoder.Indent = "\t" // Or use spaces: encoder.Indent = "  "

			if err := encoder.Encode(config); err != nil {
				return cli.Exit(fmt.Sprintf("Error writing to project.toml: %v", err), 1)
			}

			fmt.Println("\nWrote to project.toml")

			return nil
		},
	}
}
