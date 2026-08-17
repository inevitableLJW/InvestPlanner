package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"investplanner/server/internal/database"
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
		"name": plan["name"], "action": "publish", "defaultContributionBps": 8000,
		"reserveCents": 0, "roundingUnitCents": 10000, "version": plan["version"], "destinations": destinations,
	}, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("activate plan status=%d body=%s", response.Code, response.Body.String())
	}
	return decodeObject(t, response)
}

func TestPlanMonthSnapshotsConflictsAndIsolation(t *testing.T) {
	router := testRouter(t)
	owner := register(t, router, "owner-one")
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
		"name": first["name"], "action": "publish", "defaultContributionBps": 8000,
		"reserveCents": 0, "roundingUnitCents": 10000, "version": first["version"], "destinations": destinations,
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

	outsider := register(t, router, "outsider-one")
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

func TestPlanDraftLifecycleAndPermanentDeletion(t *testing.T) {
	router := testRouter(t)
	owner := register(t, router, "draft-owner")
	outsider := register(t, router, "draft-outsider")

	created := decodeObject(t, requestJSON(t, router, http.MethodPost, "/api/v1/plans", map[string]any{"name": "unfinished"}, owner))
	if created["roundingUnitCents"] != float64(10000) || created["deletable"] != true {
		t.Fatalf("unexpected new plan defaults: %#v", created)
	}
	draftPayload := func(plan map[string]any, action string) map[string]any {
		return map[string]any{
			"name": plan["name"], "action": action, "status": "active", "defaultContributionBps": 8000,
			"reserveCents": 0, "roundingUnitCents": 10000, "version": plan["version"],
			"destinations": plan["destinations"],
		}
	}

	savedResponse := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+created["id"].(string), draftPayload(created, "save_draft"), owner)
	if savedResponse.Code != http.StatusOK {
		t.Fatalf("save draft status=%d body=%s", savedResponse.Code, savedResponse.Body.String())
	}
	saved := decodeObject(t, savedResponse)
	if saved["status"] != "draft" || saved["deletable"] != true {
		t.Fatalf("save_draft must ignore spoofed status: %#v", saved)
	}

	invalidPublish := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+saved["id"].(string), draftPayload(saved, "publish"), owner)
	if invalidPublish.Code != http.StatusBadRequest {
		t.Fatalf("invalid publish status=%d body=%s", invalidPublish.Code, invalidPublish.Body.String())
	}
	afterInvalid := decodeObject(t, requestJSON(t, router, http.MethodGet, "/api/v1/plans/"+saved["id"].(string), nil, owner))
	if afterInvalid["status"] != "draft" || afterInvalid["version"] != saved["version"] {
		t.Fatalf("invalid publish changed persisted plan: %#v", afterInvalid)
	}

	active := activateSingleDestination(t, router, owner, saved)
	if active["status"] != "active" || active["deletable"] != false {
		t.Fatalf("publish did not activate plan: %#v", active)
	}
	activeDelete := requestJSON(t, router, http.MethodDelete, "/api/v1/plans/"+active["id"].(string)+"/draft?version="+strconv.Itoa(int(active["version"].(float64))), nil, owner)
	if activeDelete.Code != http.StatusConflict {
		t.Fatalf("active draft-delete status=%d body=%s", activeDelete.Code, activeDelete.Body.String())
	}

	monthResponse := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+active["id"].(string)+"/months/2026-08", map[string]any{
		"incomeCents": 100000, "contributionBps": 10000, "expenses": []any{},
	}, owner)
	if monthResponse.Code != http.StatusOK {
		t.Fatalf("create history status=%d body=%s", monthResponse.Code, monthResponse.Body.String())
	}
	historicalDraftResponse := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+active["id"].(string), draftPayload(active, "save_draft"), owner)
	if historicalDraftResponse.Code != http.StatusOK {
		t.Fatalf("pause plan status=%d body=%s", historicalDraftResponse.Code, historicalDraftResponse.Body.String())
	}
	historicalDraft := decodeObject(t, historicalDraftResponse)
	if historicalDraft["status"] != "draft" || historicalDraft["deletable"] != false {
		t.Fatalf("historical draft must not be deletable: %#v", historicalDraft)
	}
	historyDelete := requestJSON(t, router, http.MethodDelete, "/api/v1/plans/"+historicalDraft["id"].(string)+"/draft?version="+strconv.Itoa(int(historicalDraft["version"].(float64))), nil, owner)
	if historyDelete.Code != http.StatusConflict {
		t.Fatalf("historical draft-delete status=%d body=%s", historyDelete.Code, historyDelete.Body.String())
	}

	archived := decodeObject(t, requestJSON(t, router, http.MethodPost, "/api/v1/plans", map[string]any{"name": "archived"}, owner))
	archiveResponse := requestJSON(t, router, http.MethodDelete, "/api/v1/plans/"+archived["id"].(string)+"?version="+strconv.Itoa(int(archived["version"].(float64))), nil, owner)
	if archiveResponse.Code != http.StatusNoContent {
		t.Fatalf("archive status=%d body=%s", archiveResponse.Code, archiveResponse.Body.String())
	}
	archivedCurrent := decodeObject(t, requestJSON(t, router, http.MethodGet, "/api/v1/plans/"+archived["id"].(string), nil, owner))
	archivedUpdate := requestJSON(t, router, http.MethodPut, "/api/v1/plans/"+archived["id"].(string), draftPayload(archivedCurrent, "save_draft"), owner)
	if archivedUpdate.Code != http.StatusConflict {
		t.Fatalf("archived plan update status=%d body=%s", archivedUpdate.Code, archivedUpdate.Body.String())
	}
	archivedDelete := requestJSON(t, router, http.MethodDelete, "/api/v1/plans/"+archived["id"].(string)+"/draft?version=2", nil, owner)
	if archivedDelete.Code != http.StatusConflict {
		t.Fatalf("archived draft-delete status=%d body=%s", archivedDelete.Code, archivedDelete.Body.String())
	}

	deletable := decodeObject(t, requestJSON(t, router, http.MethodPost, "/api/v1/plans", map[string]any{"name": "delete-me"}, owner))
	deletePath := "/api/v1/plans/" + deletable["id"].(string) + "/draft?version=" + strconv.Itoa(int(deletable["version"].(float64)))
	otherUserDelete := requestJSON(t, router, http.MethodDelete, deletePath, nil, outsider)
	if otherUserDelete.Code != http.StatusNotFound {
		t.Fatalf("cross-user draft-delete status=%d body=%s", otherUserDelete.Code, otherUserDelete.Body.String())
	}
	staleDelete := requestJSON(t, router, http.MethodDelete, "/api/v1/plans/"+deletable["id"].(string)+"/draft?version=2", nil, owner)
	if staleDelete.Code != http.StatusConflict {
		t.Fatalf("stale draft-delete status=%d body=%s", staleDelete.Code, staleDelete.Body.String())
	}
	deleted := requestJSON(t, router, http.MethodDelete, deletePath, nil, owner)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("draft-delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	missing := requestJSON(t, router, http.MethodGet, "/api/v1/plans/"+deletable["id"].(string), nil, owner)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("deleted plan still exists status=%d body=%s", missing.Code, missing.Body.String())
	}
	db, err := database.Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	var destinations int64
	if err := db.Model(&database.PlanDestination{}).Where("plan_id = ?", deletable["id"]).Count(&destinations).Error; err != nil {
		t.Fatal(err)
	}
	if destinations != 0 {
		t.Fatalf("deleted plan left %d destinations", destinations)
	}
}
