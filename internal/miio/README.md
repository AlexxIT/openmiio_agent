## Protocol

There are four members of communication:

1. miio_agent - miIO RPC protocol implementation
2. miio_client - miIO OT protocol implementation
3. cloud - Mi Cloud
4. app - local apps (Bluetooth, Zigbee, Gateway)

Communication types:

1. app to miio_agent
   - only `bind` and `register` methods:
     - `{"address":2,"method":"bind"}`
     - `{"method":"register","key":"get_common_lib_version"}`
   - without response
2. app to app
   - with key `_to`, support multicast mask:
     - `{"id":123,"_to":16,"method":"local.query_dev","params":{}}`
   - response is app to app message
3. app to miio_client
   - methods `local.` and `_internal.`:
     - `{"id":123,"method":"local.query_status","params":""}`
   - response is miio_client to app message:
     - `{"id":123,"method":"local.status","params":"cloud_connected"}`
4. app to cloud
   - `{"id":123,"method":"props","params":{"ble_mesh_switch":"enable"}}`
   - response is cloud to app message:
     - `{"id":123,"result":"ok"}`
6. cloud to app
   - response is app to cloud message
   - local miio proto is same