package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"investplanner/server/internal/database"
	"investplanner/server/internal/domain"
	"investplanner/server/internal/service"
)

func monthResponse(record database.MonthlyRecord) gin.H {
	expenses := make([]gin.H, 0, len(record.Expenses))
	for _, expense := range record.Expenses {
		sourceID := ""
		if expense.SourceID != nil {
			sourceID = *expense.SourceID
		}
		expenses = append(expenses, gin.H{
			"id": expense.ID, "sourceId": sourceID, "sourceName": expense.SourceName,
			"amountCents": expense.AmountCents, "sortOrder": expense.SortOrder,
		})
	}
	allocations := make([]gin.H, 0, len(record.Allocations))
	actualTotal := int64(0)
	for _, allocation := range record.Allocations {
		destinationID := ""
		if allocation.DestinationID != nil {
			destinationID = *allocation.DestinationID
		}
		actualTotal += allocation.ActualCents
		allocations = append(allocations, gin.H{
			"id": allocation.ID, "destinationId": destinationID, "name": allocation.DestinationName,
			"sortOrder": allocation.SortOrder, "allocationBps": allocation.AllocationBPS,
			"recommendedCents": allocation.RecommendedCents, "actualCents": allocation.ActualCents,
			"differenceCents": allocation.ActualCents - allocation.RecommendedCents,
		})
	}
	return gin.H{
		"id": record.ID, "planId": record.PlanID, "month": record.Month,
		"incomeCents": record.IncomeCents, "expenseTotalCents": record.ExpenseTotalCents,
		"reserveCents": record.ReserveCents, "surplusCents": record.SurplusCents,
		"investableBaseCents": record.InvestableBaseCents, "contributionBps": record.ContributionBPS,
		"recommendedTotalCents": record.RecommendedTotalCents, "roundingUnitCents": record.RoundingUnitCents,
		"actualTotalCents": actualTotal, "status": record.Status, "note": record.Note,
		"version": record.Version, "expenses": expenses, "allocations": allocations,
	}
}

func (a *API) listMonths(c *gin.Context) {
	records, err := a.app.Store.ListMonths(currentUserID(c), c.Param("planID"))
	if err != nil {
		handleError(c, err)
		return
	}
	items := make([]gin.H, 0, len(records))
	for _, record := range records {
		items = append(items, monthResponse(record))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *API) getMonth(c *gin.Context) {
	record, err := a.app.Store.GetMonth(currentUserID(c), c.Param("planID"), c.Param("month"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, monthResponse(record))
}

func (a *API) upsertMonth(c *gin.Context) {
	var request service.MonthRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "月度数据格式不正确", nil)
		return
	}
	request.Month = c.Param("month")
	record, err := a.app.UpsertMonth(currentUserID(c), c.Param("planID"), request)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, monthResponse(record))
}

func (a *API) deleteMonth(c *gin.Context) {
	if err := a.app.Store.DeleteMonth(currentUserID(c), c.Param("planID"), c.Param("month")); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) previousSources(c *gin.Context) {
	before := c.Query("before")
	expenses, err := a.app.Store.PreviousMonthSources(currentUserID(c), c.Param("planID"), before)
	if err != nil {
		handleError(c, err)
		return
	}
	items := make([]gin.H, 0, len(expenses))
	for _, expense := range expenses {
		sourceID := ""
		if expense.SourceID != nil {
			sourceID = *expense.SourceID
		}
		items = append(items, gin.H{"sourceId": sourceID, "sourceName": expense.SourceName, "sortOrder": expense.SortOrder})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *API) updateActuals(c *gin.Context) {
	var request struct {
		Version int `json:"version" binding:"required"`
		Items   []struct {
			AllocationID string `json:"allocationId" binding:"required"`
			ActualCents  int64  `json:"actualCents"`
		} `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "实际金额数据格式不正确", nil)
		return
	}
	actuals := map[string]int64{}
	for _, item := range request.Items {
		if !domain.ValidateMoney(item.ActualCents) {
			respondError(c, http.StatusBadRequest, "validation_error", "实际金额不能为负数", nil)
			return
		}
		actuals[item.AllocationID] = item.ActualCents
	}
	record, err := a.app.Store.UpdateActuals(currentUserID(c), c.Param("planID"), c.Param("month"), actuals, request.Version)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, monthResponse(record))
}

func (a *API) planStats(c *gin.Context) {
	stats, err := a.app.PlanStats(currentUserID(c), c.Param("planID"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

var _ = strconv.IntSize
