package add

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nightconcept/almandine-go/internal/core/config"
	"github.com/nightconcept/almandine-go/internal/core/downloader"
	"github.com/nightconcept/almandine-go/internal/core/hasher"
	"github.com/nightconcept/almandine-go/internal/core/lockfile"
	"github.com/nightconcept/almandine-go/internal/core/project"
	"github.com/nightconcept/almandine-go/internal/core/source"
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
		sourceURLInput := ""
		if cCtx.NArg() > 0 {
			sourceURLInput = cCtx.Args().Get(0) // .First() is equivalent but .Get(0) is more explicit
		} else {
			return cli.Exit("Error: <source_url> argument is required.", 1)
		}

		targetDir := cCtx.String("directory")
		customName := cCtx.String("name")
		verbose := cCtx.Bool("verbose")

		// --- Temporary debug prints ---
		fmt.Printf("[DEBUG Action] Raw cCtx.String(\"directory\"): %s\n", cCtx.String("directory"))
		fmt.Printf("[DEBUG Action] Raw cCtx.String(\"name\"): %s\n", cCtx.String("name"))
		// --- End temporary debug prints ---

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
			return cli.Exit(fmt.Sprintf("Error parsing source URL '%s': %v", sourceURLInput, err), 1)
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
			return cli.Exit(fmt.Sprintf("Error downloading file from '%s': %v", parsedInfo.RawURL, err), 1)
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
			fileNameOnDisk = customName + suggestedExtension // Ensure extension is preserved
		} else {
			if suggestedBaseName == "" || suggestedBaseName == "." || suggestedBaseName == "/" {
				return cli.Exit(fmt.Sprintf("Error: Could not infer a valid base filename from URL's suggested filename: '%s'. Use -n to specify a name.", parsedInfo.SuggestedFilename), 1)
			}
			dependencyNameInManifest = suggestedBaseName
			fileNameOnDisk = parsedInfo.SuggestedFilename
		}

		if fileNameOnDisk == "" || fileNameOnDisk == "." || fileNameOnDisk == "/" {
			return cli.Exit("Error: Could not determine a valid final filename for saving. Inferred name was empty or invalid.", 1)
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
			return cli.Exit(fmt.Sprintf("Error creating directory '%s': %v", dirToCreate, err), 1)
		}

		// Save the downloaded content to the file
		// This is a critical point: if this succeeds but subsequent steps fail, we should try to clean up this file.
		if verbose {
			fmt.Printf("Saving file to %s...\n", fullPath)
		}
		if err := os.WriteFile(fullPath, fileContent, 0644); err != nil {
			// No file to clean up yet, as it wasn't written.
			return cli.Exit(fmt.Sprintf("Error writing file '%s': %v", fullPath, err), 1)
		}
		// File has been written. From this point on, if an error occurs, we must attempt to clean it up.
		fileWritten := true
		defer func() {
			if err != nil && fileWritten { // If an error occurred and file was written
				if verbose {
					fmt.Printf("Attempting to clean up downloaded file '%s' due to error...\n", fullPath)
				}
				cleanupErr := os.Remove(fullPath)
				if cleanupErr != nil {
					// Log cleanup error but don't override original error.
					// Using cCtx.App.ErrWriter if available, otherwise os.Stderr
					var errWriter io.Writer = os.Stderr // Explicitly type as io.Writer
					if cCtx.App != nil && cCtx.App.ErrWriter != nil {
						errWriter = cCtx.App.ErrWriter
					}
					_, _ = fmt.Fprintf(errWriter, "Warning: Failed to clean up downloaded file '%s': %v\n", fullPath, cleanupErr)
				} else {
					if verbose {
						fmt.Printf("Successfully cleaned up downloaded file '%s'.\n", fullPath)
					}
				}
			}
		}()

		// Task 2.5: Calculate hash of the downloaded content
		fileHashSHA256, hashErr := hasher.CalculateSHA256(fileContent)
		if hashErr != nil {
			err = fmt.Errorf("calculating SHA256 hash: %w", hashErr)
			return cli.Exit(fmt.Sprintf("Error %s. File '%s' was saved but is now being cleaned up.", err, fullPath), 1)
		}
		if verbose {
			fmt.Printf("SHA256 hash of downloaded file: %s\n", fileHashSHA256)
		}

		// Task 2.7: Update project.toml
		if verbose {
			fmt.Println("Updating project.toml...")
		}
		projectTomlPath := filepath.Join(projectRoot, config.ProjectTomlName)
		proj, loadTomlErr := config.LoadProjectToml(projectTomlPath)
		if loadTomlErr != nil {
			if os.IsNotExist(loadTomlErr) {
				// Task 3.4.6: Return an error if project.toml is not found
				err = fmt.Errorf("project.toml not found: %w", loadTomlErr)
				return cli.Exit(fmt.Sprintf("Error: %s. File '%s' was saved but is now being cleaned up.", err, fullPath), 1)
			} else {
				err = fmt.Errorf("loading %s: %w", config.ProjectTomlName, loadTomlErr)
				return cli.Exit(fmt.Sprintf("Error %s. File '%s' was saved but is now being cleaned up.", err, fullPath), 1)
			}
		}

		// Ensure dependencies map is initialized (NewProject should handle this, but defensive check)
		if proj.Dependencies == nil {
			proj.Dependencies = make(map[string]project.Dependency)
		}

		// For project.toml, use the canonical source identifier
		proj.Dependencies[dependencyNameInManifest] = project.Dependency{
			Source: parsedInfo.CanonicalURL,
			Path:   relativeDestPath,
		}

		if writeTomlErr := config.WriteProjectToml(projectTomlPath, proj); writeTomlErr != nil {
			err = fmt.Errorf("writing %s: %w", config.ProjectTomlName, writeTomlErr)
			return cli.Exit(fmt.Sprintf("Error %s. File '%s' was saved but is now being cleaned up. %s may be in an inconsistent state.", err, fullPath, config.ProjectTomlName), 1)
		}

		if verbose {
			fmt.Printf("Successfully updated %s for dependency '%s'.\n", config.ProjectTomlName, dependencyNameInManifest)
		}

		// Task 2.8: Implement Lockfile Update
		if verbose {
			fmt.Println("Updating almd-lock.toml...")
		}

		lf, loadLockErr := lockfile.Load(projectRoot) // Load or initialize if not found
		if loadLockErr != nil {
			// Load now also initializes, so a critical error here means we can't proceed.
			err = fmt.Errorf("loading/initializing %s: %w", lockfile.LockfileName, loadLockErr)
			// project.toml was successfully written. This is a more complex cleanup scenario.
			// For now, as per task, focus on cleaning the downloaded file.
			// User will be warned about potential inconsistency.
			return cli.Exit(fmt.Sprintf("Error %s. File '%s' saved and %s updated, but lockfile operation failed. %s and %s may be inconsistent. Downloaded file '%s' is being cleaned up.", err, fullPath, config.ProjectTomlName, config.ProjectTomlName, lockfile.LockfileName, fullPath), 1)
		}

		// Determine integrity hash: commit:<commit_hash> or sha256:<hash>
		var integrityHash string
		isLikelyCommitSHA := func(ref string) bool {
			if len(ref) != 40 { // Standard Git SHA-1 length
				return false
			}
			for _, r := range ref {
				if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
					return false
				}
			}
			return true
		}

		if parsedInfo.Provider == "github" && parsedInfo.Owner != "" && parsedInfo.Repo != "" && parsedInfo.PathInRepo != "" && parsedInfo.Ref != "" && !strings.HasPrefix(parsedInfo.Ref, "error:") {
			if isLikelyCommitSHA(parsedInfo.Ref) {
				if verbose {
					fmt.Printf("Using provided ref '%s' as commit SHA for lockfile hash.\\n", parsedInfo.Ref)
				}
				integrityHash = fmt.Sprintf("commit:%s", parsedInfo.Ref)
			} else {
				// Ref is likely a branch or tag, try to get the specific commit SHA
				if verbose {
					fmt.Printf("Attempting to resolve ref '%s' to a specific commit SHA for path '%s' in repo '%s/%s'...\\n", parsedInfo.Ref, parsedInfo.PathInRepo, parsedInfo.Owner, parsedInfo.Repo)
				}
				commitSHA, err := source.GetLatestCommitSHAForFile(parsedInfo.Owner, parsedInfo.Repo, parsedInfo.PathInRepo, parsedInfo.Ref)
				if err != nil {
					if verbose {
						fmt.Printf("Warning: Failed to get specific commit SHA for '%s@%s': %v. Falling back to SHA256 content hash for lockfile.\\n", parsedInfo.PathInRepo, parsedInfo.Ref, err)
					}
					integrityHash = fileHashSHA256
				} else {
					if verbose {
						fmt.Printf("Successfully resolved ref '%s' to commit SHA '%s'.\\n", parsedInfo.Ref, commitSHA)
					}
					integrityHash = fmt.Sprintf("commit:%s", commitSHA)
				}
			}
		} else {
			if verbose && parsedInfo.Provider == "github" {
				fmt.Printf("Insufficient information or invalid ref ('%s') to fetch specific commit SHA for GitHub source. Falling back to SHA256 content hash for lockfile.\\n", parsedInfo.Ref)
			} else if verbose {
				fmt.Printf("Source is not GitHub or ref is missing. Falling back to SHA256 content hash for lockfile.\\n")
			}
			integrityHash = fileHashSHA256 // Fallback to SHA256
		}

		// For lockfile, use the exact raw download URL and calculated integrity hash
		lf.AddOrUpdatePackage(dependencyNameInManifest, parsedInfo.RawURL, relativeDestPath, integrityHash)

		if saveLockErr := lockfile.Save(projectRoot, lf); saveLockErr != nil {
			err = fmt.Errorf("saving %s: %w", lockfile.LockfileName, saveLockErr)
			// project.toml was successfully written.
			return cli.Exit(fmt.Sprintf("Error %s. File '%s' saved and %s updated, but saving %s failed. %s and %s may be inconsistent. Downloaded file '%s' is being cleaned up.", err, fullPath, config.ProjectTomlName, lockfile.LockfileName, config.ProjectTomlName, lockfile.LockfileName, fullPath), 1)
		}

		if verbose {
			fmt.Printf("Successfully updated %s for dependency '%s'.\n", lockfile.LockfileName, dependencyNameInManifest)
		}

		fmt.Printf("Successfully added '%s' from '%s' to '%s'.\nUpdated %s and %s.\n", // Simplified success message
			dependencyNameInManifest, sourceURLInput, fullPath, config.ProjectTomlName, lockfile.LockfileName)

		return nil // err is nil, so defer func() will not trigger cleanup
	},
}
