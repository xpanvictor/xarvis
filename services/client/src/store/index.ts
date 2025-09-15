import { create } from 'zustand';
import { UserResponse, Conversation, Memory, Message, ProjectResponse, NoteResponse, ListProjectsResponse, ListNotesResponse, TaskResponse, ListTasksResponse } from '../services/api';
import { ConnectionState, ListeningState } from '../services/websocket';

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

// Conversation Store - Enhanced for WebSocket
interface ConversationState {
    conversation: Conversation | null;
    memories: Memory[];
    recentMessages: Message[];
    isLoading: boolean;
    error: string | null;

    // WebSocket state
    connectionState: ConnectionState;
    listeningState: ListeningState;
    isConnected: boolean;
    isMuted: boolean;
    audioLevel: number;
    isSessionProcessing: boolean; // Prevent new responses until session completes

    // Streaming state
    streamingMessage: {
        content: string;
        isStreaming: boolean;
        audioBlob: Blob | null;
        pcmAudioData: ArrayBuffer | null;
    } | null;

    // Actions
    setConversation: (conversation: Conversation) => void;
    addMemory: (memory: Memory) => void;
    addMessage: (message: Message) => void;
    updateMessage: (messageId: string, updates: Partial<Message>) => void;
    setLoading: (loading: boolean) => void;
    setError: (error: string | null) => void;

    // WebSocket actions
    setConnectionState: (state: ConnectionState) => void;
    setListeningState: (state: ListeningState) => void;
    setMuted: (muted: boolean) => void;
    setAudioLevel: (level: number) => void;
    setSessionProcessing: (processing: boolean) => void;

    // Streaming actions
    startStreamingMessage: () => void;
    updateStreamingContent: (content: string) => void;
    completeStreamingMessage: () => void;
    setStreamingAudio: (audioBlob: Blob) => void;
    setStreamingPCMAudio: (pcmData: ArrayBuffer) => void;
    clearStreamingMessage: () => void;

    // Real-time message handling
    addRealtimeMessage: (content: string, role: Message['msg_role'], tags?: string[]) => void;
    clearMessages: () => void;
}

export const useConversationStore = create<ConversationState>((set, get) => ({
    conversation: null,
    memories: [],
    recentMessages: [],
    isLoading: false,
    error: null,

    // WebSocket state
    connectionState: 'disconnected',
    listeningState: 'idle',
    isConnected: false,
    isMuted: false,
    audioLevel: 0,
    isSessionProcessing: false,

    // Streaming state
    streamingMessage: null,

    // Basic actions
    setConversation: (conversation) => set({
        conversation,
        memories: conversation.memories || [],
        recentMessages: (conversation.messages || [])
            .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())
            .slice(-20) // Keep last 20 messages sorted by timestamp
    }),

    addMemory: (memory) => set((state) => ({
        memories: [...state.memories, memory]
    })),

    addMessage: (message) => set((state) => {
        const existingIndex = state.recentMessages.findIndex(m => m.id === message.id);
        let updatedMessages;

        if (existingIndex >= 0) {
            // Update existing message
            updatedMessages = [...state.recentMessages];
            updatedMessages[existingIndex] = message;
        } else {
            // Add new message
            updatedMessages = [...state.recentMessages, message];
        }

        // Sort by timestamp and keep last 20
        updatedMessages.sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());

        return {
            recentMessages: updatedMessages.slice(-20)
        };
    }),

    updateMessage: (messageId, updates) => set((state) => ({
        recentMessages: state.recentMessages.map(msg =>
            msg.id === messageId ? { ...msg, ...updates } : msg
        )
    })),

    setLoading: (isLoading) => set({ isLoading }),
    setError: (error) => set({ error }),

    // WebSocket actions
    setConnectionState: (connectionState) => set({
        connectionState,
        isConnected: connectionState === 'connected'
    }),

    setListeningState: (listeningState) => set({ listeningState }),

    setMuted: (isMuted) => set({ isMuted }),

    setAudioLevel: (audioLevel) => set({ audioLevel }),

    setSessionProcessing: (isSessionProcessing) => set({ isSessionProcessing }),

    // Streaming actions
    startStreamingMessage: () => set({
        streamingMessage: {
            content: '',
            isStreaming: true,
            audioBlob: null,
            pcmAudioData: null
        }
    }),

    updateStreamingContent: (content) => set((state) => {
        if (state.streamingMessage) {
            // Replace content with new content (Dashboard handles cumulative building)
            return {
                streamingMessage: {
                    ...state.streamingMessage,
                    content: content
                }
            };
        } else {
            // Start new streaming message
            return {
                streamingMessage: {
                    content,
                    isStreaming: true,
                    audioBlob: null,
                    pcmAudioData: null
                }
            };
        }
    }),

    completeStreamingMessage: () => set((state) => {
        if (state.streamingMessage) {
            // Add the completed streaming message to recent messages
            const completedMessage: Message = {
                id: `stream_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
                conversation_id: state.conversation?.id || '',
                user_id: state.conversation?.owner_id || '',
                text: state.streamingMessage.content,
                msg_role: 'assistant',
                timestamp: new Date().toISOString(),
                tags: ['completed'] // Remove 'streaming' tag so cursor doesn't show
            };

            // Add to recent messages
            const updatedMessages = [...state.recentMessages, completedMessage];
            updatedMessages.sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());

            return {
                recentMessages: updatedMessages.slice(-20),
                streamingMessage: null // Clear the streaming message completely
            };
        }
        return state;
    }),

    setStreamingAudio: (audioBlob) => set((state) => ({
        streamingMessage: state.streamingMessage ? {
            ...state.streamingMessage,
            audioBlob
        } : {
            content: '',
            isStreaming: true,
            audioBlob,
            pcmAudioData: null
        }
    })),

    setStreamingPCMAudio: (pcmData) => set((state) => ({
        streamingMessage: state.streamingMessage ? {
            ...state.streamingMessage,
            pcmAudioData: pcmData
        } : {
            content: '',
            isStreaming: true,
            audioBlob: null,
            pcmAudioData: pcmData
        }
    })),

    clearStreamingMessage: () => set({ streamingMessage: null }),

    // Real-time message handling
    addRealtimeMessage: (content, role, tags = []) => {
        const message: Message = {
            id: `temp_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            conversation_id: get().conversation?.id || '',
            user_id: get().conversation?.owner_id || '',
            text: content,
            msg_role: role,
            timestamp: new Date().toISOString(),
            tags: [...tags, 'realtime']
        };

        get().addMessage(message);
    },

    clearMessages: () => set({ recentMessages: [] })
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

// Projects Store
interface ProjectsState {
    projects: ProjectResponse[];
    currentProject: ProjectResponse | null;
    isLoading: boolean;
    error: string | null;
    pagination: {
        total: number;
        offset: number;
        limit: number;
    } | null;

    // Actions
    setProjects: (projects: ProjectResponse[], pagination?: { total: number; offset: number; limit: number }) => void;
    addProject: (project: ProjectResponse) => void;
    updateProject: (projectId: string, updates: Partial<ProjectResponse>) => void;
    removeProject: (projectId: string) => void;
    setCurrentProject: (project: ProjectResponse | null) => void;
    setLoading: (loading: boolean) => void;
    setError: (error: string | null) => void;
    clearProjects: () => void;
}

export const useProjectsStore = create<ProjectsState>((set, get) => ({
    projects: [],
    currentProject: null,
    isLoading: false,
    error: null,
    pagination: null,

    setProjects: (projects, pagination) => set({
        projects,
        pagination: pagination || null
    }),

    addProject: (project) => set((state) => ({
        projects: [project, ...state.projects]
    })),

    updateProject: (projectId, updates) => set((state) => ({
        projects: state.projects.map(p => p.id === projectId ? { ...p, ...updates } : p),
        currentProject: state.currentProject?.id === projectId ? { ...state.currentProject, ...updates } : state.currentProject
    })),

    removeProject: (projectId) => set((state) => ({
        projects: state.projects.filter(p => p.id !== projectId),
        currentProject: state.currentProject?.id === projectId ? null : state.currentProject
    })),

    setCurrentProject: (currentProject) => set({ currentProject }),

    setLoading: (isLoading) => set({ isLoading }),

    setError: (error) => set({ error }),

    clearProjects: () => set({
        projects: [],
        currentProject: null,
        pagination: null
    })
}));

// Notes Store
interface NotesState {
    notes: NoteResponse[];
    currentNote: NoteResponse | null;
    isLoading: boolean;
    error: string | null;
    pagination: {
        total: number;
        offset: number;
        limit: number;
    } | null;

    // Actions
    setNotes: (notes: NoteResponse[], pagination?: { total: number; offset: number; limit: number }) => void;
    addNote: (note: NoteResponse) => void;
    updateNote: (noteId: string, updates: Partial<NoteResponse>) => void;
    removeNote: (noteId: string) => void;
    setCurrentNote: (note: NoteResponse | null) => void;
    setLoading: (loading: boolean) => void;
    setError: (error: string | null) => void;
    clearNotes: () => void;
}

export const useNotesStore = create<NotesState>((set, get) => ({
    notes: [],
    currentNote: null,
    isLoading: false,
    error: null,
    pagination: null,

    setNotes: (notes, pagination) => set({
        notes,
        pagination: pagination || null
    }),

    addNote: (note) => set((state) => ({
        notes: [note, ...state.notes]
    })),

    updateNote: (noteId, updates) => set((state) => ({
        notes: state.notes.map(n => n.id === noteId ? { ...n, ...updates } : n),
        currentNote: state.currentNote?.id === noteId ? { ...state.currentNote, ...updates } : state.currentNote
    })),

    removeNote: (noteId) => set((state) => ({
        notes: state.notes.filter(n => n.id !== noteId),
        currentNote: state.currentNote?.id === noteId ? null : state.currentNote
    })),

    setCurrentNote: (currentNote) => set({ currentNote }),

    setLoading: (isLoading) => set({ isLoading }),

    setError: (error) => set({ error }),

    clearNotes: () => set({
        notes: [],
        currentNote: null,
        pagination: null
    })
}));

// Tasks Store
interface TasksState {
    tasks: TaskResponse[];
    currentTask: TaskResponse | null;
    isLoading: boolean;
    error: string | null;
    pagination: {
        total: number;
        offset: number;
        limit: number;
    } | null;

    // Actions
    setTasks: (tasks: TaskResponse[], pagination?: { total: number; offset: number; limit: number }) => void;
    addTask: (task: TaskResponse) => void;
    updateTask: (taskId: string, updates: Partial<TaskResponse>) => void;
    removeTask: (taskId: string) => void;
    setCurrentTask: (task: TaskResponse | null) => void;
    setLoading: (loading: boolean) => void;
    setError: (error: string | null) => void;
    clearTasks: () => void;
}

export const useTasksStore = create<TasksState>((set, get) => ({
    tasks: [],
    currentTask: null,
    isLoading: false,
    error: null,
    pagination: null,

    setTasks: (tasks, pagination) => set({
        tasks,
        pagination: pagination || null
    }),

    addTask: (task) => set((state) => ({
        tasks: [task, ...state.tasks]
    })),

    updateTask: (taskId, updates) => set((state) => ({
        tasks: state.tasks.map(t => t.id === taskId ? { ...t, ...updates } : t),
        currentTask: state.currentTask?.id === taskId ? { ...state.currentTask, ...updates } : state.currentTask
    })),

    removeTask: (taskId) => set((state) => ({
        tasks: state.tasks.filter(t => t.id !== taskId),
        currentTask: state.currentTask?.id === taskId ? null : state.currentTask
    })),

    setCurrentTask: (currentTask) => set({ currentTask }),

    setLoading: (isLoading) => set({ isLoading }),

    setError: (error) => set({ error }),

    clearTasks: () => set({
        tasks: [],
        currentTask: null,
        pagination: null
    })
}));
