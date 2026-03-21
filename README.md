# foxbridge 🦊🌉

**CDP-to-Firefox protocol proxy.** Use any Chrome DevTools Protocol tool with Firefox-based browsers like Camoufox.

foxbridge translates Chrome DevTools Protocol (CDP) commands into Firefox's Juggler protocol, letting CDP-only tools (Puppeteer, OpenClaw, browser-use, etc.) control Firefox without modification.

## Why?

- **Anti-detect browsers** like [Camoufox](https://github.com/AIO-Camoufox/camoufox) are Firefox-based but most automation tools only speak CDP (Chrome's protocol)
- **No existing tool** bridges CDP → Firefox's Juggler protocol
- **WebDriver BiDi** is the future standard but not yet mature enough for production automation
- foxbridge solves this today with Juggler, with a modular architecture ready for BiDi

## Quick Start

```bash
# Build
go build -o foxbridge ./cmd/foxbridge/

# Run with Camoufox
./foxbridge --binary /path/to/camoufox --headless --port 9222

# Now any CDP tool can connect:
# Puppeteer: puppeteer.connect({browserWSEndpoint: 'ws://localhost:9222'})
# OpenClaw: set browser.profiles.vulpine.cdpUrl = "ws://localhost:9222"
```

## Architecture

```
CDP Client (Puppeteer, OpenClaw, etc.)
  │
  ▼ WebSocket (ws://localhost:9222)
foxbridge
  ├── CDP Server (WebSocket + HTTP discovery)
  ├── Bridge Layer (method translation)
  └── Backend Interface
       ├── Juggler Backend (pipe FD 3/4) ← current
       └── BiDi Backend (WebSocket)      ← future
  │
  ▼ Pipe transport
Firefox / Camoufox
```

## CDP Domain Coverage

| Domain | Status | Notes |
|--------|--------|-------|
| Target | ✅ | createTarget, closeTarget, contexts |
| Page | ✅ | navigate, reload, screenshot, lifecycle |
| Runtime | ✅ | evaluate, callFunction, properties |
| Input | ✅ | mouse, keyboard, touch, text |
| Network | 🔨 | cookies, interception, headers |
| Emulation | 🔨 | geo, locale, timezone, UA |
| Accessibility | 🔨 | AX tree |
| DOM | 🔨 | via Runtime.evaluate shims |
| Debugger | ⬜ | stub (not needed for agents) |

## HTTP Discovery

```bash
curl http://localhost:9222/json/version
curl http://localhost:9222/json/list
```

## License

MIT
