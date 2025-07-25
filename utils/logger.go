package utils

import (
	"io"
	"os"
	"path/filepath"
	"premium-chat-backend/models"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

func InitLogger() {
	Logger = logrus.New()
	
	// Set log level based on environment
	environment := os.Getenv("ENVIRONMENT")
	switch environment {
	case "development":
		Logger.SetLevel(logrus.DebugLevel)
	case "production":
		Logger.SetLevel(logrus.InfoLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}
	
	// Set formatter
	if environment == "production" {
		Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}
	
	// Set output
	if environment == "production" {
		// Create logs directory if it doesn't exist
		logsDir := "logs"
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			Logger.Warn("Failed to create logs directory:", err)
		} else {
			// Create log file
			logFile := filepath.Join(logsDir, "app.log")
			file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				Logger.Warn("Failed to open log file:", err)
			} else {
				// Write to both file and stdout
				multiWriter := io.MultiWriter(os.Stdout, file)
				Logger.SetOutput(multiWriter)
			}
		}
	} else {
		Logger.SetOutput(os.Stdout)
	}
	
	Logger.Info("Logger initialized successfully")
}

// Structured logging helpers

func LogUserAction(userID uuid.UUID, tier models.UserTier, action string, details map[string]interface{}) {
	Logger.WithFields(logrus.Fields{
		"user_id": userID,
		"tier":    tier,
		"action":  action,
		"details": details,
	}).Info("User action logged")
}

func LogAuthEvent(userID uuid.UUID, event string, success bool, details map[string]interface{}) {
	level := logrus.InfoLevel
	if !success {
		level = logrus.WarnLevel
	}
	
	Logger.WithFields(logrus.Fields{
		"user_id": userID,
		"event":   event,
		"success": success,
		"details": details,
	}).Log(level, "Authentication event")
}

func LogChatEvent(userID uuid.UUID, roomID uuid.UUID, event string, details map[string]interface{}) {
	Logger.WithFields(logrus.Fields{
		"user_id": userID,
		"room_id": roomID,
		"event":   event,
		"details": details,
	}).Info("Chat event logged")
}

func LogSubscriptionEvent(userID uuid.UUID, event string, plan string, details map[string]interface{}) {
	Logger.WithFields(logrus.Fields{
		"user_id": userID,
		"event":   event,
		"plan":    plan,
		"details": details,
	}).Info("Subscription event logged")
}

func LogWebSocketEvent(userID uuid.UUID, event string, details map[string]interface{}) {
	Logger.WithFields(logrus.Fields{
		"user_id": userID,
		"event":   event,
		"details": details,
	}).Debug("WebSocket event logged")
}

func LogDatabaseEvent(operation string, table string, recordID interface{}, duration time.Duration, err error) {
	fields := logrus.Fields{
		"operation": operation,
		"table":     table,
		"record_id": recordID,
		"duration":  duration,
	}
	
	if err != nil {
		fields["error"] = err.Error()
		Logger.WithFields(fields).Error("Database operation failed")
	} else {
		Logger.WithFields(fields).Debug("Database operation completed")
	}
}

func LogAPIRequest(method string, path string, userID *uuid.UUID, userTier *models.UserTier, statusCode int, duration time.Duration) {
	fields := logrus.Fields{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration":    duration,
	}
	
	if userID != nil {
		fields["user_id"] = *userID
	}
	
	if userTier != nil {
		fields["user_tier"] = *userTier
	}
	
	level := logrus.InfoLevel
	if statusCode >= 400 {
		level = logrus.WarnLevel
	}
	if statusCode >= 500 {
		level = logrus.ErrorLevel
	}
	
	Logger.WithFields(fields).Log(level, "API request processed")
}

func LogRateLimitEvent(userID uuid.UUID, tier models.UserTier, endpoint string, exceeded bool) {
	level := logrus.InfoLevel
	if exceeded {
		level = logrus.WarnLevel
	}
	
	Logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"tier":     tier,
		"endpoint": endpoint,
		"exceeded": exceeded,
	}).Log(level, "Rate limit event")
}

func LogFileUpload(userID uuid.UUID, filename string, fileSize int64, success bool, err error) {
	fields := logrus.Fields{
		"user_id":   userID,
		"filename":  filename,
		"file_size": fileSize,
		"success":   success,
	}
	
	if err != nil {
		fields["error"] = err.Error()
		Logger.WithFields(fields).Error("File upload failed")
	} else {
		Logger.WithFields(fields).Info("File upload completed")
	}
}

func LogSecurityEvent(event string, userID *uuid.UUID, ip string, details map[string]interface{}) {
	fields := logrus.Fields{
		"event":   event,
		"ip":      ip,
		"details": details,
	}
	
	if userID != nil {
		fields["user_id"] = *userID
	}
	
	Logger.WithFields(fields).Warn("Security event detected")
}

func LogPaymentEvent(userID uuid.UUID, event string, amount int64, currency string, success bool, details map[string]interface{}) {
	fields := logrus.Fields{
		"user_id":  userID,
		"event":    event,
		"amount":   amount,
		"currency": currency,
		"success":  success,
		"details":  details,
	}
	
	level := logrus.InfoLevel
	if !success {
		level = logrus.ErrorLevel
	}
	
	Logger.WithFields(fields).Log(level, "Payment event processed")
}

func LogSystemEvent(event string, details map[string]interface{}) {
	Logger.WithFields(logrus.Fields{
		"event":   event,
		"details": details,
	}).Info("System event logged")
}

func LogPerformanceMetric(operation string, duration time.Duration, details map[string]interface{}) {
	Logger.WithFields(logrus.Fields{
		"operation": operation,
		"duration":  duration,
		"details":   details,
	}).Debug("Performance metric logged")
}

// Error logging helpers

func LogError(err error, context string, fields map[string]interface{}) {
	logFields := logrus.Fields{
		"error":   err.Error(),
		"context": context,
	}
	
	// Merge additional fields
	for key, value := range fields {
		logFields[key] = value
	}
	
	Logger.WithFields(logFields).Error("Error occurred")
}

func LogCriticalError(err error, context string, fields map[string]interface{}) {
	logFields := logrus.Fields{
		"error":   err.Error(),
		"context": context,
	}
	
	// Merge additional fields
	for key, value := range fields {
		logFields[key] = value
	}
	
	Logger.WithFields(logFields).Fatal("Critical error occurred")
}

func LogWarning(message string, fields map[string]interface{}) {
	Logger.WithFields(fields).Warn(message)
}

func LogInfo(message string, fields map[string]interface{}) {
	Logger.WithFields(fields).Info(message)
}

func LogDebug(message string, fields map[string]interface{}) {
	Logger.WithFields(fields).Debug(message)
}

// Business logic logging

func LogUserRegistration(userID uuid.UUID, email string, tier models.UserTier) {
	LogUserAction(userID, tier, "registration", map[string]interface{}{
		"email": email,
	})
}

func LogUserLogin(userID uuid.UUID, tier models.UserTier, ip string) {
	LogAuthEvent(userID, "login", true, map[string]interface{}{
		"ip": ip,
	})
}

func LogUserLogout(userID uuid.UUID, tier models.UserTier) {
	LogAuthEvent(userID, "logout", true, map[string]interface{}{})
}

func LogRoomCreation(userID uuid.UUID, roomID uuid.UUID, roomType models.ChatRoomType, roomName string) {
	LogChatEvent(userID, roomID, "room_created", map[string]interface{}{
		"room_type": roomType,
		"room_name": roomName,
	})
}

func LogRoomJoin(userID uuid.UUID, roomID uuid.UUID, roomType models.ChatRoomType) {
	LogChatEvent(userID, roomID, "room_joined", map[string]interface{}{
		"room_type": roomType,
	})
}

func LogRoomLeave(userID uuid.UUID, roomID uuid.UUID, roomType models.ChatRoomType) {
	LogChatEvent(userID, roomID, "room_left", map[string]interface{}{
		"room_type": roomType,
	})
}

func LogMessageSent(userID uuid.UUID, roomID uuid.UUID, messageID uuid.UUID, messageType models.MessageType) {
	LogChatEvent(userID, roomID, "message_sent", map[string]interface{}{
		"message_id":   messageID,
		"message_type": messageType,
	})
}

func LogSubscriptionCreated(userID uuid.UUID, plan models.SubscriptionPlan) {
	LogSubscriptionEvent(userID, "subscription_created", string(plan), map[string]interface{}{})
}

func LogSubscriptionCanceled(userID uuid.UUID, plan models.SubscriptionPlan) {
	LogSubscriptionEvent(userID, "subscription_canceled", string(plan), map[string]interface{}{})
}

func LogWebSocketConnection(userID uuid.UUID, tier models.UserTier) {
	LogWebSocketEvent(userID, "connection_established", map[string]interface{}{
		"tier": tier,
	})
}

func LogWebSocketDisconnection(userID uuid.UUID, tier models.UserTier, duration time.Duration) {
	LogWebSocketEvent(userID, "connection_closed", map[string]interface{}{
		"tier":     tier,
		"duration": duration,
	})
}
