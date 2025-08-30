# Xarvis AI System (v0.1)

Xarvis is a **modular AI assistant system** written in Go.  
It integrates with **Thinker (research AI)** and is designed to behave like Jarvis:  
a single conversational brain that **listens, thinks, remembers, and acts**.  

Xarvis is not just reactive — it can **think in the background**, reflect on memory, propose new insights, and manage projects for its user.

---

## 🚀 How to Run

Xarvis is containerized with Docker Compose. A Makefile provides simple commands.

```sh
# Start core services (Go server + MQTT broker)
make up

# Add reverse proxy (Traefik, routes /v1/*)
make proxy

# Add local AI services (Ollama LLM + TEI embeddings)
make ai

# Add voice (Whisper STT + Piper TTS)
make voice

# Use Qdrant vector backend instead of TiDB
make qdrant

# Use local TiDB dev instance (Serverless preferred in prod)
make tidb

# View logs
make logs

# Stop everything
make down

# Clean volumes + remove orphans
make clean.
```

## 🧩 Services

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
- **Alternative backend:** Qdrant (pure vector DB).  
- **Embedding model:** `bge-base-en`.  
- **Tenancy:** strict isolation per user.

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

