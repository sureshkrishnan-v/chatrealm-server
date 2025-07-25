package handlers

import (
	"net/http"
	"premium-chat-backend/models"
	"premium-chat-backend/services"
	"premium-chat-backend/utils"

	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	subscriptionService *services.SubscriptionService
}

type CreateSubscriptionRequest struct {
	Plan string `json:"plan" binding:"required"`
}

func NewSubscriptionHandler(subscriptionService *services.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionService: subscriptionService,
	}
}

func (h *SubscriptionHandler) GetSubscriptionStatus(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(*models.User)

	subscription, err := h.subscriptionService.GetUserSubscription(userObj.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"subscription": nil,
			"tier": userObj.Tier,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription": subscription,
		"tier": userObj.Tier,
	})
}

func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.FormatValidationError(err)})
		return
	}

	user, _ := c.Get("user")
	userObj := user.(*models.User)

	// Validate plan
	var plan models.SubscriptionPlan
	switch req.Plan {
	case "premium_monthly":
		plan = models.PlanPremiumMonthly
	case "premium_yearly":
		plan = models.PlanPremiumYearly
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription plan"})
		return
	}

	// Create Stripe checkout session
	sessionURL, err := h.subscriptionService.CreateCheckoutSession(userObj, plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create checkout session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"checkout_url": sessionURL,
	})
}

func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(*models.User)

	if err := h.subscriptionService.CancelSubscription(userObj.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription canceled successfully"})
}

func (h *SubscriptionHandler) HandleWebhook(c *gin.Context) {
	payload, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing signature"})
		return
	}

	if err := h.subscriptionService.HandleWebhook(payload, signature); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

func (h *SubscriptionHandler) GetBillingHistory(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(*models.User)

	history, err := h.subscriptionService.GetBillingHistory(userObj.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch billing history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"billing_history": history})
}

func (h *SubscriptionHandler) UpdatePaymentMethod(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(*models.User)

	setupIntentSecret, err := h.subscriptionService.CreateSetupIntent(userObj.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create setup intent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"client_secret": setupIntentSecret,
	})
}
