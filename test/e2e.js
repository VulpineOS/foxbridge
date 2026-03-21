// End-to-end test: connect to foxbridge via Puppeteer
// Usage: node test/e2e.js
// Requires: foxbridge running on port 9222

const puppeteer = require('puppeteer-core');

async function main() {
  console.log('Connecting to foxbridge at ws://127.0.0.1:9222...');

  try {
    const browser = await puppeteer.connect({
      browserWSEndpoint: 'ws://127.0.0.1:9222/devtools/browser/foxbridge',
      defaultViewport: null,
    });
    console.log('✓ Connected to browser');

    const page = await browser.newPage();
    console.log('✓ New page created');

    await page.goto('https://example.com', { waitUntil: 'load', timeout: 30000 });
    console.log('✓ Navigated to example.com');

    const title = await page.title();
    console.log('✓ Page title:', title);

    const h1 = await page.$eval('h1', el => el.textContent);
    console.log('✓ H1 text:', h1);

    await page.screenshot({ path: '/tmp/foxbridge-test.png' });
    console.log('✓ Screenshot saved to /tmp/foxbridge-test.png');

    await page.close();
    console.log('✓ Page closed');

    await browser.disconnect();
    console.log('✓ Disconnected');

    console.log('\n🎉 All tests passed!');
  } catch (err) {
    console.error('✗ Test failed:', err.message);
    process.exit(1);
  }
}

main();
