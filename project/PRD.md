# Almandine Package Manager (Go Version) - PRD

## 1. Introduction

Almandine (`almd` as the CLI command) is a lightweight package manager for Go projects, migrating the core concepts from the original Lua version. It enables simple, direct management of single-file dependencies (initially from GitHub), project scripts, and project metadata. Almandine is designed for projects that want to pin specific versions or commits of files without managing complex dependency trees, leveraging Go's strengths and the `urfave/cli` framework.

## 2. Core Features

-   **Single-file Downloads:** Fetch individual files (initially Lua, but adaptable) from remote repositories (e.g., GitHub), pinning by git commit hash or potentially tags/versions later.
-   **No Dependency Tree Management:** Only downloads files explicitly listed in the project; does not resolve or manage full dependency trees.
-   **Project Metadata:** Maintains project name, type, version, license, and package description in `project.toml` (using TOML format).
-   **Script Runner:** Provides a central point for running project scripts defined in `project.toml`.
-   **Lockfile:** Tracks exact versions or commit hashes of all downloaded files for reproducible builds (`almd-lock.toml`, using TOML format).
-   **License & Description:** Exposes license and package description fields in `project.toml`.
-   **Cross-Platform:** Built as a standard Go binary, naturally cross-platform (Linux, macOS, Windows).

### 2.1. Core Commands (Initial Focus using `urfave/cli`)

-   **`init` command:**
    -   **Goal:** Interactively initialize a new Almandine project by creating a `project.toml` manifest file in the current directory.
    -   **Implementation:** Implemented as a `urfave/cli` command (in `internal/cli/initcmd/initcmd.go`).
    -   **Functionality:**
        -   **Interactive Prompts:** Prompts the user for core project metadata:
            -   `package` name (defaults to `my-go-project` or derived from current dir).
            -   `version` (defaults to `0.1.0`).
            -   `license` (defaults to `MIT`).
            -   `description` (defaults to `A sample Go project using Almandine.`).
            -   Optionally prompts for `language` details (e.g., name `go`, version `>= 1.21`).
        -   **Script Definition:** Interactively prompts the user to add scripts ([`scripts`] table).
            -   Prompts for script `name` and `command`.
            -   Continues prompting until an empty name is entered.
            -   **Default `run` script:** If no `run` script is provided by the user, adds a default (e.g., `go run ./cmd/almd` or a user-project specific default like `go run .`).
        -   **Dependency Placeholders (Simplified):** Interactively prompts the user to add initial dependencies ([`dependencies`] table).
            -   Prompts for dependency `name` and a simple `source/version` string.
            -   Continues prompting until an empty name is entered.
            -   **Note:** `init` creates basic dependency entries. The `add` command is responsible for fleshing these out into the full structure (`source` identifier, `path`).
        -   **Manifest Creation/Overwrite:** Creates `project.toml` in the current directory with the collected information. If `project.toml` already exists, it **overwrites** the existing file.
        -   **Output:** Prints confirmation message upon successful creation/overwrite. Reports errors clearly via `urfave/cli` if file writing fails.
    -   **Arguments & Flags (`urfave/cli`):**
        -   Typically run without arguments or flags, relying purely on interactive input.

-   **`add` command:**
    -   **Goal:** Adds a single-file dependency from a supported source (initially GitHub URLs) to the project.
    -   **Implementation:** Implemented as a `urfave/cli` command (defined in `internal/cli/add/add.go`).
    -   **Functionality:**
        -   Parses the provided URL using Go's `net/url`.
        -   **GitHub URL Processing:** Handles various GitHub URL formats (blob, raw, potentially commit/tag specific). Normalizes the input URL to the **raw content download URL**. Extracts the commit hash if present in the URL. Creates a **canonical source identifier** (e.g., `github:user/repo/path/to/file@commit_hash_or_ref`) for storage in `project.toml`. This logic is encapsulated in `internal/source`.
        -   Downloads the specified file using Go's `net/http` client from the resolved raw content URL (logic in `internal/downloader`).
        -   **Target Directory/Path Handling:** Determines the final destination path for the downloaded file based on the `-d` and `-n` flags. Ensures the target directory exists, creating it if necessary using `os.MkdirAll`.
        -   Saves the downloaded file to the determined target path using `os` and `io` packages.
        -   **Manifest Update (`project.toml`):** Updates the `project.toml` file (using a Go TOML library like `github.com/BurntSushi/toml` and logic in `internal/core/config`). Adds or modifies the dependency entry under the `[dependencies]` table. The key is the derived or specified dependency name (`-n` flag). The value **must be a sub-table** containing:
            -   `source`: The **canonical source identifier** derived from the input URL.
            -   `path`: The relative path (using forward slashes) from the project root to the downloaded file.
        -   **Lockfile Update (`almd-lock.toml`):** Updates the `almd-lock.toml` file (using a TOML library and logic in `internal/core/lockfile`). Adds or updates the entry for the dependency under the `[package]` table. This entry stores:
            -   `source`: The **exact raw download URL** used to fetch the file.
            -   `path`: The relative path (using forward slashes) matching the manifest.
            -   `hash`: A string representing the file's integrity. Format: `commit:<commit_hash>` if a commit was extracted from the URL, otherwise `sha256:<sha256_hash>` calculated from the downloaded content using `crypto/sha256` (logic in `internal/hasher`). Defines how hash calculation errors are represented (e.g., `hash_error:<reason>`).
        -   **Error Handling & Atomicity:** Implement robust error handling. If the process fails after download but before saving manifest/lockfile, attempts to clean up the downloaded file. Reports errors clearly via `urfave/cli` (e.g., using `cli.Exit`).
    -   **Arguments & Flags (`urfave/cli`):**
        -   `<source_url>`: Argument accessed from `*cli.Context` for the source URL of the file (required).
        -   `-d, --directory string`: Flag definition (`cli.StringFlag`) for specifying the target directory. If the path ends in a separator or points to an existing directory, the file is saved inside that directory using the name derived from the `-n` flag or the URL. Otherwise, the flag value is treated as the full relative path for the saved file. Defaults to saving within the `libs/` directory (or `src/lib/` if specified).
        -   `-n, --name string`: Flag definition (`cli.StringFlag`) for specifying the logical name of the dependency (used as the key in `project.toml` and `almd-lock.toml`) and the base filename. If omitted, the name is inferred from the URL's filename component.
        -   `--verbose`: Optional flag (`cli.BoolFlag`) to enable detailed output during execution.

-   **`remove` command:**
    -   **Goal:** Removes a specified dependency from the project manifest (`project.toml`) and lockfile (`almd-lock.toml`), and deletes the corresponding downloaded file.
    -   **Implementation:** Implemented as a `urfave/cli` command (e.g., `commands/remove.go`).
    -   **Functionality:**
        -   **Argument Parsing:** Takes the `<dependency_name>` as a required argument from `*cli.Context`.
        -   **Manifest Loading:** Loads the `project.toml` file (using `internal/config`).
        -   **Dependency Check:** Verifies if the specified `<dependency_name>` exists under the `[dependencies]` table in the manifest.
        -   **Path Retrieval:** Retrieves the relative `path` associated with the dependency from the manifest entry.
        -   **Manifest Update:** Removes the entry corresponding to `<dependency_name>` from the `[dependencies]` table.
        -   **Manifest Saving:** Saves the modified manifest back to `project.toml`.
        -   **File Deletion:** Deletes the file specified by the retrieved `path` using `os.Remove`. Handles potential errors gracefully (e.g., file not found, permissions).
        -   **Lockfile Update:** Loads the `almd-lock.toml` file (using `internal/lockfile`), removes the corresponding entry under the `[package]` table, and saves the updated lockfile.
        -   **Output:** Prints confirmation messages for successful removal from manifest, file deletion (or warnings if deletion fails), and lockfile update. Reports errors clearly via `urfave/cli`.
    -   **Arguments & Flags (`urfave/cli`):**
        -   `<dependency_name>`: Argument accessed from `*cli.Context` for the logical name of the dependency to remove (required).
-   **`update`**
-   **`list`**
## 3. Almandine Tool Project Structure (Go Implementation)

Standard Go project layout combined with Almandine specifics:

-   `project.toml`       # Default project manifest filename for projects using Almandine
-   `almd-lock.toml`     # Default lockfile filename for projects using Almandine
-   `go.mod`               # Go module definition for Almandine tool
-   `go.sum`               # Go module checksums for Almandine tool
-   `README.md`            # Project README for Almandine development
-   `.github/`             # GitHub-specific files (workflows, issue templates, etc.)
-   `cmd/`                 # Main applications for the project
    -   `almd/`            # The Almandine CLI application (assuming CLI command is 'almd')
        -   `main.go`      # Main entry point, CLI argument parsing, command dispatch
-   `internal/`            # Private application and library code (not for external import)
    -   `cli/`             # CLI command logic and definitions
        -   `add/`         # Logic for the 'add' command
            -   `add.go`
            -   `add_test.go`
        -   `initcmd/`     # Logic for the 'init' command
            -   `initcmd.go`
            -   `initcmd_test.go`
        -   `remove/`      # Logic for the 'remove' command
            -   `remove.go`
        -   `...`          # Other command packages/modules
    -   `core/`            # Core application logic (business logic)
        -   `config/`      # Loading, parsing, and updating `project.toml`
            -   `config.go`
            -   `config_test.go`
        -   `lockfile/`    # Loading, parsing, and updating `almd-lock.toml`
            -   `lockfile.go`
            -   `lockfile_test.go`
        -   `downloader/`  # File downloading logic
            -   `downloader.go`
            -   `downloader_test.go`
        -   `hasher/`      # Content hashing logic (e.g., SHA256)
            -   `hasher.go`
            -   `hasher_test.go`
        -   `project/`     # Go structs representing project/lockfile data models
            -   `project.go`
        -   `source/`      # Handling source URL parsing, normalization, identifier creation
            -   `source.go`
            -   `source_test.go`
    -   `util/`            # General utility functions shared across internal packages
-   `pkg/`                 # Public library code, reusable by other projects (if any - initially empty)
    -   `...`              # Example: `pkg/somepublicapi/`
-   `scripts/`             # Scripts for building, installing, analyzing the Almandine tool itself (e.g., `build.sh`, `install.sh`)
-   `configs/`             # Configuration files for the Almandine tool (e.g., for different environments - placeholder)
-   `docs/`                # Almandine tool's own documentation (user guides, design documents, PRD, etc.)
-   `test/`                # Additional tests (e.g., E2E, integration) and test data
    -   `e2e/`             # End-to-end tests
    -   `data/`            # Test data, fixtures (optional)

The directory `lib/` (mentioned in the previous structure) is not part of the Almandine tool's own source code structure. It typically refers to the default output directory within a *user's project* where Almandine might download dependencies (e.g., `src/lib/` or a user-configured path).

Unit tests (e.g., `foo_test.go`) should be co-located with the Go source files they test (e.g., in the same package/directory like `internal/core/config/config_test.go`). The top-level `test/` directory is for tests that span multiple packages or require specific data/environments (e.g., end-to-end tests).

## 3.1 Example Lua Project Structure (Using Almandine)

This shows a typical layout for a Lua project managed by the Almandine tool:

-   `project.toml`       # Defines project metadata, scripts, and dependencies for Almandine
-   `almd-lock.toml`  # Stores locked dependency versions/hashes generated by Almandine

-   `src/`                 # Lua project source code
    -   `main.lua`         # Example entry point for the Lua project
    -   `my_module.lua`
    -   `lib/`                 # Default directory where Almandine downloads dependencies (user-configurable)
        -   `some_dependency.lua` # Example file downloaded by Almandine
        -   `another_lib/`        # Example library downloaded by Almandine
        -   `module.lua`
-   `scripts/`             # Optional directory for Lua scripts runnable via `almd run <script_name>`
    -   `build.lua`

## 4. File Descriptions

### `project.toml`

Project manifest in TOML format. Example structure based on user input:

```toml
# Example project.toml
package = "sample-project"
version = "0.1.0"
license = "MIT"
description = "A sample Go project using Almandine."
# language = { name = "go", version = ">=1.21" } # Example if language details are added

# Optional: Define primary source if needed for context
# [source]
# url = "https://github.com/nvim-neorocks/luarocks-stub" # Example purpose

# Dependencies section
[dependencies]
# Name inferred from URL (e.g., 'lua-cjson')
# [dependencies."lua-cjson"] # Name can be explicitly set with -n
#   source = "github:user/repo/lua-cjson.lua@tag-2.1.0" # Canonical identifier
#   path = "src/lib/lua-cjson.lua" # Relative path in project

# Dependency added with -n flag and custom path via -d
[dependencies."plenary"] # Specified via -n plenary
  source = "github:nvim-lua/plenary.nvim/some/file.lua@v0.1.4"
  path = "src/vendor/plenary.lua" # Specified via -d src/vendor/plenary.lua
  # Could potentially add other metadata like 'pin = true' if needed later

# Optional: Build or script definitions
[build]
type = "builtin" # Example build type

# Example scripts section (similar to npm scripts)
[scripts]
game = "love src/"
debug = "love src/ --console"
test = "love src/ --console --test"

```
-   Handles project metadata (name, version, etc.).
-   Defines dependencies under `[dependencies]`. Each dependency is a key (the logical name) mapping to a **sub-table** containing `source` (canonical identifier) and `path` (relative location).
-   Defines project scripts under `[scripts]`.

### `almd-lock.toml`

Tracks resolved dependencies for reproducible installs in TOML format. Example structure:

```toml
# Example almd-lock.toml
# Lockfile format version (increment if structure changes significantly)
api_version = "1"

# Package table holds all locked dependencies
[package]

# Entry for a dependency 'mylib', locked from project.toml entry
[package.mylib]
  # The *exact* raw download URL used to fetch this version
  source = "https://raw.githubusercontent.com/user/repo/v1.0.0/path/to/file.ext"
  # Relative path within the project (forward slashes)
  path = "vendor/custom/mylib" # Example, matches project.toml path
  # Integrity hash: commit hash if available from source URL, otherwise sha256 of content
  hash = "sha256:deadbeef..." # Example sha256 hash
  # Example of a commit hash from URL:
  # hash = "commit:abcdef123..."

# Entry for 'anotherdep', locked from project.toml entry
[package."anotherdep"]
  source = "https://raw.githubusercontent.com/another/repo/main/lib.lua"
  path = "libs/lib.lua" # Example, default path
  hash = "commit:abcdef123" # Example if pinned to a specific commit hash
  # Example of a hash error state:
  # hash = "hash_error:tool_not_found" # Or "hash_error:calculation_failed"

```
-   Stores the exact resolved `source` (raw download URL), `path`, and `hash` (content or commit or error state) for each dependency under the `[package]` table.
-   The `api_version` helps manage potential future format changes.

### `go.mod` & `go.sum`

Standard Go module files defining the project module path and managing Go dependencies.

### `main.go`

Main entry point for the `almd` CLI. This file initializes and configures the primary `cli.App` instance from the `urfave/cli` library. It defines global application metadata (like name, usage, version), global flags, registers all command definitions (e.g., from the `commands/` package or defined directly), and then executes the application logic by calling `app.Run(os.Args)`, which parses arguments and routes execution to the appropriate command's `Action` function.

commands/ (Go Project Command Packages)

Contains Go packages, each implementing a specific CLI command (e.g., add, init, remove). The add package would contain the Go logic for the add command.
cmd/almd/main.go (Go Project Entrypoint)

    Main entry point for the almd CLI executable.
    Responsible for:
        Parsing CLI arguments using a Go library (e.g., standard flag, cobra, urfave/cli).
        Dispatching execution to the appropriate command package in cmd/almd/commands/.
        Handling standard command aliases (e.g., install/in/ins, remove/rm/uninstall/un, update/up/upgrade, add/i, etc.) within the CLI library configuration.
        All usage/help output, documentation, and examples must use almd as the CLI tool name (never almandine).
        Updates here are required when adding/modifying commands or aliases.

Build & Distribution (Go Context)

    The install/ directory (containing Lua bootstrap scripts) is removed.
    Distribution involves building the Go project (go build ./cmd/almd) to produce a native executable (almd or almd.exe) for each target platform (Linux, macOS, Windows).
    Standard Go cross-compilation techniques will be used. Users simply download and place the appropriate binary in their PATH.

5. Conclusion

Almandine, implemented in Go, aims to provide a simple, robust, and reproducible workflow for Lua projects needing lightweight dependency management and script automation, without the complexity of full dependency trees. It leverages Go's strengths for building reliable, cross-platform CLI tools while managing Lua project structures and manifests.
Tech Stack

    Implementation Language: Go (e.g., 1.21 or later)
    Target Project Language: Lua 5.1–5.4 / LuaJIT 2.1 (though Almandine itself is Go, it can manage files for any language)
    Platform (Tool): Cross-platform executable (Linux, macOS, Windows) via Go compilation.
    Key Go Libraries (Potential):
        Standard Library: net/http, os, os/exec, path/filepath, crypto/sha256, flag or similar for CLI.
        External Go Modules:
            A TOML parser/generator library (e.g., `github.com/BurntSushi/toml`).
            A robust CLI framework (e.g., `github.com/urfave/cli/v2`).
            An assertion library for testing (e.g., `github.com/stretchr/testify/assert`).
            Possibly a Git client library (e.g., go-git/go-git) if direct Git operations are needed beyond simple HTTP downloads (not currently planned for initial features).

## 5. Project-Specific Coding Rules (Go Implementation)

These rules supplement the mandatory Global AI Project Guidelines and define standards specific to this Go project.
### 5.1 Language, Environment & Dependencies

    Target Language: Go (specify version, e.g., Go 1.21+).
    Environment: Standard Go development environment.
    Dependencies:
        Leverage the Go Standard Library extensively.
        External Go modules should be carefully chosen and documented in go.mod. Justify non-standard library dependencies. Key required external dependency: a library for parsing/generating Lua table syntax accurately.
        The compiled almd tool must have no runtime dependencies (like needing a specific interpreter installed).

### 5.2 Go Coding Standards

These standards guide Go development within this project.

    Formatting: All Go code must be formatted using gofmt (or goimports). This is non-negotiable and should be enforced by CI.
    Style: Adhere to the principles outlined in Effective Go and the Go Code Review Comments guide.
        Naming: Use CamelCase for exported identifiers and camelCase for unexported identifiers. Package names should be short, concise, and lowercase.
        Simplicity: Prefer clear, simple code over overly complex or clever solutions.
    Error Handling: Use standard Go error handling practices (if err != nil { return ..., err }). Errors should be handled or propagated explicitly. Use errors.Is, errors.As, and error wrapping (fmt.Errorf("...: %w", err)) where appropriate.
    Concurrency: Use goroutines and channels only when concurrency genuinely simplifies the problem or improves performance, and do so carefully, considering race conditions and synchronization.
    Packages: Structure code into logical, well-defined packages with clear APIs. Minimize unnecessary coupling between packages. Utilize internal packages (internal/) for code not meant to be imported by other modules.
    Documentation:
        All exported identifiers (variables, constants, functions, types, methods) must have documentation comments (// style comments preceding the declaration).
        Package comments (// package mypackage ...) should provide an overview of the package's purpose.
        Comments should explain why something is done, not just what is being done, unless the code itself is unclear. Follow godoc conventions.

### 5.3 Testing & Behavior Specification (Prototype Phase - Go Context)

These rules specify how testing and behavior specification are implemented using Go's standard testing package during the prototype phase.

    Framework: Use Go's built-in `testing` package. Test assertions use `github.com/stretchr/testify/assert`.
    Specification Location:
        Unit/Integration tests: Place test files (`*_test.go`) alongside the Go code they are testing (e.g., `internal/core/config/config_test.go` tests `internal/core/config/config.go`). Command-specific unit tests (e.g., for `init`, `add`) are located in their respective packages (e.g., `internal/cli/add/add_test.go`).
        E2E Tests (Prototype Focus): Place E2E tests in a dedicated location, such as `cmd/almd/main_e2e_test.go` or a top-level `test/e2e/` directory. For the prototype, command-specific tests that execute the CLI (like those in `internal/cli/add/add_test.go` which run an `app.Run` instance) serve as focused E2E-like tests for command behavior.
    File Naming: Test files must end with `_test.go`. Test functions must start with `Test` (e.g., `TestAddCommand_HappyPath`).
    Test Type Focus (Prototype):
        Command Unit Tests: Test individual command actions (e.g., the `Action` func of an `urfave/cli.Command`). These tests mock external dependencies like network calls (`net/http/httptest`) and operate on a temporary file system.
        E2E Tests: Verify system behavior by executing the compiled `almd` binary (or a test `cli.App` instance) against temporary project structures. Simulate user interactions via CLI arguments.
    Test Sandboxing & Scaffolding: Tests must run in isolated, temporary directories.
        Use `t.TempDir()` (available in Go 1.15+) within test functions to create sandboxed directories.
        Develop Go helper functions (e.g., within test files or a shared test utility package) to:
            Set up temporary project structures (creating directories, minimal `project.toml`).
            Run the `almd` command or its action, capturing output and errors. For `add` command tests, this includes setting up mock HTTP servers (`net/http/httptest`).
            Provide functions for asserting file existence, file content (using `os.ReadFile`), and parsing/asserting the content of the resulting `project.toml` and `almd-lock.toml` files within the sandbox.
            Cleanup is handled automatically by `t.TempDir()` or explicit `defer os.RemoveAll()`.
    Scenario Coverage: Each test function or suite (`t.Run`) should cover specific scenarios:
        Expected Behavior: Successful flows (e.g., `almd add ...` works correctly, `almd init` creates files as expected).
        Boundary Conditions: Edge cases (e.g., adding the same file twice, invalid URLs, empty inputs for `init`).
        Undesired Situations: Error handling (e.g., non-existent URLs, file system permission errors, invalid `project.toml` format). Use helper functions to assert expected error messages or exit codes.
    Test Dependencies:
        Command unit tests mock external dependencies (network, file system interactions beyond the temp dir).
        E2E-style tests will naturally depend on the internal Go packages used for parsing/validation (e.g., `internal/core/config`, `internal/core/lockfile`).