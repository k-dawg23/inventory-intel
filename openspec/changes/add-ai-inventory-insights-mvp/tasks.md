## 1. Insight Data Preparation

- [ ] 1.1 Define the summarized inventory, sales, and stock-health inputs required for AI analysis
- [ ] 1.2 Implement backend aggregation logic that prepares structured insight input data from operational records
- [ ] 1.3 Add persistence for insight runs, timestamps, analyzed windows, and generated outputs

## 2. AI Generation Workflow

- [ ] 2.1 Implement the model integration and prompt design for slow-moving stock, overstock risk, and demand-pattern analysis
- [ ] 2.2 Add refresh-on-demand workflow and error handling for failed or incomplete generations
- [ ] 2.3 Enforce read-only guardrails so AI outputs cannot mutate operational data

## 3. Product Surface And Validation

- [ ] 3.1 Extend reporting or dashboard views to display latest insight status and recommendation outputs
- [ ] 3.2 Add insight detail views with supporting business context and historical run metadata
- [ ] 3.3 Add automated tests for prompt input shaping, persistence, advisory-only behavior, and UI presentation states
