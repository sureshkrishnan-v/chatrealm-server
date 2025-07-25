# Premium Chat Backend

A comprehensive Go-based chat server with tier-based access control, real-time messaging, and payment integration. Built with modern technologies for scalability and performance.

## Features

### 🚀 Core Features
- **Multi-tier Authentication**: Free, Premium, and Admin user levels
- **Real-time Messaging**: WebSocket-based instant communication
- **Anonymous Guest Access**: Join without registration (like chatApplication.com)
- **Username Search**: Find users by username
- **File Upload Support**: Images and documents for premium users
- **Room Management**: Public, private, and VIP chat rooms

### 💰 Premium Features
- **Stripe Integration**: Subscription billing and payments
- **Tier-based Access Control**: Feature restrictions by user level
- **Admin Dashboard**: User management and analytics
- **Rate Limiting**: Prevents spam and abuse
- **Analytics**: Usage statistics and revenue tracking

### 🔧 Technical Features
- **JWT Authentication**: Secure token-based auth
- **PostgreSQL Database**: Robust data storage with proper relationships
- **Redis Caching**: Session management and performance optimization
- **Comprehensive API**: RESTful endpoints with full documentation
- **Docker Support**: Easy deployment and scaling

## Quick Start

### Prerequisites
- Go 1.19+
- PostgreSQL 12+
- Redis 6+ (optional)

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd premium-chat-backend
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Set up environment variables**
   ```bash
   export PORT=5000
   export DATABASE_URL="postgres://user:pass@localhost:5432/premium_chat?sslmode=disable"
   export JWT_SECRET="your-super-secret-key"
   export STRIPE_SECRET_KEY="sk_test_your_stripe_key"
   ```

4. **Run the server**
   ```bash
   go run main.go
   ```

The server will start on `http://localhost:5000`

### Docker Setup

```bash
# Build and run with Docker Compose
docker-compose up -d
```

## API Overview

### Authentication
```bash
# Register new user
curl -X POST http://localhost:5000/api/v1/auth/register \
  -d '{"username":"john","email":"john@example.com","password":"secret123"}'

# Create guest user (anonymous)
curl -X POST http://localhost:5000/api/v1/auth/guest \
  -d '{"username":"guest123"}'

# Login
curl -X POST http://localhost:5000/api/v1/auth/login \
  -d '{"email":"john@example.com","password":"secret123"}'
```

### Chat Rooms
```bash
# Get available rooms
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:5000/api/v1/chat/rooms

# Send message
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:5000/api/v1/chat/rooms/$ROOM_ID/messages \
  -d '{"content":"Hello everyone!"}'

# Search users
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:5000/api/v1/chat/search/users?q=john"
```

### WebSocket Connection
```javascript
const ws = new WebSocket('ws://localhost:5000/api/v1/ws', [], {
  headers: { 'Authorization': 'Bearer ' + token }
});

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Real-time message:', data);
};
```

## Architecture

### Directory Structure
```
premium-chat-backend/
├── config/          # Configuration management
├── database/        # Database connection and migrations
├── handlers/        # HTTP request handlers
├── middleware/      # Authentication, CORS, rate limiting
├── models/          # Database models and structures
├── services/        # Business logic layer
├── utils/           # Utility functions
├── docs/            # Documentation
└── main.go          # Application entry point
```

### User Tiers

| Tier | Features |
|------|----------|
| **Free** | Public rooms, basic messaging |
| **Premium** | Private rooms, VIP access, file uploads |
| **Admin** | Full access, user management, analytics |

### Room Types

| Type | Access Level | Features |
|------|-------------|----------|
| **Public** | All users | Open chat, basic features |
| **Private** | Premium+ | Password protection, enhanced features |
| **VIP** | Premium+ | Exclusive rooms, premium features |

## Comparison with chatApplication.com

| Feature | chatApplication.com | Premium Chat Backend |
|---------|-------------|---------------------|
| Anonymous Access | ✅ | ✅ Enhanced |
| User Registration | Optional | ✅ Full featured |
| Username Search | ✅ | ✅ |
| Real-time Chat | ✅ Basic | ✅ WebSocket |
| File Uploads | ❌ | ✅ Premium feature |
| User Tiers | ❌ | ✅ Free/Premium/Admin |
| Payment Integration | ❌ | ✅ Stripe |
| Admin Panel | ❌ | ✅ Full featured |
| Mobile Support | ✅ | ✅ API-first |
| Spam Protection | Limited | ✅ Rate limiting |

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/guest` - Guest user creation
- `POST /api/v1/auth/refresh` - Token refresh
- `POST /api/v1/auth/logout` - User logout

### Chat
- `GET /api/v1/chat/rooms` - List chat rooms
- `POST /api/v1/chat/rooms` - Create room
- `GET /api/v1/chat/rooms/:id/messages` - Get messages
- `POST /api/v1/chat/rooms/:id/messages` - Send message
- `GET /api/v1/chat/search/users` - Search users
- `GET /api/v1/ws` - WebSocket connection

### Subscriptions
- `GET /api/v1/subscription/status` - Subscription status
- `POST /api/v1/subscription/create` - Create subscription
- `POST /api/v1/subscription/cancel` - Cancel subscription

### Admin
- `GET /api/v1/admin/users` - Manage users
- `GET /api/v1/admin/analytics` - View analytics
- `DELETE /api/v1/admin/messages/:id` - Delete messages

## Documentation

- **[API Documentation](docs/API.md)** - Complete API reference
- **[Setup Guide](docs/SETUP.md)** - Detailed installation and configuration
- **[Usage Examples](docs/EXAMPLES.md)** - Code examples and integrations

## Features in Detail

### Real-time Messaging
- WebSocket-based instant messaging
- User presence indicators
- Typing indicators
- Message reactions and threading
- File sharing (premium users)

### User Management
- JWT-based authentication
- Secure password hashing
- Session management with Redis
- Anonymous guest users
- User search functionality

### Payment Integration
- Stripe subscription management
- Automatic tier upgrades
- Webhook handling for real-time updates
- Billing history tracking

### Admin Features
- User management dashboard
- Message moderation
- Room management
- Analytics and reporting
- Revenue tracking

### Security Features
- Rate limiting (API and messaging)
- CORS configuration
- SQL injection prevention
- JWT token validation
- Input validation and sanitization

## Performance

### Optimizations
- Database indexing for fast queries
- Connection pooling
- Redis caching for sessions
- Efficient WebSocket handling
- Pagination for large datasets

### Scalability
- Stateless design for horizontal scaling
- Database migrations
- Docker containerization
- Load balancer friendly
- Microservice ready architecture

## Development

### Running Tests
```bash
go test ./...
```

### Development Mode
```bash
# With auto-reload
go install github.com/cosmtrek/air@latest
air

# Or standard
go run main.go
```

### Database Migrations
Migrations run automatically on startup, or manually:
```bash
go run main.go --migrate-only
```

## Deployment

### Production Checklist
- [ ] Set `GIN_MODE=release`
- [ ] Configure SSL/TLS
- [ ] Set up reverse proxy
- [ ] Configure monitoring
- [ ] Set up automated backups
- [ ] Configure error tracking

### Environment Variables
```bash
# Required
DATABASE_URL=postgres://...
JWT_SECRET=your-secret-key

# Optional
REDIS_URL=redis://localhost:6379
STRIPE_SECRET_KEY=sk_live_...
PORT=5000
ENVIRONMENT=production
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Support

For questions and support:
- Check the [Setup Guide](docs/SETUP.md) for common issues
- Review the [API Documentation](docs/API.md) for usage
- Look at [Examples](docs/EXAMPLES.md) for integration help

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by chatApplication.com's simplicity and anonymity features
- Built with modern Go best practices
- Designed for scalability and performance

---

**Ready for production use** ⚡ **Scalable architecture** 🚀 **Comprehensive documentation** 📚