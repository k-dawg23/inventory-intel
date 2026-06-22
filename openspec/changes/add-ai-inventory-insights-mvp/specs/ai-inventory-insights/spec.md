## ADDED Requirements

### Requirement: The system generates AI-powered inventory insights
The system SHALL generate inventory insight outputs from historical product, inventory, and order data to identify slow-moving stock, overstock risk, and notable demand patterns.

#### Scenario: Generate insights successfully
- **WHEN** a user requests an inventory insight refresh for a valid analysis period
- **THEN** the system produces AI-generated recommendations based on the available operational data

#### Scenario: Limit insight scope to supported data
- **WHEN** the system prepares an AI insight request
- **THEN** it includes only the supported operational summary inputs required for inventory analysis

### Requirement: The system preserves AI insight traceability
The system SHALL store each generated insight run with its generation timestamp, covered data window, and resulting recommendations.

#### Scenario: Review a prior insight run
- **WHEN** a user opens a previously generated insight entry
- **THEN** the system shows the stored recommendations and the date range used for analysis
