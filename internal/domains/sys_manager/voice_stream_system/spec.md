
# Voice Streaming System
Continuously listens to user input tokens till relevance is detected at which it's passed as `interrupt` to
pipeline as input.

## Structure
This system can be resource intensive and requires the least but efficient approach.
System is always in either `passive` or `active` listening mode. For now, I'll be moving with this approach:
1. **Audio Ring Buffer**: This is a configurable size ring buffer to keep audio input and manage consistent memory usage.
2. **Voice Activity Detection**: On system module to filter processing noise/silence. On activity detection, reads audio buffer till silence detected > `X`. Sends compiled audio to tmp audio file/buffer.
3. **STT**: Speech to text module feeds off all tmp audio buffer and generates transcript. Transcript also includes timing segmentation. 
4. **Trigger manager**: Consumes generated TTS to decide relevance and if audio is trigger and activates listening. Keeps generated buffer as part of conversation.
5. **Audio/Command handler**: Controls trigger manager and stt to gather all transcript and send `INTR` to processing pipeline as input. If system is already in active mode, the trigger manager is replaced by a less strict version that just checks if bunch of silence else, triggers another INTR.

Essentially, on `active` listening mode, only silence > `Y` determines `INTR`.

Now, the voice system also receives commands like 
1. *AUD_PROC_DONE*
2. *NEED_MORE_CTX*
3. *STOP_LISTENING*
4. *RESM_LISTENING*

but only outputs
1. *INTR* with transcription content.

### Initial system arch:
<img src="/docs/assets/stt_v1.jpeg" alt="Initial diagram" title="Initial arch" style="transform: rotate(90deg);" />

Follows thread management here:
<img src="/docs/assets/thread_mngm.jpeg" alt="Thread diagram" title="Thread mngm" style="transform: rotate(90deg);" />