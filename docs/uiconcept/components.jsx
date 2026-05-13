/* Shared components — icons, shell, helpers. */

const { useState, useEffect, useRef, useMemo, Fragment } = React;

// ───────── Icons (line, 14×14) ─────────
const I = {
  dot: (
    <svg viewBox="0 0 14 14" fill="none" className="ico"><circle cx="7" cy="7" r="2.5" fill="currentColor"/></svg>
  ),
  overview: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><rect x="2" y="2" width="4" height="4"/><rect x="8" y="2" width="4" height="4"/><rect x="2" y="8" width="4" height="4"/><rect x="8" y="8" width="4" height="4"/></svg>
  ),
  blocks: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><path d="M7 1 L13 4 L7 7 L1 4 Z"/><path d="M1 4 L1 10 L7 13 L13 10 L13 4"/><path d="M7 7 L7 13"/></svg>
  ),
  tx: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><path d="M2 4 H10 M8 2 L10 4 L8 6 M12 10 H4 M6 8 L4 10 L6 12"/></svg>
  ),
  transfer: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><circle cx="7" cy="7" r="5"/><path d="M5 6 L7 4 L9 6 M7 4 L7 10"/></svg>
  ),
  crosschain: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><circle cx="3.5" cy="7" r="2"/><circle cx="10.5" cy="7" r="2"/><path d="M5.5 7 H8.5"/></svg>
  ),
  fx: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><path d="M2 3 L11 3 M9 1 L11 3 L9 5 M12 11 L3 11 M5 9 L3 11 L5 13"/></svg>
  ),
  agents: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><rect x="3" y="3" width="8" height="8" rx="1"/><circle cx="5.5" cy="6" r="0.6" fill="currentColor"/><circle cx="8.5" cy="6" r="0.6" fill="currentColor"/><path d="M5 9 H9"/><path d="M7 1 V3 M7 11 V13 M1 7 H3 M11 7 H13"/></svg>
  ),
  graph: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><circle cx="3" cy="3" r="1.5"/><circle cx="11" cy="3" r="1.5"/><circle cx="7" cy="11" r="1.5"/><circle cx="11" cy="9" r="1.2"/><path d="M4 4 L6 10 M10 4 L8 10 M11 4.5 L11 7.5"/></svg>
  ),
  jobs: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><rect x="2" y="4" width="10" height="8" rx="0.5"/><path d="M5 4 V2.5 Q5 2 5.5 2 H8.5 Q9 2 9 2.5 V4"/></svg>
  ),
  traces: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><path d="M2 2 V12 H12 M4 9 L7 6 L9 8 L12 4"/></svg>
  ),
  search: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><circle cx="6" cy="6" r="4"/><path d="M9 9 L12 12"/></svg>
  ),
  download: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><path d="M7 2 V9 M4 6 L7 9 L10 6 M2 12 H12"/></svg>
  ),
  filter: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><path d="M2 3 H12 L9 7 V11 L6 12 V7 L2 3"/></svg>
  ),
  external: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><path d="M5 3 H3 V11 H11 V9 M8 2 H12 V6 M12 2 L7 7"/></svg>
  ),
  refresh: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><path d="M12 3 V6 H9 M2 11 V8 H5"/><path d="M11 6 A5 5 0 0 0 3 5 M3 8 A5 5 0 0 0 11 9"/></svg>
  ),
  copy: (
    <svg viewBox="0 0 14 14" fill="none" className="ico" stroke="currentColor" strokeWidth="1.4"><rect x="4" y="4" width="8" height="8" rx="1"/><path d="M2 10 V2 H10"/></svg>
  ),
};

// ───────── Logo ─────────
const Logo = () => (
  <div className="logo">
    <div className="logo-mark" />
    <div className="logo-text">ARCADIA<span className="net">·explorer</span></div>
  </div>
);

// ───────── Topbar ─────────
const Topbar = ({ headBlock, tps, blockMs }) => (
  <div className="topbar">
    <div className="search">
      {I.search}
      <input placeholder="Search address, tx, block, agent, quote_id…" />
      <span className="kbd">⌘K</span>
    </div>
    <div className="topbar-meta">
      <span className="pill"><span className="pulse-dot" /> mainnet</span>
      <span className="pill">head <span className="val">#{fmtFull(headBlock)}</span></span>
      <span className="pill">tps <span className="val">{fmtNum(tps)}</span></span>
      <span className="pill">block <span className="val">{blockMs}ms</span></span>
      <span className="pill">indexer <span className="val acc">·</span> 0 lag</span>
    </div>
  </div>
);

// ───────── Sidebar ─────────
const NAV = [
  { group: "Live",
    items: [
      { id: "overview",   label: "Overview",     ico: I.overview },
      { id: "blocks",     label: "Blocks",       ico: I.blocks, count: "live" },
      { id: "txs",        label: "Transactions", ico: I.tx, count: "live" },
      { id: "transfers",  label: "Transfers",    ico: I.transfer },
      { id: "traces",     label: "Traces",       ico: I.traces },
    ]
  },
  { group: "Flows",
    items: [
      { id: "crosschain", label: "Cross-chain",  ico: I.crosschain },
      { id: "fx",         label: "StableFX",     ico: I.fx },
    ]
  },
  { group: "Agents · ERC-8004",
    items: [
      { id: "agents",     label: "Agent registry", ico: I.agents },
      { id: "jobs",       label: "Job market",     ico: I.jobs },
      { id: "graph",      label: "Wallet graph",   ico: I.graph },
    ]
  },
];

const Sidebar = ({ active, onNav }) => (
  <aside className="sidebar">
    {NAV.map(group => (
      <div className="nav-group" key={group.group}>
        <div className="nav-label">{group.group}</div>
        {group.items.map(item => (
          <div
            key={item.id}
            className={"nav-item" + (active === item.id ? " active" : "")}
            onClick={() => onNav(item.id)}
          >
            {item.ico}
            <span>{item.label}</span>
            {item.count && <span className="count">{item.count}</span>}
          </div>
        ))}
      </div>
    ))}
  </aside>
);

// ───────── Status bar ─────────
const StatusBar = () => (
  <div className="statusbar">
    <span className="seg"><span className="dot acc"/> indexer <span className="v ok">live</span></span>
    <span className="seg">rpc <span className="v">arc.rpc.circle.com</span></span>
    <span className="seg">finality <span className="v">single-slot</span></span>
    <span className="seg right">v0.4.2-rc1</span>
    <span className="seg">build 8f3a2c1</span>
    <span className="seg">ws ↔ <span className="v ok">connected</span></span>
  </div>
);

// ───────── Sparkline ─────────
const Sparkline = ({ data, color = "var(--accent)", height = 36, fill = true, strokeWidth = 1 }) => {
  const w = 200;
  const h = height;
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const pts = data.map((v, i) => {
    const x = (i / (data.length - 1)) * w;
    const y = h - ((v - min) / range) * h;
    return [x, y];
  });
  const path = pts.map(([x, y], i) => (i === 0 ? `M${x},${y}` : `L${x},${y}`)).join(" ");
  const fillPath = `${path} L${w},${h} L0,${h} Z`;
  return (
    <svg viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" style={{ width: "100%", height: "100%", display: "block" }}>
      {fill && <path d={fillPath} fill={color} opacity="0.15"/>}
      <path d={path} fill="none" stroke={color} strokeWidth={strokeWidth} vectorEffect="non-scaling-stroke"/>
    </svg>
  );
};

// ───────── Bar chart ─────────
const BarChart = ({ data, color = "var(--accent)", height = 80 }) => {
  const max = Math.max(...data);
  return (
    <div style={{ display: "flex", alignItems: "flex-end", gap: 2, height, padding: "4px 0" }}>
      {data.map((v, i) => (
        <div key={i} style={{
          flex: 1,
          background: color,
          opacity: 0.35 + (v / max) * 0.6,
          height: `${(v / max) * 100}%`,
          minHeight: 1,
        }}/>
      ))}
    </div>
  );
};

// ───────── Address ─────────
const Addr = ({ a, name, accent }) => (
  <span className="addr" title={a}>
    {name ? <span className="acc">{name}</span> : (
      <>
        <span className="addr-prefix">{a?.slice(0, 6)}</span>
        <span style={{ color: "var(--fg-4)" }}>{a?.slice(6, -4)}</span>
        <span className="addr-prefix">{a?.slice(-4)}</span>
      </>
    )}
  </span>
);

const Hash = ({ h }) => (
  <span className="hash mono" title={h}>{shortHash(h)}</span>
);

// ───────── Token chip ─────────
const Token = ({ sym }) => {
  const t = TOKENS.find(t => t.sym === sym) || { color: "muted" };
  return <span className={"badge " + t.color}>{sym}</span>;
};

// ───────── Chain chip ─────────
const ChainChip = ({ domain }) => {
  const c = CHAINS[domain] || { short: "D" + domain };
  return <span className="chain">{c.short}</span>;
};

Object.assign(window, {
  I, Logo, Topbar, Sidebar, StatusBar, Sparkline, BarChart, Addr, Hash, Token, ChainChip,
});
