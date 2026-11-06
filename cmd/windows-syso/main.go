// Command windows-syso builds the Windows resource object (icon, manifest and
// version info) used by the desktop executable. It mirrors the small part of
// the Wails CLI needed for packaging, without requiring Wails build tooling on
// CI or cross-compilation hosts.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

func main() {
	arch := flag.String("arch", "amd64", "target architecture: amd64, arm64 or 386")
	icon := flag.String("icon", "", "path to the .ico file")
	manifest := flag.String("manifest", "", "path to the application manifest")
	info := flag.String("info", "", "path to the version info JSON")
	out := flag.String("out", "", "output .syso path")
	flag.Parse()

	if *icon == "" || *manifest == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "-icon, -manifest and -out are required")
		os.Exit(2)
	}
	target, ok := map[string]winres.Arch{
		"amd64": winres.ArchAMD64,
		"arm64": winres.ArchARM64,
		"386":   winres.ArchI386,
	}[*arch]
	if !ok {
		fmt.Fprintf(os.Stderr, "unsupported arch %q\n", *arch)
		os.Exit(2)
	}

	resourceSet := winres.ResourceSet{}

	iconFile, err := os.Open(*icon)
	if err != nil {
		fatal(err)
	}
	defer iconFile.Close()
	ico, err := winres.LoadICO(iconFile)
	if err != nil {
		fatal(fmt.Errorf("load icon: %w", err))
	}
	if err := resourceSet.SetIcon(winres.RT_ICON, ico); err != nil {
		fatal(err)
	}

	manifestData, err := os.ReadFile(*manifest)
	if err != nil {
		fatal(err)
	}
	appManifest, err := winres.AppManifestFromXML(manifestData)
	if err != nil {
		fatal(fmt.Errorf("parse manifest: %w", err))
	}
	resourceSet.SetManifest(appManifest)

	if *info != "" {
		infoData, err := os.ReadFile(*info)
		if err != nil {
			fatal(err)
		}
		if len(infoData) > 0 {
			var info version.Info
			if err := info.UnmarshalJSON(infoData); err != nil {
				fatal(fmt.Errorf("parse version info: %w", err))
			}
			resourceSet.SetVersionInfo(info)
		}
	}

	output, err := os.Create(*out)
	if err != nil {
		fatal(err)
	}
	defer output.Close()
	if err := resourceSet.WriteObject(output, target); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
