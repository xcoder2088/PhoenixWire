import { flowText } from './flow-text.js?v=0.2.2';
import { addressValid } from './model.js?v=0.2.2';

// The bundled official Iqons adapter validates public Nimiq addresses through this hook.
window.PhoenixNimiqPayments={addressValid};

const $=id=>document.getElementById(id);
const t=key=>flowText(document.documentElement.lang,key);
let configured=false,ready=false,verified=false,busy=false,claimMode=false,active=null;
const read=key=>{try{return sessionStorage.getItem(key)||'';}catch{return '';}};
const save=(key,value)=>{try{sessionStorage.setItem(key,value);}catch{}};
let requestID=read('wire-request-id');
function nextID(){const bytes=crypto.getRandomValues(new Uint8Array(16));return Array.from(bytes,b=>b.toString(16).padStart(2,'0')).join('');}
if(!/^[a-f0-9]{32}$/.test(requestID)){requestID=nextID();save('wire-request-id',requestID);}
async function api(path,data){
 const options={credentials:'same-origin',cache:'no-store',redirect:'error'};
 if(data!==undefined){options.method='POST';options.headers={'Content-Type':'application/json'};options.body=JSON.stringify(data);}
 const response=await fetch(path,options);const body=await response.json();
 if(!response.ok){const error=Error(body.code||body.error||'unavailable');error.status=response.status;throw error;}return body;
}
function status(id,key){$(id).textContent=t(key);}
function explain(error){if(error.message==='funding_pending'||error.message==='funding_already_started')return t('pending');if(error.status===403)return t('invalid');if(error.status===410)return t('expired');if(error.status===401)return t('verifyFirst');return t('failed');}
function refreshText(){
 document.querySelectorAll('[data-wire]').forEach(el=>el.textContent=t(el.dataset.wire));
 if(configured&&!claimMode){$('preview-submit').textContent=t(active&&active.stage!=='ready'?'retry':'pay');$('notice').textContent=t(ready?'note':'unavailable');document.querySelector('.security').textContent=t('security');document.querySelector('.lock').textContent=t('locked');}
}
window.addEventListener('wire-language',refreshText);
async function task(run,target='feedback',cleanup){
 if(busy)return;busy=true;status(target,'working');
 try{await run();}catch(error){$(target).textContent=explain(error);}finally{busy=false;cleanup?.();}
}
function senderUI(email){verified=Boolean(email);$('sender').value=email||$('sender').value;$('sender').readOnly=verified;$('send-sender-code').hidden=verified;$('logout').hidden=!verified;$('sender-code-box').hidden=true;if(verified)status('sender-status','verified');}
function showPayment(p){
 $('preview-empty').hidden=true;$('payment-card').hidden=false;$('preview-title').hidden=false;
 $('recipient').value=p.recipient;$('amount').value=p.amount;$('message').value=p.message;$('currency').value='NIM';
 for(const id of ['recipient','amount','message'])$(id).readOnly=true;$('currency').disabled=true;
 $('card-amount').textContent=p.amount;$('card-unit').textContent='NIM';$('card-network').textContent='Nimiq';$('card-message').textContent=p.message;$('card-recipient').textContent=p.recipient;
 const image=document.createElement('img');image.width=62;image.height=62;image.alt='Nimiq Nimicon';$('coin').textContent='';$('coin').append(image);window.PhoenixNimiqIdenticons?.paint(image,p.address);
}
async function loadHistory(){
 const data=await api('/api/payments');$('my-payments').hidden=false;$('payment-list').textContent='';
 for(const p of data.payments){
  const row=document.createElement('div');row.className='history-row';
  const label=document.createElement('p');label.textContent=`${p.amount} NIM · ${p.recipient} · ${t(p.stage)}`;row.append(label);
  const refresh=document.createElement('button');refresh.type='button';refresh.textContent=t('retry');refresh.addEventListener('click',()=>task(()=>confirm(p.id)));row.append(refresh);
  if(['sent','delivery_failed','delivery_pending'].includes(p.stage)){const resend=document.createElement('button');resend.type='button';resend.textContent=t('resend');resend.addEventListener('click',()=>task(async()=>{await api(`/api/payments/${p.id}/resend`,{});await loadHistory();status('feedback','sent');}));row.append(resend);}
  if(p.address){const recover=document.createElement('button');recover.type='button';recover.textContent=t('recover');recover.addEventListener('click',()=>task(async()=>{const result=await api(`/api/payments/${p.id}/recovery`,{});const link=document.createElement('a');link.href=validatedCashlink(result.cashlink);link.target='_blank';link.rel='noopener noreferrer';link.textContent=t('recover');recover.replaceWith(link);}));row.append(recover);}
  $('payment-list').append(row);
 }
 active=data.payments.find(p=>!['sent'].includes(p.stage))||null;
 if(active||data.payments[0])showPayment(active||data.payments[0]);
 $('new-card').hidden=Boolean(active);refreshText();
}
function validatedCashlink(value){const url=new URL(value);if(url.origin!=='https://hub.nimiq.com'||url.pathname!=='/cashlink/'||!/^#[A-Za-z0-9_=-]+$/.test(url.hash))throw Error('invalid_cashlink');return url.href;}
async function confirm(id){const result=await api(`/api/payments/${id}/confirm`,{});await loadHistory();status('feedback',result.payment.stage==='sent'?'sent':'pending');}
async function poll(id){for(let i=0;i<20;i++){try{await confirm(id);return;}catch(error){if(error.message!=='funding_pending')throw error;}await new Promise(resolve=>setTimeout(resolve,3000));}status('feedback','pending');await loadHistory();}
async function withReferrer(run){const meta=document.createElement('meta');meta.name='referrer';meta.content='strict-origin';document.head.append(meta);try{return await run();}finally{meta.remove();}}
function mobile(){return /iPhone|iPad|iPod|Android/i.test(navigator.userAgent||'')||(navigator.platform==='MacIntel'&&navigator.maxTouchPoints>1);}
$('send-sender-code').addEventListener('click',()=>task(async()=>{
 if(!$('sender').reportValidity())return;await api('/api/sender/request',{email:$('sender').value});$('sender-code-box').hidden=false;status('sender-status','codeSent');
},'sender-status'));
$('verify-sender-code').addEventListener('click',()=>task(async()=>{
 const result=await api('/api/sender/verify',{code:$('sender-code').value});$('sender-code').value='';senderUI(result.email);await loadHistory();
},'sender-status'));
$('logout').addEventListener('click',()=>task(async()=>{await api('/api/logout',{});location.reload();}));
$('new-card').addEventListener('click',()=>{requestID=nextID();save('wire-request-id',requestID);active=null;for(const id of ['recipient','amount','message'])$(id).readOnly=false;$('currency').disabled=false;$('recipient').value='';$('message').value='';$('feedback').textContent='';});
$('card-form').addEventListener('submit',event=>{
 if(!configured)return;event.preventDefault();
 if(busy)return;
 const openWallet=verified&&ready&&$('currency').value==='NIM'&&(!active||active.stage==='ready')&&!mobile();
 const popup=openWallet?window.open('about:blank','NimiqAccounts','width=800,height=850'):null;
 if(openWallet&&!popup){status('feedback','failed');return;}
 task(async()=>{
  if(!verified){status('feedback','verifyFirst');return;}
  if(!ready||$('currency').value!=='NIM'){status('feedback','unavailable');return;}
  // Restored pending records can only be checked, never automatically funded again.
  if(active&&active.stage==='preparing'){const result=await api(`/api/payments/${active.id}/prepare`,{});active=result.payment;await loadHistory();return;}
  if(active&&active.stage!=='ready'){await confirm(active.id);return;}
  let p=active;
  if(!p){const result=await api('/api/payments',{recipient:$('recipient').value,amount:$('amount').value.trim().replace(',','.'),currency:'NIM',message:$('message').value,request_id:requestID,language:document.documentElement.lang});p=result.payment;active=p;}
  showPayment(p);
  if(p.stage!=='ready'){await confirm(p.id);return;}
  // No unprotected fallback. No seed is present in this checkout request.
  if(!window.HubApi)throw Error('wallet_unavailable');
  const started=await api(`/api/payments/${p.id}/start`,{});active=started.payment;
  // Hub's checkout extraData becomes the chain transaction's recipientData.
  const checkout={appName:started.checkout.appName,recipient:started.checkout.recipient,value:started.checkout.value,extraData:new Uint8Array(started.checkout.recipientData)};
  if(mobile()){
   save('wire-pending-payment',p.id);
   const behavior=new window.HubApi.RedirectRequestBehavior(location.origin+'/',{wirePayment:p.id});
   await withReferrer(()=>new window.HubApi('https://hub.nimiq.com',behavior).checkout(checkout));return;
  }
  try{await withReferrer(()=>new window.HubApi('https://hub.nimiq.com',new window.HubApi.PopupRequestBehavior(undefined,{overlay:false})).checkout(checkout));}catch{await loadHistory();status('feedback','unknownWallet');return;}
  await poll(p.id);
 },'feedback',()=>{if(popup&&!popup.closed)popup.close();});
});
$('claim-code').addEventListener('input',()=>{$('release-claim').disabled=!/^[0-9]{6}$/.test($('claim-code').value);});
$('request-claim-code').addEventListener('click',()=>task(async()=>{await api(`/api/claims/${claimID()}/request`,{});status('claim-status','codeSent');},'claim-status'));
$('claim-form').addEventListener('submit',event=>{event.preventDefault();if(!/^[0-9]{6}$/.test($('claim-code').value))return;task(async()=>{
 const result=await api(`/api/claims/${claimID()}/release`,{code:$('claim-code').value});$('claim-code').value='';$('release-claim').disabled=true;
 $('wallet-link').href=validatedCashlink(result.cashlink);$('wallet-link').hidden=false;$('claim-status').textContent='';
},'claim-status');});
function claimID(){const id=new URLSearchParams(location.search).get('claim');if(!/^[a-f0-9]{48}$/.test(id||''))throw Error('claim_unavailable');return id;}
async function boot(){
 refreshText();claimMode=new URLSearchParams(location.search).has('claim');
 if(claimMode){
  document.querySelector('.workspace').hidden=true;$('notice').hidden=true;$('claim-panel').hidden=false;
  try{const p=await api('/api/claims/'+claimID());$('claim-info').textContent=t('from')+' '+p.sender;$('claim-amount').textContent=p.amount+' NIM';$('claim-message').textContent=p.message;
   window.PhoenixNimiqIdenticons?.paint($('claim-iqon'),p.address);
   if(new Date(p.expires_at).getTime()<=Date.now()){$('claim-form').hidden=true;$('request-claim-code').hidden=true;status('claim-status','expired');}
  }catch(error){$('claim-form').hidden=true;$('request-claim-code').hidden=true;$('claim-status').textContent=explain(error);}return;
 }
 try{const config=await api('/api/config');configured=config.mode==='integrated';ready=config.payments_enabled===true;if(!configured)return;
  $('sender-auth').hidden=false;refreshText();const session=await api('/api/session');senderUI(session.verified?session.email:'');if(verified)await loadHistory();
  const pending=read('wire-pending-payment');
  if(pending&&/^[a-f0-9]{48}$/.test(pending)&&window.HubApi){
   const hub=new window.HubApi('https://hub.nimiq.com',new window.HubApi.RedirectRequestBehavior());
   hub.on(window.HubApi.RequestType.CHECKOUT,()=>task(()=>poll(pending)),()=>status('feedback','unknownWallet'));
   await hub.checkRedirectResponse();
  }
 }catch{status('feedback','unavailable');}
}
void boot();
