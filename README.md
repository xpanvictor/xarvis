# Xarvis AI System (v0.1)

Xarvis is a **modular AI assistant system** written in Go.  
It integrates as a **thinker** and is designed to behave like Jarvis:  
a single conversational brain that **listens, thinks, remembers, and acts**.  

Xarvis is not just reactive — it can **think in the background**, reflect on memory, propose new insights, and manage projects for the user.

This is essentially the system for the entire suite and different
clients will be written to integrate and connect with it. 
Allows connecting to multiple endpoints with which audio/text
can be streamed (in or out (or both)). 
This way, users can have the `hardware: esp32 or custom chip` 
wearable like handband which can connect with earpods, speaker
to provide the assistant system. 

---

## How to Run

Xarvis is containerized with Docker Compose. You can spin up
a working version using make. However, you'll need the env variables. 

```sh
# prod:
make prod
```

## Few functions:
1. Task management with `Asynq` to schedule tasks which are
at execution time, passes back into the LLM to decide way forward
and streams output response (if any) to the user's hardware through
their endpoints. 
2. Web search using `Tavily` to add more context with web.
3. Project and notes management. This allows user to create
projects, manage logs, think with LLM (like Tony Stark working)
with Jarvis to solve complex projects.
4. Memory using `TIDB cloud` and `Gemini` embedder to allow storing
contextual details and good vector search.
5. Tool management system that unifies most functionalities and
can be *filtered* to reduce compute cost.
6. Text to speech using `tts-piper` with some configurable options.
Will be further developed later.
7. Speech to text for having conversation with `whisper` with a 
trigger system by saying `xarvis` or `assistant`. Will support
contextual trigger later (like noting important info and registering) for you.
8. Tool call security to ensure users don't access others. This 
is implemented by ensuring system passes user context to every tool
call. Will later implement a more robust `chroot like` system, thanks to learning about Operating systems.

**There are also a few other docs like in internal/domain/voicestream.../spec.md, ...**

## Notes for now:
The system is still fragile and breaks for a few things like 
- scheduled tasks may be completed (check tasks page) but when llm
is reporting, says can't find the task. 
- system checking notes for memories (will work on system prompts).
- not getting responses at times, reload and try again.
- messages disappearing after a while (not a bug actually). The
plan is to have a job that spins up periodically per user and 
summarizes all messages into memory (if necessary) then deletes 
the messages. This is to support a more `retentive memory` logic. 
Just how humans regurgitate and store important info in long term 
memory.
- some other issues as well, if system goes off at any point, for
now you can reach out via `xpanvictor@gmail.com`, will set up
properly soon.

There are a bunch of things not mentioned or even implemented yet,
it's going to be a long term project.   
Thank you.

## Services

### 1. Conversation management
Handles **all user interactions**:
- Accepts text or audio input, produces contextual responses.
- Maintains **a single conversation per user** (like talking to one assistant, not separate chats).
- Stores conversation history & tags for context recovery.
- Builds **message trees** to represent concepts and relationships across turns.

---

### 2. Audio management (STT & TTS)
Enables voice-based interaction:
- Accepts raw audio from ESP32 mic client.
- **Speech-to-text** via Whisper/Faster-Whisper.
- Splits speech into windows for accurate transcription.
- Trigger system: detect wake events, commands, or push-to-talk.
- **Text-to-speech** output via Piper/Coqui with configurable voices.
- Playback to server-paired Bluetooth speaker.

---

### 3. User management
Handles **users, tenancy, and settings**:
- User settings & authentication.
- Persona traits (style, goals, behavior).
- Tenancy system (single tenant ID per user).
- Personalization across projects, memory, and approvals.

---

### 4. Memory structure
Provides **long-term memory & context**:
- Cross-conversational **RAG (Retrieval-Augmented Generation)**.
- Context maps within each tenancy.
- Memory management includes:
  - **Creation & search** (embedding-based recall).
  - **Thread management** (linking related memories).
  - **Decay & deletion** of stale items (gradient relevance).

---

### 5. Project & Task management
Supports structured work:
- Create **project contexts** (with tasks, approvals, requests).
- Manage project & task lifecycle (planned → in progress → done).
- System threads for **cross-thought development** and reflection.
- Tie insights & requests to project timelines.

---

### 6. Network protocol
Hybrid design separating **data** and **control**:
- **Data plane:** WebSocket (real-time streaming of audio & responses).
- **Control plane:** MQTT (ESP devices, triggers, approval signals, events).

---

## 🏗️ Architecture

### Network
- **Data plane:** WebSocket — continuous streaming of audio, responses, and embeddings.  
- **Control plane:** MQTT — lightweight signaling (triggers, approvals, device control).

### Memory
- **Default vector backend:** TiDB vector engine (scalable, hybrid with SQL + full-text).  
- **Embedding model:** `bge-base-en` or `gemini-embedder`.  
- **Tenancy:** strict isolation per user. 
- **Implementation:** I have an embedder interface to support different embedders.

### Project system
- Threaded **task runner** that executes workflows inside the system.  
- Task system is driven by **user actions + background thinker reflection**.  

### Audio
- **STT:** Whisper.cpp / Faster-Whisper.  
- **TTS:** Coqui TTS or Piper.  
- Audio output routed to host (Bluetooth speaker).  

### Conversation
- **Embedding:** bge-base-en.  
- **Hybrid intelligence:**
  1. **Local LLMs:** Ollama (Mistral, LLaMA, etc.).  
  2. **Cloud LLMs:** OpenAI GPT-4 for deep reasoning.  
  3. **Late-binding thinker:** background reflection loop.  

---

## 🧠 Brain Decision System

Xarvis runs a **Brain Decision System (BDSM)** that governs autonomous thinking:

### Thinking loop
1. **Trigger:** system spin-up, new request, memory change, due task, or periodic timer.
2. **Assemble context:** fetch recent conversation, memory, projects, approvals.
3. **Reflect:** analyze what has changed since last cycle.
4. **Plan:** propose ≤5 next steps (tool calls, insights, actions).
5. **Gate risky steps:** require approvals if risk > threshold.
6. **Act safely:** execute allowed steps via tool executor.
7. **Update memory:** add insights, prune stale facts, adjust salience.
8. **Outreach:** draft message to user if valuable insight is found.
9. **Cooldown:** stop after budgets (time/tokens/actions) are hit.

### Key properties
- **Singleton per user**: only one thinker loop active at a time.  
- **Bounded**: avoids infinite loops with strict budgets.  
- **Risk-aware**: requires approval for high-risk actions.  
- **Memory-first**: context is built from memory, not endless conversation history.  

---

## 📜 License

### Logs:
[Mon Sep 8, 25] I shouldn't stream mp3 slices :), it's really terrible to 
handle on client side. Later on, I'll figure out a better format (I've used
PCM on a shazam implementation I wrote before). Anyways, lesson learnt, 
I'll move on to system internals, domains, tool calls and then audio input. 
God bless.
[same day 30 mins later] I decided to just switch to pcm and it's fine actually. Although did the conversion with AI, it's stuttering a bit but ok.
I just couldn't sleep with the issue left.
