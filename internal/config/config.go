package config

import (
	"errors"
	"fmt"

	utils "github.com/ajquack/libvirt-cloud-controller-manager/internal/utils"
)

const (
	libvirtURI   = "LIBVIRT_URI"
	libvirtDebug = "LIBVIRT_DEBUG"

	libvirtMetricsEnabled = "LIBVIRT_METRICS_ENABLED"
	libvirtMetricsAddress = ":8233"
)

type LibvirtClientConfig struct {
	LibvirtURI string
	Debug      bool
}

type MetricsConfig struct {
	Enabled bool
	Address string
}

type LCCMConfiguration struct {
	LibvirtClient LibvirtClientConfig
	Metrics       MetricsConfig
}

func Read() (LCCMConfiguration, error) {
	var err error
	var errs []error
	var config LCCMConfiguration

	config.LibvirtClient.LibvirtURI, err = utils.LookupEnv(libvirtURI)
	if err != nil {
		errs = append(errs, err)
	}
	config.LibvirtClient.Debug, err = utils.GetBool(libvirtDebug, false)
	if err != nil {
		errs = append(errs, err)
	}
	config.Metrics.Enabled, err = utils.GetBool(libvirtMetricsEnabled, false)
	if err != nil {
		errs = append(errs, err)
	}
	config.Metrics.Address = libvirtMetricsAddress

	if len(errs) > 0 {
		// Return the first error
		return LCCMConfiguration{}, errors.Join(errs...)
	}
	return config, nil
}

func (c LCCMConfiguration) Validate() (err error) {
	var errs []error
	if c.LibvirtClient.LibvirtURI == "" {
		errs = append(errs, fmt.Errorf("environment variable %s is required", libvirtURI))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
