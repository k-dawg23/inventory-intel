## MODIFIED Requirements

### Requirement: The system provides an operations dashboard
The system SHALL provide a dashboard that summarizes total products, suppliers, orders, low-stock items, inventory value, and the latest available AI insight status.

#### Scenario: View dashboard summaries
- **WHEN** a user opens the dashboard
- **THEN** the system displays current KPI counts and values derived from operational data

#### Scenario: View latest insight status
- **WHEN** AI insight data is available
- **THEN** the dashboard shows the latest generation timestamp or status for the inventory insight feature

### Requirement: The system provides operational reporting views
The system SHALL provide reporting views for orders over time, stock movement trends, top-selling products, and AI-generated inventory recommendations.

#### Scenario: View order trends
- **WHEN** a user views the reporting dashboard for a time period
- **THEN** the system displays aggregated order counts for that period

#### Scenario: View top-selling products
- **WHEN** a user requests top-selling product insights
- **THEN** the system ranks products by sold quantity within the selected reporting window

#### Scenario: View AI recommendations
- **WHEN** a user opens the reporting surface that includes AI insights
- **THEN** the system displays the latest generated recommendations with supporting operational context
