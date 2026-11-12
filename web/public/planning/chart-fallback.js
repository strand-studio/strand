"use strict";
(() => {
  /**
   * Lightweight Frontier chart shim used when TradingView charting_library
   * is not vendored. Implements the WorkspaceChart surface so planning UI
   * keeps working (plans, chat, approvals) without a hard crash.
   */
  class WorkspaceChart {
    constructor({ venue, symbol, resolution = "60", onStatus, onPrice, onContext, onError, onAvailability }) {
      this.venue = venue || "hyperliquid";
      this.symbol = symbol || "BTC";
      this.resolution = resolution;
      this.onStatus = onStatus || (() => {});
      this.onPrice = onPrice || (() => {});
      this.onContext = onContext || (() => {});
      this.onError = onError || (() => {});
      this.onAvailability = onAvailability || (() => {});
      this.disposed = false;
      this.ready = Promise.resolve();
      this.mount();
      this.onAvailability(true);
      this.onContext(this.context());
      this.refreshPrice().catch(() => {});
    }

    mount() {
      const el = document.getElementById("chart-container");
      if (!el) return;
      el.innerHTML = "";
      const card = document.createElement("div");
      card.style.cssText =
        "height:100%;min-height:320px;display:flex;flex-direction:column;justify-content:center;align-items:center;gap:10px;padding:24px;text-align:center;background:linear-gradient(180deg,#f7f9fc,#eef2f8);color:#192434;font-family:IBM Plex Sans,Segoe UI,sans-serif;";
      card.innerHTML = `
        <div style="font-size:13px;letter-spacing:.08em;text-transform:uppercase;color:#637087">Frontier chart</div>
        <div style="font-size:28px;font-weight:650;letter-spacing:-.03em" data-fb-title>${this.venue.toUpperCase()} · ${this.symbol}</div>
        <div style="font-size:42px;font-weight:600;font-variant-numeric:tabular-nums" data-fb-price>—</div>
        <p style="max-width:420px;margin:0;color:#637087;line-height:1.5;font-size:14px">
          Full TradingView charting library is not installed in this build.
          Market planning, chat and order approval still work. To enable the
          interactive chart, vendor <code>charting_library</code> under
          <code>web/public/</code> and rebuild.
        </p>`;
      el.appendChild(card);
      this.root = card;
    }

    context() {
      return {
        venue: this.venue,
        symbol: this.symbol,
        interval: this.resolution === "60" ? "1h" : this.resolution,
        resolution: this.resolution,
      };
    }

    async whenReady() {
      await this.ready;
    }

    async setContext(venue, symbol, resolution) {
      this.venue = venue || this.venue;
      this.symbol = symbol || this.symbol;
      if (resolution) this.resolution = resolution;
      const title = this.root?.querySelector("[data-fb-title]");
      if (title) title.textContent = `${this.venue.toUpperCase()} · ${this.symbol}`;
      this.onContext(this.context());
      await this.refreshPrice();
    }

    async refreshPrice() {
      try {
        const qs = new URLSearchParams({
          exchange: this.venue,
          symbol: this.symbol,
          interval: "1h",
          limit: "2",
        });
        const res = await fetch("/api/klines?" + qs.toString(), {
          credentials: "same-origin",
          headers: {
            Authorization: "Bearer " + (localStorage.getItem("auth_token") || ""),
          },
        });
        if (!res.ok) return;
        const rows = await res.json();
        if (!Array.isArray(rows) || !rows.length) return;
        const last = rows[rows.length - 1];
        const close = String(last.close ?? last.Close ?? "");
        if (!close) return;
        const priceEl = this.root?.querySelector("[data-fb-price]");
        if (priceEl) priceEl.textContent = Number(close).toLocaleString();
        this.onPrice({ close });
        this.onStatus("Live price · lightweight chart mode", false);
      } catch {
        /* ignore */
      }
    }

    async plot() {}
    async focusPlan() {}
    async draw() {}
    dispose() {
      this.disposed = true;
      if (this.root?.parentElement) this.root.parentElement.innerHTML = "";
      this.root = null;
    }
  }

  window.WorkspaceChart = WorkspaceChart;
})();
