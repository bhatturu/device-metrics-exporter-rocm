# List of Available IFOE Metrics

This section provides an overview of the IFOE statistics available using the Device Metrics Exporter.

## IFOE Metric Labels

Every IFOE metric series carries a set of identification labels. The following table lists each label name (as it appears in Prometheus output), which metric groups it applies to, and a description.

| Label | Applies to | Description |
| --- | --- | --- |
| `hostname` | All IFOE metrics | Hostname of the node running the exporter |
| `gpu_uuid` | All IFOE metrics | UUID of the GPU associated with the UAL device |
| `device_uuid` | Port and station metrics | UUID of the UAL device |
| `station_uuid` | Port and station metrics | UUID of the UAL station |
| `port_name` | Port metrics only | Human-readable name of the network port |
| `port_index` | Port metrics only | Logical index of the network port within its station |
| `station_index` | Station metrics only | Logical index of the UAL station within its device |
| `accelerator_id` | Port and station metrics | Local accelerator ID assigned to the UAL device |
| `vpod_id` | Port and station metrics | vPod identifier — logical isolation domain this accelerator belongs to |
| `physical_pod_id` | Port and station metrics | Physical pod identifier — physical fabric placement of this accelerator |

> **Note:** All labels in the table above are always present in the Prometheus output and cannot be removed. `hostname` and `gpu_uuid` correspond to the `HOSTNAME` and `GPU_UUID` keys in the `IFOEConfig.Labels` config array. The remaining labels (`device_uuid`, `station_uuid`, `port_name`, `port_index`, `station_index`, `accelerator_id`, `vpod_id`, `physical_pod_id`) are fixed structural labels added by the exporter regardless of config. Additional optional labels (for example `CARD_MODEL`, `DRIVER_VERSION`, `CLUSTER_NAME`) can be enabled via the `Labels` and `CustomLabels` fields in `IFOEConfig`.

## Supported IFOE Metrics List

The following table contains a full list of IFOE Metrics that are available using the Device Metrics Exporter.

| Metric                                                  | Description                                                                      |
|---------------------------------------------------------|----------------------------------------------------------------------------------|
| IFOE_TOTAL_DEVICES                                      | Total number of IFOE devices on the host                                         |
| IFOE_TOTAL_STATIONS                                     | Total number of IFOE stations across all devices                                 |
| IFOE_TOTAL_PORTS                                        | Total number of IFOE network ports across all stations                           |
| IFOE_NUMBER_FAILEDOVER_STREAMS                          | Number of IFOE streams that have experienced failover to redundant paths         |
| IFOE_NUMBER_PAUSED_STREAMS                              | Number of IFOE streams that are currently in a paused state                      |
| IFOE_BIT_ERROR_RATE                                     | Bit Error Rate (BER) reported by the network port                                |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS0                        | Total number of FEC codewords with 0 symbol errors (error-free codewords)        |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS1                        | Total number of FEC codewords with 1 symbol error                                |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS2                        | Total number of FEC codewords with 2 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS3                        | Total number of FEC codewords with 3 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS4                        | Total number of FEC codewords with 4 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS5                        | Total number of FEC codewords with 5 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS6                        | Total number of FEC codewords with 6 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS7                        | Total number of FEC codewords with 7 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS8                        | Total number of FEC codewords with 8 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS9                        | Total number of FEC codewords with 9 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS10                       | Total number of FEC codewords with 10 symbol errors                              |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS11                       | Total number of FEC codewords with 11 symbol errors                              |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS12                       | Total number of FEC codewords with 12 symbol errors                              |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS13                       | Total number of FEC codewords with 13 symbol errors                              |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS14                       | Total number of FEC codewords with 14 symbol errors                              |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS15                       | Total number of FEC codewords with 15 symbol errors (maximum correctable errors) |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS_UNCORRECTABLE           | Total number of FEC codewords with uncorrectable symbol errors                   |
| IFOE_PORT_LINK_STATE                                    | Current link state of the network port (0=down, 1=up)                            |
| IFOE_PORT_SPEED                                         | Current speed of the network port in Gbps                                        |
| IFOE_PORT_LINK_UP_COUNT                                 | Number of times the link has transitioned to up state                            |
| IFOE_PORT_LINK_UP_DURATION_MSEC                         | Total duration the link has been up in milliseconds                              |
| IFOE_PORT_LINK_DOWN_DURATION_MSEC                       | Total duration the link has been down in milliseconds                            |
| IFOE_PORT_LINK_TRAINING_DURATION_LATEST_MSEC            | Duration of the most recent link training in milliseconds                        |
| IFOE_PORT_LINK_TRAINING_DURATION_AVG_MSEC               | Average link training duration in milliseconds                                   |
| IFOE_TX_TOTAL_BYTES                                     | Total number of bytes transmitted on the network port                            |
| IFOE_TX_TOTAL_GOOD_BYTES                                | Total number of good bytes transmitted                                           |
| IFOE_TX_TOTAL_ERR_BYTES                                 | Total number of error bytes transmitted                                          |
| IFOE_TX_TOTAL_PACKETS                                   | Total number of packets transmitted                                              |
| IFOE_TX_TOTAL_GOOD_PACKETS                              | Total number of good packets transmitted                                         |
| IFOE_TX_FRAME_ERROR                                     | Total number of transmit frame errors                                            |
| IFOE_TX_BAD_FCS                                         | Total number of packets transmitted with bad FCS                                 |
| IFOE_RX_TOTAL_BYTES                                     | Total number of bytes received on the network port                               |
| IFOE_RX_TOTAL_GOOD_BYTES                                | Total number of good bytes received                                              |
| IFOE_RX_TOTAL_ERR_BYTES                                 | Total number of error bytes received                                             |
| IFOE_RX_TOTAL_PACKETS                                   | Total number of packets received                                                 |
| IFOE_RX_TOTAL_GOOD_PACKETS                              | Total number of good packets received                                            |
| IFOE_RX_PACKET_DROPPED                                  | Total number of received packets dropped                                         |
| IFOE_RX_BAD_FCS                                         | Total number of packets received with bad FCS                                    |
| IFOE_RX_FEC_CORRECTED_CODEWORDS                         | Total number of FEC corrected codewords received                                 |
| IFOE_RX_FEC_UNCORRECTED_CODEWORDS                       | Total number of FEC uncorrected codewords received                               |
| IFOE_TX_PAUSE                                           | Total number of pause frames transmitted                                         |
| IFOE_RX_PAUSE                                           | Total number of pause frames received                                            |
| IFOE_TX_USER_PAUSE                                      | Total number of user pause frames transmitted                                    |
| IFOE_RX_USER_PAUSE                                      | Total number of user pause frames received                                       |
| IFOE_RX_JABBER                                          | Total number of jabber frames received (oversized with bad CRC)                  |
| IFOE_RX_OVERSIZE                                        | Total number of oversized frames received                                        |
| IFOE_RX_TOO_LONG                                        | Total number of too-long frames received                                         |
| IFOE_RX_TRUNCATED                                       | Total number of truncated frames received                                        |
| IFOE_TX_LLR_OK_PACKETS                                  | Total number of successful LLR packets transmitted                               |
| IFOE_RX_LLR_OK_PACKETS                                  | Total number of successful LLR packets received                                  |
| IFOE_TX_LLR_REPLAY_COUNT                                | Total LLR replay count on transmit                                               |
| IFOE_TX_LLR_REPLAYS_COMPLETED                           | Total number of completed LLR replays on transmit                                |
| IFOE_RX_LLR_BAD_PACKETS                                 | Total number of bad LLR packets received                                         |
| IFOE_RX_LLR_DUPL_SEQ_PACKETS                            | Total number of duplicate sequence LLR packets received                          |
| IFOE_RX_FEC_BIT_ERR_0TO1_LANE0                          | FEC bit error 0-to-1 count on lane 0                                             |
| IFOE_RX_FEC_BIT_ERR_0TO1_LANE1                          | FEC bit error 0-to-1 count on lane 1                                             |
| IFOE_RX_FEC_BIT_ERR_0TO1_LANE2                          | FEC bit error 0-to-1 count on lane 2                                             |
| IFOE_RX_FEC_BIT_ERR_0TO1_LANE3                          | FEC bit error 0-to-1 count on lane 3                                             |
| IFOE_RX_FEC_BIT_ERR_1TO0_LANE0                          | FEC bit error 1-to-0 count on lane 0                                             |
| IFOE_RX_FEC_BIT_ERR_1TO0_LANE1                          | FEC bit error 1-to-0 count on lane 1                                             |
| IFOE_RX_FEC_BIT_ERR_1TO0_LANE2                          | FEC bit error 1-to-0 count on lane 2                                             |
| IFOE_RX_FEC_BIT_ERR_1TO0_LANE3                          | FEC bit error 1-to-0 count on lane 3                                             |
| IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE0                      | FEC symbol error count on lane 0                                                 |
| IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE1                      | FEC symbol error count on lane 1                                                 |
| IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE2                      | FEC symbol error count on lane 2                                                 |
| IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE3                      | FEC symbol error count on lane 3                                                 |
| IFOE_RX_BAD_CODE_COUNT                                  | Count of bad code received on the network port                                   |
| IFOE_RX_STOMPED_FCS                                     | Count of stomped FCS received on the network port                                |
| IFOE_STATION_TX_REQUEST_PACKETS                         | Total request packets transmitted by the station                                 |
| IFOE_STATION_TX_RESPONSE_PACKETS                        | Total response packets transmitted by the station                                |
| IFOE_STATION_RX_REQUEST_PACKETS                         | Total request packets received by the station                                    |
| IFOE_STATION_RX_RESPONSE_PACKETS                        | Total response packets received by the station                                   |
| IFOE_STATION_STREAM_REMAPS_TOTAL                        | Total number of stream remaps across all network ports                           |
| IFOE_STATION_PAUSED_STREAMS_COUNT                       | Number of currently paused streams on the station                                |
| IFOE_STATION_STREAM_REMAPS_NETWORK_PORT0                | Count of streams remapped on network port 0                                      |
| IFOE_STATION_STREAM_REMAPS_NETWORK_PORT1                | Count of streams remapped on network port 1                                      |
| IFOE_STATION_STREAM_REMAPS_NETWORK_PORT2                | Count of streams remapped on network port 2                                      |
| IFOE_STATION_STREAM_REMAPS_NETWORK_PORT3                | Count of streams remapped on network port 3                                      |
| IFOE_STATION_CRYPTO_TX_KEY_UPDATES_SA0                  | Crypto TX key updates for security association 0                                 |
| IFOE_STATION_CRYPTO_RX_KEY0_UPDATES_SA0                 | Crypto RX key0 updates for security association 0                                |
| IFOE_STATION_CRYPTO_RX_KEY1_UPDATES_SA0                 | Crypto RX key1 updates for security association 0                                |
| IFOE_STATION_CRYPTO_RX_KEY_DISABLES_SA0                 | Crypto RX key disables for security association 0                                |
| IFOE_STATION_CRYPTO_TX_KEY_UPDATES_SA1                  | Crypto TX key updates for security association 1                                 |
| IFOE_STATION_CRYPTO_RX_KEY0_UPDATES_SA1                 | Crypto RX key0 updates for security association 1                                |
| IFOE_STATION_CRYPTO_RX_KEY1_UPDATES_SA1                 | Crypto RX key1 updates for security association 1                                |
| IFOE_STATION_CRYPTO_RX_KEY_DISABLES_SA1                 | Crypto RX key disables for security association 1                                |
| IFOE_DISCARD_Q_RX_DROPPED_PACKETS                       | Discard queue RX dropped packets                                                 |
| IFOE_NONIFOE_Q_RX_TOTAL_PACKETS                         | Non-IFoE queue RX total packets                                                  |
| IFOE_NONIFOE_Q_RX_XOFF_TOTAL                            | Non-IFoE queue RX XOFF total                                                     |
| IFOE_NONIFOE_Q_TX_TOTAL_PACKETS                         | Non-IFoE queue TX total packets                                                  |
| IFOE_NONIFOE_Q_TX_XOFF_TOTAL                            | Non-IFoE queue TX XOFF total                                                     |
| IFOE_REQ_Q_RX_DROPPED_PACKETS                           | Request queue RX dropped packets                                                 |
| IFOE_RES_Q_RX_DROPPED_PACKETS                           | Response queue RX dropped packets                                                |
| IFOE_RX_PAUSE_PACKETS_RCVD                              | Total PFC pause packets received                                                 |
| IFOE_TX_PAUSE_PACKETS_SENT                              | Total PFC pause packets sent                                                     |
| IFOE_STATION_RENG_FREE_BLK_OUT                          | RENG free block out count                                                        |
| IFOE_STATION_RENG_FREE_PKT_OUT                          | RENG free packet out count                                                       |
| IFOE_STATION_RENG_FREE_SCH_IN                           | RENG free schedule in count                                                      |
| IFOE_STATION_RENG_READ_PKT_OUT                          | RENG read packet out count                                                       |
| IFOE_STATION_RENG_READ_SCH_IN                           | RENG read schedule in count                                                      |
| IFOE_STATION_RX_DECAP_DROPPED_PKTS                      | RX decap dropped packets                                                         |
| IFOE_STATION_RX_DECAP_RX_NAK_EGRESS_REQ                 | RX decap RX NAK egress request count                                             |
| IFOE_STATION_RX_DECAP_RX_NAK_EGRESS_RSP                 | RX decap RX NAK egress response count                                            |
| IFOE_STATION_RX_DECAP_TX_NAK_EGRESS_REQ                 | RX decap TX NAK egress request count                                             |
| IFOE_STATION_RX_DECAP_TX_NAK_EGRESS_RSP                 | RX decap TX NAK egress response count                                            |
| IFOE_STATION_RX_DROPPED_PKTS                            | Total RX dropped packets on station switch                                       |
| IFOE_STATION_RX_NONIFOE_PKTS                            | Total RX non-IFoE packets on station switch                                      |
| IFOE_STATION_SDP_RX_UNPACK_ORIGDATA_CREDITS_CONSUMED    | SDP RX unpack origdata credits consumed                                          |
| IFOE_STATION_SDP_RX_UNPACK_ORIGDATA_CREDITS_RETURNED    | SDP RX unpack origdata credits returned                                          |
| IFOE_STATION_SDP_RX_UNPACK_RDRSP_CREDITS_CONSUMED       | SDP RX unpack rdrsp credits consumed                                             |
| IFOE_STATION_SDP_RX_UNPACK_RDRSP_CREDITS_RETURNED       | SDP RX unpack rdrsp credits returned                                             |
| IFOE_STATION_SDP_RX_UNPACK_REQ_CREDITS_CONSUMED         | SDP RX unpack req credits consumed                                               |
| IFOE_STATION_SDP_RX_UNPACK_REQ_CREDITS_RETURNED         | SDP RX unpack req credits returned                                               |
| IFOE_STATION_SDP_RX_UNPACK_REQ_CYCLES_STALLED           | SDP RX unpack req cycles stalled                                                 |
| IFOE_STATION_SDP_RX_UNPACK_REQ_CYCLES_STALLED_CNT       | SDP RX unpack req cycles stalled cnt                                             |
| IFOE_STATION_SDP_RX_UNPACK_REQ_EXCESS_CREDITS_RETURNED  | SDP RX unpack req excess credits returned                                        |
| IFOE_STATION_SDP_RX_UNPACK_REQ_PAYLOAD_CREDITS_RETURNED | SDP RX unpack req payload credits returned                                       |
| IFOE_STATION_SDP_RX_UNPACK_REQ_TOTAL_CREDITS_CONSUMED   | SDP RX unpack req total credits consumed                                         |
| IFOE_STATION_SDP_RX_UNPACK_RETAG_FREED                  | SDP RX unpack retag freed                                                        |
| IFOE_STATION_SDP_RX_UNPACK_RETAG_USED                   | SDP RX unpack retag used                                                         |
| IFOE_STATION_SDP_RX_UNPACK_RSP_CYCLES_STALLED           | SDP RX unpack rsp cycles stalled                                                 |
| IFOE_STATION_SDP_RX_UNPACK_RSP_CYCLES_STALLED_CNT       | SDP RX unpack rsp cycles stalled cnt                                             |
| IFOE_STATION_SDP_RX_UNPACK_RSP_EXCESS_CREDITS_RETURNED  | SDP RX unpack rsp excess credits returned                                        |
| IFOE_STATION_SDP_RX_UNPACK_RSP_PAYLOAD_CREDITS_RETURNED | SDP RX unpack rsp payload credits returned                                       |
| IFOE_STATION_SDP_RX_UNPACK_RSP_TOTAL_CREDITS_CONSUMED   | SDP RX unpack rsp total credits consumed                                         |
| IFOE_STATION_SDP_RX_UNPACK_WRRSP_CREDITS_CONSUMED       | SDP RX unpack wrrsp credits consumed                                             |
| IFOE_STATION_SDP_RX_UNPACK_WRRSP_CREDITS_RETURNED       | SDP RX unpack wrrsp credits returned                                             |
| IFOE_STATION_SDP_TX_PACK_ATM_REQ                        | SDP TX pack atm req                                                              |
| IFOE_STATION_SDP_TX_PACK_ORIG_DATA                      | SDP TX pack orig data                                                            |
| IFOE_STATION_SDP_TX_PACK_ORIG_DATA_CREDIT_CONSUMED      | SDP TX pack orig data credit consumed                                            |
| IFOE_STATION_SDP_TX_PACK_ORIG_DATA_CREDIT_RELEASED      | SDP TX pack orig data credit released                                            |
| IFOE_STATION_SDP_TX_PACK_ORIG_DATA_EB_EMPTY             | SDP TX pack orig data eb empty                                                   |
| IFOE_STATION_SDP_TX_PACK_ORIG_DATA_EB_FULL              | SDP TX pack orig data eb full                                                    |
| IFOE_STATION_SDP_TX_PACK_ORIG_DATA_ERROR                | SDP TX pack orig data error                                                      |
| IFOE_STATION_SDP_TX_PACK_RD_REQ                         | SDP TX pack rd req                                                               |
| IFOE_STATION_SDP_TX_PACK_RD_RSP                         | SDP TX pack rd rsp                                                               |
| IFOE_STATION_SDP_TX_PACK_RD_RSP_CREDIT_CONSUMED         | SDP TX pack rd rsp credit consumed                                               |
| IFOE_STATION_SDP_TX_PACK_RD_RSP_CREDIT_RELEASED         | SDP TX pack rd rsp credit released                                               |
| IFOE_STATION_SDP_TX_PACK_RD_RSP_DATA_ERROR              | SDP TX pack rd rsp data error                                                    |
| IFOE_STATION_SDP_TX_PACK_RD_RSP_EB_EMPTY                | SDP TX pack rd rsp eb empty                                                      |
| IFOE_STATION_SDP_TX_PACK_RD_RSP_EB_FULL                 | SDP TX pack rd rsp eb full                                                       |
| IFOE_STATION_SDP_TX_PACK_RD_RSP_PCL_EMPTY               | SDP TX pack rd rsp pcl empty                                                     |
| IFOE_STATION_SDP_TX_PACK_RD_RSP_PCL_FULL                | SDP TX pack rd rsp pcl full                                                      |
| IFOE_STATION_SDP_TX_PACK_REQ_CREDIT_CONSUMED            | SDP TX pack req credit consumed                                                  |
| IFOE_STATION_SDP_TX_PACK_REQ_CREDIT_RELEASED            | SDP TX pack req credit released                                                  |
| IFOE_STATION_SDP_TX_PACK_REQ_EB_EMPTY                   | SDP TX pack req eb empty                                                         |
| IFOE_STATION_SDP_TX_PACK_REQ_EB_FULL                    | SDP TX pack req eb full                                                          |
| IFOE_STATION_SDP_TX_PACK_REQ_PCL                        | SDP TX pack req pcl                                                              |
| IFOE_STATION_SDP_TX_PACK_REQ_PCL_EMPTY                  | SDP TX pack req pcl empty                                                        |
| IFOE_STATION_SDP_TX_PACK_REQ_PCL_FULL                   | SDP TX pack req pcl full                                                         |
| IFOE_STATION_SDP_TX_PACK_REQ_POOL_EMPTY                 | SDP TX pack req pool empty                                                       |
| IFOE_STATION_SDP_TX_PACK_REQ_POOL_FULL                  | SDP TX pack req pool full                                                        |
| IFOE_STATION_SDP_TX_PACK_RSP_POOL_EMPTY                 | SDP TX pack rsp pool empty                                                       |
| IFOE_STATION_SDP_TX_PACK_RSP_POOL_FULL                  | SDP TX pack rsp pool full                                                        |
| IFOE_STATION_SDP_TX_PACK_WR_REQ                         | SDP TX pack wr req                                                               |
| IFOE_STATION_SDP_TX_PACK_WR_RSP                         | SDP TX pack wr rsp                                                               |
| IFOE_STATION_SDP_TX_PACK_WR_RSP_CREDIT_CONSUMED         | SDP TX pack wr rsp credit consumed                                               |
| IFOE_STATION_SDP_TX_PACK_WR_RSP_CREDIT_RELEASED         | SDP TX pack wr rsp credit released                                               |
| IFOE_STATION_SDP_TX_PACK_WR_RSP_EB_EMPTY                | SDP TX pack wr rsp eb empty                                                      |
| IFOE_STATION_SDP_TX_PACK_WR_RSP_EB_FULL                 | SDP TX pack wr rsp eb full                                                       |
| IFOE_STATION_SDP_TX_PACK_WR_RSP_PCL_EMPTY               | SDP TX pack wr rsp pcl empty                                                     |
| IFOE_STATION_SDP_TX_PACK_WR_RSP_PCL_FULL                | SDP TX pack wr rsp pcl full                                                      |
| IFOE_STATION_TX_ENCAP_PKT_EGRESS_XRSEC_NPORT_0          | TX encap packet egress cross-section on nport 0                                  |
| IFOE_STATION_TX_ENCAP_PKT_EGRESS_XRSEC_NPORT_1          | TX encap packet egress cross-section on nport 1                                  |
| IFOE_STATION_TX_ENCAP_PKT_EGRESS_XRSEC_NPORT_2          | TX encap packet egress cross-section on nport 2                                  |
| IFOE_STATION_TX_ENCAP_PKT_EGRESS_XRSEC_NPORT_3          | TX encap packet egress cross-section on nport 3                                  |
| IFOE_STATION_TX_ENCAP_STALL_EGRESS_XRSEC                | TX encap stall egress cross-section count                                        |
| IFOE_STATION_TX_ENCAP_STALL_INGRESS                     | TX encap stall ingress count                                                     |
| IFOE_STATION_TX_NONIFOE_PKTS                            | Total TX non-IFoE packets on station switch                                      |
| IFOE_STATION_TX_SCHED_ACTIVE_RX_ACK_PKTS                | TX scheduler active RX ACK packets                                               |
| IFOE_STATION_TX_SCHED_ACTIVE_STREAMS                    | TX scheduler active streams count                                                |
| IFOE_STATION_TX_SCHED_BOOSTED_PRI_STREAMS               | TX scheduler boosted priority streams                                            |
| IFOE_STATION_TX_SCHED_EMPTY_QUEUE_STREAMS               | TX scheduler empty queue streams                                                 |
| IFOE_STATION_TX_SCHED_EMPTY_SEND_QUEUE_STREAMS          | TX scheduler empty send queue streams                                            |
| IFOE_STATION_TX_SCHED_PAUSED_STREAMS                    | TX scheduler paused streams count                                                |
| IFOE_STATION_TX_SCHED_REQ_PKTS                          | TX scheduler request packets                                                     |
| IFOE_STATION_TX_SCHED_RES_PKTS                          | TX scheduler response packets                                                    |
| IFOE_STATION_WENG_BSTATE_FREE_BLK_DELAY_REQ             | WENG bstate free blk delay req                                                   |
| IFOE_STATION_WENG_BSTATE_FREE_BLK_DELAY_RES             | WENG bstate free blk delay res                                                   |
| IFOE_STATION_WENG_EVICTION_CHAIN_DELAY_REQ              | WENG eviction chain delay req                                                    |
| IFOE_STATION_WENG_EVICTION_CHAIN_DELAY_RES              | WENG eviction chain delay res                                                    |
| IFOE_STATION_WENG_EVICTION_EXPIRY_REQ                   | WENG eviction expiry req                                                         |
| IFOE_STATION_WENG_EVICTION_EXPIRY_RES                   | WENG eviction expiry res                                                         |
| IFOE_STATION_WENG_EVICTION_FORCE_CHAIN_DELAY_REQ        | WENG eviction force chain delay req                                              |
| IFOE_STATION_WENG_EVICTION_FORCE_CHAIN_DELAY_RES        | WENG eviction force chain delay res                                              |
| IFOE_STATION_WENG_EVICTION_FORCE_HOLD_DELAY_REQ         | WENG eviction force hold delay req                                               |
| IFOE_STATION_WENG_EVICTION_FORCE_HOLD_DELAY_RES         | WENG eviction force hold delay res                                               |
| IFOE_STATION_WENG_EVICTION_FORCE_REQ                    | WENG eviction force req                                                          |
| IFOE_STATION_WENG_EVICTION_FORCE_RES                    | WENG eviction force res                                                          |
| IFOE_STATION_WENG_EVICTION_MTU_HOLD_DELAY_REQ           | WENG eviction mtu hold delay req                                                 |
| IFOE_STATION_WENG_EVICTION_MTU_HOLD_DELAY_RES           | WENG eviction mtu hold delay res                                                 |
| IFOE_STATION_WENG_EVICTION_MTU_REQ                      | WENG eviction mtu req                                                            |
| IFOE_STATION_WENG_EVICTION_MTU_RES                      | WENG eviction mtu res                                                            |
| IFOE_STATION_WENG_EVICTION_PAYLOAD_REQ                  | WENG eviction payload req                                                        |
| IFOE_STATION_WENG_EVICTION_PAYLOAD_RES                  | WENG eviction payload res                                                        |
| IFOE_STATION_WENG_OP_SDP_CHAIN_REQ                      | WENG op sdp chain req                                                            |
| IFOE_STATION_WENG_OP_SDP_CHAIN_RES                      | WENG op sdp chain res                                                            |
| IFOE_STATION_WENG_OP_SDP_FORCE_REQ                      | WENG op sdp force req                                                            |
| IFOE_STATION_WENG_OP_SDP_FORCE_RES                      | WENG op sdp force res                                                            |
| IFOE_STATION_WENG_OP_SDP_LENGTH_REQ                     | WENG op sdp length req                                                           |
| IFOE_STATION_WENG_OP_SDP_LENGTH_RES                     | WENG op sdp length res                                                           |
| IFOE_STATION_WENG_OP_SDP_PKT_OPEN_REQ                   | WENG op sdp pkt open req                                                         |
| IFOE_STATION_WENG_OP_SDP_PKT_OPEN_RES                   | WENG op sdp pkt open res                                                         |
| IFOE_STATION_WENG_OP_SDP_REQ                            | WENG op sdp req                                                                  |
| IFOE_STATION_WENG_OP_SDP_RES                            | WENG op sdp res                                                                  |
| IFOE_STATION_WENG_OP_SDP_WORD_REQ                       | WENG op sdp word req                                                             |
| IFOE_STATION_WENG_OP_SDP_WORD_RES                       | WENG op sdp word res                                                             |
