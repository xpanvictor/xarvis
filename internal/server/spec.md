# Routing
- Domain interaction: Pure HTTP/REST. 
- Streaming: WS and MQTT.

## Audio I/O streaming
Users register their device for I/O streaming.
However, each device attaches to endpoints for individual flow.
#### Endpoints breakdown:
1. TextSink: This endpoint gets all text stream out. By configuration, all
    text sink endpoints get all text streams so you see output on all connected devices.
    This also supports `TextCollect` type.
2. AudioSink: This endpoint gets all audio streams. By configuration, most recently used
    (MRU) algorithm is used to select an endpoint to get this audio (across all devices).
3. AudioCollect: This endpoint is also (but can be separate) from the audio sink device
    especially for users using different devices for read and write. 
    If in same device with an `AudioSink` endpoint, that endpoint becomes MRU if audio collect 
    triggers. 

Same endpoint can be all types. However, new `ws|mqtt` connection is established per endpoint and 
MAX 1 endpoint for `AudioCollect`. Trying to connect another disconnects previous and hence
remaps the MRU endpoint. 
