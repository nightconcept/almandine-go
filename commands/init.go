// Package commands contains the definitions for the almd CLI commands.
package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

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

			// --- Task 1.2: Implement Interactive Prompts for Metadata ---
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

				scriptCommand, err := promptWithDefault(reader, fmt.Sprintf("Command for script '%s'", scriptName), "")
				if err != nil {
					return cli.Exit(fmt.Sprintf("Error reading command for script '%s': %v", scriptName, err), 1)
				}

				// TODO: Consider adding validation for script names and commands (e.g., no empty command?)
				scripts[scriptName] = scriptCommand
			}

			// Add default 'run' script if not provided by the user
			if _, exists := scripts["run"]; !exists {
				scripts["run"] = "go run main.go"
				fmt.Println("Default 'run' script ('go run main.go') added.")
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

			// For now, just print the collected dependencies
			fmt.Println("\n--- Collected Dependencies (Placeholders) ---")
			if len(dependencies) > 0 {
				for name, source := range dependencies {
					fmt.Printf("%s = \"%s\"\n", name, source)
				}
			} else {
				fmt.Println("(No dependencies defined)")
			}
			fmt.Println("-------------------------------------------")

			// TODO: Implement project.toml writing (Task 1.5) using collected data

			return nil
		},
	}
}
