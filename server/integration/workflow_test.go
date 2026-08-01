package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func decodeObject(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func activateSingleDestination(t *testing.T, router http.Handler, cookie *http.Cookie, plan map[string]any) map[string]any {
	t.Helper()
	destinations := plan["destinations"].([]any)
	for index, item := range destinations {
		destination := item.(map[string]any)
		destination["active"] = index == 0
		if index == 0 {
			destination["allocationBps"] = 10000
		} else {
			destination["allocationBps"] = 0
		}
	}
	response := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+plan["id"].(string), map[string]any{
		"name": plan["name"], "status": "active", "defaultContributionBps": 8000,
		"reserveCents": 0, "roundingUnitCents": 100, "version": plan["version"], "destinations": destinations,
	}, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("activate plan status=%d body=%s", response.Code, response.Body.String())
	}
	return decodeObject(t, response)
}

func TestPlanMonthSnapshotsConflictsAndIsolation(t *testing.T) {
	router := testRouter(t)
	owner := register(t, router, "owner1@example.com")
	first := decodeObject(t, requestJSON(t, router, http.MethodPost, "/api/v1/plans", map[string]any{"name": "plan-one"}, owner))
	second := decodeObject(t, requestJSON(t, router, http.MethodPost, "/api/v1/plans", map[string]any{"name": "plan-two"}, owner))
	if len(first["destinations"].([]any)) != 4 {
		t.Fatal("new plan must include four editable defaults")
	}
	first = activateSingleDestination(t, router, owner, first)
	second = activateSingleDestination(t, router, owner, second)

	createMonth := func(plan map[string]any, income int) map[string]any {
		response := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+plan["id"].(string)+"/months/2026-07", map[string]any{
			"incomeCents": income, "contributionBps": 5000,
			"expenses": []any{map[string]any{"sourceName": "wechat", "amountCents": 10000, "sortOrder": 0}},
		}, owner)
		if response.Code != http.StatusOK {
			t.Fatalf("create month status=%d body=%s", response.Code, response.Body.String())
		}
		return decodeObject(t, response)
	}
	monthOne := createMonth(first, 100000)
	monthTwo := createMonth(second, 300000)
	if monthOne["recommendedTotalCents"] == monthTwo["recommendedTotalCents"] {
		t.Fatal("same month in two plans must remain independent")
	}

	allocation := monthOne["allocations"].([]any)[0].(map[string]any)
	actualResponse := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+first["id"].(string)+"/months/2026-07/actuals", map[string]any{
		"version": monthOne["version"], "items": []any{map[string]any{"allocationId": allocation["id"], "actualCents": 45000}},
	}, owner)
	if actualResponse.Code != http.StatusOK {
		t.Fatalf("actual status=%d body=%s", actualResponse.Code, actualResponse.Body.String())
	}
	actual := decodeObject(t, actualResponse)
	if actual["status"] != "complete" {
		t.Fatalf("unexpected execution status: %v", actual["status"])
	}
	conflict := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+first["id"].(string)+"/months/2026-07/actuals", map[string]any{
		"version": monthOne["version"], "items": []any{map[string]any{"allocationId": allocation["id"], "actualCents": 1}},
	}, owner)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("stale actual status=%d body=%s", conflict.Code, conflict.Body.String())
	}

	originalName := allocation["name"].(string)
	destinations := first["destinations"].([]any)
	destinations[0].(map[string]any)["name"] = "renamed-destination"
	changed := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+first["id"].(string), map[string]any{
		"name": first["name"], "status": "active", "defaultContributionBps": 8000,
		"reserveCents": 0, "roundingUnitCents": 100, "version": first["version"], "destinations": destinations,
	}, owner)
	if changed.Code != http.StatusOK {
		t.Fatalf("rename status=%d body=%s", changed.Code, changed.Body.String())
	}
	recalculated := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+first["id"].(string)+"/months/2026-07", map[string]any{
		"incomeCents": 120000, "contributionBps": 5000, "version": actual["version"], "expenses": []any{},
	}, owner)
	if recalculated.Code != http.StatusOK {
		t.Fatalf("recalculate status=%d body=%s", recalculated.Code, recalculated.Body.String())
	}
	if decodeObject(t, recalculated)["allocations"].([]any)[0].(map[string]any)["name"] != originalName {
		t.Fatal("historical recalculation must preserve destination snapshot")
	}

	outsider := register(t, router, "outsider1@example.com")
	for _, path := range []string{
		"/api/v1/plans/" + first["id"].(string),
		"/api/v1/plans/" + first["id"].(string) + "/months/2026-07",
		"/api/v1/plans/" + first["id"].(string) + "/stats",
	} {
		response := requestJSON(t, router, http.MethodGet, path, nil, outsider)
		if response.Code != http.StatusNotFound {
			t.Fatalf("cross-account path=%s status=%d", path, response.Code)
		}
	}
	deleted := requestJSON(t, router, http.MethodDelete, "/api/v1/plans/"+first["id"].(string)+"/months/2026-07", nil, owner)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
}
