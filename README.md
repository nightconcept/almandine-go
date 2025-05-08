# Almandine – Lua Package Manager 💎

![License](https://img.shields.io/github/license/nightconcept/almandine-go)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/nightconcept/almandine-go/ci.yml)
[![Coverage Status](https://coveralls.io/repos/github/nightconcept/almandine-go/badge.svg)](https://coveralls.io/github/nightconcept/almandine-go)
![GitHub last commit](https://img.shields.io/github/last-commit/nightconcept/almandine-go)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/nightconcept/almandine-go/badge)](https://scorecard.dev/viewer/?uri=github.com/nightconcept/almandine-go)

A modern, cross-platform, developer-friendly package manager for Lua projects.
Easily manage, install, and update Lua single-file dependencies with a single CLI: `almd`.

---

## ✨ Features

- 📦 **Easy Dependency Management**: Add, remove, and update Lua single-file dependencies with simple commands.
- 🔒 **Reproducible Installs**: Lockfiles ensure consistent environments across machines.
- 🏗️ **Project Initialization**: Scaffold new Lua projects with best practices.
- 🛠️ **Cross-Platform**: Works on Linux, macOS, and Windows.

---

## Requirements

### macOS/Linux
- [Nix](https://nixos.org/)
- [devenv](https://devenv.sh/)

### Windows
- Go 1.23+
- [pre-commit](https://pre-commit.com/)
- [xc](https://github.com/joerdav/xc) task runner

_Note: These can all be installed via Scoop._

---

## 🛠️ Usage

```sh
almd init                # Create a new Lua project
almd add <package>       # Add a dependency
almd remove <package>    # Remove a dependency
almd update              # Update dependencies
almd list                # List installed dependencies
```

---

## Tasks

### lint

Run linters.

```sh
golangci-lint run
```

### build

Builds the `almd` binary.

```sh
go build -o build/almd ./cmd/almd
```

### build-win

Builds the `almd.exe` binary.

```sh
go build -o build/almd.exe ./cmd/almd
```

### test

Run tests.

```sh
go test ./...
```

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
