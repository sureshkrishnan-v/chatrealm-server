package services

import (
        "encoding/json"
        "errors"
        "premium-chat-backend/models"
        "time"

        "github.com/google/uuid"
        "github.com/stripe/stripe-go/v75"
        "github.com/stripe/stripe-go/v75/checkout/session"
        "github.com/stripe/stripe-go/v75/customer"
        "github.com/stripe/stripe-go/v75/setupintent"
        "github.com/stripe/stripe-go/v75/subscription"
        "github.com/stripe/stripe-go/v75/webhook"
        "gorm.io/gorm"
)

type SubscriptionService struct {
        db *gorm.DB
}

func NewSubscriptionService(db *gorm.DB, stripeSecretKey string) *SubscriptionService {
        stripe.Key = stripeSecretKey
        return &SubscriptionService{
                db: db,
        }
}

func (s *SubscriptionService) GetUserSubscription(userID uuid.UUID) (*models.Subscription, error) {
        var subscription models.Subscription
        if err := s.db.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
                return nil, err
        }
        return &subscription, nil
}

func (s *SubscriptionService) CreateCheckoutSession(user *models.User, plan models.SubscriptionPlan) (string, error) {
        // Create or get Stripe customer
        customerID, err := s.getOrCreateStripeCustomer(user)
        if err != nil {
                return "", err
        }

        // Create checkout session
        params := &stripe.CheckoutSessionParams{
                Customer: stripe.String(customerID),
                PaymentMethodTypes: stripe.StringSlice([]string{
                        "card",
                }),
                LineItems: []*stripe.CheckoutSessionLineItemParams{
                        {
                                PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
                                        Currency: stripe.String("usd"),
                                        ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
                                                Name: stripe.String("Premium Chat Subscription"),
                                        },
                                        UnitAmount: stripe.Int64(plan.GetPrice()),
                                        Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
                                                Interval: stripe.String(plan.GetInterval()),
                                        },
                                },
                                Quantity: stripe.Int64(1),
                        },
                },
                Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
                SuccessURL: stripe.String("https://your-domain.com/success?session_id={CHECKOUT_SESSION_ID}"),
                CancelURL:  stripe.String("https://your-domain.com/cancel"),
                Metadata: map[string]string{
                        "user_id": user.ID.String(),
                        "plan":    string(plan),
                },
        }

        sess, err := session.New(params)
        if err != nil {
                return "", err
        }

        return sess.URL, nil
}

func (s *SubscriptionService) getOrCreateStripeCustomer(user *models.User) (string, error) {
        // Check if user already has a Stripe customer ID
        var subscription models.Subscription
        if err := s.db.Where("user_id = ?", user.ID).First(&subscription).Error; err == nil {
                if subscription.StripeCustomerID != "" {
                        return subscription.StripeCustomerID, nil
                }
        }

        // Create new Stripe customer
        params := &stripe.CustomerParams{
                Email: stripe.String(user.Email),
                Name:  stripe.String(user.Username),
                Metadata: map[string]string{
                        "user_id": user.ID.String(),
                },
        }

        c, err := customer.New(params)
        if err != nil {
                return "", err
        }

        return c.ID, nil
}

func (s *SubscriptionService) CancelSubscription(userID uuid.UUID) error {
        var userSub models.Subscription
        if err := s.db.Where("user_id = ?", userID).First(&userSub).Error; err != nil {
                return errors.New("subscription not found")
        }

        if !userSub.IsActive() {
                return errors.New("subscription is not active")
        }

        // Cancel in Stripe
        _, err := subscription.Cancel(userSub.StripeSubscriptionID, nil)
        if err != nil {
                return err
        }

        // Update local subscription
        now := time.Now()
        userSub.Status = models.StatusCanceled
        userSub.CanceledAt = &now

        if err := s.db.Save(&userSub).Error; err != nil {
                return err
        }

        // Update user tier
        var user models.User
        if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
                return err
        }

        user.Tier = models.TierFree
        return s.db.Save(&user).Error
}

func (s *SubscriptionService) HandleWebhook(payload []byte, signature string) error {
        // You should set this in your environment variables
        endpointSecret := "whsec_..." // Replace with your webhook secret

        event, err := webhook.ConstructEvent(payload, signature, endpointSecret)
        if err != nil {
                return err
        }

        switch event.Type {
        case "checkout.session.completed":
                var session stripe.CheckoutSession
                if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
                        return err
                }
                return s.handleCheckoutSessionCompleted(&session)

        case "invoice.payment_succeeded":
                var invoice stripe.Invoice
                if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
                        return err
                }
                return s.handleInvoicePaymentSucceeded(&invoice)

        case "customer.subscription.updated":
                var subscription stripe.Subscription
                if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
                        return err
                }
                return s.handleSubscriptionUpdated(&subscription)

        case "customer.subscription.deleted":
                var subscription stripe.Subscription
                if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
                        return err
                }
                return s.handleSubscriptionDeleted(&subscription)
        }

        return nil
}

func (s *SubscriptionService) handleCheckoutSessionCompleted(session *stripe.CheckoutSession) error {
        userID, err := uuid.Parse(session.Metadata["user_id"])
        if err != nil {
                return err
        }

        planStr := session.Metadata["plan"]
        var plan models.SubscriptionPlan
        switch planStr {
        case "premium_monthly":
                plan = models.PlanPremiumMonthly
        case "premium_yearly":
                plan = models.PlanPremiumYearly
        default:
                return errors.New("invalid plan")
        }

        // Get the subscription from Stripe
        stripeSubscription, err := subscription.Get(session.Subscription.ID, nil)
        if err != nil {
                return err
        }

        // Create or update subscription
        subscription := &models.Subscription{
                UserID:               userID,
                StripeCustomerID:     session.Customer.ID,
                StripeSubscriptionID: session.Subscription.ID,
                Plan:                 plan,
                Status:               models.StatusActive,
                CurrentPeriodStart:   time.Unix(stripeSubscription.CurrentPeriodStart, 0),
                CurrentPeriodEnd:     time.Unix(stripeSubscription.CurrentPeriodEnd, 0),
        }

        if err := s.db.Where("user_id = ?", userID).First(&models.Subscription{}).Error; err != nil {
                // Create new subscription
                if err := s.db.Create(subscription).Error; err != nil {
                        return err
                }
        } else {
                // Update existing subscription
                if err := s.db.Where("user_id = ?", userID).Updates(subscription).Error; err != nil {
                        return err
                }
        }

        // Update user tier
        var user models.User
        if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
                return err
        }

        user.Tier = models.TierPremium
        return s.db.Save(&user).Error
}

func (s *SubscriptionService) handleInvoicePaymentSucceeded(invoice *stripe.Invoice) error {
        if invoice.Subscription == nil {
                return nil // Not a subscription invoice
        }

        var subscription models.Subscription
        if err := s.db.Where("stripe_subscription_id = ?", invoice.Subscription.ID).First(&subscription).Error; err != nil {
                return err // Subscription not found
        }

        // Create billing history record
        var paidAt *time.Time
        if invoice.StatusTransitions.PaidAt != 0 {
                paid := time.Unix(invoice.StatusTransitions.PaidAt, 0)
                paidAt = &paid
        }

        billingHistory := &models.BillingHistory{
                UserID:          subscription.UserID,
                SubscriptionID:  subscription.ID,
                StripeInvoiceID: invoice.ID,
                Amount:          invoice.AmountPaid,
                Currency:        string(invoice.Currency),
                Status:          string(invoice.Status),
                PaidAt:          paidAt,
        }

        return s.db.Create(billingHistory).Error
}

func (s *SubscriptionService) handleSubscriptionUpdated(stripeSubscription *stripe.Subscription) error {
        var subscription models.Subscription
        if err := s.db.Where("stripe_subscription_id = ?", stripeSubscription.ID).First(&subscription).Error; err != nil {
                return err
        }

        // Update subscription details
        subscription.Status = models.SubscriptionStatus(stripeSubscription.Status)
        subscription.CurrentPeriodStart = time.Unix(stripeSubscription.CurrentPeriodStart, 0)
        subscription.CurrentPeriodEnd = time.Unix(stripeSubscription.CurrentPeriodEnd, 0)

        if stripeSubscription.CanceledAt != 0 {
                canceled := time.Unix(stripeSubscription.CanceledAt, 0)
                subscription.CanceledAt = &canceled
        }

        return s.db.Save(&subscription).Error
}

func (s *SubscriptionService) handleSubscriptionDeleted(stripeSubscription *stripe.Subscription) error {
        var subscription models.Subscription
        if err := s.db.Where("stripe_subscription_id = ?", stripeSubscription.ID).First(&subscription).Error; err != nil {
                return err
        }

        // Update subscription status
        subscription.Status = models.StatusCanceled
        now := time.Now()
        subscription.CanceledAt = &now

        if err := s.db.Save(&subscription).Error; err != nil {
                return err
        }

        // Update user tier
        var user models.User
        if err := s.db.First(&user, "id = ?", subscription.UserID).Error; err != nil {
                return err
        }

        user.Tier = models.TierFree
        return s.db.Save(&user).Error
}

func (s *SubscriptionService) GetBillingHistory(userID uuid.UUID) ([]models.BillingHistory, error) {
        var history []models.BillingHistory
        if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&history).Error; err != nil {
                return nil, err
        }
        return history, nil
}

func (s *SubscriptionService) CreateSetupIntent(userID uuid.UUID) (string, error) {
        var subscription models.Subscription
        if err := s.db.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
                return "", errors.New("subscription not found")
        }

        params := &stripe.SetupIntentParams{
                Customer: stripe.String(subscription.StripeCustomerID),
                Usage:    stripe.String("off_session"),
        }

        si, err := setupintent.New(params)
        if err != nil {
                return "", err
        }

        return si.ClientSecret, nil
}
