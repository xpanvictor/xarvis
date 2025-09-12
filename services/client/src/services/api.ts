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
