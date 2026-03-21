// Direct WebSocket test — bypasses Puppeteer's abstractions
const WebSocket = require('ws');

const ws = new WebSocket('ws://127.0.0.1:9222/devtools/browser/foxbridge');
let nextId = 1;

function send(method, params = {}) {
  const msg = {id: nextId++, method, params};
  console.log(`→ ${method}`, JSON.stringify(params).substring(0, 100));
  ws.send(JSON.stringify(msg));
}

ws.on('open', () => {
  console.log('Connected\n');

  // Step 1: Get browser contexts
  send('Target.getBrowserContexts');
});

ws.on('message', (data) => {
  const msg = JSON.parse(data.toString());

  if (msg.id) {
    // Response
    console.log(`← Response #${msg.id}:`, JSON.stringify(msg.result || msg.error).substring(0, 200));

    if (msg.id === 1) {
      // Step 2: Set discover targets
      send('Target.setDiscoverTargets', {discover: true});
    } else if (msg.id === 2) {
      // Step 3: Create a new target
      send('Target.createTarget', {url: 'about:blank'});
    }
  } else if (msg.method) {
    // Event
    console.log(`← Event: ${msg.method}`, JSON.stringify(msg.params).substring(0, 200));

    if (msg.method === 'Target.attachedToTarget') {
      const sessionId = msg.params.sessionId;
      console.log(`\n✓ Got session: ${sessionId}`);

      // Step 4: Navigate using the session
      const navMsg = {id: nextId++, method: 'Page.navigate', params: {url: 'https://example.com'}, sessionId};
      console.log(`→ Page.navigate (session=${sessionId})`);
      ws.send(JSON.stringify(navMsg));
    }

    if (msg.method === 'Page.loadEventFired') {
      console.log('\n🎉 Page loaded! foxbridge is working!');
      process.exit(0);
    }
  }
});

ws.on('error', (err) => {
  console.error('WebSocket error:', err.message);
  process.exit(1);
});

setTimeout(() => {
  console.log('\nTimeout — no load event after 15s');
  process.exit(1);
}, 15000);
