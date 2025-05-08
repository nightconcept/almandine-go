// Title: Almandine CLI Application Entry Point
// Purpose: Initializes and runs the Almandine command-line interface application,
// defining its commands and default behavior.
package main

// Import the "fmt" package, which provides functions for formatted I/O
// (like printing to the console).
import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/nightconcept/almandine-go/internal/cli/add"
	"github.com/nightconcept/almandine-go/internal/cli/initcmd"
	"github.com/nightconcept/almandine-go/internal/cli/remove"
)

// The main function, where the program execution begins.
func main() {
	app := &cli.App{
		Name:    "almd",
		Usage:   "A simple project manager for single-file dependencies",
		Version: "v0.0.1", // Placeholder version
		Action: func(c *cli.Context) error {
			// Default action if no command is specified
			_ = cli.ShowAppHelp(c)
			return nil
		},
		Commands: []*cli.Command{
			initcmd.GetInitCommand(),
			add.AddCommand,
			remove.RemoveCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
