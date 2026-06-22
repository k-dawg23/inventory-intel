## ADDED Requirements

### Requirement: The system provides an operations dashboard
The system SHALL provide a dashboard that summarizes total products, suppliers, orders, low-stock items, and inventory value.

#### Scenario: View dashboard summaries
- **WHEN** a user opens the dashboard
- **THEN** the system displays current KPI counts and values derived from operational data

### Requirement: The system provides operational reporting views
The system SHALL provide reporting views for orders over time, stock movement trends, and top-selling products.

#### Scenario: View order trends
- **WHEN** a user views the reporting dashboard for a time period
- **THEN** the system displays aggregated order counts for that period

#### Scenario: View top-selling products
- **WHEN** a user requests top-selling product insights
- **THEN** the system ranks products by sold quantity within the selected reporting window

### Requirement: The system surfaces low-stock alerts
The system SHALL display low-stock items in dashboard and operational list views.

#### Scenario: Show low-stock alert count
- **WHEN** one or more products are below reorder level
- **THEN** the dashboard shows the number of low-stock products and links to the affected items
