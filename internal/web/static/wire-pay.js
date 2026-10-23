// Phoenix Wire's page: PhoenixMail's own payment window (/pm/*), used as it is. This file is only what is Wire's: who the sender is, and
// what happens to a card once the window has made it (watch the money, then e-mail the recipient the claim link, from this browser).
import { resolveLanguage } from './model.js?v=0.3.0';
import { mountNimFlow } from './flow.js?v=0.4.9';

window.PhoenixBrandName = 'Phoenix Wire'; // the payment window's texts name the app they run in
const $ = id => document.getElementById(id);
const STORE = 'wire-usdt-cards', words = {
 en: { nimRecipient: "Recipient email", nimAmount: "Amount (NIM)", nimMessage: "Message (optional)", nimCard: "Your Nimiq card", nimPreview: "Card preview", nimEmpty: "Create a card to preview it.", nimProtected: "Protected", lightTheme: "Switch to light theme", darkTheme: "Switch to dark theme", sessionHelp: "You confirm your email once. It stays confirmed for 24 hours on this device.", sessionConfirmed: "Confirmed for 24 hours.", back: "Back to PhoenixWire", title: 'One payment. One email.', sub: "Send NIM, USDC or USDT to anyone with an email address. Only they can receive it, after a code sent to that address.", email: 'Your email', send: 'Send me a code', code: 'Six-digit code', verify: 'Confirm', as: 'Signed in as', out: 'Change sender', pay: 'Payments', codeSent: 'Code sent. Check your email.', wrong: 'This code is not valid.', off: 'Payments are not switched on here.', funded: 'Card funded. Sending the invitation…', sent: 'The card was sent to {email}. It stays yours until it is claimed.', retry: 'The e-mail could not be sent yet. Trying again…', classic: 'Send NIM', fail: 'Something went wrong. Try again.' },
 fr: { nimRecipient: "Courriel du destinataire", nimAmount: "Montant (NIM)", nimMessage: "Message (facultatif)", nimCard: "Votre carte Nimiq", nimPreview: "Aperçu de la carte", nimEmpty: "Créez une carte pour la prévisualiser.", nimProtected: "Protégé", lightTheme: "Passer au thème clair", darkTheme: "Passer au thème sombre", sessionHelp: "Vous confirmez votre courriel une seule fois : il reste confirmé 24 heures sur cet appareil.", sessionConfirmed: "Confirmé pour 24 h.", back: "Retour à PhoenixWire", title: 'Un paiement. Un courriel.', sub: "Envoyez des NIM, des USDC ou des USDT à toute personne qui a une adresse courriel. Seule cette personne peut les recevoir, après un code envoyé à cette adresse.", email: 'Votre courriel', send: 'M’envoyer un code', code: 'Code à six chiffres', verify: 'Confirmer', as: 'Connecté en tant que', out: 'Changer d’expéditeur', pay: 'Paiements', codeSent: 'Code envoyé. Vérifiez votre courriel.', wrong: 'Ce code n’est pas valide.', off: 'Les paiements ne sont pas activés ici.', funded: 'Carte alimentée. Envoi de l’invitation…', sent: 'La carte a été envoyée à {email}. Elle reste à vous tant qu’elle n’est pas réclamée.', retry: 'Le courriel n’a pas pu partir. Nouvel essai…', classic: 'Envoyer des NIM', fail: 'Une erreur est survenue. Réessayez.' },
 es: { nimRecipient: "Correo del destinatario", nimAmount: "Monto (NIM)", nimMessage: "Mensaje (opcional)", nimCard: "Tu tarjeta Nimiq", nimPreview: "Vista previa de la tarjeta", nimEmpty: "Crea una tarjeta para verla.", nimProtected: "Protegido", lightTheme: "Cambiar al tema claro", darkTheme: "Cambiar al tema oscuro", sessionHelp: "Confirmas tu correo una sola vez: queda confirmado 24 horas en este dispositivo.", sessionConfirmed: "Confirmado por 24 horas.", back: "Volver a PhoenixWire", title: 'Un pago. Un correo.', sub: "Envía NIM, USDC o USDT a cualquier persona con una dirección de correo. Solo ella puede recibirlo, tras un código enviado a esa dirección.", email: 'Tu correo', send: 'Envíame un código', code: 'Código de seis dígitos', verify: 'Confirmar', as: 'Sesión de', out: 'Cambiar de remitente', pay: 'Pagos', codeSent: 'Código enviado. Revisa tu correo.', wrong: 'Este código no es válido.', off: 'Los pagos no están activados aquí.', funded: 'Tarjeta financiada. Enviando la invitación…', sent: 'La tarjeta se envió a {email}. Sigue siendo tuya hasta que se reclame.', retry: 'El correo no pudo enviarse aún. Reintentando…', classic: 'Enviar NIM', fail: 'Algo salió mal. Inténtalo de nuevo.' },
 de: { nimRecipient: "E-Mail des Empfängers", nimAmount: "Betrag (NIM)", nimMessage: "Nachricht (optional)", nimCard: "Deine Nimiq-Karte", nimPreview: "Kartenvorschau", nimEmpty: "Erstelle eine Karte für die Vorschau.", nimProtected: "Geschützt", lightTheme: "Zum hellen Design wechseln", darkTheme: "Zum dunklen Design wechseln", sessionHelp: "Du bestätigst deine E-Mail nur einmal: Sie bleibt 24 Stunden auf diesem Gerät bestätigt.", sessionConfirmed: "24 Stunden bestätigt.", back: "Zurück zu PhoenixWire", title: 'Eine Zahlung. Eine E-Mail.', sub: "Sende NIM, USDC oder USDT an jede Person mit einer E-Mail-Adresse. Nur sie kann sie empfangen, nach einem Code an diese Adresse.", email: 'Deine E-Mail', send: 'Code senden', code: 'Sechsstelliger Code', verify: 'Bestätigen', as: 'Angemeldet als', out: 'Absender wechseln', pay: 'Zahlungen', codeSent: 'Code gesendet. Prüfe deine E-Mail.', wrong: 'Dieser Code ist ungültig.', off: 'Zahlungen sind hier nicht aktiviert.', funded: 'Karte aufgeladen. Die Einladung wird gesendet…', sent: 'Die Karte wurde an {email} gesendet. Sie bleibt deine, bis sie eingelöst wird.', retry: 'Die E-Mail konnte noch nicht gesendet werden. Neuer Versuch…', classic: 'NIM senden', fail: 'Etwas ist schiefgelaufen. Versuche es erneut.' },
 zh: { nimRecipient: "收款人邮箱", nimAmount: "金额（NIM）", nimMessage: "留言（可选）", nimCard: "你的 Nimiq 卡片", nimPreview: "卡片预览", nimEmpty: "创建卡片以查看预览。", nimProtected: "受保护", lightTheme: "切换为浅色主题", darkTheme: "切换为深色主题", sessionHelp: "你只需确认一次邮箱：在此设备上 24 小时内保持已确认。", sessionConfirmed: "已确认 24 小时。", back: "返回 PhoenixWire", title: '一笔付款。一封邮件。', sub: "把 NIM、USDC 或 USDT 发给任何有邮箱的人。只有对方在输入发送到该邮箱的验证码后才能收款。", email: '你的邮箱', send: '给我发验证码', code: '六位验证码', verify: '确认', as: '登录身份', out: '更换发送者', pay: '付款', codeSent: '验证码已发送，请查收邮件。', wrong: '验证码无效。', off: '此处未启用付款。', funded: '卡片已充值。正在发送邀请…', sent: '卡片已发送到 {email}。在被领取之前它仍属于你。', retry: '邮件暂时无法发送。正在重试…', classic: '发送 NIM', fail: '出错了，请重试。' },
};

const walletChoiceWords={
 en:{choiceTitle:'How would you like to pay?',openPay:'I already have Nimiq Pay',getPay:'Get Nimiq Pay',useHub:'Use the Nimiq wallet (Hub)',cancelPay:'Cancel',choiceHelp:'Nimiq Pay opens Wire in the app. Confirm your email there once; it stays confirmed for 24 hours. Your browser session is separate.',openPayHelp:'If the app did not open, try again, install Nimiq Pay, or use the Hub. No payment has started.',storeHelp:'Choose the store for your phone.'},
 fr:{choiceTitle:'Comment voulez-vous payer ?',openPay:'J’ai déjà Nimiq Pay',getPay:'Obtenir Nimiq Pay',useHub:'Utiliser le portefeuille Nimiq (Hub)',cancelPay:'Annuler',choiceHelp:'Nimiq Pay ouvre Wire dans l’application. Confirmez-y votre courriel une fois : il reste confirmé 24 heures. La session du navigateur est distincte.',openPayHelp:'Si l’application ne s’est pas ouverte, réessayez, installez Nimiq Pay ou utilisez le Hub. Aucun paiement n’a commencé.',storeHelp:'Choisissez la boutique de votre téléphone.'},
 es:{choiceTitle:'¿Cómo quieres pagar?',openPay:'Ya tengo Nimiq Pay',getPay:'Obtener Nimiq Pay',useHub:'Usar la cartera Nimiq (Hub)',cancelPay:'Cancelar',choiceHelp:'Nimiq Pay abre Wire en la aplicación. Confirma allí tu correo una vez: queda confirmado 24 horas. La sesión del navegador es independiente.',openPayHelp:'Si la aplicación no se abrió, inténtalo de nuevo, instala Nimiq Pay o usa el Hub. No se ha iniciado ningún pago.',storeHelp:'Elige la tienda de tu teléfono.'},
 de:{choiceTitle:'Wie möchtest du bezahlen?',openPay:'Ich habe bereits Nimiq Pay',getPay:'Nimiq Pay herunterladen',useHub:'Nimiq Wallet (Hub) verwenden',cancelPay:'Abbrechen',choiceHelp:'Nimiq Pay öffnet Wire in der App. Bestätige dort einmal deine E-Mail: Sie bleibt 24 Stunden bestätigt. Die Browsersitzung ist davon getrennt.',openPayHelp:'Falls die App nicht geöffnet wurde, versuche es erneut, installiere Nimiq Pay oder nutze den Hub. Es wurde keine Zahlung gestartet.',storeHelp:'Wähle den Store für dein Smartphone.'},
 zh:{choiceTitle:'你想如何付款？',openPay:'我已安装 Nimiq Pay',getPay:'获取 Nimiq Pay',useHub:'使用 Nimiq 钱包（Hub）',cancelPay:'取消',choiceHelp:'Nimiq Pay 会在应用内打开 Wire。请在那里确认一次邮箱，确认状态将保持 24 小时。浏览器的会话与应用内的会话相互独立。',openPayHelp:'如果应用未打开，请重试、安装 Nimiq Pay，或使用 Hub。尚未开始付款。',storeHelp:'请选择你手机的应用商店。'}
};
for(const language of Object.keys(walletChoiceWords))Object.assign(words[language],walletChoiceWords[language]);
const cardWords={
 fr:{myCards:'Mes cartes',emptyCards:'Aucune carte pour le moment.',waitingCard:'En attente de votre paiement',confirmingCard:'Paiement reçu, confirmation en cours',fundedCard:'Carte alimentée. Wire envoie la carte à {email}.',sentCard:'Carte envoyée à {email}',claimedCard:'Réclamée',refundedCard:'Remboursée',expiredCard:'Expirée',cancelledCard:'Annulée',otherCard:'Dépensée ailleurs',checkingCard:'Vérification de la carte en cours…',mailFailedCard:'Carte alimentée. Le courriel n’a pas pu être envoyé. Wire réessaiera.',missingLinkCard:'Carte alimentée. Le lien d’invitation est nécessaire : utilisez le navigateur où la carte a été créée.',staleCard:'Vue périmée : le service ne répond pas. Vérification automatique en cours.',bannerCard:'Carte alimentée. Courriel envoyé à {email}.',dismissCard:'Fermer cette notification',doneCard:'Terminé',createdCard:'Carte créée. Payez-la pour que Wire l’envoie au destinataire.'},
 en:{myCards:'My cards',emptyCards:'No cards yet.',waitingCard:'Waiting for your payment',confirmingCard:'Payment received, awaiting confirmation',fundedCard:'Card funded. Wire is sending the card to {email}.',sentCard:'Card sent to {email}',claimedCard:'Claimed',refundedCard:'Refunded',expiredCard:'Expired',cancelledCard:'Cancelled',otherCard:'Spent elsewhere',checkingCard:'Checking the card…',mailFailedCard:'Card funded. The email could not be sent. Wire will retry.',missingLinkCard:'Card funded. The invitation link is needed: use the browser where this card was created.',staleCard:'Stale view: the service is not responding. Checking again automatically.',bannerCard:'Card funded. Email sent to {email}.',dismissCard:'Dismiss this notification',doneCard:'Done',createdCard:'Card created. Pay it so Wire can send it to the recipient.'},
 es:{myCards:'Mis tarjetas',emptyCards:'Aún no hay tarjetas.',waitingCard:'Esperando tu pago',confirmingCard:'Pago recibido, esperando confirmación',fundedCard:'Tarjeta financiada. Wire está enviando la tarjeta a {email}.',sentCard:'Tarjeta enviada a {email}',claimedCard:'Reclamada',refundedCard:'Reembolsada',expiredCard:'Caducada',cancelledCard:'Cancelada',otherCard:'Gastada en otro lugar',checkingCard:'Comprobando la tarjeta…',mailFailedCard:'Tarjeta financiada. No se pudo enviar el correo. Wire lo intentará de nuevo.',missingLinkCard:'Tarjeta financiada. Se necesita el enlace de invitación: usa el navegador donde se creó.',staleCard:'Vista desactualizada: el servicio no responde. La comprobación continúa automáticamente.',bannerCard:'Tarjeta financiada. Correo enviado a {email}.',dismissCard:'Cerrar esta notificación',doneCard:'Listo',createdCard:'Tarjeta creada. Págala para que Wire la envíe al destinatario.'},
 de:{myCards:'Meine Karten',emptyCards:'Noch keine Karten.',waitingCard:'Wartet auf deine Zahlung',confirmingCard:'Zahlung eingegangen, Bestätigung ausstehend',fundedCard:'Karte aufgeladen. Wire sendet die Karte an {email}.',sentCard:'Karte an {email} gesendet',claimedCard:'Eingelöst',refundedCard:'Zurückerstattet',expiredCard:'Abgelaufen',cancelledCard:'Storniert',otherCard:'Anderswo ausgegeben',checkingCard:'Karte wird geprüft…',mailFailedCard:'Karte aufgeladen. Die E-Mail konnte nicht gesendet werden. Wire versucht es erneut.',missingLinkCard:'Karte aufgeladen. Der Einladungslink wird benötigt: Nutze den Browser, in dem die Karte erstellt wurde.',staleCard:'Veraltete Ansicht: Der Dienst antwortet nicht. Die Prüfung wird automatisch wiederholt.',bannerCard:'Karte aufgeladen. E-Mail an {email} gesendet.',dismissCard:'Diese Mitteilung schließen',doneCard:'Fertig',createdCard:'Karte erstellt. Bezahle sie, damit Wire sie an den Empfänger senden kann.'},
 zh:{myCards:'我的卡片',emptyCards:'暂无卡片。',waitingCard:'等待你付款',confirmingCard:'已收到付款，正在等待确认',fundedCard:'卡片已充值。Wire 正在向 {email} 发送卡片。',sentCard:'卡片已发送至 {email}',claimedCard:'已领取',refundedCard:'已退款',expiredCard:'已过期',cancelledCard:'已取消',otherCard:'已在其他地方支出',checkingCard:'正在检查卡片…',mailFailedCard:'卡片已充值，但邮件未能发送。Wire 将重试。',missingLinkCard:'卡片已充值。需要邀请链接：请使用创建此卡片的浏览器。',staleCard:'视图已过时：服务暂未响应，正在自动重试检查。',bannerCard:'卡片已充值。邮件已发送至 {email}。',dismissCard:'关闭此通知',doneCard:'完成',createdCard:'卡片已创建。付款后 Wire 会将其发送给收件人。'}
};
for(const language of Object.keys(cardWords))Object.assign(words[language],cardWords[language]);
const checkoutWords={
 fr:{chooseCoin:'Choisissez la monnaie',sendingCode:'Envoi du code…',codeSentButton:'Code envoyé',resendCode:'Renvoyer un code',out:'Se déconnecter'},
 en:{chooseCoin:'Choose a currency',sendingCode:'Sending code…',codeSentButton:'Code sent',resendCode:'Send another code',out:'Sign out'},
 es:{chooseCoin:'Elige una moneda',sendingCode:'Enviando código…',codeSentButton:'Código enviado',resendCode:'Enviar otro código',out:'Cerrar sesión'},
 de:{chooseCoin:'Währung wählen',sendingCode:'Code wird gesendet…',codeSentButton:'Code gesendet',resendCode:'Weiteren Code senden',out:'Abmelden'},
 zh:{chooseCoin:'选择币种',sendingCode:'正在发送验证码…',codeSentButton:'验证码已发送',resendCode:'重新发送验证码',out:'退出登录'}
};
for(const language of Object.keys(checkoutWords))Object.assign(words[language],checkoutWords[language]);
const paletteWords={fr:{palette:'Thème',ocean:'Océan',livePalette:'Aléatoire en direct'},en:{palette:'Theme',ocean:'Ocean',livePalette:'Live shuffle'},es:{palette:'Tema',ocean:'Océano',livePalette:'Aleatorio en directo'},de:{palette:'Design',ocean:'Ozean',livePalette:'Zufälliger Wechsel'},zh:{palette:'主题',ocean:'海洋',livePalette:'实时随机'}};
for(const language of Object.keys(paletteWords))Object.assign(words[language],paletteWords[language]);
window.PhoenixWordOverrides=Object.fromEntries(Object.entries(cardWords).map(([language,w])=>[language,{fundedMsg:w.fundedCard,confirmingMsg:w.confirmingCard,close:w.doneCard,added:w.createdCard,created:w.createdCard}]));
function read(key, fallback) { try { return localStorage.getItem(key) || fallback; } catch { return fallback; } }
function save(key, value) { try { localStorage.setItem(key, value); } catch { /* storage may be unavailable */ } }
let preference = read('wire-language', 'auto'), lang = 'en', email = '', payments = null, usdtOn = false, nimOn = false, enabled = false, nimView = null, resumed = false;
let nimCache=null;
let choosingCoin=false,codeSending=false,codeSentTo='',codeUntil=0,codeTimer=null;
const t = key => (words[lang] || words.en)[key];
// Only a payment draft crosses contexts. Never export cookies, grants, link keys or recovery URLs.
function validNimDraft(d){
 return d&&d.v===1&&Number.isFinite(d.expires)&&d.expires>Date.now()&&d.expires<=Date.now()+86400000&&
 ['recipient','sender'].every(k=>typeof d[k]==='string'&&d[k].length<=254&&/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(d[k]))&&
 typeof d.amount==='string'&&/^\d{1,9}(?:\.\d{1,5})?$/.test(d.amount)&&Number(d.amount)>0&&
 typeof d.message==='string'&&[...d.message].length<=160&&/^[a-f0-9]{32}$/.test(d.request_id)&&
 (d.payment_id===''||/^[a-f0-9]{48}$/.test(d.payment_id))&&['fr','en','es','de','zh'].includes(d.language);
}
function readNimDraft(){try{const d=JSON.parse(sessionStorage.getItem('wire-nim-handoff')||'null');return validNimDraft(d)?d:null;}catch{return null;}}
let handoff=readNimDraft();
if(location.hash.startsWith('#wire-nim=')){
 const encoded=location.hash.slice(10);
 handoff=null;try{sessionStorage.removeItem('wire-nim-handoff');}catch{}
 // Remove the transferred draft from the address bar before any network request.
 history.replaceState(null,'',location.pathname+location.search);
 try{if(encoded.length<=6000){const d=JSON.parse(decodeURIComponent(encoded));if(validNimDraft(d)){handoff=d;sessionStorage.setItem('wire-nim-handoff',JSON.stringify(d));}}}catch{}
}
if(handoff){preference=handoff.language;save('wire-language',preference);$('wire-email').value=handoff.sender;}

async function api(path, data, signal) {
 const options = { credentials: 'same-origin', cache: 'no-store', redirect: 'error' };
 if(signal)options.signal=signal;
 if (data !== undefined) { options.method = 'POST'; options.headers = { 'Content-Type': 'application/json' }; options.body = JSON.stringify(data); }
 const response = await fetch(path, options);
 let body = {}; try { body = await response.json(); } catch { /* an answer without a body */ }
 if (!response.ok) throw Object.assign(Error(body.code || body.error || 'unavailable'), { code: body.code || body.error, status: response.status });
 return body;
}
const say = text => { $('wire-status').textContent = text; };
// The copied window calls this hook on resize/keyboard changes. Keep it inside
// the visual viewport rather than retaining a desktop drag offset on a phone.
window.PhoenixViewport={center(win){
 const v=window.visualViewport,w=v?.width||innerWidth,h=v?.height||innerHeight;
 const r=win.getBoundingClientRect();
 win.style.margin='0';win.style.inset='auto';
 win.style.left=((v?.offsetLeft||0)+Math.max(8,(w-r.width)/2))+'px';
 win.style.top=((v?.offsetTop||0)+Math.max(8,(h-r.height)/2))+'px';
 win.style.maxHeight=Math.max(100,h-16)+'px';
}};

function paint() {
 document.documentElement.lang = lang;
 $('wire-title').textContent = t('title'); $('wire-sub').textContent = t('sub');
 $('wire-email-label').textContent = t('email');paintCodeRequest(); $('wire-code-label').textContent = t('code'); $('wire-verify').textContent = t('verify');
 $('wire-as').textContent = t('as'); $('wire-logout').setAttribute('aria-label',t('out'));$('wire-logout').title=t('out');
 $('wire-language').value = preference;
 $('wire-palette').setAttribute('aria-label',t('palette'));$('wire-palette').querySelector('[value="ocean"]').textContent=t('ocean');
 $('wire-palette').querySelector('[value="live"]').textContent=t('livePalette');
 $('wire-pay-label').textContent = t('pay');
 paintTheme();
 if (nimCache) paintNim(nimCache.modal);
 window.dispatchEvent(new Event('wire-language'));
 $('wire-session-help').textContent = t('sessionHelp'); $('wire-session-confirmed').textContent = t('sessionConfirmed');
 document.querySelectorAll('.wire-window-back').forEach(button => { button.textContent = t('back'); });
 document.querySelectorAll('.wire-coin-picker .nimiq-payment-head h3').forEach(title=>{title.textContent=t('chooseCoin');});
 document.querySelectorAll('[data-wallet-text]').forEach(el=>{el.textContent=t(el.dataset.walletText);});
 renderCards();
}
// iOS cannot reliably disclose whether another app is installed. Ask, without timers or blur heuristics.
const payLink='https://nimpay.app/miniapps/open/wire.phoenixmail.ca';
const iosStore='https://apps.apple.com/app/nimiq-pay/id6471844738';
const androidStore='https://play.google.com/store/apps/details?id=com.nimiq.pay';
let choiceCallback=null,choiceFocus=null,choiceParent=null;
function closeWalletChoice(){
 $('wire-nim-choice').hidden=true;choiceCallback=null;choiceParent?.removeAttribute('inert');choiceParent=null;choiceFocus?.focus();
}
function chooseWallet(useHub,draft){
 choiceCallback=useHub;choiceFocus=document.activeElement;choiceParent=document.querySelector('.wire-nim-window');choiceParent?.setAttribute('inert','');
 $('wire-choice-status').hidden=true;$('wire-choice-stores').hidden=true;$('wire-nim-choice').hidden=false;
 const d={v:1,expires:Date.now()+86400000,recipient:draft?.recipient,amount:draft?.amount,message:draft?.message,sender:draft?.sender,request_id:draft?.request_id,payment_id:draft?.payment_id,language:lang};
 // The custom scheme carries the complete mini-app URL, including its fragment.
 // The bare HTTPS domain link cannot restore a draft in the app's separate storage.
 $('wire-choice-open').href=validNimDraft(d)?'nimiqpay://miniapp?url='+encodeURIComponent('https://wire.phoenixmail.ca/#wire-nim='+encodeURIComponent(JSON.stringify(d))):payLink;
 const ua=navigator.userAgent||'';
 $('wire-choice-get').href=/Android/i.test(ua)?androidStore:/iPhone|iPad|iPod/i.test(ua)||navigator.platform==='MacIntel'&&navigator.maxTouchPoints>1?iosStore:'#';
 $('wire-choice-open').focus();
}
$('wire-choice-open').addEventListener('click',()=>{$('wire-choice-status').hidden=false;});
$('wire-choice-get').addEventListener('click',event=>{
 if($('wire-choice-get').getAttribute('href')==='#'){event.preventDefault();$('wire-choice-stores').hidden=false;}
});
$('wire-choice-hub').addEventListener('click',()=>{const useHub=choiceCallback;closeWalletChoice();useHub?.();});
$('wire-choice-cancel').addEventListener('click',closeWalletChoice);
document.addEventListener('keydown',event=>{
 if($('wire-nim-choice').hidden)return;
 if(event.key==='Escape'){event.preventDefault();event.stopImmediatePropagation();closeWalletChoice();}
 if(event.key==='Tab'){
  const controls=[...$('wire-nim-choice').querySelectorAll('a,button')].filter(el=>!el.closest('[hidden]'));
  const first=controls[0],last=controls.at(-1);
  if(event.shiftKey&&document.activeElement===first){event.preventDefault();last.focus();}
  else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first.focus();}
 }
},true);
function paintTheme() {
 const light = document.documentElement.dataset.theme === 'light-paper';
 const button = $('wire-theme'); button.textContent = light ? '☾' : '☀';
 button.setAttribute('aria-label', t(light ? 'darkTheme' : 'lightTheme'));
 button.title = button.getAttribute('aria-label'); button.setAttribute('aria-pressed', String(light));
}
function applyTheme(value) {
 document.documentElement.dataset.theme = value === 'light' || value === 'light-paper' ? 'light-paper' : 'dark';
 paintTheme();
}
const palettes=['phoenixfire','aurora','ocean'];
let paletteMode='live',paletteTimer=null;
function setPaletteColour(palette){document.documentElement.dataset.palette=palette;document.documentElement.dataset.accent=palette;}
function schedulePalette(){
 clearTimeout(paletteTimer);paletteTimer=null;
 if(paletteMode!=='live'||document.hidden)return;
 paletteTimer=setTimeout(()=>{
  if(document.hidden)return;
  const options=palettes.filter(p=>p!==document.documentElement.dataset.palette);
  setPaletteColour(options[Math.floor(Math.random()*options.length)]);schedulePalette();
 },20000);
}
function applyPalette(value){
 paletteMode=value==='live'||palettes.includes(value)?value:'phoenixfire';
 document.documentElement.dataset.paletteMode=paletteMode;$('wire-palette').value=paletteMode;
 setPaletteColour(paletteMode==='live'?(palettes.includes(document.documentElement.dataset.palette)?document.documentElement.dataset.palette:'phoenixfire'):paletteMode);
 schedulePalette();
}
document.addEventListener('visibilitychange',schedulePalette);window.addEventListener('pageshow',schedulePalette);
function styleClose(button){button.replaceChildren($('wire-close-icon').content.cloneNode(true));}
function paintNim(root) {
 root.querySelectorAll('[data-nim-text]').forEach(el => { el.textContent = t(el.dataset.nimText); });
 root.querySelector('.wire-nim-close').setAttribute('aria-label', t('back'));
 styleClose(root.querySelector('.wire-nim-close'));
}
function closePaymentWindow() {
 document.querySelector('.nimiq-payment-head button:last-child')?.click();
}
function openNimWindow() {
 if (!email || !nimOn) return;
 choosingCoin=false;
 if(nimView)return;
 closePaymentWindow();
 if(nimCache){
  document.body.append(nimCache.modal);document.addEventListener('keydown',nimCache.escape);nimView=nimCache.view;
  paintNim(nimCache.modal);nimCache.flow.resume();nimCache.modal.querySelector('.wire-window-back').focus();return;
 }
 const modal = $('wire-nim-template').content.firstElementChild.cloneNode(true);
 const root = modal.querySelector('.wire-nim-host'), win = modal.querySelector('.nimiq-payment-window');
 const row = modal.querySelector('.wire-nim-assets');
 row.hidden=true;
 for (const [asset, label] of [['nim','NIM · Nimiq'],['usdc_polygon','USDC · Polygon'],['usdt_polygon','USDT · Polygon'],['usdt_ethereum','USDT · Ethereum']]) {
  const button = document.createElement('button'); button.type = 'button'; button.className = 'secondary-btn'; button.textContent = label;
  button.setAttribute('aria-pressed', String(asset === 'nim')); button.disabled = asset !== 'nim' && !usdtOn;
  button.onclick = () => { if (asset === 'nim') return; dismiss(); payments.open('send', asset); }; row.append(button);
 }
 const back = modal.querySelector('.wire-window-back'); back.textContent = t('back');
 let flow;
 // Keep the authenticated NIM tracker alive when the window is closed or another coin is selected.
 const dismiss = () => { closeWalletChoice();modal.remove(); document.removeEventListener('keydown', escape); nimView = null;if(root.classList.contains('wire-nim-complete')){flow?.destroy();nimCache=null;} $('pay').focus(); };
 const escape = e => { if (e.key === 'Escape') { e.preventDefault(); dismiss(); } };
 back.onclick = dismiss; modal.querySelector('.wire-nim-close').onclick = dismiss;
 document.addEventListener('keydown', escape);
 document.body.append(modal); nimView = { root: modal, dismiss };
 const draft=handoff?.sender.toLowerCase()===email.toLowerCase()?handoff:null;
 if(draft){try{
  sessionStorage.setItem('wire-request-id',draft.request_id);
  if(draft.payment_id)sessionStorage.setItem('wire-pending-payment',draft.payment_id);
  else{sessionStorage.removeItem('wire-pending-payment');sessionStorage.removeItem('wire-pending-wallet');sessionStorage.removeItem('wire-form');}
 }catch{}}
 const clearDraft=()=>{handoff=null;try{sessionStorage.removeItem('wire-nim-handoff');}catch{}};
 paintNim(modal);flow=mountNimFlow(root,{embedded:true,chooseWallet,initialDraft:draft,onDraftUsed:clearDraft,onStatus:(p,message)=>{
  nimCards.set(p.id,p);
  if(draft?.payment_id===p.id&&p.stage!=='ready')clearDraft();
  if(p.stage==='sent'){
   const notices=nimNotices();
   if(!notices.some(n=>n.id===p.id&&n.sender===email)){notices.push({id:p.id,sender:email,noticePending:true});save('wire-nim-notices',JSON.stringify(notices));}
  }
  renderCards();scheduleNimCards();say(message);
 },onClose:()=>{const message=root.querySelector('#nim-progress-message').textContent;dismiss();renderCards();say(message);}});
 nimCache={modal,flow,escape,view:nimView};
 back.focus();
}
function setLanguage(value) {
 preference = value; save('wire-language', value);
 const nav = typeof navigator === 'undefined' ? {} : navigator;
 lang = resolveLanguage(value, nav.languages?.length ? nav.languages : [nav.language]);
 paint();
}

// ---- the money of a card the window has just made: watch it, and e-mail the recipient once it is funded ----
const cards = () => { try { const v = JSON.parse(read(STORE, '[]')); return Array.isArray(v) ? v : []; } catch { return []; } };
// Never evict a claim/refund fragment to make room for dashboard history.
const keep = list => save(STORE, JSON.stringify(list));
const update = (id, patch) => { const all = cards(); const c = all.find(x => x.id === id); if (c) { Object.assign(c, patch); keep(all); } };
const watching=new Map(),nimCards=new Map(),cardControllers=new Set();
const cardStates=new Set(['awaiting_funds','ready','verifying','paid','withdrawn','cancelled','expired','settled_elsewhere']);
const terminalCards=new Set(['paid','withdrawn','cancelled','expired','settled_elsewhere']);
const assetNames={usdc_polygon:'USDC · Polygon',usdt_polygon:'USDT · Polygon',usdt_ethereum:'USDT · Ethereum'};
let cardEpoch=0,ownedPromise=null,nimPromise=null,nimTimer=null;
const fill=(key,entry)=>t(key).replace('{email}',entry.recipient||'');
function cardLabel(c){
 if(c.nim)return fill(({preparing:'checkingCard',ready:'waitingCard',funding_started:'confirmingCard',funded:'fundedCard',delivery_pending:'fundedCard',delivery_failed:'mailFailedCard',sent:'sentCard',expired:'expiredCard'})[c.state]||'checkingCard',c);
 const key=({paid:'claimedCard',withdrawn:'refundedCard',expired:'expiredCard',cancelled:'cancelledCard',settled_elsewhere:'otherCard'})[c.state];
 if(key)return t(key);
 if(c.state==='verifying')return t('confirmingCard');
 if(c.state==='ready')return fill(c.invited?'sentCard':c.mailFailed?'mailFailedCard':!c.fragment?'missingLinkCard':'fundedCard',c);
 if(c.state==='awaiting_funds')return t(c.balanceUnits>0?'confirmingCard':'waitingCard');
 return t('checkingCard');
}
function nimNotices(){try{const value=JSON.parse(read('wire-nim-notices','[]'));return Array.isArray(value)?value:[];}catch{return [];}}
function renderCards(){
 const panel=$('wire-cards');if(!panel)return;
 panel.hidden=!email; $('wire-cards-title').textContent=t('myCards');
 const all=cards().filter(c=>c.sender===email).map(c=>({...c,nim:false}));
 for(const p of nimCards.values())if(p.sender===email)all.push({id:p.id,nim:true,amount:p.amount,asset:'NIM',recipient:p.recipient,state:p.stage!=='sent'&&!nimPending(p)?'expired':p.stage,created:Date.parse(p.created)||0,invited:p.stage==='sent',noticePending:nimNotices().some(n=>n.id===p.id&&n.sender===email&&n.noticePending)});
 all.sort((a,b)=>(Number(b.created)||0)-(Number(a.created)||0));
 if(!nimView&&all.length)say(cardLabel(all[0]));
 const list=$('wire-cards-list');list.replaceChildren();
 $('wire-cards-empty').textContent=t('emptyCards');$('wire-cards-empty').hidden=all.length>0;
 for(const c of all){
  const row=document.createElement('article');row.className='wire-card-row';row.dataset.cardId=c.id;
  const amount=document.createElement('strong');amount.textContent=(c.amount||'—')+' '+(c.nim?'NIM · Nimiq':assetNames[c.asset]||'USDC/USDT');
  const recipient=document.createElement('p');recipient.className='wire-card-recipient';recipient.textContent=c.recipient||'—';
  const state=document.createElement('p');state.className='wire-card-state';state.textContent=cardLabel(c);row.append(amount,recipient,state);
  if(c.stale){const stale=document.createElement('p');stale.className='wire-card-stale';stale.textContent=t('staleCard');row.append(stale);}
  list.append(row);
 }
 const notices=$('wire-card-notices');notices.replaceChildren();
 for(const c of all.filter(c=>c.invited&&c.noticePending)){
  const banner=document.createElement('section');banner.className='wire-card-banner';
  const message=document.createElement('p');message.setAttribute('role','status');message.textContent=fill('bannerCard',c);
  const close=document.createElement('button');close.type='button';close.className='secondary-btn';close.textContent='×';close.setAttribute('aria-label',t('dismissCard'));
  close.addEventListener('click',()=>{if(c.nim)save('wire-nim-notices',JSON.stringify(nimNotices().map(n=>n.id===c.id&&n.sender===email?{...n,noticePending:false}:n)));else update(c.id,{noticePending:false});renderCards();$('wire-cards-title').focus();});
  banner.append(message,close);notices.append(banner);
 }
 notices.hidden=!email||!notices.childElementCount;
}
async function cardCall(path,data){
 const controller=new AbortController();cardControllers.add(controller);
 const timer=setTimeout(()=>controller.abort(),30000);
 try{return await api(path,data,controller.signal);}finally{clearTimeout(timer);cardControllers.delete(controller);}
}
function syncOwnedCards(){
 if(ownedPromise)return ownedPromise;
 if(!email||!usdtOn)return Promise.resolve();
 const epoch=cardEpoch,owner=email;
 const promise=(async()=>{
  const result=await cardCall('/api/usdt/cards');if(epoch!==cardEpoch||owner!==email)return;
  if(!Array.isArray(result.cards))throw Error('invalid_cards');
  const all=cards();
  for(const server of result.cards){
   if(!/^[a-f0-9]{32}$/.test(server.id||'')||!assetNames[server.asset])continue;
   let c=all.find(c=>c.id===server.id);const known=Boolean(c);
   if(!c){c={id:server.id,invited:false};all.push(c);}
   const newlySent=known&&!c.invited&&server.invited===true;
   Object.assign(c,{sender:owner,amount:String(server.amount||''),asset:server.asset,recipient:String(server.recipient||''),created:Date.parse(server.created)||c.created||0,invited:c.invited===true||server.invited===true});
   if(newlySent)c.noticePending=true;
  }
  keep(all);renderCards();
 })();ownedPromise=promise;
 promise.finally(()=>{if(ownedPromise===promise)ownedPromise=null;}).catch(()=>{});return promise;
}
function scheduleWatch(watch,delay){
 clearTimeout(watch.timer);watch.timer=null;
 if(watch.epoch!==cardEpoch||!email||document.hidden)return;
 watch.timer=setTimeout(()=>{watch.timer=null;void tickCard(watch);},delay);
}
function tickCard(watch){
 if(watch.running)return watch.running;
 if(watch.epoch!==cardEpoch||!email)return Promise.resolve();
 clearTimeout(watch.timer);watch.timer=null;const epoch=cardEpoch,owner=email,id=watch.id;
 watch.running=(async()=>{
  try{
   const st=await cardCall('/api/usdt/cashlink/status',{id});if(epoch!==cardEpoch||owner!==email)return;
   if(!cardStates.has(st.state))throw Error('invalid_card_state');
   update(id,{sender:owner,state:st.state,balanceUnits:Number(st.balanceUnits??st.balance_units)||0,stale:st.stale===true,checkedAt:Date.now()});renderCards();
   let entry=cards().find(c=>c.id===id);
   if(st.state==='ready'&&!st.stale&&!entry.invited&&Date.now()>=(entry.nextInviteAt||0)){
    // Read the persisted invitation flag before retrying a request whose reply might have been lost.
    await syncOwnedCards();if(epoch!==cardEpoch||owner!==email)return;
    entry=cards().find(c=>c.id===id);
    if(!entry.invited&&/^#c=[A-Za-z0-9_-]+\.[A-Za-z0-9_-]{43}$/.test(entry.fragment||'')){
     update(id,{nextInviteAt:Date.now()+60000});
     try{
      const result=await cardCall('/api/usdt/card/'+id+'/invite',{claim_url:location.origin+'/usdt.html?lang='+lang+entry.fragment});
      if(epoch!==cardEpoch||owner!==email)return;
      if(result.invited!==true)throw Error('invitation_unconfirmed');
      update(id,{invited:true,noticePending:true,mailFailed:false});
     }catch(error){
      if(epoch!==cardEpoch||owner!==email)return;
      update(id,{mailFailed:true});try{await syncOwnedCards();}catch{}
     }
    }
   }
  }catch(error){if(epoch===cardEpoch&&owner===email)update(id,{stale:true});}
  finally{
   watch.running=null;
   if(epoch===cardEpoch&&owner===email){renderCards();const c=cards().find(c=>c.id===id);scheduleWatch(watch,terminalCards.has(c?.state)&&!c.stale?60000:6000);}
  }
 })();return watch.running;
}
function watch(id,immediate=false){
 if(!/^[a-f0-9]{32}$/.test(id||''))return;
 if(!watching.has(id))watching.set(id,{id,epoch:cardEpoch,timer:null,running:null});
 const current=watching.get(id);if(immediate)void tickCard(current);else if(current.timer===null&&!current.running)scheduleWatch(current,2500);
}
function nimPending(p){const created=Date.parse(p.created||'');return p.stage!=='sent'&&(!Number.isFinite(created)||Date.now()<created+86400000);}
function scheduleNimCards(){
 clearTimeout(nimTimer);nimTimer=null;
 if(email&&nimOn&&!document.hidden&&[...nimCards.values()].some(nimPending))nimTimer=setTimeout(()=>{void syncNimCards();},30000);
}
function syncNimCards(){
 if(nimPromise)return nimPromise;if(!email||!nimOn)return Promise.resolve();const epoch=cardEpoch,owner=email;
 const promise=(async()=>{try{const result=await cardCall('/api/payments');if(epoch!==cardEpoch||owner!==email)return;for(const p of result.payments||[])if(p.sender===owner)nimCards.set(p.id,p);renderCards();}catch{}finally{if(epoch===cardEpoch)scheduleNimCards();}})();
 nimPromise=promise;promise.finally(()=>{if(nimPromise===promise)nimPromise=null;}).catch(()=>{});return promise;
}
function stopCards(){
 cardEpoch++;for(const w of watching.values())clearTimeout(w.timer);watching.clear();clearTimeout(nimTimer);nimTimer=null;
 for(const c of cardControllers)c.abort();cardControllers.clear();ownedPromise=null;nimPromise=null;nimCards.clear();
}
async function startCards(){
 const epoch=cardEpoch;renderCards();
 for(const c of cards())if(c.sender===email||!c.sender)watch(c.id);
 await Promise.allSettled([syncOwnedCards(),syncNimCards()]);if(epoch!==cardEpoch||!email)return;
 for(const c of cards())if(c.sender===email)watch(c.id);renderCards();
}
function resumeCards(){
 if(!email||document.hidden)return;
 for(const c of cards())if(c.sender===email)watch(c.id,true);
 if([...nimCards.values()].some(nimPending))void syncNimCards();
}
document.addEventListener('visibilitychange',()=>{if(document.hidden){for(const w of watching.values()){clearTimeout(w.timer);w.timer=null;}clearTimeout(nimTimer);}else resumeCards();});
window.addEventListener('pageshow',resumeCards);window.addEventListener('online',resumeCards);
window.addEventListener('storage',event=>{if(event.key===STORE&&email){renderCards();resumeCards();}});

// The browser retains the claim fragment locally. At invitation, Wire receives the full link to validate and e-mail it,
// without persisting it. Claims and refunds are signed locally; their routes receive signatures, not the private link key.
function attach(files) {
 for (const f of files) {
  if (!/^phoenixmail-usdt-claim/.test(f.name)) continue;
  try {
   const card = JSON.parse(atob(f.data)), hash = new URL(card.claimURL).hash;
   if (!/^[0-9a-f]{32}$/.test(card.id) || !/^#c=[A-Za-z0-9_-]+\.[A-Za-z0-9_-]{43}$/.test(hash)) continue;
   const all = cards().filter(c => c.id !== card.id);
   let entry;try{entry=JSON.parse(read('phoenixmail.usdt.sent.v2','[]')).find(e=>e.card?.id===card.id);}catch{}
   all.unshift({ id: card.id, fragment: hash, recipient: String(entry?.protectedRecipient||entry?.to||card.recipient||''),amount:String(card.amount||''),asset:entry?.asset||card.asset,sender:email, invited: false, state:'awaiting_funds',created: Date.now() });
   keep(all);renderCards();watch(card.id);void syncOwnedCards().catch(()=>{});
  } catch { /* not a card of ours */ }
 }
}

// The card library is only fetched where the engine serves it (payments switched on).
function loadCashlink() {
 if (window.PhoenixCashlink) return Promise.resolve();
 return new Promise((ok, no) => { const s = document.createElement('script'); s.src = '/cashlink.js'; s.onload = ok; s.onerror = no; document.head.append(s); });
}

function mountPayments() {
 if (payments) return;
 const addBack = win => {
  const head = win.querySelector('.nimiq-payment-head');
  if (!head || head.querySelector('.wire-window-back')) return;
  win.classList.toggle('wire-coin-picker',choosingCoin);
  const title=head.querySelector('h3');if(title)title.textContent=choosingCoin?t('chooseCoin'):title.textContent.replace(/^⤢\s*/, '');
  const assets=win.querySelector('.nimiq-payment-assets');if(assets)assets.hidden=!choosingCoin;
  win.querySelector('.nimiq-payment-tabs')?.setAttribute('hidden','');
  assets?.querySelectorAll('button').forEach(button=>{const select=button.onclick;button.onclick=()=>{choosingCoin=false;select?.();};});
  if(innerWidth<=640){head.onpointerdown=null;head.removeAttribute('title');}
  const nim = win.querySelector('.nimiq-payment-assets button:first-child');
  if (nim) { nim.disabled = !nimOn; nim.onclick = openNimWindow; }
  const close = head.querySelector('button');
  styleClose(close);
  const back = document.createElement('button'); back.type = 'button'; back.className = 'secondary-btn wire-window-back';
  back.textContent = t('back');
  // Let the copied window remove its own listeners and observers through its close control.
  back.addEventListener('click', () => { close.click(); $('pay').focus(); });
  head.prepend(back);
 };
 $('pay').addEventListener('click', () => { if (!email) return; closePaymentWindow(); if (!usdtOn) openNimWindow(); else{choosingCoin=true;addBack(payments.open('send', 'usdc_polygon'));} });
 // The copied window replaces itself when switching coins; enhance each replacement without editing pm/.
 const windows = new MutationObserver(changes => {
  for (const change of changes) for (const node of change.addedNodes) {
   if (node.nodeType === 1 && node.matches('.nimiq-payment-modal')) node.querySelectorAll('.nimiq-payment-window').forEach(addBack);
  }
 });
 windows.observe(document.body, { childList: true });
 payments = window.PhoenixNimiqPayments.create({ language: () => lang, account: () => 'wire', senderEmail: () => email, attach, status: say,
  brand: { icon: '/brand/phoenix-emblem.webp', alt: 'Phoenix Wire', first: 'Phoenix', second: 'Wire' }, snapshot: () => null, recipient: () => '', bindOTPRecipient: () => true, openRecoveredClaim: () => false });
}

function showSender(verified, who) {
 if(verified){codeSentTo='';codeUntil=0;paintCodeRequest();}
 if(!verified||email&&email!==who)stopCards();
 if(!verified&&nimCache){const cached=nimCache;nimView?.dismiss();cached.flow.destroy();cached.modal.remove();nimCache=null;resumed=false;}
 email = verified ? who : '';
 $('wire-sender').hidden = verified || !enabled; $('wire-ready').hidden = !verified || !enabled;
 $('wire-who').textContent = email;
 if (verified && enabled) {
  mountPayments();void startCards();
  try { if (nimOn && !resumed && (handoff?.sender.toLowerCase()===email.toLowerCase()||/^[a-f0-9]{48}$/.test(sessionStorage.getItem('wire-pending-payment') || ''))) { resumed = true; openNimWindow(); } } catch { /* tab storage may be unavailable */ }
 }
 renderCards();
}

$('wire-language').addEventListener('change', e => setLanguage(e.target.value));
$('wire-theme').addEventListener('click', () => { const v = document.documentElement.dataset.theme === 'light-paper' ? 'dark' : 'light'; applyTheme(v); save('wire-theme', v); });
$('wire-palette').addEventListener('change',event=>{applyPalette(event.target.value);save('wire-palette',paletteMode);});
function paintCodeRequest(){
 const button=$('wire-send-code'),remaining=Math.max(0,Math.ceil((codeUntil-Date.now())/1000));
 button.disabled=codeSending||remaining>0;
 button.textContent=codeSending?t('sendingCode'):codeSentTo?(remaining?t('codeSentButton')+' ('+remaining+' s)':t('resendCode')):t('send');
 button.dataset.state=codeSending?'sending':codeSentTo?'sent':'idle';
 $('wire-code-delivery').textContent=codeSending?t('sendingCode'):codeSentTo?t('codeSent'):'';
 $('wire-code-delivery').hidden=!codeSending&&!codeSentTo;
 clearTimeout(codeTimer);if(remaining&&!codeSending)codeTimer=setTimeout(paintCodeRequest,1000);
}
$('wire-email').addEventListener('input',()=>{
 if(codeSentTo&&$('wire-email').value.trim()!==codeSentTo){codeSentTo='';codeUntil=0;$('wire-code-box').hidden=true;$('wire-code').value='';paintCodeRequest();}
});
$('wire-send-code').addEventListener('click', async () => {
 if(codeSending||Date.now()<codeUntil)return;
 if (!$('wire-email').reportValidity() || !$('wire-email').value) return;
 codeSending=true;$('wire-email').disabled=true;$('wire-verify').disabled=true;paintCodeRequest();
 const controller=new AbortController(),timeout=setTimeout(()=>controller.abort(),30000);
 try {const to=$('wire-email').value.trim();await api('/api/sender/request', { email: to, language: lang },controller.signal);codeSentTo=to;codeUntil=Date.now()+30000;$('wire-code-box').hidden = false;say(t('codeSent'));$('wire-code').focus();}
 catch {codeSentTo='';codeUntil=0;say(t('fail'));}
 finally{clearTimeout(timeout);codeSending=false;$('wire-email').disabled=false;$('wire-verify').disabled=false;paintCodeRequest();}
});
$('wire-verify').addEventListener('click', async () => {
 try { const r = await api('/api/sender/verify', { code: $('wire-code').value.trim() }); $('wire-code').value = ''; $('wire-code-box').hidden = true; say(''); showSender(true, r.email); }
 catch { say(t('wrong')); }
});
$('wire-logout').addEventListener('click', async () => {
 const button=$('wire-logout');if(button.disabled)return;button.disabled=true;
 const controller=new AbortController(),timeout=setTimeout(()=>controller.abort(),30000);
 try{
  await api('/api/logout', {},controller.signal);closeWalletChoice();closePaymentWindow();showSender(false);
  handoff=null;try{sessionStorage.removeItem('wire-nim-handoff');}catch{}
  codeSentTo='';codeUntil=0;$('wire-email').value='';$('wire-code').value='';$('wire-code-box').hidden=true;paintCodeRequest();say('');$('wire-email').focus();
 }catch{say(t('fail'));}finally{clearTimeout(timeout);button.disabled=false;}
});

const savedPalette=read('wire-palette','');
applyPalette(savedPalette==='live'||palettes.includes(savedPalette)?savedPalette:({orange:'phoenixfire',violet:'aurora',teal:'ocean'}[read('wire-accent','')]||(savedPalette?'phoenixfire':'live')));
applyTheme(read('wire-theme', 'dark'));
setLanguage(preference);
(async () => {
 try {
  const config = await api('/api/config');
  usdtOn = config.mode === 'integrated' && (config.assets || []).some(a => /^USD[CT]_/.test(a));
  nimOn = config.mode === 'integrated' && config.payments_enabled === true && (config.assets || []).includes('NIM');
  enabled = usdtOn || nimOn;
  if (!enabled) { say(t('off')); return; }
  if (usdtOn) await loadCashlink();
  const session = await api('/api/session');
  showSender(Boolean(session.verified), session.email);
 } catch { say(t('off')); }
})();
