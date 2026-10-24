const{test}=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
const html=fs.readFileSync(__dirname+'/../index.html','utf8'),script=html.match(/<script>([\s\S]*?)<\/script>/)[1];
const tick=()=>new Promise(r=>setImmediate(r));
function node(){return{hidden:false,disabled:false,textContent:'',className:'',value:'',href:'',src:'',alt:'',firstChild:{textContent:''},events:{},attrs:{},focus(){},scrollIntoView(){},setAttribute(k,v){this.attrs[k]=v},removeAttribute(k){delete this.attrs[k]},addEventListener(name,fn){this.events[name]=fn},after(n){this.next=n}}}
const ID='ab'.repeat(24),TO='0x9431a59b2fA40A6036647FFD7e2608de4eB316Bb',TX='0x'+'c'.repeat(64);
// A scripted server: `view` is what GET returns, `routes` answers POST actions.
async function start({lang='en',view,routes={}}){
 const nodes=new Map(),get=id=>{if(!nodes.has(id))nodes.set(id,node());return nodes.get(id)};
 for(const id of ['evm-step','nimicon','evm-tx','evm-confirm'])get(id).hidden=true;
 const doc={title:'',documentElement:{style:{props:{},setProperty(k,v){this.props[k]=v}},lang:''},getElementById:get,querySelector:get,querySelectorAll:()=>Array.from({length:6},node),createElement:()=>node()};
 const calls=[];let current={...view};const timers=[];
 const context={document:doc,URLSearchParams,setInterval:()=>1,clearInterval(){},setTimeout:(f)=>{timers.push(f);return timers.length},clearTimeout(){},location:{search:'?claim='+ID+'&lang='+lang},navigator:{language:lang},window:{open(){},PhoenixNimiqIdenticons:{normalize:x=>x},NimiqIqonsOfficial:{toDataUrl:async()=>'x'}},fetch:async(path,init)=>{
  calls.push([path,init]);
  if(!init?.method)return{ok:true,json:async()=>current};
  const action=path.split('action=')[1];const handler=routes[action];
  const out=handler?handler(JSON.parse(init.body||'{}'),(patch)=>{current={...current,...patch}}):{status:404,body:{error:'unknown_action'}};
  return{ok:out.status===200,json:async()=>out.body};
 }};
 vm.runInNewContext(script,context);await tick();await tick();
 return{get,calls,doc,timers,set:p=>{current={...current,...p}}};
}
const base={id:ID,asset:'usdt_polygon',chain:'polygon',amount:'7.5',symbol:'USDT',palette:1,protected:true,email_verified:false,state:'awaiting_address',has_recipient:false,expires_at:new Date(Date.now()+3600e3).toISOString()};
const addressRoute=(b,set)=>{set({state:'ready',has_recipient:true});return{status:200,body:{...base,state:'ready',has_recipient:true}}};

test('a protected USDT card asks for the emailed code, then the recipient\'s own address, then asks him to confirm it',async()=>{
 const s=await start({view:base,routes:{request:()=>({status:200,body:{status:'sent'}}),release:(b)=>b.code==='123456'?{status:200,body:{status:'otp_verified'}}:{status:403,body:{error:'invalid_or_expired_code'}},address:addressRoute}});
 assert.equal(s.get('evm-step').hidden,true,'the address step stays hidden until the code is verified');
 assert.match(s.doc.title,/Secure USDT Claim/);assert.match(html,/Secure NIM Claim/);
 assert.equal(s.get('cashlink-iqon').src,'/brand/usdt-polygon.png');assert.equal(s.get('nimicon').hidden,false);
 await s.get('send').onclick();
 s.get('otp').value='000000';await s.get('verify').onclick();assert.match(s.get('status').textContent,/incorrect or no longer valid/);assert.equal(s.get('evm-step').hidden,true);
 s.get('otp').value='123456';await s.get('verify').onclick();
 assert.equal(s.get('evm-step').hidden,false);assert.equal(s.get('verify').hidden,true);assert.equal(s.get('send').hidden,true);
 assert.match(s.get('evm-label').textContent,/Polygon/);
 // a mistyped address never reaches the server
 s.get('evm-address').value='0x123';s.get('evm-claim').onclick();assert.match(s.get('status').textContent,/not valid/);assert.equal(s.get('evm-confirm').hidden,true);
 // a well-formed one must be confirmed first
 s.get('evm-address').value=TO;s.get('evm-claim').onclick();
 assert.equal(s.get('evm-confirm').hidden,false);assert.equal(s.get('evm-confirm-address').textContent,TO);assert.equal(s.get('evm-claim').hidden,true);
 assert.match(s.get('evm-confirm-help').textContent,/cannot be changed/);
 assert.equal(s.calls.filter(([p])=>p.endsWith('action=address')).length,0,'nothing is sent before the confirmation');
 s.get('evm-confirm-back').onclick();assert.equal(s.get('evm-confirm').hidden,true);assert.equal(s.get('evm-address').hidden,false);
 s.get('evm-claim').onclick();await s.get('evm-confirm-yes').onclick();await tick();
 const sent=s.calls.filter(([p])=>p.endsWith('action=address'));assert.equal(sent.length,1);assert.deepEqual(JSON.parse(sent[0][1].body),{to:TO});
 assert.match(s.get('status').textContent,/Your sender has been told/);assert.equal(s.get('evm-address').hidden,true);assert.equal(s.get('evm-confirm').hidden,true);
 // polling sees the payment
 await tick();assert.ok(s.timers.length>0,'the page keeps watching while it waits');
 s.set({state:'paid',tx_hash:TX,tx_url:'https://polygonscan.com/tx/'+TX});await s.timers[s.timers.length-1]();await tick();
 assert.match(s.get('status').textContent,/Payment received/);assert.equal(s.get('evm-tx').hidden,false);assert.equal(s.get('evm-tx').href,'https://polygonscan.com/tx/'+TX);
});

test('an open USDT card goes straight to the address step; unknown explorers are never linked',async()=>{
 const s=await start({view:{...base,protected:false,chain:'ethereum',asset:'usdt_ethereum',message:'cadeau'},routes:{address:()=>({status:200,body:{...base,state:'paid',tx_url:'https://evil.example/tx/'+TX}})}});
 assert.equal(s.get('evm-step').hidden,false);assert.match(s.get('evm-label').textContent,/Ethereum/);assert.equal(s.get('wallet-note').textContent,'cadeau');assert.equal(s.get('verify').hidden,true);
 s.get('evm-address').value=TO;s.get('evm-claim').onclick();await s.get('evm-confirm-yes').onclick();await tick();
 assert.equal(s.get('evm-tx').hidden,true,'a link to an unexpected host must never be shown');
});

test('already waiting, paid, cancelled and expired cards show their state and offer nothing to press',async()=>{
 let s=await start({view:{...base,state:'ready',has_recipient:true,email_verified:true}});
 assert.match(s.get('status').textContent,/Your sender has been told/);assert.equal(s.get('evm-claim').hidden,true);assert.equal(s.get('evm-address').hidden,true);assert.ok(s.timers.length>0);
 s=await start({view:{...base,state:'paid',tx_hash:TX,tx_url:'https://polygonscan.com/tx/'+TX}});assert.match(s.get('status').textContent,/Payment received/);assert.equal(s.get('evm-tx').hidden,false);assert.equal(s.timers.length,0,'no more polling once paid');
 s=await start({view:{...base,state:'cancelled'}});assert.match(s.get('status').textContent,/sender cancelled/);
 s=await start({view:{...base,state:'expired'}});assert.match(s.get('status').textContent,/expired/);assert.equal(s.get('evm-claim').hidden,true);
});

test('server errors are shown in plain words, never as codes; a refused address returns to the address step',async()=>{
 for(const[code,re]of[['same_address',/sender’s own address/],['chain_unavailable',/network is not reachable/],['claim_expired',/expired/],['card_cancelled',/cancelled/],['address_already_given',/already given/],['email_not_verified',/Verify your email code/],['invalid_address',/not valid/]]){
  const s=await start({view:{...base,protected:false},routes:{address:()=>({status:409,body:{error:code}})}});
  s.get('evm-address').value=TO;s.get('evm-claim').onclick();await s.get('evm-confirm-yes').onclick();await tick();
  assert.match(s.get('status').textContent,re,code);assert.doesNotMatch(s.get('status').textContent,/_/);
  if(code==='same_address')assert.equal(s.get('evm-address').hidden,false,'he can type another address');
 }
});

test('the page is translated into the five languages',async()=>{
 const expect={fr:/Recevoir des USDT en sécurité/,es:/Recibir USDT de forma segura/,de:/USDT sicher empfangen/,zh:/安全领取 USDT/};
 for(const[lang,re]of Object.entries(expect)){const s=await start({lang,view:{...base,protected:false}});assert.match(s.doc.title,re);assert.match(s.get('evm-claim').textContent,/→/);assert.doesNotMatch(s.get('evm-label').textContent,/\{chain\}/)}
});

test('USDT cards use mint/seafoam on Ethereum and lavender/orchid on Polygon',()=>{
 const f=script.match(/function evmPaletteFor\(v\)\{[^}]*\}/)[0],fn=new Function(f+';return evmPaletteFor')();
 const ids=['00'+'00'+'ab','00'+'01'+'ab'];
 const eth=ids.map(id=>fn({id,chain:'ethereum'})),pol=ids.map(id=>fn({id,chain:'polygon'}));
 assert.deepEqual(new Set(eth),new Set([2,11]));assert.deepEqual(new Set(pol),new Set([1,8]));
});

