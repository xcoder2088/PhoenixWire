const{test}=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
const html=fs.readFileSync(__dirname+'/../index.html','utf8'),script=html.match(/<script>([\s\S]*?)<\/script>/)[1];
const tick=()=>new Promise(r=>setImmediate(r));
function node(){return{hidden:false,disabled:false,textContent:'',value:'',firstChild:{textContent:''},events:{},focus(){},addEventListener(name,fn){this.events[name]=fn},after(n){this.next=n}}}
async function start(){
 const nodes=new Map(),get=id=>{if(!nodes.has(id))nodes.set(id,node());return nodes.get(id)};
 const doc={documentElement:{style:{setProperty(){}}},getElementById:get,querySelector:get,querySelectorAll:()=>Array.from({length:6},node),createElement:()=>node()};
 const calls=[];let valid=false;
 const context={document:doc,URLSearchParams,location:{search:'?claim='+'a'.repeat(48)},navigator:{language:'en'},window:{PhoenixNimiqIdenticons:{normalize:x=>x},NimiqIqonsOfficial:{toDataUrl:async address=>'data:image/svg+xml;base64,official-'+address}},fetch:async(path,init)=>{
  calls.push([path,init]);
  if(!init?.method)return{ok:true,json:async()=>({id:'a'.repeat(48),amount:'1',network:'mainnet',status:'otp_required',address:'NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY'})};
  if(path.endsWith('action=request'))return{ok:true,json:async()=>({status:'sent'})};
  return{ok:valid,json:async()=>valid?{cashlink:'https://hub.nimiq.com/cashlink/#test-only-secret'}:{error:'invalid_or_expired_code'}};
 }};
 vm.runInNewContext(script,context);await tick();await tick();
 return{get,calls,allow:()=>{valid=true}};
}
test('real claim has an official Nimicon and no QR, palette editor or duplicate central Phoenix emblem',async()=>{
 assert.doesNotMatch(html,/id="new-card"|palette-refresh|<img[^>]+(?:QR|qr)|<img[^>]+phoenixmail[^>]+class="center-emblem"/);
 assert.match(html,/\[hidden\]\{display:none!important\}/);
 const s=await start();assert.equal(s.get('nimicon').hidden,false);assert.match(s.get('cashlink-iqon').src,/official-NQ08/);assert.equal(s.get('verify').disabled,true);
 assert.equal(s.get('status').next,undefined);
});
test('opening a claim and rejected OTP never display a redeemable link; only successful release does',async()=>{
 const s=await start();assert.equal(s.get('status').next,undefined);
 await s.get('send').onclick();assert.equal(s.get('status').next,undefined);
 s.get('otp').value='000000';await s.get('verify').onclick();assert.equal(s.get('status').next,undefined);assert.match(s.get('status').textContent,/incorrect or no longer valid/);assert.doesNotMatch(s.get('status').textContent,/invalid_or_expired_code/);
 s.allow();s.get('otp').value='123456';await s.get('verify').onclick();assert.match(s.get('status').next.href,/^https:\/\/hub.nimiq.com\/cashlink\/#/);assert.equal(s.get('verify').disabled,true);
 const release=s.calls.filter(([path])=>path.endsWith('action=release'));assert.equal(release.length,2);assert.deepEqual(JSON.parse(release[1][1].body),{code:'123456'});
});

test('empty and incomplete codes never reach the release API or display an expiry error',async()=>{
 const s=await start();assert.equal(s.get('verify').disabled,true);
 for(const code of ['', '123', '12x456']){
  s.get('otp').value=code;await s.get('verify').onclick();
  assert.equal(s.calls.filter(([path])=>path.endsWith('action=release')).length,0);
  assert.match(s.get('status').textContent,/six-digit/);assert.doesNotMatch(s.get('status').textContent,/expired/);
 }
 s.get('otp').value='123456';s.get('otp').events.input();assert.equal(s.get('verify').disabled,false);
 s.get('otp').value='12345';s.get('otp').events.input();assert.equal(s.get('verify').disabled,true);
 await s.get('send').onclick();assert.match(s.get('status').textContent,/sent to the recipient/);assert.equal(s.get('otp').value,'');assert.equal(s.get('verify').disabled,true);
});
