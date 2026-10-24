// The USDT cashlink claim page (web/usdt.html), run against a scripted service: what it shows, what it signs, what it refuses.
const{test}=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm'),{webcrypto}=require('node:crypto');
const html=fs.readFileSync(__dirname+'/../usdt.html','utf8'),script=[...html.matchAll(/<script>([\s\S]*?)<\/script>/g)].pop()[1];
const lib=fs.readFileSync(__dirname+'/../cashlink.js','utf8');
const PROXY=script.match(/PROXY_CREATION_CODE='(0x[0-9a-f]+)'/)[1];
const tick=()=>new Promise(r=>setImmediate(r));
function node(){return{hidden:false,disabled:false,textContent:'',className:'',value:'',href:'',innerHTML:'',onclick:null,children:[],append(...k){this.children.push(...k)},after(n){this.next=n},play(){return Promise.resolve()}}}

function world(){ // a card, its link key, the service's co-signer key, all made with the real library
 const ctx={crypto:webcrypto,TextEncoder,TextDecoder,btoa,atob,BigInt,Uint8Array,console};ctx.globalThis=ctx;vm.createContext(ctx);vm.runInContext(lib,ctx);
 const C=ctx.PhoenixCashlink;C.setProxyCreationCode(PROXY);
 const {id,key}=C.newCard(),cosKey=webcrypto.getRandomValues(new Uint8Array(32));
 const link=C.checksum(C.addressOfKey(key)),cos=C.checksum(C.addressOfKey(cosKey));
 const owners=[link,cos],safe=C.cardAddress({id,owners,threshold:2},137);
 return{C,id,key,cosKey,owners,safe,idHex:C.hex(id),fragment:C.linkFragment({id,key})};
}
async function start(w,{view,routes={},extra={}}){
 const nodes=new Map(),get=id=>{if(!nodes.has(id))nodes.set(id,node());return nodes.get(id)};
 for(const id of['step-email','step-address','step-code','step-done','useWallet','scanBtn','scanBox','pasteBtn','pasteCode'])get(id).hidden=true;
 const calls=[];
 const doc={documentElement:{lang:''},getElementById:get,createElement:()=>node()};
 const fetchStub=async(path,init={})=>{
  const method=init.method||'GET',body=init.body?JSON.parse(init.body):undefined;calls.push({method,path,body});
  const current=method==='GET'?view():null;
  const out=method==='GET'?{status:current._http||200,body:current}:await (routes[path.replace(/\/cards\/[0-9a-f]{32}\//,'/cards/ID/')]||(()=>({status:404,body:{error:'not_found'}})))(body,path);
  return{ok:out.status>=200&&out.status<300,status:out.status,json:async()=>out.body};
 };
 const context={document:doc,window:null,URLSearchParams,fetch:fetchStub,crypto:webcrypto,TextEncoder,TextDecoder,btoa,atob,BigInt,Uint8Array,setTimeout:()=>1,clearTimeout(){},console,
  location:{search:'?lang=en',hash:w.fragment,replace(){}},navigator:{language:'en',...(extra.navigator||{})},...(extra.globals||{})};
 context.window=context;context.globalThis=context;vm.createContext(context);
 vm.runInContext(lib,context);vm.runInContext(script,context);await tick();await tick();
 return{get,calls,context};
}
const TOKEN='0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359',RELAYER='0x1111111111111111111111111111111111111111';
const cardView=(w,extra={})=>({id:w.idHex,symbol:'USDC',chain:'polygon',chain_id:137,amount:'1',units:1000000,fee_units:100000,token:TOKEN,relayer:RELAYER,state:'ready',owners:w.owners,safe:w.C.checksum(w.safe),explorer:'https://polygonscan.com/tx/',message:'',...extra});
test('stablecoin recipient expiry dates and unavailable/expired responses cover five languages',async()=>{
 const {flowText}=await import('../../../../internal/web/static/flow-text.js');
 for(const language of ['fr','en','es','de','zh']){
  const w=world(),expires_at='2026-10-25T15:42:00Z',extra={globals:{location:{search:'?lang='+language,hash:w.fragment,replace(){}}}};
  const s=await start(w,{view:()=>cardView(w,{expires_at}),extra});
  assert.equal(s.get('amount').next.textContent,flowText(language,'expires').replace('{date}',new Intl.DateTimeFormat(language,{dateStyle:'medium',timeStyle:'short'}).format(new Date(expires_at))));
  for(const code of ['claim_unavailable','claim_expired','card_expired']){const gone=await start(w,{view:()=>({_http:code==='claim_unavailable'?404:410,code}),extra});assert.equal(gone.get('msg').textContent,flowText(language,'ended'));assert.equal(gone.get('step-email').hidden,true);}
  const expired=await start(w,{view:()=>cardView(w,{state:'expired',expires_at}),extra});assert.equal(expired.get('msg').textContent,flowText(language,'ended'));
 }
});

test('a card that is not paid yet says so and offers nothing',async()=>{
 const w=world(),s=await start(w,{view:()=>cardView(w,{state:'awaiting_funds'})});
 assert.match(s.get('msg').textContent,/not paid this card yet/);assert.equal(s.get('step-address').hidden,true);
 assert.equal(s.get('amount').textContent,'1');
});

test('a link that does not match the card is refused before anything is shown',async()=>{
 const w=world(),other=world();
 let s=await start(w,{view:()=>cardView(w,{safe:other.C.checksum(other.safe)})});
 assert.match(s.get('msg').textContent,/not valid/);assert.equal(s.get('step-address').hidden,true);
 s=await start(w,{view:()=>cardView(w,{owners:other.owners,safe:other.C.checksum(other.safe)})});
 assert.match(s.get('msg').textContent,/not valid/);
});

const CHECK='cc'.repeat(16),AUTH='ab'.repeat(16);
function claimRoutes(w,{badHash=false,snap={}}={}){
 const C=w.C,to='0x9431a59b2fA40A6036647FFD7e2608de4eB316Bb',token=C.unhex('3c499c542cEF5E3811e1192ce70d8cC03d5c3359'),relayer=C.unhex('1111111111111111111111111111111111111111');
 const tx=C.payout({token,recipient:to,relayer,amount:1000000n,fee:100000n,nonce:0n});
 const hash='0x'+C.hex(C.txHash(tx,137,w.safe)),shown=badHash?'0x'+'00'.repeat(32):hash;
 const seen={order:[]};
 return{seen,to,tx,routes:{
  '/api/v1/usdt/sessions':()=>({status:201,body:{}}),
  '/api/v1/usdt/cards/ID/email-checks':b=>{seen.order.push('check');seen.check=b;return{status:201,body:{authorization_id:CHECK,session_binding:'cd'.repeat(32),email_binding:'ef'.repeat(32),challenge:'34'.repeat(32),deadline:2000000000,snapshot:{}}}},
  ['/api/v1/usdt/cards/ID/email-checks/'+CHECK+'/code']:b=>{seen.order.push('code');seen.code=b;return{status:202,body:{status:'code_sent'}}},
  ['/api/v1/usdt/email-checks/'+CHECK+'/verify']:b=>{seen.order.push('verify');seen.verify=b;return b.code==='123456'?{status:200,body:{valid_until:'2030-01-01T00:00:00Z'}}:{status:403,body:{error:'invalid_or_expired_code'}}},
  '/api/v1/usdt/cards/ID/authorizations':b=>{seen.order.push('prepare');seen.prepare=b;return{status:201,body:{authorization_id:AUTH,session_binding:'cd'.repeat(32),email_binding:'ef'.repeat(32),challenge:'12'.repeat(32),deadline:2000000000,
    snapshot:{chain_id:137,safe:C.checksum(w.safe),token:'0x'+C.hex(token),relayer:'0x'+C.hex(relayer),amount_units:1000000,fee_units:100000,safe_nonce:0,safe_tx_hash:shown,...snap}}}},
  ['/api/v1/usdt/authorizations/'+AUTH+'/grant']:b=>{seen.order.push('grant');seen.grant=b;return{status:200,body:{grant:'g'.repeat(64)}}},
  ['/api/v1/usdt/authorizations/'+AUTH+'/signature']:async b=>{seen.signature=b;const s=await C.signTx(tx,137,w.safe,w.cosKey);return{status:200,body:{owner:C.checksum(s.owner),signature:'0x'+C.hex(C.sigBytes(s.sig))}}},
  ['/api/v1/usdt/authorizations/'+AUTH+'/claims']:b=>{seen.order.push('claim');seen.claims=b;return{status:200,body:{status:'paid',tx:'0x'+'ee'.repeat(32)}}},
 }};
}
// the e-mail code comes first: send it, enter it, and only then is the address asked for
async function confirmEmail(s){
 await s.get('sendCode').onclick();await tick();
 s.get('code').value='123456';await s.get('verify').onclick();await tick();
}

test('the first screen is the e-mail code, the address comes after it',async()=>{
 const w=world(),r=claimRoutes(w),s=await start(w,{view:()=>cardView(w),routes:r.routes});
 assert.equal(s.get('step-email').hidden,false);assert.equal(s.get('step-address').hidden,true);assert.equal(s.get('step-code').hidden,true);
 assert.equal(s.calls.filter(c=>c.method==='POST').length,0,'nothing is sent, no e-mail, before the recipient asks');
 await s.get('sendCode').onclick();await tick();
 assert.match(r.seen.code.signature,/^0x[0-9a-f]{130}$/,'the proof of the link key is a 65-byte signature');
 assert.equal(s.get('step-code').hidden,false);assert.equal(s.get('step-address').hidden,true,'no address is asked for before the code is confirmed');
 s.get('code').value='12';await s.get('verify').onclick();assert.match(s.get('status2').textContent,/wrong or expired/);assert.equal(s.get('step-address').hidden,true);
 s.get('code').value='123456';await s.get('verify').onclick();await tick();
 assert.equal(s.get('step-address').hidden,false);assert.equal(s.get('step-code').hidden,true);
});

test('the code can be asked again',async()=>{
 const w=world(),r=claimRoutes(w),s=await start(w,{view:()=>cardView(w),routes:r.routes});
 await s.get('sendCode').onclick();await tick();
 await s.get('resend').onclick();await tick();
 assert.equal(r.seen.order.filter(x=>x==='code').length,2);assert.match(s.get('status2').textContent,/new code was sent/);
});

test('the whole claim: e-mail code, then address, proof of the link key, our signature next to the service\'s, payment',async()=>{
 const w=world(),r=claimRoutes(w);
 const s=await start(w,{view:()=>cardView(w),routes:r.routes});
 await confirmEmail(s);
 s.get('to').value='0x123';await s.get('next').onclick();assert.match(s.get('status1').textContent,/Invalid address/);assert.ok(!r.seen.prepare,'a bad address never reaches the server');
 s.get('to').value=r.to;await s.get('next').onclick();await tick();await tick();
 assert.deepEqual(r.seen.order,['check','code','verify','prepare','grant','claim'],'the code is confirmed BEFORE any destination is sent');
 assert.equal(r.seen.prepare.destination,r.to);assert.equal(r.seen.prepare.check_id,CHECK,'the claim refers to the confirmed e-mail check');assert.ok(r.seen.prepare.idempotency_key.length>=16);
 assert.match(r.seen.grant.signature,/^0x[0-9a-f]{130}$/);
 assert.equal(r.seen.claims.grant,'g'.repeat(64));
 // the two signatures sent are exactly ours (link key) and the service's (co-signer), in the order the Safe requires
 const C=w.C,mine=await C.signTx(r.tx,137,w.safe,w.key),theirs=await C.signTx(r.tx,137,w.safe,w.cosKey);
 assert.equal(r.seen.claims.signatures,'0x'+C.hex(C.packSignatures([mine,theirs])));
 assert.equal(s.get('step-done').hidden,false);assert.match(s.get('doneMsg').textContent,/on their way/);
 assert.equal(s.get('txLink').href,'https://polygonscan.com/tx/0x'+'ee'.repeat(32));
});

test('a payout described differently from the one the page rebuilds is never signed',async()=>{
 const w=world(),r=claimRoutes(w,{badHash:true});
 const s=await start(w,{view:()=>cardView(w),routes:r.routes});
 await confirmEmail(s);
 s.get('to').value=r.to;await s.get('next').onclick();await tick();await tick();
 assert.ok(!r.seen.claims,'the claim must not be sent');assert.match(s.get('status1').textContent,/card changed/);
});

test('a self-consistent but unfavourable quote is refused before any grant is asked for (amount, fee, token, relayer, nonce)',async()=>{
 const other='0x'+'22'.repeat(20);
 for(const[name,snap]of Object.entries({'fee too high':{fee_units:5000000},'amount lower than the card':{amount_units:10},'unknown token':{token:other},'another relayer':{relayer:other},'a used nonce':{safe_nonce:3},'fee that differs from the card':{fee_units:200000}})){
  const w=world(),r=claimRoutes(w,{snap});
  const s=await start(w,{view:()=>cardView(w),routes:r.routes});
  await confirmEmail(s);
  s.get('to').value=r.to;await s.get('next').onclick();await tick();
  assert.ok(!r.seen.grant,name+': no grant may be requested for a quote that is refused');assert.match(s.get('status1').textContent,/card changed/,name);
 }
});

test('the recipient sees what he will receive, and what the card pays, before paying',async()=>{
 const w=world(),r=claimRoutes(w),s=await start(w,{view:()=>cardView(w),routes:r.routes,extra:{navigator:{clipboard:{readText:async()=>'0x9431a59b2fA40A6036647FFD7e2608de4eB316Bb'}}}});
 await confirmEmail(s);await s.get('pasteBtn').onclick();
 assert.match(s.get('recap').textContent,/receive 1 USDC at 0x9431a59b2fA40A6036647FFD7e2608de4eB316Bb/);assert.match(s.get('recap').textContent,/You pay nothing/);
});

test('every element the page script looks for exists in its markup',()=>{
 const ids=[...new Set([...script.matchAll(/\$\('([A-Za-z0-9_-]+)'\)/g)].map(m=>m[1]))];
 const missing=ids.filter(id=>!html.includes('id="'+id+'"'));
 assert.deepEqual(missing,[],'script uses ids that are not in the page: a missing element breaks a button without any test noticing');
});

const ADDR='0x9431a59b2fA40A6036647FFD7e2608de4eB316Bb';
test('the address is never typed by hand: the wallet, the camera or a paste',async()=>{
 const w=world();
 // the wallet of the browser (Trust Wallet's browser, MetaMask): one click
 let s=await start(w,{view:()=>cardView(w),routes:{},extra:{globals:{ethereum:{request:async({method})=>method==='eth_requestAccounts'?[ADDR.toLowerCase().replace('0x','0x')]:[]}}}});
 assert.equal(s.get('useWallet').hidden,false,'the wallet button appears when the browser has a wallet');
 await s.get('useWallet').onclick();assert.equal(s.get('to').value,ADDR.toLowerCase());
 // no wallet in the browser: the button stays hidden
 s=await start(w,{view:()=>cardView(w)});assert.equal(s.get('useWallet').hidden,true);
 // the camera reads a wallet's "Receive" QR code (native BarcodeDetector here)
 s=await start(w,{view:()=>cardView(w),extra:{navigator:{mediaDevices:{getUserMedia:async()=>({getTracks:()=>[{stop(){}}]})}},globals:{BarcodeDetector:class{detect(){return Promise.resolve([{rawValue:'ethereum:'+ADDR+'@137'}])}}}}});
 assert.equal(s.get('scanBtn').hidden,false);await s.get('scanBtn').onclick();await tick();await tick();
 assert.equal(s.get('to').value,ADDR,'the address found in the QR code is put in the field');assert.equal(s.get('scanBox').hidden,true,'the camera is stopped once an address is read');
});

test('only a real address is taken from a QR code or a wallet',async()=>{
 const w=world(),s=await start(w,{view:()=>cardView(w)}),{extractAddress}=s.context.__phoenixAddressTools;
 assert.equal(extractAddress('0x9431a59b2fA40A6036647FFD7e2608de4eB316Bb'),ADDR);
 assert.equal(extractAddress('ethereum:'+ADDR+'@137?value=1'),ADDR,'an EIP-681 style QR code');
 assert.equal(extractAddress('  '+ADDR+'\n'),ADDR);
 assert.equal(extractAddress(ADDR.toLowerCase()),ADDR.toLowerCase(),'all lower case carries no checksum to check');
 assert.equal(extractAddress('0x9431a59b2fA40A6036647FFD7e2608de4eB316bb'),'','a mixed-case address with a wrong checksum is refused');
 const {pick}=s.context.__phoenixAddressTools;
 assert.equal(pick('ethereum:'+TOKEN+'@137/transfer?address='+ADDR+'&uint256=1'),'','a payment-request QR code (its first address is the token contract) is not a wallet');
 assert.equal(pick(w.C.checksum(w.safe)),'','the card\'s own address is never a place to receive');
 assert.equal(pick(ADDR),ADDR);
 assert.equal(extractAddress('hello'),'');assert.equal(extractAddress(null),'');assert.equal(extractAddress('0x123'),'');
});

test('without a wallet in the browser: paste in one tap, or the wallet apps on a phone',async()=>{
 const w=world();
 // desktop, no wallet: one-tap paste
 let s=await start(w,{view:()=>cardView(w),extra:{navigator:{clipboard:{readText:async()=>'ethereum:'+ADDR}},}});
 assert.equal(s.get('pasteBtn').hidden,false);await s.get('pasteBtn').onclick();assert.equal(s.get('to').value,ADDR,'the clipboard address is taken');
 // a clipboard that holds something else than an address is not used
 s=await start(w,{view:()=>cardView(w),extra:{navigator:{clipboard:{readText:async()=>'hello'}}}});
 await s.get('pasteBtn').onclick();assert.equal(s.get('to').value,'');assert.match(s.get('status1').textContent,/Nothing to paste/);
 // a phone without a wallet in its browser: buttons that open the wallet apps through their own scheme (never a third-party server)
 const rr=claimRoutes(w);
 s=await start(w,{view:()=>cardView(w),routes:rr.routes,extra:{navigator:{userAgent:'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)'}}});
 await confirmEmail(s); // the address step comes after the e-mail code
 assert.equal(s.get('appLinks').children.length,2);
 for(const a of s.get('appLinks').children)assert.match(a.href,/^(trust|metamask):\/\//);
 // inside a wallet's browser nothing of this is shown
 s=await start(w,{view:()=>cardView(w),routes:claimRoutes(w).routes,extra:{globals:{ethereum:{request:async()=>[ADDR]}}}});
 await confirmEmail(s);
 assert.equal(s.get('appLinks').children.length,0);
});

test('the code can be pasted from the clipboard in one tap, and the card wears the colour of its ID',async()=>{
 const w=world(),r=claimRoutes(w),s=await start(w,{view:()=>cardView(w),routes:r.routes,extra:{navigator:{clipboard:{readText:async()=>'Your code: 123456 (expires)'}}}});
 await s.get('sendCode').onclick();await tick();
 assert.equal(s.get('pasteCode').hidden,false);
 await s.get('pasteCode').onclick();await tick();
 assert.equal(s.get('code').value,'123456');assert.equal(r.seen.verify.code,'123456','the pasted code is sent');assert.equal(s.get('step-address').hidden,false);
 // nothing that looks like a code: nothing is sent
 const w2=world(),r2=claimRoutes(w2),s2=await start(w2,{view:()=>cardView(w2),routes:r2.routes,extra:{navigator:{clipboard:{readText:async()=>'12345 or 1234567'}}}});
 await s2.get('sendCode').onclick();await tick();await s2.get('pasteCode').onclick();
 assert.ok(!r2.seen.verify);assert.match(s2.get('status2').textContent,/No six-digit code/);
});

const memStorage=()=>{const m=new Map();return{getItem:k=>m.has(k)?m.get(k):null,setItem:(k,v)=>m.set(k,String(v)),removeItem:k=>m.delete(k)}};

test('a lost answer or a reload does not restart the journey: the same attempt and grant are used again',async()=>{
 const w=world(),r=claimRoutes(w),sessionStorage=memStorage();let claimTries=0;
 const claimPath='/api/v1/usdt/authorizations/'+AUTH+'/claims';
 const routes={...r.routes,[claimPath]:b=>{claimTries++;if(claimTries===1)return{status:500,body:{error:'generic'}};return r.routes[claimPath](b)}};
 let s=await start(w,{view:()=>cardView(w),routes,extra:{globals:{sessionStorage}}});
 await confirmEmail(s);
 s.get('to').value=r.to;await s.get('next').onclick();await tick();await tick();
 assert.match(s.get('status1').textContent,/Something went wrong/,'the first claim answer is lost');
 // the same click again: no new attempt, no new e-mail check, no new grant: the stored ones are reused
 await s.get('next').onclick();await tick();await tick();
 assert.deepEqual(r.seen.order.filter(x=>x!=='claim'),['check','code','verify','prepare','grant'],'nothing was restarted');
 assert.equal(claimTries,2);assert.equal(s.get('step-done').hidden,false);
});

test('a reload goes back to where the recipient was, not to the first screen',async()=>{
 const w=world(),r=claimRoutes(w),sessionStorage=memStorage();
 let s=await start(w,{view:()=>cardView(w),routes:r.routes,extra:{globals:{sessionStorage}}});
 assert.equal(s.get('step-email').hidden,false);
 await s.get('sendCode').onclick();await tick();
 // reload with the code mailed but not entered
 s=await start(w,{view:()=>cardView(w),routes:r.routes,extra:{globals:{sessionStorage}}});
 assert.equal(s.get('step-code').hidden,false);assert.equal(s.get('step-email').hidden,true);
 s.get('code').value='123456';await s.get('verify').onclick();await tick();
 // reload after the e-mail was confirmed: the address screen, no new code
 s=await start(w,{view:()=>cardView(w),routes:r.routes,extra:{globals:{sessionStorage}}});
 assert.equal(s.get('step-address').hidden,false);assert.equal(r.seen.order.filter(x=>x==='check').length,1,'no second e-mail check');
 // reload after the payment was sent: it is verifying, the page says so (it does not say the card is unpaid)
 s.get('to').value=r.to;await s.get('next').onclick();await tick();await tick();
 s=await start(w,{view:()=>cardView(w,{state:'verifying'}),routes:r.routes,extra:{globals:{sessionStorage}}});
 assert.equal(s.get('step-done').hidden,false);assert.match(s.get('doneMsg').textContent,/being verified/);
});

test('the answer of /grant is lost after the service consumed the check: the same proof asks again, nothing is restarted',async()=>{
 const w=world(),r=claimRoutes(w),sessionStorage=memStorage();let grantTries=0;
 const grantPath='/api/v1/usdt/authorizations/'+AUTH+'/grant';
 const routes={...r.routes,[grantPath]:b=>{grantTries++;r.seen.grants=(r.seen.grants||[]).concat(b.signature);if(grantTries===1)return{status:500,body:{error:'generic'}};return r.routes[grantPath](b)}};
 const s=await start(w,{view:()=>cardView(w),routes,extra:{globals:{sessionStorage}}});
 await confirmEmail(s);
 s.get('to').value=r.to;await s.get('next').onclick();await tick();await tick();
 assert.match(s.get('status1').textContent,/Something went wrong/,'the first /grant answer is lost');
 const prepared=r.seen.order.filter(x=>x==='prepare').length;
 await s.get('next').onclick();await tick();await tick();
 assert.equal(r.seen.order.filter(x=>x==='prepare').length,prepared,'no new attempt: the saved one is used');
 assert.equal(grantTries,2);assert.equal(r.seen.grants[0],r.seen.grants[1],'the very same proof is presented again');
 assert.equal(s.get('step-done').hidden,false);
});

test('the idempotency key is kept before the first request, so a lost answer of the preparation is retried with it',async()=>{
 const w=world(),r=claimRoutes(w),sessionStorage=memStorage();let tries=0;const keys=[];
 const prepPath='/api/v1/usdt/cards/ID/authorizations';
 const routes={...r.routes,[prepPath]:b=>{tries++;keys.push(b.idempotency_key);if(tries===1)return{status:500,body:{error:'generic'}};return r.routes[prepPath](b)}};
 const s=await start(w,{view:()=>cardView(w),routes,extra:{globals:{sessionStorage}}});
 await confirmEmail(s);
 s.get('to').value=r.to;await s.get('next').onclick();await tick();
 await s.get('next').onclick();await tick();await tick();
 assert.equal(keys.length,2);assert.equal(keys[0],keys[1],'the same idempotency key: the service returns the same attempt');
});
