package httpapi

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"investplanner/server/internal/service"
)

func TestAssistantHandlersRemainSafeWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &service.App{}
	api := &API{app: app}

	qdiiRecorder := httptest.NewRecorder()
	qdii, _ := gin.CreateTestContext(qdiiRecorder)
	qdii.Request = httptest.NewRequest("GET", "/?remainingNeedCents=10", nil)
	qdii.Params = gin.Params{{Key: "planID", Value: "p1"}, {Key: "destinationID", Value: "d1"}}
	api.qdiiAssistant(qdii)
	if qdiiRecorder.Code != 200 || containsSensitive(qdiiRecorder.Body.String()) {
		t.Fatalf("unsafe QDII response: %s", qdiiRecorder.Body.String())
	}

	fxRecorder := httptest.NewRecorder()
	fx, _ := gin.CreateTestContext(fxRecorder)
	fx.Request = httptest.NewRequest("GET", "/?window=rolling_12", nil)
	fx.Params = gin.Params{{Key: "planID", Value: "p1"}}
	api.fxAssistant(fx)
	if fxRecorder.Code != 200 || containsSensitive(fxRecorder.Body.String()) {
		t.Fatalf("unsafe FX response: %s", fxRecorder.Body.String())
	}
}

func TestAssistantHandlersStrictlyValidateInputs(t *testing.T) {
	api := &API{app: &service.App{}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/?remainingNeedCents=-1", nil)
	api.qdiiAssistant(c)
	if recorder.Code != 400 {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	confirmRecorder := httptest.NewRecorder()
	confirm, _ := gin.CreateTestContext(confirmRecorder)
	confirm.Request = httptest.NewRequest("PUT", "/", nil)
	api.confirmQDIIAssistant(confirm)
	if confirmRecorder.Code != 400 {
		t.Fatalf("confirm status=%d body=%s", confirmRecorder.Code, confirmRecorder.Body.String())
	}
}

func containsSensitive(body string) bool {
	for _, value := range []string{"TUSHARE_TOKEN", "AKTOOLS_BASE_URL", "http://aktools:8080", "secret-token"} {
		if len(value) > 0 && stringContains(body, value) {
			return true
		}
	}
	return false
}
func stringContains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
