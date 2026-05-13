/* Overview dashboard. */

const StatCard = ({ label, value, unit, delta, deltaDir, spark, sparkColor }) => (
  <div className="stat">
    <div className="label">{label}</div>
    <div className="value">{value}{unit && <span className="unit">{unit}</span>}</div>
    {delta && <div className={"delta " + (deltaDir === "down" ? "down" : "up")}>{deltaDir === "down" ? "▼" : "▲"} {delta}</div>}
    {spark && <div className="spark"><Sparkline data={spark} color={sparkColor || "var(--accent)"} height={40}/></div>}
  </div>
);

const OverviewView = () => {
  return (
    <div className="view">
      <div className="view-head">
        <div>
          <div className="view-title">Overview</div>
          <div className="view-sub">Live chain state · 26 · arcadia mainnet · refreshed 1s ago</div>
        </div>
        <div className="view-actions">
          <button className="btn ghost">{I.refresh} Pause stream</button>
          <button className="btn">{I.download} Export</button>
        </div>
      </div>

      {/* Top stat row */}
      <div className="grid grid-stats">
        <StatCard label="TPS"           value={fmtNum(STATS.tps)}            delta="+12.4%" spark={SERIES.tps}/>
        <StatCard label="Block time"    value={STATS.block_time_ms} unit="ms" delta="−1.8%" deltaDir="down" spark={SERIES.block_time} sparkColor="var(--info)"/>
        <StatCard label="Transfers 24h" value={fmtNum(STATS.transfer_count_24h, 1)} delta="+4.1%" spark={SERIES.vol}/>
        <StatCard label="Fees paid 24h" value={fmtNum(STATS.fees_paid_24h)} unit=" USDC" delta="+0.3%" spark={SERIES.gas} sparkColor="var(--warn)"/>
        <StatCard label="FX notional 24h" value={"$" + fmtNum(STATS.fx_notional_24h)} delta="+8.9%" spark={SERIES.fx_notional} sparkColor="var(--magenta)"/>
        <StatCard label="Active agents" value={fmtNum(STATS.active_agents)} delta="+7" spark={SERIES.cctp_in} sparkColor="var(--info)"/>
      </div>

      {/* Throughput + cross-chain panel row */}
      <div className="grid grid-2" style={{ marginTop: 12 }}>
        <div className="card">
          <div className="card-head">
            <div className="card-title">Throughput · transfer volume</div>
            <div className="card-sub">60 blocks · USDC equiv.</div>
            <div className="card-actions">
              <button className="btn ghost mono" style={{ height: 24, fontSize: 11 }}>1m</button>
              <button className="btn acc mono" style={{ height: 24, fontSize: 11 }}>1h</button>
              <button className="btn ghost mono" style={{ height: 24, fontSize: 11 }}>24h</button>
              <button className="btn ghost mono" style={{ height: 24, fontSize: 11 }}>7d</button>
            </div>
          </div>
          <div className="card-body" style={{ padding: 0 }}>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 0 }}>
              <div style={{ padding: 14, borderRight: "1px solid var(--border-1)" }}>
                <div className="card-sub" style={{ marginBottom: 8 }}>TRANSFER VOLUME · usd</div>
                <div style={{ height: 120 }}>
                  <Sparkline data={SERIES.vol} color="var(--accent)" height={120}/>
                </div>
                <div className="row" style={{ marginTop: 8, fontSize: 11 }}>
                  <span className="mono dim">peak</span><span className="mono fg0">$2.41B</span>
                  <span className="mono dim" style={{ marginLeft: 12 }}>avg</span><span className="mono fg0">$1.82B</span>
                </div>
              </div>
              <div style={{ padding: 14 }}>
                <div className="card-sub" style={{ marginBottom: 8 }}>TPS · 60 blocks</div>
                <div style={{ height: 120 }}>
                  <BarChart data={SERIES.tps} color="var(--info)" height={120}/>
                </div>
                <div className="row" style={{ marginTop: 8, fontSize: 11 }}>
                  <span className="mono dim">peak</span><span className="mono fg0">{fmtNum(STATS.tps_peak)}</span>
                  <span className="mono dim" style={{ marginLeft: 12 }}>p50</span><span className="mono fg0">3,940</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-head">
            <div className="card-title">Cross-chain pulse</div>
            <div className="card-sub">CCTP · 24h</div>
          </div>
          <div className="card-body" style={{ padding: "14px 14px 10px" }}>
            <div className="row" style={{ alignItems: "flex-start", gap: 18 }}>
              <div style={{ flex: 1 }}>
                <div className="card-sub">↘ INBOUND · MINTS</div>
                <div className="mono fg0" style={{ fontSize: 20, marginTop: 4 }}>{fmtNum(STATS.cctp_mints_24h)}</div>
                <div style={{ height: 50, marginTop: 4 }}><BarChart data={SERIES.cctp_in} color="var(--accent)" height={50}/></div>
              </div>
              <div style={{ flex: 1 }}>
                <div className="card-sub">↗ OUTBOUND · BURNS</div>
                <div className="mono fg0" style={{ fontSize: 20, marginTop: 4 }}>{fmtNum(STATS.cctp_burns_24h)}</div>
                <div style={{ height: 50, marginTop: 4 }}><BarChart data={SERIES.cctp_out} color="var(--info)" height={50}/></div>
              </div>
            </div>
            <div style={{ borderTop: "1px solid var(--border-1)", marginTop: 12, paddingTop: 8 }}>
              {CROSSCHAIN.slice(0, 5).map(e => (
                <div className="flow" key={e.id} style={{ borderBottom: 0, padding: "6px 0", fontSize: 11 }}>
                  <ChainChip domain={e.source_domain}/>
                  <span className="arrow">→</span>
                  <ChainChip domain={e.destination_domain}/>
                  <Token sym={e.token}/>
                  <span className="amt">{fmtNum(parseFloat(e.amount))}</span>
                  <span className="sub" style={{ marginLeft: 8 }}>{ago(e.timestamp)} ago</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Live blocks + txs */}
      <div className="grid grid-2" style={{ marginTop: 12 }}>
        <div className="card">
          <div className="card-head">
            <span className="dot acc"/>
            <div className="card-title">Latest blocks</div>
            <div className="card-sub">streaming</div>
            <div className="card-actions"><span className="mono dim" style={{ fontSize: 10 }}>SEE ALL →</span></div>
          </div>
          <div className="card-body flush">
            {BLOCKS.slice(0, 8).map(b => (
              <div className="live-row" key={b.number}>
                <span className="num">#{fmtFull(b.number)}</span>
                <span className="age">{ago(b.timestamp)}</span>
                <span className="txs">{fmtFull(b.tx_count)} tx</span>
                <span className="muted">by</span>
                <Addr a={b.miner}/>
                <span className="fees">{b.fees} USDC fees</span>
              </div>
            ))}
          </div>
        </div>

        <div className="card">
          <div className="card-head">
            <span className="dot acc"/>
            <div className="card-title">Latest transactions</div>
            <div className="card-sub">streaming · all kinds</div>
            <div className="card-actions"><span className="mono dim" style={{ fontSize: 10 }}>SEE ALL →</span></div>
          </div>
          <div className="card-body flush">
            <table className="tbl">
              <tbody>
                {TXS.slice(0, 9).map(tx => (
                  <tr key={tx.hash}>
                    <td><Hash h={tx.hash}/></td>
                    <td><span className={"badge " + (tx.status === "ok" ? "muted" : "err")}>{tx.kind}</span></td>
                    <td><Addr a={tx.from_addr}/></td>
                    <td className="muted">→</td>
                    <td><Addr a={tx.to_addr}/></td>
                    <td className="num muted">{ago(tx.timestamp)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Bottom — agents + fx */}
      <div className="grid grid-2" style={{ marginTop: 12 }}>
        <div className="card">
          <div className="card-head">
            <div className="card-title">Top agents · last 24h</div>
            <div className="card-sub">ERC-8004</div>
            <div className="card-actions"><span className="mono dim" style={{ fontSize: 10 }}>REGISTRY →</span></div>
          </div>
          <div className="card-body flush">
            {AGENTS.slice(0, 5).map(a => (
              <div className="agent-row" key={a.addr}>
                <div className="agent-avatar">{a.name[0].toUpperCase()}</div>
                <div className="agent-meta">
                  <div className="agent-name">{a.name}</div>
                  <div className="agent-sub">{a.domain} · <Addr a={a.addr}/></div>
                </div>
                <div className="agent-stats">
                  <div><span className="s-lbl">trust</span>{a.trust}</div>
                  <div><span className="s-lbl">jobs</span>{fmtNum(a.jobs)}</div>
                  <div><span className="s-lbl">succ</span>{a.success}%</div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="card">
          <div className="card-head">
            <div className="card-title">StableFX · live trades</div>
            <div className="card-sub">RFQ · last hour</div>
            <div className="card-actions"><span className="mono dim" style={{ fontSize: 10 }}>FX BOOK →</span></div>
          </div>
          <div className="card-body flush">
            <table className="tbl">
              <thead>
                <tr><th>pair</th><th>size</th><th>price</th><th>maker</th><th>state</th><th className="num">age</th></tr>
              </thead>
              <tbody>
                {FX.slice(0, 8).map(t => (
                  <tr key={t.quote_id}>
                    <td><Token sym={t.base_token}/> <span className="muted">/</span> <Token sym={t.quote_token}/></td>
                    <td className="num">{fmtNum(parseFloat(t.base_amount))}</td>
                    <td className="num">{t.price}</td>
                    <td><Addr a={t.maker}/></td>
                    <td>
                      <span className={"badge " + (t.status === "filled" ? "ok" : t.status === "partial" ? "warn" : "muted")}>
                        {t.status}
                      </span>
                    </td>
                    <td className="num muted">{ago(t.timestamp)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

window.OverviewView = OverviewView;
