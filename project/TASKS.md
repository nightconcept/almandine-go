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
    -   [x] Ensure the target executable name built by Go is `almd-cli`.
    -   [x] *Note:* A separate wrapper script/alias named `almd` will be used by end-users to call `almd-cli`. This task is about the Go build output name. (Build command might be `go build -o almd-cli .`)
    -   [x] Manual Verification: Build the project (`go build -o almd-cli .`) and confirm the output file is named `almd-cli`.

---

## CLI Tool Name

-   The CLI executable is called `almd`.
-   The built CLI shall be called `almd-cli`. A wrapper called `almd` will call it.
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
    -   [x] Implement logic to add a default `run` script (`go run .` or `go run main.go`) if the user doesn't define one.
    -   [x] Manual Verification: Run `almd init` interactively, add a few scripts, skip adding `run`, and verify the default is included conceptually (actual file writing is next).

-   [x] **Task 1.4: Implement Interactive Prompts for Dependencies (Placeholders)**
    -   [x] Add logic to loop, prompting for dependency `name` and a simple `source/version` string (as per PRD).
    -   [x] Store collected dependency placeholders (e.g., in a `map[string]string` or `map[string]interface{}`).
    -   [x] Exit the loop when an empty dependency name is entered.
    -   [x] Manual Verification: Run `almd init` interactively, add a few placeholder dependencies.

-   [ ] **Task 1.5: Implement `project.toml` Structure and Writing**
    -   [ ] Define Go structs in `internal/project/` to represent the `project.toml` structure (package info, scripts, dependencies).
    -   [ ] Create functions in `internal/config/` to marshal the collected data into the Go struct and write it to `project.toml` using a TOML library (`github.com/BurntSushi/toml`).
    -   [ ] Ensure the function correctly handles overwriting an existing `project.toml`.
    -   [ ] Integrate this writing logic into the `init` command's `Action`.
    -   [ ] Add clear output messages (success, errors).
    -   [ ] Manual Verification: Run `almd init`, provide input, and verify `project.toml` is created correctly with the specified data and defaults. Run again and verify it overwrites. Check error handling for write failures (e.g., permissions).

## Milestone 2: `add` Command Implementation

**Goal:** Implement the `almd add <source_url>` command to download a single-file dependency, update `project.toml`, and update `almd-lock.toml`.

-   [ ] **Task 2.1: `urfave/cli` Command Setup & Argument/Flag Parsing**
    -   [ ] Define the `add` command structure (`cli.Command`) in `commands/add.go`.
    -   [ ] Define the required `<source_url>` argument.
    -   [ ] Define the flags: `-d, --directory string`, `-n, --name string`, `--verbose bool`.
    -   [ ] Add the command to the `urfave/cli` App in `main.go`.
    -   [ ] Implement basic parsing logic within the `Action` to retrieve the argument and flag values.
    -   [ ] Manual Verification: Run `almd add --help` and confirm the command, argument, and flags are listed correctly. Run `almd add some-url -n test -d testdir --verbose` and verify the values are accessible within the (currently empty) action.

-   [ ] **Task 2.2: Implement Source URL Handling (`internal/source`)**
    -   [ ] Create package `internal/source`.
    -   [ ] Implement functions to parse the input `<source_url>` (`net/url`).
    -   [ ] Implement logic specifically for GitHub URLs:
        -   Normalize various formats (blob, raw) to the raw content download URL.
        -   Extract commit hash/ref if present.
        -   Create the canonical source identifier string (e.g., `github:user/repo/path@hash`).
    -   [ ] Define return structures or values for the raw URL, canonical identifier, and extracted commit hash.
    -   [ ] Manual Verification: Test the parsing functions with various valid and invalid GitHub URL formats.

-   [ ] **Task 2.3: Implement File Downloading (`internal/downloader`)**
    -   [ ] Create package `internal/downloader`.
    -   [ ] Implement a function that takes a URL (the raw download URL from Task 2.2) and fetches the content using `net/http`.
    -   [ ] Handle potential HTTP errors (status codes, network issues).
    -   [ ] Return the downloaded content (e.g., as `[]byte`).
    -   [ ] Manual Verification: Test the download function with a known raw GitHub file URL.

-   [ ] **Task 2.4: Implement Target Path Logic & File Saving**
    -   [ ] Add logic within the `add` command's `Action` (or a helper in `internal/util`) to determine the final destination path based on the `-d` flag, `-n` flag (or inferred name), and the project root.
    -   [ ] Use `os.MkdirAll` to create the target directory if it doesn't exist.
    -   [ ] Use `os.WriteFile` (or similar `io` operations) to save the downloaded content (`[]byte` from Task 2.3) to the determined path.
    -   [ ] Handle file writing errors.
    -   [ ] Manual Verification: Run `almd add <url>` with different `-d` and `-n` combinations and verify the file is saved to the correct location with the correct name. Test directory creation.

-   [ ] **Task 2.5: Implement Hashing (`internal/hasher`)**
    -   [ ] Create package `internal/hasher`.
    -   [ ] Implement a function to calculate the SHA256 hash of file content (`[]byte`) using `crypto/sha256`.
    -   [ ] Format the output hash string as `sha256:<hex_hash>`.
    -   [ ] Manual Verification: Test the hashing function with known content and verify the output hash.

-   [ ] **Task 2.6: Define Data Structures (`internal/project`)**
    -   [ ] Extend Go structs in `internal/project/` to represent the `dependencies` table structure in `project.toml` (sub-table with `source`, `path`).
    -   [ ] Define Go structs for the `almd-lock.toml` structure (`api_version`, `[package]` table with entries containing `source`, `path`, `hash`).
    -   [ ] Manual Verification: Code review confirms structs accurately model the TOML structures defined in `PRD.md`.

-   [ ] **Task 2.7: Implement Manifest Update (`internal/config`)**
    -   [ ] Add functions in `internal/config/` to:
        -   Load an existing `project.toml`.
        -   Add or update a dependency entry in the `[dependencies]` map using the dependency name (from `-n` or inferred), canonical source identifier (Task 2.2), and relative file path (Task 2.4).
        -   Save the updated manifest back to `project.toml`.
    -   [ ] Integrate this logic into the `add` command's `Action`.
    -   [ ] Manual Verification: Run `almd add <url>`, then inspect `project.toml` to verify the dependency entry is added/updated correctly.

-   [ ] **Task 2.8: Implement Lockfile Update (`internal/lockfile`)**
    -   [ ] Create package `internal/lockfile`.
    -   [ ] Add functions to:
        -   Load `almd-lock.toml` (handling file not found initially).
        -   Calculate the integrity hash string: `commit:<commit_hash>` (if available from Task 2.2) or `sha256:<hash>` (from Task 2.5). Handle potential hashing errors (`hash_error:<reason>`).
        -   Add or update an entry in the `[package]` map using the dependency name, the *exact raw download URL* (Task 2.2), the relative file path (Task 2.4), and the calculated hash string.
        -   Set/ensure `api_version = "1"`.
        -   Save the updated lockfile back to `almd-lock.toml`.
    -   [ ] Integrate this logic into the `add` command's `Action`.
    -   [ ] Manual Verification: Run `almd add <url>`, then inspect `almd-lock.toml` to verify the entry is added/updated with the correct source URL, path, and hash format.

-   [ ] **Task 2.9: Error Handling and Cleanup**
    -   [ ] Review the `add` command's `Action` logic.
    -   [ ] Implement error handling using `urfave/cli`'s error reporting (e.g., `cli.Exit`).
    -   [ ] If an error occurs *after* downloading the file but *before* successfully updating both manifest and lockfile, attempt to delete the downloaded file to maintain consistency.
    -   [ ] Ensure clear error messages are provided to the user.
    -   [ ] Manual Verification: Test error scenarios: invalid URL, download failure, write permission errors for manifest/lockfile, simulate failures mid-process to check cleanup.

## Milestone 3: Initial E2E Testing Setup (Placeholder)

**Goal:** Establish the basic structure for E2E tests for the `init` and `add` commands. (Detailed test cases TBD).

-   [ ] **Task 3.1: Define E2E Testing Strategy**
    -   [ ] Determine the framework (standard Go `testing` package likely sufficient).
    -   [ ] Decide on approach:
        -   Running `almd` as a subprocess (`os/exec`) within test functions.
        -   Directly calling command `Action` functions (may require refactoring for testability/dependency injection).
    -   [ ] Plan for handling `init` interactivity (e.g., using input redirection, pre-made config files, or potentially a future non-interactive flag).
    -   [ ] Plan for handling `add` network calls (e.g., using a mock HTTP server like `net/http/httptest`, or hitting real endpoints for specific test cases).
    -   [ ] Define setup/teardown logic (creating temporary directories, cleaning up generated files like `project.toml`, `almd-lock.toml`, downloaded libs).
    -   [ ] Manual Verification: Review the chosen strategy for feasibility.

-   [ ] **Task 3.2: Create Test File Structure**
    -   [ ] Create test files (e.g., `commands/init_test.go`, `commands/add_test.go` or a dedicated `test/e2e/` directory).
    -   [ ] Implement basic setup/teardown helpers based on the chosen strategy.
    -   [ ] Manual Verification: Run `go test ./...` and confirm the test files are picked up and basic setup/teardown executes without error.

-   [ ] **Task 3.3: Implement Basic `init` Test Case**
    -   [ ] Add a simple test case that runs `almd init` (using the chosen strategy) in a temporary directory.
    -   [ ] Verify that `project.toml` is created.
    -   [ ] (Optional) Perform basic checks on the default content of `project.toml`.
    -   [ ] Manual Verification: Run the specific test and confirm it passes and cleans up correctly.

-   [ ] **Task 3.4: Implement Basic `add` Test Case**
    -   [ ] Add a simple test case that runs `almd add <test_url>` (using the chosen strategy, potentially with a mock server) in a temporary directory containing a minimal `project.toml`.
    -   [ ] Verify the dependency file is downloaded to the expected location.
    -   [ ] Verify `project.toml` is updated correctly.
    -   [ ] Verify `almd-lock.toml` is created/updated correctly.
    -   [ ] Manual Verification: Run the specific test and confirm it passes and cleans up correctly.

## Milestone 4: Initial E2E Testing Setup (Placeholder)

**Goal:** Establish the basic structure for E2E tests for the `init` and `add` commands. (Detailed test cases TBD).

-   [ ] **Task 4.1: Define E2E Testing Strategy**
    -   [ ] Determine the framework (standard Go `testing` package likely sufficient).
    -   [ ] Decide on approach:
        -   Running `almd` as a subprocess (`os/exec`) within test functions.
        -   Directly calling command `Action` functions (may require refactoring for testability/dependency injection).
    -   [ ] Plan for handling `init` interactivity (e.g., using input redirection, pre-made config files, or potentially a future non-interactive flag).
    -   [ ] Plan for handling `add` network calls (e.g., using a mock HTTP server like `net/http/httptest`, or hitting real endpoints for specific test cases).
    -   [ ] Define setup/teardown logic (creating temporary directories, cleaning up generated files like `project.toml`, `almd-lock.toml`, downloaded libs).
    -   [ ] Manual Verification: Review the chosen strategy for feasibility.

-   [ ] **Task 4.2: Create Test File Structure**
    -   [ ] Create test files (e.g., `commands/init_test.go`, `commands/add_test.go` or a dedicated `test/e2e/` directory).
    -   [ ] Implement basic setup/teardown helpers based on the chosen strategy.
    -   [ ] Manual Verification: Run `go test ./...` and confirm the test files are picked up and basic setup/teardown executes without error.

-   [ ] **Task 4.3: Implement Basic `init` Test Case**
    -   [ ] Add a simple test case that runs `almd init` (using the chosen strategy) in a temporary directory.
    -   [ ] Verify that `project.toml` is created.
    -   [ ] (Optional) Perform basic checks on the default content of `project.toml`.
    -   [ ] Manual Verification: Run the specific test and confirm it passes and cleans up correctly.

-   [ ] **Task 4.4: Implement Basic `add` Test Case**
    -   [ ] Add a simple test case that runs `almd add <test_url>` (using the chosen strategy, potentially with a mock server) in a temporary directory containing a minimal `project.toml`.
    -   [ ] Verify the dependency file is downloaded to the expected location.
    -   [ ] Verify `project.toml` is updated correctly.
    -   [ ] Verify `almd-lock.toml` is created/updated correctly.
    -   [ ] Manual Verification: Run the specific test and confirm it passes and cleans up correctly.

---

*This `TASKS.md` outlines the implementation of the core `init` and `add` commands and the initial setup for E2E testing. Further tasks can be added for other commands (`remove`, `install`, `run`), additional features, refactoring, and more comprehensive testing.* 