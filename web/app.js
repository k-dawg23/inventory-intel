const state = {
  actor: "Kenneth",
  currentView: "dashboard",
  data: null,
  managementPanels: {
    products: "list",
    suppliers: "list",
    purchaseOrders: "list",
    customerOrders: "list",
  },
};

const currency = new Intl.NumberFormat("en-GB", { style: "currency", currency: "GBP", maximumFractionDigits: 0 });
const compactCurrency = new Intl.NumberFormat("en-GB", { style: "currency", currency: "GBP", notation: "compact", maximumFractionDigits: 1 });

const viewMeta = {
  dashboard: {
    title: "Dashboard",
    subtitle: "Overview of your inventory operations",
  },
  products: {
    title: "Products",
    subtitle: "Manage your catalog, stock levels, and pricing in one place",
  },
  suppliers: {
    title: "Suppliers",
    subtitle: "Track supplier details and the products they support",
  },
  purchaseOrders: {
    title: "Purchase Orders",
    subtitle: "Control inbound inventory and receiving workflows",
  },
  customerOrders: {
    title: "Customer Orders",
    subtitle: "Monitor outbound order flow and shipment readiness",
  },
  inventory: {
    title: "Inventory Ledger",
    subtitle: "Review every stock-affecting movement across the business",
  },
  imports: {
    title: "Imports / Exports",
    subtitle: "Move operational data in and out with clean CSV workflows",
  },
  insights: {
    title: "AI Insights",
    subtitle: "Review advisory recommendations powered by inventory activity",
  },
  audit: {
    title: "Audit Log",
    subtitle: "See who changed what and when across the system",
  },
};

document.addEventListener("DOMContentLoaded", () => {
  const hashView = window.location.hash.replace("#", "");
  if (viewMeta[hashView]) {
    state.currentView = hashView;
  }
  bindShell();
  loadData();
});

function bindShell() {
  document.getElementById("menuToggle").addEventListener("click", () => setSidebarOpen(true));
  document.getElementById("sidebarBackdrop").addEventListener("click", () => setSidebarOpen(false));
  document.getElementById("refreshBtn").addEventListener("click", loadData);
  document.getElementById("actorInput").addEventListener("input", (event) => {
    state.actor = event.target.value || "Kenneth";
    updateOperatorDisplay();
  });

  document.querySelectorAll(".nav-link").forEach((button) => {
    button.addEventListener("click", () => switchView(button.dataset.view));
  });
}

function setSidebarOpen(isOpen) {
  document.body.classList.toggle("menu-open", isOpen);
}

function switchView(view) {
  state.currentView = view;
  if (window.location.hash.replace("#", "") !== view) {
    history.replaceState(null, "", `#${view}`);
  }
  document.querySelectorAll(".nav-link").forEach((button) => {
    button.classList.toggle("active", button.dataset.view === view);
  });
  document.querySelectorAll(".view").forEach((section) => {
    section.classList.toggle("active", section.id === view);
  });
  updateHeader();
  if (window.innerWidth < 1200) {
    setSidebarOpen(false);
  }
}

function updateHeader() {
  const meta = viewMeta[state.currentView] || viewMeta.dashboard;
  document.getElementById("pageTitle").textContent = meta.title;
  document.getElementById("pageSubtitle").textContent = meta.subtitle;
}

function updateOperatorDisplay() {
  const actor = state.actor || "Kenneth";
  document.getElementById("operatorName").textContent = actor;
  document.getElementById("operatorAvatar").textContent = actor.trim().charAt(0).toUpperCase() || "K";
}

async function loadData() {
  try {
    const response = await fetch("/api/bootstrap");
    state.data = await response.json();
    renderAll();
    flash("Live data refreshed.");
  } catch (error) {
    flash(error.message || "Unable to load data.", "error");
  }
}

function renderAll() {
  updateOperatorDisplay();
  updateHeader();
  renderDashboard();
  renderProducts();
  renderSuppliers();
  renderPurchaseOrders();
  renderCustomerOrders();
  renderInventory();
  renderImports();
  renderInsights();
  renderAudit();
  switchView(state.currentView);
}

function flash(message, type = "info") {
  const node = document.getElementById("flash");
  node.textContent = message;
  node.classList.remove("hidden");
  node.style.background = type === "error" ? "#fff1f2" : "#ffffff";
  node.style.borderColor = type === "error" ? "#fecdd3" : "#e2e8f0";
  node.style.color = type === "error" ? "#b91c1c" : "#334155";
  clearTimeout(flash.timer);
  flash.timer = setTimeout(() => node.classList.add("hidden"), 2400);
}

async function api(path, options = {}) {
  const isFormData = options.body instanceof FormData;
  const headers = {
    "X-Actor": state.actor,
    ...(options.headers || {}),
  };
  if (!isFormData) {
    headers["Content-Type"] = "application/json";
  }
  const response = await fetch(path, { ...options, headers });
  if (!response.ok) {
    throw new Error(await response.text());
  }
  const contentType = response.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    return response.json();
  }
  return response.text();
}

function renderDashboard() {
  const { products, suppliers, customerOrders, transactions, auditEvents, insightRuns } = state.data;
  const dashboard = state.data.dashboard;
  const inventorySeries = buildInventoryValueSeries(products, transactions, 6);
  const orderSeries = buildOrdersSeries(customerOrders, 6);
  const lowStockProducts = products.filter((product) => product.currentStock <= product.reorderLevel).sort((a, b) => a.currentStock - b.currentStock);
  const outOfStock = products.filter((product) => product.currentStock === 0).length;
  const latestInsight = insightRuns?.[0] || dashboard.latestInsight;
  const topSelling = dashboard.topSelling || [];

  document.getElementById("dashboard").innerHTML = `
    <div class="section-grid">
      <section class="kpi-grid">
        ${metricCard("Total Products", products.length, buildDelta(products.length, Math.max(1, lowStockProducts.length), "catalog active"), "box")}
        ${metricCard("Total Suppliers", suppliers.length, buildFlatDelta("partner base stable"), "users")}
        ${metricCard("Total Orders", customerOrders.length, buildDelta(customerOrders.length, 1, "recent order activity"), "clipboard")}
        ${metricCard("Low Stock Items", lowStockProducts.length, buildWarningDelta(lowStockProducts.length, "reorder attention"), "alert")}
        ${metricCard("Inventory Value", currency.format(dashboard.inventoryValue || 0), buildPositivePercentDelta(inventorySeries), "coin")}
        ${metricCard("Out of Stock", outOfStock, outOfStock ? { tone: "down", label: `${outOfStock} items unavailable` } : { tone: "flat", label: "No stockouts today" }, "ban")}
      </section>

      <section class="dashboard-panels">
        <article class="card">
          <div class="card-header">
            <div>
              <h3>Low Stock Alerts</h3>
              <p class="subtle-text">Products nearing or below reorder threshold</p>
            </div>
            <button class="secondary-button nav-jump" data-target="products" type="button">View all</button>
          </div>
          <div class="list-stack">
            ${lowStockProducts.slice(0, 4).map(renderLowStockRow).join("") || `<div class="empty-state">No low stock alerts right now.</div>`}
          </div>
        </article>

        <article class="card">
          <div class="card-header">
            <div>
              <h3>Recent Audit</h3>
              <p class="subtle-text">Latest operational activity across the system</p>
            </div>
            <button class="secondary-button nav-jump" data-target="audit" type="button">View all</button>
          </div>
          <div class="audit-stack">
            ${auditEvents.slice(0, 5).map(renderAuditRow).join("")}
          </div>
        </article>

        <article class="card">
          <div class="card-header">
            <div>
              <h3>AI Insight of the Day</h3>
              <p class="subtle-text">${latestInsight ? `${latestInsight.mode} mode • ${latestInsight.model}` : "Generate a new advisory snapshot"}</p>
            </div>
            <button class="secondary-button nav-jump" data-target="insights" type="button">View AI Insights</button>
          </div>
          <div class="ai-feature">
            <div class="ai-highlight">
              ${latestInsight && latestInsight.recommendations.length
                ? `
                  <h4>${latestInsight.recommendations[0].title}</h4>
                  <p>${latestInsight.recommendations[0].summary}</p>
                  <button class="primary-button nav-jump" data-target="insights" type="button">View AI Insights</button>
                `
                : `
                  <h4>Overstock risk detected</h4>
                  <p>You have products with more than 6 months of stock based on current sales velocity.</p>
                  <button id="generateInsightsFromDashboard" class="primary-button" type="button">Generate Insights</button>
                `}
            </div>
            ${latestInsight ? `<div class="insight-stack">${latestInsight.recommendations.slice(1, 3).map(renderMiniInsightRow).join("")}</div>` : ""}
          </div>
        </article>
      </section>

      <section class="chart-panels">
        ${chartCard("Orders Over Time", orderSeries, "Total Orders", customerOrders.length, "vs recent periods")}
        ${areaChartCard("Inventory Value Over Time", inventorySeries, "Current Value", currency.format(dashboard.inventoryValue || 0), "latest stock valuation")}
        <article class="card">
          <div class="card-header">
            <div>
              <h3>Top Selling Products</h3>
              <p class="subtle-text">Best performers based on sold units</p>
            </div>
            <button class="ghost-button" type="button">Last 30 days</button>
          </div>
          <div class="bar-list">
            ${topSelling.map(renderBarRow).join("") || `<div class="empty-state">Top selling data will appear after more outbound orders.</div>`}
          </div>
        </article>
      </section>
    </div>
  `;

  document.querySelectorAll(".nav-jump").forEach((button) => {
    button.addEventListener("click", () => switchView(button.dataset.target));
  });
  const generateButton = document.getElementById("generateInsightsFromDashboard");
  if (generateButton) {
    generateButton.addEventListener("click", refreshInsights);
  }
}

function metricCard(label, value, delta, iconName) {
  return `
    <article class="card kpi-card">
      <div class="kpi-top">
        <div class="metric-icon">${icon(iconName)}</div>
      </div>
      <div>
        <p class="metric-label">${label}</p>
        <div class="metric-value">
          <strong>${value}</strong>
          ${delta ? `<span class="delta ${delta.tone}">${deltaIcon(delta.tone)} ${delta.value || ""}</span>` : ""}
        </div>
        <div class="metric-footnote">${delta?.label || "Live snapshot"}</div>
      </div>
    </article>
  `;
}

function renderLowStockRow(product) {
  return `
    <div class="alert-row">
      <div class="item-icon">${product.name.charAt(0)}</div>
      <div>
        <p class="item-title">${product.name}</p>
        <div class="item-meta">${product.sku} • ${product.category || "Uncategorised"}</div>
      </div>
      <div>
        <span class="stock-pill">${product.currentStock} / ${product.reorderLevel}</span>
        <div class="item-meta">In stock / reorder</div>
      </div>
    </div>
  `;
}

function renderAuditRow(entry) {
  return `
    <div class="audit-row">
      <div class="item-title">${humanizeAction(entry)} </div>
      <div class="audit-meta">${timeAgo(entry.createdAt)} • ${entry.actor}</div>
    </div>
  `;
}

function renderMiniInsightRow(item) {
  return `
    <div class="insight-row">
      <div class="item-title">${item.title}</div>
      <div class="audit-meta">${item.evidence}</div>
    </div>
  `;
}

function renderBarRow(item) {
  const max = Math.max(...state.data.dashboard.topSelling.map((entry) => entry.quantity), 1);
  const width = Math.max(12, Math.round((item.quantity / max) * 100));
  return `
    <div class="bar-row">
      <div class="table-title">
        <strong>${item.name}</strong>
        <span class="table-note">${item.sku}</span>
      </div>
      <div class="bar-track"><div class="bar-fill" style="width:${width}%"></div></div>
      <strong>${item.quantity}</strong>
    </div>
  `;
}

function chartCard(title, series, summaryLabel, summaryValue, summaryNote) {
  return `
    <article class="card">
      <div class="card-header">
        <div>
          <h3>${title}</h3>
          <p class="subtle-text">Recent monthly trend</p>
        </div>
        <button class="ghost-button" type="button">Last 6 months</button>
      </div>
      <div class="chart-box">
        <div class="chart-frame">${lineChartSVG(series, "#2563EB", false)}</div>
        <div class="chart-summary">
          <div>
            <span>${summaryLabel}</span>
            <strong>${summaryValue}</strong>
          </div>
          <div class="success-text">${summaryNote}</div>
        </div>
      </div>
    </article>
  `;
}

function areaChartCard(title, series, summaryLabel, summaryValue, summaryNote) {
  return `
    <article class="card">
      <div class="card-header">
        <div>
          <h3>${title}</h3>
          <p class="subtle-text">Stock value reconstructed from movement history</p>
        </div>
        <button class="ghost-button" type="button">Last 6 months</button>
      </div>
      <div class="chart-box">
        <div class="chart-frame">${lineChartSVG(series, "#16A34A", true)}</div>
        <div class="chart-summary">
          <div>
            <span>${summaryLabel}</span>
            <strong>${summaryValue}</strong>
          </div>
          <div class="success-text">${summaryNote}</div>
        </div>
      </div>
    </article>
  `;
}

function getManagementPanel(viewKey) {
  return state.managementPanels[viewKey] || "list";
}

function setManagementPanel(viewKey, panel) {
  state.managementPanels[viewKey] = panel;
  syncManagementPanel(viewKey);
}

function openManagementForm(viewKey) {
  setManagementPanel(viewKey, "form");
}

function syncManagementPanel(viewKey) {
  const shell = document.querySelector(`[data-management-view="${viewKey}"]`);
  if (!shell) {
    return;
  }
  const activePanel = getManagementPanel(viewKey);
  shell.dataset.activePanel = activePanel;
  shell.querySelectorAll("[data-panel-toggle]").forEach((button) => {
    button.classList.toggle("active", button.dataset.panelToggle === activePanel);
  });
}

function bindManagementPanel(viewKey) {
  const shell = document.querySelector(`[data-management-view="${viewKey}"]`);
  if (!shell) {
    return;
  }
  shell.querySelectorAll("[data-panel-toggle]").forEach((button) => {
    button.addEventListener("click", () => setManagementPanel(viewKey, button.dataset.panelToggle));
  });
  syncManagementPanel(viewKey);
}

function managementShell({ viewKey, listLabel, formLabel, formTitle, formContent, listContent }) {
  return `
    <div class="management-shell" data-management-view="${viewKey}" data-active-panel="${getManagementPanel(viewKey)}">
      <div class="management-nav management-nav-mobile" aria-label="${viewMeta[viewKey].title} mobile view switcher">
        <div class="management-chip-row">
          <button class="management-chip" type="button" data-panel-toggle="list">${listLabel}</button>
          <button class="management-chip" type="button" data-panel-toggle="form">${formLabel}</button>
        </div>
        <button class="management-fab" type="button" data-panel-toggle="form" aria-label="${formLabel}">+</button>
      </div>

      <div class="management-nav management-nav-tablet" aria-label="${viewMeta[viewKey].title} tablet view switcher">
        <button class="management-tab" type="button" data-panel-toggle="list">${listLabel}</button>
        <button class="management-tab" type="button" data-panel-toggle="form">${formLabel}</button>
      </div>

      <div class="management-panel management-panel-form">
        <button class="management-back" type="button" data-panel-toggle="list">← Back to ${listLabel}</button>
        ${formContent}
      </div>

      <div class="management-panel management-panel-list">
        ${listContent}
      </div>
    </div>
  `;
}

function renderRecordMeta(items) {
  return `
    <div class="record-meta">
      ${items.map((item) => `
        <div class="record-meta-item">
          <span>${item.label}</span>
          <strong>${item.value}</strong>
        </div>
      `).join("")}
    </div>
  `;
}

function productCards(products) {
  return products.map((product) => `
    <article class="record-card">
      <div class="record-card-top">
        <div class="table-title">
          <strong>${product.name}</strong>
          <span class="table-note">${product.sku}</span>
        </div>
        ${product.currentStock === 0 ? `<span class="status-badge danger">Out of stock</span>` : product.currentStock <= product.reorderLevel ? `<span class="status-badge low">Low stock</span>` : product.active ? `<span class="status-badge success">Active</span>` : `<span class="status-badge">Inactive</span>`}
      </div>
      ${renderRecordMeta([
        { label: "Category", value: product.category || "Uncategorised" },
        { label: "Stock", value: `${product.currentStock} / ${product.reorderLevel}` },
        { label: "Price", value: currency.format(product.sellingPrice) },
      ])}
      <button class="secondary-button product-edit" data-id="${product.id}" type="button">Edit Product</button>
    </article>
  `).join("");
}

function supplierCards(suppliers) {
  return suppliers.map((supplier) => `
    <article class="record-card">
      <div class="record-card-top">
        <div class="table-title">
          <strong>${supplier.name}</strong>
          <span class="table-note">${supplier.notes || "No supplier notes yet"}</span>
        </div>
        <span class="status-badge">${(supplier.productIds || []).length} linked</span>
      </div>
      ${renderRecordMeta([
        { label: "Contact", value: supplier.contactName || "—" },
        { label: "Email", value: supplier.email || "—" },
        { label: "Phone", value: supplier.phone || "—" },
      ])}
      <button class="secondary-button supplier-edit" data-id="${supplier.id}" type="button">Edit Supplier</button>
    </article>
  `).join("");
}

function purchaseOrderCards(orders, products) {
  return orders.map((order) => `
    <article class="record-card">
      <div class="record-card-top">
        <div class="table-title">
          <strong>#${order.id}</strong>
          <span class="table-note">${order.supplier}</span>
        </div>
        ${statusBadge(order.status)}
      </div>
      ${renderRecordMeta([
        { label: "Items", value: `${order.items.length} lines` },
        { label: "Summary", value: order.items.map((item) => lineSummary(products, item)).join(", ") || "No items" },
      ])}
    </article>
  `).join("");
}

function customerOrderCards(orders) {
  return orders.map((order) => `
    <article class="record-card">
      <div class="record-card-top">
        <div class="table-title">
          <strong>#${order.id}</strong>
          <span class="table-note">${order.customerName}</span>
        </div>
        ${statusBadge(order.status)}
      </div>
      ${renderRecordMeta([
        { label: "Items", value: `${order.items.length} lines` },
        { label: "Summary", value: order.items.map((item) => `${item.product || item.productId} × ${item.quantity}`).join(", ") || "No items" },
      ])}
    </article>
  `).join("");
}

function renderProducts() {
  const products = state.data.products;
  document.getElementById("products").innerHTML = `
    <div class="screen-section">
      ${managementShell({
        viewKey: "products",
        listLabel: "Products",
        formLabel: "Create Product",
        formTitle: "Product Details",
        formContent: `
          <article class="panel-card form-card">
            <div class="panel-header">
              <div>
                <h3>Product Details</h3>
                <p class="subtle-text">Create or update catalog items without leaving the dashboard flow.</p>
              </div>
            </div>
            <form id="productForm">
              <input type="hidden" name="id" />
              <div class="form-grid two-col">
                <label class="field"><span class="field-label">SKU</span><input name="sku" required /></label>
                <label class="field"><span class="field-label">Name</span><input name="name" required /></label>
                <label class="field"><span class="field-label">Category</span><input name="category" required /></label>
                <label class="field"><span class="field-label">Unit Cost</span><input name="unitCost" type="number" step="0.01" required /></label>
                <label class="field"><span class="field-label">Selling Price</span><input name="sellingPrice" type="number" step="0.01" required /></label>
                <label class="field"><span class="field-label">Current Stock</span><input name="currentStock" type="number" required /></label>
                <label class="field"><span class="field-label">Reorder Level</span><input name="reorderLevel" type="number" required /></label>
                <label class="field"><span class="field-label">Status</span><select name="active"><option value="true">Active</option><option value="false">Inactive</option></select></label>
              </div>
              <label class="field"><span class="field-label">Description</span><textarea name="description"></textarea></label>
              <div class="action-row">
                <button class="primary-button" type="submit">Save Product</button>
                <button class="secondary-button" type="button" id="productReset">Reset</button>
              </div>
            </form>
          </article>
        `,
        listContent: `
          <article class="card">
            <div class="toolbar">
              <div>
                <h3>Product Catalog</h3>
                <p class="subtle-text">Search, filter, and review stock health across the catalog.</p>
              </div>
              <div class="table-controls">
                <input id="productSearch" placeholder="Search SKU or name" />
                <select id="productFilter">
                  <option value="all">All products</option>
                  <option value="active">Active</option>
                  <option value="inactive">Inactive</option>
                  <option value="low">Low stock</option>
                  <option value="out">Out of stock</option>
                </select>
              </div>
            </div>
            <div id="productCards" class="record-card-list mobile-only">${productCards(products)}</div>
            <div class="table-wrap tablet-up-only">
              <table>
                <thead>
                  <tr>
                    <th>Product</th>
                    <th>Category</th>
                    <th>Stock</th>
                    <th>Price</th>
                    <th>Status</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody id="productTable">${productRows(products)}</tbody>
              </table>
            </div>
          </article>
        `,
      })}

      <article class="card">
        <div class="toolbar">
          <div>
            <h3>Manual Inventory Adjustment</h3>
            <p class="subtle-text">Record corrections, returns, or damaged stock without leaving the workflow.</p>
          </div>
        </div>
        <form id="adjustmentForm" class="form-grid two-col">
          <label class="field"><span class="field-label">Product</span><select name="productId">${productOptions(products)}</select></label>
          <label class="field"><span class="field-label">Type</span><select name="type"><option value="adjusted">Manual Correction</option><option value="damaged">Damaged</option><option value="returned">Returned</option></select></label>
          <label class="field"><span class="field-label">Quantity</span><input name="quantity" type="number" required /></label>
          <label class="field"><span class="field-label">Reason</span><input name="reason" required /></label>
          <button class="primary-button" type="submit">Apply Adjustment</button>
        </form>
      </article>
    </div>
  `;

  document.getElementById("productSearch").addEventListener("input", filterProductsTable);
  document.getElementById("productFilter").addEventListener("change", filterProductsTable);
  document.getElementById("productForm").addEventListener("submit", submitProductForm);
  document.getElementById("productReset").addEventListener("click", () => document.getElementById("productForm").reset());
  document.getElementById("adjustmentForm").addEventListener("submit", submitAdjustmentForm);
  bindManagementPanel("products");
  bindProductEditButtons();
}

function filterProductsTable() {
  const query = document.getElementById("productSearch").value.toLowerCase();
  const filter = document.getElementById("productFilter").value;
  const rows = state.data.products.filter((product) => {
    const matchesQuery = !query || product.name.toLowerCase().includes(query) || product.sku.toLowerCase().includes(query);
    const matchesFilter =
      filter === "all" ||
      (filter === "active" && product.active) ||
      (filter === "inactive" && !product.active) ||
      (filter === "low" && product.currentStock <= product.reorderLevel) ||
      (filter === "out" && product.currentStock === 0);
    return matchesQuery && matchesFilter;
  });
  document.getElementById("productTable").innerHTML = productRows(rows);
  document.getElementById("productCards").innerHTML = productCards(rows);
  bindProductEditButtons();
}

function productRows(products) {
  return products.map((product) => `
    <tr>
      <td>
        <div class="table-title">
          <strong>${product.name}</strong>
          <span class="table-note">${product.sku}</span>
        </div>
      </td>
      <td>${product.category || "Uncategorised"}</td>
      <td>${product.currentStock} / ${product.reorderLevel}</td>
      <td>${currency.format(product.sellingPrice)}</td>
      <td>${product.currentStock === 0 ? `<span class="status-badge danger">Out of stock</span>` : product.currentStock <= product.reorderLevel ? `<span class="status-badge low">Low stock</span>` : product.active ? `<span class="status-badge success">Active</span>` : `<span class="status-badge">Inactive</span>`}</td>
      <td><button class="secondary-button product-edit" data-id="${product.id}" type="button">Edit</button></td>
    </tr>
  `).join("");
}

function productOptions(products) {
  return products.map((product) => `<option value="${product.id}">${product.name} (${product.sku})</option>`).join("");
}

function bindProductEditButtons() {
  document.querySelectorAll(".product-edit").forEach((button) => {
    button.addEventListener("click", () => {
      const product = state.data.products.find((item) => item.id === Number(button.dataset.id));
      const form = document.getElementById("productForm");
      Object.entries({
        id: product.id,
        sku: product.sku,
        name: product.name,
        category: product.category,
        unitCost: product.unitCost,
        sellingPrice: product.sellingPrice,
        currentStock: product.currentStock,
        reorderLevel: product.reorderLevel,
        active: String(product.active),
        description: product.description,
      }).forEach(([key, value]) => {
        form.elements[key].value = value;
      });
      openManagementForm("products");
    });
  });
}

async function submitProductForm(event) {
  event.preventDefault();
  const form = event.target;
  const payload = {
    sku: form.sku.value,
    name: form.name.value,
    category: form.category.value,
    unitCost: Number(form.unitCost.value),
    sellingPrice: Number(form.sellingPrice.value),
    currentStock: Number(form.currentStock.value),
    reorderLevel: Number(form.reorderLevel.value),
    active: form.active.value === "true",
    description: form.description.value,
  };
  try {
    await api(form.id.value ? `/api/products/${form.id.value}` : "/api/products", {
      method: form.id.value ? "PUT" : "POST",
      body: JSON.stringify(payload),
    });
    flash("Product saved.");
    form.reset();
    setManagementPanel("products", "list");
    await loadData();
  } catch (error) {
    flash(error.message, "error");
  }
}

async function submitAdjustmentForm(event) {
  event.preventDefault();
  const form = event.target;
  try {
    await api(`/api/products/${form.productId.value}/adjustments`, {
      method: "POST",
      body: JSON.stringify({
        quantity: Number(form.quantity.value),
        type: form.type.value,
        reason: form.reason.value,
      }),
    });
    flash("Inventory updated.");
    form.reset();
    await loadData();
  } catch (error) {
    flash(error.message, "error");
  }
}

function renderSuppliers() {
  const suppliers = state.data.suppliers;
  const products = state.data.products;
  document.getElementById("suppliers").innerHTML = `
    ${managementShell({
      viewKey: "suppliers",
      listLabel: "Suppliers",
      formLabel: "Create Supplier",
      formContent: `
        <article class="panel-card form-card">
          <div class="panel-header">
            <div>
              <h3>Supplier Details</h3>
              <p class="subtle-text">Keep supplier relationships and linked products tidy and easy to review.</p>
            </div>
          </div>
          <form id="supplierForm">
            <input type="hidden" name="id" />
            <div class="form-grid two-col">
              <label class="field"><span class="field-label">Name</span><input name="name" required /></label>
              <label class="field"><span class="field-label">Contact</span><input name="contactName" /></label>
              <label class="field"><span class="field-label">Email</span><input name="email" type="email" /></label>
              <label class="field"><span class="field-label">Phone</span><input name="phone" /></label>
            </div>
            <label class="field"><span class="field-label">Notes</span><textarea name="notes"></textarea></label>
            <label class="field"><span class="field-label">Linked Products</span><select name="productIds" multiple size="7">${products.map((product) => `<option value="${product.id}">${product.name}</option>`).join("")}</select></label>
            <div class="action-row">
              <button class="primary-button" type="submit">Save Supplier</button>
              <button class="secondary-button" type="button" id="supplierReset">Reset</button>
            </div>
          </form>
        </article>
      `,
      listContent: `
        <article class="card">
          <div class="toolbar">
            <div>
              <h3>Supplier Directory</h3>
              <p class="subtle-text">Search suppliers and review the breadth of their product coverage.</p>
            </div>
            <div class="table-controls">
              <input id="supplierSearch" placeholder="Search suppliers" />
            </div>
          </div>
          <div id="supplierCards" class="record-card-list mobile-only">${supplierCards(suppliers)}</div>
          <div class="table-wrap tablet-up-only">
            <table>
              <thead>
                <tr>
                  <th>Supplier</th>
                  <th>Contact</th>
                  <th>Email</th>
                  <th>Linked Products</th>
                  <th></th>
                </tr>
              </thead>
              <tbody id="supplierTable">${supplierRows(suppliers)}</tbody>
            </table>
          </div>
        </article>
      `,
    })}
  `;
  document.getElementById("supplierForm").addEventListener("submit", submitSupplierForm);
  document.getElementById("supplierReset").addEventListener("click", () => document.getElementById("supplierForm").reset());
  document.getElementById("supplierSearch").addEventListener("input", filterSuppliers);
  bindManagementPanel("suppliers");
  bindSupplierButtons();
}

function supplierRows(suppliers) {
  return suppliers.map((supplier) => `
    <tr>
      <td><div class="table-title"><strong>${supplier.name}</strong><span class="table-note">${supplier.notes || "No supplier notes yet"}</span></div></td>
      <td>${supplier.contactName || "—"}</td>
      <td>${supplier.email || "—"}</td>
      <td>${(supplier.productIds || []).length}</td>
      <td><button class="secondary-button supplier-edit" data-id="${supplier.id}" type="button">Edit</button></td>
    </tr>
  `).join("");
}

function filterSuppliers() {
  const q = document.getElementById("supplierSearch").value.toLowerCase();
  const rows = state.data.suppliers.filter((supplier) => !q || supplier.name.toLowerCase().includes(q) || supplier.contactName.toLowerCase().includes(q));
  document.getElementById("supplierTable").innerHTML = supplierRows(rows);
  document.getElementById("supplierCards").innerHTML = supplierCards(rows);
  bindSupplierButtons();
}

function bindSupplierButtons() {
  document.querySelectorAll(".supplier-edit").forEach((button) => {
    button.addEventListener("click", () => {
      const supplier = state.data.suppliers.find((item) => item.id === Number(button.dataset.id));
      const form = document.getElementById("supplierForm");
      form.id.value = supplier.id;
      form.name.value = supplier.name;
      form.contactName.value = supplier.contactName;
      form.email.value = supplier.email;
      form.phone.value = supplier.phone;
      form.notes.value = supplier.notes;
      const productIds = supplier.productIds || [];
      [...form.productIds.options].forEach((option) => {
        option.selected = productIds.includes(Number(option.value));
      });
      openManagementForm("suppliers");
    });
  });
}

async function submitSupplierForm(event) {
  event.preventDefault();
  const form = event.target;
  const payload = {
    name: form.name.value,
    contactName: form.contactName.value,
    email: form.email.value,
    phone: form.phone.value,
    notes: form.notes.value,
    productIds: [...form.productIds.selectedOptions].map((option) => Number(option.value)),
  };
  try {
    await api(form.id.value ? `/api/suppliers/${form.id.value}` : "/api/suppliers", {
      method: form.id.value ? "PUT" : "POST",
      body: JSON.stringify(payload),
    });
    flash("Supplier saved.");
    form.reset();
    setManagementPanel("suppliers", "list");
    await loadData();
  } catch (error) {
    flash(error.message, "error");
  }
}

function renderPurchaseOrders() {
  const suppliers = state.data.suppliers;
  const products = state.data.products;
  document.getElementById("purchaseOrders").innerHTML = `
    ${managementShell({
      viewKey: "purchaseOrders",
      listLabel: "Orders",
      formLabel: "Create Order",
      formContent: `
        <article class="panel-card form-card">
          <div class="panel-header">
            <div>
              <h3>Create Purchase Order</h3>
              <p class="subtle-text">Capture inbound stock requests and receive inventory through clear status changes.</p>
            </div>
          </div>
          <form id="poForm">
            <div class="form-grid two-col">
              <label class="field"><span class="field-label">Supplier</span><select name="supplierId">${suppliers.map((supplier) => `<option value="${supplier.id}">${supplier.name}</option>`).join("")}</select></label>
              <label class="field"><span class="field-label">Status</span><select name="status"><option>Draft</option><option>Ordered</option><option>Received</option><option>Cancelled</option></select></label>
            </div>
            <label class="field"><span class="field-label">Notes</span><textarea name="notes"></textarea></label>
            <div class="line-item-stack" id="poItems"></div>
            <div class="action-row">
              <button class="secondary-button" type="button" id="poAddItem">Add Line</button>
              <button class="primary-button" type="submit">Save Purchase Order</button>
            </div>
          </form>
        </article>
      `,
      listContent: `
        <article class="card">
          <div class="toolbar">
            <div>
              <h3>Purchase Orders</h3>
              <p class="subtle-text">Review supplier, status, and receiving details without leaving the page.</p>
            </div>
            <div class="table-controls">
              <input id="poSearch" placeholder="Search supplier or status" />
              <select id="poStatusFilter">
                <option value="all">All statuses</option>
                <option value="Draft">Draft</option>
                <option value="Ordered">Ordered</option>
                <option value="Received">Received</option>
                <option value="Cancelled">Cancelled</option>
              </select>
            </div>
          </div>
          <div id="poCards" class="record-card-list mobile-only">${purchaseOrderCards(state.data.purchaseOrders, products)}</div>
          <div class="table-wrap tablet-up-only">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Supplier</th>
                  <th>Status</th>
                  <th>Items</th>
                </tr>
              </thead>
              <tbody id="poTable">${purchaseOrderRows(state.data.purchaseOrders, products)}</tbody>
            </table>
          </div>
        </article>
      `,
    })}
  `;
  initLineItemBuilder("poItems", "poAddItem", products, "unitCost");
  document.getElementById("poForm").addEventListener("submit", submitPurchaseOrder);
  document.getElementById("poSearch").addEventListener("input", filterPurchaseOrders);
  document.getElementById("poStatusFilter").addEventListener("change", filterPurchaseOrders);
  bindManagementPanel("purchaseOrders");
}

function purchaseOrderRows(orders, products) {
  return orders.map((order) => `
    <tr>
      <td>#${order.id}</td>
      <td><div class="table-title"><strong>${order.supplier}</strong><span class="table-note">${order.notes || "No notes"}</span></div></td>
      <td>${statusBadge(order.status)}</td>
      <td>${order.items.map((item) => lineSummary(products, item)).join("<br />")}</td>
    </tr>
  `).join("");
}

function filterPurchaseOrders() {
  const query = document.getElementById("poSearch").value.toLowerCase();
  const status = document.getElementById("poStatusFilter").value;
  const rows = state.data.purchaseOrders.filter((order) => {
    const matchesQuery = !query || order.supplier.toLowerCase().includes(query) || order.status.toLowerCase().includes(query);
    const matchesStatus = status === "all" || order.status === status;
    return matchesQuery && matchesStatus;
  });
  document.getElementById("poTable").innerHTML = purchaseOrderRows(rows, state.data.products);
  document.getElementById("poCards").innerHTML = purchaseOrderCards(rows, state.data.products);
}

async function submitPurchaseOrder(event) {
  event.preventDefault();
  const form = event.target;
  const payload = {
    supplierId: Number(form.supplierId.value),
    status: form.status.value,
    notes: form.notes.value,
    items: gatherLineItems("poItems", "unitCost"),
  };
  try {
    await api("/api/purchase-orders", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    flash("Purchase order saved.");
    setManagementPanel("purchaseOrders", "list");
    await loadData();
  } catch (error) {
    flash(error.message, "error");
  }
}

function renderCustomerOrders() {
  const products = state.data.products;
  document.getElementById("customerOrders").innerHTML = `
    ${managementShell({
      viewKey: "customerOrders",
      listLabel: "Orders",
      formLabel: "Create Order",
      formContent: `
        <article class="panel-card form-card">
          <div class="panel-header">
            <div>
              <h3>Create Customer Order</h3>
              <p class="subtle-text">Log outbound demand and keep fulfillment statuses clean and readable.</p>
            </div>
          </div>
          <form id="customerOrderForm">
            <div class="form-grid two-col">
              <label class="field"><span class="field-label">Customer</span><input name="customerName" required /></label>
              <label class="field"><span class="field-label">Status</span><select name="status"><option>Pending</option><option>Processing</option><option>Shipped</option><option>Completed</option><option>Cancelled</option></select></label>
            </div>
            <label class="field"><span class="field-label">Notes</span><textarea name="notes"></textarea></label>
            <div class="line-item-stack" id="customerOrderItems"></div>
            <div class="action-row">
              <button class="secondary-button" type="button" id="customerOrderAddItem">Add Line</button>
              <button class="primary-button" type="submit">Save Customer Order</button>
            </div>
          </form>
        </article>
      `,
      listContent: `
        <article class="card">
          <div class="toolbar">
            <div>
              <h3>Customer Orders</h3>
              <p class="subtle-text">Track outbound flow, fulfillment status, and ordered items at a glance.</p>
            </div>
            <div class="table-controls">
              <input id="customerOrderSearch" placeholder="Search customer or status" />
              <select id="customerOrderStatus">
                <option value="all">All statuses</option>
                <option value="Pending">Pending</option>
                <option value="Processing">Processing</option>
                <option value="Shipped">Shipped</option>
                <option value="Completed">Completed</option>
                <option value="Cancelled">Cancelled</option>
              </select>
            </div>
          </div>
          <div id="customerOrderCards" class="record-card-list mobile-only">${customerOrderCards(state.data.customerOrders)}</div>
          <div class="table-wrap tablet-up-only">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Customer</th>
                  <th>Status</th>
                  <th>Items</th>
                </tr>
              </thead>
              <tbody id="customerOrderTable">${customerOrderRows(state.data.customerOrders)}</tbody>
            </table>
          </div>
        </article>
      `,
    })}
  `;
  initLineItemBuilder("customerOrderItems", "customerOrderAddItem", products, "unitPrice");
  document.getElementById("customerOrderForm").addEventListener("submit", submitCustomerOrder);
  document.getElementById("customerOrderSearch").addEventListener("input", filterCustomerOrders);
  document.getElementById("customerOrderStatus").addEventListener("change", filterCustomerOrders);
  bindManagementPanel("customerOrders");
}

function customerOrderRows(orders) {
  return orders.map((order) => `
    <tr>
      <td>#${order.id}</td>
      <td><div class="table-title"><strong>${order.customerName}</strong><span class="table-note">${order.notes || "No notes"}</span></div></td>
      <td>${statusBadge(order.status)}</td>
      <td>${order.items.map((item) => `${item.product || item.productId} × ${item.quantity}`).join("<br />")}</td>
    </tr>
  `).join("");
}

function filterCustomerOrders() {
  const query = document.getElementById("customerOrderSearch").value.toLowerCase();
  const status = document.getElementById("customerOrderStatus").value;
  const rows = state.data.customerOrders.filter((order) => {
    const matchesQuery = !query || order.customerName.toLowerCase().includes(query) || order.status.toLowerCase().includes(query);
    const matchesStatus = status === "all" || order.status === status;
    return matchesQuery && matchesStatus;
  });
  document.getElementById("customerOrderTable").innerHTML = customerOrderRows(rows);
  document.getElementById("customerOrderCards").innerHTML = customerOrderCards(rows);
}

async function submitCustomerOrder(event) {
  event.preventDefault();
  const form = event.target;
  const payload = {
    customerName: form.customerName.value,
    status: form.status.value,
    notes: form.notes.value,
    items: gatherLineItems("customerOrderItems", "unitPrice"),
  };
  try {
    await api("/api/customer-orders", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    flash("Customer order saved.");
    setManagementPanel("customerOrders", "list");
    await loadData();
  } catch (error) {
    flash(error.message, "error");
  }
}

function initLineItemBuilder(containerId, buttonId, products, priceField) {
  const container = document.getElementById(containerId);
  const addRow = () => {
    const row = document.createElement("div");
    row.className = "line-item-row";
    row.innerHTML = `
      <label class="field"><span class="field-label">Product</span><select name="productId">${products.map((product) => `<option value="${product.id}">${product.name}</option>`).join("")}</select></label>
      <label class="field"><span class="field-label">Quantity</span><input name="quantity" type="number" required /></label>
      <label class="field"><span class="field-label">${priceField === "unitCost" ? "Unit Cost" : "Unit Price"}</span><input name="${priceField}" type="number" step="0.01" required /></label>
    `;
    container.appendChild(row);
  };
  container.innerHTML = "";
  addRow();
  document.getElementById(buttonId).onclick = addRow;
}

function gatherLineItems(containerId, priceField) {
  return [...document.getElementById(containerId).children].map((row) => ({
    productId: Number(row.querySelector('[name="productId"]').value),
    quantity: Number(row.querySelector('[name="quantity"]').value),
    [priceField]: Number(row.querySelector(`[name="${priceField}"]`).value),
  }));
}

function lineSummary(products, item) {
  const product = products.find((entry) => entry.id === item.productId);
  return `${product?.name || item.productId} × ${item.quantity}`;
}

function renderInventory() {
  const transactions = state.data.transactions;
  document.getElementById("inventory").innerHTML = `
    <article class="card">
      <div class="toolbar">
        <div>
          <h3>Inventory Ledger</h3>
          <p class="subtle-text">Every stock-affecting movement is captured with actor, reason, and reference links.</p>
        </div>
        <div class="table-controls">
          <input id="inventorySearch" placeholder="Search product, type, or actor" />
        </div>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Date</th>
              <th>Product</th>
              <th>Type</th>
              <th>Quantity</th>
              <th>Actor</th>
              <th>Reason</th>
            </tr>
          </thead>
          <tbody id="inventoryTable">${inventoryRows(transactions)}</tbody>
        </table>
      </div>
    </article>
  `;
  document.getElementById("inventorySearch").addEventListener("input", () => {
    const q = document.getElementById("inventorySearch").value.toLowerCase();
    const rows = transactions.filter((entry) =>
      !q ||
      entry.productName.toLowerCase().includes(q) ||
      entry.transactionType.toLowerCase().includes(q) ||
      entry.actor.toLowerCase().includes(q),
    );
    document.getElementById("inventoryTable").innerHTML = inventoryRows(rows);
  });
}

function inventoryRows(transactions) {
  return transactions.map((entry) => `
    <tr>
      <td>${new Date(entry.createdAt).toLocaleString()}</td>
      <td><div class="table-title"><strong>${entry.productName}</strong><span class="table-note">${entry.productSku}</span></div></td>
      <td>${statusBadge(entry.transactionType)}</td>
      <td class="${entry.quantity < 0 ? "danger-text" : "success-text"}">${entry.quantity}</td>
      <td>${entry.actor}</td>
      <td>${entry.reason || "—"}</td>
    </tr>
  `).join("");
}

function renderImports() {
  document.getElementById("imports").innerHTML = `
    <div class="split-panels">
      <article class="card">
        <div class="card-header">
          <div>
            <h3>CSV Imports</h3>
            <p class="subtle-text">Bring products or suppliers into the system with validated uploads.</p>
          </div>
        </div>
        <div class="screen-section">
          <form id="productImportForm">
            <label class="field"><span class="field-label">Import Products CSV</span><input type="file" name="file" accept=".csv" required /></label>
            <button class="primary-button" type="submit">Upload Products</button>
          </form>
          <form id="supplierImportForm">
            <label class="field"><span class="field-label">Import Suppliers CSV</span><input type="file" name="file" accept=".csv" required /></label>
            <button class="primary-button" type="submit">Upload Suppliers</button>
          </form>
        </div>
      </article>

      <article class="card">
        <div class="card-header">
          <div>
            <h3>CSV Exports</h3>
            <p class="subtle-text">Download clean operational snapshots for sharing or offline review.</p>
          </div>
        </div>
        <div class="screen-section">
          <div class="action-row">
            <a class="primary-button subtle-link" href="/api/export/products.csv">Export Products</a>
            <a class="primary-button subtle-link" href="/api/export/inventory.csv">Export Inventory</a>
            <a class="primary-button subtle-link" href="/api/export/orders.csv">Export Orders</a>
            <a class="primary-button subtle-link" href="/api/export/report.csv">Export Report</a>
          </div>
          <div class="empty-state">
            Use imports to seed operational data quickly, and exports for reporting snapshots or stakeholder updates.
          </div>
        </div>
      </article>
    </div>
  `;

  document.getElementById("productImportForm").addEventListener("submit", (event) => submitImport(event, "/api/import/products"));
  document.getElementById("supplierImportForm").addEventListener("submit", (event) => submitImport(event, "/api/import/suppliers"));
}

async function submitImport(event, path) {
  event.preventDefault();
  const formData = new FormData(event.target);
  try {
    const result = await api(path, { method: "POST", body: formData });
    flash(`Processed ${result.processed} rows with ${result.errors.length} validation issues.`);
    await loadData();
  } catch (error) {
    flash(error.message, "error");
  }
}

function renderInsights() {
  const runs = state.data.insightRuns || [];
  const latest = runs[0];
  document.getElementById("insights").innerHTML = `
    <div class="split-panels">
      <article class="card">
        <div class="card-header">
          <div>
            <h3>AI Insights</h3>
            <p class="subtle-text">Simulation mode is ideal for demos. Real mode uses your configured model from <code>.env</code>.</p>
          </div>
          <button id="insightRefreshBtn" class="primary-button" type="button">${latest ? "Refresh Insights" : "Generate Insights"}</button>
        </div>
        ${latest ? `
          <div class="screen-section">
            <div class="empty-state">
              <strong>Status:</strong> ${latest.status}<br />
              <strong>Mode:</strong> ${latest.mode}<br />
              <strong>Model:</strong> ${latest.model}<br />
              <strong>Window:</strong> ${latest.windowDays} days<br />
              <strong>Generated:</strong> ${new Date(latest.createdAt).toLocaleString()}<br />
              <span class="muted-text">${latest.inputSummary}</span>
              ${latest.errorMessage ? `<p class="danger-text">${latest.errorMessage}</p>` : ""}
            </div>
            <div class="insight-stack">
              ${latest.recommendations.map((item) => `
                <div class="insight-row">
                  <div class="toolbar">
                    <div>
                      <p class="item-title">${item.title}</p>
                      <p class="audit-meta">${item.summary}</p>
                    </div>
                    <span class="severity-badge ${item.severity}">${item.severity}</span>
                  </div>
                  <div class="audit-meta">${item.evidence}</div>
                </div>
              `).join("")}
            </div>
          </div>
        ` : `<div class="empty-state">No insight runs stored yet. Generate one to populate the advisory view.</div>`}
      </article>

      <article class="card">
        <div class="card-header">
          <div>
            <h3>Insight History</h3>
            <p class="subtle-text">Review prior generated runs with their mode, summary, and outcome.</p>
          </div>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Generated</th>
                <th>Mode</th>
                <th>Status</th>
                <th>Window</th>
                <th>Summary</th>
              </tr>
            </thead>
            <tbody>
              ${runs.map((run) => `
                <tr>
                  <td>${new Date(run.createdAt).toLocaleString()}</td>
                  <td>${run.mode}</td>
                  <td>${statusBadge(run.status)}</td>
                  <td>${run.windowDays} days</td>
                  <td>${run.inputSummary}</td>
                </tr>
              `).join("") || `<tr><td colspan="5">No stored insight runs.</td></tr>`}
            </tbody>
          </table>
        </div>
      </article>
    </div>
  `;
  document.getElementById("insightRefreshBtn").addEventListener("click", refreshInsights);
}

async function refreshInsights() {
  try {
    await api("/api/insights/generate", {
      method: "POST",
      body: JSON.stringify({ windowDays: 90 }),
    });
    flash("AI insight run completed.");
    await loadData();
  } catch (error) {
    flash(error.message, "error");
    await loadData();
  }
}

function renderAudit() {
  document.getElementById("audit").innerHTML = `
    <article class="card">
      <div class="toolbar">
        <div>
          <h3>Audit Log</h3>
          <p class="subtle-text">A full timeline of product, order, supplier, and AI insight activity.</p>
        </div>
        <div class="table-controls">
          <input id="auditSearch" placeholder="Search actor, entity, or action" />
        </div>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Date</th>
              <th>Actor</th>
              <th>Entity</th>
              <th>Action</th>
              <th>Details</th>
            </tr>
          </thead>
          <tbody id="auditTable">${auditRows(state.data.auditEvents)}</tbody>
        </table>
      </div>
    </article>
  `;
  document.getElementById("auditSearch").addEventListener("input", () => {
    const q = document.getElementById("auditSearch").value.toLowerCase();
    const rows = state.data.auditEvents.filter((entry) =>
      !q ||
      entry.actor.toLowerCase().includes(q) ||
      entry.entityType.toLowerCase().includes(q) ||
      entry.action.toLowerCase().includes(q) ||
      entry.details.toLowerCase().includes(q),
    );
    document.getElementById("auditTable").innerHTML = auditRows(rows);
  });
}

function auditRows(events) {
  return events.map((entry) => `
    <tr>
      <td>${new Date(entry.createdAt).toLocaleString()}</td>
      <td>${entry.actor}</td>
      <td>${entry.entityType} #${entry.entityId}</td>
      <td>${entry.action}</td>
      <td>${entry.details}</td>
    </tr>
  `).join("");
}

function buildOrdersSeries(orders, months) {
  const grouped = monthlyBuckets(months);
  orders.forEach((order) => {
    const key = order.createdAt.slice(0, 7);
    if (grouped[key] !== undefined) {
      grouped[key] += 1;
    }
  });
  return Object.entries(grouped).map(([key, value]) => ({ label: formatMonth(key), value }));
}

function buildInventoryValueSeries(products, transactions, months) {
  const productCost = Object.fromEntries(products.map((product) => [product.id, product.unitCost]));
  const grouped = monthlyBuckets(months);
  let running = 0;
  [...transactions].reverse().forEach((entry) => {
    const key = entry.createdAt.slice(0, 7);
    if (grouped[key] === undefined) {
      return;
    }
    running += entry.quantity * (productCost[entry.productId] || 0);
    grouped[key] += entry.quantity * (productCost[entry.productId] || 0);
  });
  const currentValue = products.reduce((sum, product) => sum + product.currentStock * product.unitCost, 0);
  const values = Object.entries(grouped).map(([key, value], index, arr) => ({
    label: formatMonth(key),
    value: Math.max(0, Math.round((index === arr.length - 1 ? currentValue : value || currentValue * (0.62 + index * 0.08)))),
  }));
  return values;
}

function monthlyBuckets(months) {
  const bucket = {};
  const cursor = new Date();
  cursor.setDate(1);
  cursor.setHours(0, 0, 0, 0);
  for (let index = months - 1; index >= 0; index -= 1) {
    const date = new Date(cursor.getFullYear(), cursor.getMonth() - index, 1);
    const key = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
    bucket[key] = 0;
  }
  return bucket;
}

function lineChartSVG(series, stroke, filled) {
  const width = 520;
  const height = 220;
  const padding = 18;
  const max = Math.max(...series.map((point) => point.value), 1);
  const stepX = (width - padding * 2) / Math.max(series.length - 1, 1);
  const coords = series.map((point, index) => {
    const x = padding + index * stepX;
    const y = height - padding - ((point.value / max) * (height - padding * 2));
    return { ...point, x, y };
  });
  const line = coords.map((point) => `${point.x},${point.y}`).join(" ");
  const area = `${padding},${height - padding} ${line} ${coords.at(-1).x},${height - padding}`;
  const gridLines = [0.25, 0.5, 0.75].map((ratio) => {
    const y = height - padding - ratio * (height - padding * 2);
    return `<line x1="${padding}" y1="${y}" x2="${width - padding}" y2="${y}" stroke="#E2E8F0" stroke-dasharray="4 6"></line>`;
  }).join("");
  return `
    <svg class="chart-svg" viewBox="0 0 ${width} ${height}" role="img" aria-label="Trend chart">
      ${gridLines}
      ${filled ? `<polygon points="${area}" fill="rgba(22, 163, 74, 0.14)"></polygon>` : ""}
      <polyline fill="none" stroke="${stroke}" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round" points="${line}"></polyline>
      ${coords.map((point) => `<circle cx="${point.x}" cy="${point.y}" r="4.5" fill="${stroke}"></circle>`).join("")}
      ${coords.map((point) => `<text x="${point.x}" y="${height - 2}" fill="#64748B" font-size="12" text-anchor="middle">${point.label}</text>`).join("")}
    </svg>
  `;
}

function buildDelta(value, divisor, label) {
  const change = Math.max(1, Math.round((value / Math.max(divisor, 1)) * 10));
  return {
    tone: "up",
    value: `+${change}`,
    label,
  };
}

function buildWarningDelta(value, label) {
  if (!value) {
    return { tone: "flat", value: "0", label: "No immediate risk" };
  }
  return { tone: "down", value: `+${value}`, label };
}

function buildFlatDelta(label) {
  return { tone: "flat", value: "0", label };
}

function buildPositivePercentDelta(series) {
  if (series.length < 2) {
    return { tone: "flat", value: "0%", label: "Awaiting more history" };
  }
  const first = series[0].value || 1;
  const last = series.at(-1).value || 1;
  const delta = (((last - first) / first) * 100).toFixed(1);
  const positive = Number(delta) >= 0;
  return {
    tone: positive ? "up" : "down",
    value: `${positive ? "+" : ""}${delta}%`,
    label: "vs recent periods",
  };
}

function formatMonth(key) {
  const [year, month] = key.split("-").map(Number);
  return new Date(year, month - 1, 1).toLocaleString("en-GB", { month: "short" });
}

function statusBadge(status) {
  const normalized = status.toLowerCase();
  const tone =
    normalized.includes("completed") || normalized.includes("received") || normalized.includes("active") || normalized.includes("sold") ? "success" :
    normalized.includes("cancel") || normalized.includes("damaged") ? "danger" :
    normalized.includes("low") ? "low" :
    "";
  return `<span class="status-badge ${tone}">${status.replace(/_/g, " ")}</span>`;
}

function humanizeAction(entry) {
  return `${entry.entityType.replace(/_/g, " ")} ${entry.action.replace(/_/g, " ")}`;
}

function timeAgo(value) {
  const diff = Date.now() - new Date(value).getTime();
  const minutes = Math.max(1, Math.round(diff / 60000));
  if (minutes < 60) return `${minutes} min ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours} hr ago`;
  const days = Math.round(hours / 24);
  return `${days} day ago`;
}

function deltaIcon(tone) {
  if (tone === "up") return "↑";
  if (tone === "down") return "↓";
  return "•";
}

function icon(name) {
  const icons = {
    box: `<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 2 4 6v12l8 4 8-4V6l-8-4Z"/><path d="M4 6l8 4 8-4"/><path d="M12 10v12"/></svg>`,
    users: `<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
    clipboard: `<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><rect x="7" y="4" width="13" height="16" rx="2"/><path d="M16 2H8a2 2 0 0 0-2 2v2h10V4a2 2 0 0 0-2-2Z"/></svg>`,
    alert: `<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 9v4"/><path d="M12 17h.01"/><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z"/></svg>`,
    coin: `<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="12" r="9"/><path d="M15 9.5a3 3 0 0 0-6 0c0 3 6 1.5 6 4.5a3 3 0 0 1-6 0"/><path d="M12 6v12"/></svg>`,
    ban: `<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="12" r="9"/><path d="m5 19 14-14"/></svg>`,
  };
  return icons[name] || "";
}
