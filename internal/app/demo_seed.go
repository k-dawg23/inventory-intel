package app

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

const demoSeedVersion = "techtrend-direct-v1"

type demoProductSeed struct {
	Name     string
	Category string
	Tier     string
}

type demoProductRuntime struct {
	ID           int64
	SKU          string
	Name         string
	Category     string
	Tier         string
	CurrentStock int
	ReorderLevel int
	UnitCost     float64
	SellingPrice float64
	DemandWeight int
}

type demoSupplierSeed struct {
	Name        string
	ContactName string
	Email       string
	Phone       string
	Notes       string
	Group       string
}

func (a *App) ensureDemoSeedState() error {
	_, err := a.db.Exec(`
		CREATE TABLE IF NOT EXISTS demo_seed_state (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			version TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`)
	return err
}

func (a *App) currentDemoSeedVersion() (string, error) {
	var version string
	err := a.db.QueryRow(`SELECT version FROM demo_seed_state WHERE id = 1`).Scan(&version)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return version, err
}

func storeDemoSeedVersionTx(tx *sql.Tx, version string) error {
	_, err := tx.Exec(`
		INSERT INTO demo_seed_state (id, version, updated_at)
		VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET version = excluded.version, updated_at = excluded.updated_at`,
		version, nowString(),
	)
	return err
}

func resetDemoSeedTx(tx *sql.Tx) error {
	for _, stmt := range []string{
		`DELETE FROM ai_insight_runs`,
		`DELETE FROM audit_events`,
		`DELETE FROM inventory_transactions`,
		`DELETE FROM customer_order_items`,
		`DELETE FROM customer_orders`,
		`DELETE FROM purchase_order_items`,
		`DELETE FROM purchase_orders`,
		`DELETE FROM supplier_products`,
		`DELETE FROM suppliers`,
		`DELETE FROM products`,
		`DELETE FROM demo_seed_state`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func seedDemoBusinessTx(tx *sql.Tx) error {
	rng := rand.New(rand.NewSource(42))
	now := time.Now().UTC()

	suppliers := demoSuppliers()
	supplierIDs := make(map[string]int64, len(suppliers))
	for _, supplier := range suppliers {
		res, err := tx.Exec(`
			INSERT INTO suppliers (name, contact_name, email, phone, notes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			supplier.Name, supplier.ContactName, supplier.Email, supplier.Phone, supplier.Notes, nowString(), nowString(),
		)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		supplierIDs[supplier.Group] = id
	}

	specs := demoProductSpecs()
	products := make([]demoProductRuntime, 0, len(specs))
	for idx, spec := range specs {
		currentStock, reorderLevel := stockForTier(spec, rng)
		unitCost := unitCostForCategory(spec.Category, rng)
		sellingPrice := sellingPriceForCategory(spec.Category, unitCost, rng)
		sku := fmt.Sprintf("SKU-%04d", 1001+idx)
		res, err := tx.Exec(`
			INSERT INTO products (sku, name, description, category, unit_cost, selling_price, current_stock, reorder_level, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)`,
			sku, spec.Name, productDescription(spec.Category), spec.Category, unitCost, sellingPrice, currentStock, reorderLevel, nowString(), nowString(),
		)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		products = append(products, demoProductRuntime{
			ID:           id,
			SKU:          sku,
			Name:         spec.Name,
			Category:     spec.Category,
			Tier:         spec.Tier,
			CurrentStock: currentStock,
			ReorderLevel: reorderLevel,
			UnitCost:     unitCost,
			SellingPrice: sellingPrice,
			DemandWeight: demandWeightForProduct(spec.Name, spec.Category, spec.Tier),
		})
		if err := linkProductToSupplier(tx, supplierIDs, id, spec.Category); err != nil {
			return err
		}
	}

	months := monthBuckets(now, []int{85, 90, 95, 100, 105, 110, 115, 120, 125, 130, 170, 255})
	poMonths := monthBuckets(now, []int{18, 18, 20, 20, 22, 22, 23, 23, 24, 25, 35, 50})
	customerStatusCounts := expandedStatuses(
		[]statusCount{{"Completed", 1275}, {"Processing", 105}, {"Pending", 75}, {"Cancelled", 45}},
		rng,
	)
	purchaseStatusCounts := expandedStatuses(
		[]statusCount{{"Received", 240}, {"Ordered", 45}, {"Draft", 15}},
		rng,
	)

	completedItemCounts := expandedCounts([]countBucket{{4, 750}, {5, 525}}, rng)
	receivedItemCounts := expandedCounts([]countBucket{{3, 15}, {4, 45}, {5, 180}}, rng)

	customerCustomers := demoCustomerNames()
	purchaseOrderID := int64(0)
	customerOrderID := int64(0)
	completedIndex := 0
	receivedIndex := 0

	customerOrderDates := months
	purchaseOrderDates := poMonths

	for i := 0; i < len(purchaseStatusCounts); i++ {
		status := purchaseStatusCounts[i]
		date := purchaseOrderDates[i%len(purchaseOrderDates)]
		supplierGroup := supplierGroups[(i/3)%len(supplierGroups)]
		supplierID := supplierIDs[supplierGroup]
		itemCount := 2 + rng.Intn(2)
		if status.Value == "Received" && receivedIndex < len(receivedItemCounts) {
			itemCount = receivedItemCounts[receivedIndex]
			receivedIndex++
		}
		items := pickPurchaseItems(products, rng, itemCount)
		poID, err := insertPurchaseOrderTx(tx, supplierID, status.Value, "demo procurement cycle", items, date.UTC().Format(time.RFC3339))
		if err != nil {
			return err
		}
		purchaseOrderID = poID
		if status.Value == "Received" {
			for _, item := range items {
				if err := insertInventoryTransactionTx(tx, item.ProductID, "received", item.Quantity, "purchase_order", poID, "stock received from supplier", supplierGroupActor(supplierGroup), date); err != nil {
					return err
				}
			}
		}
	}

	for i := 0; i < len(customerStatusCounts); i++ {
		status := customerStatusCounts[i]
		date := customerOrderDates[i%len(customerOrderDates)]
		customerName := customerCustomers[i%len(customerCustomers)]
		itemCount := 1 + rng.Intn(3)
		if status.Value == "Completed" && completedIndex < len(completedItemCounts) {
			itemCount = completedItemCounts[completedIndex]
			completedIndex++
		} else if status.Value == "Cancelled" {
			itemCount = 1 + rng.Intn(2)
		}
		items := pickCustomerItems(products, rng, itemCount)
		coID, err := insertCustomerOrderTx(tx, customerName, status.Value, "demo demand cycle", items, date.UTC().Format(time.RFC3339))
		if err != nil {
			return err
		}
		customerOrderID = coID
		if status.Value == "Completed" {
			for _, item := range items {
				if err := insertInventoryTransactionTx(tx, item.ProductID, "sold", -item.Quantity, "customer_order", coID, "customer order shipped", customerActor(i), date); err != nil {
					return err
				}
			}
		}
	}

	return seedSupplementaryTransactions(tx, products, rng, customerOrderID, purchaseOrderID)
}

func seedSupplementaryTransactions(tx *sql.Tx, products []demoProductRuntime, rng *rand.Rand, lastCustomerOrderID, lastPurchaseOrderID int64) error {
	now := time.Now().UTC()
	returnDates := monthBuckets(now, []int{30, 28, 30, 30, 30, 32, 33, 30, 28, 28, 33, 33})
	adjustmentDates := monthBuckets(now, []int{18, 18, 18, 18, 18, 18, 19, 19, 19, 19, 20, 21})
	damageDates := monthBuckets(now, []int{12, 12, 12, 12, 12, 12, 13, 13, 13, 13, 13, 13})

	for i := 0; i < 375; i++ {
		product := pickProductByPredicate(products, rng, func(p demoProductRuntime) bool {
			return p.DemandWeight >= 5
		})
		quantity := 1 + rng.Intn(3)
		if err := insertInventoryTransactionTx(tx, product.ID, "return", quantity, "customer_order", lastCustomerOrderID, "customer return processed", customerActor(i+2000), returnDates[i%len(returnDates)]); err != nil {
			return err
		}
	}

	for i := 0; i < 225; i++ {
		product := pickProductByPredicate(products, rng, func(p demoProductRuntime) bool {
			return p.Tier != "out"
		})
		quantity := 1 + rng.Intn(4)
		if rng.Intn(2) == 0 {
			quantity = -quantity
		}
		if err := insertInventoryTransactionTx(tx, product.ID, "adjustment", quantity, "manual", 0, "inventory count correction", actorCycle(i), adjustmentDates[i%len(adjustmentDates)]); err != nil {
			return err
		}
	}

	for i := 0; i < 150; i++ {
		product := pickProductByPredicate(products, rng, func(p demoProductRuntime) bool {
			return p.CurrentStock > 0 && p.Tier != "out"
		})
		quantity := -(1 + rng.Intn(3))
		if err := insertInventoryTransactionTx(tx, product.ID, "damage", quantity, "manual", 0, "damaged in handling", actorCycle(i+500), damageDates[i%len(damageDates)]); err != nil {
			return err
		}
	}

	for i := 0; i < 750; i++ {
		product := pickProductByPredicate(products, rng, func(p demoProductRuntime) bool {
			return p.Tier != "out"
		})
		action := auditActions[i%len(auditActions)]
		entity := auditEntities[i%len(auditEntities)]
		details := fmt.Sprintf("Demo event for %s", product.Name)
		if err := auditTx(tx, actorCycle(i+800), entity, product.ID, action, details); err != nil {
			return err
		}
	}

	return nil
}

func demoSuppliers() []demoSupplierSeed {
	return []demoSupplierSeed{
		{Name: "TechSource Ltd", ContactName: "Ava Turner", Email: "ava@techsource.example", Phone: "0207 111 2201", Notes: "UK - core peripherals", Group: "peripherals"},
		{Name: "Global Peripherals Ltd", ContactName: "Mila Jensen", Email: "mila@globalperipherals.example", Phone: "+49 30 555 201", Notes: "Germany - high-volume accessories", Group: "peripherals"},
		{Name: "Bright Electronics", ContactName: "Jonas de Vries", Email: "jonas@brightelectronics.example", Phone: "+31 20 555 202", Notes: "Netherlands - display hardware", Group: "display"},
		{Name: "Apex Imports", ContactName: "Ling Chen", Email: "ling@apeximports.example", Phone: "+86 21 555 203", Notes: "China - mixed imports", Group: "import"},
		{Name: "Digital Wholesale UK", ContactName: "Sophie Patel", Email: "sophie@digitalwholesale.example", Phone: "0207 111 2205", Notes: "UK - mobile accessories", Group: "mobile"},
		{Name: "Prime Components", ContactName: "Mateusz Kowalski", Email: "mateusz@primecomponents.example", Phone: "+48 22 555 206", Notes: "Poland - office hardware", Group: "office"},
		{Name: "OfficeGear Distribution", ContactName: "Hannah Reed", Email: "hannah@officegear.example", Phone: "0207 111 2207", Notes: "UK - office equipment", Group: "office"},
		{Name: "Nexus Electronics", ContactName: "Wei Huang", Email: "wei@nexuselectronics.example", Phone: "+886 2 555 208", Notes: "Taiwan - premium hardware", Group: "display"},
		{Name: "BluePeak Trading", ContactName: "Elena Rossi", Email: "elena@bluepeaktrading.example", Phone: "+39 02 555 209", Notes: "Italy - power and charging", Group: "mobile"},
		{Name: "Northline Supply Co", ContactName: "Oliver Grant", Email: "oliver@northline.example", Phone: "0207 111 2210", Notes: "UK - fulfillment support", Group: "import"},
	}
}

func demoProductSpecs() []demoProductSeed {
	specs := []demoProductSeed{
		{Name: "Mechanical Keyboard MK100", Category: "Keyboards", Tier: "healthy"},
		{Name: "Wireless Mouse Lite", Category: "Mice", Tier: "low"},
		{Name: "USB-C Dock", Category: "Office Equipment", Tier: "moderate"},
	}
	appendTier := func(category, tier string, names ...string) {
		for _, name := range names {
			specs = append(specs, demoProductSeed{Name: name, Category: category, Tier: tier})
		}
	}

	appendTier("Keyboards", "healthy",
		"Mechanical Keyboard MK200 RGB",
		"Compact Wireless Keyboard",
		"Ergonomic Office Keyboard",
		"Mechanical Keyboard MK300 Silent",
		"Mechanical Keyboard MK500 TKL",
		"Slim Bluetooth Keyboard",
		"Split Ergonomic Keyboard",
	)
	appendTier("Mice", "healthy",
		"Gaming Mouse Pro",
		"Ergonomic Mouse Plus",
		"Precision Mouse Pro",
		"Compact Wireless Mouse",
		"Silent Office Mouse",
		"RGB Gaming Mouse Elite",
		"Bluetooth Mouse Air",
	)
	appendTier("Headsets", "healthy",
		"Gaming Headset X2 Wireless",
		"Office USB Headset",
		"Noise Cancelling Headset",
		"Stereo Headset Pro",
		"Wireless Conference Headset",
		"Bluetooth Headset Air",
		"Call Centre Headset",
	)
	appendTier("Monitors", "healthy",
		"24\" Monitor",
		"27\" Gaming Monitor",
		"Ultrawide Monitor",
		"24\" USB-C Monitor",
		"Thin Bezel Office Monitor",
	)
	appendTier("Mobile Accessories", "healthy",
		"USB-C Cable",
		"Lightning Cable",
		"Phone Stand",
		"Screen Protector Pack",
		"Power Bank 20k",
		"Wireless Charger Pad",
		"MagSafe Charger",
	)
	appendTier("Office Equipment", "healthy",
		"Webcam HD",
		"Desk Lamp",
		"Laptop Stand",
		"USB Dock",
	)
	appendTier("Cables & Power", "healthy",
		"GaN Charger 65W",
	)
	appendTier("Storage & Hubs", "healthy",
		"Portable SSD 1TB",
	)

	appendTier("Mice", "low",
		"Bluetooth Travel Mouse",
	)
	appendTier("Headsets", "low",
		"Over-Ear Headset Lite",
		"Kids Headset SafeSound",
	)
	appendTier("Mobile Accessories", "low",
		"Phone Grip",
		"Tablet Sleeve",
		"Car Charger Dual Port",
	)
	appendTier("Office Equipment", "low",
		"Document Scanner Lite",
	)
	appendTier("Cables & Power", "low",
		"HDMI 2m Cable",
		"USB-A to C Cable",
	)
	appendTier("Storage & Hubs", "low",
		"USB-C Hub 8-in-1",
		"SD Card Reader Pro",
	)

	appendTier("Keyboards", "overstock",
		"Mechanical Keyboard MK700 Pro",
		"Keyboard Wrist Rest Bundle",
	)
	appendTier("Mice", "overstock",
		"Mouse and Pad Combo",
	)
	appendTier("Headsets", "overstock",
		"Gaming Headset X1",
		"Gaming Headset X3 Surround",
		"Headset Stand Bundle",
	)
	appendTier("Mobile Accessories", "overstock",
		"Power Bank 10k",
		"USB-C Cable 3 Pack",
		"Wireless Charger Duo",
		"Phone Stand XL",
	)
	appendTier("Office Equipment", "overstock",
		"Monitor Arm Kit",
		"Ring Light Desk",
	)
	appendTier("Cables & Power", "overstock",
		"Extension Lead 4-Socket",
	)
	appendTier("Storage & Hubs", "overstock",
		"USB Hub Ultra 12-in-1",
		"Docking Station Pro",
	)

	appendTier("Monitors", "out",
		"Portable Monitor 15.6\"",
	)
	appendTier("Mobile Accessories", "out",
		"Wireless Charger",
	)
	appendTier("Cables & Power", "out",
		"Travel Charger 45W",
	)
	appendTier("Storage & Hubs", "out",
		"Portable SSD 512GB",
	)

	appendTier("Keyboards", "moderate",
		"Gaming Keyboard Pro",
		"Backlit Office Keyboard",
		"Programmable Macro Keyboard",
		"Mechanical Keyboard MK400 Pro",
	)
	appendTier("Mice", "moderate",
		"Gaming Mouse X2",
		"Vertical Ergonomic Mouse",
		"Rechargeable Mouse Air",
		"Trackball Mouse Desk",
	)
	appendTier("Headsets", "moderate",
		"Streamer Headset Pro",
		"USB-C Work Headset",
	)
	appendTier("Monitors", "moderate",
		"32\" Monitor",
		"27\" HDR Monitor",
		"Curved Gaming Monitor",
		"34\" Ultrawide Monitor",
		"4K Creator Monitor",
		"Monitor Arm Kit Pro",
	)
	appendTier("Mobile Accessories", "moderate",
		"Wireless Power Bank Stand",
		"Phone Case Pack",
		"Screen Guard 3-Pack",
		"USB-C Car Adapter",
		"Tablet Stand Foldable",
		"Fast Charge Cable",
		"MagSafe Wallet",
		"Desk Cable Clip",
		"Bluetooth Speaker Mini",
		"Travel Adapter Kit",
		"PopSocket Duo",
		"Phone Camera Grip",
	)
	appendTier("Office Equipment", "moderate",
		"Whiteboard Planner",
		"Ergonomic Foot Rest",
		"Cable Management Tray",
	)
	appendTier("Cables & Power", "moderate",
		"Cable Pack Assorted",
	)
	appendTier("Storage & Hubs", "moderate",
	)

	return specs
}

type statusCount struct {
	Value string
	Count int
}

type countBucket struct {
	Value int
	Count int
}

func expandedStatuses(items []statusCount, rng *rand.Rand) []statusCount {
	out := make([]statusCount, 0)
	for _, item := range items {
		for i := 0; i < item.Count; i++ {
			out = append(out, statusCount{Value: item.Value, Count: 1})
		}
	}
	rng.Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}

func expandedCounts(items []countBucket, rng *rand.Rand) []int {
	out := make([]int, 0)
	for _, item := range items {
		for i := 0; i < item.Count; i++ {
			out = append(out, item.Value)
		}
	}
	rng.Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}

func monthBuckets(now time.Time, counts []int) []time.Time {
	out := make([]time.Time, 0)
	year := now.Year()
	month := int(now.Month())
	for idx, count := range counts {
		currentMonth := month - len(counts) + idx + 1
		currentYear := year
		for currentMonth <= 0 {
			currentMonth += 12
			currentYear--
		}
		for i := 0; i < count; i++ {
			day := 1 + (i % 24)
			hour := 8 + (i % 10)
			out = append(out, time.Date(currentYear, time.Month(currentMonth), day, hour, i%60, 0, 0, time.UTC))
		}
	}
	return out
}

func demoCustomerNames() []string {
	first := []string{"Olivia", "Noah", "Mia", "Liam", "Ava", "Ethan", "Sophia", "Lucas", "Grace", "Jack", "Ella", "Henry", "Zoe", "Leo", "Ruby", "Oscar", "Iris", "Theo"}
	last := []string{"Bennett", "Morgan", "Patel", "Hughes", "Clark", "Turner", "Reed", "Baker", "Hill", "Cole", "Ward", "Foster", "Price", "Brooks", "Jenkins"}
	suffix := []string{"Studio", "Retail", "Outlet", "Works", "Lab", "Store", "Market"}
	out := make([]string, 0, 120)
	for i := 0; i < 120; i++ {
		name := fmt.Sprintf("%s %s", first[i%len(first)], last[(i*3)%len(last)])
		if i%5 == 0 {
			name = fmt.Sprintf("%s %s", name, suffix[i%len(suffix)])
		}
		out = append(out, name)
	}
	return out
}

var supplierGroups = []string{"peripherals", "display", "mobile", "office", "import"}

func linkProductToSupplier(tx *sql.Tx, supplierIDs map[string]int64, productID int64, category string) error {
	group := supplierGroupForCategory(category)
	supplierID := supplierIDs[group]
	if supplierID == 0 {
		return fmt.Errorf("no supplier for group %s", group)
	}
	if _, err := tx.Exec(`INSERT INTO supplier_products (supplier_id, product_id) VALUES (?, ?)`, supplierID, productID); err != nil {
		return err
	}
	return nil
}

func supplierGroupForCategory(category string) string {
	switch category {
	case "Monitors", "Storage & Hubs":
		return "display"
	case "Mobile Accessories", "Cables & Power":
		return "mobile"
	case "Office Equipment":
		return "office"
	default:
		return "peripherals"
	}
}

func productDescription(category string) string {
	switch category {
	case "Keyboards":
		return "Designed for compact productivity and responsive typing."
	case "Mice":
		return "Built for fast navigation, comfort, and precision."
	case "Headsets":
		return "Optimized for calls, gaming, and all-day audio comfort."
	case "Monitors":
		return "High-clarity display hardware for retail and creator workflows."
	case "Mobile Accessories":
		return "Essential add-ons for modern phones and tablets."
	case "Office Equipment":
		return "Practical desk hardware for distributed and home office setups."
	case "Cables & Power":
		return "Reliable power and connectivity accessories."
	case "Storage & Hubs":
		return "Fast storage and connectivity expansions."
	default:
		return "Demo product for e-commerce inventory workflows."
	}
}

func stockForTier(spec demoProductSeed, rng *rand.Rand) (int, int) {
	switch spec.Tier {
	case "healthy":
		switch spec.Name {
		case "Mechanical Keyboard MK100":
			return 198, 50
		case "Mechanical Keyboard MK200 RGB":
			return 176, 50
		case "USB-C Cable":
			return 210, 50
		case "Webcam HD":
			return 162, 40
		case "Laptop Stand":
			return 144, 40
		default:
			return 120 + rng.Intn(85), 50
		}
	case "low":
		switch spec.Name {
		case "Wireless Mouse Lite":
			return 12, 25
		case "USB-C Hub 8-in-1":
			return 18, 25
		default:
			return 8 + rng.Intn(11), 25
		}
	case "overstock":
		switch spec.Name {
		case "USB Hub Ultra 12-in-1":
			return 280, 100
		case "Gaming Headset X1":
			return 500, 100
		default:
			return 250 + rng.Intn(450), 100
		}
	case "out":
		return 0, 50
	default:
		switch spec.Name {
		case "USB-C Dock":
			return 60, 25
		case "Portable SSD 2TB":
			return 72, 25
		default:
			return 40 + rng.Intn(45), 25
		}
	}
}

func unitCostForCategory(category string, rng *rand.Rand) float64 {
	switch category {
	case "Keyboards":
		return 18 + rng.Float64()*28
	case "Mice":
		return 10 + rng.Float64()*18
	case "Headsets":
		return 18 + rng.Float64()*36
	case "Monitors":
		return 70 + rng.Float64()*110
	case "Mobile Accessories":
		return 4 + rng.Float64()*14
	case "Office Equipment":
		return 14 + rng.Float64()*52
	case "Cables & Power":
		return 3 + rng.Float64()*12
	case "Storage & Hubs":
		return 15 + rng.Float64()*95
	default:
		return 10 + rng.Float64()*20
	}
}

func sellingPriceForCategory(category string, unitCost float64, rng *rand.Rand) float64 {
	switch category {
	case "Monitors":
		return unitCost * (1.18 + rng.Float64()*0.12)
	case "Mobile Accessories", "Cables & Power":
		return unitCost * (1.9 + rng.Float64()*0.35)
	default:
		return unitCost * (1.75 + rng.Float64()*0.35)
	}
}

func demandWeightForProduct(name, category, tier string) int {
	switch name {
	case "Wireless Mouse Lite":
		return 12
	case "USB-C Cable":
		return 11
	case "Phone Stand":
		return 10
	case "Webcam HD":
		return 9
	case "Laptop Stand":
		return 8
	case "Mechanical Keyboard MK100":
		return 9
	case "Mechanical Keyboard MK200 RGB":
		return 8
	case "USB-C Dock":
		return 7
	case "Gaming Headset X1":
		return 1
	case "USB Hub Ultra 12-in-1":
		return 1
	}
	switch tier {
	case "healthy":
		if category == "Monitors" {
			return 6
		}
		return 7
	case "low":
		return 5
	case "overstock":
		return 1
	case "out":
		return 1
	default:
		return 5
	}
}

func pickCustomerItems(products []demoProductRuntime, rng *rand.Rand, itemCount int) []CustomerOrderItem {
	selected := sampleProducts(products, rng, itemCount, func(p demoProductRuntime) int {
		return maxInt(1, p.DemandWeight)
	})
	items := make([]CustomerOrderItem, 0, len(selected))
	for _, product := range selected {
		items = append(items, CustomerOrderItem{
			ProductID: product.ID,
			Quantity:  1 + rng.Intn(3),
			UnitPrice: product.SellingPrice,
		})
	}
	return items
}

func pickPurchaseItems(products []demoProductRuntime, rng *rand.Rand, itemCount int) []PurchaseOrderItem {
	selected := sampleProducts(products, rng, itemCount, func(p demoProductRuntime) int {
		score := p.ReorderLevel*2 - p.CurrentStock/3
		if score < 1 {
			score = 1
		}
		return score + p.DemandWeight/2
	})
	items := make([]PurchaseOrderItem, 0, len(selected))
	for _, product := range selected {
		items = append(items, PurchaseOrderItem{
			ProductID: product.ID,
			Quantity:  12 + rng.Intn(28),
			UnitCost:  product.UnitCost,
		})
	}
	return items
}

func sampleProducts(products []demoProductRuntime, rng *rand.Rand, count int, weightFn func(demoProductRuntime) int) []demoProductRuntime {
	pool := append([]demoProductRuntime(nil), products...)
	selected := make([]demoProductRuntime, 0, count)
	for len(selected) < count && len(pool) > 0 {
		total := 0
		for _, product := range pool {
			total += maxInt(1, weightFn(product))
		}
		pick := rng.Intn(total)
		running := 0
		for idx, product := range pool {
			running += maxInt(1, weightFn(product))
			if pick < running {
				selected = append(selected, product)
				pool = append(pool[:idx], pool[idx+1:]...)
				break
			}
		}
	}
	return selected
}

func pickProductByPredicate(products []demoProductRuntime, rng *rand.Rand, predicate func(demoProductRuntime) bool) demoProductRuntime {
	candidates := make([]demoProductRuntime, 0)
	for _, product := range products {
		if predicate(product) {
			candidates = append(candidates, product)
		}
	}
	if len(candidates) == 0 {
		return products[rng.Intn(len(products))]
	}
	return candidates[rng.Intn(len(candidates))]
}

func insertInventoryTransactionTx(tx *sql.Tx, productID int64, transactionType string, quantity int, referenceType string, referenceID int64, reason string, actor string, when time.Time) error {
	_, err := tx.Exec(
		`INSERT INTO inventory_transactions (product_id, transaction_type, quantity, reference_type, reference_id, reason, actor, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		productID, transactionType, quantity, referenceType, referenceID, reason, actor, when.UTC().Format(time.RFC3339),
	)
	return err
}

type auditGroup struct {
	Actor      string
	EntityType string
	Action     string
}

var auditActions = []string{"created", "updated", "stock_adjusted", "received", "processed", "reviewed"}
var auditEntities = []string{"product", "supplier", "purchase_order", "customer_order", "inventory", "ai_insight"}

func actorCycle(index int) string {
	actors := []string{"Admin User", "Warehouse Manager", "Inventory Clerk", "Operations Manager"}
	return actors[index%len(actors)]
}

func customerActor(index int) string {
	actors := []string{"Admin User", "Warehouse Manager", "Inventory Clerk", "Operations Manager"}
	return actors[index%len(actors)]
}

func supplierGroupActor(group string) string {
	switch group {
	case "peripherals":
		return "Warehouse Manager"
	case "display":
		return "Operations Manager"
	case "mobile":
		return "Inventory Clerk"
	default:
		return "Admin User"
	}
}
