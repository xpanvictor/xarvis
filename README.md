#### Server: Go
#### Services:
1. Conversation management: 
	1. Take in text messages and generate appropriate response. 
	2. Handle message storage and chained system for context. 
	3. Message tree management (concept relationship) and tags.
2. Audio management (TTS & STT): 
	1. Take in recorded audio bin. 
	2. Speech to text conversion. 
	3. Text windowing and context derivation.
	4. Trigger system.
	5. Text to speech conversion with settings. 
3. User management:
	1. User settings and auth. 
	2. Tenancy system. 
4. Memory structure:
	1. Cross conversational RAG. 
	2. Context map within tenancy. 
	3. Managing memory:
		1. Creating and Search.
		2. General memory thread management. 
		3. Deleting or gradient relevance management. 
5. Project/Task management.
	1. Creating project context.
	2. Managing project flow.
	3. In system threads for cross thoughts and development.
6. Network protocol.
	1. Hybrid communication:
		1. Data plane (audio streaming).
		2. Control plane.

#### Architecture
##### Network: 
1. Data plane: websocket. 
2. Control plane: MQTT. 
##### Memory:
1. RAG: Qdrant + a small embedding layer.
2. Single tenancy ID. 
##### Project:
1. Threaded task runner (system wise).
2. Task system based on user action. 
##### Audio:
1. STT:  OpenAI whisper.cpp (or faster-whisper).
2. TTS: Coqui or Piper. 
###### Conversation:
1. Embedding: bge-base-en. 
2. Intelligence: 
	1. Hybrid approach:
		1. Ollama with Mistral self host.
		2. OpenAI gpt4 deep logic.
		3. Late binding thinking service.