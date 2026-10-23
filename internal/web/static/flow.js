import { flowText } from './flow-text.js?v=0.3.0';
import { addressValid,cardPresentation,paintCard } from './model.js?v=0.3.0';

// The same protected NIM flow mounts either on classic.html or inside the authenticated payment window.
export function mountNimFlow(root=document,{embedded=false,chooseWallet=null}={}){
// The bundled official Iqons adapter validates public Nimiq addresses through this hook.
if(!embedded)window.PhoenixNimiqPayments={addressValid};

const $=id=>root.querySelector('[id="'+id+'"]');
const t=key=>flowText(document.documentElement.lang,key);
const walletWords={
 en:{cancelled:'Payment cancelled in Nimiq Pay.',failed:'Nimiq Pay could not complete the request. Check your wallet before using the Hub: a payment may already have been sent.',hub:'Use the Nimiq wallet (Hub)',confirm:'Only continue if your wallet shows that this card has NOT been paid. Pay the same card with the Hub?',network:'Switch Nimiq Pay to the network used by this service before paying.'},
 fr:{cancelled:'Paiement annulé dans Nimiq Pay.',failed:'Nimiq Pay n’a pas pu terminer la demande. Vérifiez votre portefeuille avant d’utiliser le Hub : un paiement peut déjà être parti.',hub:'Utiliser le portefeuille Nimiq (Hub)',confirm:'Continuez seulement si votre portefeuille indique que cette carte n’a PAS été payée. Alimenter la même carte avec le Hub ?',network:'Choisissez dans Nimiq Pay le réseau utilisé par ce service avant de payer.'},
 es:{cancelled:'Pago cancelado en Nimiq Pay.',failed:'Nimiq Pay no pudo completar la solicitud. Revisa tu cartera antes de usar el Hub: el pago podría haberse enviado.',hub:'Usar la cartera Nimiq (Hub)',confirm:'Continúa solo si tu cartera indica que esta tarjeta NO se ha pagado. ¿Pagar la misma tarjeta con el Hub?',network:'Selecciona en Nimiq Pay la red que usa este servicio antes de pagar.'},
 de:{cancelled:'Zahlung in Nimiq Pay abgebrochen.',failed:'Nimiq Pay konnte die Anfrage nicht abschließen. Prüfe deine Wallet vor der Nutzung des Hubs: Die Zahlung könnte bereits gesendet sein.',hub:'Nimiq Wallet (Hub) verwenden',confirm:'Nur fortfahren, wenn diese Karte laut deiner Wallet NICHT bezahlt wurde. Dieselbe Karte mit dem Hub bezahlen?',network:'Wähle vor der Zahlung in Nimiq Pay das Netzwerk dieses Dienstes.'},
 zh:{cancelled:'已在 Nimiq Pay 中取消付款。',failed:'Nimiq Pay 未能完成请求。使用 Hub 前请检查钱包：付款可能已发送。',hub:'使用 Nimiq 钱包（Hub）',confirm:'仅在钱包显示此卡尚未付款时继续。是否通过 Hub 为同一张卡付款？',network:'付款前，请在 Nimiq Pay 中选择此服务所用的网络。'}
};
const wt=key=>(walletWords[document.documentElement.lang]||walletWords.en)[key];
let fallbackCheckout=null,destroyed=false;
const fallback=document.createElement('button');fallback.type='button';fallback.className='secondary-btn wire-nim-hub-fallback';fallback.hidden=true;
$('feedback').after(fallback);
let configured=false,ready=false,verified=false,busy=false,claimMode=false,active=null;
let receiveURL='',cooling=0,coolTimer=null;
let paymentList=[];
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
 fallback.textContent=wt('hub');
 root.querySelectorAll('[data-wire]').forEach(el=>el.textContent=t(el.dataset.wire));
 if(configured&&!claimMode){$('preview-submit').textContent=t(active&&active.stage!=='ready'?'retry':'pay');$('notice').textContent=t(ready?'note':'unavailable');root.querySelector('.security').textContent=t('security');root.querySelector('.lock').textContent=cardPresentation(document.documentElement.lang).locked;}
 if(receiveURL)$('release-claim').textContent=t('receive');
 if(cooling)updateClaimControls();
}
window.addEventListener('wire-language',refreshText);
async function task(run,target='feedback',cleanup){
 if(busy)return;busy=true;status(target,'working');
 try{await run();}catch(error){$(target).textContent=explain(error);}finally{busy=false;cleanup?.();}
}
let usdtAssets=[];
function senderUI(email){verified=Boolean(email);window.wireUsdtState={assets:usdtAssets,isVerified:verified};if(!embedded)window.WireUSDT?.configure(window.wireUsdtState);$('sender').value=email||$('sender').value;$('sender').readOnly=verified;$('send-sender-code').hidden=verified;$('logout').hidden=!verified;$('sender-code-box').hidden=true;if(verified)status('sender-status','verified');}
function showPayment(p){
 $('preview-empty').hidden=true;$('payment-card').hidden=false;$('preview-title').hidden=false;
 $('recipient').value=p.recipient;$('amount').value=p.amount;$('message').value=p.message;$('currency').value='NIM';
 for(const id of ['recipient','amount','message'])$(id).readOnly=true;$('currency').disabled=true;
 $('card-amount').textContent=p.amount;$('card-unit').textContent='NIM';$('card-network').textContent='Nimiq';$('card-message').textContent=p.message;$('card-recipient').textContent=p.recipient;
 $('payment-card').dataset.currency='NIM';$('card-network').hidden=true;
 const words=paintCard($('payment-card'),document.documentElement.lang,p.id);
 root.querySelector('.card-heading').textContent='Nimiq · '+words.heading;root.querySelector('.sample').textContent=words.protected;
 root.querySelector('.lock').textContent=words.locked;
 $('preview-note').hidden=true;
 const image=document.createElement('img');image.width=62;image.height=62;image.alt='Nimiq Nimicon';$('coin').textContent='';$('coin').append(image);window.PhoenixNimiqIdenticons?.paint(image,p.address);
}
async function loadHistory(){
 const data=await api('/api/payments');$('my-payments').hidden=false;$('payment-list').textContent='';
 paymentList=data.payments;
 for(const p of data.payments){
  const row=document.createElement('div');row.className='history-row';
  const label=document.createElement('p');label.textContent=`${p.amount} NIM · ${p.recipient} · ${t(p.stage)}`;row.append(label);
  const refresh=document.createElement('button');refresh.type='button';refresh.textContent=t('retry');refresh.addEventListener('click',()=>task(()=>confirm(p.id)));row.append(refresh);
  if(['sent','delivery_failed','delivery_pending'].includes(p.stage)){const resend=document.createElement('button');resend.type='button';resend.textContent=t('resend');resend.addEventListener('click',()=>task(async()=>{await ensureArtwork(p);await api(`/api/payments/${p.id}/resend`,{});await loadHistory();status('feedback','sent');}));row.append(resend);}
  if(p.address){const recover=document.createElement('button');recover.type='button';recover.textContent=t('recover');recover.addEventListener('click',()=>task(async()=>{const result=await api(`/api/payments/${p.id}/recovery`,{});const link=document.createElement('a');link.href=validatedCashlink(result.cashlink);link.target='_blank';link.rel='noopener noreferrer';link.textContent=t('recover');recover.replaceWith(link);}));row.append(recover);}
  $('payment-list').append(row);
 }
 active=data.payments.find(p=>!['sent'].includes(p.stage))||null;
 if(active||data.payments[0])showPayment(active||data.payments[0]);
 $('new-card').hidden=Boolean(active);refreshText();
}
function validatedCashlink(value){const url=new URL(value);if(url.origin!=='https://hub.nimiq.com'||url.pathname!=='/cashlink/'||!/^#[A-Za-z0-9_=-]+$/.test(url.hash))throw Error('invalid_cashlink');return url.href;}
const artSaved=new Set();
async function ensureArtwork(p){if(artSaved.has(p.id)||!p.address)return;const png=await window.PhoenixNimiqIdenticons.png(p.address);await api(`/api/payments/${p.id}/art`,{png});artSaved.add(p.id);}
async function confirm(id){const p=active?.id===id?active:paymentList.find(p=>p.id===id);if(p)await ensureArtwork(p);const result=await api(`/api/payments/${id}/confirm`,{});if(result.payment.stage==='sent'){fallback.hidden=true;fallbackCheckout=null;if(read('wire-pending-payment')===id){save('wire-pending-payment','');save('wire-pending-wallet','');save('wire-form','');}}await loadHistory();status('feedback',result.payment.stage==='sent'?'sent':'pending');}
async function poll(id){for(let i=0;i<20;i++){try{await confirm(id);return;}catch(error){if(error.message!=='funding_pending')throw error;}await new Promise(resolve=>setTimeout(resolve,3000));}status('feedback','pending');await loadHistory();}
async function withReferrer(run){const meta=document.createElement('meta');meta.name='referrer';meta.content='strict-origin';document.head.append(meta);try{return await run();}finally{meta.remove();}}
function mobile(){return /iPhone|iPad|iPod|Android/i.test(navigator.userAgent||'')||(navigator.platform==='MacIntel'&&navigator.maxTouchPoints>1);}
$('send-sender-code').addEventListener('click',()=>task(async()=>{
 if(!$('sender').reportValidity())return;await api('/api/sender/request',{email:$('sender').value,language:document.documentElement.lang});$('sender-code-box').hidden=false;status('sender-status','codeSent');
},'sender-status'));
$('verify-sender-code').addEventListener('click',()=>task(async()=>{
 const result=await api('/api/sender/verify',{code:$('sender-code').value});$('sender-code').value='';senderUI(result.email);await loadHistory();
},'sender-status'));
$('logout').addEventListener('click',()=>task(async()=>{await api('/api/logout',{});location.reload();}));
$('new-card').addEventListener('click',()=>{requestID=nextID();save('wire-request-id',requestID);active=null;for(const id of ['recipient','amount','message'])$(id).readOnly=false;$('currency').disabled=false;$('recipient').value='';$('message').value='';$('feedback').textContent='';});
function nativeWallet(){return typeof window.nimiq?.sendBasicTransactionWithData==='function';}
function rememberPayment(p,wallet){
 save('wire-pending-payment',p.id);save('wire-pending-wallet',wallet);
 save('wire-form',JSON.stringify({sender:$('sender').value,recipient:p.recipient,amount:p.amount,message:p.message,currency:'NIM',payment:p.id}));
}
async function hubCheckout(p,checkout){
 if(!window.HubApi)throw Error('wallet_unavailable');
 rememberPayment(p,'hub');
 if(mobile()){
  const behavior=new window.HubApi.RedirectRequestBehavior(location.origin+'/',{wirePayment:p.id});
  await withReferrer(()=>new window.HubApi('https://hub.nimiq.com',behavior).checkout(checkout));return;
 }
 try{await withReferrer(()=>new window.HubApi('https://hub.nimiq.com',new window.HubApi.PopupRequestBehavior(undefined,{overlay:false})).checkout(checkout));}catch{await loadHistory();status('feedback','unknownWallet');return;}
 await poll(p.id);
}
// A failed native request can have been broadcast. Check the same card before offering another wallet.
fallback.addEventListener('click',()=>task(async()=>{
 const saved=fallbackCheckout;if(!saved)return;
 try{await confirm(saved.p.id);fallback.hidden=true;fallbackCheckout=null;return;}catch(error){if(error.message!=='funding_pending')throw error;}
 if(!window.confirm(wt('confirm')))return;
 fallback.hidden=true;await hubCheckout(saved.p,saved.checkout);
}));
function fundNim(forceHub=false){
 if(busy)return;
 const native=!forceHub&&nativeWallet();
 const openWallet=!native&&verified&&ready&&$('currency').value==='NIM'&&(!active||active.stage==='ready')&&!mobile();
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
  await ensureArtwork(p);
  if(p.stage!=='ready'){await confirm(p.id);return;}
  // No unprotected fallback. No seed is present in this checkout request.
  if(!native&&!window.HubApi)throw Error('wallet_unavailable');
  if(native&&window.nimiqPay?.network&&window.nimiqPay.network!=='mainnet'){$('feedback').textContent=wt('network');return;}
  const started=await api(`/api/payments/${p.id}/start`,{});active=started.payment;
  // Hub's checkout extraData becomes the chain transaction's recipientData.
  const checkout={appName:started.checkout.appName,recipient:started.checkout.recipient,value:started.checkout.value,extraData:new Uint8Array(started.checkout.recipientData)};
  if(native){
   // The host API accepts text, not bytes. Its UTF-8 encoding is 00c282c280c292c287.
   // The server must accept that exact marker as well as the Hub's original 0082809287.
   const marker=[0,130,128,146,135];
   if(!Number.isSafeInteger(checkout.value)||checkout.value<=0||!addressValid(checkout.recipient)||started.checkout.recipientData?.length!==marker.length||!marker.every((b,i)=>started.checkout.recipientData[i]===b))throw Error('invalid_checkout');
   rememberPayment(p,'pay');
   let failure;
   try{
    const result=await window.nimiq.sendBasicTransactionWithData({recipient:checkout.recipient,value:checkout.value,data:'\u0000\u0082\u0080\u0092\u0087'});
    // Direct providers can return the older { error } envelope instead of rejecting.
    if(result?.error)throw result.error;
   }catch(error){failure=error||{};}
   if(failure){
    const refused=failure.type==='PERMISSION_DENIED'||failure.code===4001||failure.name==='PermissionDeniedError';
    if(refused){$('feedback').textContent=wt('cancelled');return;}
    try{await confirm(p.id);return;}catch(error){if(error.message!=='funding_pending')throw error;}
    fallbackCheckout={p,checkout};fallback.hidden=false;$('feedback').textContent=wt('failed');return;
   }
   // The returned hash is not evidence of funding; the existing server check decides.
   await poll(p.id);return;
  }
  await hubCheckout(p,checkout);
 },'feedback',()=>{if(popup&&!popup.closed)popup.close();});
}
$('card-form').addEventListener('submit',event=>{
 if(configured&&window.WireUSDT?.enabled($('currency').value)){event.preventDefault();window.WireUSDT.submit();return;}
 if(!configured)return;event.preventDefault();
 if(busy)return;
 if(verified&&ready&&$('currency').value==='NIM'&&(!active||active.stage==='ready')&&!nativeWallet()&&!window.nimiqPay&&chooseWallet){chooseWallet(()=>{if(!destroyed)fundNim(true);});return;}
 fundNim();
});
function updateClaimControls(){
 $('request-claim-code').disabled=busy||cooling>0;
 $('request-claim-code').textContent=cooling?t('resendIn')+' '+cooling+' s':t('requestCode');
 $('release-claim').disabled=busy||(!receiveURL&&!/^[0-9]{6}$/.test($('claim-code').value));
}
function cooldown(){clearInterval(coolTimer);cooling=20;updateClaimControls();coolTimer=setInterval(()=>{cooling--;updateClaimControls();if(cooling<=0)clearInterval(coolTimer);},1000);}
async function claimTask(run,button){if(busy)return;button.setAttribute('aria-busy','true');let ok=false;await task(async()=>{updateClaimControls();await run();ok=true;},'claim-status',()=>{button.removeAttribute('aria-busy');$('claim-status').className=ok?'ok':'error';updateClaimControls();});}
$('claim-code').addEventListener('input',()=>{$('claim-code').value=$('claim-code').value.replace(/\D/g,'').slice(0,6);updateClaimControls();});
$('request-claim-code').addEventListener('click',()=>claimTask(async()=>{try{await api(`/api/claims/${claimID()}/request`,{});status('claim-status','codeSent');cooldown();}catch(error){if(error.status===429){cooldown();status('claim-status','waitCode');return;}throw error;}},$('request-claim-code')));
$('claim-form').addEventListener('submit',event=>{event.preventDefault();if(receiveURL){if(mobile())location.assign(receiveURL);else window.open(receiveURL,'_blank','noopener,noreferrer');return;}if(!/^[0-9]{6}$/.test($('claim-code').value))return;claimTask(async()=>{
 const result=await api(`/api/claims/${claimID()}/release`,{code:$('claim-code').value});receiveURL=validatedCashlink(result.cashlink);$('claim-code').value='';
 $('request-claim-code').hidden=true;$('claim-code').closest('label').hidden=true;clearInterval(coolTimer);cooling=0;
 $('claim-code').required=false;$('claim-code').disabled=true;
 $('release-claim').textContent=t('receive');status('claim-status','accepted');
},$('release-claim'));});
function claimID(){const id=new URLSearchParams(location.search).get('claim');if(!/^[a-f0-9]{48}$/.test(id||''))throw Error('claim_unavailable');return id;}
async function boot(){
 refreshText();claimMode=!embedded&&new URLSearchParams(location.search).has('claim');
 if(claimMode){
  root.querySelector('.workspace').hidden=true;$('notice').hidden=true;$('claim-panel').hidden=false;
  try{const p=await api('/api/claims/'+claimID());paintCard($('claim-card'),document.documentElement.lang,p.id);$('claim-info').textContent=t('from')+' '+p.sender;$('claim-amount').textContent=p.amount+' NIM';$('claim-message').textContent=p.message;
   window.PhoenixNimiqIdenticons?.paint($('claim-iqon'),p.address);
   if(new Date(p.expires_at).getTime()<=Date.now()){hideClaimInputs();status('claim-status','expired');}
  }catch(error){hideClaimInputs();$('claim-status').textContent=explain(error);}return;
 }
 try{const config=await api('/api/config');configured=config.mode==='integrated';ready=config.payments_enabled===true;usdtAssets=Array.isArray(config.assets)?config.assets:[];if(!configured)return;
  $('sender-auth').hidden=embedded;refreshText();const session=await api('/api/session');senderUI(session.verified?session.email:'');if(verified)await loadHistory();
  try{const draft=JSON.parse(read('wire-form')||'null');if(draft&&active?.id===draft.payment)showPayment(active);}catch{}
  const pending=read('wire-pending-payment');
  if(pending&&/^[a-f0-9]{48}$/.test(pending)&&read('wire-pending-wallet')==='pay'){
   try{await confirm(pending);}catch(error){$('feedback').textContent=explain(error);}return;
  }
  if(pending&&/^[a-f0-9]{48}$/.test(pending)&&window.HubApi){
   const hub=new window.HubApi('https://hub.nimiq.com',new window.HubApi.RedirectRequestBehavior());
   hub.on(window.HubApi.RequestType.CHECKOUT,()=>task(()=>poll(pending)),()=>status('feedback','unknownWallet'));
   await hub.checkRedirectResponse();
  }
 }catch{status('feedback','unavailable');}
}
const started=boot();
function hideClaimInputs(){$('request-claim-code').hidden=true;$('release-claim').hidden=true;$('claim-code').closest('label').hidden=true;$('claim-code').disabled=true;}

return {ready:started,destroy(){destroyed=true;window.removeEventListener('wire-language',refreshText);clearInterval(coolTimer);}};
}

if(typeof document!=='undefined'&&document.getElementById('card-form'))mountNimFlow();
