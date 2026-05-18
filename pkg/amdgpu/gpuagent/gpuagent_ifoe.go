/*
*
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
*
*/
package gpuagent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ROCm/device-metrics-exporter/pkg/amdgpu/gen/amdgpu"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/gen/exportermetrics"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/globals"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/logger"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/utils"
	"github.com/ROCm/device-metrics-exporter/pkg/types"
)

type GPUAgentIFOEClient struct {
	sync.Mutex
	gpuHandler             *GPUAgentClient
	metrics                *IFOEMetrics
	ualClient              amdgpu.UALSvcClient
	exportLabels           map[string]bool
	exportFieldMap         map[string]bool
	customLabelMap         map[string]string
	computeNodeHealthState bool
	allowedCustomLabels    []string
	fl                     *fieldLogger
	extraPodLabelsMap      map[string]string
	k8PodInfoMap           map[string]types.K8sPodInfo
	fieldMetricsMap        map[string]FieldMeta
	staticHostLabels       map[string]string
	podInfoEnabled         bool

	// Capability gate (GPUOP-736): when gpuagent reports zero IFOE devices,
	// the client permanently disables itself for the rest of the process
	// lifetime — subsequent updateMetrics calls short-circuit, no IFOE metric
	// series are published, and the structured "IFOE disabled" log fires
	// exactly once.
	disabled     bool
	disabledOnce sync.Once
}

func NewGPUAgentIFOEClient(gpuHandler *GPUAgentClient) (*GPUAgentIFOEClient, error) {
	ifoeClient := &GPUAgentIFOEClient{
		gpuHandler:     gpuHandler,
		exportLabels:   map[string]bool{},
		customLabelMap: map[string]string{},
		allowedCustomLabels: []string{
			exportermetrics.MetricLabel_CLUSTER_NAME.String(),
		},
		fl: gpuHandler.fl,
	}
	return ifoeClient, nil
}

// nolint
func (ga *GPUAgentIFOEClient) Close() {
	// No op for now
}

// InitClients initializes the IFOE client
func (ga *GPUAgentIFOEClient) InitClients() error {
	conn := ga.gpuHandler.GetGRPCConnection()
	if conn == nil {
		return fmt.Errorf("gRPC connection is nil")
	}
	ga.ualClient = amdgpu.NewUALSvcClient(conn)
	return nil
}

// GetGPUHealthStates - no op
func (ga *GPUAgentIFOEClient) GetHealthStates() (map[string]interface{}, error) {
	return nil, nil
}

// SetError - no op
func (ga *GPUAgentIFOEClient) SetError(id string, fields []string, counts []uint32) error {
	return nil
}

// processHealthValidation - no op
func (ga *GPUAgentIFOEClient) processHealthValidation() error {
	return nil
}

// sendNodeLabelUpdate - no op
func (ga *GPUAgentIFOEClient) sendNodeLabelUpdate() error {
	return nil
}

// IsActive checks if the IFOE client is active
func (ga *GPUAgentIFOEClient) isActive() bool {
	return ga.ualClient != nil
}

func (ga *GPUAgentIFOEClient) GetContext() context.Context {
	ctx := ga.gpuHandler.GetContext()
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (ga *GPUAgentIFOEClient) GetDeviceType() globals.DeviceType {
	return globals.IFOEDevice
}

func (ga *GPUAgentIFOEClient) GetExporterNonIFOELabels() []string {
	labelList := []string{
		strings.ToLower(exportermetrics.MetricLabel_HOSTNAME.String()),
	}
	// Add custom labels
	for label := range ga.customLabelMap {
		labelList = append(labelList, strings.ToLower(label))
	}
	return labelList
}

func (ga *GPUAgentIFOEClient) GetExportLabels() []string {
	labelList := []string{}
	for key, enabled := range ga.exportLabels {
		if !enabled {
			continue
		}
		labelList = append(labelList, strings.ToLower(key))
	}

	for key := range ga.extraPodLabelsMap {
		exists := false
		for _, label := range labelList {
			if key == label {
				exists = true
				break
			}
		}
		if !exists {
			labelList = append(labelList, key)
		}
	}

	for key := range ga.customLabelMap {
		exists := false
		for _, label := range labelList {
			if key == label {
				exists = true
				break
			}
		}

		// Add only unique labels to export labels
		if !exists {
			labelList = append(labelList, key)
		}
	}

	if ga.exportLabels[exportermetrics.MetricLabel_POD_UUID.String()] {
		ga.podInfoEnabled = true
	}
	logger.Log.Printf("IFOE Export labels: %v", labelList)
	return labelList
}

func (ga *GPUAgentIFOEClient) listNetworkPort(ctx context.Context) (*amdgpu.UALNetworkPortGetResponse, error) {
	req := &amdgpu.UALNetworkPortGetRequest{}
	resp, err := ga.ualClient.UALNetworkPortGet(ctx, req)
	if err != nil {
		logger.Log.Printf("UALNetworkPortGet gRPC call failed: %v", err)
		return nil, err
	}
	if resp != nil && resp.ApiStatus != 0 {
		logger.Log.Printf("resp status :%v", resp.ApiStatus)
		return nil, fmt.Errorf("%v", resp.ApiStatus)
	}
	return resp, nil
}

func (ga *GPUAgentIFOEClient) listStation(ctx context.Context) (*amdgpu.UALStationGetResponse, error) {
	req := &amdgpu.UALStationGetRequest{}
	resp, err := ga.ualClient.UALStationGet(ctx, req)
	if err != nil {
		logger.Log.Printf("UALStationGet gRPC call failed: %v", err)
		return nil, err
	}
	if resp != nil && resp.ApiStatus != 0 {
		logger.Log.Printf("resp status :%v", resp.ApiStatus)
		return nil, fmt.Errorf("%v", resp.ApiStatus)
	}
	return resp, nil
}

func (ga *GPUAgentIFOEClient) listDevice(ctx context.Context) (*amdgpu.UALDeviceGetResponse, error) {
	req := &amdgpu.UALDeviceGetRequest{}
	resp, err := ga.ualClient.UALDeviceGet(ctx, req)
	if err != nil {
		logger.Log.Printf("UALDeviceGet gRPC call failed: %v", err)
		return nil, err
	}
	if resp != nil && resp.ApiStatus != 0 {
		logger.Log.Printf("resp status :%v", resp.ApiStatus)
		return nil, fmt.Errorf("%v", resp.ApiStatus)
	}
	return resp, nil
}

func (ga *GPUAgentIFOEClient) updateMetrics(ctx context.Context) error {
	// Capability gate: once we've determined the host has no IFOE-capable
	// devices, short-circuit subsequent polls for the rest of the process
	// lifetime. The "IFOE disabled" log already fired exactly once below
	// when we first observed zero devices.
	ga.Lock()
	disabled := ga.disabled
	ga.Unlock()
	if disabled {
		return nil
	}

	if !ga.isActive() {
		// nolint
		_ = ga.InitClients()
	}
	labels := ga.populateLabelsFromObject(nil, nil, nil, false)

	resp, err := ga.listNetworkPort(ctx)
	if err != nil {
		return err
	}
	if resp != nil && resp.ApiStatus != 0 {
		ga.metrics.totalNetworkPorts.With(labels).Set(float64(0))
		logger.Log.Printf("UALNetworkPortGet api status :%v", resp.ApiStatus)
		return fmt.Errorf("UALNetworkPortGet api status: %v", resp.ApiStatus)
	}

	dresp, err := ga.listDevice(ctx)
	if err != nil {
		return err
	}
	if dresp != nil && dresp.ApiStatus != 0 {
		ga.metrics.totalDevices.With(labels).Set(float64(0))
		logger.Log.Printf("UALDeviceGet api status :%v", dresp.ApiStatus)
		return fmt.Errorf("UALDeviceGet api status: %v", dresp.ApiStatus)
	}

	// Capability check: if gpuagent reports zero IFOE devices, this host
	// has no IFOE-capable hardware. Disable IFOE for the process lifetime
	// — log once with reason, publish no metric series, and short-circuit
	// future polls. Distinct from the gRPC-error path above (where we'd
	// want to retry on transient failures); this is a positive "no devices"
	// answer and won't change without a process restart.
	if dresp == nil || len(dresp.Response) == 0 {
		ga.disabledOnce.Do(func() {
			logger.Log.Printf("IFOE disabled: gpuagent reported zero IFOE-capable devices (UALDeviceGet returned empty response). No IFOE metrics will be published; subsequent scrape polls will short-circuit. This is the expected behavior on hosts without Pensando IFOE hardware.")
		})
		ga.Lock()
		ga.disabled = true
		ga.Unlock()
		return nil
	}

	sresp, err := ga.listStation(ctx)
	if err != nil {
		return err
	}
	if sresp != nil && sresp.ApiStatus != 0 {
		ga.metrics.totalStations.With(labels).Set(float64(0))
		logger.Log.Printf("UALStationGet api status :%v", sresp.ApiStatus)
		return fmt.Errorf("UALStationGet api status: %v", sresp.ApiStatus)
	}
	ualStationMap := make(map[string]*amdgpu.UALStation)
	for _, ualStation := range sresp.Response {
		uuid := utils.UUIDToString(ualStation.Spec.Id)
		ualStationMap[uuid] = ualStation
	}

	devMap := make(map[string]*amdgpu.UALDevice)
	for _, ualDevice := range dresp.Response {
		if ualDevice.Spec == nil {
			continue
		}
		devUuid := utils.UUIDToString(ualDevice.Spec.Id)
		devMap[devUuid] = ualDevice
	}

	ga.metrics.totalNetworkPorts.With(labels).Set(float64(len(resp.Response)))
	ga.metrics.totalDevices.With(labels).Set(float64(len(dresp.Response)))
	ga.metrics.totalStations.With(labels).Set(float64(len(sresp.Response)))

	for _, ualPort := range resp.Response {
		portUuid := utils.UUIDToString(ualPort.Spec.Id)
		portName := ""
		if ualPort.Status != nil {
			portName = ualPort.Status.Name
		}
		stationUuid := utils.UUIDToString(ualPort.Spec.UALStation)
		station, ok := ualStationMap[stationUuid]
		if !ok {
			continue
		}
		devUuid := utils.UUIDToString(station.Spec.UALDevice)
		ifoeLabels := ga.populateLabelsFromObject(nil, nil, devMap[devUuid], true)
		// TBD : remove after testing
		logger.Log.Printf("Processing UALPort: %v, Station: %v Device: %v PortName: %s", portUuid, stationUuid, devUuid, portName)

		ifoeLabels["station_uuid"] = stationUuid
		ifoeLabels["port_name"] = portName
		ifoeLabels["device_uuid"] = devUuid

		if ualPort.Status != nil {
			status := ualPort.Status
			ga.metrics.portLinkState.With(ifoeLabels).Set(float64(int32(status.LinkState)))
			ga.metrics.portSpeed.With(ifoeLabels).Set(float64(int32(status.Speed)))
			ga.metrics.portLinkUpCount.With(ifoeLabels).Set(float64(status.LinkUpCount))
			ga.metrics.portLinkUpDurationMsec.With(ifoeLabels).Set(float64(status.LinkUpDuration))
			ga.metrics.portLinkDownDurationMsec.With(ifoeLabels).Set(float64(status.LinkDownDuration))
			ga.metrics.portLinkTrainingDurationLatestMsec.With(ifoeLabels).Set(float64(status.LinkTrainingDurationLatest))
			ga.metrics.portLinkTrainingDurationAvgMsec.With(ifoeLabels).Set(float64(status.LinkTrainingDurationAvg))
		}

		stats := ualPort.Stats
		if stats != nil {
			ga.metrics.numFailedoverStreams.With(ifoeLabels).Set(float64(stats.NumFailedoverStreams))
			ga.metrics.numPausedStreams.With(ifoeLabels).Set(float64(stats.NumPausedStreams))
			ga.metrics.bitErrorRate.With(ifoeLabels).Set(float64(stats.BitErrorRate))
			ga.metrics.fecCodeWordSymbolErrors0.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors0))
			ga.metrics.fecCodeWordSymbolErrors1.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors1))
			ga.metrics.fecCodeWordSymbolErrors2.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors2))
			ga.metrics.fecCodeWordSymbolErrors3.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors3))
			ga.metrics.fecCodeWordSymbolErrors4.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors4))
			ga.metrics.fecCodeWordSymbolErrors5.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors5))
			ga.metrics.fecCodeWordSymbolErrors6.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors6))
			ga.metrics.fecCodeWordSymbolErrors7.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors7))
			ga.metrics.fecCodeWordSymbolErrors8.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors8))
			ga.metrics.fecCodeWordSymbolErrors9.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors9))
			ga.metrics.fecCodeWordSymbolErrors10.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors10))
			ga.metrics.fecCodeWordSymbolErrors11.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors11))
			ga.metrics.fecCodeWordSymbolErrors12.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors12))
			ga.metrics.fecCodeWordSymbolErrors13.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors13))
			ga.metrics.fecCodeWordSymbolErrors14.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors14))
			ga.metrics.fecCodeWordSymbolErrors15.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrors15))
			ga.metrics.fecCodeWordSymbolErrorsUncorrectable.With(ifoeLabels).Set(float64(stats.FECCodeWordSymbolErrorsUncorrectable))
			ga.metrics.txTotalBytes.With(ifoeLabels).Set(float64(stats.TxTotalBytes))
			ga.metrics.txTotalGoodBytes.With(ifoeLabels).Set(float64(stats.TxTotalGoodBytes))
			ga.metrics.txTotalErrBytes.With(ifoeLabels).Set(float64(stats.TxTotalErrBytes))
			ga.metrics.txTotalPackets.With(ifoeLabels).Set(float64(stats.TxTotalPackets))
			ga.metrics.txTotalGoodPackets.With(ifoeLabels).Set(float64(stats.TxTotalGoodPackets))
			ga.metrics.txFrameError.With(ifoeLabels).Set(float64(stats.TxFrameError))
			ga.metrics.txBadFCS.With(ifoeLabels).Set(float64(stats.TxBadFCS))
			ga.metrics.rxTotalBytes.With(ifoeLabels).Set(float64(stats.RxTotalBytes))
			ga.metrics.rxTotalGoodBytes.With(ifoeLabels).Set(float64(stats.RxTotalGoodBytes))
			ga.metrics.rxTotalErrBytes.With(ifoeLabels).Set(float64(stats.RxTotalErrBytes))
			ga.metrics.rxTotalPackets.With(ifoeLabels).Set(float64(stats.RxTotalPackets))
			ga.metrics.rxTotalGoodPackets.With(ifoeLabels).Set(float64(stats.RxTotalGoodPackets))
			ga.metrics.rxPacketDropped.With(ifoeLabels).Set(float64(stats.RxPacketDropped))
			ga.metrics.rxBadFCS.With(ifoeLabels).Set(float64(stats.RxBadFCS))
			ga.metrics.rxFECCorrectedCodewords.With(ifoeLabels).Set(float64(stats.RxFECCorrectedCodewords))
			ga.metrics.rxFECUncorrectedCodewords.With(ifoeLabels).Set(float64(stats.RxFECUncorrectedCodewords))
			ga.metrics.txPause.With(ifoeLabels).Set(float64(stats.TxPause))
			ga.metrics.rxPause.With(ifoeLabels).Set(float64(stats.RxPause))
			ga.metrics.txUserPause.With(ifoeLabels).Set(float64(stats.TxUserPause))
			ga.metrics.rxUserPause.With(ifoeLabels).Set(float64(stats.RxUserPause))
			ga.metrics.rxJabber.With(ifoeLabels).Set(float64(stats.RxJabber))
			ga.metrics.rxOversize.With(ifoeLabels).Set(float64(stats.RxOversize))
			ga.metrics.rxTooLong.With(ifoeLabels).Set(float64(stats.RxTooLong))
			ga.metrics.rxTruncated.With(ifoeLabels).Set(float64(stats.RxTruncated))
			ga.metrics.txLLROkPackets.With(ifoeLabels).Set(float64(stats.TxLLROkPackets))
			ga.metrics.rxLLROkPackets.With(ifoeLabels).Set(float64(stats.RxLLROkPackets))
			ga.metrics.txLLRReplayCount.With(ifoeLabels).Set(float64(stats.TxLLRReplayCt))
			ga.metrics.txLLRReplaysCompleted.With(ifoeLabels).Set(float64(stats.TxLLRReplaysCompleted))
			ga.metrics.rxLLRBadPackets.With(ifoeLabels).Set(float64(stats.RxLLRBadPackets))
			ga.metrics.rxLLRDuplSeqPackets.With(ifoeLabels).Set(float64(stats.RxLLRDuplSeqPackets))
			ga.metrics.rxFECBitErr0To1Lane0.With(ifoeLabels).Set(float64(stats.RxFECBitErr0To1Lane0))
			ga.metrics.rxFECBitErr0To1Lane1.With(ifoeLabels).Set(float64(stats.RxFECBitErr0To1Lane1))
			ga.metrics.rxFECBitErr0To1Lane2.With(ifoeLabels).Set(float64(stats.RxFECBitErr0To1Lane2))
			ga.metrics.rxFECBitErr0To1Lane3.With(ifoeLabels).Set(float64(stats.RxFECBitErr0To1Lane3))
			ga.metrics.rxFECBitErr1To0Lane0.With(ifoeLabels).Set(float64(stats.RxFECBitErr1To0Lane0))
			ga.metrics.rxFECBitErr1To0Lane1.With(ifoeLabels).Set(float64(stats.RxFECBitErr1To0Lane1))
			ga.metrics.rxFECBitErr1To0Lane2.With(ifoeLabels).Set(float64(stats.RxFECBitErr1To0Lane2))
			ga.metrics.rxFECBitErr1To0Lane3.With(ifoeLabels).Set(float64(stats.RxFECBitErr1To0Lane3))
			ga.metrics.rxFECSymbolErrCountLane0.With(ifoeLabels).Set(float64(stats.RxFECSymbolErrCountLane0))
			ga.metrics.rxFECSymbolErrCountLane1.With(ifoeLabels).Set(float64(stats.RxFECSymbolErrCountLane1))
			ga.metrics.rxFECSymbolErrCountLane2.With(ifoeLabels).Set(float64(stats.RxFECSymbolErrCountLane2))
			ga.metrics.rxFECSymbolErrCountLane3.With(ifoeLabels).Set(float64(stats.RxFECSymbolErrCountLane3))
			ga.metrics.rxBadCodeCount.With(ifoeLabels).Set(float64(stats.RxBadCodeCount))
			ga.metrics.rxStompedFCS.With(ifoeLabels).Set(float64(stats.RxStompedFCS))
		}
	}

	for _, ualStation := range sresp.Response {
		if ualStation.Spec == nil || ualStation.Stats == nil {
			continue
		}
		stationUuid := utils.UUIDToString(ualStation.Spec.Id)
		devUuid := utils.UUIDToString(ualStation.Spec.UALDevice)
		stationLabels := ga.populateLabelsFromObject(nil, nil, devMap[devUuid], true)
		stationLabels["station_uuid"] = stationUuid
		stationLabels["device_uuid"] = devUuid

		stats := ualStation.Stats
		ga.metrics.stationTxRequestPackets.With(stationLabels).Set(float64(stats.TxRequestPacketCount))
		ga.metrics.stationTxResponsePackets.With(stationLabels).Set(float64(stats.TxResponsePacketCount))
		ga.metrics.stationRxRequestPackets.With(stationLabels).Set(float64(stats.RxRequestPacketCount))
		ga.metrics.stationRxResponsePackets.With(stationLabels).Set(float64(stats.RxResponsePacketCount))
		ga.metrics.stationStreamRemapsTotal.With(stationLabels).Set(float64(stats.StreamRemapsTotal))
		ga.metrics.stationPausedStreamsCount.With(stationLabels).Set(float64(stats.PausedStreamsCount))
		ga.metrics.stationStreamRemapsNetworkPort0.With(stationLabels).Set(float64(stats.StreamRemapsNetworkPort0))
		ga.metrics.stationStreamRemapsNetworkPort1.With(stationLabels).Set(float64(stats.StreamRemapsNetworkPort1))
		ga.metrics.stationStreamRemapsNetworkPort2.With(stationLabels).Set(float64(stats.StreamRemapsNetworkPort2))
		ga.metrics.stationStreamRemapsNetworkPort3.With(stationLabels).Set(float64(stats.StreamRemapsNetworkPort3))
		ga.metrics.stationCryptoTxKeyUpdatesSA0.With(stationLabels).Set(float64(stats.CryptoTxKeyUpdatesSA0))
		ga.metrics.stationCryptoRxKey0UpdatesSA0.With(stationLabels).Set(float64(stats.CryptoRxKey0UpdatesSA0))
		ga.metrics.stationCryptoRxKey1UpdatesSA0.With(stationLabels).Set(float64(stats.CryptoRxKey1UpdatesSA0))
		ga.metrics.stationCryptoRxKeyDisablesSA0.With(stationLabels).Set(float64(stats.CryptoRxKeyDisablesSA0))
		ga.metrics.stationCryptoTxKeyUpdatesSA1.With(stationLabels).Set(float64(stats.CryptoTxKeyUpdatesSA1))
		ga.metrics.stationCryptoRxKey0UpdatesSA1.With(stationLabels).Set(float64(stats.CryptoRxKey0UpdatesSA1))
		ga.metrics.stationCryptoRxKey1UpdatesSA1.With(stationLabels).Set(float64(stats.CryptoRxKey1UpdatesSA1))
		ga.metrics.stationCryptoRxKeyDisablesSA1.With(stationLabels).Set(float64(stats.CryptoRxKeyDisablesSA1))
	}
	return nil
}
