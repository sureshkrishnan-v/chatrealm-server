# Deployment Guide

## Production Deployment Options

### 1. Traditional Server Deployment

#### Server Requirements
- **CPU**: 2+ cores for moderate load (100+ concurrent users)
- **RAM**: 4GB minimum, 8GB recommended
- **Storage**: 20GB+ SSD
- **OS**: Ubuntu 20.04 LTS, CentOS 8, or similar

#### Setup Steps

1. **Install Dependencies**
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Go
wget https://go.dev/dl/go1.19.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.19.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install PostgreSQL
sudo apt install postgresql postgresql-contrib -y

# Install Redis
sudo apt install redis-server -y

# Install Nginx
sudo apt install nginx -y
```

2. **Configure Database**
```bash
sudo -u postgres createuser --interactive
sudo -u postgres createdb premium_chat
```

3. **Deploy Application**
```bash
# Clone repository
git clone <your-repo> /opt/premium-chat
cd /opt/premium-chat

# Build application
go build -o premium-chat-server main.go

# Create systemd service
sudo tee /etc/systemd/system/premium-chat.service > /dev/null <<EOF
[Unit]
Description=Premium Chat Backend
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/premium-chat
ExecStart=/opt/premium-chat/premium-chat-server
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=premium-chat
KillMode=mixed
KillSignal=SIGINT

Environment=PORT=5000
Environment=DATABASE_URL=postgres://username:password@localhost:5432/premium_chat?sslmode=disable
Environment=JWT_SECRET=your-production-secret
Environment=STRIPE_SECRET_KEY=sk_live_your_key
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl daemon-reload
sudo systemctl enable premium-chat
sudo systemctl start premium-chat
```

4. **Configure Nginx**
```nginx
server {
    listen 80;
    server_name yourdomain.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    # SSL Configuration
    ssl_certificate /path/to/your/certificate.pem;
    ssl_certificate_key /path/to/your/private.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;

    # API routes
    location /api/ {
        proxy_pass http://localhost:5000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # CORS headers
        add_header Access-Control-Allow-Origin "https://yourdomain.com" always;
        add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
        add_header Access-Control-Allow-Headers "Authorization, Content-Type" always;
    }

    # WebSocket support
    location /api/v1/ws {
        proxy_pass http://localhost:5000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        proxy_read_timeout 86400;
    }

    # Health check
    location /health {
        proxy_pass http://localhost:5000;
        access_log off;
    }
}
```

### 2. Docker Deployment

#### Docker Compose (Recommended)

```yaml
version: '3.8'

services:
  app:
    build: 
      context: .
      dockerfile: Dockerfile
    container_name: premium-chat-app
    ports:
      - "5000:5000"
    environment:
      - PORT=5000
      - DATABASE_URL=postgres://postgres:password@db:5432/premium_chat?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=${JWT_SECRET}
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
      - GIN_MODE=release
    depends_on:
      - db
      - redis
    restart: unless-stopped
    volumes:
      - ./uploads:/app/uploads
    networks:
      - premium-chat-network

  db:
    image: postgres:13-alpine
    container_name: premium-chat-db
    environment:
      POSTGRES_DB: premium_chat
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    restart: unless-stopped
    networks:
      - premium-chat-network

  redis:
    image: redis:6-alpine
    container_name: premium-chat-redis
    ports:
      - "6379:6379"
    restart: unless-stopped
    networks:
      - premium-chat-network

  nginx:
    image: nginx:alpine
    container_name: premium-chat-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - app
    restart: unless-stopped
    networks:
      - premium-chat-network

volumes:
  postgres_data:

networks:
  premium-chat-network:
    driver: bridge
```

#### Multi-stage Dockerfile

```dockerfile
# Build stage
FROM golang:1.19-alpine AS builder

WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Create uploads directory
RUN mkdir -p uploads

# Expose port
EXPOSE 5000

# Command to run
CMD ["./main"]
```

### 3. Cloud Platform Deployment

#### Heroku Deployment

1. **Create Heroku App**
```bash
heroku create premium-chat-backend
```

2. **Add PostgreSQL**
```bash
heroku addons:create heroku-postgresql:hobby-dev
```

3. **Add Redis**
```bash
heroku addons:create heroku-redis:hobby-dev
```

4. **Configure Environment Variables**
```bash
heroku config:set JWT_SECRET=your-secret-key
heroku config:set STRIPE_SECRET_KEY=sk_live_your_key
heroku config:set GIN_MODE=release
```

5. **Deploy**
```bash
git push heroku main
```

#### AWS ECS Deployment

```yaml
# task-definition.json
{
  "family": "premium-chat-backend",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::account:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "premium-chat-app",
      "image": "your-registry/premium-chat:latest",
      "portMappings": [
        {
          "containerPort": 5000,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "PORT",
          "value": "5000"
        },
        {
          "name": "GIN_MODE",
          "value": "release"
        }
      ],
      "secrets": [
        {
          "name": "DATABASE_URL",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:db-url"
        },
        {
          "name": "JWT_SECRET",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:jwt-secret"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/premium-chat",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "essential": true
    }
  ]
}
```

#### Google Cloud Run

```yaml
# cloudbuild.yaml
steps:
  - name: 'gcr.io/cloud-builders/docker'
    args: ['build', '-t', 'gcr.io/$PROJECT_ID/premium-chat:$BUILD_ID', '.']
  - name: 'gcr.io/cloud-builders/docker'
    args: ['push', 'gcr.io/$PROJECT_ID/premium-chat:$BUILD_ID']
  - name: 'gcr.io/cloud-builders/gcloud'
    args:
      - 'run'
      - 'deploy'
      - 'premium-chat-backend'
      - '--image=gcr.io/$PROJECT_ID/premium-chat:$BUILD_ID'
      - '--platform=managed'
      - '--region=us-central1'
      - '--allow-unauthenticated'
      - '--set-env-vars=GIN_MODE=release'
```

### 4. Kubernetes Deployment

#### Kubernetes Manifests

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: premium-chat

---
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: premium-chat-config
  namespace: premium-chat
data:
  PORT: "5000"
  GIN_MODE: "release"

---
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: premium-chat-secrets
  namespace: premium-chat
type: Opaque
stringData:
  DATABASE_URL: "postgres://user:pass@db:5432/premium_chat"
  JWT_SECRET: "your-jwt-secret"
  STRIPE_SECRET_KEY: "sk_live_your_key"

---
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: premium-chat-backend
  namespace: premium-chat
spec:
  replicas: 3
  selector:
    matchLabels:
      app: premium-chat-backend
  template:
    metadata:
      labels:
        app: premium-chat-backend
    spec:
      containers:
      - name: premium-chat
        image: your-registry/premium-chat:latest
        ports:
        - containerPort: 5000
        envFrom:
        - configMapRef:
            name: premium-chat-config
        - secretRef:
            name: premium-chat-secrets
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 5000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 5000
          initialDelaySeconds: 5
          periodSeconds: 5

---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: premium-chat-service
  namespace: premium-chat
spec:
  selector:
    app: premium-chat-backend
  ports:
  - port: 80
    targetPort: 5000
  type: LoadBalancer

---
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: premium-chat-ingress
  namespace: premium-chat
  annotations:
    kubernetes.io/ingress.class: "nginx"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  tls:
  - hosts:
    - api.yourdomain.com
    secretName: premium-chat-tls
  rules:
  - host: api.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: premium-chat-service
            port:
              number: 80
```

## Database Setup for Production

### PostgreSQL Configuration

```sql
-- Create production database
CREATE DATABASE premium_chat_prod;

-- Create dedicated user
CREATE USER premium_chat_user WITH PASSWORD 'secure_random_password';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE premium_chat_prod TO premium_chat_user;

-- Connect to database
\c premium_chat_prod;

-- Grant schema privileges
GRANT ALL ON SCHEMA public TO premium_chat_user;
```

### Managed Database Services

#### AWS RDS PostgreSQL
```bash
aws rds create-db-instance \
  --db-instance-identifier premium-chat-db \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --master-username premium_chat_user \
  --master-user-password secure_password \
  --allocated-storage 20 \
  --vpc-security-group-ids sg-xxxxxx \
  --db-subnet-group-name premium-chat-subnet-group
```

#### Google Cloud SQL
```bash
gcloud sql instances create premium-chat-db \
  --database-version=POSTGRES_13 \
  --tier=db-f1-micro \
  --region=us-central1
```

## Monitoring and Logging

### Application Metrics

```go
// Add to main.go for Prometheus metrics
import (
    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Add metrics endpoint
router.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

### Health Checks

```bash
# Create health check script
#!/bin/bash
curl -f http://localhost:5000/health || exit 1
```

### Log Aggregation

```yaml
# For ELK Stack
version: '3.8'
services:
  filebeat:
    image: docker.elastic.co/beats/filebeat:7.15.0
    volumes:
      - ./filebeat.yml:/usr/share/filebeat/filebeat.yml
      - /var/log:/var/log:ro
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

## Security Hardening

### SSL/TLS Configuration

```bash
# Generate Let's Encrypt certificate
sudo certbot --nginx -d yourdomain.com
```

### Firewall Rules

```bash
# UFW configuration
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw deny 5000/tcp   # Block direct access to app
sudo ufw enable
```

### Environment Security

```bash
# Create secure environment file
sudo tee /etc/premium-chat/.env > /dev/null <<EOF
JWT_SECRET=$(openssl rand -base64 32)
DATABASE_PASSWORD=$(openssl rand -base64 32)
STRIPE_SECRET_KEY=sk_live_your_actual_key
EOF

sudo chmod 600 /etc/premium-chat/.env
sudo chown premium-chat:premium-chat /etc/premium-chat/.env
```

## Backup and Recovery

### Database Backup

```bash
#!/bin/bash
# backup-database.sh
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="premium_chat_backup_$DATE.sql"

pg_dump -h localhost -U premium_chat_user -d premium_chat_prod > $BACKUP_FILE
gzip $BACKUP_FILE

# Upload to S3 (optional)
aws s3 cp $BACKUP_FILE.gz s3://your-backup-bucket/database/
```

### Automated Backups

```bash
# Add to crontab
0 2 * * * /usr/local/bin/backup-database.sh
```

## Performance Optimization

### Database Optimization

```sql
-- Create performance indexes
CREATE INDEX CONCURRENTLY idx_messages_room_created 
ON messages(chat_room_id, created_at DESC);

CREATE INDEX CONCURRENTLY idx_users_tier 
ON users(tier) WHERE is_active = true;

-- Analyze tables
ANALYZE messages;
ANALYZE users;
ANALYZE chat_rooms;
```

### Connection Pooling

```go
// Configure in database/database.go
sqlDB, _ := db.DB()
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
sqlDB.SetConnMaxLifetime(time.Hour)
```

### Caching Strategy

```go
// Redis configuration
redisClient := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     10,
    MinIdleConns: 5,
})
```

## Troubleshooting

### Common Production Issues

1. **High Memory Usage**
   ```bash
   # Monitor memory
   top -p $(pgrep premium-chat)
   
   # Check for memory leaks
   go tool pprof http://localhost:5000/debug/pprof/heap
   ```

2. **Database Connection Issues**
   ```bash
   # Check connections
   sudo -u postgres psql -c "SELECT * FROM pg_stat_activity;"
   
   # Kill hanging connections
   sudo -u postgres psql -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle in transaction';"
   ```

3. **WebSocket Issues**
   ```bash
   # Check nginx WebSocket configuration
   nginx -t
   
   # Monitor WebSocket connections
   netstat -an | grep :5000
   ```

### Log Analysis

```bash
# Application logs
journalctl -u premium-chat -f

# Nginx logs
tail -f /var/log/nginx/access.log
tail -f /var/log/nginx/error.log

# Database logs
tail -f /var/log/postgresql/postgresql-13-main.log
```

## Maintenance

### Updates and Patches

```bash
#!/bin/bash
# update-app.sh
git pull origin main
go build -o premium-chat-server main.go
sudo systemctl restart premium-chat
sudo systemctl status premium-chat
```

### Database Maintenance

```sql
-- Regular maintenance
VACUUM ANALYZE;
REINDEX DATABASE premium_chat_prod;

-- Check database size
SELECT pg_size_pretty(pg_database_size('premium_chat_prod'));
```

This deployment guide covers all major deployment scenarios and production considerations for the Premium Chat Backend.