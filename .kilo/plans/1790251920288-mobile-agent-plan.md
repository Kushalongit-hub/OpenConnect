# OpenConnect Mobile Agent Plan

## Goal
Build a speed-optimized interactive remote interface in `mobile-term/` that streams the Windows desktop GUI and a live terminal to a Samsung S20 FE over Tailscale, with touch controls like an interactive flat panel.

## Tech Stack
- Node.js + Express
- socket.io
- node-pty
- WebRTC (GUI stream + interactive data channel)
- Windows Graphics Capture native bridge (Go/Rust helper)
- xterm.js + xterm-addon-fit (CDN)

## Directory Structure
```
mobile-term/
  package.json
  server.js
  public/
    index.html
  native-bridge/   (separate small native helper for screen capture + input injection)
```

## Key Decisions
- **Primary client**: Samsung S20 FE (1080x2400, touch-first)
- **Layout**: stacked view — GUI stream on TOP (50-60% height), live terminal on BOTTOM, swipeable/resizable divider
- **Default terminal cwd**: `C:\`
- **GUI capture**: native Windows Graphics Capture API via small bridge executable
- **Streaming**: WebRTC peer connection, browser-native on S20 FE
- **Input**: touch events → WebRTC data channel → native bridge → mouse/keyboard injection
- **Speed**: minimize hops, avoid transcoding, direct frame pipe to WebRTC
- **No auth**: Tailscale trust assumed

## Implementation Steps
1. Create `mobile-term/` and initialize npm.
2. Install `express`, `socket.io`, `node-pty`.
3. Write `server.js`:
   - Express serves `public/` on port 3000.
   - WebRTC signaling endpoint for GUI stream.
   - Single terminal session in `C:\` via `node-pty`.
   - `socket.io` pipes terminal stdin/stdout/resize.
4. Write `public/index.html`:
   - Stacked layout: GUI canvas top, xterm.js bottom.
   - Touch controls: tap-to-click, drag-to-move, two-finger scroll, pinch-to-zoom.
   - WebRTC client for GUI stream + data channel for input.
   - Resizable split via drag handle.
5. Build native bridge (`native-bridge/`):
   - Windows Graphics Capture API to grab desktop frames.
   - Encode frames to VP8/VP9 or raw I420.
   - Feed frames into WebRTC peer connection.
   - Receive touch events via WebRTC data channel, inject as mouse/keyboard.
6. Run `node server.js` + bridge, verify on S20 FE.

## Validation
- Open `http://localhost:3000`, confirm GUI stream renders and terminal works.
- On S20 FE over Tailscale, confirm split view, touch controls, and latency <200ms.

## Out of Scope
- Auth layer.
- Multi-user sessions.
- Stylus support.
- Annotation/whiteboard features.
