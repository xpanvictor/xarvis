import React, { useEffect, useMemo, useRef } from 'react';
import { useConversationStore } from '../../store';
import './ChatThread.css';

export const ChatThread: React.FC = () => {
  const { recentMessages, streamingMessage } = useConversationStore();
  const listRef = useRef<HTMLDivElement>(null);

  // Compose displayed items: recent messages + optional streaming message at end
  const items = useMemo(() => {
    const base = [...recentMessages];
    // Only show the streaming bubble while streaming; once completed,
    // the finalized message lives in recentMessages.
    if (streamingMessage && streamingMessage.isStreaming) {
      base.push({
        id: `streaming_${Date.now()}`,
        conversation_id: '',
        user_id: '',
        text: streamingMessage.content,
        msg_role: 'assistant' as const,
        timestamp: new Date().toISOString(),
        tags: ['streaming']
      });
    }
    return base;
  }, [recentMessages, streamingMessage]);

  // Auto-scroll to bottom on updates
  useEffect(() => {
      const el = listRef.current;
      if (el) {
        el.scrollTop = el.scrollHeight;
      }
  }, [items.length, streamingMessage?.content]);

  return (
    <div className="chat-thread" ref={listRef}>
      {items.map((m, idx) => (
        <div key={`${m.id}_${idx}`} className={`chat-bubble ${m.msg_role}`}>
          <div className="role">{m.msg_role}</div>
          <div className="content">
            {m.text}
            {m.tags?.includes('streaming') && (
              <span className="cursor">|</span>
            )}
          </div>
        </div>
      ))}
    </div>
  );
};

export default ChatThread;
