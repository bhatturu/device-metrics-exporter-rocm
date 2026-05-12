/**
# Copyright (c) Advanced Micro Devices, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the \"License\");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an \"AS IS\" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package gpuagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ROCm/device-metrics-exporter/pkg/amdgpu/gen/amdgpu"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/gen/exportermetrics"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/globals"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/logger"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/scheduler"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/utils"
	"github.com/ROCm/device-metrics-exporter/pkg/types"
)

// local variables
var (
	ifoeMandatoryLables = []string{
		exportermetrics.MetricLabel_HOSTNAME.String(),
		exportermetrics.GPUMetricLabel_GPU_UUID.String(),
	}
)

// model for IFOE metrics
// Device->Station->NetworkPort
type IFOEMetrics struct {
	// IFOE network port stats
	totalNetworkPorts                    prometheus.GaugeVec
	numFailedoverStreams                 prometheus.GaugeVec
	numPausedStreams                     prometheus.GaugeVec
	bitErrorRate                         prometheus.GaugeVec
	fecCodeWordSymbolErrors0             prometheus.GaugeVec
	fecCodeWordSymbolErrors1             prometheus.GaugeVec
	fecCodeWordSymbolErrors2             prometheus.GaugeVec
	fecCodeWordSymbolErrors3             prometheus.GaugeVec
	fecCodeWordSymbolErrors4             prometheus.GaugeVec
	fecCodeWordSymbolErrors5             prometheus.GaugeVec
	fecCodeWordSymbolErrors6             prometheus.GaugeVec
	fecCodeWordSymbolErrors7             prometheus.GaugeVec
	fecCodeWordSymbolErrors8             prometheus.GaugeVec
	fecCodeWordSymbolErrors9             prometheus.GaugeVec
	fecCodeWordSymbolErrors10            prometheus.GaugeVec
	fecCodeWordSymbolErrors11            prometheus.GaugeVec
	fecCodeWordSymbolErrors12            prometheus.GaugeVec
	fecCodeWordSymbolErrors13            prometheus.GaugeVec
	fecCodeWordSymbolErrors14            prometheus.GaugeVec
	fecCodeWordSymbolErrors15            prometheus.GaugeVec
	fecCodeWordSymbolErrorsUncorrectable prometheus.GaugeVec

	// IFOE network port status
	portLinkState                      prometheus.GaugeVec
	portSpeed                          prometheus.GaugeVec
	portLinkUpCount                    prometheus.GaugeVec
	portLinkUpDurationMsec             prometheus.GaugeVec
	portLinkDownDurationMsec           prometheus.GaugeVec
	portLinkTrainingDurationLatestMsec prometheus.GaugeVec
	portLinkTrainingDurationAvgMsec    prometheus.GaugeVec

	// IFOE network port TX/RX core stats
	txTotalBytes              prometheus.GaugeVec
	txTotalGoodBytes          prometheus.GaugeVec
	txTotalErrBytes           prometheus.GaugeVec
	txTotalPackets            prometheus.GaugeVec
	txTotalGoodPackets        prometheus.GaugeVec
	txFrameError              prometheus.GaugeVec
	txBadFCS                  prometheus.GaugeVec
	rxTotalBytes              prometheus.GaugeVec
	rxTotalGoodBytes          prometheus.GaugeVec
	rxTotalErrBytes           prometheus.GaugeVec
	rxTotalPackets            prometheus.GaugeVec
	rxTotalGoodPackets        prometheus.GaugeVec
	rxPacketDropped           prometheus.GaugeVec
	rxBadFCS                  prometheus.GaugeVec
	rxFECCorrectedCodewords   prometheus.GaugeVec
	rxFECUncorrectedCodewords prometheus.GaugeVec
	txPause                   prometheus.GaugeVec
	rxPause                   prometheus.GaugeVec
	txUserPause               prometheus.GaugeVec
	rxUserPause               prometheus.GaugeVec
	rxJabber                  prometheus.GaugeVec
	rxOversize                prometheus.GaugeVec
	rxTooLong                 prometheus.GaugeVec
	rxTruncated               prometheus.GaugeVec
	txLLROkPackets            prometheus.GaugeVec
	rxLLROkPackets            prometheus.GaugeVec
	txLLRReplayCount          prometheus.GaugeVec
	txLLRReplaysCompleted     prometheus.GaugeVec
	rxLLRBadPackets           prometheus.GaugeVec
	rxLLRDuplSeqPackets       prometheus.GaugeVec

	// IFOE device stats
	totalDevices prometheus.GaugeVec

	// IFOE station stats
	totalStations                   prometheus.GaugeVec
	stationTxRequestPackets         prometheus.GaugeVec
	stationTxResponsePackets        prometheus.GaugeVec
	stationRxRequestPackets         prometheus.GaugeVec
	stationRxResponsePackets        prometheus.GaugeVec
	stationStreamRemapsTotal        prometheus.GaugeVec
	stationPausedStreamsCount       prometheus.GaugeVec
	stationStreamRemapsNetworkPort0 prometheus.GaugeVec
	stationStreamRemapsNetworkPort1 prometheus.GaugeVec
	stationStreamRemapsNetworkPort2 prometheus.GaugeVec
	stationStreamRemapsNetworkPort3 prometheus.GaugeVec
}

func GetIFOEMandatoryLabels() []string {
	return ifoeMandatoryLables
}

// initCustomLabels initializes custom label configuration for the IFOE client.
func (ga *GPUAgentIFOEClient) initCustomLabels(config *exportermetrics.IFOEMetricConfig) {
	ga.customLabelMap = make(map[string]string)
	disallowedLabels := []string{}
	if config != nil && config.GetCustomLabels() != nil {
		for _, name := range exportermetrics.GPUMetricLabel_name {
			found := false
			for _, cname := range ga.allowedCustomLabels {
				if name == cname {
					found = true
					break
				}
			}
			if !found {
				disallowedLabels = append(disallowedLabels, strings.ToLower(name))
			}
		}
		cl := config.GetCustomLabels()
		labelCount := 0

		for l, value := range cl {
			if labelCount >= globals.MaxSupportedCustomLabels {
				logger.Log.Printf("Max custom labels supported: %v, ignoring extra labels.", globals.MaxSupportedCustomLabels)
				break
			}
			label := strings.ToLower(l)

			// Check if custom label is a mandatory label, ignore if true
			found := false
			for _, dlabel := range disallowedLabels {
				if dlabel == label {
					logger.Log.Printf("Label %s cannot be customized, ignoring...", dlabel)
					found = true
					break
				}
			}
			if found {
				continue
			}

			// Store all custom labels
			ga.customLabelMap[label] = value
			labelCount++
		}
	}
	logger.Log.Printf("custom labels being exported: %v", ga.customLabelMap)
}

func (ga *GPUAgentIFOEClient) initLabelConfigs(config *exportermetrics.IFOEMetricConfig) {
	// list of supported labels
	ga.exportLabels = make(map[string]bool)

	for _, name := range exportermetrics.MetricLabel_name {
		ga.exportLabels[name] = false
	}
	for _, name := range exportermetrics.GPUMetricLabel_name {
		ga.exportLabels[name] = false
	}
	// only mandatory labels are set for default
	for _, name := range ifoeMandatoryLables {
		ga.exportLabels[name] = true
	}

	if config != nil {
		for _, name := range config.GetLabels() {
			name = strings.ToUpper(name)
			if _, ok := ga.exportLabels[name]; ok {
				logger.Log.Printf("label %v enabled", name)
				ga.exportLabels[name] = true
			}
		}
	}
	logger.Log.Printf("export-labels updated to %v", ga.exportLabels)
}

func (ga *GPUAgentIFOEClient) initFieldConfig(config *exportermetrics.IFOEMetricConfig) {
	ga.exportFieldMap = make(map[string]bool)
	// setup metric fields in map to be monitored
	// init the map with all supported strings from enum
	enable_default := true
	if config != nil && len(config.GetFields()) != 0 {
		enable_default = false
	}
	for _, name := range exportermetrics.IFOEMetricField_name {
		ga.exportFieldMap[name] = enable_default
	}
	if config == nil || len(config.GetFields()) == 0 {
		return
	}
	for _, fieldName := range config.GetFields() {
		fieldName = strings.ToUpper(fieldName)
		if _, ok := ga.exportFieldMap[fieldName]; ok {
			ga.exportFieldMap[fieldName] = true
		}
	}
	// print disabled short list
	for k, v := range ga.exportFieldMap {
		if !v {
			logger.Log.Printf("%v field is disabled", k)
		}
	}
}

func (ga *GPUAgentIFOEClient) initPrometheusMetrics() {
	labels := ga.GetExportLabels()
	nonIfoeLabels := ga.GetExporterNonIFOELabels()
	logger.Log.Printf("initPrometheusMetrics with labels: %v", labels)
	logger.Log.Printf("initPrometheusMetrics with non-IFOE labels: %v", nonIfoeLabels)
	ga.metrics = &IFOEMetrics{
		totalDevices: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_total_devices",
				Help: "Total number of IFOE devices",
			},
			nonIfoeLabels,
		),
		totalStations: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_total_stations",
				Help: "Total number of IFOE stations",
			},
			nonIfoeLabels,
		),
		totalNetworkPorts: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_total_ports",
				Help: "Total number of IFOE network ports",
			},
			nonIfoeLabels,
		),
		numFailedoverStreams: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_num_failedover_streams",
				Help: "Number of failed over IFOE streams",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		numPausedStreams: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_num_paused_streams",
				Help: "Number of paused IFOE streams",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		bitErrorRate: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_bit_error_rate",
				Help: "Bit Error Rate (BER) reported by the network port expressed as errors per 10^12 bits",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors0",
				Help: "Total number of FEC codewords with 0 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors1",
				Help: "Total number of FEC codewords with 1 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors2: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors2",
				Help: "Total number of FEC codewords with 2 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors3: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors3",
				Help: "Total number of FEC codewords with 3 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors4: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors4",
				Help: "Total number of FEC codewords with 4 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors5: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors5",
				Help: "Total number of FEC codewords with 5 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors6: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors6",
				Help: "Total number of FEC codewords with 6 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors7: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors7",
				Help: "Total number of FEC codewords with 7 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors8: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors8",
				Help: "Total number of FEC codewords with 8 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors9: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors9",
				Help: "Total number of FEC codewords with 9 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors10: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors10",
				Help: "Total number of FEC codewords with 10 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors11: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors11",
				Help: "Total number of FEC codewords with 11 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors12: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors12",
				Help: "Total number of FEC codewords with 12 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors13: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors13",
				Help: "Total number of FEC codewords with 13 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors14: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors14",
				Help: "Total number of FEC codewords with 14 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrors15: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors15",
				Help: "Total number of FEC codewords with 15 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		fecCodeWordSymbolErrorsUncorrectable: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors_uncorrectable",
				Help: "Total number of FEC codewords that are uncorrectable",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		portLinkState: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_state",
				Help: "UAL network port link state (0=NONE, 1=UP, 2=DOWN)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		portSpeed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_speed",
				Help: "UAL network port speed (0=NONE, 1=400G, 2=800G)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		portLinkUpCount: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_up_count",
				Help: "Number of times the UAL network link has come up",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		portLinkUpDurationMsec: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_up_duration_msec",
				Help: "Total time the UAL network link has been up (msec)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		portLinkDownDurationMsec: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_down_duration_msec",
				Help: "Total time the UAL network link has been down (msec)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txTotalBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_total_bytes",
				Help: "Total number of bytes transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txTotalGoodBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_total_good_bytes",
				Help: "Total number of good bytes transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txTotalErrBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_total_err_bytes",
				Help: "Total number of bad bytes transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txTotalPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_total_packets",
				Help: "Total number of packets transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txTotalGoodPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_total_good_packets",
				Help: "Total number of good packets transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txFrameError: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_frame_error",
				Help: "Total number of packets with frame error transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txBadFCS: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_bad_fcs",
				Help: "Total number of packets with bad FCS transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxTotalBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_total_bytes",
				Help: "Total number of bytes received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxTotalGoodBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_total_good_bytes",
				Help: "Total number of good bytes received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxTotalErrBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_total_err_bytes",
				Help: "Total number of bad bytes received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxTotalPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_total_packets",
				Help: "Total number of packets received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxTotalGoodPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_total_good_packets",
				Help: "Total number of good packets received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxPacketDropped: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_packet_dropped",
				Help: "Total number of dropped packets on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxBadFCS: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_bad_fcs",
				Help: "Total number of packets with bad FCS received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		stationTxRequestPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_request_packets",
				Help: "Count of IFoE request packets transmitted on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid"}, labels...)),
		stationTxResponsePackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_response_packets",
				Help: "Count of IFoE response packets transmitted on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid"}, labels...)),
		stationRxRequestPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_rx_request_packets",
				Help: "Count of IFoE request packets received on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid"}, labels...)),
		stationRxResponsePackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_rx_response_packets",
				Help: "Count of IFoE response packets received on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid"}, labels...)),
		stationStreamRemapsTotal: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_stream_remaps_total",
				Help: "Total count of streams remapped due to retransmission timeout on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid"}, labels...)),
		stationPausedStreamsCount: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_paused_streams_count",
				Help: "Live number of IFoE streams paused due to retransmission timeouts on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid"}, labels...)),
		portLinkTrainingDurationLatestMsec: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_training_duration_latest_msec",
				Help: "Time taken to complete link training for most recent link up (msec)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		portLinkTrainingDurationAvgMsec: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_training_duration_avg_msec",
				Help: "Average time taken to complete link training (msec)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxFECCorrectedCodewords: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_corrected_codewords",
				Help: "Count of FEC corrected codewords on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxFECUncorrectedCodewords: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_uncorrected_codewords",
				Help: "Count of FEC uncorrected codewords on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txPause: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_pause",
				Help: "Total number of 802.3x MAC pause packets transmitted",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxPause: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_pause",
				Help: "Total number of 802.3x MAC pause packets received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txUserPause: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_user_pause",
				Help: "Total number of priority based pause packets transmitted",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxUserPause: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_user_pause",
				Help: "Total number of priority based pause packets received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxJabber: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_jabber",
				Help: "Total number of packets longer than max length with bad FCS received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxOversize: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_oversize",
				Help: "Total number of packets longer than max length with good FCS received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxTooLong: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_too_long",
				Help: "Total number of packets longer than max length received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxTruncated: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_truncated",
				Help: "Total number of truncated packets received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txLLROkPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_llr_ok_packets",
				Help: "Count of successfully transmitted LLR packets",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxLLROkPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_llr_ok_packets",
				Help: "Count of successfully received LLR packets",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txLLRReplayCount: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_llr_replay_count",
				Help: "Count of LLR replay events on transmit",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		txLLRReplaysCompleted: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_llr_replays_completed",
				Help: "Count of completed LLR replays on transmit",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxLLRBadPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_llr_bad_packets",
				Help: "Count of bad LLR packets received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		rxLLRDuplSeqPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_llr_dupl_seq_packets",
				Help: "Count of duplicate sequence number LLR packets received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid"}, labels...)),
		stationStreamRemapsNetworkPort0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_stream_remaps_network_port0",
				Help: "Count of streams remapped on network port 0 of the UAL station",
			},
			append([]string{"station_uuid", "device_uuid"}, labels...)),
		stationStreamRemapsNetworkPort1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_stream_remaps_network_port1",
				Help: "Count of streams remapped on network port 1 of the UAL station",
			},
			append([]string{"station_uuid", "device_uuid"}, labels...)),
		stationStreamRemapsNetworkPort2: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_stream_remaps_network_port2",
				Help: "Count of streams remapped on network port 2 of the UAL station",
			},
			append([]string{"station_uuid", "device_uuid"}, labels...)),
		stationStreamRemapsNetworkPort3: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_stream_remaps_network_port3",
				Help: "Count of streams remapped on network port 3 of the UAL station",
			},
			append([]string{"station_uuid", "device_uuid"}, labels...)),
	}
	ga.initFieldMetricsMap()
}

func (ga *GPUAgentIFOEClient) initFieldMetricsMap() {
	// nolint
	ga.fieldMetricsMap = map[string]FieldMeta{
		exportermetrics.IFOEMetricField_IFOE_TOTAL_DEVICES.String():                            FieldMeta{Metric: ga.metrics.totalDevices},
		exportermetrics.IFOEMetricField_IFOE_TOTAL_STATIONS.String():                           FieldMeta{Metric: ga.metrics.totalStations},
		exportermetrics.IFOEMetricField_IFOE_TOTAL_PORTS.String():                              FieldMeta{Metric: ga.metrics.totalNetworkPorts},
		exportermetrics.IFOEMetricField_IFOE_NUMBER_FAILEDOVER_STREAMS.String():                FieldMeta{Metric: ga.metrics.numFailedoverStreams},
		exportermetrics.IFOEMetricField_IFOE_NUMBER_PAUSED_STREAMS.String():                    FieldMeta{Metric: ga.metrics.numPausedStreams},
		exportermetrics.IFOEMetricField_IFOE_BIT_ERROR_RATE.String():                           FieldMeta{Metric: ga.metrics.bitErrorRate},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS0.String():              FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors0},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS1.String():              FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors1},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS2.String():              FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors2},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS3.String():              FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors3},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS4.String():              FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors4},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS5.String():              FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors5},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS6.String():              FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors6},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS7.String():              FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors7},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS8.String():              FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors8},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS9.String():              FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors9},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS10.String():             FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors10},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS11.String():             FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors11},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS12.String():             FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors12},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS13.String():             FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors13},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS14.String():             FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors14},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS15.String():             FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrors15},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS_UNCORRECTABLE.String(): FieldMeta{Metric: ga.metrics.fecCodeWordSymbolErrorsUncorrectable},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_STATE.String():                          FieldMeta{Metric: ga.metrics.portLinkState},
		exportermetrics.IFOEMetricField_IFOE_PORT_SPEED.String():                               FieldMeta{Metric: ga.metrics.portSpeed},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_UP_COUNT.String():                       FieldMeta{Metric: ga.metrics.portLinkUpCount},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_UP_DURATION_MSEC.String():               FieldMeta{Metric: ga.metrics.portLinkUpDurationMsec},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_DOWN_DURATION_MSEC.String():             FieldMeta{Metric: ga.metrics.portLinkDownDurationMsec},
		exportermetrics.IFOEMetricField_IFOE_TX_TOTAL_BYTES.String():                           FieldMeta{Metric: ga.metrics.txTotalBytes},
		exportermetrics.IFOEMetricField_IFOE_TX_TOTAL_GOOD_BYTES.String():                      FieldMeta{Metric: ga.metrics.txTotalGoodBytes},
		exportermetrics.IFOEMetricField_IFOE_TX_TOTAL_ERR_BYTES.String():                       FieldMeta{Metric: ga.metrics.txTotalErrBytes},
		exportermetrics.IFOEMetricField_IFOE_TX_TOTAL_PACKETS.String():                         FieldMeta{Metric: ga.metrics.txTotalPackets},
		exportermetrics.IFOEMetricField_IFOE_TX_TOTAL_GOOD_PACKETS.String():                    FieldMeta{Metric: ga.metrics.txTotalGoodPackets},
		exportermetrics.IFOEMetricField_IFOE_TX_FRAME_ERROR.String():                           FieldMeta{Metric: ga.metrics.txFrameError},
		exportermetrics.IFOEMetricField_IFOE_TX_BAD_FCS.String():                               FieldMeta{Metric: ga.metrics.txBadFCS},
		exportermetrics.IFOEMetricField_IFOE_RX_TOTAL_BYTES.String():                           FieldMeta{Metric: ga.metrics.rxTotalBytes},
		exportermetrics.IFOEMetricField_IFOE_RX_TOTAL_GOOD_BYTES.String():                      FieldMeta{Metric: ga.metrics.rxTotalGoodBytes},
		exportermetrics.IFOEMetricField_IFOE_RX_TOTAL_ERR_BYTES.String():                       FieldMeta{Metric: ga.metrics.rxTotalErrBytes},
		exportermetrics.IFOEMetricField_IFOE_RX_TOTAL_PACKETS.String():                         FieldMeta{Metric: ga.metrics.rxTotalPackets},
		exportermetrics.IFOEMetricField_IFOE_RX_TOTAL_GOOD_PACKETS.String():                    FieldMeta{Metric: ga.metrics.rxTotalGoodPackets},
		exportermetrics.IFOEMetricField_IFOE_RX_PACKET_DROPPED.String():                        FieldMeta{Metric: ga.metrics.rxPacketDropped},
		exportermetrics.IFOEMetricField_IFOE_RX_BAD_FCS.String():                               FieldMeta{Metric: ga.metrics.rxBadFCS},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_REQUEST_PACKETS.String():               FieldMeta{Metric: ga.metrics.stationTxRequestPackets},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_RESPONSE_PACKETS.String():              FieldMeta{Metric: ga.metrics.stationTxResponsePackets},
		exportermetrics.IFOEMetricField_IFOE_STATION_RX_REQUEST_PACKETS.String():               FieldMeta{Metric: ga.metrics.stationRxRequestPackets},
		exportermetrics.IFOEMetricField_IFOE_STATION_RX_RESPONSE_PACKETS.String():              FieldMeta{Metric: ga.metrics.stationRxResponsePackets},
		exportermetrics.IFOEMetricField_IFOE_STATION_STREAM_REMAPS_TOTAL.String():              FieldMeta{Metric: ga.metrics.stationStreamRemapsTotal},
		exportermetrics.IFOEMetricField_IFOE_STATION_PAUSED_STREAMS_COUNT.String():             FieldMeta{Metric: ga.metrics.stationPausedStreamsCount},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_TRAINING_DURATION_LATEST_MSEC.String():  FieldMeta{Metric: ga.metrics.portLinkTrainingDurationLatestMsec},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_TRAINING_DURATION_AVG_MSEC.String():     FieldMeta{Metric: ga.metrics.portLinkTrainingDurationAvgMsec},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_CORRECTED_CODEWORDS.String():               FieldMeta{Metric: ga.metrics.rxFECCorrectedCodewords},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_UNCORRECTED_CODEWORDS.String():             FieldMeta{Metric: ga.metrics.rxFECUncorrectedCodewords},
		exportermetrics.IFOEMetricField_IFOE_TX_PAUSE.String():                                 FieldMeta{Metric: ga.metrics.txPause},
		exportermetrics.IFOEMetricField_IFOE_RX_PAUSE.String():                                 FieldMeta{Metric: ga.metrics.rxPause},
		exportermetrics.IFOEMetricField_IFOE_TX_USER_PAUSE.String():                            FieldMeta{Metric: ga.metrics.txUserPause},
		exportermetrics.IFOEMetricField_IFOE_RX_USER_PAUSE.String():                            FieldMeta{Metric: ga.metrics.rxUserPause},
		exportermetrics.IFOEMetricField_IFOE_RX_JABBER.String():                                FieldMeta{Metric: ga.metrics.rxJabber},
		exportermetrics.IFOEMetricField_IFOE_RX_OVERSIZE.String():                              FieldMeta{Metric: ga.metrics.rxOversize},
		exportermetrics.IFOEMetricField_IFOE_RX_TOO_LONG.String():                              FieldMeta{Metric: ga.metrics.rxTooLong},
		exportermetrics.IFOEMetricField_IFOE_RX_TRUNCATED.String():                             FieldMeta{Metric: ga.metrics.rxTruncated},
		exportermetrics.IFOEMetricField_IFOE_TX_LLR_OK_PACKETS.String():                        FieldMeta{Metric: ga.metrics.txLLROkPackets},
		exportermetrics.IFOEMetricField_IFOE_RX_LLR_OK_PACKETS.String():                        FieldMeta{Metric: ga.metrics.rxLLROkPackets},
		exportermetrics.IFOEMetricField_IFOE_TX_LLR_REPLAY_COUNT.String():                      FieldMeta{Metric: ga.metrics.txLLRReplayCount},
		exportermetrics.IFOEMetricField_IFOE_TX_LLR_REPLAYS_COMPLETED.String():                 FieldMeta{Metric: ga.metrics.txLLRReplaysCompleted},
		exportermetrics.IFOEMetricField_IFOE_RX_LLR_BAD_PACKETS.String():                       FieldMeta{Metric: ga.metrics.rxLLRBadPackets},
		exportermetrics.IFOEMetricField_IFOE_RX_LLR_DUPL_SEQ_PACKETS.String():                  FieldMeta{Metric: ga.metrics.rxLLRDuplSeqPackets},
		exportermetrics.IFOEMetricField_IFOE_STATION_STREAM_REMAPS_NETWORK_PORT0.String():      FieldMeta{Metric: ga.metrics.stationStreamRemapsNetworkPort0},
		exportermetrics.IFOEMetricField_IFOE_STATION_STREAM_REMAPS_NETWORK_PORT1.String():      FieldMeta{Metric: ga.metrics.stationStreamRemapsNetworkPort1},
		exportermetrics.IFOEMetricField_IFOE_STATION_STREAM_REMAPS_NETWORK_PORT2.String():      FieldMeta{Metric: ga.metrics.stationStreamRemapsNetworkPort2},
		exportermetrics.IFOEMetricField_IFOE_STATION_STREAM_REMAPS_NETWORK_PORT3.String():      FieldMeta{Metric: ga.metrics.stationStreamRemapsNetworkPort3},
	}
}

func (ga *GPUAgentIFOEClient) initFieldRegistration() error {
	for field, enabled := range ga.exportFieldMap {
		if !enabled {
			continue
		}
		prommetric, ok := ga.fieldMetricsMap[field]
		if !ok {
			logger.Log.Printf("invalid field found ignore %v", field)
			continue
		}
		if err := ga.gpuHandler.mh.RegisterMetric(prommetric.Metric); err != nil {
			logger.Log.Printf("Field %v registration failed with err : %v", field, err)
		}
	}

	return nil
}

func (ga *GPUAgentIFOEClient) InitConfigs() error {
	logger.Log.Printf("Initializing GPU Agent IFOE Client configs")
	filedConfigs := ga.gpuHandler.mh.GetIFOEMetricsConfig()

	ga.initCustomLabels(filedConfigs)
	ga.initLabelConfigs(filedConfigs)
	ga.initFieldConfig(filedConfigs)
	ga.InitPodExtraLabels(filedConfigs)
	ga.initPrometheusMetrics()
	return ga.initFieldRegistration()
}

func (ga *GPUAgentIFOEClient) InitPodExtraLabels(config *exportermetrics.IFOEMetricConfig) {
	// initialize pod labels maps
	ga.k8PodInfoMap = make(map[string]types.K8sPodInfo)
	if config != nil {
		ga.extraPodLabelsMap = utils.NormalizeExtraPodLabels(config.GetExtraPodLabels())
		if len(ga.extraPodLabelsMap) > 0 {
			ga.podInfoEnabled = true
		}
	}
	logger.Log.Printf("export-labels updated to %v", ga.extraPodLabelsMap)
}

func (ga *GPUAgentIFOEClient) PopulateStaticHostLabels() error {
	ga.staticHostLabels = map[string]string{}
	hostname, err := utils.GetHostName()
	if err != nil {
		return err
	}
	logger.Log.Printf("hostame %v", hostname)
	ga.staticHostLabels[exportermetrics.MetricLabel_HOSTNAME.String()] = hostname
	return nil
}

func (ga *GPUAgentIFOEClient) populateLabelsFromObject(
	wls map[string]scheduler.Workload,
	ualStationMap map[string]*amdgpu.UALStation,
	ualDevice *amdgpu.UALDevice,
	populatePlaceholders bool) map[string]string {

	var podInfo scheduler.PodResourceInfo

	labels := make(map[string]string)

	// Pull real values from the UAL device when available; placeholders
	// remain only for fields the device does not report.
	var (
		gpuUUID         string
		driverVersion   = "driver_version_placeholder"
		firmwareVersion = "vbios_version_placeholder"
	)
	if ualDevice != nil && ualDevice.Status != nil {
		gpuUUID = utils.UUIDToString(ualDevice.Status.GPU)
		if ver := ualDevice.Status.Version; ver != nil {
			if v := ver.UALLibVersion; v != nil {
				driverVersion = fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
			}
			if v := ver.FirmwareVersion; v != nil {
				firmwareVersion = fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
			}
		}
	}

	for ckey, enabled := range ga.exportLabels {
		if !enabled {
			continue
		}
		key := strings.ToLower(ckey)
		switch ckey {
		case exportermetrics.GPUMetricLabel_GPU_UUID.String():
			if populatePlaceholders {
				labels[key] = gpuUUID
			}
		case exportermetrics.MetricLabel_CARD_SERIES.String():
			if populatePlaceholders {
				labels[key] = "card_series_placeholder"
			}
		case exportermetrics.MetricLabel_CARD_MODEL.String():
			if populatePlaceholders {
				labels[key] = "card_model_placeholder"
			}
		case exportermetrics.MetricLabel_CARD_VENDOR.String():
			if populatePlaceholders {
				labels[key] = "AMD"
			}
		case exportermetrics.MetricLabel_DRIVER_VERSION.String():
			if populatePlaceholders {
				labels[key] = driverVersion
			}
		case exportermetrics.MetricLabel_VBIOS_VERSION.String():
			if populatePlaceholders {
				labels[key] = firmwareVersion
			}
		case exportermetrics.MetricLabel_POD.String():
			if populatePlaceholders {
				labels[key] = "pod_placeholder"
			}
		case exportermetrics.MetricLabel_NAMESPACE.String():
			if populatePlaceholders {
				labels[key] = "namespace_placeholder"
			}
		case exportermetrics.MetricLabel_CONTAINER.String():
			if populatePlaceholders {
				labels[key] = "container_placeholder"
			}
		case exportermetrics.MetricLabel_JOB_ID.String():
			if populatePlaceholders {
				labels[key] = "job_id_placeholder"
			}
		case exportermetrics.MetricLabel_JOB_USER.String():
			if populatePlaceholders {
				labels[key] = "job_user_placeholder"
			}
		case exportermetrics.MetricLabel_JOB_PARTITION.String():
			if populatePlaceholders {
				labels[key] = "job_partition_placeholder"
			}
		case exportermetrics.MetricLabel_CLUSTER_NAME.String():
			if populatePlaceholders {
				labels[key] = "cluster_name_placeholder"
			}
		case exportermetrics.MetricLabel_SERIAL_NUMBER.String():
			if populatePlaceholders {
				labels[key] = "serial_number_placeholder"
			}
		case exportermetrics.MetricLabel_HOSTNAME.String():
			labels[key] = ga.staticHostLabels[exportermetrics.MetricLabel_HOSTNAME.String()]
		default:
			logger.Log.Printf("Invalid label is ignored %v", key)
		}
	}

	// Add extra pod labels only if config has mapped any
	if populatePlaceholders && len(ga.extraPodLabelsMap) > 0 {
		podLabels := utils.GetPodLabels(&podInfo, ga.k8PodInfoMap)
		for prometheusPodlabel, k8Podlabel := range ga.extraPodLabelsMap {
			label := strings.ToLower(prometheusPodlabel)
			labels[label] = podLabels[k8Podlabel]
		}
	}

	// Add custom labels
	for label, value := range ga.customLabelMap {
		labels[label] = value
	}
	return labels
}

func (ga *GPUAgentIFOEClient) ResetMetrics() error {
	// reset all label based fields
	for _, prommetric := range ga.fieldMetricsMap {
		prommetric.Metric.Reset()
	}
	return nil
}

func (ga *GPUAgentIFOEClient) UpdateStaticMetrics(ctx context.Context) error {
	return ga.updateMetrics(ctx)
}

func (ga *GPUAgentIFOEClient) UpdateMetricsStats(ctx context.Context) error {
	return ga.updateMetrics(ctx)
}

func (ga *GPUAgentIFOEClient) QueryMetrics() (interface{}, error) {
	// No op for now
	return nil, nil
}

func (ga *GPUAgentIFOEClient) QueryInbandRASErrors(severity string) (interface{}, error) {
	// No op for now
	return nil, nil
}

// SetComputeNodeHealthState sets the compute node health state
func (ga *GPUAgentIFOEClient) SetComputeNodeHealthState(state bool) {
	ga.Lock()

	if ga.computeNodeHealthState == state {
		return
	}
	logger.Log.Printf("updating compute node health from: %v, to: %v", ga.computeNodeHealthState, state)
	ga.computeNodeHealthState = state
	ga.Unlock()

	// for now no metrics to update or health states to send
}

// FetchPodInfoForNode fetches pod labels for all pods running on this node
func (ga *GPUAgentIFOEClient) FetchPodInfoForNode() (map[string]types.K8sPodInfo, error) {
	if !ga.gpuHandler.enabledK8sApi {
		return nil, nil
	}
	k8sSchedClient := ga.gpuHandler.GetK8sApiClient()
	if k8sSchedClient == nil {
		return nil, fmt.Errorf("k8s scheduler client is nil")
	}
	listMap := make(map[string]types.K8sPodInfo)
	if ga.gpuHandler.enabledK8sApi && ga.podInfoEnabled {
		return k8sSchedClient.GetAllPods()
	}
	return listMap, nil
}
