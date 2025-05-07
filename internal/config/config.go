package config

import (
	"bytes"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/nightconcept/almandine-go/internal/project" // Corrected module path
)

// WriteProjectToml marshals the Project data and writes it to the specified filePath.
// It will overwrite the file if it already exists.
func WriteProjectToml(filePath string, data *project.Project) error {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(data); err != nil {
		return err
	}

	// Write the TOML content to the file, overwriting if it exists.
	// Create the file if it doesn't exist, with default permissions (0666 before umask).
	// O_TRUNC ensures that if the file exists, its content is truncated.
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(buf.Bytes())
	return err
}
