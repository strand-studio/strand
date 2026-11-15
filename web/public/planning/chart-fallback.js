"use strict";
(() => {
  /**
   * Interactive Frontier chart used when TradingView charting_library is not
   * vendored. Built on lightweight-charts + /api/planning/candles so market
   * planning, plan overlays, chat and order approval all keep working.
   */
  const resolutions = {
    1: "1m",
    5: "5m",
    15: "15m",
    60: "1h",
    240: "4h",
    "1D": "1d",
  };
  const resolutionSeconds = {
    1: 60,
    5: 300,
    15: 900,
    60: 3600,
    240: 14400,
    "1D": 86400,
  };

  function cssVar(name, fallback) {
    try {
      const value = getComputedStyle(document.documentElement)
        .getPropertyValue(name)
        .trim();
      return value || fallback;
    } catch {
      return fallback;
    }
  }

  function surfaceColors() {
    const dark =
      document.documentElement.getAttribute("data-theme") === "dark";
    return {
      background: cssVar("--panel-bg", dark ? "#171d27" : "#ffffff"),
      text: cssVar("--text-primary", dark ? "#e8edf5" : "#192434"),
      muted: cssVar("--text-secondary", dark ? "#93a0b5" : "#637087"),
      grid: cssVar("--panel-border", dark ? "rgba(232,237,245,0.12)" : "rgba(25,36,52,0.08)"),
      border: cssVar("--panel-border", dark ? "rgba(232,237,245,0.12)" : "rgba(25,36,52,0.14)"),
      up: cssVar("--binance-green", "#2E8B57"),
      down: cssVar("--binance-red", "#D6433A"),
    };
  }

  function loadLightweightCharts() {
    if (window.LightweightCharts?.createChart) {
      return Promise.resolve(window.LightweightCharts);
    }
    return new Promise((resolve, reject) => {
      const existing = document.querySelector(
        'script[data-strand-lwc="1"]'
      );
      if (existing) {
        existing.addEventListener("load", () =>
          window.LightweightCharts
            ? resolve(window.LightweightCharts)
            : reject(new Error("lightweight-charts failed to load"))
        );
        existing.addEventListener("error", () =>
          reject(new Error("lightweight-charts failed to load"))
        );
        return;
      }
      const script = document.createElement("script");
      script.src = "/vendor/lightweight-charts.standalone.production.js";
      script.async = true;
      script.dataset.strandLwc = "1";
      script.onload = () =>
        window.LightweightCharts
          ? resolve(window.LightweightCharts)
          : reject(new Error("lightweight-charts global missing"));
      script.onerror = () =>
        reject(new Error("Could not load /vendor/lightweight-charts.standalone.production.js"));
      document.head.append(script);
    });
  }

  class WorkspaceChart {
    constructor({
      venue,
      symbol,
      resolution = "60",
      onStatus,
      onPrice,
      onContext,
      onError,
      onAvailability,
      onAnnotations,
    }) {
      this.venue = venue || "hyperliquid";
      this.symbol = symbol || "BTC";
      this.resolution = resolutions[resolution] ? resolution : "60";
      this.onStatus = onStatus || (() => {});
      this.onPrice = onPrice || (() => {});
      this.onContext = onContext || (() => {});
      this.onError = onError || (() => {});
      this.onAvailability = onAvailability || (() => {});
      this.onAnnotations = onAnnotations || (() => {});
      this.disposed = false;
      this.plan = null;
      this.planPhase = "draft";
      this.comparisonPlans = [];
      this.planKey = "";
      this.plotVersion = 0;
      this.candles = [];
      this.priceLines = [];
      this.aiLines = [];
      this.chart = null;
      this.series = null;
      this.LWC = null;
      this._themeObserver = null;
      this._resizeObserver = null;
      this._pollTimer = null;

      this.ready = this.boot();
    }

    async boot() {
      try {
        this.mountShell();
        this.LWC = await loadLightweightCharts();
        if (this.disposed) return;
        this.createChart();
        this.watchTheme();
        await this.reloadCandles();
        if (this.disposed) return;
        this.onAvailability(true);
        this.onContext(this.context());
        this.onStatus("Live · Strand chart", false);
        this._pollTimer = setInterval(() => {
          if (!this.disposed) this.reloadCandles({ quiet: true }).catch(() => {});
        }, 30000);
      } catch (e) {
        this.onAvailability(false, e.message);
        this.onError(e.message || "Chart failed to start");
        throw e;
      }
    }

    mountShell() {
      const el = document.getElementById("chart-container");
      if (!el) throw new Error("chart-container not found");
      el.innerHTML = "";
      el.style.position = "relative";

      const toolbar = document.createElement("div");
      toolbar.className = "strand-fallback-toolbar";
      toolbar.style.cssText =
        "display:flex;gap:6px;align-items:center;padding:8px 10px;border-bottom:1px solid var(--panel-border);background:var(--panel-bg);flex-wrap:wrap;";
      const intervals = [
        ["1", "1m"],
        ["5", "5m"],
        ["15", "15m"],
        ["60", "1H"],
        ["240", "4H"],
        ["1D", "1D"],
      ];
      for (const [id, label] of intervals) {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.textContent = label;
        btn.dataset.resolution = id;
        btn.style.cssText =
          "padding:4px 8px;font-size:11px;border-radius:6px;border:1px solid var(--panel-border);background:transparent;color:var(--text-secondary);cursor:pointer;";
        if (id === this.resolution) {
          btn.style.background = "var(--brand-soft)";
          btn.style.color = "var(--strand-gold)";
          btn.style.borderColor = "var(--strand-gold)";
        }
        btn.addEventListener("click", () => {
          if (this.resolution === id) return;
          this.resolution = id;
          for (const b of toolbar.querySelectorAll("button[data-resolution]")) {
            const active = b.dataset.resolution === id;
            b.style.background = active ? "var(--brand-soft)" : "transparent";
            b.style.color = active ? "var(--strand-gold)" : "var(--text-secondary)";
            b.style.borderColor = active ? "var(--strand-gold)" : "var(--panel-border)";
          }
          this.onContext(this.context());
          this.reloadCandles().then(() => this.redrawPlan()).catch((e) => this.onError(e.message));
        });
        toolbar.appendChild(btn);
      }

      const meta = document.createElement("div");
      meta.dataset.fbMeta = "1";
      meta.style.cssText =
        "margin-left:auto;font-size:11px;color:var(--text-secondary);font-variant-numeric:tabular-nums;";
      meta.textContent = `${this.venue.toUpperCase()} · ${this.symbol}`;
      toolbar.appendChild(meta);

      const host = document.createElement("div");
      host.dataset.fbHost = "1";
      host.style.cssText = "flex:1;min-height:0;height:calc(100% - 38px);width:100%;";

      const wrap = document.createElement("div");
      wrap.style.cssText = "height:100%;display:flex;flex-direction:column;background:var(--panel-bg);";
      wrap.appendChild(toolbar);
      wrap.appendChild(host);
      el.appendChild(wrap);
      this.root = wrap;
      this.host = host;
      this.metaEl = meta;
    }

    createChart() {
      if (!this.host || !this.LWC) return;
      const colors = surfaceColors();
      this.chart = this.LWC.createChart(this.host, {
        autoSize: true,
        layout: {
          background: { color: colors.background },
          textColor: colors.text,
          fontSize: 12,
        },
        grid: {
          vertLines: { color: colors.grid },
          horzLines: { color: colors.grid },
        },
        rightPriceScale: { borderColor: colors.border },
        timeScale: {
          borderColor: colors.border,
          timeVisible: true,
          secondsVisible: false,
        },
        crosshair: { mode: 1 },
      });

      const candleOpts = {
        upColor: colors.up,
        downColor: colors.down,
        borderUpColor: colors.up,
        borderDownColor: colors.down,
        wickUpColor: colors.up,
        wickDownColor: colors.down,
      };
      if (typeof this.LWC.CandlestickSeries !== "undefined" && this.chart.addSeries) {
        this.series = this.chart.addSeries(this.LWC.CandlestickSeries, candleOpts);
      } else {
        this.series = this.chart.addCandlestickSeries(candleOpts);
      }

      this._resizeObserver = new ResizeObserver(() => {
        if (!this.chart || !this.host) return;
        const { clientWidth: width, clientHeight: height } = this.host;
        if (width > 0 && height > 0) this.chart.applyOptions({ width, height });
      });
      this._resizeObserver.observe(this.host);
    }

    watchTheme() {
      this._themeObserver = new MutationObserver(() => {
        if (!this.chart || this.disposed) return;
        const colors = surfaceColors();
        this.chart.applyOptions({
          layout: {
            background: { color: colors.background },
            textColor: colors.text,
          },
          grid: {
            vertLines: { color: colors.grid },
            horzLines: { color: colors.grid },
          },
          rightPriceScale: { borderColor: colors.border },
          timeScale: { borderColor: colors.border },
        });
        this.series?.applyOptions({
          upColor: colors.up,
          downColor: colors.down,
          borderUpColor: colors.up,
          borderDownColor: colors.down,
          wickUpColor: colors.up,
          wickDownColor: colors.down,
        });
      });
      this._themeObserver.observe(document.documentElement, {
        attributes: true,
        attributeFilter: ["data-theme", "class"],
      });
    }

    authHeaders() {
      const token = localStorage.getItem("auth_token") || "";
      const headers = {};
      if (token) headers.Authorization = "Bearer " + token;
      return headers;
    }

    async fetchCandles(limit = 300) {
      const qs = new URLSearchParams({
        venue: this.venue,
        symbol: this.symbol,
        interval: resolutions[this.resolution] || "1h",
        limit: String(limit),
      });
      const res = await fetch("/api/planning/candles?" + qs.toString(), {
        credentials: "same-origin",
        headers: this.authHeaders(),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data.error || "Market data request failed");
      const rows = Array.isArray(data.candles) ? data.candles : Array.isArray(data) ? data : [];
      return rows
        .map((c) => {
          let time = Number(c.time ?? c.Time ?? 0);
          if (time > 1e12) time = Math.floor(time / 1000);
          return {
            time,
            open: Number(c.open ?? c.Open),
            high: Number(c.high ?? c.High),
            low: Number(c.low ?? c.Low),
            close: Number(c.close ?? c.Close),
            volume: Number(c.volume ?? c.Volume ?? 0),
          };
        })
        .filter((c) => Number.isFinite(c.time) && c.time > 0 && Number.isFinite(c.close))
        .sort((a, b) => a.time - b.time);
    }

    async reloadCandles({ quiet = false } = {}) {
      // Do not await this.ready here — boot() calls reloadCandles while ready
      // is still pending, which would deadlock.
      if (this.disposed || !this.series) return;
      const rows = await this.fetchCandles();
      if (this.disposed) return;
      this.candles = rows;
      this.series.setData(
        rows.map((c) => ({
          time: c.time,
          open: c.open,
          high: c.high,
          low: c.low,
          close: c.close,
        }))
      );
      if (rows.length) {
        const last = rows[rows.length - 1];
        this.onPrice({ close: String(last.close) }, { candles: rows });
        if (this.metaEl) {
          this.metaEl.textContent = `${this.venue.toUpperCase()} · ${this.symbol} · ${Number(
            last.close
          ).toLocaleString()}`;
        }
        if (!quiet) this.chart?.timeScale().fitContent();
      } else if (!quiet) {
        this.onStatus("No candles for this market", true);
      }
    }

    context() {
      let visible_from;
      let visible_to;
      try {
        const range = this.chart?.timeScale()?.getVisibleRange?.();
        if (range && range.to > range.from) {
          visible_from = Math.floor(range.from);
          visible_to = Math.floor(range.to);
        }
      } catch {
        /* ignore */
      }
      return {
        venue: this.venue,
        symbol: this.symbol,
        interval: resolutions[this.resolution] || "1h",
        resolution: this.resolution,
        ...(visible_from && visible_to ? { visible_from, visible_to } : {}),
      };
    }

    async whenReady() {
      await this.ready;
      if (this.disposed || !this.chart) {
        throw new Error("Chart is not ready yet. Try again.");
      }
    }

    async setContext(venue, symbol, resolution) {
      this.venue = venue || this.venue;
      this.symbol = symbol || this.symbol;
      if (resolution && resolutions[resolution]) this.resolution = resolution;
      if (this.metaEl) {
        this.metaEl.textContent = `${this.venue.toUpperCase()} · ${this.symbol}`;
      }
      await this.whenReady();
      if (this.disposed) return;
      this.onContext(this.context());
      this.clearPriceLines();
      this.planKey = "";
      await this.reloadCandles();
      if (this.plan) await this.redrawPlan();
    }

    clearPriceLines() {
      if (!this.series) {
        this.priceLines = [];
        return;
      }
      for (const line of this.priceLines) {
        try {
          this.series.removePriceLine(line);
        } catch {
          /* ignore */
        }
      }
      this.priceLines = [];
    }

    addPlanLine(price, title, color) {
      if (!this.series || !Number.isFinite(price) || price <= 0) return;
      const line = this.series.createPriceLine({
        price,
        color,
        lineWidth: 2,
        lineStyle: 2,
        axisLabelVisible: true,
        title,
      });
      this.priceLines.push(line);
    }

    async redrawPlan() {
      if (!this.plan) return;
      this.planKey = "";
      await this.plot(this.plan, this.planPhase, this.comparisonPlans);
    }

    async plot(plan, phase = "draft", comparisons = []) {
      const comparisonPlans = (comparisons || [])
        .filter(
          (p) =>
            p?.snapshot?.kind === "perpetual" &&
            p.snapshot.venue === this.venue &&
            p.snapshot.symbol === this.symbol
        )
        .slice(0, 4);
      const key =
        (plan ? plan.hash + phase : "") +
        comparisonPlans.map((p) => p.hash).join(":");
      this.comparisonPlans = comparisonPlans;
      if (key === this.planKey && (plan ? this.plan?.hash === plan.hash : !this.plan)) {
        return;
      }
      this.planKey = key;
      this.plan = plan;
      this.planPhase = phase;
      const version = ++this.plotVersion;
      try {
        await this.whenReady();
      } catch {
        return;
      }
      if (this.disposed || version !== this.plotVersion) return;
      this.clearPriceLines();
      if (
        !plan ||
        (plan.snapshot?.kind && plan.snapshot.kind !== "perpetual") ||
        plan.snapshot?.venue !== this.venue ||
        plan.snapshot?.symbol !== this.symbol
      ) {
        return;
      }

      const entries =
        plan.idea?.execution?.kind === "ladder" && Array.isArray(plan.orders)
          ? plan.orders.map((o, i) => [
              `Limit ${o.index ?? i + 1}`,
              Number(o.price),
              "#9b6a14",
            ])
          : [
              [
                plan.idea?.execution?.kind === "market"
                  ? "Est. market entry"
                  : "Entry",
                Number(plan.idea?.entry),
                "#9b6a14",
              ],
            ];

      for (const [label, price, color] of entries) {
        this.addPlanLine(price, label, color);
      }
      this.addPlanLine(Number(plan.idea?.stop), "Stop", "#D6433A");
      this.addPlanLine(Number(plan.idea?.target), "Target", "#2E8B57");

      for (let i = 0; i < comparisonPlans.length; i++) {
        const p = comparisonPlans[i];
        if (p.id === plan.id) continue;
        this.addPlanLine(
          Number(p.idea?.entry),
          `Alt ${String.fromCharCode(65 + i)}`,
          "#3575a8"
        );
      }
    }

    async focusPlan() {
      await this.whenReady();
      if (!this.plan || !this.candles.length || !this.chart) return;
      const seconds = resolutionSeconds[this.resolution] || 3600;
      const last = this.candles[this.candles.length - 1].time;
      const from = last - 64 * seconds;
      try {
        this.chart.timeScale().setVisibleRange({ from, to: last + 4 * seconds });
      } catch {
        this.chart.timeScale().fitContent();
      }
      const prices = [
        Number(this.plan.idea?.entry),
        Number(this.plan.idea?.stop),
        Number(this.plan.idea?.target),
        ...(this.plan.orders || []).map((o) => Number(o.price)),
        ...this.candles.slice(-80).flatMap((c) => [c.high, c.low]),
      ].filter((n) => Number.isFinite(n) && n > 0);
      if (prices.length >= 2) {
        const lo = Math.min(...prices);
        const hi = Math.max(...prices);
        const pad = (hi - lo) * 0.15 || hi * 0.01;
        try {
          this.chart.priceScale("right").setVisibleRange({
            from: lo - pad,
            to: hi + pad,
          });
        } catch {
          /* auto scale is fine */
        }
      }
    }

    clearAILines() {
      if (!this.series) {
        this.aiLines = [];
        return;
      }
      for (const line of this.aiLines) {
        try {
          this.series.removePriceLine(line);
        } catch {
          /* ignore */
        }
      }
      this.aiLines = [];
    }

    async draw(drawings = [], source = "manual") {
      await this.whenReady();
      if (this.disposed || !this.series) return;
      this.clearAILines();
      const rows = Array.isArray(drawings) ? drawings.slice(0, 8) : [];
      for (const d of rows) {
        if (d?.kind && d.kind !== "horizontal_line") continue;
        const price = Number(d?.points?.[0]?.price);
        if (!Number.isFinite(price) || price <= 0) continue;
        const label = String(d.label || "AI");
        const support = /support|\bS[1-9]\b/i.test(label);
        const resistance = /resistance|\bR[1-9]\b/i.test(label);
        const color = support
          ? "#2E8B57"
          : resistance
            ? "#D6433A"
            : source === "ai"
              ? "#3575a8"
              : "#9b6a14";
        try {
          const line = this.series.createPriceLine({
            price,
            color,
            lineWidth: 1,
            lineStyle: 2,
            axisLabelVisible: true,
            title: label.slice(0, 18),
          });
          this.aiLines.push(line);
        } catch {
          /* ignore overlay failures */
        }
      }
      try {
        this.onAnnotations({
          total: rows.length,
          visible: this.aiLines.length,
          expanded: false,
          shown: this.aiLines.length > 0,
          comparing: false,
          items: rows,
        });
      } catch {
        /* optional callback */
      }
    }

    dispose() {
      this.disposed = true;
      if (this._pollTimer) clearInterval(this._pollTimer);
      if (this._themeObserver) this._themeObserver.disconnect();
      if (this._resizeObserver) this._resizeObserver.disconnect();
      this.clearPriceLines();
      this.clearAILines();
      try {
        this.chart?.remove();
      } catch {
        /* ignore */
      }
      this.chart = null;
      this.series = null;
      const el = document.getElementById("chart-container");
      if (el) el.innerHTML = "";
      this.root = null;
      this.host = null;
    }
  }

  window.WorkspaceChart = WorkspaceChart;
})();
