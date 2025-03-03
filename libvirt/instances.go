package libvirt

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	cloudprovider "k8s.io/cloud-provider"

	"github.com/ajquack/libvirt-cloud-controller-manager/internal/config"
	"github.com/ajquack/libvirt-cloud-controller-manager/internal/metrics"
	"github.com/ajquack/libvirt-cloud-controller-manager/internal/providerid"
	"libvirt.org/go/libvirt"
)

type instances struct {
	client   *libvirt.Connect
	recorder record.EventRecorder
	config   config.LCCMConfiguration
}

type libvirtDomain struct {
	*libvirt.Domain
}

type genericServer interface {
	IsShutdown() (bool, error)
	Metadata(cfg config.LCCMConfiguration) (*cloudprovider.InstanceMetadata, error)
}

var (
	errDomainNotFound = errors.New("domain not found")
)

const (
	ProvidedBy = "instance.libvirt.local/provided-by"
)

func newInstances(client *libvirt.Connect, recorder record.EventRecorder, config config.LCCMConfiguration) *instances {
	return &instances{
		client:   client,
		recorder: recorder,
		config:   config,
	}
}

func (i *instances) lookupDomain(node *corev1.Node) (genericServer, error) {
	if node.Spec.ProviderID != "" {
		var domainUUID string
		domainUUID, err := providerid.ToDomainID(node.Spec.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("failed to convert provider id to domain id: %w", err)
		}
		domain, err := getDomainByUUID(i.client, domainUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get domain \"%s\": %w", domainUUID, err)
		}
		return libvirtDomain{domain}, nil
	}
	domainByName, err := getDomainByName(i.client, node.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain \"%s\": %w", node.Name, err)
	}
	switch {
	case domainByName != nil:
		return libvirtDomain{domainByName}, nil
	default:
		return nil, nil
	}
}

func domainNodeAdresses(d *libvirt.Domain) ([]corev1.NodeAddress, error) {
	var addresses []corev1.NodeAddress
	domainName, err := d.GetName()
	if err != nil {
		return nil, fmt.Errorf("failed to get domain name: %w", err)
	}
	addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeHostName, Address: domainName})

	interfaces, err := d.ListAllInterfaceAddresses(1) // 1 means use the guest agent
	if err != nil {
		return nil, fmt.Errorf("failed to get interface addresses: %w", err)
	}

	// No interfaces found
	if len(interfaces) == 0 {
		return nil, fmt.Errorf("no network interfaces found")
	}

	// Strategy: Look for the first non-loopback interface with an IPv4 address
	for _, iface := range interfaces {
		// Skip loopback interfaces
		if iface.Name == "lo" {
			continue
		}

		// Look for IPv4 addresses
		for _, addr := range iface.Addrs {
			if addr.Type == libvirt.IP_ADDR_TYPE_IPV4 {
				addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: addr.Addr})
			}
		}
	}

	// If no IPv4 found, try IPv6
	for _, iface := range interfaces {
		if iface.Name == "lo" {
			continue
		}

		for _, addr := range iface.Addrs {
			if addr.Type == libvirt.IP_ADDR_TYPE_IPV6 {
				addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: addr.Addr})
			}
		}
	}
	return addresses, nil
}

func (i *instances) InstanceExists(ctx context.Context, node *corev1.Node) (bool, error) {
	const op = "libvirt/instancesv2.InstanceExists"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	domain, err := i.lookupDomain(node)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return domain != nil, nil
}

func (i *instances) InstanceShutdown(ctx context.Context, node *corev1.Node) (bool, error) {
	const op = "libvirt/instancesv2.InstanceShutdown"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	domain, err := i.lookupDomain(node)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if domain == nil {
		return false, fmt.Errorf("%s: failed to get instance metadata: no matching server found for node '%s': %w",
			op, node.Name, errDomainNotFound)
	}

	isShutdown, err := domain.IsShutdown()
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return isShutdown, nil
}

func (s libvirtDomain) IsShutdown() (bool, error) {
	state, _, err := s.GetState()
	if err != nil {
		return false, fmt.Errorf("failed to get domain state: %w", err)
	}
	return state == libvirt.DOMAIN_SHUTOFF, nil
}

func (s libvirtDomain) Metadata(cfg config.LCCMConfiguration) (*cloudprovider.InstanceMetadata, error) {
	uuid, err := s.GetUUIDString()
	if err != nil {
		return nil, fmt.Errorf("failed to get domain UUID: %w", err)
	}
	domainNodeAdresses, err := domainNodeAdresses(s.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain node addresses: %w", err)
	}
	domcon, err := s.DomainGetConnect()
	if err != nil {
		return nil, fmt.Errorf("failed to get domain connection: %w", err)
	}
	nodeName, err := domcon.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	return &cloudprovider.InstanceMetadata{
		ProviderID:    providerid.FromDomainID(uuid),
		InstanceType:  generateDomainType(*s.Domain),
		NodeAddresses: domainNodeAdresses,
		Zone:          nodeName,
		Region:        nodeName,
		AdditionalLabels: map[string]string{
			ProvidedBy: "libvirt",
		},
	}, nil
}

func (i *instances) InstanceMetadata(ctx context.Context, node *corev1.Node) (*cloudprovider.InstanceMetadata, error) {
	const op = "libvirt/instancesv2.InstanceMetadata"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	server, err := i.lookupDomain(node)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if server == nil {
		return nil, fmt.Errorf(
			"%s: failed to get instance metadata: no matching server found for node '%s': %w",
			op, node.Name, errDomainNotFound)
	}

	metadata, err := server.Metadata(i.config)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return metadata, nil
}
