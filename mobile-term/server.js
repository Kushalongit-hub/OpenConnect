const express = require('express');
const http = require('http');
const socketIo = require('socket.io');
const pty = require('node-pty');
const os = require('os');
const { WebSocketServer } = require('ws');
const net = require('net');

const app = express();
const server = http.createServer(app);
const io = new socketIo.Server(server);

app.use(express.static('public'));
app.use(express.json());

function detectShell() {
  if (process.env.SHELL) return process.env.SHELL;
  if (process.env.COMSPEC) return process.env.COMSPEC;
  if (os.platform() === 'win32') return 'powershell.exe';
  if (os.platform() === 'darwin') return '/bin/bash';
  return '/bin/sh';
}

const shell = detectShell();
console.log(`Using shell: ${shell}`);

const ptyProcess = pty.spawn(shell, [], {
  name: 'xterm-color',
  cols: 80,
  rows: 30,
  cwd: 'C:\\',
  env: process.env,
});

io.on('connection', (socket) => {
  ptyProcess.onData((data) => {
    socket.emit('output', data);
  });

  socket.on('input', (data) => {
    ptyProcess.write(data);
  });

  socket.on('resize', (size) => {
    ptyProcess.resize(size.cols, size.rows);
  });

  socket.on('bridge-input', (data) => {
    io.emit('bridge-input', data);
  });
});

app.get('/api/sunshine-status', (req, res) => {
  const socket = new net.Socket();
  const timeout = 800;
  let open = false;

  const timer = setTimeout(() => {
    socket.destroy();
    res.json({ running: false });
  }, timeout);

  socket.connect(47990, '127.0.0.1', () => {
    open = true;
    clearTimeout(timer);
    socket.end();
    res.json({ running: true });
  });

  socket.on('error', () => {
    if (!open) {
      clearTimeout(timer);
      res.json({ running: false });
    }
  });
});

let bridgeWs = null;
const wss = new WebSocketServer({ server, path: '/bridge' });

wss.on('connection', (ws) => {
  console.log('Native bridge connected');
  bridgeWs = ws;

  ws.on('message', (raw) => {
    let msg;
    try {
      msg = JSON.parse(raw.toString());
    } catch {
      return;
    }

    if (msg.type === 'input' && msg.payload) {
      io.emit('bridge-input', JSON.parse(msg.payload));
    } else if (msg.type === 'keyboard' && msg.payload) {
      io.emit('bridge-keyboard', JSON.parse(msg.payload));
    }
  });

  ws.on('close', () => {
    console.log('Native bridge disconnected');
    bridgeWs = null;
  });
});

io.on('bridge-input', (data) => {
  if (bridgeWs && bridgeWs.readyState === 1) {
    if (data && data.type === 'keyboard') {
      bridgeWs.send(JSON.stringify({ type: 'keyboard', payload: JSON.stringify(data) }));
    } else {
      bridgeWs.send(JSON.stringify({ type: 'input', payload: JSON.stringify(data) }));
    }
  }
});

io.on('bridge-keyboard', (data) => {
  if (bridgeWs && bridgeWs.readyState === 1) {
    bridgeWs.send(JSON.stringify({ type: 'keyboard', payload: JSON.stringify(data) }));
  }
});

const PORT = 3000;
const HOST = '0.0.0.0';
server.listen(PORT, HOST, () => {
  console.log(`Server running at http://${HOST}:${PORT}`);
});
