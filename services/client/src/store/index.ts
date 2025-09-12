import { create } from 'zustand';
import { UserResponse, Conversation, Memory, Message } from '../services/api';

// Auth Store
interface AuthState {
    user: UserResponse | null;
    isAuthenticated: boolean;
    isLoading: boolean;
    error: string | null;
    setUser: (user: UserResponse | null) => void;
    setLoading: (loading: boolean) => void;
    setError: (error: string | null) => void;
    logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
    user: null,
    isAuthenticated: false,
    isLoading: false,
    error: null,
    setUser: (user) => set({ user, isAuthenticated: !!user }),
    setLoading: (isLoading) => set({ isLoading }),
    setError: (error) => set({ error }),
    logout: () => set({ user: null, isAuthenticated: false }),
}));

// Conversation Store
interface ConversationState {
    conversation: Conversation | null;
    memories: Memory[];
    recentMessages: Message[];
    isLoading: boolean;
    error: string | null;
    setConversation: (conversation: Conversation) => void;
    addMemory: (memory: Memory) => void;
    addMessage: (message: Message) => void;
    setLoading: (loading: boolean) => void;
    setError: (error: string | null) => void;
}

export const useConversationStore = create<ConversationState>((set, get) => ({
    conversation: null,
    memories: [],
    recentMessages: [],
    isLoading: false,
    error: null,
    setConversation: (conversation) => set({
        conversation,
        memories: conversation.memories || [],
        recentMessages: conversation.messages?.slice(-10) || [] // Keep last 10 messages
    }),
    addMemory: (memory) => set((state) => ({
        memories: [...state.memories, memory]
    })),
    addMessage: (message) => set((state) => {
        const updatedMessages = [...state.recentMessages, message];
        // Keep only last 10 messages
        return {
            recentMessages: updatedMessages.slice(-10)
        };
    }),
    setLoading: (isLoading) => set({ isLoading }),
    setError: (error) => set({ error }),
}));

// UI Store for general UI state
interface UIState {
    sidebarOpen: boolean;
    currentPage: string;
    theme: 'light' | 'dark';
    setSidebarOpen: (open: boolean) => void;
    setCurrentPage: (page: string) => void;
    setTheme: (theme: 'light' | 'dark') => void;
}

export const useUIStore = create<UIState>((set) => ({
    sidebarOpen: true,
    currentPage: 'conversation',
    theme: 'dark',
    setSidebarOpen: (sidebarOpen) => set({ sidebarOpen }),
    setCurrentPage: (currentPage) => set({ currentPage }),
    setTheme: (theme) => set({ theme }),
}));
