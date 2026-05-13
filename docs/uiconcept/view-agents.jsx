/* Agents registry, Jobs market, and Wallet graph */

const AgentsView = () => {
  return (
    <div className="view">
      <div className="view-head">
        <div>
          <div className="view-title">Agent registry</div>
          <div className="view-sub">ERC-8004 · 1,287 registered · 412 active in last hour</div>
        </div>
        <div className="view-actions">
          <button className="btn ghost">{I.filter} Filter</button>
          <button className="btn acc">+ Register agent</button>
        </div>
      </div>

      <div className="grid" style={{ gridTemplateColumns: "repeat(4, 1fr)", marginBottom: 12 }}>
        <div className="stat"><div className="label">Total registered</div><div className="value">{fmtNum(1287)}</div><div className="delta up">▲ 14 this week</div></div>
        <div className="stat"><div className="label">Active 24h</div><div className="value">{fmtNum(891)}</div><div className="delta up">▲ 6.2%</div></div>
        <div className="stat"><div className="label">Jobs in-flight</div><div className="value">{fmtNum(312)}</div><div className="delta up">▲ 18</div></div>
        <div className="stat"><div className="label">Avg trust score</div><div className="value">87.4</div><div className="delta up">▲ 0.3</div></div>
      </div>

      <div className="filter-bar">
        <FilterChip on label="status" value="all"/>
        <FilterChip label="status" value="active"/>
        <FilterChip label="status" value="idle"/>
        <FilterChip label="capability" value="market_making"/>
        <FilterChip label="capability" value="settlement"/>
        <FilterChip label="capability" value="compliance"/>
      </div>

      <div className="grid grid-2-eq">
        {AGENTS.map(a => (
          <div className="card" key={a.addr}>
            <div className="card-body" style={{ display: "flex", gap: 14, alignItems: "flex-start" }}>
              <div className="agent-avatar" style={{ width: 44, height: 44, fontSize: 16 }}>{a.name[0].toUpperCase()}</div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="row" style={{ alignItems: "baseline" }}>
                  <div className="agent-name" style={{ fontSize: 14 }}>{a.name}</div>
                  <span className={"badge " + (a.trust > 90 ? "ok" : a.trust > 80 ? "info" : "warn")} style={{ marginLeft: 8 }}>trust {a.trust}</span>
                  <span className="spacer"/>
                  <span className="dim mono" style={{ fontSize: 10 }}>ERC-8004</span>
                </div>
                <div className="agent-sub" style={{ marginTop: 4 }}>{a.domain} · <Addr a={a.addr}/></div>
                <div style={{ marginTop: 10, display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 12 }}>
                  <div><div className="card-sub">JOBS</div><div className="mono fg0" style={{ fontSize: 14, marginTop: 2 }}>{fmtNum(a.jobs)}</div></div>
                  <div><div className="card-sub">SUCCESS</div><div className="mono acc" style={{ fontSize: 14, marginTop: 2 }}>{a.success}%</div></div>
                  <div><div className="card-sub">VOLUME</div><div className="mono fg0" style={{ fontSize: 14, marginTop: 2 }}>${fmtNum(a.jobs * 14210)}</div></div>
                  <div><div className="card-sub">LAST SEEN</div><div className="mono fg0" style={{ fontSize: 14, marginTop: 2 }}>{Math.floor(Math.random() * 5) + 1}m</div></div>
                </div>
                <div style={{ marginTop: 10, display: "flex", gap: 6, flexWrap: "wrap" }}>
                  {pickCapabilities(a).map(c => <span key={c} className="badge muted">{c}</span>)}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

function pickCapabilities(a) {
  const all = ["market_making", "settlement", "rfq_quoting", "compliance", "yield_routing", "oracle_signing", "policy_attestation"];
  // deterministic-ish based on addr last char
  const seed = parseInt(a.addr.slice(-1), 16);
  return [all[seed % all.length], all[(seed * 3 + 1) % all.length], all[(seed * 7 + 5) % all.length]];
}

const JobsView = () => (
  <div className="view">
    <div className="view-head">
      <div>
        <div className="view-title">Agent job market</div>
        <div className="view-sub">on-chain task posting · settlements in USDC</div>
      </div>
      <div className="view-actions">
        <button className="btn ghost">{I.filter} Filter</button>
        <button className="btn acc">+ Post job</button>
      </div>
    </div>

    <div className="grid" style={{ gridTemplateColumns: "repeat(5, 1fr)", marginBottom: 12 }}>
      <div className="stat"><div className="label">Open</div><div className="value">{fmtNum(312)}</div></div>
      <div className="stat"><div className="label">In progress</div><div className="value">{fmtNum(184)}</div></div>
      <div className="stat"><div className="label">Completed 24h</div><div className="value">{fmtNum(2891)}</div><div className="delta up">▲ 12%</div></div>
      <div className="stat"><div className="label">Disputed</div><div className="value">{fmtNum(7)}</div><div className="delta down">▼ 2</div></div>
      <div className="stat"><div className="label">TVL escrow</div><div className="value">${fmtNum(412_910)}</div></div>
    </div>

    <div className="filter-bar">
      <FilterChip on label="status" value="all"/>
      <FilterChip label="status" value="proposed"/>
      <FilterChip label="status" value="active"/>
      <FilterChip label="status" value="completed"/>
      <FilterChip label="status" value="disputed"/>
    </div>

    <div className="card">
      <table className="tbl">
        <thead>
          <tr>
            <th>job id</th>
            <th>title</th>
            <th>employer</th>
            <th>worker</th>
            <th className="num">bounty</th>
            <th>status</th>
            <th>posted block</th>
            <th className="num">age</th>
          </tr>
        </thead>
        <tbody>
          {JOBS.map(j => (
            <tr key={j.job_id}>
              <td className="mono dim">{j.job_id}</td>
              <td className="fg0">{j.title}</td>
              <td><Addr a={j.employer_address} name={j.employer_name.split("/")[0]}/></td>
              <td><Addr a={j.worker_address} name={j.worker_name.split("/")[0]}/></td>
              <td className="num"><span className="fg0">${j.bounty_usdc}</span></td>
              <td>
                <span className={"badge " + (
                  j.status === "completed" ? "ok" :
                  j.status === "active" ? "acc" :
                  j.status === "proposed" ? "info" :
                  j.status === "disputed" ? "err" : "muted"
                )}>{j.status}</span>
              </td>
              <td><span className="acc">#{fmtFull(j.created_at_block)}</span></td>
              <td className="num muted">{ago(j.created_at)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <Pagination from="1" to="40" total={8_912}/>
    </div>
  </div>
);

// ───────── Wallet graph view ─────────
const GraphView = () => {
  const stageRef = useRef(null);
  const [hovered, setHovered] = useState(null);

  // Build node positions: cluster ADDR_POOL + AGENT addrs
  const nodes = useMemo(() => {
    const all = [...new Set([...ADDR_POOL, ...AGENTS.map(a => a.addr)])];
    const cx = 480, cy = 320;
    return all.map((a, i) => {
      const angle = (i / all.length) * Math.PI * 2;
      const radius = 90 + (i % 4) * 60 + (i % 7) * 14;
      const agent = AGENTS.find(x => x.addr === a);
      return {
        addr: a,
        x: cx + Math.cos(angle + i * 0.7) * radius,
        y: cy + Math.sin(angle + i * 0.7) * radius,
        size: 5 + (i % 5) * 1.4 + (agent ? 4 : 0),
        agent,
        cluster: i % 4,
      };
    });
  }, []);

  const links = useMemo(() => {
    return EDGES.slice(0, 60).map(e => {
      const from = nodes.find(n => n.addr === e.from_wallet);
      const to = nodes.find(n => n.addr === e.to_wallet);
      if (!from || !to) return null;
      return { from, to, weight: e.tx_count, volume: parseFloat(e.volume), ...e };
    }).filter(Boolean);
  }, [nodes]);

  const clusterColor = ["var(--accent)", "var(--info)", "var(--magenta)", "var(--warn)"];

  return (
    <div className="view">
      <div className="view-head">
        <div>
          <div className="view-title">Wallet graph</div>
          <div className="view-sub">force-directed · 412k nodes · 1.8M edges · projection: top 64 by tx_count</div>
        </div>
        <div className="view-actions">
          <button className="btn ghost">{I.filter} Cluster</button>
          <button className="btn ghost">{I.download} Export</button>
          <button className="btn acc">View in 3D →</button>
        </div>
      </div>

      <div className="graph-stage" ref={stageRef}>
        <svg viewBox="0 0 960 640" preserveAspectRatio="xMidYMid meet" style={{ width: "100%", height: "100%" }}>
          <defs>
            <radialGradient id="nodeGlow" cx="50%" cy="50%" r="50%">
              <stop offset="0%" stopColor="oklch(0.86 0.17 128 / 0.7)"/>
              <stop offset="100%" stopColor="oklch(0.86 0.17 128 / 0)"/>
            </radialGradient>
            <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
              <path d="M 40 0 L 0 0 0 40" fill="none" stroke="var(--border-1)" strokeWidth="0.5"/>
            </pattern>
          </defs>
          <rect width="960" height="640" fill="url(#grid)"/>

          {/* Edges */}
          {links.map((l, i) => (
            <line
              key={i}
              x1={l.from.x} y1={l.from.y}
              x2={l.to.x} y2={l.to.y}
              stroke={clusterColor[l.from.cluster]}
              strokeWidth={Math.min(2, 0.3 + Math.log(l.weight) * 0.2)}
              opacity={0.15 + Math.min(0.4, l.weight / 800 * 0.4)}
            />
          ))}

          {/* Nodes */}
          {nodes.map((n, i) => (
            <g key={n.addr} onMouseEnter={() => setHovered(n)} onMouseLeave={() => setHovered(null)} style={{ cursor: "pointer" }}>
              {n.agent && (
                <circle cx={n.x} cy={n.y} r={n.size + 8} fill="url(#nodeGlow)"/>
              )}
              <circle
                cx={n.x} cy={n.y} r={n.size}
                fill={clusterColor[n.cluster]}
                opacity={hovered && hovered.addr !== n.addr ? 0.3 : 0.9}
                stroke={n.agent ? "var(--fg-0)" : "transparent"}
                strokeWidth="1"
              />
              {(n.agent || hovered?.addr === n.addr) && (
                <text
                  x={n.x + n.size + 6}
                  y={n.y + 3}
                  fill="var(--fg-1)"
                  fontFamily="var(--mono)"
                  fontSize="9"
                >{n.agent ? n.agent.name : shortAddr(n.addr)}</text>
              )}
            </g>
          ))}
        </svg>

        <div className="graph-legend">
          <div style={{ color: "var(--fg-0)", fontWeight: 500, marginBottom: 4 }}>CLUSTERS · 4 detected</div>
          <div className="row"><span className="dot" style={{ background: "var(--accent)" }}/> market makers</div>
          <div className="row"><span className="dot" style={{ background: "var(--info)" }}/> settlement</div>
          <div className="row"><span className="dot" style={{ background: "var(--magenta)" }}/> bridges / cctp</div>
          <div className="row"><span className="dot" style={{ background: "var(--warn)" }}/> retail</div>
          <div style={{ marginTop: 8, paddingTop: 8, borderTop: "1px solid var(--border-2)", color: "var(--fg-3)" }}>
            edge width = tx_count<br/>node halo = ERC-8004 agent
          </div>
        </div>

        <div className="graph-controls">
          <button className="btn">−</button>
          <button className="btn">＋</button>
          <button className="btn">⌂</button>
        </div>

        <div className="graph-detail">
          {hovered ? (
            <>
              <div className="mono dim" style={{ fontSize: 10, letterSpacing: "0.08em", textTransform: "uppercase" }}>WALLET</div>
              <div style={{ marginTop: 4, fontFamily: "var(--mono)", fontSize: 12, color: "var(--fg-0)" }}>{shortAddr(hovered.addr)}</div>
              {hovered.agent && (
                <div style={{ marginTop: 6 }}>
                  <span className="badge acc">{hovered.agent.name}</span>
                </div>
              )}
              <div style={{ marginTop: 10, display: "grid", gridTemplateColumns: "1fr 1fr", gap: 6, fontSize: 11, fontFamily: "var(--mono)" }}>
                <div><div className="dim" style={{ fontSize: 9 }}>EDGES OUT</div><div className="fg0">{Math.floor(Math.random()*40+5)}</div></div>
                <div><div className="dim" style={{ fontSize: 9 }}>EDGES IN</div><div className="fg0">{Math.floor(Math.random()*40+5)}</div></div>
                <div><div className="dim" style={{ fontSize: 9 }}>VOLUME 24h</div><div className="fg0">${fmtNum(Math.random()*1e7)}</div></div>
                <div><div className="dim" style={{ fontSize: 9 }}>CLUSTER</div><div className="fg0">#{hovered.cluster}</div></div>
              </div>
            </>
          ) : (
            <>
              <div className="mono dim" style={{ fontSize: 10, letterSpacing: "0.08em", textTransform: "uppercase" }}>PROJECTION</div>
              <div style={{ marginTop: 4, fontFamily: "var(--mono)", fontSize: 12, color: "var(--fg-1)" }}>
                top 64 by tx_count<br/>force-directed, t-sne color
              </div>
              <div className="dim" style={{ marginTop: 8, fontSize: 11 }}>Hover a node for details, or open the full 3D viewer.</div>
            </>
          )}
        </div>
      </div>

      <div className="grid grid-3" style={{ marginTop: 12 }}>
        <div className="card">
          <div className="card-head"><div className="card-title">Top edges by tx_count</div></div>
          <div className="card-body flush">
            <table className="tbl">
              <thead><tr><th>from</th><th>to</th><th className="num">tx</th><th className="num">vol</th></tr></thead>
              <tbody>
                {EDGES.slice(0, 6).sort((a,b) => b.tx_count - a.tx_count).map((e, i) => (
                  <tr key={i}>
                    <td><Addr a={e.from_wallet}/></td>
                    <td><Addr a={e.to_wallet}/></td>
                    <td className="num">{fmtNum(e.tx_count)}</td>
                    <td className="num muted">${fmtNum(parseFloat(e.volume))}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
        <div className="card">
          <div className="card-head"><div className="card-title">New edges · 24h</div></div>
          <div className="card-body" style={{ padding: 14 }}>
            <div style={{ fontFamily: "var(--mono)", fontSize: 28, color: "var(--fg-0)" }}>+ 14,209</div>
            <div className="dim mono" style={{ fontSize: 11, marginTop: 4 }}>↑ 12.4% vs prior 24h</div>
            <div style={{ height: 60, marginTop: 14 }}><BarChart data={SERIES.cctp_in} color="var(--accent)" height={60}/></div>
          </div>
        </div>
        <div className="card">
          <div className="card-head"><div className="card-title">Cluster sizes</div></div>
          <div className="card-body" style={{ padding: 14 }}>
            {[
              ["market makers", 41, "accent"],
              ["settlement",    18, "info"],
              ["bridges / cctp",24, "magenta"],
              ["retail",       317, "warn"],
            ].map(([name, n, c]) => (
              <div className="row" key={name} style={{ marginBottom: 8 }}>
                <span className="dot" style={{ background: `var(--${c})` }}/>
                <span className="mono" style={{ flex: 1 }}>{name}</span>
                <span className="mono fg0">{n}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

Object.assign(window, { AgentsView, JobsView, GraphView });
