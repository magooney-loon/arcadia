/* List views: Blocks, Transactions, Transfers, Traces */

const FilterChip = ({ on, label, value, onToggle }) => (
  <span className={"chip" + (on ? " on" : "")} onClick={onToggle}>
    {label}{value && <span className="mono fg0">:{value}</span>}
    {on && <span className="x">×</span>}
  </span>
);

const Pagination = ({ from, to, total }) => (
  <div className="row" style={{ padding: "10px 14px", borderTop: "1px solid var(--border-1)", fontSize: 11, fontFamily: "var(--mono)" }}>
    <span className="dim">showing</span>
    <span className="fg0">{from}–{to}</span>
    <span className="dim">of {fmtFull(total)}</span>
    <div className="spacer"/>
    <button className="btn ghost" style={{ height: 24 }}>← prev</button>
    <button className="btn" style={{ height: 24 }}>next →</button>
  </div>
);

const BlocksView = () => (
  <div className="view">
    <div className="view-head">
      <div>
        <div className="view-title">Blocks</div>
        <div className="view-sub">head #{fmtFull(HEAD_BLOCK)} · 412ms avg · 26,180 blocks/day</div>
      </div>
      <div className="view-actions">
        <button className="btn ghost">{I.filter} Filter</button>
        <button className="btn">{I.download} CSV</button>
      </div>
    </div>

    <div className="filter-bar">
      <FilterChip on label="height" value={"≤ " + fmtFull(HEAD_BLOCK)} />
      <FilterChip label="proposer" />
      <FilterChip label="min txs" />
      <FilterChip label="last 24h" />
    </div>

    <div className="card">
      <table className="tbl">
        <thead>
          <tr>
            <th>height</th>
            <th>age</th>
            <th>proposer</th>
            <th className="num">txs</th>
            <th className="num">transfers</th>
            <th className="num">gas used</th>
            <th className="num">block time</th>
            <th className="num">fees</th>
            <th>hash</th>
          </tr>
        </thead>
        <tbody>
          {BLOCKS.map(b => (
            <tr key={b.number}>
              <td><span className="acc">#{fmtFull(b.number)}</span></td>
              <td className="muted">{ago(b.timestamp)} ago</td>
              <td><Addr a={b.miner}/></td>
              <td className="num">{fmtFull(b.tx_count)}</td>
              <td className="num">{fmtFull(b.transfer_count)}</td>
              <td className="num">{fmtNum(b.gas_used)}</td>
              <td className="num"><span className="muted">{b.block_time_ms}ms</span></td>
              <td className="num">{b.fees} <span className="muted">USDC</span></td>
              <td><Hash h={b.hash}/></td>
            </tr>
          ))}
        </tbody>
      </table>
      <Pagination from="1" to="40" total={26_891_201}/>
    </div>
  </div>
);

const TxsView = () => (
  <div className="view">
    <div className="view-head">
      <div>
        <div className="view-title">Transactions</div>
        <div className="view-sub">streaming · 312,457,891 total (24h) · all kinds</div>
      </div>
      <div className="view-actions">
        <button className="btn ghost">{I.filter} Filter</button>
        <button className="btn">{I.download} JSON</button>
      </div>
    </div>

    <div className="filter-bar">
      <FilterChip on label="kind" value="all"/>
      <FilterChip label="kind" value="transfer"/>
      <FilterChip label="kind" value="swap"/>
      <FilterChip label="kind" value="cctp_burn"/>
      <FilterChip label="kind" value="cctp_mint"/>
      <FilterChip label="kind" value="agent_call"/>
      <FilterChip label="status" value="reverted"/>
    </div>

    <div className="card">
      <table className="tbl">
        <thead>
          <tr>
            <th>tx hash</th>
            <th>block</th>
            <th>age</th>
            <th>kind</th>
            <th>from</th>
            <th></th>
            <th>to</th>
            <th className="num">value</th>
            <th className="num">fee</th>
            <th>status</th>
          </tr>
        </thead>
        <tbody>
          {TXS.map(tx => (
            <tr key={tx.hash}>
              <td><Hash h={tx.hash}/></td>
              <td><span className="acc">#{fmtFull(tx.block_number)}</span></td>
              <td className="muted">{ago(tx.timestamp)}</td>
              <td><span className="badge muted">{tx.kind}</span></td>
              <td><Addr a={tx.from_addr}/></td>
              <td className="muted">→</td>
              <td><Addr a={tx.to_addr}/></td>
              <td className="num">{parseFloat(tx.value) > 0 ? <span><span className="fg0">{fmtNum(parseFloat(tx.value), 2)}</span> <span className="muted">USDC</span></span> : <span className="muted">—</span>}</td>
              <td className="num muted">{tx.fee}</td>
              <td><span className={"badge " + (tx.status === "ok" ? "ok" : "err")}>{tx.status}</span></td>
            </tr>
          ))}
        </tbody>
      </table>
      <Pagination from="1" to="80" total={312_457_891}/>
    </div>
  </div>
);

const TransfersView = () => (
  <div className="view">
    <div className="view-head">
      <div>
        <div className="view-title">Token transfers</div>
        <div className="view-sub">ERC-20 + native · 14.2M transfers in last 24h</div>
      </div>
      <div className="view-actions">
        <button className="btn ghost">{I.filter} Filter</button>
        <button className="btn">{I.download} CSV</button>
      </div>
    </div>

    <div className="filter-bar">
      <FilterChip on label="token" value="all"/>
      {TOKENS.map(t => <FilterChip key={t.sym} label="token" value={t.sym}/>)}
    </div>

    <div className="grid grid-2" style={{ marginBottom: 12 }}>
      <div className="card">
        <div className="card-head">
          <div className="card-title">Volume by token · 24h</div>
          <div className="card-sub">USD equivalent</div>
        </div>
        <div className="card-body">
          {[
            ["USDC", 38_201_010_000, "info", 91],
            ["EURC", 1_802_410_000,  "info", 4.3],
            ["BRZ",    420_109_000,  "ok",   1.0],
            ["MXNe",   181_010_000,  "warn", 0.4],
            ["PYUSD",   72_910_000,  "info", 0.18],
            ["ARC",  1_290_410_000,  "acc",  3.1],
          ].map(([sym, vol, color, pct]) => (
            <div key={sym} className="row" style={{ marginBottom: 8 }}>
              <Token sym={sym}/>
              <div style={{ flex: 1, height: 8, background: "var(--bg-3)", borderRadius: 2, overflow: "hidden" }}>
                <div style={{
                  height: "100%",
                  width: pct + "%",
                  background: `var(--${color === "info" ? "info" : color === "warn" ? "warn" : color === "ok" ? "ok" : "accent"})`,
                }}/>
              </div>
              <span className="mono fg0" style={{ width: 80, textAlign: "right" }}>${fmtNum(vol)}</span>
              <span className="mono dim" style={{ width: 40, textAlign: "right" }}>{pct}%</span>
            </div>
          ))}
        </div>
      </div>

      <div className="card">
        <div className="card-head">
          <div className="card-title">Transfer count · 60 blocks</div>
          <div className="card-sub">5s windows</div>
        </div>
        <div className="card-body" style={{ padding: 14 }}>
          <div style={{ height: 110 }}>
            <BarChart data={SERIES.tps.map(v => v * 0.9)} color="var(--accent)" height={110}/>
          </div>
          <div className="row" style={{ marginTop: 8, fontSize: 11, fontFamily: "var(--mono)" }}>
            <span className="dim">peak</span><span className="fg0">5,812 / 5s</span>
            <span className="dim" style={{ marginLeft: 12 }}>median</span><span className="fg0">3,901 / 5s</span>
            <span className="dim" style={{ marginLeft: 12 }}>now</span><span className="acc">4,287 / 5s</span>
          </div>
        </div>
      </div>
    </div>

    <div className="card">
      <table className="tbl">
        <thead>
          <tr>
            <th>tx</th>
            <th>block</th>
            <th>age</th>
            <th>token</th>
            <th>from</th>
            <th></th>
            <th>to</th>
            <th className="num">amount</th>
            <th className="num">usd</th>
          </tr>
        </thead>
        <tbody>
          {TRANSFERS.map(t => (
            <tr key={t.tx_hash + t.log_index}>
              <td><Hash h={t.tx_hash}/></td>
              <td><span className="acc">#{fmtFull(t.block_number)}</span></td>
              <td className="muted">{ago(t.timestamp)}</td>
              <td><Token sym={t.token_symbol}/></td>
              <td><Addr a={t.from_addr}/></td>
              <td className="muted">→</td>
              <td><Addr a={t.to_addr}/></td>
              <td className="num"><span className="fg0">{fmtNum(parseFloat(t.amount), 2)}</span> <span className="muted">{t.token_symbol}</span></td>
              <td className="num muted">${fmtNum(parseFloat(t.usd_value))}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <Pagination from="1" to="60" total={14_201_882}/>
    </div>
  </div>
);

const TracesView = () => (
  <div className="view">
    <div className="view-head">
      <div>
        <div className="view-title">Internal traces</div>
        <div className="view-sub">contract-to-contract calls · filter by tx hash or address</div>
      </div>
      <div className="view-actions">
        <button className="btn ghost">{I.filter} Filter</button>
      </div>
    </div>

    <div className="card">
      <div className="card-head">
        <div className="card-title">Call tree · tx 0x82af…91c</div>
        <div className="card-sub">14 internal calls · gas 412,891</div>
      </div>
      <div className="card-body" style={{ fontFamily: "var(--mono)", fontSize: 12, lineHeight: 1.8 }}>
        {[
          [0, "CALL",         "0x7c1d…f8c0e", "RouterV3.fillRFQ(quote_id, signature)", "gas 412891", "ok"],
          [1, "STATICCALL",   "0xae4b…2c4f6", "FXOracle.priceOf(USDC, EURC)",          "gas 8121",  "ok"],
          [1, "CALL",         "0xb1c2…b8c9d", "USDC.transferFrom(taker, router, 12000.00)", "gas 21391", "ok"],
          [1, "DELEGATECALL", "0xff89…7e6d5", "FXVault.execute(swapData)",             "gas 89102", "ok"],
          [2, "CALL",         "0xb1c2…b8c9d", "USDC.transfer(maker, 11999.04)",        "gas 21391", "ok"],
          [2, "CALL",         "0x9d8c…170f", "EURC.transferFrom(maker, taker, 11038.21)", "gas 21891", "ok"],
          [2, "CALL",         "0x2c4e…0a2c", "Settlement.recordFill(quote_id)",        "gas 41091", "ok"],
          [1, "CALL",         "0x4d6e…0a2c", "Compliance.attest(taker, USDC)",         "gas 19421", "ok"],
          [2, "STATICCALL",   "0x3a5c…9a1c", "Policy.allow(taker)",                    "gas 4012",  "ok"],
          [1, "LOG",          "—",           "Fill(quote_id, taker, maker, 12000, 11038.21)", "", "emit"],
        ].map(([depth, op, addr, fn, gas, status], i) => (
          <div key={i} style={{ paddingLeft: depth * 22, display: "flex", gap: 12, alignItems: "baseline" }}>
            <span className="dim" style={{ width: 24 }}>{String(i+1).padStart(2, "0")}</span>
            <span className={"badge " + (status === "emit" ? "mag" : "muted")} style={{ minWidth: 86, justifyContent: "center" }}>{op}</span>
            <span className="addr">{addr}</span>
            <span className="fg0">{fn}</span>
            <span className="spacer"/>
            <span className="dim">{gas}</span>
            <span className={"badge " + (status === "ok" ? "ok" : "mag")}>{status}</span>
          </div>
        ))}
      </div>
    </div>
  </div>
);

Object.assign(window, { BlocksView, TxsView, TransfersView, TracesView });
