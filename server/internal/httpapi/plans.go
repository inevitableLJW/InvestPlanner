package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"investplanner/server/internal/database"
	"investplanner/server/internal/domain"
	"investplanner/server/internal/repository"
	"investplanner/server/internal/service"
)

func planResponse(plan database.Plan) gin.H {
	destinations := make([]gin.H, 0, len(plan.Destinations))
	for _, destination := range plan.Destinations {
		destinations = append(destinations, gin.H{
			"id": destination.ID, "name": destination.Name, "active": destination.Active,
			"allocationBps": destination.AllocationBPS, "sortOrder": destination.SortOrder,
			"archived": destination.ArchivedAt != nil, "version": destination.Version,
		})
	}
	return gin.H{
		"id": plan.ID, "name": plan.Name, "status": plan.Status,
		"defaultContributionBps": plan.DefaultContributionBPS, "reserveCents": plan.ReserveCents,
		"roundingUnitCents": plan.RoundingUnitCents, "version": plan.Version,
		"destinations": destinations,
	}
}

func sourceResponse(source database.ExpenseSource) gin.H {
	return gin.H{
		"id": source.ID, "name": source.Name, "sortOrder": source.SortOrder, "active": source.Active,
	}
}

func (a *API) listPlans(c *gin.Context) {
	includeArchived, _ := strconv.ParseBool(c.Query("includeArchived"))
	plans, err := a.app.Store.ListPlans(currentUserID(c), includeArchived)
	if err != nil {
		handleError(c, err)
		return
	}
	items := make([]gin.H, 0, len(plans))
	for _, plan := range plans {
		item := planResponse(plan)
		stats, statsErr := a.app.PlanStats(currentUserID(c), plan.ID)
		if statsErr == nil {
			item["summary"] = stats
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *API) createPlan(c *gin.Context) {
	var request struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "计划名称不能为空", nil)
		return
	}
	plan, err := a.app.CreatePlan(currentUserID(c), request.Name)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, planResponse(plan))
}

func (a *API) getPlan(c *gin.Context) {
	plan, err := a.app.Store.GetPlan(currentUserID(c), c.Param("planID"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, planResponse(plan))
}

func (a *API) updatePlan(c *gin.Context) {
	var request service.PlanInput
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "计划数据格式不正确", nil)
		return
	}
	plan, err := a.app.UpdatePlan(currentUserID(c), c.Param("planID"), request)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, planResponse(plan))
}

func (a *API) archivePlan(c *gin.Context) {
	version, err := strconv.Atoi(c.Query("version"))
	if err != nil || version < 1 {
		respondError(c, http.StatusBadRequest, "validation_error", "需要有效版本号", nil)
		return
	}
	if err := a.app.Store.ArchivePlan(currentUserID(c), c.Param("planID"), version); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) listSources(c *gin.Context) {
	sources, err := a.app.Store.ListSources(currentUserID(c))
	if err != nil {
		handleError(c, err)
		return
	}
	items := make([]gin.H, 0, len(sources))
	for _, source := range sources {
		items = append(items, sourceResponse(source))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *API) saveSource(c *gin.Context) {
	var request struct {
		Name      string `json:"name" binding:"required"`
		SortOrder int    `json:"sortOrder"`
		Active    *bool  `json:"active"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "支出来源名称不能为空", nil)
		return
	}
	active := true
	if request.Active != nil {
		active = *request.Active
	}
	source, err := a.app.Store.SaveSource(currentUserID(c), database.ExpenseSource{
		ID: c.Param("sourceID"), Name: request.Name, SortOrder: request.SortOrder, Active: active,
	})
	if err != nil {
		if err == repository.ErrConflict {
			handleError(c, err)
			return
		}
		handleError(c, err)
		return
	}
	status := http.StatusOK
	if c.Param("sourceID") == "" {
		status = http.StatusCreated
	}
	c.JSON(status, sourceResponse(source))
}

var _ = domain.PlanActive
