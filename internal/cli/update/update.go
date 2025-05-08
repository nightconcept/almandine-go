package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/nightconcept/almandine-go/internal/core/config"
	"github.com/nightconcept/almandine-go/internal/core/downloader"
	"github.com/nightconcept/almandine-go/internal/core/hasher"
	"github.com/nightconcept/almandine-go/internal/core/lockfile"
	"github.com/nightconcept/almandine-go/internal/core/source"
)

var isCommitSHARegex = regexp.MustCompile(`^[0-9a-f]{7,40}$`) // Common Git SHA lengths

// NewUpdateCommand creates a new cli.Command for the "update" command.
func NewUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Updates project dependencies based on project.toml",
		ArgsUsage: "[dependency_names...]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Force update even if versions appear to match",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose output",
			},
		},
		Action: func(c *cli.Context) error {
			verbose := c.Bool("verbose")
			force := c.Bool("force") // Keep force for later use

			if verbose {
				_, _ = fmt.Fprintln(os.Stdout, "Executing 'update' command...")
				if force {
					_, _ = fmt.Fprintln(os.Stdout, "Force update enabled.")
				}
			}

			dependencyNames := c.Args().Slice()
			if verbose {
				if len(dependencyNames) > 0 {
					_, _ = fmt.Fprintf(os.Stdout, "Targeted dependencies for update: %v\n", dependencyNames)
				} else {
					_, _ = fmt.Fprintln(os.Stdout, "Targeting all dependencies for update.")
				}
			}

			// Load project.toml
			projCfg, err := config.LoadProjectToml(".") // Corrected function name
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return cli.Exit("Error: project.toml not found in the current directory. Please run 'almd init' first.", 1)
				}
				return cli.Exit(fmt.Sprintf("Error loading project.toml: %v", err), 1)
			}
			if verbose {
				_, _ = fmt.Fprintf(os.Stdout, "Successfully loaded project.toml (Package: %s)\n", projCfg.Package.Name)
			}

			// Load almd-lock.toml
			lf, err := lockfile.Load(".") // Corrected function name
			if err != nil {
				// lockfile.Load returns a new lockfile if not found, so os.ErrNotExist is handled internally by lockfile.Load
				// We only need to check for other errors.
				// However, the current lockfile.Load returns lf, nil on NotExist. Let's adjust to that.
				// The previous logic was: if errors.Is(err, os.ErrNotExist) then initialize.
				// The new lockfile.Load does this: if os.IsNotExist(err) { return lf, nil }
				// So, if err is nil, lf is either loaded or a new one. If err is not nil, it's a real error.

				// Re-evaluating based on lockfile.Load's actual behavior:
				// It returns a new *Lockfile and nil error if not found.
				// So, if err != nil here, it's a genuine error other than NotExist.
				return cli.Exit(fmt.Sprintf("Error loading almd-lock.toml: %v", err), 1)
			}
			// If lf was nil (e.g. if Load could return nil *Lockfile on error), we'd need to initialize.
			// But Load always returns a valid *Lockfile or an error.
			// If it was newly created by Load, APIVersion and Packages map are set by lockfile.New() called within lockfile.Load().
			if verbose {
				// Check if it was just created (e.g. by checking if its Packages map is empty and it has default API version)
				// This is a bit heuristic. A better way would be for lockfile.Load to return a flag.
				// For now, just log successful load.
				_, _ = fmt.Fprintln(os.Stdout, "Successfully loaded or initialized almd-lock.toml.")
			}
			// Ensure Packages map is initialized, though lockfile.New should do it.
			if lf.Package == nil {
				lf.Package = make(map[string]lockfile.PackageEntry)
			}
			// Ensure APIVersion is set, though lockfile.New should do it.
			if lf.ApiVersion == "" { // This check might be redundant if lockfile.Load guarantees it.
				lf.ApiVersion = lockfile.APIVersion
			}
			// The 'else' block that was here previously was syntactically incorrect
			// and its intended verbose message for a successful load is covered by the
			// "Successfully loaded or initialized almd-lock.toml." message at line 94.

			// --- Task 6.3: Dependency Iteration and Configuration Retrieval ---
			type dependencyToProcess struct {
				Name   string
				Source string
				Path   string
			}
			var dependenciesToProcessList []dependencyToProcess

			if len(dependencyNames) == 0 { // Update all dependencies defined in project.toml
				if len(projCfg.Dependencies) == 0 {
					_, _ = fmt.Fprintln(os.Stdout, "No dependencies found in project.toml to update.")
					return nil
				}
				if verbose {
					_, _ = fmt.Fprintf(os.Stdout, "Processing all %d dependencies from project.toml...\n", len(projCfg.Dependencies))
				}
				for name, depDetails := range projCfg.Dependencies {
					dependenciesToProcessList = append(dependenciesToProcessList, dependencyToProcess{
						Name:   name,
						Source: depDetails.Source,
						Path:   depDetails.Path,
					})
					if verbose {
						_, _ = fmt.Fprintf(os.Stdout, "  Targeting: %s (Source: %s, Path: %s)\n", name, depDetails.Source, depDetails.Path)
					}
				}
			} else { // Update specific dependencies
				if verbose {
					_, _ = fmt.Fprintf(os.Stdout, "Processing %d specified dependencies...\n", len(dependencyNames))
				}
				for _, name := range dependencyNames {
					depDetails, ok := projCfg.Dependencies[name]
					if !ok {
						_, _ = fmt.Fprintf(os.Stderr, "Warning: Dependency '%s' specified for update not found in project.toml. Skipping.\n", name)
						continue
					}
					dependenciesToProcessList = append(dependenciesToProcessList, dependencyToProcess{
						Name:   name,
						Source: depDetails.Source,
						Path:   depDetails.Path,
					})
					if verbose {
						_, _ = fmt.Fprintf(os.Stdout, "  Targeting: %s (Source: %s, Path: %s)\n", name, depDetails.Source, depDetails.Path)
					}
				}
				if len(dependenciesToProcessList) == 0 {
					_, _ = fmt.Fprintln(os.Stdout, "No specified dependencies were found in project.toml to update.")
					return nil
				}
			}

			if verbose {
				_, _ = fmt.Fprintf(os.Stdout, "Total dependencies to process: %d\n", len(dependenciesToProcessList))
			}

			// --- Task 6.4: Target Version Resolution and Lockfile State Retrieval ---
			type dependencyUpdateState struct {
				Name              string
				ProjectTomlSource string // Original source string from project.toml
				ProjectTomlPath   string // Path from project.toml
				TargetRawURL      string // Resolved raw URL for download
				TargetCommitHash  string // Resolved definitive commit hash (or tag/branch if not resolvable to commit)
				LockedRawURL      string // Raw URL from almd-lock.toml
				LockedCommitHash  string // Hash from almd-lock.toml (could be commit:<sha> or sha256:<hash>)
				Provider          string
				Owner             string
				Repo              string
				PathInRepo        string
				NeedsUpdate       bool   // Flag to indicate if this dependency needs to be updated
				UpdateReason      string // Reason why an update is needed
			}
			var updateStates []dependencyUpdateState

			if verbose && len(dependenciesToProcessList) > 0 {
				_, _ = fmt.Fprintln(os.Stdout, "\nResolving target versions and current lock states...")
			}

			for _, depToProcess := range dependenciesToProcessList {
				if verbose {
					_, _ = fmt.Fprintf(os.Stdout, "Processing dependency: %s (Source: %s)\n", depToProcess.Name, depToProcess.Source)
				}

				parsedSourceInfo, err := source.ParseSourceURL(depToProcess.Source)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not parse source URL for dependency '%s' (%s): %v. Skipping.\n", depToProcess.Name, depToProcess.Source, err)
					continue
				}

				var resolvedCommitHash = parsedSourceInfo.Ref // Default to the ref from parsing
				var finalTargetRawURL = parsedSourceInfo.RawURL

				// If it's a GitHub source and the ref doesn't look like a full commit SHA, try to resolve it to one.
				if parsedSourceInfo.Provider == "github" && !isCommitSHARegex.MatchString(parsedSourceInfo.Ref) {
					if verbose {
						_, _ = fmt.Fprintf(os.Stdout, "  Ref '%s' for '%s' is not a full commit SHA. Attempting to resolve latest commit for path '%s'...\n", parsedSourceInfo.Ref, depToProcess.Name, parsedSourceInfo.PathInRepo)
					}
					latestSHA, err := source.GetLatestCommitSHAForFile(parsedSourceInfo.Owner, parsedSourceInfo.Repo, parsedSourceInfo.PathInRepo, parsedSourceInfo.Ref)
					if err != nil {
						_, _ = fmt.Fprintf(os.Stderr, "  Warning: Could not resolve ref '%s' to a specific commit for '%s': %v. Proceeding with ref as is.\n", parsedSourceInfo.Ref, depToProcess.Name, err)
					} else {
						if verbose {
							_, _ = fmt.Fprintf(os.Stdout, "  Resolved ref '%s' to commit SHA: %s for '%s'\n", parsedSourceInfo.Ref, latestSHA, depToProcess.Name)
						}
						resolvedCommitHash = latestSHA
						finalTargetRawURL = strings.Replace(parsedSourceInfo.RawURL, "/"+parsedSourceInfo.Ref+"/", "/"+latestSHA+"/", 1)
					}
				} else if verbose && parsedSourceInfo.Provider == "github" {
					_, _ = fmt.Fprintf(os.Stdout, "  Ref '%s' for '%s' appears to be a commit SHA. Using it directly.\n", parsedSourceInfo.Ref, depToProcess.Name)
				}

				currentState := dependencyUpdateState{
					Name:              depToProcess.Name,
					ProjectTomlSource: depToProcess.Source,
					ProjectTomlPath:   depToProcess.Path,
					TargetRawURL:      finalTargetRawURL,
					TargetCommitHash:  resolvedCommitHash,
					Provider:          parsedSourceInfo.Provider,
					Owner:             parsedSourceInfo.Owner,
					Repo:              parsedSourceInfo.Repo,
					PathInRepo:        parsedSourceInfo.PathInRepo,
				}

				if lockDetails, ok := lf.Package[depToProcess.Name]; ok {
					currentState.LockedRawURL = lockDetails.Source
					currentState.LockedCommitHash = lockDetails.Hash
					if verbose {
						_, _ = fmt.Fprintf(os.Stdout, "  Found in lockfile: Name: %s, Locked Source: %s, Locked Hash: %s\n", depToProcess.Name, lockDetails.Source, lockDetails.Hash)
					}
				} else {
					if verbose {
						_, _ = fmt.Fprintf(os.Stdout, "  Dependency '%s' not found in lockfile.\n", depToProcess.Name)
					}
				}
				updateStates = append(updateStates, currentState)
			}

			if verbose && len(updateStates) > 0 {
				_, _ = fmt.Fprintln(os.Stdout, "\nFinished resolving versions. States to compare:")
				for _, s := range updateStates {
					_, _ = fmt.Fprintf(os.Stdout, "  - Name: %s, TargetCommit: %s, TargetURL: %s, LockedHash: %s, LockedURL: %s\n", s.Name, s.TargetCommitHash, s.TargetRawURL, s.LockedCommitHash, s.LockedRawURL)
				}
			}

			// --- Task 6.5: Comparison Logic and Update Decision ---
			var dependenciesThatNeedUpdate []dependencyUpdateState

			if verbose && len(updateStates) > 0 {
				_, _ = fmt.Fprintln(os.Stdout, "\nDetermining which dependencies need updates...")
			}

			for i, state := range updateStates {
				reason := ""
				needsUpdate := false

				// 1. --force flag
				if force {
					needsUpdate = true
					reason = "Update forced by user (--force)."
					if verbose {
						_, _ = fmt.Fprintf(os.Stdout, "  - %s: Needs update (forced).\n", state.Name)
					}
				}

				// 2. Dependency in project.toml but missing from almd-lock.toml
				if !needsUpdate && state.LockedCommitHash == "" {
					needsUpdate = true
					reason = "Dependency present in project.toml but not in almd-lock.toml."
					if verbose {
						_, _ = fmt.Fprintf(os.Stdout, "  - %s: Needs update (not in lockfile).\n", state.Name)
					}
				}

				// 3. Local file at path is missing
				if !needsUpdate {
					if _, err := os.Stat(state.ProjectTomlPath); errors.Is(err, os.ErrNotExist) {
						needsUpdate = true
						reason = fmt.Sprintf("Local file missing at path: %s.", state.ProjectTomlPath)
						if verbose {
							_, _ = fmt.Fprintf(os.Stdout, "  - %s: Needs update (file missing at %s).\n", state.Name, state.ProjectTomlPath)
						}
					} else if err != nil {
						// Other error stating file, potentially permissions, treat as needing check/potential update
						_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not stat file for dependency '%s' at '%s': %v. Assuming update check is needed.\n", state.Name, state.ProjectTomlPath, err)
						needsUpdate = true // Or handle more gracefully, for now, assume update if stat fails unexpectedly
						reason = fmt.Sprintf("Error checking local file status at %s: %v.", state.ProjectTomlPath, err)
					}
				}

				// 4. Resolved target commit hash differs from locked commit hash
				if !needsUpdate && state.TargetCommitHash != "" && state.LockedCommitHash != "" {
					var lockedSHA string
					if strings.HasPrefix(state.LockedCommitHash, "commit:") {
						lockedSHA = strings.TrimPrefix(state.LockedCommitHash, "commit:")
					}
					// If LockedCommitHash is sha256, this comparison won't match, which is fine.
					// An update would be triggered if TargetCommitHash (a specific commit) is now available
					// and the lockfile had a content hash (implying it wasn't locked to a specific commit before).

					if lockedSHA != "" && state.TargetCommitHash != lockedSHA {
						needsUpdate = true
						reason = fmt.Sprintf("Target commit hash (%s) differs from locked commit hash (%s).", state.TargetCommitHash, lockedSHA)
						if verbose {
							_, _ = fmt.Fprintf(os.Stdout, "  - %s: Needs update (target commit %s != locked commit %s).\n", state.Name, state.TargetCommitHash, lockedSHA)
						}
					} else if lockedSHA == "" && strings.HasPrefix(state.LockedCommitHash, "sha256:") && isCommitSHARegex.MatchString(state.TargetCommitHash) {
						// Case: Lockfile has content hash, but project.toml now resolved to a specific commit. This is an update.
						needsUpdate = true
						reason = fmt.Sprintf("Target is now a specific commit (%s), but lockfile has a content hash (%s).", state.TargetCommitHash, state.LockedCommitHash)
						if verbose {
							_, _ = fmt.Fprintf(os.Stdout, "  - %s: Needs update (target is specific commit %s, lockfile has content hash %s).\n", state.Name, state.TargetCommitHash, state.LockedCommitHash)
						}
					}
				}

				if needsUpdate {
					updateStates[i].NeedsUpdate = true
					updateStates[i].UpdateReason = reason
					dependenciesThatNeedUpdate = append(dependenciesThatNeedUpdate, updateStates[i])
				} else if verbose {
					_, _ = fmt.Fprintf(os.Stdout, "  - %s: Already up-to-date.\n", state.Name)
				}
			}

			if len(dependenciesThatNeedUpdate) == 0 {
				_, _ = fmt.Fprintln(os.Stdout, "All targeted dependencies are already up-to-date.")
				return nil
			}

			if verbose {
				_, _ = fmt.Fprintf(os.Stdout, "\nDependencies to be updated (%d):\n", len(dependenciesThatNeedUpdate))
				for _, dep := range dependenciesThatNeedUpdate {
					_, _ = fmt.Fprintf(os.Stdout, "  - %s (Reason: %s)\n", dep.Name, dep.UpdateReason)
				}
			}

			// --- Task 6.6: Perform Update (If Required) ---
			// This block was already applied in the previous step and seems correct.
			// The following is the same content, ensuring it matches the file.
			if verbose && len(dependenciesThatNeedUpdate) > 0 {
				_, _ = fmt.Fprintln(os.Stdout, "\nPerforming updates for identified dependencies...")
			}

			var successfulUpdates int
			for _, dep := range dependenciesThatNeedUpdate {
				if verbose {
					_, _ = fmt.Fprintf(os.Stdout, "  Updating '%s' from %s\n", dep.Name, dep.TargetRawURL)
				}

				// 1. Download the file
				fileContent, err := downloader.DownloadFile(dep.TargetRawURL)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "Error: Failed to download dependency '%s' from '%s': %v\n", dep.Name, dep.TargetRawURL, err)
					continue // Skip to next dependency
				}
				if verbose {
					_, _ = fmt.Fprintf(os.Stdout, "    Successfully downloaded %s (%d bytes)\n", dep.Name, len(fileContent))
				}

				// 2. Calculate integrity hash
				var integrityHash string
				if dep.Provider == "github" && isCommitSHARegex.MatchString(dep.TargetCommitHash) {
					integrityHash = "commit:" + dep.TargetCommitHash
					if verbose {
						_, _ = fmt.Fprintf(os.Stdout, "    Using commit hash for integrity: %s\n", integrityHash)
					}
				} else {
					contentHash, err := hasher.CalculateSHA256(fileContent)
					if err != nil {
						_, _ = fmt.Fprintf(os.Stderr, "Error: Failed to calculate SHA256 hash for dependency '%s': %v\n", dep.Name, err)
						continue // Skip to next dependency
					}
					integrityHash = contentHash
					if verbose {
						_, _ = fmt.Fprintf(os.Stdout, "    Calculated content hash for integrity: %s\n", integrityHash)
					}
				}

				// 3. Save the downloaded file
				// Ensure core/downloader and core/hasher are imported
				// Ensure path/filepath is imported for filepath.Dir
				targetDir := filepath.Dir(dep.ProjectTomlPath)
				if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "Error: Failed to create directory '%s' for dependency '%s': %v\n", targetDir, dep.Name, err)
					continue
				}
				if err := os.WriteFile(dep.ProjectTomlPath, fileContent, 0644); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "Error: Failed to write file '%s' for dependency '%s': %v\n", dep.ProjectTomlPath, dep.Name, err)
					continue
				}
				if verbose {
					_, _ = fmt.Fprintf(os.Stdout, "    Successfully saved %s to %s\n", dep.Name, dep.ProjectTomlPath)
				}

				// 4. Update almd-lock.toml data (in memory)
				lf.Package[dep.Name] = lockfile.PackageEntry{ // Corrected type
					Source: dep.TargetRawURL, // Store the exact URL used for download
					Path:   dep.ProjectTomlPath,
					Hash:   integrityHash,
				}
				if verbose {
					_, _ = fmt.Fprintf(os.Stdout, "    Updated lockfile entry for %s: Path=%s, Hash=%s, SourceURL=%s\n", dep.Name, dep.ProjectTomlPath, integrityHash, dep.TargetRawURL)
				}
				successfulUpdates++
			}

			if successfulUpdates > 0 {
				lf.ApiVersion = lockfile.APIVersion            // Ensure API version is set, Corrected constant
				if err := lockfile.Save(".", lf); err != nil { // Corrected function name and params
					return cli.Exit(fmt.Sprintf("Error: Failed to save updated almd-lock.toml: %v", err), 1)
				}
				if verbose {
					_, _ = fmt.Fprintf(os.Stdout, "\nSuccessfully saved almd-lock.toml with %d update(s).\n", successfulUpdates)
				}
				_, _ = fmt.Fprintf(os.Stdout, "Successfully updated %d dependenc(ies).\n", successfulUpdates)
			} else {
				if len(dependenciesThatNeedUpdate) > 0 { // Some were identified, but all failed
					_, _ = fmt.Fprintln(os.Stderr, "No dependencies were successfully updated due to errors.")
					return cli.Exit("Update process completed with errors for all targeted dependencies.", 1)
				}
				// This case should have been caught earlier if dependenciesThatNeedUpdate was empty
			}
			return nil
		},
	}
}
