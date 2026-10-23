import { nimFunded,unpaidCard,recoverableCard,historyEntries,historyPending,senderHistoryText,nimHistoryLabel,cancelUnpaidCard } from '../internal/web/static/model.js';
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import { webcrypto } from 'node:crypto';
import { TextEncoder, TextDecoder } from 'node:util';
import { JSDOM } from 'jsdom';
import { flowText } from '../internal/web/static/flow-text.js';
import { addressValid, cardPresentation, paintCard } from '../internal/web/static/model.js';
import { usdtText } from '../internal/web/static/usdt-text.js';

const read = p => fs.readFileSync(new URL(p, import.meta.url), 'utf8');
const tick = () => new Promise(resolve => setImmediate(resolve));
async function settle() { for (let i = 0; i < 25; i++) await tick(); }
const claimPage = read('../engine/secureclaim/web/usdt.html');
const PROXY = claimPage.match(/PROXY_CREATION_CODE='(0x[0-9a-f]+)'/)[1];

const doms = [];
test.after(() => doms.forEach(d => d.window.close()));
async function start({ answer = x => x, status = () => ({ state: 'awaiting_funds', balance_units: 0 }), invite = () => ({ invited: true }), refundAnswer } = {}) {
  const dom = new JSDOM(read('../internal/web/static/classic.html'), { url: 'https://wire.example.test/', runScripts: 'outside-only' });
  const { window } = dom;
  doms.push(dom);
  window.open = () => null;
  window.document.documentElement.lang = 'en';
  Object.defineProperty(window, 'crypto', { value: webcrypto, configurable: true });
  Object.assign(window, { TextEncoder, TextDecoder, Uint8Array, BigInt });
  const ctx = dom.getInternalVMContext();
  vm.runInContext(read('../engine/secureclaim/web/cashlink.js'), ctx);
  window.PHOENIX_PROXY_CREATION_CODE = PROXY;
  window.qrcode = (type, level) => ({ addData() {}, make() {}, createSvgTag: () => '<svg data-test="qr"></svg>' });
  const C = window.PhoenixCashlink;
  const calls = [];
  const response = (status, body) => ({ ok: status >= 200 && status < 300, status, json: async () => body });
  let made = null;
  window.fetch = async (path, options = {}) => {
    const body = options.body ? JSON.parse(options.body) : undefined;
    calls.push([path, body]);
    if (path === '/api/config') return response(200, { mode: 'integrated', payments_enabled: true, assets: ['NIM', 'USDC_POLYGON', 'USDT_POLYGON', 'USDT_ETHEREUM'] });
    if (path === '/api/session') return response(200, { verified: true, email: 'sender@example.test' });
    if (path === '/api/payments') return response(200, { payments: [] });
    if (path === '/api/usdt/cards') return response(200, { cards: [] });
    if (path === '/api/usdt/create') {
      C.setProxyCreationCode(PROXY);
      const cos = C.addressOfKey(webcrypto.getRandomValues(new Uint8Array(32)));
      const owners = [body.link_owner, C.checksum(cos)];
      const idBytes = C.unhex(body.card_id);
      const safe = C.checksum(C.cardAddress({ id: idBytes, owners, threshold: 2 }, 137));
      made = { card: { id: body.card_id, asset: body.asset, chain: 'polygon', symbol: 'USDC', amount: body.amount, units: 2500000, feeUnits: 150000, refundFeeUnits: 100000, chainId: 137, safe, owners, token: '0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359', relayer: '0x' + '33'.repeat(20) },
        pay: { recipient: safe, units: 2650000, amount: '2.65', link: 'ethereum:0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359@137/transfer?address=' + safe + '&uint256=2650000' } };
      return response(200, answer(made));
    }
    if (path.endsWith('/status')) return response(200, status(path));
    if (path.endsWith('/invite')) return response(200, invite(body));
    if (path.endsWith('/refund')) return response(200, refundAnswer ? refundAnswer(body) : {});
    if (path.endsWith('/refund-claim')) return response(200, { status: 'refunded' });
    throw Error('unexpected API ' + path);
  };
  window.confirm = () => true;
  Object.assign(ctx, { nimFunded,unpaidCard:p=>unpaidCard(p,window.Date.now()),recoverableCard,historyEntries:entries=>historyEntries(entries,window.Date.now()),historyPending:p=>historyPending(p,window.Date.now()),senderHistoryText,nimHistoryLabel:(p,language)=>nimHistoryLabel(p,language,window.Date.now()),cancelUnpaidCard,  flowText, addressValid, cardPresentation, paintCard, usdtText });
  const strip = s => '{' + s.replace(/^import[^\n]+\n/gm, '').replace(/^export /gm, '') + '\n}';
  vm.runInContext(strip(read('../internal/web/static/flow.js')), ctx);
  vm.runInContext(strip(read('../internal/web/static/usdt-flow.js')), ctx);
  await settle();
  const $ = id => window.document.getElementById(id);
  const submit = async () => { $('card-form').dispatchEvent(new window.Event('submit', { cancelable: true })); await settle(); };
  return { dom, window, $, calls, submit, C, get made() { return made; } };
}

function fillForm(t, currency = 'USDC_POLYGON') {
  t.$('currency').value = currency; t.$('amount').value = '2.5'; t.$('recipient').value = 'friend@example.test'; t.$('message').value = 'test';
}

test('a USDC card: made in the browser, checked against the service, paid by QR, invited once funded', async () => {
  const t = await start();
  fillForm(t);
  await t.submit();
  const create = t.calls.find(c => c[0] === '/api/usdt/create');
  assert.ok(create, 'the card is created');
  assert.match(create[1].card_id, /^[0-9a-f]{32}$/);
  assert.match(create[1].link_owner, /^0x[0-9a-fA-F]{40}$/);
  assert.equal(create[1].asset, 'usdc_polygon');
  assert.ok(!JSON.stringify(create[1]).includes('#c='), 'the link key never goes to the service at creation');
  assert.equal(t.$('usdt-panel').hidden, false);
  assert.equal(t.$('usdt-title').textContent, 'Pay 2.65 USDC on Polygon');
  // the funding step is the card itself, in the colour its e-mail will have, with the QR inside it
  assert.equal(t.$('payment-card').dataset.mode, 'fund');
  assert.ok(t.$('payment-card').contains(t.$('usdt-panel')));
  assert.equal(t.$('card-amount').textContent, '2.65');
  assert.equal(t.$('card-unit').textContent, 'USDC');
  assert.equal(t.$('usdt-address').textContent, t.made.card.safe);
  assert.ok(t.$('usdt-wallet').href.startsWith('ethereum:0x'));
  assert.ok(t.$('usdt-qr').innerHTML.includes('<svg'));
  // the key stays in this browser
  const stored = JSON.parse(t.window.localStorage.getItem('wire-usdt-cards'));
  assert.equal(stored.length, 1);
  assert.match(stored[0].fragment, /^#c=[A-Za-z0-9_-]+\.[A-Za-z0-9_-]{43}$/);
  // not funded: nothing is e-mailed
  const entry = stored[0];
  assert.equal(await t.window.WireUSDT.check(entry), false);
  assert.ok(!t.calls.some(c => c[0].endsWith('/invite')));
  // the money arrives: the claim link goes to the service once, with this card's own key
  const t2 = await start({ status: () => ({ state: 'ready', balance_units: 2650000 }) });
  fillForm(t2);
  await t2.submit();
  const entry2 = JSON.parse(t2.window.localStorage.getItem('wire-usdt-cards'))[0];
  assert.equal(await t2.window.WireUSDT.check(entry2), true);
  const invite = t2.calls.find(c => c[0].endsWith('/invite'));
  assert.ok(invite[1].claim_url.startsWith('https://wire.example.test/usdt.html?lang=en#c='));
  assert.ok(invite[1].claim_url.endsWith(entry2.fragment.slice(1)) || invite[1].claim_url.includes(entry2.fragment));
  assert.equal(JSON.parse(t2.window.localStorage.getItem('wire-usdt-cards'))[0].invited, true);
});

test('a service answer that does not match the card is never shown for payment', async () => {
  for (const [name, answer] of Object.entries({
    'another safe address': m => ({ ...m, card: { ...m.card, safe: '0x' + '11'.repeat(20) }, pay: { ...m.pay, recipient: '0x' + '11'.repeat(20) } }),
    'a link key that is not an owner': m => ({ ...m, card: { ...m.card, owners: ['0x' + '22'.repeat(20), '0x' + '33'.repeat(20)] } }),
    'another token': m => ({ ...m, card: { ...m.card, token: '0x' + '44'.repeat(20) } }),
    'a pay amount that is not the card plus its fee': m => ({ ...m, pay: { ...m.pay, units: 99999999 } }),
    'a fee for the refund above the card fee': m => ({ ...m, card: { ...m.card, refundFeeUnits: 999999 } }),
    'a wallet link to somebody else': m => ({ ...m, pay: { ...m.pay, link: 'ethereum:0x' + '55'.repeat(20) + '@137/transfer?address=0x' + '66'.repeat(20) + '&uint256=1' } }),
  })) {
    const t = await start({ answer });
    fillForm(t);
    await t.submit();
    assert.equal(t.$('usdt-panel').hidden, true, name + ': the payment panel stays closed');
    assert.equal(t.$('feedback').textContent, usdtText.en.errors.invalid_service_answer, name);
    assert.equal(t.window.localStorage.getItem('wire-usdt-cards'), null, name + ': no card is kept');
  }
});

test('bad inputs never reach the service', async () => {
  for (const [name, mutate] of Object.entries({
    'amount too small': () => '0.5', 'amount too large': () => '1001', 'too many decimals': () => '1.1234567', 'not a number': () => 'abc',
  })) {
    const t = await start();
    fillForm(t);
    t.$('amount').value = mutate();
    await t.submit();
    assert.ok(!t.calls.some(c => c[0] === '/api/usdt/create'), name);
    assert.equal(t.$('feedback').textContent, usdtText.en.invalid);
  }
  const t = await start();
  fillForm(t);
  t.$('recipient').value = '';
  await t.submit();
  assert.ok(!t.calls.some(c => c[0] === '/api/usdt/create'));
});

test('NIM still goes through its own flow, USDC cards do not touch it', async () => {
  const t = await start();
  fillForm(t, 'NIM');
  await t.submit();
  assert.ok(!t.calls.some(c => c[0] === '/api/usdt/create'));
});

test('taking the money back: only the flat gas fee fixed at creation is accepted, and the sender sees the net and the fee', async () => {
  const sigHex = '0x' + '11'.repeat(64) + '1b', owner = '0x' + '44'.repeat(20), to = '0x' + '55'.repeat(20);
  let asked = null;
  for (const [fee, accepted] of [[100000, true], [30000000, false], [undefined, false], [100001, false]]) {
    const t = await start({ refundAnswer: () => ({ owner, signature: sigHex, units: 2550000, nonce: 0, ...(fee === undefined ? {} : { fee_units: fee }) }) });
    fillForm(t);
    await t.submit();
    const entry = JSON.parse(t.window.localStorage.getItem('wire-usdt-cards'))[0];
    t.window.confirm = m => { asked = m; return true; };
    if (accepted) {
      await t.window.WireUSDT.refund(entry, to);
      assert.ok(t.calls.some(c => c[0].endsWith('/refund-claim')), 'the honest refund goes through');
      assert.match(asked, /2\.55 USDC/); assert.match(asked, /0\.1 USDC/);
    } else {
      await assert.rejects(() => t.window.WireUSDT.refund(entry, to), /invalid_service_answer/);
      assert.ok(!t.calls.some(c => c[0].endsWith('/refund-claim')), 'nothing is signed for fee ' + fee);
    }
  }
  // the sender can say no
  const t = await start({ refundAnswer: () => ({ owner, signature: sigHex, units: 2550000, nonce: 0, fee_units: 100000 }) });
  fillForm(t);
  await t.submit();
  const entry = JSON.parse(t.window.localStorage.getItem('wire-usdt-cards'))[0];
  t.window.confirm = () => false;
  await assert.rejects(() => t.window.WireUSDT.refund(entry, to), /cancelled/);
  assert.ok(!t.calls.some(c => c[0].endsWith('/refund-claim')));
});

test('the five languages all have every word', () => {
  const keys = Object.keys(usdtText.en), errorKeys = Object.keys(usdtText.en.errors);
  for (const lang of ['fr', 'es', 'de', 'zh']) {
    assert.deepEqual(Object.keys(usdtText[lang]), keys, lang);
    assert.deepEqual(Object.keys(usdtText[lang].errors), errorKeys, lang);
    for (const k of keys) if (k !== 'errors') assert.ok(String(usdtText[lang][k]).length > 0, lang + ' ' + k);
    for (const k of ['payTitle', 'payHelp', 'invited', 'refundConfirm']) assert.deepEqual((usdtText[lang][k].match(/\{\w+\}/g) || []).sort(), (usdtText.en[k].match(/\{\w+\}/g) || []).sort(), lang + ' ' + k + ' keeps its placeholders');
  }
});

