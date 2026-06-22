## ADDED Requirements

### Requirement: Users can manage a product catalog
The system SHALL allow authorized users to create, view, update, deactivate, and search products with SKU, name, description, category, unit cost, selling price, current stock, reorder level, and active status.

#### Scenario: Create a product
- **WHEN** a user submits a new product with all required fields and a unique SKU
- **THEN** the system creates the product and makes it available in product listings and operational workflows

#### Scenario: Prevent duplicate SKUs
- **WHEN** a user attempts to create or update a product with an SKU already assigned to another product
- **THEN** the system rejects the change and identifies the SKU conflict

#### Scenario: Filter active products
- **WHEN** a user filters the product catalog by status or searches by SKU or name
- **THEN** the system returns only matching products with current stock and pricing details

### Requirement: The system identifies products below reorder level
The system SHALL flag products whose current stock is below their configured reorder level.

#### Scenario: Product falls below reorder level
- **WHEN** a product's current stock becomes less than its reorder level
- **THEN** the system marks the product as low stock in product views and dashboard summaries
