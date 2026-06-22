## ADDED Requirements

### Requirement: The system records all inventory movements
The system SHALL create an inventory transaction record for each stock-affecting event, including stock received, stock sold, stock adjusted, damaged stock, and returned stock.

#### Scenario: Receive stock
- **WHEN** stock is received for a product through an approved workflow
- **THEN** the system records a received transaction with date, product, quantity, and acting user

#### Scenario: Adjust stock manually
- **WHEN** a user submits a manual stock adjustment with a quantity and reason
- **THEN** the system records an adjustment transaction and updates the product's current stock accordingly

### Requirement: The system preserves stock integrity
The system SHALL prevent stock reductions that would result in negative inventory unless an authorized adjustment workflow explicitly permits correction handling.

#### Scenario: Order exceeds stock on hand
- **WHEN** a stock-reducing workflow attempts to deduct more inventory than is currently available
- **THEN** the system rejects the transaction and leaves stock unchanged

#### Scenario: View transaction history
- **WHEN** a user opens a product's inventory history
- **THEN** the system shows the chronological list of stock transactions for that product
