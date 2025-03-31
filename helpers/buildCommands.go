package helpers

import (
	"context"
	"log"
	"time"

	"github.com/gen2brain/beeep"
)

func RunCommand(ctx context.Context, command string) error {
	//dry run
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		log.Println(command)
		time.Sleep(1 * time.Second)
		return nil
	}
}

func GetBuildCommand(pkg NodePackage, webappPath string) []string {
	if pkg.Strategy == "TRANSPILED" {
		return []string{
			"yarn transpile",
			"rm -rf " + webappPath + "/node_modules/" + pkg.PackageJson.Name + "/dist",
			"cp -R " + pkg.Path + "/dist " + webappPath + "/node_modules/" + pkg.PackageJson.Name,
		}
	}
	if pkg.Strategy == "AMEND_NATIVE" {
		return []string{
			"rm -rf " + webappPath + "/node_modules/@mediatool/mt-utils/lib",
			"rm -rf " + webappPath + "/node_modules/@mediatool/mt-utils/boundaries",
			"rm -rf " + webappPath + "/node_modules/@mediatool/mt-utils/amend",
			"cp -R lib " + webappPath + "/node_modules/@mediatool/mt-utils",
			"cp -R boundaries " + webappPath + "/node_modules/@mediatool/mt-utils",
			"cp -R amend " + webappPath + "/node_modules/@mediatool/mt-utils",
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
		err := RunCommand(ctx, command)
		if err != nil {
			return err
		}
	}
	beeep.Notify("Build completed", pkg.PackageJson.Name+" build completed in "+time.Since(startTime).String(), "")
	return nil
}
