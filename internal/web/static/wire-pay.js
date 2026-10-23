// Phoenix Wire's page: PhoenixMail's own payment window (/pm/*), used as it is. This file is only what is Wire's: who the sender is, and
// what happens to a card once the window has made it (watch the money, then e-mail the recipient the claim link, from this browser).
import { resolveLanguage } from './model.js?v=0.3.0';

window.PhoenixBrandName = 'Phoenix Wire'; // the payment window's texts name the app they run in
const $ = id => document.getElementById(id);
const STORE = 'wire-usdt-cards', words = {
 en: { title: 'One payment. One email.', sub: 'Send USDC or USDT to anyone with an email address. Only they can receive it, after a code sent to that address.', email: 'Your email', send: 'Send me a code', code: 'Six-digit code', verify: 'Confirm', as: 'Signed in as', out: 'Change sender', pay: 'Payments', codeSent: 'Code sent. Check your email.', wrong: 'This code is not valid.', off: 'Payments are not switched on here.', funded: 'Card funded. Sending the invitation…', sent: 'The card was sent to {email}. It stays yours until it is claimed.', retry: 'The e-mail could not be sent yet. Trying again…', classic: 'Send NIM', fail: 'Something went wrong. Try again.' },
 fr: { title: 'Un paiement. Un courriel.', sub: 'Envoyez des USDC ou des USDT à toute personne qui a une adresse courriel. Seule cette personne peut les recevoir, après un code envoyé à cette adresse.', email: 'Votre courriel', send: 'M’envoyer un code', code: 'Code à six chiffres', verify: 'Confirmer', as: 'Connecté en tant que', out: 'Changer d’expéditeur', pay: 'Paiements', codeSent: 'Code envoyé. Vérifiez votre courriel.', wrong: 'Ce code n’est pas valide.', off: 'Les paiements ne sont pas activés ici.', funded: 'Carte alimentée. Envoi de l’invitation…', sent: 'La carte a été envoyée à {email}. Elle reste à vous tant qu’elle n’est pas réclamée.', retry: 'Le courriel n’a pas pu partir. Nouvel essai…', classic: 'Envoyer des NIM', fail: 'Une erreur est survenue. Réessayez.' },
 es: { title: 'Un pago. Un correo.', sub: 'Envía USDC o USDT a cualquier persona con una dirección de correo. Solo ella puede recibirlo, tras un código enviado a esa dirección.', email: 'Tu correo', send: 'Envíame un código', code: 'Código de seis dígitos', verify: 'Confirmar', as: 'Sesión de', out: 'Cambiar de remitente', pay: 'Pagos', codeSent: 'Código enviado. Revisa tu correo.', wrong: 'Este código no es válido.', off: 'Los pagos no están activados aquí.', funded: 'Tarjeta financiada. Enviando la invitación…', sent: 'La tarjeta se envió a {email}. Sigue siendo tuya hasta que se reclame.', retry: 'El correo no pudo enviarse aún. Reintentando…', classic: 'Enviar NIM', fail: 'Algo salió mal. Inténtalo de nuevo.' },
 de: { title: 'Eine Zahlung. Eine E-Mail.', sub: 'Sende USDC oder USDT an jede Person mit einer E-Mail-Adresse. Nur sie kann sie empfangen, nach einem Code an diese Adresse.', email: 'Deine E-Mail', send: 'Code senden', code: 'Sechsstelliger Code', verify: 'Bestätigen', as: 'Angemeldet als', out: 'Absender wechseln', pay: 'Zahlungen', codeSent: 'Code gesendet. Prüfe deine E-Mail.', wrong: 'Dieser Code ist ungültig.', off: 'Zahlungen sind hier nicht aktiviert.', funded: 'Karte aufgeladen. Die Einladung wird gesendet…', sent: 'Die Karte wurde an {email} gesendet. Sie bleibt deine, bis sie eingelöst wird.', retry: 'Die E-Mail konnte noch nicht gesendet werden. Neuer Versuch…', classic: 'NIM senden', fail: 'Etwas ist schiefgelaufen. Versuche es erneut.' },
 zh: { title: '一笔付款。一封邮件。', sub: '把 USDC 或 USDT 发给任何有邮箱的人。只有对方在输入发送到该邮箱的验证码后才能收款。', email: '你的邮箱', send: '给我发验证码', code: '六位验证码', verify: '确认', as: '登录身份', out: '更换发送者', pay: '付款', codeSent: '验证码已发送，请查收邮件。', wrong: '验证码无效。', off: '此处未启用付款。', funded: '卡片已充值。正在发送邀请…', sent: '卡片已发送到 {email}。在被领取之前它仍属于你。', retry: '邮件暂时无法发送。正在重试…', classic: '发送 NIM', fail: '出错了，请重试。' },
};

function read(key, fallback) { try { return localStorage.getItem(key) || fallback; } catch { return fallback; } }
function save(key, value) { try { localStorage.setItem(key, value); } catch { /* storage may be unavailable */ } }
let preference = read('wire-language', 'auto'), lang = 'en', email = '', payments = null, usdtOn = false;
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
 $('wire-as').textContent = t('as'); $('wire-logout').textContent = t('out'); $('wire-classic').textContent = t('classic');
 $('wire-language').value = preference;
 $('wire-pay-label').textContent = t('pay');
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

// The window hands over the files it would attach to an e-mail. The claim link is in one of them: it stays here, in this browser.
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
 $('pay').addEventListener('click', () => payments.open('send', 'usdc_polygon')); // NIM and payment requests stay with the earlier page until they move to this window
 payments = window.PhoenixNimiqPayments.create({ language: () => lang, account: () => 'wire', senderEmail: () => email, attach, status: say,
  brand: { icon: '/brand/phoenix-emblem.webp', alt: 'Phoenix Wire', first: 'Phoenix', second: 'Wire' }, snapshot: () => null, recipient: () => '', bindOTPRecipient: () => true, openRecoveredClaim: () => false });
}

function showSender(verified, who) {
 email = verified ? who : '';
 $('wire-sender').hidden = verified || !usdtOn; $('wire-ready').hidden = !verified || !usdtOn;
 $('wire-who').textContent = email;
 if (verified && usdtOn) { mountPayments(); for (const c of cards()) if (!c.invited) watch(c.id); }
}

$('wire-language').addEventListener('change', e => setLanguage(e.target.value));
$('wire-theme').addEventListener('click', () => { const v = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark'; document.documentElement.dataset.theme = v; save('wire-theme', v); });
$('wire-send-code').addEventListener('click', async () => {
 if (!$('wire-email').reportValidity() || !$('wire-email').value) return;
 try { await api('/api/sender/request', { email: $('wire-email').value.trim(), language: lang }); $('wire-code-box').hidden = false; say(t('codeSent')); } catch { say(t('fail')); }
});
$('wire-verify').addEventListener('click', async () => {
 try { const r = await api('/api/sender/verify', { code: $('wire-code').value.trim() }); $('wire-code').value = ''; $('wire-code-box').hidden = true; say(''); showSender(true, r.email); }
 catch { say(t('wrong')); }
});
$('wire-logout').addEventListener('click', async () => { try { await api('/api/logout', {}); } catch { /* the session is gone anyway */ } showSender(false); say(''); });

document.documentElement.dataset.theme = read('wire-theme', 'dark') === 'light' ? 'light' : 'dark';
setLanguage(preference);
(async () => {
 try {
  const config = await api('/api/config');
  usdtOn = config.mode === 'integrated' && (config.assets || []).some(a => /^USD[CT]_/.test(a));
  if (!usdtOn) { say(t('off')); return; }
  await loadCashlink();
  const session = await api('/api/session');
  showSender(Boolean(session.verified), session.email);
 } catch { say(t('off')); }
})();
