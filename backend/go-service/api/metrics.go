package api

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	processingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stratai_demo_processing_phase_seconds",
			Help:    "Duration of each demo processing phase.",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 12),
		},
		[]string{"phase"},
	)
	demoSizeBytes = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "stratai_demo_size_bytes",
			Help:    "Size of demos submitted for analysis.",
			Buckets: prometheus.ExponentialBuckets(1024*1024, 2, 10),
		},
	)
	processingErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stratai_demo_processing_errors_total",
			Help: "Demo processing errors by phase.",
		},
		[]string{"phase"},
	)
	activeProcessing = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "stratai_demo_processing_active",
			Help: "Number of demos currently being processed.",
		},
	)
)
