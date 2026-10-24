import { flowText } from './flow-text.js?v=0.4.24';
// The sender's side of a USDC / USDT card. The browser makes the card id and the link key and KEEPS them: they are what lets the sender
// take the money back. Wire's server never stores the claim link; the browser hands it over once, to be e-mailed, after the card is funded.
import { usdtText } from './usdt-text.js?v=0.4.21';
import { paintCard } from './model.js?v=0.4.24';

const $ = id => document.getElementById(id);
const STORE = 'wire-usdt-cards';
const ASSETS = {
 USDC_POLYGON: { id: 'usdc_polygon', symbol: 'USDC', chain: 'Polygon', chainId: 137, token: '0x3c499c542cef5e3811e1192ce70d8cc03d5c3359' },
 USDT_POLYGON: { id: 'usdt_polygon', symbol: 'USDT', chain: 'Polygon', chainId: 137, token: '0xc2132d05d31c914a87c6611c10748aeb04b58e8f' },
 USDT_ETHEREUM: { id: 'usdt_ethereum', symbol: 'USDT', chain: 'Ethereum', chainId: 1, token: '0xdac17f958d2ee523a2206206994597c13d831ec7' },
};
const byId = Object.fromEntries(Object.values(ASSETS).map(a => [a.id, a]));
const FLAT_GAS_FEE = { usdt_ethereum: 750000, usdc_polygon: 100000, usdt_polygon: 100000 }; // what a refund pays the relayer (an older card without a stored value)

let enabledAssets = [], verified = false, lib = null, timers = new Map(), busy = false, current = null;

const words = () => usdtText[document.documentElement.lang] || usdtText.en;
const fill = (text, values) => text.replace(/\{(\w+)\}/g, (_, k) => String(values[k] ?? ''));
const unitsText = u => (Number(u) / 1e6).toFixed(6).replace(/0+$/, '').replace(/\.$/, '');
const fail = code => Object.assign(Error(code), { code });
const errorText = e => words().errors[e && e.code] || words().errors.generic;

function readCards() { try { const v = JSON.parse(localStorage.getItem(STORE) || '[]'); return Array.isArray(v) ? v : []; } catch { return []; } }
function writeCards(list) { try { localStorage.setItem(STORE, JSON.stringify(list.slice(0, 100))); return true; } catch { return false; } }
function saveCard(entry) { const all = readCards().filter(c => c.id !== entry.id); all.unshift(entry); if (!writeCards(all)) throw fail('local_storage'); }
function updateCard(id, patch) { const all = readCards(); const i = all.findIndex(c => c.id === id); if (i >= 0) { Object.assign(all[i], patch); writeCards(all); } }

async function api(path, data) {
 const options = { credentials: 'same-origin', cache: 'no-store', redirect: 'error' };
 if (data !== undefined) { options.method = 'POST'; options.headers = { 'Content-Type': 'application/json' }; options.body = JSON.stringify(data); }
 const response = await fetch(path, options);
 let body; try { body = await response.json(); } catch { throw fail('chain_unavailable'); }
 if (!response.ok) throw Object.assign(fail(body.code || body.error || 'generic'), { status: response.status });
 return body;
}

function loadScript(src) {
 return new Promise((ok, no) => { const s = document.createElement('script'); s.src = src; s.onload = ok; s.onerror = () => no(fail('unavailable')); document.head.append(s); });
}
async function library() {
 if (lib) return lib;
 if (!window.PhoenixCashlink) { await loadScript('/cashlink.js'); await loadScript('/proxycode.js'); }
 if (!window.PhoenixCashlink || !window.PHOENIX_PROXY_CREATION_CODE) throw fail('unavailable');
 window.PhoenixCashlink.setProxyCreationCode(window.PHOENIX_PROXY_CREATION_CODE);
 return (lib = window.PhoenixCashlink);
}
async function qrLibrary() { if (!window.qrcode) await loadScript('/qrcode.js'); return window.qrcode; }

const hexBytes = (C, h) => C.unhex(String(h).replace(/^0x/, ''));

// ---- making a card ----
async function createCard(sym, form) {
 const a = ASSETS[sym], C = await library();
 const { id, key } = C.newCard(), idHex = C.hex(id), linkOwner = C.checksum(C.addressOfKey(key));
 const made = await api('/api/usdt/create', { asset: a.id, card_id: idHex, link_owner: linkOwner, amount: form.amount, recipient: form.recipient, message: form.message, language: document.documentElement.lang });
 const cd = made.card, bad = () => fail('invalid_service_answer');
 if (!cd || cd.id !== idHex || cd.asset !== a.id || !Array.isArray(cd.owners) || cd.owners.length !== 2 || !cd.owners.includes(linkOwner) || cd.chainId !== a.chainId || String(cd.token).toLowerCase() !== a.token) throw bad();
 // The card's address is computed here too, from the owners: if the service's differs, the card is never shown for payment.
 const safe = C.checksum(C.cardAddress({ id, owners: cd.owners, threshold: 2 }, cd.chainId));
 if (safe !== cd.safe || !made.pay || made.pay.recipient !== safe || !Number.isSafeInteger(cd.units) || !Number.isSafeInteger(cd.feeUnits) || made.pay.units !== cd.units + cd.feeUnits) throw bad();
 if (typeof made.pay.link !== 'string' || !made.pay.link.startsWith('ethereum:0x') || !made.pay.link.includes(safe)) throw bad();
 if (!Number.isSafeInteger(cd.refundFeeUnits) || cd.refundFeeUnits <= 0 || cd.refundFeeUnits > cd.feeUnits) throw bad();
 const entry = { id: idHex, asset: a.id, symbol: a.symbol, chain: a.chain, amount: cd.amount, recipient: form.recipient, message: form.message, fragment: C.linkFragment({ id, key }), safe, owners: cd.owners,
  units: cd.units, feeUnits: cd.feeUnits, refundFeeUnits: cd.refundFeeUnits, chainId: cd.chainId, token: cd.token, relayer: cd.relayer, payLink: made.pay.link, payAmount: made.pay.amount, invited: false, created: Date.now() };
 saveCard(entry);
 return entry;
}

function claimURL(entry) { return location.origin + '/usdt.html?lang=' + document.documentElement.lang + entry.fragment; }

// ---- showing it, and watching the money ----
function setStatus(text, kind) { const el = $('usdt-status'); el.textContent = text; el.dataset.kind = kind || ''; }

// The funding step IS the card: same look as the card the recipient will get (its colour comes from its id), with the QR where the coin was.
function fundingCard(entry) {
 const card = $('payment-card');
 card.dataset.mode = 'fund'; card.dataset.currency = entry.asset.toUpperCase();
 paintCard(card, document.documentElement.lang, entry.id);
 $('preview-empty').hidden = true; $('preview-note').hidden = true; card.hidden = false;
 $('card-amount').textContent = entry.payAmount; $('card-unit').textContent = entry.symbol;
 $('card-network').textContent = entry.chain; $('card-network').hidden = false;
 $('card-heading').textContent = words().fundHeading;
 $('card-message').textContent = entry.message || '';
 // the coin of this card
 const coin = $('coin'); coin.textContent = ''; const img = document.createElement('img'); img.width = img.height = 80; img.alt = entry.symbol;
 img.src = entry.asset === 'usdc_polygon' ? '/brand/usdc-coin.png?v=0.3.0' : '/brand/usdt.svg?v=0.3.0'; coin.append(img);
}

async function showPay(entry) {
 const t = words();
 fundingCard(entry);
 current = entry;
 $('usdt-panel').hidden = false;
 $('usdt-title').textContent = fill(t.payTitle, { amount: entry.payAmount, symbol: entry.symbol, chain: entry.chain });
 $('usdt-help').textContent = fill(t.payHelp, { fee: unitsText(entry.feeUnits), symbol: entry.symbol })+' '+flowText(document.documentElement.lang,'life');
 $('usdt-address').textContent = entry.safe;
 const wallet = $('usdt-wallet');
 wallet.textContent = t.openWallet; wallet.href = entry.payLink; wallet.hidden = !entry.payLink.startsWith('ethereum:0x');
 $('usdt-copy').textContent = t.copyAddress;
 $('usdt-warning').textContent = t.keyWarning;
 try { const q = (await qrLibrary())(0, 'M'); q.addData(entry.payLink); q.make(); $('usdt-qr').innerHTML = q.createSvgTag({ cellSize: 4, margin: 2, scalable: true }); } catch { $('usdt-qr').textContent = ''; }
 setStatus(t.waiting);
 watch(entry.id);
}

function watch(id) {
 clearTimeout(timers.get('main'));
 let tries = 0;
 const tick = async () => {
  const entry = readCards().find(c => c.id === id);
  if (!entry) return;
  const stop = await check(entry, ++tries);
  if (!stop) timers.set('main', setTimeout(tick, 6000));
 };
 timers.set('main', setTimeout(tick, 2500));
}

// One look at the card: what the chain says, and, when it is funded, the invitation. Returns true when there is nothing more to watch.
function expiryLabel(entry){const date=typeof entry.created==='number'?entry.created:Date.parse(entry.created||'');return Number.isFinite(date)?' '+flowText(document.documentElement.lang,'validUntil').replace('{date}',new Intl.DateTimeFormat(document.documentElement.lang,{dateStyle:'medium',timeStyle:'short'}).format(date+86400000)):'';}
async function check(entry, tries = 0) {
 const t = words();
 let st;
 try { st = await api('/api/usdt/card/' + entry.id + '/status', {}); } catch (e) { setStatus(errorText(e), 'error'); return false; }
 const needed = entry.units + entry.feeUnits;
 switch (st.state) {
  case 'awaiting_funds': setStatus((st.balance_units || 0) >= needed ? t.seen : t.waiting); return false;
  case 'verifying': setStatus(t.verifying); return false;
  case 'ready': {
   if (entry.invited) { setStatus(fill(t.invited, { email: entry.recipient })+expiryLabel(entry), 'ok'); return true; }
   setStatus(t.funded);
   try { await api('/api/usdt/card/' + entry.id + '/invite', { claim_url: claimURL(entry) }); updateCard(entry.id, { invited: true }); setStatus(fill(t.invited, { email: entry.recipient })+expiryLabel(entry), 'ok'); renderList(); return true; }
   catch (e) { setStatus(e.code === 'rate_limited' ? t.inviteRetry : errorText(e), 'error'); return tries > 20; }
  }
  case 'paid': setStatus(t.paid, 'ok'); return true;
  case 'withdrawn': setStatus(t.withdrawn, 'ok'); return true;
  case 'cancelled': setStatus(t.cancelled); return true;
  case 'expired': setStatus(t.expired); return true;
  default: return false;
 }
}

// ---- taking the money back: the service co-signs, this browser adds the link key's signature ----
async function refund(entry, to, automatic=false) {
 const C = await library(), t = words();
 const r = await api('/api/usdt/card/' + entry.id + '/refund', { to });
 // The fee a refund pays is the flat gas fee fixed at the card's creation: the service may never ask for more.
 const legacy = entry.refundFeeUnits === undefined, flat = FLAT_GAS_FEE[entry.asset];
 const accepted = legacy ? (Number.isSafeInteger(flat) && Number.isSafeInteger(entry.feeUnits) ? Math.min(flat, entry.feeUnits) : NaN) : entry.refundFeeUnits;
 if (!(Number.isSafeInteger(accepted) && accepted > 0 && r.fee_units === accepted && Number.isSafeInteger(r.units) && r.units > 0)) throw fail('invalid_service_answer');
 if (!automatic&&!confirm(fill(t.refundConfirm, { net: unitsText(r.units), fee: unitsText(accepted), symbol: entry.symbol }))) throw fail('cancelled');
 const { id, key } = C.parseFragment(entry.fragment);
 const tx = C.payout({ token: entry.token, recipient: to, relayer: entry.relayer, amount: BigInt(r.units), fee: BigInt(accepted), nonce: BigInt(r.nonce || 0) });
 const mine = await C.signTx(tx, entry.chainId, C.cardAddress({ id, owners: entry.owners, threshold: 2 }, entry.chainId), key);
 const raw = hexBytes(C, r.signature), theirs = { owner: hexBytes(C, r.owner), sig: { r: raw.slice(0, 32), s: raw.slice(32, 64), v: raw[64] } };
 const packed = C.packSignatures([mine, theirs]);
 const result=await api('/api/usdt/card/' + entry.id + '/refund-claim', { signatures: '0x' + C.hex(packed) });
 if(['refunded','refunding','verifying','withdrawn'].includes(result.status||result.state)){try{localStorage.setItem('wire-refund-address:'+entry.chain,to);}catch{}}return result;
}

const expiredRefunds=new Set();
const validRefund=value=>typeof value==='string'&&/^0x[0-9a-fA-F]{40}$/.test(value)&&!/^0x0{40}$/i.test(value);
function knownRefund(c){let saved='';try{saved=localStorage.getItem('wire-refund-address:'+c.chain)||'';}catch{}return validRefund(c.fundingAddress)?c.fundingAddress:validRefund(saved)?saved:'';}
// ---- the list of my cards ----
async function renderList() {
 const t = words(), list = $('usdt-list');
 list.textContent = '';
 let server = [];
 try { server = (await api('/api/usdt/cards')).cards; } catch { /* the local ones are still shown */ }
 const local = readCards(), ids = new Set(local.map(c => c.id));
 const rows = [...local.map(c => ({ ...c, here: true })), ...server.filter(c => !ids.has(c.id)).map(c => ({ ...c, here: false }))];
 $('usdt-history').hidden = rows.length === 0;
 $('usdt-history-title').textContent = t.history;
 for (const c of rows) {
  const row = document.createElement('div'); row.className = 'history-row';
  const label = document.createElement('p'); label.textContent = `${c.amount} ${c.symbol} · ${c.recipient}`; row.append(label);
  const status = document.createElement('small'); row.append(status);
  let currentState;try{currentState=await api('/api/usdt/card/'+c.id+'/status',{});Object.assign(c,{state:currentState.state,balanceUnits:Number(currentState.balance_units??currentState.balanceUnits)||0});if(c.here)updateCard(c.id,c);}catch{}
  const expiredFunds=Boolean(currentState)&&c.state==='expired'&&c.balanceUnits>0&&!currentState.stale;
  if(expiredFunds&&c.here&&!c.recovering&&knownRefund(c)&&!expiredRefunds.has(c.id)){
   expiredRefunds.add(c.id);status.textContent=flowText(document.documentElement.lang,'refundWorking');
   try{const result=await refund(c,knownRefund(c),true);status.textContent=flowText(document.documentElement.lang,result.status==='refunded'?'refundDone':'refundWorking');updateCard(c.id,{recovering:true});}
   catch{status.textContent=flowText(document.documentElement.lang,'refundFailed');}
  }
  const button = (text, run) => { const b = document.createElement('button'); b.type = 'button'; b.textContent = text; b.addEventListener('click', async () => { b.disabled = true; try { await run(); } catch (e) { status.textContent = e.code === 'cancelled' ? '' : errorText(e); } finally { b.disabled = false; } }); row.append(b); return b; };
  button(t.status, async () => { const st = await api('/api/usdt/card/' + c.id + '/status', {}); status.textContent = ({ ready: t.funded, awaiting_funds: t.waiting, paid: t.paid, withdrawn: t.withdrawn, cancelled: t.cancelled, expired: t.expired, verifying: t.verifying })[st.state] || ''; if (c.here && st.state === 'ready' && !c.invited) await check(c); });
  if (c.here && !['paid','withdrawn','cancelled'].includes(c.state)) {
   if(c.state!=='expired')button(t.resend, async () => { await api('/api/usdt/card/' + c.id + '/invite', { claim_url: claimURL(c) }); updateCard(c.id, { invited: true }); status.textContent = fill(t.invited, { email: c.recipient }); });
   const input = document.createElement('input');input.value=knownRefund(c); input.type = 'text'; input.autocomplete = 'off'; input.spellcheck = false; input.placeholder = t.refundTo + ' (0x…)';
   const go = document.createElement('button'); go.type = 'button'; go.textContent = t.refund;
   go.addEventListener('click', async () => { const to = input.value.trim(); if (!/^0x[0-9a-fA-F]{40}$/.test(to)) { status.textContent = t.refundTo; return; } go.disabled = true; try { await refund(c, to); status.textContent = t.refundDone; } catch (e) { status.textContent = e.code === 'cancelled' ? '' : errorText(e); } finally { go.disabled = false; } });
   row.append(input, go);
  } else { const note = document.createElement('small'); note.textContent = t.refundNeedsKey; row.append(note); }
  list.append(row);
 }
}

// ---- the form ----
async function submit() {
 if (busy) return;
 const t = words(), feedback = $('feedback');
 if (!verified) { feedback.textContent = t.verifyFirst; return; }
 const sym = $('currency').value, amount = $('amount').value.trim().replace(',', '.'), n = Number(amount);
 if (!enabledAssets.includes(sym) || !/^\d{1,4}(\.\d{1,6})?$/.test(amount) || !(n >= 1 && n <= 1000) || !$('recipient').checkValidity() || !$('recipient').value) { feedback.textContent = t.invalid; return; }
 busy = true; feedback.textContent = t.working;
 try {
  const entry = await createCard(sym, { amount, recipient: $('recipient').value.trim(), message: $('message').value });
  feedback.textContent = '';
  await showPay(entry);
  await renderList();
 } catch (e) { feedback.textContent = errorText(e); } finally { busy = false; }
}

function refreshText() {
 const t = words();
 $('usdt-history-title').textContent = t.history; $('usdt-copy').textContent = t.copyAddress; $('usdt-wallet').textContent = t.openWallet;
}

window.WireUSDT = {
 // flow.js tells us what the server offers and whether the sender is verified
 configure({ assets, isVerified }) { enabledAssets = (assets || []).filter(a => ASSETS[a]); verified = Boolean(isVerified); if (verified && enabledAssets.length) renderList(); },
 enabled: sym => enabledAssets.includes(sym),
 submit,
 check,
 refund,
};
$('usdt-copy').addEventListener('click', async () => { try { await navigator.clipboard.writeText($('usdt-address').textContent); $('usdt-copy').textContent = words().copied; setTimeout(refreshText, 1400); } catch { /* the address is on screen */ } });
window.addEventListener('wire-language', refreshText);
// flow.js may have learned the server's offer before this module ran: take it from where it left it.
if (window.wireUsdtState) window.WireUSDT.configure(window.wireUsdtState);
