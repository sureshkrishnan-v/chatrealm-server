package models

import (
        "time"

        "github.com/google/uuid"
        "gorm.io/gorm"
)

type SubscriptionStatus string

const (
        StatusActive    SubscriptionStatus = "active"
        StatusCanceled  SubscriptionStatus = "canceled"
        StatusExpired   SubscriptionStatus = "expired"
        StatusPending   SubscriptionStatus = "pending"
        StatusTrialing  SubscriptionStatus = "trialing"
)

type SubscriptionPlan string

const (
        PlanPremiumMonthly SubscriptionPlan = "premium_monthly"
        PlanPremiumYearly  SubscriptionPlan = "premium_yearly"
)

type Subscription struct {
        ID                   uuid.UUID          `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
        UserID               uuid.UUID          `json:"user_id" gorm:"type:uuid;not null;unique"`
        StripeCustomerID     string             `json:"stripe_customer_id" gorm:"unique"`
        StripeSubscriptionID string             `json:"stripe_subscription_id" gorm:"unique"`
        Plan                 SubscriptionPlan   `json:"plan" gorm:"not null"`
        Status               SubscriptionStatus `json:"status" gorm:"default:'pending'"`
        CurrentPeriodStart   time.Time          `json:"current_period_start"`
        CurrentPeriodEnd     time.Time          `json:"current_period_end"`
        CanceledAt           *time.Time         `json:"canceled_at"`
        TrialStart           *time.Time         `json:"trial_start"`
        TrialEnd             *time.Time         `json:"trial_end"`
        CreatedAt            time.Time          `json:"created_at"`
        UpdatedAt            time.Time          `json:"updated_at"`
        DeletedAt            gorm.DeletedAt     `json:"-" gorm:"index"`

        // Relationships
        User User `json:"user"`
}

type BillingHistory struct {
        ID             uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
        UserID         uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
        SubscriptionID uuid.UUID `json:"subscription_id" gorm:"type:uuid;not null"`
        StripeInvoiceID string   `json:"stripe_invoice_id"`
        Amount         int64     `json:"amount"` // in cents
        Currency       string    `json:"currency" gorm:"default:'usd'"`
        Status         string    `json:"status"`
        PaidAt         *time.Time `json:"paid_at"`
        CreatedAt      time.Time `json:"created_at"`

        // Relationships
        User         User         `json:"user" gorm:"foreignKey:UserID"`
        Subscription Subscription `json:"subscription" gorm:"foreignKey:SubscriptionID"`
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
        if s.ID == uuid.Nil {
                s.ID = uuid.New()
        }
        return nil
}

func (bh *BillingHistory) BeforeCreate(tx *gorm.DB) error {
        if bh.ID == uuid.Nil {
                bh.ID = uuid.New()
        }
        return nil
}

func (s *Subscription) IsActive() bool {
        return s.Status == StatusActive || s.Status == StatusTrialing
}

func (s *Subscription) IsExpired() bool {
        if s.Status == StatusExpired {
                return true
        }
        
        if s.Status == StatusActive && time.Now().After(s.CurrentPeriodEnd) {
                return true
        }
        
        return false
}

func (s *Subscription) DaysUntilExpiry() int {
        if s.IsExpired() {
                return 0
        }
        
        duration := time.Until(s.CurrentPeriodEnd)
        return int(duration.Hours() / 24)
}

func (sp SubscriptionPlan) GetPrice() int64 {
        switch sp {
        case PlanPremiumMonthly:
                return 999 // $9.99
        case PlanPremiumYearly:
                return 9999 // $99.99
        default:
                return 0
        }
}

func (sp SubscriptionPlan) GetInterval() string {
        switch sp {
        case PlanPremiumMonthly:
                return "month"
        case PlanPremiumYearly:
                return "year"
        default:
                return "month"
        }
}
