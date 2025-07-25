# Premium Chat Backend - Project Overview

## Overview

A comprehensive Go backend server for a premium chat application featuring tier-based user access, real-time messaging, subscription management, and advanced chat functionality. The server provides separate chatrooms for free and premium users with WebSocket support for real-time communication.

## User Preferences

Preferred communication style: Simple, everyday language.

## System Architecture

**Backend Framework**: Go 1.19 with Gin HTTP framework
**Database**: PostgreSQL with GORM ORM
**Real-time Communication**: WebSocket support via Gorilla WebSocket
**Session Management**: Redis for caching and session storage
**Payment Processing**: Stripe integration for subscriptions
**Authentication**: JWT tokens with bcrypt password hashing

### Core Components:
- **REST API**: Comprehensive endpoints for chat, auth, subscriptions, and admin
- **WebSocket Server**: Real-time messaging and user presence
- **Database Layer**: PostgreSQL with proper foreign key relationships
- **Authentication System**: JWT-based with tier-based access control
- **Payment Integration**: Stripe webhooks and subscription management

## Key Components

### Models
- **User**: Username, email, password, tier (Free/Premium/Admin)
- **ChatRoom**: Room management with type-based access (Public/Private/VIP)
- **Message**: Text messages with threading and file upload support
- **Subscription**: Stripe integration with billing history
- **Analytics**: User activity and revenue tracking

### Services
- **AuthService**: Registration, login, JWT token management
- **ChatService**: Room and message management with tier restrictions
- **SubscriptionService**: Stripe payment processing and webhooks
- **WebSocketService**: Real-time message broadcasting
- **AnalyticsService**: Usage analytics and reporting

### Handlers
- **Auth**: `/api/v1/auth/*` - Registration, login, token refresh
- **Chat**: `/api/v1/chat/*` - Room management and messaging  
- **Subscription**: `/api/v1/subscription/*` - Payment and billing
- **Admin**: `/api/v1/admin/*` - User management and analytics
- **WebSocket**: `/api/v1/ws` - Real-time connections

## Data Flow

**Authentication Flow**:
1. User registers/logs in via `/auth` endpoints
2. Server validates credentials and returns JWT tokens
3. Client includes JWT in Authorization header for protected routes
4. Middleware validates token and extracts user context

**Chat Flow**:
1. User joins chat rooms based on their tier level
2. Messages sent via REST API and broadcasted via WebSocket
3. File uploads processed and stored with tier-based limits
4. Message history filtered by user access level

**Subscription Flow**:
1. User initiates subscription via Stripe checkout
2. Stripe webhook confirms payment
3. User tier upgraded automatically
4. Access to premium features granted

## External Dependencies

### Core Framework
- `gin-gonic/gin` - HTTP web framework
- `gorm.io/gorm` - Database ORM
- `gorm.io/driver/postgres` - PostgreSQL driver

### Authentication & Security
- `golang-jwt/jwt/v5` - JWT token handling
- `golang.org/x/crypto` - Password hashing

### Real-time & Caching
- `gorilla/websocket` - WebSocket implementation
- `go-redis/redis/v8` - Redis client

### Payment Processing
- `stripe/stripe-go/v75` - Stripe payment integration

### Utilities
- `google/uuid` - UUID generation
- `sirupsen/logrus` - Structured logging
- `go-playground/validator/v10` - Input validation

## Deployment Strategy

**Current Status**: Production-ready server running on port 5000
**Database**: PostgreSQL with auto-migrations
**Environment**: Replit environment with Go 1.19

### Server Endpoints
- Health check: `GET /health`
- API base: `/api/v1/`
- WebSocket: `GET /api/v1/ws`

### Features Implemented
✓ User registration and authentication
✓ Tier-based access control (Free/Premium/Admin)
✓ Chat room management with type restrictions
✓ Real-time messaging via WebSocket
✓ File upload support
✓ Stripe subscription integration
✓ Admin panel for user and analytics management
✓ Rate limiting and input validation
✓ Comprehensive logging and error handling

## Recent Changes

**July 25, 2025**:
- Fixed Go version compatibility (1.21 → 1.19 for Replit)
- Resolved circular import dependencies between handlers and services
- Fixed GORM foreign key relationships for all models
- Corrected Stripe subscription package imports
- Successfully deployed complete chat backend server
- All 23 API endpoints properly configured and running
- Database migrations completed with proper indexing
- Server running on port 5000 with full functionality
- Added root route with comprehensive API documentation
- Analyzed chatusa.com features for enhancement planning

## Enhancement Roadmap (Based on chatusa.com Analysis)

**Current Premium Features vs chatusa.com**:
- ✅ Anonymous guest access (like chatusa.com)
- ✅ User registration with optional email (enhanced vs chatusa.com)
- ✅ Gender/demographic filtering capability
- ✅ Real-time messaging with WebSocket
- ✅ Tier-based access (Premium enhancement)
- ✅ File upload support (Premium enhancement)
- ✅ Admin panel and analytics (Premium enhancement)
- ✅ Stripe payment integration (Premium enhancement)

**Recently Implemented (July 25, 2025)**:
- ✅ Username search functionality (`/api/v1/chat/search/users`)
- ✅ Anonymous guest user creation (`/api/v1/auth/guest`)
- ✅ Enhanced API documentation at root route

**Future Enhancement Opportunities**:
- Sound notification controls for clients
- Enhanced emoji/sticker support
- Improved spam/bot detection algorithms
- Guest user profile customization
- Regional/demographic chat rooms
- Message sound toggles
- Enhanced user filtering options

## Documentation Package (July 25, 2025)

**Complete Documentation Suite Created**:
- ✅ `README.md` - Project overview and quick start guide
- ✅ `docs/API.md` - Complete API reference (25+ endpoints)
- ✅ `docs/SETUP.md` - Installation and configuration guide
- ✅ `docs/EXAMPLES.md` - Usage examples and integrations
- ✅ `docs/DEPLOYMENT.md` - Production deployment guide
- ✅ `docs/SUMMARY.md` - Project summary and achievements
- ✅ `.env.example` - Environment configuration template

## API Endpoints Summary (25+ Total)

**Authentication (5 endpoints)**:
- `POST /api/v1/auth/register` - Full user registration
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/guest` - Anonymous guest user creation ⭐
- `POST /api/v1/auth/refresh` - Token refresh
- `POST /api/v1/auth/logout` - User logout

**Chat Features (8 endpoints)**:
- `GET /api/v1/chat/rooms` - List accessible chat rooms
- `POST /api/v1/chat/rooms` - Create new chat room
- `GET /api/v1/chat/rooms/:id` - Get room details
- `POST /api/v1/chat/rooms/:id/join` - Join room
- `POST /api/v1/chat/rooms/:id/leave` - Leave room
- `GET /api/v1/chat/rooms/:id/messages` - Get messages
- `POST /api/v1/chat/rooms/:id/messages` - Send message
- `GET /api/v1/chat/search/users` - Search users by username ⭐

**Real-time & File Features**:
- `GET /api/v1/ws` - WebSocket connection for real-time chat
- `POST /api/v1/chat/upload` - File upload (premium feature)
- `POST /api/v1/chat/messages/:id/react` - Message reactions

**Premium Features (4 endpoints)**:
- `GET /api/v1/subscription/status` - Subscription status
- `POST /api/v1/subscription/create` - Create subscription
- `POST /api/v1/subscription/cancel` - Cancel subscription
- `POST /api/v1/subscription/webhook` - Stripe webhooks

**Admin Panel (6+ endpoints)**:
- `GET /api/v1/admin/users` - Manage users
- `PUT /api/v1/admin/users/:id/tier` - Update user tier
- `GET /api/v1/admin/analytics` - View analytics
- `GET /api/v1/admin/rooms` - Manage rooms
- `DELETE /api/v1/admin/rooms/:id` - Delete rooms
- `DELETE /api/v1/admin/messages/:id` - Delete messages

**System Endpoints**:
- `GET /` - API documentation and status
- `GET /health` - Health check for monitoring

⭐ = Recently implemented chatusa.com inspired features