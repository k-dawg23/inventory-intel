const state = {
  actor: "Kenneth",
  data: null,
};

const currency = new Intl.NumberFormat("en-GB", { style: "currency", currency: "GBP" });

document.addEventListener("DOMContentLoaded", () => {
  wireNavigation();
  document.getElementById("actorInput").addEventListener("input", (event) => {
    state.actor = event.target.value || "Kenneth";
  });
  document.getElementById("refreshBtn").addEventListener("click", loadData);
  loadData();
});

function wireNavigation() {
  document.querySelectorAll(".nav-link").forEach((button) => {
    button.addEventListener("click", () => {
      document.querySelectorAll(".nav-link").forEach((link) => link.classList.remove("active"));
      document.querySelectorAll(".view").forEach((view) => view.classList.remove("active"));
      button.classList.add("active");
      document.getElementById(button.dataset.view).classList.add("active");
    });
  });
}

async function loadData() {
  const response = await fetch("/api/bootstrap");
  state.data = await response.json();
  renderAll();
}

function renderAll() {
  renderDashboard();
  renderProducts();
  renderSuppliers();
  renderPurchaseOrders();
  renderCustomerOrders();
  renderInventory();
  renderImports();
  renderAudit();
}

function flash(message, type = "info") {
  const node = document.getElementById("flash");
  node.textContent = message;
  node.classList.remove("hidden");
  node.style.background = type === "error" ? "#fbe1dd" : "#fff1d6";
  node.style.borderColor = type === "error" ? "#e2aea4" : "#f2d39d";
  setTimeout(() => node.classList.add("hidden"), 3600);
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: {
      "Content-Type": options.body instanceof FormData ? undefined : "application/json",
      "X-Actor": state.actor,
      ...(options.headers || {}),
    },
  });
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
  const data = state.data.dashboard;
  document.getElementById("dashboard").innerHTML = `
    <div class="grid kpi-grid">
      ${kpiCard("Products", data.products)}
      ${kpiCard("Suppliers", data.suppliers)}
      ${kpiCard("Orders", data.orders)}
      ${kpiCard("Low Stock", data.lowStockItems)}
      ${kpiCard("Inventory Value", currency.format(data.inventoryValue))}
    </div>
    <div class="grid" style="grid-template-columns: 1.2fr 1fr; margin-top: 16px;">
      <article class="card">
        <h2>Low Stock Alerts</h2>
        <ul class="mini-list">
          ${data.lowStockProducts.map((product) => `<li class="mini-row"><span>${product.name} <span class="muted">(${product.sku})</span></span><strong>${product.currentStock}/${product.reorderLevel}</strong></li>`).join("") || "<li>No low stock items.</li>"}
        </ul>
      </article>
      <article class="card">
        <h2>Recent Audit</h2>
        <ul class="mini-list">
          ${data.recentAudit.map((entry) => `<li class="mini-row"><span>${entry.actor} ${entry.action} ${entry.entityType} #${entry.entityId}</span><span class="muted">${new Date(entry.createdAt).toLocaleString()}</span></li>`).join("")}
        </ul>
      </article>
    </div>
    <div class="grid" style="grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); margin-top: 16px;">
      ${listCard("Orders Per Month", data.ordersPerMonth)}
      ${listCard("Stock Movements", data.stockMovements)}
      <article class="card">
        <h2>Top Selling Products</h2>
        <ul class="chart-list">
          ${data.topSelling.map((item) => `<li class="chart-row"><span>${item.name} <span class="muted">(${item.sku})</span></span><strong>${item.quantity}</strong></li>`).join("")}
        </ul>
      </article>
    </div>`;
}

function kpiCard(label, value) {
  return `<article class="card"><p class="eyebrow">${label}</p><p class="kpi">${value}</p></article>`;
}

function listCard(title, items) {
  return `<article class="card"><h2>${title}</h2><ul class="chart-list">${items.map((item) => `<li class="chart-row"><span>${item.label}</span><strong>${item.value}</strong></li>`).join("")}</ul></article>`;
}

function renderProducts() {
  const products = state.data.products;
  document.getElementById("products").innerHTML = `
    <div class="grid" style="grid-template-columns: 1fr 1.3fr;">
      <article class="card">
        <h2>Add or Update Product</h2>
        <form id="productForm">
          <input type="hidden" name="id" />
          <div class="two-col">
            <label>SKU<input name="sku" required /></label>
            <label>Name<input name="name" required /></label>
            <label>Category<input name="category" required /></label>
            <label>Unit Cost<input name="unitCost" type="number" step="0.01" required /></label>
            <label>Selling Price<input name="sellingPrice" type="number" step="0.01" required /></label>
            <label>Current Stock<input name="currentStock" type="number" required /></label>
            <label>Reorder Level<input name="reorderLevel" type="number" required /></label>
            <label>Status<select name="active"><option value="true">Active</option><option value="false">Inactive</option></select></label>
          </div>
          <label>Description<textarea name="description"></textarea></label>
          <div class="actions">
            <button class="primary" type="submit">Save Product</button>
            <button class="secondary" type="button" id="productReset">Reset</button>
          </div>
        </form>
      </article>
      <article class="card">
        <div class="toolbar">
          <h2>Catalog</h2>
          <input id="productSearch" placeholder="Search SKU or name" />
        </div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>SKU</th><th>Name</th><th>Stock</th><th>Price</th><th>Status</th><th></th></tr></thead>
            <tbody id="productTable">${productRows(products)}</tbody>
          </table>
        </div>
      </article>
    </div>
    <article class="card" style="margin-top: 16px;">
      <h2>Manual Inventory Adjustment</h2>
      <form id="adjustmentForm" class="two-col">
        <label>Product<select name="productId">${productOptions(products)}</select></label>
        <label>Type<select name="type"><option value="adjusted">Manual Correction</option><option value="damaged">Damaged</option><option value="returned">Returned</option></select></label>
        <label>Quantity<input name="quantity" type="number" required /></label>
        <label>Reason<input name="reason" required /></label>
        <button class="primary" type="submit">Apply Adjustment</button>
      </form>
    </article>`;

  document.getElementById("productSearch").addEventListener("input", (event) => {
    const q = event.target.value.toLowerCase();
    const filtered = products.filter((product) => product.sku.toLowerCase().includes(q) || product.name.toLowerCase().includes(q));
    document.getElementById("productTable").innerHTML = productRows(filtered);
    bindProductEditButtons();
  });
  document.getElementById("productForm").addEventListener("submit", submitProductForm);
  document.getElementById("productReset").addEventListener("click", () => document.getElementById("productForm").reset());
  document.getElementById("adjustmentForm").addEventListener("submit", submitAdjustmentForm);
  bindProductEditButtons();
}

function productRows(products) {
  return products.map((product) => `
    <tr>
      <td>${product.sku}</td>
      <td>${product.name}</td>
      <td>${product.currentStock} ${product.lowStock ? '<span class="status-pill low">Low</span>' : ""}</td>
      <td>${currency.format(product.sellingPrice)}</td>
      <td>${product.active ? "Active" : "Inactive"}</td>
      <td><button class="secondary product-edit" data-id="${product.id}">Edit</button></td>
    </tr>`).join("");
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
      }).forEach(([key, value]) => form.elements[key].value = value);
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
  const id = form.id.value;
  try {
    await api(id ? `/api/products/${id}` : "/api/products", {
      method: id ? "PUT" : "POST",
      body: JSON.stringify(payload),
    });
    flash("Product saved.");
    form.reset();
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
    <div class="grid" style="grid-template-columns: 1fr 1.2fr;">
      <article class="card">
        <h2>Add or Update Supplier</h2>
        <form id="supplierForm">
          <input type="hidden" name="id" />
          <div class="two-col">
            <label>Name<input name="name" required /></label>
            <label>Contact<input name="contactName" /></label>
            <label>Email<input name="email" type="email" /></label>
            <label>Phone<input name="phone" /></label>
          </div>
          <label>Notes<textarea name="notes"></textarea></label>
          <label>Linked Products
            <select name="productIds" multiple size="6">${products.map((product) => `<option value="${product.id}">${product.name}</option>`).join("")}</select>
          </label>
          <div class="actions">
            <button class="primary" type="submit">Save Supplier</button>
            <button class="secondary" type="button" id="supplierReset">Reset</button>
          </div>
        </form>
      </article>
      <article class="card">
        <h2>Suppliers</h2>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Name</th><th>Contact</th><th>Linked Products</th><th></th></tr></thead>
            <tbody>${suppliers.map((supplier) => `<tr><td>${supplier.name}</td><td>${supplier.contactName || "—"}</td><td>${supplier.productIds.length}</td><td><button class="secondary supplier-edit" data-id="${supplier.id}">Edit</button></td></tr>`).join("")}</tbody>
          </table>
        </div>
      </article>
    </div>`;
  document.getElementById("supplierForm").addEventListener("submit", submitSupplierForm);
  document.getElementById("supplierReset").addEventListener("click", () => document.getElementById("supplierForm").reset());
  document.querySelectorAll(".supplier-edit").forEach((button) => button.addEventListener("click", () => populateSupplier(button.dataset.id)));
}

function populateSupplier(id) {
  const supplier = state.data.suppliers.find((item) => item.id === Number(id));
  const form = document.getElementById("supplierForm");
  form.id.value = supplier.id;
  form.name.value = supplier.name;
  form.contactName.value = supplier.contactName;
  form.email.value = supplier.email;
  form.phone.value = supplier.phone;
  form.notes.value = supplier.notes;
  [...form.productIds.options].forEach((option) => option.selected = supplier.productIds.includes(Number(option.value)));
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
    await loadData();
  } catch (error) {
    flash(error.message, "error");
  }
}

function renderPurchaseOrders() {
  const suppliers = state.data.suppliers;
  const products = state.data.products;
  document.getElementById("purchaseOrders").innerHTML = `
    <div class="grid" style="grid-template-columns: 1fr 1.2fr;">
      <article class="card">
        <h2>Create Purchase Order</h2>
        <form id="poForm">
          <label>Supplier<select name="supplierId">${suppliers.map((supplier) => `<option value="${supplier.id}">${supplier.name}</option>`)}</select></label>
          <label>Status<select name="status"><option>Draft</option><option>Ordered</option><option>Received</option><option>Cancelled</option></select></label>
          <label>Notes<textarea name="notes"></textarea></label>
          <div id="poItems"></div>
          <button class="secondary" type="button" id="poAddItem">Add Line</button>
          <button class="primary" type="submit">Save Purchase Order</button>
        </form>
      </article>
      <article class="card">
        <h2>Purchase Orders</h2>
        <div class="table-wrap">
          <table>
            <thead><tr><th>ID</th><th>Supplier</th><th>Status</th><th>Items</th></tr></thead>
            <tbody>${state.data.purchaseOrders.map((order) => `<tr><td>#${order.id}</td><td>${order.supplier}</td><td><span class="status-pill">${order.status}</span></td><td>${order.items.map((item) => lineSummary(products, item)).join("<br />")}</td></tr>`).join("")}</tbody>
          </table>
        </div>
      </article>
    </div>`;
  initLineItemBuilder("poItems", "poAddItem", products, ["quantity", "unitCost"]);
  document.getElementById("poForm").addEventListener("submit", submitPurchaseOrder);
}

async function submitPurchaseOrder(event) {
  event.preventDefault();
  const form = event.target;
  const payload = {
    supplierId: Number(form.supplierId.value),
    status: form.status.value,
    notes: form.notes.value,
    items: gatherLineItems("poItems", true),
  };
  try {
    await api("/api/purchase-orders", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    flash("Purchase order saved.");
    await loadData();
  } catch (error) {
    flash(error.message, "error");
  }
}

function renderCustomerOrders() {
  const products = state.data.products;
  document.getElementById("customerOrders").innerHTML = `
    <div class="grid" style="grid-template-columns: 1fr 1.2fr;">
      <article class="card">
        <h2>Create Customer Order</h2>
        <form id="customerOrderForm">
          <label>Customer<input name="customerName" required /></label>
          <label>Status<select name="status"><option>Pending</option><option>Processing</option><option>Shipped</option><option>Completed</option><option>Cancelled</option></select></label>
          <label>Notes<textarea name="notes"></textarea></label>
          <div id="customerOrderItems"></div>
          <button class="secondary" type="button" id="customerOrderAddItem">Add Line</button>
          <button class="primary" type="submit">Save Customer Order</button>
        </form>
      </article>
      <article class="card">
        <h2>Customer Orders</h2>
        <div class="table-wrap">
          <table>
            <thead><tr><th>ID</th><th>Customer</th><th>Status</th><th>Items</th></tr></thead>
            <tbody>${state.data.customerOrders.map((order) => `<tr><td>#${order.id}</td><td>${order.customerName}</td><td><span class="status-pill">${order.status}</span></td><td>${order.items.map((item) => `${item.product || item.productId} x ${item.quantity}`).join("<br />")}</td></tr>`).join("")}</tbody>
          </table>
        </div>
      </article>
    </div>`;
  initLineItemBuilder("customerOrderItems", "customerOrderAddItem", products, ["quantity", "unitPrice"]);
  document.getElementById("customerOrderForm").addEventListener("submit", submitCustomerOrder);
}

async function submitCustomerOrder(event) {
  event.preventDefault();
  const form = event.target;
  const payload = {
    customerName: form.customerName.value,
    status: form.status.value,
    notes: form.notes.value,
    items: gatherLineItems("customerOrderItems", false),
  };
  try {
    await api("/api/customer-orders", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    flash("Customer order saved.");
    await loadData();
  } catch (error) {
    flash(error.message, "error");
  }
}

function initLineItemBuilder(containerId, buttonId, products, extraFields) {
  const container = document.getElementById(containerId);
  const addRow = () => {
    const row = document.createElement("div");
    row.className = "two-col";
    row.innerHTML = `
      <label>Product<select name="productId">${products.map((product) => `<option value="${product.id}">${product.name}</option>`)}</select></label>
      ${extraFields.map((field) => `<label>${field}<input name="${field}" type="number" step="${field.toLowerCase().includes("cost") || field.toLowerCase().includes("price") ? "0.01" : "1"}" required /></label>`).join("")}
    `;
    container.appendChild(row);
  };
  addRow();
  document.getElementById(buttonId).onclick = addRow;
}

function gatherLineItems(containerId, purchaseOrder) {
  return [...document.getElementById(containerId).children].map((row) => ({
    productId: Number(row.querySelector('[name="productId"]').value),
    quantity: Number(row.querySelector('[name="quantity"]').value),
    [purchaseOrder ? "unitCost" : "unitPrice"]: Number(row.querySelector(`[name="${purchaseOrder ? "unitCost" : "unitPrice"}"]`).value),
  }));
}

function lineSummary(products, item) {
  const product = products.find((entry) => entry.id === item.productId);
  return `${product?.name || item.productId} x ${item.quantity}`;
}

function renderInventory() {
  document.getElementById("inventory").innerHTML = `
    <article class="card">
      <h2>Inventory Ledger</h2>
      <div class="table-wrap">
        <table>
          <thead><tr><th>Date</th><th>Product</th><th>Type</th><th>Quantity</th><th>Actor</th><th>Reason</th></tr></thead>
          <tbody>${state.data.transactions.map((entry) => `<tr><td>${new Date(entry.createdAt).toLocaleString()}</td><td>${entry.productName}</td><td>${entry.transactionType}</td><td class="${entry.quantity < 0 ? "danger" : ""}">${entry.quantity}</td><td>${entry.actor}</td><td>${entry.reason}</td></tr>`).join("")}</tbody>
        </table>
      </div>
    </article>`;
}

function renderImports() {
  document.getElementById("imports").innerHTML = `
    <div class="grid" style="grid-template-columns: 1fr 1fr;">
      <article class="card">
        <h2>CSV Imports</h2>
        <form id="productImportForm">
          <label>Import Products CSV<input type="file" name="file" accept=".csv" required /></label>
          <button class="primary" type="submit">Upload Products</button>
        </form>
        <form id="supplierImportForm" style="margin-top: 18px;">
          <label>Import Suppliers CSV<input type="file" name="file" accept=".csv" required /></label>
          <button class="primary" type="submit">Upload Suppliers</button>
        </form>
      </article>
      <article class="card">
        <h2>CSV Exports</h2>
        <div class="actions">
          <a class="primary" href="/api/export/products.csv">Products</a>
          <a class="primary" href="/api/export/inventory.csv">Inventory</a>
          <a class="primary" href="/api/export/orders.csv">Orders</a>
          <a class="primary" href="/api/export/report.csv">Report</a>
        </div>
        <p class="muted">Use imports for onboarding and exports for reporting snapshots.</p>
      </article>
    </div>`;
  document.getElementById("productImportForm").addEventListener("submit", (event) => submitImport(event, "/api/import/products"));
  document.getElementById("supplierImportForm").addEventListener("submit", (event) => submitImport(event, "/api/import/suppliers"));
}

async function submitImport(event, path) {
  event.preventDefault();
  const formData = new FormData(event.target);
  try {
    const result = await api(path, {
      method: "POST",
      body: formData,
      headers: { "X-Actor": state.actor },
    });
    flash(`Processed ${result.processed} rows. Errors: ${result.errors.length}`);
    await loadData();
  } catch (error) {
    flash(error.message, "error");
  }
}

function renderAudit() {
  document.getElementById("audit").innerHTML = `
    <article class="card">
      <h2>Audit Log</h2>
      <div class="table-wrap">
        <table>
          <thead><tr><th>Date</th><th>Actor</th><th>Entity</th><th>Action</th><th>Details</th></tr></thead>
          <tbody>${state.data.auditEvents.map((entry) => `<tr><td>${new Date(entry.createdAt).toLocaleString()}</td><td>${entry.actor}</td><td>${entry.entityType} #${entry.entityId}</td><td>${entry.action}</td><td>${entry.details}</td></tr>`).join("")}</tbody>
        </table>
      </div>
    </article>`;
}
