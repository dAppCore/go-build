package build

import (
	"embed"

	coreerr "dappco.re/go/log"
)

//go:embed images/*.yml
var linuxKitBaseTemplateFS embed.FS

// LinuxKitBaseImage describes a built-in immutable image template.
type LinuxKitBaseImage struct {
	Name            string
	Description     string
	Version         string
	DefaultPackages []string
}

var linuxKitBaseCatalog = []LinuxKitBaseImage{
	{
		Name:            "core-dev",
		Description:     "Go toolchain, git, task, core CLI, linters",
		Version:         "2026.04.08",
		DefaultPackages: []string{"bash", "git", "go", "openssh-client", "task", "wget"},
	},
	{
		Name:            "core-ml",
		Description:     "Go toolchain, ML runtimes, model loaders",
		Version:         "2026.04.08",
		DefaultPackages: []string{"bash", "git", "go", "python3", "py3-pip", "wget"},
	},
	{
		Name:            "core-minimal",
		Description:     "Go toolchain only",
		Version:         "2026.04.08",
		DefaultPackages: []string{"go"},
	},
}

// LinuxKitBaseImages returns the built-in immutable image templates.
func LinuxKitBaseImages() []LinuxKitBaseImage {
	result := make([]LinuxKitBaseImage, len(linuxKitBaseCatalog))
	for i, image := range linuxKitBaseCatalog {
		result[i] = LinuxKitBaseImage{
			Name:            image.Name,
			Description:     image.Description,
			Version:         image.Version,
			DefaultPackages: append([]string(nil), image.DefaultPackages...),
		}
	}
	return result
}

// LookupLinuxKitBaseImage resolves a built-in immutable image template.
func LookupLinuxKitBaseImage(name string) (LinuxKitBaseImage, bool) {
	for _, image := range linuxKitBaseCatalog {
		if image.Name == name {
			return LinuxKitBaseImage{
				Name:            image.Name,
				Description:     image.Description,
				Version:         image.Version,
				DefaultPackages: append([]string(nil), image.DefaultPackages...),
			}, true
		}
	}
	return LinuxKitBaseImage{}, false
}

// LinuxKitBaseTemplate loads the built-in LinuxKit template for a named base image.
func LinuxKitBaseTemplate(name string) (string, error) {
	if _, ok := LookupLinuxKitBaseImage(name); !ok {
		return "", coreerr.E("build.LinuxKitBaseTemplate", "unknown LinuxKit image base: "+name, nil)
	}

	content, err := linuxKitBaseTemplateFS.ReadFile("images/" + name + ".yml")
	if err != nil {
		return "", coreerr.E("build.LinuxKitBaseTemplate", "failed to read embedded LinuxKit template", err)
	}

	return string(content), nil
}
