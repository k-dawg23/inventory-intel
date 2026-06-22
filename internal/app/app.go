package app

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Config struct {
	DBPath string
	AI     AIConfig
}

type App struct {
	db *sql.DB
	ai AIConfig
}

type Product struct {
	ID           int64   `json:"id"`
	SKU          string  `json:"sku"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Category     string  `json:"category"`
	UnitCost     float64 `json:"unitCost"`
	SellingPrice float64 `json:"sellingPrice"`
	CurrentStock int     `json:"currentStock"`
	ReorderLevel int     `json:"reorderLevel"`
	Active       bool    `json:"active"`
	LowStock     bool    `json:"lowStock"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

type Supplier struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	ContactName string  `json:"contactName"`
	Email       string  `json:"email"`
	Phone       string  `json:"phone"`
	Notes       string  `json:"notes"`
	ProductIDs  []int64 `json:"productIds"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

type PurchaseOrderItem struct {
	ProductID int64   `json:"productId"`
	Quantity  int     `json:"quantity"`
	UnitCost  float64 `json:"unitCost"`
}

type PurchaseOrder struct {
	ID         int64               `json:"id"`
	SupplierID int64               `json:"supplierId"`
	Supplier   string              `json:"supplier"`
	Status     string              `json:"status"`
	Notes      string              `json:"notes"`
	Items      []PurchaseOrderItem `json:"items"`
	CreatedAt  string              `json:"createdAt"`
	UpdatedAt  string              `json:"updatedAt"`
}

type CustomerOrderItem struct {
	ProductID  int64   `json:"productId"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unitPrice"`
	ProductSKU string  `json:"productSku,omitempty"`
	Product    string  `json:"product,omitempty"`
}

type CustomerOrder struct {
	ID           int64               `json:"id"`
	CustomerName string              `json:"customerName"`
	Status       string              `json:"status"`
	Notes        string              `json:"notes"`
	Items        []CustomerOrderItem `json:"items"`
	CreatedAt    string              `json:"createdAt"`
	UpdatedAt    string              `json:"updatedAt"`
}

type InventoryTransaction struct {
	ID              int64  `json:"id"`
	ProductID       int64  `json:"productId"`
	ProductName     string `json:"productName"`
	ProductSKU      string `json:"productSku"`
	TransactionType string `json:"transactionType"`
	Quantity        int    `json:"quantity"`
	ReferenceType   string `json:"referenceType"`
	ReferenceID     int64  `json:"referenceId"`
	Reason          string `json:"reason"`
	Actor           string `json:"actor"`
	CreatedAt       string `json:"createdAt"`
}

type AuditEvent struct {
	ID         int64  `json:"id"`
	Actor      string `json:"actor"`
	EntityType string `json:"entityType"`
	EntityID   int64  `json:"entityId"`
	Action     string `json:"action"`
	Details    string `json:"details"`
	CreatedAt  string `json:"createdAt"`
}

type DashboardSummary struct {
	Products         int          `json:"products"`
	Suppliers        int          `json:"suppliers"`
	Orders           int          `json:"orders"`
	LowStockItems    int          `json:"lowStockItems"`
	InventoryValue   float64      `json:"inventoryValue"`
	OrdersPerMonth   []ChartPoint `json:"ordersPerMonth"`
	StockMovements   []ChartPoint `json:"stockMovements"`
	TopSelling       []TopProduct `json:"topSelling"`
	LowStockProducts []Product    `json:"lowStockProducts"`
	RecentAudit      []AuditEvent `json:"recentAudit"`
	LatestInsight    *InsightRun  `json:"latestInsight,omitempty"`
}

type ChartPoint struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type TopProduct struct {
	ProductID int64  `json:"productId"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
}

type BootstrapPayload struct {
	Dashboard      DashboardSummary       `json:"dashboard"`
	Products       []Product              `json:"products"`
	Suppliers      []Supplier             `json:"suppliers"`
	PurchaseOrders []PurchaseOrder        `json:"purchaseOrders"`
	CustomerOrders []CustomerOrder        `json:"customerOrders"`
	Transactions   []InventoryTransaction `json:"transactions"`
	AuditEvents    []AuditEvent           `json:"auditEvents"`
	InsightRuns    []InsightRun           `json:"insightRuns"`
}

type csvResult struct {
	Processed int      `json:"processed"`
	Created   int      `json:"created"`
	Updated   int      `json:"updated"`
	Errors    []string `json:"errors"`
}

func New(cfg Config) (*App, error) {
	if cfg.DBPath == "" {
		return nil, errors.New("db path is required")
	}
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, err
	}

	app := &App{db: db, ai: normalizeAIConfig(cfg.AI)}
	if err := app.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := app.seed(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return app, nil
}

func (a *App) Close() error {
	return a.db.Close()
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/bootstrap", a.handleBootstrap)
	mux.HandleFunc("GET /api/products", a.handleListProducts)
	mux.HandleFunc("POST /api/products", a.handleCreateProduct)
	mux.HandleFunc("PUT /api/products/{id}", a.handleUpdateProduct)
	mux.HandleFunc("GET /api/products/{id}/transactions", a.handleProductTransactions)
	mux.HandleFunc("POST /api/products/{id}/adjustments", a.handleAdjustProduct)

	mux.HandleFunc("GET /api/suppliers", a.handleListSuppliers)
	mux.HandleFunc("POST /api/suppliers", a.handleCreateSupplier)
	mux.HandleFunc("PUT /api/suppliers/{id}", a.handleUpdateSupplier)

	mux.HandleFunc("GET /api/purchase-orders", a.handleListPurchaseOrders)
	mux.HandleFunc("POST /api/purchase-orders", a.handleCreatePurchaseOrder)
	mux.HandleFunc("PUT /api/purchase-orders/{id}", a.handleUpdatePurchaseOrder)

	mux.HandleFunc("GET /api/customer-orders", a.handleListCustomerOrders)
	mux.HandleFunc("POST /api/customer-orders", a.handleCreateCustomerOrder)
	mux.HandleFunc("PUT /api/customer-orders/{id}", a.handleUpdateCustomerOrder)

	mux.HandleFunc("GET /api/dashboard", a.handleDashboard)
	mux.HandleFunc("GET /api/audit", a.handleAudit)
	mux.HandleFunc("GET /api/insights", a.handleListInsightRuns)
	mux.HandleFunc("POST /api/insights/generate", a.handleGenerateInsightRun)
	mux.HandleFunc("POST /api/import/products", a.handleImportProducts)
	mux.HandleFunc("POST /api/import/suppliers", a.handleImportSuppliers)
	mux.HandleFunc("GET /api/export/products.csv", a.handleExportProducts)
	mux.HandleFunc("GET /api/export/inventory.csv", a.handleExportInventory)
	mux.HandleFunc("GET /api/export/orders.csv", a.handleExportOrders)
	mux.HandleFunc("GET /api/export/report.csv", a.handleExportReport)

	fileServer := http.FileServer(http.Dir("web"))
	mux.Handle("/", fileServer)

	return recoverMiddleware(mux)
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				http.Error(w, fmt.Sprintf("internal error: %v", recovered), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (a *App) migrate() error {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		contents, err := os.ReadFile(filepath.Join("migrations", entry.Name()))
		if err != nil {
			return err
		}
		if _, err := a.db.Exec(string(contents)); err != nil {
			return fmt.Errorf("migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func (a *App) seed() error {
	if err := a.ensureDemoSeedState(); err != nil {
		return err
	}

	version, err := a.currentDemoSeedVersion()
	if err != nil {
		return err
	}
	if version == demoSeedVersion {
		return nil
	}

	tx, err := a.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := resetDemoSeedTx(tx); err != nil {
		return err
	}
	if err := seedDemoBusinessTx(tx); err != nil {
		return err
	}
	if err := storeDemoSeedVersionTx(tx, demoSeedVersion); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	if _, err := a.generateInsightRun(90, "system-seed"); err != nil {
		return err
	}
	return nil
}

func insertPurchaseOrderTx(tx *sql.Tx, supplierID int64, status, notes string, items []PurchaseOrderItem, now string) (int64, error) {
	res, err := tx.Exec(`INSERT INTO purchase_orders (supplier_id, status, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, supplierID, status, notes, now, now)
	if err != nil {
		return 0, err
	}
	poID, _ := res.LastInsertId()
	for _, item := range items {
		if _, err := tx.Exec(`INSERT INTO purchase_order_items (purchase_order_id, product_id, quantity, unit_cost) VALUES (?, ?, ?, ?)`, poID, item.ProductID, item.Quantity, item.UnitCost); err != nil {
			return 0, err
		}
	}
	return poID, nil
}

func insertCustomerOrderTx(tx *sql.Tx, customerName, status, notes string, items []CustomerOrderItem, now string) (int64, error) {
	res, err := tx.Exec(`INSERT INTO customer_orders (customer_name, status, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, customerName, status, notes, now, now)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	for _, item := range items {
		if _, err := tx.Exec(`INSERT INTO customer_order_items (customer_order_id, product_id, quantity, unit_price) VALUES (?, ?, ?, ?)`, id, item.ProductID, item.Quantity, item.UnitPrice); err != nil {
			return 0, err
		}
	}
	return id, nil
}

type inventoryChange struct {
	ProductID     int64
	Delta         int
	Kind          string
	ReferenceType string
	ReferenceID   int64
	Reason        string
	Actor         string
}

func applyInventoryChangeTx(tx *sql.Tx, change inventoryChange) error {
	var current int
	if err := tx.QueryRow(`SELECT current_stock FROM products WHERE id = ?`, change.ProductID).Scan(&current); err != nil {
		return err
	}
	next := current + change.Delta
	if next < 0 {
		return fmt.Errorf("insufficient stock for product %d", change.ProductID)
	}
	if _, err := tx.Exec(`UPDATE products SET current_stock = ?, updated_at = ? WHERE id = ?`, next, nowString(), change.ProductID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO inventory_transactions (product_id, transaction_type, quantity, reference_type, reference_id, reason, actor, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		change.ProductID, change.Kind, change.Delta, change.ReferenceType, change.ReferenceID, change.Reason, change.Actor, nowString(),
	); err != nil {
		return err
	}
	return nil
}

func auditTx(tx *sql.Tx, actor, entityType string, entityID int64, action, details string) error {
	_, err := tx.Exec(`INSERT INTO audit_events (actor, entity_type, entity_id, action, details, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		actor, entityType, entityID, action, details, nowString())
	return err
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func parseActor(r *http.Request) string {
	actor := strings.TrimSpace(r.Header.Get("X-Actor"))
	if actor == "" {
		actor = "admin"
	}
	return actor
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeJSON(r *http.Request, out any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(out)
}

func readID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func (a *App) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	payload, err := a.bootstrapPayload()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handleListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := a.listProducts(r.URL.Query().Get("q"), r.URL.Query().Get("status"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (a *App) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var input Product
	if err := decodeJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	product, err := a.saveProduct(0, input, parseActor(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, product)
}

func (a *App) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var input Product
	if err := decodeJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	product, err := a.saveProduct(id, input, parseActor(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (a *App) handleProductTransactions(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rows, err := a.db.Query(`
		SELECT t.id, t.product_id, p.name, p.sku, t.transaction_type, t.quantity, t.reference_type, t.reference_id, t.reason, t.actor, t.created_at
		FROM inventory_transactions t
		JOIN products p ON p.id = t.product_id
		WHERE t.product_id = ?
		ORDER BY t.created_at DESC`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var transactions []InventoryTransaction
	for rows.Next() {
		var tx InventoryTransaction
		if err := rows.Scan(&tx.ID, &tx.ProductID, &tx.ProductName, &tx.ProductSKU, &tx.TransactionType, &tx.Quantity, &tx.ReferenceType, &tx.ReferenceID, &tx.Reason, &tx.Actor, &tx.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		transactions = append(transactions, tx)
	}
	writeJSON(w, http.StatusOK, transactions)
}

func (a *App) handleAdjustProduct(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var input struct {
		Quantity int    `json:"quantity"`
		Type     string `json:"type"`
		Reason   string `json:"reason"`
	}
	if err := decodeJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if input.Type == "" || input.Type == "adjusted" {
		input.Type = "adjusted"
	}
	delta := input.Quantity
	if input.Type == "damaged" && delta > 0 {
		delta = -delta
	}
	if input.Type == "returned" && delta < 0 {
		delta = -delta
	}
	if err := a.adjustInventory(id, delta, input.Type, input.Reason, parseActor(r)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleListSuppliers(w http.ResponseWriter, r *http.Request) {
	suppliers, err := a.listSuppliers(r.URL.Query().Get("q"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, suppliers)
}

func (a *App) handleCreateSupplier(w http.ResponseWriter, r *http.Request) {
	var input Supplier
	if err := decodeJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	supplier, err := a.saveSupplier(0, input, parseActor(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, supplier)
}

func (a *App) handleUpdateSupplier(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var input Supplier
	if err := decodeJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	supplier, err := a.saveSupplier(id, input, parseActor(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, supplier)
}

func (a *App) handleListPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := a.listPurchaseOrders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (a *App) handleCreatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	var input PurchaseOrder
	if err := decodeJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	order, err := a.savePurchaseOrder(0, input, parseActor(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (a *App) handleUpdatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var input PurchaseOrder
	if err := decodeJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	order, err := a.savePurchaseOrder(id, input, parseActor(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (a *App) handleListCustomerOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := a.listCustomerOrders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (a *App) handleCreateCustomerOrder(w http.ResponseWriter, r *http.Request) {
	var input CustomerOrder
	if err := decodeJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	order, err := a.saveCustomerOrder(0, input, parseActor(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (a *App) handleUpdateCustomerOrder(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var input CustomerOrder
	if err := decodeJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	order, err := a.saveCustomerOrder(id, input, parseActor(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (a *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, err := a.dashboard()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (a *App) handleAudit(w http.ResponseWriter, r *http.Request) {
	auditEvents, err := a.listAudit()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, auditEvents)
}

func (a *App) handleImportProducts(w http.ResponseWriter, r *http.Request) {
	result, err := a.importProducts(r, parseActor(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *App) handleImportSuppliers(w http.ResponseWriter, r *http.Request) {
	result, err := a.importSuppliers(r, parseActor(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *App) handleExportProducts(w http.ResponseWriter, r *http.Request) {
	if err := a.exportProducts(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) handleExportInventory(w http.ResponseWriter, r *http.Request) {
	if err := a.exportInventory(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) handleExportOrders(w http.ResponseWriter, r *http.Request) {
	if err := a.exportOrders(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) handleExportReport(w http.ResponseWriter, r *http.Request) {
	if err := a.exportReport(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) bootstrapPayload() (BootstrapPayload, error) {
	dashboard, err := a.dashboard()
	if err != nil {
		return BootstrapPayload{}, err
	}
	products, err := a.listProducts("", "")
	if err != nil {
		return BootstrapPayload{}, err
	}
	suppliers, err := a.listSuppliers("")
	if err != nil {
		return BootstrapPayload{}, err
	}
	purchaseOrders, err := a.listPurchaseOrders()
	if err != nil {
		return BootstrapPayload{}, err
	}
	customerOrders, err := a.listCustomerOrders()
	if err != nil {
		return BootstrapPayload{}, err
	}
	transactions, err := a.listTransactions(100)
	if err != nil {
		return BootstrapPayload{}, err
	}
	auditEvents, err := a.listAudit()
	if err != nil {
		return BootstrapPayload{}, err
	}
	insightRuns, err := a.listInsightRuns(20)
	if err != nil {
		return BootstrapPayload{}, err
	}
	return BootstrapPayload{
		Dashboard:      dashboard,
		Products:       products,
		Suppliers:      suppliers,
		PurchaseOrders: purchaseOrders,
		CustomerOrders: customerOrders,
		Transactions:   transactions,
		AuditEvents:    auditEvents,
		InsightRuns:    insightRuns,
	}, nil
}

func (a *App) listProducts(query, status string) ([]Product, error) {
	var args []any
	sqlQuery := `
		SELECT id, sku, name, description, category, unit_cost, selling_price, current_stock, reorder_level, active, created_at, updated_at
		FROM products
		WHERE 1 = 1`
	if query != "" {
		sqlQuery += ` AND (LOWER(name) LIKE ? OR LOWER(sku) LIKE ?)`
		term := "%" + strings.ToLower(query) + "%"
		args = append(args, term, term)
	}
	switch status {
	case "active":
		sqlQuery += ` AND active = 1`
	case "inactive":
		sqlQuery += ` AND active = 0`
	case "low":
		sqlQuery += ` AND current_stock < reorder_level AND current_stock > 0`
	}
	sqlQuery += ` ORDER BY name`

	rows, err := a.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []Product
	for rows.Next() {
		var product Product
		var active int
		if err := rows.Scan(&product.ID, &product.SKU, &product.Name, &product.Description, &product.Category, &product.UnitCost, &product.SellingPrice, &product.CurrentStock, &product.ReorderLevel, &active, &product.CreatedAt, &product.UpdatedAt); err != nil {
			return nil, err
		}
		product.Active = active == 1
		product.LowStock = product.CurrentStock < product.ReorderLevel
		products = append(products, product)
	}
	return products, nil
}

func (a *App) saveProduct(id int64, input Product, actor string) (Product, error) {
	input.SKU = strings.TrimSpace(input.SKU)
	input.Name = strings.TrimSpace(input.Name)
	if input.SKU == "" || input.Name == "" {
		return Product{}, errors.New("sku and name are required")
	}
	if input.ReorderLevel < 0 {
		return Product{}, errors.New("reorder level cannot be negative")
	}
	if input.CurrentStock < 0 {
		return Product{}, errors.New("current stock cannot be negative")
	}

	tx, err := a.db.BeginTx(context.Background(), nil)
	if err != nil {
		return Product{}, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var duplicateID int64
	err = tx.QueryRow(`SELECT id FROM products WHERE sku = ? AND id <> ?`, input.SKU, id).Scan(&duplicateID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Product{}, err
	}
	if duplicateID != 0 {
		return Product{}, errors.New("sku already exists")
	}

	now := nowString()
	action := "created"
	if id == 0 {
		res, err := tx.Exec(`
			INSERT INTO products (sku, name, description, category, unit_cost, selling_price, current_stock, reorder_level, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			input.SKU, input.Name, input.Description, input.Category, input.UnitCost, input.SellingPrice, input.CurrentStock, input.ReorderLevel, boolInt(input.Active), now, now,
		)
		if err != nil {
			return Product{}, err
		}
		id, _ = res.LastInsertId()
	} else {
		action = "updated"
		if _, err := tx.Exec(`
			UPDATE products
			SET sku = ?, name = ?, description = ?, category = ?, unit_cost = ?, selling_price = ?, current_stock = ?, reorder_level = ?, active = ?, updated_at = ?
			WHERE id = ?`,
			input.SKU, input.Name, input.Description, input.Category, input.UnitCost, input.SellingPrice, input.CurrentStock, input.ReorderLevel, boolInt(input.Active), now, id,
		); err != nil {
			return Product{}, err
		}
	}
	if err := auditTx(tx, actor, "product", id, action, fmt.Sprintf("%s (%s)", input.Name, input.SKU)); err != nil {
		return Product{}, err
	}
	if err := tx.Commit(); err != nil {
		return Product{}, err
	}
	tx = nil
	return a.getProduct(id)
}

func (a *App) getProduct(id int64) (Product, error) {
	var product Product
	var active int
	err := a.db.QueryRow(`
		SELECT id, sku, name, description, category, unit_cost, selling_price, current_stock, reorder_level, active, created_at, updated_at
		FROM products WHERE id = ?`, id,
	).Scan(&product.ID, &product.SKU, &product.Name, &product.Description, &product.Category, &product.UnitCost, &product.SellingPrice, &product.CurrentStock, &product.ReorderLevel, &active, &product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		return Product{}, err
	}
	product.Active = active == 1
	product.LowStock = product.CurrentStock < product.ReorderLevel
	return product, nil
}

func (a *App) adjustInventory(productID int64, delta int, kind, reason, actor string) error {
	if delta == 0 {
		return errors.New("quantity must be non-zero")
	}
	tx, err := a.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	if err := applyInventoryChangeTx(tx, inventoryChange{
		ProductID:     productID,
		Delta:         delta,
		Kind:          kind,
		ReferenceType: "manual",
		ReferenceID:   0,
		Reason:        reason,
		Actor:         actor,
	}); err != nil {
		return err
	}
	if err := auditTx(tx, actor, "product", productID, "inventory_"+kind, reason); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	tx = nil
	return nil
}

func (a *App) listSuppliers(query string) ([]Supplier, error) {
	sqlQuery := `SELECT id, name, contact_name, email, phone, notes, created_at, updated_at FROM suppliers`
	var args []any
	if query != "" {
		sqlQuery += ` WHERE LOWER(name) LIKE ?`
		args = append(args, "%"+strings.ToLower(query)+"%")
	}
	sqlQuery += ` ORDER BY name`
	rows, err := a.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var suppliers []Supplier
	for rows.Next() {
		var supplier Supplier
		if err := rows.Scan(&supplier.ID, &supplier.Name, &supplier.ContactName, &supplier.Email, &supplier.Phone, &supplier.Notes, &supplier.CreatedAt, &supplier.UpdatedAt); err != nil {
			return nil, err
		}
		productIDs, err := a.supplierProductIDs(supplier.ID)
		if err != nil {
			return nil, err
		}
		supplier.ProductIDs = productIDs
		suppliers = append(suppliers, supplier)
	}
	return suppliers, nil
}

func (a *App) supplierProductIDs(supplierID int64) ([]int64, error) {
	rows, err := a.db.Query(`SELECT product_id FROM supplier_products WHERE supplier_id = ? ORDER BY product_id`, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (a *App) saveSupplier(id int64, input Supplier, actor string) (Supplier, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Supplier{}, errors.New("supplier name is required")
	}
	tx, err := a.db.BeginTx(context.Background(), nil)
	if err != nil {
		return Supplier{}, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	now := nowString()
	action := "created"
	if id == 0 {
		res, err := tx.Exec(`INSERT INTO suppliers (name, contact_name, email, phone, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			input.Name, input.ContactName, input.Email, input.Phone, input.Notes, now, now)
		if err != nil {
			return Supplier{}, err
		}
		id, _ = res.LastInsertId()
	} else {
		action = "updated"
		if _, err := tx.Exec(`UPDATE suppliers SET name = ?, contact_name = ?, email = ?, phone = ?, notes = ?, updated_at = ? WHERE id = ?`,
			input.Name, input.ContactName, input.Email, input.Phone, input.Notes, now, id); err != nil {
			return Supplier{}, err
		}
		if _, err := tx.Exec(`DELETE FROM supplier_products WHERE supplier_id = ?`, id); err != nil {
			return Supplier{}, err
		}
	}
	for _, productID := range input.ProductIDs {
		if _, err := tx.Exec(`INSERT INTO supplier_products (supplier_id, product_id) VALUES (?, ?)`, id, productID); err != nil {
			return Supplier{}, err
		}
	}
	if err := auditTx(tx, actor, "supplier", id, action, input.Name); err != nil {
		return Supplier{}, err
	}
	if err := tx.Commit(); err != nil {
		return Supplier{}, err
	}
	tx = nil
	suppliers, err := a.listSuppliers("")
	if err != nil {
		return Supplier{}, err
	}
	for _, supplier := range suppliers {
		if supplier.ID == id {
			return supplier, nil
		}
	}
	return Supplier{}, sql.ErrNoRows
}

func (a *App) listPurchaseOrders() ([]PurchaseOrder, error) {
	rows, err := a.db.Query(`
		SELECT po.id, po.supplier_id, s.name, po.status, po.notes, po.created_at, po.updated_at
		FROM purchase_orders po
		JOIN suppliers s ON s.id = po.supplier_id
		ORDER BY po.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []PurchaseOrder
	for rows.Next() {
		var order PurchaseOrder
		if err := rows.Scan(&order.ID, &order.SupplierID, &order.Supplier, &order.Status, &order.Notes, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, err
		}
		items, err := a.purchaseOrderItems(order.ID)
		if err != nil {
			return nil, err
		}
		order.Items = items
		orders = append(orders, order)
	}
	return orders, nil
}

func (a *App) purchaseOrderItems(id int64) ([]PurchaseOrderItem, error) {
	rows, err := a.db.Query(`SELECT product_id, quantity, unit_cost FROM purchase_order_items WHERE purchase_order_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PurchaseOrderItem
	for rows.Next() {
		var item PurchaseOrderItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.UnitCost); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func validTransition(current, next string, allowed map[string][]string) bool {
	if current == next {
		return true
	}
	return slices.Contains(allowed[current], next)
}

func (a *App) savePurchaseOrder(id int64, input PurchaseOrder, actor string) (PurchaseOrder, error) {
	if input.SupplierID == 0 || len(input.Items) == 0 {
		return PurchaseOrder{}, errors.New("supplier and items are required")
	}
	allowed := map[string][]string{
		"Draft":     {"Ordered", "Cancelled"},
		"Ordered":   {"Received", "Cancelled"},
		"Received":  {},
		"Cancelled": {},
	}

	tx, err := a.db.BeginTx(context.Background(), nil)
	if err != nil {
		return PurchaseOrder{}, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	now := nowString()
	currentStatus := input.Status
	action := "created"
	if currentStatus == "" {
		currentStatus = "Draft"
	}
	if id == 0 {
		res, err := tx.Exec(`INSERT INTO purchase_orders (supplier_id, status, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			input.SupplierID, currentStatus, input.Notes, now, now)
		if err != nil {
			return PurchaseOrder{}, err
		}
		id, _ = res.LastInsertId()
		for _, item := range input.Items {
			if _, err := tx.Exec(`INSERT INTO purchase_order_items (purchase_order_id, product_id, quantity, unit_cost) VALUES (?, ?, ?, ?)`,
				id, item.ProductID, item.Quantity, item.UnitCost); err != nil {
				return PurchaseOrder{}, err
			}
		}
		if currentStatus == "Received" {
			for _, item := range input.Items {
				if err := applyInventoryChangeTx(tx, inventoryChange{
					ProductID: item.ProductID, Delta: item.Quantity, Kind: "received", ReferenceType: "purchase_order", ReferenceID: id, Reason: "purchase order received", Actor: actor,
				}); err != nil {
					return PurchaseOrder{}, err
				}
			}
		}
	} else {
		action = "updated"
		var previousStatus string
		if err := tx.QueryRow(`SELECT status FROM purchase_orders WHERE id = ?`, id).Scan(&previousStatus); err != nil {
			return PurchaseOrder{}, err
		}
		if !validTransition(previousStatus, input.Status, allowed) {
			return PurchaseOrder{}, fmt.Errorf("invalid purchase order transition: %s -> %s", previousStatus, input.Status)
		}
		if _, err := tx.Exec(`UPDATE purchase_orders SET supplier_id = ?, status = ?, notes = ?, updated_at = ? WHERE id = ?`,
			input.SupplierID, input.Status, input.Notes, now, id); err != nil {
			return PurchaseOrder{}, err
		}
		if previousStatus != "Received" {
			if _, err := tx.Exec(`DELETE FROM purchase_order_items WHERE purchase_order_id = ?`, id); err != nil {
				return PurchaseOrder{}, err
			}
			for _, item := range input.Items {
				if _, err := tx.Exec(`INSERT INTO purchase_order_items (purchase_order_id, product_id, quantity, unit_cost) VALUES (?, ?, ?, ?)`, id, item.ProductID, item.Quantity, item.UnitCost); err != nil {
					return PurchaseOrder{}, err
				}
			}
		}
		if previousStatus != "Received" && input.Status == "Received" {
			for _, item := range input.Items {
				if err := applyInventoryChangeTx(tx, inventoryChange{
					ProductID: item.ProductID, Delta: item.Quantity, Kind: "received", ReferenceType: "purchase_order", ReferenceID: id, Reason: "purchase order received", Actor: actor,
				}); err != nil {
					return PurchaseOrder{}, err
				}
			}
		}
	}

	if err := auditTx(tx, actor, "purchase_order", id, action+"_"+currentStatus, fmt.Sprintf("supplier %d", input.SupplierID)); err != nil {
		return PurchaseOrder{}, err
	}
	if err := tx.Commit(); err != nil {
		return PurchaseOrder{}, err
	}
	tx = nil
	return a.getPurchaseOrder(id)
}

func (a *App) getPurchaseOrder(id int64) (PurchaseOrder, error) {
	orders, err := a.listPurchaseOrders()
	if err != nil {
		return PurchaseOrder{}, err
	}
	for _, order := range orders {
		if order.ID == id {
			return order, nil
		}
	}
	return PurchaseOrder{}, sql.ErrNoRows
}

func (a *App) listCustomerOrders() ([]CustomerOrder, error) {
	rows, err := a.db.Query(`SELECT id, customer_name, status, notes, created_at, updated_at FROM customer_orders ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []CustomerOrder
	for rows.Next() {
		var order CustomerOrder
		if err := rows.Scan(&order.ID, &order.CustomerName, &order.Status, &order.Notes, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, err
		}
		items, err := a.customerOrderItems(order.ID)
		if err != nil {
			return nil, err
		}
		order.Items = items
		orders = append(orders, order)
	}
	return orders, nil
}

func (a *App) customerOrderItems(orderID int64) ([]CustomerOrderItem, error) {
	rows, err := a.db.Query(`
		SELECT coi.product_id, coi.quantity, coi.unit_price, p.sku, p.name
		FROM customer_order_items coi
		JOIN products p ON p.id = coi.product_id
		WHERE coi.customer_order_id = ?`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CustomerOrderItem
	for rows.Next() {
		var item CustomerOrderItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.UnitPrice, &item.ProductSKU, &item.Product); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (a *App) saveCustomerOrder(id int64, input CustomerOrder, actor string) (CustomerOrder, error) {
	if strings.TrimSpace(input.CustomerName) == "" || len(input.Items) == 0 {
		return CustomerOrder{}, errors.New("customer name and items are required")
	}
	allowed := map[string][]string{
		"Pending":    {"Processing", "Cancelled"},
		"Processing": {"Shipped", "Cancelled"},
		"Shipped":    {"Completed"},
		"Completed":  {},
		"Cancelled":  {},
	}
	tx, err := a.db.BeginTx(context.Background(), nil)
	if err != nil {
		return CustomerOrder{}, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	now := nowString()
	status := input.Status
	if status == "" {
		status = "Pending"
	}
	action := "created"
	if id == 0 {
		res, err := tx.Exec(`INSERT INTO customer_orders (customer_name, status, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			input.CustomerName, status, input.Notes, now, now)
		if err != nil {
			return CustomerOrder{}, err
		}
		id, _ = res.LastInsertId()
		for _, item := range input.Items {
			if _, err := tx.Exec(`INSERT INTO customer_order_items (customer_order_id, product_id, quantity, unit_price) VALUES (?, ?, ?, ?)`, id, item.ProductID, item.Quantity, item.UnitPrice); err != nil {
				return CustomerOrder{}, err
			}
		}
		if status == "Shipped" || status == "Completed" {
			for _, item := range input.Items {
				if err := applyInventoryChangeTx(tx, inventoryChange{
					ProductID: item.ProductID, Delta: -item.Quantity, Kind: "sold", ReferenceType: "customer_order", ReferenceID: id, Reason: "customer order shipped", Actor: actor,
				}); err != nil {
					return CustomerOrder{}, err
				}
			}
		}
	} else {
		action = "updated"
		var previousStatus string
		if err := tx.QueryRow(`SELECT status FROM customer_orders WHERE id = ?`, id).Scan(&previousStatus); err != nil {
			return CustomerOrder{}, err
		}
		if !validTransition(previousStatus, input.Status, allowed) {
			return CustomerOrder{}, fmt.Errorf("invalid customer order transition: %s -> %s", previousStatus, input.Status)
		}
		if _, err := tx.Exec(`UPDATE customer_orders SET customer_name = ?, status = ?, notes = ?, updated_at = ? WHERE id = ?`,
			input.CustomerName, input.Status, input.Notes, now, id); err != nil {
			return CustomerOrder{}, err
		}
		if previousStatus == "Pending" || previousStatus == "Processing" {
			if _, err := tx.Exec(`DELETE FROM customer_order_items WHERE customer_order_id = ?`, id); err != nil {
				return CustomerOrder{}, err
			}
			for _, item := range input.Items {
				if _, err := tx.Exec(`INSERT INTO customer_order_items (customer_order_id, product_id, quantity, unit_price) VALUES (?, ?, ?, ?)`, id, item.ProductID, item.Quantity, item.UnitPrice); err != nil {
					return CustomerOrder{}, err
				}
			}
		}
		if previousStatus != "Shipped" && previousStatus != "Completed" && input.Status == "Shipped" {
			for _, item := range input.Items {
				if err := applyInventoryChangeTx(tx, inventoryChange{
					ProductID: item.ProductID, Delta: -item.Quantity, Kind: "sold", ReferenceType: "customer_order", ReferenceID: id, Reason: "customer order shipped", Actor: actor,
				}); err != nil {
					return CustomerOrder{}, err
				}
			}
		}
	}
	if err := auditTx(tx, actor, "customer_order", id, action+"_"+status, input.CustomerName); err != nil {
		return CustomerOrder{}, err
	}
	if err := tx.Commit(); err != nil {
		return CustomerOrder{}, err
	}
	tx = nil
	return a.getCustomerOrder(id)
}

func (a *App) getCustomerOrder(id int64) (CustomerOrder, error) {
	orders, err := a.listCustomerOrders()
	if err != nil {
		return CustomerOrder{}, err
	}
	for _, order := range orders {
		if order.ID == id {
			return order, nil
		}
	}
	return CustomerOrder{}, sql.ErrNoRows
}

func (a *App) dashboard() (DashboardSummary, error) {
	var products, suppliers, orders, lowStock int
	var inventoryValue float64
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&products); err != nil {
		return DashboardSummary{}, err
	}
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM suppliers`).Scan(&suppliers); err != nil {
		return DashboardSummary{}, err
	}
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM customer_orders`).Scan(&orders); err != nil {
		return DashboardSummary{}, err
	}
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM products WHERE current_stock < reorder_level AND current_stock > 0`).Scan(&lowStock); err != nil {
		return DashboardSummary{}, err
	}
	if err := a.db.QueryRow(`SELECT COALESCE(SUM(current_stock * unit_cost), 0) FROM products`).Scan(&inventoryValue); err != nil {
		return DashboardSummary{}, err
	}

	lowStockProducts, err := a.listProducts("", "low")
	if err != nil {
		return DashboardSummary{}, err
	}
	auditEvents, err := a.listAudit()
	if err != nil {
		return DashboardSummary{}, err
	}
	topSelling, err := a.topSellingProducts()
	if err != nil {
		return DashboardSummary{}, err
	}
	ordersPerMonth, err := a.ordersPerMonth()
	if err != nil {
		return DashboardSummary{}, err
	}
	movements, err := a.stockMovements()
	if err != nil {
		return DashboardSummary{}, err
	}
	if len(auditEvents) > 5 {
		auditEvents = auditEvents[:5]
	}
	latestInsight, err := a.latestInsightRun()
	if err != nil {
		return DashboardSummary{}, err
	}
	return DashboardSummary{
		Products:         products,
		Suppliers:        suppliers,
		Orders:           orders,
		LowStockItems:    lowStock,
		InventoryValue:   inventoryValue,
		OrdersPerMonth:   ordersPerMonth,
		StockMovements:   movements,
		TopSelling:       topSelling,
		LowStockProducts: lowStockProducts,
		RecentAudit:      auditEvents,
		LatestInsight:    latestInsight,
	}, nil
}

func (a *App) ordersPerMonth() ([]ChartPoint, error) {
	rows, err := a.db.Query(`
		SELECT substr(created_at, 1, 7) AS month, COUNT(*)
		FROM customer_orders
		GROUP BY month
		ORDER BY month DESC
		LIMIT 6`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var points []ChartPoint
	for rows.Next() {
		var point ChartPoint
		if err := rows.Scan(&point.Label, &point.Value); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}

func (a *App) stockMovements() ([]ChartPoint, error) {
	rows, err := a.db.Query(`
		SELECT transaction_type, COUNT(*)
		FROM inventory_transactions
		GROUP BY transaction_type
		ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var points []ChartPoint
	for rows.Next() {
		var point ChartPoint
		if err := rows.Scan(&point.Label, &point.Value); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}

func (a *App) topSellingProducts() ([]TopProduct, error) {
	rows, err := a.db.Query(`
		SELECT p.id, p.sku, p.name, COALESCE(SUM(ABS(t.quantity)), 0) AS sold_qty
		FROM products p
		LEFT JOIN inventory_transactions t ON t.product_id = p.id AND t.transaction_type = 'sold'
		GROUP BY p.id, p.sku, p.name
		ORDER BY sold_qty DESC, p.name
		LIMIT 5`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []TopProduct
	for rows.Next() {
		var product TopProduct
		if err := rows.Scan(&product.ProductID, &product.SKU, &product.Name, &product.Quantity); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

func (a *App) listTransactions(limit int) ([]InventoryTransaction, error) {
	rows, err := a.db.Query(`
		SELECT t.id, t.product_id, p.name, p.sku, t.transaction_type, t.quantity, t.reference_type, t.reference_id, t.reason, t.actor, t.created_at
		FROM inventory_transactions t
		JOIN products p ON p.id = t.product_id
		ORDER BY t.created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var transactions []InventoryTransaction
	for rows.Next() {
		var tx InventoryTransaction
		if err := rows.Scan(&tx.ID, &tx.ProductID, &tx.ProductName, &tx.ProductSKU, &tx.TransactionType, &tx.Quantity, &tx.ReferenceType, &tx.ReferenceID, &tx.Reason, &tx.Actor, &tx.CreatedAt); err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

func (a *App) listAudit() ([]AuditEvent, error) {
	rows, err := a.db.Query(`SELECT id, actor, entity_type, entity_id, action, details, created_at FROM audit_events ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []AuditEvent
	for rows.Next() {
		var event AuditEvent
		if err := rows.Scan(&event.ID, &event.Actor, &event.EntityType, &event.EntityID, &event.Action, &event.Details, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func (a *App) importProducts(r *http.Request, actor string) (csvResult, error) {
	return a.importCSV(r, actor, "products")
}

func (a *App) importSuppliers(r *http.Request, actor string) (csvResult, error) {
	return a.importCSV(r, actor, "suppliers")
}

func (a *App) importCSV(r *http.Request, actor, mode string) (csvResult, error) {
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		return csvResult{}, err
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		return csvResult{}, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return csvResult{}, err
	}
	if len(rows) < 2 {
		return csvResult{}, errors.New("csv must include a header and at least one row")
	}
	header := rows[0]
	result := csvResult{}
	for idx, row := range rows[1:] {
		if len(row) != len(header) {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: column count mismatch", idx+2))
			continue
		}
		record := map[string]string{}
		for i, key := range header {
			record[strings.TrimSpace(strings.ToLower(key))] = strings.TrimSpace(row[i])
		}
		switch mode {
		case "products":
			product, err := productFromCSV(record)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", idx+2, err))
				continue
			}
			existingID := a.lookupProductBySKU(product.SKU)
			if _, err := a.saveProduct(existingID, product, actor); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", idx+2, err))
				continue
			}
			if existingID == 0 {
				result.Created++
			} else {
				result.Updated++
			}
		case "suppliers":
			supplier, err := supplierFromCSV(record)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", idx+2, err))
				continue
			}
			existingID := a.lookupSupplierByName(supplier.Name)
			if _, err := a.saveSupplier(existingID, supplier, actor); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", idx+2, err))
				continue
			}
			if existingID == 0 {
				result.Created++
			} else {
				result.Updated++
			}
		}
		result.Processed++
	}
	return result, nil
}

func productFromCSV(record map[string]string) (Product, error) {
	required := []string{"sku", "name", "category", "unit_cost", "selling_price", "current_stock", "reorder_level", "active"}
	for _, key := range required {
		if record[key] == "" {
			return Product{}, fmt.Errorf("missing %s", key)
		}
	}
	unitCost, err := strconv.ParseFloat(record["unit_cost"], 64)
	if err != nil {
		return Product{}, fmt.Errorf("invalid unit_cost")
	}
	sellingPrice, err := strconv.ParseFloat(record["selling_price"], 64)
	if err != nil {
		return Product{}, fmt.Errorf("invalid selling_price")
	}
	currentStock, err := strconv.Atoi(record["current_stock"])
	if err != nil {
		return Product{}, fmt.Errorf("invalid current_stock")
	}
	reorderLevel, err := strconv.Atoi(record["reorder_level"])
	if err != nil {
		return Product{}, fmt.Errorf("invalid reorder_level")
	}
	active := strings.EqualFold(record["active"], "true") || record["active"] == "1" || strings.EqualFold(record["active"], "yes")
	return Product{
		SKU:          record["sku"],
		Name:         record["name"],
		Description:  record["description"],
		Category:     record["category"],
		UnitCost:     unitCost,
		SellingPrice: sellingPrice,
		CurrentStock: currentStock,
		ReorderLevel: reorderLevel,
		Active:       active,
	}, nil
}

func supplierFromCSV(record map[string]string) (Supplier, error) {
	if record["name"] == "" {
		return Supplier{}, errors.New("missing name")
	}
	return Supplier{
		Name:        record["name"],
		ContactName: record["contact_name"],
		Email:       record["email"],
		Phone:       record["phone"],
		Notes:       record["notes"],
	}, nil
}

func (a *App) lookupProductBySKU(sku string) int64 {
	var id int64
	_ = a.db.QueryRow(`SELECT id FROM products WHERE sku = ?`, sku).Scan(&id)
	return id
}

func (a *App) lookupSupplierByName(name string) int64 {
	var id int64
	_ = a.db.QueryRow(`SELECT id FROM suppliers WHERE name = ?`, name).Scan(&id)
	return id
}

func (a *App) exportProducts(w http.ResponseWriter) error {
	products, err := a.listProducts("", "")
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="products.csv"`)
	writer := csv.NewWriter(w)
	defer writer.Flush()
	_ = writer.Write([]string{"sku", "name", "category", "unit_cost", "selling_price", "current_stock", "reorder_level", "active"})
	for _, product := range products {
		_ = writer.Write([]string{
			product.SKU, product.Name, product.Category,
			fmt.Sprintf("%.2f", product.UnitCost),
			fmt.Sprintf("%.2f", product.SellingPrice),
			strconv.Itoa(product.CurrentStock),
			strconv.Itoa(product.ReorderLevel),
			strconv.FormatBool(product.Active),
		})
	}
	return writer.Error()
}

func (a *App) exportInventory(w http.ResponseWriter) error {
	transactions, err := a.listTransactions(1000)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="inventory.csv"`)
	writer := csv.NewWriter(w)
	defer writer.Flush()
	_ = writer.Write([]string{"date", "sku", "product", "type", "quantity", "reference_type", "reference_id", "actor", "reason"})
	for _, tx := range transactions {
		_ = writer.Write([]string{
			tx.CreatedAt, tx.ProductSKU, tx.ProductName, tx.TransactionType, strconv.Itoa(tx.Quantity), tx.ReferenceType, strconv.FormatInt(tx.ReferenceID, 10), tx.Actor, tx.Reason,
		})
	}
	return writer.Error()
}

func (a *App) exportOrders(w http.ResponseWriter) error {
	orders, err := a.listCustomerOrders()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="orders.csv"`)
	writer := csv.NewWriter(w)
	defer writer.Flush()
	_ = writer.Write([]string{"order_id", "customer", "status", "product_id", "quantity", "unit_price", "created_at"})
	for _, order := range orders {
		for _, item := range order.Items {
			_ = writer.Write([]string{
				strconv.FormatInt(order.ID, 10), order.CustomerName, order.Status, strconv.FormatInt(item.ProductID, 10), strconv.Itoa(item.Quantity), fmt.Sprintf("%.2f", item.UnitPrice), order.CreatedAt,
			})
		}
	}
	return writer.Error()
}

func (a *App) exportReport(w http.ResponseWriter) error {
	dashboard, err := a.dashboard()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="report.csv"`)
	writer := csv.NewWriter(w)
	defer writer.Flush()
	_ = writer.Write([]string{"metric", "value"})
	_ = writer.Write([]string{"products", strconv.Itoa(dashboard.Products)})
	_ = writer.Write([]string{"suppliers", strconv.Itoa(dashboard.Suppliers)})
	_ = writer.Write([]string{"orders", strconv.Itoa(dashboard.Orders)})
	_ = writer.Write([]string{"low_stock_items", strconv.Itoa(dashboard.LowStockItems)})
	_ = writer.Write([]string{"inventory_value", fmt.Sprintf("%.2f", dashboard.InventoryValue)})
	return writer.Error()
}
