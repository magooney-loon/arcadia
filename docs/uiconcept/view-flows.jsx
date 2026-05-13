/* Cross-chain + StableFX views */

const CrossChainView = () => {
  return (
    <div className="view">
      <div className="view-head">
        <div>
          <div className="view-title">Cross-chain</div>
          <div className="view-sub">CCTP V2 + Circle Gateway · domain 26 (Arcadia)</div>
        </div>
        <div className="view-actions">
          <button className="btn ghost">{I.filter} Filter</button>
          <button className="btn">{I.download} Export</button>
        </div>
      </div>

      <div className="grid" style={{ gridTemplateColumns: "repeat(4, 1fr)", marginBottom: 12 }}>
        <div className="stat">
          <div className="label">↘ Inbound mints 24h</div>
          <div className="value">{fmtNum(STATS.cctp_mints_24h)}</div>
          <div className="delta up">▲ 12.1%</div>
          <div className="spark"><Sparkline data={SERIES.cctp_in} color="var(--accent)"/></div>
        </div>
        <div className="stat">
          <div className="label">↗ Outbound burns 24h</div>
          <div className="value">{fmtNum(STATS.cctp_burns_24h)}</div>
          <div className="delta up">▲ 8.4%</div>
          <div className="spark"><Sparkline data={SERIES.cctp_out} color="var(--info)"/></div>
        </div>
        <div className="stat">
          <div className="label">Net flow 24h</div>
          <div className="value">+{fmtNum(STATS.cctp_mints_24h - STATS.cctp_burns_24h)}</div>
          <div className="delta up">▲ inbound</div>
        </div>
        <div className="stat">
          <div className="label">Median finality</div>
          <div className="value">14.2<span className="unit"> s</span></div>
          <div className="delta down">▼ 0.3s</div>
        </div>
      </div>

      <div className="grid grid-2" style={{ marginBottom: 12 }}>
        <div className="card">
          <div className="card-head">
            <div className="card-title">Counterparty chains · 24h volume</div>
            <div className="card-sub">USDC equivalent</div>
          </div>
          <div className="card-body">
            {[
              ["Ethereum",  0,  21_408_201_000, "info"],
              ["Base",      6,  8_402_910_000,  "info"],
              ["Arbitrum",  3,  4_201_410_000,  "info"],
              ["Solana",    9,  3_811_290_000,  "mag"],
              ["OP Mainnet",2,  1_902_500_000,  "info"],
              ["Polygon",   7,  1_412_010_000,  "mag"],
              ["Avalanche", 1,    910_810_000,  "warn"],
            ].map(([name, dom, vol, color]) => {
              const pct = (vol / 21_408_201_000) * 100;
              return (
                <div key={name} className="row" style={{ marginBottom: 8 }}>
                  <span className="mono fg0" style={{ width: 90 }}>{name}</span>
                  <span className="badge muted" style={{ width: 50, justifyContent: "center" }}>D{dom}</span>
                  <div style={{ flex: 1, height: 8, background: "var(--bg-3)", borderRadius: 2, overflow: "hidden" }}>
                    <div style={{ height: "100%", width: pct + "%", background: `var(--${color === "info" ? "info" : color === "warn" ? "warn" : "magenta"})` }}/>
                  </div>
                  <span className="mono fg0" style={{ width: 80, textAlign: "right" }}>${fmtNum(vol)}</span>
                </div>
              );
            })}
          </div>
        </div>

        <div className="card">
          <div className="card-head">
            <div className="card-title">Flow direction · last 30 events</div>
            <div className="card-sub">live</div>
          </div>
          <div className="card-body flush" style={{ maxHeight: 320, overflowY: "auto" }}>
            {CROSSCHAIN.slice(0, 14).map(e => {
              const inbound = e.destination_domain === 26;
              return (
                <div className="flow" key={e.id}>
                  <ChainChip domain={e.source_domain}/>
                  <span className="arrow">→</span>
                  <ChainChip domain={e.destination_domain}/>
                  <span className={"badge " + (inbound ? "ok" : "info")}>{inbound ? "MINT" : "BURN"}</span>
                  <span className="sub" style={{ marginLeft: 6 }}>{e.protocol}</span>
                  <span className="amt"><Token sym={e.token}/> {fmtNum(parseFloat(e.amount))}</span>
                  <span className="sub" style={{ marginLeft: 8 }}>{ago(e.timestamp)}</span>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <div className="filter-bar">
        <FilterChip on label="protocol" value="all"/>
        <FilterChip label="protocol" value="CCTP"/>
        <FilterChip label="protocol" value="Gateway"/>
        <FilterChip label="direction" value="inbound"/>
        <FilterChip label="direction" value="outbound"/>
        <FilterChip label="status" value="pending"/>
      </div>

      <div className="card">
        <table className="tbl">
          <thead>
            <tr>
              <th>id</th>
              <th>protocol</th>
              <th>event</th>
              <th>route</th>
              <th>sender</th>
              <th>recipient</th>
              <th className="num">amount</th>
              <th>token</th>
              <th>status</th>
              <th className="num">age</th>
            </tr>
          </thead>
          <tbody>
            {CROSSCHAIN.map(e => (
              <tr key={e.id}>
                <td className="mono dim">{e.id}</td>
                <td><span className={"badge " + (e.protocol === "CCTP" ? "info" : "mag")}>{e.protocol}</span></td>
                <td className="mono">{e.event_type}</td>
                <td>
                  <ChainChip domain={e.source_domain}/>
                  <span className="acc" style={{ margin: "0 6px" }}>→</span>
                  <ChainChip domain={e.destination_domain}/>
                </td>
                <td><Addr a={e.sender}/></td>
                <td><Addr a={e.recipient}/></td>
                <td className="num"><span className="fg0">{fmtNum(parseFloat(e.amount))}</span></td>
                <td><Token sym={e.token}/></td>
                <td><span className={"badge " + (e.status === "finalized" ? "ok" : "warn")}>{e.status}</span></td>
                <td className="num muted">{ago(e.timestamp)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination from="1" to="40" total={891_201}/>
      </div>
    </div>
  );
};

const FXView = () => {
  const PAIRS = [
    { p: "USDC/EURC", px: "0.9203", chg: "+0.04%", up: true,  vol: "2.81B" },
    { p: "USDC/BRZ",  px: "5.4231", chg: "−0.31%", up: false, vol: "1.42B" },
    { p: "USDC/MXNe", px: "17.8912", chg: "+0.18%", up: true,  vol: "918M" },
    { p: "EURC/BRZ",  px: "5.8901", chg: "−0.22%", up: false, vol: "412M" },
    { p: "USDC/PYUSD",px: "1.0001", chg: "+0.00%", up: true,  vol: "281M" },
  ];

  return (
    <div className="view">
      <div className="view-head">
        <div>
          <div className="view-title">StableFX</div>
          <div className="view-sub">RFQ orderbook · 5 pairs · 184k trades in 24h</div>
        </div>
        <div className="view-actions">
          <button className="btn ghost">{I.filter} Filter</button>
          <button className="btn">{I.download} Trades CSV</button>
        </div>
      </div>

      <div className="grid" style={{ gridTemplateColumns: "repeat(5, 1fr)", marginBottom: 12 }}>
        {PAIRS.map(p => (
          <div className="stat" key={p.p}>
            <div className="label mono">{p.p}</div>
            <div className="value" style={{ fontSize: 22 }}>{p.px}</div>
            <div className={"delta " + (p.up ? "up" : "down")}>{p.up ? "▲" : "▼"} {p.chg}</div>
            <div className="card-sub" style={{ marginTop: 4 }}>vol {p.vol}</div>
          </div>
        ))}
      </div>

      <div className="grid grid-2" style={{ marginBottom: 12 }}>
        <div className="card">
          <div className="card-head">
            <div className="card-title">RFQ depth · USDC/EURC</div>
            <div className="card-sub">live · top 8 quotes</div>
          </div>
          <div className="card-body flush">
            <table className="tbl">
              <thead>
                <tr><th>side</th><th>maker</th><th className="num">price</th><th className="num">size</th><th className="num">expiry</th></tr>
              </thead>
              <tbody>
                {[
                  ["bid", AGENTS[1], 0.92034, 2_100_000, 14],
                  ["bid", AGENTS[7], 0.92031, 950_000,  21],
                  ["bid", AGENTS[1], 0.92028, 1_400_000, 7],
                  ["bid", AGENTS[4], 0.92024, 600_000,  18],
                  ["ask", AGENTS[4], 0.92041, 800_000,  11],
                  ["ask", AGENTS[1], 0.92044, 2_400_000, 19],
                  ["ask", AGENTS[7], 0.92048, 1_100_000, 8],
                  ["ask", AGENTS[4], 0.92053, 500_000,  22],
                ].map(([side, agent, px, size, exp], i) => (
                  <tr key={i}>
                    <td><span className={"badge " + (side === "bid" ? "ok" : "err")}>{side}</span></td>
                    <td className="mono">{agent.name}</td>
                    <td className={"num fg0 " + (side === "bid" ? "" : "")} style={{ color: side === "bid" ? "var(--ok)" : "var(--err)" }}>{px.toFixed(5)}</td>
                    <td className="num">{fmtNum(size)}</td>
                    <td className="num muted">{exp}s</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className="card">
          <div className="card-head">
            <div className="card-title">Notional · 24h hourly</div>
            <div className="card-sub">all pairs</div>
          </div>
          <div className="card-body" style={{ padding: 14 }}>
            <div style={{ height: 240 }}>
              <BarChart data={SERIES.fx_notional} color="var(--magenta)" height={240}/>
            </div>
          </div>
        </div>
      </div>

      <div className="card">
        <div className="card-head">
          <div className="card-title">Recent fills</div>
          <div className="card-sub">all pairs · sorted by block</div>
        </div>
        <table className="tbl">
          <thead>
            <tr>
              <th>quote_id</th><th>block</th><th>pair</th>
              <th>maker</th><th>taker</th>
              <th className="num">base</th><th className="num">quote</th><th className="num">price</th>
              <th>status</th><th className="num">age</th>
            </tr>
          </thead>
          <tbody>
            {FX.slice(0, 24).map(t => (
              <tr key={t.quote_id}>
                <td className="mono dim">{t.quote_id}</td>
                <td><span className="acc">#{fmtFull(t.block_number)}</span></td>
                <td><Token sym={t.base_token}/> <span className="muted">/</span> <Token sym={t.quote_token}/></td>
                <td><Addr a={t.maker}/></td>
                <td><Addr a={t.taker}/></td>
                <td className="num">{fmtNum(parseFloat(t.base_amount))}</td>
                <td className="num">{fmtNum(parseFloat(t.quote_amount))}</td>
                <td className="num mono">{t.price}</td>
                <td><span className={"badge " + (t.status === "filled" ? "ok" : t.status === "partial" ? "warn" : "muted")}>{t.status}</span></td>
                <td className="num muted">{ago(t.timestamp)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination from="1" to="24" total={184_001}/>
      </div>
    </div>
  );
};

Object.assign(window, { CrossChainView, FXView });
