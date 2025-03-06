package libvirtops

import (
	"context"

	"github.com/ajquack/libvirt-cloud-controller-manager/internal/config"
	"k8s.io/client-go/tools/record"
)

type LibvirLoadBalancerClient interface {
	Create(ctx context.Context, opts LoadBalancerCreateOpts) (LoadBalancerCreateResult, error)
	Update(ctx context.Context, opts LoadBalancerUpdateOpts) (LoadBalancerUpdateResult, error)
	Delete(ctx context.Context, lb *LibvirtLoadBalancerOps) (LoadBalancerDeleteResult, error)
}

type LibvirtLoadBalancerOps struct {
	LBClient  LibvirLoadBalancerClient
	Config    config.LCCMConfiguration
	NetworkID string
	Recorder  record.EventRecorder
}

type LibvirtLoadBalancer struct {
}

func (l *LibvirtLoadBalancerOps) Create(ctx context.Context, lbName string) (*LibvirtLoadBalancer, error) {
	return nil, nil
}
