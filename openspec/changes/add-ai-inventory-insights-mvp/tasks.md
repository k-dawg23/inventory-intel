## 1. Insight Data Preparation

- [x] 1.1 Define the summarized inventory, sales, and stock-health inputs required for AI analysis
- [x] 1.2 Implement backend aggregation logic that prepares structured insight input data from operational records
- [x] 1.3 Add persistence for insight runs, timestamps, analyzed windows, and generated outputs

## 2. AI Generation Workflow

- [x] 2.1 Implement the model integration and prompt design for slow-moving stock, overstock risk, and demand-pattern analysis
- [x] 2.2 Add refresh-on-demand workflow and error handling for failed or incomplete generations
- [x] 2.3 Enforce read-only guardrails so AI outputs cannot mutate operational data

## 3. Product Surface And Validation

- [x] 3.1 Extend reporting or dashboard views to display latest insight status and recommendation outputs
- [x] 3.2 Add insight detail views with supporting business context and historical run metadata
- [x] 3.3 Add automated tests for prompt input shaping, persistence, advisory-only behavior, and UI presentation states
