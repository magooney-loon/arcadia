package jobs

import "sync"

// heavyJobMu serializes the heavy analytics jobs (token_analytics and the 7d
// analytics snapshot) so they don't both pin SQLite at the same time and
// throttle the indexer write path. Callers use TryLock and skip the run if
// another heavy job is in flight — the next cron tick will pick it up.
var heavyJobMu sync.Mutex
