# OpenGoPro conformance matrix

Validated 2026-08-23 against gopro.github.io/OpenGoPro (docs/ble/*) and the
official Python SDK constants (demos/python/sdk_wireless_camera_control).
Verdicts: conform | fixed | deviation (kept, reason given) | enhancement (not implemented).

## BLE setup and GATT

| Item | Ours | Spec | Verdict |
|---|---|---|---|
| Scan filter | FEA6 UUID + "gopro" name fallback | FEA6 | conform (fallback is additive) |
| Service GP-0001 SSID/password chars | GP-0002 read, GP-0003 read | same | conform |
| Control chars GP-0072/73/74/75/76/77 | match | same | conform |
| Network mgmt GP-0091 write, GP-0092 notify | match | same | conform |
| WiFi AP Power GP-0004 / State GP-0005 | unused; we use command 0x17 | either path valid | conform |
| Re-subscribe on each connect | yes (no caching assumed) | required | conform |
| Adv manufacturer data (pairing flag, new-media flag, model id, serial) | parsed; drives identity, auto-pair, status checks | available | conform (implemented 2026-08-23) |

## Pairing

| Item | Ours | Spec | Verdict |
|---|---|---|---|
| Bond during connect (D-Bus, Linux) | yes, pre-service-discovery | pair once per client | conform |
| RequestPairingFinish feature/action | 0x03 / 0x01 on GP-0091 | WIRELESS_MANAGEMENT / SET_PAIRING_STATE | conform |
| RequestPairingFinish protobuf bytes | field1 varint SUCCESS, field2 phoneName | same (network_management.proto) | conform |
| Protobuf message framing | was sent with NO packet header | all messages use packet headers | fixed: buildProtobufPacket now frames via BuildTLVPackets. Explains "camera never responds to 0x03": the camera parsed byte 0x03 as a length header |

## Packetization (data_protocol)

| Item | Ours | Spec | Verdict |
|---|---|---|---|
| 20-byte packet limit | MaxPacketSize 20 | <= 20 | conform |
| Send format | ext-13 always | ext-13 recommended | conform |
| Ext-16 receive-only | parsed, never sent | receive only | conform |
| Continuation counter | starts 0x0, wraps at 0xF | same | conform (golden test) |
| Response layout | [id][status][payload] | same | conform |

## Commands (id_tables / control)

| Item | Ours | Spec | Verdict |
|---|---|---|---|
| Sleep | 0x05, no params | SLEEP 0x05 | conform |
| Set Date Time fallback | 0x0D, 7-byte payload | SET_DATE_TIME | conform |
| Set Local Date Time | 0x0F, 10-byte payload (date+utc offset+dst) | SET_DATE_TIME_DST | conform |
| AP control | 0x17, param 1 byte | SET_WIFI 0x17 | conform; mode 2 "bounce" is not in spec and unused, kept |
| Get Hardware Info | 0x3C, length-prefixed field parse | GET_HW_INFO | conform |
| Third-party client | was 0x6B ("empirically works") | SET_THIRD_PARTY_CLIENT_INFO 0x50 | fixed: now 0x50; 0x6B is not in the command table |
| Keep-alive | was command 0x5B on GP-0072 | setting LED(91)=66 on GP-0074 | fixed: now a settings write; bytes were right, characteristic was wrong |
| Keep-alive cadence | 3s ticker during sync | SDK default 3s | conform |

## Queries and statuses

| Item | Ours | Spec | Verdict |
|---|---|---|---|
| Get status values | 0x13 | GET_STATUS_VAL | conform |
| Register status updates | 0x53 | REG_STATUS_VAL_UPDATE | conform |
| Unregister | 0x73 | UNREG_STATUS_VAL_UPDATE | conform |
| Status push | 0x93 (0x92 also routed) | STATUS_VAL_PUSH / SETTING_VAL_PUSH | conform |
| Status 33 SD status | PRIMARY_STORAGE | same | conform |
| Status 38/39 photo/video counts | PHOTOS / VIDEOS | same | conform |
| Status 54 remaining KB | SD_CARD_REMAINING | same | conform |
| Status 70 battery percent | INTERNAL_BATTERY_PERCENTAGE | same | conform |
| -1 / 255 sentinels after wake | special-cased, not stored | not documented upstream | deviation kept: observed HERO13 firmware behavior |

## State management

| Item | Ours | Spec | Verdict |
|---|---|---|---|
| Readiness after connect | poll Get Hardware Info until success | ble_setup prescribes exactly this | conform |
| Busy/encoding gating | sync waits for idle; Sleep skipped while busy/recording | spec: wait for System Busy (8) and Encoding (10) unset | conform (implemented 2026-08-23) |
| HERO13+ reconnect-to-sleep (model >= 64) | workaround | not documented upstream | deviation kept: firmware quirk, documented in code |

## WiFi / HTTP

| Item | Ours | Spec | Verdict |
|---|---|---|---|
| Base URL | 10.5.5.9:8080 | same | conform |
| Camera state | /gopro/camera/state | same | conform |
| Media list | /gopro/media/list | same | conform |
| Download | /videos/DCIM/<dir>/<file>, Range resume | same endpoint; Range supported | conform |
| Media list parse | n/cre/mod/s + group fields | same schema | conform; group fields (b/l/g) parsed but unused |
| File types | .mp4/.jpg only | .360, LRV/THM also exist | deviation kept: by design, primary media only |
| cre epoch seconds | time.Unix(cre, 0) | epoch seconds | conform |

## Follow-ups (not code fixes)

- Hardware validation needed on real cameras (HERO12 and HERO13+ on Linux,
  any camera on macOS 15): keep-alive fix, pairing-finish framing,
  third-party-client 0x50, advertisement parsing, busy gating, and the
  CoreWLAN wifi_join helper path.
