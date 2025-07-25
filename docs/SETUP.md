# Premium Chat Backend Setup Guide

## Prerequisites

- Go 1.19 or higher
- PostgreSQL 12+
- Redis 6+ (optional, for session management)
- Stripe account (for payments)

## Environment Variables

Create a `.env` file or set these environment variables:

```bash
# Server Configuration
PORT=5000
ENVIRONMENT=development

# Database
DATABASE_URL=postgres://username:password@localhost:5432/premium_chat?sslmode=disable

# Redis (optional)
REDIS_URL=redis://localhost:6379

# JWT Secret
JWT_SECRET=your-super-secret-jwt-key-here

# Stripe Configuration
STRIPE_SECRET_KEY=sk_test_your_stripe_secret_key
STRIPE_PUBLIC_KEY=pk_test_your_stripe_public_key
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret
```

## Installation

### 1. Clone and Install Dependencies

```bash
git clone <repository-url>
cd premium-chat-backend
go mod tidy
```

### 2. Database Setup

#### PostgreSQL Setup
```sql
-- Create database
CREATE DATABASE premium_chat;

-- Create user (optional)
CREATE USER chat_user WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE premium_chat TO chat_user;
```

#### Redis Setup (Optional)
```bash
# Install Redis
sudo apt-get install redis-server

# Start Redis
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

### 3. Environment Configuration

```bash
# Copy example environment file
cp .env.example .env

# Edit with your values
nano .env
```

### 4. Run Database Migrations

The application automatically runs migrations on startup, but you can also run them manually:

```bash
go run main.go
```

## Running the Application

### Development Mode

```bash
# Run with automatic reload (using air)
go install github.com/cosmtrek/air@latest
air

# Or run directly
go run main.go
```

### Production Mode

```bash
# Build the application
go build -o premium-chat-server main.go

# Run the server
./premium-chat-server
```

## Docker Setup (Optional)

### Dockerfile
```dockerfile
FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
CMD ["./main"]
```

### Docker Compose
```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "5000:5000"
    environment:
      - DATABASE_URL=postgres://postgres:password@db:5432/premium_chat?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=your-jwt-secret
    depends_on:
      - db
      - redis

  db:
    image: postgres:13
    environment:
      POSTGRES_DB: premium_chat
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:6-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
```

## Configuration

### Chat Room Types

- **Public**: Accessible to all users
- **Private**: Accessible to premium users and above
- **VIP**: Accessible to premium users and above (premium room features)

### User Tiers

- **Free**: Basic access, public rooms only
- **Premium**: Full access, file uploads, private rooms
- **Admin**: Full access + administration capabilities

### File Upload Configuration

Configure in `config/config.go`:

```go
type FileUploadConfig struct {
    MaxFileSize    int64  // bytes
    AllowedTypes   []string
    UploadPath     string
    PublicURL      string
}
```

## Stripe Integration

### 1. Create Stripe Products

```bash
# Create products in Stripe Dashboard or via API
curl -X POST https://api.stripe.com/v1/products \
  -u sk_test_...: \
  -d name="Premium Monthly" \
  -d description="Monthly premium subscription"

curl -X POST https://api.stripe.com/v1/prices \
  -u sk_test_...: \
  -d product=prod_... \
  -d unit_amount=999 \
  -d currency=usd \
  -d recurring[interval]=month
```

### 2. Configure Webhooks

Set up webhook endpoint in Stripe Dashboard:
- URL: `https://yourdomain.com/api/v1/subscription/webhook`
- Events: `invoice.payment_succeeded`, `customer.subscription.deleted`, etc.

## Security Considerations

### JWT Configuration
- Use a strong, random JWT secret (minimum 32 characters)
- Set appropriate token expiration times
- Implement token rotation for refresh tokens

### Database Security
- Use environment variables for database credentials
- Enable SSL/TLS for database connections in production
- Regular database backups

### Rate Limiting
Default rate limits are configured in `middleware/rate_limit.go`:
- API requests: 100/minute per IP
- Message sending: 10/minute per user
- File uploads: 5/minute per user

### CORS Configuration
Configure CORS in `middleware/cors.go` for production:

```go
config := cors.Config{
    AllowOrigins: []string{"https://yourdomain.com"},
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
}
```

## Monitoring and Logging

### Structured Logging
The application uses `logrus` for structured logging:

```go
log.WithFields(logrus.Fields{
    "user_id": userID,
    "action": "login",
}).Info("User logged in")
```

### Health Check
Monitor application health via:
```bash
curl http://localhost:5000/health
```

### Metrics Endpoints
- `/health` - Application health status
- `/api/v1/admin/analytics` - Usage analytics (admin only)

## Troubleshooting

### Common Issues

1. **Database Connection Failed**
   ```
   Error: failed to connect to database
   Solution: Check DATABASE_URL and ensure PostgreSQL is running
   ```

2. **JWT Token Invalid**
   ```
   Error: invalid token
   Solution: Check JWT_SECRET configuration and token expiration
   ```

3. **File Upload Failed**
   ```
   Error: file too large
   Solution: Check file size limits and user tier permissions
   ```

4. **WebSocket Connection Failed**
   ```
   Error: websocket upgrade failed
   Solution: Check CORS configuration and authentication headers
   ```

### Debug Mode

Enable debug logging:
```bash
export GIN_MODE=debug
export LOG_LEVEL=debug
go run main.go
```

### Database Debugging

Enable GORM logging:
```go
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
    Logger: logger.Default.LogMode(logger.Info),
})
```

## Performance Optimization

### Database Indexing
Key indexes are automatically created:
- User email/username for login
- Message timestamps for pagination
- Room membership for access control

### Caching Strategy
- Redis for session storage
- In-memory caching for frequently accessed data
- CDN for file uploads (recommended)

### Connection Pooling
Configure database connection pool:
```go
sqlDB, _ := db.DB()
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
sqlDB.SetConnMaxLifetime(time.Hour)
```

## Deployment

### Production Checklist

- [ ] Set `GIN_MODE=release`
- [ ] Configure SSL/TLS certificates
- [ ] Set up reverse proxy (nginx/Apache)
- [ ] Configure database connection pooling
- [ ] Set up monitoring and logging
- [ ] Configure automated backups
- [ ] Set up CI/CD pipeline
- [ ] Configure error tracking (Sentry, etc.)

### Example Nginx Configuration

```nginx
server {
    listen 80;
    server_name yourdomain.com;

    location / {
        proxy_pass http://localhost:5000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /api/v1/ws {
        proxy_pass http://localhost:5000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

## Support

For issues and questions:
1. Check the troubleshooting section
2. Review application logs
3. Check database connectivity
4. Verify environment configuration

## License

This project is licensed under the MIT License.