package update

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

// NewUpdateCommand creates a new cli.Command for the "update" command.
func NewUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Updates project dependencies based on project.toml",
		ArgsUsage: "[dependency_names...]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Force update even if versions appear to match",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose output",
			},
		},
		Action: func(c *cli.Context) error {
			_, _ = fmt.Fprintf(os.Stdout, "Update command called.\n")
			if c.NArg() > 0 {
				_, _ = fmt.Fprintf(os.Stdout, "Dependencies to update: %s\n", c.Args().Slice())
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "Updating all dependencies.\n")
			}
			if c.Bool("force") {
				_, _ = fmt.Fprintf(os.Stdout, "Force flag is set.\n")
			}
			if c.Bool("verbose") {
				_, _ = fmt.Fprintf(os.Stdout, "Verbose flag is set.\n")
			}
			return nil
		},
	}
}
