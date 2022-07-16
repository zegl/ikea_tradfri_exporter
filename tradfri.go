package main

import (
	"fmt"
	"github.com/adrianliechti/go-tradfri"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type tradfriCollector struct {
	logger *zap.Logger
	client *tradfri.Tradfri

	bulbPower     *prometheus.GaugeVec
	bulbDimmer    *prometheus.GaugeVec
	scrapesFailed prometheus.Counter
}

var variableGroupLabelNames = []string{
	"id",
	"name",
	"manufacturer",
	"model",
}

func NewTradfriCollector(namespace string, logger *zap.Logger, client *tradfri.Tradfri) prometheus.Collector {
	c := &tradfriCollector{
		logger: logger,
		client: client,
		bulbPower: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "bulb",
				Name:      "power",
				Help:      "Bulb Power Level",
			},
			variableGroupLabelNames,
		),
		bulbDimmer: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "bulb",
				Name:      "dimmer",
				Help:      "Bulb Dimmer Level",
			},
			variableGroupLabelNames,
		),
		scrapesFailed: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "scrapes",
				Name:      "failed",
				Help:      "Count of scrapes that have failed",
			},
		),
	}

	return c
}

func (c tradfriCollector) Describe(ch chan<- *prometheus.Desc) {
	c.bulbPower.Describe(ch)
	c.bulbDimmer.Describe(ch)
}

func (c *tradfriCollector) Collect(ch chan<- prometheus.Metric) {
	c.bulbPower.Reset()
	c.bulbDimmer.Reset()

	if devices, err := c.client.Devices(); err != nil {
		c.scrapesFailed.Inc()
		c.logger.Error("failed to get devices", zap.Error(err))
	} else {
		for _, id := range devices {
			if device, err := c.client.Device(id); err == nil {
				l := prometheus.Labels{
					"id":           fmt.Sprintf("%d", id),
					"name":         device.Name,
					"manufacturer": device.Metadata.Manufacturer,
					"model":        device.Metadata.Model,
				}

				if device.Type == tradfri.DeviceTypeBulb {
					for _, light := range device.LightSettings {
						c.bulbPower.With(l).Set(float64(*light.Power))
						c.bulbDimmer.With(l).Set(float64(*light.Dimmer))
					}
				}
			}
		}
	}

	c.bulbPower.Collect(ch)
	c.bulbDimmer.Collect(ch)
}
