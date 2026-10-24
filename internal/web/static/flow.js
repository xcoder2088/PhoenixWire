import { mountAdmin,adminText } from './admin.js?v=0.4.25';
import { nimFunded,unpaidCard,recoverableCard,historyEntries,historyPending,senderHistoryText,nimHistoryLabel,cancelUnpaidCard } from './model.js?v=0.4.24';
import { flowText } from './flow-text.js?v=0.4.24';
import { addressValid,cardPresentation,paintCard } from './model.js?v=0.4.24';

// The same protected NIM flow mounts either on classic.html or inside the authenticated payment window.
export function mountNimFlow(root=document,{embedded=false,chooseWallet=null,onClose=null,onStatus=()=>{},initialDraft=null,onDraftUsed=()=>{},initialPayment=null,scopedPaymentID=''}={}){
// The bundled official Iqons adapter validates public Nimiq addresses through this hook.
if(!embedded)window.PhoenixNimiqPayments={addressValid};

const $=id=>root.querySelector('[id="'+id+'"]');
const t=key=>flowText(document.documentElement.lang,key);
const expiryDate=value=>{const ms=typeof value==='number'?value:Date.parse(value||'');return Number.isFinite(ms)?new Intl.DateTimeFormat(document.documentElement.lang,{dateStyle:'medium',timeStyle:'short'}).format(ms):'';};
const lifeNote=document.createElement('p');lifeNote.className='wire-lifetime-note';lifeNote.dataset.wire='life';$('amount').closest('label').after(lifeNote);
const claimExpiry=document.createElement('p');claimExpiry.className='wire-card-expiry';claimExpiry.hidden=true;$('claim-amount').after(claimExpiry);
let senderInviteOnly=false,adminPanel=null,inviteOnly=false,accessExpanded=false,accessKnown=false;
const senderGate=document.createElement('div'),senderNotice=document.createElement('p'),senderAccess=document.createElement('button');senderGate.id='sender-invite-gate';senderGate.hidden=true;senderAccess.id='sender-existing-access';senderAccess.type='button';senderAccess.className='secondary-btn wire-access-link';senderAccess.setAttribute('aria-controls','sender-auth');senderGate.append(senderNotice,senderAccess);$('sender').closest('label').before(senderGate);
function paintSenderAccess(){if(embedded||claimMode)return;senderGate.hidden=!accessKnown||verified||!inviteOnly;senderNotice.textContent=t('inviteOnly');senderAccess.textContent=adminText(document.documentElement.lang,'already');senderAccess.hidden=accessExpanded;senderAccess.setAttribute('aria-expanded',String(accessExpanded));$('sender').closest('label').hidden=!accessKnown||!verified&&inviteOnly&&!accessExpanded;$('sender-auth').hidden=!accessKnown||verified||inviteOnly&&!accessExpanded;}
senderAccess.addEventListener('click',()=>{accessExpanded=true;paintSenderAccess();$('sender').focus();});
function configureAdmin(allowed){if(embedded)return;adminPanel?.destroy();adminPanel=mountAdmin({root:root.querySelector('.app-window')||root.body||root,language:()=>document.documentElement.lang,request:api,authorized:allowed===true&&verified,onAccess:open=>{inviteOnly=!open;paintSenderAccess();}});}

const walletWords={
 en:{cancelled:'Payment cancelled in Nimiq Pay.',failed:'Nimiq Pay could not complete the request. Check your wallet before using Nimiq Wallet: a payment may already have been sent.',hub:'Nimiq Wallet',confirm:'Only continue if your wallet shows that this card has NOT been paid. Pay the same card with Nimiq Wallet?',network:'Switch Nimiq Pay to the network used by this service before paying.'},
 fr:{cancelled:'Paiement annulé dans Nimiq Pay.',failed:'Nimiq Pay n’a pas pu terminer la demande. Vérifiez votre portefeuille avant d’utiliser Nimiq Wallet : un paiement peut déjà être parti.',hub:'Portefeuille Nimiq',confirm:'Continuez seulement si votre portefeuille indique que cette carte n’a PAS été payée. Alimenter la même carte avec Nimiq Wallet ?',network:'Choisissez dans Nimiq Pay le réseau utilisé par ce service avant de payer.'},
 es:{cancelled:'Pago cancelado en Nimiq Pay.',failed:'Nimiq Pay no pudo completar la solicitud. Revisa tu cartera antes de usar Nimiq Wallet: el pago podría haberse enviado.',hub:'Cartera Nimiq',confirm:'Continúa solo si tu cartera indica que esta tarjeta NO se ha pagado. ¿Pagar la misma tarjeta con Nimiq Wallet?',network:'Selecciona en Nimiq Pay la red que usa este servicio antes de pagar.'},
 de:{cancelled:'Zahlung in Nimiq Pay abgebrochen.',failed:'Nimiq Pay konnte die Anfrage nicht abschließen. Prüfe deine Wallet vor der Nutzung von Nimiq Wallet: Die Zahlung könnte bereits gesendet sein.',hub:'Nimiq Wallet',confirm:'Nur fortfahren, wenn diese Karte laut deiner Wallet NICHT bezahlt wurde. Dieselbe Karte mit Nimiq Wallet bezahlen?',network:'Wähle vor der Zahlung in Nimiq Pay das Netzwerk dieses Dienstes.'},
 zh:{cancelled:'已在 Nimiq Pay 中取消付款。',failed:'Nimiq Pay 未能完成请求。使用 Nimiq Wallet 前请检查钱包：付款可能已发送。',hub:'Nimiq 钱包',confirm:'仅在钱包显示此卡尚未付款时继续。是否通过 Nimiq Wallet 为同一张卡付款？',network:'付款前，请在 Nimiq Pay 中选择此服务所用的网络。'}
};
const wt=key=>(walletWords[document.documentElement.lang]||walletWords.en)[key];
const progressWords={
 en:{waiting:'Payment sent. Waiting for network confirmation…',checking:'Checking the payment on the network…',mail:'Payment confirmed. Sending the card to {email}…',sent:'Card sent to {email}. It stays yours until it is claimed.',failed:'Payment confirmed, but the card email could not be sent. You can resend it without paying again.',close:'Close',resend:'Resend',offline:'The service is not responding. Checking will resume automatically. Do not pay again.',session:'Confirm your sender email again to resume tracking this card. Do not pay again.',expired:'This card has reached its 24-hour expiry. Check its history to recover any remaining funds.'},
 fr:{waiting:'Paiement envoyé. Confirmation sur le réseau en cours…',checking:'Vérification du paiement sur le réseau en cours…',mail:'Paiement confirmé. Envoi de la carte à {email}…',sent:'Carte envoyée à {email}. Elle reste à vous tant qu’elle n’est pas réclamée.',failed:'Paiement confirmé, mais le courriel de la carte n’a pas pu être envoyé. Vous pouvez le renvoyer sans repayer.',close:'Fermer',resend:'Renvoyer',offline:'Le service ne répond pas. La vérification reprendra automatiquement. Ne payez pas une deuxième fois.',session:'Confirmez de nouveau votre courriel d’expéditeur pour reprendre le suivi de cette carte. Ne payez pas une deuxième fois.',expired:'Cette carte a atteint son échéance de 24 heures. Consultez son historique pour reprendre les fonds restants.'},
 es:{waiting:'Pago enviado. Esperando la confirmación de la red…',checking:'Comprobando el pago en la red…',mail:'Pago confirmado. Enviando la tarjeta a {email}…',sent:'Tarjeta enviada a {email}. Sigue siendo tuya hasta que se reclame.',failed:'Pago confirmado, pero no se pudo enviar el correo de la tarjeta. Puedes reenviarlo sin volver a pagar.',close:'Cerrar',resend:'Reenviar',offline:'El servicio no responde. La comprobación se reanudará automáticamente. No vuelvas a pagar.',session:'Confirma de nuevo tu correo de remitente para reanudar el seguimiento. No vuelvas a pagar.',expired:'Esta tarjeta ha llegado a su vencimiento de 24 horas. Consulta su historial para recuperar los fondos restantes.'},
 de:{waiting:'Zahlung gesendet. Netzwerkbestätigung ausstehend…',checking:'Zahlung wird im Netzwerk geprüft…',mail:'Zahlung bestätigt. Die Karte wird an {email} gesendet…',sent:'Karte an {email} gesendet. Sie bleibt deine, bis sie eingelöst wird.',failed:'Zahlung bestätigt, aber die Karten-E-Mail konnte nicht gesendet werden. Du kannst sie ohne erneute Zahlung senden.',close:'Schließen',resend:'Erneut senden',offline:'Der Dienst antwortet nicht. Die Prüfung wird automatisch fortgesetzt. Bezahle nicht erneut.',session:'Bestätige deine Absender-E-Mail erneut, um die Karte weiterzuverfolgen. Bezahle nicht erneut.',expired:'Die Karte hat ihre Laufzeit von 24 Stunden erreicht. Prüfe den Verlauf, um übriges Guthaben zurückzuholen.'},
 zh:{waiting:'付款已发送。正在等待网络确认…',checking:'正在网络上检查付款…',mail:'付款已确认。正在向 {email} 发送卡片…',sent:'卡片已发送至 {email}。在被领取之前它仍属于你。',failed:'付款已确认，但卡片邮件未能发送。你可以重新发送，无需再次付款。',close:'关闭',resend:'重新发送',offline:'服务暂未响应。检查将自动恢复，请勿再次付款。',session:'请重新确认发件人邮箱以继续跟踪此卡片。请勿再次付款。',expired:'此卡片已达到 24 小时有效期。请查看历史记录以取回剩余资金。'}
};
const doneWords={fr:'Terminé',en:'Done',es:'Listo',de:'Fertig',zh:'完成'};
const pt=key=>embedded&&key==='close'?(doneWords[document.documentElement.lang]||doneWords.en):(progressWords[document.documentElement.lang]||progressWords.en)[key];

const retryWords={
  "fr": {
    "unpaid": "Paiement non reçu selon la dernière vérification.",
    "unknown": "Impossible de vérifier le paiement pour le moment.",
    "received": "Paiement reçu. Vérification et envoi de la carte en cours…",
    "retry": "Réessayer le paiement",
    "wait": "Un paiement peut encore arriver.",
    "remaining": "Réessai possible dans {time}.",
    "close": "Tu peux fermer cette fenêtre.",
    "copy": "Copier le détail",
    "copied": "Détail copié",
    "select": "Sélectionne et copie le détail ci-dessous.",
    "detail": "Détail technique (données personnelles masquées)",
    "expired": "Carte expirée. Ne repaie pas. Les fonds arrivés tardivement restent récupérables par l’expéditeur dans Safari."
  },
  "en": {
    "unpaid": "Not yet paid according to the last check.",
    "unknown": "Unable to check the payment right now.",
    "received": "Payment received. Checking and sending the card…",
    "retry": "Retry payment",
    "wait": "A payment may still arrive.",
    "remaining": "Retry available in {time}.",
    "close": "You can close this window.",
    "copy": "Copy details",
    "copied": "Details copied",
    "select": "Select and copy the details below.",
    "detail": "Technical details (personal data hidden)",
    "expired": "Card expired. Do not pay again. The sender can recover late-arriving funds in Safari."
  },
  "es": {
    "unpaid": "Aún no pagada según la última comprobación.",
    "unknown": "No se puede comprobar el pago ahora.",
    "received": "Pago recibido. Comprobando y enviando la tarjeta…",
    "retry": "Reintentar el pago",
    "wait": "Todavía puede llegar un pago.",
    "remaining": "Podrás reintentar en {time}.",
    "close": "Puedes cerrar esta ventana.",
    "copy": "Copiar detalles",
    "copied": "Detalles copiados",
    "select": "Selecciona y copia los detalles siguientes.",
    "detail": "Detalles técnicos (datos personales ocultos)",
    "expired": "Tarjeta caducada. No vuelvas a pagar. El remitente puede recuperar en Safari los fondos que lleguen tarde."
  },
  "de": {
    "unpaid": "Laut letzter Prüfung noch nicht bezahlt.",
    "unknown": "Die Zahlung kann gerade nicht geprüft werden.",
    "received": "Zahlung eingegangen. Die Karte wird geprüft und gesendet…",
    "retry": "Zahlung erneut versuchen",
    "wait": "Eine Zahlung könnte noch eingehen.",
    "remaining": "Neuer Versuch in {time} möglich.",
    "close": "Du kannst schließen.",
    "copy": "Details kopieren",
    "copied": "Details kopiert",
    "select": "Wähle und kopiere die folgenden Details.",
    "detail": "Technische Details (persönliche Daten ausgeblendet)",
    "expired": "Karte abgelaufen. Bezahle nicht erneut. Der Absender kann verspätete Zahlungen in Safari zurückholen."
  },
  "zh": {
    "unpaid": "根据上次检查，尚未付款。",
    "unknown": "暂时无法检查付款。",
    "received": "已收到付款。正在检查并发送卡片…",
    "retry": "重试付款",
    "wait": "付款仍可能到账。",
    "remaining": "{time} 后可重试。",
    "close": "你可以关闭此窗口。",
    "copy": "复制详情",
    "copied": "详情已复制",
    "select": "请选择并复制下面的详情。",
    "detail": "技术详情（已隐藏个人数据）",
    "expired": "卡片已过期。请勿再次付款。发件人可在 Safari 中取回延迟到账的资金。"
  }
};
const rowWords={
  "fr": {
    "started": "Démarrée à {time}.",
    "arrival": "Si tu as payé, le paiement peut apparaître dans quelques minutes. Rien de nouveau n’est débité sans confirmation dans ton portefeuille.",
    "busy": "Une demande de portefeuille est déjà en cours. Termine-la dans ton portefeuille ; le nouveau formulaire reste disponible.",
    "recoveryReady": "Lien prêt. Ouvre-le pour récupérer les fonds.",
    "openRecovery": "Ouvrir la récupération",
    "recoveryFailed": "Impossible de préparer la récupération. Réessaie sur cette ligne."
  },
  "en": {
    "started": "Started at {time}.",
    "arrival": "If you paid, the payment may appear in a few minutes. Nothing new is debited without approval in your wallet.",
    "busy": "A wallet request is already in progress. Finish it in your wallet; the new form remains available.",
    "recoveryReady": "Link ready. Open it to recover the funds.",
    "openRecovery": "Open recovery",
    "recoveryFailed": "Recovery could not be prepared. Try again on this row."
  },
  "es": {
    "started": "Iniciada a las {time}.",
    "arrival": "Si has pagado, el pago puede aparecer en unos minutos. No se debita nada nuevo sin confirmar en tu cartera.",
    "busy": "Ya hay una solicitud en la cartera. Termínala allí; el nuevo formulario sigue disponible.",
    "recoveryReady": "Enlace listo. Ábrelo para recuperar los fondos.",
    "openRecovery": "Abrir recuperación",
    "recoveryFailed": "No se pudo preparar la recuperación. Reintenta en esta fila."
  },
  "de": {
    "started": "Gestartet um {time}.",
    "arrival": "Falls du bezahlt hast, kann die Zahlung in einigen Minuten erscheinen. Ohne Bestätigung in deiner Wallet wird nichts erneut abgebucht.",
    "busy": "Eine Wallet-Anfrage läuft bereits. Beende sie in deiner Wallet; das neue Formular bleibt verfügbar.",
    "recoveryReady": "Link bereit. Öffne ihn, um das Guthaben zurückzuholen.",
    "openRecovery": "Rückholung öffnen",
    "recoveryFailed": "Die Rückholung konnte nicht vorbereitet werden. Versuche es in dieser Zeile erneut."
  },
  "zh": {
    "started": "于 {time} 开始。",
    "arrival": "如果你已付款，付款可能在几分钟后显示。未经钱包确认，不会再次扣款。",
    "busy": "钱包请求仍在进行。请先在钱包中完成；新建表单仍可填写。",
    "recoveryReady": "链接已准备好。打开即可取回资金。",
    "openRecovery": "打开资金恢复",
    "recoveryFailed": "无法准备资金恢复。请在此行重试。"
  }
};
for(const language of Object.keys(rowWords))Object.assign(retryWords[language],rowWords[language]);
const rt=key=>(retryWords[document.documentElement.lang]||retryWords.en)[key];
const ht=key=>senderHistoryText(document.documentElement.lang,key);
const cancelling=new Set();
let historyTimer=null,historyLoading=false;
const walletFailures=new Map();
let walletTimer=null,nativeRequestInFlight=false;

let fallbackCheckout=null,destroyed=false,handedOver=false;
const fallback=document.createElement('button');fallback.type='button';fallback.className='secondary-btn wire-nim-hub-fallback';fallback.hidden=true;
$('feedback').after(fallback);
let configured=false,ready=false,verified=false,busy=false,claimMode=false,active=null;
let receiveURL='',cooling=0,coolTimer=null;
// Page-local recipient state: no code or destination is stored in browser storage.
const reception={route:'',to:'',code:'',payout:null,known:false,locked:false,expired:false,message:'',timer:null,read:null,retry:false};
const receptionCode=()=>reception.code||$('claim-code').value;
const ct=key=>t('claim_'+key);
let paymentList=[];
let displayed=null;
let preparedContext=Boolean(embedded&&(initialPayment||scopedPaymentID));
let editingTransferredDraft=Boolean(embedded&&initialDraft&&!initialDraft.payment_id);
const completedForms=new Set();
const monitors=new Map(),checks=new Map(),controllers=new Set();
let progress=$('nim-progress');
if(!progress){
 progress=document.createElement('section');progress.id='nim-progress';progress.className='wire-nim-progress';progress.hidden=true;
 const message=document.createElement('p');message.id='nim-progress-message';message.setAttribute('role','status');message.setAttribute('aria-live','polite');
 const close=document.createElement('button');close.id='nim-progress-close';close.type='button';close.className='secondary-btn';close.hidden=true;
 const resend=document.createElement('button');resend.id='nim-progress-resend';resend.type='button';resend.className='secondary-btn';resend.hidden=true;
 progress.append(message,close,resend);root.querySelector('.workspace').before(progress);
}
// Built in the host flow, never in the mirrored payment module.
const walletPanel=document.createElement('section');walletPanel.id='nim-wallet-failure';walletPanel.hidden=true;
const walletNote=document.createElement('p');walletNote.id='nim-wallet-close-note';
const detailLabel=document.createElement('p');detailLabel.id='nim-wallet-detail-label';
const walletDetail=document.createElement('pre');walletDetail.id='nim-wallet-detail';walletDetail.tabIndex=0;
const copyDetail=document.createElement('button');copyDetail.id='nim-wallet-copy';copyDetail.type='button';copyDetail.className='secondary-btn';
const retryButton=document.createElement('button');retryButton.id='nim-wallet-retry';retryButton.type='button';retryButton.className='secondary-btn';
const retryWait=document.createElement('p');retryWait.id='nim-wallet-wait';retryWait.setAttribute('role','status');
const cancelButton=document.createElement('button');cancelButton.id='nim-wallet-cancel';cancelButton.type='button';cancelButton.className='secondary-btn';
const details=document.createElement('details');details.id='nim-wallet-details';const summary=document.createElement('summary');details.append(summary,detailLabel,walletDetail,copyDetail);
walletNote.setAttribute('role','status');walletPanel.append(retryButton,cancelButton,walletNote,retryWait,details);progress.append(walletPanel);
cancelButton.addEventListener('click',()=>{if(displayed)void cancelCard(displayed.id);});
copyDetail.addEventListener('click',async()=>{
 try{if(!navigator.clipboard?.writeText)throw Error('clipboard_unavailable');await navigator.clipboard.writeText(walletDetail.textContent);copyDetail.textContent=rt('copied');}
 catch{walletDetail.focus();const range=document.createRange();range.selectNodeContents(walletDetail);const selection=window.getSelection();selection.removeAllRanges();selection.addRange(range);copyDetail.textContent=rt('select');}
});
retryButton.addEventListener('click',retryPayment);
function diagnostic(error,p){
 const clean=(value,message=false)=>{
  if(typeof value!=='string'&&typeof value!=='number')return '-';
  let text=String(value).slice(0,2000);
  try{text=decodeURIComponent(text);}catch{}
  for(const secret of [p.sender,p.recipient,p.address,p.id,p.message]){
   if(typeof secret==='string'&&secret)text=text.split(secret).join('[redacted]');
  }
  text=text.replace(/(?:https?:\/\/|[a-z]+:\/\/)[^\s"'<>]+/gi,'[redacted]')
   .replace(/[^\s@]+@[^\s@]+\.[^\s@]+/g,'[redacted]')
   .replace(/NQ\d{2}(?:[ -]?[A-Z0-9]){32}/gi,'[redacted]')
   .replace(/\b(?:0x)?[A-Za-z0-9_-]{32,}\b/g,'[redacted]');
  if(message)text=text.replace(/\b\d+(?:[.,]\d+)?\b/g,'[number]');
  return text.replace(/[\u0000-\u001f\u007f-\u009f]/g,c=>'\\u'+c.charCodeAt(0).toString(16).padStart(4,'0')).slice(0,350);
 };
 const field=key=>{try{return clean(error?.[key],key==='message');}catch{return '-';}};
 return 'type='+field('type')+' | code='+field('code')+' | message='+field('message');
}
function renderWallet(p){
 clearTimeout(walletTimer);walletTimer=null;
 const failure=p&&walletFailures.get(p.id);walletPanel.hidden=!failure||p.stage==='sent';
 if(walletPanel.hidden){
  if(!document.hidden&&paymentList.some(p=>(walletFailures.get(p.id)?.waitUntil||0)>Date.now()))walletTimer=setTimeout(()=>renderProgress(),1000);
  return;
 }
 // The payment status already explains approval, delay and expiry; do not repeat it below.
 walletNote.textContent=recoveryStates.get(p.id)?.notice||'';walletNote.hidden=!walletNote.textContent;summary.textContent=ht('details');detailLabel.textContent=rt('detail');details.hidden=!failure.detail;
 cancelButton.textContent=ht('cancel');cancelButton.hidden=Boolean(scopedPaymentID)||!unpaidCard(p);cancelButton.disabled=cancelling.has(p.id)||busy||nativeRequestInFlight;
 if(walletDetail.textContent!==(failure.detail||''))walletDetail.textContent=failure.detail||'';
 walletDetail.hidden=!failure.detail;detailLabel.hidden=!failure.detail;copyDetail.hidden=!failure.detail;
 copyDetail.textContent=rt('copy');retryButton.textContent=rt('retry');
 const expired=failure.expired||Date.now()>=deadline(p),remaining=Math.max(0,Math.ceil(((failure.waitUntil||0)-Date.now())/1000));
 if(expired||failure.funded)fallback.hidden=true;
 retryButton.hidden=expired||failure.funded;retryButton.disabled=busy||nativeRequestInFlight||remaining>0;
 retryWait.hidden=!failure.tooSoon||(failure.waitUntil>0&&remaining===0);
 retryWait.textContent=failure.tooSoon?rt('wait')+(remaining?' '+rt('remaining').replace('{time}',Math.floor(remaining/60)+':'+String(remaining%60).padStart(2,'0')):''):'';
 if(!document.hidden&&!expired)walletTimer=setTimeout(()=>renderProgress(),remaining>0?1000:60000);
}
const read=key=>{try{return sessionStorage.getItem(key)||'';}catch{return '';}};
const save=(key,value)=>{try{sessionStorage.setItem(key,value);}catch{}};
const foldedIDs=new Set();
try{const ids=JSON.parse(localStorage.getItem('wire-nim-folded')||'[]');if(Array.isArray(ids))for(const id of ids)if(/^[a-f0-9]{48}$/.test(id))foldedIDs.add(id);}catch{}
const isFolded=p=>!scopedPaymentID&&foldedIDs.has(p.id);
function fold(p){if(!p||scopedPaymentID)return;foldedIDs.add(p.id);try{localStorage.setItem('wire-nim-folded',JSON.stringify([...foldedIDs]));}catch{}}
let freshForm=!scopedPaymentID&&read('wire-nim-new-form')==='yes',formEpoch=0;
const rowViews=new Map(),recoveryStates=new Map();
function rememberRecord(p){const index=paymentList.findIndex(x=>x.id===p.id);if(index<0)paymentList.push(p);else paymentList[index]=p;}
function pendingFailure(p){
 if(p.stage!=='funding_started'||walletFailures.has(p.id)||read('wire-nim-approved:'+p.id)==='yes')return;
 const at=Date.parse(p.started_at||'');
 walletFailures.set(p.id,{state:'checking',detail:'',waitUntil:Number.isFinite(at)?at+180000:0,tooSoon:Number.isFinite(at)});
}
function recoveryAllowed(p){return !scopedPaymentID&&recoverableCard(p);}
function renderHistory(){
 if(destroyed)return;
 const list=$('payment-list'),seen=new Set();
 for(const p of preparedContext?[]:historyEntries(paymentList)){
  seen.add(p.id);let view=rowViews.get(p.id);
  if(!view){
   const row=document.createElement('div');row.className='history-row';row.dataset.paymentId=p.id;
   const label=document.createElement('p'),state=document.createElement('p');state.className='nim-row-state';
   const actions=document.createElement('div');actions.className='nim-row-actions';
   const retry=document.createElement('button');retry.type='button';retry.className='secondary-btn nim-row-retry';retry.onclick=()=>void retryPayment(false,p.id);
   const create=document.createElement('button');create.type='button';create.className='secondary-btn nim-row-new';create.onclick=()=>newCard(p.id);
   const recover=document.createElement('button');recover.type='button';recover.className='secondary-btn nim-row-recover';recover.onclick=()=>void recoverCard(p.id);
   const link=document.createElement('a');link.className='nim-row-recovery-link';link.target='_blank';link.rel='noopener noreferrer';
   const resend=document.createElement('button');resend.type='button';resend.className='secondary-btn';resend.onclick=()=>resendCard(p.id);
   const notice=document.createElement('p');notice.className='nim-row-notice';notice.setAttribute('role','status');notice.setAttribute('aria-live','polite');
   const cancel=document.createElement('button');cancel.type='button';cancel.className='secondary-btn nim-row-cancel';cancel.onclick=()=>void cancelCard(p.id);
   actions.append(retry,cancel,recover,link,resend);row.append(label,state,actions,notice);
   view={row,label,state,retry,cancel,create,recover,link,resend,notice};rowViews.set(p.id,view);list.append(row);
  }
  const failure=walletFailures.get(p.id),recovery=recoveryStates.get(p.id)||{},wait=Math.max(0,Math.ceil(((failure?.waitUntil||0)-Date.now())/1000));
  view.label.textContent=p.amount+' NIM · '+p.recipient;
  view.state.textContent=nimHistoryLabel(p,document.documentElement.lang);
  if(wait>0)view.state.textContent+=' '+rt('remaining').replace('{time}',Math.floor(wait/60)+':'+String(wait%60).padStart(2,'0'));
  view.retry.textContent=rt('retry');view.retry.hidden=!unpaidCard(p)||failure?.funded;view.retry.disabled=busy||nativeRequestInFlight||wait>0;
  view.cancel.textContent=ht('cancel');view.cancel.hidden=Boolean(scopedPaymentID)||!unpaidCard(p);view.cancel.disabled=cancelling.has(p.id)||busy||nativeRequestInFlight;
  view.recover.textContent=recoveryAllowed(p)?t('recover'):'';view.recover.hidden=!recoveryAllowed(p)||Boolean(recovery.href);view.recover.disabled=Boolean(recovery.loading);
  view.link.hidden=!recoveryAllowed(p)||!recovery.href;if(recovery.href)view.link.href=recovery.href;else view.link.removeAttribute('href');view.link.textContent=rt('openRecovery');
  view.resend.textContent=p.stage==='delivery_failed'?pt('resend'):'';view.resend.hidden=p.stage!=='delivery_failed';view.resend.disabled=busy||checks.has(p.id);
  list.append(view.row);
  view.notice.textContent=recovery.loading?t('working'):recovery.error?rt('recoveryFailed'):recovery.href?rt('recoveryReady'):recovery.notice||'';
 }
 for(const[id,view]of rowViews)if(!seen.has(id)){view.row.remove();rowViews.delete(id);}
}
async function cancelCard(id){
 const p=paymentList.find(p=>p.id===id);
 if(!p||scopedPaymentID||!unpaidCard(p)||cancelling.has(id)||busy||nativeRequestInFlight||!window.confirm(ht('confirm')))return;
 cancelling.add(id);renderProgress();
 try{
  const result=await cancelUnpaidCard(p,(path,data)=>api(path,data,{timeout:true}));
  rememberRecord(result.payment);walletFailures.delete(id);stopMonitor(monitor(result.payment));
  if(result.received){recoveryStates.set(id,{notice:ht('arrived')});observe(result.payment);}
  else if(displayed?.id===id)newCard(id);
  onStatus(result.payment,result.received?ht('arrived'):'');
  await loadHistory();
 }catch(error){
  recoveryStates.set(id,{notice:ht(error.message==='cannot_cancel'?'cannot':'failed')});
  $('nim-progress-message').textContent=ht(error.message==='cannot_cancel'?'cannot':'failed');
  try{await loadHistory();}catch{}
 }finally{cancelling.delete(id);renderProgress();}
}
function scheduleHistory(){
 clearTimeout(historyTimer);historyTimer=null;
 if(!destroyed&&verified&&!document.hidden&&paymentList.some(p=>historyPending(p)||recoverableCard(p)))historyTimer=setTimeout(async()=>{
  if(historyLoading)return;historyLoading=true;
  try{await loadHistory();}catch{}finally{historyLoading=false;scheduleHistory();}
 },6000);
}
async function recoverCard(id){
 const p=paymentList.find(p=>p.id===id);if(!p||!recoveryAllowed(p)||recoveryStates.get(id)?.loading)return;
 recoveryStates.set(id,{loading:true});renderHistory();
 try{const result=await api('/api/payments/'+id+'/recovery',{},{timeout:true});recoveryStates.set(id,{href:validatedCashlink(result.cashlink)});}
 catch{recoveryStates.set(id,{error:true});}
 finally{renderHistory();}
}
function newCard(id=displayed?.id){
 if(scopedPaymentID)return;
 formEpoch++;
 for(const p of paymentList)if(p.id===id||p.id===displayed?.id)fold(p);
 fold(displayed);fold(active);
 requestID=nextID();save('wire-request-id',requestID);save('wire-nim-new-form','yes');
 freshForm=true;editingTransferredDraft=false;preparedContext=false;root.classList?.remove('wire-nim-prepared');
 active=null;displayed=null;save('wire-pending-payment','');save('wire-pending-wallet','');save('wire-form','');
 for(const key of ['recipient','amount','message']){$(key).readOnly=false;$(key).value='';}
 $('currency').disabled=false;$('currency').value='NIM';
 $('preview-empty').hidden=false;$('payment-card').hidden=true;$('preview-title').hidden=true;
 if($('nim-handoff-card')?.contains($('payment-card')))root.querySelector('.preview-area').append($('payment-card'));
 $('card-form').append($('feedback'),fallback);fallback.hidden=true;fallbackCheckout=null;
 $('feedback').textContent=busy?rt('busy'):'';
 renderProgress();renderHistory();renderDraftPreview();$('recipient').focus();
}
let requestID=read('wire-request-id');
function nextID(){const bytes=crypto.getRandomValues(new Uint8Array(16));return Array.from(bytes,b=>b.toString(16).padStart(2,'0')).join('');}
if(!/^[a-f0-9]{32}$/.test(requestID)){requestID=nextID();if(embedded||!new URLSearchParams(location.search).has('claim'))save('wire-request-id',requestID);}
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
function explain(error){if(error.message==='invite_only')return t('inviteOnly');if(error.message==='funding_pending'||error.message==='funding_already_started')return t('pending');if(['claim_unavailable','claim_expired','card_expired'].includes(error.message))return t('ended');if(error.status===403)return t('invalid');if(error.status===410)return t('expired');if(error.status===401)return t('verifyFirst');return t('failed');}
function refreshText(){
 paintSenderAccess();adminPanel?.render();
 if(senderInviteOnly)status('sender-status','inviteOnly');
 fallback.textContent=wt('hub');
 root.querySelectorAll('[data-wire]').forEach(el=>el.textContent=t(el.dataset.wire));
 if(configured&&!claimMode){$('preview-submit').textContent=t(active&&active.stage!=='ready'?'retry':'pay');$('notice').textContent=t(ready?'note':'unavailable');root.querySelector('.security').textContent=t('security');root.querySelector('.lock').textContent=cardPresentation(document.documentElement.lang).locked;}
 if(cooling)updateClaimControls();
 if(claimMode)updateClaimControls();
 renderProgress();
 renderDraftPreview();
}
function renderDraftPreview(){
 if(claimMode||preparedContext||active||displayed||$('currency').value!=='NIM')return;
 const amount=$('amount').value.trim().replace(',','.');
 const valid=/^\d{1,7}(?:\.\d{1,5})?$/.test(amount)&&Number(amount)>0&&Number(amount)<=1000;
 $('preview-empty').hidden=valid;$('payment-card').hidden=!valid;$('preview-title').hidden=!valid;
 if(!valid)return;
 const words=paintCard($('payment-card'),document.documentElement.lang,requestID);
 $('card-amount').textContent=amount;$('card-unit').textContent='NIM';$('card-network').hidden=true;
 $('card-message').textContent=$('message').value;$('card-recipient').textContent=$('recipient').value;
 root.querySelector('.card-heading').textContent='Nimiq · '+words.heading;root.querySelector('.sample').textContent=words.protected;root.querySelector('.lock').textContent=words.locked;
 if(!$('coin').firstChild){const icon=document.createElement('img');icon.src='/brand/nimiq-logo.svg';icon.alt='Nimiq';icon.width=62;icon.height=62;$('coin').append(icon);}
}
for(const id of ['recipient','amount','message'])$(id).addEventListener('input',renderDraftPreview);
window.addEventListener('wire-language',refreshText);
async function task(run,target='feedback',cleanup){
 if(busy){$(target).textContent=rt('busy');return;}busy=true;status(target,'working');
 renderProgress();
 try{await run();}catch(error){$(target).textContent=explain(error);}finally{busy=false;cleanup?.();renderProgress();}
}
let usdtAssets=[];
function senderUI(email,admin=false){verified=Boolean(email);window.wireUsdtState={assets:usdtAssets,isVerified:verified};if(!embedded)window.WireUSDT?.configure(window.wireUsdtState);$('sender').value=email||$('sender').value;$('sender').readOnly=verified;$('send-sender-code').hidden=verified;$('logout').hidden=!verified;$('sender-code-box').hidden=true;if(verified)status('sender-status','verified');configureAdmin(admin);paintSenderAccess();}
function showPayment(p){
 if(isFolded(p))return;
 displayed=p;
 if(embedded&&p.stage==='sent'){
  if(!completedForms.has(p.id)){completedForms.add(p.id);requestID=nextID();save('wire-request-id',requestID);save('wire-form','');}
  for(const id of ['recipient','amount','message']){$(id).value='';$(id).readOnly=false;}
  $('currency').disabled=false;
  renderProgress();return;
 }
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
 const data=await api('/api/payments',undefined,{timeout:true});$('my-payments').hidden=Boolean(scopedPaymentID);
 paymentList=scopedPaymentID?data.payments.filter(p=>p.id===scopedPaymentID):data.payments;
 const current=paymentList.find(p=>p.id===displayed?.id);
 if(current?.stage==='cancelled'&&!scopedPaymentID)newCard(current.id);
 for(const p of paymentList)pendingFailure(p);
 const visible=paymentList.filter(p=>!isFolded(p)&&p.stage!=='cancelled');
 active=preparedContext?visible.find(p=>p.id===(initialPayment?.id||scopedPaymentID))||null:freshForm||editingTransferredDraft?null:visible.find(p=>p.id===displayed?.id)||visible.find(p=>p.stage!=='sent')||null;
 const pending=read('wire-pending-payment');
 const shown=preparedContext?active:freshForm||editingTransferredDraft?null:visible.find(p=>p.id===pending)||visible.find(p=>p.id===displayed?.id)||active||(!embedded?visible[0]:null);
 for(const p of paymentList){const watch=monitors.get(p.id);if(watch){watch.payment=p;if(p.stage==='sent'||p.stage==='cancelled')stopMonitor(watch);}}
 if(shown)showPayment(shown);
 renderHistory();refreshText();scheduleHistory();
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
 const watch=monitors.get(p.id),failure=walletFailures.get(p.id);
 if(watch?.note&&['offline','session'].includes(watch.note)&&p.stage!=='sent')return pt(watch.note);
 if(failure&&p.stage!=='sent'){
  if(failure.expired||Date.now()>=deadline(p))return rt('expired');
  if(failure.funded)return rt('received');
  if(failure.state==='unfunded'){
   const at=Date.parse(p.started_at||'');
   const started=Number.isFinite(at)?rt('started').replace('{time}',new Intl.DateTimeFormat(document.documentElement.lang,{hour:'2-digit',minute:'2-digit',hourCycle:'h23'}).format(at)):'';
   return rt('unpaid')+(started?'\n'+started:'');
  }
  return rt('unknown');
 }
 if(p.stage==='sent'){const date=expiryDate(Date.parse(p.created||'')+86400000);return pt('sent').replace('{email}',p.recipient)+(date?'\n'+t('validUntil').replace('{date}',date):'');}
 if(Date.now()>=(watch?.until??deadline(p)))return pt('expired');
 if(p.stage==='delivery_failed')return pt('failed');
 if(watch?.note)return pt(watch.note)||watch.note;
 if(['funded','delivery_pending'].includes(p.stage))return pt('mail').replace('{email}',p.recipient);
 return pt(read('wire-nim-approved:'+p.id)==='yes'?'waiting':'checking');
}
function renderProgress(){
 if(destroyed)return;
 const p=displayed,show=p&&(['funding_started','funded','delivery_pending','delivery_failed','sent'].includes(p.stage)||walletFailures.has(p.id));
 renderWallet(p);
 $('new-card').hidden=Boolean(scopedPaymentID)||!verified||embedded&&p?.stage==='sent';
 if(show)progress.append($('new-card'));else root.querySelector('.workspace').before($('new-card'));
 $('preview-submit').disabled=busy||nativeRequestInFlight;
 renderHistory();
 const complete=embedded&&p?.stage==='sent';
 const newlyComplete=complete&&!root.classList.contains('wire-nim-complete');
 if(embedded)root.classList.toggle('wire-nim-complete',complete);
 if(embedded){
  root.querySelector('.workspace').hidden=complete||preparedContext;$('notice').hidden=complete||preparedContext;$('my-payments').hidden=complete||preparedContext;
  if($('nim-handoff-summary')){
   $('nim-handoff-summary').hidden=!preparedContext||complete;
   const pay=$('nim-handoff-pay');pay.textContent=t('pay');pay.disabled=busy;
   pay.hidden=!p||p.stage!=='ready'||Date.now()>=deadline(p);
   if(preparedContext&&p&&Date.now()>=deadline(p)&&p.stage!=='sent')$('feedback').textContent=pt('expired');
  }
  root.closest('.wire-nim-window')?.querySelectorAll(':scope>.nimiq-payment-head').forEach(el=>el.hidden=complete);
  root.closest('.wire-nim-window')?.querySelectorAll(':scope>.wire-nim-assets').forEach(el=>el.hidden=true);
 }
 progress.hidden=!show;if(!claimMode)$('preview-submit').hidden=Boolean(show);if(!show)return;
 const text=progressText(p);$('nim-progress-message').textContent=text;
 $('nim-progress-close').textContent=walletFailures.has(p.id)&&p.stage!=='sent'?(progressWords[document.documentElement.lang]||progressWords.en).close:pt('close');$('nim-progress-close').hidden=!walletFailures.has(p.id)&&p.stage!=='sent'&&Date.now()<deadline(p);
 $('nim-progress-resend').textContent=pt('resend');$('nim-progress-resend').hidden=p.stage!=='delivery_failed'||Date.now()>=deadline(p);
 $('nim-progress-resend').disabled=checks.has(p.id)||busy;
 $('preview-submit').hidden=true;
 if(newlyComplete&&root.isConnected)$('nim-progress-close').focus();
 onStatus(p,text);
}
function observe(p){
 if(destroyed)return;
 const watch=monitor(p);watch.note='';
 if(['funded','delivery_pending','delivery_failed','sent'].includes(p.stage))walletFailures.delete(p.id);
 if(p.stage==='sent'){
  stopMonitor(watch);fallback.hidden=true;fallbackCheckout=null;
  if(read('wire-pending-payment')===p.id){save('wire-pending-payment','');save('wire-pending-wallet','');save('wire-form','');}
 }
 if(!isFolded(p)&&(!freshForm||displayed?.id===p.id)&&(!displayed||displayed.id===p.id||read('wire-pending-payment')===p.id))showPayment(p);
 else onStatus(p,progressText(p));
}
function confirm(id,resend=false){
 if(checks.has(id))return checks.get(id);
 const promise=(async()=>{
  await loadHistory();
  const p=paymentList.find(p=>p.id===id);if(!p)throw Error('payment_unavailable');
  observe(p);
  if(p.stage==='cancelled'||p.claim_state==='claimed'||p.claim_state==='claiming'||p.stage==='sent'||!resend&&['delivery_pending','delivery_failed'].includes(p.stage))return {payment:p};
  if(Date.now()>=deadline(p))throw Object.assign(Error('claim_expired'),{status:410});
  if(resend&&p.stage!=='delivery_failed')return {payment:p};
  await ensureArtwork(p);
  try{const result=await api(`/api/payments/${id}/${resend?'resend':'confirm'}`,{},{timeout:true});observe(result.payment);await loadHistory();if(displayed?.id===result.payment.id)$('feedback').textContent=progressText(result.payment);return result;}
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
  catch(error){const f=walletFailures.get(watch.payment.id);if(f&&error.message==='funding_pending'&&!f.funded)f.state='unfunded';watch.note=error.status===401?'session':error.message==='funding_pending'?'':error.status===410?'expired':'offline';if(error.status===410)stopMonitor(watch);}
  finally{watch.running=null;if(!destroyed){renderProgress();schedule(watch);}}
 })();return watch.running;
}
async function poll(id){
 const p=paymentList.find(p=>p.id===id)||active;
 if(!p||p.id!==id)return;
 const watch=monitor(p);if(p.stage==='sent'){observe(p);return;}
 await runMonitor(watch);
}
function approve(p){save('wire-nim-approved:'+p.id,'yes');observe(p);if(displayed?.id===p.id)$('feedback').textContent=progressText(p);}
function resume(){
 if(claimMode){if(!document.hidden)void refreshPayout();return;}
 if(destroyed||claimMode||document.hidden||!verified||editingTransferredDraft)return;
 renderProgress();scheduleHistory();
 for(const p of paymentList){if(['funding_started','funded','delivery_pending','delivery_failed'].includes(p.stage))void runMonitor(monitor(p));}
}
function visibility(){if(document.hidden){clearTimeout(historyTimer);clearTimeout(walletTimer);clearTimeout(reception.timer);for(const watch of monitors.values()){clearTimeout(watch.timer);watch.timer=null;}}else resume();}
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
 if(!$('sender').reportValidity())return;
 senderInviteOnly=false;
 try{await api('/api/sender/request',{email:$('sender').value,language:document.documentElement.lang});}
 catch(error){if(error.message==='invite_only'){senderInviteOnly=true;$('sender-code-box').hidden=true;$('sender-code').value='';$('sender').readOnly=false;}throw error;}
 $('sender-code-box').hidden=false;status('sender-status','codeSent');
},'sender-status'));
$('verify-sender-code').addEventListener('click',()=>task(async()=>{
 const result=await api('/api/sender/verify',{code:$('sender-code').value});$('sender-code').value='';senderUI(result.email);void api('/api/session').then(session=>configureAdmin(session.admin===true)).catch(()=>{});await loadHistory();
},'sender-status'));
$('logout').addEventListener('click',()=>task(async()=>{await api('/api/logout',{});location.reload();}));
$('new-card').addEventListener('click',()=>newCard());
function nativeWallet(){return typeof window.nimiq?.sendBasicTransactionWithData==='function';}
function rememberPayment(p,wallet){
 save('wire-pending-payment',p.id);save('wire-pending-wallet',wallet);
 save('wire-form',JSON.stringify({sender:$('sender').value,recipient:p.recipient,amount:p.amount,message:p.message,currency:'NIM',payment:p.id}));
 onDraftUsed();
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
fallback.addEventListener('click',()=>{
 if(busy||!fallbackCheckout)return;
 void retryPayment(true);
});

// Preparing a handoff never starts funding or opens a wallet.
let handoffPreparation=null;
function prepareHandoff(){
 if(handoffPreparation)return handoffPreparation;
 const pending=prepareHandoffPayment();handoffPreparation=pending;
 pending.finally(()=>{handoffPreparation=null;}).catch(()=>{});
 return pending;
}
function useHubAfterPreparation(){
 const open=()=>{if(!destroyed)fundNim(true);};
 if(handoffPreparation)handoffPreparation.finally(open).catch(()=>{});else open();
}
async function prepareHandoffPayment(){
 const epoch=formEpoch;
 if(busy||!verified||scopedPaymentID)throw Error('sender_verification_required');
 busy=true;renderProgress();
 try{
  let p=active;
  if(preparedContext&&!p)throw Error('payment_unavailable');
  if(!p){const result=await api('/api/payments',{recipient:$('recipient').value,amount:$('amount').value.trim().replace(',','.'),currency:'NIM',message:$('message').value,request_id:requestID,language:document.documentElement.lang},{timeout:true});p=result.payment;active=p;}
  if(p.stage==='preparing'){const result=await api(`/api/payments/${p.id}/prepare`,{},{timeout:true});p=result.payment;active=p;}
  rememberRecord(p);
  if(epoch!==formEpoch){if(active?.id===p.id)active=null;fold(p);renderHistory();throw Error('view_changed');}
  freshForm=false;save('wire-nim-new-form','');
  showPayment(p);
  if(p.stage!=='ready')throw Object.assign(Error('payment_not_fundable'),{status:409});
  handedOver=true;
  return p;
 }finally{busy=false;renderProgress();}
}


async function requestWallet(p,started,forceHub=false){
 const native=!forceHub&&nativeWallet();
 p=started.payment;rememberRecord(p);
 if(!isFolded(p)&&!freshForm)active=p;
  // Hub's checkout extraData becomes the chain transaction's recipientData.
  const checkout={appName:started.checkout.appName,recipient:started.checkout.recipient,value:started.checkout.value,extraData:new Uint8Array(started.checkout.recipientData)};
  if(native){
   // The host API accepts text, not bytes. Its UTF-8 encoding is 00c282c280c292c287.
   // The server must accept that exact marker as well as the Hub's original 0082809287.
   const marker=[0,130,128,146,135];
   if(!Number.isSafeInteger(checkout.value)||checkout.value<=0||!addressValid(checkout.recipient)||started.checkout.recipientData?.length!==marker.length||!marker.every((b,i)=>started.checkout.recipientData[i]===b))throw Error('invalid_checkout');
   rememberPayment(p,'pay');
   let failure;
   nativeRequestInFlight=true;renderProgress();
   try{
    const result=await window.nimiq.sendBasicTransactionWithData({recipient:checkout.recipient,value:checkout.value,data:'\u0000\u0082\u0080\u0092\u0087'});
    // Direct providers can return the older { error } envelope instead of rejecting.
    if(result?.error)throw result.error;
   }catch(error){failure=error||{};}finally{nativeRequestInFlight=false;}
   if(failure){
    const detail=diagnostic(failure,p);
    console.warn('Nimiq Pay request failed:',detail);
    const failed={detail,state:'checking',waitUntil:0};
    walletFailures.set(p.id,failed);
    const refused=failure.type==='PERMISSION_DENIED'||failure.code===4001||failure.name==='PermissionDeniedError';
    try{await confirm(p.id);return;}
    catch(error){failed.state=error.message==='funding_pending'?'unfunded':'unknown';failed.expired=error.status===410;}
    fallbackCheckout={p,checkout};fallback.hidden=refused||isFolded(p)||displayed?.id!==p.id;
    if(!failed.expired){const watch=monitor(p);schedule(watch);}
    if(displayed?.id===p.id)$('feedback').textContent=refused?wt('cancelled'):'';
    renderProgress();return;
   }
   walletFailures.delete(p.id);
   // The returned hash is not evidence of funding; the existing server check decides.
   approve(p);await poll(p.id);return;
  }
  await hubCheckout(p,checkout);
}
async function retryPayment(forceHub=false,paymentID=displayed?.id){
 const id=typeof paymentID==='string'?paymentID:displayed?.id;
 const retryCard=paymentList.find(p=>p.id===id);
 if(!id||scopedPaymentID&&id!==scopedPaymentID||!retryCard||!unpaidCard(retryCard))return;
 if(!walletFailures.has(id))walletFailures.set(id,{state:'checking',detail:''});
 if(busy||nativeRequestInFlight){const state=recoveryStates.get(id)||{};state.notice=rt('busy');recoveryStates.set(id,state);renderHistory();return;}
 void task(async()=>{
  let failure=walletFailures.get(id);
  if(!failure)return;
  try{await confirm(id);return;}
  catch(error){
   if(error.status===410){failure.expired=true;return;}
   if(!['funding_pending','funding_not_started'].includes(error.message)){failure.state='unknown';return;}
  }
  const p=paymentList.find(p=>p.id===id);
  if(!p||['funded','delivery_pending','delivery_failed','sent'].includes(p.stage))return;
  let result;
  try{result=await api(`/api/payments/${id}/checkout`,{},{timeout:true});}
  catch(error){
   if(error.message==='retry_too_soon'){
    await loadHistory();failure.tooSoon=true;failure.state='unfunded';
    const started=Date.parse(paymentList.find(p=>p.id===id)?.started_at||'');failure.waitUntil=Number.isFinite(started)?started+180000:0;return;
   }
   if(error.message==='already_funded'){
    failure.funded=true;await poll(id);return;
   }
   if(error.message==='funding_not_started'){
    try{result=await api(`/api/payments/${id}/start`,{},{timeout:true});}
    catch(startError){if(startError.message==='funding_already_started'){await poll(id);return;}if(startError.status===410){failure.expired=true;return;}throw startError;}
   }else if(error.status===410){failure.expired=true;return;}
   else{failure.state='unknown';return;}
  }
  if(forceHub===true&&!window.confirm(wt('confirm')))return;
  walletFailures.delete(id);fallback.hidden=true;fallbackCheckout=null;
  await requestWallet(p,result,forceHub===true);
 });
}

function fundNim(forceHub=false){
 if(busy){$('feedback').textContent=rt('busy');return;}
 if(active&&isFolded(active))active=null;
 if(displayed?.stage==='sent')return;
 const epoch=formEpoch;
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
  if(preparedContext&&!p)throw Error('payment_unavailable');
  if(!p){const result=await api('/api/payments',{recipient:$('recipient').value,amount:$('amount').value.trim().replace(',','.'),currency:'NIM',message:$('message').value,request_id:requestID,language:document.documentElement.lang},{timeout:true});p=result.payment;active=p;}
  rememberRecord(p);
  if(epoch!==formEpoch){if(active?.id===p.id)active=null;fold(p);onStatus(p,progressText(p));renderHistory();return;}
  freshForm=false;save('wire-nim-new-form','');editingTransferredDraft=false;
  showPayment(p);
  await ensureArtwork(p);
  if(p.stage!=='ready'){await confirm(p.id);return;}
  // No unprotected fallback. No seed is present in this checkout request.
  if(!native&&!window.HubApi)throw Error('wallet_unavailable');
  if(native&&window.nimiqPay?.network&&window.nimiqPay.network!=='mainnet'){$('feedback').textContent=wt('network');return;}
  // A ready UI can be stale (the other browser may already have started).
  // Confirm first; the service reports funding_not_started for a still-ready card.
  // Any other failure blocks the wallet. The service's atomic start guards the race.
  if(preparedContext||handedOver){
   try{await confirm(p.id);return;}catch(error){if(error.message!=='funding_not_started'||active?.stage!=='ready')throw error;}
   if(active?.stage!=='ready')return;
  }
  let started;
  try{started=await api(`/api/payments/${p.id}/start`,{},{timeout:true});}
  catch(error){if(error.message==='funding_already_started'){await loadHistory();await poll(p.id);return;}throw error;}
  if(epoch!==formEpoch){fold(started.payment);rememberRecord(started.payment);void poll(p.id);return;}
  await requestWallet(p,started,forceHub);
 },'feedback',()=>{if(popup&&!popup.closed)popup.close();});
}
$('card-form').addEventListener('submit',event=>{
 if(configured&&window.WireUSDT?.enabled($('currency').value)){event.preventDefault();window.WireUSDT.submit();return;}
 if(!configured)return;event.preventDefault();
 if(busy)return;
 if(verified&&ready&&$('currency').value==='NIM'&&(!active||active.stage==='ready')&&!nativeWallet()&&!window.nimiqPay&&chooseWallet){chooseWallet(useHubAfterPreparation,prepareHandoff);return;}
 fundNim();
});
$('nim-handoff-pay')?.addEventListener('click',()=>fundNim());
function updateClaimControls(){
 if(!claimMode)return;
 const locked=reception.locked||Boolean(reception.payout),available=reception.known&&!reception.expired&&!locked;
 root.querySelectorAll('[data-claim]').forEach(el=>el.textContent=ct(el.dataset.claim));
 const validCode=/^[0-9]{6}$/.test(receptionCode());
 $('claim-choices').hidden=locked||reception.expired;
 $('claim-choices').disabled=busy||!available||(!validCode&&!receiveURL);
 $('claim-pay').hidden=Boolean(receiveURL);
 $('claim-pay').disabled=busy||!available||!validCode;
 $('claim-wallet').disabled=busy||!available||(!validCode&&!receiveURL);
 $('request-claim-code').hidden=!available||Boolean(receiveURL);
 $('request-claim-code').disabled=busy||cooling>0;
 $('request-claim-code').textContent=cooling?ct('resend')+' · '+cooling+' s':ct('resend');
 $('claim-code').closest('label').hidden=!available||Boolean(receiveURL)||Boolean(reception.code);
 $('claim-code').disabled=busy||!available||Boolean(receiveURL);
 $('claim-code').required=available&&!receiveURL;
 $('claim-retry').hidden=!reception.retry;$('claim-retry').disabled=busy||Boolean(reception.read);
 $('claim-payout-details').hidden=!reception.payout;
 if(reception.payout){const p=reception.payout;
  $('claim-payout-details').textContent=ct('details').replace('{value}',new Intl.NumberFormat(document.documentElement.lang,{maximumFractionDigits:5}).format(p.value/100000)).replace('{to}',p.to).replace('{hash}',p.hash||'').replace('{confirmations}',p.confirmations||0);
 }
 $('claim-status').textContent=reception.message?t(reception.message):reception.payout?ct(reception.payout.status):'';
 $('claim-status').className=reception.payout?.status==='payout_confirmed'?'ok':reception.retry||reception.payout?.status==='payout_failed'?'error':'';
}
function cooldown(){clearInterval(coolTimer);cooling=20;updateClaimControls();coolTimer=setInterval(()=>{cooling--;updateClaimControls();if(cooling<=0)clearInterval(coolTimer);},1000);}
function schedulePayout(){
 clearTimeout(reception.timer);
 if(!destroyed&&!document.hidden&&(reception.payout?.status==='payout_pending'||!reception.known&&!reception.expired))reception.timer=setTimeout(()=>void refreshPayout(),3000);
}
function showPayout(p){
 if(!['payout_pending','payout_confirmed','payout_failed'].includes(p.status)||typeof p.to!=='string'||!Number.isSafeInteger(p.value)||p.value<0)throw Error('invalid_payout_response');
 reception.payout=p;reception.to=p.to;reception.locked=true;reception.known=true;reception.message='';reception.retry=p.status==='payout_pending';
 reception.code='';$('claim-code').value='';clearInterval(coolTimer);cooling=0;
 updateClaimControls();schedulePayout();
}
function refreshPayout(){
 if(destroyed)return Promise.resolve();
 if(reception.read)return reception.read;
 reception.read=(async()=>{
  try{showPayout(await api('/api/claims/'+claimID()+'/payout_status',{},{timeout:true}));}
  catch(error){
   if(error.status===404&&error.message==='payout_not_started'){
    reception.known=true;if(!reception.payout&&reception.message!=='claim_payout_already_started')reception.locked=false;
    if(!reception.locked){reception.retry=false;if(['claim_checkingPayout','claim_payoutUnavailable'].includes(reception.message))reception.message='';}
   }else if(['claim_unavailable','claim_expired','card_expired'].includes(error.message)){reception.expired=true;reception.message='ended';}else{reception.message='claim_payoutUnavailable';reception.retry=true;if(!reception.payout)reception.known=false;}
  }finally{reception.read=null;updateClaimControls();schedulePayout();}
 })();
 return reception.read;
}
async function claimTask(run,button){
 if(busy)return;busy=true;button.setAttribute('aria-busy','true');updateClaimControls();
 try{await run();}
 catch(error){
  const code=error.message;
  if(['claim_unavailable','claim_expired','card_expired'].includes(code)){reception.expired=true;reception.message='ended';}
  else if(code==='payout_already_started'){
   reception.locked=true;reception.message='claim_payout_already_started';reception.retry=true;await refreshPayout();
  }else if(code==='invalid_destination'||code==='cashlink_not_confirmed_or_invalid'){
   reception.locked=false;reception.message='claim_'+code;
  }else if(error.status===403){reception.code='';$('claim-code').value='';reception.locked=false;reception.message='invalid';}
  else if(error.status===410){reception.expired=true;reception.message='ended';}
  else{await refreshPayout();if(!reception.payout){reception.message='claim_payoutUnavailable';reception.retry=true;}}
 }finally{busy=false;button.removeAttribute('aria-busy');updateClaimControls();}
}
async function requestClaimCode(){
 if(!reception.known||reception.locked||reception.expired||receiveURL||cooling>0)return;
 try{await api('/api/claims/'+claimID()+'/request',{},{timeout:true});reception.code='';$('claim-code').value='';reception.message='codeSent';cooldown();}
 catch(error){if(error.status===429){cooldown();reception.message='waitCode';return;}throw error;}
}
function openReleasedWallet(){
 if(mobile())window.open(receiveURL,'_self','noopener,noreferrer');
 else window.open(receiveURL,'_blank','noopener,noreferrer');
}
function chooseReception(route){
 if(busy||!reception.known||reception.locked||reception.payout||reception.expired)return;
 if(route==='wallet'&&receiveURL){openReleasedWallet();return;}
 if(receiveURL||!/^[0-9]{6}$/.test(receptionCode()))return;
 if(route==='pay'&&!window.nimiq){
  return claimTask(async()=>{
   const code=receptionCode();
   const result=await api('/api/claims/'+claimID()+'/handoff',{code},{timeout:true});
   if(typeof result.token!=='string'||!/^[A-Za-z0-9_-]{43}$/.test(result.token))throw Error('invalid_handoff');
   // Only the opaque, short-lived right crosses browser contexts, never the code itself.
   const target='https://wire.phoenixmail.ca/?claim='+claimID()+'&h='+encodeURIComponent(result.token);
   reception.code=code;$('claim-code').value='';
   try{window.open('nimiqpay://miniapp?url='+encodeURIComponent(target),'_self','noopener,noreferrer');}catch{}
   reception.message='claim_openPay';
  },$('claim-pay'));
 }
 return claimTask(async()=>{
  reception.route=route;
  if(route==='pay'){
   try{
    const accounts=await window.nimiq.listAccounts();
    if(accounts?.error)throw accounts.error;
    const first=Array.isArray(accounts)?accounts[0]:null;
    if(typeof first!=='string'||!softAddress(first))throw Error('no_accounts');
    reception.to=first;
   }catch(error){reception.message='claim_'+(error?.type==='PERMISSION_DENIED'||error?.code===4001?'accountsCancelled':'accountsUnavailable');return;}
  }
  await submitClaim();
 },$('claim-'+route));
}
function softAddress(value){return /^NQ[0-9]{2}[0-9A-Z]{32}$/.test(value.replace(/\s/g,'').toUpperCase());}
$('claim-pay')?.addEventListener('click',()=>chooseReception('pay'));
$('claim-wallet')?.addEventListener('click',()=>chooseReception('wallet'));
$('claim-retry')?.addEventListener('click',()=>claimTask(async()=>{
 await refreshPayout();
 // Resume only the server's locked destination, without consuming another code.
 if(reception.payout?.status==='payout_pending')showPayout(await api('/api/claims/'+claimID()+'/payout',{to:reception.payout.to},{timeout:true}));
 else if(!reception.payout&&reception.known&&!reception.locked&&/^[0-9]{6}$/.test(receptionCode()))await submitClaim();
},$('claim-retry')));
$('claim-code').addEventListener('input',()=>{$('claim-code').value=$('claim-code').value.replace(/\D/g,'').slice(0,6);updateClaimControls();});
$('request-claim-code').addEventListener('click',()=>claimTask(requestClaimCode,$('request-claim-code')));
// Enter must not silently choose one of the two wallets.
$('claim-form').addEventListener('submit',event=>event.preventDefault());
async function submitClaim(){
 if(!reception.route||!reception.known||reception.locked||reception.expired||!/^[0-9]{6}$/.test(receptionCode()))return;
 if(reception.route==='pay'){
  if(!softAddress(reception.to)){reception.message='claim_invalid_destination';return;}
  reception.locked=true;updateClaimControls();
  showPayout(await api('/api/claims/'+claimID()+'/payout',{code:receptionCode(),to:reception.to},{timeout:true}));return;
 }
 const result=await api('/api/claims/'+claimID()+'/release',{code:receptionCode()},{timeout:true});
 receiveURL=validatedCashlink(result.cashlink);reception.code='';$('claim-code').value='';
 clearInterval(coolTimer);cooling=0;reception.message='accepted';
}
function claimID(){const id=new URLSearchParams(location.search).get('claim');if(!/^[a-f0-9]{48}$/.test(id||''))throw Error('claim_unavailable');return id;}
async function boot(){
 refreshText();claimMode=!embedded&&new URLSearchParams(location.search).has('claim');
 if(claimMode){
  const parameters=new URLSearchParams(location.search),hasHandoff=parameters.has('h');
  let handoffToken=parameters.get('h')||'';
  if(hasHandoff){
   // Erase before any network request, including failed/used-token recovery and reloads.
   parameters.delete('h');
   history.replaceState(history.state,'',location.pathname+(parameters.size?'?'+parameters.toString():'')+location.hash);
  }
  root.querySelector('.workspace').hidden=true;$('notice').hidden=true;$('claim-panel').hidden=false;
  reception.message='claim_checkingPayout';updateClaimControls();await refreshPayout();
  if(hasHandoff){
   try{
    const result=await api('/api/claims/'+claimID()+'/handoff/redeem',{token:handoffToken},{timeout:true});
    if(!reception.payout&&typeof result.code==='string'&&/^[0-9]{6}$/.test(result.code))reception.code=result.code;
   }catch{/* A used or expired right is a normal code-entry screen, not an alarm. */}
   finally{handoffToken='';}
  }
  try{const p=await api('/api/claims/'+claimID());paintCard($('claim-card'),document.documentElement.lang,p.id);$('claim-info').textContent=t('from')+' '+p.sender;$('claim-amount').textContent=p.amount+' NIM';$('claim-message').textContent=p.message;const expires=expiryDate(p.expires_at);claimExpiry.textContent=expires?t('expires').replace('{date}',expires):'';claimExpiry.hidden=!expires;
   window.PhoenixNimiqIdenticons?.paint($('claim-iqon'),p.address);
   if(new Date(p.expires_at).getTime()<=Date.now()&&!reception.payout){reception.expired=true;reception.message='ended';}
   if(reception.known&&!reception.payout&&!reception.expired){
    // Redeeming is a one-shot continuation of the explicit browser choice.
    if(reception.code&&window.nimiq)await chooseReception('pay');
    else if(window.nimiq||hasHandoff)reception.message='claim_enterCode';
    else await claimTask(requestClaimCode,$('request-claim-code'));
   }
   updateClaimControls();
  }catch(error){if(!reception.payout){reception.expired=true;reception.message=['claim_unavailable','claim_expired','card_expired'].includes(error.message)?'ended':'failed';updateClaimControls();}}return;
 }
 try{const config=await api('/api/config');accessKnown=true;inviteOnly=config.invite_only===true;configured=config.mode==='integrated';ready=config.payments_enabled===true;usdtAssets=Array.isArray(config.assets)?config.assets:[];if(!configured)return;
  $('sender-auth').hidden=embedded;refreshText();const session=await api('/api/session');const limited=scopedPaymentID&&session.handoff?.payment_id===scopedPaymentID;
  senderUI(session.verified?session.email:limited?(initialPayment?.sender||'handoff'):'',session.admin);
  if(preparedContext){
   root.classList.add('wire-nim-prepared');
   $('nim-handoff-card').append($('payment-card'));
   $('nim-handoff-summary').append($('feedback'),fallback);
  }
  if(verified)await loadHistory();
  if(!scopedPaymentID&&!preparedContext&&(freshForm||paymentList.length>0&&paymentList.every(isFolded))){
   freshForm=true;save('wire-nim-new-form','yes');
   for(const id of ['recipient','amount','message']){$(id).value='';$(id).readOnly=false;}
   $('currency').disabled=false;$('payment-card').hidden=true;$('preview-title').hidden=true;renderProgress();
  }
  if(preparedContext){
   if(!verified||!active){$('feedback').textContent=pt('session');renderProgress();return;}
   pendingFailure(active);
   showPayment(active);resume();return;
  }
  if(embedded&&initialDraft){
   if(initialDraft.payment_id){
    const original=paymentList.find(p=>p.id===initialDraft.payment_id);
    if(original){active=original;showPayment(original);}
   }else{
    active=null;displayed=null;
    for(const id of ['recipient','amount','message']){$(id).value=initialDraft[id];$(id).readOnly=false;}
    $('currency').disabled=false;$('preview-empty').hidden=false;$('payment-card').hidden=true;$('preview-title').hidden=true;
    renderProgress();
   }
  }else try{const draft=JSON.parse(read('wire-form')||'null');if(draft&&active?.id===draft.payment)showPayment(active);}catch{}
  const pending=read('wire-pending-payment');
  if(pending&&/^[a-f0-9]{48}$/.test(pending)&&read('wire-pending-wallet')==='pay'){
   await poll(pending);resume();return;
  }
  if(pending&&/^[a-f0-9]{48}$/.test(pending)&&window.HubApi){
   const hub=new window.HubApi('https://hub.nimiq.com',new window.HubApi.RedirectRequestBehavior());
   hub.on(window.HubApi.RequestType.CHECKOUT,()=>task(async()=>{const p=paymentList.find(p=>p.id===pending);if(p)approve(p);await poll(pending);}),()=>{status('feedback','unknownWallet');resume();});
   await hub.checkRedirectResponse();
  }
  resume();renderDraftPreview();
 }catch{status('feedback','unavailable');}
}
const started=boot();

return {ready:started,resume,destroy(){destroyed=true;window.removeEventListener('wire-language',refreshText);document.removeEventListener('visibilitychange',visibility);window.removeEventListener('pageshow',resume);window.removeEventListener('online',resume);clearInterval(coolTimer);clearTimeout(reception.timer);clearTimeout(walletTimer);clearTimeout(historyTimer);for(const watch of monitors.values())stopMonitor(watch);for(const controller of controllers)controller.abort();controllers.clear();}};
}

if(typeof document!=='undefined'&&document.getElementById('card-form'))mountNimFlow();

