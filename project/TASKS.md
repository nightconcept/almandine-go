# Task Checklist: Almandine Go Implementation - `init` & `add` Commands

**Purpose:** Tracks tasks and milestones for implementing the core `init` and `add` commands for the Almandine Go CLI (`almd`), based on the specifications in `project/PRD.md`.

**Multiplatform Policy:** All implementations MUST be compatible with Linux, macOS, and Windows.

---

## Milestone 0: Initial Setup & `main.go` Entrypoint

**Goal:** Create the basic Go project structure and the main CLI entry point using `urfave/cli`.

-   [x] **Task 0.1: Initialize Go Module**
    -   [x] Run `go mod init <module_path>` (e.g., `go mod init github.com/your-user/almandine-go`). *User needs to determine the module path.*
    -   [x] Add `urfave/cli/v2` dependency (`go get github.com/urfave/cli/v2`).
    -   [x] Manual Verification: `go.mod` and `go.sum` are created/updated.

-   [x] **Task 0.2: Create `main.go`**
    -   [x] Create the `main.go` file at the project root.
    -   [x] Add the basic `main` function.
    -   [x] Manual Verification: File exists.

-   [x] **Task 0.3: Basic `urfave/cli` App Setup**
    -   [x] Import `urfave/cli/v2`.
    -   [x] Create a new `cli.App` instance in `main`.
    -   [x] Set the `Name` (`almd`), `Usage`, and `Version` for the app.
    -   [x] Implement the `app.Run(os.Args)` call.
    -   [x] Manual Verification: Run `go run main.go --version` and confirm the version is printed. Run `go run main.go --help` and confirm basic usage is shown.

-   [x] **Task 0.4: Define CLI Binary Name Convention**
    -   [x] Ensure the target executable name built by Go is `almd`.
    -   [x] *Note:* A separate wrapper script/alias named `almd` will be used by end-users to call `almd`. This task is about the Go build output name. (Build command might be `go build -o almd .`)
    -   [x] Manual Verification: Build the project (`go build -o almd .`) and confirm the output file is named `almd`.

---

## CLI Tool Name

-   The CLI executable is called `almd`.
-   All documentation, usage, and examples should refer to the CLI as `almd`.

---

## Milestone 1: `init` Command Implementation

**Goal:** Implement the `almd init` command to interactively create a `project.toml` manifest file.

-   [x] **Task 1.1: `urfave/cli` Command Setup**
    -   [x] Define the `init` command structure (`cli.Command`) within `commands/init.go`.
    -   [x] Add the command to the `urfave/cli` App in `main.go`.
    -   [x] Ensure basic command registration works (`almd init --help`).
    -   [x] Manual Verification: Run `almd init --help` and confirm the command is listed.

-   [x] **Task 1.2: Implement Interactive Prompts for Metadata**
    -   [x] Add logic within the `init` command's `Action` to prompt the user for:
        -   `package` name (with default).
        -   `version` (with default `0.1.0`).
        -   `license` (with default `MIT`).
        -   `description` (with default).
        -   Optional: `language` details (consider defaulting initially).
    -   [x] Manual Verification: Run `almd init` interactively and confirm prompts appear and capture input correctly.

-   [x] **Task 1.3: Implement Interactive Prompts for Scripts**
    -   [x] Add logic to loop, prompting for script `name` and `command`.
    -   [x] Store collected scripts (e.g., in a `map[string]string`).
    -   [x] Exit the loop when an empty script name is entered.
    -   [x] Implement logic to add a default `run` script (`lua src/main.lua`) if the user doesn't define one.
    -   [x] Manual Verification: Run `almd init` interactively, add a few scripts, skip adding `run`, and verify the default is included conceptually (actual file writing is next).

-   [x] **Task 1.4: Implement Interactive Prompts for Dependencies (Placeholders)**
    -   [x] Add logic to loop, prompting for dependency `name` and a simple `source/version` string (as per PRD).
    -   [x] Store collected dependency placeholders (e.g., in a `map[string]string` or `map[string]interface{}`).
    -   [x] Exit the loop when an empty dependency name is entered.
    -   [x] Manual Verification: Run `almd init` interactively, add a few placeholder dependencies.

-   [x] **Task 1.5: Implement `project.toml` Structure and Writing**
    -   [x] Define Go structs in `internal/project/` to represent the `project.toml` structure (package info, scripts, dependencies).
    -   [x] Create functions in `internal/config/` to marshal the collected data into the Go struct and write it to `project.toml` using a TOML library (`github.com/BurntSushi/toml`).
    -   [x] Ensure the function correctly handles overwriting an existing `project.toml`.
    -   [x] Integrate this writing logic into the `init` command's `Action`.
    -   [x] Add clear output messages (success, errors).
    -   [x] Manual Verification: Run `almd init`, provide input, and verify `project.toml` is created correctly with the specified data and defaults. Run again and verify it overwrites. Check error handling for write failures (e.g., permissions).

## Milestone 2: `add` Command Implementation

**Goal:** Implement the `almd add <source_url>` command to download a single-file dependency, update `project.toml`, and update `almd-lock.toml`.

-   [x] **Task 2.1: `urfave/cli` Command Setup & Argument/Flag Parsing**
    -   [x] Define the `add` command structure (`cli.Command`) in `commands/add.go`.
    -   [x] Define the required `<source_url>` argument.
    -   [x] Define the flags: `-d, --directory string`, `-n, --name string`, `--verbose bool`.
    -   [x] Add the command to the `urfave/cli` App in `main.go`.
    -   [x] Implement basic parsing logic within the `Action` to retrieve the argument and flag values.
    -   [x] Manual Verification: Run `almd add --help` and confirm the command, argument, and flags are listed correctly. Run `almd add some-url -n test -d testdir --verbose` and verify the values are accessible within the (currently empty) action.

-   [x] **Task 2.2: Implement Source URL Handling (`internal/source`)**
    -   [x] Create package `internal/source`.
    -   [x] Implement functions to parse the input `<source_url>` (`net/url`).
    -   [x] Implement logic specifically for GitHub URLs:
        -   Normalize various formats (blob, raw) to the raw content download URL.
        -   Extract commit hash/ref if present.
        -   Create the canonical source identifier string (e.g., `github:user/repo/path@hash`).
    -   [x] Define return structures or values for the raw URL, canonical identifier, and extracted commit hash.
    -   [x] Manual Verification: Test the parsing functions with various valid and invalid GitHub URL formats. (Code review of parsing logic done, specific unit tests are outside this immediate task but recommended next)

-   [x] **Task 2.3: Implement File Downloading (`internal/downloader`)**
    -   [x] Create package `internal/downloader`.
    -   [x] Implement a function that takes a URL (the raw download URL from Task 2.2) and fetches the content using `net/http`.
    -   [x] Handle potential HTTP errors (status codes, network issues).
    -   [x] Return the downloaded content (e.g., as `[]byte`).
    -   [x] Manual Verification: Test the download function with a known raw GitHub file URL. (Code implemented; manual test by user pending integration)

-   [x] **Task 2.4: Implement Target Path Logic & File Saving**
    -   [x] Add logic within the `add` command's `Action` to determine the final destination path based on the `-d` flag, `-n` flag (or inferred name), and the project root.
    -   [x] Use `os.MkdirAll` to create the target directory if it doesn't exist.
    -   [x] Use `os.WriteFile` to save the downloaded content (`[]byte` from Task 2.3) to the determined path.
    -   [x] Handle file writing errors.
    -   [x] Manual Verification: Run `almd add <url>` with different `-d` and `-n` combinations and verify the file is saved to the correct location with the correct name. Test directory creation.

-   [x] **Task 2.5: Implement Hashing (`internal/hasher`)**
    -   [x] Create package `internal/hasher`.
    -   [x] Implement a function to calculate the SHA256 hash of file content (`[]byte`) using `crypto/sha256`.
    -   [x] Format the output hash string as `sha256:<hex_hash>`.
    -   [x] Manual Verification: Test the hashing function with known content and verify the output hash.

-   [x] **Task 2.6: Define Data Structures (`internal/project`)**
    -   [x] Extend Go structs in `internal/project/` to represent the `dependencies` table structure in `project.toml` (sub-table with `source`, `path`).
    -   [x] Define Go structs for the `almd-lock.toml` structure (`api_version`, `[package]` table with entries containing `source`, `path`, `hash`).
    -   [x] Manual Verification: Code review confirms structs accurately model the TOML structures defined in `PRD.md`.

-   [x] **Task 2.7: Implement Manifest Update (`internal/config`)**
    -   [x] Add functions in `internal/config/` to:
        -   [x] Load an existing `project.toml`.
        -   [x] Add or update a dependency entry in the `[dependencies]` map using the dependency name (from `-n` or inferred), canonical source identifier (Task 2.2), and relative file path (Task 2.4).
        -   [x] Save the updated manifest back to `project.toml`.
    -   [x] Integrate this logic into the `add` command's `Action`.
    -   [x] Manual Verification: Run `almd add <url>`, then inspect `project.toml` to verify the dependency entry is added/updated correctly.

-   [x] **Task 2.8: Implement Lockfile Update (`internal/lockfile`)**
    -   [x] Create package `internal/lockfile`.
    -   [x] Add functions to:
        -   [x] Load `almd-lock.toml` (handling file not found initially).
        -   [x] Calculate the integrity hash string: `commit:<commit_hash>` (if available from Task 2.2) or `sha256:<hash>` (from Task 2.5). Handle potential hashing errors (`hash_error:<reason>`).
        -   [x] Add or update an entry in the `[package]` map using the dependency name, the *exact raw download URL* (Task 2.2), the relative file path (Task 2.4), and the calculated hash string.
        -   [x] Set/ensure `api_version = "1"`.
        -   [x] Save the updated lockfile back to `almd-lock.toml`.
    -   [x] Integrate this logic into the `add` command's `Action`.
    -   [x] Manual Verification: Run `almd add <url>`, then inspect `almd-lock.toml` to verify the entry is added/updated with the correct source URL, path, and hash format.

-   [x] **Task 2.9: Error Handling and Cleanup**
    -   [x] Review the `add` command's `Action` logic.
    -   [x] Implement error handling using `urfave/cli`'s error reporting (e.g., `cli.Exit`).
    -   [x] If an error occurs *after* downloading the file but *before* successfully updating both manifest and lockfile, attempt to delete the downloaded file to maintain consistency.
    -   [x] Ensure clear error messages are provided to the user.
    -   [x] Manual Verification: Test error scenarios: invalid URL, download failure, write permission errors for manifest/lockfile, simulate failures mid-process to check cleanup.

## Milestone 3: Initial Testing Setup

**Goal:** Establish the basic structure for tests for the `init` and `add` commands.

-   [x] **Task 3.1: Define Testing Strategy**
    -   [x] Framework: Standard Go `testing` package with `testify` for assertions.
    -   [x] `init` command: Unit tests directly invoking the command's `Action`, simulating user input (as in `commands/init_test.go`).
    -   [x] `add` command: Unit tests directly invoking the command's `Action` (via `app.Run` within the test).
        -   [x] Network calls for `add` will be mocked using `net/http/httptest`.
        -   [x] File system operations will occur in temporary directories created by tests.
    -   [x] Setup/Teardown: Tests will create temporary directories and necessary initial files (e.g., `project.toml`), and these will be cleaned up automatically by `t.TempDir()` or explicit `defer os.RemoveAll`.
    -   [x] Manual Verification: Review the chosen strategy for feasibility.

-   [x] **Task 3.2: Create Test File Structure**
    -   [x] Test file for `init` command: `commands/init_test.go` (exists).
    -   [x] Create test file for `add` command: `commands/add_test.go`.
    -   [x] Implement shared test helpers if applicable (e.g., for creating temp env, running command actions).
    -   [x] Manual Verification: Run `go test ./...` and confirm test files are picked up.

-   [ ] **Task 3.3: Implement `init` Command Test Cases (Existing)**
    -   [x] Basic `init` test case (as in `commands/init_test.go`).
    -   [x] `init` test case with defaults and empty inputs (as in `commands/init_test.go`).

-   [ ] **Task 3.4: Implement `add` Command Unit Test Cases**
    -   [x] **Sub-Task 3.4.1: Setup for `add` tests in `commands/add_test.go`**
        -   [x] Define `TestMain` if any global setup/teardown for `add` tests is needed.
        -   [x] Create helper: `setupAddTestEnvironment(t *testing.T, initialProjectTomlContent string) (tempDir string)` that creates a temp dir and a `project.toml`.
        -   [x] Create helper: `runAddCommand(t *testing.T, tempDir string, mockServerURL string, cliArgs ...string) error` to set up and run the `add` command's action using an `cli.App` instance.
        -   [x] Create helper: `startMockHTTPServer(t *testing.T, content string, expectedPath string, statusCode int) *httptest.Server`.
    -   [x] **Sub-Task 3.4.2: Test `almd add` - Successful Download and Update (Explicit Name, Custom Directory)**
        -   [x] Setup: Temp dir, basic `project.toml`, mock HTTP server serving test content.
        -   [x] Execute: `almd add <mock_url_to_file> -n mylib -d vendor/custom`.
        -   [x] Verify:
            -   `vendor/custom/mylib` created with correct content.
            -   `project.toml` updated with `[dependencies.mylib]` pointing to `source` and `path="vendor/custom/mylib"`.
            -   `almd-lock.toml` created/updated with `[package.mylib]` including `source`, `path`, and `hash="sha256:..."`.
    -   [x] **Sub-Task 3.4.3: Test `almd add` - Successful Download (Inferred Name, Default Directory)**
        -   [x] Execute: `almd add <mock_url_to_file.lua>`.
        -   [x] Verify:
            -   `libs/file.lua` (or project root, per PRD) created.
            -   Manifest and lockfile updated with inferred name `file.sh`.
    -   [x] **Sub-Task 3.4.4: Test `almd add` - GitHub URL with Commit Hash**
        -   [x] URL can include a commit hash segment (e.g., `file.lua@commitsha`) or a branch/tag name (e.g., `file.lua@main`).
        -   [x] Verify `almd-lock.toml` `hash` field reflects `commit:<actual_commit_sha>`. If original URL was a branch/tag, it's resolved to the latest commit SHA for that file on that branch/tag. If original URL was a commit SHA, that SHA is used.
        -   [x] If GitHub API call fails to resolve a branch/tag, or if not a GitHub URL, verify fallback to `sha256:<content_hash>`.
    -   [x] **Sub-Task 3.4.5: Test `almd add` - Error: Download Failure (HTTP Error)**
        -   [x] Mock server returns non-200 status.
        -   [x] Verify command returns an error.
        -   [x] Verify no dependency file is created.
        -   [x] Verify `project.toml` and `almd-lock.toml` are not modified (or created if they didn't exist).
    -   [x] **Sub-Task 3.4.6: Test `almd add` - Error: `project.toml` Not Found**
        -   [x] Run `add` in a temp dir without `project.toml`.
        -   [x] Verify command returns an appropriate error.
    -   [x] **Sub-Task 3.4.7: Test `almd add` - Cleanup on Failure (e.g., Lockfile Write Error)**
        -   [x] Difficult to precisely mock file system write errors without more DI.
        -   [x] Focus on: If download happens, but a subsequent step like TOML marshaling or lockfile writing fails, does the downloaded file get removed? (This might require a test where the mock HTTP server succeeds, but we introduce an error in a subsequent, controllable step if possible, or inspect code paths for this cleanup logic). Initially, can be a lower priority if hard to test cleanly.
    -   [x] **Sub-Task 3.4.8: Fix `TestAddCommand_ProjectTomlNotFound` (2025-05-07)**
        -   [x] Modified error message in `internal/cli/add/add.go` to include "no such file or directory" when `project.toml` is not found.
        -   [x] Refactored `Action` in `internal/cli/add/add.go` to use a named return error, ensuring the deferred cleanup logic correctly removes downloaded files when `project.toml` is missing and an error is returned.
        -   [x] Corrected variable types for `proj` (to `*project.Project`) and `lf` (to `*lockfile.Lockfile`) in `internal/cli/add/add.go` to resolve compiler errors.


## Milestone 4: `remove` Command Implementation

**Goal:** Implement the `almd remove <dependency_name>` command to remove a dependency from the project.

-   [x] **Task 4.1: `urfave/cli` Command Setup**
    -   [x] Define the `remove` command structure (`cli.Command`) in `commands/remove.go` (or `internal/cli/remove/remove.go` as per PRD folder structure).
    -   [x] Add the command to the `urfave/cli` App in `main.go`.
    -   [x] Define the required `<dependency_name>` argument.
    -   [x] Manual Verification: Run `almd remove --help` and confirm the command and argument are listed correctly. Run `almd remove some-dep` and verify the argument value is accessible within the (currently empty) action.

-   [x] **Task 4.2: Implement Manifest Loading and Dependency Path Retrieval**
    -   [x] Add logic within the `remove` command's `Action` to load `project.toml` (using `internal/config`).
    -   [x] Verify if the specified `<dependency_name>` exists in the `[dependencies]` table.
    -   [x] If it exists, retrieve the relative `path` of the dependency.
    -   [x] Handle errors if `project.toml` is not found or the dependency does not exist.
    -   [x] Manual Verification: Test with an existing `project.toml`. Try removing an existing and a non-existing dependency. Check error messages.

-   [x] **Task 4.3: Implement Manifest Update and File Deletion**
    -   [x] Remove the entry for `<dependency_name>` from the `[dependencies]` table in the loaded manifest data.
    -   [x] Save the updated manifest back to `project.toml`.
    -   [x] Delete the file specified by the retrieved `path` using `os.Remove`.
    -   [x] Handle potential errors during file saving and deletion (e.g., permissions, file not found for deletion).
    -   [x] Manual Verification: Add a dependency using `almd add`. Then use `almd remove <dep_name>`. Verify `project.toml` is updated and the file is deleted. Test error conditions like read-only `project.toml` or non-existent dependency file.

-   [x] **Task 4.4: Implement Lockfile Update**
    -   [x] Load `almd-lock.toml` (using `internal/lockfile`).
    -   [x] Remove the entry for `<dependency_name>` from the `[package]` table in the loaded lockfile data.
    -   [x] Save the updated lockfile back to `almd-lock.toml`.
    -   [x] Handle errors if `almd-lock.toml` is not found or during saving. Handle cases where the dependency might not be in the lockfile even if it was in the manifest.
    -   [x] Manual Verification: After successfully running `almd add`, run `almd remove <dep_name>`. Verify `almd-lock.toml` is updated. Test with missing or read-only `almd-lock.toml`.

-   [x] **Task 4.5: Error Handling and Output**
    -   [x] Ensure robust error handling for all operations using `urfave/cli`'s error reporting (e.g., `cli.Exit`).
    -   [x] Provide clear confirmation messages for successful removal (manifest, file, lockfile).
    -   [x] Provide clear error messages for different failure scenarios.
    -   [x] Manual Verification: Test various error paths (missing files, non-existent dependency, permission issues) and check for clear, user-friendly output.

-   [x] **Task 4.6: Implement Empty Directory Cleanup**
    -   [x] After successful file deletion in `remove` (Task 4.3), check if the parent directory of the deleted file is empty.
    -   [x] If the directory is empty, delete it.
    -   [x] Repeat this process, moving upwards to parent directories, deleting them if they become empty.
    -   [x] Stop if a directory is not empty, an error occurs, or a predefined boundary (e.g., project root, `libs/`, `vendor/`) is reached.
    -   [x] Ensure directory emptiness check is robust to prevent accidental deletion of non-empty directories.
    -   [x] Manual Verification: Test scenarios where single and multiple empty parent directories are cleaned up. Test scenarios where cleanup stops appropriately. (Note: Manual verification by user is pending actual use, code implements the logic).

## Milestone 5: `remove` Command Testing

**Goal:** Implement unit tests for the `remove` command.

-   [x] **Task 5.1: Create Test File Structure for `remove`**
    -   [x] Create test file: `internal/cli/remove/remove_test.go`.
    -   [x] Implement shared test helpers if applicable (e.g., for creating temp env with `project.toml`, `almd-lock.toml`, and dummy dependency files).

-   [ ] **Task 5.2: Implement `remove` Command Unit Test Cases**
    -   [x] **Sub-Task 5.2.1: Setup for `remove` tests**
        -   [x] Define `TestMain` if any global setup/teardown for `remove` tests is needed. (Skipped for now, can be added if specific global setup is identified)
        -   [x] Create helper: `setupRemoveTestEnvironment(t *testing.T, initialProjectTomlContent string, initialLockfileContent string, depFiles map[string]string) (tempDir string)` that creates a temp dir, `project.toml`, `almd-lock.toml`, and specified dependency files.
        -   [x] Create helper: `runRemoveCommand(t *testing.T, tempDir string, cliArgs ...string) error` to set up and run the `remove` command's action.
    -   [x] **Sub-Task 5.2.2: Test `almd remove` - Successful Removal**
        -   [x] Setup: Temp dir with `project.toml`, `almd-lock.toml`, and a dummy dependency file, all correctly linked.
        -   [x] Execute: `almd remove <dependency_name>`.
        -   [x] Verify:
            -   Dependency entry removed from `project.toml`.
            -   Dependency entry removed from `almd-lock.toml`.
            -   Dependency file deleted from the filesystem.
            -   Command returns no error.
    -   [x] **Sub-Task 5.2.3: Test `almd remove` - Error: Dependency Not Found in Manifest**
        -   [x] Setup: Temp dir with `project.toml` that does not contain the target dependency.
        -   [x] Execute: `almd remove <non_existent_dependency_name>`.
        -   [x] Verify:
            -   Command returns an appropriate error.
            -   `project.toml` and `almd-lock.toml` remain unchanged.
            -   No file deletion attempted for the non-existent dependency.
           -   [x] **Sub-Task 5.2.4: Test `almd remove` - Error: Dependency File Not Found for Deletion**
            -   [x] Setup: Temp dir with `project.toml` and `almd-lock.toml` listing a dependency, but the actual dependency file is missing.
            -   [x] Execute: `almd remove <dependency_name>`.
        -   [x] Verify:
        	-   Dependency entry removed from `project.toml`.
        	-   Dependency entry removed from `almd-lock.toml`.
        	-   Command may return a warning or error about file deletion failure, but manifest/lockfile changes should persist.
        	-   PRD: "Handles potential errors gracefully (e.g., file not found, permissions)."
       -   [x] **Sub-Task 5.2.5: Test `almd remove` - Error: `project.toml` Not Found (2025-05-07)**
        -   [x] Setup: Run `remove` in a temp dir without `project.toml`.
        -   [x] Execute: `almd remove <dependency_name>`.
        -   [x] Verify: Command returns an appropriate error.
    -   [ ] **Sub-Task 5.2.6: Test `almd remove` - Dependency in Manifest but not Lockfile**
        -   [ ] Setup: Temp dir with `project.toml` listing a dependency, `almd-lock.toml` exists but doesn't list it, and the dependency file exists.
        -   [ ] Execute: `almd remove <dependency_name>`.
        -   [ ] Verify:
            -   Dependency entry removed from `project.toml`.
            -   `almd-lock.toml` is processed (attempt to remove, no error if not found).
            -   Dependency file deleted.
            -   Command completes successfully or with a notice about the lockfile state.
    -   [ ] **Sub-Task 5.2.7: Test `almd remove` - Empty `project.toml` or `almd-lock.toml`**
        -   [ ] Setup: Temp dir with empty `project.toml` and/or `almd-lock.toml`.
        -   [ ] Execute: `almd remove <dependency_name>`.
        -   [ ] Verify: Command returns an error indicating dependency not found, and files remain empty or unchanged.
