package libvirt

import (
	"fmt"

	"github.com/ajquack/libvirt-cloud-controller-manager/internal/config"
	"github.com/ajquack/libvirt-cloud-controller-manager/internal/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
	"libvirt.org/go/libvirt"
)

type cloud struct {
	client   *libvirt.Connect
	config   config.LCCMConfiguration
	recorder record.EventRecorder
	cidr     string
}

const (
	providerName = "libvirt"
)

var providerVersion = "unknown"

func NewCloud(cidr string) (cloudprovider.Interface, error) {
	const op = "libvirt/NewCloud"
	metrics.OperationCalled.WithLabelValues(op).Inc()

	config, err := config.Read()
	if err != nil {
		return nil, err
	}
	err = config.Validate()
	if err != nil {
		return nil, err
	}

	if config.Metrics.Enabled {
		go metrics.Serve(config.Metrics.Address)
	}

	if config.General.Debug {
		klog.Infof("%s: Debug mode enabled, turning on verbose logging", op)
	}

	client, err := libvirt.NewConnect(config.LibvirtClient.LibvirtURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to libvirt: %v", err)
	}

	_, err = client.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	klog.Infof("Libvirt Cloud Controller %s started\n", providerVersion)
	return &cloud{
		client: client,
		config: config,
		cidr:   cidr,
	}, nil
}

func (c *cloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
	client, _ := clientBuilder.Client("lccm-event-broadcaster")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1.EventSinkImpl{Interface: client.CoreV1().Events("")})

	go func() {
		<-stop
		eventBroadcaster.Shutdown()
	}()

	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "libvirt-cloud-controller-manager"})
	c.recorder = recorder
}

func (c *cloud) Instances() (cloudprovider.Instances, bool) {
	return nil, false
}

func (c *cloud) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return newInstances(c.client, c.recorder, c.config), true
}

func (c *cloud) ProviderName() string {
	return providerName
}

func (c *cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return nil, false
}

func (c *cloud) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

func (c *cloud) Zones() (cloudprovider.Zones, bool) {
	return nil, false
}

func (c *cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}
func (c *cloud) HasClusterID() bool {
	return false
}
