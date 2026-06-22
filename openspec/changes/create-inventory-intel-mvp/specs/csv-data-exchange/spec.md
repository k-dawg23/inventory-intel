## ADDED Requirements

### Requirement: Users can import operational data from CSV
The system SHALL allow authorized users to import products and suppliers from CSV files using a validated file structure.

#### Scenario: Import valid product CSV
- **WHEN** a user uploads a valid product import file
- **THEN** the system creates or updates products according to the import rules and reports the processed row counts

#### Scenario: Reject invalid import rows
- **WHEN** a CSV file contains missing required columns or invalid field values
- **THEN** the system rejects the affected rows and reports the validation errors to the user

### Requirement: Users can export operational data to CSV
The system SHALL allow authorized users to export products, inventory views, customer orders, and reporting datasets to CSV.

#### Scenario: Export inventory data
- **WHEN** a user requests an inventory export
- **THEN** the system generates a CSV file containing the selected inventory records

#### Scenario: Export order data
- **WHEN** a user requests a customer order export for a date range
- **THEN** the system generates a CSV file containing the matching order records
