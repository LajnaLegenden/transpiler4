package cli

import (
	"context"
	"fmt"
	"sync"

	"github.com/LajnaLegenden/transpiler4/helpers"
	"github.com/urfave/cli/v2"
)

// BuildCommand returns the CLI command for the build operation
func BuildCommand() *cli.Command {
	return &cli.Command{
		Name:    "build",
		Aliases: []string{"b"},
		Usage:   "Build and copy this project once",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "Path to the project folder",
			},
		},
		Action: BuildAction,
	}
}

// BuildAction handles the build command execution
func BuildAction(c *cli.Context) error {
	fmt.Println("We are building and copying this project once")
	projectPath, err := helpers.GetProjectPath(c.String("path"))
	webappPath := projectPath + "/webapp"
	if err != nil {
		return err
	}
	fmt.Printf("Using project path: %s\n", projectPath)
	packages, err := helpers.FindNodePackages(projectPath)
	if err != nil {
		return err
	}
	buildablePackages := helpers.GetBuildablePackages(packages)
	selectedPackages := helpers.SelectPackages(buildablePackages)
	var wg sync.WaitGroup
	for _, pkg := range selectedPackages {
		wg.Add(1)
		go func(pkg helpers.NodePackage) {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			err := helpers.BuildPackage(ctx, pkg, webappPath)
			if err != nil {
				fmt.Printf("Error building package: %s\n", err)
			}
		}(pkg)
	}
	wg.Wait()
	return nil
}
