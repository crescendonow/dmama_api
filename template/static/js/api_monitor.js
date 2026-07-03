(function () {
  "use strict";

  var STORAGE_KEY = "dmama_api_monitor_key";
  var THEME_STORAGE_KEY = "theme";
  var API_URL = "/api/monitor/usage";

  var state = {
    data: emptyData(),
    tables: {
      latest: { search: "", sortKey: "started_at", direction: "desc" },
      endpoints: { search: "", sortKey: "calls", direction: "desc" },
      apiKeys: { search: "", sortKey: "calls", direction: "desc" }
    }
  };

  var els = {};

  document.addEventListener("DOMContentLoaded", function () {
    cacheElements();
    initTheme();
    restoreKey();
    setDefaultCustomRange();
    bindEvents();
    toggleCustomFields();
    loadUsage();
  });

  function cacheElements() {
    els.monitorKey = document.getElementById("monitor-key");
    els.saveKey = document.getElementById("save-key");
    els.clearKey = document.getElementById("clear-key");
    els.refreshTop = document.getElementById("refresh-top");
    els.themeToggle = document.getElementById("theme-toggle");
    els.iconSun = document.getElementById("icon-sun");
    els.iconMoon = document.getElementById("icon-moon");
    els.filterForm = document.getElementById("filter-form");
    els.resetFilters = document.getElementById("reset-filters");
    els.preset = document.getElementById("preset");
    els.fromTime = document.getElementById("from-time");
    els.toTime = document.getElementById("to-time");
    els.customFields = Array.prototype.slice.call(document.querySelectorAll(".custom-field"));
    els.message = document.getElementById("message");
    els.loadState = document.getElementById("load-state");
    els.kpiCalls = document.getElementById("kpi-calls");
    els.kpiWindow = document.getElementById("kpi-window");
    els.kpiErrorRate = document.getElementById("kpi-error-rate");
    els.kpiErrors = document.getElementById("kpi-errors");
    els.kpiLatency = document.getElementById("kpi-latency");
    els.kpiVolume = document.getElementById("kpi-volume");
    els.statusCaption = document.getElementById("status-caption");
    els.statusBars = document.getElementById("status-bars");
    els.latestBody = document.getElementById("latest-body");
    els.latestCount = document.getElementById("latest-count");
    els.endpointBody = document.getElementById("endpoint-body");
    els.endpointCount = document.getElementById("endpoint-count");
    els.apiKeyBody = document.getElementById("api-key-body");
    els.apiKeyCount = document.getElementById("api-key-count");
  }

  function bindEvents() {
    els.saveKey.addEventListener("click", function () {
      writeSessionKey(els.monitorKey.value.trim());
      showMessage("Saved X-API-Key for this browser tab.", "ok");
    });

    els.clearKey.addEventListener("click", function () {
      els.monitorKey.value = "";
      writeSessionKey("");
      showMessage("Cleared X-API-Key.", "ok");
    });

    els.refreshTop.addEventListener("click", loadUsage);

    if (els.themeToggle) {
      els.themeToggle.addEventListener("click", function () {
        setTheme(!document.documentElement.classList.contains("dark"));
      });
    }

    els.preset.addEventListener("change", function () {
      toggleCustomFields();
    });

    els.filterForm.addEventListener("submit", function (event) {
      event.preventDefault();
      loadUsage();
    });

    els.resetFilters.addEventListener("click", function () {
      els.filterForm.reset();
      setDefaultCustomRange();
      toggleCustomFields();
      loadUsage();
    });

    document.querySelectorAll("[data-table-search]").forEach(function (input) {
      input.addEventListener("input", function () {
        state.tables[input.dataset.tableSearch].search = input.value.trim().toLowerCase();
        renderDashboard();
      });
    });

    document.querySelectorAll("[data-table][data-sort]").forEach(function (button) {
      button.addEventListener("click", function () {
        var table = state.tables[button.dataset.table];
        if (!table) return;
        if (table.sortKey === button.dataset.sort) {
          table.direction = table.direction === "asc" ? "desc" : "asc";
        } else {
          table.sortKey = button.dataset.sort;
          table.direction = "asc";
        }
        renderDashboard();
      });
    });
  }

  function initTheme() {
    var saved = readStoredTheme();
    var prefersDark = false;
    try {
      prefersDark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
    } catch (_err) {
      prefersDark = false;
    }
    setTheme(saved === "dark" || (!saved && prefersDark), false);
  }

  function setTheme(dark, persist) {
    document.documentElement.classList.toggle("dark", dark);
    if (els.iconSun) els.iconSun.classList.toggle("hidden", !dark);
    if (els.iconMoon) els.iconMoon.classList.toggle("hidden", dark);
    if (els.themeToggle) els.themeToggle.setAttribute("aria-pressed", dark ? "true" : "false");
    if (persist !== false) writeStoredTheme(dark ? "dark" : "light");
  }

  function readStoredTheme() {
    try {
      return localStorage.getItem(THEME_STORAGE_KEY);
    } catch (_err) {
      return null;
    }
  }

  function writeStoredTheme(value) {
    try {
      localStorage.setItem(THEME_STORAGE_KEY, value);
    } catch (_err) {
      // Some hardened browser contexts disable localStorage.
    }
  }

  function restoreKey() {
    try {
      els.monitorKey.value = sessionStorage.getItem(STORAGE_KEY) || "";
    } catch (_err) {
      els.monitorKey.value = "";
    }
  }

  function writeSessionKey(value) {
    try {
      if (value) {
        sessionStorage.setItem(STORAGE_KEY, value);
      } else {
        sessionStorage.removeItem(STORAGE_KEY);
      }
    } catch (_err) {
      // Some hardened browser contexts disable sessionStorage.
    }
  }

  function setDefaultCustomRange() {
    var now = new Date();
    var from = new Date(now.getTime() - 24 * 60 * 60 * 1000);
    els.fromTime.value = toDatetimeLocalValue(from);
    els.toTime.value = toDatetimeLocalValue(now);
  }

  function toggleCustomFields() {
    var show = els.preset.value === "custom";
    els.customFields.forEach(function (field) {
      field.hidden = !show;
    });
  }

  function buildParams() {
    var params = new URLSearchParams();
    var preset = document.getElementById("preset").value || "24h";
    params.set("preset", preset);

    if (preset === "custom") {
      if (els.fromTime.value) params.set("from", localInputToISOString(els.fromTime.value));
      if (els.toTime.value) params.set("to", localInputToISOString(els.toTime.value));
    }

    appendParam(params, "api_key", document.getElementById("filter-api-key").value);
    appendParam(params, "method", document.getElementById("method").value);
    appendParam(params, "path", document.getElementById("path").value);
    appendParam(params, "status", document.getElementById("status").value);
    appendParam(params, "limit", document.getElementById("limit").value || "100");
    return params;
  }

  function appendParam(params, name, value) {
    value = String(value || "").trim();
    if (value) params.set(name, value);
  }

  async function loadUsage() {
    var params;
    try {
      params = buildParams();
    } catch (err) {
      showMessage(err.message || "Invalid filter values.", "error");
      return;
    }

    setLoading(true);
    hideMessage();

    var headers = {};
    var key = els.monitorKey.value.trim();
    if (key) headers["X-API-Key"] = key;

    try {
      var response = await fetch(API_URL + "?" + params.toString(), { headers: headers });
      var payload = await response.json().catch(function () {
        return { success: false, error: "Response is not JSON." };
      });

      if (!response.ok || !payload.success) {
        throw new Error(payload.error || "Request failed with HTTP " + response.status);
      }

      state.data = normalizeData(payload.data);
      renderDashboard();
      els.loadState.textContent = "Loaded " + formatClock(new Date());
    } catch (err) {
      state.data = emptyData();
      renderDashboard();
      showMessage(err.message || "Unable to load monitor data.", "error");
      els.loadState.textContent = "Error";
    } finally {
      setLoading(false);
    }
  }

  function setLoading(loading) {
    els.loadState.textContent = loading ? "Loading" : els.loadState.textContent;
    els.refreshTop.disabled = loading;
    els.filterForm.querySelectorAll("button").forEach(function (button) {
      button.disabled = loading;
    });
  }

  function normalizeData(data) {
    data = data || {};
    return {
      latest: Array.isArray(data.latest) ? data.latest : [],
      endpoints: Array.isArray(data.endpoints) ? data.endpoints : [],
      statuses: Array.isArray(data.statuses) ? data.statuses : [],
      api_keys: Array.isArray(data.api_keys) ? data.api_keys : [],
      filters: data.filters || null
    };
  }

  function emptyData() {
    return { latest: [], endpoints: [], statuses: [], api_keys: [], filters: null };
  }

  function renderDashboard() {
    renderKpis();
    renderStatuses();
    renderLatestTable();
    renderEndpointTable();
    renderAPIKeyTable();
  }

  function renderKpis() {
    var statuses = state.data.statuses;
    var endpoints = state.data.endpoints;
    var apiKeys = state.data.api_keys;
    var totalCalls = sum(statuses, "calls");
    var errors = statuses.reduce(function (acc, row) {
      return acc + (Number(row.status) >= 400 ? Number(row.calls || 0) : 0);
    }, 0);
    var weightedLatency = weightedAverage(endpoints, "avg_ms", "calls");
    var totalBytes = sum(apiKeys, "total_bytes");

    els.kpiCalls.textContent = formatInteger(totalCalls);
    els.kpiWindow.textContent = formatFilterWindow(state.data.filters);
    els.kpiErrorRate.textContent = totalCalls ? formatPercent(errors / totalCalls) : "0%";
    els.kpiErrors.textContent = formatInteger(errors) + " errors";
    els.kpiLatency.textContent = Math.round(weightedLatency) + " ms";
    els.kpiVolume.textContent = formatBytes(totalBytes);
  }

  function renderStatuses() {
    var statuses = state.data.statuses.slice().sort(function (a, b) {
      return Number(a.status) - Number(b.status);
    });
    var total = sum(statuses, "calls");
    var maxCalls = Math.max.apply(null, statuses.map(function (row) { return Number(row.calls || 0); }).concat([1]));

    els.statusCaption.textContent = total ? formatInteger(total) + " calls" : "No data";
    clearNode(els.statusBars);

    if (!statuses.length) {
      els.statusBars.appendChild(emptyBlock("No status data"));
      return;
    }

    statuses.forEach(function (row) {
      var calls = Number(row.calls || 0);
      var percentOfMax = Math.max(2, Math.round((calls / maxCalls) * 100));
      var line = document.createElement("div");
      line.className = "status-row";

      var badge = document.createElement("span");
      badge.className = "badge status-badge " + statusClass(row.status);
      badge.textContent = safeText(row.status);
      line.appendChild(badge);

      var track = document.createElement("div");
      track.className = "status-track";
      var fill = document.createElement("div");
      fill.className = "status-fill " + statusClass(row.status);
      fill.style.width = percentOfMax + "%";
      track.appendChild(fill);
      line.appendChild(track);

      var count = document.createElement("span");
      count.textContent = formatInteger(calls);
      line.appendChild(count);

      els.statusBars.appendChild(line);
    });
  }

  function renderLatestTable() {
    var columns = ["started_at", "method", "path", "status", "duration_ms", "size_value", "api_key", "request_id"];
    var rows = filteredAndSorted("latest", state.data.latest, columns);
    els.latestCount.textContent = rows.length + " rows";
    clearNode(els.latestBody);

    if (!rows.length) {
      appendEmptyRow(els.latestBody, 8, "No requests");
      return;
    }

    rows.forEach(function (row) {
      var tr = document.createElement("tr");
      appendTextCell(tr, formatDateTime(row.started_at), "mono");
      appendBadgeCell(tr, row.method, "method-badge");
      appendTextCell(tr, row.path, "mono path-cell");
      appendBadgeCell(tr, row.status, "status-badge " + statusClass(row.status));
      appendTextCell(tr, formatInteger(row.duration_ms || 0) + " ms");
      appendTextCell(tr, formatSizeValue(row));
      appendTextCell(tr, row.api_key, "mono");
      appendTextCell(tr, row.request_id, "mono");
      els.latestBody.appendChild(tr);
    });
  }

  function renderEndpointTable() {
    var columns = ["path", "method", "status", "calls", "avg_ms"];
    var rows = filteredAndSorted("endpoints", state.data.endpoints, columns);
    els.endpointCount.textContent = rows.length + " rows";
    clearNode(els.endpointBody);

    if (!rows.length) {
      appendEmptyRow(els.endpointBody, 5, "No endpoint data");
      return;
    }

    rows.forEach(function (row) {
      var tr = document.createElement("tr");
      appendTextCell(tr, row.path, "mono path-cell");
      appendBadgeCell(tr, row.method, "method-badge");
      appendBadgeCell(tr, row.status, "status-badge " + statusClass(row.status));
      appendTextCell(tr, formatInteger(row.calls || 0));
      appendTextCell(tr, formatInteger(row.avg_ms || 0) + " ms");
      els.endpointBody.appendChild(tr);
    });
  }

  function renderAPIKeyTable() {
    var columns = ["api_key", "calls", "total_bytes", "avg_ms"];
    var rows = filteredAndSorted("apiKeys", state.data.api_keys, columns);
    els.apiKeyCount.textContent = rows.length + " rows";
    clearNode(els.apiKeyBody);

    if (!rows.length) {
      appendEmptyRow(els.apiKeyBody, 4, "No API key data");
      return;
    }

    rows.forEach(function (row) {
      var tr = document.createElement("tr");
      appendTextCell(tr, row.api_key, "mono");
      appendTextCell(tr, formatInteger(row.calls || 0));
      appendTextCell(tr, formatBytes(row.total_bytes || 0));
      appendTextCell(tr, formatInteger(row.avg_ms || 0) + " ms");
      els.apiKeyBody.appendChild(tr);
    });
  }

  function filteredAndSorted(tableName, rows, columns) {
    var table = state.tables[tableName];
    var search = table.search;
    var output = rows.slice();

    if (search) {
      output = output.filter(function (row) {
        return columns.some(function (key) {
          return tableValue(row, key).toLowerCase().indexOf(search) !== -1;
        });
      });
    }

    output.sort(function (a, b) {
      var result = compareValues(sortValue(a, table.sortKey), sortValue(b, table.sortKey));
      return table.direction === "desc" ? -result : result;
    });
    return output;
  }

  function tableValue(row, key) {
    if (key === "size_value") return formatSizeValue(row);
    if (key === "started_at") return formatDateTime(row.started_at);
    if (key === "total_bytes") return formatBytes(row.total_bytes || 0);
    return safeText(row[key]);
  }

  function sortValue(row, key) {
    if (key === "started_at") return new Date(row.started_at).getTime() || 0;
    if (key === "size_value") return sizeToBytes(row.size_value, row.size_unit);
    if (["status", "duration_ms", "calls", "avg_ms", "total_bytes"].indexOf(key) !== -1) {
      return Number(row[key] || 0);
    }
    return safeText(row[key]).toLowerCase();
  }

  function compareValues(a, b) {
    if (typeof a === "number" && typeof b === "number") return a - b;
    return String(a).localeCompare(String(b));
  }

  function appendTextCell(tr, value, className) {
    var td = document.createElement("td");
    td.textContent = safeText(value);
    if (className) td.className = className;
    tr.appendChild(td);
  }

  function appendBadgeCell(tr, value, className) {
    var td = document.createElement("td");
    var badge = document.createElement("span");
    badge.className = "badge " + className;
    badge.textContent = safeText(value);
    td.appendChild(badge);
    tr.appendChild(td);
  }

  function appendEmptyRow(tbody, colSpan, text) {
    var tr = document.createElement("tr");
    var td = document.createElement("td");
    td.className = "empty-row";
    td.colSpan = colSpan;
    td.textContent = text;
    tr.appendChild(td);
    tbody.appendChild(tr);
  }

  function emptyBlock(text) {
    var div = document.createElement("div");
    div.className = "empty-row";
    div.textContent = text;
    return div;
  }

  function clearNode(node) {
    while (node.firstChild) node.removeChild(node.firstChild);
  }

  function statusClass(status) {
    status = Number(status);
    if (status >= 500) return "status-error";
    if (status >= 400) return "status-warn";
    if (status >= 200 && status < 400) return "status-ok";
    return "";
  }

  function sum(rows, key) {
    return rows.reduce(function (acc, row) {
      return acc + Number(row[key] || 0);
    }, 0);
  }

  function weightedAverage(rows, valueKey, weightKey) {
    var weight = sum(rows, weightKey);
    if (!weight) return 0;
    var total = rows.reduce(function (acc, row) {
      return acc + Number(row[valueKey] || 0) * Number(row[weightKey] || 0);
    }, 0);
    return total / weight;
  }

  function formatInteger(value) {
    return Number(value || 0).toLocaleString("en-US");
  }

  function formatPercent(value) {
    return (value * 100).toFixed(value > 0 && value < 0.1 ? 2 : 1) + "%";
  }

  function formatBytes(bytes) {
    bytes = Number(bytes || 0);
    if (bytes >= 1024 * 1024 * 1024) return (bytes / (1024 * 1024 * 1024)).toFixed(2) + " GB";
    if (bytes >= 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + " MB";
    return (bytes / 1024).toFixed(2) + " KB";
  }

  function formatSizeValue(row) {
    if (row.size_value === undefined || row.size_value === null) return "-";
    return Number(row.size_value || 0).toLocaleString("en-US", { maximumFractionDigits: 2 }) + " " + safeText(row.size_unit);
  }

  function sizeToBytes(value, unit) {
    var n = Number(value || 0);
    unit = String(unit || "").toUpperCase();
    if (unit === "GB") return n * 1024 * 1024 * 1024;
    if (unit === "MB") return n * 1024 * 1024;
    return n * 1024;
  }

  function formatDateTime(value) {
    if (!value) return "-";
    var date = new Date(value);
    if (Number.isNaN(date.getTime())) return "-";
    return date.toLocaleString("en-US", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit"
    });
  }

  function formatClock(date) {
    return date.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit", second: "2-digit" });
  }

  function formatFilterWindow(filters) {
    if (!filters) return "No range";
    if (filters.preset && filters.preset !== "custom") return filters.preset;
    return formatDateTime(filters.from) + " - " + formatDateTime(filters.to);
  }

  function toDatetimeLocalValue(date) {
    function pad(value) {
      return String(value).padStart(2, "0");
    }
    return [
      date.getFullYear(),
      pad(date.getMonth() + 1),
      pad(date.getDate())
    ].join("-") + "T" + [pad(date.getHours()), pad(date.getMinutes())].join(":");
  }

  function localInputToISOString(value) {
    var date = new Date(value);
    if (Number.isNaN(date.getTime())) throw new Error("Custom date range is invalid.");
    return date.toISOString();
  }

  function showMessage(text, type) {
    els.message.textContent = text;
    els.message.className = "message" + (type === "ok" ? " ok" : "");
    els.message.hidden = false;
  }

  function hideMessage() {
    els.message.hidden = true;
    els.message.textContent = "";
  }

  function safeText(value) {
    if (value === null || value === undefined || value === "") return "-";
    return String(value);
  }
})();
