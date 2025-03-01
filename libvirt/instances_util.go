package libvirt

import (
	"fmt"

	"github.com/ajquack/libvirt-cloud-controller-manager/internal/metrics"
	"libvirt.org/go/libvirt"
)

func getDomainByName(l *libvirt.Connect, name string) (*libvirt.Domain, error) {
	const op = "libvirt/getDomainByName"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	dom, err := l.LookupDomainByName(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return dom, nil
}

func getDomainByUUID(l *libvirt.Connect, uuid string) (*libvirt.Domain, error) {
	const op = "libvirt/getDomainByUUID"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	dom, err := l.LookupDomainByUUIDString(uuid)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return dom, nil
}
