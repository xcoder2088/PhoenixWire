import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import { webcrypto } from 'node:crypto';
import { TextEncoder, TextDecoder } from 'node:util';
import { JSDOM } from 'jsdom';
import { flowText } from '../internal/web/static/flow-text.js';
import { addressValid, cardPresentation, paintCard } from '../internal/web/static/model.js';
import { resolveLanguage } from '../internal/web/static/model.js';

const read = p => fs.readFileSync(new URL(p, import.meta.url), 'utf8');
const tick = () => new Promise(resolve => setImmediate(resolve));
const settle = async () => { for (let i = 0; i < 30; i++) await tick(); };
const proxy = read('../engine/secureclaim/web/usdt.html').match(/PROXY_CREATION_CODE='(0x[0-9a-f]+)'/)[1];
const doms = [];
test.after(() => doms.forEach(d => d.window.close()));

// Advance logical time without sleeping, including a phone that suspends its timers.
function fakeClock(window){
  let now=Date.parse('2026-10-23T14:00:00Z'),next=0,hidden=false;
  const jobs=new Map(),RealDate=window.Date;
  window.Date=class extends RealDate{constructor(...args){super(...(args.length?args:[now]));}static now(){return now;}};
  window.setTimeout=(fn,delay=0,...args)=>{const id=++next;jobs.set(id,{at:now+Number(delay),fn:()=>fn(...args)});return id;};
  window.clearTimeout=id=>jobs.delete(id);
  Object.defineProperty(window.document,'hidden',{configurable:true,get:()=>hidden});
  return {now:()=>now,suspend(ms){now+=ms;},hidden(value){hidden=value;window.document.dispatchEvent(new window.Event('visibilitychange'));},
    async advance(ms){const until=now+ms;let count=0;for(;;){const nextJob=[...jobs].filter(([,j])=>j.at<=until).sort((a,b)=>a[1].at-b[1].at)[0];if(!nextJob)break;if(++count>10000)throw Error('timer loop');const[id,job]=nextJob;jobs.delete(id);now=Math.max(now,job.at);job.fn();await settle();}now=until;await settle();},
    pending:()=>jobs.size};
}

// Wire's page with PhoenixMail's own payment window (/pm/*) and a fake Wire server.
async function start({ session = { verified: true, email: 'sender@example.test' }, state = () => ({ state: 'awaiting_funds', balanceUnits: 0 }), invite = () => ({ invited: true }), theme = 'dark', nimPending = false, mobile = false, returning = false, nimOnly = false, nativeMode = '', network = 'mainnet', rawMarker = [0,130,128,146,135], android = false, clocked=false, manualNim=false, nimStage='', saved={}, createdAgo=0,local={},serverCards=[],inviteMode='',ownedUnavailable=false } = {}) {
  const dom = new JSDOM(read('../internal/web/static/index.html'), { url: 'https://wire.example.test/', runScripts: 'outside-only', pretendToBeVisual: true });
  doms.push(dom);
  const { window } = dom, ctx = dom.getInternalVMContext();
  const clock=clocked?fakeClock(window):null;
  for(const[key,value]of Object.entries(saved))window.sessionStorage.setItem(key,value);
  for(const[key,value]of Object.entries(local))window.localStorage.setItem(key,value);
  Object.defineProperty(window, 'crypto', { value: webcrypto, configurable: true });
  Object.assign(window, { TextEncoder, TextDecoder, open: () => ({ closed: false, close() { this.closed = true; } }) });
  window.localStorage.setItem('wire-theme', theme);
  if (mobile || android) Object.defineProperty(window.navigator, 'userAgent', { value: android ? 'Android' : 'iPhone' });
  for (const f of ['../engine/secureclaim/web/cashlink.js', '../internal/web/static/pm/usdt-cashlink.js', '../internal/web/static/pm/usdt-payments.js', '../internal/web/static/pm/nimiq-payments.js']) vm.runInContext(read(f), ctx);
  const C = window.PhoenixCashlink, calls = [], checkouts = [], behaviors = [], nativeCalls = [], confirmations = [];
  let currentSession=session;
  const owned=JSON.parse(JSON.stringify(serverCards));let failStatus=false;
  window.confirm = message => { confirmations.push(message); return true; };
  let fundingVisible = !nativeMode;
  if(nativeMode){
    window.nimiqPay={network,language:'en'};
    if(nativeMode!=='missing')window.nimiq={async sendBasicTransactionWithData(request){
      // Model the native host: String(data), then BufferUtils.fromUtf8(message).
      const encoded = new TextEncoder().encode(String(request.data ?? ''));
      nativeCalls.push({request,hex:Buffer.from(encoded).toString('hex')});
      assert.equal(request.data, '\u0000\u0082\u0080\u0092\u0087');
      assert.equal(Buffer.from(encoded).toString('hex'), '00c282c280c292c287');
      if(nativeMode==='denied')throw {type:'PERMISSION_DENIED',code:4001};
      if(nativeMode==='legacy-denied')return {error:{type:'PERMISSION_DENIED'}};
      fundingVisible=nativeMode==='success'||nativeMode==='error-after-send';
      if(nativeMode==='error'||nativeMode==='error-after-send')throw {type:'NETWORK_ERROR'};
      return 'b'.repeat(64);
    }};
  }
  const nim = { id: 'a'.repeat(48), sender: 'sender@example.test', recipient: 'friend@example.test', amount: '2.5', luna: 250000, message: 'Hi', address: 'NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY', stage: nimStage || (nimPending || returning ? 'funding_started' : 'ready'),created:new window.Date(window.Date.now()-createdAgo).toISOString() };
  let nimPayments = nimPending || returning || nimStage ? [nim] : [],holdConfirm=null,activeConfirms=0,maxConfirms=0,mailFails=false,readsFail=false;
  if (returning) window.sessionStorage.setItem('wire-pending-payment', nim.id);
  window.PhoenixNimiqIdenticons = { async png() { return 'fixture-artwork'; }, paint() {} };
  class Hub {
    constructor(url, behavior) { assert.equal(url, 'https://hub.nimiq.com'); this.behavior = behavior; }
    async checkout(request) { checkouts.push(request); fundingVisible=true; return { hash: 'f'.repeat(64) }; }
    on(type, success) { this.callback = success; }
    async checkRedirectResponse() { if (returning) await this.callback(); }
  }
  Hub.RequestType = { CHECKOUT: 'checkout' };
  Hub.PopupRequestBehavior = class {};
  Hub.RedirectRequestBehavior = class { constructor(url, state) { behaviors.push({ url, state }); } };
  window.HubApi = Hub;
  const answer = (status, body) => ({ ok: status < 300, status, json: async () => JSON.parse(JSON.stringify(body)) });
  window.fetch = async (path, options = {}) => {
    const body = options.body ? JSON.parse(options.body) : undefined;
    calls.push([path, body]);
    if (path === '/api/config') return answer(200, { mode: 'integrated', payments_enabled: true, assets: nimOnly ? ['NIM'] : ['NIM', 'USDC_POLYGON', 'USDT_POLYGON', 'USDT_ETHEREUM'] });
    if (path === '/api/session') return answer(200, currentSession);
    if(path==='/api/usdt/cards')return ownedUnavailable?answer(503,{code:'unavailable'}):answer(200,{cards:owned});
    if (path === '/api/payments' && !options.method) return readsFail?answer(503,{code:'chain_unavailable'}):answer(200, { payments: nimPayments });
    if (path === '/api/payments') { nimPayments = [nim]; return answer(201, { payment: nim }); }
    if (path.endsWith('/art')) return answer(200, { saved: true });
    if (path.endsWith('/start')) { nim.stage = 'funding_started'; return answer(200, { payment: nim, checkout: { appName: 'Phoenix Wire', recipient: nim.address, value: nim.luna, recipientData: rawMarker } }); }
    if (path.endsWith('/confirm')) {
      activeConfirms++;maxConfirms=Math.max(maxConfirms,activeConfirms);
      try{
      if(holdConfirm){const held=holdConfirm;holdConfirm=null;await new Promise((resolve,reject)=>{held.release=resolve;options.signal?.addEventListener('abort',()=>reject(Error('aborted')),{once:true});});}
      if(manualNim){
        if(nim.stage==='funding_started')return answer(409,{code:'funding_pending'});
        if(nim.stage==='funded')nim.stage='delivery_pending';
        return answer(200,{payment:nim});
      }
      if(nativeMode==='success-no-confirm')return answer(503,{code:'chain_unavailable'});
      if(!fundingVisible)return answer(409,{code:'funding_pending'});
      nim.stage = 'sent'; return answer(200, { payment: nim });
      }finally{activeConfirms--;}
    }
    if(path.endsWith('/resend')){nim.stage=mailFails?'delivery_failed':'sent';return mailFails?answer(502,{code:'email_delivery_unavailable'}):answer(200,{payment:nim});}
    if (path.endsWith('/recovery')) return answer(200, { cashlink: 'https://hub.nimiq.com/cashlink/#' + Buffer.alloc(40, 5).toString('base64url') });
    if (path === '/api/usdt/status') return answer(200, { requests: false, send: true, protected: true, cashlink: true, gas: false, assets: ['usdc_polygon'], max_send: 1000 });
    if (path === '/api/usdt/cashlink/gas-status') return answer(200, { available: false, reason: 'off' });
    if (path === '/api/usdt/cashlink/create') {
      C.setProxyCreationCode(proxy);
      const owners = [body.linkOwner, C.checksum(C.addressOfKey(webcrypto.getRandomValues(new Uint8Array(32))))];
      const safe = C.checksum(C.cardAddress({ id: C.unhex(body.cardId), owners, threshold: 2 }, 137));
      const token = '0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359';
      owned.push({id:body.cardId,asset:body.asset,amount:body.amount,recipient:body.recipient,invited:false,created:new window.Date().toISOString()});
      return answer(200, { card: { id: body.cardId, asset: body.asset, amount: body.amount, units: 2500000, feeUnits: 150000, refundFeeUnits: 100000, chainId: 137, safe, owners, token, relayer: '0x' + '33'.repeat(20) },
        pay: { recipient: safe, units: 2650000, amount: '2.65', link: 'ethereum:' + token + '@137/transfer?address=' + safe + '&uint256=2650000', qr: 'iVBORw0KGgo=' }, claimBase: 'https://wire.example.test/usdt.html', senderToken: body.cardId });
    }
    if (path === '/api/usdt/cashlink/status') return failStatus?answer(503,{code:'unavailable'}):answer(200, state(body));
    if (path.endsWith('/invite')) {
      const result=invite(body),id=path.split('/').at(-2),card=owned.find(c=>c.id===id);
      if(result.invited===true&&card)card.invited=true;
      if(inviteMode==='lost')throw Error('lost response');
      return answer(200,result);
    }
    if (path === '/api/sender/verify') {currentSession={verified:true,email:'sender@example.test'};return answer(200,currentSession);}
    if (path === '/api/sender/request' || path === '/api/logout') return answer(200, { email: 'sender@example.test', ok: true });
    throw Error('unexpected API ' + path);
  };
  const source = read('../internal/web/static/wire-pay.js').replace(/^import[^\n]+\n/gm, '');
  Object.assign(ctx, { resolveLanguage, flowText, addressValid, cardPresentation, paintCard });
  vm.runInContext(read('../internal/web/static/flow.js').replace(/^import[^\n]+\n/gm, '').replace(/^export /gm, ''), ctx);
  vm.runInContext('{' + source + '\n}', ctx);
  await settle();
  return { window, $: id => window.document.getElementById(id), calls, doc: window.document, checkouts, behaviors, nativeCalls, confirmations,clock,
    fundOnChain(){fundingVisible=true;},setStage(stage){nim.stage=stage;},failMail(value){mailFails=value;},failReads(value){readsFail=value;},holdNextConfirm(){const held={};holdConfirm=held;return held;},maxConfirms:()=>maxConfirms,
    snapshot(){return Object.fromEntries(Object.keys(window.sessionStorage).map(k=>[k,window.sessionStorage.getItem(k)]));},
    snapshotLocal(){return Object.fromEntries(Object.keys(window.localStorage).map(k=>[k,window.localStorage.getItem(k)]));},serverCards:owned,failCardStatus(value){failStatus=value;}};
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

test('a verified sender gets PhoenixMail’s window: four coins behind one session, protected cards only, Wire on the card', async () => {
  const t = await start();
  assert.equal(t.$('wire-ready').hidden, false); assert.equal(t.$('wire-sender').hidden, true);
  assert.equal(t.$('wire-who').textContent, 'sender@example.test');
  t.$('pay').click(); await settle();
  assert.ok(t.doc.querySelector('.nimiq-payment-window'));
  const css = read('../internal/web/static/wire-pay.css');
  assert.match(css, /nimiq-payment-assets\+\.nimiq-payment-controls button:first-child/, 'only payment requests stay hidden');
  assert.equal(t.doc.querySelector('.nimiq-payment-assets button').disabled, false, 'NIM is available in the same selector');
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

test('NIM uses the common sender session and existing protected checkout in the same application', async () => {
  const t = await start(); const library = t.window.PhoenixNimiqPayments;
  assert.equal(t.$('wire-classic'), null, 'there is no separate NIM exit link');
  t.$('pay').click(); await settle();
  t.doc.querySelector('.nimiq-payment-assets button').click(); await settle();
  assert.ok(t.doc.querySelector('.wire-nim-window'));
  assert.equal(t.$('sender-auth').hidden, true, 'NIM does not ask for another sender code');
  assert.equal(t.calls.filter(([path]) => path === '/api/sender/request').length, 0);
  assert.equal(t.window.PhoenixNimiqPayments, library, 'mounting NIM preserves the copied payment library');
  assert.equal(t.doc.querySelectorAll('.wire-nim-assets button').length, 4);
  t.$('recipient').value = 'friend@example.test';
  t.$('card-form').dispatchEvent(new t.window.Event('submit', { cancelable: true })); await settle();
  t.$('wire-choice-hub').click(); await settle();
  assert.equal(t.checkouts.length, 1);
  assert.equal(t.checkouts[0].recipient, 'NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY');
  assert.deepEqual(Array.from(t.checkouts[0].extraData), [0,130,128,146,135]);
  assert.equal(t.checkouts[0].value, 250000); assert.equal(t.checkouts[0].seed, undefined);
  assert.ok(t.calls.some(([path]) => path.endsWith('/art')));
  assert.equal(t.$('feedback').textContent, 'Card sent to friend@example.test. It stays yours until it is claimed.');
  t.$('card-form').dispatchEvent(new t.window.Event('submit', { cancelable: true })); await settle();
  assert.equal(t.checkouts.length, 1, 'a repeated submit cannot fund twice');
  [...t.doc.querySelectorAll('.wire-nim-assets button')][1].click(); await settle();
  assert.ok(!t.doc.querySelector('.wire-nim-window'));
  assert.ok(t.doc.querySelector('.nimiq-payment-panel'));
  assert.equal(t.doc.querySelectorAll('.nimiq-payment-modal').length, 1);
});

test('restored NIM payments are checked without a new checkout and a NIM-only service is supported', async () => {
  const t = await start({ nimPending: true, nimOnly: true });
  assert.equal(t.$('wire-ready').hidden, false);
  t.$('pay').click(); await settle();
  assert.ok(t.doc.querySelector('.wire-nim-window'));
  t.$('card-form').dispatchEvent(new t.window.Event('submit', { cancelable: true })); await settle();
  assert.equal(t.checkouts.length, 0);
  assert.ok(!t.calls.some(([path]) => path.endsWith('/start')));
  assert.ok(t.calls.some(([path]) => path.endsWith('/confirm')));
});

test('mobile NIM checkout and return use the same saved payment behind the shared session', async () => {
  const t = await start({ mobile: true, nimOnly: true });
  t.$('pay').click(); await settle(); t.$('recipient').value = 'friend@example.test';
  t.$('card-form').dispatchEvent(new t.window.Event('submit', { cancelable: true })); await settle();
  t.$('wire-choice-hub').click(); await settle();
  assert.equal(t.checkouts.length, 1);
  assert.equal(t.window.sessionStorage.getItem('wire-pending-payment'), 'a'.repeat(48));
  assert.equal(t.behaviors.at(-1).state.wirePayment, 'a'.repeat(48));
  const back = await start({ mobile: true, returning: true });
  assert.ok(back.doc.querySelector('.wire-nim-window'), 'the root reopens NIM on return from Hub');
  assert.equal(back.checkouts.length, 0);
  assert.ok(back.calls.some(([path]) => path.endsWith('/confirm')));
  assert.equal(back.window.sessionStorage.getItem('wire-pending-payment'), '');
});

test('the theme button selects a supported light palette, keeps its preference and translates its action', async () => {
  const t = await start();
  assert.equal(t.doc.documentElement.dataset.theme, 'dark');
  assert.equal(t.$('wire-theme').getAttribute('aria-label'), 'Switch to light theme');
  t.$('wire-theme').click();
  assert.equal(t.doc.documentElement.dataset.theme, 'light-paper');
  assert.equal(t.window.localStorage.getItem('wire-theme'), 'light');
  assert.equal(t.$('wire-theme').getAttribute('aria-pressed'), 'true');
  for (const [lang, label] of Object.entries({ fr:'Passer au thème sombre', en:'Switch to dark theme', es:'Cambiar al tema oscuro', de:'Zum dunklen Design wechseln', zh:'切换为深色主题' })) {
    t.$('wire-language').value = lang; t.$('wire-language').dispatchEvent(new t.window.Event('change'));
    assert.equal(t.$('wire-theme').getAttribute('aria-label'), label);
    assert.match(t.$('wire-sub').textContent, /NIM/);
  }
  const restored = await start({ theme: 'light' });
  assert.equal(restored.doc.documentElement.dataset.theme, 'light-paper');
  restored.$('wire-theme').click(); assert.equal(restored.doc.documentElement.dataset.theme, 'dark');
  const style = t.doc.createElement('style'); style.textContent = read('../internal/web/static/wire-pay.css'); t.doc.head.append(style);
  assert.equal(t.window.getComputedStyle(t.doc.documentElement).getPropertyValue('--panel').trim(), '#fff');
  t.$('wire-theme').click();
  assert.equal(t.window.getComputedStyle(t.doc.documentElement).getPropertyValue('--panel').trim(), '#141925');
});

test('NIM cannot be opened before sender confirmation and does not ask again afterward', async () => {
  const t = await start({ session: { verified: false }, nimOnly: true });
  t.$('pay').click(); await settle();
  assert.equal(t.doc.querySelector('.wire-nim-window'), null);
  assert.ok(!t.calls.some(([path]) => path.startsWith('/api/payments')));
  t.$('wire-email').value = 'sender@example.test'; t.$('wire-send-code').click(); await settle();
  t.$('wire-code').value = '123456'; t.$('wire-verify').click(); await settle();
  t.$('pay').click(); await settle();
  assert.ok(t.doc.querySelector('.wire-nim-window'));
  assert.equal(t.calls.filter(([path]) => path === '/api/sender/request').length, 1);
  assert.equal(t.calls.filter(([path]) => path === '/api/sender/verify').length, 1);
});

test('the embedded NIM form follows the host language in all five languages', async () => {
  const t = await start({ nimOnly: true }); t.$('pay').click(); await settle();
  for (const [lang, recipient] of Object.entries({ fr:'Courriel du destinataire', en:'Recipient email', es:'Correo del destinatario', de:'E-Mail des Empfängers', zh:'收款人邮箱' })) {
    t.$('wire-language').value = lang; t.$('wire-language').dispatchEvent(new t.window.Event('change'));
    assert.equal(t.doc.querySelector('[data-nim-text="nimRecipient"]').textContent, recipient);
    assert.equal(t.$('preview-submit').textContent, flowText(lang, 'pay'));
    assert.equal(t.doc.querySelector('.wire-nim-host .security').textContent, flowText(lang, 'security'));
  }
});

async function submitNim(t){
  t.$('pay').click();await settle();
  if(!t.doc.querySelector('.wire-nim-window')){t.doc.querySelector('.nimiq-payment-assets button').click();await settle();}
  t.$('recipient').value='friend@example.test';
  t.$('card-form').dispatchEvent(new t.window.Event('submit',{cancelable:true}));await settle();
}

test('wallet choice precedes card creation, stays retryable after opening Pay and can be cancelled', async()=>{
  const t=await start({nimOnly:true,mobile:true});await submitNim(t);
  assert.equal(t.$('wire-nim-choice').hidden,false);
  assert.ok(!t.calls.some(([p,b])=>p==='/api/payments'&&b));
  assert.equal(t.checkouts.length,0);
  const link=t.$('wire-choice-open');
  assert.equal(link.href,'https://nimpay.app/miniapps/open/wire.phoenixmail.ca');
  assert.equal(link.target,'_blank');assert.match(link.rel,/noopener/);assert.match(link.rel,/noreferrer/);
  link.addEventListener('click',e=>e.preventDefault());link.click();link.click();
  assert.equal(t.$('wire-choice-status').hidden,false);
  assert.equal(t.$('wire-nim-choice').hidden,false);
  t.$('wire-choice-cancel').click();await settle();
  assert.equal(t.$('wire-nim-choice').hidden,true);
  assert.ok(!t.doc.querySelector('.wire-nim-window').hasAttribute('inert'));
  assert.equal(t.nativeCalls.length,0);assert.equal(t.checkouts.length,0);
});

test('wallet choice has all four actions in the five languages and Escape preserves the NIM form',async()=>{
  const t=await start({nimOnly:true});await submitNim(t);
  const expected={
    fr:['J’ai déjà Nimiq Pay','Obtenir Nimiq Pay','Utiliser le portefeuille Nimiq (Hub)','Annuler'],
    en:['I already have Nimiq Pay','Get Nimiq Pay','Use the Nimiq wallet (Hub)','Cancel'],
    es:['Ya tengo Nimiq Pay','Obtener Nimiq Pay','Usar la cartera Nimiq (Hub)','Cancelar'],
    de:['Ich habe bereits Nimiq Pay','Nimiq Pay herunterladen','Nimiq Wallet (Hub) verwenden','Abbrechen'],
    zh:['我已安装 Nimiq Pay','获取 Nimiq Pay','使用 Nimiq 钱包（Hub）','取消']
  };
  for(const [lang,labels]of Object.entries(expected)){
    t.$('wire-language').value=lang;t.$('wire-language').dispatchEvent(new t.window.Event('change'));
    assert.deepEqual(['open','get','hub','cancel'].map(id=>t.$('wire-choice-'+id).textContent),labels);
    assert.match(t.$('wire-choice-help').textContent,/24/);
    assert.ok(t.$('wire-choice-title').textContent);assert.ok(t.$('wire-choice-status').textContent);
  }
  t.$('wire-choice-open').focus();
  t.doc.dispatchEvent(new t.window.KeyboardEvent('keydown',{key:'Tab',shiftKey:true,bubbles:true,cancelable:true}));
  assert.equal(t.doc.activeElement,t.$('wire-choice-cancel'));
  t.doc.dispatchEvent(new t.window.KeyboardEvent('keydown',{key:'Escape',bubbles:true,cancelable:true}));
  assert.equal(t.$('wire-nim-choice').hidden,true);assert.ok(t.doc.querySelector('.wire-nim-window'));
});

test('Get Nimiq Pay uses the phone store and offers both stores on desktop',async()=>{
  for(const options of [{mobile:true},{android:true},{}]){
    const t=await start({nimOnly:true,...options});await submitNim(t);
    const link=t.$('wire-choice-get');
    if(options.mobile)assert.equal(link.href,'https://apps.apple.com/app/nimiq-pay/id6471844738');
    else if(options.android)assert.equal(link.href,'https://play.google.com/store/apps/details?id=com.nimiq.pay');
    else {link.click();assert.equal(t.$('wire-choice-stores').hidden,false);assert.equal(t.$('wire-choice-stores').querySelectorAll('a').length,2);}
    assert.equal(t.checkouts.length,0);assert.equal(t.nativeCalls.length,0);
  }
});

test('native NIM funding preserves the exact text including NUL and its UTF-8 bytes, recipient and Luna',async()=>{
  const t=await start({nimOnly:true,nativeMode:'success',mobile:true});await submitNim(t);
  assert.equal(t.$('wire-nim-choice').hidden,true);assert.equal(t.nativeCalls.length,1);
  const {request,hex}=t.nativeCalls[0];
  assert.equal(request.recipient,'NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY');assert.equal(request.value,250000);
  assert.equal(request.data,'\u0000\u0082\u0080\u0092\u0087');
  assert.equal(typeof request.data,'string');assert.equal(request.data.charCodeAt(0),0);
  assert.equal(hex,'00c282c280c292c287');assert.notEqual(hex,'0082809287');
  assert.deepEqual(Object.keys(request).sort(),['data','recipient','value']);
  assert.ok(t.calls.some(([p])=>p.endsWith('/confirm')));assert.equal(t.$('feedback').textContent,'Card sent to friend@example.test. It stays yours until it is claimed.');
  assert.equal(t.checkouts.length,0);assert.equal(t.window.sessionStorage.getItem('wire-pending-payment'),'');
  t.$('card-form').dispatchEvent(new t.window.Event('submit',{cancelable:true}));await settle();
  assert.equal(t.nativeCalls.length,1);assert.equal(t.calls.filter(([p])=>p.endsWith('/start')).length,1);
});

test('a returned native hash alone is never shown as a successful card payment',async()=>{
  const t=await start({nimOnly:true,nativeMode:'success-no-confirm'});await submitNim(t);
  assert.equal(t.nativeCalls.length,1);assert.notEqual(t.$('feedback').textContent,'Card sent to friend@example.test. It stays yours until it is claimed.');
  assert.equal(t.checkouts.length,0);assert.equal(t.window.sessionStorage.getItem('wire-pending-payment'),'a'.repeat(48));
});

test('an app without an injected NIM provider uses the existing Hub checkout',async()=>{
  const t=await start({nimOnly:true,nativeMode:'missing',mobile:true});await submitNim(t);
  assert.equal(t.$('wire-nim-choice').hidden,true);assert.equal(t.nativeCalls.length,0);assert.equal(t.checkouts.length,1);
  assert.deepEqual(Array.from(t.checkouts[0].extraData),[0,130,128,146,135]);
  assert.equal(t.checkouts[0].value,250000);
});

test('native refusal, including the legacy error envelope, is cancellation without Hub or re-payment',async()=>{
  for(const nativeMode of ['denied','legacy-denied']){
    const t=await start({nimOnly:true,nativeMode});await submitNim(t);
    assert.equal(t.$('feedback').textContent,'Payment cancelled in Nimiq Pay.');
    assert.equal(t.checkouts.length,0);assert.equal(t.nativeCalls.length,1);
    assert.equal(t.doc.querySelector('.wire-nim-hub-fallback').hidden,true);
    t.$('card-form').dispatchEvent(new t.window.Event('submit',{cancelable:true}));await settle();
    assert.equal(t.nativeCalls.length,1);assert.ok(t.calls.some(([p])=>p.endsWith('/confirm')));
  }
});

test('native errors check funding before offering Hub, then reuse the same card and original raw Hub marker',async()=>{
  const t=await start({nimOnly:true,nativeMode:'error'});await submitNim(t);
  assert.ok(t.calls.some(([p])=>p.endsWith('/confirm')));assert.equal(t.checkouts.length,0);
  const fallback=t.doc.querySelector('.wire-nim-hub-fallback');assert.equal(fallback.hidden,false);
  fallback.click();await settle();
  assert.equal(t.confirmations.length,1);assert.equal(t.checkouts.length,1);
  assert.equal(t.checkouts[0].recipient,t.nativeCalls[0].request.recipient);
  assert.equal(t.checkouts[0].value,t.nativeCalls[0].request.value);
  assert.deepEqual(Array.from(t.checkouts[0].extraData),[0,130,128,146,135]);
  assert.equal(t.calls.filter(([p])=>p.endsWith('/start')).length,1);
  assert.equal(t.calls.filter(([p,b])=>p==='/api/payments'&&b).length,1);
  assert.equal(t.$('feedback').textContent,'Card sent to friend@example.test. It stays yours until it is claimed.');
});

test('an error after native broadcast, or later funding before the fallback click, never launches Hub',async()=>{
  const sent=await start({nimOnly:true,nativeMode:'error-after-send'});await submitNim(sent);
  assert.equal(sent.$('feedback').textContent,'Card sent to friend@example.test. It stays yours until it is claimed.');assert.equal(sent.checkouts.length,0);
  assert.equal(sent.doc.querySelector('.wire-nim-hub-fallback').hidden,true);
  const late=await start({nimOnly:true,nativeMode:'error'});await submitNim(late);
  late.fundOnChain();late.doc.querySelector('.wire-nim-hub-fallback').click();await settle();
  assert.equal(late.checkouts.length,0);assert.equal(late.confirmations.length,0);
  assert.equal(late.$('feedback').textContent,'Card sent to friend@example.test. It stays yours until it is claimed.');
});

test('native checkout rejects another funding marker and a different host network before sending money',async()=>{
  const altered=await start({nimOnly:true,nativeMode:'success',rawMarker:[0,130,128,146,136]});await submitNim(altered);
  assert.equal(altered.nativeCalls.length,0);assert.equal(altered.checkouts.length,0);
  const wrong=await start({nimOnly:true,nativeMode:'success',network:'testnet'});await submitNim(wrong);
  assert.equal(wrong.nativeCalls.length,0);assert.equal(wrong.checkouts.length,0);
  assert.ok(!wrong.calls.some(([p])=>p.endsWith('/start')));
});

test('a new Nimiq Pay context uses the existing sender email confirmation and does not inherit browser state',async()=>{
  const t=await start({nimOnly:true,nativeMode:'success',session:{verified:false}});
  assert.equal(t.$('wire-sender').hidden,false);assert.equal(t.$('wire-ready').hidden,true);
  assert.equal(t.nativeCalls.length,0);
  t.$('wire-email').value='sender@example.test';t.$('wire-send-code').click();await settle();
  t.$('wire-code').value='123456';t.$('wire-verify').click();await settle();
  await submitNim(t);assert.equal(t.nativeCalls.length,1);
  assert.equal(t.calls.filter(([p])=>p==='/api/sender/verify').length,1);
});

async function tracked(options={}){
  const t=await start({nimOnly:true,manualNim:true,clocked:true,nimPending:true,...options});
  if(!t.doc.querySelector('.wire-nim-window')){t.$('pay').click();await settle();if(!t.doc.querySelector('.wire-nim-window')){t.doc.querySelector('.nimiq-payment-assets button').click();await settle();}}
  return t;
}
const checks=t=>t.calls.filter(([p])=>p.endsWith('/confirm')).length;

test('NIM tracking continues beyond ten minutes, then stops at sent with an explicit Close action',async()=>{
  const t=await tracked();
  await t.clock.advance(12*60000);
  assert.ok(checks(t)>20,'the old twenty-check limit is gone');
  assert.equal(t.checkouts.length,0);assert.equal(t.nativeCalls.length,0);
  t.setStage('sent');await t.clock.advance(30000);
  assert.equal(t.$('nim-progress-message').textContent,'Card sent to friend@example.test. It stays yours until it is claimed.');
  assert.equal(t.$('nim-progress-close').hidden,false);assert.equal(t.$('nim-progress-close').textContent,'Close');
  assert.equal(t.$('nim-progress-resend').hidden,true);assert.ok(!t.$('payment-list').textContent.includes('Resend'));
  const count=t.calls.length;await t.clock.advance(10*60000);
  t.window.dispatchEvent(new t.window.Event('pageshow'));await settle();assert.equal(t.calls.length,count,'sent cards are not polled again');
  t.$('nim-progress-close').click();assert.equal(t.doc.querySelector('.wire-nim-window'),null);
  assert.match(t.$('wire-status').textContent,/Card sent to friend@example.test/);
});

test('a funded card shows email delivery progress, then a clear failure and a resend that never funds again',async()=>{
  const t=await tracked();t.setStage('funded');const held=t.holdNextConfirm();await t.clock.advance(4000);
  assert.equal(t.$('nim-progress-message').textContent,'Payment confirmed. Sending the card to friend@example.test…');
  assert.equal(t.$('nim-progress-close').hidden,true);
  t.setStage('delivery_pending');held.release();await settle();
  assert.match(t.$('nim-progress-message').textContent,/Payment confirmed/);
  t.setStage('delivery_failed');await t.clock.advance(4000);
  assert.match(t.$('nim-progress-message').textContent,/email could not be sent.*without paying again/);
  assert.equal(t.$('nim-progress-resend').hidden,false);
  t.failMail(true);t.$('nim-progress-resend').click();await settle();
  assert.equal(t.$('nim-progress-close').hidden,true);assert.equal(t.$('nim-progress-resend').disabled,false);
  t.failMail(false);t.$('nim-progress-resend').click();t.$('nim-progress-resend').click();await settle();
  assert.equal(t.calls.filter(([p])=>p.endsWith('/resend')).length,2,'one failed try, one successful try');
  assert.equal(t.$('nim-progress-close').hidden,false);
  assert.equal(t.checkouts.length,0);assert.equal(t.nativeCalls.length,0);assert.ok(!t.calls.some(([p])=>p.endsWith('/start')));
});

test('an approved payment shows the waiting status in each language; funded, sent, Close and Resend are translated',async()=>{
  const t=await tracked({saved:{['wire-nim-approved:'+'a'.repeat(48)]:'yes'}});
  const expected={
    fr:['Paiement envoyé. Confirmation sur le réseau en cours…','Paiement confirmé. Envoi de la carte à friend@example.test…','Carte envoyée à friend@example.test. Elle reste à vous tant qu’elle n’est pas réclamée.','Fermer','Renvoyer'],
    en:['Payment sent. Waiting for network confirmation…','Payment confirmed. Sending the card to friend@example.test…','Card sent to friend@example.test. It stays yours until it is claimed.','Close','Resend'],
    es:['Pago enviado. Esperando la confirmación de la red…','Pago confirmado. Enviando la tarjeta a friend@example.test…','Tarjeta enviada a friend@example.test. Sigue siendo tuya hasta que se reclame.','Cerrar','Reenviar'],
    de:['Zahlung gesendet. Netzwerkbestätigung ausstehend…','Zahlung bestätigt. Die Karte wird an friend@example.test gesendet…','Karte an friend@example.test gesendet. Sie bleibt deine, bis sie eingelöst wird.','Schließen','Erneut senden'],
    zh:['付款已发送。正在等待网络确认…','付款已确认。正在向 friend@example.test 发送卡片…','卡片已发送至 friend@example.test。在被领取之前它仍属于你。','关闭','重新发送']
  };
  for(const[lang,labels]of Object.entries(expected)){
    t.$('wire-language').value=lang;t.$('wire-language').dispatchEvent(new t.window.Event('change'));
    assert.equal(t.$('nim-progress-message').textContent,labels[0]);
    assert.equal(t.$('nim-progress-close').textContent,labels[3]);assert.equal(t.$('nim-progress-resend').textContent,labels[4]);
  }
  t.setStage('delivery_pending');await t.clock.advance(4000);
  for(const[lang,labels]of Object.entries(expected)){t.$('wire-language').value=lang;t.$('wire-language').dispatchEvent(new t.window.Event('change'));assert.equal(t.$('nim-progress-message').textContent,labels[1]);}
  t.setStage('delivery_failed');await t.clock.advance(4000);
  for(const lang of Object.keys(expected)){t.$('wire-language').value=lang;t.$('wire-language').dispatchEvent(new t.window.Event('change'));assert.ok(t.$('nim-progress-message').textContent.length>25);assert.equal(t.$('nim-progress-resend').hidden,false);}
  t.setStage('sent');await t.clock.advance(30000);
  for(const[lang,labels]of Object.entries(expected)){t.$('wire-language').value=lang;t.$('wire-language').dispatchEvent(new t.window.Event('change'));assert.equal(t.$('nim-progress-message').textContent,labels[2]);}
});

test('iPhone suspended timers resume immediately on visibilitychange and pageshow, without opening a wallet',async()=>{
  const t=await tracked();const before=checks(t);
  t.clock.hidden(true);t.clock.suspend(10*60000);
  assert.equal(checks(t),before);
  t.clock.hidden(false);await settle();assert.equal(checks(t),before+1);
  t.clock.suspend(5*60000);t.window.dispatchEvent(new t.window.Event('pageshow'));await settle();
  assert.equal(checks(t),before+2);
  t.setStage('sent');t.window.dispatchEvent(new t.window.Event('pageshow'));await settle();
  assert.equal(t.$('nim-progress-close').hidden,false);
  assert.equal(t.checkouts.length,0);assert.equal(t.nativeCalls.length,0);assert.equal(t.maxConfirms(),1);
});

test('reload restores a native payment and its original expiry without another payment or manual refresh',async()=>{
  const first=await tracked({saved:{'wire-pending-payment':'a'.repeat(48),'wire-pending-wallet':'pay',['wire-nim-approved:'+'a'.repeat(48)]:'yes'}});
  const saved=first.snapshot();first.window.close();
  const t=await start({nimOnly:true,manualNim:true,clocked:true,nimPending:true,saved});
  await settle();assert.ok(t.doc.querySelector('.wire-nim-window'));assert.ok(checks(t)>=1);
  assert.match(t.$('nim-progress-message').textContent,/Payment sent/);
  t.setStage('sent');await t.clock.advance(4000);
  assert.equal(t.$('nim-progress-close').hidden,false);
  assert.equal(t.window.sessionStorage.getItem('wire-pending-payment'),'');
  assert.equal(t.checkouts.length,0);assert.equal(t.nativeCalls.length,0);
});

test('closing or changing the NIM window keeps its tracker alive and reopening reuses it',async()=>{
  const t=await tracked();t.doc.querySelector('.wire-nim-close').click();
  await t.clock.advance(3*60000);assert.ok(checks(t)>20);
  t.setStage('sent');await t.clock.advance(10000);assert.match(t.$('wire-status').textContent,/Card sent to/);
  t.$('pay').click();await settle();assert.equal(t.$('nim-progress-close').hidden,false);
  assert.equal(t.doc.querySelectorAll('.wire-nim-window').length,1);
  assert.equal(t.maxConfirms(),1);assert.equal(t.checkouts.length,0);
});

test('foreground events share an in-flight confirmation, and a stalled request times out then recovers',async()=>{
  const t=await tracked();t.holdNextConfirm();
  t.window.dispatchEvent(new t.window.Event('pageshow'));await settle();const before=checks(t);
  for(let i=0;i<5;i++){t.window.dispatchEvent(new t.window.Event('pageshow'));t.doc.dispatchEvent(new t.window.Event('visibilitychange'));}
  await settle();assert.equal(checks(t),before);assert.equal(t.maxConfirms(),1);
  await t.clock.advance(30000);assert.match(t.$('nim-progress-message').textContent,/resume automatically/);
  t.window.dispatchEvent(new t.window.Event('pageshow'));await settle();assert.equal(checks(t),before+1);assert.equal(t.maxConfirms(),1);
  t.setStage('sent');await t.clock.advance(4000);assert.equal(t.$('nim-progress-close').hidden,false);
});

test('a temporary read failure recovers automatically, while a card stops tracking at its original 24-hour deadline',async()=>{
  const t=await tracked();t.failReads(true);await t.clock.advance(4000);
  assert.match(t.$('nim-progress-message').textContent,/not responding/);
  t.failReads(false);t.setStage('sent');await t.clock.advance(4000);
  assert.equal(t.$('nim-progress-close').hidden,false);
  const exp=await tracked({createdAgo:86400000-20000});await exp.clock.advance(20000);
  assert.match(exp.$('nim-progress-message').textContent,/24-hour expiry/);const count=exp.calls.length;
  exp.window.dispatchEvent(new exp.window.Event('pageshow'));await exp.clock.advance(10*60000);
  assert.equal(exp.calls.length,count);assert.equal(exp.checkouts.length,0);
});

test('logout stops the cached NIM tracker',async()=>{
  const t=await tracked();t.$('wire-logout').click();await settle();const count=t.calls.length;
  await t.clock.advance(10*60000);t.window.dispatchEvent(new t.window.Event('pageshow'));await settle();
  assert.equal(t.calls.length,count);assert.equal(t.doc.querySelector('.wire-nim-window'),null);
});

test('after a live Hub or native approval, twelve minutes of tracking never creates or funds a second card',async()=>{
  for(const nativeMode of ['', 'success']){
    const t=await start({nimOnly:true,manualNim:true,clocked:true,nativeMode});await submitNim(t);
    if(!nativeMode){t.$('wire-choice-hub').click();await settle();}
    assert.equal(t.$('nim-progress-message').textContent,'Payment sent. Waiting for network confirmation…');
    await t.clock.advance(12*60000);t.window.dispatchEvent(new t.window.Event('pageshow'));await settle();
    assert.ok(checks(t)>20);
    assert.equal(t.calls.filter(([p,b])=>p==='/api/payments'&&b).length,1);
    assert.equal(t.calls.filter(([p])=>p.endsWith('/start')).length,1);
    assert.equal(t.nativeCalls.length,nativeMode?1:0);assert.equal(t.checkouts.length,nativeMode?0:1);
    t.setStage('sent');await t.clock.advance(30000);assert.equal(t.$('nim-progress-close').hidden,false);
  }
});

test('switching to a stablecoin leaves NIM tracking active and returning to NIM reuses the same window',async()=>{
  const t=await tracked({nimOnly:false});const original=t.doc.querySelector('.wire-nim-window');
  original.querySelectorAll('.wire-nim-assets button')[1].click();await settle();
  assert.equal(t.doc.querySelector('.wire-nim-window'),null);assert.ok(t.doc.querySelector('.nimiq-payment-window'));
  await t.clock.advance(2*60000);assert.ok(checks(t)>20);
  t.setStage('sent');await t.clock.advance(10000);assert.match(t.$('wire-status').textContent,/Card sent to/);
  t.doc.querySelector('.nimiq-payment-assets button').click();await settle();
  assert.equal(t.doc.querySelector('.wire-nim-window'),original);assert.equal(t.$('nim-progress-close').hidden,false);
  assert.equal(t.nativeCalls.length,0);assert.equal(t.checkouts.length,0);
});

test('the sender card overrides PhoenixMail reflections with Wire’s shared linear gradient and removes the ring',async()=>{
  const t=await start();await makeCard(t);
  const style=t.doc.createElement('style');style.textContent=read('../internal/web/static/wire-pay.css');t.doc.head.append(style);
  const rules=[...style.sheet.cssRules];
  const cardRule=rules.find(r=>r.selectorText==='.wire-body .nimiq-payment-card');
  assert.ok(cardRule,'the host stylesheet supplies a scoped override');
  assert.equal(cardRule.style.getPropertyValue('background-image'),'linear-gradient(125deg,var(--card-bg,#ece3ff) 0%,#e2eeff 56%,#ffe8f4 100%)');
  assert.ok(!cardRule.style.getPropertyValue('background-image').includes('radial-gradient'));
  assert.equal(cardRule.style.getPropertyValue('border-radius'),'26px');
  assert.equal(cardRule.style.getPropertyValue('border-color'),'var(--card-accent,var(--card-border,#c7b1ee))');
  const ring=rules.find(r=>r.selectorText==='.wire-body .nimiq-payment-card::before');
  assert.equal(ring.style.getPropertyValue('content'),'none');assert.equal(ring.style.getPropertyValue('display'),'none');
  assert.ok(t.doc.querySelector('.nimiq-payment-card').matches(cardRule.selectorText));
  const nimRule=rules.findIndex(r=>r.selectorText==='.wire-nim-host .payment-card');
  assert.ok(rules.indexOf(cardRule)>nimRule,'the override also wins over the embedded NIM background shorthand');
});

test('Wire funding copy names the recipient and Done; closing the window leaves My cards and the sent banner',async()=>{
  const t=await start({clocked:true,state:()=>({state:'ready',balanceUnits:2650000})});await makeCard(t);
  const stored=JSON.parse(t.window.localStorage.getItem('wire-usdt-cards'))[0];
  assert.equal(stored.recipient,'friend@example.test');assert.equal(stored.amount,'2.5');assert.equal(stored.asset,'usdc_polygon');
  await t.clock.advance(3000);
  assert.equal(t.doc.querySelector('.usdt-funded').textContent,'✓ Card funded. Wire is sending the card to friend@example.test.');
  assert.equal(t.doc.querySelector('.usdt-close').textContent,'Done');
  assert.ok(!t.doc.querySelector('.usdt-note').textContent.includes('added to the email'));
  t.doc.querySelector('.usdt-close').click();await settle();
  assert.equal(t.doc.querySelector('.nimiq-payment-window'),null);
  assert.equal(t.$('wire-cards').hidden,false);assert.match(t.$('wire-cards-list').textContent,/2.5 USDC · Polygon.*friend@example.test.*Card sent to friend@example.test/s);
  assert.equal(t.$('wire-card-notices').hidden,false);assert.match(t.$('wire-card-notices').textContent,/Card funded. Email sent to friend@example.test./);
  await t.clock.advance(120000);assert.equal(t.$('wire-card-notices').hidden,false,'notification has no automatic timeout');
  assert.equal(t.calls.filter(([p])=>p.endsWith('/invite')).length,1);
});

test('the funded window and dashboard states are translated in all five languages',async()=>{
  const labels={
    fr:['Mes cartes','Carte alimentée. Wire envoie la carte à friend@example.test.','Terminé','En attente de votre paiement','Paiement reçu, confirmation en cours','Carte envoyée à friend@example.test','Réclamée','Remboursée','Expirée','Carte alimentée. Courriel envoyé à friend@example.test.'],
    en:['My cards','Card funded. Wire is sending the card to friend@example.test.','Done','Waiting for your payment','Payment received, awaiting confirmation','Card sent to friend@example.test','Claimed','Refunded','Expired','Card funded. Email sent to friend@example.test.'],
    es:['Mis tarjetas','Tarjeta financiada. Wire está enviando la tarjeta a friend@example.test.','Listo','Esperando tu pago','Pago recibido, esperando confirmación','Tarjeta enviada a friend@example.test','Reclamada','Reembolsada','Caducada','Tarjeta financiada. Correo enviado a friend@example.test.'],
    de:['Meine Karten','Karte aufgeladen. Wire sendet die Karte an friend@example.test.','Fertig','Wartet auf deine Zahlung','Zahlung eingegangen, Bestätigung ausstehend','Karte an friend@example.test gesendet','Eingelöst','Zurückerstattet','Abgelaufen','Karte aufgeladen. E-Mail an friend@example.test gesendet.'],
    zh:['我的卡片','卡片已充值。Wire 正在向 friend@example.test 发送卡片。','完成','等待你付款','已收到付款，正在等待确认','卡片已发送至 friend@example.test','已领取','已退款','已过期','卡片已充值。邮件已发送至 friend@example.test。']
  };
  const t=await start({clocked:true,state:()=>({state:'ready',balanceUnits:2650000})});await makeCard(t);await t.clock.advance(3000);
  for(const[lang,w]of Object.entries(labels)){
    t.$('wire-language').value=lang;t.$('wire-language').dispatchEvent(new t.window.Event('change'));
    const cl=t.window.PhoenixUSDTCashlink.create({language:()=>lang});
    assert.equal(cl.words().fundedMsg.replace('{email}','friend@example.test'),w[1]);assert.equal(cl.words().close,w[2]);
    assert.equal(t.$('wire-cards-title').textContent,w[0]);assert.equal(t.doc.querySelector('.wire-card-state').textContent,w[5]);
    assert.equal(t.doc.querySelector('.wire-card-banner p').textContent,w[9]);
  }
  const id='c'.repeat(32),server={id,asset:'usdc_polygon',amount:'2.5',recipient:'friend@example.test',invited:true};let state='awaiting_funds';
  const history=await start({clocked:true,serverCards:[server],state:()=>({state,balanceUnits:0})});
  for(const[stage,index]of [['awaiting_funds',3],['verifying',4],['ready',5],['paid',6],['withdrawn',7],['expired',8]]){
    state=stage;await history.clock.advance(60000);
    for(const[lang,w]of Object.entries(labels)){history.$('wire-language').value=lang;history.$('wire-language').dispatchEvent(new history.window.Event('change'));assert.equal(history.doc.querySelector('.wire-card-state').textContent,w[index]);}
  }
});

test('without host overrides the mirrored words remain unchanged',()=>{
  const dom=new JSDOM('',{runScripts:'outside-only'});doms.push(dom);
  vm.runInContext(read('../internal/web/static/pm/usdt-cashlink.js'),dom.getInternalVMContext());
  const module=dom.window.PhoenixUSDTCashlink;
  for(const lang of ['fr','en','es','de','zh']){
    const base=module.WORDS[lang],actual=module.create({language:()=>lang}).words('USDT');
    assert.equal(actual.fundedMsg,base.fundedMsg);assert.equal(actual.close,base.close);assert.equal(actual.errors.gas_not_needed,base.errors.gas_not_needed);
  }
});

test('reload restores the dashboard and its notification; manual dismissal survives another visit',async()=>{
  const first=await start({clocked:true,state:()=>({state:'ready',balanceUnits:2650000})});await makeCard(first);await first.clock.advance(8000);
  assert.equal(first.doc.querySelector('.nimiq-payment-window'),null,'automatic window close does not erase the result');
  const local=first.snapshotLocal(),serverCards=first.serverCards;first.window.close();
  const second=await start({clocked:true,local,serverCards,state:()=>({state:'ready',balanceUnits:2650000})});await second.clock.advance(2500);
  assert.match(second.$('wire-cards-list').textContent,/Card sent to friend@example.test/);assert.equal(second.$('wire-card-notices').hidden,false);
  second.doc.querySelector('.wire-card-banner button').click();assert.equal(second.$('wire-card-notices').hidden,true);
  const dismissed=second.snapshotLocal();second.window.close();
  const third=await start({clocked:true,local:dismissed,serverCards,state:()=>({state:'ready',balanceUnits:2650000})});await third.clock.advance(6000);
  assert.equal(third.$('wire-card-notices').hidden,true);assert.match(third.$('wire-cards-list').textContent,/Card sent to/);
  assert.equal(third.calls.filter(([p])=>p.endsWith('/invite')).length,0);
});

test('an invite failure or unconfirmed reply never becomes a sent banner or a new funding request',async()=>{
  for(const invite of [()=>{throw Error('mail unavailable');},()=>({})]){
    const t=await start({clocked:true,invite,state:()=>({state:'ready',balanceUnits:2650000})});await makeCard(t);await t.clock.advance(8000);
    assert.equal(t.$('wire-card-notices').hidden,true);assert.match(t.$('wire-cards-list').textContent,/email could not be sent/);
    assert.equal(JSON.parse(t.window.localStorage.getItem('wire-usdt-cards'))[0].invited,false);
    assert.equal(t.calls.filter(([p])=>p==='/api/usdt/cashlink/create').length,1);assert.equal(t.checkouts.length,0);
    assert.equal(t.calls.filter(([p])=>p.endsWith('/invite')).length,1,'failed invitations are not retried every status tick');
  }
});

test('a lost invitation reply is reconciled with the persisted server flag instead of sending a second email',async()=>{
  const t=await start({clocked:true,inviteMode:'lost',state:()=>({state:'ready',balanceUnits:2650000})});await makeCard(t);await t.clock.advance(130000);
  assert.equal(t.calls.filter(([p])=>p.endsWith('/invite')).length,1);assert.equal(t.$('wire-card-notices').hidden,false);
  assert.match(t.$('wire-cards-list').textContent,/Card sent to friend@example.test/);
});

test('legacy records regain amount and recipient from owned metadata, with no claim fragment in dashboard or status calls',async()=>{
  const id='d'.repeat(32),fragment='#c='+'A'.repeat(22)+'.'+'B'.repeat(43);
  const local={'wire-usdt-cards':JSON.stringify([{id,fragment,recipient:'',invited:false}])};
  const t=await start({clocked:true,local,serverCards:[{id,asset:'usdt_ethereum',amount:'12.5',recipient:'friend@example.test',invited:false}],state:()=>({state:'awaiting_funds',balanceUnits:0})});await t.clock.advance(2500);
  assert.match(t.$('wire-cards-list').textContent,/12.5 USDT · Ethereum.*friend@example.test.*Waiting for your payment/s);
  assert.ok(!t.$('wire-cards').innerHTML.includes(fragment));assert.ok(!JSON.stringify(t.calls.filter(([p])=>p.endsWith('/status'))).includes(fragment));
  assert.equal(JSON.parse(t.window.localStorage.getItem('wire-usdt-cards'))[0].fragment,fragment);
});

test('an invited card keeps updating after window close, preserves its last state on failure and resumes on foreground',async()=>{
  let state='ready';const t=await start({clocked:true,state:()=>({state,balanceUnits:2650000})});await makeCard(t);await t.clock.advance(8000);
  t.failCardStatus(true);await t.clock.advance(6000);
  assert.match(t.$('wire-cards-list').textContent,/Card sent to.*Stale view/s);
  t.clock.hidden(true);const before=t.calls.length;t.clock.suspend(300000);assert.equal(t.calls.length,before);
  t.failCardStatus(false);state='paid';t.clock.hidden(false);await settle();
  assert.equal(t.doc.querySelector('.wire-card-state').textContent,'Claimed');assert.equal(t.doc.querySelector('.wire-card-stale'),null);
  state='withdrawn';await t.clock.advance(60000);assert.equal(t.doc.querySelector('.wire-card-state').textContent,'Refunded');
  state='expired';await t.clock.advance(60000);assert.equal(t.doc.querySelector('.wire-card-state').textContent,'Expired');
});

test('only the signed-in sender’s records are shown; logout hides private history and stops dashboard polling',async()=>{
  const foreign={id:'e'.repeat(32),sender:'someone@example.test',recipient:'private@example.test',amount:'900',asset:'usdc_polygon',invited:true};
  const t=await start({clocked:true,local:{'wire-usdt-cards':JSON.stringify([foreign])}});
  assert.ok(!t.$('wire-cards-list').textContent.includes('private@example.test'));assert.equal(t.$('wire-cards-empty').hidden,false);
  await makeCard(t);await t.clock.advance(2500);t.$('wire-logout').click();await settle();const count=t.calls.length;
  assert.equal(t.$('wire-cards').hidden,true);assert.equal(t.$('wire-card-notices').hidden,true);
  await t.clock.advance(120000);assert.equal(t.calls.length,count);
});

test('recipient and notification values are literal text, and NIM history appears without creating a payment',async()=>{
  const id='f'.repeat(32),recipient='<img src=x onerror=alert(1)>',local={'wire-usdt-cards':JSON.stringify([{id,sender:'sender@example.test',recipient,amount:'2',asset:'usdt_polygon',invited:true,noticePending:true}])};
  const t=await start({clocked:true,local,serverCards:[{id,asset:'usdt_polygon',recipient,amount:'2',invited:true}],nimStage:'sent',state:()=>({state:'ready',balanceUnits:2100000})});await t.clock.advance(2500);
  assert.ok(t.$('wire-cards-list').textContent.includes(recipient));assert.ok(t.$('wire-card-notices').textContent.includes(recipient));
  assert.equal(t.$('wire-cards-list').querySelector('img'),null);assert.equal(t.$('wire-card-notices').querySelector('img'),null);
  assert.match(t.$('wire-cards-list').textContent,/2.5 NIM · Nimiq/);
  assert.ok(!t.calls.some(([p,b])=>p==='/api/payments'&&b));assert.equal(t.checkouts.length,0);assert.equal(t.nativeCalls.length,0);
});
