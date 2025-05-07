package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nightconcept/almandine-go/internal/config"
	"github.com/nightconcept/almandine-go/internal/downloader"
	"github.com/nightconcept/almandine-go/internal/hasher"
	"github.com/nightconcept/almandine-go/internal/lockfile"
	"github.com/nightconcept/almandine-go/internal/project"
	"github.com/nightconcept/almandine-go/internal/source"
	"github.com/urfave/cli/v2"
)

// Helper function to get filename without extension
func getFileNameWithoutExtension(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

// Helper function to get file extension
func getFileExtension(fileName string) string {
	return filepath.Ext(fileName)
}

// AddCommand defines the structure for the "add" command.
var AddCommand = &cli.Command{
	Name:      "add",
	Usage:     "Downloads a dependency and adds it to the project",
	ArgsUsage: "<source_url>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "directory",
			Aliases: []string{"d"},
			Usage:   "Specify the target directory for the dependency",
			Value:   "src/lib/",
		},
		&cli.StringFlag{
			Name:    "name",
			Aliases: []string{"n"},
			Usage:   "Specify the name for the dependency (defaults to filename from URL)",
		},
		&cli.BoolFlag{
			Name:  "verbose",
			Usage: "Enable verbose output",
		},
	},
	Action: func(cCtx *cli.Context) error {
		// --- BEGIN DIAGNOSTIC ---
		// fmt.Println("--- Dumping All Flags ---")
		// for _, flag := range cCtx.Command.Flags {
		// 	flagName := flag.Names()[0] // Get the primary name of the flag
		// 	isSet := cCtx.IsSet(flagName)
		// 	value := cCtx.Generic(flagName)
		// 	fmt.Printf("Flag: %s, IsSet: %t, Value: %v (Type: %T)\n", flagName, isSet, value, value)
		// }
		// // Also check specific flags directly as parsed by urfave/cli
		// fmt.Printf("Direct access cCtx.String(\"name\"): '[%s]'\n", cCtx.String("name"))
		// fmt.Printf("Direct access cCtx.String(\"directory\"): '[%s]'\n", cCtx.String("directory"))
		// fmt.Printf("Direct access cCtx.Bool(\"verbose\"): %t\n", cCtx.Bool("verbose"))
		// fmt.Println("--- End Dumping All Flags ---")
		// --- END DIAGNOSTIC ---

		sourceURLInput := ""
		if cCtx.NArg() > 0 {
			sourceURLInput = cCtx.Args().First()
		} else {
			return cli.Exit("Error: <source_url> argument is required.", 1)
		}

		targetDir := cCtx.String("directory")
		customName := cCtx.String("name")
		verbose := cCtx.Bool("verbose")

		if verbose {
			fmt.Printf("Attempting to add dependency:\n")
			fmt.Printf("  Source URL Input: %s\n", sourceURLInput)
			fmt.Printf("  Target Directory: %s\n", targetDir)
			fmt.Printf("  Custom Name (from -n flag): '[%s]'\n", customName)
			fmt.Printf("  Verbose Output: %t\n", verbose)
		}

		// Task 2.2: Parse the source URL
		parsedInfo, err := source.ParseSourceURL(sourceURLInput)
		if err != nil {
			return cli.Exit(fmt.Sprintf("Error parsing source URL: %v", err), 1)
		}

		if verbose {
			fmt.Printf("Parsed Source Info:\n")
			fmt.Printf("  Raw Download URL: %s\n", parsedInfo.RawURL)
			fmt.Printf("  Canonical URL for Manifest: %s\n", parsedInfo.CanonicalURL)
			fmt.Printf("  Extracted Ref (commit/branch/tag): %s\n", parsedInfo.Ref)
			fmt.Printf("  Suggested Filename from URL: %s\n", parsedInfo.SuggestedFilename)
		}

		// Task 2.3: Download the file using the RawURL
		if verbose {
			fmt.Printf("Downloading from %s...\n", parsedInfo.RawURL)
		}
		fileContent, err := downloader.DownloadFile(parsedInfo.RawURL)
		if err != nil {
			return cli.Exit(fmt.Sprintf("Error downloading file: %v", err), 1)
		}
		if verbose {
			fmt.Printf("Downloaded %d bytes successfully.\n", len(fileContent))
		}

		// Task 2.4: Determine target path and save file
		var dependencyNameInManifest string
		var fileNameOnDisk string

		suggestedBaseName := getFileNameWithoutExtension(parsedInfo.SuggestedFilename)
		suggestedExtension := getFileExtension(parsedInfo.SuggestedFilename)

		if customName != "" {
			dependencyNameInManifest = customName
			fileNameOnDisk = customName + suggestedExtension
		} else {
			if suggestedBaseName == "" || suggestedBaseName == "." || suggestedBaseName == "/" {
				return cli.Exit(fmt.Sprintf("Could not infer a valid base filename from URL's suggested filename: %s", parsedInfo.SuggestedFilename), 1)
			}
			dependencyNameInManifest = suggestedBaseName
			fileNameOnDisk = parsedInfo.SuggestedFilename
		}

		if fileNameOnDisk == "" || fileNameOnDisk == "." || fileNameOnDisk == "/" {
			return cli.Exit("Could not determine a valid final filename for saving.", 1)
		}

		if verbose {
			fmt.Printf("Effective filename for saving: %s\n", fileNameOnDisk)
			fmt.Printf("Dependency name in manifest/lockfile: %s\n", dependencyNameInManifest)
		}

		// Construct the full path relative to the current directory (project root)
		projectRoot := "."
		fullPath := filepath.Join(projectRoot, targetDir, fileNameOnDisk)
		relativeDestPath := filepath.ToSlash(filepath.Join(targetDir, fileNameOnDisk))

		if verbose {
			fmt.Printf("Resolved full path for saving: %s\n", fullPath)
			fmt.Printf("Relative destination path for manifest: %s\n", relativeDestPath)
		}

		// Create the target directory if it doesn't exist
		dirToCreate := filepath.Dir(fullPath)
		if verbose {
			fmt.Printf("Ensuring directory exists: %s\n", dirToCreate)
		}
		if err := os.MkdirAll(dirToCreate, 0755); err != nil {
			return cli.Exit(fmt.Sprintf("Error creating directory %s: %v", dirToCreate, err), 1)
		}

		// Save the downloaded content to the file
		if verbose {
			fmt.Printf("Saving file to %s...\n", fullPath)
		}
		if err := os.WriteFile(fullPath, fileContent, 0644); err != nil {
			return cli.Exit(fmt.Sprintf("Error writing file %s: %v", fullPath, err), 1)
		}

		// Task 2.5: Calculate hash of the downloaded content
		fileHashSHA256, err := hasher.CalculateSHA256(fileContent)
		if err != nil {
			return cli.Exit(fmt.Sprintf("Error calculating SHA256 hash: %v", err), 1)
		}
		if verbose {
			fmt.Printf("SHA256 hash of downloaded file: %s\n", fileHashSHA256)
		}

		// Task 2.7: Update project.toml
		if verbose {
			fmt.Println("Updating project.toml...")
		}
		proj, err := config.LoadProjectToml(filepath.Join(projectRoot, config.ProjectTomlName))
		if err != nil {
			if os.IsNotExist(err) {
				if verbose {
					fmt.Printf("%s not found, creating a new one for the dependency.\n", config.ProjectTomlName)
				}
				proj = project.NewProject()
			} else {
				return cli.Exit(fmt.Sprintf("Error loading %s: %v", config.ProjectTomlName, err), 1)
			}
		}

		// Ensure dependencies map is initialized (NewProject should handle this)
		if proj.Dependencies == nil {
			proj.Dependencies = make(map[string]project.Dependency)
		}

		// For project.toml, use the canonical source identifier
		proj.Dependencies[dependencyNameInManifest] = project.Dependency{
			Source: parsedInfo.CanonicalURL,
			Path:   relativeDestPath,
		}

		if err := config.WriteProjectToml(filepath.Join(projectRoot, config.ProjectTomlName), proj); err != nil {
			os.Remove(fullPath)
			return cli.Exit(fmt.Sprintf("Error writing %s: %v. Cleaned up downloaded file %s.", config.ProjectTomlName, err, fullPath), 1)
		}

		if verbose {
			fmt.Printf("Successfully updated %s for dependency '%s'.\n", config.ProjectTomlName, dependencyNameInManifest)
		}

		// Task 2.8: Implement Lockfile Update
		if verbose {
			fmt.Println("Updating almd-lock.toml...")
		}

		lf, err := lockfile.Load(projectRoot)
		if err != nil {
			os.Remove(fullPath)
			return cli.Exit(fmt.Sprintf("Error loading/initializing %s: %v. Downloaded file %s and %s updated, but lockfile update failed.", lockfile.LockfileName, err, fullPath, config.ProjectTomlName), 1)
		}

		// Determine integrity hash: github:<commit_hash> or sha256:<hash>
		var integrityHash string
		if parsedInfo.Provider == "github" && parsedInfo.Ref != "" {
			integrityHash = fmt.Sprintf("github:%s", parsedInfo.Ref)
		} else {
			integrityHash = fileHashSHA256
		}

		// For lockfile, use the exact raw download URL and calculated integrity hash
		lf.AddOrUpdatePackage(dependencyNameInManifest, parsedInfo.RawURL, relativeDestPath, integrityHash)

		if err := lockfile.Save(projectRoot, lf); err != nil {
			os.Remove(fullPath)
			return cli.Exit(fmt.Sprintf("Error saving %s: %v. Downloaded file %s and %s updated, but lockfile save failed. Please check state.", lockfile.LockfileName, err, fullPath, config.ProjectTomlName), 1)
		}

		if verbose {
			fmt.Printf("Successfully updated %s for dependency '%s'.\n", lockfile.LockfileName, dependencyNameInManifest)
		}

		fmt.Printf("Successfully added '%s' from '%s' to '%s', updated %s and %s\n",
			dependencyNameInManifest, sourceURLInput, fullPath, config.ProjectTomlName, lockfile.LockfileName)

		return nil
	},
}
