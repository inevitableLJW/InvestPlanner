## 1. Plan defaults and lifecycle contract

- [x] 1.1 Change the repository creation value and GORM model default for `rounding_unit_cents` to 10,000 cents, while leaving existing plan rows untouched.
- [x] 1.2 Add a validated lifecycle action (`save_draft` or `publish`) to the plan settings write contract and derive the persisted plan status on the server instead of accepting a client-selected status.
- [x] 1.3 Make `save_draft` persist valid editable settings even when allocations are incomplete, and make `publish` atomically persist settings, run full plan validation, and activate the plan under the existing optimistic-lock transaction.
- [x] 1.4 Add backend tests proving new plans default to 100 yuan, draft saves do not activate, valid publishes activate, invalid publishes remain inactive, and saving an active plan as draft pauses new monthly calculation.
- [x] 1.5 Add a server-derived `deletable` flag and an authenticated permanent-draft-delete command that preserves the existing archive endpoint, requires the expected version, and transactionally deletes only an owned `draft` plan with zero monthly records and its destinations.
- [x] 1.6 Add repository, service, and HTTP tests proving eligible drafts are removed completely while active, archived, historical, stale-version, and other-user plans are rejected without partial deletion.

## 2. Cash remainder allocation

- [x] 2.1 Update the domain allocation algorithm to identify the single enabled canonical `现金` destination, round every non-cash theoretical allocation down to the plan unit, and assign the exact remaining total to cash.
- [x] 2.2 Preserve the deterministic largest-remainder allocation path when no enabled cash destination exists and keep all allocation amounts non-negative with an exact total.
- [x] 2.3 Add domain tests for cash plus multiple targets, cash as the only target, disabled or absent cash, stable no-cash tie breaking, and totals smaller than or equal to one rounding unit.
- [x] 2.4 Update the month result calculation explanation so users can tell when cash includes the per-target rounding remainder.

## 3. Plan settings experience

- [x] 3.1 Replace the status-editing/single-save interaction with separate “保存草稿” and “发布计划” submissions that send the same edited settings with the appropriate lifecycle action.
- [x] 3.2 Allow draft saving with incomplete allocations, disable publishing when locally known requirements are incomplete, and keep server validation errors and unsaved field values visible after a failed publish.
- [x] 3.3 Add distinct busy and confirmation states for draft save versus publish, including a warning that saving a running plan as draft pauses monthly entry and success text that states whether the plan has started.
- [x] 3.4 Ensure a newly created plan displays a 100-yuan rounding unit and update plan overview/month-entry guidance to treat only published `active` plans as started.
- [x] 3.5 Add frontend interaction tests for the two action payloads, incomplete draft saving, publish gating and feedback, active-to-draft warning, and the 100-yuan default display.
- [x] 3.6 Show a “删除草稿计划” danger action only when the server marks the plan deletable, require an irreversible-action confirmation containing the plan name, and return to the plan list after successful deletion.
- [x] 3.7 Add frontend tests for displaying and cancelling eligible deletion, successful navigation after deletion, hiding the action for ineligible plans, duplicate-submit prevention, and stale-eligibility error feedback.

## 4. Integration and verification

- [x] 4.1 Run backend domain, service, repository, and HTTP API tests covering plan writes, permanent draft deletion, archive preservation, and monthly allocation snapshots.
- [x] 4.2 Run the frontend test suite, lint, production build, and `git diff --check`, fixing any lifecycle-label or accessibility regressions.
- [ ] 4.3 Manually verify the end-to-end flow: create a 100-yuan-default plan, save and permanently delete an unstarted draft, confirm active and historical plans cannot be permanently deleted, publish a valid plan, calculate a month with cash remainder, and confirm existing plans/history retain their saved values.

## 5. List deletion, income expressions, and exact cash remainder

- [x] 5.1 Move the eligible draft deletion action and irreversible confirmation from plan settings to the plan-list card, retaining server-side eligibility and optimistic-version protection.
- [x] 5.2 Allow the monthly income field to parse safe simple addition/subtraction expressions and use the evaluated amount for preview and submission validation.
- [x] 5.3 Keep the investable preview exact and, when cash is active, keep the backend recommendation total exact while assigning all non-cash rounding and total loose change to cash; retain the no-cash whole-unit fallback.
- [x] 5.4 Add focused frontend and domain regression tests, then run backend and frontend verification for the revised behavior.
- [x] 5.5 Submit the evaluated income expression result as `incomeCents`, show its formatted current total below the field, and add a regression assertion for the request payload.
- [x] 5.6 Guard destructive integration setup so it refuses any database whose name does not end in `_test`, update the documented DSN, and verify the guard without connecting to or clearing the development database.
