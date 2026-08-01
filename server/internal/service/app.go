package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"investplanner/server/internal/database"
	"investplanner/server/internal/domain"
	"investplanner/server/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrValidation         = errors.New("validation error")
)

var defaultSources = []string{"微信", "支付宝", "美团", "京东", "抖音", "银行卡", "交通", "其他"}
var passwordLetter = regexp.MustCompile(`[A-Za-z]`)
var passwordDigit = regexp.MustCompile(`[0-9]`)
var validUsername = regexp.MustCompile(`^[\p{L}\p{N}_-]{3,32}$`)
var defaultDestinations = []string{"现金", "债券类基金", "纳斯达克100指数（QDII）", "标普500指数（QDII）"}

type App struct {
	Store      *repository.Store
	SessionTTL time.Duration
	Now        func() time.Time
}

type AuthResult struct {
	User  database.User
	Token string
}

func New(store *repository.Store, sessionTTL time.Duration) *App {
	return &App{Store: store, SessionTTL: sessionTTL, Now: time.Now}
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func (a *App) Register(username, password string) (AuthResult, error) {
	username = normalizeUsername(username)
	if !validUsername.MatchString(username) {
		return AuthResult{}, fmt.Errorf("%w: 用户名需为 3–32 位中英文、数字、下划线或短横线", ErrValidation)
	}
	if len(password) < 10 || len([]byte(password)) > 72 || !passwordLetter.MatchString(password) || !passwordDigit.MatchString(password) {
		return AuthResult{}, fmt.Errorf("%w: password must be 10 characters or more and contain a letter and number", ErrValidation)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, err
	}
	user, err := a.Store.CreateUser(username, string(hash), defaultSources)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return AuthResult{}, repository.ErrConflict
		}
		return AuthResult{}, err
	}
	return a.createAuthResult(user)
}

func (a *App) Login(username, password string) (AuthResult, error) {
	user, err := a.Store.FindUserByUsername(username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$012345678901234567890u27J4QdIhA8wJnP7fS8K7YjQe7rC"), []byte(password))
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return AuthResult{}, ErrInvalidCredentials
	}
	return a.createAuthResult(user)
}

func (a *App) createAuthResult(user database.User) (AuthResult, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return AuthResult{}, err
	}
	token := hex.EncodeToString(raw)
	if _, err := a.Store.CreateSession(user.ID, HashToken(token), a.Now().Add(a.SessionTTL)); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{User: user, Token: token}, nil
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (a *App) Authenticate(token string) (database.User, error) {
	if token == "" {
		return database.User{}, repository.ErrNotFound
	}
	session, err := a.Store.FindActiveSession(HashToken(token), a.Now())
	if err != nil {
		return database.User{}, err
	}
	return session.User, nil
}

func (a *App) Logout(token string) error {
	if token == "" {
		return nil
	}
	return a.Store.RevokeSession(HashToken(token))
}

type DestinationInput struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Active        bool   `json:"active"`
	AllocationBPS int    `json:"allocationBps"`
	SortOrder     int    `json:"sortOrder"`
}

type PlanInput struct {
	Name                   string             `json:"name"`
	Status                 domain.PlanStatus  `json:"status"`
	DefaultContributionBPS int                `json:"defaultContributionBps"`
	ReserveCents           int64              `json:"reserveCents"`
	RoundingUnitCents      int64              `json:"roundingUnitCents"`
	Version                int                `json:"version"`
	Destinations           []DestinationInput `json:"destinations"`
}

func (a *App) CreatePlan(userID, name string) (database.Plan, error) {
	if strings.TrimSpace(name) == "" {
		return database.Plan{}, fmt.Errorf("%w: plan name is required", ErrValidation)
	}
	return a.Store.CreatePlan(userID, name, defaultDestinations)
}

func (a *App) UpdatePlan(userID, planID string, input PlanInput) (database.Plan, error) {
	plan := domain.Plan{
		ID: planID, UserID: userID, Name: input.Name, Status: input.Status,
		DefaultContributionBPS: input.DefaultContributionBPS, ReserveCents: input.ReserveCents,
		RoundingUnitCents: input.RoundingUnitCents, Version: input.Version,
	}
	modelDestinations := make([]database.PlanDestination, 0, len(input.Destinations))
	for _, destination := range input.Destinations {
		plan.Destinations = append(plan.Destinations, domain.Destination{
			ID: destination.ID, Name: destination.Name, Active: destination.Active,
			AllocationBPS: destination.AllocationBPS, SortOrder: destination.SortOrder,
		})
		modelDestinations = append(modelDestinations, database.PlanDestination{
			ID: destination.ID, PlanID: planID, Name: destination.Name, Active: destination.Active,
			AllocationBPS: destination.AllocationBPS, SortOrder: destination.SortOrder,
		})
	}
	if input.Status == domain.PlanActive {
		if err := domain.ValidatePlan(plan); err != nil {
			return database.Plan{}, fmt.Errorf("%w: %v", ErrValidation, err)
		}
	} else {
		if strings.TrimSpace(input.Name) == "" || input.DefaultContributionBPS < 0 || input.DefaultContributionBPS > 10000 ||
			!domain.ValidateMoney(input.ReserveCents) || input.RoundingUnitCents <= 0 {
			return database.Plan{}, fmt.Errorf("%w: invalid draft plan", ErrValidation)
		}
	}
	modelPlan := database.Plan{
		ID: planID, UserID: userID, Name: input.Name, Status: string(input.Status),
		DefaultContributionBPS: input.DefaultContributionBPS, ReserveCents: input.ReserveCents,
		RoundingUnitCents: input.RoundingUnitCents, Version: input.Version,
	}
	return a.Store.UpdatePlan(userID, modelPlan, modelDestinations, input.Version)
}

type ExpenseRequest struct {
	SourceID    string `json:"sourceId"`
	SourceName  string `json:"sourceName"`
	AmountCents int64  `json:"amountCents"`
	SortOrder   int    `json:"sortOrder"`
}

type MonthRequest struct {
	Month           string           `json:"month"`
	IncomeCents     int64            `json:"incomeCents"`
	ContributionBPS int              `json:"contributionBps"`
	Expenses        []ExpenseRequest `json:"expenses"`
	Note            string           `json:"note"`
	Version         *int             `json:"version"`
}

func planToDomain(plan database.Plan) domain.Plan {
	result := domain.Plan{
		ID: plan.ID, UserID: plan.UserID, Name: plan.Name, Status: domain.PlanStatus(plan.Status),
		DefaultContributionBPS: plan.DefaultContributionBPS, ReserveCents: plan.ReserveCents,
		RoundingUnitCents: plan.RoundingUnitCents, Version: plan.Version,
	}
	for _, destination := range plan.Destinations {
		result.Destinations = append(result.Destinations, domain.Destination{
			ID: destination.ID, Name: destination.Name, Active: destination.Active,
			Archived: destination.ArchivedAt != nil, SortOrder: destination.SortOrder,
			AllocationBPS: destination.AllocationBPS,
		})
	}
	return result
}

func (a *App) UpsertMonth(userID, planID string, request MonthRequest) (database.MonthlyRecord, error) {
	planModel, err := a.Store.GetPlan(userID, planID)
	if err != nil {
		return database.MonthlyRecord{}, err
	}
	plan := planToDomain(planModel)
	if plan.Status != domain.PlanActive {
		return database.MonthlyRecord{}, fmt.Errorf("%w: plan is not active", ErrValidation)
	}

	destinations := plan.Destinations
	actuals := map[string]int64{}
	existing, existingErr := a.Store.GetMonth(userID, planID, request.Month)
	if existingErr == nil {
		destinations = nil
		for _, allocation := range existing.Allocations {
			destinationID := ""
			if allocation.DestinationID != nil {
				destinationID = *allocation.DestinationID
			}
			destinations = append(destinations, domain.Destination{
				ID: destinationID, Name: allocation.DestinationName, Active: true,
				SortOrder: allocation.SortOrder, AllocationBPS: allocation.AllocationBPS,
			})
			actuals[destinationID] = allocation.ActualCents
		}
	} else if !errors.Is(existingErr, repository.ErrNotFound) {
		return database.MonthlyRecord{}, existingErr
	}

	expenses := make([]domain.ExpenseInput, 0, len(request.Expenses))
	for _, expense := range request.Expenses {
		expenses = append(expenses, domain.ExpenseInput{
			SourceID: expense.SourceID, SourceName: expense.SourceName, AmountCents: expense.AmountCents,
		})
	}
	calculation, err := domain.Calculate(plan, domain.MonthlyInput{
		Month: request.Month, IncomeCents: request.IncomeCents, ContributionBPS: request.ContributionBPS,
		Expenses: expenses, Note: request.Note,
	}, destinations)
	if err != nil {
		return database.MonthlyRecord{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	record := database.MonthlyRecord{
		PlanID: planID, Month: request.Month, IncomeCents: calculation.IncomeCents,
		ExpenseTotalCents: calculation.ExpenseTotalCents, ReserveCents: calculation.ReserveCents,
		SurplusCents: calculation.SurplusCents, InvestableBaseCents: calculation.InvestableBaseCents,
		ContributionBPS: calculation.ContributionBPS, RecommendedTotalCents: calculation.RecommendedTotalCents,
		RoundingUnitCents: calculation.RoundingUnitCents, Status: string(calculation.Status), Note: request.Note,
	}
	expenseModels := make([]database.MonthlyExpense, 0, len(request.Expenses))
	for _, expense := range request.Expenses {
		var sourceID *string
		if expense.SourceID != "" {
			value := expense.SourceID
			sourceID = &value
		}
		expenseModels = append(expenseModels, database.MonthlyExpense{
			SourceID: sourceID, SourceName: expense.SourceName, AmountCents: expense.AmountCents, SortOrder: expense.SortOrder,
		})
	}
	allocationModels := make([]database.MonthlyDestinationAllocation, 0, len(calculation.Allocations))
	for _, allocation := range calculation.Allocations {
		var destinationID *string
		if allocation.DestinationID != "" {
			value := allocation.DestinationID
			destinationID = &value
		}
		allocationModels = append(allocationModels, database.MonthlyDestinationAllocation{
			DestinationID: destinationID, DestinationName: allocation.Name, SortOrder: allocation.SortOrder,
			AllocationBPS: allocation.AllocationBPS, RecommendedCents: allocation.RecommendedCents,
			ActualCents: actuals[allocation.DestinationID],
		})
	}
	actualTotal := int64(0)
	for _, allocation := range allocationModels {
		actualTotal += allocation.ActualCents
	}
	record.Status = string(domain.DeriveExecutionStatus(record.RecommendedTotalCents, actualTotal))
	return a.Store.SaveMonth(userID, record, expenseModels, allocationModels, request.Version)
}

func RecordToCalculation(record database.MonthlyRecord) domain.Calculation {
	result := domain.Calculation{
		Month: record.Month, IncomeCents: record.IncomeCents, ExpenseTotalCents: record.ExpenseTotalCents,
		ReserveCents: record.ReserveCents, SurplusCents: record.SurplusCents,
		InvestableBaseCents: record.InvestableBaseCents, ContributionBPS: record.ContributionBPS,
		RecommendedTotalCents: record.RecommendedTotalCents, RoundingUnitCents: record.RoundingUnitCents,
		Status: domain.ExecutionStatus(record.Status),
	}
	for _, allocation := range record.Allocations {
		destinationID := ""
		if allocation.DestinationID != nil {
			destinationID = *allocation.DestinationID
		}
		result.Allocations = append(result.Allocations, domain.DestinationAllocation{
			DestinationID: destinationID, Name: allocation.DestinationName, SortOrder: allocation.SortOrder,
			AllocationBPS: allocation.AllocationBPS, RecommendedCents: allocation.RecommendedCents,
			ActualCents: allocation.ActualCents,
		})
	}
	return result
}

func (a *App) PlanStats(userID, planID string) (domain.PlanStats, error) {
	if err := a.Store.EnsurePlanOwned(userID, planID); err != nil {
		return domain.PlanStats{}, err
	}
	records, err := a.Store.ListMonths(userID, planID)
	if err != nil {
		return domain.PlanStats{}, err
	}
	calculations := make([]domain.Calculation, 0, len(records))
	for _, record := range records {
		calculations = append(calculations, RecordToCalculation(record))
	}
	return domain.BuildStats(calculations), nil
}
