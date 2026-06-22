## ADDED Requirements

### Requirement: Users can manage customer orders
The system SHALL allow authorized users to create and update customer orders with statuses of Pending, Processing, Shipped, Completed, and Cancelled.

#### Scenario: Create a customer order
- **WHEN** a user creates a customer order with one or more product line items
- **THEN** the system stores the order with Pending status and its requested quantities

#### Scenario: Progress an order through fulfillment
- **WHEN** a user updates an order from Pending to Processing to Shipped to Completed
- **THEN** the system persists each valid status transition in order

#### Scenario: Reject invalid status transition
- **WHEN** a user attempts an unsupported order status transition
- **THEN** the system rejects the change and preserves the current order state

### Requirement: Customer order fulfillment deducts stock
The system SHALL deduct stock for customer order line items at the configured fulfillment transition and record the resulting inventory transactions.

#### Scenario: Deduct stock during fulfillment
- **WHEN** a customer order reaches the configured stock-deduction status with sufficient stock available
- **THEN** the system reduces inventory for each line item and records stock sold transactions

#### Scenario: Cancel before stock deduction
- **WHEN** a customer order is cancelled before the stock-deduction transition occurs
- **THEN** the system cancels the order without changing inventory
