// Phoenix Wire's page: PhoenixMail's own payment window (/pm/*), used as it is. This file is only what is Wire's: who the sender is, and
// what happens to a card once the window has made it (watch the money, then e-mail the recipient the claim link, from this browser).
import { resolveLanguage } from './model.js?v=0.3.0';
import { mountNimFlow } from './flow.js?v=0.4.5';

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
function read(key, fallback) { try { return localStorage.getItem(key) || fallback; } catch { return fallback; } }
function save(key, value) { try { localStorage.setItem(key, value); } catch { /* storage may be unavailable */ } }
let preference = read('wire-language', 'auto'), lang = 'en', email = '', payments = null, usdtOn = false, nimOn = false, enabled = false, nimView = null, resumed = false;
let nimCache=null;
const t = key => (words[lang] || words.en)[key];

async function api(path, data) {
 const options = { credentials: 'same-origin', cache: 'no-store', redirect: 'error' };
 if (data !== undefined) { options.method = 'POST'; options.headers = { 'Content-Type': 'application/json' }; options.body = JSON.stringify(data); }
 const response = await fetch(path, options);
 let body = {}; try { body = await response.json(); } catch { /* an answer without a body */ }
 if (!response.ok) throw Object.assign(Error(body.code || body.error || 'unavailable'), { code: body.code || body.error, status: response.status });
 return body;
}
const say = text => { $('wire-status').textContent = text; };

function paint() {
 document.documentElement.lang = lang;
 $('wire-title').textContent = t('title'); $('wire-sub').textContent = t('sub');
 $('wire-email-label').textContent = t('email'); $('wire-send-code').textContent = t('send'); $('wire-code-label').textContent = t('code'); $('wire-verify').textContent = t('verify');
 $('wire-as').textContent = t('as'); $('wire-logout').textContent = t('out');
 $('wire-language').value = preference;
 $('wire-pay-label').textContent = t('pay');
 paintTheme();
 if (nimCache) paintNim(nimCache.modal);
 window.dispatchEvent(new Event('wire-language'));
 $('wire-session-help').textContent = t('sessionHelp'); $('wire-session-confirmed').textContent = t('sessionConfirmed');
 document.querySelectorAll('.wire-window-back').forEach(button => { button.textContent = '← ' + t('back'); });
 document.querySelectorAll('[data-wallet-text]').forEach(el=>{el.textContent=t(el.dataset.walletText);});
}
// iOS cannot reliably disclose whether another app is installed. Ask, without timers or blur heuristics.
const payLink='https://nimpay.app/miniapps/open/wire.phoenixmail.ca';
const iosStore='https://apps.apple.com/app/nimiq-pay/id6471844738';
const androidStore='https://play.google.com/store/apps/details?id=com.nimiq.pay';
let choiceCallback=null,choiceFocus=null,choiceParent=null;
function closeWalletChoice(){
 $('wire-nim-choice').hidden=true;choiceCallback=null;choiceParent?.removeAttribute('inert');choiceParent=null;choiceFocus?.focus();
}
function chooseWallet(useHub){
 choiceCallback=useHub;choiceFocus=document.activeElement;choiceParent=document.querySelector('.wire-nim-window');choiceParent?.setAttribute('inert','');
 $('wire-choice-status').hidden=true;$('wire-choice-stores').hidden=true;$('wire-nim-choice').hidden=false;
 $('wire-choice-open').href=payLink;
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
function paintNim(root) {
 root.querySelectorAll('[data-nim-text]').forEach(el => { el.textContent = t(el.dataset.nimText); });
 root.querySelector('.wire-nim-close').setAttribute('aria-label', t('back'));
}
function closePaymentWindow() {
 document.querySelector('.nimiq-payment-head button:last-child')?.click();
}
function openNimWindow() {
 if (!email || !nimOn) return;
 if(nimView)return;
 closePaymentWindow();
 if(nimCache){
  document.body.append(nimCache.modal);document.addEventListener('keydown',nimCache.escape);nimView=nimCache.view;
  paintNim(nimCache.modal);nimCache.flow.resume();nimCache.modal.querySelector('.wire-window-back').focus();return;
 }
 const modal = $('wire-nim-template').content.firstElementChild.cloneNode(true);
 const root = modal.querySelector('.wire-nim-host'), win = modal.querySelector('.nimiq-payment-window');
 const row = modal.querySelector('.wire-nim-assets');
 for (const [asset, label] of [['nim','NIM · Nimiq'],['usdc_polygon','USDC · Polygon'],['usdt_polygon','USDT · Polygon'],['usdt_ethereum','USDT · Ethereum']]) {
  const button = document.createElement('button'); button.type = 'button'; button.className = 'secondary-btn'; button.textContent = label;
  button.setAttribute('aria-pressed', String(asset === 'nim')); button.disabled = asset !== 'nim' && !usdtOn;
  button.onclick = () => { if (asset === 'nim') return; dismiss(); payments.open('send', asset); }; row.append(button);
 }
 const back = modal.querySelector('.wire-window-back'); back.textContent = '← ' + t('back');
 let flow;
 // Keep the authenticated NIM tracker alive when the window is closed or another coin is selected.
 const dismiss = () => { closeWalletChoice();modal.remove(); document.removeEventListener('keydown', escape); nimView = null; $('pay').focus(); };
 const escape = e => { if (e.key === 'Escape') { e.preventDefault(); dismiss(); } };
 back.onclick = dismiss; modal.querySelector('.wire-nim-close').onclick = dismiss;
 document.addEventListener('keydown', escape);
 document.body.append(modal); nimView = { root: modal, dismiss };
 paintNim(modal);flow=mountNimFlow(root,{embedded:true,chooseWallet,onStatus:(_p,message)=>say(message),onClose:()=>{const message=root.querySelector('#nim-progress-message').textContent;dismiss();say(message);}});
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
const keep = list => save(STORE, JSON.stringify(list.slice(0, 100)));
const update = (id, patch) => { const all = cards(); const c = all.find(x => x.id === id); if (c) { Object.assign(c, patch); keep(all); } };
const watching = new Set();

function watch(id) {
 if (watching.has(id)) return; watching.add(id);
 let tries = 0;
 const tick = async () => {
  const entry = cards().find(c => c.id === id);
  if (!entry || entry.invited) { watching.delete(id); return; }
  let again = true;
  try {
   const st = await api('/api/usdt/cashlink/status', { id });
   if (st.state === 'ready') {
    say(t('funded'));
    try {
     await api('/api/usdt/card/' + id + '/invite', { claim_url: location.origin + '/usdt.html?lang=' + lang + entry.fragment });
     update(id, { invited: true }); say(t('sent').replace('{email}', entry.recipient)); again = false;
    } catch (e) { if (e.code !== 'rate_limited') say(t('retry')); again = ++tries < 30; }
   } else if (['paid', 'withdrawn', 'cancelled', 'expired'].includes(st.state)) again = false;
  } catch { again = ++tries < 200; }
  if (again) setTimeout(tick, 6000); else watching.delete(id);
 };
 setTimeout(tick, 2500);
}

// The browser retains the claim fragment locally. At invitation, Wire receives the full link to validate and e-mail it,
// without persisting it. Claims and refunds are signed locally; their routes receive signatures, not the private link key.
function attach(files) {
 for (const f of files) {
  if (!/^phoenixmail-usdt-claim/.test(f.name)) continue;
  try {
   const card = JSON.parse(atob(f.data)), hash = new URL(card.claimURL).hash;
   if (!/^[0-9a-f]{32}$/.test(card.id) || !/^#c=[A-Za-z0-9_-]+\.[A-Za-z0-9_-]{43}$/.test(hash)) continue;
   const all = cards().filter(c => c.id !== card.id);
   all.unshift({ id: card.id, fragment: hash, recipient: String(card.recipient || ''), invited: false, created: Date.now() });
   keep(all); watch(card.id);
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
  const nim = win.querySelector('.nimiq-payment-assets button:first-child');
  if (nim) { nim.disabled = !nimOn; nim.onclick = openNimWindow; }
  const close = head.querySelector('button');
  const back = document.createElement('button'); back.type = 'button'; back.className = 'secondary-btn wire-window-back';
  back.textContent = '← ' + t('back');
  // Let the copied window remove its own listeners and observers through its close control.
  back.addEventListener('click', () => { close.click(); $('pay').focus(); });
  head.prepend(back);
 };
 $('pay').addEventListener('click', () => { if (!email) return; closePaymentWindow(); if (!usdtOn) openNimWindow(); else addBack(payments.open('send', 'usdc_polygon')); });
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
 if(!verified&&nimCache){nimView?.dismiss();nimCache.flow.destroy();nimCache.modal.remove();nimCache=null;resumed=false;}
 email = verified ? who : '';
 $('wire-sender').hidden = verified || !enabled; $('wire-ready').hidden = !verified || !enabled;
 $('wire-who').textContent = email;
 if (verified && enabled) {
  mountPayments(); for (const c of cards()) if (!c.invited) watch(c.id);
  try { if (nimOn && !resumed && /^[a-f0-9]{48}$/.test(sessionStorage.getItem('wire-pending-payment') || '')) { resumed = true; openNimWindow(); } } catch { /* tab storage may be unavailable */ }
 }
}

$('wire-language').addEventListener('change', e => setLanguage(e.target.value));
$('wire-theme').addEventListener('click', () => { const v = document.documentElement.dataset.theme === 'light-paper' ? 'dark' : 'light'; applyTheme(v); save('wire-theme', v); });
$('wire-send-code').addEventListener('click', async () => {
 if (!$('wire-email').reportValidity() || !$('wire-email').value) return;
 try { await api('/api/sender/request', { email: $('wire-email').value.trim(), language: lang }); $('wire-code-box').hidden = false; say(t('codeSent')); } catch { say(t('fail')); }
});
$('wire-verify').addEventListener('click', async () => {
 try { const r = await api('/api/sender/verify', { code: $('wire-code').value.trim() }); $('wire-code').value = ''; $('wire-code-box').hidden = true; say(''); showSender(true, r.email); }
 catch { say(t('wrong')); }
});
$('wire-logout').addEventListener('click', async () => { try { await api('/api/logout', {}); } catch { /* the session is gone anyway */ } closePaymentWindow(); showSender(false); say(''); });

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
