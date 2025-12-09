# Funky NATS + Via Demo Ideas

Showcasing real-time synchronization between NATS and Via across multiple browser tabs/instances.



---

## Implemented

### 1. Live Theme Sync via NATS KV
**Page:** `/themes`
**Status:** DONE

Change theme on one Via instance -> NATS KV -> ALL Via instances update in real-time. Multiple browser tabs, all sync instantly.

- Writes to NATS KV bucket `via_config` key `theme`
- All Via instances watch this key
- When it changes, they all see the update

**Try it:** Open two browser tabs to `/themes`, click a theme in one - watch the "NATS theme" update in the other!

---

### 2. NATS Service Registry Dashboard
**Page:** `/nats`
**Status:** DONE

Via page showing the live `services_registry` KV bucket - see services appear/disappear as they start/stop.

- Polls NATS KV every 2 seconds
- Shows service name, host, and registration time
- Services with TTL auto-expire if they stop heartbeating

---

### 3. Cross-Instance Chat
**Page:** `/chat`
**Status:** DONE

Multiple Via instances can send messages to each other through NATS pub/sub - click in one browser, appears in all.

- Messages published to NATS subject `via.chat`
- All Via instances subscribe to this subject
- Messages appear in real-time across all tabs

**Try it:** Open two browser tabs to `/chat`, click "Hello!" in one - it appears in both!

---

### 4. Bidirectional Process Control
**Page:** `/processes`
**Status:** DONE

Via controls process-compose which manages Via - the ultimate dog-fooding demo.

- Via -> process-compose: Click Stop/Start/Restart to control processes
- process-compose -> Via: Via is started and managed by process-compose
- Real-time updates every 2 seconds via SSE

**Try it:** Click "Restart" on "via" - the page reconnects automatically after Via restarts!

---

### 5. Distributed Counter
**Page:** `/counter`
**Status:** DONE

A counter stored in NATS KV that ALL browser tabs can increment - everyone sees the same value update in real-time.

- Counter stored in NATS KV bucket `via_config` key `counter`
- Increment (+1), Decrement (-1), Add 10, Reset buttons
- All tabs watch the key via NATS KV watcher
- Shows "Last updated by: user-XXX"

**Try it:** Open 5 browser tabs to `/counter`, click +1 in any tab - all 5 tabs update instantly!

---

### 6. Real-Time NATS Message Viewer
**Page:** `/monitor`
**Status:** DONE

Via page that subscribes to NATS subjects and displays messages as they flow through - like a live activity monitor.

- Subscribe to `>` (all subjects), `via.>`, or `$KV.>` patterns
- Display messages in a scrolling log with timestamps (newest first)
- Shows message count, rate (msgs/sec), and subjects seen
- Clear messages and stop/start subscription buttons

**Try it:** Open `/monitor`, click "All (>)", then use /chat or /counter in another tab - see the messages flow!

---

### 7. Config Hot-Reload Demo
**Page:** `/config`
**Status:** DONE

Change a config value in Via UI -> writes to NATS KV -> services watching that key react immediately.

- Config values stored in NATS KV bucket `via_config` with `config.` prefix
- Toggle buttons cycle through common values (true/false, debug/info/warn/error, enabled/disabled)
- All Via instances watch the `config.>` pattern
- Shows last config change with key, old/new values, who changed it, and when
- Real-time sync across all browser tabs

**Config Keys:**
- `app.name`, `app.debug`, `app.log_level`
- `feature.flag1`, `feature.flag2`
- `service.timeout`, `service.retry_count`

**Try it:** Open two browser tabs to `/config`, toggle a value in one - watch the other update instantly!

---

## Not Yet Implemented

### 8. Scale NATS Leaf Nodes from Via UI
**Page:** `/scale` (TODO)
**Status:** NOT STARTED

Button to spawn additional NATS leaf node processes via process-compose API.

**Implementation:**
- "Add Leaf Node" button calls process-compose to start new instance
- New instance appears in `/nats` service registry
- "Remove" button to scale down
- Show leaf connection topology

**Why it's cool:** Demonstrates dynamic scaling - add capacity from the UI, see it join the mesh.

---

### 9. NATS JetStream Streams Dashboard
**Page:** `/streams` (TODO)
**Status:** NOT STARTED

View and manage JetStream streams from Via.

**Implementation:**
- List all streams with message counts
- View stream config (retention, limits)
- Publish test messages
- Consume/peek messages

---

### 10. Service Dependency Graph
**Page:** `/graph` (TODO)
**Status:** NOT STARTED

Visualize which services depend on which other services, based on NATS registry.

**Implementation:**
- Parse `service:org/repo` tags from registrations
- Render dependency graph (simple ASCII or via external lib)
- Highlight unhealthy dependencies
- Click to see service details

---

## Running the Demo

```bash
# Start NATS + Via with process-compose
cd /Users/apple/workspace/go/src/github.com/joeblew999/wellnown-env
process-compose up --port 8181

# Open in browser
open http://localhost:3000
```

**Ports:**
- NATS: 4222
- Via: 3000
- process-compose API: 8181
