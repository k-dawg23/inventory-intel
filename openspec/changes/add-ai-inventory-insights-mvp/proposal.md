## Why

Small retail and e-commerce operators often know their current stock levels but still struggle to identify slow-moving items, overstock risk, and demand patterns early enough to act. This change adds AI-powered inventory insights to the MVP so the system can move beyond recordkeeping and provide decision support based on existing operational data.

## What Changes

- Introduce an AI insights capability that analyzes product, order, and inventory history to highlight slow-moving stock, overstock risk, and notable demand patterns.
- Add an insights dashboard or panel that presents generated recommendations in operator-friendly language with supporting business context.
- Add configurable insight generation triggers so users can refresh analysis on demand and the system can support scheduled analysis later.
- Add traceability for insight outputs, including source data windows, generation timestamps, and model response storage or summaries.
- Add guardrails that prevent the AI layer from mutating inventory or order data directly.

## Capabilities

### New Capabilities
- `ai-inventory-insights`: Generate actionable inventory recommendations from historical product, stock movement, and order data.
- `insight-review-workflow`: Present, review, and track AI-generated insight outputs with supporting context and timestamps.

### Modified Capabilities
- `inventory-reporting`: Extend reporting surfaces to include AI-generated insights alongside operational metrics.

## Impact

- Adds an AI-assisted analysis layer on top of the operational inventory platform.
- Requires prompt design, model integration, insight persistence, and UI presentation for generated recommendations.
- Depends on reliable operational data from products, orders, stock movements, and reporting summaries.
- Introduces AI-specific concerns around freshness, explainability, cost control, and user trust.
