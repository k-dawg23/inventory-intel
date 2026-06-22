package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type AIConfig struct {
	Mode       string
	Model      string
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

type InsightRun struct {
	ID              int64                   `json:"id"`
	Actor           string                  `json:"actor"`
	Mode            string                  `json:"mode"`
	Status          string                  `json:"status"`
	Model           string                  `json:"model"`
	WindowDays      int                     `json:"windowDays"`
	InputSummary    string                  `json:"inputSummary"`
	InputPayload    string                  `json:"inputPayload,omitempty"`
	Recommendations []InsightRecommendation `json:"recommendations"`
	RawOutput       string                  `json:"rawOutput,omitempty"`
	ErrorMessage    string                  `json:"errorMessage,omitempty"`
	CreatedAt       string                  `json:"createdAt"`
}

type InsightRecommendation struct {
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Evidence    string `json:"evidence"`
	ProductID   int64  `json:"productId,omitempty"`
	ProductSKU  string `json:"productSku,omitempty"`
	ProductName string `json:"productName,omitempty"`
}

type InsightInput struct {
	WindowDays     int                  `json:"windowDays"`
	GeneratedAt    string               `json:"generatedAt"`
	Summary        InsightInputSummary  `json:"summary"`
	ProductSignals []InsightProductData `json:"productSignals"`
}

type InsightInputSummary struct {
	ProductCount       int     `json:"productCount"`
	LowStockItems      int     `json:"lowStockItems"`
	InventoryValue     float64 `json:"inventoryValue"`
	OrderCount         int     `json:"orderCount"`
	SoldUnits          int     `json:"soldUnits"`
	ReceivedUnits      int     `json:"receivedUnits"`
	SlowMovingProducts int     `json:"slowMovingProducts"`
	OverstockProducts  int     `json:"overstockProducts"`
}

type InsightProductData struct {
	ProductID         int64   `json:"productId"`
	SKU               string  `json:"sku"`
	Name              string  `json:"name"`
	Category          string  `json:"category"`
	CurrentStock      int     `json:"currentStock"`
	ReorderLevel      int     `json:"reorderLevel"`
	UnitCost          float64 `json:"unitCost"`
	InventoryValue    float64 `json:"inventoryValue"`
	SoldUnits         int     `json:"soldUnits"`
	ReceivedUnits     int     `json:"receivedUnits"`
	AdjustmentUnits   int     `json:"adjustmentUnits"`
	LastSoldAt        string  `json:"lastSoldAt,omitempty"`
	DaysSinceLastSale int     `json:"daysSinceLastSale"`
	SalesVelocity     float64 `json:"salesVelocity"`
	LowStock          bool    `json:"lowStock"`
	OverstockRisk     bool    `json:"overstockRisk"`
	SlowMoving        bool    `json:"slowMoving"`
}

type openAIResponseEnvelope struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func defaultAIConfig() AIConfig {
	return AIConfig{
		Mode:       "simulation",
		Model:      "gpt-5-nano",
		BaseURL:    "https://api.openai.com/v1",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func normalizeAIConfig(cfg AIConfig) AIConfig {
	defaults := defaultAIConfig()
	if strings.TrimSpace(cfg.Mode) == "" {
		cfg.Mode = defaults.Mode
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaults.Model
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = defaults.BaseURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = defaults.HTTPClient
	}
	return cfg
}

func (a *App) handleListInsightRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := a.listInsightRuns(20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

func (a *App) handleGenerateInsightRun(w http.ResponseWriter, r *http.Request) {
	var input struct {
		WindowDays int `json:"windowDays"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := decodeJSON(r, &input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	run, err := a.generateInsightRun(input.WindowDays, parseActor(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (a *App) listInsightRuns(limit int) ([]InsightRun, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := a.db.Query(`
		SELECT id, actor, mode, status, model, window_days, input_summary, input_payload, recommendations_json, raw_output, error_message, created_at
		FROM ai_insight_runs
		ORDER BY created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []InsightRun
	for rows.Next() {
		run, err := scanInsightRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (a *App) latestInsightRun() (*InsightRun, error) {
	runs, err := a.listInsightRuns(1)
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, nil
	}
	return &runs[0], nil
}

func scanInsightRun(scanner interface{ Scan(dest ...any) error }) (InsightRun, error) {
	var run InsightRun
	var recommendationsJSON string
	if err := scanner.Scan(
		&run.ID,
		&run.Actor,
		&run.Mode,
		&run.Status,
		&run.Model,
		&run.WindowDays,
		&run.InputSummary,
		&run.InputPayload,
		&recommendationsJSON,
		&run.RawOutput,
		&run.ErrorMessage,
		&run.CreatedAt,
	); err != nil {
		return InsightRun{}, err
	}
	if recommendationsJSON != "" {
		if err := json.Unmarshal([]byte(recommendationsJSON), &run.Recommendations); err != nil {
			return InsightRun{}, err
		}
	}
	return run, nil
}

func (a *App) generateInsightRun(windowDays int, actor string) (InsightRun, error) {
	if windowDays <= 0 {
		windowDays = 90
	}
	input, err := a.buildInsightInput(windowDays)
	if err != nil {
		return InsightRun{}, err
	}

	mode := strings.ToLower(strings.TrimSpace(a.ai.Mode))
	if mode == "" {
		mode = "simulation"
	}
	run := InsightRun{
		Actor:        actor,
		Mode:         mode,
		Status:       "completed",
		Model:        a.ai.Model,
		WindowDays:   windowDays,
		InputSummary: insightInputSummaryText(input),
		CreatedAt:    nowString(),
	}

	inputJSON, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return InsightRun{}, err
	}
	run.InputPayload = string(inputJSON)

	var recommendations []InsightRecommendation
	var rawOutput string
	switch mode {
	case "real":
		recommendations, rawOutput, err = a.generateRealInsights(input)
	default:
		run.Mode = "simulation"
		recommendations, rawOutput, err = a.generateSimulationInsights(input)
	}

	if err != nil {
		run.Status = "failed"
		run.ErrorMessage = err.Error()
		run.RawOutput = rawOutput
		persisted, persistErr := a.persistInsightRun(run)
		if persistErr != nil {
			return InsightRun{}, persistErr
		}
		return persisted, err
	}

	run.Recommendations = recommendations
	run.RawOutput = rawOutput
	persisted, err := a.persistInsightRun(run)
	if err != nil {
		return InsightRun{}, err
	}
	return persisted, nil
}

func (a *App) persistInsightRun(run InsightRun) (InsightRun, error) {
	recommendationsJSON, err := json.Marshal(run.Recommendations)
	if err != nil {
		return InsightRun{}, err
	}
	tx, err := a.db.BeginTx(context.Background(), nil)
	if err != nil {
		return InsightRun{}, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	res, err := tx.Exec(`
		INSERT INTO ai_insight_runs (actor, mode, status, model, window_days, input_summary, input_payload, recommendations_json, raw_output, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.Actor, run.Mode, run.Status, run.Model, run.WindowDays, run.InputSummary, run.InputPayload, string(recommendationsJSON), run.RawOutput, run.ErrorMessage, run.CreatedAt,
	)
	if err != nil {
		return InsightRun{}, err
	}
	id, _ := res.LastInsertId()
	if err := auditTx(tx, run.Actor, "ai_insight", id, "generated_"+run.Mode+"_"+run.Status, run.InputSummary); err != nil {
		return InsightRun{}, err
	}
	if err := tx.Commit(); err != nil {
		return InsightRun{}, err
	}
	tx = nil

	row := a.db.QueryRow(`
		SELECT id, actor, mode, status, model, window_days, input_summary, input_payload, recommendations_json, raw_output, error_message, created_at
		FROM ai_insight_runs
		WHERE id = ?`, id)
	return scanInsightRun(row)
}

func (a *App) buildInsightInput(windowDays int) (InsightInput, error) {
	products, err := a.listProducts("", "")
	if err != nil {
		return InsightInput{}, err
	}
	if len(products) == 0 {
		return InsightInput{}, errors.New("no product data available for insights")
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -windowDays)
	transactions, err := a.listTransactions(5000)
	if err != nil {
		return InsightInput{}, err
	}
	customerOrders, err := a.listCustomerOrders()
	if err != nil {
		return InsightInput{}, err
	}

	signals := map[int64]*InsightProductData{}
	for _, product := range products {
		signals[product.ID] = &InsightProductData{
			ProductID:      product.ID,
			SKU:            product.SKU,
			Name:           product.Name,
			Category:       product.Category,
			CurrentStock:   product.CurrentStock,
			ReorderLevel:   product.ReorderLevel,
			UnitCost:       product.UnitCost,
			InventoryValue: float64(product.CurrentStock) * product.UnitCost,
			LowStock:       product.LowStock,
		}
	}

	orderCount := 0
	for _, order := range customerOrders {
		createdAt, parseErr := time.Parse(time.RFC3339, order.CreatedAt)
		if parseErr == nil && createdAt.Before(cutoff) {
			continue
		}
		orderCount++
	}

	totalSold := 0
	totalReceived := 0
	for _, tx := range transactions {
		signal := signals[tx.ProductID]
		if signal == nil {
			continue
		}
		createdAt, parseErr := time.Parse(time.RFC3339, tx.CreatedAt)
		if parseErr == nil && createdAt.Before(cutoff) {
			continue
		}
		switch tx.TransactionType {
		case "sold":
			units := absInt(tx.Quantity)
			signal.SoldUnits += units
			totalSold += units
			if tx.CreatedAt > signal.LastSoldAt {
				signal.LastSoldAt = tx.CreatedAt
			}
		case "received":
			units := absInt(tx.Quantity)
			signal.ReceivedUnits += units
			totalReceived += units
		default:
			signal.AdjustmentUnits += tx.Quantity
		}
	}

	now := time.Now().UTC()
	var productSignals []InsightProductData
	slowMoving := 0
	overstock := 0
	for _, signal := range signals {
		if signal.LastSoldAt != "" {
			if soldAt, err := time.Parse(time.RFC3339, signal.LastSoldAt); err == nil {
				signal.DaysSinceLastSale = int(now.Sub(soldAt).Hours() / 24)
			}
		} else {
			signal.DaysSinceLastSale = windowDays + 1
		}
		signal.SalesVelocity = float64(signal.SoldUnits) / float64(windowDays)
		signal.SlowMoving = signal.SoldUnits == 0 || signal.DaysSinceLastSale >= 45
		threshold := maxInt(signal.ReorderLevel*3, signal.SoldUnits*2)
		if threshold == 0 {
			threshold = maxInt(signal.ReorderLevel*2, 10)
		}
		signal.OverstockRisk = signal.CurrentStock > threshold && !signal.LowStock
		if signal.SlowMoving {
			slowMoving++
		}
		if signal.OverstockRisk {
			overstock++
		}
		productSignals = append(productSignals, *signal)
	}
	sort.Slice(productSignals, func(i, j int) bool {
		if productSignals[i].SoldUnits == productSignals[j].SoldUnits {
			return productSignals[i].CurrentStock > productSignals[j].CurrentStock
		}
		return productSignals[i].SoldUnits < productSignals[j].SoldUnits
	})

	dashboard, err := a.dashboard()
	if err != nil {
		return InsightInput{}, err
	}

	return InsightInput{
		WindowDays:  windowDays,
		GeneratedAt: nowString(),
		Summary: InsightInputSummary{
			ProductCount:       len(products),
			LowStockItems:      dashboard.LowStockItems,
			InventoryValue:     dashboard.InventoryValue,
			OrderCount:         orderCount,
			SoldUnits:          totalSold,
			ReceivedUnits:      totalReceived,
			SlowMovingProducts: slowMoving,
			OverstockProducts:  overstock,
		},
		ProductSignals: productSignals,
	}, nil
}

func insightInputSummaryText(input InsightInput) string {
	return fmt.Sprintf(
		"%d-day window across %d products, %d orders, %d low-stock items, %d slow-moving products, and %d overstock risks.",
		input.WindowDays,
		input.Summary.ProductCount,
		input.Summary.OrderCount,
		input.Summary.LowStockItems,
		input.Summary.SlowMovingProducts,
		input.Summary.OverstockProducts,
	)
}

func (a *App) generateSimulationInsights(input InsightInput) ([]InsightRecommendation, string, error) {
	var recommendations []InsightRecommendation
	for _, signal := range input.ProductSignals {
		if signal.SlowMoving {
			recommendations = append(recommendations, InsightRecommendation{
				Category:    "slow_moving",
				Severity:    severityForSlowMoving(signal),
				Title:       fmt.Sprintf("%s is moving slowly", signal.Name),
				Summary:     fmt.Sprintf("%s has %d units on hand and %d units sold in the last %d days.", signal.Name, signal.CurrentStock, signal.SoldUnits, input.WindowDays),
				Evidence:    fmt.Sprintf("Days since last sale: %d. Inventory value tied up: %.2f.", signal.DaysSinceLastSale, signal.InventoryValue),
				ProductID:   signal.ProductID,
				ProductSKU:  signal.SKU,
				ProductName: signal.Name,
			})
		}
		if signal.OverstockRisk {
			recommendations = append(recommendations, InsightRecommendation{
				Category:    "overstock_risk",
				Severity:    severityForOverstock(signal),
				Title:       fmt.Sprintf("%s may be overstocked", signal.Name),
				Summary:     fmt.Sprintf("%s is carrying %d units against a reorder level of %d and %d sold units in the recent window.", signal.Name, signal.CurrentStock, signal.ReorderLevel, signal.SoldUnits),
				Evidence:    fmt.Sprintf("Stock exceeds heuristic threshold while current inventory value is %.2f.", signal.InventoryValue),
				ProductID:   signal.ProductID,
				ProductSKU:  signal.SKU,
				ProductName: signal.Name,
			})
		}
		if signal.LowStock && signal.SoldUnits > 0 {
			recommendations = append(recommendations, InsightRecommendation{
				Category:    "demand_trend",
				Severity:    "medium",
				Title:       fmt.Sprintf("%s needs replenishment attention", signal.Name),
				Summary:     fmt.Sprintf("%s is below reorder level with %d units remaining and %d sold units in the last %d days.", signal.Name, signal.CurrentStock, signal.SoldUnits, input.WindowDays),
				Evidence:    fmt.Sprintf("Reorder level is %d and recent sales velocity is %.2f units per day.", signal.ReorderLevel, signal.SalesVelocity),
				ProductID:   signal.ProductID,
				ProductSKU:  signal.SKU,
				ProductName: signal.Name,
			})
		}
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, InsightRecommendation{
			Category: "demand_trend",
			Severity: "low",
			Title:    "Inventory health is stable",
			Summary:  fmt.Sprintf("No major slow-moving, overstock, or urgent replenishment signals were detected in the last %d days.", input.WindowDays),
			Evidence: fmt.Sprintf("Reviewed %d products and %d orders.", input.Summary.ProductCount, input.Summary.OrderCount),
		})
	}
	if len(recommendations) > 6 {
		recommendations = recommendations[:6]
	}
	raw, _ := json.MarshalIndent(map[string]any{
		"mode":            "simulation",
		"recommendations": recommendations,
	}, "", "  ")
	return recommendations, string(raw), nil
}

func severityForSlowMoving(signal InsightProductData) string {
	if signal.InventoryValue >= 1000 || signal.DaysSinceLastSale >= 90 {
		return "high"
	}
	return "medium"
}

func severityForOverstock(signal InsightProductData) string {
	if signal.CurrentStock >= maxInt(signal.ReorderLevel*5, signal.SoldUnits*3) {
		return "high"
	}
	return "medium"
}

func (a *App) generateRealInsights(input InsightInput) ([]InsightRecommendation, string, error) {
	if strings.TrimSpace(a.ai.APIKey) == "" {
		return nil, "", errors.New("real AI mode requires OPENAI_API_KEY")
	}

	payloadBytes, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return nil, "", err
	}

	requestBody := map[string]any{
		"model": a.ai.Model,
		"input": "You are an inventory analyst for a small retail and e-commerce business.\n" +
			"Return valid JSON only with this shape: " +
			`{"recommendations":[{"category":"slow_moving|overstock_risk|demand_trend","severity":"low|medium|high","title":"...","summary":"...","evidence":"...","productSku":"...","productName":"..."}]}` +
			"\nDo not suggest automatic actions. Recommendations must remain advisory and grounded in the supplied metrics.\n\n" +
			"Inventory analysis input:\n" + string(payloadBytes),
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, strings.TrimRight(a.ai.BaseURL, "/")+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+a.ai.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.ai.HTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	rawOutput := string(responseBody)
	if resp.StatusCode >= 300 {
		return nil, rawOutput, fmt.Errorf("OpenAI API error: %s", resp.Status)
	}

	var envelope openAIResponseEnvelope
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return nil, rawOutput, err
	}
	if envelope.Error != nil && envelope.Error.Message != "" {
		return nil, rawOutput, errors.New(envelope.Error.Message)
	}

	textOutput := extractResponseText(envelope)
	if textOutput == "" {
		return nil, rawOutput, errors.New("model returned no text output")
	}

	var parsed struct {
		Recommendations []InsightRecommendation `json:"recommendations"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(textOutput)), &parsed); err != nil {
		return nil, rawOutput, fmt.Errorf("failed to parse model JSON output: %w", err)
	}
	if len(parsed.Recommendations) == 0 {
		return nil, rawOutput, errors.New("model returned no recommendations")
	}
	return parsed.Recommendations, rawOutput, nil
}

func extractResponseText(envelope openAIResponseEnvelope) string {
	if strings.TrimSpace(envelope.OutputText) != "" {
		return strings.TrimSpace(envelope.OutputText)
	}
	var parts []string
	for _, output := range envelope.Output {
		for _, content := range output.Content {
			if strings.TrimSpace(content.Text) != "" {
				parts = append(parts, strings.TrimSpace(content.Text))
			}
		}
	}
	return strings.Join(parts, "\n")
}

func extractJSONObject(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return text
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
