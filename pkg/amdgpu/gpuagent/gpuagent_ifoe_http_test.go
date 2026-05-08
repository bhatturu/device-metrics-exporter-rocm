/**
# Copyright (c) Advanced Micro Devices, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package gpuagent

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/mock/gomock"
	"gotest.tools/assert"

	amdgpu "github.com/ROCm/device-metrics-exporter/pkg/amdgpu/gen/amdgpu"
	"github.com/ROCm/device-metrics-exporter/pkg/amdgpu/mock_gen"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/config"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/gen/exportermetrics"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/globals"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/logger"
	"github.com/ROCm/device-metrics-exporter/pkg/exporter/metricsutil"
)

// TestGPUAgentIFOEMetricsHTTP is a regression test for the prometheus
// "inconsistent label cardinality" panic that occurred in the IFOE station
// loop when the deployed config enabled additional labels beyond the two
// mandatory ones. With a rich label config the station/port GaugeVecs
// register with > 5 labels, so any code path that populates fewer panics on
// the first scrape.
//
// Steps:
//  1. Build fully-populated mock UAL responses (Device with version info,
//     Station with full Stats, Port with full Status+Stats).
//  2. Load an in-memory IFOE config that enables a broad label set plus a
//     custom label.
//  3. Spin up an IFOE-only agent, register it with a fresh MetricsHandler
//     and serve /metrics over httptest.
//  4. GET /metrics and assert the body contains expected dummy values and
//     the full label cardinality on station/port series.
func TestGPUAgentIFOEMetricsHTTP(t *testing.T) {
	logger.Init(true)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Stable UUIDs so we can assert on them in the /metrics body.
	deviceUUID := uuid.New().String()
	stationUUID := uuid.New().String()
	portUUID := uuid.New().String()
	gpuUUID := uuid.New().String()

	ualMock := mock_gen.NewMockUALSvcClient(ctrl)
	ualMock.EXPECT().UALDeviceGet(gomock.Any(), gomock.Any()).
		Return(buildIFOEDeviceResp(deviceUUID, gpuUUID), nil).AnyTimes()
	ualMock.EXPECT().UALStationGet(gomock.Any(), gomock.Any()).
		Return(buildIFOEStationResp(stationUUID, deviceUUID), nil).AnyTimes()
	ualMock.EXPECT().UALNetworkPortGet(gomock.Any(), gomock.Any()).
		Return(buildIFOEPortResp(portUUID, stationUUID), nil).AnyTimes()

	// Fresh ConfigHandler with no on-disk config; LoadConfig pushes the
	// rich label set in-memory so RefreshConfig (called from InitConfig)
	// does not clobber it.
	cfgHandler := config.NewConfigHandler("", config.GPUAgentConfig{GrpcPort: globals.GPUAgentPort})
	mh, err := metricsutil.NewMetrics(cfgHandler)
	assert.NilError(t, err)

	// InitConfig before any client is registered: this creates the
	// prometheus registry but no goroutines fan out yet (mh.clients is
	// still empty). RefreshConfig fails on the empty path; that is fine
	// because LoadConfig below puts the real config in place.
	mh.InitConfig()

	ifoeCfg := &exportermetrics.IFOEMetricConfig{
		// Enable 6 non-mandatory labels on top of HOSTNAME + GPU_UUID.
		// Combined with the custom label below, GetExportLabels()
		// returns a list of length 9; station GaugeVecs register with
		// 11 (station_uuid + device_uuid + 9), port GaugeVecs with 12.
		Labels: []string{
			"CARD_SERIES",
			"CARD_MODEL",
			"CARD_VENDOR",
			"DRIVER_VERSION",
			"VBIOS_VERSION",
			"SERIAL_NUMBER",
		},
		CustomLabels: map[string]string{
			"cluster_name": "test-cluster",
		},
	}
	assert.NilError(t, cfgHandler.LoadConfig(&exportermetrics.MetricConfig{IFOEConfig: ifoeCfg}))

	ga := NewAgent(mh,
		WithK8sClient(nil),
		WithK8sSchedulerClient(nil),
		WithSlurmClient(false),
		WithGPUMonitoring(false),
		WithIFOEMonitoring(true),
	)
	assert.Assert(t, ga != nil, "expected IFOE-only agent")
	defer ga.Close()

	assert.NilError(t, ga.Init())

	// Swap the real UAL client for the mock now that Init has wired conn.
	var ifoeClient *GPUAgentIFOEClient
	for _, c := range ga.clients {
		if c.GetDeviceType() == globals.IFOEDevice {
			ifoeClient = c.(*GPUAgentIFOEClient)
			break
		}
	}
	assert.Assert(t, ifoeClient != nil, "expected IFOE client to be registered")
	ifoeClient.ualClient = ualMock

	// Register IFOE prom collectors directly so we can synchronously
	// catch a cardinality panic. Doing this through mh.InitConfig would
	// fan out into goroutines and a panic there would tear down the
	// test binary.
	assert.NilError(t, ifoeClient.InitConfigs())

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("UpdateStaticMetrics panicked (cardinality regression?): %v", r)
			}
		}()
		assert.NilError(t, ifoeClient.UpdateStaticMetrics())
	}()

	// Integration HTTP path. Mirrors the production /metrics route
	// (exporter.go:115-159): middleware re-runs UpdateMetrics before
	// promhttp serves so each scrape reflects the latest gRPC payload.
	reg := mh.GetRegistry()
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		_ = mh.UpdateMetrics(r.Context())
		promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}).ServeHTTP(w, r)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/metrics")
	assert.NilError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK,
		"GET /metrics should succeed; cardinality mismatch surfaces as panic")

	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NilError(t, err)
	body := string(bodyBytes)

	// Sanity: totals show 1 of each (we returned one device/station/port).
	requireSubstring(t, body, "ifoe_total_devices{")
	requireSubstring(t, body, "ifoe_total_stations{")
	requireSubstring(t, body, "ifoe_total_ports{")

	// Station-loop series carry the dummy stats — the original panic
	// site at gpuagent_ifoe.go:375.
	requireSubstring(t, body, "ifoe_station_tx_request_packets{")
	requireSubstring(t, body, "ifoe_station_rx_request_packets{")
	requireSubstring(t, body, "ifoe_station_stream_remaps_total{")
	requireSubstring(t, body, "ifoe_station_paused_streams_count{")
	requireSubstring(t, body, "ifoe_station_stream_remaps_network_port0{")

	// The dummy station Tx/Rx values must reach the wire.
	requireSubstring(t, body, "} 1001")
	requireSubstring(t, body, "} 1002")

	// Port-loop series + port_name label.
	requireSubstring(t, body, "ifoe_port_link_state{")
	requireSubstring(t, body, "ifoe_num_failedover_streams{")
	requireSubstring(t, body, `port_name="ual-port-1"`)

	// Full label set on station/port series. Each of these would have
	// been missing under the pre-fix code.
	requireSubstring(t, body, fmt.Sprintf(`station_uuid="%s"`, stationUUID))
	requireSubstring(t, body, fmt.Sprintf(`device_uuid="%s"`, deviceUUID))
	requireSubstring(t, body, fmt.Sprintf(`gpu_uuid="%s"`, gpuUUID))
	requireSubstring(t, body, `cluster_name="test-cluster"`)
	requireSubstring(t, body, `card_series="card_series_placeholder"`)
	requireSubstring(t, body, `card_model="card_model_placeholder"`)
	requireSubstring(t, body, `card_vendor="AMD"`)
	requireSubstring(t, body, `serial_number="serial_number_placeholder"`)

	// Version labels must come from the UALDevice, not placeholders —
	// guards the recent "pull real data from UALDevice" change.
	requireSubstring(t, body, `driver_version="1.2.3"`)
	requireSubstring(t, body, `vbios_version="4.5.6"`)
}

func requireSubstring(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("/metrics body missing expected substring %q", needle)
	}
}

func buildIFOEDeviceResp(deviceUUID, gpuUUID string) *amdgpu.UALDeviceGetResponse {
	return &amdgpu.UALDeviceGetResponse{
		ApiStatus: amdgpu.ApiStatus_API_STATUS_OK,
		Response: []*amdgpu.UALDevice{
			{
				Spec: &amdgpu.UALDeviceSpec{
					Id: []byte(deviceUUID),
				},
				Status: &amdgpu.UALDeviceStatus{
					GPU: []byte(gpuUUID),
					Version: &amdgpu.UALDeviceVersionInfo{
						UALLibVersion:    &amdgpu.SemanticVersion{Major: 1, Minor: 2, Patch: 3},
						FirmwareVersion:  &amdgpu.SemanticVersion{Major: 4, Minor: 5, Patch: 6},
						TelemetryVersion: &amdgpu.SemanticVersion{Major: 7, Minor: 8, Patch: 9},
					},
				},
				Stats: &amdgpu.UALDeviceStats{},
			},
		},
	}
}

func buildIFOEStationResp(stationUUID, deviceUUID string) *amdgpu.UALStationGetResponse {
	return &amdgpu.UALStationGetResponse{
		ApiStatus: amdgpu.ApiStatus_API_STATUS_OK,
		Response: []*amdgpu.UALStation{
			{
				Spec: &amdgpu.UALStationSpec{
					Id:        []byte(stationUUID),
					UALDevice: []byte(deviceUUID),
				},
				Status: &amdgpu.UALStationStatus{
					Name: "ual-station-1",
				},
				Stats: &amdgpu.UALStationStats{
					TxRequestPacketCount:     1001,
					TxResponsePacketCount:    1003,
					RxRequestPacketCount:     1002,
					RxResponsePacketCount:    1004,
					StreamRemapsTotal:        10,
					PausedStreamsCount:       2,
					StreamRemapsNetworkPort0: 3,
					StreamRemapsNetworkPort1: 4,
					StreamRemapsNetworkPort2: 5,
					StreamRemapsNetworkPort3: 6,
				},
			},
		},
	}
}

func buildIFOEPortResp(portUUID, stationUUID string) *amdgpu.UALNetworkPortGetResponse {
	return &amdgpu.UALNetworkPortGetResponse{
		ApiStatus: amdgpu.ApiStatus_API_STATUS_OK,
		Response: []*amdgpu.UALNetworkPort{
			{
				Spec: &amdgpu.UALNetworkPortSpec{
					Id:         []byte(portUUID),
					UALStation: []byte(stationUUID),
				},
				Status: &amdgpu.UALNetworkPortStatus{
					Name:                       "ual-port-1",
					LogicalIndex:               32,
					LocalPortIndex:             1,
					LinkState:                  amdgpu.UALLinkState_UAL_LINK_STATE_UP,
					Speed:                      amdgpu.UALPortSpeed_UAL_PORT_SPEED_400G,
					LinkUpCount:                7,
					LinkUpDuration:             123456,
					LinkDownDuration:           789,
					LinkTrainingDurationLatest: 11,
					LinkTrainingDurationAvg:    12,
				},
				Stats: &amdgpu.UALNetworkPortStats{
					NumFailedoverStreams:                 5,
					NumPausedStreams:                     2,
					BitErrorRate:                         42,
					FECCodeWordSymbolErrors0:             100,
					FECCodeWordSymbolErrors1:             101,
					FECCodeWordSymbolErrors2:             102,
					FECCodeWordSymbolErrors3:             103,
					FECCodeWordSymbolErrors4:             104,
					FECCodeWordSymbolErrors5:             105,
					FECCodeWordSymbolErrors6:             106,
					FECCodeWordSymbolErrors7:             107,
					FECCodeWordSymbolErrors8:             108,
					FECCodeWordSymbolErrors9:             109,
					FECCodeWordSymbolErrors10:            110,
					FECCodeWordSymbolErrors11:            111,
					FECCodeWordSymbolErrors12:            112,
					FECCodeWordSymbolErrors13:            113,
					FECCodeWordSymbolErrors14:            114,
					FECCodeWordSymbolErrors15:            115,
					FECCodeWordSymbolErrorsUncorrectable: 116,
					TxTotalBytes:                         1 << 20,
					TxTotalGoodBytes:                     1 << 19,
					TxTotalErrBytes:                      32,
					TxTotalPackets:                       2048,
					TxTotalGoodPackets:                   2000,
					TxFrameError:                         3,
					TxBadFCS:                             4,
					RxTotalBytes:                         1 << 21,
					RxTotalGoodBytes:                     1 << 20,
					RxTotalErrBytes:                      64,
					RxTotalPackets:                       4096,
					RxTotalGoodPackets:                   4000,
					RxPacketDropped:                      8,
					RxBadFCS:                             9,
					RxFECCorrectedCodewords:              17,
					RxFECUncorrectedCodewords:            18,
					TxPause:                              21,
					RxPause:                              22,
					TxUserPause:                          23,
					RxUserPause:                          24,
					RxJabber:                             25,
					RxOversize:                           26,
					RxTooLong:                            27,
					RxTruncated:                          28,
					TxLLROkPackets:                       29,
					RxLLROkPackets:                       30,
					TxLLRReplayCt:                        31,
					TxLLRReplaysCompleted:                32,
					RxLLRBadPackets:                      33,
					RxLLRDuplSeqPackets:                  34,
				},
			},
		},
	}
}
