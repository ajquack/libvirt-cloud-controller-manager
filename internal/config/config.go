package config

import (
	"errors"
	"fmt"

	utils "github.com/ajquack/libvirt-cloud-controller-manager/internal/utils"
)

const (
	LibvirtURI   = "LIBVIRT_URI"
	LibvirtDebug = "LIBVIRT_DEBUG"

	MetricsEnabled = "METRICS_ENABLED"
	MetricsAddress = ":8233"

	GeneralDebug = "DEBUG"
)

type LibvirtClientConfig struct {
	LibvirtURI string
	Debug      string
}

type MetricsConfig struct {
	Enabled bool
	Address string
}

type GeneralConfig struct {
	Debug bool
}

type LCCMConfiguration struct {
	LibvirtClient LibvirtClientConfig
	Metrics       MetricsConfig
	General       GeneralConfig
}

func Read() (LCCMConfiguration, error) {
	var err error
	var errs []error
	var config LCCMConfiguration

	config.LibvirtClient.LibvirtURI, err = utils.LookupEnv(LibvirtURI)
	if err != nil {
		errs = append(errs, err)
	}
	config.LibvirtClient.Debug, err = utils.LookupEnv(LibvirtDebug)
	if err != nil {
		errs = append(errs, err)
	}
	config.Metrics.Enabled, err = utils.GetBool(MetricsEnabled, false)
	if err != nil {
		errs = append(errs, err)
	}
	config.Metrics.Address = MetricsAddress
	config.General.Debug, err = utils.GetBool(GeneralDebug, false)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		// Return the first error
		return LCCMConfiguration{}, errors.Join(errs...)
	}
	return config, nil
}

func (c LCCMConfiguration) Validate() (err error) {
	var errs []error
	if c.LibvirtClient.LibvirtURI == "" {
		errs = append(errs, fmt.Errorf("environment variable %s is required", LibvirtURI))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
