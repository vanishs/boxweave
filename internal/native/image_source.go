package native

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

// Only a trusted Incus simplestreams catalog or an existing local alias may be
// selected. The MCP caller cannot choose a server, path, fingerprint or URL.
const defaultCatalogURL = "https://images.linuxcontainers.org"
const defaultCatalogAlias = "ubuntu/24.04"

var imageAliasPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$`)

func validImageAlias(name string) bool {
	if !imageAliasPattern.MatchString(name) {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "." || part == ".." || part == "" {
			return false
		}
	}
	return true
}

// ValidImageChoice deliberately permits no user-provided URL or foreign
// Incus remote. A missing choice uses native operator aliases or the catalog.
func ValidImageChoice(choice string) bool {
	if choice == "" {
		return true
	}
	name := strings.TrimPrefix(strings.TrimPrefix(choice, "local:"), "images:")
	return validImageAlias(name) && !strings.Contains(name, ":")
}

func catalogImageSource(alias string) api.InstanceSource {
	return api.InstanceSource{Type: "image", Protocol: "simplestreams", Server: defaultCatalogURL, Alias: alias}
}

func localImageSource(c incus.InstanceServer, name string) (api.InstanceSource, bool, error) {
	alias, _, err := c.GetImageAlias(name)
	if api.StatusErrorCheck(err, http.StatusNotFound) {
		return api.InstanceSource{}, false, nil
	}
	if err != nil {
		return api.InstanceSource{}, false, err
	}
	if alias == nil || alias.Target == "" {
		return api.InstanceSource{}, false, ErrNativeUnsupported
	}
	image, _, err := c.GetImage(alias.Target)
	if err != nil || image == nil || image.Type != "container" {
		return api.InstanceSource{}, false, ErrNativeUnsupported
	}
	return api.InstanceSource{Type: "image", Fingerprint: image.Fingerprint}, true, nil
}

func resolveInstanceImage(c incus.InstanceServer, choice string) (api.InstanceSource, error) {
	if !ValidImageChoice(choice) {
		return api.InstanceSource{}, ErrNativeUnsupported
	}
	if strings.HasPrefix(choice, "images:") {
		return catalogImageSource(strings.TrimPrefix(choice, "images:")), nil
	}
	if choice != "" {
		name := strings.TrimPrefix(choice, "local:")
		source, found, err := localImageSource(c, name)
		if err != nil {
			return api.InstanceSource{}, err
		}
		if !found {
			return api.InstanceSource{}, errors.New("selected local image alias not found in the confined Incus Project")
		}
		return source, nil
	}
	for _, name := range []string{"boxweave-default", "boxweave-ubuntu-24.04"} {
		source, found, err := localImageSource(c, name)
		if err != nil {
			return api.InstanceSource{}, err
		}
		if found {
			return source, nil
		}
	}
	return catalogImageSource(defaultCatalogAlias), nil
}
