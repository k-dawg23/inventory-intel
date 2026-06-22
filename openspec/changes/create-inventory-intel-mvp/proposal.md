## Why

Small retail and e-commerce businesses often manage stock, suppliers, and orders across spreadsheets or disconnected tools, which makes stock accuracy, reorder planning, and order fulfillment error-prone. This change defines a focused inventory management MVP that demonstrates a realistic business workflow with clear operational value and a path to future expansion.

## What Changes

- Introduce a product catalog for SKUs, pricing, stock levels, reorder thresholds, and active status management.
- Add an inventory movement ledger that records every stock-affecting event, including receiving, selling, returns, damage, and manual adjustments.
- Add supplier management and purchase order workflows so inbound stock can be ordered and received in a controlled way.
- Add customer order management with status progression and automatic stock deduction from sellable inventory.
- Add an operations dashboard with low-stock alerts, inventory valuation, order summaries, and top-product reporting.
- Add CSV import and export flows for key business records to support onboarding and reporting.
- Add an audit trail for administrative changes across products, orders, inventory, and supplier workflows.
- Keep AI-driven inventory insights as a separate MVP proposal so the operational core and the analytics layer can evolve independently.

## Capabilities

### New Capabilities
- `product-catalog`: Manage products with SKU, descriptive metadata, pricing, current stock, reorder levels, and lifecycle status.
- `inventory-ledger`: Record and validate all stock movements as auditable inventory transactions.
- `supplier-procurement`: Manage suppliers and purchase orders, including receiving stock into inventory.
- `customer-order-processing`: Manage customer orders through operational statuses and deduct stock from inventory.
- `inventory-reporting`: Provide dashboards, low-stock visibility, inventory valuation, and sales or movement summaries.
- `csv-data-exchange`: Support CSV import and export for operational data such as products, suppliers, inventory, and orders.
- `audit-trail`: Capture user actions and before or after state changes for operational accountability.

### Modified Capabilities

None.

## Impact

- Establishes the initial product contract for the Inventory Intel application.
- Requires new data models for products, suppliers, purchase orders, customer orders, inventory transactions, and audit events.
- Requires backend workflows for stock validation, transactional updates, and reporting queries.
- Requires frontend administrative interfaces for CRUD operations, dashboards, and operational status management.
- Introduces CSV processing and reporting export behavior that must be validated for data integrity.
