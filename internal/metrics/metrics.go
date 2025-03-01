package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/component-base/metrics/legacyregistry"

	// Initialize cloud-provider internal metrics (e.g. workqueue).
	_ "k8s.io/component-base/metrics/prometheus/clientgo"
	"k8s.io/klog/v2"
)

const (
	readTimeout    = 5 * time.Second
	requestTimeout = 10 * time.Second
	writeTimeout   = 20 * time.Second
)

var (
	OperationCalled = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cloud_controller_manager_operations_total",
		Help: "The total number of operation was called",
	}, []string{"op"})
)

func init() {
	GetRegistry().MustRegister(OperationCalled)
}

func GetRegistry() prometheus.Registerer {
	return legacyregistry.Registerer()
}

func GetHandler() http.Handler {
	return legacyregistry.Handler()
}

func Serve(address string) {
	// The metrics are also served by k8s.io/cloud-provider on the secure serving port.
	mux := http.NewServeMux()
	mux.Handle("/metrics", GetHandler())

	server := &http.Server{
		Addr:         address,
		Handler:      http.TimeoutHandler(mux, requestTimeout, "timeout"),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	klog.Info("Starting metrics server at ", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		klog.ErrorS(err, "create metrics service")
	}
}
