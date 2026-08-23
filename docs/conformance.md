# OpenGoPro conformance matrix

Validated 2026-08-23 against gopro.github.io/OpenGoPro (docs/ble/*) and the
official Python SDK constants (demos/python/sdk_wireless_camera_control).
Re-validated 2026-08-23 against the full gh-pages spec build plus all upstream
demo implementations (Python SDK, Kotlin kmp_sdk, Swift, C#, C/C++, tutorials).
Verdicts: conform | fixed | deviation (kept, reason given) | enhancement (not implemented).

Trust note: the spec's advertisement bit tables are LSB-first (Bit 0 = 0x01).
The Python SDK parses them MSB-first via construct BitStruct; that parser is
unused upstream and wrong. The Kotlin SDK matches the spec and has a
real-capture test vector. When SDKs disagree, the spec + Kotlin vector win.

## BLE setup and GATT

| Item | Ours | Spec | Verdict |
|---|---|---|---|
| Scan filter | FEA6 UUID + "gopro" name fallback | FEA6 | conform (fallback is additive) |
| Service GP-0001 SSID/password chars | GP-0002 read, GP-0003 read | same | conform |
| Control chars GP-0072/73/74/75/76/77 | match | same | conform |
| Network mgmt GP-0091 write, GP-0092 notify | match | same | conform |
| WiFi AP Power GP-0004 / State GP-0005 | unused; we use command 0x17 | either path valid | conform |
| Re-subscribe on each connect | yes (no caching assumed) | required | conform |
| Adv manufacturer data (pairing flag, new-media flag, model id, serial) | parsed; drives identity, auto-pair, status checks | available | fixed 2026-08-23: bit masks were MSB-first (copied from the buggy Python SDK parser); now LSB-first per spec (processor 0x01, wifi 0x02, pairing 0x04, new media 0x10) |
| Serial from advertisement | assembled: model prefix + schema v2 id_hash or v3 chars 4-5 + service data tail (4 or 8 chars) | serial is split across manuf + service data; first 4 chars never broadcast | fixed 2026-08-23: previously expected 8+ chars directly after the AP MAC, which never matches; partial serials are never stored |
| Fragment accumulation | one collector per notify characteristic | accumulate per source UUID | fixed 2026-08-23: was one collector per device; a push during a fragmented response corrupted both |

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
| AP control | 0x17, param 1 byte | SET_WIFI 0x17 | conform; mode 2 "bounce" is in spec ("disable then enable") and now used for AP recovery |
| WiFi AP readiness | poll status 69 up to 10s before joining; bounce AP on timeout | spec: wait for AP Mode (69) == 1 before connecting | conform (implemented 2026-08-23; replaced a blind 2s sleep) |
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

## Settings

| Item | Ours | Spec | Verdict |
|---|---|---|---|
| Get setting values | bare 0x12 (empty = all) | GET_SETTING_VAL, "empty array queries all" | conform; works on HERO11 (267B response) |
| Get setting capabilities | 0x32 with one setting ID per query | GET_CAPABILITIES_VAL | deviation from bare form kept: the spec allows empty = all, but no reference implementation sends it (Python SDK queries per setting) and a HERO11 never finishes answering it. Per-setting matches upstream practice |
| Set setting | TLV [id][len][value] on GP-0074 | Change Setting | conform; status 2 = option invalid in current state |
| Per-model gating | camera-reported capabilities only | capabilities depend on model and state | conform by construction: no static model tables |

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
| Media list parse | n/cre/mod/s + group fields | same schema | conform |
| Grouped media (burst, time lapse) | members expanded from b..l skipping m; "s" treated as count, not bytes | grouped entries list only the first member; s is the member count | fixed 2026-08-23: previously only the first member downloaded and s misread as bytes |
| Turbo transfer | enabled for the download window, disabled after (best effort) | recommended only during media offload | conform (implemented 2026-08-23) |
| File types | .mp4/.jpg only | .360, LRV/THM also exist | deviation kept: by design, primary media only |
| cre epoch seconds | time.Unix(cre, 0) | epoch seconds | conform |

## Enhancements not implemented (deliberate)

- Set Camera Control (protobuf 0xF1/0x69, claim EXTERNAL_CONTROL): spec says a
  third-party client "should" claim control. Fleet cameras are headless;
  revisit if camera-UI contention appears (status 114 reports the holder).
- Get Open GoPro Version (0x51) handshake: SDK asserts "2.0". Low value; our
  commands fail loudly enough on unsupported cameras.
- Status 82 (Ready): documented but never prescribed; hardware-info polling is
  the spec's readiness gate and we do that.
- New-media flag clearing (protobuf 0xF1/0x7C): we track sync state ourselves;
  clearing the camera flag would break other clients' view.

## Known model gaps (documented, not coded)

- LIT HERO (model 70): Sleep 0x05 and Set Date Time 0x0D are not listed as
  supported. Our Sleep on disconnect will fail harmlessly there.
- Cameras advertise for only 8 hours after sleep; beyond that they need a
  button press. Fleet reachability windows must assume this.
- 2-byte setting/status IDs (Mission 1 family) are not implemented; gate on
  Get Camera Capabilities if those models ever matter.

## Hardware validation

Runnable harness: `make probe`, then next to a charged camera:

1. `bin/ble-probe scan`
   Dumps raw + parsed advertisements and reprints on any flag change.
   Toggle pairing mode on the camera: the pairing flag must flip.
   Proves: bit-order fix, serial assembly, model id.
2. `bin/ble-probe pair "GoPro 0711"` (camera in pairing mode, unbonded client)
   Proves: bonding, credentials read, pairing bit, and (manual check:
   the camera's pairing screen closes by itself) RequestPairingFinish framing.
3. `bin/ble-probe validate "GoPro 0711" -wifi -sleep`
   Proves: readiness poll, keep-alive on GP-0074/0075, status query,
   AP-ready gate (status 69 timing printed), WiFi join, HTTP state,
   media list incl. group expansion, turbo transfer, Sleep.

Run 2 then 3 on: HERO12 and HERO13+ on Linux; any camera on macOS 15
(after `make wifi-helper`). Non-zero exit means a failed check.

Validated so far:
- 2026-08-23, HERO11 Black (fw H22.01.02.32.00), Linux:
  full app flow works end to end, and `ble-probe validate -wifi -sleep`
  passed 12/12: connect+readiness, keep-alive success on GP-0075,
  statuses 8/10/33/70, AP-ready via status 69, credentials, WiFi join,
  HTTP state, media list, turbo transfer (transfer UI seen), Sleep.
- Bit-order fix proven with a changing value: the status byte read
  0x00 while advertising from sleep and 0x03 once awake with AP on,
  exactly bits 0 (processor) and 1 (wifi AP) LSB-first per spec.
- Observed: this HERO11's schema 2 id_hash is the reversed BLE MAC,
  not serial chars. Confirms never fabricating serials from schema 2;
  service data tail matched the GATT serial's last 4 ("8614").
  Model 58 in the advertisement matched GATT model and the C347 prefix.
  Still open: pair on an unbonded client (pairing-finish manual check),
  grouped media expansion (needs burst/timelapse content on the card),
  HERO13+ (reconnect-to-sleep quirk, model id heuristic),
  schema 3 serial assembly (needs a newer camera), macOS paths.
- 2026-08-24, HERO11 Black, Linux, MT7925 host NIC: downloads capped
  at 0.6 MB/s in every configuration (BLE held or dropped, 1 or 4
  parallel streams, turbo off or on, 5GHz or 2.4GHz, any distance,
  host powersave and PCIe runtime PM off; curl reproduces it, so the
  Go path was never the bottleneck). Symptom in `iw station dump`:
  camera tx pinned at MCS 0-1 while the laptop uplink runs MCS 9.
  Root cause: mt7925 driver bug in kernel 7.1.x; the firmware keeps
  stale station state after an AP switch, so joining the camera AP
  after any other network pins the rate. Fixed upstream, not in 7.1.
  Proof: `modprobe -r mt7925e && modprobe mt7925e`, then joining the
  camera AP first gives 22.5 MB/s; one switch through another AP and
  back re-breaks it to 0.6.
  Healthy-link matrix (20s samples): 1 stream no turbo 22.5 MB/s
  (BLE held 20.5), 4 chunks 20.5, turbo 17-18. Conclusions: keep the
  single sequential stream, keep turbo_enabled off (turbo loses ~20%
  on HERO11), holding BLE during transfer costs ~10% (acceptable).
  DownloadVideos warns when a whole sync sustains under 2 MB/s.
- 2026-08-24, macOS readiness review (code only, no hardware yet):
  all nine fixes from the macOS debug session (tmp/FIXES.md) are merged:
  name-based dedup for random CoreBluetooth UUIDs, callback capture,
  connect-done timing, write-with-response on darwin, CoreWLAN helper,
  networksetup verification, 120s connect timeout, BLE address re-read
  before sync. tinygo bluetooth v0.14.0 exposes LocalName, ServiceUUIDs,
  ManufacturerData and ServiceData on darwin, so the FEA6 filter,
  status flags and serial-tail assembly all work; the first 4 serial
  chars still come from GATT as on Linux. Gaps closed this pass:
  packaged app was missing NSBluetoothAlwaysUsageDescription (TCC kills
  the backend on first CoreBluetooth call; dev runs hid this because
  Terminal/Electron carry their own keys) and location keys for CoreWLAN
  scans; wifi_join scan timeout and helper failure now degrade to
  networksetup retries instead of failing the sync, since macOS 15
  redacts scan results without location permission. Still hardware-only:
  TCC prompt flow, wifi_join under location gating, pair from unbonded
  client, Sleep on disconnect, HERO13+ paths. Known accepted limits:
  isTransientBLEError always false on darwin (no status-check retry);
  CI dmg is arm64 only while the Go binary is universal.
