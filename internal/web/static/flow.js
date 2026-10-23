import { flowText } from './flow-text.js?v=0.3.0';
import { addressValid,cardPresentation,paintCard } from './model.js?v=0.3.0';

// The same protected NIM flow mounts either on classic.html or inside the authenticated payment window.
export function mountNimFlow(root=document,{embedded=false,chooseWallet=null,onClose=null,onStatus=()=>{}}={}){
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
const progressWords={
 en:{waiting:'Payment sent. Waiting for network confirmation…',checking:'Checking the payment on the network…',mail:'Payment confirmed. Sending the card to {email}…',sent:'Card sent to {email}. It stays yours until it is claimed.',failed:'Payment confirmed, but the card email could not be sent. You can resend it without paying again.',close:'Close',resend:'Resend',offline:'The service is not responding. Checking will resume automatically. Do not pay again.',session:'Confirm your sender email again to resume tracking this card. Do not pay again.',expired:'This card has reached its 24-hour expiry. Check its history to recover any remaining funds.'},
 fr:{waiting:'Paiement envoyé. Confirmation sur le réseau en cours…',checking:'Vérification du paiement sur le réseau en cours…',mail:'Paiement confirmé. Envoi de la carte à {email}…',sent:'Carte envoyée à {email}. Elle reste à vous tant qu’elle n’est pas réclamée.',failed:'Paiement confirmé, mais le courriel de la carte n’a pas pu être envoyé. Vous pouvez le renvoyer sans repayer.',close:'Fermer',resend:'Renvoyer',offline:'Le service ne répond pas. La vérification reprendra automatiquement. Ne payez pas une deuxième fois.',session:'Confirmez de nouveau votre courriel d’expéditeur pour reprendre le suivi de cette carte. Ne payez pas une deuxième fois.',expired:'Cette carte a atteint son échéance de 24 heures. Consultez son historique pour reprendre les fonds restants.'},
 es:{waiting:'Pago enviado. Esperando la confirmación de la red…',checking:'Comprobando el pago en la red…',mail:'Pago confirmado. Enviando la tarjeta a {email}…',sent:'Tarjeta enviada a {email}. Sigue siendo tuya hasta que se reclame.',failed:'Pago confirmado, pero no se pudo enviar el correo de la tarjeta. Puedes reenviarlo sin volver a pagar.',close:'Cerrar',resend:'Reenviar',offline:'El servicio no responde. La comprobación se reanudará automáticamente. No vuelvas a pagar.',session:'Confirma de nuevo tu correo de remitente para reanudar el seguimiento. No vuelvas a pagar.',expired:'Esta tarjeta ha llegado a su vencimiento de 24 horas. Consulta su historial para recuperar los fondos restantes.'},
 de:{waiting:'Zahlung gesendet. Netzwerkbestätigung ausstehend…',checking:'Zahlung wird im Netzwerk geprüft…',mail:'Zahlung bestätigt. Die Karte wird an {email} gesendet…',sent:'Karte an {email} gesendet. Sie bleibt deine, bis sie eingelöst wird.',failed:'Zahlung bestätigt, aber die Karten-E-Mail konnte nicht gesendet werden. Du kannst sie ohne erneute Zahlung senden.',close:'Schließen',resend:'Erneut senden',offline:'Der Dienst antwortet nicht. Die Prüfung wird automatisch fortgesetzt. Bezahle nicht erneut.',session:'Bestätige deine Absender-E-Mail erneut, um die Karte weiterzuverfolgen. Bezahle nicht erneut.',expired:'Die Karte hat ihre Laufzeit von 24 Stunden erreicht. Prüfe den Verlauf, um übriges Guthaben zurückzuholen.'},
 zh:{waiting:'付款已发送。正在等待网络确认…',checking:'正在网络上检查付款…',mail:'付款已确认。正在向 {email} 发送卡片…',sent:'卡片已发送至 {email}。在被领取之前它仍属于你。',failed:'付款已确认，但卡片邮件未能发送。你可以重新发送，无需再次付款。',close:'关闭',resend:'重新发送',offline:'服务暂未响应。检查将自动恢复，请勿再次付款。',session:'请重新确认发件人邮箱以继续跟踪此卡片。请勿再次付款。',expired:'此卡片已达到 24 小时有效期。请查看历史记录以取回剩余资金。'}
};
const pt=key=>(progressWords[document.documentElement.lang]||progressWords.en)[key];
let fallbackCheckout=null,destroyed=false;
const fallback=document.createElement('button');fallback.type='button';fallback.className='secondary-btn wire-nim-hub-fallback';fallback.hidden=true;
$('feedback').after(fallback);
let configured=false,ready=false,verified=false,busy=false,claimMode=false,active=null;
let receiveURL='',cooling=0,coolTimer=null;
let paymentList=[];
let displayed=null;
const monitors=new Map(),checks=new Map(),controllers=new Set();
let progress=$('nim-progress');
if(!progress){
 progress=document.createElement('section');progress.id='nim-progress';progress.className='wire-nim-progress';progress.hidden=true;
 const message=document.createElement('p');message.id='nim-progress-message';message.setAttribute('role','status');message.setAttribute('aria-live','polite');
 const close=document.createElement('button');close.id='nim-progress-close';close.type='button';close.className='secondary-btn';close.hidden=true;
 const resend=document.createElement('button');resend.id='nim-progress-resend';resend.type='button';resend.className='secondary-btn';resend.hidden=true;
 progress.append(message,close,resend);root.querySelector('.workspace').before(progress);
}
const read=key=>{try{return sessionStorage.getItem(key)||'';}catch{return '';}};
const save=(key,value)=>{try{sessionStorage.setItem(key,value);}catch{}};
let requestID=read('wire-request-id');
function nextID(){const bytes=crypto.getRandomValues(new Uint8Array(16));return Array.from(bytes,b=>b.toString(16).padStart(2,'0')).join('');}
if(!/^[a-f0-9]{32}$/.test(requestID)){requestID=nextID();save('wire-request-id',requestID);}
async function api(path,data,{timeout=false}={}){
 if(destroyed)throw Error('flow_closed');
 const options={credentials:'same-origin',cache:'no-store',redirect:'error'};
 if(data!==undefined){options.method='POST';options.headers={'Content-Type':'application/json'};options.body=JSON.stringify(data);}
 const controller=timeout?new AbortController():null;
 if(controller)controllers.add(controller);
 const timer=controller?setTimeout(()=>controller.abort(),30000):null;
 if(controller)options.signal=controller.signal;
 try{const response=await fetch(path,options);const body=await response.json();if(destroyed)throw Error('flow_closed');
 if(!response.ok){const error=Error(body.code||body.error||'unavailable');error.status=response.status;throw error;}return body;
 }finally{if(timer!==null)clearTimeout(timer);if(controller)controllers.delete(controller);}
}
function status(id,key){$(id).textContent=t(key);}
function explain(error){if(error.message==='funding_pending'||error.message==='funding_already_started')return t('pending');if(error.status===403)return t('invalid');if(error.status===410)return t('expired');if(error.status===401)return t('verifyFirst');return t('failed');}
function refreshText(){
 fallback.textContent=wt('hub');
 root.querySelectorAll('[data-wire]').forEach(el=>el.textContent=t(el.dataset.wire));
 if(configured&&!claimMode){$('preview-submit').textContent=t(active&&active.stage!=='ready'?'retry':'pay');$('notice').textContent=t(ready?'note':'unavailable');root.querySelector('.security').textContent=t('security');root.querySelector('.lock').textContent=cardPresentation(document.documentElement.lang).locked;}
 if(receiveURL)$('release-claim').textContent=t('receive');
 if(cooling)updateClaimControls();
 renderProgress();
}
window.addEventListener('wire-language',refreshText);
async function task(run,target='feedback',cleanup){
 if(busy)return;busy=true;status(target,'working');
 renderProgress();
 try{await run();}catch(error){$(target).textContent=explain(error);}finally{busy=false;cleanup?.();renderProgress();}
}
let usdtAssets=[];
function senderUI(email){verified=Boolean(email);window.wireUsdtState={assets:usdtAssets,isVerified:verified};if(!embedded)window.WireUSDT?.configure(window.wireUsdtState);$('sender').value=email||$('sender').value;$('sender').readOnly=verified;$('send-sender-code').hidden=verified;$('logout').hidden=!verified;$('sender-code-box').hidden=true;if(verified)status('sender-status','verified');}
function showPayment(p){
 displayed=p;
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
 renderProgress();
}
async function loadHistory(){
 const data=await api('/api/payments',undefined,{timeout:true});$('my-payments').hidden=false;$('payment-list').textContent='';
 paymentList=data.payments;
 for(const p of data.payments){
  const row=document.createElement('div');row.className='history-row';
  const label=document.createElement('p');label.textContent=`${p.amount} NIM · ${p.recipient} · ${t(p.stage)}`;row.append(label);
  if(p.stage==='delivery_failed'){const resend=document.createElement('button');resend.type='button';resend.textContent=pt('resend');resend.disabled=checks.has(p.id);resend.addEventListener('click',()=>resendCard(p.id));row.append(resend);}
  if(p.address){const recover=document.createElement('button');recover.type='button';recover.textContent=t('recover');recover.addEventListener('click',()=>task(async()=>{const result=await api(`/api/payments/${p.id}/recovery`,{});const link=document.createElement('a');link.href=validatedCashlink(result.cashlink);link.target='_blank';link.rel='noopener noreferrer';link.textContent=t('recover');recover.replaceWith(link);}));row.append(recover);}
  $('payment-list').append(row);
 }
 active=data.payments.find(p=>!['sent'].includes(p.stage))||null;
 const pending=read('wire-pending-payment');
 const shown=data.payments.find(p=>p.id===pending)||data.payments.find(p=>p.id===displayed?.id)||active||data.payments[0];
 for(const p of data.payments){const watch=monitors.get(p.id);if(watch){watch.payment=p;if(p.stage==='sent')stopMonitor(watch);}}
 if(shown)showPayment(shown);
 $('new-card').hidden=Boolean(active);refreshText();
}
function validatedCashlink(value){const url=new URL(value);if(url.origin!=='https://hub.nimiq.com'||url.pathname!=='/cashlink/'||!/^#[A-Za-z0-9_=-]+$/.test(url.hash))throw Error('invalid_cashlink');return url.href;}
const artSaved=new Set();
async function ensureArtwork(p){if(artSaved.has(p.id)||!p.address)return;const png=await window.PhoenixNimiqIdenticons.png(p.address);await api(`/api/payments/${p.id}/art`,{png},{timeout:true});artSaved.add(p.id);}
function deadline(p){
 const created=Date.parse(p.created||'');if(Number.isFinite(created))return created+86400000;
 const key='wire-nim-until:'+p.id;let until=Number(read(key));if(!until){until=Date.now()+86400000;save(key,String(until));}return until;
}
function monitor(p){
 if(!monitors.has(p.id))monitors.set(p.id,{payment:p,until:deadline(p),since:Date.now(),timer:null,running:null,note:'',stopped:false});
 const watch=monitors.get(p.id);watch.payment=p;return watch;
}
function stopMonitor(watch){clearTimeout(watch.timer);watch.timer=null;watch.stopped=true;}
function progressText(p){
 const watch=monitors.get(p.id);
 if(p.stage==='sent')return pt('sent').replace('{email}',p.recipient);
 if(Date.now()>=(watch?.until??deadline(p)))return pt('expired');
 if(p.stage==='delivery_failed')return pt('failed');
 if(watch?.note)return pt(watch.note)||watch.note;
 if(['funded','delivery_pending'].includes(p.stage))return pt('mail').replace('{email}',p.recipient);
 return pt(read('wire-nim-approved:'+p.id)==='yes'?'waiting':'checking');
}
function renderProgress(){
 if(destroyed)return;
 const p=displayed,show=p&&['funding_started','funded','delivery_pending','delivery_failed','sent'].includes(p.stage);
 progress.hidden=!show;if(!claimMode)$('preview-submit').hidden=Boolean(show);if(!show)return;
 const text=progressText(p);$('nim-progress-message').textContent=text;
 $('nim-progress-close').textContent=pt('close');$('nim-progress-close').hidden=p.stage!=='sent'&&Date.now()<deadline(p);
 $('nim-progress-resend').textContent=pt('resend');$('nim-progress-resend').hidden=p.stage!=='delivery_failed'||Date.now()>=deadline(p);
 $('nim-progress-resend').disabled=checks.has(p.id)||busy;
 $('preview-submit').hidden=true;
 onStatus(p,text);
}
function observe(p){
 if(destroyed)return;
 const watch=monitor(p);watch.note='';
 if(p.stage==='sent'){
  stopMonitor(watch);fallback.hidden=true;fallbackCheckout=null;
  if(read('wire-pending-payment')===p.id){save('wire-pending-payment','');save('wire-pending-wallet','');save('wire-form','');}
 }
 if(!displayed||displayed.id===p.id||read('wire-pending-payment')===p.id)showPayment(p);
}
function confirm(id,resend=false){
 if(checks.has(id))return checks.get(id);
 const promise=(async()=>{
  await loadHistory();
  const p=paymentList.find(p=>p.id===id);if(!p)throw Error('payment_unavailable');
  observe(p);
  if(p.stage==='sent'||!resend&&['delivery_pending','delivery_failed'].includes(p.stage))return {payment:p};
  if(Date.now()>=deadline(p))throw Object.assign(Error('claim_expired'),{status:410});
  if(resend&&p.stage!=='delivery_failed')return {payment:p};
  await ensureArtwork(p);
  try{const result=await api(`/api/payments/${id}/${resend?'resend':'confirm'}`,{},{timeout:true});observe(result.payment);await loadHistory();$('feedback').textContent=progressText(result.payment);return result;}
  catch(error){
   if(error.message==='funding_pending')throw error;
   try{await loadHistory();const current=paymentList.find(p=>p.id===id);if(current){observe(current);if(current.stage==='sent')return {payment:current};}}catch{}
   throw error;
  }
 })();
 checks.set(id,promise);
 promise.finally(()=>{checks.delete(id);if(!destroyed)renderProgress();}).catch(()=>{});
 return promise;
}
function schedule(watch){
 clearTimeout(watch.timer);watch.timer=null;
 if(destroyed||watch.stopped||document.hidden)return;
 const remaining=watch.until-Date.now();if(remaining<=0){stopMonitor(watch);renderProgress();return;}
 const age=Date.now()-watch.since;
 const interval=watch.payment.stage==='delivery_failed'||age>=300000?30000:age>=60000?10000:4000;
 watch.timer=setTimeout(()=>{watch.timer=null;void runMonitor(watch);},Math.min(interval,remaining));
}
function runMonitor(watch){
 if(watch.running)return watch.running;
 if(destroyed||watch.stopped)return Promise.resolve();
 if(Date.now()>=watch.until){stopMonitor(watch);renderProgress();return Promise.resolve();}
 clearTimeout(watch.timer);watch.timer=null;
 watch.running=(async()=>{
  try{await confirm(watch.payment.id);watch.note='';}
  catch(error){watch.note=error.status===401?'session':error.message==='funding_pending'?'':error.status===410?'expired':'offline';if(error.status===410)stopMonitor(watch);}
  finally{watch.running=null;if(!destroyed){renderProgress();schedule(watch);}}
 })();return watch.running;
}
async function poll(id){
 const p=paymentList.find(p=>p.id===id)||active;
 if(!p||p.id!==id)return;
 const watch=monitor(p);if(p.stage==='sent'){observe(p);return;}
 await runMonitor(watch);
}
function approve(p){save('wire-nim-approved:'+p.id,'yes');observe(p);$('feedback').textContent=progressText(p);}
function resume(){
 if(destroyed||claimMode||document.hidden||!verified)return;
 renderProgress();
 for(const p of paymentList){if(['funding_started','funded','delivery_pending','delivery_failed'].includes(p.stage))void runMonitor(monitor(p));}
}
function visibility(){if(document.hidden){for(const watch of monitors.values()){clearTimeout(watch.timer);watch.timer=null;}}else resume();}
document.addEventListener('visibilitychange',visibility);window.addEventListener('pageshow',resume);window.addEventListener('online',resume);
$('nim-progress-close').addEventListener('click',()=>{if(onClose)onClose(displayed);else location.assign('/');});
function resendCard(id){
 if(busy||checks.has(id))return;
 void task(async()=>{try{await confirm(id,true);}catch(error){const watch=monitors.get(id);if(watch)watch.note=error.message==='wait_before_resend'?t('waitCode'):pt('offline');throw error;}finally{const watch=monitors.get(id);if(watch)schedule(watch);renderProgress();}});
}
$('nim-progress-resend').addEventListener('click',()=>{if(displayed)resendCard(displayed.id);});
async function withReferrer(run){const meta=document.createElement('meta');meta.name='referrer';meta.content='strict-origin';document.head.append(meta);try{return await run();}finally{meta.remove();}}
function mobile(){return /iPhone|iPad|iPod|Android/i.test(navigator.userAgent||'')||(navigator.platform==='MacIntel'&&navigator.maxTouchPoints>1);}
$('send-sender-code').addEventListener('click',()=>task(async()=>{
 if(!$('sender').reportValidity())return;await api('/api/sender/request',{email:$('sender').value,language:document.documentElement.lang});$('sender-code-box').hidden=false;status('sender-status','codeSent');
},'sender-status'));
$('verify-sender-code').addEventListener('click',()=>task(async()=>{
 const result=await api('/api/sender/verify',{code:$('sender-code').value});$('sender-code').value='';senderUI(result.email);await loadHistory();
},'sender-status'));
$('logout').addEventListener('click',()=>task(async()=>{await api('/api/logout',{});location.reload();}));
$('new-card').addEventListener('click',()=>{requestID=nextID();save('wire-request-id',requestID);active=null;displayed=null;for(const id of ['recipient','amount','message'])$(id).readOnly=false;$('currency').disabled=false;$('recipient').value='';$('message').value='';$('feedback').textContent='';renderProgress();});
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
 approve(active?.id===p.id?active:p);await poll(p.id);
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
 if(displayed?.stage==='sent')return;
 const native=!forceHub&&nativeWallet();
 const openWallet=!native&&verified&&ready&&$('currency').value==='NIM'&&(!active||active.stage==='ready')&&!mobile();
 const popup=openWallet?window.open('about:blank','NimiqAccounts','width=800,height=850'):null;
 if(openWallet&&!popup){status('feedback','failed');return;}
 task(async()=>{
  if(!verified){status('feedback','verifyFirst');return;}
  if(!ready||$('currency').value!=='NIM'){status('feedback','unavailable');return;}
  // Restored pending records can only be checked, never automatically funded again.
  if(active&&active.stage==='preparing'){const result=await api(`/api/payments/${active.id}/prepare`,{});active=result.payment;await loadHistory();return;}
  if(active&&active.stage!=='ready'){await poll(active.id);return;}
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
   approve(active);await poll(p.id);return;
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
   await poll(pending);resume();return;
  }
  if(pending&&/^[a-f0-9]{48}$/.test(pending)&&window.HubApi){
   const hub=new window.HubApi('https://hub.nimiq.com',new window.HubApi.RedirectRequestBehavior());
   hub.on(window.HubApi.RequestType.CHECKOUT,()=>task(async()=>{const p=paymentList.find(p=>p.id===pending);if(p)approve(p);await poll(pending);}),()=>{status('feedback','unknownWallet');resume();});
   await hub.checkRedirectResponse();
  }
  resume();
 }catch{status('feedback','unavailable');}
}
const started=boot();
function hideClaimInputs(){$('request-claim-code').hidden=true;$('release-claim').hidden=true;$('claim-code').closest('label').hidden=true;$('claim-code').disabled=true;}

return {ready:started,resume,destroy(){destroyed=true;window.removeEventListener('wire-language',refreshText);document.removeEventListener('visibilitychange',visibility);window.removeEventListener('pageshow',resume);window.removeEventListener('online',resume);clearInterval(coolTimer);for(const watch of monitors.values())stopMonitor(watch);for(const controller of controllers)controller.abort();controllers.clear();}};
}

if(typeof document!=='undefined'&&document.getElementById('card-form'))mountNimFlow();
