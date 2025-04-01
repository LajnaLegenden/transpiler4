package helpers

import (
	"context"
	"log"
	"os/exec"
	"time"

	"github.com/gen2brain/beeep"
)

func RunCommand(ctx context.Context, command string, path string) error {
	//dry run
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		cmd := exec.Command("sh", "-c", command)
		cmd.Dir = path
		cmd.Stdout = log.Writer()
		cmd.Stderr = log.Writer()

		err := cmd.Run()
		if err != nil {
			if err.Error() != "context: canceled" {
				beeep.Notify("Running command failed", err.Error(), "")
				return err
			}
		}
		return nil
	}
}

// RunCommandWithLogger runs a command with a custom logger
func RunCommandWithLogger(ctx context.Context, command string, path string, logger *log.Logger) error {
	//dry run
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		cmd := exec.Command("sh", "-c", command)
		cmd.Dir = path
		cmd.Stdout = logger.Writer()
		cmd.Stderr = logger.Writer()

		err := cmd.Run()
		if err != nil {
			if err.Error() != "context: canceled" {
				beeep.Notify("Running command failed", err.Error(), "")
				return err
			}
		}
		return nil
	}
}

func GetBuildCommand(pkg NodePackage, webappPath string) []string {
	if pkg.Strategy == "TRANSPILED_YARN" {
		return []string{
			"yarn transpile",
			"rm -rf " + webappPath + "/node_modules/" + pkg.PackageJson.Name + "/dist",
			"cp -R " + pkg.Path + "/dist " + webappPath + "/node_modules/" + pkg.PackageJson.Name,
		}
	}
	if pkg.Strategy == "TRANSPILED" {
		return []string{
			"pnpm transpile",
			"rm -rf " + webappPath + "/node_modules/" + pkg.PackageJson.Name + "/dist",
			"cp -R " + pkg.Path + "/dist " + webappPath + "/node_modules/" + pkg.PackageJson.Name,
		}
	}
	if pkg.Strategy == "AMEND_NATIVE" {
		commands := []string{}

		// Remove old folders first
		commands = append(commands,
			"rm -rf "+webappPath+"/node_modules/"+pkg.PackageJson.Name+"/lib",
			"rm -rf "+webappPath+"/node_modules/"+pkg.PackageJson.Name+"/boundaries",
			"rm -rf "+webappPath+"/node_modules/"+pkg.PackageJson.Name+"/amend")

		// Only copy folders if they exist
		if pkg.FolderItems["lib"] {
			commands = append(commands, "cp -R lib "+webappPath+"/node_modules/"+pkg.PackageJson.Name)
		}
		if pkg.FolderItems["boundaries"] {
			commands = append(commands, "cp -R boundaries "+webappPath+"/node_modules/"+pkg.PackageJson.Name)
		}
		if pkg.FolderItems["amend"] {
			commands = append(commands, "cp -R amend "+webappPath+"/node_modules/"+pkg.PackageJson.Name)
		}

		return commands
	}
	if pkg.Strategy == "MAKEFILE_BUILD" {
		return []string{
			"make build",
		}
	}
	if pkg.Strategy == "TRANSPILED_LEGACY" {
		return []string{
			"pnpm prepublishOnly",
			"rm -rf " + webappPath + "/node_modules/" + pkg.PackageJson.Name + "/dist",
			"cp -R " + pkg.Path + "/dist " + webappPath + "/node_modules/" + pkg.PackageJson.Name,
		}
	}
	return []string{}
}

func BuildPackage(ctx context.Context, pkg NodePackage, webappPath string) error {
	commands := GetBuildCommand(pkg, webappPath)
	//Store start time
	startTime := time.Now()
	SendNotification("Build started", pkg.PackageJson.Name+" build started")
	for _, command := range commands {
		err := RunCommand(ctx, command, pkg.Path)
		if err != nil {
			return err
		}
	}
	SendNotification("Build completed", pkg.PackageJson.Name+" completed in "+time.Since(startTime).String())
	return nil
}

// BuildPackageWithLogger builds a package using a custom logger
func BuildPackageWithLogger(ctx context.Context, pkg NodePackage, webappPath string, logger *log.Logger) error {
	commands := GetBuildCommand(pkg, webappPath)
	//Store start time
	startTime := time.Now()
	SendNotification("Build started", pkg.PackageJson.Name+" build started")

	for _, command := range commands {
		err := RunCommandWithLogger(ctx, command, pkg.Path, logger)
		if err != nil {
			return err
		}
	}

	duration := time.Since(startTime).String()
	SendNotification("Build completed", pkg.PackageJson.Name+" completed in "+duration)
	return nil
}
