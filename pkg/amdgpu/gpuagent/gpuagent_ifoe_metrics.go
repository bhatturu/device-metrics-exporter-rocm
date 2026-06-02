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
	// IFOE per-lane FEC and error counters (port-level)
	rxFECBitErr0To1Lane0     prometheus.GaugeVec
	rxFECBitErr0To1Lane1     prometheus.GaugeVec
	rxFECBitErr0To1Lane2     prometheus.GaugeVec
	rxFECBitErr0To1Lane3     prometheus.GaugeVec
	rxFECBitErr1To0Lane0     prometheus.GaugeVec
	rxFECBitErr1To0Lane1     prometheus.GaugeVec
	rxFECBitErr1To0Lane2     prometheus.GaugeVec
	rxFECBitErr1To0Lane3     prometheus.GaugeVec
	rxFECSymbolErrCountLane0 prometheus.GaugeVec
	rxFECSymbolErrCountLane1 prometheus.GaugeVec
	rxFECSymbolErrCountLane2 prometheus.GaugeVec
	rxFECSymbolErrCountLane3 prometheus.GaugeVec
	rxBadCodeCount           prometheus.GaugeVec
	rxStompedFCS             prometheus.GaugeVec

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
	// IFOE crypto key update counters (station-level)
	stationCryptoTxKeyUpdatesSA0                prometheus.GaugeVec
	stationCryptoRxKey0UpdatesSA0               prometheus.GaugeVec
	stationCryptoRxKey1UpdatesSA0               prometheus.GaugeVec
	stationCryptoRxKeyDisablesSA0               prometheus.GaugeVec
	stationCryptoTxKeyUpdatesSA1                prometheus.GaugeVec
	stationCryptoRxKey0UpdatesSA1               prometheus.GaugeVec
	stationCryptoRxKey1UpdatesSA1               prometheus.GaugeVec
	stationCryptoRxKeyDisablesSA1               prometheus.GaugeVec
	discardQRXDroppedPackets                    prometheus.GaugeVec
	nonifoeQRXTotalPackets                      prometheus.GaugeVec
	nonifoeQRXXoffTotal                         prometheus.GaugeVec
	nonifoeQTXTotalPackets                      prometheus.GaugeVec
	nonifoeQTXXoffTotal                         prometheus.GaugeVec
	reqQRXDroppedPackets                        prometheus.GaugeVec
	resQRXDroppedPackets                        prometheus.GaugeVec
	rxPausePacketsRcvd                          prometheus.GaugeVec
	txPausePacketsSent                          prometheus.GaugeVec
	stationRengFreeBlkOut                       prometheus.GaugeVec
	stationRengFreePktOut                       prometheus.GaugeVec
	stationRengFreeSchIn                        prometheus.GaugeVec
	stationRengReadPktOut                       prometheus.GaugeVec
	stationRengReadSchIn                        prometheus.GaugeVec
	stationRXDecapDroppedPkts                   prometheus.GaugeVec
	stationRXDecapRXNAKEgressReq                prometheus.GaugeVec
	stationRXDecapRXNAKEgressRsp                prometheus.GaugeVec
	stationRXDecapTXNAKEgressReq                prometheus.GaugeVec
	stationRXDecapTXNAKEgressRsp                prometheus.GaugeVec
	stationRXDroppedPkts                        prometheus.GaugeVec
	stationRXNonifoePkts                        prometheus.GaugeVec
	stationSDPRXUnpackOrigdataCreditsConsumed   prometheus.GaugeVec
	stationSDPRXUnpackOrigdataCreditsReturned   prometheus.GaugeVec
	stationSDPRXUnpackRdrspCreditsConsumed      prometheus.GaugeVec
	stationSDPRXUnpackRdrspCreditsReturned      prometheus.GaugeVec
	stationSDPRXUnpackReqCreditsConsumed        prometheus.GaugeVec
	stationSDPRXUnpackReqCreditsReturned        prometheus.GaugeVec
	stationSDPRXUnpackReqCyclesStalled          prometheus.GaugeVec
	stationSDPRXUnpackReqCyclesStalledCnt       prometheus.GaugeVec
	stationSDPRXUnpackReqExcessCreditsReturned  prometheus.GaugeVec
	stationSDPRXUnpackReqPayloadCreditsReturned prometheus.GaugeVec
	stationSDPRXUnpackReqTotalCreditsConsumed   prometheus.GaugeVec
	stationSDPRXUnpackRetagFreed                prometheus.GaugeVec
	stationSDPRXUnpackRetagUsed                 prometheus.GaugeVec
	stationSDPRXUnpackRspCyclesStalled          prometheus.GaugeVec
	stationSDPRXUnpackRspCyclesStalledCnt       prometheus.GaugeVec
	stationSDPRXUnpackRspExcessCreditsReturned  prometheus.GaugeVec
	stationSDPRXUnpackRspPayloadCreditsReturned prometheus.GaugeVec
	stationSDPRXUnpackRspTotalCreditsConsumed   prometheus.GaugeVec
	stationSDPRXUnpackWrrspCreditsConsumed      prometheus.GaugeVec
	stationSDPRXUnpackWrrspCreditsReturned      prometheus.GaugeVec
	stationSDPTXPackAtmReq                      prometheus.GaugeVec
	stationSDPTXPackOrigData                    prometheus.GaugeVec
	stationSDPTXPackOrigDataCreditConsumed      prometheus.GaugeVec
	stationSDPTXPackOrigDataCreditReleased      prometheus.GaugeVec
	stationSDPTXPackOrigDataEbEmpty             prometheus.GaugeVec
	stationSDPTXPackOrigDataEbFull              prometheus.GaugeVec
	stationSDPTXPackOrigDataError               prometheus.GaugeVec
	stationSDPTXPackRdReq                       prometheus.GaugeVec
	stationSDPTXPackRdRsp                       prometheus.GaugeVec
	stationSDPTXPackRdRspCreditConsumed         prometheus.GaugeVec
	stationSDPTXPackRdRspCreditReleased         prometheus.GaugeVec
	stationSDPTXPackRdRspDataError              prometheus.GaugeVec
	stationSDPTXPackRdRspEbEmpty                prometheus.GaugeVec
	stationSDPTXPackRdRspEbFull                 prometheus.GaugeVec
	stationSDPTXPackRdRspPclEmpty               prometheus.GaugeVec
	stationSDPTXPackRdRspPclFull                prometheus.GaugeVec
	stationSDPTXPackReqCreditConsumed           prometheus.GaugeVec
	stationSDPTXPackReqCreditReleased           prometheus.GaugeVec
	stationSDPTXPackReqEbEmpty                  prometheus.GaugeVec
	stationSDPTXPackReqEbFull                   prometheus.GaugeVec
	stationSDPTXPackReqPcl                      prometheus.GaugeVec
	stationSDPTXPackReqPclEmpty                 prometheus.GaugeVec
	stationSDPTXPackReqPclFull                  prometheus.GaugeVec
	stationSDPTXPackReqPoolEmpty                prometheus.GaugeVec
	stationSDPTXPackReqPoolFull                 prometheus.GaugeVec
	stationSDPTXPackRspPoolEmpty                prometheus.GaugeVec
	stationSDPTXPackRspPoolFull                 prometheus.GaugeVec
	stationSDPTXPackWrReq                       prometheus.GaugeVec
	stationSDPTXPackWrRsp                       prometheus.GaugeVec
	stationSDPTXPackWrRspCreditConsumed         prometheus.GaugeVec
	stationSDPTXPackWrRspCreditReleased         prometheus.GaugeVec
	stationSDPTXPackWrRspEbEmpty                prometheus.GaugeVec
	stationSDPTXPackWrRspEbFull                 prometheus.GaugeVec
	stationSDPTXPackWrRspPclEmpty               prometheus.GaugeVec
	stationSDPTXPackWrRspPclFull                prometheus.GaugeVec
	stationTXEncapPktEgressXrsecNport0          prometheus.GaugeVec
	stationTXEncapPktEgressXrsecNport1          prometheus.GaugeVec
	stationTXEncapPktEgressXrsecNport2          prometheus.GaugeVec
	stationTXEncapPktEgressXrsecNport3          prometheus.GaugeVec
	stationTXEncapStallEgressXrsec              prometheus.GaugeVec
	stationTXEncapStallIngress                  prometheus.GaugeVec
	stationTXNonifoePkts                        prometheus.GaugeVec
	stationTXSchedActiveRXACKPkts               prometheus.GaugeVec
	stationTXSchedActiveStreams                 prometheus.GaugeVec
	stationTXSchedBoostedPriStreams             prometheus.GaugeVec
	stationTXSchedEmptyQueueStreams             prometheus.GaugeVec
	stationTXSchedEmptySendQueueStreams         prometheus.GaugeVec
	stationTXSchedPausedStreams                 prometheus.GaugeVec
	stationTXSchedReqPkts                       prometheus.GaugeVec
	stationTXSchedResPkts                       prometheus.GaugeVec
	stationWengBstateFreeBlkDelayReq            prometheus.GaugeVec
	stationWengBstateFreeBlkDelayRes            prometheus.GaugeVec
	stationWengEvictionChainDelayReq            prometheus.GaugeVec
	stationWengEvictionChainDelayRes            prometheus.GaugeVec
	stationWengEvictionExpiryReq                prometheus.GaugeVec
	stationWengEvictionExpiryRes                prometheus.GaugeVec
	stationWengEvictionForceChainDelayReq       prometheus.GaugeVec
	stationWengEvictionForceChainDelayRes       prometheus.GaugeVec
	stationWengEvictionForceHoldDelayReq        prometheus.GaugeVec
	stationWengEvictionForceHoldDelayRes        prometheus.GaugeVec
	stationWengEvictionForceReq                 prometheus.GaugeVec
	stationWengEvictionForceRes                 prometheus.GaugeVec
	stationWengEvictionMtuHoldDelayReq          prometheus.GaugeVec
	stationWengEvictionMtuHoldDelayRes          prometheus.GaugeVec
	stationWengEvictionMtuReq                   prometheus.GaugeVec
	stationWengEvictionMtuRes                   prometheus.GaugeVec
	stationWengEvictionPayloadReq               prometheus.GaugeVec
	stationWengEvictionPayloadRes               prometheus.GaugeVec
	stationWengOpSDPChainReq                    prometheus.GaugeVec
	stationWengOpSDPChainRes                    prometheus.GaugeVec
	stationWengOpSDPForceReq                    prometheus.GaugeVec
	stationWengOpSDPForceRes                    prometheus.GaugeVec
	stationWengOpSDPLengthReq                   prometheus.GaugeVec
	stationWengOpSDPLengthRes                   prometheus.GaugeVec
	stationWengOpSDPPktOpenReq                  prometheus.GaugeVec
	stationWengOpSDPPktOpenRes                  prometheus.GaugeVec
	stationWengOpSDPReq                         prometheus.GaugeVec
	stationWengOpSDPRes                         prometheus.GaugeVec
	stationWengOpSDPWordReq                     prometheus.GaugeVec
	stationWengOpSDPWordRes                     prometheus.GaugeVec
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
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		numPausedStreams: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_num_paused_streams",
				Help: "Number of paused IFOE streams",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		bitErrorRate: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_bit_error_rate",
				Help: "Bit Error Rate (BER) reported by the network port expressed as errors per 10^12 bits",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors0",
				Help: "Total number of FEC codewords with 0 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors1",
				Help: "Total number of FEC codewords with 1 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors2: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors2",
				Help: "Total number of FEC codewords with 2 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors3: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors3",
				Help: "Total number of FEC codewords with 3 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors4: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors4",
				Help: "Total number of FEC codewords with 4 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors5: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors5",
				Help: "Total number of FEC codewords with 5 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors6: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors6",
				Help: "Total number of FEC codewords with 6 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors7: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors7",
				Help: "Total number of FEC codewords with 7 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors8: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors8",
				Help: "Total number of FEC codewords with 8 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors9: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors9",
				Help: "Total number of FEC codewords with 9 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors10: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors10",
				Help: "Total number of FEC codewords with 10 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors11: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors11",
				Help: "Total number of FEC codewords with 11 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors12: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors12",
				Help: "Total number of FEC codewords with 12 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors13: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors13",
				Help: "Total number of FEC codewords with 13 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors14: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors14",
				Help: "Total number of FEC codewords with 14 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrors15: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors15",
				Help: "Total number of FEC codewords with 15 symbol errors",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		fecCodeWordSymbolErrorsUncorrectable: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_fec_codeword_symbol_errors_uncorrectable",
				Help: "Total number of FEC codewords that are uncorrectable",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		portLinkState: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_state",
				Help: "UAL network port link state (0=NONE, 1=UP, 2=DOWN)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		portSpeed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_speed",
				Help: "UAL network port speed (0=NONE, 1=400G, 2=800G)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		portLinkUpCount: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_up_count",
				Help: "Number of times the UAL network link has come up",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		portLinkUpDurationMsec: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_up_duration_msec",
				Help: "Total time the UAL network link has been up (msec)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		portLinkDownDurationMsec: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_down_duration_msec",
				Help: "Total time the UAL network link has been down (msec)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txTotalBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_total_bytes",
				Help: "Total number of bytes transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txTotalGoodBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_total_good_bytes",
				Help: "Total number of good bytes transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txTotalErrBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_total_err_bytes",
				Help: "Total number of bad bytes transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txTotalPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_total_packets",
				Help: "Total number of packets transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txTotalGoodPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_total_good_packets",
				Help: "Total number of good packets transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txFrameError: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_frame_error",
				Help: "Total number of packets with frame error transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txBadFCS: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_bad_fcs",
				Help: "Total number of packets with bad FCS transmitted on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxTotalBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_total_bytes",
				Help: "Total number of bytes received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxTotalGoodBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_total_good_bytes",
				Help: "Total number of good bytes received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxTotalErrBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_total_err_bytes",
				Help: "Total number of bad bytes received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxTotalPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_total_packets",
				Help: "Total number of packets received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxTotalGoodPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_total_good_packets",
				Help: "Total number of good packets received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxPacketDropped: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_packet_dropped",
				Help: "Total number of dropped packets on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxBadFCS: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_bad_fcs",
				Help: "Total number of packets with bad FCS received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		stationTxRequestPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_request_packets",
				Help: "Count of IFoE request packets transmitted on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTxResponsePackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_response_packets",
				Help: "Count of IFoE response packets transmitted on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRxRequestPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_rx_request_packets",
				Help: "Count of IFoE request packets received on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRxResponsePackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_rx_response_packets",
				Help: "Count of IFoE response packets received on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationStreamRemapsTotal: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_stream_remaps_total",
				Help: "Total count of streams remapped due to retransmission timeout on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationPausedStreamsCount: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_paused_streams_count",
				Help: "Live number of IFoE streams paused due to retransmission timeouts on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		portLinkTrainingDurationLatestMsec: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_training_duration_latest_msec",
				Help: "Time taken to complete link training for most recent link up (msec)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		portLinkTrainingDurationAvgMsec: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_port_link_training_duration_avg_msec",
				Help: "Average time taken to complete link training (msec)",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECCorrectedCodewords: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_corrected_codewords",
				Help: "Count of FEC corrected codewords on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECUncorrectedCodewords: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_uncorrected_codewords",
				Help: "Count of FEC uncorrected codewords on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txPause: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_pause",
				Help: "Total number of 802.3x MAC pause packets transmitted",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxPause: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_pause",
				Help: "Total number of 802.3x MAC pause packets received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txUserPause: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_user_pause",
				Help: "Total number of priority based pause packets transmitted",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxUserPause: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_user_pause",
				Help: "Total number of priority based pause packets received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxJabber: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_jabber",
				Help: "Total number of packets longer than max length with bad FCS received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxOversize: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_oversize",
				Help: "Total number of packets longer than max length with good FCS received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxTooLong: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_too_long",
				Help: "Total number of packets longer than max length received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxTruncated: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_truncated",
				Help: "Total number of truncated packets received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txLLROkPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_llr_ok_packets",
				Help: "Count of successfully transmitted LLR packets",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxLLROkPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_llr_ok_packets",
				Help: "Count of successfully received LLR packets",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txLLRReplayCount: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_llr_replay_count",
				Help: "Count of LLR replay events on transmit",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txLLRReplaysCompleted: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_llr_replays_completed",
				Help: "Count of completed LLR replays on transmit",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxLLRBadPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_llr_bad_packets",
				Help: "Count of bad LLR packets received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxLLRDuplSeqPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_llr_dupl_seq_packets",
				Help: "Count of duplicate sequence number LLR packets received",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECBitErr0To1Lane0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_bit_err_0to1_lane0",
				Help: "FEC bit error 0-to-1 count on lane 0",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECBitErr0To1Lane1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_bit_err_0to1_lane1",
				Help: "FEC bit error 0-to-1 count on lane 1",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECBitErr0To1Lane2: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_bit_err_0to1_lane2",
				Help: "FEC bit error 0-to-1 count on lane 2",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECBitErr0To1Lane3: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_bit_err_0to1_lane3",
				Help: "FEC bit error 0-to-1 count on lane 3",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECBitErr1To0Lane0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_bit_err_1to0_lane0",
				Help: "FEC bit error 1-to-0 count on lane 0",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECBitErr1To0Lane1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_bit_err_1to0_lane1",
				Help: "FEC bit error 1-to-0 count on lane 1",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECBitErr1To0Lane2: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_bit_err_1to0_lane2",
				Help: "FEC bit error 1-to-0 count on lane 2",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECBitErr1To0Lane3: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_bit_err_1to0_lane3",
				Help: "FEC bit error 1-to-0 count on lane 3",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECSymbolErrCountLane0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_symbol_err_count_lane0",
				Help: "FEC symbol error count on lane 0",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECSymbolErrCountLane1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_symbol_err_count_lane1",
				Help: "FEC symbol error count on lane 1",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECSymbolErrCountLane2: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_symbol_err_count_lane2",
				Help: "FEC symbol error count on lane 2",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxFECSymbolErrCountLane3: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_fec_symbol_err_count_lane3",
				Help: "FEC symbol error count on lane 3",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxBadCodeCount: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_bad_code_count",
				Help: "Count of bad code received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxStompedFCS: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_stomped_fcs",
				Help: "Count of stomped FCS received on the UAL network port",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		stationStreamRemapsNetworkPort0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_stream_remaps_network_port0",
				Help: "Count of streams remapped on network port 0 of the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationStreamRemapsNetworkPort1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_stream_remaps_network_port1",
				Help: "Count of streams remapped on network port 1 of the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationStreamRemapsNetworkPort2: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_stream_remaps_network_port2",
				Help: "Count of streams remapped on network port 2 of the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationStreamRemapsNetworkPort3: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_stream_remaps_network_port3",
				Help: "Count of streams remapped on network port 3 of the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationCryptoTxKeyUpdatesSA0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_crypto_tx_key_updates_sa0",
				Help: "Crypto TX key updates for SA0 on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationCryptoRxKey0UpdatesSA0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_crypto_rx_key0_updates_sa0",
				Help: "Crypto RX key0 updates for SA0 on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationCryptoRxKey1UpdatesSA0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_crypto_rx_key1_updates_sa0",
				Help: "Crypto RX key1 updates for SA0 on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationCryptoRxKeyDisablesSA0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_crypto_rx_key_disables_sa0",
				Help: "Crypto RX key disables for SA0 on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationCryptoTxKeyUpdatesSA1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_crypto_tx_key_updates_sa1",
				Help: "Crypto TX key updates for SA1 on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationCryptoRxKey0UpdatesSA1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_crypto_rx_key0_updates_sa1",
				Help: "Crypto RX key0 updates for SA1 on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationCryptoRxKey1UpdatesSA1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_crypto_rx_key1_updates_sa1",
				Help: "Crypto RX key1 updates for SA1 on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationCryptoRxKeyDisablesSA1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_crypto_rx_key_disables_sa1",
				Help: "Crypto RX key disables for SA1 on the UAL station",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		discardQRXDroppedPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_discard_q_rx_dropped_packets",
				Help: "Discard Q Rx dropped packets",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		nonifoeQRXTotalPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_nonifoe_q_rx_total_packets",
				Help: "Nonifoe Q Rx total packets",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		nonifoeQRXXoffTotal: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_nonifoe_q_rx_xoff_total",
				Help: "Nonifoe Q Rx xoff total",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		nonifoeQTXTotalPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_nonifoe_q_tx_total_packets",
				Help: "Nonifoe Q Tx total packets",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		nonifoeQTXXoffTotal: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_nonifoe_q_tx_xoff_total",
				Help: "Nonifoe Q Tx xoff total",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		reqQRXDroppedPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_req_q_rx_dropped_packets",
				Help: "Req Q Rx dropped packets",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		resQRXDroppedPackets: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_res_q_rx_dropped_packets",
				Help: "Res Q Rx dropped packets",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		rxPausePacketsRcvd: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_rx_pause_packets_rcvd",
				Help: "Rx pause packets rcvd",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		txPausePacketsSent: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_tx_pause_packets_sent",
				Help: "Tx pause packets sent",
			},
			append([]string{"station_uuid", "port_name", "device_uuid", "port_index", "accelerator_id"}, labels...)),
		stationRengFreeBlkOut: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_reng_free_blk_out",
				Help: "Reng free blk out",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRengFreePktOut: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_reng_free_pkt_out",
				Help: "Reng free pkt out",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRengFreeSchIn: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_reng_free_sch_in",
				Help: "Reng free sch in",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRengReadPktOut: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_reng_read_pkt_out",
				Help: "Reng read pkt out",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRengReadSchIn: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_reng_read_sch_in",
				Help: "Reng read sch in",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRXDecapDroppedPkts: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_rx_decap_dropped_pkts",
				Help: "Rx decap dropped pkts",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRXDecapRXNAKEgressReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_rx_decap_rx_nak_egress_req",
				Help: "Rx decap Rx NAK egress req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRXDecapRXNAKEgressRsp: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_rx_decap_rx_nak_egress_rsp",
				Help: "Rx decap Rx NAK egress rsp",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRXDecapTXNAKEgressReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_rx_decap_tx_nak_egress_req",
				Help: "Rx decap Tx NAK egress req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRXDecapTXNAKEgressRsp: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_rx_decap_tx_nak_egress_rsp",
				Help: "Rx decap Tx NAK egress rsp",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRXDroppedPkts: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_rx_dropped_pkts",
				Help: "Rx dropped pkts",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationRXNonifoePkts: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_rx_nonifoe_pkts",
				Help: "Rx nonifoe pkts",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackOrigdataCreditsConsumed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_origdata_credits_consumed",
				Help: "SDP Rx unpack origdata credits consumed",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackOrigdataCreditsReturned: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_origdata_credits_returned",
				Help: "SDP Rx unpack origdata credits returned",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackRdrspCreditsConsumed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_rdrsp_credits_consumed",
				Help: "SDP Rx unpack rdrsp credits consumed",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackRdrspCreditsReturned: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_rdrsp_credits_returned",
				Help: "SDP Rx unpack rdrsp credits returned",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackReqCreditsConsumed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_req_credits_consumed",
				Help: "SDP Rx unpack req credits consumed",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackReqCreditsReturned: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_req_credits_returned",
				Help: "SDP Rx unpack req credits returned",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackReqCyclesStalled: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_req_cycles_stalled",
				Help: "SDP Rx unpack req cycles stalled",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackReqCyclesStalledCnt: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_req_cycles_stalled_cnt",
				Help: "SDP Rx unpack req cycles stalled cnt",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackReqExcessCreditsReturned: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_req_excess_credits_returned",
				Help: "SDP Rx unpack req excess credits returned",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackReqPayloadCreditsReturned: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_req_payload_credits_returned",
				Help: "SDP Rx unpack req payload credits returned",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackReqTotalCreditsConsumed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_req_total_credits_consumed",
				Help: "SDP Rx unpack req total credits consumed",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackRetagFreed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_retag_freed",
				Help: "SDP Rx unpack retag freed",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackRetagUsed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_retag_used",
				Help: "SDP Rx unpack retag used",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackRspCyclesStalled: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_rsp_cycles_stalled",
				Help: "SDP Rx unpack rsp cycles stalled",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackRspCyclesStalledCnt: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_rsp_cycles_stalled_cnt",
				Help: "SDP Rx unpack rsp cycles stalled cnt",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackRspExcessCreditsReturned: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_rsp_excess_credits_returned",
				Help: "SDP Rx unpack rsp excess credits returned",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackRspPayloadCreditsReturned: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_rsp_payload_credits_returned",
				Help: "SDP Rx unpack rsp payload credits returned",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackRspTotalCreditsConsumed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_rsp_total_credits_consumed",
				Help: "SDP Rx unpack rsp total credits consumed",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackWrrspCreditsConsumed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_wrrsp_credits_consumed",
				Help: "SDP Rx unpack wrrsp credits consumed",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPRXUnpackWrrspCreditsReturned: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_rx_unpack_wrrsp_credits_returned",
				Help: "SDP Rx unpack wrrsp credits returned",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackAtmReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_atm_req",
				Help: "SDP Tx pack atm req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackOrigData: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_orig_data",
				Help: "SDP Tx pack orig data",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackOrigDataCreditConsumed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_orig_data_credit_consumed",
				Help: "SDP Tx pack orig data credit consumed",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackOrigDataCreditReleased: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_orig_data_credit_released",
				Help: "SDP Tx pack orig data credit released",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackOrigDataEbEmpty: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_orig_data_eb_empty",
				Help: "SDP Tx pack orig data eb empty",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackOrigDataEbFull: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_orig_data_eb_full",
				Help: "SDP Tx pack orig data eb full",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackOrigDataError: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_orig_data_error",
				Help: "SDP Tx pack orig data error",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackRdReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_rd_req",
				Help: "SDP Tx pack rd req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackRdRsp: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_rd_rsp",
				Help: "SDP Tx pack rd rsp",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackRdRspCreditConsumed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_rd_rsp_credit_consumed",
				Help: "SDP Tx pack rd rsp credit consumed",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackRdRspCreditReleased: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_rd_rsp_credit_released",
				Help: "SDP Tx pack rd rsp credit released",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackRdRspDataError: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_rd_rsp_data_error",
				Help: "SDP Tx pack rd rsp data error",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackRdRspEbEmpty: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_rd_rsp_eb_empty",
				Help: "SDP Tx pack rd rsp eb empty",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackRdRspEbFull: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_rd_rsp_eb_full",
				Help: "SDP Tx pack rd rsp eb full",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackRdRspPclEmpty: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_rd_rsp_pcl_empty",
				Help: "SDP Tx pack rd rsp pcl empty",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackRdRspPclFull: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_rd_rsp_pcl_full",
				Help: "SDP Tx pack rd rsp pcl full",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackReqCreditConsumed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_req_credit_consumed",
				Help: "SDP Tx pack req credit consumed",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackReqCreditReleased: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_req_credit_released",
				Help: "SDP Tx pack req credit released",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackReqEbEmpty: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_req_eb_empty",
				Help: "SDP Tx pack req eb empty",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackReqEbFull: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_req_eb_full",
				Help: "SDP Tx pack req eb full",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackReqPcl: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_req_pcl",
				Help: "SDP Tx pack req pcl",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackReqPclEmpty: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_req_pcl_empty",
				Help: "SDP Tx pack req pcl empty",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackReqPclFull: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_req_pcl_full",
				Help: "SDP Tx pack req pcl full",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackReqPoolEmpty: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_req_pool_empty",
				Help: "SDP Tx pack req pool empty",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackReqPoolFull: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_req_pool_full",
				Help: "SDP Tx pack req pool full",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackRspPoolEmpty: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_rsp_pool_empty",
				Help: "SDP Tx pack rsp pool empty",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackRspPoolFull: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_rsp_pool_full",
				Help: "SDP Tx pack rsp pool full",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackWrReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_wr_req",
				Help: "SDP Tx pack wr req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackWrRsp: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_wr_rsp",
				Help: "SDP Tx pack wr rsp",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackWrRspCreditConsumed: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_wr_rsp_credit_consumed",
				Help: "SDP Tx pack wr rsp credit consumed",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackWrRspCreditReleased: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_wr_rsp_credit_released",
				Help: "SDP Tx pack wr rsp credit released",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackWrRspEbEmpty: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_wr_rsp_eb_empty",
				Help: "SDP Tx pack wr rsp eb empty",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackWrRspEbFull: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_wr_rsp_eb_full",
				Help: "SDP Tx pack wr rsp eb full",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackWrRspPclEmpty: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_wr_rsp_pcl_empty",
				Help: "SDP Tx pack wr rsp pcl empty",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationSDPTXPackWrRspPclFull: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_sdp_tx_pack_wr_rsp_pcl_full",
				Help: "SDP Tx pack wr rsp pcl full",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXEncapPktEgressXrsecNport0: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_encap_pkt_egress_xrsec_nport_0",
				Help: "Tx encap pkt egress xrsec nport 0",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXEncapPktEgressXrsecNport1: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_encap_pkt_egress_xrsec_nport_1",
				Help: "Tx encap pkt egress xrsec nport 1",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXEncapPktEgressXrsecNport2: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_encap_pkt_egress_xrsec_nport_2",
				Help: "Tx encap pkt egress xrsec nport 2",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXEncapPktEgressXrsecNport3: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_encap_pkt_egress_xrsec_nport_3",
				Help: "Tx encap pkt egress xrsec nport 3",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXEncapStallEgressXrsec: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_encap_stall_egress_xrsec",
				Help: "Tx encap stall egress xrsec",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXEncapStallIngress: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_encap_stall_ingress",
				Help: "Tx encap stall ingress",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXNonifoePkts: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_nonifoe_pkts",
				Help: "Tx nonifoe pkts",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXSchedActiveRXACKPkts: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_sched_active_rx_ack_pkts",
				Help: "Tx sched active Rx ACK pkts",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXSchedActiveStreams: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_sched_active_streams",
				Help: "Tx sched active streams",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXSchedBoostedPriStreams: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_sched_boosted_pri_streams",
				Help: "Tx sched boosted pri streams",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXSchedEmptyQueueStreams: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_sched_empty_queue_streams",
				Help: "Tx sched empty queue streams",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXSchedEmptySendQueueStreams: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_sched_empty_send_queue_streams",
				Help: "Tx sched empty send queue streams",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXSchedPausedStreams: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_sched_paused_streams",
				Help: "Tx sched paused streams",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXSchedReqPkts: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_sched_req_pkts",
				Help: "Tx sched req pkts",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationTXSchedResPkts: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_tx_sched_res_pkts",
				Help: "Tx sched res pkts",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengBstateFreeBlkDelayReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_bstate_free_blk_delay_req",
				Help: "Weng bstate free blk delay req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengBstateFreeBlkDelayRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_bstate_free_blk_delay_res",
				Help: "Weng bstate free blk delay res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionChainDelayReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_chain_delay_req",
				Help: "Weng eviction chain delay req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionChainDelayRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_chain_delay_res",
				Help: "Weng eviction chain delay res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionExpiryReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_expiry_req",
				Help: "Weng eviction expiry req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionExpiryRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_expiry_res",
				Help: "Weng eviction expiry res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionForceChainDelayReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_force_chain_delay_req",
				Help: "Weng eviction force chain delay req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionForceChainDelayRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_force_chain_delay_res",
				Help: "Weng eviction force chain delay res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionForceHoldDelayReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_force_hold_delay_req",
				Help: "Weng eviction force hold delay req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionForceHoldDelayRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_force_hold_delay_res",
				Help: "Weng eviction force hold delay res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionForceReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_force_req",
				Help: "Weng eviction force req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionForceRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_force_res",
				Help: "Weng eviction force res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionMtuHoldDelayReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_mtu_hold_delay_req",
				Help: "Weng eviction mtu hold delay req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionMtuHoldDelayRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_mtu_hold_delay_res",
				Help: "Weng eviction mtu hold delay res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionMtuReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_mtu_req",
				Help: "Weng eviction mtu req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionMtuRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_mtu_res",
				Help: "Weng eviction mtu res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionPayloadReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_payload_req",
				Help: "Weng eviction payload req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengEvictionPayloadRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_eviction_payload_res",
				Help: "Weng eviction payload res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPChainReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_chain_req",
				Help: "Weng op SDP chain req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPChainRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_chain_res",
				Help: "Weng op SDP chain res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPForceReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_force_req",
				Help: "Weng op SDP force req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPForceRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_force_res",
				Help: "Weng op SDP force res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPLengthReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_length_req",
				Help: "Weng op SDP length req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPLengthRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_length_res",
				Help: "Weng op SDP length res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPPktOpenReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_pkt_open_req",
				Help: "Weng op SDP pkt open req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPPktOpenRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_pkt_open_res",
				Help: "Weng op SDP pkt open res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_req",
				Help: "Weng op SDP req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_res",
				Help: "Weng op SDP res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPWordReq: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_word_req",
				Help: "Weng op SDP word req",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
		stationWengOpSDPWordRes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ifoe_station_weng_op_sdp_word_res",
				Help: "Weng op SDP word res",
			},
			append([]string{"station_uuid", "device_uuid", "station_index", "accelerator_id"}, labels...)),
	}
	ga.initFieldMetricsMap()
	ga.initTelemetryNameMaps()
}

func (ga *GPUAgentIFOEClient) initFieldMetricsMap() {
	// nolint
	// Alias is the gpuagent telemetry counter name (from generate_telemetry_strings.py).
	// Entries without Alias are set directly from proto status fields, not from
	// the generic UALTelemetryCounter list. The STATION_ prefix distinguishes station-level
	// counters from port-level counters when building the telemetry name maps.
	ga.fieldMetricsMap = map[string]FieldMeta{
		exportermetrics.IFOEMetricField_IFOE_TOTAL_DEVICES.String():                                      {Metric: ga.metrics.totalDevices},
		exportermetrics.IFOEMetricField_IFOE_TOTAL_STATIONS.String():                                     {Metric: ga.metrics.totalStations},
		exportermetrics.IFOEMetricField_IFOE_TOTAL_PORTS.String():                                        {Metric: ga.metrics.totalNetworkPorts},
		exportermetrics.IFOEMetricField_IFOE_NUMBER_FAILEDOVER_STREAMS.String():                          {Metric: ga.metrics.numFailedoverStreams},
		exportermetrics.IFOEMetricField_IFOE_NUMBER_PAUSED_STREAMS.String():                              {Metric: ga.metrics.numPausedStreams},
		exportermetrics.IFOEMetricField_IFOE_BIT_ERROR_RATE.String():                                     {Metric: ga.metrics.bitErrorRate, Alias: "Bit error rate"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS0.String():                        {Metric: ga.metrics.fecCodeWordSymbolErrors0, Alias: "FEC CW symbol errs 0"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS1.String():                        {Metric: ga.metrics.fecCodeWordSymbolErrors1, Alias: "FEC CW symbol errs 1"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS2.String():                        {Metric: ga.metrics.fecCodeWordSymbolErrors2, Alias: "FEC CW symbol errs 2"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS3.String():                        {Metric: ga.metrics.fecCodeWordSymbolErrors3, Alias: "FEC CW symbol errs 3"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS4.String():                        {Metric: ga.metrics.fecCodeWordSymbolErrors4, Alias: "FEC CW symbol errs 4"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS5.String():                        {Metric: ga.metrics.fecCodeWordSymbolErrors5, Alias: "FEC CW symbol errs 5"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS6.String():                        {Metric: ga.metrics.fecCodeWordSymbolErrors6, Alias: "FEC CW symbol errs 6"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS7.String():                        {Metric: ga.metrics.fecCodeWordSymbolErrors7, Alias: "FEC CW symbol errs 7"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS8.String():                        {Metric: ga.metrics.fecCodeWordSymbolErrors8, Alias: "FEC CW symbol errs 8"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS9.String():                        {Metric: ga.metrics.fecCodeWordSymbolErrors9, Alias: "FEC CW symbol errs 9"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS10.String():                       {Metric: ga.metrics.fecCodeWordSymbolErrors10, Alias: "FEC CW symbol errs 10"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS11.String():                       {Metric: ga.metrics.fecCodeWordSymbolErrors11, Alias: "FEC CW symbol errs 11"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS12.String():                       {Metric: ga.metrics.fecCodeWordSymbolErrors12, Alias: "FEC CW symbol errs 12"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS13.String():                       {Metric: ga.metrics.fecCodeWordSymbolErrors13, Alias: "FEC CW symbol errs 13"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS14.String():                       {Metric: ga.metrics.fecCodeWordSymbolErrors14, Alias: "FEC CW symbol errs 14"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS15.String():                       {Metric: ga.metrics.fecCodeWordSymbolErrors15, Alias: "FEC CW symbol errs 15"},
		exportermetrics.IFOEMetricField_IFOE_FEC_CODEWORD_SYMBOL_ERRORS_UNCORRECTABLE.String():           {Metric: ga.metrics.fecCodeWordSymbolErrorsUncorrectable, Alias: "FEC CW symbol errs uncorrectable"},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_STATE.String():                                    {Metric: ga.metrics.portLinkState},
		exportermetrics.IFOEMetricField_IFOE_PORT_SPEED.String():                                         {Metric: ga.metrics.portSpeed},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_UP_COUNT.String():                                 {Metric: ga.metrics.portLinkUpCount},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_UP_DURATION_MSEC.String():                         {Metric: ga.metrics.portLinkUpDurationMsec},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_DOWN_DURATION_MSEC.String():                       {Metric: ga.metrics.portLinkDownDurationMsec},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_TRAINING_DURATION_LATEST_MSEC.String():            {Metric: ga.metrics.portLinkTrainingDurationLatestMsec},
		exportermetrics.IFOEMetricField_IFOE_PORT_LINK_TRAINING_DURATION_AVG_MSEC.String():               {Metric: ga.metrics.portLinkTrainingDurationAvgMsec},
		exportermetrics.IFOEMetricField_IFOE_TX_TOTAL_BYTES.String():                                     {Metric: ga.metrics.txTotalBytes, Alias: "Tx total bytes"},
		exportermetrics.IFOEMetricField_IFOE_TX_TOTAL_GOOD_BYTES.String():                                {Metric: ga.metrics.txTotalGoodBytes, Alias: "Tx total good bytes"},
		exportermetrics.IFOEMetricField_IFOE_TX_TOTAL_ERR_BYTES.String():                                 {Metric: ga.metrics.txTotalErrBytes, Alias: "Tx total ERR bytes"},
		exportermetrics.IFOEMetricField_IFOE_TX_TOTAL_PACKETS.String():                                   {Metric: ga.metrics.txTotalPackets, Alias: "Tx total packets"},
		exportermetrics.IFOEMetricField_IFOE_TX_TOTAL_GOOD_PACKETS.String():                              {Metric: ga.metrics.txTotalGoodPackets, Alias: "Tx total good packets"},
		exportermetrics.IFOEMetricField_IFOE_TX_FRAME_ERROR.String():                                     {Metric: ga.metrics.txFrameError, Alias: "Tx frame error"},
		exportermetrics.IFOEMetricField_IFOE_TX_BAD_FCS.String():                                         {Metric: ga.metrics.txBadFCS, Alias: "Tx BAD FCS"},
		exportermetrics.IFOEMetricField_IFOE_RX_TOTAL_BYTES.String():                                     {Metric: ga.metrics.rxTotalBytes, Alias: "Rx total bytes"},
		exportermetrics.IFOEMetricField_IFOE_RX_TOTAL_GOOD_BYTES.String():                                {Metric: ga.metrics.rxTotalGoodBytes, Alias: "Rx total good bytes"},
		exportermetrics.IFOEMetricField_IFOE_RX_TOTAL_ERR_BYTES.String():                                 {Metric: ga.metrics.rxTotalErrBytes, Alias: "Rx total ERR bytes"},
		exportermetrics.IFOEMetricField_IFOE_RX_TOTAL_PACKETS.String():                                   {Metric: ga.metrics.rxTotalPackets, Alias: "Rx total packets"},
		exportermetrics.IFOEMetricField_IFOE_RX_TOTAL_GOOD_PACKETS.String():                              {Metric: ga.metrics.rxTotalGoodPackets, Alias: "Rx total good packets"},
		exportermetrics.IFOEMetricField_IFOE_RX_PACKET_DROPPED.String():                                  {Metric: ga.metrics.rxPacketDropped, Alias: "Rx packet dropped"},
		exportermetrics.IFOEMetricField_IFOE_RX_BAD_FCS.String():                                         {Metric: ga.metrics.rxBadFCS, Alias: "Rx BAD FCS"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_CORRECTED_CODEWORDS.String():                         {Metric: ga.metrics.rxFECCorrectedCodewords, Alias: "Rx FEC corrected CW"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_UNCORRECTED_CODEWORDS.String():                       {Metric: ga.metrics.rxFECUncorrectedCodewords, Alias: "Rx FEC uncorrected CW"},
		exportermetrics.IFOEMetricField_IFOE_TX_PAUSE.String():                                           {Metric: ga.metrics.txPause, Alias: "Tx pause"},
		exportermetrics.IFOEMetricField_IFOE_RX_PAUSE.String():                                           {Metric: ga.metrics.rxPause, Alias: "Rx pause"},
		exportermetrics.IFOEMetricField_IFOE_TX_USER_PAUSE.String():                                      {Metric: ga.metrics.txUserPause, Alias: "Tx user pause"},
		exportermetrics.IFOEMetricField_IFOE_RX_USER_PAUSE.String():                                      {Metric: ga.metrics.rxUserPause, Alias: "Rx user pause"},
		exportermetrics.IFOEMetricField_IFOE_RX_JABBER.String():                                          {Metric: ga.metrics.rxJabber, Alias: "Rx jabber"},
		exportermetrics.IFOEMetricField_IFOE_RX_OVERSIZE.String():                                        {Metric: ga.metrics.rxOversize, Alias: "Rx oversize"},
		exportermetrics.IFOEMetricField_IFOE_RX_TOO_LONG.String():                                        {Metric: ga.metrics.rxTooLong, Alias: "Rx toolong"},
		exportermetrics.IFOEMetricField_IFOE_RX_TRUNCATED.String():                                       {Metric: ga.metrics.rxTruncated, Alias: "Rx truncated"},
		exportermetrics.IFOEMetricField_IFOE_TX_LLR_OK_PACKETS.String():                                  {Metric: ga.metrics.txLLROkPackets, Alias: "Tx LLR OK packets"},
		exportermetrics.IFOEMetricField_IFOE_RX_LLR_OK_PACKETS.String():                                  {Metric: ga.metrics.rxLLROkPackets, Alias: "Rx LLR OK packets"},
		exportermetrics.IFOEMetricField_IFOE_TX_LLR_REPLAY_COUNT.String():                                {Metric: ga.metrics.txLLRReplayCount, Alias: "Tx LLR replay ct"},
		exportermetrics.IFOEMetricField_IFOE_TX_LLR_REPLAYS_COMPLETED.String():                           {Metric: ga.metrics.txLLRReplaysCompleted, Alias: "Tx LLR replays completed"},
		exportermetrics.IFOEMetricField_IFOE_RX_LLR_BAD_PACKETS.String():                                 {Metric: ga.metrics.rxLLRBadPackets, Alias: "Rx LLR BAD packets"},
		exportermetrics.IFOEMetricField_IFOE_RX_LLR_DUPL_SEQ_PACKETS.String():                            {Metric: ga.metrics.rxLLRDuplSeqPackets, Alias: "Rx LLR dupl SEQ packets"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_BIT_ERR_0TO1_LANE0.String():                          {Metric: ga.metrics.rxFECBitErr0To1Lane0, Alias: "Rx FEC bit ERR 0to1 L0"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_BIT_ERR_0TO1_LANE1.String():                          {Metric: ga.metrics.rxFECBitErr0To1Lane1, Alias: "Rx FEC bit ERR 0to1 L1"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_BIT_ERR_0TO1_LANE2.String():                          {Metric: ga.metrics.rxFECBitErr0To1Lane2, Alias: "Rx FEC bit ERR 0to1 L2"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_BIT_ERR_0TO1_LANE3.String():                          {Metric: ga.metrics.rxFECBitErr0To1Lane3, Alias: "Rx FEC bit ERR 0to1 L3"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_BIT_ERR_1TO0_LANE0.String():                          {Metric: ga.metrics.rxFECBitErr1To0Lane0, Alias: "Rx FEC bit ERR 1to0 L0 LSB"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_BIT_ERR_1TO0_LANE1.String():                          {Metric: ga.metrics.rxFECBitErr1To0Lane1, Alias: "Rx FEC bit ERR 1to0 L1 LSB"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_BIT_ERR_1TO0_LANE2.String():                          {Metric: ga.metrics.rxFECBitErr1To0Lane2, Alias: "Rx FEC bit ERR 1to0 L2 LSB"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_BIT_ERR_1TO0_LANE3.String():                          {Metric: ga.metrics.rxFECBitErr1To0Lane3, Alias: "Rx FEC bit ERR 1to0 L3 LSB"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE0.String():                      {Metric: ga.metrics.rxFECSymbolErrCountLane0, Alias: "Rx FEC ERR count L0 LSB"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE1.String():                      {Metric: ga.metrics.rxFECSymbolErrCountLane1, Alias: "Rx FEC ERR count L1 LSB"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE2.String():                      {Metric: ga.metrics.rxFECSymbolErrCountLane2, Alias: "Rx FEC ERR count L2 LSB"},
		exportermetrics.IFOEMetricField_IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE3.String():                      {Metric: ga.metrics.rxFECSymbolErrCountLane3, Alias: "Rx FEC ERR count L3 LSB"},
		exportermetrics.IFOEMetricField_IFOE_RX_BAD_CODE_COUNT.String():                                  {Metric: ga.metrics.rxBadCodeCount, Alias: "Rx BAD code count"},
		exportermetrics.IFOEMetricField_IFOE_RX_STOMPED_FCS.String():                                     {Metric: ga.metrics.rxStompedFCS, Alias: "Rx stomped FCS"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_REQUEST_PACKETS.String():                         {Metric: ga.metrics.stationTxRequestPackets, Alias: "Tx request pkt count"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_RESPONSE_PACKETS.String():                        {Metric: ga.metrics.stationTxResponsePackets, Alias: "Tx response pkt count"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RX_REQUEST_PACKETS.String():                         {Metric: ga.metrics.stationRxRequestPackets, Alias: "Rx request pkt count"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RX_RESPONSE_PACKETS.String():                        {Metric: ga.metrics.stationRxResponsePackets, Alias: "Rx response pkt count"},
		exportermetrics.IFOEMetricField_IFOE_STATION_STREAM_REMAPS_TOTAL.String():                        {Metric: ga.metrics.stationStreamRemapsTotal, Alias: "Stream remaps total"},
		exportermetrics.IFOEMetricField_IFOE_STATION_PAUSED_STREAMS_COUNT.String():                       {Metric: ga.metrics.stationPausedStreamsCount, Alias: "Paused streams count"},
		exportermetrics.IFOEMetricField_IFOE_STATION_STREAM_REMAPS_NETWORK_PORT0.String():                {Metric: ga.metrics.stationStreamRemapsNetworkPort0, Alias: "Stream remaps nport 0"},
		exportermetrics.IFOEMetricField_IFOE_STATION_STREAM_REMAPS_NETWORK_PORT1.String():                {Metric: ga.metrics.stationStreamRemapsNetworkPort1, Alias: "Stream remaps nport 1"},
		exportermetrics.IFOEMetricField_IFOE_STATION_STREAM_REMAPS_NETWORK_PORT2.String():                {Metric: ga.metrics.stationStreamRemapsNetworkPort2, Alias: "Stream remaps nport 2"},
		exportermetrics.IFOEMetricField_IFOE_STATION_STREAM_REMAPS_NETWORK_PORT3.String():                {Metric: ga.metrics.stationStreamRemapsNetworkPort3, Alias: "Stream remaps nport 3"},
		exportermetrics.IFOEMetricField_IFOE_STATION_CRYPTO_TX_KEY_UPDATES_SA0.String():                  {Metric: ga.metrics.stationCryptoTxKeyUpdatesSA0, Alias: "Crypto Tx key updates sa0"},
		exportermetrics.IFOEMetricField_IFOE_STATION_CRYPTO_RX_KEY0_UPDATES_SA0.String():                 {Metric: ga.metrics.stationCryptoRxKey0UpdatesSA0, Alias: "Crypto Rx key0 updates sa0"},
		exportermetrics.IFOEMetricField_IFOE_STATION_CRYPTO_RX_KEY1_UPDATES_SA0.String():                 {Metric: ga.metrics.stationCryptoRxKey1UpdatesSA0, Alias: "Crypto Rx key1 updates sa0"},
		exportermetrics.IFOEMetricField_IFOE_STATION_CRYPTO_RX_KEY_DISABLES_SA0.String():                 {Metric: ga.metrics.stationCryptoRxKeyDisablesSA0, Alias: "Crypto Rx key disables sa0"},
		exportermetrics.IFOEMetricField_IFOE_STATION_CRYPTO_TX_KEY_UPDATES_SA1.String():                  {Metric: ga.metrics.stationCryptoTxKeyUpdatesSA1, Alias: "Crypto Tx key updates sa1"},
		exportermetrics.IFOEMetricField_IFOE_STATION_CRYPTO_RX_KEY0_UPDATES_SA1.String():                 {Metric: ga.metrics.stationCryptoRxKey0UpdatesSA1, Alias: "Crypto Rx key0 updates sa1"},
		exportermetrics.IFOEMetricField_IFOE_STATION_CRYPTO_RX_KEY1_UPDATES_SA1.String():                 {Metric: ga.metrics.stationCryptoRxKey1UpdatesSA1, Alias: "Crypto Rx key1 updates sa1"},
		exportermetrics.IFOEMetricField_IFOE_STATION_CRYPTO_RX_KEY_DISABLES_SA1.String():                 {Metric: ga.metrics.stationCryptoRxKeyDisablesSA1, Alias: "Crypto Rx key disables sa1"},
		exportermetrics.IFOEMetricField_IFOE_DISCARD_Q_RX_DROPPED_PACKETS.String():                       {Metric: ga.metrics.discardQRXDroppedPackets, Alias: "Discard Q Rx dropped packets"},
		exportermetrics.IFOEMetricField_IFOE_NONIFOE_Q_RX_TOTAL_PACKETS.String():                         {Metric: ga.metrics.nonifoeQRXTotalPackets, Alias: "Nonifoe Q Rx total packets"},
		exportermetrics.IFOEMetricField_IFOE_NONIFOE_Q_RX_XOFF_TOTAL.String():                            {Metric: ga.metrics.nonifoeQRXXoffTotal, Alias: "Nonifoe Q Rx xoff total"},
		exportermetrics.IFOEMetricField_IFOE_NONIFOE_Q_TX_TOTAL_PACKETS.String():                         {Metric: ga.metrics.nonifoeQTXTotalPackets, Alias: "Nonifoe Q Tx total packets"},
		exportermetrics.IFOEMetricField_IFOE_NONIFOE_Q_TX_XOFF_TOTAL.String():                            {Metric: ga.metrics.nonifoeQTXXoffTotal, Alias: "Nonifoe Q Tx xoff total"},
		exportermetrics.IFOEMetricField_IFOE_REQ_Q_RX_DROPPED_PACKETS.String():                           {Metric: ga.metrics.reqQRXDroppedPackets, Alias: "Req Q Rx dropped packets"},
		exportermetrics.IFOEMetricField_IFOE_RES_Q_RX_DROPPED_PACKETS.String():                           {Metric: ga.metrics.resQRXDroppedPackets, Alias: "Res Q Rx dropped packets"},
		exportermetrics.IFOEMetricField_IFOE_RX_PAUSE_PACKETS_RCVD.String():                              {Metric: ga.metrics.rxPausePacketsRcvd, Alias: "Rx pause packets rcvd"},
		exportermetrics.IFOEMetricField_IFOE_TX_PAUSE_PACKETS_SENT.String():                              {Metric: ga.metrics.txPausePacketsSent, Alias: "Tx pause packets sent"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RENG_FREE_BLK_OUT.String():                          {Metric: ga.metrics.stationRengFreeBlkOut, Alias: "Reng free blk out"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RENG_FREE_PKT_OUT.String():                          {Metric: ga.metrics.stationRengFreePktOut, Alias: "Reng free pkt out"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RENG_FREE_SCH_IN.String():                           {Metric: ga.metrics.stationRengFreeSchIn, Alias: "Reng free sch in"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RENG_READ_PKT_OUT.String():                          {Metric: ga.metrics.stationRengReadPktOut, Alias: "Reng read pkt out"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RENG_READ_SCH_IN.String():                           {Metric: ga.metrics.stationRengReadSchIn, Alias: "Reng read sch in"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RX_DECAP_DROPPED_PKTS.String():                      {Metric: ga.metrics.stationRXDecapDroppedPkts, Alias: "Rx decap dropped pkts"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RX_DECAP_RX_NAK_EGRESS_REQ.String():                 {Metric: ga.metrics.stationRXDecapRXNAKEgressReq, Alias: "Rx decap Rx NAK egress req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RX_DECAP_RX_NAK_EGRESS_RSP.String():                 {Metric: ga.metrics.stationRXDecapRXNAKEgressRsp, Alias: "Rx decap Rx NAK egress rsp"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RX_DECAP_TX_NAK_EGRESS_REQ.String():                 {Metric: ga.metrics.stationRXDecapTXNAKEgressReq, Alias: "Rx decap Tx NAK egress req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RX_DECAP_TX_NAK_EGRESS_RSP.String():                 {Metric: ga.metrics.stationRXDecapTXNAKEgressRsp, Alias: "Rx decap Tx NAK egress rsp"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RX_DROPPED_PKTS.String():                            {Metric: ga.metrics.stationRXDroppedPkts, Alias: "Rx dropped pkts"},
		exportermetrics.IFOEMetricField_IFOE_STATION_RX_NONIFOE_PKTS.String():                            {Metric: ga.metrics.stationRXNonifoePkts, Alias: "Rx nonifoe pkts"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_ORIGDATA_CREDITS_CONSUMED.String():    {Metric: ga.metrics.stationSDPRXUnpackOrigdataCreditsConsumed, Alias: "SDP Rx unpack origdata credits consumed"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_ORIGDATA_CREDITS_RETURNED.String():    {Metric: ga.metrics.stationSDPRXUnpackOrigdataCreditsReturned, Alias: "SDP Rx unpack origdata credits returned"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_RDRSP_CREDITS_CONSUMED.String():       {Metric: ga.metrics.stationSDPRXUnpackRdrspCreditsConsumed, Alias: "SDP Rx unpack rdrsp credits consumed"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_RDRSP_CREDITS_RETURNED.String():       {Metric: ga.metrics.stationSDPRXUnpackRdrspCreditsReturned, Alias: "SDP Rx unpack rdrsp credits returned"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_REQ_CREDITS_CONSUMED.String():         {Metric: ga.metrics.stationSDPRXUnpackReqCreditsConsumed, Alias: "SDP Rx unpack req credits consumed"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_REQ_CREDITS_RETURNED.String():         {Metric: ga.metrics.stationSDPRXUnpackReqCreditsReturned, Alias: "SDP Rx unpack req credits returned"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_REQ_CYCLES_STALLED.String():           {Metric: ga.metrics.stationSDPRXUnpackReqCyclesStalled, Alias: "SDP Rx unpack req cycles stalled"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_REQ_CYCLES_STALLED_CNT.String():       {Metric: ga.metrics.stationSDPRXUnpackReqCyclesStalledCnt, Alias: "SDP Rx unpack req cycles stalled cnt"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_REQ_EXCESS_CREDITS_RETURNED.String():  {Metric: ga.metrics.stationSDPRXUnpackReqExcessCreditsReturned, Alias: "SDP Rx unpack req excess credits returned"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_REQ_PAYLOAD_CREDITS_RETURNED.String(): {Metric: ga.metrics.stationSDPRXUnpackReqPayloadCreditsReturned, Alias: "SDP Rx unpack req payload credits returned"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_REQ_TOTAL_CREDITS_CONSUMED.String():   {Metric: ga.metrics.stationSDPRXUnpackReqTotalCreditsConsumed, Alias: "SDP Rx unpack req total credits consumed"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_RETAG_FREED.String():                  {Metric: ga.metrics.stationSDPRXUnpackRetagFreed, Alias: "SDP Rx unpack retag freed"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_RETAG_USED.String():                   {Metric: ga.metrics.stationSDPRXUnpackRetagUsed, Alias: "SDP Rx unpack retag used"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_RSP_CYCLES_STALLED.String():           {Metric: ga.metrics.stationSDPRXUnpackRspCyclesStalled, Alias: "SDP Rx unpack rsp cycles stalled"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_RSP_CYCLES_STALLED_CNT.String():       {Metric: ga.metrics.stationSDPRXUnpackRspCyclesStalledCnt, Alias: "SDP Rx unpack rsp cycles stalled cnt"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_RSP_EXCESS_CREDITS_RETURNED.String():  {Metric: ga.metrics.stationSDPRXUnpackRspExcessCreditsReturned, Alias: "SDP Rx unpack rsp excess credits returned"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_RSP_PAYLOAD_CREDITS_RETURNED.String(): {Metric: ga.metrics.stationSDPRXUnpackRspPayloadCreditsReturned, Alias: "SDP Rx unpack rsp payload credits returned"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_RSP_TOTAL_CREDITS_CONSUMED.String():   {Metric: ga.metrics.stationSDPRXUnpackRspTotalCreditsConsumed, Alias: "SDP Rx unpack rsp total credits consumed"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_WRRSP_CREDITS_CONSUMED.String():       {Metric: ga.metrics.stationSDPRXUnpackWrrspCreditsConsumed, Alias: "SDP Rx unpack wrrsp credits consumed"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_RX_UNPACK_WRRSP_CREDITS_RETURNED.String():       {Metric: ga.metrics.stationSDPRXUnpackWrrspCreditsReturned, Alias: "SDP Rx unpack wrrsp credits returned"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_ATM_REQ.String():                        {Metric: ga.metrics.stationSDPTXPackAtmReq, Alias: "SDP Tx pack atm req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_ORIG_DATA.String():                      {Metric: ga.metrics.stationSDPTXPackOrigData, Alias: "SDP Tx pack orig data"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_ORIG_DATA_CREDIT_CONSUMED.String():      {Metric: ga.metrics.stationSDPTXPackOrigDataCreditConsumed, Alias: "SDP Tx pack orig data credit consumed"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_ORIG_DATA_CREDIT_RELEASED.String():      {Metric: ga.metrics.stationSDPTXPackOrigDataCreditReleased, Alias: "SDP Tx pack orig data credit released"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_ORIG_DATA_EB_EMPTY.String():             {Metric: ga.metrics.stationSDPTXPackOrigDataEbEmpty, Alias: "SDP Tx pack orig data eb empty"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_ORIG_DATA_EB_FULL.String():              {Metric: ga.metrics.stationSDPTXPackOrigDataEbFull, Alias: "SDP Tx pack orig data eb full"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_ORIG_DATA_ERROR.String():                {Metric: ga.metrics.stationSDPTXPackOrigDataError, Alias: "SDP Tx pack orig data error"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_RD_REQ.String():                         {Metric: ga.metrics.stationSDPTXPackRdReq, Alias: "SDP Tx pack rd req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_RD_RSP.String():                         {Metric: ga.metrics.stationSDPTXPackRdRsp, Alias: "SDP Tx pack rd rsp"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_RD_RSP_CREDIT_CONSUMED.String():         {Metric: ga.metrics.stationSDPTXPackRdRspCreditConsumed, Alias: "SDP Tx pack rd rsp credit consumed"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_RD_RSP_CREDIT_RELEASED.String():         {Metric: ga.metrics.stationSDPTXPackRdRspCreditReleased, Alias: "SDP Tx pack rd rsp credit released"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_RD_RSP_DATA_ERROR.String():              {Metric: ga.metrics.stationSDPTXPackRdRspDataError, Alias: "SDP Tx pack rd rsp data error"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_RD_RSP_EB_EMPTY.String():                {Metric: ga.metrics.stationSDPTXPackRdRspEbEmpty, Alias: "SDP Tx pack rd rsp eb empty"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_RD_RSP_EB_FULL.String():                 {Metric: ga.metrics.stationSDPTXPackRdRspEbFull, Alias: "SDP Tx pack rd rsp eb full"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_RD_RSP_PCL_EMPTY.String():               {Metric: ga.metrics.stationSDPTXPackRdRspPclEmpty, Alias: "SDP Tx pack rd rsp pcl empty"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_RD_RSP_PCL_FULL.String():                {Metric: ga.metrics.stationSDPTXPackRdRspPclFull, Alias: "SDP Tx pack rd rsp pcl full"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_REQ_CREDIT_CONSUMED.String():            {Metric: ga.metrics.stationSDPTXPackReqCreditConsumed, Alias: "SDP Tx pack req credit consumed"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_REQ_CREDIT_RELEASED.String():            {Metric: ga.metrics.stationSDPTXPackReqCreditReleased, Alias: "SDP Tx pack req credit released"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_REQ_EB_EMPTY.String():                   {Metric: ga.metrics.stationSDPTXPackReqEbEmpty, Alias: "SDP Tx pack req eb empty"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_REQ_EB_FULL.String():                    {Metric: ga.metrics.stationSDPTXPackReqEbFull, Alias: "SDP Tx pack req eb full"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_REQ_PCL.String():                        {Metric: ga.metrics.stationSDPTXPackReqPcl, Alias: "SDP Tx pack req pcl"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_REQ_PCL_EMPTY.String():                  {Metric: ga.metrics.stationSDPTXPackReqPclEmpty, Alias: "SDP Tx pack req pcl empty"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_REQ_PCL_FULL.String():                   {Metric: ga.metrics.stationSDPTXPackReqPclFull, Alias: "SDP Tx pack req pcl full"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_REQ_POOL_EMPTY.String():                 {Metric: ga.metrics.stationSDPTXPackReqPoolEmpty, Alias: "SDP Tx pack req pool empty"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_REQ_POOL_FULL.String():                  {Metric: ga.metrics.stationSDPTXPackReqPoolFull, Alias: "SDP Tx pack req pool full"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_RSP_POOL_EMPTY.String():                 {Metric: ga.metrics.stationSDPTXPackRspPoolEmpty, Alias: "SDP Tx pack rsp pool empty"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_RSP_POOL_FULL.String():                  {Metric: ga.metrics.stationSDPTXPackRspPoolFull, Alias: "SDP Tx pack rsp pool full"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_WR_REQ.String():                         {Metric: ga.metrics.stationSDPTXPackWrReq, Alias: "SDP Tx pack wr req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_WR_RSP.String():                         {Metric: ga.metrics.stationSDPTXPackWrRsp, Alias: "SDP Tx pack wr rsp"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_WR_RSP_CREDIT_CONSUMED.String():         {Metric: ga.metrics.stationSDPTXPackWrRspCreditConsumed, Alias: "SDP Tx pack wr rsp credit consumed"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_WR_RSP_CREDIT_RELEASED.String():         {Metric: ga.metrics.stationSDPTXPackWrRspCreditReleased, Alias: "SDP Tx pack wr rsp credit released"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_WR_RSP_EB_EMPTY.String():                {Metric: ga.metrics.stationSDPTXPackWrRspEbEmpty, Alias: "SDP Tx pack wr rsp eb empty"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_WR_RSP_EB_FULL.String():                 {Metric: ga.metrics.stationSDPTXPackWrRspEbFull, Alias: "SDP Tx pack wr rsp eb full"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_WR_RSP_PCL_EMPTY.String():               {Metric: ga.metrics.stationSDPTXPackWrRspPclEmpty, Alias: "SDP Tx pack wr rsp pcl empty"},
		exportermetrics.IFOEMetricField_IFOE_STATION_SDP_TX_PACK_WR_RSP_PCL_FULL.String():                {Metric: ga.metrics.stationSDPTXPackWrRspPclFull, Alias: "SDP Tx pack wr rsp pcl full"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_ENCAP_PKT_EGRESS_XRSEC_NPORT_0.String():          {Metric: ga.metrics.stationTXEncapPktEgressXrsecNport0, Alias: "Tx encap pkt egress xrsec nport 0"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_ENCAP_PKT_EGRESS_XRSEC_NPORT_1.String():          {Metric: ga.metrics.stationTXEncapPktEgressXrsecNport1, Alias: "Tx encap pkt egress xrsec nport 1"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_ENCAP_PKT_EGRESS_XRSEC_NPORT_2.String():          {Metric: ga.metrics.stationTXEncapPktEgressXrsecNport2, Alias: "Tx encap pkt egress xrsec nport 2"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_ENCAP_PKT_EGRESS_XRSEC_NPORT_3.String():          {Metric: ga.metrics.stationTXEncapPktEgressXrsecNport3, Alias: "Tx encap pkt egress xrsec nport 3"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_ENCAP_STALL_EGRESS_XRSEC.String():                {Metric: ga.metrics.stationTXEncapStallEgressXrsec, Alias: "Tx encap stall egress xrsec"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_ENCAP_STALL_INGRESS.String():                     {Metric: ga.metrics.stationTXEncapStallIngress, Alias: "Tx encap stall ingress"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_NONIFOE_PKTS.String():                            {Metric: ga.metrics.stationTXNonifoePkts, Alias: "Tx nonifoe pkts"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_SCHED_ACTIVE_RX_ACK_PKTS.String():                {Metric: ga.metrics.stationTXSchedActiveRXACKPkts, Alias: "Tx sched active Rx ACK pkts"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_SCHED_ACTIVE_STREAMS.String():                    {Metric: ga.metrics.stationTXSchedActiveStreams, Alias: "Tx sched active streams"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_SCHED_BOOSTED_PRI_STREAMS.String():               {Metric: ga.metrics.stationTXSchedBoostedPriStreams, Alias: "Tx sched boosted pri streams"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_SCHED_EMPTY_QUEUE_STREAMS.String():               {Metric: ga.metrics.stationTXSchedEmptyQueueStreams, Alias: "Tx sched empty queue streams"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_SCHED_EMPTY_SEND_QUEUE_STREAMS.String():          {Metric: ga.metrics.stationTXSchedEmptySendQueueStreams, Alias: "Tx sched empty send queue streams"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_SCHED_PAUSED_STREAMS.String():                    {Metric: ga.metrics.stationTXSchedPausedStreams, Alias: "Tx sched paused streams"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_SCHED_REQ_PKTS.String():                          {Metric: ga.metrics.stationTXSchedReqPkts, Alias: "Tx sched req pkts"},
		exportermetrics.IFOEMetricField_IFOE_STATION_TX_SCHED_RES_PKTS.String():                          {Metric: ga.metrics.stationTXSchedResPkts, Alias: "Tx sched res pkts"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_BSTATE_FREE_BLK_DELAY_REQ.String():             {Metric: ga.metrics.stationWengBstateFreeBlkDelayReq, Alias: "Weng bstate free blk delay req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_BSTATE_FREE_BLK_DELAY_RES.String():             {Metric: ga.metrics.stationWengBstateFreeBlkDelayRes, Alias: "Weng bstate free blk delay res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_CHAIN_DELAY_REQ.String():              {Metric: ga.metrics.stationWengEvictionChainDelayReq, Alias: "Weng eviction chain delay req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_CHAIN_DELAY_RES.String():              {Metric: ga.metrics.stationWengEvictionChainDelayRes, Alias: "Weng eviction chain delay res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_EXPIRY_REQ.String():                   {Metric: ga.metrics.stationWengEvictionExpiryReq, Alias: "Weng eviction expiry req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_EXPIRY_RES.String():                   {Metric: ga.metrics.stationWengEvictionExpiryRes, Alias: "Weng eviction expiry res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_FORCE_CHAIN_DELAY_REQ.String():        {Metric: ga.metrics.stationWengEvictionForceChainDelayReq, Alias: "Weng eviction force chain delay req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_FORCE_CHAIN_DELAY_RES.String():        {Metric: ga.metrics.stationWengEvictionForceChainDelayRes, Alias: "Weng eviction force chain delay res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_FORCE_HOLD_DELAY_REQ.String():         {Metric: ga.metrics.stationWengEvictionForceHoldDelayReq, Alias: "Weng eviction force hold delay req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_FORCE_HOLD_DELAY_RES.String():         {Metric: ga.metrics.stationWengEvictionForceHoldDelayRes, Alias: "Weng eviction force hold delay res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_FORCE_REQ.String():                    {Metric: ga.metrics.stationWengEvictionForceReq, Alias: "Weng eviction force req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_FORCE_RES.String():                    {Metric: ga.metrics.stationWengEvictionForceRes, Alias: "Weng eviction force res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_MTU_HOLD_DELAY_REQ.String():           {Metric: ga.metrics.stationWengEvictionMtuHoldDelayReq, Alias: "Weng eviction mtu hold delay req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_MTU_HOLD_DELAY_RES.String():           {Metric: ga.metrics.stationWengEvictionMtuHoldDelayRes, Alias: "Weng eviction mtu hold delay res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_MTU_REQ.String():                      {Metric: ga.metrics.stationWengEvictionMtuReq, Alias: "Weng eviction mtu req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_MTU_RES.String():                      {Metric: ga.metrics.stationWengEvictionMtuRes, Alias: "Weng eviction mtu res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_PAYLOAD_REQ.String():                  {Metric: ga.metrics.stationWengEvictionPayloadReq, Alias: "Weng eviction payload req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_EVICTION_PAYLOAD_RES.String():                  {Metric: ga.metrics.stationWengEvictionPayloadRes, Alias: "Weng eviction payload res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_CHAIN_REQ.String():                      {Metric: ga.metrics.stationWengOpSDPChainReq, Alias: "Weng op SDP chain req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_CHAIN_RES.String():                      {Metric: ga.metrics.stationWengOpSDPChainRes, Alias: "Weng op SDP chain res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_FORCE_REQ.String():                      {Metric: ga.metrics.stationWengOpSDPForceReq, Alias: "Weng op SDP force req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_FORCE_RES.String():                      {Metric: ga.metrics.stationWengOpSDPForceRes, Alias: "Weng op SDP force res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_LENGTH_REQ.String():                     {Metric: ga.metrics.stationWengOpSDPLengthReq, Alias: "Weng op SDP length req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_LENGTH_RES.String():                     {Metric: ga.metrics.stationWengOpSDPLengthRes, Alias: "Weng op SDP length res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_PKT_OPEN_REQ.String():                   {Metric: ga.metrics.stationWengOpSDPPktOpenReq, Alias: "Weng op SDP pkt open req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_PKT_OPEN_RES.String():                   {Metric: ga.metrics.stationWengOpSDPPktOpenRes, Alias: "Weng op SDP pkt open res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_REQ.String():                            {Metric: ga.metrics.stationWengOpSDPReq, Alias: "Weng op SDP req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_RES.String():                            {Metric: ga.metrics.stationWengOpSDPRes, Alias: "Weng op SDP res"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_WORD_REQ.String():                       {Metric: ga.metrics.stationWengOpSDPWordReq, Alias: "Weng op SDP word req"},
		exportermetrics.IFOEMetricField_IFOE_STATION_WENG_OP_SDP_WORD_RES.String():                       {Metric: ga.metrics.stationWengOpSDPWordRes, Alias: "Weng op SDP word res"},
	}
}

// initTelemetryNameMaps derives the display-name → gauge lookup maps from fieldMetricsMap.
// IFOE_STATION_ prefixed entries go to stationStatsNameMap, the rest to portStatsNameMap.
// Entries without Alias (status fields, aggregates) are skipped.
func (ga *GPUAgentIFOEClient) initTelemetryNameMaps() {
	ga.portStatsNameMap = make(map[string]*prometheus.GaugeVec)
	ga.stationStatsNameMap = make(map[string]*prometheus.GaugeVec)
	for field, meta := range ga.fieldMetricsMap {
		if meta.Alias == "" {
			continue
		}
		// Safe: GaugeVec wraps *MetricVec, so the copy shares underlying metric state
		gauge := meta.Metric
		if strings.HasPrefix(field, "IFOE_STATION_") {
			ga.stationStatsNameMap[meta.Alias] = &gauge
		} else {
			ga.portStatsNameMap[meta.Alias] = &gauge
		}
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
