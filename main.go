// Declare the package name. The main package is special in Go,
// it's where the program execution starts.
package main

// Import the "fmt" package, which provides functions for formatted I/O
// (like printing to the console).
import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

// The main function, where the program execution begins.
func main() {
	app := &cli.App{
		Name:    "almd",
		Usage:   "A project management tool",
		Version: "v0.0.1", // Start with an initial version
		Action: func(c *cli.Context) error {
			// Default action if no command is specified
			cli.ShowAppHelp(c)
			return nil
		},
		// Define commands here later
		Commands: []*cli.Command{},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
