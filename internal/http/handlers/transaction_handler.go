package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"avagostar-form-server/internal/models"
	"avagostar-form-server/internal/repo"
	"avagostar-form-server/internal/services"
	"avagostar-form-server/internal/utils"
	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	transactions *services.TransactionService
}

type TransactionCreateRequest struct {
	ReceiverType  string  `json:"receiver_type" binding:"required,oneof=individual legal"`
	ReceiverName  string  `json:"receiver_name" binding:"required"`
	ReceiverID    *string `json:"receiver_id"`
	PayerType     string  `json:"payer_type" binding:"required,oneof=individual legal"`
	PayerName     string  `json:"payer_name" binding:"required"`
	PayerID       *string `json:"payer_id"`
	PaymentMethod string  `json:"payment_method" binding:"required,oneof=cash account"`
	Currency      string  `json:"currency" binding:"required,oneof=IRR IRT USD EUR AED TRY"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Description   *string `json:"description"`
	DatetimeISO   string  `json:"datetime_iso" binding:"required"`
	Timezone      string  `json:"timezone" binding:"required"`
}

type TransactionResponse struct {
	ID              string    `json:"id"`
	CreatedByUserID string    `json:"created_by_user_id"`
	ReceiverType    string    `json:"receiver_type"`
	ReceiverName    string    `json:"receiver_name"`
	ReceiverID      *string   `json:"receiver_id,omitempty"`
	PayerType       string    `json:"payer_type"`
	PayerName       string    `json:"payer_name"`
	PayerID         *string   `json:"payer_id,omitempty"`
	PaymentMethod   string    `json:"payment_method"`
	Currency        string    `json:"currency"`
	Amount          float64   `json:"amount"`
	Description     *string   `json:"description,omitempty"`
	DatetimeISO     string    `json:"datetime_iso"`
	Timezone        string    `json:"timezone"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func NewTransactionHandler(transactions *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{transactions: transactions}
}

func (h *TransactionHandler) Create(c *gin.Context) {
	var req TransactionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	parsedTime, err := time.Parse(time.RFC3339, req.DatetimeISO)
	if err != nil {
		utils.RespondValidationError(c, "datetime_iso must be RFC3339")
		return
	}

	userID := c.GetString("user_id")

	tx := &models.Transaction{
		CreatedByUserID: userID,
		ReceiverType:    req.ReceiverType,
		ReceiverName:    req.ReceiverName,
		ReceiverID:      req.ReceiverID,
		PayerType:       req.PayerType,
		PayerName:       req.PayerName,
		PayerID:         req.PayerID,
		PaymentMethod:   req.PaymentMethod,
		Currency:        req.Currency,
		Amount:          req.Amount,
		Description:     req.Description,
		DatetimeUTC:     parsedTime.UTC(),
		Timezone:        req.Timezone,
	}

	created, err := h.transactions.Create(c.Request.Context(), tx)
	if err != nil {
		utils.RespondError(c, err)
		return
	}

	utils.RespondCreated(c, transactionToResponse(*created))
}

func (h *TransactionHandler) List(c *gin.Context) {
	filters, err := parseTransactionFilters(c)
	if err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	items, total, err := h.transactions.List(c.Request.Context(), filters)
	if err != nil {
		utils.RespondError(c, err)
		return
	}

	data := make([]TransactionResponse, 0, len(items))
	for _, item := range items {
		data = append(data, transactionToResponse(item))
	}

	pagination := utils.NewPagination(filters.Page, filters.PerPage, total)
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": pagination,
	})
}

func (h *TransactionHandler) Summary(c *gin.Context) {
	filters, err := parseTransactionFilters(c)
	if err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	summary, err := h.transactions.Summary(c.Request.Context(), filters)
	if err != nil {
		utils.RespondError(c, err)
		return
	}

	monthly := make([]gin.H, 0, 12)
	for i := 1; i <= 12; i++ {
		month := strconv.Itoa(i)
		if i < 10 {
			month = "0" + month
		}
		monthly = append(monthly, gin.H{
			"month":  month,
			"amount": summary.Monthly[month],
		})
	}

	byCurrency := make([]gin.H, 0, len(summary.ByCurrency))
	for currency, amount := range summary.ByCurrency {
		percent := 0.0
		if summary.TotalAmount > 0 {
			percent = (amount / summary.TotalAmount) * 100
		}
		byCurrency = append(byCurrency, gin.H{
			"currency": currency,
			"amount":   amount,
			"percent":  percent,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"kpis": gin.H{
			"total_amount": summary.TotalAmount,
			"avg_amount":   summary.AvgAmount,
			"count":        summary.Count,
		},
		"monthly":     monthly,
		"by_currency": byCurrency,
	})
}

func (h *TransactionHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	item, err := h.transactions.GetByID(c.Request.Context(), id, &userID)
	if err != nil {
		utils.RespondError(c, utils.NewAppError(http.StatusNotFound, "NOT_FOUND", "transaction not found", nil))
		return
	}

	c.JSON(http.StatusOK, transactionToResponse(*item))
}

func (h *TransactionHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	deleted, err := h.transactions.Delete(c.Request.Context(), id, &userID)
	if err != nil {
		utils.RespondError(c, err)
		return
	}
	if !deleted {
		utils.RespondError(c, utils.NewAppError(http.StatusNotFound, "NOT_FOUND", "transaction not found", nil))
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func parseTransactionFilters(c *gin.Context) (repo.TransactionFilters, error) {
	filters := repo.TransactionFilters{}
	filters.Search = c.Query("search")
	filters.Currency = c.Query("currency")
	filters.SortBy = c.Query("sort_by")
	filters.SortDir = c.Query("sort_dir")
	filters.Page = parseIntDefault(c.Query("page"), 1)
	filters.PerPage = parseIntDefault(c.Query("per_page"), 10)

	if minAmountStr := c.Query("min_amount"); minAmountStr != "" {
		val, err := strconv.ParseFloat(minAmountStr, 64)
		if err != nil {
			return filters, err
		}
		filters.MinAmount = &val
	}

	if monthStr := c.Query("month"); monthStr != "" {
		val, err := strconv.Atoi(monthStr)
		if err != nil || val < 1 || val > 12 {
			return filters, fmt.Errorf("invalid month")
		}
		filters.Month = &val
	}

	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		parsed, err := time.Parse("2006-01-02", dateFromStr)
		if err != nil {
			return filters, err
		}
		filters.DateFrom = &parsed
	}

	if dateToStr := c.Query("date_to"); dateToStr != "" {
		parsed, err := time.Parse("2006-01-02", dateToStr)
		if err != nil {
			return filters, err
		}
		end := parsed.Add(24 * time.Hour)
		filters.DateTo = &end
	}

	userID := c.GetString("user_id")
	if userID != "" {
		filters.CreatedBy = &userID
	}

	return filters, nil
}

func parseIntDefault(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func transactionToResponse(tx models.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:              tx.ID,
		CreatedByUserID: tx.CreatedByUserID,
		ReceiverType:    tx.ReceiverType,
		ReceiverName:    tx.ReceiverName,
		ReceiverID:      tx.ReceiverID,
		PayerType:       tx.PayerType,
		PayerName:       tx.PayerName,
		PayerID:         tx.PayerID,
		PaymentMethod:   tx.PaymentMethod,
		Currency:        tx.Currency,
		Amount:          tx.Amount,
		Description:     tx.Description,
		DatetimeISO:     tx.DatetimeUTC.UTC().Format(time.RFC3339),
		Timezone:        tx.Timezone,
		CreatedAt:       tx.CreatedAt,
		UpdatedAt:       tx.UpdatedAt,
	}
}
