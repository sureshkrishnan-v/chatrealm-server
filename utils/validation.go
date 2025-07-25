package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var (
	validate      *validator.Validate
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

func init() {
	validate = validator.New()
	
	// Register custom validators
	validate.RegisterValidation("strongpassword", validateStrongPassword)
	validate.RegisterValidation("username", validateUsername)
	validate.RegisterValidation("roomname", validateRoomName)
}

// FormatValidationError formats validation errors into user-friendly messages
func FormatValidationError(err error) string {
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		var errorMessages []string
		
		for _, fieldError := range validationErrors {
			errorMessages = append(errorMessages, formatFieldError(fieldError))
		}
		
		return strings.Join(errorMessages, "; ")
	}
	
	return err.Error()
}

func formatFieldError(fieldError validator.FieldError) string {
	field := strings.ToLower(fieldError.Field())
	
	switch fieldError.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, fieldError.Param())
	case "max":
		return fmt.Sprintf("%s must not exceed %s characters", field, fieldError.Param())
	case "strongpassword":
		return "password must contain at least 8 characters with uppercase, lowercase, number, and special character"
	case "username":
		return "username can only contain letters, numbers, underscores, and hyphens"
	case "roomname":
		return "room name contains invalid characters"
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, fieldError.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// Custom validators

func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	
	if len(password) < 8 {
		return false
	}
	
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	
	return hasUpper && hasLower && hasNumber && hasSpecial
}

func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	return usernameRegex.MatchString(username)
}

func validateRoomName(fl validator.FieldLevel) bool {
	roomName := fl.Field().String()
	
	// Check for basic requirements
	if len(strings.TrimSpace(roomName)) == 0 {
		return false
	}
	
	// Check for prohibited characters (example: no control characters)
	for _, char := range roomName {
		if unicode.IsControl(char) && char != '\t' && char != '\n' && char != '\r' {
			return false
		}
	}
	
	return true
}

// Validation helper functions

func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

func ValidateUsername(username string) error {
	if len(username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}
	
	if len(username) > 50 {
		return errors.New("username must not exceed 50 characters")
	}
	
	if !usernameRegex.MatchString(username) {
		return errors.New("username can only contain letters, numbers, underscores, and hyphens")
	}
	
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters long")
	}
	
	if len(password) > 128 {
		return errors.New("password must not exceed 128 characters")
	}
	
	return nil
}

func ValidateStrongPassword(password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	
	if len(password) < 8 {
		return errors.New("strong password must be at least 8 characters long")
	}
	
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}
	
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}
	
	return nil
}

func ValidateRoomName(name string) error {
	name = strings.TrimSpace(name)
	
	if len(name) == 0 {
		return errors.New("room name cannot be empty")
	}
	
	if len(name) > 100 {
		return errors.New("room name must not exceed 100 characters")
	}
	
	// Check for prohibited characters
	for _, char := range name {
		if unicode.IsControl(char) {
			return errors.New("room name contains invalid characters")
		}
	}
	
	return nil
}

func ValidateMessageContent(content string) error {
	content = strings.TrimSpace(content)
	
	if len(content) == 0 {
		return errors.New("message content cannot be empty")
	}
	
	if len(content) > 4000 {
		return errors.New("message content must not exceed 4000 characters")
	}
	
	return nil
}

func ValidateFileUpload(filename string, fileSize int64, maxSize int64) error {
	if len(filename) == 0 {
		return errors.New("filename cannot be empty")
	}
	
	if len(filename) > 255 {
		return errors.New("filename too long")
	}
	
	if fileSize <= 0 {
		return errors.New("file cannot be empty")
	}
	
	if fileSize > maxSize {
		return fmt.Errorf("file size (%d bytes) exceeds maximum allowed size (%d bytes)", fileSize, maxSize)
	}
	
	return nil
}

func SanitizeInput(input string) string {
	// Remove control characters except newlines and tabs
	var result strings.Builder
	for _, char := range input {
		if !unicode.IsControl(char) || char == '\n' || char == '\t' || char == '\r' {
			result.WriteRune(char)
		}
	}
	
	return strings.TrimSpace(result.String())
}

func SanitizeHTML(input string) string {
	// Basic HTML sanitization - remove common dangerous tags
	dangerous := []string{
		"<script", "</script>",
		"<iframe", "</iframe>",
		"<object", "</object>",
		"<embed", "</embed>",
		"<form", "</form>",
		"javascript:",
		"vbscript:",
		"data:",
	}
	
	sanitized := input
	for _, danger := range dangerous {
		sanitized = strings.ReplaceAll(strings.ToLower(sanitized), danger, "")
	}
	
	return sanitized
}

func ValidateEmojiReaction(emoji string) error {
	if len(emoji) == 0 {
		return errors.New("emoji cannot be empty")
	}
	
	if len(emoji) > 10 {
		return errors.New("emoji too long")
	}
	
	// Basic emoji validation - check if it contains only emoji characters
	// This is a simple check; a more sophisticated implementation would use
	// a proper emoji library
	if len(strings.TrimSpace(emoji)) == 0 {
		return errors.New("invalid emoji")
	}
	
	return nil
}

func ValidatePaginationParams(page, limit int) error {
	if page < 1 {
		return errors.New("page must be at least 1")
	}
	
	if limit < 1 {
		return errors.New("limit must be at least 1")
	}
	
	if limit > 100 {
		return errors.New("limit cannot exceed 100")
	}
	
	return nil
}

func ValidateUUIDString(uuidStr string) error {
	if len(uuidStr) == 0 {
		return errors.New("UUID cannot be empty")
	}
	
	// Basic UUID format validation
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !uuidRegex.MatchString(uuidStr) {
		return errors.New("invalid UUID format")
	}
	
	return nil
}

// Content moderation helpers

func ContainsProfanity(content string) bool {
	// Basic profanity filter - in production, use a proper content moderation service
	profanity := []string{
		"spam", "scam", "hack", "cheat", // Basic terms
		// Add more as needed
	}
	
	lowerContent := strings.ToLower(content)
	for _, word := range profanity {
		if strings.Contains(lowerContent, word) {
			return true
		}
	}
	
	return false
}

func ValidateContentPolicy(content string) error {
	if ContainsProfanity(content) {
		return errors.New("content violates community guidelines")
	}
	
	// Check for excessive capitalization
	upperCount := 0
	for _, char := range content {
		if unicode.IsUpper(char) {
			upperCount++
		}
	}
	
	if len(content) > 0 && float64(upperCount)/float64(len(content)) > 0.7 {
		return errors.New("excessive use of capital letters")
	}
	
	return nil
}
