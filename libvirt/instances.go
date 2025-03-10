package libvirt

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"

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

	// Strategy: Look for the first non-loopback interface with an IP address
	for _, iface := range interfaces {
		// Skip loopback interfaces
		if iface.Name == "lo" {
			continue
		}

		// Look for IP addresses
		for _, addr := range iface.Addrs {
			if addr.Type == libvirt.IP_ADDR_TYPE_IPV4 {
				addresses = append(addresses, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: addr.Addr})
				return addresses, nil // Return only the first IP address found
			}
		}
	}

	return nil, fmt.Errorf("no valid IP addresses found")
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

func (s libvirtDomain) Metadata(config config.LCCMConfiguration) (*cloudprovider.InstanceMetadata, error) {
	if config.General.Debug {
		domainName, _ := s.GetName()
		klog.Infof("libvirt/Metadata: Getting metadata for domain %s", domainName)
	}

	uuid, err := s.GetUUIDString()
	if err != nil {
		if config.General.Debug {
			klog.Errorf("libvirt/Metadata: Failed to get domain UUID: %v", err)
		}
		return nil, fmt.Errorf("failed to get domain UUID: %w", err)
	}

	if config.General.Debug {
		klog.Infof("libvirt/Metadata: Got domain UUID: %s", uuid)
	}

	domainNodeAdresses, err := domainNodeAdresses(s.Domain)
	if err != nil {
		if config.General.Debug {
			klog.Errorf("libvirt/Metadata: Failed to get domain node addresses: %v", err)
		}
		return nil, fmt.Errorf("failed to get domain node addresses: %w", err)
	}

	if config.General.Debug {
		klog.Infof("libvirt/Metadata: Got %d node addresses", len(domainNodeAdresses))
		for _, addr := range domainNodeAdresses {
			klog.Infof("  - %s: %s", addr.Type, addr.Address)
		}
	}

	domcon, err := s.DomainGetConnect()
	if err != nil {
		if config.General.Debug {
			klog.Errorf("libvirt/Metadata: Failed to get domain connection: %v", err)
		}
		return nil, fmt.Errorf("failed to get domain connection: %w", err)
	}

	if config.General.Debug {
		klog.Infof("libvirt/Metadata: Got domain connection")
	}

	nodeName, err := domcon.GetHostname()
	if err != nil {
		if config.General.Debug {
			klog.Errorf("libvirt/Metadata: Failed to get hostname: %v", err)
		}
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}
	if config.General.Debug {
		klog.Infof("libvirt/Metadata: Got hostname: %s", nodeName)
	}

	domainType, err := generateDomainType(*s.Domain)
	if err != nil {
		if config.General.Debug {
			klog.Errorf("libvirt/Metadata: Failed to generate domain type: %v", err)
		}
		return nil, fmt.Errorf("failed to generate domain type: %w", err)
	}

	return &cloudprovider.InstanceMetadata{
		ProviderID:    providerid.FromDomainID(uuid),
		InstanceType:  domainType,
		NodeAddresses: domainNodeAdresses,
		Zone:          nodeName,
		Region:        nodeName,
	}, nil
}

func (i *instances) InstanceMetadata(ctx context.Context, node *corev1.Node) (*cloudprovider.InstanceMetadata, error) {
	const op = "libvirt/instancesv2.InstanceMetadata"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	// Add debug log at the beginning
	if i.config.General.Debug {
		klog.Infof("%s: Getting metadata for node %s (provider ID: %s)", op, node.Name, node.Spec.ProviderID)
	}

	server, err := i.lookupDomain(node)
	if err != nil {
		klog.Errorf("%s: Failed to lookup domain for node %s: %v", op, node.Name, err)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if server == nil {
		klog.Errorf("%s: No matching domain found for node %s", op, node.Name)
		return nil, fmt.Errorf(
			"%s: failed to get instance metadata: no matching server found for node '%s': %w",
			op, node.Name, errDomainNotFound)
	}

	if i.config.General.Debug {
		klog.Infof("%s: Found domain for node %s, retrieving metadata", op, node.Name)
	}

	metadata, err := server.Metadata(i.config)
	if err != nil {
		klog.Errorf("%s: Failed to get metadata for node %s: %v", op, node.Name, err)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if i.config.General.Debug {
		klog.Infof("%s: Retrieved metadata for node %s:", op, node.Name)
		klog.Infof("  - ProviderID: %s", metadata.ProviderID)
		klog.Infof("  - InstanceType: %s", metadata.InstanceType)
		klog.Infof("  - Zone: %s", metadata.Zone)
		klog.Infof("  - Region: %s", metadata.Region)
		klog.Infof("  - NodeAddresses:")
		for _, addr := range metadata.NodeAddresses {
			klog.Infof("    - %s: %s", addr.Type, addr.Address)
		}
		klog.Infof("  - Labels: %v", metadata.AdditionalLabels)
	}

	return metadata, nil
}
