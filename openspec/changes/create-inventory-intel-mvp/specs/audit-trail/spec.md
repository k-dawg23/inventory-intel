## ADDED Requirements

### Requirement: The system records audit events for administrative actions
The system SHALL create an audit event for user-driven create, update, delete, status transition, and stock-affecting actions across core operational entities.

#### Scenario: Audit a product update
- **WHEN** a user updates a product's pricing or reorder settings
- **THEN** the system stores an audit event with actor, timestamp, target entity, action type, and changed values

#### Scenario: Audit an order status change
- **WHEN** a user changes a purchase order or customer order status
- **THEN** the system stores an audit event describing the transition and acting user

### Requirement: Users can review audit history
The system SHALL provide an audit log view that lists recorded events in reverse chronological order.

#### Scenario: View audit log
- **WHEN** a user opens the audit log
- **THEN** the system shows a time-ordered list of operational actions with enough detail to understand what changed
