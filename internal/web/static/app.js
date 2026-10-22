import { assets, previewAmount, resolveLanguage } from './model.js?v=0.2.2';

const messages = {
  fr: ['Clair / sombre','Un paiement. Un courriel.','Créez une carte, envoyez un paiement. Simplement.','Aperçu local : aucun paiement ni courriel n’est envoyé.','Votre carte de paiement','Votre courriel','Courriel du destinataire','Monnaie','Montant','Petit mot (facultatif)','Le parcours prévu déverrouillera le paiement après vérification du courriel par code à six chiffres.','Prévisualiser la carte','Aperçu de la carte','MAQUETTE','Code par courriel · parcours à intégrer','Pas de QR réclamable, de clé ou de fonds dans cet aperçu.','Aperçu mis à jour. Aucun envoi effectué.','Montant invalide ou trop de décimales.'],
  en: ['Light / dark','One payment. One email.','Create a card, send a payment. Simply.','Local preview: no payment or email is sent.','Your payment card','Your email','Recipient’s email','Currency','Amount','A note (optional)','The planned flow will unlock the payment after email verification with a six-digit code.','Preview the card','Card preview','PREVIEW','Email code · integration pending','No claimable QR, key, or funds in this preview.','Preview updated. Nothing was sent.','Invalid amount or too many decimal places.'],
  es: ['Claro / oscuro','Un pago. Un correo.','Crea una tarjeta y envía un pago. Así de simple.','Vista previa local: no se envían pagos ni correos.','Tu tarjeta de pago','Tu correo','Correo del destinatario','Moneda','Importe','Una nota (opcional)','El flujo previsto desbloqueará el pago tras verificar el correo con un código de seis dígitos.','Vista previa de la tarjeta','Vista previa','MAQUETA','Código por correo · integración pendiente','Sin QR canjeable, claves ni fondos en esta vista previa.','Vista previa actualizada. No se ha enviado nada.','Importe no válido o demasiados decimales.'],
  de: ['Hell / dunkel','Eine Zahlung. Eine E-Mail.','Karte erstellen, Zahlung senden. Ganz einfach.','Lokale Vorschau: Es werden weder Zahlungen noch E-Mails gesendet.','Deine Zahlungskarte','Deine E-Mail','E-Mail des Empfängers','Währung','Betrag','Eine Nachricht (optional)','Der geplante Ablauf gibt die Zahlung nach E-Mail-Bestätigung mit einem sechsstelligen Code frei.','Kartenvorschau anzeigen','Kartenvorschau','ENTWURF','E-Mail-Code · Integration ausstehend','Kein einlösbarer QR-Code, Schlüssel oder Guthaben in dieser Vorschau.','Vorschau aktualisiert. Nichts wurde gesendet.','Ungültiger Betrag oder zu viele Nachkommastellen.'],
  zh: ['浅色 / 深色','一笔付款，一封邮件。','创建卡片，轻松发送付款。','本地预览：不会发送付款或邮件。','您的付款卡片','您的邮箱','收款人邮箱','币种','金额','留言（可选）','计划在通过六位邮件验证码验证后解锁付款。','预览卡片','卡片预览','设计预览','邮件验证码 · 尚待集成','此预览不含可领取的二维码、密钥或资金。','预览已更新，未发送任何内容。','金额无效或小数位过多。']
};
const keys = ['theme','title','subtitle','notice','create','sender','recipient','currency','amount','message','security','preview','previewTitle','sample','locked','previewNote','updated','invalid'];
const $ = id => document.getElementById(id);
let lang = 'en';
const automaticLabels = { fr: 'Automatique', en: 'Automatic', es: 'Automático', de: 'Automatisch', zh: '自动' };
const slogans = { fr:'Rapide. Simple. Sécuritaire.', en:'Fast. Simple. Secure.', es:'Rápido. Sencillo. Seguro.', de:'Schnell. Einfach. Sicher.', zh:'快速。简单。安全。' };
const cardLabels = {
 fr:{emptyPreview:'Créez une carte pour la prévisualiser.',cardHeading:'Paiement sécurisé',cardAction:'Vérifier mon code',locked:'Un code envoyé par courriel permet de déverrouiller le paiement.'},
 en:{emptyPreview:'Create a card to preview it.',cardHeading:'Protected payment',cardAction:'Verify my code',locked:'An emailed code unlocks this payment.'},
 es:{emptyPreview:'Crea una tarjeta para previsualizarla.',cardHeading:'Pago protegido',cardAction:'Verificar mi código',locked:'Un código enviado por correo desbloquea este pago.'},
 de:{emptyPreview:'Erstelle eine Karte für die Vorschau.',cardHeading:'Geschützte Zahlung',cardAction:'Code bestätigen',locked:'Ein E-Mail-Code gibt diese Zahlung frei.'},
 zh:{emptyPreview:'创建卡片以查看预览。',cardHeading:'受保护的付款',cardAction:'验证我的验证码',locked:'使用邮件验证码解锁此付款。'}
};
function t(key) { return cardLabels[lang][key] || (key === 'subtitle' ? slogans[lang] : key === 'automatic' ? automaticLabels[lang] : messages[lang][keys.indexOf(key)]); }
function readPreference(key, fallback) { try { return localStorage.getItem(key) || fallback; } catch { return fallback; } }
function savePreference(key, value) { try { localStorage.setItem(key, value); } catch { /* Storage may be unavailable in embedded browsers. */ } }
let languagePreference = readPreference('wire-language', 'auto');
if (languagePreference !== 'auto' && !Object.hasOwn(messages, languagePreference)) languagePreference = 'auto';
function translate(value) {
  const browser = typeof navigator === 'undefined' ? {} : navigator;
  lang = resolveLanguage(value, browser.languages?.length ? browser.languages : [browser.language]);
  document.documentElement.lang = lang;
  $('language').value = languagePreference;
  document.querySelectorAll('[data-i18n]').forEach(el => { el.textContent = t(el.dataset.i18n); });
  $('feedback').textContent = '';
  globalThis.dispatchEvent?.(new CustomEvent('wire-language'));
}
translate(languagePreference);
const theme = readPreference('wire-theme', 'dark');
document.documentElement.dataset.theme = theme === 'light' ? 'light' : 'dark';
const palettes = ['aurora','phoenixfire','ocean'];
const oldPalettes = {orange:'phoenixfire',violet:'aurora',teal:'ocean'};
const savedPalette = readPreference('wire-palette', '');
function applyPalette(value) {
  const palette=palettes.includes(value)?value:'phoenixfire';
  $('accent').value=palette;
  document.documentElement.dataset.palette=palette;
  document.documentElement.dataset.accent=palette;
}
applyPalette(palettes.includes(savedPalette)?savedPalette:(oldPalettes[readPreference('wire-accent','orange')]||'phoenixfire'));
$('language').addEventListener('change', e => {
  languagePreference = e.target.value === 'auto' || Object.hasOwn(messages, e.target.value) ? e.target.value : 'auto';
  translate(languagePreference);
  savePreference('wire-language', languagePreference);
});
globalThis.addEventListener?.('languagechange', () => { if (languagePreference === 'auto') translate('auto'); });
$('theme').addEventListener('click', () => {
  const value = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
  document.documentElement.dataset.theme = value;
  savePreference('wire-theme', value);
});
$('accent').addEventListener('change', e => { applyPalette(e.target.value); savePreference('wire-palette', document.documentElement.dataset.palette); });
function renderCard(amount) {
  const id = $('currency').value;
  const asset = assets[id];
  $('payment-card').dataset.currency = id;
  $('coin').textContent = '';
  const image=document.createElement('img');image.width=image.height=80;
  image.src=id==='NIM'?'/brand/nimiq-logo.svg?v=0.2.2':'/brand/usdt.svg?v=0.2.2';
  image.alt=id==='NIM'?'Nimiq Nimicon':'Tether USDT';$('coin').append(image);
  // An illustrative public address only; this preview has no seed or funding intent.
  if(id==='NIM')globalThis.PhoenixNimiqIdenticons?.paint(image,'NQ08ACT8T0FEPTG8P5RLH2S3QGXHV15RNVXY');
  $('card-unit').textContent = asset.unit;
  $('card-network').textContent = asset.network;
  $('card-amount').textContent = amount;
  // User content is text only. Emails and messages are never stored or transmitted.
  $('card-recipient').textContent = $('recipient').value;
  $('card-message').textContent = $('message').value;
}
$('card-form').addEventListener('submit', e => {
  e.preventDefault();
  try { renderCard(previewAmount($('amount').value, $('currency').value)); $('preview-empty').hidden=true;$('payment-card').hidden=false;$('preview-title').hidden=false;$('preview-note').hidden=false;$('feedback').textContent = t('updated'); }
  catch { $('feedback').textContent = t('invalid'); }
});
$('currency').addEventListener('change', () => {
  try { renderCard(previewAmount($('amount').value, $('currency').value)); $('feedback').textContent = ''; }
  catch { $('feedback').textContent = t('invalid'); }
});
