## Context

The system is intended for small retail and e-commerce businesses that need a practical way to manage products, inbound purchasing, outbound orders, and stock visibility without the complexity of multi-warehouse, manufacturing, or food-distribution systems. The MVP should feel commercially realistic, emphasize auditability, and support CRUD-heavy operational workflows with clear reporting outcomes.

The proposed technical direction from the source material is a Go backend with Chi, SQLite for MVP persistence, a React and TypeScript frontend, Tailwind for UI styling, and Docker Compose for local deployment. The core design challenge is preserving inventory accuracy while keeping workflows simple enough for a first release.

## Goals / Non-Goals

**Goals:**
- Provide a single-product-location inventory model that is easy to understand and test.
- Use a transaction-ledger approach so stock-affecting actions are recorded explicitly and can be audited.
- Support the main operational loop of supplier to purchase order to inventory and inventory to customer order to stock reduction.
- Provide reporting surfaces that make the application useful to operators and compelling as a portfolio-quality business tool.
- Keep the data model and workflows extensible for future additions such as barcode scanning, multiple warehouses, and forecasting.

**Non-Goals:**
- Multi-warehouse inventory, location transfers, or bin-level storage.
- Manufacturing workflows such as bills of materials or production runs.
- Batch, lot, or expiry tracking needed for food or regulated inventory.
- Complex integrations with marketplaces, carriers, accounting tools, or third-party ERPs.
- AI-generated insights implementation within this specific change, which is covered by a separate MVP proposal.

## Decisions

### Use an inventory ledger as the source of stock movement truth

Inventory changes will be recorded as explicit transaction records for receiving, sales, adjustments, damage, and returns. Product stock on hand can be stored as a denormalized current value for fast reads, but every change must be backed by a ledger entry.

Rationale:
- Supports auditability and operational troubleshooting.
- Makes reporting on stock movement possible without reconstructing behavior from unrelated tables.
- Reduces ambiguity around why stock changed.

Alternatives considered:
- Updating stock counts directly on products without a ledger was rejected because it weakens traceability.
- Computing stock exclusively from ledger aggregation was rejected for MVP because it complicates read performance and common list views.

### Keep the inventory model single-location for MVP

All stock will belong to one logical inventory pool. Purchase orders increase that pool and customer orders reduce it.

Rationale:
- Fits the target audience and the attached scope guidance.
- Avoids premature complexity in transfer logic, reporting, and UI.
- Keeps future expansion possible by adding location dimensions later.

Alternatives considered:
- Supporting multiple warehouses from day one was rejected because it expands nearly every workflow and report.

### Use stateful order workflows with inventory side effects at controlled transitions

Purchase orders and customer orders will use status lifecycles. Inventory updates occur only at defined transitions, such as receiving a purchase order or confirming an order progression that reserves or deducts stock.

Rationale:
- Prevents accidental double-counting from repeated edits.
- Matches how operators reason about operational progress.
- Makes behavior easier to validate with scenario-based tests.

Alternatives considered:
- Allowing free-form status changes with manual stock edits was rejected because it increases operator error and weakens system guarantees.

### Treat CSV exchange as a first-class operational capability

CSV import and export will be supported for a narrow set of high-value data domains, starting with products and suppliers for import and products, inventory, orders, and reports for export.

Rationale:
- Reflects common small-business adoption patterns.
- Improves onboarding and interoperability without full integrations.
- Strengthens the product’s practical value for the target market.

Alternatives considered:
- Deferring CSV support entirely was rejected because it is a common operational expectation for this type of product.

### Record audit events for user-driven mutations

Create, update, status transition, and stock-affecting actions will generate audit entries that capture actor, action, target entity, timestamp, and key before or after values where relevant.

Rationale:
- Supports accountability and debugging.
- Improves trust for administrative workflows.
- Complements the inventory ledger by covering non-stock operational changes.

Alternatives considered:
- Limiting audit behavior to authentication events was rejected because it would miss the most business-critical actions.

## Risks / Trade-offs

- [Inventory drift from inconsistent workflow handling] → Centralize stock mutations behind service-layer transaction rules and validate side effects at status transitions.
- [SQLite limitations as data volume grows] → Keep the MVP schema portable and isolate persistence concerns so PostgreSQL can replace SQLite later.
- [CSV imports causing bad data or duplicate records] → Validate headers, required fields, and row-level errors before applying imports.
- [Operators expecting advanced workflows too early] → Make non-goals explicit in product positioning and future enhancement notes.
- [Reporting queries becoming expensive] → Start with scoped summary queries and precomputed current stock fields rather than building generalized analytics infrastructure.
