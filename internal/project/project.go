package project

// Project represents the overall structure of the project.toml file.
type Project struct {
	Package      *PackageInfo      `toml:"package"`
	Scripts      map[string]string `toml:"scripts,omitempty"`
	Dependencies map[string]string `toml:"dependencies,omitempty"`
}

// PackageInfo holds metadata for the project.
type PackageInfo struct {
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	License     string `toml:"license,omitempty"`
	Description string `toml:"description,omitempty"`
}
