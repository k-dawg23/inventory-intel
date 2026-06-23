# Inventory Intel

Inventory Intel is a small-business inventory and order management MVP for retail and e-commerce teams. It includes product catalog management, supplier and purchase order workflows, customer order processing, inventory ledger tracking, dashboard reporting, CSV import/export, and audit logging.

It also includes AI-powered inventory insights with two modes:
- `simulation` mode for demo output without external API calls
- `real` mode using the OpenAI Responses API when configured in `.env`

## Demo login

The MVP includes a single seeded administrator account:

- Email: `admin@inventoryintel.demo`
- Username: `kenneth`
- Password: `DemoAdmin123!`

## Run locally

```bash
cp .env.example .env
go mod tidy
go run ./cmd/inventory-intel
```

Open [http://localhost:8080](http://localhost:8080).

## AI insight configuration

`.env.example` contains the supported settings:

```text
AI_INSIGHTS_MODE=simulation
OPENAI_MODEL=gpt-5-nano
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_API_KEY=
```

Notes:
- Leave `AI_INSIGHTS_MODE=simulation` for demo mode.
- Set `AI_INSIGHTS_MODE=real` and provide `OPENAI_API_KEY` to enable live model-backed insights.
- `OPENAI_MODEL` defaults to `gpt-5-nano` and can be changed later without code changes.

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
