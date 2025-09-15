import axios, { AxiosResponse, InternalAxiosRequestConfig, AxiosError } from 'axios';

// API Configuration
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8088/api/v1';

const api = axios.create({
    baseURL: API_BASE_URL,
    headers: {
        'Content-Type': 'application/json',
    },
});

// Add auth token to requests
api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('accessToken');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

// Handle token refresh on 401
api.interceptors.response.use(
    (response: AxiosResponse) => response,
    async (error: AxiosError) => {
        const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

        if (error.response?.status === 401 && !originalRequest._retry) {
            originalRequest._retry = true;

            try {
                const refreshToken = localStorage.getItem('refreshToken');
                if (refreshToken) {
                    const response = await api.post('/auth/refresh', { refreshToken });
                    const { accessToken, expiresAt } = response.data.tokens;

                    localStorage.setItem('accessToken', accessToken);
                    localStorage.setItem('tokenExpiresAt', expiresAt);

                    if (originalRequest.headers) {
                        originalRequest.headers.Authorization = `Bearer ${accessToken}`;
                    }
                    return api(originalRequest);
                }
            } catch (refreshError) {
                // Refresh failed, redirect to login
                localStorage.removeItem('accessToken');
                localStorage.removeItem('refreshToken');
                localStorage.removeItem('tokenExpiresAt');
                window.location.href = '/login';
            }
        }

        return Promise.reject(error);
    }
);

// Types based on the Swagger documentation
export interface LoginRequest {
    email: string;
    password: string;
}

export interface RegisterRequest {
    email: string;
    password: string;
    displayName: string;
    timezone?: string;
    settings?: object;
}

export interface AuthTokens {
    accessToken: string;
    refreshToken: string;
    expiresAt: string;
}

export interface UserResponse {
    id: string;
    email: string;
    displayName: string;
    timezone: string;
    settings: object;
    createdAt: string;
    updatedAt: string;
}

export interface LoginResponse {
    message: string;
    user: UserResponse;
    tokens: AuthTokens;
}

export interface RegisterResponse {
    message: string;
    user: UserResponse;
}

export interface Memory {
    id: string;
    conversation_id: string;
    string: string;
    memory_type: 'episodic' | 'semantic';
    saliency_score: number;
    created_at: string;
    updated_at: string;
}

export interface Message {
    id: string;
    conversation_id: string;
    user_id: string;
    text: string;
    msg_role: 'user' | 'assistant' | 'system' | 'tool';
    timestamp: string;
    tags: string[];
}

export interface Conversation {
    id: string;
    owner_id: string;
    summary: string;
    created_at: string;
    updated_at: string;
    messages: Message[];
    memories: Memory[];
}

export interface CreateMemoryRequest {
    content: string;
    type: 'episodic' | 'semantic';
}

export interface CreateMessageRequest {
    text: string;
    timestamp: string;
}

export interface UpdateProfileRequest {
    displayName?: string;
    timezone?: string;
    settings?: object;
}

export interface ErrorResponse {
    error: string;
    details?: string;
}

// Project types
export type ProjectPriority = 'low' | 'med' | 'high' | 'urgent';

export type ProjectStatus = 'planned' | 'in_progress' | 'blocked' | 'done' | 'archived';

export interface ProgressEvent {
    at: string;
    by: string;
    kind: 'planned' | 'started' | 'blocked' | 'unblocked' | 'completed' | 'comment';
    memo: string;
}

export interface ProjectResponse {
    id: string;
    name: string;
    description: string;
    status: ProjectStatus;
    priority: ProjectPriority;
    tags: string[];
    progress: ProgressEvent[];
    createdAt: string;
    updatedAt: string;
    dueAt: string;
    userId: string;
}

export interface CreateProjectRequest {
    name: string;
    description?: string;
    status?: ProjectStatus;
    priority?: ProjectPriority;
    tags?: string[];
    dueAt?: string;
}

export interface UpdateProjectRequest {
    name?: string;
    description?: string;
    status?: ProjectStatus;
    priority?: ProjectPriority;
    tags?: string[];
    dueAt?: string;
}

export interface AddProgressEventRequest {
    by: string;
    kind: 'planned' | 'started' | 'blocked' | 'unblocked' | 'completed' | 'comment';
    memo: string;
}

// Note types
export interface NoteResponse {
    id: string;
    content: string;
    tags: string[];
    projectId?: string;
    userId: string;
    createdAt: string;
}

export interface CreateNoteRequest {
    content: string;
    tags?: string[];
    projectId?: string;
}

export interface UpdateNoteRequest {
    content?: string;
    tags?: string[];
    projectId?: string;
}

export interface PaginationInfo {
    total: number;
    offset: number;
    limit: number;
}

export interface ListNotesResponse {
    notes: NoteResponse[];
    pagination: PaginationInfo;
}

export interface ListProjectsResponse {
    projects: ProjectResponse[];
    pagination: PaginationInfo;
}

export interface SearchNotesResponse {
    notes: NoteResponse[];
    pagination: PaginationInfo;
    query: string;
    tags: string[];
}

// Task types
export type TaskStatus = 'pending' | 'cancelled' | 'done';

export type TaskPriority = 1 | 2 | 3 | 4 | 5;

export type RecurrenceType = 'none' | 'daily' | 'weekly' | 'monthly' | 'yearly' | 'custom';

export interface RecurrenceConfig {
    type: RecurrenceType;
    interval?: number;
    daysOfWeek?: number[];
    daysOfMonth?: number[];
    monthsOfYear?: number[];
    endDate?: string;
    maxOccurrences?: number;
    timeZone?: string;
}

export interface TaskResponse {
    id: string;
    userId: string;
    title: string;
    description: string;
    status: TaskStatus;
    priority: TaskPriority;
    tags: string[];
    scheduledAt?: string;
    dueAt?: string;
    completedAt?: string;
    cancelledAt?: string;
    isRecurring: boolean;
    recurrenceConfig?: RecurrenceConfig;
    parentTaskId?: string;
    nextExecution?: string;
    executionCount: number;
    metadata?: Record<string, any>;
    createdAt: string;
    updatedAt: string;
}

export interface CreateTaskRequest {
    title: string;
    description?: string;
    priority?: TaskPriority;
    tags?: string[];
    scheduledAt?: string;
    dueAt?: string;
    isRecurring?: boolean;
    recurrenceConfig?: RecurrenceConfig;
    metadata?: Record<string, any>;
}

export interface UpdateTaskRequest {
    title?: string;
    description?: string;
    priority?: TaskPriority;
    tags?: string[];
    scheduledAt?: string;
    dueAt?: string;
    isRecurring?: boolean;
    recurrenceConfig?: RecurrenceConfig;
    metadata?: Record<string, any>;
}

export interface UpdateTaskStatusRequest {
    status: TaskStatus;
}

export interface ListTasksRequest {
    status?: TaskStatus;
    priority?: TaskPriority;
    tags?: string[];
    search?: string;
    orderBy?: string;
    order?: 'asc' | 'desc';
    offset?: number;
    limit?: number;
}

export interface ListTasksResponse {
    tasks: TaskResponse[];
    pagination: PaginationInfo;
}

export interface CalendarTaskResponse {
    id: string;
    title: string;
    description: string;
    status: TaskStatus;
    priority: TaskPriority;
    dueAt?: string;
    scheduledAt?: string;
    tags: string[];
}

// Authentication API
export const authAPI = {
    login: async (credentials: LoginRequest): Promise<LoginResponse> => {
        const response: AxiosResponse<LoginResponse> = await api.post('/auth/login', credentials);
        return response.data;
    },

    register: async (userData: RegisterRequest): Promise<RegisterResponse> => {
        const response: AxiosResponse<RegisterResponse> = await api.post('/auth/register', userData);
        return response.data;
    },

    refreshToken: async (refreshToken: string): Promise<{ tokens: AuthTokens }> => {
        const response = await api.post('/auth/refresh', { refreshToken });
        return response.data;
    },

    logout: () => {
        localStorage.removeItem('accessToken');
        localStorage.removeItem('refreshToken');
        localStorage.removeItem('tokenExpiresAt');
    },
};

// User API
export const userAPI = {
    getProfile: async (): Promise<UserResponse> => {
        const response: AxiosResponse<{ user: UserResponse }> = await api.get('/user/profile');
        return response.data.user;
    },

    updateProfile: async (updates: UpdateProfileRequest): Promise<UserResponse> => {
        const response: AxiosResponse<{ user: UserResponse }> = await api.put('/user/profile', updates);
        return response.data.user;
    },

    deleteAccount: async (): Promise<{ message: string }> => {
        const response = await api.delete('/user/account');
        return response.data;
    },
};

// Conversation API
export const conversationAPI = {
    getConversation: async (): Promise<Conversation> => {
        const response: AxiosResponse<{ conversation: Conversation }> = await api.get('/conversation');
        return response.data.conversation;
    },

    sendMessage: async (message: CreateMessageRequest): Promise<Message> => {
        const response: AxiosResponse<{ message: Message }> = await api.post('/conversation/message', message);
        return response.data.message;
    },

    createMemory: async (memory: CreateMemoryRequest): Promise<Memory> => {
        const response: AxiosResponse<{ memory: Memory }> = await api.post('/conversation/memory', memory);
        return response.data.memory;
    },
};

// Projects API
export const projectsAPI = {
    listProjects: async (params?: {
        status?: ProjectStatus;
        priority?: ProjectPriority;
        tags?: string[];
        offset?: number;
        limit?: number;
    }): Promise<ListProjectsResponse> => {
        const queryParams = new URLSearchParams();
        if (params?.status) queryParams.append('status', params.status);
        if (params?.priority) queryParams.append('priority', params.priority);
        if (params?.tags) params.tags.forEach(tag => queryParams.append('tags', tag));
        if (params?.offset !== undefined) queryParams.append('offset', params.offset.toString());
        if (params?.limit !== undefined) queryParams.append('limit', params.limit.toString());

        const response = await api.get(`/projects?${queryParams.toString()}`);
        return response.data;
    },

    createProject: async (project: CreateProjectRequest): Promise<ProjectResponse> => {
        const response: AxiosResponse<{ project: ProjectResponse }> = await api.post('/projects', project);
        return response.data.project;
    },

    getProject: async (id: string): Promise<ProjectResponse> => {
        const response: AxiosResponse<{ project: ProjectResponse }> = await api.get(`/projects/${id}`);
        return response.data.project;
    },

    updateProject: async (id: string, updates: UpdateProjectRequest): Promise<ProjectResponse> => {
        const response: AxiosResponse<{ project: ProjectResponse }> = await api.put(`/projects/${id}`, updates);
        return response.data.project;
    },

    deleteProject: async (id: string): Promise<{ message: string }> => {
        const response = await api.delete(`/projects/${id}`);
        return response.data;
    },

    addProgressEvent: async (id: string, event: AddProgressEventRequest): Promise<ProjectResponse> => {
        const response: AxiosResponse<{ project: ProjectResponse }> = await api.post(`/projects/${id}/progress`, event);
        return response.data.project;
    },
};

// Notes API
export const notesAPI = {
    listNotes: async (params?: {
        search?: string;
        tags?: string[];
        orderBy?: string;
        order?: 'asc' | 'desc';
        offset?: number;
        limit?: number;
    }): Promise<ListNotesResponse> => {
        const queryParams = new URLSearchParams();
        if (params?.search) queryParams.append('search', params.search);
        if (params?.tags) params.tags.forEach(tag => queryParams.append('tags', tag));
        if (params?.orderBy) queryParams.append('orderBy', params.orderBy);
        if (params?.order) queryParams.append('order', params.order);
        if (params?.offset !== undefined) queryParams.append('offset', params.offset.toString());
        if (params?.limit !== undefined) queryParams.append('limit', params.limit.toString());

        const response = await api.get(`/notes?${queryParams.toString()}`);
        return response.data;
    },

    createNote: async (note: CreateNoteRequest): Promise<NoteResponse> => {
        const response: AxiosResponse<{ note: NoteResponse }> = await api.post('/notes', note);
        return response.data.note;
    },

    getNote: async (id: string): Promise<NoteResponse> => {
        const response: AxiosResponse<{ note: NoteResponse }> = await api.get(`/notes/${id}`);
        return response.data.note;
    },

    updateNote: async (id: string, updates: UpdateNoteRequest): Promise<NoteResponse> => {
        const response: AxiosResponse<{ note: NoteResponse }> = await api.put(`/notes/${id}`, updates);
        return response.data.note;
    },

    deleteNote: async (id: string): Promise<{ message: string }> => {
        const response = await api.delete(`/notes/${id}`);
        return response.data;
    },

    searchNotes: async (query: string, tags?: string[], offset?: number, limit?: number): Promise<SearchNotesResponse> => {
        const queryParams = new URLSearchParams();
        queryParams.append('q', query);
        if (tags) tags.forEach(tag => queryParams.append('tags', tag));
        if (offset !== undefined) queryParams.append('offset', offset.toString());
        if (limit !== undefined) queryParams.append('limit', limit.toString());

        const response = await api.get(`/notes/search?${queryParams.toString()}`);
        return response.data;
    },

    getNotesByTags: async (tags: string[], offset?: number, limit?: number): Promise<ListNotesResponse> => {
        const queryParams = new URLSearchParams();
        tags.forEach(tag => queryParams.append('tags', tag));
        if (offset !== undefined) queryParams.append('offset', offset.toString());
        if (limit !== undefined) queryParams.append('limit', limit.toString());

        const response = await api.get(`/notes/tags?${queryParams.toString()}`);
        return response.data;
    },
};

// Tasks API
export const tasksAPI = {
    listTasks: async (params?: ListTasksRequest): Promise<ListTasksResponse> => {
        const queryParams = new URLSearchParams();
        if (params?.status) queryParams.append('status', params.status);
        if (params?.priority) queryParams.append('priority', params.priority.toString());
        if (params?.tags) params.tags.forEach(tag => queryParams.append('tags', tag));
        if (params?.search) queryParams.append('search', params.search);
        if (params?.orderBy) queryParams.append('orderBy', params.orderBy);
        if (params?.order) queryParams.append('order', params.order);
        if (params?.offset !== undefined) queryParams.append('offset', params.offset.toString());
        if (params?.limit !== undefined) queryParams.append('limit', params.limit.toString());

        const response = await api.get(`/tasks?${queryParams.toString()}`);
        return response.data;
    },

    createTask: async (task: CreateTaskRequest): Promise<TaskResponse> => {
        const response: AxiosResponse<{ task: TaskResponse }> = await api.post('/tasks', task);
        return response.data.task;
    },

    getTask: async (id: string): Promise<TaskResponse> => {
        const response: AxiosResponse<{ task: TaskResponse }> = await api.get(`/tasks/${id}`);
        return response.data.task;
    },

    updateTask: async (id: string, updates: UpdateTaskRequest): Promise<TaskResponse> => {
        const response: AxiosResponse<{ task: TaskResponse }> = await api.put(`/tasks/${id}`, updates);
        return response.data.task;
    },

    deleteTask: async (id: string): Promise<{ message: string }> => {
        const response = await api.delete(`/tasks/${id}`);
        return response.data;
    },

    updateTaskStatus: async (id: string, statusUpdate: UpdateTaskStatusRequest): Promise<TaskResponse> => {
        const response: AxiosResponse<{ task: TaskResponse }> = await api.patch(`/tasks/${id}/status`, statusUpdate);
        return response.data.task;
    },

    markTaskCompleted: async (id: string): Promise<TaskResponse> => {
        const response: AxiosResponse<{ task: TaskResponse }> = await api.post(`/tasks/${id}/complete`);
        return response.data.task;
    },

    markTaskCancelled: async (id: string): Promise<TaskResponse> => {
        const response: AxiosResponse<{ task: TaskResponse }> = await api.post(`/tasks/${id}/cancel`);
        return response.data.task;
    },

    getTasksDueToday: async (): Promise<TaskResponse[]> => {
        const response = await api.get('/tasks/due-today');
        return response.data.tasks;
    },

    getOverdueTasks: async (): Promise<TaskResponse[]> => {
        const response = await api.get('/tasks/overdue');
        return response.data.tasks;
    },

    getUpcomingTasks: async (days: number = 7): Promise<TaskResponse[]> => {
        const response = await api.get(`/tasks/upcoming?days=${days}`);
        return response.data.tasks;
    },

    getCalendarTasks: async (fromDate: string, toDate: string): Promise<CalendarTaskResponse[]> => {
        const response = await api.get(`/tasks/calendar?from=${fromDate}&to=${toDate}`);
        return response.data.tasks;
    },

    searchTasks: async (query: string, filters?: ListTasksRequest): Promise<ListTasksResponse> => {
        const queryParams = new URLSearchParams();
        queryParams.append('q', query);
        if (filters?.status) queryParams.append('status', filters.status);
        if (filters?.priority) queryParams.append('priority', filters.priority.toString());
        if (filters?.tags) filters.tags.forEach(tag => queryParams.append('tags', tag));
        if (filters?.offset !== undefined) queryParams.append('offset', filters.offset.toString());
        if (filters?.limit !== undefined) queryParams.append('limit', filters.limit.toString());

        const response = await api.get(`/tasks/search?${queryParams.toString()}`);
        return response.data;
    },

    getTasksByTags: async (tags: string[], offset?: number, limit?: number): Promise<ListTasksResponse> => {
        const queryParams = new URLSearchParams();
        tags.forEach(tag => queryParams.append('tags', tag));
        if (offset !== undefined) queryParams.append('offset', offset.toString());
        if (limit !== undefined) queryParams.append('limit', limit.toString());

        const response = await api.get(`/tasks/tags?${queryParams.toString()}`);
        return response.data;
    },
};

// Admin API
export const adminAPI = {
    getUsers: async (offset = 0, limit = 20): Promise<{ users: UserResponse[]; pagination: { total: number; offset: number; limit: number } }> => {
        const response = await api.get(`/admin/users?offset=${offset}&limit=${limit}`);
        return response.data;
    },

    getUserById: async (id: string): Promise<UserResponse> => {
        const response: AxiosResponse<{ user: UserResponse }> = await api.get(`/admin/users/${id}`);
        return response.data.user;
    },
};

// Utility functions
export const isAuthenticated = (): boolean => {
    const token = localStorage.getItem('accessToken');
    const expiresAt = localStorage.getItem('tokenExpiresAt');

    if (!token || !expiresAt) return false;

    return new Date(expiresAt) > new Date();
};

export const getCurrentUser = (): UserResponse | null => {
    const userStr = localStorage.getItem('currentUser');
    return userStr ? JSON.parse(userStr) : null;
};

export const setCurrentUser = (user: UserResponse): void => {
    localStorage.setItem('currentUser', JSON.stringify(user));
};

export default api;
