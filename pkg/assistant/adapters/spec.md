# Xarvis LLM adapters spec

## Description
Support multimodel + routing. 
Each provider can integrate with different services using their auth structure,
request and response. 
Adapter layer wraps around providing n-1 for contract types.

## Components 
- Stream
- Complete (stream/single)
- ToolCall
- Response json. (To be supported later)

## Breakdown
Takes in `AssistantInput` and returns `AssistantOutputStream`.
Stream can be actual stream or all response at once. 

