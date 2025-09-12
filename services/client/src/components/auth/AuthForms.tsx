import React, { useState } from 'react';
import { useForm } from 'react-hook-form';
import { Eye, EyeOff, Mail, Lock, User, AlertCircle } from 'lucide-react';
import { authAPI, LoginRequest, RegisterRequest, setCurrentUser } from '../../services/api';
import { useAuthStore } from '../../store';
import './Auth.css';

interface LoginFormData extends LoginRequest { }

interface RegisterFormData extends RegisterRequest {
    confirmPassword: string;
}

interface AuthFormProps {
    onSuccess?: () => void;
}

export const LoginForm: React.FC<AuthFormProps> = ({ onSuccess }) => {
    const [showPassword, setShowPassword] = useState(false);
    const [isLoading, setIsLoading] = useState(false);
    const { setUser, setError } = useAuthStore();

    const {
        register,
        handleSubmit,
        formState: { errors },
        setError: setFormError
    } = useForm<LoginFormData>();

    const onSubmit = async (data: LoginFormData) => {
        setIsLoading(true);
        setError(null);

        try {
            const response = await authAPI.login(data);

            // Store tokens
            localStorage.setItem('accessToken', response.tokens.accessToken);
            localStorage.setItem('refreshToken', response.tokens.refreshToken);
            localStorage.setItem('tokenExpiresAt', response.tokens.expiresAt);

            // Store user data
            setCurrentUser(response.user);
            setUser(response.user);

            onSuccess?.();
        } catch (error: any) {
            const errorMessage = error.response?.data?.error || 'Login failed. Please try again.';
            setError(errorMessage);
            setFormError('root', { message: errorMessage });
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="auth-form">
            <div className="auth-header">
                <h1>Welcome Back</h1>
                <p>Sign in to your Xarvis account</p>
            </div>

            <form onSubmit={handleSubmit(onSubmit)} className="auth-form-content">
                {errors.root && (
                    <div className="error-message">
                        <AlertCircle size={16} />
                        {errors.root.message}
                    </div>
                )}

                <div className="form-group">
                    <label htmlFor="email">Email</label>
                    <div className="input-wrapper">
                        <Mail className="input-icon" size={18} />
                        <input
                            id="email"
                            type="email"
                            {...register('email', {
                                required: 'Email is required',
                                pattern: {
                                    value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                                    message: 'Invalid email address'
                                }
                            })}
                            placeholder="Enter your email"
                            className={errors.email ? 'error' : ''}
                        />
                    </div>
                    {errors.email && <span className="field-error">{errors.email.message}</span>}
                </div>

                <div className="form-group">
                    <label htmlFor="password">Password</label>
                    <div className="input-wrapper">
                        <Lock className="input-icon" size={18} />
                        <input
                            id="password"
                            type={showPassword ? 'text' : 'password'}
                            {...register('password', {
                                required: 'Password is required',
                                minLength: {
                                    value: 8,
                                    message: 'Password must be at least 8 characters'
                                }
                            })}
                            placeholder="Enter your password"
                            className={errors.password ? 'error' : ''}
                        />
                        <button
                            type="button"
                            className="password-toggle"
                            onClick={() => setShowPassword(!showPassword)}
                        >
                            {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
                        </button>
                    </div>
                    {errors.password && <span className="field-error">{errors.password.message}</span>}
                </div>

                <button type="submit" className="auth-button" disabled={isLoading}>
                    {isLoading ? 'Signing In...' : 'Sign In'}
                </button>
            </form>
        </div>
    );
};

export const RegisterForm: React.FC<AuthFormProps> = ({ onSuccess }) => {
    const [showPassword, setShowPassword] = useState(false);
    const [isLoading, setIsLoading] = useState(false);
    const { setError } = useAuthStore();

    const {
        register,
        handleSubmit,
        formState: { errors },
        watch,
        setError: setFormError
    } = useForm<RegisterFormData>();

    const password = watch('password');

    const onSubmit = async (data: RegisterFormData) => {
        setIsLoading(true);
        setError(null);

        try {
            const response = await authAPI.register(data);

            // Auto-login after successful registration
            const loginResponse = await authAPI.login({
                email: data.email,
                password: data.password
            });

            // Store tokens
            localStorage.setItem('accessToken', loginResponse.tokens.accessToken);
            localStorage.setItem('refreshToken', loginResponse.tokens.refreshToken);
            localStorage.setItem('tokenExpiresAt', loginResponse.tokens.expiresAt);

            // Store user data
            setCurrentUser(loginResponse.user);

            onSuccess?.();
        } catch (error: any) {
            const errorMessage = error.response?.data?.error || 'Registration failed. Please try again.';
            setError(errorMessage);
            setFormError('root', { message: errorMessage });
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="auth-form">
            <div className="auth-header">
                <h1>Create Account</h1>
                <p>Join Xarvis to get started</p>
            </div>

            <form onSubmit={handleSubmit(onSubmit)} className="auth-form-content">
                {errors.root && (
                    <div className="error-message">
                        <AlertCircle size={16} />
                        {errors.root.message}
                    </div>
                )}

                <div className="form-group">
                    <label htmlFor="displayName">Display Name</label>
                    <div className="input-wrapper">
                        <User className="input-icon" size={18} />
                        <input
                            id="displayName"
                            type="text"
                            {...register('displayName', {
                                required: 'Display name is required',
                                minLength: {
                                    value: 2,
                                    message: 'Display name must be at least 2 characters'
                                },
                                maxLength: {
                                    value: 100,
                                    message: 'Display name must be less than 100 characters'
                                }
                            })}
                            placeholder="Enter your display name"
                            className={errors.displayName ? 'error' : ''}
                        />
                    </div>
                    {errors.displayName && <span className="field-error">{errors.displayName.message}</span>}
                </div>

                <div className="form-group">
                    <label htmlFor="email">Email</label>
                    <div className="input-wrapper">
                        <Mail className="input-icon" size={18} />
                        <input
                            id="email"
                            type="email"
                            {...register('email', {
                                required: 'Email is required',
                                pattern: {
                                    value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                                    message: 'Invalid email address'
                                }
                            })}
                            placeholder="Enter your email"
                            className={errors.email ? 'error' : ''}
                        />
                    </div>
                    {errors.email && <span className="field-error">{errors.email.message}</span>}
                </div>

                <div className="form-group">
                    <label htmlFor="password">Password</label>
                    <div className="input-wrapper">
                        <Lock className="input-icon" size={18} />
                        <input
                            id="password"
                            type={showPassword ? 'text' : 'password'}
                            {...register('password', {
                                required: 'Password is required',
                                minLength: {
                                    value: 8,
                                    message: 'Password must be at least 8 characters'
                                },
                                pattern: {
                                    value: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/,
                                    message: 'Password must contain at least one uppercase letter, one lowercase letter, and one number'
                                }
                            })}
                            placeholder="Enter your password"
                            className={errors.password ? 'error' : ''}
                        />
                        <button
                            type="button"
                            className="password-toggle"
                            onClick={() => setShowPassword(!showPassword)}
                        >
                            {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
                        </button>
                    </div>
                    {errors.password && <span className="field-error">{errors.password.message}</span>}
                </div>

                <div className="form-group">
                    <label htmlFor="confirmPassword">Confirm Password</label>
                    <div className="input-wrapper">
                        <Lock className="input-icon" size={18} />
                        <input
                            id="confirmPassword"
                            type={showPassword ? 'text' : 'password'}
                            {...register('confirmPassword', {
                                required: 'Please confirm your password',
                                validate: (value) => value === password || 'Passwords do not match'
                            })}
                            placeholder="Confirm your password"
                            className={errors.confirmPassword ? 'error' : ''}
                        />
                    </div>
                    {errors.confirmPassword && <span className="field-error">{errors.confirmPassword.message}</span>}
                </div>

                <button type="submit" className="auth-button" disabled={isLoading}>
                    {isLoading ? 'Creating Account...' : 'Create Account'}
                </button>
            </form>
        </div>
    );
};
