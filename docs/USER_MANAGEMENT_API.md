# User Management API Documentation

This document describes the user management endpoints that have been added to the Xarvis system.

## Authentication Flow

The system uses JWT (JSON Web Tokens) for authentication. The flow is:

1. **Register** a new user account
2. **Login** to get JWT tokens
3. Use the **access token** in the `Authorization: Bearer <token>` header for protected endpoints
4. **Refresh** the token when it expires using the refresh token

## Endpoints

### Public Endpoints (No Authentication Required)

#### POST /api/v1/auth/register
Register a new user account.

**Request Body:**
```json
{
  "displayName": "John Doe",
  "email": "john@example.com",
  "password": "secure-password-123",
  "timezone": "America/New_York",
  "settings": {
    "theme": "dark",
    "notifications": true
  }
}
```

**Response (201 Created):**
```json
{
  "message": "User registered successfully",
  "user": {
    "id": "uuid-string",
    "displayName": "John Doe",
    "email": "john@example.com",
    "timezone": "America/New_York",
    "settings": {...},
    "offTimes": [],
    "createdAt": "2025-01-11T12:00:00Z",
    "updatedAt": "2025-01-11T12:00:00Z"
  }
}
```

#### POST /api/v1/auth/login
Login with email and password.

**Request Body:**
```json
{
  "email": "john@example.com",
  "password": "secure-password-123"
}
```

**Response (200 OK):**
```json
{
  "message": "Login successful",
  "user": {
    "id": "uuid-string",
    "displayName": "John Doe",
    "email": "john@example.com",
    ...
  },
  "tokens": {
    "accessToken": "jwt-access-token",
    "refreshToken": "jwt-refresh-token", 
    "expiresAt": "2025-01-12T12:00:00Z"
  }
}
```

#### POST /api/v1/auth/refresh
Refresh an expired access token.

**Request Body:**
```json
{
  "refreshToken": "jwt-refresh-token"
}
```

**Response (200 OK):**
```json
{
  "message": "Token refreshed successfully",
  "tokens": {
    "accessToken": "new-jwt-access-token",
    "refreshToken": "new-jwt-refresh-token",
    "expiresAt": "2025-01-12T12:00:00Z"
  }
}
```

### Protected Endpoints (Authentication Required)

Add `Authorization: Bearer <access-token>` header to all requests.

#### GET /api/v1/user/profile
Get the current user's profile.

**Response (200 OK):**
```json
{
  "user": {
    "id": "uuid-string",
    "displayName": "John Doe",
    "email": "john@example.com",
    "timezone": "America/New_York",
    "settings": {...},
    "offTimes": [],
    "createdAt": "2025-01-11T12:00:00Z",
    "updatedAt": "2025-01-11T12:00:00Z"
  }
}
```

#### PUT /api/v1/user/profile
Update the current user's profile.

**Request Body (all fields optional):**
```json
{
  "displayName": "John Smith",
  "timezone": "Europe/London",
  "settings": {
    "theme": "light",
    "notifications": false
  },
  "offTimes": [
    {
      "start": "2025-01-15T09:00:00Z",
      "end": "2025-01-15T17:00:00Z",
      "label": "Work hours"
    }
  ]
}
```

**Response (200 OK):**
```json
{
  "message": "Profile updated successfully",
  "user": {
    "id": "uuid-string",
    "displayName": "John Smith",
    "email": "john@example.com",
    "timezone": "Europe/London",
    ...
  }
}
```

#### DELETE /api/v1/user/account
Delete the current user's account (soft delete).

**Response (200 OK):**
```json
{
  "message": "Account deleted successfully"
}
```

### Admin Endpoints (Admin Role Required)

*Note: Admin middleware is currently a placeholder. Role-based access control needs to be implemented.*

#### GET /api/v1/admin/users
List all users with pagination.

**Query Parameters:**
- `offset` (optional): Number of users to skip (default: 0)
- `limit` (optional): Number of users to return (default: 20, max: 100)

**Response (200 OK):**
```json
{
  "users": [...],
  "pagination": {
    "total": 150,
    "offset": 0,
    "limit": 20
  }
}
```

#### GET /api/v1/admin/users/:id
Get a specific user by ID.

**Response (200 OK):**
```json
{
  "user": {
    "id": "uuid-string",
    "displayName": "John Doe",
    ...
  }
}
```

## Error Responses

All endpoints return appropriate HTTP status codes with error messages:

**400 Bad Request:**
```json
{
  "error": "Invalid request data",
  "details": "validation error details"
}
```

**401 Unauthorized:**
```json
{
  "error": "Invalid credentials"
}
```

**404 Not Found:**
```json
{
  "error": "User not found"
}
```

**409 Conflict:**
```json
{
  "error": "Email already exists"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Internal server error"
}
```

## Configuration

Add the following to your `config_dev.yaml`:

```yaml
auth:
  jwt_secret: "your-secret-key-here"
  token_ttl_hours: 24
```

**Important:** Change the JWT secret to a secure random string in production!

## Database Schema

The user table includes:
- `id` (UUID, primary key)
- `display_name` (string, not null)
- `email` (string, unique, not null)
- `password_hash` (string, not null, never exposed in API)
- `timezone` (string, default: 'UTC')
- `settings` (JSONB)
- `off_times` (JSONB array of time ranges)
- `created_at` (timestamp)
- `updated_at` (timestamp)

## WebSocket Integration

The existing WebSocket endpoints remain unchanged:
- `/ws/` - Main websocket with audio/text support
- `/ws/audio` - Audio-only websocket
- `/ws/text` - Text-only websocket
- `/ws/legacy` - Legacy demo websocket

## Next Steps

1. Implement role-based access control for admin endpoints
2. Add password reset functionality
3. Add email verification for new accounts
4. Add rate limiting for authentication endpoints
5. Add audit logging for user actions
6. Consider adding OAuth2/Social login support
