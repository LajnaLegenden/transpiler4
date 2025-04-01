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
		log.Println(cmd.String())

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
		logger.Println(cmd.String())

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
		return []string{
			"rm -rf " + webappPath + "/node_modules/" + pkg.PackageJson.Name + "/lib",
			"rm -rf " + webappPath + "/node_modules/" + pkg.PackageJson.Name + "/boundaries",
			"rm -rf " + webappPath + "/node_modules/" + pkg.PackageJson.Name + "/amend",
			"cp -R lib " + webappPath + "/node_modules/" + pkg.PackageJson.Name,
			"cp -R boundaries " + webappPath + "/node_modules/" + pkg.PackageJson.Name,
			"cp -R amend " + webappPath + "/node_modules/" + pkg.PackageJson.Name,
		}
	}
	if pkg.Strategy == "MAKEFILE_BUILD" {
		return []string{
			"make build",
		}
	}
	if pkg.Strategy == "TRANSPILED_LEGACY" {
		beeep.Notify("Build failed", "TRANSPILED_LEGACY is not supported yet", "")
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
	beeep.Notify("Build started", pkg.PackageJson.Name+" build started", "")
	for _, command := range commands {
		err := RunCommand(ctx, command, pkg.Path)
		if err != nil {
			return err
		}
	}
	beeep.Notify("Build completed", pkg.PackageJson.Name+" completed in "+time.Since(startTime).String(), "")
	return nil
}

// BuildPackageWithLogger builds a package using a custom logger
func BuildPackageWithLogger(ctx context.Context, pkg NodePackage, webappPath string, logger *log.Logger) error {
	commands := GetBuildCommand(pkg, webappPath)
	//Store start time
	startTime := time.Now()
	beeep.Notify("Build started", pkg.PackageJson.Name+" build started", "")
	logger.Printf("Build started for package %s", pkg.PackageJson.Name)

	for _, command := range commands {
		err := RunCommandWithLogger(ctx, command, pkg.Path, logger)
		if err != nil {
			return err
		}
	}

	duration := time.Since(startTime).String()
	beeep.Notify("Build completed", pkg.PackageJson.Name+" completed in "+duration, "")
	logger.Printf("Build completed for package %s in %s", pkg.PackageJson.Name, duration)
	return nil
}
