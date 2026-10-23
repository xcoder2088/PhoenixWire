import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import { JSDOM } from 'jsdom';
import { flowText, flowLanguages } from '../internal/web/static/flow-text.js';
import { addressValid,cardPresentation,paintCard } from '../internal/web/static/model.js';

const id='a'.repeat(48);
const address='NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY';
const tick=()=>new Promise(resolve=>setImmediate(resolve));
async function settle(){for(let i=0;i<12;i++)await tick();}
async function start({claim=false,verified=true,pending=false,mobile=false,returning=false,expired=false}={}){
 const dom=new JSDOM(fs.readFileSync(new URL('../internal/web/static/classic.html',import.meta.url),'utf8'),{url:'https://wire.example.test/'+(claim?'?claim='+id:''),runScripts:'outside-only'});
 const {window}=dom;window.document.documentElement.lang='en';
 if(returning){window.sessionStorage.setItem('wire-pending-payment',id);window.sessionStorage.setItem('wire-form',JSON.stringify({payment:id,message:'Hi',amount:'2.5',recipient:'recipient@example.test'}));}
 if(mobile)Object.defineProperty(window.navigator,'userAgent',{value:'iPhone'});
 const calls=[],checkouts=[],behaviors=[],opened=[],referrers=[];
 let validOTP=false,deliveryVerified=verified;
 const p={id,sender:'sender@example.test',recipient:'recipient@example.test',amount:'2.5',luna:250000,message:'Hi',address,stage:pending?'funding_started':'ready'};
 let payments=pending?[p]:[];
 const response=(status,body)=>({ok:status>=200&&status<300,status,json:async()=>body});
 window.fetch=async(path,options)=>{
  calls.push([path,options]);
  if(path==='/api/config')return response(200,{mode:'integrated',payments_enabled:true});
  if(path==='/api/session')return response(200,{verified:deliveryVerified,email:deliveryVerified?p.sender:''});
  if(path==='/api/payments'&&!options.method)return response(200,{payments});
  if(path==='/api/payments'){payments=[p];return response(201,{payment:p});}
  if(path.endsWith('/start')){p.stage='funding_started';return response(200,{payment:p,checkout:{appName:'Phoenix Wire',recipient:address,value:250000,recipientData:[0,130,128,146,135]}});}
  if(path.endsWith('/confirm')){p.stage='sent';return response(200,{payment:p});}
  if(path.endsWith('/art'))return response(200,{saved:true});
  if(path==='/api/sender/request')return response(200,{sent:true});
  if(path==='/api/sender/verify'){deliveryVerified=true;return response(200,{email:p.sender,verified:true});}
  if(path==='/api/claims/'+id)return response(200,{...p,expires_at:new Date(Date.now()+(expired?-3600000:3600000)).toISOString()});
  if(path.endsWith('/payout_status'))return response(404,{code:'payout_not_started'});
  if(path.endsWith('/request'))return response(200,{sent:true});
  if(path.endsWith('/release'))return validOTP?response(200,{cashlink:'https://hub.nimiq.com/cashlink/#'+Buffer.alloc(40,5).toString('base64url')}):response(403,{error:'invalid_or_expired_code'});
  throw Error('unexpected API '+path);
 };
 const popup={closed:false,close(){this.closed=true;}};
 window.open=(url)=>{opened.push(url);return popup;};
 class Hub{
  constructor(endpoint,behavior){assert.equal(endpoint,'https://hub.nimiq.com');this.behavior=behavior;}
  async checkout(request){referrers.push(Array.from(window.document.querySelectorAll('meta[name=referrer]')).at(-1)?.content);checkouts.push(request);return{hash:'f'.repeat(64)};}
  on(type,success){this.callback=success;}
  async checkRedirectResponse(){if(returning)await this.callback();}
 }
 Hub.PopupRequestBehavior=class{constructor(){}};
 Hub.RequestType={CHECKOUT:'checkout'};
 Hub.RedirectRequestBehavior=class{constructor(url,state){behaviors.push({url,state});}};
 window.HubApi=Hub;
 window.PhoenixNimiqIdenticons={async png(){return 'fixture-artwork';},paint(image){image.hidden=false;image.src='data:image/png;base64,test';}};
 Object.assign(dom.getInternalVMContext(),{flowText,addressValid,cardPresentation,paintCard});
 const source=fs.readFileSync(new URL('../internal/web/static/flow.js',import.meta.url),'utf8').replace(/^import[^\n]+\n/gm,'').replace(/^export /gm,'');
 vm.runInContext(source,dom.getInternalVMContext());await settle();
 const $=id=>window.document.getElementById(id);
 const submit=async()=>{$('card-form').dispatchEvent(new window.Event('submit',{cancelable:true}));await settle();};
 return{dom,window,$,calls,checkouts,behaviors,opened,referrers,submit,allowOTP(){validOTP=true;}};
}

test('real DOM sender verification and checkout use the same immutable public address',async()=>{
 const s=await start({verified:false});
 assert.equal(s.$('sender-auth').hidden,false);
 s.$('sender').value='sender@example.test';s.$('send-sender-code').click();await settle();
 assert.equal(s.$('sender-code-box').hidden,false);
 s.$('sender-code').value='123456';s.$('verify-sender-code').click();await settle();
 assert.equal(s.$('sender').readOnly,true);
 s.$('recipient').value='recipient@example.test';s.$('amount').value='2.5';
 await s.submit();assert.equal(s.checkouts.length,1);
 assert.equal(s.checkouts[0].recipient,address);
 assert.equal(s.checkouts[0].value,250000);
 assert.deepEqual(Array.from(s.checkouts[0].extraData),[0,130,128,146,135]);
 assert.equal(s.checkouts[0].seed,undefined);
 assert.equal(s.$('feedback').textContent,'Card sent to recipient@example.test. It stays yours until it is claimed.');
 assert.equal(s.$('recipient').readOnly,true);
 assert.equal(s.$('coin').querySelector('img').alt,'Nimiq Nimicon');
 await s.submit();assert.equal(s.checkouts.length,1,'repeated send cannot fund again');
 s.dom.window.close();
});
test('restoring pending checkout only checks funding without opening a second wallet',async()=>{
 const s=await start({pending:true});await s.submit();
 assert.equal(s.checkouts.length,0);
 assert.equal(s.calls.filter(([path])=>path.endsWith('/start')).length,0);
 assert.ok(s.calls.some(([path])=>path.endsWith('/confirm')));
 s.dom.window.close();
});
test('claim reveals no released wallet link before a correct six-digit code',async()=>{
 const s=await start({claim:true});assert.equal(s.$('claim-panel').hidden,false);
 assert.equal(s.$('wallet-link').hidden,true);assert.equal(s.$('claim-wallet').disabled,true);
 for(const value of ['','123','12x456']){s.$('claim-code').value=value;s.$('claim-code').dispatchEvent(new s.window.Event('input'));assert.equal(s.$('claim-wallet').disabled,true);}
 s.$('claim-code').value='123456';s.$('claim-code').dispatchEvent(new s.window.Event('input'));
 s.$('claim-wallet').click();await settle();assert.match(s.$('claim-status').textContent,/Invalid or expired/);
 s.allowOTP();s.$('claim-code').value='123456';s.$('claim-code').dispatchEvent(new s.window.Event('input'));s.$('claim-wallet').click();await settle();
 assert.equal(s.$('wallet-link').hidden,true,'no duplicate action link');
 assert.equal(s.$('claim-wallet').textContent,'Receive in Nimiq Wallet');assert.equal(s.$('claim-wallet').disabled,false);
 assert.equal(s.$('request-claim-code').hidden,true);s.$('claim-wallet').click();
 assert.match(s.opened.at(-1),/^https:\/\/hub.nimiq.com\/cashlink\/#/);s.dom.window.close();
});
test('mobile checkout records its existing payment before redirecting to Hub',async()=>{
 const s=await start({mobile:true});s.$('recipient').value='recipient@example.test';await s.submit();
 assert.equal(s.checkouts.length,1);
 assert.equal(s.behaviors[0].url,'https://wire.example.test/');
 assert.equal(s.behaviors[0].state.wirePayment,id);
 assert.equal(s.window.sessionStorage.getItem('wire-pending-payment'),id);
 assert.equal(JSON.parse(s.window.sessionStorage.getItem('wire-form')).payment,id);
 assert.equal(s.referrers[0],'strict-origin');
 assert.equal(s.window.document.querySelectorAll('meta[name=referrer]').length,1);
 assert.equal(s.window.document.querySelector('meta[name=referrer]').content,'no-referrer');
 s.dom.window.close();
});
test('flow labels and safety feedback exist in every supported language',()=>{
 for(const lang of flowLanguages){for(const key of ['verifyEmail','verifyFirst','pending','recover','unlock','receive','unknownWallet'])assert.notEqual(flowText(lang,key),key);}
 assert.equal(addressValid(address),true);assert.equal(addressValid(address.slice(0,-1)+'0'),false);
});

test('claim code request gives immediate feedback and a 20-second resend countdown',async()=>{
 const s=await start({claim:true});
 assert.match(s.$('claim-status').textContent,/Code sent/);
 assert.match(s.$('request-claim-code').textContent,/20 s/);
 assert.equal(s.$('request-claim-code').disabled,true);
 assert.equal(s.$('request-claim-code').nextElementSibling,s.$('claim-choices'));
 s.dom.window.close();
});
test('iPhone return restores the original form and confirms without a second checkout',async()=>{
 const s=await start({pending:true,mobile:true,returning:true});
 assert.equal(s.$('recipient').value,'recipient@example.test');
 assert.equal(s.$('amount').value,'2.5');assert.equal(s.$('message').value,'Hi');
 assert.equal(s.checkouts.length,0);assert.equal(s.calls.filter(([p])=>p.endsWith('/start')).length,0);
 assert.ok(s.calls.some(([p])=>p.endsWith('/confirm')));
 assert.equal(s.$('feedback').textContent,'Card sent to recipient@example.test. It stays yours until it is claimed.');s.dom.window.close();
});
test('expired claim keeps its explanation visible while disabling code actions',async()=>{
 const s=await start({claim:true,expired:true});
 assert.match(s.$('claim-status').textContent,/period has ended/);
 assert.equal(s.$('claim-form').hidden,false);
 assert.equal(s.$('request-claim-code').hidden,true);
 assert.equal(s.$('claim-choices').hidden,true);s.dom.window.close();
});
