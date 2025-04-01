package cli

import (
	"fmt"
	"strings"

	"github.com/LajnaLegenden/transpiler4/helpers"
	"github.com/urfave/cli/v2"
)

// ListCommand returns the CLI command for the list operation
func ListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"l"},
		Usage:   "List all projects in the specified directory",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "Path to the directory to list projects from",
			},
		},
		Action: ListAction,
	}
}

// ListAction handles the list command execution
func ListAction(c *cli.Context) error {
	fmt.Println("We are listing all projects in the specified directory")
	projectPath, err := helpers.GetProjectPath(c.String("path"))
	if err != nil {
		return err
	}
	projects, err := helpers.FindNodePackages(projectPath)
	if err != nil {
		return err
	}
	buildablePackages := helpers.GetBuildablePackages(projects)
	fmt.Printf("%-40s %-20s %-10s\n", "Package Name", "Strategy", "Frontend")
	fmt.Println(strings.Repeat("-", 70))
	for _, pkg := range buildablePackages {
		fmt.Printf("%-40s %-20s %-10v\n",
			pkg.PackageJson.Name,
			pkg.Strategy,
			pkg.IsFrontend)
	}
	return nil
}
