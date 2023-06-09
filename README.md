# openmiio_agent

This project allows you to significantly extend the functionality of your gateways on the original firmware, keeping
almost all default functionality of the device in the Xiaomi Mi Home ecosystem.

**Features**

- Full support original Xiaomi firmware
- Access to gateway's MQTT
- miIO to MQTT for send commands and receive device updates (also without Internet)
- ZHA and zigbee2mqtt support (on-demand mode)
- Lua scripts for changing default gateway logic
- BLE events without Internet
- Fix difference for BLE specs
- Zigbee custom firmware support

| Supported gateway  | Xiaomi Multimode Gateway     | Xiaomi Multimode Gateway 2 | Aqara Hub E1 |
|--------------------|------------------------------|----------------------------|--------------|
| Supported models   | `ZNDMWG03LM`<br>`ZNDMWG02LM` | `DMWG03LM`<br>`ZNDMWG04LM` | `ZHWG16LM`   |
| Mi Home Zigbee     | yes                          | yes                        | yes          |
| Mi Home BLE+Mesh   | yes                          | yes                        | no           |
| HomeKit for Zigbee | yes                          | no                         | yes          | 
| Beeper and Alarm   | yes                          | no                         | no           |
| Buggy hardware     | yes                          | no                         | no           |
| Zigbee range       | high                         | unknown                    | medium       |

**Comments**

- For the first Multimode the Chinese and Euro model are supported, but it is recommended to use the Chinese Cloud
  because of the supported subdevice list
- For the Aqara Hub only Chinese model supported because only it works with Mi Home Cloud
- First Multimode support HomeKit, but only for Zigbee devices
- Only first Multimode has Alarm function for Mi Home ecosystem and Beeper
- Only first Multimode has buggy hardware, you may have minor stability issues with Zigbee, BLE, Mesh devices and
  Gateway Wi-Fi connection

## Install

This binary embed into [Home Assistant](https://www.home-assistant.io/) custom
integration [Xiaomi Gateway 3](https://github.com/AlexxIT/XiaomiGateway3). Integration can automatically:

- get the gateway's token from the MiHome cloud
- open Telnet on gateway
- download and run latest `openmiio_agent` binary
- run the binary after gateway restarts

But you can download binary manually from [latest release](https://github.com/AlexxIT/openmiio_agent/releases/latest).

- **MIPS** for Xiaomi Multimode Gateway
- **ARM** for Xiaomi Multimode Gateway 2 and Aqara Hub E1

## Run

All agruments are optional:

`/data/openmiio_agent miio central mqtt cache z3 --zigbee.tcp=8888 --log.level=trace`

- `miio` - enable miIO module for control all internal gateway communications instead of `miio_agent`
- `central` - enable central module for catch BLE/Mesh local updates
- `mqtt` - enable MQTT module and run gateways MQTT on public `1883` port
- `cache` - enable cache module for process BLE sensors without Integrnet
- `z3` - enable publish Z3GatewayHost stdout to MQTT (for reading zigbee stats)
- `--log.level=trace` - change log level, default `warn`
- `--zigbee.tcp=8888` - enable ser2net feature for zigbee chip

## miIO

- These are the same commands used in [miio proto](https://github.com/rytilahti/python-miio)
- You can add optional `id` key
- Some methods require empty `params`
- Some methods don't work (ex `miIO.info`)

```
mosquitto_sub -t miio/command_ack
mosquitto_pub -t miio/command -m '{"method":"get_common_lib_version","params":[]}'
```

## ZHA and zigbee2mqtt

Support on-demand access to zigbee chip via TCP for [ZHA](https://www.home-assistant.io/integrations/zha/) and [zigbee2mqtt](https://www.zigbee2mqtt.io/) projects.

By default, the standard gateway software will work with the zigbee chip. At the first connection to the TCP port the standard software will be stopped.

**Important:** Zigbee devices can't work simultaniously with MiHome and ZHA/zigbee2mqtt.

**All of your thanks for supporting the EZSP in zigbee2mqtt can say to [@kirovilya](https://github.com/kirovilya).**

| Feature                                 | ZHA       | zigbee2mqtt                                                             |
|-----------------------------------------|-----------|-------------------------------------------------------------------------|
| Support EFR32 EZSP (gateway's chip)     | excellent | [experimental](https://www.zigbee2mqtt.io/guide/adapters/#experimental) |
| Support EZSPv7 (original chip firmware) | excellent | [experimental](https://github.com/Koenkk/zigbee-herdsman/pull/598)      |
| Support EZSPv8 (custom chip firmware)   | excellent | [experimental](https://github.com/Koenkk/zigbee-herdsman/issues/319)    |
| Keep MiHome network settings            | yes       | no                                                                      |

When using ZHA, you can switch from MiHome mode to ZHA mode at any time. You won't need to repair your devices. But may need additional reconfiguration.

When you return from ZHA to MiHome mode, your old MiHome devices will continue to work. But, new ZHA devices will not appear.

zigbee2mqtt will replace the chip settings with its own. You will lose all your paired devices.

When returning from zigbee2mqtt mode to MiHome mode, you need to reset the gateway to factory settings.

## Lua

With lua scripts you can:

- Read all miIO requests and responses between gateway apps and cloud
- Change or prevent this requests and responses
- Make your own miIO commands, like `cli` command in example below
- Subscribe and publish to MQTT
- Read and write files
- Execute any bash scripts

Important

- If you write a function, it will be processed
- The error handler are disabled, if there is an error in the script - the whole application will crash
- Learn lua [here](https://programming-idioms.org/cheatsheet/Python/Lua) and [here](https://www.lua.org/manual/5.1/)

**Code**

- `function miio_request(from, method, req)`
    - `from` -
      int, [app ID](https://github.com/AlexxIT/openmiio_agent/blob/1eadf485bfff62520887b0767fd26a936d6760f0/internal/miio/miio.go#L11-L19)
    - `method` - string, miIO method
    - `req` - string, raw JSON request (use `json.decode(req)` for parsing)
    - `return` nothing - no change to the request
    - `return` string - replace request
    - `return nil` - prevent request
- `function miio_response(to, method, req, res)`
    - `to` - int, app ID (the original source of the request)
    - `method` - string, miIO method from request (not response)
    - `req` - string, raw JSON request
    - `res` - string, raw JSON response
    - `return` nothing - no change to the response
    - `return` string - replace response
    - `return nil` - prevent response
- `function mosquitto_sub(topic, payload)`
- `miio_send(to, msg)` - raw JSON to app ID
- `mosquitto_pub(topic, payload, retain)`

Place file `openmiio_agent.lua` next to the binary:

```lua
json = require("json")

function miio_request(from, method, req)
    -- prevent beeper for Motion Sensor 5 sec hack
    if from == 4 and method == "local.status" and req:find("dev_query_connect") then
        return nil
    end

    if from <= 0 then
        if method == "cli" then
            req = json.decode(req)
            os.execute("sh -c '" .. req.params[1] .. "'")
            local res = { id = req.id, result = { "ok" } }
            miio_send(from, json.encode(res))
            return nil -- prevent request to local
        end
    end
end

function miio_response(to, method, req, res)
    if to == 4 then
        if method == "_sync.zigbee3_bind" then
            req = json.decode(req)
            res = { id = req.id, result = { code = 0, message = "ok" } }
            return json.encode(res)
        end
    end
end
```

## MQTT

| Topic                     | App            | Mode      | Description                              |
|---------------------------|----------------|-----------|------------------------------------------|
| `gw/IEEE/commands`        | Z3GatewayHost  | subscribe | commands to zigbee stack (Silabs format) |
| `gw/IEEE/executed`        | Z3GatewayHost  | publish   | executed commands                        |
| `gw/IEEE/heartbeat`       | Z3GatewayHost  | publish   | zigbee network alive messages (1 min)    |
| `gw/IEEE/MessageReceived` | Z3GatewayHost  | publish   | raw messages from zigbee stack           |
| `miio/command`            | openmiio_agent | subscribe | commands to gateway (miIO format)        |
| `miio/command_ack`        | openmiio_agent | publish   | response on commands                     |
| `miio/report`             | openmiio_agent | publish   | updates from gateway to cloud            |
| `miio/report_ack`         | openmiio_agent | publish   | response from cloud to gateway           |
| `central/report`          | openmiio_agent | publish   | updates from bluetooth to central app    |
| `openmiio/report`         | openmiio_agent | publish   | openmiio_agent alive messages (30 sec)   |
| `broker/ping`             | zigbee_agent   | publish   | zigbee_agent alive message               |
| `zigbee/recv`             | zigbee_agent   | subscribe | commands to zigbee stack (Lumi format)   |
| `zigbee/send`             | zigbee_agent   | publish   | response from zigbee stack               | 

**openmiio/report**

```json5
{
  "gateway": {
    "model": "lumi.gateway.mgl03",
    "firmware": "1.5.4_0090",
  },
//  "miio": {
//    "cloud_starts": 123,
//    "cloud_uptime": "10s"
//  },
  "openmiio": {
    "version": "1.1.1",
    "uptime": "10s"
  },
  "serial": {
    "bluetooth_rx": 12345,
    "bluetooth_tx": 12345,
    "bluetooth_oe": 123,
    "zigbee_rx": 12345,
    "zigbee_tx": 12345,
    "zigbee_oe": 123
  },
  "zigbee": {
    "tcp_remote": "192.168.1.123",
    "tcp_starts": 123,
    "tcp_uptime": "10s",
    "z3_starts": 123,
    "z3_uptime": "10s"
  }
}
```