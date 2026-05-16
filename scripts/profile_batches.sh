#!/usr/bin/env bash
# Summarize indexer batch_profile log lines: count, avg, p50, p95, max per phase.
#
# Usage:
#   ./scripts/profile_batches.sh <logfile>
#   tail -f logs/server.log | ./scripts/profile_batches.sh -    # not streaming; pipe a finite chunk
#   journalctl -u arcadia | ./scripts/profile_batches.sh -
#
# Expects lines of the form:
#   [indexer] batch_profile blocks=N txs=N logs=N traces=N | seen=Nms tx_total=Nms | blocks=Nms txs=Nms logs=Nms traces=Nms backfill=Nms stats=Nms edges=Nms agents=Nms

set -euo pipefail

src="${1:-}"
if [[ -z "$src" ]]; then
  echo "usage: $0 <logfile|->" >&2
  exit 1
fi

awk '
/batch_profile/ {
  for (i = 1; i <= NF; i++) {
    # Phase timings: key=Nms — tag with _ms suffix so they do not collide with input counts.
    if (match($i, /^([a-z_]+)=([0-9]+)ms/, m)) {
      k = m[1] "_ms"; v = m[2] + 0
      vals[k, ++cnt[k]] = v
      sum[k] += v
      if (v > max[k]) max[k] = v
    } else if (match($i, /^([a-z_]+)=([0-9]+)$/, m)) {
      k = m[1]; v = m[2] + 0
      vals[k, ++cnt[k]] = v
      sum[k] += v
      if (v > max[k]) max[k] = v
    }
  }
}
END {
  # Stable ordering: input counts first, then timings.
  order = "blocks txs logs traces seen tx_total blocks_ms txs_ms logs_ms traces_ms backfill stats edges agents"
  # but the keys above duplicate "blocks" / "txs" / "logs" / "traces" between input-count
  # and phase-timing — they share names in the log line.  Group: counts vs timings separated
  # by the order they appear (count keys appear before "tx_total", phase keys after).
  # Simplest: just iterate cnt and print every distinct key once with its stats.
  printf "%-12s %8s %10s %10s %10s %10s\n", "key", "n", "avg", "p50", "p95", "max"
  printf "%-12s %8s %10s %10s %10s %10s\n", "---", "-", "---", "---", "---", "---"
  for (k in cnt) {
    n = cnt[k]
    # copy into tmp array and sort
    delete tmp
    for (i = 1; i <= n; i++) tmp[i] = vals[k, i]
    # insertion sort (small n typical)
    for (i = 2; i <= n; i++) {
      key = tmp[i]; j = i - 1
      while (j >= 1 && tmp[j] > key) { tmp[j+1] = tmp[j]; j-- }
      tmp[j+1] = key
    }
    p50_idx = int((n + 1) * 0.50); if (p50_idx < 1) p50_idx = 1; if (p50_idx > n) p50_idx = n
    p95_idx = int((n + 1) * 0.95); if (p95_idx < 1) p95_idx = 1; if (p95_idx > n) p95_idx = n
    avg = sum[k] / n
    printf "%-12s %8d %10.1f %10d %10d %10d\n", k, n, avg, tmp[p50_idx], tmp[p95_idx], max[k]
  }
}
' "$src" | (read header; read sep; echo "$header"; echo "$sep"; sort)
