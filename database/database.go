package database

import (
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Initialize(databaseURL string) (*gorm.DB, error) {
	// If DATABASE_URL is not provided, construct from individual components
	if databaseURL == "" {
		host := os.Getenv("PGHOST")
		if host == "" {
			host = "localhost"
		}
		
		port := os.Getenv("PGPORT")
		if port == "" {
			port = "5432"
		}
		
		user := os.Getenv("PGUSER")
		if user == "" {
			user = "postgres"
		}
		
		password := os.Getenv("PGPASSWORD")
		if password == "" {
			password = "password"
		}
		
		dbname := os.Getenv("PGDATABASE")
		if dbname == "" {
			dbname = "premium_chat"
		}
		
		databaseURL = "host=" + host + " user=" + user + " password=" + password + " dbname=" + dbname + " port=" + port + " sslmode=disable TimeZone=UTC"
	}

	// Configure GORM logger
	var gormLogger logger.Interface
	if os.Getenv("ENVIRONMENT") == "development" {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	// Open database connection
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger:         gormLogger,
		NowFunc:        func() time.Time { return time.Now().UTC() },
		PrepareStmt:    true,
		CreateBatchSize: 1000,
	})

	if err != nil {
		return nil, err
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func InitRedis(redisURL string) *redis.Client {
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	// Parse Redis URL
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		// Fallback to default settings
		opt = &redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}
	}

	// Configure Redis client options
	opt.PoolSize = 50
	opt.MinIdleConns = 10
	opt.MaxConnAge = time.Hour
	opt.PoolTimeout = time.Second * 30
	opt.IdleTimeout = time.Minute * 5
	opt.IdleCheckFrequency = time.Minute

	client := redis.NewClient(opt)

	return client
}

func TestConnection(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	
	return sqlDB.Ping()
}

func TestRedisConnection(client *redis.Client) error {
	ctx := client.Context()
	return client.Ping(ctx).Err()
}

func CloseConnections(db *gorm.DB, redisClient *redis.Client) error {
	// Close Redis connection
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			return err
		}
	}

	// Close database connection
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}

	return nil
}
