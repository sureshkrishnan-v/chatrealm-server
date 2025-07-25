# Premium Chat Backend Usage Examples

## Table of Contents
- [Authentication Examples](#authentication-examples)
- [Chat Room Examples](#chat-room-examples)
- [Real-time Messaging](#real-time-messaging)
- [File Upload Examples](#file-upload-examples)
- [Admin Operations](#admin-operations)
- [Frontend Integration](#frontend-integration)

## Authentication Examples

### Register New User
```bash
curl -X POST http://localhost:5000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "securepass123"
  }'
```

**Response:**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "email": "john@example.com",
    "tier": "free",
    "is_active": true,
    "created_at": "2025-07-25T13:00:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600
}
```

### Create Guest User
```bash
curl -X POST http://localhost:5000/api/v1/auth/guest \
  -H "Content-Type: application/json" \
  -d '{
    "username": "anonymous_user_123"
  }'
```

### Login Existing User
```bash
curl -X POST http://localhost:5000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securepass123"
  }'
```

### Refresh Token
```bash
curl -X POST http://localhost:5000/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

## Chat Room Examples

### Get Available Chat Rooms
```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

curl -X GET "http://localhost:5000/api/v1/chat/rooms?page=1&limit=10&type=public" \
  -H "Authorization: Bearer $TOKEN"
```

### Create New Chat Room
```bash
curl -X POST http://localhost:5000/api/v1/chat/rooms \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Tech Discussion",
    "description": "A room for technical discussions",
    "type": "public",
    "max_members": 100,
    "is_private": false
  }'
```

### Join Chat Room
```bash
ROOM_ID="550e8400-e29b-41d4-a716-446655440001"

curl -X POST "http://localhost:5000/api/v1/chat/rooms/$ROOM_ID/join" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

### Send Message to Room
```bash
curl -X POST "http://localhost:5000/api/v1/chat/rooms/$ROOM_ID/messages" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Hello everyone! 👋",
    "type": "text"
  }'
```

### Reply to Message (Threading)
```bash
PARENT_MESSAGE_ID="550e8400-e29b-41d4-a716-446655440002"

curl -X POST "http://localhost:5000/api/v1/chat/rooms/$ROOM_ID/messages" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Great point! I agree.",
    "type": "text",
    "parent_id": "'$PARENT_MESSAGE_ID'"
  }'
```

### Get Room Messages
```bash
curl -X GET "http://localhost:5000/api/v1/chat/rooms/$ROOM_ID/messages?page=1&limit=50" \
  -H "Authorization: Bearer $TOKEN"
```

### Search Users
```bash
curl -X GET "http://localhost:5000/api/v1/chat/search/users?q=john&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

### React to Message
```bash
MESSAGE_ID="550e8400-e29b-41d4-a716-446655440003"

curl -X POST "http://localhost:5000/api/v1/chat/messages/$MESSAGE_ID/react" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "emoji": "👍"
  }'
```

## Real-time Messaging

### JavaScript WebSocket Client
```javascript
// Connect to WebSocket
const token = 'your-jwt-token-here';
const ws = new WebSocket(`ws://localhost:5000/api/v1/ws`, [], {
  headers: {
    'Authorization': `Bearer ${token}`
  }
});

// Handle connection
ws.onopen = function(event) {
  console.log('Connected to chat server');
  
  // Join a room
  ws.send(JSON.stringify({
    type: 'join_room',
    room_id: '550e8400-e29b-41d4-a716-446655440001'
  }));
};

// Handle incoming messages
ws.onmessage = function(event) {
  const data = JSON.parse(event.data);
  
  switch(data.type) {
    case 'message':
      console.log('New message:', data.data);
      displayMessage(data.data);
      break;
      
    case 'user_joined':
      console.log('User joined:', data.data.user.username);
      break;
      
    case 'user_left':
      console.log('User left:', data.data.user.username);
      break;
      
    case 'typing':
      displayTypingIndicator(data.data);
      break;
  }
};

// Send message via WebSocket
function sendMessage(roomId, content) {
  ws.send(JSON.stringify({
    type: 'send_message',
    room_id: roomId,
    content: content,
    message_type: 'text'
  }));
}

// Send typing indicator
function sendTyping(roomId, isTyping) {
  ws.send(JSON.stringify({
    type: 'typing',
    room_id: roomId,
    is_typing: isTyping
  }));
}
```

### Python WebSocket Client
```python
import asyncio
import websockets
import json

async def chat_client():
    token = "your-jwt-token-here"
    uri = "ws://localhost:5000/api/v1/ws"
    
    headers = {
        "Authorization": f"Bearer {token}"
    }
    
    async with websockets.connect(uri, extra_headers=headers) as websocket:
        # Join room
        await websocket.send(json.dumps({
            "type": "join_room",
            "room_id": "550e8400-e29b-41d4-a716-446655440001"
        }))
        
        # Listen for messages
        async for message in websocket:
            data = json.loads(message)
            print(f"Received: {data}")
            
            if data["type"] == "message":
                print(f"Message from {data['data']['user']['username']}: {data['data']['content']}")

# Run the client
asyncio.run(chat_client())
```

## File Upload Examples

### Upload Image File
```bash
curl -X POST http://localhost:5000/api/v1/chat/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/path/to/image.jpg" \
  -F "content=Check out this image!" \
  -F "room_id=$ROOM_ID"
```

### Upload with Message
```bash
curl -X POST "http://localhost:5000/api/v1/chat/rooms/$ROOM_ID/messages" \
  -H "Authorization: Bearer $TOKEN" \
  -F "content=Here's the document we discussed" \
  -F "type=file" \
  -F "file=@/path/to/document.pdf"
```

## Admin Operations

### Get All Users (Admin Only)
```bash
ADMIN_TOKEN="admin-jwt-token-here"

curl -X GET "http://localhost:5000/api/v1/admin/users?page=1&limit=50&tier=free" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Upgrade User to Premium
```bash
USER_ID="550e8400-e29b-41d4-a716-446655440004"

curl -X PUT "http://localhost:5000/api/v1/admin/users/$USER_ID/tier" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tier": "premium"
  }'
```

### Get Analytics Data
```bash
# User analytics
curl -X GET "http://localhost:5000/api/v1/admin/analytics?type=users&start_date=2025-07-01&end_date=2025-07-25" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Revenue analytics
curl -X GET "http://localhost:5000/api/v1/admin/analytics?type=revenue&start_date=2025-07-01&end_date=2025-07-25" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Message analytics
curl -X GET "http://localhost:5000/api/v1/admin/analytics?type=messages&start_date=2025-07-01&end_date=2025-07-25" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Delete Message (Admin)
```bash
MESSAGE_ID="550e8400-e29b-41d4-a716-446655440005"

curl -X DELETE "http://localhost:5000/api/v1/admin/messages/$MESSAGE_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Delete Chat Room (Admin)
```bash
curl -X DELETE "http://localhost:5000/api/v1/admin/rooms/$ROOM_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Subscription Management

### Get Subscription Status
```bash
curl -X GET http://localhost:5000/api/v1/subscription/status \
  -H "Authorization: Bearer $TOKEN"
```

### Create Premium Subscription
```bash
curl -X POST http://localhost:5000/api/v1/subscription/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "plan": "premium_monthly",
    "payment_method_id": "pm_1234567890"
  }'
```

### Cancel Subscription
```bash
curl -X POST http://localhost:5000/api/v1/subscription/cancel \
  -H "Authorization: Bearer $TOKEN"
```

## Frontend Integration

### React.js Chat Component Example
```jsx
import React, { useState, useEffect } from 'react';

const ChatRoom = ({ roomId, token }) => {
  const [messages, setMessages] = useState([]);
  const [newMessage, setNewMessage] = useState('');
  const [ws, setWs] = useState(null);

  useEffect(() => {
    // Connect to WebSocket
    const websocket = new WebSocket(`ws://localhost:5000/api/v1/ws`, [], {
      headers: { 'Authorization': `Bearer ${token}` }
    });

    websocket.onopen = () => {
      console.log('Connected to chat');
      websocket.send(JSON.stringify({
        type: 'join_room',
        room_id: roomId
      }));
    };

    websocket.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'message') {
        setMessages(prev => [...prev, data.data]);
      }
    };

    setWs(websocket);

    return () => websocket.close();
  }, [roomId, token]);

  const sendMessage = () => {
    if (newMessage.trim() && ws) {
      ws.send(JSON.stringify({
        type: 'send_message',
        room_id: roomId,
        content: newMessage,
        message_type: 'text'
      }));
      setNewMessage('');
    }
  };

  return (
    <div className="chat-room">
      <div className="messages">
        {messages.map(msg => (
          <div key={msg.id} className="message">
            <strong>{msg.user.username}:</strong> {msg.content}
          </div>
        ))}
      </div>
      
      <div className="message-input">
        <input
          type="text"
          value={newMessage}
          onChange={(e) => setNewMessage(e.target.value)}
          onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
          placeholder="Type a message..."
        />
        <button onClick={sendMessage}>Send</button>
      </div>
    </div>
  );
};

export default ChatRoom;
```

### Authentication Hook (React)
```jsx
import { useState, useEffect } from 'react';

const useAuth = () => {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('access_token'));

  const login = async (email, password) => {
    try {
      const response = await fetch('http://localhost:5000/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      });

      const data = await response.json();
      
      if (response.ok) {
        setUser(data.user);
        setToken(data.access_token);
        localStorage.setItem('access_token', data.access_token);
        localStorage.setItem('refresh_token', data.refresh_token);
        return { success: true };
      } else {
        return { success: false, error: data.error };
      }
    } catch (error) {
      return { success: false, error: 'Network error' };
    }
  };

  const createGuestUser = async (username) => {
    try {
      const response = await fetch('http://localhost:5000/api/v1/auth/guest', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username })
      });

      const data = await response.json();
      
      if (response.ok) {
        setUser(data.user);
        setToken(data.access_token);
        localStorage.setItem('access_token', data.access_token);
        return { success: true };
      } else {
        return { success: false, error: data.error };
      }
    } catch (error) {
      return { success: false, error: 'Network error' };
    }
  };

  const logout = () => {
    setUser(null);
    setToken(null);
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
  };

  return { user, token, login, createGuestUser, logout };
};

export default useAuth;
```

### API Client Helper
```javascript
class ChatAPI {
  constructor(baseURL = 'http://localhost:5000/api/v1', token = null) {
    this.baseURL = baseURL;
    this.token = token;
  }

  setToken(token) {
    this.token = token;
  }

  async request(endpoint, options = {}) {
    const url = `${this.baseURL}${endpoint}`;
    const config = {
      headers: {
        'Content-Type': 'application/json',
        ...(this.token && { 'Authorization': `Bearer ${this.token}` }),
        ...options.headers
      },
      ...options
    };

    const response = await fetch(url, config);
    const data = await response.json();

    if (!response.ok) {
      throw new Error(data.error || 'Request failed');
    }

    return data;
  }

  // Auth methods
  async register(username, email, password) {
    return this.request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, email, password })
    });
  }

  async login(email, password) {
    return this.request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password })
    });
  }

  async createGuest(username) {
    return this.request('/auth/guest', {
      method: 'POST',
      body: JSON.stringify({ username })
    });
  }

  // Chat methods
  async getRooms(page = 1, limit = 20, type = null) {
    const params = new URLSearchParams({ page, limit });
    if (type) params.append('type', type);
    return this.request(`/chat/rooms?${params}`);
  }

  async createRoom(roomData) {
    return this.request('/chat/rooms', {
      method: 'POST',
      body: JSON.stringify(roomData)
    });
  }

  async joinRoom(roomId, password = null) {
    return this.request(`/chat/rooms/${roomId}/join`, {
      method: 'POST',
      body: JSON.stringify(password ? { password } : {})
    });
  }

  async getMessages(roomId, page = 1, limit = 50) {
    return this.request(`/chat/rooms/${roomId}/messages?page=${page}&limit=${limit}`);
  }

  async sendMessage(roomId, content, type = 'text', parentId = null) {
    return this.request(`/chat/rooms/${roomId}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content, type, parent_id: parentId })
    });
  }

  async searchUsers(query, limit = 20) {
    return this.request(`/chat/search/users?q=${encodeURIComponent(query)}&limit=${limit}`);
  }

  async reactToMessage(messageId, emoji) {
    return this.request(`/chat/messages/${messageId}/react`, {
      method: 'POST',
      body: JSON.stringify({ emoji })
    });
  }
}

// Usage
const api = new ChatAPI();
api.setToken('your-jwt-token');

// Use the API
try {
  const rooms = await api.getRooms();
  console.log('Available rooms:', rooms);
} catch (error) {
  console.error('Failed to get rooms:', error.message);
}
```

## Error Handling Examples

### Handling Rate Limits
```javascript
async function sendMessageWithRetry(api, roomId, content, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await api.sendMessage(roomId, content);
    } catch (error) {
      if (error.message.includes('rate limit') && i < maxRetries - 1) {
        // Wait before retrying (exponential backoff)
        await new Promise(resolve => setTimeout(resolve, Math.pow(2, i) * 1000));
        continue;
      }
      throw error;
    }
  }
}
```

### Token Refresh Implementation
```javascript
class AuthenticatedAPI extends ChatAPI {
  constructor(baseURL, accessToken, refreshToken) {
    super(baseURL, accessToken);
    this.refreshToken = refreshToken;
  }

  async request(endpoint, options = {}) {
    try {
      return await super.request(endpoint, options);
    } catch (error) {
      if (error.message.includes('unauthorized') || error.message.includes('token')) {
        // Try to refresh token
        const newTokens = await this.refreshTokens();
        this.setToken(newTokens.access_token);
        
        // Retry original request
        return super.request(endpoint, options);
      }
      throw error;
    }
  }

  async refreshTokens() {
    return this.request('/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: this.refreshToken })
    });
  }
}
```

These examples demonstrate the full functionality of the Premium Chat Backend API, from basic authentication to advanced real-time messaging and administrative operations.