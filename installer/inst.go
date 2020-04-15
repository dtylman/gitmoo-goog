package main

import (
	"fmt"
	"os"

	"github.com/dtylman/gitmoo-goog/version"
	"github.com/dtylman/gopack/deb"
)

var options struct {
	Name     string
	Arch     string
	Branch   string
	BuildNum string
	MediaBin string
}

func init() {
	options.Name = "gitmoo-goog"
	options.Branch = os.Getenv("CIRCLE_BRANCH")
	options.BuildNum = os.Getenv("CIRCLE_BUILD_NUM")
	options.Arch = "amd64"
}

func createDeb() error {
	if options.Branch != "" && options.Branch != "master" {
		version.Version += "-" + options.Branch
	}

	d, err := deb.New(options.Name, version.Version, options.BuildNum, options.Arch)
	if err != nil {
		return err
	}

	d.Info.Maintainer = "dtylman@gmail.com"
	d.Info.Section = "base"
	d.Info.Homepage = "https://github.com/dtylman/gitmoo-goog"
	d.Info.Description = `A Google Photos backup tool. `
	files := map[string]string{
		"../gitmoo-goog":      "/usr/local/bin/gitmoo-goog",
		"gitmoo-goog.service": "/etc/systemd/system/gitmoo-goog.service",
	}

	for source, target := range files {
		err = d.AddFile(source, target)
		if err != nil {
			return err
		}
	}

	debFileName, err := d.Create("")
	fmt.Println("Created " + debFileName)
	return err
}

func main() {
	err := createDeb()
	if err != nil {
		fmt.Println("Error creating deb: " + err.Error())
	}
}
