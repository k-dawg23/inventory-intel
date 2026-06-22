package app

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	oldWD, _ := os.Getwd()
	root := filepath.Join(oldWD, "..", "..")
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	app, err := New(Config{DBPath: filepath.Join(dir, "test.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })
	return app
}

func productIDBySKU(t *testing.T, db *sql.DB, sku string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`SELECT id FROM products WHERE sku = ?`, sku).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func countRows(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func loginAsDemo(t *testing.T, app *App) *http.Cookie {
	t.Helper()
	body := strings.NewReader(`{"identifier":"kenneth","password":"DemoAdmin123!"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })
	cookies := res.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}
	return cookies[0]
}

func TestDemoSeedPopulatesRealisticDataset(t *testing.T) {
	app := newTestApp(t)
	if got, want := countRows(t, app.db, `SELECT COUNT(*) FROM products`), 104; got != want {
		t.Fatalf("products: got %d want %d", got, want)
	}
	if got, want := countRows(t, app.db, `SELECT COUNT(*) FROM suppliers`), 10; got != want {
		t.Fatalf("suppliers: got %d want %d", got, want)
	}
	if got, want := countRows(t, app.db, `SELECT COUNT(*) FROM customer_orders`), 1500; got != want {
		t.Fatalf("customer orders: got %d want %d", got, want)
	}
	if got, want := countRows(t, app.db, `SELECT COUNT(*) FROM purchase_orders`), 300; got != want {
		t.Fatalf("purchase orders: got %d want %d", got, want)
	}
	if got, want := countRows(t, app.db, `SELECT COUNT(*) FROM inventory_transactions`), 7500; got != want {
		t.Fatalf("inventory transactions: got %d want %d", got, want)
	}
	if got, want := countRows(t, app.db, `SELECT COUNT(*) FROM audit_events`), 751; got != want {
		t.Fatalf("audit events: got %d want %d", got, want)
	}
	if got, want := countRows(t, app.db, `SELECT COUNT(*) FROM products WHERE current_stock < reorder_level AND current_stock > 0`), 12; got != want {
		t.Fatalf("low stock products: got %d want %d", got, want)
	}
	if got, want := countRows(t, app.db, `SELECT COUNT(*) FROM products WHERE current_stock = 0`), 4; got != want {
		t.Fatalf("out of stock products: got %d want %d", got, want)
	}
}

func TestAdjustInventoryPreventsNegativeStock(t *testing.T) {
	app := newTestApp(t)
	productID := productIDBySKU(t, app.db, "SKU-1002")
	if err := app.adjustInventory(productID, -100, "damaged", "failed count", "qa"); err == nil {
		t.Fatal("expected insufficient stock error")
	}
}

func TestCustomerOrderTransitionDeductsStockOnce(t *testing.T) {
	app := newTestApp(t)
	productID := productIDBySKU(t, app.db, "SKU-1003")
	before, err := app.getProduct(productID)
	if err != nil {
		t.Fatal(err)
	}
	order, err := app.saveCustomerOrder(0, CustomerOrder{
		CustomerName: "Test Shop",
		Status:       "Pending",
		Items:        []CustomerOrderItem{{ProductID: productID, Quantity: 2, UnitPrice: 79}},
	}, "qa")
	if err != nil {
		t.Fatal(err)
	}
	order.Status = "Processing"
	if _, err := app.saveCustomerOrder(order.ID, order, "qa"); err != nil {
		t.Fatal(err)
	}
	order.Status = "Shipped"
	if _, err := app.saveCustomerOrder(order.ID, order, "qa"); err != nil {
		t.Fatal(err)
	}
	after, err := app.getProduct(productID)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := after.CurrentStock, before.CurrentStock-2; got != want {
		t.Fatalf("stock mismatch: got %d want %d", got, want)
	}
}

func TestImportProductsReportsValidationErrors(t *testing.T) {
	app := newTestApp(t)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "products.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, "sku,name,category,unit_cost,selling_price,current_stock,reorder_level,active\nBROKEN,,Parts,10,20,5,2,true\n"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/import/products", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	result, err := app.importProducts(req, "qa")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected validation errors")
	}
}

func TestAuditEventCreatedForProductUpdate(t *testing.T) {
	app := newTestApp(t)
	productID := productIDBySKU(t, app.db, "SKU-1001")
	product, err := app.getProduct(productID)
	if err != nil {
		t.Fatal(err)
	}
	product.SellingPrice = 99
	if _, err := app.saveProduct(product.ID, product, "qa"); err != nil {
		t.Fatal(err)
	}
	events, err := app.listAudit()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range events {
		if event.EntityType == "product" && event.EntityID == productID && event.Actor == "qa" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected product audit event")
	}
}

func TestBuildInsightInputIncludesSignals(t *testing.T) {
	app := newTestApp(t)
	input, err := app.buildInsightInput(90)
	if err != nil {
		t.Fatal(err)
	}
	if input.Summary.ProductCount == 0 {
		t.Fatal("expected product count in insight summary")
	}
	if len(input.ProductSignals) == 0 {
		t.Fatal("expected product signals")
	}
}

func TestGenerateInsightRunSimulationPersistsRun(t *testing.T) {
	app := newTestApp(t)
	app.ai.Mode = "simulation"
	run, err := app.generateInsightRun(90, "qa")
	if err != nil {
		t.Fatal(err)
	}
	if run.Mode != "simulation" {
		t.Fatalf("expected simulation mode, got %s", run.Mode)
	}
	if len(run.Recommendations) == 0 {
		t.Fatal("expected recommendations")
	}
	runs, err := app.listInsightRuns(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) == 0 {
		t.Fatal("expected persisted insight run")
	}
}

func TestGenerateInsightRunRealModeRequiresAPIKey(t *testing.T) {
	app := newTestApp(t)
	app.ai.Mode = "real"
	app.ai.APIKey = ""
	run, err := app.generateInsightRun(90, "qa")
	if err == nil {
		t.Fatal("expected missing api key error")
	}
	if run.Status != "failed" {
		t.Fatalf("expected failed run status, got %s", run.Status)
	}
	if run.ErrorMessage == "" {
		t.Fatal("expected stored error message")
	}
}

func TestProtectedAPIRequiresAuthentication(t *testing.T) {
	app := newTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/bootstrap", nil)
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)
	if got, want := rec.Code, http.StatusUnauthorized; got != want {
		t.Fatalf("status: got %d want %d", got, want)
	}
}

func TestDemoAdminLoginCreatesSessionAndUpdatesLastLogin(t *testing.T) {
	app := newTestApp(t)
	body := strings.NewReader(`{"identifier":"kenneth@inventoryintel.demo","password":"DemoAdmin123!"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)
	if got, want := rec.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d want %d body=%s", got, want, rec.Body.String())
	}

	var payload authSessionPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Authenticated || payload.User == nil {
		t.Fatal("expected authenticated session payload")
	}
	if got, want := payload.User.Name, demoAdminName; got != want {
		t.Fatalf("user name: got %q want %q", got, want)
	}
	if payload.User.LastLogin == "" {
		t.Fatal("expected last login to be populated")
	}
	if len(rec.Result().Cookies()) == 0 {
		t.Fatal("expected session cookie to be set")
	}
}

func TestAuthenticatedRequestsUseSessionUserForAuditAttribution(t *testing.T) {
	app := newTestApp(t)
	cookie := loginAsDemo(t, app)
	productID := productIDBySKU(t, app.db, "SKU-1001")

	body := strings.NewReader(`{"sku":"SKU-1001","name":"Noise Cancelling Headphones","description":"Updated","category":"Audio","unitCost":40,"sellingPrice":120,"currentStock":18,"reorderLevel":6,"active":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/products/"+strconv.FormatInt(productID, 10), body)
	req.SetPathValue("id", strconv.FormatInt(productID, 10))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)
	if got, want := rec.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d want %d body=%s", got, want, rec.Body.String())
	}

	events, err := app.listAudit()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range events {
		if event.EntityType == "product" && event.EntityID == productID && event.Actor == demoAdminName {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected audit event attributed to demo admin")
	}
}
