package finance

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IHandler interface {
	RegisterRoutes(protected *gin.RouterGroup)
}

type Handler struct {
	svc    IService
	logger *zap.Logger
}

func NewHandler(svc IService, logger *zap.Logger) IHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{svc: svc, logger: logger.With(zap.String("layer", "finance.handler"))}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	trip := protected.Group("/trips/:id/finance")
	{
		// Expenses
		trip.GET("/expenses", h.ListExpenses)
		trip.POST("/expenses", h.CreateExpense)

		// Budgets
		trip.GET("/budgets", h.ListBudgets)
		trip.POST("/budgets", h.UpsertBudget)
		trip.DELETE("/budgets/:budget_id", h.DeleteBudget)

		// Settlements
		trip.GET("/settlements", h.ListSettlements)
		trip.POST("/settlements", h.ProposeSettlements)

		// Stats + export
		trip.GET("/balances", h.GetTripBalances)
		trip.GET("/stats/me", h.GetPersonalStats)
		trip.GET("/export.csv", h.ExportCSV)
	}

	// Non trip-scoped: individual expense / settlement operations
	exp := protected.Group("/finance/expenses/:expense_id")
	{
		exp.GET("", h.GetExpense)
		exp.PATCH("", h.UpdateExpense)
		exp.DELETE("", h.DeleteExpense)
	}
	settle := protected.Group("/finance/settlements/:settlement_id")
	{
		settle.POST("/confirm", h.ConfirmSettlement)
		settle.POST("/cancel", h.CancelSettlement)
	}

	// FX admin — flat, not trip scoped (rates are global to the workspace).
	fx := protected.Group("/finance/fx")
	{
		fx.GET("", h.ListFxRates)
		fx.POST("", h.UpsertFxRate)
	}
}

// ---- Expenses ----

// CreateExpense godoc
// @Summary  Record a new expense in a trip (with split strategy + optional receipt)
// @Description Split strategies: `equal` splits evenly across participants, `custom` uses per-user amounts (must sum to total), `percentage` uses per-user share_pct (must sum to 100). FX rate is resolved from the snapshot table with provider fallback.
// @Tags     finance
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    trip_id  path      string                true  "trip id"
// @Param    body     body      CreateExpensePayload  true  "expense"
// @Success  201      {object}  response.Response{data=ExpenseDTO}
// @Router   /trips/{trip_id}/finance/expenses [post]
func (h *Handler) CreateExpense(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	var p CreateExpensePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.CreateExpense(c.Request.Context(), uid, tripID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.Created(c, out)
}

// ListExpenses godoc
// @Summary  List all expenses recorded against a trip
// @Tags     finance
// @Security BearerAuth
// @Produce  json
// @Param    trip_id  path      string  true  "trip id"
// @Success  200      {object}  response.Response{data=[]ExpenseDTO}
// @Router   /trips/{trip_id}/finance/expenses [get]
func (h *Handler) ListExpenses(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.ListExpenses(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// GetExpense godoc
// @Summary  Fetch a single expense (with shares)
// @Tags     finance
// @Security BearerAuth
// @Produce  json
// @Param    expense_id  path      string  true  "expense id"
// @Success  200         {object}  response.Response{data=ExpenseDTO}
// @Router   /finance/expenses/{expense_id} [get]
func (h *Handler) GetExpense(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	expID, ok := parseUUID(c, "expense_id", "invalid expense id")
	if !ok {
		return
	}
	out, err := h.svc.GetExpense(c.Request.Context(), uid, expID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// UpdateExpense godoc
// @Summary  Patch an existing expense (payer or creator only)
// @Description Any subset of fields may be sent; splits are recomputed only when amount, participants, or strategy change.
// @Tags     finance
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    expense_id  path      string                true  "expense id"
// @Param    body        body      UpdateExpensePayload  true  "patch"
// @Success  200         {object}  response.Response{data=ExpenseDTO}
// @Router   /finance/expenses/{expense_id} [patch]
func (h *Handler) UpdateExpense(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	expID, ok := parseUUID(c, "expense_id", "invalid expense id")
	if !ok {
		return
	}
	var p UpdateExpensePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.UpdateExpense(c.Request.Context(), uid, expID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// DeleteExpense godoc
// @Summary  Soft-delete an expense (payer or creator only)
// @Tags     finance
// @Security BearerAuth
// @Produce  json
// @Param    expense_id  path      string  true  "expense id"
// @Success  200         {object}  response.Response
// @Router   /finance/expenses/{expense_id} [delete]
func (h *Handler) DeleteExpense(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	expID, ok := parseUUID(c, "expense_id", "invalid expense id")
	if !ok {
		return
	}
	if err := h.svc.DeleteExpense(c.Request.Context(), uid, expID); err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, nil)
}

// ---- Budgets ----

// UpsertBudget godoc
// @Summary  Create or update a per-category budget for a trip
// @Description Budgets are unique per (trip, category); resending the same category updates the amount.
// @Tags     finance
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    trip_id  path      string               true  "trip id"
// @Param    body     body      UpsertBudgetPayload  true  "budget"
// @Success  200      {object}  response.Response{data=BudgetDTO}
// @Router   /trips/{trip_id}/finance/budgets [post]
func (h *Handler) UpsertBudget(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	var p UpsertBudgetPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.UpsertBudget(c.Request.Context(), uid, tripID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// ListBudgets godoc
// @Summary  List budgets with per-category spent/remaining/over-budget flags
// @Tags     finance
// @Security BearerAuth
// @Produce  json
// @Param    trip_id  path      string  true  "trip id"
// @Success  200      {object}  response.Response{data=[]BudgetDTO}
// @Router   /trips/{trip_id}/finance/budgets [get]
func (h *Handler) ListBudgets(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.ListBudgets(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// DeleteBudget godoc
// @Summary  Delete a per-category budget
// @Tags     finance
// @Security BearerAuth
// @Produce  json
// @Param    trip_id    path      string  true  "trip id"
// @Param    budget_id  path      string  true  "budget id"
// @Success  200        {object}  response.Response
// @Router   /trips/{trip_id}/finance/budgets/{budget_id} [delete]
func (h *Handler) DeleteBudget(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	bid, ok := parseUUID(c, "budget_id", "invalid budget id")
	if !ok {
		return
	}
	if err := h.svc.DeleteBudget(c.Request.Context(), uid, tripID, bid); err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, nil)
}

// ---- Settlements ----

// ProposeSettlements godoc
// @Summary  Propose settlements (auto min-cashflow reduction, or manual entries)
// @Description `mode=auto` wipes any prior proposed settlements and generates the minimum set of transfers to zero net balances (N-1 transfers max). `mode=manual` appends the provided entries alongside existing ones.
// @Tags     finance
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    trip_id  path      string                    true  "trip id"
// @Param    body     body      ProposeSettlementPayload  true  "proposal"
// @Success  200      {object}  response.Response{data=[]SettlementDTO}
// @Router   /trips/{trip_id}/finance/settlements [post]
func (h *Handler) ProposeSettlements(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	var p ProposeSettlementPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.ProposeSettlements(c.Request.Context(), uid, tripID, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// ListSettlements godoc
// @Summary  List all settlements for a trip (proposed / confirmed / cancelled)
// @Tags     finance
// @Security BearerAuth
// @Produce  json
// @Param    trip_id  path      string  true  "trip id"
// @Success  200      {object}  response.Response{data=[]SettlementDTO}
// @Router   /trips/{trip_id}/finance/settlements [get]
func (h *Handler) ListSettlements(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.ListSettlements(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// ConfirmSettlement godoc
// @Summary  Payee confirms receipt of a settlement (payee only)
// @Description Only the payee can confirm — the payer sending money is not proof of settlement, receipt is. Body is optional.
// @Tags     finance
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    settlement_id  path      string                     true   "settlement id"
// @Param    body           body      ConfirmSettlementPayload  false  "confirmation notes"
// @Success  200            {object}  response.Response{data=SettlementDTO}
// @Router   /finance/settlements/{settlement_id}/confirm [post]
func (h *Handler) ConfirmSettlement(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	sid, ok := parseUUID(c, "settlement_id", "invalid settlement id")
	if !ok {
		return
	}
	var p ConfirmSettlementPayload
	// Body is optional for confirm — an empty POST is a valid "yes, received".
	_ = c.ShouldBindJSON(&p)
	out, err := h.svc.ConfirmSettlement(c.Request.Context(), uid, sid, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// CancelSettlement godoc
// @Summary  Cancel a proposed settlement (payer, payee, or creator)
// @Tags     finance
// @Security BearerAuth
// @Produce  json
// @Param    settlement_id  path      string  true  "settlement id"
// @Success  200            {object}  response.Response{data=SettlementDTO}
// @Router   /finance/settlements/{settlement_id}/cancel [post]
func (h *Handler) CancelSettlement(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	sid, ok := parseUUID(c, "settlement_id", "invalid settlement id")
	if !ok {
		return
	}
	out, err := h.svc.CancelSettlement(c.Request.Context(), uid, sid)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// ---- Stats + export ----

// GetTripBalances godoc
// @Summary  Per-member net balance for a trip (paid - owed - confirmed settlements)
// @Tags     finance
// @Security BearerAuth
// @Produce  json
// @Param    trip_id  path      string  true  "trip id"
// @Success  200      {object}  response.Response{data=[]TripBalance}
// @Router   /trips/{trip_id}/finance/balances [get]
func (h *Handler) GetTripBalances(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.GetTripBalances(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// GetPersonalStats godoc
// @Summary  Caller's own totals for a trip (paid, owed, net, by category)
// @Tags     finance
// @Security BearerAuth
// @Produce  json
// @Param    trip_id  path      string  true  "trip id"
// @Success  200      {object}  response.Response{data=PersonalStats}
// @Router   /trips/{trip_id}/finance/stats/me [get]
func (h *Handler) GetPersonalStats(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	out, err := h.svc.GetPersonalStats(c.Request.Context(), uid, tripID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// ExportCSV godoc
// @Summary  Export trip expenses as CSV (optionally date-bounded)
// @Description Columns are stable and always in the trip base currency. `to` is expanded to end-of-day so `to=2026-07-21` includes 23:59 that day.
// @Tags     finance
// @Security BearerAuth
// @Produce  text/csv
// @Param    trip_id  path      string  true   "trip id"
// @Param    from     query     string  false  "start date YYYY-MM-DD"
// @Param    to       query     string  false  "end date YYYY-MM-DD (inclusive)"
// @Success  200      {file}    file
// @Router   /trips/{trip_id}/finance/export.csv [get]
func (h *Handler) ExportCSV(c *gin.Context) {
	uid, tripID, ok := h.bindUserAndTrip(c)
	if !ok {
		return
	}
	from, to, err := parseDateRange(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	blob, err := h.svc.ExportCSV(c.Request.Context(), uid, tripID, from, to)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	filename := "trip-" + tripID.String() + "-finance.csv"
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", blob)
}

// ---- FX ----

// UpsertFxRate godoc
// @Summary  Manually record an FX rate snapshot (workspace-global, not trip-scoped)
// @Description Rates are keyed by (base, quote, as_of date). Resending overwrites the snapshot for that day.
// @Tags     finance
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body  body      UpsertFxRatePayload  true  "rate"
// @Success  200   {object}  response.Response{data=FxRateDTO}
// @Router   /finance/fx [post]
func (h *Handler) UpsertFxRate(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	var p UpsertFxRatePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.svc.UpsertFxRate(c.Request.Context(), uid, p)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// ListFxRates godoc
// @Summary  List recorded FX snapshots (filter by base currency)
// @Tags     finance
// @Security BearerAuth
// @Produce  json
// @Param    base  query     string  false  "base currency (e.g. USD)"
// @Success  200   {object}  response.Response{data=[]FxRateDTO}
// @Router   /finance/fx [get]
func (h *Handler) ListFxRates(c *gin.Context) {
	uid, ok := requireUser(c)
	if !ok {
		return
	}
	base := c.Query("base")
	out, err := h.svc.ListFxRates(c.Request.Context(), uid, base)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.OK(c, out)
}

// ---- helpers ----

func (h *Handler) bindUserAndTrip(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := requireUser(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	tripID, ok := parseUUID(c, "id", "invalid trip id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return uid, tripID, true
}

func (h *Handler) respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrExpenseNotFound),
		errors.Is(err, ErrBudgetNotFound),
		errors.Is(err, ErrSettlementNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Forbidden(c)
	case errors.Is(err, ErrInvalidPayload),
		errors.Is(err, ErrShareSumMismatch),
		errors.Is(err, ErrPctSumMismatch),
		errors.Is(err, ErrParticipantMissing),
		errors.Is(err, ErrCurrencyMismatch),
		errors.Is(err, ErrRateUnavailable):
		response.BadRequest(c, err.Error())
	default:
		h.logger.Error("finance handler: internal", zap.Error(err))
		response.InternalError(c, "internal error")
	}
}

func requireUser(c *gin.Context) (uuid.UUID, bool) {
	uid := middleware.GetUserID(c)
	if uid == uuid.Nil {
		response.Unauthorized(c)
		return uuid.Nil, false
	}
	return uid, true
}

func parseUUID(c *gin.Context, param, msg string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		response.BadRequest(c, msg)
		return uuid.Nil, false
	}
	return id, true
}

// parseDateRange reads ?from=YYYY-MM-DD&to=YYYY-MM-DD from the export
// endpoint. Both bounds are optional; a missing value = unbounded on that
// side. The `to` bound is expanded to end-of-day so a filter of
// "2026-07-21" includes expenses recorded at 23:59 that day.
func parseDateRange(c *gin.Context) (*time.Time, *time.Time, error) {
	const layout = "2006-01-02"
	var from, to *time.Time
	if s := strings.TrimSpace(c.Query("from")); s != "" {
		t, err := time.Parse(layout, s)
		if err != nil {
			return nil, nil, errors.New("invalid from date")
		}
		from = &t
	}
	if s := strings.TrimSpace(c.Query("to")); s != "" {
		t, err := time.Parse(layout, s)
		if err != nil {
			return nil, nil, errors.New("invalid to date")
		}
		eod := t.Add(24*time.Hour - time.Nanosecond)
		to = &eod
	}
	return from, to, nil
}
