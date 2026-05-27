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
	"context"
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
	mh.InitConfig(context.Background())

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
		assert.NilError(t, ifoeClient.UpdateStaticMetrics(context.Background()))
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
	requireSubstring(t, body, "ifoe_station_crypto_tx_key_updates_sa0{")
	requireSubstring(t, body, "ifoe_station_crypto_rx_key0_updates_sa0{")
	requireSubstring(t, body, "ifoe_station_crypto_rx_key1_updates_sa0{")
	requireSubstring(t, body, "ifoe_station_crypto_rx_key_disables_sa0{")
	requireSubstring(t, body, "ifoe_station_crypto_tx_key_updates_sa1{")
	requireSubstring(t, body, "ifoe_station_crypto_rx_key0_updates_sa1{")
	requireSubstring(t, body, "ifoe_station_crypto_rx_key1_updates_sa1{")
	requireSubstring(t, body, "ifoe_station_crypto_rx_key_disables_sa1{")

	// The dummy station Tx/Rx values must reach the wire.
	requireSubstring(t, body, "} 1001")
	requireSubstring(t, body, "} 1002")
	// Crypto key update values (SA0: 77-80, SA1: 81-84).
	requireSubstring(t, body, "} 77")
	requireSubstring(t, body, "} 80")
	requireSubstring(t, body, "} 81")
	requireSubstring(t, body, "} 84")

	// Port-loop series + port_name label.
	requireSubstring(t, body, "ifoe_port_link_state{")
	requireSubstring(t, body, "ifoe_num_failedover_streams{")
	requireSubstring(t, body, `port_name="ual-port-1"`)

	// All 14 per-lane FEC and error counter metrics.
	requireSubstring(t, body, "ifoe_rx_fec_bit_err_0to1_lane0{")
	requireSubstring(t, body, "ifoe_rx_fec_bit_err_0to1_lane1{")
	requireSubstring(t, body, "ifoe_rx_fec_bit_err_0to1_lane2{")
	requireSubstring(t, body, "ifoe_rx_fec_bit_err_0to1_lane3{")
	requireSubstring(t, body, "ifoe_rx_fec_bit_err_1to0_lane0{")
	requireSubstring(t, body, "ifoe_rx_fec_bit_err_1to0_lane1{")
	requireSubstring(t, body, "ifoe_rx_fec_bit_err_1to0_lane2{")
	requireSubstring(t, body, "ifoe_rx_fec_bit_err_1to0_lane3{")
	requireSubstring(t, body, "ifoe_rx_fec_symbol_err_count_lane0{")
	requireSubstring(t, body, "ifoe_rx_fec_symbol_err_count_lane1{")
	requireSubstring(t, body, "ifoe_rx_fec_symbol_err_count_lane2{")
	requireSubstring(t, body, "ifoe_rx_fec_symbol_err_count_lane3{")
	requireSubstring(t, body, "ifoe_rx_bad_code_count{")
	requireSubstring(t, body, "ifoe_rx_stomped_fcs{")
	// Spot-check representative values (500, 507, 511, 512, 513).
	requireSubstring(t, body, "} 500")
	requireSubstring(t, body, "} 507")
	requireSubstring(t, body, "} 511")
	requireSubstring(t, body, "} 512")
	requireSubstring(t, body, "} 513")

	// New port-level counters (queue stats, PFC)
	requireSubstring(t, body, "ifoe_discard_q_rx_dropped_packets{")
	requireSubstring(t, body, "ifoe_rx_pause_packets_rcvd{")
	requireSubstring(t, body, "ifoe_tx_pause_packets_sent{")
	requireSubstring(t, body, "} 600")
	requireSubstring(t, body, "} 601")
	requireSubstring(t, body, "} 602")

	// New station-level counters (SDP, WENG, RENG, TX encap, TX sched, RX decap, switch)
	requireSubstring(t, body, "ifoe_station_sdp_tx_pack_rd_req{")
	requireSubstring(t, body, "ifoe_station_sdp_rx_unpack_req_credits_consumed{")
	requireSubstring(t, body, "ifoe_station_weng_eviction_force_req{")
	requireSubstring(t, body, "ifoe_station_reng_free_blk_out{")
	requireSubstring(t, body, "ifoe_station_tx_encap_pkt_egress_xrsec_nport_0{")
	requireSubstring(t, body, "ifoe_station_tx_sched_active_streams{")
	requireSubstring(t, body, "ifoe_station_rx_decap_dropped_pkts{")
	requireSubstring(t, body, "ifoe_station_rx_dropped_pkts{")
	requireSubstring(t, body, "ifoe_station_tx_nonifoe_pkts{")
	requireSubstring(t, body, "} 700")
	requireSubstring(t, body, "} 708")

	// Unknown counters must not appear in /metrics output
	if strings.Contains(body, "future_unknown_counter") {
		t.Error("unknown counter 'future_unknown_counter' should not appear in /metrics output")
	}

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
					Stats: []*amdgpu.UALTelemetryCounter{
						{Name: "Tx request pkt count", Value: 1001},
						{Name: "Tx response pkt count", Value: 1003},
						{Name: "Rx request pkt count", Value: 1002},
						{Name: "Rx response pkt count", Value: 1004},
						{Name: "Stream remaps total", Value: 10},
						{Name: "Paused streams count", Value: 2},
						{Name: "Stream remaps nport 0", Value: 3},
						{Name: "Stream remaps nport 1", Value: 4},
						{Name: "Stream remaps nport 2", Value: 5},
						{Name: "Stream remaps nport 3", Value: 6},
						{Name: "Crypto Tx key updates sa0", Value: 77},
						{Name: "Crypto Rx key0 updates sa0", Value: 78},
						{Name: "Crypto Rx key1 updates sa0", Value: 79},
						{Name: "Crypto Rx key disables sa0", Value: 80},
						{Name: "Crypto Tx key updates sa1", Value: 81},
						{Name: "Crypto Rx key0 updates sa1", Value: 82},
						{Name: "Crypto Rx key1 updates sa1", Value: 83},
						{Name: "Crypto Rx key disables sa1", Value: 84},
						{Name: "SDP Tx pack rd req", Value: 700},
						{Name: "SDP Rx unpack req credits consumed", Value: 701},
						{Name: "Weng eviction force req", Value: 702},
						{Name: "Reng free blk out", Value: 703},
						{Name: "Tx encap pkt egress xrsec nport 0", Value: 704},
						{Name: "Tx sched active streams", Value: 705},
						{Name: "Rx decap dropped pkts", Value: 706},
						{Name: "Rx dropped pkts", Value: 707},
						{Name: "Tx nonifoe pkts", Value: 708},
					},
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
					NumFailedoverStreams: 5,
					NumPausedStreams:     2,
					Stats: []*amdgpu.UALTelemetryCounter{
						{Name: "Bit error rate", Value: 42},
						{Name: "FEC CW symbol errs 0", Value: 100},
						{Name: "FEC CW symbol errs 1", Value: 101},
						{Name: "FEC CW symbol errs 2", Value: 102},
						{Name: "FEC CW symbol errs 3", Value: 103},
						{Name: "FEC CW symbol errs 4", Value: 104},
						{Name: "FEC CW symbol errs 5", Value: 105},
						{Name: "FEC CW symbol errs 6", Value: 106},
						{Name: "FEC CW symbol errs 7", Value: 107},
						{Name: "FEC CW symbol errs 8", Value: 108},
						{Name: "FEC CW symbol errs 9", Value: 109},
						{Name: "FEC CW symbol errs 10", Value: 110},
						{Name: "FEC CW symbol errs 11", Value: 111},
						{Name: "FEC CW symbol errs 12", Value: 112},
						{Name: "FEC CW symbol errs 13", Value: 113},
						{Name: "FEC CW symbol errs 14", Value: 114},
						{Name: "FEC CW symbol errs 15", Value: 115},
						{Name: "FEC CW symbol errs uncorrectable", Value: 116},
						{Name: "Tx total bytes", Value: 1 << 20},
						{Name: "Tx total good bytes", Value: 1 << 19},
						{Name: "Tx total ERR bytes", Value: 32},
						{Name: "Tx total packets", Value: 2048},
						{Name: "Tx total good packets", Value: 2000},
						{Name: "Tx frame error", Value: 3},
						{Name: "Tx BAD FCS", Value: 4},
						{Name: "Rx total bytes", Value: 1 << 21},
						{Name: "Rx total good bytes", Value: 1 << 20},
						{Name: "Rx total ERR bytes", Value: 64},
						{Name: "Rx total packets", Value: 4096},
						{Name: "Rx total good packets", Value: 4000},
						{Name: "Rx packet dropped", Value: 8},
						{Name: "Rx BAD FCS", Value: 9},
						{Name: "Rx FEC corrected CW", Value: 17},
						{Name: "Rx FEC uncorrected CW", Value: 18},
						{Name: "Tx pause", Value: 21},
						{Name: "Rx pause", Value: 22},
						{Name: "Tx user pause", Value: 23},
						{Name: "Rx user pause", Value: 24},
						{Name: "Rx jabber", Value: 25},
						{Name: "Rx oversize", Value: 26},
						{Name: "Rx toolong", Value: 27},
						{Name: "Rx truncated", Value: 28},
						{Name: "Tx LLR OK packets", Value: 29},
						{Name: "Rx LLR OK packets", Value: 30},
						{Name: "Tx LLR replay ct", Value: 31},
						{Name: "Tx LLR replays completed", Value: 32},
						{Name: "Rx LLR BAD packets", Value: 33},
						{Name: "Rx LLR dupl SEQ packets", Value: 34},
						{Name: "Rx FEC bit ERR 0to1 L0", Value: 500},
						{Name: "Rx FEC bit ERR 0to1 L1", Value: 501},
						{Name: "Rx FEC bit ERR 0to1 L2", Value: 502},
						{Name: "Rx FEC bit ERR 0to1 L3", Value: 503},
						{Name: "Rx FEC bit ERR 1to0 L0 LSB", Value: 504},
						{Name: "Rx FEC bit ERR 1to0 L1 LSB", Value: 505},
						{Name: "Rx FEC bit ERR 1to0 L2 LSB", Value: 506},
						{Name: "Rx FEC bit ERR 1to0 L3 LSB", Value: 507},
						{Name: "Rx FEC ERR count L0 LSB", Value: 508},
						{Name: "Rx FEC ERR count L1 LSB", Value: 509},
						{Name: "Rx FEC ERR count L2 LSB", Value: 510},
						{Name: "Rx FEC ERR count L3 LSB", Value: 511},
						{Name: "Rx BAD code count", Value: 512},
						{Name: "Rx stomped FCS", Value: 513},
						{Name: "Discard Q Rx dropped packets", Value: 600},
						{Name: "Rx pause packets rcvd", Value: 601},
						{Name: "Tx pause packets sent", Value: 602},
						{Name: "Future unknown counter", Value: 999},
					},
				},
			},
		},
	}
}
