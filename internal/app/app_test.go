package app

import (
	"bytes"
	"database/sql"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
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
