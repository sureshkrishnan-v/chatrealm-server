# Premium Chat Backend API Documentation

## Overview

The Premium Chat Backend is a comprehensive Go-based chat server with tier-based access control, real-time messaging, and payment integration. This API provides endpoints for authentication, chat management, subscriptions, and administration.

**Base URL**: `http://localhost:5000/api/v1`
**WebSocket URL**: `ws://localhost:5000/api/v1/ws`

## Authentication

All protected endpoints require a Bearer token in the Authorization header:
```
Authorization: Bearer <your_jwt_token>
```

### User Tiers
- **Free**: Basic chat access to public rooms
- **Premium**: Access to private and VIP rooms, file uploads
- **Admin**: Full access including user management and analytics

## Endpoints

### Authentication Endpoints

#### POST /auth/register
Register a new user account.

**Request Body:**
```json
{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "securepassword123"
}
```

**Response:**
```json
{
  "user": {
    "id": "uuid-here",
    "username": "johndoe",
    "email": "john@example.com",
    "tier": "free",
    "is_active": true,
    "created_at": "2025-07-25T13:00:00Z"
  },
  "access_token": "jwt-token-here",
  "refresh_token": "refresh-token-here",
  "expires_in": 3600
}
```

#### POST /auth/login
Login with existing credentials.

**Request Body:**
```json
{
  "email": "john@example.com",
  "password": "securepassword123"
}
```

**Response:** Same as registration

#### POST /auth/guest
Create anonymous guest user (no email required).

**Request Body:**
```json
{
  "username": "guest123"
}
```

**Response:**
```json
{
  "user": {
    "id": "uuid-here",
    "username": "guest123",
    "email": "",
    "tier": "free",
    "is_active": true,
    "created_at": "2025-07-25T13:00:00Z"
  },
  "access_token": "jwt-token-here",
  "refresh_token": "refresh-token-here",
  "expires_in": 3600,
  "user_type": "guest"
}
```

#### POST /auth/refresh
Refresh JWT tokens using refresh token.

**Request Body:**
```json
{
  "refresh_token": "refresh-token-here"
}
```

#### POST /auth/logout
Logout and invalidate tokens (requires authentication).

**Response:**
```json
{
  "message": "Logged out successfully"
}
```

### Chat Endpoints

#### GET /chat/rooms
Get list of accessible chat rooms.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20, max: 100)
- `type` (optional): Filter by room type (public, private, vip)

**Response:**
```json
{
  "rooms": [
    {
      "id": "uuid-here",
      "name": "General Chat",
      "description": "Public chat for all users",
      "type": "public",
      "max_members": 1000,
      "is_private": false,
      "settings": {
        "allow_file_uploads": false,
        "allow_image_uploads": true,
        "max_file_size": 10485760,
        "allow_reactions": true
      },
      "creator": {
        "id": "uuid-here",
        "username": "admin",
        "tier": "admin"
      },
      "created_at": "2025-07-25T13:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

#### POST /chat/rooms
Create a new chat room (requires authentication).

**Request Body:**
```json
{
  "name": "My Private Room",
  "description": "A private room for friends",
  "type": "private",
  "max_members": 50,
  "is_private": true,
  "password": "optional-password"
}
```

#### GET /chat/rooms/:id
Get specific chat room details.

#### POST /chat/rooms/:id/join
Join a chat room.

**Request Body (for private rooms):**
```json
{
  "password": "room-password"
}
```

#### POST /chat/rooms/:id/leave
Leave a chat room.

#### GET /chat/rooms/:id/messages
Get messages from a chat room.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Messages per page (default: 50)
- `before` (optional): Get messages before this timestamp

**Response:**
```json
{
  "messages": [
    {
      "id": "uuid-here",
      "content": "Hello everyone!",
      "type": "text",
      "user": {
        "id": "uuid-here",
        "username": "johndoe",
        "tier": "free"
      },
      "chat_room_id": "uuid-here",
      "parent_id": null,
      "reactions": [],
      "created_at": "2025-07-25T13:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 50
}
```

#### POST /chat/rooms/:id/messages
Send a message to a chat room (rate limited).

**Request Body:**
```json
{
  "content": "Hello everyone!",
  "type": "text",
  "parent_id": "uuid-here",
  "mentions": ["user-uuid-1", "user-uuid-2"]
}
```

**File Upload (multipart/form-data):**
```
content: "Check out this image!"
type: "file"
file: [uploaded file]
```

#### POST /chat/messages/:id/react
React to a message.

**Request Body:**
```json
{
  "emoji": "👍"
}
```

#### GET /chat/search/users
Search users by username.

**Query Parameters:**
- `q` (required): Search query
- `limit` (optional): Max results (default: 20, max: 50)

**Response:**
```json
{
  "users": [
    {
      "id": "uuid-here",
      "username": "johndoe",
      "tier": "free",
      "created_at": "2025-07-25T13:00:00Z"
    }
  ],
  "count": 1
}
```

### WebSocket Connection

#### GET /ws
Establish WebSocket connection for real-time messaging.

**Headers:**
```
Authorization: Bearer <jwt-token>
Upgrade: websocket
Connection: Upgrade
```

**Message Format:**
```json
{
  "type": "message",
  "room_id": "uuid-here",
  "content": "Hello real-time!",
  "user": {
    "id": "uuid-here",
    "username": "johndoe"
  },
  "timestamp": "2025-07-25T13:00:00Z"
}
```

**Message Types:**
- `message`: New chat message
- `user_joined`: User joined room
- `user_left`: User left room
- `typing`: User typing indicator
- `reaction`: Message reaction

### Subscription Endpoints

#### GET /subscription/status
Get current subscription status.

**Response:**
```json
{
  "subscription": {
    "id": "uuid-here",
    "plan": "premium_monthly",
    "status": "active",
    "current_period_start": "2025-07-01T00:00:00Z",
    "current_period_end": "2025-08-01T00:00:00Z",
    "trial_end": null
  },
  "user": {
    "tier": "premium"
  }
}
```

#### POST /subscription/create
Create new subscription.

**Request Body:**
```json
{
  "plan": "premium_monthly",
  "payment_method_id": "stripe-payment-method-id"
}
```

#### POST /subscription/cancel
Cancel subscription.

**Response:**
```json
{
  "message": "Subscription cancelled",
  "cancelled_at": "2025-07-25T13:00:00Z"
}
```

#### POST /subscription/webhook
Stripe webhook endpoint (internal use only).

### Admin Endpoints (Admin tier required)

#### GET /admin/users
Get list of all users.

**Query Parameters:**
- `page`, `limit`: Pagination
- `tier`: Filter by user tier
- `active`: Filter by active status

#### PUT /admin/users/:id/tier
Update user tier.

**Request Body:**
```json
{
  "tier": "premium"
}
```

#### GET /admin/analytics
Get analytics data.

**Query Parameters:**
- `type`: Analytics type (users, revenue, messages)
- `start_date`: Start date (YYYY-MM-DD)
- `end_date`: End date (YYYY-MM-DD)

**Response:**
```json
{
  "type": "users",
  "period": {
    "start": "2025-07-01",
    "end": "2025-07-25"
  },
  "data": {
    "total_users": 1500,
    "active_users": 800,
    "new_users": 150,
    "by_tier": {
      "free": 1200,
      "premium": 290,
      "admin": 10
    }
  }
}
```

#### GET /admin/rooms
Get all chat rooms (admin view).

#### DELETE /admin/rooms/:id
Delete a chat room.

#### GET /admin/messages/:id
Get specific message details.

#### DELETE /admin/messages/:id
Delete a message.

## Error Responses

All endpoints return errors in this format:

```json
{
  "error": "Error description",
  "code": "ERROR_CODE",
  "details": {}
}
```

**Common HTTP Status Codes:**
- `400`: Bad Request - Invalid input
- `401`: Unauthorized - Missing or invalid token
- `403`: Forbidden - Insufficient permissions
- `404`: Not Found - Resource doesn't exist
- `409`: Conflict - Resource already exists
- `429`: Too Many Requests - Rate limit exceeded
- `500`: Internal Server Error

## Rate Limiting

- Message sending: 10 messages per minute per user
- API requests: 100 requests per minute per IP
- File uploads: 5 uploads per minute per user

## File Upload Limits

- **Free users**: No file uploads
- **Premium users**: 
  - Max file size: 10MB
  - Allowed types: Images (jpg, png, gif), Documents (pdf, txt)
- **Admin users**: No restrictions

## WebSocket Events

### Client to Server:
```json
{
  "type": "join_room",
  "room_id": "uuid-here"
}

{
  "type": "send_message",
  "room_id": "uuid-here",
  "content": "Hello!",
  "message_type": "text"
}

{
  "type": "typing",
  "room_id": "uuid-here",
  "is_typing": true
}
```

### Server to Client:
```json
{
  "type": "message",
  "data": {
    "id": "uuid-here",
    "content": "Hello!",
    "user": {...},
    "room_id": "uuid-here",
    "timestamp": "2025-07-25T13:00:00Z"
  }
}

{
  "type": "user_joined",
  "data": {
    "user": {...},
    "room_id": "uuid-here"
  }
}

{
  "type": "error",
  "data": {
    "message": "Error description"
  }
}
```