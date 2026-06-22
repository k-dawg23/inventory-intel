# Inventory Intel

Inventory Intel is a small-business inventory and order management MVP for retail and e-commerce teams. It includes product catalog management, supplier and purchase order workflows, customer order processing, inventory ledger tracking, dashboard reporting, CSV import/export, and audit logging.

## Run locally

```bash
go mod tidy
go run ./cmd/inventory-intel
```

Open [http://localhost:8080](http://localhost:8080).

## Docker

```bash
docker compose up --build
```

## CSV formats

Products header:

```text
sku,name,description,category,unit_cost,selling_price,current_stock,reorder_level,active
```

Suppliers header:

```text
name,contact_name,email,phone,notes
```
