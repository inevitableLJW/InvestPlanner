## 1. Frontend and Backend Foundation

- [x] 1.1 Create a React, TypeScript, and Vite frontend with development, build, lint, and test scripts.
- [x] 1.2 Create a Go module with Gin, GORM, MySQL driver, layered server packages, and development/test commands.
- [x] 1.3 Add local MySQL container configuration, example environment variables, health checks, and documented startup commands without committing secrets.
- [x] 1.4 Define the /api/v1 conventions for authentication, plan resources, nested plan subresources, validation errors, conflicts, and retryable server errors.
- [x] 1.5 Add responsive public and authenticated shells with routes for login, registration, plan list/create, plan details, plan settings, monthly entry, history, and statistics.

## 2. Authentication and User Isolation

- [x] 2.1 Add versioned MySQL migrations and GORM models for users and hashed, expiring, revocable sessions with required unique indexes.
- [x] 2.2 Implement username normalization, password-policy validation, bcrypt hashing, and registration that seeds user-level default expense sources but does not auto-create a plan.
- [x] 2.3 Implement login, current-user, and logout endpoints with generic credential failures and HttpOnly, SameSite, production-Secure session cookies.
- [x] 2.4 Implement Gin authentication middleware that resolves the current user exclusively from the session and rejects expired or revoked sessions.
- [x] 2.5 Configure production same-origin behavior, explicit credentialed development CORS, Origin checks for state-changing requests, and rate limits for registration and login.
- [x] 2.6 Add authentication tests for registration, duplicate usernames, password policy, login failure, session restore, expiry, logout, cookie attributes, and unauthorized access.
- [x] 2.7 Add two-user integration tests proving plans and all nested destinations, months, contributions, history, updates, deletions, and statistics cannot cross account boundaries.

## 3. Plan-Centered Domain and Calculation Engine

- [x] 3.1 Define Go domain models for plans, plan status, configurable destinations, expense sources, plan-scoped monthly cashflow, snapshots, actual amounts, and execution status.
- [x] 3.2 Implement plan validation for valid defaults, at least one active destination, unique active names, non-negative destination ratios, and a 10,000-basis-point active total.
- [x] 3.3 Implement plan lifecycle rules for draft, active, and archived plans, including archive-only handling for plans and destinations referenced by history.
- [x] 3.4 Implement monthly-input validation and the pure investable-base and recommended-total calculation using integer cents, basis points, and downward rounding.
- [x] 3.5 Implement deterministic largest-remainder allocation for any positive number of active destinations using sort order and stable ID as tie breakers.
- [x] 3.6 Implement derived execution-status and plan-scoped statistics functions, including cash, custom destinations, over-contribution, and zero recommendations.
- [x] 3.7 Add Go unit tests for zero/one/many destinations, subsets of defaults, custom and cash destinations, rounding ties, invalid plans, monthly overrides, statuses, and plan statistics.

## 4. MySQL Persistence and Plan-Scoped REST API

- [x] 4.1 Add versioned migrations and GORM models for plans, plan_destinations, expense_sources, monthly_records, monthly_expenses, and monthly_destination_allocations.
- [x] 4.2 Add user/plan ownership foreign keys, plan-scoped unique constraints, (plan_id, month) uniqueness, integer money columns, basis-point columns, historical snapshots, and version numbers.
- [x] 4.3 Implement repositories whose plan methods require user ID and whose nested-resource methods require both user ID and plan ID; never update or delete by a bare child ID.
- [x] 4.4 Implement transactional plan creation that inserts the editable 支付宝基金、A股、港美股、现金 destination template without assigning mandatory choices.
- [x] 4.5 Implement authenticated plan list/create/update/archive endpoints and destination add/rename/reorder/enable/disable/delete-or-archive endpoints.
- [x] 4.6 Implement active-plan validation and optimistic concurrency that returns 409 instead of silently overwriting newer plan or destination settings.
- [x] 4.7 Implement plan-nested month list/detail/upsert/delete endpoints, prior-month layout lookup, server-authoritative calculation, and snapshot-based historical recalculation.
- [x] 4.8 Implement transactional monthly writes for a variable number of destination snapshots and actual-amount updates that preserve historical destination structure.
- [x] 4.9 Implement plan-scoped dashboard endpoints plus an account plan-summary endpoint that does not aggregate cross-plan income or expenses.
- [x] 4.10 Add MySQL repository and API tests for transactions, rollback, ownership, two plans sharing a month, destination archival, snapshots, conflicts, deletion, and plan-scoped aggregation.

## 5. Frontend Authentication and API Integration

- [x] 5.1 Implement a typed frontend API client that sends session credentials and consistently handles 400, 401, 404, 409, and retryable server errors.
- [x] 5.2 Build registration and login forms with accessible validation, generic login failures, loading states, and safe post-login navigation.
- [x] 5.3 Implement current-user session bootstrap, protected routes, logout, and automatic navigation to login after a 401 response.
- [x] 5.4 Add frontend tests for registration, login, refresh restore, logout, expired sessions, and protected-route behavior.

## 6. Plan Creation and Destination Management UI

- [x] 6.1 Build the plan list and explicit “创建定投计划” entry so users select or create a plan before entering any monthly cashflow.
- [x] 6.2 Build plan creation that opens a draft containing editable 支付宝基金、A股、港美股、现金 destinations and makes clear that none of the four is mandatory.
- [x] 6.3 Build plan settings for name, reserve amount, rounding unit, preset/custom default rate, and draft/active/archive status.
- [x] 6.4 Build destination management for adding custom entries, renaming, sorting, enabling, disabling, deleting unused entries, and archiving historically referenced entries.
- [x] 6.5 Display the active allocation total, support a single 100% destination, and prevent monthly calculation until at least one active destination totals exactly 100%.
- [x] 6.6 Build plan-detail navigation for overview, settings, monthly entry, history, and statistics, and prevent new entry for draft or archived plans.
- [x] 6.7 Add component tests for initial defaults, removing defaults, custom destinations, one/many destinations, invalid totals, plan persistence, archive behavior, and plan switching.

## 7. Plan-Scoped Monthly Workflow, Tracking, and Statistics UI

- [x] 7.1 Build plan-detail month selection that creates or loads one editable record per (plan, YYYY-MM).
- [x] 7.2 Build total-income and per-source expense entry inside the selected plan, treat blanks as zero, show a live total, and display duplicate-source and cross-plan reuse guidance.
- [x] 7.3 Add copying of the same plan's prior-month source layout without amounts and add preset/custom monthly rate controls initialized from that plan.
- [x] 7.4 Submit inputs to the plan-scoped authoritative calculation API and render any number of destination results, including custom and cash destinations.
- [x] 7.5 Present formula, insufficient-surplus, rounding, allocation-total, and destination-remainder explanations without assuming three fixed outputs.
- [x] 7.6 Persist complete plan and destination snapshots and require confirmation before recalculating a historical month using its saved destination structure.
- [x] 7.7 Build actual-amount entry for every destination snapshot with validation, suggested-versus-actual differences, totals, and server-derived execution status.
- [x] 7.8 Build reverse-chronological plan history and detail views with editing and confirmed deletion.
- [x] 7.9 Build plan-scoped dashboard cards and dynamic destination visualizations, plus account plan cards that do not merge cross-plan income or expenses.
- [x] 7.10 Add empty states and the “不适用” completion-rate presentation when a plan's cumulative recommendations are zero.
- [x] 7.11 Add component and integration tests for plan switching, same month in two plans, monthly overrides, dynamic destinations, cash, snapshots, corrections, deletion, conflicts, and statistics.

## 8. Quality, Security, and Acceptance

- [x] 8.1 Verify desktop and mobile layouts and ensure forms, dialogs, dynamic destination controls, navigation, and visualizations are keyboard accessible.
- [x] 8.2 Add semantic labels, focus handling, validation summaries, and non-color-only indicators for plan, calculation, and execution states.
- [x] 8.3 Ensure logs and error responses never include passwords, password hashes, raw session tokens, MySQL credentials, or another user's plan data.
- [x] 8.4 Run and pass frontend tests, lint and type checks, Go tests and static checks, MySQL integration tests, and production builds.
- [x] 8.5 Manually verify all OpenSpec scenarios end to end, including plan-first onboarding, optional defaults, added destinations, cash, one-destination allocation, plan isolation, rollback, and historical stability.
- [x] 8.6 Document development setup, migrations and rollback, production environment variables, HTTPS/cookie requirements, and deployment order.
