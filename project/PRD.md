Almandine Package Manager (Go Implementation) - PRD

1. Introduction

Almandine (almd as the CLI command) is a lightweight package manager implemented in Go for Lua projects. It enables simple, direct management of single-file Lua dependencies (from GitHub or other supported repositories), project scripts, and project metadata. Almandine is designed for projects that want to pin specific versions or commits of files without managing complex dependency trees.

2. Core Features

    Single-file Downloads: Fetch individual Lua files from remote repositories (e.g., GitHub), pinning by semver (if available via tags) or git commit hash.
    No Dependency Tree Management: Only downloads files explicitly listed in the project; does not resolve or manage full Lua dependency trees.
    Project Metadata: Maintains project name, type, version, license, and package description in almd.toml.
    Script Runner: Provides a central point for running project scripts (similar to npm scripts) defined in almd.toml.
    Lockfile: Tracks exact versions or commit hashes of all downloaded files for reproducible builds (almd-lock.toml).
    License & Description: Exposes license and package description fields in almd.toml for clarity and compliance.
    Cross-Platform: Built as a cross-platform Go binary (Linux, macOS, Windows).

2.1. Core Commands (Initial Focus)

    (New) add command:
        Goal: Adds a single-file Lua dependency from a supported source (initially GitHub URLs) to the project.
        Functionality:
            Parses the provided URL (e.g., GitHub file link).
            Uses Go's standard HTTP client to download the specified file (handling raw content URLs).
            Saves the downloaded file to the configured target directory (default: lib/, overrideable with -d).
            Updates the almd.toml file, adding or modifying the dependency entry under the [dependencies] table. The key will be the derived filename or a name specified via -n. The value will be the source URL or a simplified representation (e.g., github:user/repo/path/file.lua@commit_hash).
            Updates the almd-lock.toml file with the resolved details: path within the project, exact source URL used, and a hash (either the commit hash if specified in the URL, or a calculated sha256 hash of the downloaded content otherwise).
        Arguments:
            <url>: The URL to the file (required).
            -d <directory> or --directory <directory>: Specifies a target directory relative to the project root (e.g., src/engine/lib). Defaults to lib/.
            -n <name> or --name <name>: Specifies the name to use for the dependency in almd.toml and as the base filename (without extension). Defaults to the filename derived from the URL.

3. Folder Structure (Go Project)

Sample minimal structure for an Almandine-managed Lua project:

    almd.toml # Project manifest (metadata, scripts, dependencies)
    almd-lock.toml # Lockfile (exact versions/hashes of dependencies)
    scripts/ # (Optional) Project scripts referenced in almd.toml
    lib/ # (Optional) Default directory for downloaded Lua packages/files
    src/ # (Optional) Project Lua source code

Structure of the Almandine Go project itself:

    almd.toml # Manifest for the Lua project being managed
    almd-lock.toml # Lockfile for the Lua project being managed
    cmd/almd/ # Main application package for the almd CLI
        main.go # Entry point, using Cobra (github.com/spf13/cobra) for CLI structure and parsing
    internal/ # Internal Go packages (not intended for external use)
        config/ # Logic for parsing almd.toml and almd-lock.toml
        downloader/ # Logic for fetching files from URLs
        lockfile/ # Logic for managing the lock file
        commands/ # Cobra command implementations (add, init, etc.)
    pkg/ # Library packages intended for external use (if any)
    go.mod # Go module definition
    go.sum # Go module checksums
    scripts/ # Build or utility scripts for the Go project itself (optional)
    testdata/ # Files used during testing (optional)

4. File Descriptions
almd.toml

Project manifest (TOML format). Example fields:

# Project metadata
name = "my-lua-project"
lua = ">=5.1" # Informational: Minimum or specific Lua version for the project
type = "library" # or "application"
version = "1.0.0"
license = "MIT"
description = "A sample Lua project using Almandine."

# Project scripts
[scripts]
test = "lua tests/run.lua"
build = "lua build.lua"

# Project dependencies (Lua files)
[dependencies]
# Example entry after `almd add <url> -n lunajson` (if semver parsing added later)
# lunajson = "~1.3.4" # Semver support TBD

# Example entry after `almd add https://github.com/user/repo/path/file.lua@abcdef`
file = "github:user/repo/path/file.lua@abcdef"

# Example entry after `almd add <url> -n other -d src/otherlib`
other = { source = "github:user/repo/some/other.lua@main", path = "src/otherlib/other.lua" }

    name (string): Project name.
    lua (string, optional): Informational field indicating the target Lua version or range (e.g., ">=5.1", "=5.1"). Almandine itself doesn't enforce this but uses it for metadata.
    type (string): Project type, either "library" or "application".
    version (string): Project version.
    license (string): Project license.
    description (string): Project description.
    [scripts] (table): Project scripts. Keys are script names, values are the commands to run.
    [dependencies] (table): Project dependencies. Keys are dependency names, values can be strings (source URLs/identifiers) or tables for more complex definitions (e.g., specifying a custom path).

almd-lock.toml

Tracks resolved dependencies for reproducible installs (TOML format). Example fields:

# Lockfile format version
api_version = "1"

# Locked packages
[package]
  # Entry corresponding to almd.toml's ["file"] example above
  [package.file]
  source = "github:user/repo/path/file.lua@abcdef" # The exact source identifier
  path = "lib/file.lua"                            # Relative path within the project
  hash_type = "commit"                             # Type of hash ("commit" or "sha256")
  hash = "abcdef"                                  # The commit hash

  # Entry corresponding to almd.toml's ["other"] example above
  [package.other]
  source = "github:user/repo/some/other.lua@main" # The exact source identifier used
  path = "src/otherlib/other.lua"                 # Custom relative path
  hash_type = "sha256"                            # Type of hash
  hash = "sha256:..."                             # sha256 hash of the downloaded content

  # Example if semver resolution was implemented
  # [package.lunajson]
  # version = "1.3.4"
  # source = "github:user/repo/lunajson.lua@v1.3.4" # Resolved source URL/tag
  # path = "lib/lunajson.lua"
  # hash_type = "sha256"
  # hash = "sha256:..."

Go Project Structure (cmd/, internal/, pkg/)

    cmd/almd/main.go: Main entrypoint for the CLI. Responsible for:
        Setting up and executing the root Cobra command.
        Cobra handles argument parsing, dispatching to subcommands, and flag handling.
        Handling command aliases via Cobra's alias feature.
        Generating usage/help output using Cobra's built-in help commands. All output must use almd as the tool name.
    internal/: Contains Go packages crucial for Almandine's operation but not intended for reuse by other Go projects. This includes command logic (defined as Cobra commands), configuration parsing, file downloading, lockfile management, etc.
    pkg/: (Optional) Contains Go packages designed for potential reuse by other Go projects. Might be empty initially.

Installation

Installation involves standard Go practices:

    Build: go build ./cmd/almd creates an almd executable in the current directory.
    Install: go install ./cmd/almd builds and installs the almd executable into $GOPATH/bin or $GOBIN. Ensure this directory is in the system's PATH.
    Cross-Compilation: Use Go's cross-compilation features (setting GOOS and GOARCH environment variables) to build binaries for Linux, macOS, and Windows from a single machine.

5. Conclusion

Almandine, implemented in Go using the Cobra CLI framework, aims to provide a simple, robust, and reproducible workflow for Lua projects that need lightweight dependency management and script automation, without the complexity of full dependency trees.
Tech Stack

    Implementation Language: Go (>= 1.18 for generics, or specify latest stable)
    CLI Framework: Cobra (github.com/spf13/cobra)
    Managed Language: Lua (Targeting versions 5.1–5.4 / LuaJIT 2.1 compatibility for managed packages)
    Configuration Format: TOML
    Platform: Cross-platform Go binaries (Linux, macOS, Windows)

Project-Specific Coding Rules (Go)

These rules supplement any global AI project guidelines. They define standards and practices unique to this Go project.

    Language, Environment & Dependencies
        Target Language: Go (specify version, e.g., 1.21+).
        Dependencies: Use Go Modules (go.mod, go.sum) for managing Go dependencies. Minimize external dependencies, preferring the Go standard library where possible (e.g., net/http, os, io, encoding/json, crypto/sha256). Required external dependencies include github.com/spf13/cobra for the CLI and a TOML parsing library (e.g., github.com/BurntSushi/toml).
        Environment: Code should run correctly on Linux, macOS, and Windows. Use platform-agnostic Go APIs (e.g., path/filepath instead of hardcoding / or \).

    Go Coding Standards
        Formatting: Strictly adhere to gofmt. Code must pass go vet and staticcheck (or similar linters) without errors.
        Style: Follow principles outlined in Effective Go.
        Naming Conventions: Use camelCase for variables and function names (starting lowercase for unexported, uppercase for exported). Use PascalCase for types and interfaces. Keep names short but descriptive. Package names should be lowercase and concise.
        Error Handling: Use explicit error checking (if err != nil). Errors should be handled or wrapped with additional context (using fmt.Errorf with %w or the errors package). Avoid panicking except for truly unrecoverable situations during initialization. Cobra commands should return errors to be handled by the main execution flow.
        Concurrency: Use goroutines and channels appropriately if concurrency is needed. Ensure proper synchronization to avoid race conditions (use go run -race during testing).
        Package Design: Keep packages focused and cohesive. Use internal packages (internal/) for code not meant to be imported by other projects. Exported symbols (PascalCase) should have clear purposes and stable APIs. Command logic should be encapsulated within functions called by the Cobra command's RunE (or similar) function.

    Documentation & Comments (Go)
        Doc Comments: Every exported (public) function, type, interface, constant, and variable requires documentation comments (// Comment...). The first sentence should be a concise summary. Cobra commands (cobra.Command structs) require Short and Long descriptions.
        Package Comments: Each package should have a package comment (// package mypackage ...) describing its purpose.
        Clarity: Explain the why not just the what in comments where necessary. Document non-obvious logic.
        Examples: Provide runnable examples using Go's Example function convention where appropriate, especially for library code (pkg/). Cobra command examples can be added to the Example field.

    Testing & Behavior Specification (Go)
        Framework: Use Go's built-in testing package (go test).
        Test File Location: Test files must be named *_test.go and reside in the same package as the code they test.
        Test Types:
            Unit Tests: Focus on testing individual functions and methods in isolation. Use table-driven tests where appropriate. Minimize reliance on external systems or the filesystem; use interfaces and test doubles (mocks/stubs) if needed. Test helper functions used by Cobra commands.
            Integration Tests: Test interactions between different components or packages within the Almandine tool itself. May involve limited filesystem interaction within controlled test environments.
            End-to-End (E2E) Tests: Verify core user flows by executing the compiled almd binary as a subprocess. These tests should simulate user interactions (passing arguments that Cobra will parse) and validate outcomes (file creation/content, almd.toml/almd-lock.toml state, exit codes, stdout/stderr).
        E2E Test Sandboxing & Scaffolding:
            E2E tests must run in isolated, temporary directories created using t.TempDir() (available in Go 1.15+) or io/ioutil.TempDir (older versions) within the test function.
            A helper package (e.g., internal/testutil or within the specific E2E test file) should provide functions to:
                Set up a temporary project directory with necessary initial files (e.g., a basic almd.toml).
                Run the almd command (compiled beforehand or via go run) using os/exec targeting the sandboxed project directory. Capture stdout, stderr, and exit codes.
                Assert file existence, file content (os.ReadFile), TOML file content (parsing the TOML within the test), and command output/exit status.
                Cleanup is usually handled automatically by t.TempDir().
        Scenario Coverage: Use t.Run for subtests to organize tests by scenario (happy path, boundary conditions, error handling). E2E tests should cover the primary CLI commands and flags as defined in the Cobra command structure.
        Test Execution: Run tests with go test ./.... Use the -race flag frequently (go test -race ./...) to detect race conditions. Use -cover (go test -cover ./...) to monitor test coverage.

4.1. Example E2E Specification Scenarios (add command)

The following outlines E2E test cases required for the add command, to be implemented in a relevant *_test.go file (e.g., internal/commands/add_test.go or a dedicated e2e/add_test.go) using Go's testing package and the sandboxing helper. These tests will execute almd add ... with various arguments parsed by Cobra:

    Test adding a valid GitHub file URL (almd add <url>). Verify:
        The Lua file is downloaded to the default lib/ directory.
        almd.toml contains the new dependency entry with the correct name and source.
        almd-lock.toml contains the corresponding entry with path, source, hash_type (commit or sha256), and the correct hash value.
        Command exits successfully (exit code 0).
    Test adding a valid GitHub file URL with a specific commit hash (almd add <url@hash>). Verify:
        File downloaded correctly.
        almd.toml entry reflects the URL with the hash.
        almd-lock.toml entry uses hash_type = "commit" and the correct commit hash.
    Test adding a file using the -d flag (almd add <url> -d src/deps). Verify:
        File is downloaded to the specified directory (src/deps/).
        almd.toml entry may or may not reflect the path (depending on design decision).
        almd-lock.toml entry shows the correct custom path.
    Test adding a file using the -n flag (almd add <url> -n mydep). Verify:
        The downloaded file is named mydep.lua.
        almd.toml uses mydep as the dependency key.
        almd-lock.toml uses mydep as the package key and the correct filename in the path.
    Test adding a file using both -n and -d (almd add <url> -n mydep -d src/deps). Verify combined effects.
    Test adding a duplicate dependency (should potentially update or warn/error, depending on desired behavior).
    Test adding an invalid URL (almd add invalid-url). Verify:
        No file is downloaded.
        almd.toml and almd-lock.toml are unchanged.
        Command exits with a non-zero exit code and prints an informative error message to stderr.
    Test adding a file when almd.toml doesn't exist (should likely error, prompting almd init first).
    Test adding a file with network issues (simulate via test setup if possible, or ensure robust error handling).