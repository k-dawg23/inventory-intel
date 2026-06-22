## ADDED Requirements

### Requirement: Users can manage suppliers
The system SHALL allow authorized users to create, view, update, and search supplier records with name, contact details, and notes, and to associate suppliers with products they provide.

#### Scenario: Create a supplier
- **WHEN** a user submits supplier details with a name and contact information
- **THEN** the system stores the supplier and makes it available for purchase order workflows

#### Scenario: Link a supplier to products
- **WHEN** a user associates a supplier with one or more products
- **THEN** the system stores the relationship for supplier and product reference views

### Requirement: Users can manage purchase orders
The system SHALL allow authorized users to create purchase orders with statuses of Draft, Ordered, Received, and Cancelled.

#### Scenario: Create a draft purchase order
- **WHEN** a user creates a purchase order for a supplier with one or more line items
- **THEN** the system stores the purchase order with Draft status and its requested quantities

#### Scenario: Receive a purchase order
- **WHEN** a user marks an Ordered purchase order as Received
- **THEN** the system increases stock for each received line item and records corresponding inventory transactions

#### Scenario: Cancel a purchase order
- **WHEN** a user cancels a purchase order that has not been received
- **THEN** the system updates the order status to Cancelled without changing inventory
