package settings

import "path/filepath"

// BootstrapUser prepares personal defaults from resolved host inputs. It does
// not read the host environment or create preference files. An invalid detected
// timezone leaves the container default unchanged.
func BootstrapUser(configDirectory, containerDNS, detectedTimezone string) (User, error) {
	if _, err := ParseTimezone(detectedTimezone); err != nil {
		detectedTimezone = ""
	}
	return ParseUser(RawUser{
		AuthStore:         filepath.Join(configDirectory, "clamp", "auth-projects.tsv"),
		ContainerDNS:      containerDNS,
		ContainerTimezone: detectedTimezone,
	})
}
