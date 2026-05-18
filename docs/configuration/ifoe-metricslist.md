# List of Available IFOE Metrics

This section provides an overview of the IFOE statistics available using the Device Metrics Exporter.

## Supported IFOE Metrics List

The following table contains a full list of IFOE Metrics that are available using the Device Metrics Exporter.

| Metric                                                | Description                                                                      |
|-------------------------------------------------------|----------------------------------------------------------------------------------|
| IFOE_TOTAL_DEVICES                                    | Total number of IFOE devices on the host                                         |
| IFOE_TOTAL_STATIONS                                   | Total number of IFOE stations across all devices                                 |
| IFOE_TOTAL_PORTS                                      | Total number of IFOE network ports across all stations                           |
| IFOE_NUMBER_FAILEDOVER_STREAMS                        | Number of IFOE streams that have experienced failover to redundant paths         |
| IFOE_NUMBER_PAUSED_STREAMS                            | Number of IFOE streams that are currently in a paused state                      |
| IFOE_BIT_ERROR_RATE                                   | Bit Error Rate (BER) reported by the network port                                |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS0                      | Total number of FEC codewords with 0 symbol errors (error-free codewords)        |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS1                      | Total number of FEC codewords with 1 symbol error                                |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS2                      | Total number of FEC codewords with 2 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS3                      | Total number of FEC codewords with 3 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS4                      | Total number of FEC codewords with 4 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS5                      | Total number of FEC codewords with 5 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS6                      | Total number of FEC codewords with 6 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS7                      | Total number of FEC codewords with 7 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS8                      | Total number of FEC codewords with 8 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS9                      | Total number of FEC codewords with 9 symbol errors                               |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS10                     | Total number of FEC codewords with 10 symbol errors                              |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS11                     | Total number of FEC codewords with 11 symbol errors                              |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS12                     | Total number of FEC codewords with 12 symbol errors                              |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS13                     | Total number of FEC codewords with 13 symbol errors                              |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS14                     | Total number of FEC codewords with 14 symbol errors                              |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS15                     | Total number of FEC codewords with 15 symbol errors (maximum correctable errors) |
| IFOE_FEC_CODEWORD_SYMBOL_ERRORS_UNCORRECTABLE         | Total number of FEC codewords with uncorrectable symbol errors                   |
| IFOE_PORT_LINK_STATE                                  | Current link state of the network port (0=down, 1=up)                            |
| IFOE_PORT_SPEED                                       | Current speed of the network port in Gbps                                        |
| IFOE_PORT_LINK_UP_COUNT                               | Number of times the link has transitioned to up state                            |
| IFOE_PORT_LINK_UP_DURATION_MSEC                       | Total duration the link has been up in milliseconds                              |
| IFOE_PORT_LINK_DOWN_DURATION_MSEC                     | Total duration the link has been down in milliseconds                            |
| IFOE_PORT_LINK_TRAINING_DURATION_LATEST_MSEC          | Duration of the most recent link training in milliseconds                        |
| IFOE_PORT_LINK_TRAINING_DURATION_AVG_MSEC             | Average link training duration in milliseconds                                   |
| IFOE_TX_TOTAL_BYTES                                   | Total number of bytes transmitted on the network port                            |
| IFOE_TX_TOTAL_GOOD_BYTES                              | Total number of good bytes transmitted                                           |
| IFOE_TX_TOTAL_ERR_BYTES                               | Total number of error bytes transmitted                                          |
| IFOE_TX_TOTAL_PACKETS                                 | Total number of packets transmitted                                              |
| IFOE_TX_TOTAL_GOOD_PACKETS                            | Total number of good packets transmitted                                         |
| IFOE_TX_FRAME_ERROR                                   | Total number of transmit frame errors                                            |
| IFOE_TX_BAD_FCS                                       | Total number of packets transmitted with bad FCS                                 |
| IFOE_RX_TOTAL_BYTES                                   | Total number of bytes received on the network port                               |
| IFOE_RX_TOTAL_GOOD_BYTES                              | Total number of good bytes received                                              |
| IFOE_RX_TOTAL_ERR_BYTES                               | Total number of error bytes received                                             |
| IFOE_RX_TOTAL_PACKETS                                 | Total number of packets received                                                 |
| IFOE_RX_TOTAL_GOOD_PACKETS                            | Total number of good packets received                                            |
| IFOE_RX_PACKET_DROPPED                                | Total number of received packets dropped                                         |
| IFOE_RX_BAD_FCS                                       | Total number of packets received with bad FCS                                    |
| IFOE_RX_FEC_CORRECTED_CODEWORDS                       | Total number of FEC corrected codewords received                                 |
| IFOE_RX_FEC_UNCORRECTED_CODEWORDS                     | Total number of FEC uncorrected codewords received                               |
| IFOE_TX_PAUSE                                         | Total number of pause frames transmitted                                         |
| IFOE_RX_PAUSE                                         | Total number of pause frames received                                            |
| IFOE_TX_USER_PAUSE                                    | Total number of user pause frames transmitted                                    |
| IFOE_RX_USER_PAUSE                                    | Total number of user pause frames received                                       |
| IFOE_RX_JABBER                                        | Total number of jabber frames received (oversized with bad CRC)                  |
| IFOE_RX_OVERSIZE                                      | Total number of oversized frames received                                        |
| IFOE_RX_TOO_LONG                                      | Total number of too-long frames received                                         |
| IFOE_RX_TRUNCATED                                     | Total number of truncated frames received                                        |
| IFOE_TX_LLR_OK_PACKETS                                | Total number of successful LLR packets transmitted                               |
| IFOE_RX_LLR_OK_PACKETS                                | Total number of successful LLR packets received                                  |
| IFOE_TX_LLR_REPLAY_COUNT                              | Total LLR replay count on transmit                                               |
| IFOE_TX_LLR_REPLAYS_COMPLETED                         | Total number of completed LLR replays on transmit                                |
| IFOE_RX_LLR_BAD_PACKETS                               | Total number of bad LLR packets received                                         |
| IFOE_RX_LLR_DUPL_SEQ_PACKETS                          | Total number of duplicate sequence LLR packets received                          |
| IFOE_RX_FEC_BIT_ERR_0TO1_LANE0                        | FEC bit error 0-to-1 count on lane 0                                             |
| IFOE_RX_FEC_BIT_ERR_0TO1_LANE1                        | FEC bit error 0-to-1 count on lane 1                                             |
| IFOE_RX_FEC_BIT_ERR_0TO1_LANE2                        | FEC bit error 0-to-1 count on lane 2                                             |
| IFOE_RX_FEC_BIT_ERR_0TO1_LANE3                        | FEC bit error 0-to-1 count on lane 3                                             |
| IFOE_RX_FEC_BIT_ERR_1TO0_LANE0                        | FEC bit error 1-to-0 count on lane 0                                             |
| IFOE_RX_FEC_BIT_ERR_1TO0_LANE1                        | FEC bit error 1-to-0 count on lane 1                                             |
| IFOE_RX_FEC_BIT_ERR_1TO0_LANE2                        | FEC bit error 1-to-0 count on lane 2                                             |
| IFOE_RX_FEC_BIT_ERR_1TO0_LANE3                        | FEC bit error 1-to-0 count on lane 3                                             |
| IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE0                    | FEC symbol error count on lane 0                                                 |
| IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE1                    | FEC symbol error count on lane 1                                                 |
| IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE2                    | FEC symbol error count on lane 2                                                 |
| IFOE_RX_FEC_SYMBOL_ERR_COUNT_LANE3                    | FEC symbol error count on lane 3                                                 |
| IFOE_RX_BAD_CODE_COUNT                                | Count of bad code received on the network port                                   |
| IFOE_RX_STOMPED_FCS                                   | Count of stomped FCS received on the network port                                |
| IFOE_STATION_TX_REQUEST_PACKETS                       | Total request packets transmitted by the station                                 |
| IFOE_STATION_TX_RESPONSE_PACKETS                      | Total response packets transmitted by the station                                |
| IFOE_STATION_RX_REQUEST_PACKETS                       | Total request packets received by the station                                    |
| IFOE_STATION_RX_RESPONSE_PACKETS                      | Total response packets received by the station                                   |
| IFOE_STATION_STREAM_REMAPS_TOTAL                      | Total number of stream remaps across all network ports                           |
| IFOE_STATION_PAUSED_STREAMS_COUNT                     | Number of currently paused streams on the station                                |
| IFOE_STATION_STREAM_REMAPS_NETWORK_PORT0              | Count of streams remapped on network port 0                                      |
| IFOE_STATION_STREAM_REMAPS_NETWORK_PORT1              | Count of streams remapped on network port 1                                      |
| IFOE_STATION_STREAM_REMAPS_NETWORK_PORT2              | Count of streams remapped on network port 2                                      |
| IFOE_STATION_STREAM_REMAPS_NETWORK_PORT3              | Count of streams remapped on network port 3                                      |
| IFOE_STATION_CRYPTO_TX_KEY_UPDATES_SA0                | Crypto TX key updates for security association 0                                 |
| IFOE_STATION_CRYPTO_RX_KEY0_UPDATES_SA0               | Crypto RX key0 updates for security association 0                                |
| IFOE_STATION_CRYPTO_RX_KEY1_UPDATES_SA0               | Crypto RX key1 updates for security association 0                                |
| IFOE_STATION_CRYPTO_RX_KEY_DISABLES_SA0               | Crypto RX key disables for security association 0                                |
| IFOE_STATION_CRYPTO_TX_KEY_UPDATES_SA1                | Crypto TX key updates for security association 1                                 |
| IFOE_STATION_CRYPTO_RX_KEY0_UPDATES_SA1               | Crypto RX key0 updates for security association 1                                |
| IFOE_STATION_CRYPTO_RX_KEY1_UPDATES_SA1               | Crypto RX key1 updates for security association 1                                |
| IFOE_STATION_CRYPTO_RX_KEY_DISABLES_SA1               | Crypto RX key disables for security association 1                                |
