## ADDED Requirements

### Requirement: Users can review AI insight outputs in the application
The system SHALL present AI-generated inventory recommendations in a dashboard or reporting surface with supporting context.

#### Scenario: View insight recommendations
- **WHEN** a user opens the insights surface after a successful generation
- **THEN** the system displays the recommendation summaries with their generation timestamp

#### Scenario: Show supporting business context
- **WHEN** the system presents a recommendation about stock or demand behavior
- **THEN** it includes the related product or metric context needed to interpret the recommendation

### Requirement: AI insights remain advisory
The system SHALL not allow AI insight outputs to directly modify products, stock levels, supplier records, or order records.

#### Scenario: Review advisory output
- **WHEN** a user reads an AI-generated recommendation
- **THEN** the system presents it as a recommendation without automatically executing any business action
