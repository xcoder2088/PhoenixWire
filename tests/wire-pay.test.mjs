import { mountAdmin,adminText } from '../internal/web/static/admin.js';
import { nimFunded,unpaidCard,recoverableCard,historyEntries,historyPending,senderHistoryText,nimHistoryLabel,cancelUnpaidCard } from '../internal/web/static/model.js';
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import { webcrypto } from 'node:crypto';
import { TextEncoder, TextDecoder } from 'node:util';
import { JSDOM } from 'jsdom';
import { flowText } from '../internal/web/static/flow-text.js';
import { addressValid, cardPresentation, paintCard, assetMark, renderCardPreview } from '../internal/web/static/model.js';
import { resolveLanguage } from '../internal/web/static/model.js';

const read = p => fs.readFileSync(new URL(p, import.meta.url), 'utf8');
const tick = () => new Promise(resolve => setImmediate(resolve));
const settle = async () => { for (let i = 0; i < 30; i++) await tick(); };
const proxy = read('../engine/secureclaim/web/usdt.html').match(/PROXY_CREATION_CODE='(0x[0-9a-f]+)'/)[1];
const doms = [];
test('initial navigation stays dark and hides unstyled content until external CSS arrives, without JavaScript',()=>{
 const html=read('../internal/web/static/index.html'),css=read('../internal/web/static/wire-pay.css');
 const page=new JSDOM(html),doc=page.window.document;
 try{
  const scheme=doc.querySelector('meta[name="color-scheme"]');
  assert.equal(scheme.content,'dark');
  assert.ok(scheme.compareDocumentPosition(doc.querySelector('link[rel="stylesheet"]'))&page.window.Node.DOCUMENT_POSITION_FOLLOWING);
  assert.equal(doc.body.hidden,true);
  assert.equal(page.window.getComputedStyle(doc.body).display,'none');
  const style=doc.createElement('style');style.textContent=css;doc.head.append(style);
  assert.equal(page.window.getComputedStyle(doc.body).display,'block');
  assert.equal(page.window.getComputedStyle(doc.documentElement).colorScheme,'dark');
  doc.documentElement.dataset.theme='light-paper';
  assert.equal(page.window.getComputedStyle(doc.documentElement).colorScheme,'light');
  assert.equal(page.window.getComputedStyle(doc.body).display,'block');
  assert.equal(doc.querySelectorAll('script:not([src]),[style]').length,0,'initial paint needs no inline script/style or CSP relaxation');
 }finally{page.window.close();}
});
test('NIM draft shows a safe live card without creating or paying, with one lifetime note',async()=>{
 for(const language of ['fr','en','es','de','zh']){
  const app=await start({nimOnly:true,clocked:true});app.$('wire-language').value=language;app.$('wire-language').dispatchEvent(new app.window.Event('change'));app.$('pay').click();await settle();
  const before=app.calls.length;
  for(const [id,value]of Object.entries({recipient:'demo@example.com',amount:'1,25',message:'<img src=x onerror=alert(1)>'})){
   app.$(id).value=value;app.$(id).dispatchEvent(new app.window.Event('input',{bubbles:true}));
  }
  await settle();assert.equal(app.$('payment-card').hidden,false);assert.equal(app.$('card-amount').textContent,'1.25');assert.equal(app.$('card-recipient').textContent,'demo@example.com');assert.equal(app.$('card-message').children.length,0);assert.equal(app.$('card-message').textContent,'<img src=x onerror=alert(1)>');
  assert.equal(app.doc.querySelectorAll('.wire-nim-window .wire-lifetime-note').length,1);assert.equal(app.calls.length,before);assert.equal(app.checkouts.length,0);
  app.$('amount').value='0';app.$('amount').dispatchEvent(new app.window.Event('input'));assert.equal(app.$('payment-card').hidden,true);
 }
});
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
const handoffToken='Q'.repeat(43);
const handoffDraft={recipient:'friend@example.test',amount:'2.5',message:'Bonjour 🐦',language:'fr'};
async function start({confirmCode='',confirmStatus=409,audioReject=false,maintenance=false,inviteOnly=false,adminAccess=null,cancelCode="",cancelConfirm=true,classic=false,extraNim=[],recoveryStatus=200,holdRecovery=null,nativeError=null,nativeResults=[],holdNative=null,holdNativeAt=2,checkoutCode='',checkoutError=409,startedAgo=0, handoffStatus=200,redeemStatus=200,redeemDraft=handoffDraft,redeemCode='',alreadySignedIn=false,redeemStage='ready',startRace=false,holdRedeem=null,holdHandoff=null,popupBlocked=false, session = { verified: true, email: 'sender@example.test' }, state = () => ({ state: 'awaiting_funds', balanceUnits: 0 }), invite = () => ({ invited: true }), theme = 'dark', nimPending = false, mobile = false, returning = false, nimOnly = false, nativeMode = '', network = 'mainnet', rawMarker = [0,130,128,146,135], android = false, clocked=false, manualNim=false, nimStage='', saved={}, createdAgo=0,local={},serverCards=[],inviteMode='',ownedUnavailable=false,url='https://wire.example.test/',width=1024,gas={polygon:{available:false,reason:'off'},ethereum:{available:false,reason:'off'}} } = {}) {
  const dom = new JSDOM(read(classic?'../internal/web/static/classic.html':'../internal/web/static/index.html'), { url, runScripts: 'outside-only', pretendToBeVisual: true });
  doms.push(dom);
  const { window } = dom, ctx = dom.getInternalVMContext();
  Object.defineProperty(window,'innerWidth',{value:width,configurable:true});
  const clock=clocked?fakeClock(window):null;
  const soundInstances=[],soundPlays=[];window.Audio=class{constructor(src){this.src=src;soundInstances.push(this);}play(){soundPlays.push({src:this.src,volume:this.volume,time:this.currentTime});return audioReject?Promise.reject(Error('not allowed')):Promise.resolve();}};
  for(const[key,value]of Object.entries(saved))window.sessionStorage.setItem(key,value);
  for(const[key,value]of Object.entries(local))window.localStorage.setItem(key,value);
  Object.defineProperty(window, 'crypto', { value: webcrypto, configurable: true });
  const opened=[],consoleMessages=[],redeemHashes=[];
  for(const method of ['log','info','warn','error','debug'])window.console[method]=(...args)=>consoleMessages.push(args);
  Object.assign(window, { TextEncoder, TextDecoder, open: (...args) => {if(popupBlocked)return null;const target={args,location:{},closed:false,close(){this.closed=true;}};opened.push(target);return target;} });
  window.localStorage.setItem('wire-theme', theme);
  if (mobile || android) Object.defineProperty(window.navigator, 'userAgent', { value: android ? 'Android' : 'iPhone' });
  for (const f of ['../engine/secureclaim/web/cashlink.js', '../internal/web/static/pm/usdt-cashlink.js', '../internal/web/static/usdt-refund-text.js', '../internal/web/static/pm/usdt-payments.js', '../internal/web/static/pm/nimiq-payments.js']) vm.runInContext(read(f), ctx);
  const C = window.PhoenixCashlink, calls = [], checkouts = [], behaviors = [], nativeCalls = [], confirmations = [];
  let currentSession=session,codeFails=false,heldCode=null,logoutFails=false;
  const owned=JSON.parse(JSON.stringify(serverCards)).map(c=>({created:new window.Date().toISOString(),...c}));let failStatus=false;
  window.confirm = message => { confirmations.push(message); return cancelConfirm; };
  let nativeActive=0,maxNativeActive=0;
  let fundingVisible = !nativeMode && Boolean(nimPending || returning || nimStage && nimStage!=='ready');
  if(nativeMode){
    window.nimiqPay={network,language:'en'};
    if(nativeMode!=='missing')window.nimiq={async sendBasicTransactionWithData(request){
      // Model the native host: String(data), then BufferUtils.fromUtf8(message).
      const encoded = new TextEncoder().encode(String(request.data ?? ''));
      nativeCalls.push({request,hex:Buffer.from(encoded).toString('hex')});
      assert.equal(request.data, '\u0000\u0082\u0080\u0092\u0087');
      assert.equal(Buffer.from(encoded).toString('hex'), '00c282c280c292c287');
      const outcome=nativeResults[nativeCalls.length-1]||nativeMode;
      nativeActive++;maxNativeActive=Math.max(maxNativeActive,nativeActive);
      try{
      if(holdNative&&nativeCalls.length===holdNativeAt)await new Promise(resolve=>{holdNative.release=resolve;});
      if(outcome==='denied')throw {type:'PERMISSION_DENIED',code:4001};
      if(outcome==='legacy-denied')return {error:{type:'PERMISSION_DENIED'}};
      fundingVisible=outcome==='success'||outcome==='error-after-send';
      if(outcome==='error'||outcome==='error-after-send')throw nativeError||{type:'NETWORK_ERROR'};
      return 'b'.repeat(64);
      }finally{nativeActive--;}
    }};
  }
  const nim = { id: 'a'.repeat(48), sender: 'sender@example.test', recipient: 'friend@example.test', amount: '2.5', luna: 250000, message: 'Hi', address: 'NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY', started_at:new window.Date(window.Date.now()-startedAgo).toISOString(),stage: nimStage || (nimPending || returning ? 'funding_started' : 'ready'),created:new window.Date(window.Date.now()-createdAgo).toISOString() };
  if(['funded','delivery_pending','delivery_failed','sent'].includes(nim.stage))nim.claim_state='unclaimed';
  let nimPayments = [...(nimPending || returning || nimStage || session.handoff ? [nim] : []),...extraNim],holdConfirm=null,activeConfirms=0,maxConfirms=0,mailFails=false,readsFail=false;
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
  const createRequests=new Map();
  const answer = (status, body) => ({ ok: status < 300, status, json: async () => JSON.parse(JSON.stringify(body)) });
  window.fetch = async (path, options = {}) => {
    const body = options.body ? JSON.parse(options.body) : undefined;
    if (path === '/api/gas') return answer(200, gas);
    calls.push([path, body]);
    if(path==='/api/handoff'){
      if(holdHandoff)await new Promise((resolve,reject)=>{holdHandoff.release=resolve;options.signal?.addEventListener('abort',()=>reject(Error('aborted')),{once:true});});
      if(handoffStatus==='network')throw Error('offline');
      return answer(handoffStatus,handoffStatus===200?{token:handoffToken,expires_in:300}:{code:handoffStatus===401?'sender_verification_required':handoffStatus===429?'rate_limited':'invalid_request'});
    }
    if(path==='/api/handoff/redeem'){
      redeemHashes.push(window.location.hash);
      if(holdRedeem)await new Promise((resolve,reject)=>{holdRedeem.release=resolve;options.signal?.addEventListener('abort',()=>reject(Error('aborted')),{once:true});});
      if(redeemStatus!==200)return answer(redeemStatus,{code:redeemCode||'invalid_or_expired'});
      Object.assign(nim,redeemDraft,{stage:redeemStage});nimPayments=[nim];
      if(alreadySignedIn){currentSession={verified:true,email:nim.sender};return answer(200,{payment:nim,already_signed_in:true});}
      currentSession={verified:false,email:'',handoff:{payment_id:nim.id}};
      return answer(200,{payment:nim,scoped:true,expires_at:new window.Date(window.Date.now()+7200000).toISOString()});
    }
    if(currentSession.handoff?.payment_id){
      const id=currentSession.handoff.payment_id;
      if(!['/api/config','/api/session','/api/logout','/api/payments'].includes(path)&&!['art','start','checkout','confirm','resend','cancel'].some(action=>path==='/api/payments/'+id+'/'+action))return answer(403,{code:'handoff_scope'});
      if(path==='/api/payments'&&options.method)return answer(403,{code:'handoff_scope'});
    }
    if(path.startsWith('/api/admin/'))return answer(200,adminAccess||{open:!inviteOnly,allowed:[],blocked:[],registered:['sender@example.test'],admins:['sender@example.test']});
    if (path === '/api/config') return answer(200, { mode: 'integrated', payments_enabled: true, invite_only:inviteOnly, maintenance, assets: nimOnly ? ['NIM'] : ['NIM', 'USDC_POLYGON', 'USDT_POLYGON', 'USDT_ETHEREUM'] });
    if (path === '/api/session') return answer(200, currentSession);
    if(path==='/api/usdt/cards')return ownedUnavailable?answer(503,{code:'unavailable'}):answer(200,{cards:owned});
    if (path === '/api/payments' && !options.method) return readsFail?answer(503,{code:'chain_unavailable'}):answer(200, { payments: nimPayments });
    if (path === '/api/payments') {
      if(createRequests.has(body.request_id))return answer(201,{payment:createRequests.get(body.request_id)});
      if(nimPayments.some(p=>p.id===nim.id)){nimPayments=nimPayments.map(p=>({...p}));nim.id=('c'+createRequests.size.toString(16)).padEnd(48,'0');}
      Object.assign(nim,{recipient:body.recipient,amount:body.amount,luna:Math.round(Number(body.amount)*100000),message:body.message,language:body.language,stage:'ready',created:new window.Date().toISOString()});
      nimPayments=[nim,...nimPayments.filter(p=>p.id!==nim.id)];createRequests.set(body.request_id,nim);return answer(201,{payment:nim});
    }
    if(path.endsWith('/cancel')){if(cancelCode){if(cancelCode==='already_funded'){fundingVisible=true;nim.stage='funded';nim.claim_state='unclaimed';}return answer(cancelCode==='chain_unavailable'?503:409,{code:cancelCode});}nim.stage='cancelled';return answer(200,{payment:nim});}
    if (path.endsWith('/art')) return answer(200, { saved: true });
    if (path.endsWith('/start')) {if(startRace){nim.stage='funding_started';fundingVisible=true;return answer(409,{code:'funding_already_started'});}if(nim.stage!=='ready')return answer(409,{code:'funding_already_started'});nim.stage = 'funding_started';nim.started_at=new window.Date().toISOString(); return answer(200, { payment: nim, checkout: { appName: 'Phoenix Wire', recipient: nim.address, value: nim.luna, recipientData: rawMarker } }); }
    if(path.endsWith('/checkout')){
      if(checkoutCode){if(checkoutCode==='already_funded')fundingVisible=true;if(checkoutCode==='funding_not_started')nim.stage='ready';return answer(checkoutError,{code:checkoutCode});}
      if(nim.stage!=='funding_started')return answer(409,{code:'funding_not_started'});
      if(window.Date.now()-Date.parse(nim.started_at)<180000)return answer(409,{code:'retry_too_soon'});
      if(fundingVisible)return answer(409,{code:'already_funded'});
      return answer(200,{payment:nim,checkout:{appName:'Phoenix Wire',recipient:nim.address,value:nim.luna,recipientData:rawMarker}});
    }
    if (path.endsWith('/confirm')) {
      if(confirmCode)return answer(confirmStatus,{code:confirmCode});
      activeConfirms++;maxConfirms=Math.max(maxConfirms,activeConfirms);
      try{
      if(nim.stage==='ready')return answer(409,{code:'funding_not_started'});
      if(holdConfirm){const held=holdConfirm;holdConfirm=null;await new Promise((resolve,reject)=>{held.release=resolve;options.signal?.addEventListener('abort',()=>reject(Error('aborted')),{once:true});});}
      if(manualNim){
        if(nim.stage==='ready'||nim.stage==='funding_started')return answer(409,{code:'funding_pending'});
        if(nim.stage==='funded')nim.stage='delivery_pending';
        return answer(200,{payment:nim});
      }
      if(nativeMode==='success-no-confirm')return answer(503,{code:'chain_unavailable'});
      if(!fundingVisible)return answer(409,{code:'funding_pending'});
      nim.stage = 'sent'; nim.claim_state='unclaimed'; return answer(200, { payment: nim });
      }finally{activeConfirms--;}
    }
    if(path.endsWith('/resend')){nim.stage=mailFails?'delivery_failed':'sent';return mailFails?answer(502,{code:'email_delivery_unavailable'}):answer(200,{payment:nim});}
    if (path.endsWith('/recovery')) {
      if(holdRecovery)await new Promise((resolve,reject)=>{holdRecovery.release=resolve;options.signal?.addEventListener('abort',()=>reject(Error('aborted')),{once:true});});
      return answer(recoveryStatus,recoveryStatus===200?{cashlink:'https://hub.nimiq.com/cashlink/#'+Buffer.alloc(40,5).toString('base64url')}:{code:'recovery_unavailable'});
    }
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
    if(path==='/api/sender/request'){
     if(heldCode){const held=heldCode;heldCode=null;await new Promise((resolve,reject)=>{held.release=resolve;options.signal?.addEventListener('abort',()=>reject(Error('aborted')),{once:true});});}
     return codeFails==='maintenance'?answer(503,{code:'maintenance'}):codeFails==='wait'?answer(429,{code:'wait_before_resend',retry_after:90}):codeFails==='invite_only'?answer(403,{code:'invite_only'}):codeFails?answer(503,{code:'mail_unavailable'}):answer(200,{email:'sender@example.test',ok:true});
    }
    if (path === '/api/logout') {if(logoutFails)return answer(503,{code:'unavailable'});currentSession={verified:false};return answer(200, {ok: true});}
    throw Error('unexpected API ' + path);
  };
  const source = read('../internal/web/static/wire-pay.js').replace(/^import[^\n]+\n/gm, '');
  Object.assign(ctx, { nimFunded,unpaidCard:p=>unpaidCard(p,window.Date.now()),recoverableCard,historyEntries:entries=>historyEntries(entries,window.Date.now()),historyPending:p=>historyPending(p,window.Date.now()),senderHistoryText,nimHistoryLabel:(p,language)=>nimHistoryLabel(p,language,window.Date.now()),cancelUnpaidCard,mountAdmin,adminText,  resolveLanguage, flowText, addressValid, cardPresentation, paintCard, assetMark, renderCardPreview });
  vm.runInContext(read('../internal/web/static/flow.js').replace(/^import[^\n]+\n/gm, '').replace(/^export /gm, ''), ctx);
  if(!classic)vm.runInContext('{' + source + '\n}', ctx);
  await settle();
  return { window, soundInstances,soundPlays, $: id => window.document.getElementById(id), calls, doc: window.document, checkouts, behaviors, nativeCalls, confirmations,clock,
    fundOnChain(){fundingVisible=true;},setClaimState(value){if(value===undefined)delete nim.claim_state;else nim.claim_state=value;},setStage(stage){nim.stage=stage;if(['funded','delivery_pending','delivery_failed','sent'].includes(stage))nim.claim_state='unclaimed';},failMail(value){mailFails=value;},failReads(value){readsFail=value;},holdNextConfirm(){const held={};holdConfirm=held;return held;},maxConfirms:()=>maxConfirms,
    failCode(value){codeFails=value;},holdCode(){const held={};heldCode=held;return held;},
    failLogout(value){logoutFails=value;},
    opened,consoleMessages,redeemHashes,maxNativeActive:()=>maxNativeActive,setRecoveryStatus(value){recoveryStatus=value;},
    snapshot(){return Object.fromEntries(Object.keys(window.sessionStorage).map(k=>[k,window.sessionStorage.getItem(k)]));},
    snapshotLocal(){return Object.fromEntries(Object.keys(window.localStorage).map(k=>[k,window.localStorage.getItem(k)]));},serverCards:owned,failCardStatus(value){failStatus=value;}};
}

async function makeCard(t) {
  t.$('pay').click(); await settle();
  [...t.doc.querySelectorAll('.wire-coin-picker .nimiq-payment-assets button')].find(b=>b.textContent==='USDC · Polygon')?.click();await settle();
  const form = t.doc.querySelector('.nimiq-payment-panel');
  const input = label => [...form.querySelectorAll('input')].find(i => i.closest('label') && i.closest('label').textContent.includes(label));
  const set = (el, v) => { el.value = v; el.dispatchEvent(new t.window.Event('input', { bubbles: true })); el.dispatchEvent(new t.window.Event('change', { bubbles: true })); };
  set(input('Amount'), '2.5'); set(input('Message'), 'merci'); set(input('Recipient'), 'friend@example.test');
  [...form.querySelectorAll('button')].find(b => b.textContent === 'Create card').click();
  await settle();
}

for(const language of ['fr','en','es','de','zh'])for(const [asset,label,symbol,network]of [['usdc_polygon','USDC · Polygon','USDC','POLYGON'],['usdt_polygon','USDT · Polygon','USDT','POLYGON'],['usdt_ethereum','USDT · Ethereum','USDT','ETHEREUM']]){
 test(`local stablecoin preview follows every keystroke without API or wallet / ${asset} / ${language}`,async()=>{
  const app=await start({clocked:true});app.$('wire-language').value=language;app.$('wire-language').dispatchEvent(new app.window.Event('change'));app.$('pay').click();await settle();
  [...app.doc.querySelectorAll('.wire-coin-picker .nimiq-payment-assets button')].find(b=>b.textContent===label).click();await settle();
  const win=app.doc.querySelector('.nimiq-payment-window'),form=win.querySelector('.nimiq-payment-panel'),card=win.querySelector('.wire-local-card'),field=name=>card.querySelector('[data-card-part="'+name+'"]');
  assert.ok(card);assert.ok(win.classList.contains('wire-has-local-preview'));assert.equal(field('card-amount').textContent,'0');assert.equal(field('card-unit').textContent,symbol);assert.equal(card.querySelector('.sample').textContent,cardPresentation(language).protected);assert.match(card.querySelector('.wire-nim-brand').getAttribute('src'),/^\/brand\/phoenix-wire.webp/);assert.equal(card.querySelector('.card-heading').textContent,symbol+' · '+network+' · '+cardPresentation(language).heading.toLocaleUpperCase(language));
  const before=app.calls.length,opened=app.opened.length,amount=form.querySelector('input[inputmode="decimal"]'),message=form.querySelector('input[maxlength="140"]'),recipient=form.querySelector('input[type="email"]');
  const input=(el,value)=>{el.value=value;el.dispatchEvent(new app.window.Event('input',{bubbles:true}));};
  for(const invalid of ['', '0','-1','1e2','Infinity','0.001','1000.1','2.1234567','<b>2</b>']){input(amount,invalid);assert.equal(field('card-amount').textContent,'0');}
  input(amount,'2');assert.equal(field('card-amount').textContent,'2');input(amount,'2.123456');assert.equal(field('card-amount').textContent,'2.123456');
  const inert='<img src=x onerror="alert(1)"><script>alert(2)</script>';input(message,inert);input(recipient,'demo@example.com');
  assert.equal(field('card-message').textContent,inert);assert.equal(field('card-message').children.length,0);assert.equal(card.querySelector('script,[onerror]'),null);assert.equal(field('card-recipient').textContent,'demo@example.com');assert.match(field('coin').querySelector('img').getAttribute('src'),new RegExp(asset.replace('_','-')+'\\.png$'));
  assert.equal(field('coin').querySelector('img').alt,assetMark(asset,language).alt);
  assert.equal(card.querySelector('.lock').textContent,flowText(language,'cardCode'));assert.equal(win.querySelectorAll('.wire-lifetime-note').length,1);assert.equal(card.querySelector('.wire-lifetime-note').textContent,flowText(language,'cardLife'));assert.equal(win.textContent.split(flowText(language,'cardLife')).length-1,1);assert.equal(card.querySelectorAll('[id]').length,0);assert.doesNotMatch(card.textContent,/Mainnet|OTP|Hub/);
  await settle();assert.equal(app.calls.length,before);assert.equal(app.opened.length,opened);assert.equal(app.checkouts.length,0);assert.equal(app.nativeCalls.length,0);assert.equal(app.window.localStorage.getItem('phoenixmail.usdt.sent.v2'),null);
 });
}
test('stablecoin draft refreshes its language without resetting fields and stays visible on the mobile form view',async()=>{
 const app=await start({clocked:true,width:375});app.$('pay').click();await settle();[...app.doc.querySelectorAll('.wire-coin-picker .nimiq-payment-assets button')].find(b=>b.textContent==='USDC · Polygon').click();await settle();
 const win=app.doc.querySelector('.nimiq-payment-window'),amount=win.querySelector('input[inputmode="decimal"]');amount.value='7';amount.dispatchEvent(new app.window.Event('input',{bubbles:true}));const card=win.querySelector('.wire-local-card'),color=card.style.getPropertyValue('--card-bg');
 for(const language of ['fr','en','es','de','zh']){app.$('wire-language').value=language;app.$('wire-language').dispatchEvent(new app.window.Event('change'));await settle();assert.equal(win.querySelector('.wire-local-card'),card);assert.equal(card.querySelector('.lock').textContent,flowText(language,'cardCode'));assert.equal(card.querySelector('[data-card-part="card-amount"]').textContent,'7');assert.equal(card.style.getPropertyValue('--card-bg'),color);assert.equal(win.querySelectorAll('.wire-lifetime-note').length,1);}
 assert.equal(win.dataset.view,'form');const css=read('../internal/web/static/wire-pay.css');assert.match(css,/\.wire-body \.wire-has-local-preview\[data-view="form"\] \.nimiq-payment-preview\{display:block\}/);assert.match(css,/@media\(max-width:680px\)\{\.wire-has-local-preview \.nimiq-payment-body\{grid-template-columns:minmax\(0,1fr\)\}/);
});
test('network marks identify the asset picker, native server card, sent card and history in all languages',async()=>{
 for(const language of ['fr','en','es','de','zh']){
  const app=await start({clocked:true,state:()=>({state:'ready',balanceUnits:2650000})});app.$('wire-language').value=language;app.$('wire-language').dispatchEvent(new app.window.Event('change'));app.$('pay').click();await settle();
  const icons=[...app.doc.querySelectorAll('.nimiq-payment-assets .wire-coin-icon')];
  for(const asset of ['usdc_polygon','usdt_polygon','usdt_ethereum']){const image=icons.find(i=>i.dataset.asset===asset);assert.equal(image.getAttribute('src'),assetMark(asset,language).src);assert.equal(image.alt,assetMark(asset,language).alt);}
  assert.notEqual(icons.find(i=>i.dataset.asset==='usdt_polygon').src,icons.find(i=>i.dataset.asset==='usdt_ethereum').src);
  // The service fixture creates a USDC card; check the mirrored artwork is overridden before and after funding.
  app.$('wire-language').value='en';app.$('wire-language').dispatchEvent(new app.window.Event('change'));await makeCard(app);
  app.$('wire-language').value=language;app.$('wire-language').dispatchEvent(new app.window.Event('change'));await app.clock.advance(3000);
  const server=app.doc.querySelector('.nimiq-payment-preview .usdt-coin');assert.ok(server);assert.equal(server.getAttribute('src'),'/brand/usdc-polygon.png');assert.equal(server.alt,assetMark('usdc_polygon',language).alt);assert.ok(app.doc.querySelector('.usdt-funded'));
  const rows=JSON.parse(app.window.localStorage.getItem('wire-usdt-cards')),original=rows[0];
  app.window.localStorage.setItem('wire-usdt-cards',JSON.stringify(['usdc_polygon','usdt_polygon','usdt_ethereum'].map((asset,i)=>({...original,id:String(i+1).repeat(32),asset,created:app.clock.now()}))));app.window.dispatchEvent(new app.window.StorageEvent('storage',{key:'wire-usdt-cards'}));await settle();
  for(const asset of ['usdc_polygon','usdt_polygon','usdt_ethereum']){const image=app.doc.querySelector('#wire-cards-list img[data-asset="'+asset+'"]');assert.ok(image);assert.equal(image.getAttribute('src'),assetMark(asset,language).src);assert.equal(image.alt,assetMark(asset,language).alt);}
 }
});
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
  assert.equal(t.doc.querySelector('.wire-local-card'),null,'the server card replaces the local draft');assert.equal(t.doc.querySelector('.wire-has-local-preview'),null);assert.equal(t.doc.querySelectorAll('.nimiq-payment-window .wire-lifetime-note').length,1);
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
  assert.equal(t.doc.querySelector('.wire-navigation'),null);
  assert.equal(t.doc.querySelector('footer .wire-powered').href,'https://phoenixmail.ca/');
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
    assert.equal(doc.querySelector('.wire-navigation'),null);assert.ok(doc.querySelector('footer .wire-powered'));
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
  assert.equal(t.$('feedback').textContent.split('\n')[0], 'Card sent to friend@example.test. It stays yours until it is claimed.');
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
  assert.equal(t.window.getComputedStyle(t.doc.documentElement).getPropertyValue('--panel').trim(), '#fff9f3');
  t.$('wire-theme').click();
  assert.equal(t.window.getComputedStyle(t.doc.documentElement).getPropertyValue('--panel').trim(), '#1b1e22');
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
  assert.equal(link.getAttribute('href'),'#');
  assert.equal(t.calls.filter(([p])=>p==='/api/handoff').length,0);
  assert.equal(link.target,'_blank');assert.match(link.rel,/noopener/);assert.match(link.rel,/noreferrer/);
  link.click();link.click();await settle();
  assert.equal(t.calls.filter(([p])=>p==='/api/handoff').length,1);
  const destination=new URL(new URL(t.opened.at(-1).location.href).searchParams.get('url'));
  assert.equal(destination.origin,'https://wire.phoenixmail.ca');assert.equal(destination.hash,'#h='+handoffToken);
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
    fr:['Payer avec Nimiq Pay','Obtenir Nimiq Pay','Portefeuille Nimiq','Annuler'],
    en:['Pay with Nimiq Pay','Get Nimiq Pay','Nimiq Wallet','Cancel'],
    es:['Pagar con Nimiq Pay','Obtener Nimiq Pay','Cartera Nimiq','Cancelar'],
    de:['Mit Nimiq Pay bezahlen','Nimiq Pay herunterladen','Nimiq Wallet','Abbrechen'],
    zh:['使用 Nimiq Pay 付款','获取 Nimiq Pay','Nimiq 钱包','取消']
  };
  for(const [lang,labels]of Object.entries(expected)){
    t.$('wire-language').value=lang;t.$('wire-language').dispatchEvent(new t.window.Event('change'));
    assert.deepEqual(['open','get','hub','cancel'].map(id=>t.$('wire-choice-'+id).textContent),labels);
    assert.ok(t.$('wire-choice-help').textContent);
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
  assert.ok(t.calls.some(([p])=>p.endsWith('/confirm')));assert.equal(t.$('feedback').textContent.split('\n')[0],'Card sent to friend@example.test. It stays yours until it is claimed.');
  assert.equal(t.checkouts.length,0);assert.equal(t.window.sessionStorage.getItem('wire-pending-payment'),'');
  t.$('card-form').dispatchEvent(new t.window.Event('submit',{cancelable:true}));await settle();
  assert.equal(t.nativeCalls.length,1);assert.equal(t.calls.filter(([p])=>p.endsWith('/start')).length,1);
});

test('a returned native hash alone is never shown as a successful card payment',async()=>{
  const t=await start({nimOnly:true,nativeMode:'success-no-confirm'});await submitNim(t);
  assert.equal(t.nativeCalls.length,1);assert.notEqual(t.$('feedback').textContent.split('\n')[0],'Card sent to friend@example.test. It stays yours until it is claimed.');
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
  const t=await start({nimOnly:true,nativeMode:'error',clocked:true});await submitNim(t);
  await t.clock.advance(180000);
  assert.ok(t.calls.some(([p])=>p.endsWith('/confirm')));assert.equal(t.checkouts.length,0);
  const fallback=t.doc.querySelector('.wire-nim-hub-fallback');assert.equal(fallback.hidden,false);
  fallback.click();await settle();
  assert.equal(t.confirmations.length,1);assert.equal(t.checkouts.length,1);
  assert.equal(t.checkouts[0].recipient,t.nativeCalls[0].request.recipient);
  assert.equal(t.checkouts[0].value,t.nativeCalls[0].request.value);
  assert.deepEqual(Array.from(t.checkouts[0].extraData),[0,130,128,146,135]);
  assert.equal(t.calls.filter(([p])=>p.endsWith('/start')).length,1);
  assert.equal(t.calls.filter(([p,b])=>p==='/api/payments'&&b).length,1);
  assert.equal(t.$('feedback').textContent.split('\n')[0],'Card sent to friend@example.test. It stays yours until it is claimed.');
});

test('an error after native broadcast, or later funding before the fallback click, never launches Hub',async()=>{
  const sent=await start({nimOnly:true,nativeMode:'error-after-send'});await submitNim(sent);
  assert.equal(sent.$('feedback').textContent.split('\n')[0],'Card sent to friend@example.test. It stays yours until it is claimed.');assert.equal(sent.checkouts.length,0);
  assert.equal(sent.doc.querySelector('.wire-nim-hub-fallback').hidden,true);
  const late=await start({nimOnly:true,nativeMode:'error'});await submitNim(late);
  late.fundOnChain();late.doc.querySelector('.wire-nim-hub-fallback').click();await settle();
  assert.equal(late.checkouts.length,0);assert.equal(late.confirmations.length,0);
  assert.equal(late.$('feedback').textContent.split('\n')[0],'Card sent to friend@example.test. It stays yours until it is claimed.');
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

test('NIM tracking continues beyond ten minutes, then stops at sent with an explicit Done action',async()=>{
  const t=await tracked();
  await t.clock.advance(12*60000);
  assert.ok(checks(t)>20,'the old twenty-check limit is gone');
  assert.equal(t.checkouts.length,0);assert.equal(t.nativeCalls.length,0);
  t.setStage('sent');await t.clock.advance(30000);
  assert.equal(t.$('nim-progress-message').textContent.split('\n')[0],'Card sent to friend@example.test. It stays yours until it is claimed.');
  assert.equal(t.$('nim-progress-close').hidden,false);assert.equal(t.$('nim-progress-close').textContent,'Done');
  assert.equal(t.$('nim-progress-resend').hidden,true);assert.ok(!t.$('payment-list').textContent.includes('Resend'));
  const count=t.calls.length,confirmed=checks(t);await t.clock.advance(10*60000);
  t.window.dispatchEvent(new t.window.Event('pageshow'));await settle();assert.equal(checks(t),confirmed,'sent cards do not request funding confirmation again');assert.ok(t.calls.length>count,'sent cards continue checking claim state');
  t.$('nim-progress-close').click();assert.equal(t.doc.querySelector('.wire-nim-window'),null);
  assert.match(t.$('wire-cards-list').textContent,/Sent, not yet received/);
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

test('an approved payment shows the waiting status in each language; funded, sent, Done and Resend are translated',async()=>{
  const t=await tracked({saved:{['wire-nim-approved:'+'a'.repeat(48)]:'yes'}});
  const expected={
    fr:['Paiement envoyé. Confirmation sur le réseau en cours…','Paiement confirmé. Envoi de la carte à friend@example.test…','Carte envoyée à friend@example.test. Elle reste à vous tant qu’elle n’est pas réclamée.','Terminé','Renvoyer'],
    en:['Payment sent. Waiting for network confirmation…','Payment confirmed. Sending the card to friend@example.test…','Card sent to friend@example.test. It stays yours until it is claimed.','Done','Resend'],
    es:['Pago enviado. Esperando la confirmación de la red…','Pago confirmado. Enviando la tarjeta a friend@example.test…','Tarjeta enviada a friend@example.test. Sigue siendo tuya hasta que se reclame.','Listo','Reenviar'],
    de:['Zahlung gesendet. Netzwerkbestätigung ausstehend…','Zahlung bestätigt. Die Karte wird an friend@example.test gesendet…','Karte an friend@example.test gesendet. Sie bleibt deine, bis sie eingelöst wird.','Fertig','Erneut senden'],
    zh:['付款已发送。正在等待网络确认…','付款已确认。正在向 friend@example.test 发送卡片…','卡片已发送至 friend@example.test。在被领取之前它仍属于你。','完成','重新发送']
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
  for(const[lang,labels]of Object.entries(expected)){t.$('wire-language').value=lang;t.$('wire-language').dispatchEvent(new t.window.Event('change'));assert.equal(t.$('nim-progress-message').textContent.split('\n')[0],labels[2]);}
});

test('iPhone suspended timers resume immediately on visibilitychange and pageshow, without opening a wallet',async()=>{
  const t=await tracked();const before=checks(t);
  t.clock.hidden(true);t.clock.suspend(10*60000);
  assert.equal(checks(t),before);
  t.clock.hidden(false);await settle();assert.equal(checks(t),before+1);
  t.clock.suspend(5*60000);t.window.dispatchEvent(new t.window.Event('pageshow'));await settle();
  assert.equal(checks(t),before+2);
  t.setStage('sent');t.window.dispatchEvent(new t.window.Event('pageshow'));await settle();
  assert.equal(t.doc.querySelector('.wire-nim-window'),null);
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
  t.setStage('sent');await t.clock.advance(10000);assert.match(t.$('wire-cards-list').textContent,/Sent, not yet received/);
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
  assert.match(exp.$('nim-progress-message').textContent,/expired|24-hour expiry/);const count=exp.calls.length;
  exp.window.dispatchEvent(new exp.window.Event('pageshow'));await exp.clock.advance(10*60000);
  assert.ok(exp.calls.length>count,'foreground checks for a server refund after expiry');assert.equal(exp.nativeCalls.length,0);assert.equal(exp.checkouts.length,0);
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
  t.setStage('sent');await t.clock.advance(10000);assert.match(t.$('wire-cards-list').textContent,/Sent, not yet received/);
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
    assert.ok(t.doc.querySelector('.wire-card-banner p').textContent.startsWith(w[9]));
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
  assert.match(second.$('wire-cards-list').textContent,/Card sent to/);assert.equal(second.$('wire-card-notices').hidden,false);
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
  assert.match(t.$('wire-cards-list').textContent,/Card sent to/);
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
  assert.ok([...t.$('wire-cards-list').querySelectorAll('img')].every(img=>img.className==='wire-coin-icon'&&img.getAttribute('src').startsWith('/brand/')));assert.equal(t.$('wire-card-notices').querySelector('img'),null);
  assert.match(t.$('wire-cards-list').textContent,/2.5 NIM · Nimiq/);
  assert.ok(!t.calls.some(([p,b])=>p==='/api/payments'&&b));assert.equal(t.checkouts.length,0);assert.equal(t.nativeCalls.length,0);
});

test('the page scrolls: PhoenixMail’s stylesheet locks html, Wire’s frame frees it again', () => {
  const css = read('../internal/web/static/wire-pay.css'), app = read('../internal/web/static/pm/app.css');
  assert.match(app, /html\{overflow:hidden/, 'the copied stylesheet does lock the page (its mail app scrolls inside itself)');
  assert.match(css, /html\{[^}]*height:auto!important[^}]*overflow-y:auto!important/, 'Wire’s page is a normal page: it must scroll on a phone');
});

test('after too many codes the page counts the wait down, and the button comes back by itself',async()=>{
 const t=await start({session:{verified:false},clocked:true});
 t.failCode('wait');t.$('wire-email').value='sender@example.test';
 const button=t.$('wire-send-code');button.click();await settle();
 assert.equal(button.disabled,true);assert.match(button.textContent,/1:30/);assert.match(t.$('wire-status').textContent,/1:30/);
 await t.clock.advance(31000);assert.match(button.textContent,/0:59/);assert.match(t.$('wire-status').textContent,/0:59/);
 button.click();await settle();assert.equal(t.calls.filter(([p])=>p==='/api/sender/request').length,1);
 await t.clock.advance(60000);assert.equal(button.disabled,false);assert.equal(button.textContent,'Send me a code');assert.equal(t.$('wire-status').textContent,'');
 t.failCode('');button.click();await settle();assert.equal(t.calls.filter(([p])=>p==='/api/sender/request').length,2);
});
test('maintenance tells visitors the site is closed, in five languages, and shows nothing to an administrator',async()=>{
 for(const language of ['fr','en','es','de','zh']){
  const app=await start({maintenance:true,session:{verified:false},local:{'wire-language':language},clocked:true});
  const banner=app.$('wire-maintenance');assert.equal(banner.hidden,false);assert.equal(banner.textContent,flowText(language,'maintenance'));
  app.failCode('maintenance');app.$('wire-email').value='sender@example.test';app.$('wire-send-code').click();await settle();
  assert.equal(app.$('wire-status').textContent,flowText(language,'maintenance'));
 }
 const open=await start({session:{verified:false},clocked:true});assert.equal(open.$('wire-maintenance').hidden,true);
 const admin=await start({maintenance:true,session:{verified:true,email:'sender@example.test',admin:true},adminAccess:{open:true,allowed:[],blocked:[],admins:['sender@example.test'],maintenance:true,registered_count:1},clocked:true});await settle();
 assert.equal(admin.$('wire-maintenance').hidden,true);assert.ok(admin.doc.getElementById('wire-admin-maintenance'));
});
test('code request visibly changes from sending to sent in five languages and blocks duplicate sends',async()=>{
 const labels={fr:['Envoi du code…','Code envoyé','Renvoyer un code'],en:['Sending code…','Code sent','Send another code'],es:['Enviando código…','Código enviado','Enviar otro código'],de:['Code wird gesendet…','Code gesendet','Weiteren Code senden'],zh:['正在发送验证码…','验证码已发送','重新发送验证码']};
 for(const[language,text]of Object.entries(labels)){
  const t=await start({session:{verified:false},clocked:true});
  t.$('wire-language').value=language;t.$('wire-language').dispatchEvent(new t.window.Event('change'));
  t.$('wire-email').value='sender@example.test';const held=t.holdCode();
  const button=t.$('wire-send-code');button.click();button.click();await settle();
  assert.equal(button.textContent,text[0]);assert.equal(button.disabled,true);assert.equal(button.dataset.state,'sending');
  assert.equal(t.$('wire-code-box').hidden,true);assert.equal(t.$('wire-email').disabled,true);
  assert.equal(t.calls.filter(([p])=>p==='/api/sender/request').length,1);
  held.release();await settle();
  assert.ok(button.textContent.startsWith(text[1]));assert.equal(button.dataset.state,'sent');assert.equal(button.disabled,true);
  assert.equal(t.$('wire-code-box').hidden,false);assert.equal(t.$('wire-code-delivery').hidden,false);assert.equal(t.doc.activeElement,t.$('wire-code'));
  button.click();await settle();assert.equal(t.calls.filter(([p])=>p==='/api/sender/request').length,1);
  await t.clock.advance(30000);assert.equal(button.textContent,text[2]);assert.equal(button.disabled,false);
  t.$('wire-email').value='other@example.test';t.$('wire-email').dispatchEvent(new t.window.Event('input'));
  assert.equal(button.dataset.state,'idle');assert.equal(t.$('wire-code-delivery').hidden,true);assert.equal(t.$('wire-code-box').hidden,true);
 }
 assert.match(read('../internal/web/static/wire-pay.css'),/#wire-send-code\[data-state="sent"\]\{[^}]*border-color:#369d70/);
});

const inviteOnlyCopy={
 fr:"Phoenix Wire est ouvert sur invitation pour le moment. Tu peux quand même ouvrir une carte que quelqu'un t'a envoyée.",
 en:'Phoenix Wire is invite-only for now. You can still open a card someone has sent you.',
 es:'Por ahora, Phoenix Wire está disponible solo por invitación. Aun así, puedes abrir una tarjeta que alguien te haya enviado.',
 de:'Phoenix Wire ist zurzeit nur auf Einladung verfügbar. Du kannst weiterhin eine Karte öffnen, die dir jemand geschickt hat.',
 zh:'Phoenix Wire 目前仅限受邀用户发送。你仍然可以打开别人发给你的卡片。'
};
test('public invitation access starts folded and existing senders can expand it in both interfaces',async()=>{
 for(const classic of [false,true])for(const language of ['fr','en','es','de','zh']){
  const app=await start({inviteOnly:true,classic,clocked:true,session:{verified:false},local:{'wire-language':language}});
  if(classic){app.doc.documentElement.lang=language;app.window.dispatchEvent(new app.window.Event('wire-language'));}
  const gate=app.$(classic?'sender-invite-gate':'wire-invite-gate'),link=app.$(classic?'sender-existing-access':'wire-existing-access'),input=app.$(classic?'sender':'wire-email');
  assert.equal(gate.hidden,false);assert.ok(gate.textContent.includes(flowText(language,'inviteOnly')));assert.equal(link.textContent,adminText(language,'already'));assert.equal(classic?input.closest('label').hidden:app.$('wire-sender').hidden,true);assert.equal(app.calls.some(([p])=>p==='/api/sender/request'),false);
  link.click();assert.equal(classic?input.closest('label').hidden:app.$('wire-sender').hidden,false);assert.equal(link.getAttribute('aria-expanded'),'true');assert.equal(app.doc.activeElement,input);assert.equal(input.disabled,false);
 }
});
test('open access presents email immediately; only a server-designated signed-in admin gets the panel',async()=>{
 for(const classic of [false,true]){
  const publicApp=await start({classic,clocked:true,inviteOnly:false,session:{verified:false}});assert.equal(publicApp.$(classic?'sender-invite-gate':'wire-invite-gate').hidden,true);assert.equal(publicApp.doc.getElementById('wire-admin'),null);assert.equal(classic?publicApp.$('sender').closest('label').hidden:publicApp.$('wire-sender').hidden,false);
  const member=await start({classic,clocked:true});assert.equal(member.doc.getElementById('wire-admin'),null);assert.equal(member.calls.some(([p])=>p.startsWith('/api/admin/')),false);
  const admin=await start({classic,clocked:true,session:{verified:true,email:'sender@example.test',admin:true}});assert.ok(admin.doc.getElementById('wire-admin'));assert.equal(admin.doc.getElementById('wire-admin-open').checked,false);
 }
});
test('24-hour sender terms and local completion deadlines cover NIM and stablecoin windows in five languages',async()=>{
 for(const language of ['fr','en','es','de','zh']){
  const app=await start({clocked:true,nimOnly:true,nativeMode:'success'});app.$('wire-language').value=language;app.$('wire-language').dispatchEvent(new app.window.Event('change'));app.$('pay').click();await settle();
  assert.equal(app.doc.querySelector('.wire-nim-host .wire-lifetime-note').textContent,flowText(language,'life'));
  app.$('recipient').value='friend@example.test';app.$('amount').value='2.5';const created=app.clock.now();app.$('preview-submit').click();await settle();await app.clock.advance(6000);
  const date=new Intl.DateTimeFormat(language,{dateStyle:'medium',timeStyle:'short'}).format(created+86400000);assert.ok(app.$('nim-progress-message').textContent.includes(flowText(language,'validUntil').replace('{date}',date)));
  const stable=await start({clocked:true});stable.$('wire-language').value=language;stable.$('wire-language').dispatchEvent(new stable.window.Event('change'));stable.$('pay').click();await settle();assert.ok([...stable.doc.querySelectorAll('.nimiq-payment-window .wire-lifetime-note')].some(n=>n.textContent===flowText(language,'life')));
 }
});
test('expired stablecoins show one address field and remember a manually confirmed refund',async()=>{
 let expired=false;const app=await start({clocked:true,state:()=>({state:expired?'expired':'awaiting_funds',balanceUnits:expired?2650000:0})});await makeCard(app);expired=true;
 const list=JSON.parse(app.window.localStorage.getItem('wire-usdt-cards')),id=list[0].id;list[0].created=app.clock.now()-90000000;app.window.localStorage.setItem('wire-usdt-cards',JSON.stringify(list));
 const calls=[],original=app.window.PhoenixUSDTCashlink.create;app.window.PhoenixUSDTCashlink.create=options=>{const cl=original(options);cl.refund=async(entry,to,settings)=>{calls.push(to);assert.equal(entry.card.id,id);assert.ok(settings.confirm({units:2500000,fee:150000}));return{status:'refunded'};};return cl;};
 await app.clock.advance(2500);const form=app.doc.querySelector('.wire-refund-form');assert.ok(form);assert.equal(form.querySelectorAll('input').length,1);assert.equal(form.querySelector('input').value,'');assert.equal(calls.length,0);
 const to='0x'+'55'.repeat(20);form.querySelector('input').value=to;form.dispatchEvent(new app.window.Event('submit',{cancelable:true}));await settle();assert.deepEqual(calls,[to]);assert.equal(app.window.localStorage.getItem('wire-refund-address:polygon:sender@example.test'),to);assert.match(app.doc.querySelector('.wire-refund-result').textContent,/Refund sent/);assert.equal(app.confirmations.length,1);assert.doesNotMatch(app.confirmations[0],/gas fee/);
});
test('the stablecoin sent screen uses the server creation date plus 24 hours',async()=>{
 const app=await start({clocked:true,state:()=>({state:'ready',balanceUnits:2650000})});await makeCard(app);const c=JSON.parse(app.window.localStorage.getItem('wire-usdt-cards'))[0];await app.clock.advance(3000);
 const date=new Intl.DateTimeFormat('en',{dateStyle:'medium',timeStyle:'short'}).format(c.created+86400000);assert.ok(app.doc.querySelector('.usdt-funded'));assert.equal(app.doc.querySelector('.usdt-funded').parentElement.querySelector('.wire-card-expiry').textContent,flowText('en','validUntil').replace('{date}',date));
});
test('known expired-card refunds run once on open, surface failures and require the browser key',async()=>{
 const source=await start({clocked:true});await makeCard(source);const c=JSON.parse(source.window.localStorage.getItem('wire-usdt-cards'))[0];c.created=source.clock.now()-90000000;
 const to='0x'+'66'.repeat(20),local={'wire-usdt-cards':JSON.stringify([c]),'phoenixmail.usdt.sent.v2':source.window.localStorage.getItem('phoenixmail.usdt.sent.v2'),'wire-refund-address:polygon:sender@example.test':to};
 for(const failure of [false,true]){
  const app=await start({clocked:true,local,state:()=>({state:'expired',balanceUnits:2650000})});let count=0;const original=app.window.PhoenixUSDTCashlink.create;app.window.PhoenixUSDTCashlink.create=options=>{const cl=original(options);cl.refund=async(entry,address,settings)=>{count++;assert.equal(address,to);assert.equal(entry.card.id,c.id);assert.equal(settings.confirm({units:2500000,fee:150000}),true);if(failure)throw Error('chain_unavailable');return{status:'refunded'};};return cl;};
  await app.clock.advance(2500);assert.equal(count,1);assert.equal(app.confirmations.length,0);assert.ok(app.doc.querySelector('.wire-refund-result').textContent.includes(failure?'unavailable':'Refund sent'));app.window.dispatchEvent(new app.window.Event('pageshow'));await settle();await app.clock.advance(6000);assert.equal(count,1);
 }
 const noKey=await start({clocked:true,local:{'wire-usdt-cards':JSON.stringify([c]),'wire-refund-address:polygon:sender@example.test':to},state:()=>({state:'expired',balanceUnits:2650000})});await noKey.clock.advance(2500);assert.equal(noKey.doc.querySelector('.wire-refund-form'),null);assert.match(noKey.doc.querySelector('.wire-refund-result').textContent,/browser where/);
});
test('refund memory follows verified sender A, is erased on sign-out and never funds sender B from shared storage',async()=>{
 let expired=false;const a=await start({clocked:true,state:()=>({state:expired?'expired':'awaiting_funds',balanceUnits:expired?2650000:0})});await makeCard(a);
 const list=JSON.parse(a.window.localStorage.getItem('wire-usdt-cards'));list[0].created=a.clock.now()-90000000;a.window.localStorage.setItem('wire-usdt-cards',JSON.stringify(list));expired=true;
 const to='0x'+'88'.repeat(20),original=a.window.PhoenixUSDTCashlink.create;a.window.PhoenixUSDTCashlink.create=options=>{const cl=original(options);cl.refund=async()=>({status:'refunded'});return cl;};
 await a.clock.advance(2500);const form=a.doc.querySelector('.wire-refund-form');form.querySelector('input').value=to;form.dispatchEvent(new a.window.Event('submit',{cancelable:true}));await settle();
 assert.equal(a.window.localStorage.getItem('wire-refund-address:polygon:sender@example.test'),to);
 a.window.localStorage.setItem('wire-refund-address:ethereum:sender@example.test',to);a.window.localStorage.setItem('wire-refund-address:polygon:other@example.test',to);
 a.$('wire-logout').click();await settle();assert.equal(a.window.localStorage.getItem('wire-refund-address:polygon:sender@example.test'),null);assert.equal(a.window.localStorage.getItem('wire-refund-address:ethereum:sender@example.test'),null);assert.equal(a.window.localStorage.getItem('wire-refund-address:polygon:other@example.test'),to);
 const shared=a.snapshotLocal();assert.ok(shared['phoenixmail.usdt.sent.v2']);assert.ok(shared['wire-usdt-cards']);
 const bSource=await start({clocked:true,session:{verified:true,email:'b@example.test'}});await makeCard(bSource);const bCards=JSON.parse(bSource.window.localStorage.getItem('wire-usdt-cards'));bCards[0].created=bSource.clock.now()-90000000;bCards[0].fundingAddress=to;
 const local={...shared,'wire-refund-address:polygon':to,'wire-refund-address:Polygon':to,'wire-refund-address:ethereum':to,'wire-refund-address:Ethereum':to,'wire-usdt-cards':JSON.stringify([...JSON.parse(shared['wire-usdt-cards']),...bCards]),'phoenixmail.usdt.sent.v2':JSON.stringify([...JSON.parse(shared['phoenixmail.usdt.sent.v2']),...JSON.parse(bSource.window.localStorage.getItem('phoenixmail.usdt.sent.v2'))])};
 const b=await start({clocked:true,session:{verified:true,email:'b@example.test'},local,state:()=>({state:'expired',balanceUnits:2650000})});let refunds=0;const create=b.window.PhoenixUSDTCashlink.create;b.window.PhoenixUSDTCashlink.create=options=>{const cl=create(options);cl.refund=async()=>{refunds++;return{status:'refunded'};};return cl;};
 await b.clock.advance(2500);assert.equal(refunds,0);assert.equal(b.doc.querySelector('.wire-refund-form input').value,'');for(const chain of ['polygon','Polygon','ethereum','Ethereum'])assert.equal(b.window.localStorage.getItem('wire-refund-address:'+chain),null);
});
test('an unverified session or a card rejected by the ownership API never triggers an automatic refund',async()=>{
 const source=await start({clocked:true});await makeCard(source);const cards=JSON.parse(source.window.localStorage.getItem('wire-usdt-cards'));cards[0].created=source.clock.now()-90000000;
 const local={...source.snapshotLocal(),'wire-usdt-cards':JSON.stringify(cards),'wire-refund-address:polygon:b@example.test':'0x'+'99'.repeat(20)};
 for(const session of [{verified:false},{verified:true,email:'b@example.test'}]){
  const app=await start({clocked:true,session,local,state:()=>({state:'expired',balanceUnits:2650000})});const fetch=app.window.fetch;app.window.fetch=(path,options)=>path==='/api/usdt/cashlink/status'?Promise.resolve({ok:false,status:404,json:async()=>({code:'not_found'})}):fetch(path,options);let refunds=0;const create=app.window.PhoenixUSDTCashlink.create;app.window.PhoenixUSDTCashlink.create=options=>{const cl=create(options);cl.refund=async()=>{refunds++;return{status:'refunded'};};return cl;};await app.clock.advance(6000);assert.equal(refunds,0);assert.equal(app.doc.querySelector('.wire-refund-form'),null);
 }
});
test('NIM refund stages have no recovery, retry or cancel actions in the actual dashboard',async()=>{
 const app=await start({clocked:true,nimPending:true,createdAgo:90000000});for(const stage of ['refunding','refunded']){app.setStage(stage);app.window.dispatchEvent(new app.window.Event('pageshow'));await settle();const row=app.doc.querySelector('.wire-card-row');assert.ok(row);assert.equal(row.querySelectorAll('button,a').length,0);assert.equal(row.querySelector('.wire-card-state').textContent,flowText('en',stage));}
});
test('a pending stablecoin refund becomes completed, or offers manual retry when the server stops processing it',async()=>{
 for(const completed of [false,true]){
  const initial=await start({clocked:true});await makeCard(initial);const c=JSON.parse(initial.window.localStorage.getItem('wire-usdt-cards'))[0];c.created=initial.clock.now()-90000000;c.refundPending=true;c.refundPendingAt=initial.clock.now()-90000;
  const to='0x'+'77'.repeat(20),app=await start({clocked:true,local:{'wire-usdt-cards':JSON.stringify([c]),'phoenixmail.usdt.sent.v2':initial.window.localStorage.getItem('phoenixmail.usdt.sent.v2'),'wire-refund-address:polygon:sender@example.test':to},state:()=>({state:completed?'withdrawn':'expired',balanceUnits:completed?0:2650000,recovering:false})});await app.clock.advance(2500);
  assert.ok(app.doc.querySelector('.wire-refund-result').textContent.includes(completed?'Refund sent':'unavailable'));assert.equal(Boolean(app.doc.querySelector('.wire-refund-form')),!completed);
 }
});
test('NIM recipient deadlines and expired-card errors are translated and disable both receive actions',async()=>{
 for(const language of ['fr','en','es','de','zh']){
  const app=await recipientPage({language});const date=new Intl.DateTimeFormat(language,{dateStyle:'medium',timeStyle:'short'}).format(app.clock.now()+86400000);assert.equal(app.doc.querySelector('.wire-card-expiry').textContent,flowText(language,'expires').replace('{date}',date));
  for(const claimError of ['claim_unavailable','claim_expired','card_expired']){const gone=await recipientPage({language,claimError});assert.equal(gone.$('claim-status').textContent,flowText(language,'ended'));assert.equal(gone.$('claim-pay').disabled,true);assert.equal(gone.$('claim-wallet').disabled,true);}
 }
});
for(const classic of [false,true])for(const [language,message]of Object.entries(inviteOnlyCopy)){
 test(`invite-only sender access in ${classic?'classic':'payment-window'} / ${language} keeps retry and receiving available`,async()=>{
  const app=await start({session:{verified:false},clocked:true,classic});
  if(classic){app.doc.documentElement.lang=language;app.window.dispatchEvent(new app.window.Event('wire-language'));}
  else{app.$('wire-language').value=language;app.$('wire-language').dispatchEvent(new app.window.Event('change'));}
  const input=app.$(classic?'sender':'wire-email'),button=app.$(classic?'send-sender-code':'wire-send-code');
  const status=app.$(classic?'sender-status':'wire-status'),box=app.$(classic?'sender-code-box':'wire-code-box');
  input.value='new@example.test';app.failCode('invite_only');const timers=app.clock.pending();
  button.click();await settle();
  assert.equal(status.textContent,message);assert.equal(flowText(language,'inviteOnly'),message);
  assert.doesNotMatch(status.textContent,/invite_only|Mainnet|OTP|Hub/);
  assert.equal(input.disabled,false);assert.equal(input.readOnly,false);assert.equal(button.disabled,false);assert.equal(box.hidden,true);
  assert.equal(app.clock.pending(),timers,'no resend countdown or request timeout survives the refusal');
  if(!classic){assert.equal(button.dataset.state,'idle');assert.equal(app.$('wire-code-delivery').hidden,true);}
  const other=language==='en'?'fr':'en';
  for(const selected of [other,language]){
   if(classic){app.doc.documentElement.lang=selected;app.window.dispatchEvent(new app.window.Event('wire-language'));}
   else{app.$('wire-language').value=selected;app.$('wire-language').dispatchEvent(new app.window.Event('change'));}
   assert.equal(status.textContent,inviteOnlyCopy[selected],'the invitation follows a language change');
  }
  button.click();await settle();
  assert.equal(app.calls.filter(([path])=>path==='/api/sender/request').length,2,'an immediate retry is allowed');
  await app.clock.advance(31000);assert.equal(status.textContent,message);assert.equal(button.disabled,false);
  input.value='invited@example.test';input.dispatchEvent(new app.window.Event('input'));app.failCode(false);
  button.click();await settle();assert.equal(box.hidden,false,'another address can request a code');
  assert.equal(app.calls.filter(([path])=>path==='/api/sender/request').length,3);
  if(!classic){assert.equal(button.dataset.state,'sent');assert.equal(button.disabled,true,'only a successful request starts the cooldown');}
 });
}

test('a failed or stalled code request never says sent and leaves retry available',async()=>{
 for(const stalled of [false,true]){
  const t=await start({session:{verified:false},clocked:true});t.$('wire-email').value='sender@example.test';
  if(stalled)t.holdCode();else t.failCode(true);
  t.$('wire-send-code').click();await settle();if(stalled)await t.clock.advance(30000);
  assert.equal(t.$('wire-send-code').dataset.state,'idle');assert.equal(t.$('wire-send-code').disabled,false);
  assert.equal(t.$('wire-code-box').hidden,true);assert.equal(t.$('wire-code-delivery').hidden,true);
  assert.match(t.$('wire-status').textContent,/Something went wrong/);assert.equal(t.$('wire-email').disabled,false);
 }
});

test('home has an explicit sign-out action in five languages, ends the session and preserves card recovery data',async()=>{
 const labels={fr:'Se déconnecter',en:'Sign out',es:'Cerrar sesión',de:'Abmelden',zh:'退出登录'};
 const t=await start({clocked:true});await makeCard(t);await t.clock.advance(2500);
 for(const[language,label]of Object.entries(labels)){
  t.$('wire-language').value=language;t.$('wire-language').dispatchEvent(new t.window.Event('change'));
  assert.equal(t.$('wire-logout').getAttribute('aria-label'),label);assert.equal(t.$('wire-logout').title,label);assert.equal(t.$('wire-logout').closest('[hidden]'),null);
 }
 const recovery=t.window.localStorage.getItem('wire-usdt-cards');assert.ok(JSON.parse(recovery)[0].fragment);
 t.$('wire-logout').click();await settle();
 assert.equal(t.calls.filter(([p])=>p==='/api/logout').length,1);
 assert.deepEqual(await(await t.window.fetch('/api/session')).json(),{verified:false});
 assert.equal(t.$('wire-sender').hidden,false);assert.equal(t.$('wire-ready').hidden,true);assert.equal(t.$('wire-cards').hidden,true);
 assert.equal(t.doc.querySelector('.nimiq-payment-modal'),null);assert.equal(t.$('wire-email').value,'');assert.equal(t.$('wire-code').value,'');
 assert.equal(t.window.localStorage.getItem('wire-usdt-cards'),recovery);
 const count=t.calls.length;await t.clock.advance(120000);assert.equal(t.calls.length,count);
});

test('failed server sign-out stays visibly signed in and allows another attempt',async()=>{
 const t=await start({clocked:true});t.failLogout(true);t.$('wire-logout').click();await settle();
 assert.equal(t.$('wire-ready').hidden,false);assert.equal(t.$('wire-sender').hidden,true);assert.equal(t.$('wire-logout').disabled,false);
 assert.match(t.$('wire-status').textContent,/Something went wrong/);
 t.failLogout(false);t.$('wire-logout').click();await settle();assert.equal(t.$('wire-ready').hidden,true);
});

test('PhoenixFire, Aurora and Ocean are available, persist across visits and remain independent of light and dark',async()=>{
 const t=await start();assert.equal(t.$('wire-palette').value,'live');assert.equal(t.doc.documentElement.dataset.palette,'phoenixfire');
 assert.deepEqual([...t.$('wire-palette').options].map(o=>o.value),['live','phoenixfire','aurora','ocean']);
 for(const palette of ['aurora','ocean','phoenixfire']){
  t.$('wire-palette').value=palette;t.$('wire-palette').dispatchEvent(new t.window.Event('change'));
  assert.equal(t.doc.documentElement.dataset.palette,palette);assert.equal(t.doc.documentElement.dataset.accent,palette);
  t.$('wire-theme').click();assert.equal(t.doc.documentElement.dataset.palette,palette);assert.equal(t.doc.documentElement.dataset.theme,'light-paper');
  t.$('wire-theme').click();assert.equal(t.doc.documentElement.dataset.palette,palette);assert.equal(t.doc.documentElement.dataset.theme,'dark');
  const restored=await start({local:t.snapshotLocal()});assert.equal(restored.$('wire-palette').value,palette);
 }
 const legacy=await start({local:{'wire-accent':'teal'}});assert.equal(legacy.$('wire-palette').value,'ocean');
 const invalid=await start({local:{'wire-palette':'unknown'}});assert.equal(invalid.$('wire-palette').value,'phoenixfire');
});

test('palette selector is localized, Phoenix dark is charcoal, and Aurora/Ocean retain their original dark colours',async()=>{
 const t=await start();
 for(const[language,label,ocean]of [['fr','Thème','Océan'],['en','Theme','Ocean'],['es','Tema','Océano'],['de','Design','Ozean'],['zh','主题','海洋']]){
  t.$('wire-language').value=language;t.$('wire-language').dispatchEvent(new t.window.Event('change'));
  assert.equal(t.$('wire-palette').getAttribute('aria-label'),label);assert.equal(t.$('wire-palette').querySelector('[value="ocean"]').textContent,ocean);
 }
 const style=t.doc.createElement('style');style.textContent=read('../internal/web/static/wire-pay.css');t.doc.head.append(style);
 const original=t.doc.createElement('style');original.textContent=read('../internal/web/static/design.css');t.doc.head.append(original);
 for(const theme of ['dark'])for(const palette of ['phoenixfire','aurora','ocean']){
  const restored=[...style.sheet.cssRules].find(r=>r.selectorText===`html.wire-app[data-theme="${theme}"][data-palette="${palette}"]`);
  const old=[...original.sheet.cssRules].find(r=>r.selectorText===(theme==='dark'?`:root[data-palette=${palette}]`:`:root[data-theme=light][data-palette=${palette}]`));
  if(palette==='phoenixfire'){
   for(const[token,value]of Object.entries({'--bg':'#121416','--panel':'#1b1e22','--panel2':'#272b30','--line':'#454b53','--accent':'#ffb16b'}))assert.equal(restored.style.getPropertyValue(token),value);
   continue;
  }
  for(const token of ['--bg','--panel','--line','--accent'])assert.equal(restored.style.getPropertyValue(token),old.style.getPropertyValue(token));
  assert.equal(restored.style.getPropertyValue('--panel2'),old.style.getPropertyValue('--field'));
 }
});

test('live palette changes colours without touching the NIM form or wallet, pauses in background, and stops for a fixed theme',async()=>{
 const t=await start({nimOnly:true,clocked:true});t.$('pay').click();await settle();
 t.$('recipient').value='friend@example.test';t.$('amount').value='2.5';t.$('message').value='Keep this draft';
 const fields=['recipient','amount','message'].map(id=>t.$(id)),focus=fields[2];focus.focus();
 assert.equal(t.doc.documentElement.dataset.palette,'phoenixfire');
 await t.clock.advance(20000);const first=t.doc.documentElement.dataset.palette;assert.notEqual(first,'phoenixfire');
 await t.clock.advance(20000);assert.notEqual(t.doc.documentElement.dataset.palette,first);
 assert.deepEqual(fields.map(el=>el.value),['friend@example.test','2.5','Keep this draft']);assert.equal(t.doc.activeElement,focus);
 assert.equal(t.nativeCalls.length,0);assert.equal(t.checkouts.length,0);assert.ok(!t.calls.some(([p,b])=>p==='/api/payments'&&b));
 t.clock.hidden(true);const paused=t.doc.documentElement.dataset.palette;await t.clock.advance(120000);assert.equal(t.doc.documentElement.dataset.palette,paused);
 t.clock.hidden(false);await t.clock.advance(20000);assert.notEqual(t.doc.documentElement.dataset.palette,paused);
 t.$('wire-palette').value='ocean';t.$('wire-palette').dispatchEvent(new t.window.Event('change'));await t.clock.advance(120000);assert.equal(t.doc.documentElement.dataset.palette,'ocean');
 t.$('wire-palette').value='live';t.$('wire-palette').dispatchEvent(new t.window.Event('change'));
 const restored=await start({clocked:true,local:t.snapshotLocal()});assert.equal(restored.$('wire-palette').value,'live');
 await restored.clock.advance(20000);assert.notEqual(restored.doc.documentElement.dataset.palette,'phoenixfire');
 const texts={fr:'Aléatoire en direct',en:'Live shuffle',es:'Aleatorio en directo',de:'Zufälliger Wechsel',zh:'实时随机'};
 for(const[language,label]of Object.entries(texts)){t.$('wire-language').value=language;t.$('wire-language').dispatchEvent(new t.window.Event('change'));assert.equal(t.$('wire-palette').querySelector('[value="live"]').textContent,label);}
 const css=read('../internal/web/static/wire-pay.css');assert.match(css,/transition:background-color 2s ease/);assert.match(css,/@media\(prefers-reduced-motion:reduce\)/);
});

test('sign out keeps its outlined lock icon and PhoenixMail-sized rounded square across themes',async()=>{
 const t=await start(),style=t.doc.createElement('style');style.textContent=read('../internal/web/static/wire-pay.css');t.doc.head.append(style);
 const button=t.$('wire-logout'),icon=button.querySelector('svg');assert.ok(icon);assert.equal(icon.getAttribute('aria-hidden'),'true');
 assert.ok(icon.querySelector('rect'));assert.ok(icon.querySelector('path'));
 for(const language of ['fr','en','es','de','zh']){
  t.$('wire-language').value=language;t.$('wire-language').dispatchEvent(new t.window.Event('change'));assert.equal(button.querySelector('svg'),icon);
 }
 for(let i=0;i<2;i++){
  const c=t.window.getComputedStyle(button);assert.equal(c.width,'44px');assert.equal(c.height,'44px');assert.equal(c.borderRadius,'9px');
  t.$('wire-theme').click();
 }
 const rules=[...style.sheet.cssRules],rule=rules.find(r=>r.selectorText==='.wire-body .wire-session-actions .wire-session-lock');
 assert.equal(rule.style.getPropertyValue('color'),'var(--accent)');assert.equal(rule.style.getPropertyValue('border'),'1px solid var(--accent)');
 const nim=await start({nimOnly:true});nim.$('pay').click();await settle();
 const ns=nim.doc.createElement('style');ns.textContent=style.textContent;nim.doc.head.append(ns);
 const close=nim.doc.querySelector('.wire-nim-close');assert.equal(nim.window.getComputedStyle(close).borderRadius,'9px');assert.ok(close.querySelector('svg'));
});

test('signing out from the completed NIM window closes it and stops its cached flow cleanly',async()=>{
 const t=await tracked();t.setStage('sent');await t.clock.advance(4000);
 assert.equal(t.$('nim-progress-close').hidden,false);
 t.$('wire-logout').click();await settle();
 assert.equal(t.doc.querySelector('.wire-nim-window'),null);assert.equal(t.$('wire-sender').hidden,false);
 assert.equal(t.$('wire-ready').hidden,true);assert.equal(t.$('wire-status').textContent,'');
 const count=t.calls.length;await t.clock.advance(120000);assert.equal(t.calls.length,count);
});

test('currency choice comes first, then other coins and preview tabs disappear for each selected asset',async()=>{
 for(const label of ['NIM','USDC · Polygon','USDT · Polygon','USDT · Ethereum']){
  const t=await start({width:375});t.$('pay').click();await settle();
  const picker=t.doc.querySelector('.wire-coin-picker');assert.ok(picker);assert.equal(picker.querySelector('h3').textContent,'Choose a currency');
  assert.equal(picker.querySelector('.nimiq-payment-assets').hidden,false);assert.equal(picker.querySelector('.nimiq-payment-tabs').hidden,true);
  assert.ok(!t.calls.some(([p,b])=>p==='/api/payments'&&b||p==='/api/usdt/cashlink/create'));
  [...picker.querySelectorAll('.nimiq-payment-assets button')].find(b=>b.textContent===label).click();await settle();
  const win=t.doc.querySelector('.nimiq-payment-window');assert.ok(win);assert.equal(win.classList.contains('wire-coin-picker'),false);
  assert.equal(win.querySelector('.nimiq-payment-assets').hidden,true);
  if(label==='NIM'){assert.ok(t.$('card-form'));assert.equal(win.querySelector('.nimiq-payment-tabs'),null);}
  else assert.equal(win.querySelector('.nimiq-payment-tabs').hidden,true);
  assert.equal(t.checkouts.length,0);assert.equal(t.nativeCalls.length,0);
 }
 const t=await start({clocked:true});await makeCard(t);
 const win=t.doc.querySelector('.nimiq-payment-window');
 assert.equal(win.dataset.view,'preview');assert.equal(win.querySelector('.nimiq-payment-tabs').hidden,true);
 assert.ok(win.querySelector('.nimiq-payment-preview .nimiq-payment-card'),'the actual funding card remains available without a preview tab');
});

test('handoff is created only on explicit Pay choice and the URL contains only an opaque token',async()=>{
 const browser=await start({nimOnly:true,mobile:true,clocked:true});await submitNim(browser);
 browser.$('wire-choice-cancel').click();browser.$('message').value='Private handoff note';
 browser.$('card-form').dispatchEvent(new browser.window.Event('submit',{cancelable:true}));await settle();
 assert.equal(browser.calls.filter(([p])=>p==='/api/handoff').length,0);
 browser.$('wire-choice-open').click();await settle();
 const creation=browser.calls.find(([p])=>p==='/api/handoff')[1];
 assert.deepEqual(creation,{payment_id:'a'.repeat(48)});
 const prepared=browser.calls.find(([p,b])=>p==='/api/payments'&&b)[1];
 assert.equal(prepared.amount,'2.5');assert.equal(prepared.message,'Private handoff note');
 assert.ok(browser.calls.findIndex(([p,b])=>p==='/api/payments'&&b)<browser.calls.findIndex(([p])=>p==='/api/handoff'));
 const link=browser.opened.at(-1).location.href,destination=new URL(link).searchParams.get('url');
 assert.equal(destination,'https://wire.phoenixmail.ca/#h='+handoffToken);
 assert.ok(!/example|%7B|wire-nim|2\.5|Private|recipient|sender/i.test(link));
 assert.equal(browser.opened.at(-1).opener,null);
 assert.equal(browser.calls.filter(([p,b])=>p==='/api/payments'&&b).length,1);
 assert.ok(!browser.calls.some(([p])=>p.endsWith('/start')));
 for(const data of [browser.snapshot(),browser.snapshotLocal(),browser.consoleMessages])assert.ok(!JSON.stringify(data).includes(handoffToken));
});

test('redeem clears the token before the request, opens only the prepared card without a code or auto payment',async()=>{
 const gate={};
 const app=await start({nimOnly:true,nativeMode:'success',session:{verified:false},clocked:true,url:'https://wire.phoenixmail.ca/#h='+handoffToken,holdRedeem:gate});
 assert.equal(app.window.location.hash,'');
 assert.deepEqual(app.redeemHashes,['']);
 assert.equal(app.doc.querySelector('.wire-nim-window'),null);
 assert.equal(app.nativeCalls.length,0);
 for(const data of [app.snapshot(),app.snapshotLocal(),app.consoleMessages])assert.ok(!JSON.stringify(data).includes(handoffToken));
 gate.release();await settle();
 assert.equal(app.$('wire-entry').hidden,true);
 assert.equal(app.$('wire-ready').hidden,true);assert.equal(app.$('wire-cards').hidden,true);
 assert.equal(app.$('wire-handoff-home').hidden,false);
 assert.equal(app.$('wire-language').value,'fr');
 assert.ok(app.doc.querySelector('.wire-nim-window'));
 assert.equal(app.$('card-amount').textContent,'2.5');assert.equal(app.$('card-recipient').textContent,handoffDraft.recipient);assert.equal(app.$('card-message').textContent,handoffDraft.message);
 assert.equal(app.doc.querySelector('.workspace').hidden,true);
 assert.equal(app.$('my-payments').hidden,true);assert.equal(app.$('payment-list').children.length,0);
 assert.equal(app.$('nim-handoff-summary').hidden,false);assert.equal(app.$('nim-handoff-pay').hidden,false);
 assert.ok(!app.calls.some(([p])=>p.startsWith('/api/usdt/')));
 assert.ok(!app.calls.some(([p])=>p==='/api/sender/code'||p==='/api/sender/verify'));
 assert.equal(app.nativeCalls.length,0);assert.equal(app.checkouts.length,0);
 app.window.dispatchEvent(new app.window.Event('pageshow'));
 app.doc.dispatchEvent(new app.window.Event('visibilitychange'));await settle();
 assert.equal(app.calls.filter(([p])=>p==='/api/handoff/redeem').length,1);
 app.$('nim-handoff-pay').click();await settle();
 assert.equal(app.nativeCalls.length,1);assert.equal(app.nativeCalls[0].request.value,250000);
 assert.equal(app.nativeCalls[0].request.recipient,'NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY');
 assert.equal(app.nativeCalls[0].hex,'00c282c280c292c287');
 const reloaded=await start({nimOnly:true,clocked:true,url:app.window.location.href,saved:app.snapshot(),local:app.snapshotLocal()});
 assert.equal(reloaded.calls.filter(([p])=>p==='/api/handoff/redeem').length,0);
 for(const data of [app.snapshot(),app.snapshotLocal(),app.consoleMessages])assert.ok(!JSON.stringify(data).includes(handoffToken));
});

test('invalid, expired or used handoff returns to normal email verification with translated feedback and no retry',async()=>{
 const expected={fr:'Le lien est invalide ou a expiré. Rouvrez le lien depuis Safari.',en:'The link is invalid or has expired. Reopen the link from Safari.',es:'El enlace no es válido o ha caducado. Vuelve a abrirlo desde Safari.',de:'Der Link ist ungültig oder abgelaufen. Öffne ihn erneut in Safari.',zh:'链接无效或已过期。请从 Safari 重新打开链接。'};
 for(const [language,text]of Object.entries(expected)){
  const app=await start({nimOnly:true,session:{verified:false},clocked:true,redeemStatus:404,url:'https://wire.phoenixmail.ca/#h='+handoffToken,local:{'wire-language':language}});
  assert.equal(app.window.location.hash,'');assert.equal(app.$('wire-status').textContent,text);
  assert.equal(app.$('wire-sender').hidden,false);assert.equal(app.doc.querySelector('.wire-nim-window'),null);
  assert.equal(app.nativeCalls.length,0);assert.equal(app.checkouts.length,0);
  app.window.dispatchEvent(new app.window.Event('pageshow'));await app.clock.advance(120000);
  assert.equal(app.calls.filter(([p])=>p==='/api/handoff/redeem').length,1);
  const again=await start({nimOnly:true,session:{verified:false},clocked:true,url:app.window.location.href,saved:app.snapshot()});
  assert.equal(again.calls.filter(([p])=>p==='/api/handoff/redeem').length,0);
 }
});

test('handoff creation errors are translated, retryable and keep Hub and stores usable',async()=>{
 for(const language of ['fr','en','es','de','zh'])for(const handoffStatus of [401,429,'network']){
  const t=await start({nimOnly:true,clocked:true,handoffStatus,local:{'wire-language':language}});
  await submitNim(t);t.$('wire-choice-open').click();await settle();
  const message=t.$('wire-choice-status').textContent;
  assert.ok(message.length>10);assert.equal(t.$('wire-choice-status').hidden,false);
  assert.equal(t.$('wire-nim-choice').hidden,false);
  assert.equal(t.$('wire-choice-open').getAttribute('aria-disabled'),null);
  assert.equal(t.$('wire-choice-open').getAttribute('href'),'#');
  assert.equal(t.opened.at(-1).closed,true);
  t.$('wire-choice-get').click();assert.equal(t.$('wire-choice-stores').hidden,false);
  t.$('wire-choice-open').click();await settle();
  assert.equal(t.calls.filter(([p])=>p==='/api/handoff').length,2);
  t.$('wire-choice-hub').click();await settle();
  assert.equal(t.$('wire-nim-choice').hidden,true);assert.equal(t.checkouts.length,1);
 }
});

test('cancel during creation prevents late navigation and blocked popups offer a direct retry',async()=>{
 const gate={},t=await start({nimOnly:true,clocked:true,holdHandoff:gate});await submitNim(t);
 t.$('wire-choice-open').click();await settle();t.$('wire-choice-cancel').click();
 gate.release();await settle();assert.equal(t.opened.at(-1).closed,true);assert.equal(t.opened.at(-1).location.href,undefined);
 const blocked=await start({nimOnly:true,clocked:true,popupBlocked:true});await submitNim(blocked);
 blocked.$('wire-choice-open').click();await settle();
 assert.equal(new URL(blocked.$('wire-choice-open').href).searchParams.get('url'),'https://wire.phoenixmail.ca/#h='+handoffToken);
 assert.match(blocked.$('wire-choice-status').textContent,/again/);
 assert.equal(blocked.checkouts.length,0);
});

test('stalled handoff requests time out without a blank page, leaking tokens or automatic retries',async()=>{
 const app=await start({nimOnly:true,session:{verified:false},clocked:true,holdRedeem:{},url:'https://wire.phoenixmail.ca/#h='+handoffToken});
 assert.equal(app.window.location.hash,'');await app.clock.advance(15000);
 assert.equal(app.$('wire-sender').hidden,false);assert.match(app.$('wire-status').textContent,/Reopen the link from Safari/);
 assert.equal(app.calls.filter(([p])=>p==='/api/handoff/redeem').length,1);
 for(const data of [app.snapshot(),app.snapshotLocal(),app.consoleMessages])assert.ok(!JSON.stringify(data).includes(handoffToken));
 const browser=await start({nimOnly:true,clocked:true,holdHandoff:{}});await submitNim(browser);
 browser.$('wire-choice-open').click();await settle();await browser.clock.advance(15000);
 assert.equal(browser.$('wire-nim-choice').hidden,false);assert.equal(browser.opened.at(-1).closed,true);
 assert.match(browser.$('wire-choice-status').textContent,/Could not open/);
 assert.equal(browser.$('wire-choice-open').getAttribute('aria-disabled'),null);
 browser.$('wire-choice-hub').click();await settle();assert.equal(browser.checkouts.length,1);
});

test('retired personal-data fragments and saved drafts are erased and never redeemed or restored',async()=>{
 const app=await start({nimOnly:true,session:{verified:false},clocked:true,url:'https://wire.phoenixmail.ca/#wire-nim=%7B%22recipient%22%3A%22private%40example.test%22%7D',saved:{'wire-nim-handoff':JSON.stringify(handoffDraft)}});
 assert.equal(app.window.location.hash,'');assert.equal(app.window.sessionStorage.getItem('wire-nim-handoff'),null);
 assert.equal(app.calls.filter(([p])=>p==='/api/handoff/redeem').length,0);
 assert.equal(app.$('wire-email').value,'');assert.equal(app.doc.querySelector('.wire-nim-window'),null);
 assert.match(app.$('wire-status').textContent,/expired/);
});

test('a redeemed prepared card is not replaced by an old local intent and cannot cause automatic payment',async()=>{
 const app=await start({nimOnly:true,nativeMode:'success',clocked:true,manualNim:true,nimPending:true,url:'https://wire.phoenixmail.ca/#h='+handoffToken,redeemDraft:{...handoffDraft,recipient:'new@example.test',message:'A new card'},saved:{'wire-pending-payment':'a'.repeat(48),'wire-pending-wallet':'pay'}});
 assert.equal(app.$('card-recipient').textContent,'new@example.test');assert.equal(app.$('card-message').textContent,'A new card');
 await app.clock.advance(120000);app.window.dispatchEvent(new app.window.Event('pageshow'));await settle();
 assert.equal(app.$('card-recipient').textContent,'new@example.test');assert.equal(app.nativeCalls.length,0);
 assert.ok(!app.calls.some(([p,b])=>p==='/api/payments'&&b));
});

test('same sender already signed in keeps the full session but opens the prepared card, never a second creation',async()=>{
 const app=await start({alreadySignedIn:true,nativeMode:'success',clocked:true,url:'https://wire.phoenixmail.ca/#h='+handoffToken});
 assert.equal(app.$('wire-ready').hidden,false);assert.equal(app.$('wire-who').textContent,'sender@example.test');
 assert.equal(app.$('wire-handoff-home').hidden,true);assert.equal(app.$('nim-handoff-summary').hidden,false);
 assert.equal(app.$('card-amount').textContent,'2.5');assert.equal(app.nativeCalls.length,0);
 app.$('nim-handoff-pay').click();await settle();assert.equal(app.nativeCalls.length,1);
 assert.ok(!app.calls.some(([p,b])=>p==='/api/payments'&&b));
 app.$('nim-progress-close').click();await settle();app.$('pay').click();await settle();
 app.doc.querySelector('.nimiq-payment-assets button').click();await settle();
 assert.equal(app.$('nim-handoff-summary').hidden,true,'a completed handoff does not trap the full sender in the old card');
});

test('scoped session reload and foreground resume only its card without redeem, code, dashboard or payment',async()=>{
 const scope={verified:false,email:'',handoff:{payment_id:'a'.repeat(48)}};
 const app=await start({session:scope,nimStage:'funding_started',manualNim:true,nativeMode:'success',clocked:true,width:375});
 assert.equal(app.$('wire-entry').hidden,true);assert.equal(app.$('wire-handoff-home').hidden,false);
 assert.equal(app.$('my-payments').hidden,true);assert.equal(app.$('payment-list').children.length,0);
 assert.equal(app.$('wire-cards').hidden,true);assert.equal(app.$('nim-handoff-pay').hidden,true);
 app.clock.hidden(true);app.clock.suspend(300000);app.clock.hidden(false);app.window.dispatchEvent(new app.window.Event('pageshow'));await settle();
 assert.equal(app.nativeCalls.length,0);assert.equal(app.checkouts.length,0);
 const allowed=p=>['/api/config','/api/session','/api/payments'].includes(p)||['art','confirm'].some(a=>p==='/api/payments/'+'a'.repeat(48)+'/'+a);
 assert.ok(app.calls.every(([p])=>allowed(p)),JSON.stringify(app.calls.map(([p])=>p)));
 app.doc.querySelector('.wire-nim-close').click();assert.equal(app.doc.querySelector('.wire-nim-window'),null);
 app.$('wire-handoff-open').click();await settle();assert.equal(app.$('nim-handoff-summary').hidden,false);
 app.$('wire-handoff-logout').click();await settle();assert.equal(app.$('wire-sender').hidden,false);
 assert.equal(app.doc.querySelector('.wire-nim-window'),null);
});

test('session conflict preserves the other sender and requires explicit sign-out, never redeeming twice',async()=>{
 const app=await start({redeemStatus:409,redeemCode:'session_conflict',session:{verified:true,email:'other@example.test'},nativeMode:'success',clocked:true,url:'https://wire.phoenixmail.ca/#h='+handoffToken});
 assert.equal(app.$('wire-who').textContent,'other@example.test');assert.match(app.$('wire-status').textContent,/Another sender/);
 assert.equal(app.calls.filter(([p])=>p==='/api/logout').length,0);
 assert.equal(app.doc.querySelector('.wire-nim-window'),null);assert.equal(app.nativeCalls.length,0);
 app.window.dispatchEvent(new app.window.Event('pageshow'));await settle();
 assert.equal(app.calls.filter(([p])=>p==='/api/handoff/redeem').length,1);
 app.$('wire-logout').click();await settle();assert.equal(app.$('wire-sender').hidden,false);
});

test('not fundable and expired cards display clear states in five languages and never open a payment',async()=>{
 for(const language of ['fr','en','es','de','zh'])for(const [status,code]of [[409,'payment_not_fundable'],[410,'claim_expired']]){
  const app=await start({redeemStatus:status,redeemCode:code,session:{verified:false},nativeMode:'success',clocked:true,url:'https://wire.phoenixmail.ca/#h='+handoffToken,local:{'wire-language':language}});
  assert.ok(app.$('wire-status').textContent.length>20);assert.match(app.$('wire-status').textContent,/Safari/);
  assert.equal(app.$('pay').disabled,true);assert.equal(app.doc.querySelector('.wire-nim-window'),null);
  app.$('pay').click();await app.clock.advance(300000);
  assert.equal(app.nativeCalls.length,0);assert.equal(app.checkouts.length,0);
  assert.equal(app.calls.filter(([p])=>p==='/api/handoff/redeem').length,1);
 }
});

test('chain confirmation before start and a racing start never ask the wallet to pay twice',async()=>{
 for(const race of [false,true]){
  const app=await start({nativeMode:'success',clocked:true,startRace:race,url:'https://wire.phoenixmail.ca/#h='+handoffToken});
  if(!race){app.setStage('funding_started');app.fundOnChain();}app.$('nim-handoff-pay').click();await settle();
  assert.equal(app.nativeCalls.length,0);assert.equal(app.checkouts.length,0);
  assert.equal(app.calls.filter(([p])=>p.endsWith('/start')).length,race?1:0);
  assert.equal(app.$('nim-progress-close').hidden,false);
  const confirm=app.calls.findIndex(([p])=>p.endsWith('/confirm')),startIndex=app.calls.findIndex(([p])=>p.endsWith('/start'));
  if(race)assert.ok(confirm<startIndex);
 }
});

test('prepared card native refusal is cancellation without a second start, and no provider uses Hub',async()=>{
 for(const mode of ['denied','missing']){
  const app=await start({nativeMode:mode,clocked:true,url:'https://wire.phoenixmail.ca/#h='+handoffToken});
  app.$('nim-handoff-pay').click();await settle();
  if(mode==='denied'){
   assert.match(app.$('feedback').textContent,/annulé/);assert.equal(app.checkouts.length,0);
   app.$('nim-handoff-pay').click();await settle();assert.equal(app.nativeCalls.length,1);
  }else{assert.equal(app.nativeCalls.length,0);assert.equal(app.checkouts.length,1);}
  assert.equal(app.calls.filter(([p])=>p.endsWith('/start')).length,1);
 }
});

async function failedNative(options={}){
 const app=await start({nativeMode:'error',clocked:true,url:'https://wire.phoenixmail.ca/#h='+handoffToken,...options});
 app.$('nim-handoff-pay').click();await settle();return app;
}

test('native diagnostic exposes type/code/message as copyable text and logs only the redacted diagnostic',async()=>{
 const message='Control character \u0000 is not allowed; friend@example.test sender@example.test NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY https://example.test/#secret '+handoffToken+' Bonjour 🐦 amount 2.5';
 const app=await failedNative({nativeError:{type:'INVALID_DATA',code:'BAD_DATA',message},width:375});
 const detail=app.$('nim-wallet-detail').textContent;
 assert.match(detail,/type=INVALID_DATA/);assert.match(detail,/code=BAD_DATA/);assert.match(detail,/message=Control character \\u0000 is not allowed/);
 for(const secret of ['friend@example.test','sender@example.test','NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY','https://example.test',handoffToken,'Bonjour 🐦','2.5'])assert.ok(!detail.includes(secret),secret);
 assert.deepEqual(app.consoleMessages,[['Nimiq Pay request failed:',detail]]);
 const copies=[];Object.defineProperty(app.window.navigator,'clipboard',{value:{async writeText(text){copies.push(text);}},configurable:true});
 app.$('nim-wallet-copy').click();await settle();assert.deepEqual(copies,[detail]);
 assert.equal(app.$('nim-wallet-failure').hidden,false);assert.equal(app.$('nim-wallet-retry').hidden,false);
 assert.equal(app.$('nim-progress-close').hidden,true);
 assert.equal(app.nativeCalls[0].request.data,'\u0000\u0082\u0080\u0092\u0087');
 for(const store of [app.snapshot(),app.snapshotLocal()])assert.ok(!JSON.stringify(store).includes('INVALID_DATA'));
 const css=read('../internal/web/static/wire-pay.css');assert.match(css,/#nim-wallet-detail\{[^}]*white-space:pre-wrap[^}]*overflow-wrap:anywhere/);
});

test('native retry waits from server started_at, counts down, and only pays on an explicit click after three minutes',async()=>{
 const app=await failedNative({nativeResults:['error','success'],createdAgo:7200000});
 assert.match(app.$('nim-progress-message').textContent,/erreur réseau/);assert.ok(app.$('nim-progress-message').textContent.split('\n').length<=2);
 await app.clock.advance(30000);const before=app.calls.length;
 app.$('nim-wallet-retry').click();await settle();
 assert.equal(app.nativeCalls.length,1);assert.match(app.$('nim-wallet-wait').textContent,/2:30/);assert.equal(app.$('nim-wallet-retry').disabled,true);
 const after=app.calls.slice(before);assert.ok(after.findIndex(([p])=>p.endsWith('/confirm'))<after.findIndex(([p])=>p.endsWith('/checkout')));
 await app.clock.advance(1000);assert.match(app.$('nim-wallet-wait').textContent,/2:29/);
 await app.clock.advance(149000);assert.equal(app.$('nim-wallet-retry').disabled,false);assert.equal(app.nativeCalls.length,1,'the timer must never pay');
 app.$('nim-wallet-retry').click();await settle();
 assert.equal(app.nativeCalls.length,2);assert.equal(app.nativeCalls[1].hex,'00c282c280c292c287');
 assert.equal(app.calls.filter(([p])=>p.endsWith('/start')).length,1);
 assert.equal(app.calls.filter(([p])=>p.endsWith('/checkout')).length,2);
 assert.equal(app.$('nim-progress-close').hidden,false);assert.match(app.$('nim-progress-message').textContent,/Carte envoyée/);
});

test('already_funded after a pending confirmation follows the card and never opens a second wallet',async()=>{
 const app=await failedNative({checkoutCode:'already_funded'});
 app.$('nim-wallet-retry').click();await settle();
 assert.equal(app.nativeCalls.length,1);assert.equal(app.checkouts.length,0);
 assert.equal(app.calls.filter(([p])=>p.endsWith('/checkout')).length,1);
 assert.match(app.$('nim-progress-message').textContent,/Carte envoyée/);
 assert.equal(app.$('nim-wallet-failure').hidden,true);
});

test('funding received after native error is tracked normally, without displaying an unpaid state or a retry',async()=>{
 const app=await start({nativeMode:'error-after-send',clocked:true,url:'https://wire.phoenixmail.ca/#h='+handoffToken});
 app.$('nim-handoff-pay').click();await settle();
 assert.equal(app.nativeCalls.length,1);assert.equal(app.$('nim-wallet-failure').hidden,true);
 assert.match(app.$('nim-progress-message').textContent,/Carte envoyée/);
 assert.equal(app.calls.filter(([p])=>p.endsWith('/checkout')).length,0);
});

test('repeat retry, form and pay clicks cannot overlap native wallet requests',async()=>{
 const gate={},app=await failedNative({nativeResults:['error','success'],holdNative:gate,holdNativeAt:2});
 await app.clock.advance(180000);app.$('nim-wallet-retry').click();await settle();
 assert.equal(app.nativeCalls.length,2);
 for(let i=0;i<5;i++){app.$('nim-wallet-retry').click();app.$('nim-handoff-pay').click();app.$('card-form').dispatchEvent(new app.window.Event('submit',{cancelable:true}));}
 await settle();assert.equal(app.nativeCalls.length,2);assert.equal(app.maxNativeActive(),1);
 gate.release();await settle();assert.equal(app.nativeCalls.length,2);assert.equal(app.maxNativeActive(),1);
});

test('retry handles never-started and expired checkouts, and fails closed when the chain cannot be read',async()=>{
 const unstarted=await failedNative({checkoutCode:'funding_not_started',nativeResults:['error','success']});
 unstarted.$('nim-wallet-retry').click();await settle();assert.equal(unstarted.nativeCalls.length,2);
 assert.equal(unstarted.calls.filter(([p])=>p.endsWith('/start')).length,2);assert.equal(unstarted.calls.filter(([p,b])=>p==='/api/payments'&&b).length,0);
 const expired=await failedNative({checkoutCode:'claim_expired',checkoutError:410});expired.$('nim-wallet-retry').click();await settle();
 assert.equal(expired.nativeCalls.length,1);assert.equal(expired.$('nim-wallet-retry').hidden,true);assert.match(expired.$('nim-progress-message').textContent,/Carte expirée/);
 const offline=await failedNative();offline.failReads(true);offline.$('nim-wallet-retry').click();await settle();
 assert.equal(offline.nativeCalls.length,1);assert.equal(offline.calls.filter(([p])=>p.endsWith('/checkout')).length,0);
 assert.match(offline.$('nim-progress-message').textContent,/Impossible de vérifier/);
});

test('native failure state, retry, copy and close explanation are translated in all five languages',async()=>{
 const expected={fr:['erreur réseau','Réessayer le paiement','Tu peux fermer'],en:['network error','Retry payment','You can close'],es:['error de red','Reintentar el pago','Puedes cerrar'],de:['Netzwerkfehler','Zahlung erneut versuchen','Du kannst schließen'],zh:['网络错误','重试付款','你可以关闭']};
 for(const [language,labels]of Object.entries(expected)){
  const app=await failedNative({redeemDraft:{...handoffDraft,language}});
  assert.ok(app.$('nim-progress-message').textContent.includes(labels[0]));assert.equal(app.$('nim-wallet-retry').textContent,labels[1]);
  assert.equal(app.$('nim-wallet-close-note').hidden,true);assert.ok(app.$('nim-wallet-copy').textContent);
  assert.equal(app.$('nim-wallet-cancel').hidden,false,'limited sessions can cancel their own unpaid card');
  app.$('nim-wallet-retry').click();await settle();assert.match(app.$('nim-wallet-wait').textContent,/3:00/);
  assert.ok(app.$('nim-wallet-wait').textContent.length>15);
 }
});

test('scoped reload after an interrupted native request restores retry without persisting the error or paying automatically',async()=>{
 const app=await start({session:{verified:false,email:'',handoff:{payment_id:'a'.repeat(48)}},nimStage:'funding_started',nativeMode:'error',clocked:true});
 assert.equal(app.$('nim-wallet-retry').hidden,false);assert.equal(app.nativeCalls.length,0);assert.equal(app.$('nim-wallet-detail').hidden,true);
 assert.match(app.$('nim-progress-message').textContent,/Not yet paid/);
 app.$('nim-wallet-retry').click();await settle();assert.match(app.$('nim-wallet-wait').textContent,/3:00/);
 assert.equal(app.nativeCalls.length,0);assert.equal(app.$('wire-cards').hidden,true);
});

test('Safari keeps the prepared NIM card and truthful ready, confirmation-pending and sent history states',async()=>{
 const app=await start({nimOnly:true,nimStage:'ready',clocked:true});
 assert.equal(app.$('wire-cards').hidden,false);assert.match(app.$('wire-cards-list').textContent,/Payment not received/);
 app.setStage('funding_started');await app.clock.advance(30000);
 assert.match(app.$('wire-cards-list').textContent,/Payment not received/);assert.ok(!/Payment received/.test(app.$('wire-cards-list').textContent));
 app.setStage('sent');await app.clock.advance(30000);assert.match(app.$('wire-cards-list').textContent,/Sent, not yet received/);
 assert.equal(app.$('wire-cards-list').children.length,1);assert.equal(app.nativeCalls.length,0);
});

test('wallet choice presents Pay, Wallet then Get and Phoenix buttons retain neutral surfaces with accent outlines',async()=>{
 const app=await start({nimOnly:true});await submitNim(app);
 const choices=[...app.doc.querySelectorAll('.wire-wallet-choice>a,.wire-wallet-choice>button')];
 assert.deepEqual(choices.map(el=>el.id),['wire-choice-open','wire-choice-hub','wire-choice-get','wire-choice-cancel']);
 assert.equal(choices[0].textContent,'Pay with Nimiq Pay');assert.equal(choices[1].textContent,'Nimiq Wallet');assert.equal(choices[2].textContent,'Get Nimiq Pay');
 const css=read('../internal/web/static/wire-pay.css');
 assert.match(css,/data-theme="dark"\]\{--wire-button-bg:#1b1e22;--wire-button-hover:#272b30/);
 assert.match(css,/data-theme="light-paper"\]\{--wire-button-bg:#ffffff;--wire-button-hover:#eef1f5/);
 assert.match(css,/background:var\(--wire-button-bg\);background-image:none/);
 assert.match(css,/border:1px solid var\(--accent\)/);
});

test('an unfunded started card has a non-sticky state and can always fold into a blank new form',async()=>{
 const app=await tracked({width:375});
 assert.match(app.$('nim-progress-message').textContent,/Not yet paid/);assert.match(app.$('nim-progress-message').textContent,/Started at \d\d:\d\d/);
 assert.equal(app.$('new-card').hidden,false);assert.equal(app.$('nim-wallet-retry').disabled,true);assert.match(app.$('nim-wallet-wait').textContent,/3:00/);
 const row=app.doc.querySelector('.history-row');assert.equal(row.querySelector('.nim-row-recover').hidden,true);assert.ok(!row.textContent.includes('Recover funds'));
 const before=app.window.sessionStorage.getItem('wire-request-id'),calls=app.calls.length;
 app.$('new-card').click();await settle();
 assert.notEqual(app.window.sessionStorage.getItem('wire-request-id'),before);
 assert.equal(app.doc.querySelector('.workspace').hidden,false);assert.equal(app.$('nim-progress').hidden,true);
 for(const id of ['recipient','amount','message']){assert.equal(app.$(id).value,'');assert.equal(app.$(id).readOnly,false);}
 assert.equal(app.$('preview-submit').hidden,false);assert.equal(app.$('preview-submit').disabled,false);
 assert.equal(app.$('new-card').hidden,false);assert.equal(app.calls.length,calls,'New card makes no request or cancellation');
 assert.deepEqual(JSON.parse(app.window.localStorage.getItem('wire-nim-folded')),['a'.repeat(48)]);
 assert.equal(app.$('payment-list').children.length,1);
 const style=app.doc.createElement('style');style.textContent=read('../internal/web/static/wire-pay.css');app.doc.head.append(style);
 assert.equal(app.window.getComputedStyle(app.$('nim-progress')).position,'static');
});

test('folded card ids persist across reloads, without restoring the old card or losing its history',async()=>{
 const first=await tracked();first.$('new-card').click();await settle();
 const saved=first.snapshot(),local=first.snapshotLocal();
 for(const tabState of [saved,{}]){
  const app=await tracked({saved:tabState,local,width:375});
  assert.equal(app.$('nim-progress').hidden,true);assert.equal(app.$('payment-card').hidden,true);
  for(const id of ['recipient','amount','message'])assert.equal(app.$(id).value,'');
  assert.equal(app.$('payment-list').children.length,1);assert.equal(app.$('wire-cards-list').children.length,1);
  assert.equal(app.nativeCalls.length,0);assert.equal(app.checkouts.length,0);
  await app.clock.advance(60000);assert.equal(app.$('nim-progress').hidden,true);
  assert.equal(app.$('preview-submit').hidden,false);
 }
});

test('a fresh form prepares a distinct card with a fresh request id and retains the previous pending record',async()=>{
 const app=await tracked();const original=app.window.sessionStorage.getItem('wire-request-id');
 app.$('new-card').click();app.$('recipient').value='new@example.test';app.$('amount').value='1.25';app.$('message').value='New intent';
 app.$('card-form').dispatchEvent(new app.window.Event('submit',{cancelable:true}));await settle();
 app.$('wire-choice-open').click();await settle();
 const creation=app.calls.find(([p,b])=>p==='/api/payments'&&b)[1];assert.notEqual(creation.request_id,original);assert.equal(creation.recipient,'new@example.test');assert.equal(creation.amount,'1.25');
 const handed=app.calls.find(([p])=>p==='/api/handoff')[1];assert.notEqual(handed.payment_id,'a'.repeat(48));
 assert.equal(app.$('payment-list').children.length,2);assert.ok(app.$('payment-list').textContent.includes('2.5 NIM'));
 assert.ok(!app.calls.some(([p])=>p.includes('/cancel')));assert.equal(app.nativeCalls.length,0);
});

test('late funding of a folded card updates My cards without replacing or clearing the new draft',async()=>{
 const app=await tracked();app.$('new-card').click();
 app.$('recipient').value='next@example.test';app.$('amount').value='3.75';app.$('message').value='Next card';
 const requestID=app.window.sessionStorage.getItem('wire-request-id');
 app.setStage('sent');await app.clock.advance(30000);
 assert.equal(app.$('recipient').value,'next@example.test');assert.equal(app.$('amount').value,'3.75');assert.equal(app.$('message').value,'Next card');
 assert.equal(app.window.sessionStorage.getItem('wire-request-id'),requestID);assert.equal(app.$('nim-progress').hidden,true);
 assert.equal(app.doc.querySelector('.workspace').hidden,false);assert.match(app.$('wire-cards-list').textContent,/Sent, not yet received/);
 assert.equal(app.nativeCalls.length,0);assert.equal(app.checkouts.length,0);
});

test('funded recovery errors and the ready link live on the same history row and survive refreshes',async()=>{
 const app=await tracked({nimStage:'funded',recoveryStatus:503});
 const row=app.doc.querySelector('.history-row'),recover=row.querySelector('.nim-row-recover');assert.equal(recover.hidden,false);
 recover.click();await settle();assert.equal(app.calls.filter(([p])=>p.endsWith('/recovery')).length,1);
 assert.match(row.querySelector('.nim-row-notice').textContent,/could not be prepared/);assert.equal(recover.disabled,false);
 app.$('new-card').click();await app.clock.advance(30000);
 assert.equal(app.doc.querySelector('.history-row'),row);assert.match(row.querySelector('.nim-row-notice').textContent,/could not be prepared/);
 app.setRecoveryStatus(200);recover.click();await settle();
 const link=row.querySelector('.nim-row-recovery-link');assert.equal(link.hidden,false);assert.match(link.href,/^https:\/\/hub\.nimiq\.com\/cashlink\/#/);
 assert.match(row.querySelector('.nim-row-notice').textContent,/Link ready/);
 assert.equal(app.calls.filter(([p])=>p.endsWith('/recovery')).length,2);
});

test('recovery never silently drops behind the wallet busy lock; folding while the wallet is open preserves the new form',async()=>{
 const gate={},paid={id:'b'.repeat(48),sender:'sender@example.test',recipient:'paid@example.test',amount:'1',luna:100000,message:'',address:'NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY',stage:'sent',claim_state:'unclaimed',created:'2026-10-23T13:00:00Z'};
 const app=await start({nimOnly:true,nativeMode:'success',nimStage:'ready',clocked:true,extraNim:[paid],holdNative:gate,holdNativeAt:1});
 await submitNim(app);assert.equal(app.nativeCalls.length,1);
 const row=app.doc.querySelector('[data-payment-id="'+paid.id+'"]'),recover=row.querySelector('.nim-row-recover');
 assert.equal(recover.disabled,false);recover.click();await settle();
 assert.ok(app.calls.some(([p])=>p==='/api/payments/'+paid.id+'/recovery'));assert.match(row.querySelector('.nim-row-notice').textContent,/Link ready/);
 app.$('new-card').click();assert.equal(app.$('recipient').value,'');assert.equal(app.$('preview-submit').disabled,true);
 gate.release();await settle();
 assert.equal(app.$('recipient').value,'');assert.equal(app.$('amount').value,'');assert.equal(app.$('nim-progress').hidden,true);
 assert.equal(app.$('preview-submit').disabled,false);assert.equal(app.nativeCalls.length,1);
});

test('a stalled recovery has visible progress, times out on its row and never locks New card',async()=>{
 const gate={},app=await tracked({nimStage:'funded',holdRecovery:gate});
 const row=app.doc.querySelector('.history-row'),recover=row.querySelector('.nim-row-recover');
 recover.click();recover.click();await settle();assert.equal(recover.disabled,true);assert.match(row.querySelector('.nim-row-notice').textContent,/Working/);
 assert.equal(app.$('new-card').hidden,false);app.$('new-card').click();
 await app.clock.advance(30000);assert.equal(app.doc.querySelector('.history-row'),row);
 assert.match(row.querySelector('.nim-row-notice').textContent,/could not be prepared/);assert.equal(recover.disabled,false);
 assert.equal(app.calls.filter(([p])=>p.endsWith('/recovery')).length,1);assert.equal(app.$('preview-submit').disabled,false);
});

test('limited app sessions can retry their card but never create a new one or recover funds',async()=>{
 const app=await start({session:{verified:false,email:'',handoff:{payment_id:'a'.repeat(48)}},nimStage:'funding_started',manualNim:true,nativeMode:'success',clocked:true});
 assert.equal(app.$('new-card').hidden,true);assert.equal(app.$('my-payments').hidden,true);assert.equal(app.$('nim-wallet-retry').hidden,false);
 app.$('new-card').click();assert.equal(app.$('nim-handoff-summary').hidden,false);
 assert.equal(app.$('nim-wallet-retry').disabled,true);await app.clock.advance(180000);assert.equal(app.$('nim-wallet-retry').disabled,false);
 assert.ok(!app.calls.some(([p,b])=>p.endsWith('/recovery')||p==='/api/payments'&&b));
 assert.equal(app.$('nim-wallet-close-note').hidden,true);
});

test('pending state and New card remain translated in five languages',async()=>{
 const expected={fr:['Démarrée à','Nouvelle carte'],en:['Started at','New card'],es:['Iniciada a las','Nueva tarjeta'],de:['Gestartet um','Neue Karte'],zh:['开始','新建卡片']};
 for(const[language,labels]of Object.entries(expected)){
  const app=await tracked({local:{'wire-language':language}});
  assert.ok(app.$('nim-progress-message').textContent.includes(labels[0]));assert.equal(app.$('new-card').textContent,labels[1]);
  app.$('new-card').click();assert.equal(app.$('preview-submit').hidden,false);
 }
});

test('classic NIM also folds its pending card into a new form without cancelling or breaking claim markup',async()=>{
 const app=await start({classic:true,nimPending:true,manualNim:true,clocked:true});
 assert.equal(app.$('new-card').hidden,false);app.$('new-card').click();await settle();
 for(const id of ['recipient','amount','message'])assert.equal(app.$(id).value,'');
 assert.equal(app.$('nim-progress').hidden,true);assert.equal(app.$('preview-submit').hidden,false);
 assert.ok(app.$('claim-form'));assert.equal(app.nativeCalls.length,0);
 assert.ok(!app.calls.some(([p])=>p.includes('/cancel')));
});

test('navigation has no decorative arrows, the close control is a large rounded square, and mobile controls avoid Safari focus zoom',async()=>{
 const t=await start({width:375});t.$('pay').click();await settle();
 assert.equal(t.doc.querySelector('.wire-navigation'),null);
 assert.ok(!/[←↗⤢]/.test(t.doc.querySelector('.nimiq-payment-head').textContent));
 const css=read('../internal/web/static/wire-pay.css'),style=t.doc.createElement('style');style.textContent=css;t.doc.head.append(style);
 const close=t.doc.querySelector('.nimiq-payment-head>button:last-child'),computed=t.window.getComputedStyle(close);
 assert.equal(computed.borderRadius,'9px');assert.equal(computed.width,'44px');assert.equal(computed.height,'44px');assert.ok(close.querySelector('svg'));
 assert.equal(t.window.getComputedStyle(t.$('wire-email')).fontSize,'16px');
 assert.equal(t.doc.querySelector('.nimiq-payment-head').onpointerdown,null);
 assert.ok(!/user-scalable\s*=\s*no|maximum-scale\s*=\s*1/.test(t.doc.querySelector('meta[name="viewport"]').content));
 assert.match(css,/\.wire-body \.nimiq-payment-window\{[^}]*max-width:calc\(100vw - 24px\)/);
 const win=t.doc.querySelector('.nimiq-payment-window');
 win.getBoundingClientRect=()=>({width:351,height:400});
 Object.defineProperty(t.window,'visualViewport',{value:{width:375,height:500,offsetTop:20,offsetLeft:0},configurable:true});
 t.window.PhoenixViewport.center(win);
 assert.equal(win.style.left,'12px');assert.equal(win.style.top,'70px');assert.equal(win.style.maxHeight,'484px');
});
test('the compact footer attribution keeps fixed PhoenixMail colours across Wire themes and palettes',async()=>{
 const app=await start({clocked:true}),style=app.doc.createElement('style');style.textContent=read('../internal/web/static/wire-pay.css');app.doc.head.append(style);
 const link=app.doc.querySelector('footer .wire-powered');assert.equal(link.textContent,'Powered by PhoenixMail');assert.equal(link.href,'https://phoenixmail.ca/');assert.equal(link.rel,'noopener noreferrer');
 for(const theme of ['dark','light-paper'])for(const palette of ['phoenixfire','aurora','ocean']){
  app.doc.documentElement.dataset.theme=theme;app.doc.documentElement.dataset.palette=palette;
  const css=app.window.getComputedStyle(link);assert.equal(css.borderTopColor,'rgb(255, 138, 24)');assert.equal(css.backgroundColor,'rgb(23, 27, 34)');assert.equal(css.fontSize,'14px');assert.equal(css.borderRadius,'999px');assert.equal(app.window.getComputedStyle(link.querySelector('strong span')).color,'rgb(255, 138, 24)');
 }
});

test('sent NIM replaces the form with one completion screen, clears its draft and Done removes the entire window',async()=>{
 const t=await tracked();
 const requestBefore=t.window.sessionStorage.getItem('wire-request-id');
 assert.equal(t.$('recipient').value,'friend@example.test');assert.equal(t.$('amount').value,'2.5');
 t.setStage('sent');await t.clock.advance(4000);
 const host=t.doc.querySelector('.wire-nim-host');
 assert.equal(host.classList.contains('wire-nim-complete'),true);
 for(const id of ['recipient','amount','message'])assert.equal(t.$(id).value,'');
 assert.equal(host.querySelector('.workspace').hidden,true);assert.equal(t.$('my-payments').hidden,true);assert.equal(t.$('notice').hidden,true);
 assert.equal(t.$('nim-progress').hidden,false);assert.equal(t.$('nim-progress-resend').hidden,true);
 assert.equal(t.doc.activeElement,t.$('nim-progress-close'));
 assert.notEqual(t.window.sessionStorage.getItem('wire-request-id'),requestBefore);
 assert.equal(t.window.sessionStorage.getItem('wire-form'),'');
 t.$('nim-progress-close').click();await settle();
 assert.equal(t.doc.querySelector('.wire-nim-window'),null);assert.equal(t.doc.querySelector('#nim-progress'),null);
 assert.equal(t.doc.activeElement,t.$('pay'));
 assert.match(t.$('wire-cards-list').textContent,/2.5 NIM.*friend@example.test.*Sent, not yet received/s);
 assert.equal(t.$('wire-card-notices').hidden,false);assert.match(t.$('wire-card-notices').textContent,/Email sent to friend@example.test/);
 const local=t.snapshotLocal();
 t.doc.querySelector('.wire-card-banner button').click();assert.equal(t.$('wire-card-notices').hidden,true);
 const dismissed=t.snapshotLocal();
 t.$('pay').click();await settle();
 assert.equal(t.doc.querySelector('.wire-nim-host').classList.contains('wire-nim-complete'),false);
 assert.equal(t.$('recipient').value,'');assert.equal(t.$('message').value,'');
 assert.equal(t.checkouts.length,0);assert.equal(t.nativeCalls.length,0);
 assert.equal(t.calls.filter(([p,b])=>p==='/api/payments'&&b).length,0);
 const restored=await start({clocked:true,nimStage:'sent',local});
 assert.equal(restored.$('wire-card-notices').hidden,false);
 const quiet=await start({clocked:true,nimStage:'sent',local:dismissed});
 assert.equal(quiet.$('wire-card-notices').hidden,true);
});

test('at 375 px the NIM completion hides every other panel and has no sticky or floating success box',async()=>{
 const t=await tracked();Object.defineProperty(t.window,'innerWidth',{value:375});
 const css=read('../internal/web/static/wire-pay.css');
 const style=t.doc.createElement('style');style.textContent=css;t.doc.head.append(style);
 t.setStage('sent');await t.clock.advance(4000);
 const visible=el=>!el.closest('[hidden]')&&t.window.getComputedStyle(el).display!=='none';
 assert.equal(t.window.innerWidth,375);
 assert.equal(t.window.getComputedStyle(t.$('nim-progress')).position,'static');
 assert.equal(t.window.getComputedStyle(t.$('nim-progress')).textAlign,'center');
 assert.equal(visible(t.doc.querySelector('.wire-nim-window>.nimiq-payment-head')),false);
 assert.equal(visible(t.doc.querySelector('.wire-nim-assets')),false);
 assert.equal(visible(t.doc.querySelector('.workspace')),false);
 assert.deepEqual([...t.doc.querySelector('.wire-nim-host').children].filter(visible).map(el=>el.id),['nim-progress']);
 t.$('nim-progress-close').click();await settle();assert.equal(t.doc.querySelector('.wire-nim-complete'),null);
 assert.match(css,/\.wire-nim-window:has\(\.wire-nim-complete\)\{[^}]*width:min\(520px,calc\(100vw - 24px\)\);min-width:0;box-sizing:border-box/);
 assert.match(css,/\.wire-nim-complete \.wire-nim-progress\{[^}]*position:static[^}]*align-items:center[^}]*width:100%[^}]*box-sizing:border-box/);
 assert.match(css,/@media\(max-width:420px\)\{\.wire-nim-host\.wire-nim-complete\{padding:16px\}/);
});

test('the gas light is a quiet line on the sign-in screen, in five languages, shown only while Wire really pays the fee', async () => {
  const on = { polygon: { available: true, reason: 'ok' }, ethereum: { available: false, reason: 'off' } };
  const t0 = await start({ session: { verified: false }, gas: on });
  assert.equal(t0.$('wire-sender').hidden, false, 'the sign-in screen is what is shown');
  assert.equal(t0.$('wire-gas').hidden, false, 'the light is there before anyone signs in');
  assert.equal(t0.$('wire-gas-text').textContent, 'Network fees on USDC and USDT are paid by Phoenix Wire');
  for (const [lang, word] of Object.entries({ fr: 'frais de réseau', es: 'comisiones de red', de: 'Netzwerkgebühren', zh: '网络费用' })) {
    t0.$('wire-language').value = lang;
    t0.$('wire-language').dispatchEvent(new t0.window.Event('change'));
    assert.match(t0.$('wire-gas-text').textContent, new RegExp(word), lang);
    assert.ok(t0.$('wire-gas-text').textContent.includes('Phoenix Wire'), lang);
  }
  const css = read('../internal/web/static/wire-pay.css');
  assert.match(css, /\.wire-led\{[^}]*width:8px[^}]*animation:wire-heartbeat/, 'a small beating dot, not a pill');
  assert.match(css, /prefers-reduced-motion:reduce\)\{\.wire-led\{animation:none/, 'no motion for those who ask for none');
  // signed in, it stays; not available (or off): nothing is shown
  assert.equal((await start({ gas: on })).$('wire-gas').hidden, false);
  assert.equal((await start({ session: { verified: false } })).$('wire-gas').hidden, true);
});

test('light palettes visibly separate backgrounds and keep text and field boundaries legible',async()=>{
 const t=await start({theme:'light'}),style=t.doc.createElement('style');style.textContent=read('../internal/web/static/wire-pay.css');t.doc.head.append(style);
 const luminance=hex=>{const c=hex.replace('#','');const v=[0,2,4].map(i=>parseInt(c.slice(i,i+2),16)/255).map(n=>n<=.04045?n/12.92:((n+.055)/1.055)**2.4);return .2126*v[0]+.7152*v[1]+.0722*v[2];};
 const contrast=(a,b)=>{const x=luminance(a),y=luminance(b);return(Math.max(x,y)+.05)/(Math.min(x,y)+.05);};
 const rules=[...style.sheet.cssRules],base=rules.find(r=>r.selectorText==='html.wire-app[data-theme="light-paper"]');
 for(const palette of ['phoenixfire','aurora','ocean']){
  const p=rules.find(r=>r.selectorText===`html.wire-app[data-theme="light-paper"][data-palette="${palette}"]`).style;
  for(const surface of ['--bg','--panel','--panel2']){
   assert.ok(contrast(base.style.getPropertyValue('--text'),p.getPropertyValue(surface))>=4.5,palette+' text');
   assert.ok(contrast(base.style.getPropertyValue('--muted'),p.getPropertyValue(surface))>=4.5,palette+' secondary text');
  }
  assert.ok(contrast(p.getPropertyValue('--line'),p.getPropertyValue('--panel2'))>=3,palette+' field boundary');
  assert.ok(contrast(p.getPropertyValue('--accent'),p.getPropertyValue('--panel'))>=3,palette+' accent border');
  const c=p.getPropertyValue('--bg').slice(1);const channels=[0,2,4].map(i=>parseInt(c.slice(i,i+2),16));assert.ok(Math.max(...channels)-Math.min(...channels)>=30,palette+' visible tint');
 }
});

test('claimed status stays inside My cards, with official currency icons in history and the currency picker',async()=>{
 const id='1'.repeat(32),t=await start({clocked:true,serverCards:[{id,asset:'usdc_polygon',amount:'2',recipient:'friend@example.test',invited:true}],state:()=>({state:'paid'})});
 await t.clock.advance(2500);assert.equal(t.$('wire-status').textContent,'');
 assert.match(t.$('wire-cards-list').textContent,/Claimed/);assert.ok(t.$('wire-cards').compareDocumentPosition(t.$('pay'))&t.window.Node.DOCUMENT_POSITION_PRECEDING);
 assert.equal(t.$('wire-cards-list').querySelector('.wire-coin-icon').getAttribute('src'),'/brand/usdc-polygon.png');
 t.$('pay').click();await settle();
 const buttons=[...t.doc.querySelectorAll('.wire-coin-picker .nimiq-payment-assets button')];
 assert.equal(buttons.length,4);for(const button of buttons){const img=button.querySelector('.wire-coin-icon');assert.ok(img);assert.equal(img.width,24);assert.equal(img.height,24);assert.equal(img.alt,assetMark(img.dataset.asset,'en').alt);}
 buttons.find(b=>b.textContent==='USDT · Ethereum').click();await settle();
 assert.equal(t.doc.querySelector('.nimiq-payment-head h3 img').getAttribute('src'),'/brand/usdt-ethereum.png');
 assert.equal(t.doc.querySelector('.nimiq-payment-head h3').textContent,'USDT · Ethereum');
});

test('history formats only server timestamps in five languages and never uses the observation time as a claim date',async()=>{
 const id='2'.repeat(32),created='2026-10-23T10:00:00Z',sent='2026-10-23T10:05:00Z',claimed='2026-10-23T11:00:00Z';
 const t=await start({clocked:true,serverCards:[{id,asset:'usdt_ethereum',amount:'5',recipient:'friend@example.test',invited:true,created,sent_at:sent}],state:()=>({state:'paid',claimed_at:claimed})});await t.clock.advance(2500);
 assert.deepEqual([...t.$('wire-cards-list').querySelectorAll('time')].map(el=>el.dateTime),[created,sent,claimed].map(d=>new Date(d).toISOString()));
 const labels={fr:['Créée le','Envoyée le','Réclamée le'],en:['Created','Sent','Claimed'],es:['Creada','Enviada','Reclamada'],de:['Erstellt','Gesendet','Eingelöst'],zh:['创建时间','发送时间','领取时间']};
 for(const[language,text]of Object.entries(labels)){
  t.$('wire-language').value=language;t.$('wire-language').dispatchEvent(new t.window.Event('change'));
  assert.deepEqual([...t.$('wire-cards-list').querySelectorAll('dt')].map(el=>el.textContent),text);
  assert.equal(t.$('wire-cards-list').querySelector('time').textContent,new Intl.DateTimeFormat(language,{dateStyle:'medium',timeStyle:'short'}).format(new Date(created)));
 }
 const old=await start({clocked:true,serverCards:[{id,asset:'usdc_polygon',amount:'2',recipient:'friend@example.test',invited:true,created:"invalid"}],state:()=>({state:'paid',claimed_at:'0001-01-01T00:00:00Z'})});await old.clock.advance(2500);
 assert.equal(old.$('wire-cards-list').querySelector('time'),null,'missing or zero dates are not invented');
 assert.equal(old.$('wire-cards-list').children.length,0,'undated records cannot enter the recent window');assert.equal(old.$('wire-status').textContent,'');
});

// Public NIM recipient page: no sender session, wallet payment, real chain or email.
const receivingAddress='NQ28 CQ7G QEBT 10UK R9LG 83C7 YAMY JD8Q K8S4';
const otherReceivingAddress='NQ24 0358 HL85 UQ05 SLSS C72R KEX3 NU2K RN0N';
async function recipientPage({language='en',accounts=null,accountError=null,payout=null,payoutError='',statusError='',expired=false,claimError='',holdStatus=null,token='',redeemStatus=200,redeemCode='123456',handoffStatus=200,holdRedeem=null}={}){
 const dom=new JSDOM(read('../internal/web/static/classic.html'),{url:'https://wire.example.test/?claim='+'a'.repeat(48)+(token?'&h='+encodeURIComponent(token):''),runScripts:'outside-only',pretendToBeVisual:true});doms.push(dom);
 const {window}=dom,doc=window.document,ctx=dom.getInternalVMContext();doc.documentElement.lang=language;
 const clock=fakeClock(window),calls=[],opened=[],logs=[],redeemURLs=[];let accountCalls=0;
 for(const method of ['log','info','warn','error','debug'])window.console[method]=(...args)=>logs.push(args);
 Object.defineProperty(window,'innerWidth',{value:375});
 if(accounts!==null||accountError)window.nimiq={async listAccounts(){accountCalls++;if(accountError)throw accountError;return accounts;}};
 window.open=url=>opened.push(url);
 window.PhoenixNimiqIdenticons={paint(){}};
 const answer=(status,body)=>({ok:status>=200&&status<300,status,json:async()=>body});
 let held=holdStatus,remote=payout;
 window.fetch=async(path,options={})=>{
  const body=options.body?JSON.parse(options.body):undefined;calls.push([path,body,options]);
  if(path.endsWith('/handoff/redeem')){
   redeemURLs.push(window.location.href);
   if(holdRedeem)await new Promise(resolve=>{holdRedeem.release=resolve;});
   return answer(redeemStatus,redeemStatus===200?{code:redeemCode}:{error:'invalid_or_expired'});
  }
  if(path.endsWith('/handoff'))return answer(handoffStatus,handoffStatus===200?{token:'R'.repeat(43),expires_in:300}:{code:'unavailable'});
  if(path.endsWith('/payout_status')){
   if(held){const gate=held;held=null;await new Promise(resolve=>{gate.release=resolve;});}
   if(statusError)return answer(503,{code:statusError});
   return remote?answer(200,remote):answer(404,{code:'payout_not_started'});
  }
  if(path==='/api/claims/'+'a'.repeat(48))return claimError?answer(claimError==='claim_unavailable'?404:410,{code:claimError}):answer(200,{id:'a'.repeat(48),sender:'sender@example.test',amount:'2.5',message:'Hello',address:receivingAddress,expires_at:new window.Date(window.Date.now()+(expired?-1:86400000)).toISOString()});
  if(path.endsWith('/request'))return answer(200,{sent:true});
  if(path.endsWith('/payout')){
   if(payoutError)return answer(({invalid_destination:400,invalid_or_expired_code:403,cashlink_not_confirmed_or_invalid:409,payout_already_started:409}[payoutError]||503),{code:payoutError});
   remote=remote||{status:'payout_pending',hash:'b'.repeat(64),to:body.to,value:250000,confirmations:0};return answer(200,remote);
  }
  if(path.endsWith('/release'))return answer(200,{cashlink:'https://hub.nimiq.com/cashlink/#'+Buffer.alloc(40,5).toString('base64url')});
  throw Error('Unexpected recipient request '+path);
 };
 Object.assign(ctx, { nimFunded,unpaidCard:p=>unpaidCard(p,window.Date.now()),recoverableCard,historyEntries:entries=>historyEntries(entries,window.Date.now()),historyPending:p=>historyPending(p,window.Date.now()),senderHistoryText,nimHistoryLabel:(p,language)=>nimHistoryLabel(p,language,window.Date.now()),cancelUnpaidCard,mountAdmin,adminText, flowText,addressValid,cardPresentation,paintCard,assetMark,renderCardPreview});
 const src=read('../internal/web/static/flow.js').replace(/^import[^\n]+\n/gm,'').replace(/^export /gm,'');
 // Keep the lifecycle handle so tests can assert timer cleanup too.
 vm.runInContext(src.replace("if(typeof document!=='undefined'&&document.getElementById('card-form'))mountNimFlow();","window.testFlow=mountNimFlow();"),ctx);
 await settle();
 const $=id=>doc.getElementById(id);
 return {window,doc,$,calls,opened,clock,logs,redeemURLs,accountCalls:()=>accountCalls,setRemote(p){remote=p;},setPayoutError(e){payoutError=e;},setStatusError(e){statusError=e;},
  async choose(route){$('claim-'+route).click();await settle();},
  enterCode(code='123456'){$('claim-code').value=code;$('claim-code').dispatchEvent(new window.Event('input'));}
 };
}

test('recipient has exactly two wallet choices after code and resend, disabled until six digits',async()=>{
 const gate={},app=await recipientPage({holdStatus:gate});
 assert.equal(app.calls.length,1);assert.ok(app.calls[0][0].endsWith('/payout_status'));
 assert.equal(app.$('claim-choices').disabled,true);gate.release();await settle();
 assert.equal(app.calls.filter(([p])=>p.endsWith('/request')).length,1);
 assert.match(app.$('claim-status').textContent,/Code sent/);
 const choices=[...app.$('claim-choices').querySelectorAll('button')];
 assert.deepEqual(choices.map(b=>b.textContent),['Receive in Nimiq Pay','Receive in Nimiq Wallet']);
 assert.equal(app.$('claim-code').closest('label').nextElementSibling,app.$('request-claim-code'));
 assert.equal(app.$('request-claim-code').nextElementSibling,app.$('claim-choices'));
 assert.equal(app.$('claim-form').querySelector('select'),null);
 assert.equal(app.$('claim-form').querySelectorAll('input').length,1);
 assert.equal(app.doc.querySelector('#release-claim'),null);
 for(const value of ['','123','12x456']){
  app.$('claim-code').value=value;app.$('claim-code').dispatchEvent(new app.window.Event('input'));
  for(const button of choices)assert.equal(button.disabled,true);
 }
 app.enterCode();for(const button of choices)assert.equal(button.disabled,false);
 assert.ok(!/\bHub\b/.test(app.$('claim-panel').textContent));
});

test('browser receipt transfers the code by POST only on click, with an opaque token in the app link',async()=>{
 const app=await recipientPage();
 assert.equal(app.calls.filter(([p])=>p.endsWith('/handoff')).length,0);
 app.enterCode();await app.choose('pay');
 assert.deepEqual(app.calls.find(([p])=>p.endsWith('/handoff'))[1],{code:'123456'});
 const uri=app.opened[0];assert.ok(uri.startsWith('nimiqpay://miniapp?url='));
 const target=new URL(new URL(uri).searchParams.get('url'));
 assert.equal(target.origin,'https://wire.phoenixmail.ca');assert.equal(target.searchParams.get('claim'),'a'.repeat(48));
 assert.equal(target.searchParams.get('h'),'R'.repeat(43));assert.equal(target.searchParams.size,2);
 assert.ok(!uri.includes('123456'));assert.ok(!uri.includes('%7B'));
 assert.equal(app.$('claim-code').value,'');assert.equal(app.$('claim-pay').disabled,false);
 assert.equal(app.calls.filter(([p])=>p.endsWith('/payout')||p.endsWith('/release')).length,0);
 await app.choose('pay');assert.equal(app.opened.length,2);
 app.window.open=()=>{throw Error('blocked');};await app.choose('pay');
 assert.equal(app.$('claim-wallet').disabled,false);
 await app.choose('wallet');assert.ok(app.calls.some(([p])=>p.endsWith('/release')));
 assert.deepEqual(app.logs,[]);assert.equal(app.window.localStorage.length,0);assert.equal(app.window.sessionStorage.length,0);
});

test('recipient handoff redemption removes the token before the response then pays once without code entry or email',async()=>{
 const gate={},token='T'.repeat(43),app=await recipientPage({token,accounts:[receivingAddress],holdRedeem:gate});
 assert.equal(app.window.location.search,'?claim='+'a'.repeat(48));
 assert.ok(app.redeemURLs.every(url=>!url.includes(token)&&!url.includes('h=')));
 assert.deepEqual(app.calls.find(([p])=>p.endsWith('/handoff/redeem'))[1],{token});
 assert.equal(app.accountCalls(),0);gate.release();await settle();
 assert.equal(app.accountCalls(),1);
 assert.deepEqual(app.calls.find(([p])=>p.endsWith('/payout'))[1],{code:'123456',to:receivingAddress});
 assert.equal(app.$('claim-code').value,'');assert.ok(!app.$('claim-panel').textContent.includes('123456'));
 assert.equal(app.$('claim-status').textContent,'Sending…');
 assert.equal(app.calls.filter(([p])=>p.endsWith('/request')).length,0);
 app.window.dispatchEvent(new app.window.Event('pageshow'));app.clock.hidden(false);await settle();
 assert.equal(app.calls.filter(([p])=>p.endsWith('/handoff/redeem')).length,1);
 assert.equal(app.calls.filter(([p])=>p.endsWith('/payout')).length,1);
 assert.deepEqual(app.logs,[]);assert.equal(app.window.localStorage.length,0);assert.equal(app.window.sessionStorage.length,0);
 app.setRemote({status:'payout_confirmed',hash:'b'.repeat(64),to:receivingAddress,value:250000,confirmations:2});
 await app.clock.advance(3000);assert.equal(app.$('claim-status').textContent,'Received!');
});

test('used or expired handoff quietly restores normal code entry; refused transferred code is cleared for re-entry',async()=>{
 for(const language of ['fr','en','es','de','zh']){
  const app=await recipientPage({language,token:'U'.repeat(43),accounts:[receivingAddress],redeemStatus:404});
  assert.equal(app.$('claim-code').closest('label').hidden,false);assert.equal(app.$('claim-code').value,'');
  assert.equal(app.$('claim-status').textContent,flowText(language,'claim_enterCode'));
  assert.equal(app.$('claim-status').className,'');assert.equal(app.$('claim-pay').disabled,true);
  assert.equal(app.calls.filter(([p])=>p.endsWith('/request')||p.endsWith('/payout')).length,0);
 }
 const denied=await recipientPage({token:'V'.repeat(43),accounts:[receivingAddress],payoutError:'invalid_or_expired_code'});
 assert.equal(denied.$('claim-code').closest('label').hidden,false);assert.equal(denied.$('claim-code').value,'');
 assert.equal(denied.$('claim-status').textContent,flowText('en','invalid'));
 denied.setPayoutError('');denied.enterCode('654321');await denied.choose('pay');
 assert.deepEqual(denied.calls.filter(([p])=>p.endsWith('/payout')).at(-1)[1],{code:'654321',to:receivingAddress});
});

test('a reloaded or already paid handoff follows the payout without another native call or redeem',async()=>{
 const payout={status:'payout_confirmed',hash:'b'.repeat(64),to:receivingAddress,value:250000,confirmations:2};
 const app=await recipientPage({token:'W'.repeat(43),accounts:[receivingAddress],payout});
 assert.equal(app.$('claim-status').textContent,'Received!');assert.equal(app.accountCalls(),0);
 assert.equal(app.$('claim-code').value,'');assert.equal(app.window.location.search,'?claim='+'a'.repeat(48));
 assert.equal(app.calls.filter(([p])=>p.endsWith('/payout')).length,0);
 const reload=await recipientPage({accounts:[receivingAddress],payout});
 assert.equal(reload.calls.filter(([p])=>p.endsWith('/handoff/redeem')||p.endsWith('/request')||p.endsWith('/payout')).length,0);
 assert.equal(reload.$('claim-status').textContent,'Received!');
 const malformed=await recipientPage({token:'Y'.repeat(43),accounts:[receivingAddress],redeemCode:'bad'});
 assert.equal(malformed.$('claim-code').closest('label').hidden,false);
 assert.equal(malformed.accountCalls(),0);assert.equal(malformed.$('claim-status').className,'');
});

test('handoff creation failures keep both wallet choices available and never open an app or consume the code',async()=>{
 const app=await recipientPage({handoffStatus:503});app.enterCode();await app.choose('pay');
 assert.equal(app.opened.length,0);assert.equal(app.$('claim-code').value,'123456');
 assert.equal(app.$('claim-pay').disabled,false);assert.equal(app.$('claim-wallet').disabled,false);
 assert.equal(app.calls.filter(([p])=>p.endsWith('/payout')||p.endsWith('/release')).length,0);
 assert.equal(app.$('claim-status').textContent,flowText('en','claim_payoutUnavailable'));
});

test('Nimiq Pay receipt uses the first account with no selector or automatic replacement code',async()=>{
 const app=await recipientPage({accounts:[receivingAddress,otherReceivingAddress]});
 assert.equal(app.calls.filter(([p])=>p.endsWith('/request')).length,0);
 assert.equal(app.accountCalls(),0);app.enterCode();await app.choose('pay');
 assert.equal(app.accountCalls(),1);
 const post=app.calls.find(([p])=>p.endsWith('/payout'));
 assert.deepEqual(post[1],{code:'123456',to:receivingAddress});
 assert.equal(post[2].method,'POST');assert.equal(post[2].headers['Content-Type'],'application/json');
 assert.equal(app.$('claim-form').querySelector('select'),null);
 assert.equal(app.$('claim-choices').hidden,true);assert.equal(app.$('claim-code').value,'');
 const requests=app.calls.length;await app.clock.advance(2999);assert.equal(app.calls.length,requests);
 app.setRemote({status:'payout_confirmed',to:receivingAddress,value:250000,hash:'b'.repeat(64),confirmations:2});
 await app.clock.advance(1);assert.equal(app.$('claim-status').textContent,'Received!');
 const stopped=app.calls.length;await app.clock.advance(30000);assert.equal(app.calls.length,stopped);
 assert.equal(app.window.localStorage.length,0);assert.equal(app.window.sessionStorage.length,0);
});

test('in-app resend is explicit, replaces the entered code and never starts a payout',async()=>{
 const app=await recipientPage({accounts:[receivingAddress]});app.enterCode();
 assert.equal(app.calls.filter(([p])=>p.endsWith('/request')).length,0);
 app.$('request-claim-code').click();await settle();
 assert.equal(app.calls.filter(([p])=>p.endsWith('/request')).length,1);
 assert.equal(app.$('claim-code').value,'');assert.equal(app.$('claim-pay').disabled,true);
 assert.equal(app.$('claim-wallet').disabled,true);assert.equal(app.$('request-claim-code').disabled,true);
 assert.equal(app.accountCalls(),0);assert.ok(!app.calls.some(([p])=>p.endsWith('/payout')));
});

test('account refusal, legacy errors and empty or malformed first accounts keep both choices usable',async()=>{
 for(const options of [{accountError:{type:'PERMISSION_DENIED'}},{accounts:{error:{type:'PERMISSION_DENIED'}}},{accounts:[]},{accounts:['invalid',receivingAddress]},{accountError:Error('unavailable')}]){
  const app=await recipientPage(options);app.enterCode();await app.choose('pay');
  assert.ok(app.$('claim-status').textContent.length>10);assert.equal(app.$('claim-pay').disabled,false);
  assert.equal(app.$('claim-wallet').disabled,false);
  assert.ok(!app.calls.some(([p])=>p.endsWith('/request')||p.endsWith('/payout')));
  assert.equal(app.$('claim-code').value,'123456');
 }
});

test('Nimiq Wallet uses the existing release action and only its original button opens the released link',async()=>{
 const app=await recipientPage();app.enterCode();await app.choose('wallet');
 assert.deepEqual(app.calls.find(([p])=>p.endsWith('/release'))[1],{code:'123456'});
 assert.match(app.$('claim-status').textContent,/Code accepted/);assert.equal(app.$('request-claim-code').hidden,true);
 assert.equal(app.$('claim-pay').hidden,true);assert.equal(app.$('claim-wallet').disabled,false);
 assert.equal(app.$('claim-wallet').textContent,'Receive in Nimiq Wallet');
 await app.choose('wallet');assert.match(app.opened[0],/^https:\/\/hub.nimiq.com\/cashlink\/#/);
 assert.ok(!app.calls.some(([p])=>p.endsWith('/payout')));
 assert.equal(app.calls.filter(([p])=>p.endsWith('/release')).length,1);
});

test('recipient resumes pending, confirmed and failed payouts without code even after expiry',async()=>{
 for(const status of ['payout_pending','payout_confirmed','payout_failed']){
  const app=await recipientPage({payout:{status,hash:'c'.repeat(64),to:receivingAddress,value:250000,confirmations:status==='payout_confirmed'?2:0},expired:true});
  assert.equal(app.$('claim-status').textContent,flowText('en','claim_'+status));assert.equal(app.$('claim-choices').hidden,true);
  assert.equal(app.$('request-claim-code').hidden,true);assert.ok(!app.calls.some(([p])=>p.endsWith('/request')||p.endsWith('/payout')));
  if(status==='payout_pending'){
   app.$('claim-retry').click();await settle();assert.deepEqual(app.calls.find(([p])=>p.endsWith('/payout'))[1],{to:receivingAddress});
   app.clock.hidden(true);const n=app.calls.length;await app.clock.advance(9000);assert.equal(app.calls.length,n);
   app.clock.hidden(false);await settle();assert.ok(app.calls.length>n);
   app.window.testFlow.destroy();const count=app.calls.length;await app.clock.advance(30000);assert.equal(app.calls.length,count);
  }else{await app.clock.advance(30000);assert.equal(app.calls.filter(([p])=>p.endsWith('/payout_status')).length,1);}
 }
});

test('server destination/code/funding errors preserve the code and allow retry with the same automatic account',async()=>{
 for(const error of ['invalid_destination','invalid_or_expired_code','cashlink_not_confirmed_or_invalid']){
  const app=await recipientPage({accounts:[receivingAddress],payoutError:error});app.enterCode();await app.choose('pay');
  assert.equal(app.$('claim-code').value,error==='invalid_or_expired_code'?'':'123456');
  assert.equal(app.$('claim-status').textContent,error==='invalid_or_expired_code'?flowText('en','invalid'):flowText('en','claim_'+error));
  app.setPayoutError('');app.enterCode();await app.choose('pay');
  assert.equal(app.$('claim-status').textContent,'Sending…');
 }
});

test('lost replies and destination conflicts resume only the locked server destination',async()=>{
 const app=await recipientPage({accounts:[receivingAddress]});app.enterCode();
 const original=app.window.fetch;app.window.fetch=async(p,o)=>{const result=await original(p,o);if(p.endsWith('/payout'))throw Error('response lost');return result;};
 await app.choose('pay');assert.equal(app.$('claim-status').textContent,'Sending…');
 assert.equal(app.$('claim-choices').hidden,true);assert.equal(app.$('claim-code').value,'');
 const conflict=await recipientPage({accounts:[receivingAddress],payoutError:'payout_already_started'});conflict.enterCode();
 conflict.setRemote({status:'payout_pending',hash:'b'.repeat(64),to:otherReceivingAddress,value:250000,confirmations:0});
 await conflict.choose('pay');assert.ok(conflict.$('claim-payout-details').textContent.includes(otherReceivingAddress));
 assert.equal(conflict.$('claim-wallet').closest('fieldset').hidden,true);
 conflict.setPayoutError('');conflict.$('claim-retry').click();await settle();
 assert.equal(conflict.calls.filter(([p])=>p.endsWith('/payout')).at(-1)[1].to,otherReceivingAddress);
});

test('unavailable status fails closed and unavailable payouts remain retryable',async()=>{
 const app=await recipientPage({statusError:'chain_unavailable'});app.enterCode();
 assert.equal(app.$('claim-choices').disabled,true);assert.equal(app.$('claim-retry').hidden,false);
 await app.choose('wallet');assert.equal(app.calls.filter(([p])=>p.endsWith('/release')).length,0);
 app.setStatusError('');app.$('claim-retry').click();await settle();assert.equal(app.$('claim-choices').disabled,false);
 for(const code of ['chain_unavailable','payout_unavailable']){
  const failed=await recipientPage({accounts:[receivingAddress],payoutError:code});failed.enterCode();await failed.choose('pay');
  assert.equal(failed.$('claim-status').textContent,flowText('en','claim_payoutUnavailable'));
  failed.setPayoutError('');failed.$('claim-retry').click();await settle();assert.equal(failed.$('claim-status').textContent,'Sending…');
 }
});

test('duplicate wallet taps send one payout; Enter does not silently choose a wallet',async()=>{
 const app=await recipientPage({accounts:[receivingAddress]});app.enterCode();
 app.$('claim-form').dispatchEvent(new app.window.Event('submit',{cancelable:true}));await settle();
 assert.equal(app.accountCalls(),0);
 for(let i=0;i<3;i++)app.$('claim-pay').click();
 await settle();assert.equal(app.calls.filter(([p])=>p.endsWith('/payout')).length,1);
 assert.ok(app.calls.every(([p])=>p.startsWith('/api/claims/')));
});

test('two-choice recipient labels and narrow layout cover five languages with no technical wallet name',async()=>{
 const labels={fr:['Recevoir dans Nimiq Pay','Recevoir dans Nimiq Wallet','Renvoyer le code'],en:['Receive in Nimiq Pay','Receive in Nimiq Wallet','Resend code'],es:['Recibir en Nimiq Pay','Recibir en Nimiq Wallet','Reenviar código'],de:['In Nimiq Pay empfangen','In Nimiq Wallet empfangen','Code erneut senden'],zh:['接收到 Nimiq Pay','接收到 Nimiq Wallet','重新发送验证码']};
 for(const [language,expected]of Object.entries(labels)){
  const app=await recipientPage({language,accounts:[receivingAddress]});
  assert.deepEqual([app.$('claim-pay').textContent,app.$('claim-wallet').textContent,app.$('request-claim-code').textContent],expected);
  for(const key of ['choose','pay','wallet','resend','enterCode','openPay','retry','checkingPayout','payout_pending','payout_confirmed','payout_failed','invalid_destination','cashlink_not_confirmed_or_invalid','payout_already_started','payoutUnavailable','accountsUnavailable','accountsCancelled','details']){
   assert.notEqual(flowText(language,'claim_'+key),'claim_'+key);assert.ok(!/Mainnet|OTP|\bHub\b/.test(flowText(language,'claim_'+key)));
  }
  app.enterCode();await app.choose('pay');
  const css=app.doc.createElement('style');css.textContent=read('../internal/web/static/wire-pay.css');app.doc.head.append(css);
  assert.equal(app.window.innerWidth,375);assert.equal(app.window.getComputedStyle(app.$('claim-choices')).display,'none');
  assert.equal(app.doc.querySelector('#claim-payout-details img'),null);
 }
});

test('NIM success contains only message and Done, and unpaid actions precede diagnostic details',async()=>{
 const app=await tracked({width:375});app.setStage('sent');await app.clock.advance(4000);
 assert.equal(app.$('new-card').hidden,true);
 const css=app.doc.createElement('style');css.textContent=read('../internal/web/static/wire-pay.css');app.doc.head.append(css);
 const visible=el=>!el.hidden&&app.window.getComputedStyle(el).display!=='none';
 assert.deepEqual([...app.$('nim-progress').children].filter(visible).map(x=>x.id),['nim-progress-message','nim-progress-close']);
 const failed=await failedNative({local:{'wire-language':'en'}});const style=failed.doc.createElement('style');style.textContent=css.textContent;failed.doc.head.append(style);
 failed.$('wire-language').value='en';failed.$('wire-language').dispatchEvent(new failed.window.Event('change'));await settle();
 assert.equal(failed.$('nim-wallet-close-note').hidden,true);
 assert.equal(failed.$('nim-wallet-close-note').textContent.includes('debited'),false);
 assert.equal(failed.window.getComputedStyle(failed.$('nim-wallet-retry')).order,'-2');
});

test('375px unpaid NIM diagnostics use content-sized rows, never 240px paragraph spacers',async()=>{
 const app=await failedNative({width:375});
 const style=app.doc.createElement('style');style.textContent=read('../internal/web/static/wire-pay.css');app.doc.head.append(style);
 const panel=app.$('nim-wallet-failure'),css=app.window.getComputedStyle(panel);
 assert.equal(css.display,'grid');assert.equal(css.gridTemplateColumns,'minmax(0,1fr)');assert.equal(css.gap,'10px');
 for(const p of panel.querySelectorAll('p')){
  const computed=app.window.getComputedStyle(p);
  assert.equal(computed.flexGrow,'0');assert.equal(computed.flexShrink,'0');assert.equal(computed.flexBasis,'auto');
  assert.equal(computed.minHeight,'0');assert.equal(computed.height,'auto');
 }
 assert.equal(app.window.getComputedStyle(app.$('nim-wallet-detail')).margin,'0px');
 assert.equal(app.window.getComputedStyle(app.$('nim-wallet-copy')).margin,'0px');
 assert.equal(app.window.getComputedStyle(app.$('nim-progress')).position,'static');
 assert.equal(app.$('nim-wallet-retry').hidden,false);
});


test('dashboard polls missing, unclaimed and claiming state every six seconds and removes terminal actions',async()=>{
 const app=await start({nimStage:'sent',clocked:true,nimOnly:true});app.setClaimState(undefined);
 await app.clock.advance(6000);let count=app.calls.filter(([p])=>p==='/api/payments').length;
 assert.match(app.$('wire-cards-list').textContent,/Checking/);
 app.setClaimState('unclaimed');await app.clock.advance(6000);
 assert.ok(app.calls.filter(([p])=>p==='/api/payments').length>count);assert.match(app.$('wire-cards-list').textContent,/Recover funds/);
 app.setClaimState('claiming');await app.clock.advance(6000);
 assert.match(app.$('wire-cards-list').textContent,/Sending/);assert.equal(app.$('wire-cards-list').querySelectorAll('button,a').length,0);
 app.setClaimState('claimed');await app.clock.advance(6000);
 assert.match(app.$('wire-cards-list').textContent,/Received/);assert.equal(app.$('wire-cards-list').querySelectorAll('button,a').length,0);
 count=app.calls.filter(([p])=>p==='/api/payments').length;await app.clock.advance(18000);assert.equal(app.calls.filter(([p])=>p==='/api/payments').length,count);
});
test('dashboard cancellation uses one confirmation, hides the card, and never pays',async()=>{
 const app=await start({nimStage:'ready',clocked:true,nimOnly:true});
 const button=[...app.$('wire-cards-list').querySelectorAll('button')].find(b=>b.textContent==='Cancel card');assert.ok(button);button.click();await settle();
 assert.deepEqual(app.confirmations,['Cancel this unpaid card?']);assert.equal(app.$('wire-cards-list').children.length,0);
 assert.deepEqual(app.calls.find(([p])=>p.endsWith('/cancel'))[1],{});assert.equal(app.checkouts.length,0);assert.equal(app.nativeCalls.length,0);
});
test('cancellation refused by the sender makes no request',async()=>{
 const app=await start({nimStage:'ready',cancelConfirm:false,nimOnly:true});[...app.$('wire-cards-list').querySelectorAll('button')].find(b=>b.textContent==='Cancel card').click();await settle();
 assert.ok(!app.calls.some(([p])=>p.endsWith('/cancel')));assert.equal(app.$('wire-cards-list').children.length,1);
});
test('already funded cancellation is successful feedback and confirms the same card',async()=>{
 const app=await start({nimStage:'ready',cancelCode:'already_funded',nimOnly:true,clocked:true});[...app.$('wire-cards-list').querySelectorAll('button')].find(b=>b.textContent==='Cancel card').click();await settle();
 assert.match(app.$('wire-cards-list').textContent,/The payment has arrived/);assert.match(app.$('wire-cards-list').textContent,/Recover funds/);
 assert.ok(app.calls.some(([p])=>p.endsWith('/confirm')));assert.ok(!app.$('wire-cards-list').textContent.includes('Cancel card'));assert.equal(app.checkouts.length,0);
});
test('unpaid error panel and embedded history both cancel the same card',async()=>{
 for(const fromHistory of [false,true]){
  const app=await tracked();assert.equal(app.$('nim-wallet-cancel').hidden,false);
  const cancel=fromHistory?app.doc.querySelector('.nim-row-cancel'):app.$('nim-wallet-cancel');assert.equal(cancel.hidden,false);cancel.click();await settle();
  assert.equal(app.$('payment-list').children.length,0);assert.equal(app.$('nim-progress').hidden,true);assert.equal(app.$('wire-cards-list').children.length,0);
  assert.equal(app.calls.filter(([p])=>p.endsWith('/cancel')).length,1);assert.equal(app.nativeCalls.length,0);
 }
});
test('chain unavailability keeps cancellation retryable and cannot_cancel refreshes normal tracking',async()=>{
 for(const cancelCode of ['chain_unavailable','cannot_cancel']){
  const app=await start({nimStage:'ready',cancelCode,nimOnly:true});[...app.$('wire-cards-list').querySelectorAll('button')].find(b=>b.textContent==='Cancel card').click();await settle();
  assert.match(app.$('wire-cards-list').textContent,cancelCode==='cannot_cancel'?/no longer be cancelled/:/please try again/);
  assert.equal([...app.$('wire-cards-list').querySelectorAll('button')].find(b=>b.textContent==='Cancel card').disabled,false);
 }
});
test('combined dashboard has three recent rows plus only the funded unclaimed older exception',async()=>{
 const mk=(id,age,stage='sent',claim_state='unclaimed')=>({id:id.repeat(48),sender:'sender@example.test',recipient:id+'@example.test',amount:'1',address:'NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY',created:new Date(Date.parse('2026-10-23T14:00:00Z')-age).toISOString(),stage,claim_state});
 const app=await start({clocked:true,extraNim:[mk('b',1000),mk('c',3000),mk('d',4000),mk('e',90000000),mk('f',90000000,'ready',undefined)],serverCards:[{id:'1'.repeat(32),asset:'usdc_polygon',amount:'1',created:'2026-10-23T13:59:58Z',claimed_at:'2026-10-23T14:00:00Z'}]});
 assert.deepEqual([...app.$('wire-cards-list').children].map(r=>r.dataset.cardId),['b'.repeat(48),'1'.repeat(32),'c'.repeat(48),'e'.repeat(48)]);
 assert.match(app.$('wire-cards-list').children[1].textContent,/Received/);
});
test('cancelled late-funded cards remain recoverable while other cancellations are hidden',async()=>{
 const p={id:'b'.repeat(48),sender:'sender@example.test',recipient:'late@example.test',amount:'1',address:'NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY',created:'2026-10-21T14:00:00Z',stage:'cancelled',claim_state:'unclaimed'};
 const app=await start({clocked:true,nimStage:'cancelled',extraNim:[p]});
 assert.equal(app.$('wire-cards-list').children.length,1);assert.match(app.$('wire-cards-list').textContent,/Recover funds/);assert.ok(!app.$('wire-cards-list').textContent.includes('Retry payment'));
});
test('NIM history revokes an already prepared recovery link as claiming and claimed arrive',async()=>{
 const app=await tracked({nimStage:'sent'});
 let row=app.doc.querySelector('.history-row');row.querySelector('.nim-row-recover').click();await settle();assert.equal(row.querySelector('.nim-row-recovery-link').hidden,false);
 for(const state of ['claiming','claimed']){
  app.setClaimState(state);await app.clock.advance(6000);row=app.doc.querySelector('.history-row');
  for(const selector of ['.nim-row-recover','.nim-row-recovery-link','.nim-row-retry','.nim-row-cancel'])assert.equal(row.querySelector(selector).hidden,true);
 }
 assert.equal(row.querySelector('.nim-row-state').textContent,'Received');
});
test('cancelled recent cards keep checking for late funds without showing payment controls',async()=>{
 const app=await start({nimStage:'cancelled',clocked:true,nimOnly:true});assert.equal(app.$('wire-cards-list').children.length,0);
 app.setClaimState('unclaimed');await app.clock.advance(6000);
 assert.match(app.$('wire-cards-list').textContent,/Recover funds/);assert.ok(!app.$('wire-cards-list').textContent.includes('Retry payment'));
});




test('sent-card audio plays once for a recent NIM card, but not for old or future cards, and rejection is harmless',async()=>{
 for(const [createdAgo,expected] of [[0,1],[29*60000,1],[30*60000,0],[31*60000,0],[-60000,0]]){
  const app=await tracked({createdAgo,audioReject:true});
  assert.equal(app.soundPlays.length,0);app.setStage('sent');await app.clock.advance(4000);await settle();
  assert.equal(app.soundPlays.length,expected);
  app.window.dispatchEvent(new app.window.Event('pageshow'));await app.clock.advance(10000);await settle();
  assert.equal(app.soundPlays.length,expected);
  if(expected){assert.equal(app.soundPlays[0].src,'/sounds/phoenix-notification.wav?v=0.4.38');assert.equal(app.soundPlays[0].volume,.35);assert.equal(app.soundPlays[0].time,0);}
 }
});
test('maintenance always leaves administrator sign-in reachable, including invitation-only classic pages',async()=>{
 for(const classic of [false,true])for(const language of ['fr','en','es','de','zh']){
  const app=await start({classic,maintenance:true,inviteOnly:true,session:{verified:false},clocked:true,local:{'wire-language':language}});
  if(classic){app.doc.documentElement.lang=language;app.window.dispatchEvent(new app.window.Event('wire-language'));}
  const banner=app.$('wire-maintenance');assert.equal(banner.hidden,false);assert.equal(banner.textContent,flowText(language,'maintenance'));
  const input=app.$(classic?'sender':'wire-email');assert.equal(classic?input.closest('label').hidden:app.$('wire-sender').hidden,false);
  assert.equal(app.$(classic?'sender-invite-gate':'wire-invite-gate').hidden,true);
  app.failCode('maintenance');input.value='sender@example.test';app.$(classic?'send-sender-code':'wire-send-code').click();await settle();
  assert.equal(app.$(classic?'sender-status':'wire-status').textContent,flowText(language,'maintenance'));assert.equal(input.disabled,false);assert.equal(input.readOnly,false);
  assert.equal(app.$(classic?'sender-code-box':'wire-code-box').hidden,true);
 }
});
test('stablecoin sent audio is limited to a newly sent recent card already known on this browser',async()=>{
 for(const asset of ['usdc_polygon','usdt_polygon','usdt_ethereum'])for(const age of [0,31*60000]){
  const id='e'.repeat(32),created=new Date(Date.parse('2026-10-23T14:00:00Z')-age).toISOString();
  const card={id,asset,amount:'2',sender:'sender@example.test',created,invited:true};
  const local={'wire-usdt-cards':JSON.stringify([{...card,created:Date.parse(created),invited:false}])};
  const app=await start({clocked:true,serverCards:[card],local});
  assert.equal(app.soundPlays.length,age?0:1,asset);
  app.window.dispatchEvent(new app.window.Event('pageshow'));await app.clock.advance(10000);await settle();assert.equal(app.soundPlays.length,age?0:1);
 }
 const app=await start({clocked:true,serverCards:[{id:'e'.repeat(32),asset:'usdc_polygon',amount:'2',created:'2026-10-23T14:00:00Z',invited:true}]});
 assert.equal(app.soundPlays.length,0,'a card discovered on another browser is not a new send');
});

test('the sent banner turns into "payment received" once the card was claimed, and is not asked to stay on "valid until"',async()=>{
 const id='f'.repeat(32),card={id,asset:'usdc_polygon',amount:'2',sender:'sender@example.test',recipient:'friend@example.test',invited:true,noticePending:true};
 const claimed=await start({clocked:true,local:{'wire-usdt-cards':JSON.stringify([{...card,state:'paid'}])},serverCards:[{...card,state:'paid'}],state:()=>({state:'paid',balanceUnits:0})});await claimed.clock.advance(7000);
 const text=claimed.$('wire-card-notices').textContent;
 assert.equal(claimed.$('wire-card-notices').hidden,false);assert.match(text,/Payment received by friend@example.test\./);assert.doesNotMatch(text,/Card funded|alid until/);
 const waiting=await start({clocked:true,local:{'wire-usdt-cards':JSON.stringify([card])},serverCards:[card],state:()=>({state:'ready',balanceUnits:2100000})});await waiting.clock.advance(2500);
 assert.match(waiting.$('wire-card-notices').textContent,/Card funded\. Email sent to friend@example\.test\./);
});


test('known low balance is simple, localized and cancellable in the limited payment window',async()=>{
 const words={fr:'Ton solde est insuffisant pour payer cette carte.',en:'Your balance is too low to pay for this card.',es:'Tu saldo es insuficiente para pagar esta tarjeta.',de:'Dein Guthaben reicht nicht aus, um diese Karte zu bezahlen.',zh:'你的余额不足以支付这张卡片。'};
 for(const [language,text]of Object.entries(words)){
  const app=await failedNative({redeemDraft:{...handoffDraft,language},nativeError:{type:'INTERNAL_ERROR',code:-32603,message:'Failed to send payment transaction: Transaction value exceeds balance'},width:375});
  assert.equal(app.$('nim-progress-message').textContent,text);
  assert.equal(app.$('nim-wallet-details').open,false);
  assert.match(read('../internal/web/static/wire-pay.css'),/#nim-wallet-details:not\(\[open\]\) > :not\(summary\) \{ display:none; \}/);
  assert.equal(app.$('nim-wallet-cancel').hidden,false);
  assert.equal(app.$('nim-wallet-retry').hidden,false);
  const before=app.calls.length;
  app.$('nim-wallet-cancel').click();await settle();
  const calls=app.calls.slice(before);
  assert.deepEqual(calls.filter(([p])=>p.endsWith('/cancel')), [['/api/payments/'+ 'a'.repeat(48)+'/cancel',{}]]);
  assert.equal(calls.filter(([p,b])=>p==='/api/payments'&&b||p.startsWith('/api/sender/')).length,0);
  assert.equal(app.nativeCalls.length,1);
  assert.equal(app.doc.querySelector('.wire-nim-window'),null);
  assert.equal(app.$('wire-handoff-state').textContent,({fr:'Carte annulée.',en:'Card cancelled.',es:'Tarjeta cancelada.',de:'Karte storniert.',zh:'卡片已取消。'})[language]);
  assert.equal(app.confirmations.length,1);
  assert.equal(app.doc.querySelector('.wire-nim-window'),null);
  assert.equal(app.$('wire-entry').hidden,true,'no reauthentication after cancellation');
  assert.equal(app.$('wire-handoff-open').textContent,({fr:'Terminé',en:'Done',es:'Listo',de:'Fertig',zh:'完成'})[language]);
  assert.equal(app.$('wire-handoff-exit-help').hidden,false);
 }
});

test('limited cancellation resumes normal tracking when funding won the race',async()=>{
 const app=await failedNative({cancelCode:'already_funded'});
 const before=app.calls.length;app.$('nim-wallet-cancel').click();await settle();
 const calls=app.calls.slice(before);
 assert.ok(calls.some(([p])=>p.endsWith('/cancel')));
 assert.ok(calls.some(([p])=>p.endsWith('/confirm')));
 assert.equal(app.$('nim-wallet-failure').hidden,true);
 assert.equal(app.$('nim-handoff-pay').hidden,true);
 assert.ok(!/cancelled|annulée/i.test(app.$('nim-progress-message').textContent));
 assert.equal(app.nativeCalls.length,1);
 assert.equal(calls.filter(([p,b])=>p==='/api/payments'&&b||p.startsWith('/api/sender/')).length,0);
});

test('decline, transport failure and pending verification are distinct from a low balance',async()=>{
 const denied=await failedNative({nativeMode:'denied',redeemDraft:{...handoffDraft,language:'en'}});
 assert.equal(denied.$('nim-progress-message').textContent,'Payment declined in Nimiq Pay.');
 const offline=await failedNative({redeemDraft:{...handoffDraft,language:'en'}});
 assert.equal(offline.$('nim-progress-message').textContent,'A network error interrupted the payment request.');
 const pending=await failedNative({nativeMode:'success-no-confirm',redeemDraft:{...handoffDraft,language:'en'}});
 assert.ok(!/balance|declined|INTERNAL_ERROR|-32603/.test(pending.$('nim-progress-message').textContent));
 assert.equal(pending.$('nim-wallet-failure').hidden,true,'a returned hash is not a wallet rejection');
 for(const app of [denied,offline])assert.equal(app.$('nim-wallet-details').open,false);
 const unknown=await failedNative({nativeError:{type:'INTERNAL_ERROR',code:-32603,message:'Unexpected provider failure'},redeemDraft:{...handoffDraft,language:'en'}});
 assert.ok(!/balance|INTERNAL_ERROR|-32603/.test(unknown.$('nim-progress-message').textContent));
});

test('creation inside Nimiq Pay cancels the same card without a cross-browser detour',async()=>{
 const app=await start({nimOnly:true,nativeMode:'error',nativeError:{type:'INTERNAL_ERROR',code:-32603,message:'Transaction value exceeds balance'}});
 await submitNim(app);
 assert.equal(app.$('nim-progress-message').textContent,'Your balance is too low to pay for this card.');
 const before=app.calls.length;app.$('nim-wallet-cancel').click();await settle();
 assert.equal(app.calls.slice(before).filter(([p])=>p.endsWith('/cancel')).length,1);
 assert.equal(app.calls.slice(before).filter(([p,b])=>p==='/api/payments'&&b||p.startsWith('/api/sender/')).length,0);
 assert.equal(app.nativeCalls.length,1);
 assert.equal(app.$('nim-progress').hidden,true);
 assert.equal(app.$('wire-cards-list').children.length,0);
});


test('declining cancellation keeps the limited card without POST or navigation',async()=>{
 const app=await failedNative({cancelConfirm:false});const before=app.calls.length;
 app.$('nim-wallet-cancel').click();await settle();
 assert.equal(app.calls.slice(before).filter(([p])=>p.endsWith('/cancel')).length,0);
 assert.equal(app.$('nim-wallet-cancel').hidden,false);
 assert.equal(app.$('wire-entry').hidden,true);
 assert.equal(app.nativeCalls.length,1);
});


test('cancelled limited cards replace View with Done and keep translated exit help when close is refused',async()=>{
 const labels={fr:'Terminé',en:'Done',es:'Listo',de:'Fertig',zh:'完成'};
 const hints={fr:'Tu peux revenir à ton navigateur',en:'You can return to your browser',es:'Puedes volver a tu navegador',de:'zu deinem Browser zurückkehren',zh:'返回浏览器'};
 for(const [language,label]of Object.entries(labels)){
  const app=await failedNative({redeemDraft:{...handoffDraft,language}});
  app.$('nim-wallet-cancel').click();await settle();
  const button=app.$('wire-handoff-open'),help=app.$('wire-handoff-exit-help');
  assert.equal(button.hidden,false);assert.equal(button.textContent,label);
  assert.equal(app.$('wire-handoff-help').hidden,true);
  assert.equal(help.hidden,false);assert.ok(help.textContent.includes(hints[language]));
  assert.match(help.textContent,/‹/);assert.match(help.textContent,/✕/);assert.ok(!/Safari|Nimiq Pay/.test(help.textContent));
  assert.equal(help.previousElementSibling.id,'wire-handoff-state');
  const originalClose=app.window.close;let closes=0;app.window.close=()=>{closes++;};const before=app.calls.length;
  const location=app.window.location.href,opened=app.opened.length;
  await app.clock.advance(300);assert.equal(closes,0,'no close without a click');
  button.click();assert.equal(closes,1);button.click();assert.equal(closes,1,'ignore repeated click while closing');
  await app.clock.advance(300);await settle();
  assert.equal(help.hidden,false);assert.equal(button.disabled,false);
  assert.equal(app.calls.length,before);assert.equal(app.window.location.href,location);assert.equal(app.opened.length,opened);
  assert.equal(app.doc.querySelector('.wire-nim-window'),null,'Done never reopens the cancelled card');
  app.window.close=originalClose;
 }
});

test('limited cards that are not cancelled keep View the card and do not attempt to close the host',async()=>{
 for(const nimStage of ['ready','funding_started','sent']){
  const app=await start({session:{verified:false,email:'',handoff:{payment_id:'a'.repeat(48)}},nimStage,clocked:true,nativeMode:'error'});
  app.doc.querySelector('.wire-window-back').click();await settle();
  const originalClose=app.window.close;let closes=0;app.window.close=()=>{closes++;};
  assert.equal(app.$('wire-handoff-open').textContent,'View the card');
  assert.equal(app.$('wire-handoff-exit-help').hidden,true);assert.equal(app.$('wire-handoff-help').hidden,false);
  app.$('wire-handoff-open').click();await settle();assert.equal(closes,0);
  assert.ok(app.doc.querySelector('.wire-nim-window'));
  app.window.close=originalClose;
 }
});

test('a reloaded cancelled limited card also offers Done without reopening it or making requests',async()=>{
 const app=await start({session:{verified:false,email:'',handoff:{payment_id:'a'.repeat(48)}},nimStage:'cancelled',clocked:true,nativeMode:'error'});
 assert.equal(app.doc.querySelector('.wire-nim-window'),null);
 assert.equal(app.$('wire-handoff-open').textContent,'Done');
 const originalClose=app.window.close;let closes=0;app.window.close=()=>{closes++;throw Error('not permitted');};
 const before=app.calls.length;app.$('wire-handoff-open').click();await app.clock.advance(300);await settle();
 assert.equal(closes,1);assert.equal(app.$('wire-handoff-exit-help').hidden,false);assert.equal(app.calls.length,before);
 app.window.close=originalClose;
});


test('limited failure has retry and cancel, with Back and X replacing the redundant Close in five languages',async()=>{
 for(const language of ['fr','en','es','de','zh']){
  for(const selector of ['.wire-window-back','.wire-nim-close']){
   const app=await failedNative({redeemDraft:{...handoffDraft,language}});
   assert.equal(app.$('nim-progress-close').hidden,true);
   assert.equal(app.$('nim-wallet-retry').hidden,false);
   assert.equal(app.$('nim-wallet-cancel').hidden,false);
   const back=app.doc.querySelector(selector);assert.ok(back);assert.equal(back.hidden,false);
   const before=app.calls.length;back.click();await settle();
   assert.equal(app.doc.querySelector('.wire-nim-window'),null);
   assert.equal(app.$('wire-handoff-home').hidden,false);
   assert.equal(app.calls.length,before,'closing only dismisses the payment window');
  }
 }
 const full=await start({nimOnly:true,nativeMode:'error',clocked:true});await submitNim(full);
 assert.equal(full.$('nim-progress-close').hidden,false,'full sender session retains Close');
 full.$('nim-progress-close').click();await settle();assert.equal(full.doc.querySelector('.wire-nim-window'),null);
});


test('limited cancellation has one terminal screen and cannot reopen its payment window in all languages',async()=>{
 for(const language of ['fr','en','es','de','zh']){
  const app=await failedNative({redeemDraft:{...handoffDraft,language}});
  app.$('nim-wallet-cancel').click();await settle();
  assert.equal(app.doc.querySelector('.wire-nim-window'),null);
  assert.equal(app.doc.querySelector('#nim-progress'),null);
  assert.equal(app.$('wire-handoff-exit-help').hidden,false);
  const text=app.$('wire-handoff-state').textContent;
  assert.equal(app.doc.body.textContent.split(text).length-1,1);
  app.window.dispatchEvent(new app.window.Event('pageshow'));await settle();
  assert.equal(app.doc.querySelector('.wire-nim-window'),null);
 }
});

test('foreground refresh closes stale wallet choice and payment view and updates sender history',async()=>{
 for(const event of ['visibilitychange','pageshow'])for(const stage of ['cancelled','funded','sent']){
  const app=await start({nimOnly:true,clocked:true});await submitNim(app);
  assert.equal(app.$('wire-nim-choice').hidden,false);
  app.$('wire-choice-open').click();await settle();
  app.setStage(stage);const before=app.calls.length;
  if(event==='visibilitychange'){app.clock.hidden(true);app.clock.hidden(false);}else app.window.dispatchEvent(new app.window.Event('pageshow'));
  await settle();
  assert.ok(app.calls.slice(before).some(([p,b])=>p==='/api/payments'&&!b));
  assert.equal(app.$('wire-nim-choice').hidden,true);
  assert.equal(app.doc.querySelector('.wire-nim-window'),null);
  if(stage==='cancelled')assert.equal(app.$('wire-cards-list').children.length,0);
 }
});


test('funding_not_started is quiet while genuine server failures still show verification unavailable',async()=>{
 for(const language of ['fr','en','es','de','zh']){
  const app=await start({redeemDraft:{...handoffDraft,language},redeemStage:'funding_started',nativeMode:'error',clocked:true,url:'https://wire.phoenixmail.ca/#h='+handoffToken,confirmCode:'funding_not_started'});
  await app.clock.advance(4000);await settle();
  assert.ok(!/Unable to check|Impossible de vérifier|No se puede comprobar|nicht geprüft|无法/.test(app.$('nim-progress-message').textContent));
 }
 const app=await start({nimStage:'funding_started',nimOnly:true,nativeMode:'error',clocked:true,confirmCode:'chain_unavailable',confirmStatus:503});
 app.$('pay').click();await settle();await app.clock.advance(4000);await settle();
 assert.match(app.$('nim-progress-message').textContent,/The service is not responding/);
});
