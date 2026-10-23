import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import { webcrypto } from 'node:crypto';
import { TextEncoder, TextDecoder } from 'node:util';
import { JSDOM } from 'jsdom';
import { resolveLanguage } from '../internal/web/static/model.js';

const read = p => fs.readFileSync(new URL(p, import.meta.url), 'utf8');
const tick = () => new Promise(resolve => setImmediate(resolve));
const settle = async () => { for (let i = 0; i < 30; i++) await tick(); };
const proxy = read('../engine/secureclaim/web/usdt.html').match(/PROXY_CREATION_CODE='(0x[0-9a-f]+)'/)[1];
const doms = [];
test.after(() => doms.forEach(d => d.window.close()));

// Wire's page with PhoenixMail's own payment window (/pm/*) and a fake Wire server.
async function start({ session = { verified: true, email: 'sender@example.test' }, state = () => ({ state: 'awaiting_funds', balanceUnits: 0 }), invite = () => ({ invited: true }) } = {}) {
  const dom = new JSDOM(read('../internal/web/static/index.html'), { url: 'https://wire.example.test/', runScripts: 'outside-only', pretendToBeVisual: true });
  doms.push(dom);
  const { window } = dom, ctx = dom.getInternalVMContext();
  Object.defineProperty(window, 'crypto', { value: webcrypto, configurable: true });
  Object.assign(window, { TextEncoder, TextDecoder, open: () => null });
  for (const f of ['../engine/secureclaim/web/cashlink.js', '../internal/web/static/pm/usdt-cashlink.js', '../internal/web/static/pm/usdt-payments.js', '../internal/web/static/pm/nimiq-payments.js']) vm.runInContext(read(f), ctx);
  const C = window.PhoenixCashlink, calls = [];
  const answer = (status, body) => ({ ok: status < 300, status, json: async () => body });
  window.fetch = async (path, options = {}) => {
    const body = options.body ? JSON.parse(options.body) : undefined;
    calls.push([path, body]);
    if (path === '/api/config') return answer(200, { mode: 'integrated', payments_enabled: true, assets: ['USDC_POLYGON', 'USDT_POLYGON', 'USDT_ETHEREUM'] });
    if (path === '/api/session') return answer(200, session);
    if (path === '/api/usdt/status') return answer(200, { requests: false, send: true, protected: true, cashlink: true, gas: false, assets: ['usdc_polygon'], max_send: 1000 });
    if (path === '/api/usdt/cashlink/gas-status') return answer(200, { available: false, reason: 'off' });
    if (path === '/api/usdt/cashlink/create') {
      C.setProxyCreationCode(proxy);
      const owners = [body.linkOwner, C.checksum(C.addressOfKey(webcrypto.getRandomValues(new Uint8Array(32))))];
      const safe = C.checksum(C.cardAddress({ id: C.unhex(body.cardId), owners, threshold: 2 }, 137));
      const token = '0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359';
      return answer(200, { card: { id: body.cardId, asset: body.asset, amount: body.amount, units: 2500000, feeUnits: 150000, refundFeeUnits: 100000, chainId: 137, safe, owners, token, relayer: '0x' + '33'.repeat(20) },
        pay: { recipient: safe, units: 2650000, amount: '2.65', link: 'ethereum:' + token + '@137/transfer?address=' + safe + '&uint256=2650000', qr: 'iVBORw0KGgo=' }, claimBase: 'https://wire.example.test/usdt.html', senderToken: body.cardId });
    }
    if (path === '/api/usdt/cashlink/status') return answer(200, state(body));
    if (path.endsWith('/invite')) return answer(200, invite(body));
    if (path === '/api/sender/request' || path === '/api/sender/verify' || path === '/api/logout') return answer(200, { email: 'sender@example.test', ok: true });
    throw Error('unexpected API ' + path);
  };
  const source = read('../internal/web/static/wire-pay.js').replace(/^import[^\n]+\n/gm, '');
  Object.assign(ctx, { resolveLanguage });
  vm.runInContext('{' + source + '\n}', ctx);
  await settle();
  return { window, $: id => window.document.getElementById(id), calls, doc: window.document };
}

async function makeCard(t) {
  t.$('pay').click(); await settle();
  const form = t.doc.querySelector('.nimiq-payment-panel');
  const input = label => [...form.querySelectorAll('input')].find(i => i.closest('label') && i.closest('label').textContent.includes(label));
  const set = (el, v) => { el.value = v; el.dispatchEvent(new t.window.Event('input', { bubbles: true })); el.dispatchEvent(new t.window.Event('change', { bubbles: true })); };
  set(input('Amount'), '2.5'); set(input('Message'), 'merci'); set(input('Recipient'), 'friend@example.test');
  [...form.querySelectorAll('button')].find(b => b.textContent === 'Create card').click();
  await settle();
}

test('a verified sender gets PhoenixMail’s window: USDC / USDT only, protected cards only, Wire on the card', async () => {
  const t = await start();
  assert.equal(t.$('wire-ready').hidden, false); assert.equal(t.$('wire-sender').hidden, true);
  assert.equal(t.$('wire-who').textContent, 'sender@example.test');
  t.$('pay').click(); await settle();
  assert.ok(t.doc.querySelector('.nimiq-payment-window'));
  const css = read('../internal/web/static/wire-pay.css');
  assert.match(css, /nimiq-payment-assets button:first-child/, 'NIM and payment requests are hidden until they move into this window');
  assert.match(t.doc.querySelector('.nimiq-payment-panel').textContent, /Protected/);
  await makeCard(t);
  const create = t.calls.find(c => c[0] === '/api/usdt/cashlink/create');
  assert.ok(create, 'the card is made through PhoenixMail’s route');
  assert.equal(create[1].asset, 'usdc_polygon'); assert.match(create[1].cardId, /^[0-9a-f]{32}$/); assert.equal(create[1].recipient, 'friend@example.test');
  assert.ok(!JSON.stringify(create[1]).includes('#c='), 'the link key never goes to the server at creation');
  const card = t.doc.querySelector('.nimiq-payment-preview .nimiq-payment-card');
  assert.ok(card, 'the card is shown');
  assert.equal(card.querySelector('.nimiq-card-name').textContent, 'PhoenixWire');
  assert.match(card.querySelector('.nimiq-card-brand img').src, /phoenix-emblem\.webp$/);
  assert.ok(card.querySelector('img.nimiq-payment-qr'), 'the QR to fund it is on the card');
  assert.match(card.textContent, /Phoenix Wire does not hold it/);
  assert.ok(!/PhoenixMail/.test(card.textContent));
  // the link key stays in this browser, to be used for the invitation and for taking the money back
  const stored = JSON.parse(t.window.localStorage.getItem('wire-usdt-cards'));
  assert.equal(stored.length, 1); assert.match(stored[0].fragment, /^#c=[A-Za-z0-9_-]+\.[A-Za-z0-9_-]{43}$/); assert.equal(stored[0].invited, false);
});

test('once the card is funded the recipient is invited, once, with this card’s own link', async () => {
  const t = await start({ state: () => ({ state: 'ready', balanceUnits: 2650000 }) });
  await makeCard(t);
  const entry = JSON.parse(t.window.localStorage.getItem('wire-usdt-cards'))[0];
  const realSetTimeout = t.window.setTimeout;
  await new Promise(resolve => { const check = () => { if (t.calls.some(c => c[0].endsWith('/invite'))) resolve(); else realSetTimeout(check, 50); }; check(); setTimeout(resolve, 9000); });
  const invite = t.calls.filter(c => c[0].endsWith('/invite'));
  assert.equal(invite.length, 1);
  assert.equal(invite[0][0], '/api/usdt/card/' + entry.id + '/invite');
  assert.equal(invite[0][1].claim_url, 'https://wire.example.test/usdt.html?lang=en' + entry.fragment);
  await settle();
  assert.equal(JSON.parse(t.window.localStorage.getItem('wire-usdt-cards'))[0].invited, true);
});

test('without a session the sender is asked for an e-mail code, and nothing is offered before', async () => {
  const t = await start({ session: { verified: false } });
  assert.equal(t.$('wire-sender').hidden, false); assert.equal(t.$('wire-ready').hidden, true);
  assert.ok(!t.doc.querySelector('.nimiq-payment-window'));
  assert.ok(!t.calls.some(c => c[0] === '/api/usdt/status'));
});

test('the two 24-hour notices render in all five languages before and after sender confirmation', async () => {
  const t = await start({ session: { verified: false } });
  const expected = {
    fr: ['Vous confirmez votre courriel une seule fois : il reste confirmé 24 heures sur cet appareil.', 'Confirmé pour 24 h.'],
    en: ['You confirm your email once. It stays confirmed for 24 hours on this device.', 'Confirmed for 24 hours.'],
    es: ['Confirmas tu correo una sola vez: queda confirmado 24 horas en este dispositivo.', 'Confirmado por 24 horas.'],
    de: ['Du bestätigst deine E-Mail nur einmal: Sie bleibt 24 Stunden auf diesem Gerät bestätigt.', '24 Stunden bestätigt.'],
    zh: ['你只需确认一次邮箱：在此设备上 24 小时内保持已确认。', '已确认 24 小时。'],
  };
  for (const [lang, [help, confirmed]] of Object.entries(expected)) {
    t.$('wire-language').value = lang;
    t.$('wire-language').dispatchEvent(new t.window.Event('change'));
    assert.equal(t.$('wire-session-help').textContent, help);
    assert.equal(t.$('wire-session-confirmed').textContent, confirmed);
  }
  assert.equal(t.$('wire-email').getAttribute('aria-describedby'), 'wire-session-help');
  assert.ok(t.$('wire-email').closest('label').compareDocumentPosition(t.$('wire-session-help')) & t.window.Node.DOCUMENT_POSITION_FOLLOWING);
  assert.ok(t.$('wire-session-help').compareDocumentPosition(t.$('wire-send-code')) & t.window.Node.DOCUMENT_POSITION_FOLLOWING);
  t.$('wire-code').value = '123456'; t.$('wire-verify').click(); await settle();
  assert.equal(t.$('wire-ready').hidden, false);
  assert.equal(t.$('wire-session-confirmed').closest('.wire-who'), t.$('wire-who').closest('.wire-who'));
  assert.equal(t.$('wire-session-confirmed').textContent, expected.zh[1]);
});

test('return controls work on direct entry and after replacing the payment window', async () => {
  const t = await start();
  const links = [...t.doc.querySelectorAll('.wire-navigation a')].map(a => a.href);
  assert.deepEqual(links, ['https://phoenixmail.ca/#phoenixwire', 'https://phoenixmail.ca/#phoenixmail']);
  t.$('pay').click(); await settle();
  const before = t.doc.querySelector('.nimiq-payment-window');
  const assets = [...before.querySelectorAll('.nimiq-payment-assets button')];
  assert.ok(assets.length > 2);
  assets[assets.length - 1].click(); await settle();
  const after = t.doc.querySelector('.nimiq-payment-window');
  assert.notEqual(after, before);
  const back = after.querySelector('.wire-window-back');
  assert.ok(back); back.click(); await settle();
  assert.equal(t.doc.querySelector('.nimiq-payment-modal'), null);
  assert.equal(t.doc.activeElement, t.$('pay'));
  assert.ok(!t.calls.some(([path]) => /create|cancel|refund/.test(path)), 'navigation never changes a card or sends a payment');
  t.$('pay').click(); await settle();
  assert.ok(t.doc.querySelector('.wire-window-back'), 'the window can be reopened');
});

test('both sender pages declare the approved shortcut icons and PhoenixWire tab name', () => {
  for (const file of ['index.html', 'classic.html']) {
    const doc = new JSDOM(read('../internal/web/static/' + file)).window.document;
    assert.equal(doc.title, 'Phoenix Wire');
    assert.equal(doc.querySelector('meta[name="apple-mobile-web-app-title"]').content, 'Phoenix Wire');
    assert.equal(doc.querySelector('link[rel="apple-touch-icon"]').getAttribute('href'), '/brand/wire-icon-180.png?v=0.4.1');
    assert.ok(doc.querySelector('link[rel="icon"]'));
    assert.equal(doc.querySelector('link[rel="manifest"]').getAttribute('href'), '/site.webmanifest?v=0.4.1');
    assert.ok(doc.querySelector('.wire-navigation a'));
    doc.defaultView.close();
  }
  const manifest = JSON.parse(read('../internal/web/static/site.webmanifest'));
  assert.equal(manifest.name, 'Phoenix Wire'); assert.equal(manifest.start_url, '/');
  for (const size of [192, 512]) assert.ok(manifest.icons.some(icon => icon.sizes === size + 'x' + size));
});
