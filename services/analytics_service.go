package services

import (
	"context"
	"encoding/json"
	"premium-chat-backend/models"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AnalyticsService struct {
	db          *gorm.DB
	redisClient *redis.Client
}

type SystemAnalytics struct {
	TotalUsers       int64             `json:"total_users"`
	ActiveUsers      int64             `json:"active_users"`
	PremiumUsers     int64             `json:"premium_users"`
	FreeUsers        int64             `json:"free_users"`
	TotalRooms       int64             `json:"total_rooms"`
	TotalMessages    int64             `json:"total_messages"`
	MessagesToday    int64             `json:"messages_today"`
	RevenueThisMonth float64           `json:"revenue_this_month"`
	ConversionRate   float64           `json:"conversion_rate"`
	UsersByTier      map[string]int64  `json:"users_by_tier"`
	RoomsByType      map[string]int64  `json:"rooms_by_type"`
	DailyStats       []DailyStats      `json:"daily_stats"`
	TopRooms         []RoomStats       `json:"top_rooms"`
	SubscriptionStats SubscriptionStats `json:"subscription_stats"`
}

type DailyStats struct {
	Date          string `json:"date"`
	NewUsers      int64  `json:"new_users"`
	ActiveUsers   int64  `json:"active_users"`
	Messages      int64  `json:"messages"`
	NewPremium    int64  `json:"new_premium"`
	Revenue       float64 `json:"revenue"`
}

type RoomStats struct {
	RoomID      uuid.UUID `json:"room_id"`
	RoomName    string    `json:"room_name"`
	RoomType    string    `json:"room_type"`
	MemberCount int64     `json:"member_count"`
	MessageCount int64    `json:"message_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserStats struct {
	UserID           uuid.UUID `json:"user_id"`
	Username         string    `json:"username"`
	Tier             string    `json:"tier"`
	MessagesCount    int64     `json:"messages_count"`
	RoomsJoined      int64     `json:"rooms_joined"`
	LastActive       *time.Time `json:"last_active"`
	SubscriptionDays int       `json:"subscription_days"`
	TotalRevenue     float64   `json:"total_revenue"`
	JoinDate         time.Time `json:"join_date"`
}

type SubscriptionStats struct {
	ActiveSubscriptions   int64   `json:"active_subscriptions"`
	CanceledSubscriptions int64   `json:"canceled_subscriptions"`
	MonthlyRevenue       float64 `json:"monthly_revenue"`
	YearlyRevenue        float64 `json:"yearly_revenue"`
	ChurnRate            float64 `json:"churn_rate"`
	AverageLifetime      float64 `json:"average_lifetime"`
}

func NewAnalyticsService(db *gorm.DB, redisClient *redis.Client) *AnalyticsService {
	return &AnalyticsService{
		db:          db,
		redisClient: redisClient,
	}
}

func (s *AnalyticsService) GetSystemAnalytics() (*SystemAnalytics, error) {
	analytics := &SystemAnalytics{
		UsersByTier: make(map[string]int64),
		RoomsByType: make(map[string]int64),
	}

	// Get total users
	s.db.Model(&models.User{}).Count(&analytics.TotalUsers)

	// Get active users (last 24 hours)
	s.db.Model(&models.User{}).
		Where("last_seen_at > ?", time.Now().Add(-24*time.Hour)).
		Count(&analytics.ActiveUsers)

	// Get users by tier
	var tierCounts []struct {
		Tier  string `json:"tier"`
		Count int64  `json:"count"`
	}
	s.db.Model(&models.User{}).
		Select("tier, count(*) as count").
		Group("tier").
		Scan(&tierCounts)

	for _, tier := range tierCounts {
		analytics.UsersByTier[tier.Tier] = tier.Count
		if tier.Tier == string(models.TierPremium) {
			analytics.PremiumUsers = tier.Count
		} else if tier.Tier == string(models.TierFree) {
			analytics.FreeUsers = tier.Count
		}
	}

	// Get total rooms
	s.db.Model(&models.ChatRoom{}).Count(&analytics.TotalRooms)

	// Get rooms by type
	var roomTypeCounts []struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	s.db.Model(&models.ChatRoom{}).
		Select("type, count(*) as count").
		Group("type").
		Scan(&roomTypeCounts)

	for _, roomType := range roomTypeCounts {
		analytics.RoomsByType[roomType.Type] = roomType.Count
	}

	// Get total messages
	s.db.Model(&models.Message{}).Count(&analytics.TotalMessages)

	// Get messages today
	today := time.Now().Truncate(24 * time.Hour)
	s.db.Model(&models.Message{}).
		Where("created_at >= ?", today).
		Count(&analytics.MessagesToday)

	// Get revenue this month
	firstOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	var monthlyRevenue struct {
		Total float64 `json:"total"`
	}
	s.db.Model(&models.BillingHistory{}).
		Select("SUM(amount::float / 100) as total").
		Where("paid_at >= ? AND status = 'paid'", firstOfMonth).
		Scan(&monthlyRevenue)
	analytics.RevenueThisMonth = monthlyRevenue.Total

	// Calculate conversion rate
	if analytics.TotalUsers > 0 {
		analytics.ConversionRate = float64(analytics.PremiumUsers) / float64(analytics.TotalUsers) * 100
	}

	// Get daily stats for the last 7 days
	dailyStats, err := s.getDailyStats(7)
	if err == nil {
		analytics.DailyStats = dailyStats
	}

	// Get top rooms
	topRooms, err := s.getTopRooms(10)
	if err == nil {
		analytics.TopRooms = topRooms
	}

	// Get subscription stats
	subscriptionStats, err := s.getSubscriptionStats()
	if err == nil {
		analytics.SubscriptionStats = *subscriptionStats
	}

	return analytics, nil
}

func (s *AnalyticsService) getDailyStats(days int) ([]DailyStats, error) {
	var stats []DailyStats
	
	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Truncate(24 * time.Hour)
		dateEnd := date.Add(24 * time.Hour)
		dateStr := date.Format("2006-01-02")

		var dailyStat DailyStats
		dailyStat.Date = dateStr

		// New users
		s.db.Model(&models.User{}).
			Where("created_at >= ? AND created_at < ?", date, dateEnd).
			Count(&dailyStat.NewUsers)

		// Active users
		s.db.Model(&models.User{}).
			Where("last_seen_at >= ? AND last_seen_at < ?", date, dateEnd).
			Count(&dailyStat.ActiveUsers)

		// Messages
		s.db.Model(&models.Message{}).
			Where("created_at >= ? AND created_at < ?", date, dateEnd).
			Count(&dailyStat.Messages)

		// New premium subscriptions
		s.db.Model(&models.Subscription{}).
			Where("created_at >= ? AND created_at < ? AND status = 'active'", date, dateEnd).
			Count(&dailyStat.NewPremium)

		// Revenue
		var revenue struct {
			Total float64 `json:"total"`
		}
		s.db.Model(&models.BillingHistory{}).
			Select("SUM(amount::float / 100) as total").
			Where("paid_at >= ? AND paid_at < ? AND status = 'paid'", date, dateEnd).
			Scan(&revenue)
		dailyStat.Revenue = revenue.Total

		stats = append(stats, dailyStat)
	}

	return stats, nil
}

func (s *AnalyticsService) getTopRooms(limit int) ([]RoomStats, error) {
	var rooms []RoomStats

	rows, err := s.db.Raw(`
		SELECT 
			cr.id as room_id,
			cr.name as room_name,
			cr.type as room_type,
			COUNT(DISTINCT crm.user_id) as member_count,
			COUNT(DISTINCT m.id) as message_count,
			cr.created_at
		FROM chat_rooms cr
		LEFT JOIN chat_room_members crm ON cr.id = crm.chat_room_id
		LEFT JOIN messages m ON cr.id = m.chat_room_id
		WHERE cr.deleted_at IS NULL
		GROUP BY cr.id, cr.name, cr.type, cr.created_at
		ORDER BY message_count DESC, member_count DESC
		LIMIT ?
	`, limit).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var room RoomStats
		if err := rows.Scan(
			&room.RoomID,
			&room.RoomName,
			&room.RoomType,
			&room.MemberCount,
			&room.MessageCount,
			&room.CreatedAt,
		); err != nil {
			continue
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

func (s *AnalyticsService) getSubscriptionStats() (*SubscriptionStats, error) {
	stats := &SubscriptionStats{}

	// Active subscriptions
	s.db.Model(&models.Subscription{}).
		Where("status = 'active'").
		Count(&stats.ActiveSubscriptions)

	// Canceled subscriptions
	s.db.Model(&models.Subscription{}).
		Where("status = 'canceled'").
		Count(&stats.CanceledSubscriptions)

	// Monthly revenue (current month)
	firstOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	var monthlyRevenue struct {
		Total float64 `json:"total"`
	}
	s.db.Model(&models.BillingHistory{}).
		Select("SUM(amount::float / 100) as total").
		Where("paid_at >= ? AND status = 'paid'", firstOfMonth).
		Scan(&monthlyRevenue)
	stats.MonthlyRevenue = monthlyRevenue.Total

	// Yearly revenue
	firstOfYear := time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	var yearlyRevenue struct {
		Total float64 `json:"total"`
	}
	s.db.Model(&models.BillingHistory{}).
		Select("SUM(amount::float / 100) as total").
		Where("paid_at >= ? AND status = 'paid'", firstOfYear).
		Scan(&yearlyRevenue)
	stats.YearlyRevenue = yearlyRevenue.Total

	// Calculate churn rate (canceled in last 30 days / total active at start of period)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	var canceledLast30Days int64
	s.db.Model(&models.Subscription{}).
		Where("canceled_at >= ? AND canceled_at IS NOT NULL", thirtyDaysAgo).
		Count(&canceledLast30Days)

	totalActive := stats.ActiveSubscriptions + canceledLast30Days
	if totalActive > 0 {
		stats.ChurnRate = float64(canceledLast30Days) / float64(totalActive) * 100
	}

	// Average subscription lifetime
	var avgLifetime struct {
		Days float64 `json:"days"`
	}
	s.db.Model(&models.Subscription{}).
		Select("AVG(EXTRACT(epoch FROM (COALESCE(canceled_at, NOW()) - created_at))/86400) as days").
		Where("status IN ('active', 'canceled')").
		Scan(&avgLifetime)
	stats.AverageLifetime = avgLifetime.Days

	return stats, nil
}

func (s *AnalyticsService) GetUserStats(userID uuid.UUID) (*UserStats, error) {
	var user models.User
	if err := s.db.Preload("Subscription").First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}

	stats := &UserStats{
		UserID:   user.ID,
		Username: user.Username,
		Tier:     string(user.Tier),
		JoinDate: user.CreatedAt,
	}

	if user.LastSeenAt != nil {
		stats.LastActive = user.LastSeenAt
	}

	// Count messages
	s.db.Model(&models.Message{}).
		Where("user_id = ?", userID).
		Count(&stats.MessagesCount)

	// Count rooms joined
	s.db.Model(&models.ChatRoomMember{}).
		Where("user_id = ?", userID).
		Count(&stats.RoomsJoined)

	// Subscription info
	if user.Subscription != nil {
		stats.SubscriptionDays = int(time.Since(user.Subscription.CreatedAt).Hours() / 24)

		// Calculate total revenue from this user
		var revenue struct {
			Total float64 `json:"total"`
		}
		s.db.Model(&models.BillingHistory{}).
			Select("SUM(amount::float / 100) as total").
			Where("user_id = ? AND status = 'paid'", userID).
			Scan(&revenue)
		stats.TotalRevenue = revenue.Total
	}

	return stats, nil
}

func (s *AnalyticsService) TrackEvent(eventType, userID string, data map[string]interface{}) error {
	ctx := context.Background()
	
	event := map[string]interface{}{
		"type":       eventType,
		"user_id":    userID,
		"data":       data,
		"timestamp":  time.Now(),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Store in Redis for real-time analytics
	key := "analytics:events:" + time.Now().Format("2006-01-02")
	return s.redisClient.LPush(ctx, key, eventJSON).Err()
}

func (s *AnalyticsService) TrackUserAction(userID uuid.UUID, action string, metadata map[string]interface{}) error {
	return s.TrackEvent("user_action", userID.String(), map[string]interface{}{
		"action":   action,
		"metadata": metadata,
	})
}

func (s *AnalyticsService) TrackSubscriptionEvent(userID uuid.UUID, event string, plan string) error {
	return s.TrackEvent("subscription", userID.String(), map[string]interface{}{
		"event": event,
		"plan":  plan,
	})
}

func (s *AnalyticsService) TrackMessageSent(userID uuid.UUID, roomID uuid.UUID, messageType string) error {
	return s.TrackEvent("message_sent", userID.String(), map[string]interface{}{
		"room_id":      roomID.String(),
		"message_type": messageType,
	})
}

func (s *AnalyticsService) GetRealtimeMetrics() (map[string]interface{}, error) {
	ctx := context.Background()
	
	metrics := make(map[string]interface{})
	
	// Get online users count from WebSocket service
	onlineUsersKey := "user_status:*"
	keys, err := s.redisClient.Keys(ctx, onlineUsersKey).Result()
	if err == nil {
		onlineCount := 0
		for _, key := range keys {
			status, err := s.redisClient.Get(ctx, key).Result()
			if err == nil && status != "" {
				var userStatus map[string]interface{}
				if json.Unmarshal([]byte(status), &userStatus) == nil {
					if userStatus["status"] == "online" {
						onlineCount++
					}
				}
			}
		}
		metrics["online_users"] = onlineCount
	}

	// Get today's events count
	todayKey := "analytics:events:" + time.Now().Format("2006-01-02")
	eventsCount, err := s.redisClient.LLen(ctx, todayKey).Result()
	if err == nil {
		metrics["events_today"] = eventsCount
	}

	// Add more real-time metrics as needed
	metrics["timestamp"] = time.Now()

	return metrics, nil
}

func (s *AnalyticsService) GenerateReport(reportType string, startDate, endDate time.Time) (interface{}, error) {
	switch reportType {
	case "user_growth":
		return s.generateUserGrowthReport(startDate, endDate)
	case "revenue":
		return s.generateRevenueReport(startDate, endDate)
	case "engagement":
		return s.generateEngagementReport(startDate, endDate)
	default:
		return nil, gorm.ErrRecordNotFound
	}
}

func (s *AnalyticsService) generateUserGrowthReport(startDate, endDate time.Time) (interface{}, error) {
	type UserGrowthData struct {
		Date      string `json:"date"`
		NewUsers  int64  `json:"new_users"`
		TotalUsers int64 `json:"total_users"`
		PremiumUsers int64 `json:"premium_users"`
	}

	var report []UserGrowthData
	current := startDate

	for current.Before(endDate) || current.Equal(endDate) {
		nextDay := current.Add(24 * time.Hour)
		
		var newUsers, totalUsers, premiumUsers int64
		
		// New users for this day
		s.db.Model(&models.User{}).
			Where("created_at >= ? AND created_at < ?", current, nextDay).
			Count(&newUsers)

		// Total users up to this day
		s.db.Model(&models.User{}).
			Where("created_at <= ?", nextDay).
			Count(&totalUsers)

		// Premium users up to this day
		s.db.Model(&models.User{}).
			Where("created_at <= ? AND tier = 'premium'", nextDay).
			Count(&premiumUsers)

		report = append(report, UserGrowthData{
			Date:         current.Format("2006-01-02"),
			NewUsers:     newUsers,
			TotalUsers:   totalUsers,
			PremiumUsers: premiumUsers,
		})

		current = nextDay
	}

	return report, nil
}

func (s *AnalyticsService) generateRevenueReport(startDate, endDate time.Time) (interface{}, error) {
	type RevenueData struct {
		Date     string  `json:"date"`
		Revenue  float64 `json:"revenue"`
		Orders   int64   `json:"orders"`
	}

	var report []RevenueData
	current := startDate

	for current.Before(endDate) || current.Equal(endDate) {
		nextDay := current.Add(24 * time.Hour)
		
		var orders int64
		var revenue struct {
			Total float64 `json:"total"`
		}
		
		// Orders for this day
		s.db.Model(&models.BillingHistory{}).
			Where("paid_at >= ? AND paid_at < ? AND status = 'paid'", current, nextDay).
			Count(&orders)

		// Revenue for this day
		s.db.Model(&models.BillingHistory{}).
			Select("SUM(amount::float / 100) as total").
			Where("paid_at >= ? AND paid_at < ? AND status = 'paid'", current, nextDay).
			Scan(&revenue)

		report = append(report, RevenueData{
			Date:    current.Format("2006-01-02"),
			Revenue: revenue.Total,
			Orders:  orders,
		})

		current = nextDay
	}

	return report, nil
}

func (s *AnalyticsService) generateEngagementReport(startDate, endDate time.Time) (interface{}, error) {
	type EngagementData struct {
		Date          string `json:"date"`
		ActiveUsers   int64  `json:"active_users"`
		Messages      int64  `json:"messages"`
		NewRooms      int64  `json:"new_rooms"`
		FileUploads   int64  `json:"file_uploads"`
	}

	var report []EngagementData
	current := startDate

	for current.Before(endDate) || current.Equal(endDate) {
		nextDay := current.Add(24 * time.Hour)
		
		var activeUsers, messages, newRooms, fileUploads int64
		
		// Active users for this day
		s.db.Model(&models.User{}).
			Where("last_seen_at >= ? AND last_seen_at < ?", current, nextDay).
			Count(&activeUsers)

		// Messages for this day
		s.db.Model(&models.Message{}).
			Where("created_at >= ? AND created_at < ?", current, nextDay).
			Count(&messages)

		// New rooms for this day
		s.db.Model(&models.ChatRoom{}).
			Where("created_at >= ? AND created_at < ?", current, nextDay).
			Count(&newRooms)

		// File uploads for this day
		s.db.Model(&models.Message{}).
			Where("created_at >= ? AND created_at < ? AND type IN ('image', 'file', 'audio', 'video')", current, nextDay).
			Count(&fileUploads)

		report = append(report, EngagementData{
			Date:        current.Format("2006-01-02"),
			ActiveUsers: activeUsers,
			Messages:    messages,
			NewRooms:    newRooms,
			FileUploads: fileUploads,
		})

		current = nextDay
	}

	return report, nil
}
