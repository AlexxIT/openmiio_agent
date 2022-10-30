# openmiio_agent

This project allows you to significantly extend the functionality of your gateways on the original firmware, keeping almost all default functionality of the device in the Xiaomi Mi Home ecosystem.

Supported gateway  | Xiaomi Multimode Gateway | Xiaomi Multimode Gateway 2 | Aqara Hub E1
-------------------|--------------------------|----------------------------|-------------
Supported models   | ZNDMWG03LM ZNDMWG02LM    | DMWG03LM                   | ZHWG16LM
Mi Home Zigbee     | yes                      | yes                        | yes
Mi Home BLE+Mesh   | yes                      | yes                        | no
HomeKit for Zigbee | yes                      | no                         | yes
Beeper and Alarm   | yes                      | no                         | no
Buggy hardware     | yes                      | no                         | no
Zigbee range       | high                     | unknown                    | medium

**Comments**

- For the first Multimode the Chinese and Euro model are supported, but it is recommended to use the Chinese Cloud because of the supported subdevice list
- For the Aqara Hub only Chinese model supported because only it works with Mi Home Cloud
- First Multimode support HomeKit, but only for Zigbee devices
- Only first Multimode has Alarm function for Mi Home ecosystem and Beeper
- Only first Multimode has buggy hardware, you may have minor stability issues with Zigbee, BLE, Mesh devices and Gateway Wi-Fi connection

**Features**

- Full support original Xiaomi firmware
- Access to gateway MQTT
- miIO to MQTT for send commands and receive device updates (also without Internet)
- ZHA and zigbee2mqtt support (on-demand mode)
- LUA scripts for changing default gateway logic
- BLE events without Internet
- Fix difference for BLE specs
- Zigbee custom firmware support

## MQTT

Topic | App | Mode | Description
------|-----|------|------------
gw/IEEE/commands        | Z3GatewayHost  | subscribe | commands to zigbee stack (Silabs format)
gw/IEEE/executed        | Z3GatewayHost  | publish   | executed commands
gw/IEEE/heartbeat       | Z3GatewayHost  | publish   | zigbee network alive messages (1 min)
gw/IEEE/MessageReceived | Z3GatewayHost  | publish   | raw messages from zigbee stack
miio/command            | openmiio_agent | subscribe | commands to gateways (miIO format)
miio/command_ack        | openmiio_agent | publish   | response on commands
miio/report             | openmiio_agent | publish   | updates from gateway to cloud
miio/report_ack         | openmiio_agent | publish   | response from cloud to gateway
zigbee/recv             | zigbee_agent   | subscribe | commands to zigbee stack (Lumi format)
zigbee/send             | zigbee_agent   | publish   | response from zigbee stack 
