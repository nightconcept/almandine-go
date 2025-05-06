// Declare the package name. The main package is special in Go,
// it's where the program execution starts.
package main

// Import the "fmt" package, which provides functions for formatted I/O
// (like printing to the console).
import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/nightconcept/almandine-go/commands"
)

// The main function, where the program execution begins.
func main() {
	app := &cli.App{
		Name:    "almd",
		Usage:   "A simple project manager for single-file dependencies",
		Version: "v0.0.1", // Placeholder version
		Action: func(c *cli.Context) error {
			// Default action if no command is specified
			cli.ShowAppHelp(c)
			return nil
		},
		Commands: []*cli.Command{
			commands.GetInitCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
