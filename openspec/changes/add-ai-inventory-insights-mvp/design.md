## Context

The core MVP already captures the business data needed to power AI-assisted analysis: products, stock levels, inventory transactions, purchase activity, and customer order history. The purpose of this change is to turn that operational data into plain-language recommendations that help small retail and e-commerce operators spot slow-moving stock, overstock exposure, and notable demand behavior without adding workflow automation risk.

This AI layer should remain advisory. It must not perform stock changes, supplier actions, or order updates automatically. The design needs to balance usefulness, trust, and cost, while fitting naturally into the existing dashboard and reporting model.

## Goals / Non-Goals

**Goals:**
- Generate AI insights from existing operational data with a clear summary of the analyzed time window and data basis.
- Surface high-value recommendations such as slow-moving items, overstock risk, and demand trend observations.
- Let users review insights in the application without granting the AI layer authority to change business records.
- Preserve enough generation metadata for auditability and troubleshooting.

**Non-Goals:**
- Automated purchasing, pricing, or stock adjustment actions driven by AI.
- Fine-tuned forecasting models or statistically rigorous demand planning in the first release.
- External market intelligence, competitor monitoring, or supplier performance prediction.
- Continuous real-time streaming analysis.

## Decisions

### Generate insights from curated operational summaries instead of raw table dumps

The AI workflow will assemble a structured summary of product performance, stock movement, sales activity, and reorder exposure before sending context to the model.

Rationale:
- Reduces prompt size and inference cost.
- Improves consistency across generations.
- Limits exposure of irrelevant or noisy data.

Alternatives considered:
- Sending raw operational records directly to the model was rejected because it is less predictable and more expensive.

### Keep the AI layer read-only and advisory

Generated insights will be viewable by users but cannot trigger stock mutations, order transitions, or supplier actions automatically.

Rationale:
- Protects inventory integrity and operator trust.
- Keeps accountability with human users.
- Reduces rollout risk for the MVP.

Alternatives considered:
- AI-driven automatic reorder suggestions with one-click execution was rejected for the initial release because it expands operational risk significantly.

### Persist insight runs with metadata and rendered output

Each insight generation will store the analysis timestamp, covered date range, key input summary, and the resulting recommendation set or narrative.

Rationale:
- Lets users compare recent runs and judge freshness.
- Supports debugging and product iteration.
- Makes the AI feature feel like a durable business tool rather than a transient chat response.

Alternatives considered:
- Returning ephemeral insights without persistence was rejected because it weakens trust and reviewability.

### Integrate the feature into reporting rather than a separate assistant workflow

The first release will expose AI insights through dashboard or reporting surfaces tied to inventory health.

Rationale:
- Matches the user’s mental model of reviewing business performance.
- Avoids introducing a separate conversational paradigm for a focused MVP feature.
- Reuses existing reporting navigation and permissions.

Alternatives considered:
- Building a free-form chat assistant first was rejected because it broadens scope and weakens task focus.

## Risks / Trade-offs

- [Insights may be generic or low-signal] → Constrain prompts to defined business questions and structured inputs.
- [Users may over-trust AI recommendations] → Present insights as advisory with visible supporting metrics and timestamps.
- [Model usage may increase operating cost] → Use refresh-on-demand behavior first and keep prompts compact.
- [Poor source data quality may produce misleading recommendations] → Depend on validated operational records and expose the analyzed time window.
- [Reporting UI may become crowded] → Separate AI insight cards or views from baseline KPI summaries.
