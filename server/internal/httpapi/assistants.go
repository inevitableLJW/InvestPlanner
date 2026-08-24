package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"investplanner/server/internal/service"
)

func (a *API) qdiiAssistant(c *gin.Context) {
	remaining := int64(0)
	if raw := c.Query("remainingNeedCents"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 0 {
			returnValidation(c, "本月剩余建议需求无效")
			return
		}
		remaining = value
	}
	if a.app.QDII == nil {
		c.JSON(http.StatusOK, gin.H{"destinationId": c.Param("destinationID"), "status": "unavailable", "reason": "QDII 数据服务暂不可用", "otc": []any{}})
		return
	}
	result, err := a.app.QDII.Assist(currentUserID(c), c.Param("planID"), c.Param("destinationID"), remaining)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (a *API) fxAssistant(c *gin.Context) {
	window := c.DefaultQuery("window", "rolling_12")
	if window != "rolling_12" && window != "next_calendar_year" {
		returnValidation(c, "预测窗口无效")
		return
	}
	scenarios := map[string]float64{}
	for _, currency := range []string{"USD", "HKD"} {
		raw := c.Query("scenario" + currency)
		if raw == "" {
			continue
		}
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || value <= 0 || value > 1000 {
			returnValidation(c, "情景汇率无效")
			return
		}
		scenarios[currency] = value
	}
	if a.app.FX == nil {
		c.JSON(http.StatusOK, service.FXForecast{Window: window, Status: "unavailable", Reason: "年度换汇预测暂不可用", Currencies: []string{}, Months: []service.FXMonth{}, Summaries: []service.FXCurrencySummary{}, Rates: []service.FXRateInfo{}, ComplianceNotice: service.FXComplianceNotice})
		return
	}
	result, err := a.app.FX.Forecast(currentUserID(c), c.Param("planID"), window, scenarios)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
func returnValidation(c *gin.Context, message string) {
	respondError(c, http.StatusBadRequest, "validation_error", message, nil)
}
