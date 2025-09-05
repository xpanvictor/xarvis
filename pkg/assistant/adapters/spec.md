# Xarvis LLM adapters spec

## Description
Support multimodel + routing. 
Each provider can integrate with different services using their auth structure,
request and response. 
Adapter layer wraps around providing n-1 for contract types.

## Components 
- Stream (output)
- Complete (stream/single)
- ToolCall
- Response json. (To be supported later)

## Breakdown
Takes in `AssistantInput` and returns `AssistantOutputStream`.
Stream can be actual stream or all response at once. 
while
	ctx not done & buffer not full & !bufferTimeout
	buffer stream else insert into stream
	but controlled (little 100ms rebounce)
- Model should be time conscious.

Adapter handles tool structure
- Start streaming.
- Detect tool_call:
- Buffer tool_call:
- Send as single stream to writer directly
- See you next time protocol


