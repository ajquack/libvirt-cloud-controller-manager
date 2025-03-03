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

func generateDomainType(d libvirt.Domain) (string, error) {
	const op = "libvirt/generateDomainType"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	domainVCPU, err := d.GetVcpus()
	if err != nil {
		return "", fmt.Errorf("%s: %v", op, err)
	}
	vcpus := len(domainVCPU)

	domainMemory, err := d.GetMaxMemory()
	if err != nil {
		return "", fmt.Errorf("%s: %v", op, err)
	}
	memory := int64(domainMemory) / (1024 * 1024)

	return fmt.Sprintf("%dCPU%dRAM", vcpus, memory), nil
}
