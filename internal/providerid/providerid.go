package providerid

import (
	"fmt"
	"strings"
)

const (
	prefixLibvirt = "libvirt://"
)

type unknownPrefixErr struct {
	ProviderID string
}

func (e *unknownPrefixErr) Error() string {
	return fmt.Sprintf("unknown provider ID prefix: %q", e.ProviderID)
}

func ToDomainID(providerID string) (string, error) {
	uuid := ""
	switch {
	case strings.HasPrefix(providerID, prefixLibvirt):
		uuid = strings.ReplaceAll(providerID, prefixLibvirt, "")
	default:
		return "", &unknownPrefixErr{ProviderID: providerID}

	}
	return uuid, nil
}

func FromDomainID(domainID string) string {
	return fmt.Sprintf("%s%s", prefixLibvirt, domainID)
}
