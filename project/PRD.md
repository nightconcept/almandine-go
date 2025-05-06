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
    -   **Implementation:** Implemented as a `urfave/cli` command (e.g., in `commands/init.go`).
    -   **Functionality:**
        -   **Interactive Prompts:** Prompts the user for core project metadata:
            -   `package` name (defaults to `my-go-project` or derived from current dir).
            -   `version` (defaults to `0.1.0`).
            -   `license` (defaults to `MIT`).
            -   `description` (defaults to `A sample Lua project using Almandine.`).
            -   Optionally prompts for `language` details (e.g., name `go`, version `>= 1.18`). *Note: Defaulting might be sufficient initially.*
        -   **Script Definition:** Interactively prompts the user to add scripts ([`scripts`] table).
            -   Prompts for script `name` and `command`.
            -   Continues prompting until an empty name is entered.
            -   **Default `run` script:** If no `run` script is provided by the user, adds a default (e.g., `go run .` or `go run main.go`).
        -   **Dependency Placeholders (Simplified):** Interactively prompts the user to add initial dependencies ([`dependencies`] table).
            -   Prompts for dependency `name` and a simple `source/version` string.
            -   Continues prompting until an empty name is entered.
            -   **Note:** `init` creates basic dependency entries (e.g., `dep_name = "<user_input_string>"` or stores it in the `source` field of a sub-table). The `add` command is responsible for fleshing these out into the full structure (`source` identifier, `path`).
        -   **Manifest Creation/Overwrite:** Creates `project.toml` in the current directory with the collected information. If `project.toml` already exists, it **overwrites** the existing file (as per the Lua spec behavior).
        -   **Output:** Prints confirmation message upon successful creation/overwrite. Reports errors clearly via `urfave/cli` if file writing fails.
    -   **Arguments & Flags (`urfave/cli`):**
        -   Typically run without arguments or flags, relying purely on interactive input. (Consider adding `--force` later if overwrite confirmation is desired).

-   **`add` command:**
    -   **Goal:** Adds a single-file dependency from a supported source (initially GitHub URLs) to the project.
    -   **Implementation:** Implemented as a `urfave/cli` command, likely defined in a dedicated file (e.g., `commands/add.go` or similar).
    -   **Functionality:**
        -   Parses the provided URL using Go's `net/url`.
        -   **GitHub URL Processing:** Handles various GitHub URL formats (blob, raw, potentially commit/tag specific). Normalizes the input URL to the **raw content download URL**. Extracts the commit hash if present in the URL. Creates a **canonical source identifier** (e.g., `github:user/repo/path/to/file@commit_hash_or_ref`) for storage in `project.toml`. This logic should be encapsulated (e.g., in `internal/source`).
        -   Downloads the specified file using Go's `net/http` client from the resolved raw content URL.
        -   **Target Directory/Path Handling:** Determines the final destination path for the downloaded file based on the `-d` and `-n` flags. Ensures the target directory exists, creating it if necessary using `os.MkdirAll`.
        -   Saves the downloaded file to the determined target path using `os` and `io` packages.
        -   **Manifest Update (`project.toml`):** Updates the `project.toml` file (using a Go TOML library like `github.com/BurntSushi/toml`). Adds or modifies the dependency entry under the `[dependencies]` table. The key is the derived or specified dependency name (`-n` flag). The value **must be a sub-table** containing:
            -   `source`: The **canonical source identifier** derived from the input URL.
            -   `path`: The relative path (using forward slashes) from the project root to the downloaded file.
        -   **Lockfile Update (`almd-lock.toml`):** Updates the `almd-lock.toml` file (using a TOML library). Adds or updates the entry for the dependency under the `[package]` table. This entry stores:
            -   `source`: The **exact raw download URL** used to fetch the file.
            -   `path`: The relative path (using forward slashes) matching the manifest.
            -   `hash`: A string representing the file's integrity. Format: `commit:<commit_hash>` if a commit was extracted from the URL, otherwise `sha256:<sha256_hash>` calculated from the downloaded content using `crypto/sha256`. Define how hash calculation errors are represented (e.g., `hash_error:<reason>`).
        -   **Error Handling & Atomicity:** Implement robust error handling. If the process fails after download but before saving manifest/lockfile, attempt to clean up the downloaded file to avoid inconsistent states. Report errors clearly via `urfave/cli` (e.g., using `cli.Exit`).
    -   **Arguments & Flags (`urfave/cli`):**
        -   `<source_url>`: Argument accessed from `*cli.Context` for the source URL of the file (required).
        -   `-d, --directory string`: Flag definition (`cli.StringFlag`) for specifying the target. If the path ends in a separator or points to an existing directory, the file is saved inside that directory using the name derived from the `-n` flag or the URL. Otherwise, the flag value is treated as the full relative path for the saved file. Defaults to saving within the `src/lib/` directory.
        -   `-n, --name string`: Flag definition (`cli.StringFlag`) for specifying the logical name of the dependency (used as the key in `project.toml` and `almd-lock.toml`) and the base filename (without extension, defaults to `.lua` initially but should adapt). If omitted, the name is inferred from the URL's filename component.
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

## 3. Almandine Tool Project Structure (Go Implementation)

Standard Go project layout combined with Almandine specifics:

-   `project.toml`       # Default project manifest filename (can be overridden via flag?)
-   `almd-lock.toml`     # Default lockfile filename
-   `go.mod`               # Go module definition
-   `go.sum`               # Go module checksums
-   `main.go`              # Main application entry point (configures and runs the `urfave/cli` App)
-   `commands/`            # `urfave/cli` command definitions (optional structure, could be flat in `main.go` for simple cases)
    -   `add.go`           # `almd add` command logic/definition
    -   `...`              # Other commands (e.g., `init.go`, `remove.go`)
-   `internal/`            # Internal Go packages (not intended for import by other projects)
    -   `config/`          # Loading/parsing/updating `project.toml`
    -   `lockfile/`        # Loading/parsing/updating `almd-lock.toml`
    -   `downloader/`      # File downloading logic
    -   `hasher/`          # Content hashing logic (sha256)
    -   `project/`         # Go structs representing project/lockfile data models
    -   `source/`          # Handling source URL parsing, normalization, identifier creation (e.g., GitHub logic)
    -   `util/`            # General utility functions (e.g., path manipulation)
    -   `...`              # Other internal helpers
-   `scripts/`             # (Optional) Project scripts referenced in `project.toml`
-   `lib/`                 # (Optional) Default directory for downloaded dependencies

## 3.1 Example Lua Project Structure (Using Almandine)

This shows a typical layout for a Lua project managed by the Almandine tool:

-   `project.toml`       # Defines project metadata, scripts, and dependencies for Almandine
-   `almd-lock.toml`  # Stores locked dependency versions/hashes generated by Almandine

-   `src/`                 # Lua project source code
    -   `main.lua`         # Example entry point for the Lua project
    -   `my_module.lua`
    -   `lib/`                 # Default directory where Almandine downloads dependencies
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
lua = "5.1"

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

# Entry for 'lua-cjson', locked from project.toml entry
[package."lua-cjson"]
  # The *exact* raw download URL used to fetch this version
  source = "https://raw.githubusercontent.com/user/repo/tag-2.1.0/lua-cjson.lua"
  # Relative path within the project (forward slashes)
  path = "src/lib/lua-cjson.lua"
  # Integrity hash: commit hash if available from source URL, otherwise sha256 of content
  hash = "sha256:deadbeef..." # Example sha256 hash
  # Example of a commit hash from URL:
  # hash = "commit:abcdef123..."

# Entry for 'plenary', locked from project.toml entry
[package."plenary"]
  source = "https://raw.githubusercontent.com/nvim-lua/plenary.nvim/v0.1.4/some/file.lua"
  path = "src/vendor/plenary.lua"
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

### `commands/`

Contains Go files defining the `