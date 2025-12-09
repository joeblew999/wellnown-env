# FAQ

Design decisions and "why did Claude do that?" explanations.

---

## Eating Our Own Dog Food

**wellnown-env manages itself.**

The Via dashboard isn't just a UI - it's a wellnown-env service that uses wellnown-env to configure itself. When you change the theme in the GUI:

1. The Via action writes `VIA_THEME=purple` to NATS KV
2. wellnown-env picks up the change (it's watching its own config)
3. The change propagates to all connected Via instances
4. Every browser tab updates via SSE

**This is the whole point**: The system demonstrates its own capabilities. If wellnown-env can manage its own theme/config in real-time, it can manage ANY service's config the same way.

```
┌─────────────────────────────────────────────────────────┐
│                   wellnown-env                          │
│                                                         │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐ │
│  │ Via UI      │───▶│ NATS KV     │───▶│ Via UI      │ │
│  │ (editor)    │    │ VIA_THEME=  │    │ (consumer)  │ │
│  └─────────────┘    └─────────────┘    └─────────────┘ │
│        │                  │                   │        │
│        └──────────────────┴───────────────────┘        │
│                    Same system!                        │
└─────────────────────────────────────────────────────────┘
```

**The meta-beauty**:
- Via is a wellnown-env service
- Via's config (theme, etc) is stored in NATS
- Via's UI edits that config
- Via receives those edits in real-time
- Turtles all the way down

This proves the architecture works - if we can't manage ourselves, why should anyone trust us to manage their services?

---

## Architecture Overview

**Three embedded components work together:**

| Component | Role |
|-----------|------|
| **NATS embedded** | Services register themselves, discover each other, share config |
| **Process-compose embedded** | Manage binaries/processes |
| **Via** | Route external traffic to discovered services, reactive UI |

---

## Why Via + SSE + NATS?

Via is for routing/proxying - it's how you expose services discovered via NATS to external clients. It completes the picture.

When NATS tells you "service X is at host:port", Via can dynamically route traffic to it. Via uses **Server-Sent Events (SSE)** for real-time reactivity - no JavaScript polling needed.

**The key insight**: Via's SSE + NATS pub/sub = real-time config propagation to all connected clients.

---

## Real-Time Theme Switching (Example of the Pattern)

The themes page isn't just a display - it demonstrates the core wellnown-env concept:

```
User clicks "Purple" in GUI
       ↓
Via action writes to NATS KV: VIA_THEME=purple
       ↓
All connected services receive the update (real-time via NATS)
       ↓
Via's SSE pushes changes to all browser tabs
       ↓
UI updates everywhere simultaneously
```

**Why this matters**: Any env var change flows through the system in real-time. Theme is just a visible example - the same pattern works for:
- Feature flags
- Log levels
- Service endpoints
- Rate limits
- Any runtime configuration

---

## Session Storage in NATS

**Future capability**: Store Via session cookies inside NATS.

Benefits:
- **HA sessions** - User session available everywhere in real-time
- **Multi-tab sync** - SSE updates push to any browser tab
- **Horizontal scaling** - Any Via instance can serve any user
- **No sticky sessions** - Load balancer can route anywhere

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Browser 1  │     │  Browser 2  │     │  Browser 3  │
│  (Tab A)    │     │  (Tab B)    │     │  (Mobile)   │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └─────────────┬─────┴─────────────┬─────┘
                     │                   │
              ┌──────▼──────┐     ┌──────▼──────┐
              │   Via #1    │     │   Via #2    │
              └──────┬──────┘     └──────┬──────┘
                     │                   │
                     └─────────┬─────────┘
                               │
                        ┌──────▼──────┐
                        │  NATS KV    │
                        │  (sessions) │
                        └─────────────┘
```

All instances share session state. User changes theme on phone → desktop tabs update instantly.

---

## Why No JavaScript?

Via uses SSE (Server-Sent Events) for reactivity:
- Pure Go on the server
- Browser's native EventSource API
- No build step, no transpilation, no node_modules
- Type-safe UI composition with `github.com/go-via/via/h`

The "JavaScript" you see is just Datastar's tiny runtime (~14KB) that Via injects - it handles SSE connection and DOM morphing. You never write or maintain JS.

---

## Environment-Based Theme Selection

Themes are configured via `VIA_THEME` env var:

```bash
VIA_THEME=purple go run main.go
VIA_THEME=amber go run main.go
```

Available themes (19 total):
Amber, Blue, Cyan, Fuchsia, Green, Grey, Indigo (default), Jade, Lime, Orange, Pink, Pumpkin, Purple, Red, Sand, Slate, Violet, Yellow, Zinc

**Why env var instead of config file?**
- Follows wellnown-env philosophy: env vars are the interface
- Works with any runner (Docker, k8s, systemd, bare metal)
- Can be dynamically updated via NATS in production

---

## PicoCSS Color Classes

With `IncludeColors: true` in picocss options:

```go
picocss.WithOptions(picocss.Options{
    Theme:         picocss.ThemeIndigo,
    IncludeColors: true,  // Enables color utility classes
})
```

You get:
- `pico-background-{color}` - Background colors
- `pico-color-{color}` - Text colors

These work regardless of which theme is active - useful for color-coded status indicators, badges, etc.
