import { mountAdmin,adminText } from './admin.js?v=0.4.41';
import { nimFunded,unpaidCard,recoverableCard,historyEntries,historyPending,senderHistoryText,nimHistoryLabel,cancelUnpaidCard } from './model.js?v=0.4.30';
// Phoenix Wire's page: PhoenixMail's own payment window (/pm/*), used as it is. This file is only what is Wire's: who the sender is, and
// what happens to a card once the window has made it (watch the money, then e-mail the recipient the claim link, from this browser).
import { resolveLanguage,renderCardPreview,assetMark } from './model.js?v=0.4.30';
import { mountNimFlow } from './flow.js?v=0.4.52';
import { flowText } from './flow-text.js?v=0.4.50';

// CSS can reveal the page without scripts; clear the startup attribute so
// visibility checks and keyboard focus traps see the rendered controls.
document.body.hidden = false;
window.PhoenixBrandName = 'Phoenix Wire'; // the payment window's texts name the app they run in
const $ = id => document.getElementById(id);
const STORE = 'wire-usdt-cards', words = {
 en: { nimRecipient: "Recipient email", nimAmount: "Amount (NIM)", nimMessage: "Message (optional)", nimCard: "Your Nimiq card", nimPreview: "Card preview", nimEmpty: "Create a card to preview it.", nimProtected: "Protected", lightTheme: "Switch to light theme", darkTheme: "Switch to dark theme", sessionHelp: "You confirm your email once. It stays confirmed for 24 hours on this device.", sessionConfirmed: "Confirmed for 24 hours.", back: "Back to PhoenixWire", title: 'One payment. One email.', sub: "Send NIM, USDC or USDT to anyone with an email address. Only they can receive it, after a code sent to that address.", email: 'Your email', send: 'Send me a code', code: 'Six-digit code', verify: 'Confirm', as: 'Signed in as', out: 'Change sender', pay: 'Payments', codeSent: 'Code sent. Check your email.', wrong: 'This code is not valid.', off: 'Payments are not switched on here.', funded: 'Card funded. Sending the invitation…', sent: 'The card was sent to {email}. It stays yours until it is claimed.', retry: 'The e-mail could not be sent yet. Trying again…', classic: 'Send NIM', fail: 'Something went wrong. Try again.' },
 fr: { nimRecipient: "Courriel du destinataire", nimAmount: "Montant (NIM)", nimMessage: "Message (facultatif)", nimCard: "Votre carte Nimiq", nimPreview: "Aperçu de la carte", nimEmpty: "Créez une carte pour la prévisualiser.", nimProtected: "Protégé", lightTheme: "Passer au thème clair", darkTheme: "Passer au thème sombre", sessionHelp: "Vous confirmez votre courriel une seule fois : il reste confirmé 24 heures sur cet appareil.", sessionConfirmed: "Confirmé pour 24 h.", back: "Retour à PhoenixWire", title: 'Un paiement. Un courriel.', sub: "Envoyez des NIM, des USDC ou des USDT à toute personne qui a une adresse courriel. Seule cette personne peut les recevoir, après un code envoyé à cette adresse.", email: 'Votre courriel', send: 'M’envoyer un code', code: 'Code à six chiffres', verify: 'Confirmer', as: 'Connecté en tant que', out: 'Changer d’expéditeur', pay: 'Paiements', codeSent: 'Code envoyé. Vérifiez votre courriel.', wrong: 'Ce code n’est pas valide.', off: 'Les paiements ne sont pas activés ici.', funded: 'Carte alimentée. Envoi de l’invitation…', sent: 'La carte a été envoyée à {email}. Elle reste à vous tant qu’elle n’est pas réclamée.', retry: 'Le courriel n’a pas pu partir. Nouvel essai…', classic: 'Envoyer des NIM', fail: 'Une erreur est survenue. Réessayez.' },
 es: { nimRecipient: "Correo del destinatario", nimAmount: "Monto (NIM)", nimMessage: "Mensaje (opcional)", nimCard: "Tu tarjeta Nimiq", nimPreview: "Vista previa de la tarjeta", nimEmpty: "Crea una tarjeta para verla.", nimProtected: "Protegido", lightTheme: "Cambiar al tema claro", darkTheme: "Cambiar al tema oscuro", sessionHelp: "Confirmas tu correo una sola vez: queda confirmado 24 horas en este dispositivo.", sessionConfirmed: "Confirmado por 24 horas.", back: "Volver a PhoenixWire", title: 'Un pago. Un correo.', sub: "Envía NIM, USDC o USDT a cualquier persona con una dirección de correo. Solo ella puede recibirlo, tras un código enviado a esa dirección.", email: 'Tu correo', send: 'Envíame un código', code: 'Código de seis dígitos', verify: 'Confirmar', as: 'Sesión de', out: 'Cambiar de remitente', pay: 'Pagos', codeSent: 'Código enviado. Revisa tu correo.', wrong: 'Este código no es válido.', off: 'Los pagos no están activados aquí.', funded: 'Tarjeta financiada. Enviando la invitación…', sent: 'La tarjeta se envió a {email}. Sigue siendo tuya hasta que se reclame.', retry: 'El correo no pudo enviarse aún. Reintentando…', classic: 'Enviar NIM', fail: 'Algo salió mal. Inténtalo de nuevo.' },
 de: { nimRecipient: "E-Mail des Empfängers", nimAmount: "Betrag (NIM)", nimMessage: "Nachricht (optional)", nimCard: "Deine Nimiq-Karte", nimPreview: "Kartenvorschau", nimEmpty: "Erstelle eine Karte für die Vorschau.", nimProtected: "Geschützt", lightTheme: "Zum hellen Design wechseln", darkTheme: "Zum dunklen Design wechseln", sessionHelp: "Du bestätigst deine E-Mail nur einmal: Sie bleibt 24 Stunden auf diesem Gerät bestätigt.", sessionConfirmed: "24 Stunden bestätigt.", back: "Zurück zu PhoenixWire", title: 'Eine Zahlung. Eine E-Mail.', sub: "Sende NIM, USDC oder USDT an jede Person mit einer E-Mail-Adresse. Nur sie kann sie empfangen, nach einem Code an diese Adresse.", email: 'Deine E-Mail', send: 'Code senden', code: 'Sechsstelliger Code', verify: 'Bestätigen', as: 'Angemeldet als', out: 'Absender wechseln', pay: 'Zahlungen', codeSent: 'Code gesendet. Prüfe deine E-Mail.', wrong: 'Dieser Code ist ungültig.', off: 'Zahlungen sind hier nicht aktiviert.', funded: 'Karte aufgeladen. Die Einladung wird gesendet…', sent: 'Die Karte wurde an {email} gesendet. Sie bleibt deine, bis sie eingelöst wird.', retry: 'Die E-Mail konnte noch nicht gesendet werden. Neuer Versuch…', classic: 'NIM senden', fail: 'Etwas ist schiefgelaufen. Versuche es erneut.' },
 zh: { nimRecipient: "收款人邮箱", nimAmount: "金额（NIM）", nimMessage: "留言（可选）", nimCard: "你的 Nimiq 卡片", nimPreview: "卡片预览", nimEmpty: "创建卡片以查看预览。", nimProtected: "受保护", lightTheme: "切换为浅色主题", darkTheme: "切换为深色主题", sessionHelp: "你只需确认一次邮箱：在此设备上 24 小时内保持已确认。", sessionConfirmed: "已确认 24 小时。", back: "返回 PhoenixWire", title: '一笔付款。一封邮件。', sub: "把 NIM、USDC 或 USDT 发给任何有邮箱的人。只有对方在输入发送到该邮箱的验证码后才能收款。", email: '你的邮箱', send: '给我发验证码', code: '六位验证码', verify: '确认', as: '登录身份', out: '更换发送者', pay: '付款', codeSent: '验证码已发送，请查收邮件。', wrong: '验证码无效。', off: '此处未启用付款。', funded: '卡片已充值。正在发送邀请…', sent: '卡片已发送到 {email}。在被领取之前它仍属于你。', retry: '邮件暂时无法发送。正在重试…', classic: '发送 NIM', fail: '出错了，请重试。' },
};


const handoffWords={
  "fr": {
    "miniAppOpen": "Utiliser Nimpay",
    "browserOpen": "Utiliser Nimiq Wallet (navigateur)",
    "handoffDone": "Terminé",
    "handoffCancelled": "Carte annulée.",
    "handoffExitHelp": "Tu peux revenir à ton navigateur avec la flèche ‹ en haut à gauche de l’iPhone, ou fermer cette fenêtre avec ✕.",
    "handoffExpired": "Le lien est invalide ou a expiré. Rouvrez le lien depuis Safari.",
    "handoffFailed": "Impossible d’ouvrir Nimpay. Réessayez ou utilisez Nimiq Wallet.",
    "handoffConflict": "Un autre expéditeur est connecté ici. Déconnectez-vous, puis rouvrez le lien depuis Safari.",
    "handoffNotFundable": "Cette carte ne peut plus être alimentée par ce lien. Consultez son état dans Safari. Ne payez pas une deuxième fois.",
    "handoffCardExpired": "Cette carte a expiré. Consultez son état dans Safari. Ne payez pas une deuxième fois.",
    "handoffLimited": "Cet accès permet de payer, suivre ou annuler cette carte non payée. Votre compte complet reste dans le navigateur d’origine.",
    "handoffResume": "Afficher la carte",
    "handoffUnavailable": "Impossible de reprendre cette carte. Rouvrez le lien depuis Safari."
  },
  "en": {
    "miniAppOpen": "Use Nimpay",
    "browserOpen": "Use Nimiq Wallet (browser)",
    "handoffDone": "Done",
    "handoffCancelled": "Card cancelled.",
    "handoffExitHelp": "You can return to your browser using the ‹ arrow at the top left of the iPhone, or close this window with ✕.",
    "handoffExpired": "The link is invalid or has expired. Reopen the link from Safari.",
    "handoffFailed": "Could not open Nimpay. Try again or use Nimiq Wallet.",
    "handoffConflict": "Another sender is signed in here. Sign out, then reopen the link from Safari.",
    "handoffNotFundable": "This card can no longer be funded through this link. Check its status in Safari. Do not pay again.",
    "handoffCardExpired": "This card has expired. Check its status in Safari. Do not pay again.",
    "handoffLimited": "This access lets you pay, follow or cancel this unpaid card. Your full account stays in the original browser.",
    "handoffResume": "View the card",
    "handoffUnavailable": "Unable to resume this card. Reopen the link from Safari."
  },
  "es": {
    "miniAppOpen": "Usar Nimpay",
    "browserOpen": "Usar Nimiq Wallet (navegador)",
    "handoffDone": "Listo",
    "handoffCancelled": "Tarjeta cancelada.",
    "handoffExitHelp": "Puedes volver a tu navegador con la flecha ‹ de la esquina superior izquierda del iPhone, o cerrar esta ventana con ✕.",
    "handoffExpired": "El enlace no es válido o ha caducado. Vuelve a abrirlo desde Safari.",
    "handoffFailed": "No se pudo abrir Nimpay. Reintenta o usa Nimiq Wallet.",
    "handoffConflict": "Hay otro remitente conectado aquí. Cierra su sesión y vuelve a abrir el enlace desde Safari.",
    "handoffNotFundable": "Esta tarjeta ya no se puede financiar mediante este enlace. Consulta su estado en Safari. No vuelvas a pagar.",
    "handoffCardExpired": "Esta tarjeta ha caducado. Consulta su estado en Safari. No vuelvas a pagar.",
    "handoffLimited": "Este acceso permite pagar, seguir o cancelar esta tarjeta sin pagar. Tu cuenta completa permanece en el navegador de origen.",
    "handoffResume": "Ver la tarjeta",
    "handoffUnavailable": "No se pudo retomar esta tarjeta. Vuelve a abrir el enlace desde Safari."
  },
  "de": {
    "miniAppOpen": "Nimpay verwenden",
    "browserOpen": "Nimiq Wallet verwenden (Browser)",
    "handoffDone": "Fertig",
    "handoffCancelled": "Karte storniert.",
    "handoffExitHelp": "Du kannst mit dem Pfeil ‹ oben links auf dem iPhone zu deinem Browser zurückkehren oder dieses Fenster mit ✕ schließen.",
    "handoffExpired": "Der Link ist ungültig oder abgelaufen. Öffne ihn erneut in Safari.",
    "handoffFailed": "Nimpay konnte nicht geöffnet werden. Versuche es erneut oder nutze Nimiq Wallet.",
    "handoffConflict": "Hier ist ein anderer Absender angemeldet. Melde ihn ab und öffne den Link erneut in Safari.",
    "handoffNotFundable": "Diese Karte kann über diesen Link nicht mehr aufgeladen werden. Prüfe ihren Status in Safari. Bezahle nicht erneut.",
    "handoffCardExpired": "Diese Karte ist abgelaufen. Prüfe ihren Status in Safari. Bezahle nicht erneut.",
    "handoffLimited": "Mit diesem Zugang kannst du diese unbezahlte Karte bezahlen, verfolgen oder stornieren. Dein vollständiges Konto bleibt im ursprünglichen Browser.",
    "handoffResume": "Karte anzeigen",
    "handoffUnavailable": "Diese Karte konnte nicht fortgesetzt werden. Öffne den Link erneut in Safari."
  },
  "zh": {
    "miniAppOpen": "使用 Nimpay",
    "browserOpen": "使用 Nimiq Wallet（浏览器）",
    "handoffDone": "完成",
    "handoffCancelled": "卡片已取消。",
    "handoffExitHelp": "你可以点击 iPhone 左上角的 ‹ 箭头返回浏览器，或点击 ✕ 关闭此窗口。",
    "handoffExpired": "链接无效或已过期。请从 Safari 重新打开链接。",
    "handoffFailed": "无法打开 Nimpay。请重试或使用 Nimiq Wallet。",
    "handoffConflict": "这里已登录另一位发件人。请先退出，再从 Safari 重新打开链接。",
    "handoffNotFundable": "此链接已无法为该卡片充值。请在 Safari 查看状态，勿再次付款。",
    "handoffCardExpired": "此卡片已过期。请在 Safari 查看状态，勿再次付款。",
    "handoffLimited": "此访问权限可用于支付、跟踪或取消这张未付款的卡片。完整账户仍在原浏览器中。",
    "handoffResume": "查看卡片",
    "handoffUnavailable": "无法继续处理此卡片。请从 Safari 重新打开链接。"
  }
};
for(const language of Object.keys(handoffWords))Object.assign(words[language],handoffWords[language]);
// The public light: Wire pays the network fee (gas) of an USDC / USDT payment. It is a quiet line, shown only while it is true.
const gasLightWords={
 en:'Network fees on USDC and USDT are paid by Phoenix Wire',
 fr:'Les frais de réseau USDC et USDT sont payés par Phoenix Wire',
 es:'Phoenix Wire paga las comisiones de red de USDC y USDT',
 de:'Phoenix Wire übernimmt die Netzwerkgebühren für USDC und USDT',
 zh:'USDC 和 USDT 的网络费用由 Phoenix Wire 承担'
};
for(const language of Object.keys(gasLightWords))words[language].gasLed=gasLightWords[language];
const cardWords={
 fr:{myCards:'Mes cartes',emptyCards:'Aucune carte pour le moment.',waitingCard:'En attente de votre paiement',confirmingCard:'Paiement reçu, confirmation en cours',fundedCard:'Carte alimentée. Wire envoie la carte à {email}.',sentCard:'Carte envoyée à {email}',claimedCard:'Réclamée',refundedCard:'Remboursée',expiredCard:'Expirée',cancelledCard:'Annulée',otherCard:'Dépensée ailleurs',checkingCard:'Vérification de la carte en cours…',mailFailedCard:'Carte alimentée. Le courriel n’a pas pu être envoyé. Wire réessaiera.',missingLinkCard:'Carte alimentée. Le lien d’invitation est nécessaire : utilisez le navigateur où la carte a été créée.',staleCard:'Vue périmée : le service ne répond pas. Vérification automatique en cours.',bannerClaimed:'Paiement reçu par {email}.',bannerCard:'Carte alimentée. Courriel envoyé à {email}.',dismissCard:'Fermer cette notification',doneCard:'Terminé',createdCard:'Carte créée. Payez-la pour que Wire l’envoie au destinataire.'},
 en:{myCards:'My cards',emptyCards:'No cards yet.',waitingCard:'Waiting for your payment',confirmingCard:'Payment received, awaiting confirmation',fundedCard:'Card funded. Wire is sending the card to {email}.',sentCard:'Card sent to {email}',claimedCard:'Claimed',refundedCard:'Refunded',expiredCard:'Expired',cancelledCard:'Cancelled',otherCard:'Spent elsewhere',checkingCard:'Checking the card…',mailFailedCard:'Card funded. The email could not be sent. Wire will retry.',missingLinkCard:'Card funded. The invitation link is needed: use the browser where this card was created.',staleCard:'Stale view: the service is not responding. Checking again automatically.',bannerClaimed:'Payment received by {email}.',bannerCard:'Card funded. Email sent to {email}.',dismissCard:'Dismiss this notification',doneCard:'Done',createdCard:'Card created. Pay it so Wire can send it to the recipient.'},
 es:{myCards:'Mis tarjetas',emptyCards:'Aún no hay tarjetas.',waitingCard:'Esperando tu pago',confirmingCard:'Pago recibido, esperando confirmación',fundedCard:'Tarjeta financiada. Wire está enviando la tarjeta a {email}.',sentCard:'Tarjeta enviada a {email}',claimedCard:'Reclamada',refundedCard:'Reembolsada',expiredCard:'Caducada',cancelledCard:'Cancelada',otherCard:'Gastada en otro lugar',checkingCard:'Comprobando la tarjeta…',mailFailedCard:'Tarjeta financiada. No se pudo enviar el correo. Wire lo intentará de nuevo.',missingLinkCard:'Tarjeta financiada. Se necesita el enlace de invitación: usa el navegador donde se creó.',staleCard:'Vista desactualizada: el servicio no responde. La comprobación continúa automáticamente.',bannerClaimed:'Pago recibido por {email}.',bannerCard:'Tarjeta financiada. Correo enviado a {email}.',dismissCard:'Cerrar esta notificación',doneCard:'Listo',createdCard:'Tarjeta creada. Págala para que Wire la envíe al destinatario.'},
 de:{myCards:'Meine Karten',emptyCards:'Noch keine Karten.',waitingCard:'Wartet auf deine Zahlung',confirmingCard:'Zahlung eingegangen, Bestätigung ausstehend',fundedCard:'Karte aufgeladen. Wire sendet die Karte an {email}.',sentCard:'Karte an {email} gesendet',claimedCard:'Eingelöst',refundedCard:'Zurückerstattet',expiredCard:'Abgelaufen',cancelledCard:'Storniert',otherCard:'Anderswo ausgegeben',checkingCard:'Karte wird geprüft…',mailFailedCard:'Karte aufgeladen. Die E-Mail konnte nicht gesendet werden. Wire versucht es erneut.',missingLinkCard:'Karte aufgeladen. Der Einladungslink wird benötigt: Nutze den Browser, in dem die Karte erstellt wurde.',staleCard:'Veraltete Ansicht: Der Dienst antwortet nicht. Die Prüfung wird automatisch wiederholt.',bannerClaimed:'Zahlung von {email} erhalten.',bannerCard:'Karte aufgeladen. E-Mail an {email} gesendet.',dismissCard:'Diese Mitteilung schließen',doneCard:'Fertig',createdCard:'Karte erstellt. Bezahle sie, damit Wire sie an den Empfänger senden kann.'},
 zh:{myCards:'我的卡片',emptyCards:'暂无卡片。',waitingCard:'等待你付款',confirmingCard:'已收到付款，正在等待确认',fundedCard:'卡片已充值。Wire 正在向 {email} 发送卡片。',sentCard:'卡片已发送至 {email}',claimedCard:'已领取',refundedCard:'已退款',expiredCard:'已过期',cancelledCard:'已取消',otherCard:'已在其他地方支出',checkingCard:'正在检查卡片…',mailFailedCard:'卡片已充值，但邮件未能发送。Wire 将重试。',missingLinkCard:'卡片已充值。需要邀请链接：请使用创建此卡片的浏览器。',staleCard:'视图已过时：服务暂未响应，正在自动重试检查。',bannerClaimed:'{email} 已收到付款。',bannerCard:'卡片已充值。邮件已发送至 {email}。',dismissCard:'关闭此通知',doneCard:'完成',createdCard:'卡片已创建。付款后 Wire 会将其发送给收件人。'}
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
const dateWords={fr:{createdAt:'Créée le',sentAt:'Envoyée le',claimedAt:'Réclamée le'},en:{createdAt:'Created',sentAt:'Sent',claimedAt:'Claimed'},es:{createdAt:'Creada',sentAt:'Enviada',claimedAt:'Reclamada'},de:{createdAt:'Erstellt',sentAt:'Gesendet',claimedAt:'Eingelöst'},zh:{createdAt:'创建时间',sentAt:'发送时间',claimedAt:'领取时间'}};
for(const language of Object.keys(dateWords))Object.assign(words[language],dateWords[language]);
window.PhoenixWordOverrides=Object.fromEntries(Object.entries(cardWords).map(([language,w])=>[language,{fundedMsg:w.fundedCard,confirmingMsg:w.confirmingCard,close:w.doneCard,added:w.createdCard,created:w.createdCard}]));
function read(key, fallback) { try { return localStorage.getItem(key) || fallback; } catch { return fallback; } }
function save(key, value) { try { localStorage.setItem(key, value); } catch { /* storage may be unavailable */ } }
// Legacy addresses were shared between senders. Delete them; never migrate them.
try{for(const chain of ['polygon','ethereum','Polygon','Ethereum'])localStorage.removeItem('wire-refund-address:'+chain);}catch{}
let preference = read('wire-language', 'auto'), lang = 'en', email = '', payments = null, usdtOn = false, nimOn = false, enabled = false, nimView = null, resumed = false;
let nimCache=null;
// The sound PhoenixMail plays when a card is sent (the same recording, the same volume), once, when a card that was just made is seen sent.
let sentSound=null;
function playSentSound(created){
 const age=Date.now()-Date.parse(created||'');if(!Number.isFinite(age)||age<0||age>=30*60*1000)return; // an old card seen on a new device is not news
 try{sentSound??=new Audio('/sounds/phoenix-notification.wav?v=0.4.38');sentSound.volume=.35;sentSound.currentTime=0;void sentSound.play().catch(()=>{});}catch{}
}
let adminPanel=null,inviteOnly=false,accessExpanded=false,maintenance=false,adminActive=false;
const inviteGate=document.createElement('div'),inviteNotice=document.createElement('p'),accessLink=document.createElement('button');
inviteGate.id='wire-invite-gate';inviteGate.hidden=true;inviteNotice.setAttribute('role','status');accessLink.id='wire-existing-access';accessLink.type='button';accessLink.className='secondary-btn wire-access-link';accessLink.setAttribute('aria-controls','wire-sender');
inviteGate.append(inviteNotice,accessLink);$('wire-sender').before(inviteGate);
// Maintenance: the administrator closed the site for a while. Visitors are told so; only an administrator signs in (the server refuses anybody else).
const maintenanceBanner=document.createElement('p');maintenanceBanner.id='wire-maintenance';maintenanceBanner.hidden=true;maintenanceBanner.setAttribute('role','status');$('wire-sender').before(maintenanceBanner);
let onMaintenance=()=>{maintenance=true;paintSenderAccess();};
window.addEventListener('wire-maintenance',()=>onMaintenance());
function paintMaintenance(){maintenanceBanner.textContent=flowText(lang,'maintenance');maintenanceBanner.hidden=!maintenance||adminActive;}
accessLink.addEventListener('click',()=>{accessExpanded=true;paintSenderAccess();$('wire-email').focus();});
function paintSenderAccess(){paintMiniAppSuggestion();paintMaintenance();inviteGate.hidden=mobileEntryOnly()||Boolean(email)||!enabled||!inviteOnly||maintenance;inviteNotice.textContent=flowText(lang,'inviteOnly');accessLink.textContent=adminText(lang,'already');accessLink.hidden=mobileEntryOnly()||accessExpanded;accessLink.setAttribute('aria-expanded',String(accessExpanded));$('wire-sender').hidden=mobileEntryOnly()||Boolean(email)||!enabled||inviteOnly&&!accessExpanded&&!maintenance;}
function configureAdmin(allowed){adminPanel?.destroy();adminPanel=mountAdmin({root:document.querySelector('.wire-shell'),language:()=>lang,request:api,authorized:allowed===true&&Boolean(email)&&!mobileEntryOnly(),onAccess:(open,view)=>{inviteOnly=!open;if(view)maintenance=view.maintenance===true;paintSenderAccess();}});adminActive=allowed===true&&Boolean(email)&&!mobileEntryOnly();paintMaintenance();}

let choosingCoin=false,codeSending=false,codeSentTo='',codeUntil=0,codeTimer=null,senderInviteOnly=false;
const t = key => (words[lang] || words.en)[key];
// Only an opaque, one-use credential crosses contexts. Remove it before any request.
let incomingToken=null,handoffError='',handoffPayment=null,scopedID='',fundingBlocked=false;
let browserEntry=false,senderAdmin=false;
try{browserEntry=sessionStorage.getItem('wire-mobile-browser')==='yes';}catch{}
if(location.hash.startsWith('#h=')){
 incomingToken=location.hash.slice(3);
 history.replaceState(null,'',location.pathname+location.search);
}
// Never accept the retired personal-data fragment or its saved draft.
if(location.hash.startsWith('#wire-nim=')){
 history.replaceState(null,'',location.pathname+location.search);
 handoffError='handoffExpired';
}
try{sessionStorage.removeItem('wire-nim-handoff');}catch{}

async function api(path, data, signal) {
 const options = { credentials: 'same-origin', cache: 'no-store', redirect: 'error' };
 if(signal)options.signal=signal;
 if (data !== undefined) { options.method = 'POST'; options.headers = { 'Content-Type': 'application/json' }; options.body = JSON.stringify(data); }
 const response = await fetch(path, options);
 let body = {}; try { body = await response.json(); } catch { /* an answer without a body */ }
 if (body.code === 'maintenance') onMaintenance();
 if (!response.ok) throw Object.assign(Error(body.code || body.error || 'unavailable'), { code: body.code || body.error, status: response.status, retryAfter: Number.isFinite(body.retry_after) ? body.retry_after : 0 });
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

// The flow's phone detector is private to its mounted view; use the same device rule here.
function mobile(){return /iPhone|iPad|iPod|Android/i.test(navigator.userAgent||'')||(navigator.platform==='MacIntel'&&navigator.maxTouchPoints>1);}
function mobileEntryOnly(){return !browserEntry&&mobile()&&typeof window.nimiqPay==='undefined'&&!new URLSearchParams(location.search).has('claim')&&!scopedID;}
function paintMiniAppSuggestion(){
 const box=$('wire-miniapp-suggestion');
 box.hidden=!(mobileEntryOnly()&&nimOn);
 const link=$('wire-miniapp-open');link.href='https://nimpay.app/miniapps/open/'+location.host;
 link.textContent=t('miniAppOpen');$('wire-browser-open').textContent=t('browserOpen');
}
const cancelledHandoff=()=>Boolean(scopedID&&handoffPayment?.id===scopedID&&handoffPayment.stage==='cancelled');
function paintHandoff(){
 const cancelled=cancelledHandoff(),button=$('wire-handoff-open');
 button.textContent=t(cancelled?'handoffDone':'handoffResume');
 button.hidden=!cancelled&&(fundingBlocked||Boolean(handoffError));
 button.setAttribute('aria-describedby',cancelled?'wire-handoff-exit-help':'wire-handoff-help');
 $('wire-handoff-help').hidden=cancelled;
 const help=$('wire-handoff-exit-help');help.textContent=t('handoffExitHelp');help.hidden=!cancelled;
 if(cancelled)$('wire-handoff-state').textContent=t('handoffCancelled');
}
function paint() {
 paintMiniAppSuggestion();
 document.documentElement.lang = lang;
 $('wire-handoff-title').textContent=t('nimCard');$('wire-handoff-help').textContent=t('handoffLimited');
 paintHandoff();
 $('wire-handoff-logout').setAttribute('aria-label',t('out'));$('wire-handoff-logout').title=t('out');
 $('wire-title').textContent = t('title'); $('wire-sub').textContent = t('sub'); $('wire-gas-text').textContent = t('gasLed');
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
 renderCards();paintSenderAccess();adminPanel?.render();document.querySelectorAll('.nimiq-payment-window:not(.wire-nim-window)').forEach(enhanceStableWindow);
}
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
function coinIcon(asset){
 const image=document.createElement('img');image.className='wire-coin-icon';image.width=24;image.height=24;image.dataset.asset=asset;
 const mark=assetMark(asset,lang);if(mark){image.src=mark.src;image.alt=mark.alt;}return image;
}
function buttonAsset(button){const label=button.textContent;return label.startsWith('NIM')?'nim':label.includes('USDC')?'usdc_polygon':label.includes('Ethereum')?'usdt_ethereum':'usdt_polygon';}
function paintNim(root) {
 root.querySelectorAll('[data-nim-text]').forEach(el => { el.textContent = t(el.dataset.nimText); });
 root.querySelector('.wire-nim-close').setAttribute('aria-label', t('back'));
 styleClose(root.querySelector('.wire-nim-close'));
 const title=root.querySelector('.nimiq-payment-head h3');if(!title.querySelector('img'))title.prepend(coinIcon('nim'));
}
function closePaymentWindow() {
 document.querySelector('.nimiq-payment-head button:last-child')?.click();
}
function openNimWindow(payment=null) {
 if(payment instanceof Event)payment=null;
 if(payment){nimView?.dismiss();nimCache?.flow.destroy();nimCache=null;handoffPayment=payment;}
 if (mobileEntryOnly() || (!email&&!scopedID) || !nimOn) return;
 choosingCoin=false;
 if(cancelledHandoff()){paintHandoff();return;}
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
 const dismiss = () => { modal.remove(); document.removeEventListener('keydown', escape); nimView = null;if(root.classList.contains('wire-nim-complete')){flow?.destroy();nimCache=null;if(!scopedID)handoffPayment=null;} $(scopedID?'wire-handoff-open':'pay').focus(); };
 const escape = e => { if (e.key === 'Escape') { e.preventDefault(); dismiss(); } };
 back.onclick = dismiss; modal.querySelector('.wire-nim-close').onclick = dismiss;
 document.addEventListener('keydown', escape);
 document.body.append(modal); nimView = { root: modal, dismiss };
 const prepared=handoffPayment;
 paintNim(modal);flow=mountNimFlow(root,{embedded:true,initialPayment:prepared,scopedPaymentID:scopedID,onStatus:(p,message)=>{
  if(scopedID){handoffPayment=p;$('wire-handoff-state').textContent=message;paintHandoff();if(p.stage==='cancelled'){dismiss();flow?.destroy();nimCache=null;}return;}
  nimCards.set(p.id,p);
  if(p.stage==='sent'){
   const notices=nimNotices();
   if(!notices.some(n=>n.id===p.id&&n.sender===email)){notices.push({id:p.id,sender:email,noticePending:true});save('wire-nim-notices',JSON.stringify(notices));playSentSound(p.created);}
  }
  renderCards();scheduleNimCards();
 },onUnpayable:()=>{dismiss();handoffPayment=null;},onClose:()=>{dismiss();if(!scopedID)renderCards();}});
 nimCache={modal,flow,escape,view:nimView};
 back.focus();
}
function setLanguage(value) {
 preference = value; save('wire-language', value);
 const nav = typeof navigator === 'undefined' ? {} : navigator;
 lang = resolveLanguage(value, nav.languages?.length ? nav.languages : [nav.language]);
 paint();
 if(senderInviteOnly)say(flowText(lang,'inviteOnly'));
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
const ht=key=>senderHistoryText(lang,key);
const nimActions=new Map();
const stableActions=new Map(),autoRefunds=new Set();
const life=key=>flowText(lang,key);
const validRefundAddress=value=>typeof value==='string'&&/^0x[0-9a-fA-F]{40}$/.test(value)&&!/^0x0{40}$/i.test(value);
const refundSender=value=>String(value||'').trim().toLowerCase();
const refundMemoryKey=(asset,owner=email)=>'wire-refund-address:'+String(asset).split('_').at(-1).toLowerCase()+':'+refundSender(owner);
function forgetRefundAddress(owner){if(!refundSender(owner))return;try{for(const chain of ['polygon','ethereum'])localStorage.removeItem(refundMemoryKey(chain,owner));}catch{}}
function knownRefundAddress(c){if(!email||refundSender(c.sender)!==refundSender(email))return '';const saved=read(refundMemoryKey(c.asset),'');return validRefundAddress(saved)?saved:'';}
function localRefundEntry(id){try{return JSON.parse(read('phoenixmail.usdt.sent.v2','[]')).find(e=>e.card?.id===id&&e.senderToken&&e.cashlink&&/^#c=/.test(new URL(e.card.claimURL).hash));}catch{return null;}}
function stableFunds(c){return c.state==='expired'&&!c.stale&&!c.claimedAt&&Number(c.balanceUnits)>0;}
function appendStableActions(row,c){
 if(!stableFunds(c)&&!stableActions.has(c.id))return;
 const state=stableActions.get(c.id)||(c.refundPending?{sent:true,message:'refundWorking'}:{}),note=document.createElement('p');note.className='wire-refund-result';note.setAttribute('role','status');note.textContent=state.message?life(state.message):'';
 if(stableFunds(c)&&!state.sent){
  if(!localRefundEntry(c.id))note.textContent=life('refundKey');
  else{const form=document.createElement('form'),label=document.createElement('label'),input=document.createElement('input'),go=document.createElement('button');
   form.className='wire-refund-form';label.textContent=life('refundTo');input.type='text';input.autocomplete='off';input.spellcheck=false;input.value=state.address||knownRefundAddress(c);input.required=true;input.pattern='0x[0-9a-fA-F]{40}';label.append(input);
   go.type='submit';go.className='primary-btn';go.textContent=life('refund');go.disabled=Boolean(state.loading);input.disabled=Boolean(state.loading);
   input.addEventListener('input',()=>{stableActions.set(c.id,{...stableActions.get(c.id),address:input.value});});
   form.addEventListener('submit',e=>{e.preventDefault();if(validRefundAddress(input.value.trim()))void refundStable(c,input.value.trim(),false);});form.append(label,go);row.append(form);
  }
 }row.append(note);
}
async function refundStable(c,to,automatic){
 if(!email||refundSender(c.sender)!==refundSender(email)||stableActions.get(c.id)?.loading||!stableFunds(c)||!validRefundAddress(to)||automatic&&knownRefundAddress(c)!==to)return;
 const entry=localRefundEntry(c.id);if(!entry)return;
 const epoch=cardEpoch,owner=email;stableActions.set(c.id,{loading:true,address:to,message:'refundWorking'});renderCards();
 try{
  const CL=window.PhoenixUSDTCashlink.create({language:()=>lang});
  const result=await CL.refund(entry,to,{confirm:({units,fee})=>epoch===cardEpoch&&owner===email&&(automatic||window.confirm(CL.fmt(CL.words(assetNames[c.asset]?.startsWith('USDC')?'USDC':'USDT').refundConfirm,{net:String(units/1e6),fee:String(fee/1e6)})))});
  if(epoch!==cardEpoch||email!==owner)return;
  if(!['refunded','refunding','verifying','withdrawn'].includes(result.status||result.state))throw Error('unconfirmed_refund');
  save(refundMemoryKey(c.asset,owner),to);stableActions.set(c.id,{address:to,message:result.status==='refunded'?'refundDone':'refundWorking',sent:true});
  update(c.id,{state:result.status==='refunded'?'withdrawn':'expired',recovering:true,refundPending:true,refundPendingAt:Date.now()});
 }catch{if(epoch===cardEpoch&&owner===email)stableActions.set(c.id,{address:to,message:'refundFailed'});}
 finally{if(epoch===cardEpoch&&owner===email){const state=stableActions.get(c.id);if(state)state.loading=false;renderCards();}}
}

const fill=(key,entry)=>t(key).replace('{email}',entry.recipient||'');
const nimPendingWords={fr:'En attente de confirmation du paiement',en:'Awaiting payment confirmation',es:'Esperando confirmación del pago',de:'Zahlungsbestätigung ausstehend',zh:'等待付款确认'};
function cardLabel(c){
 if(c.nim)return nimHistoryLabel(c,lang);
 if(c.claimedAt)return ht('received');
 if(c.nim)return fill(({preparing:'checkingCard',ready:'waitingCard',funding_started:'confirmingCard',funded:'fundedCard',delivery_pending:'fundedCard',delivery_failed:'mailFailedCard',sent:'sentCard',claimed:'claimedCard',expired:'expiredCard'})[c.state]||'checkingCard',c);
 const key=({paid:'claimedCard',withdrawn:'refundedCard',expired:'expiredCard',cancelled:'cancelledCard',settled_elsewhere:'otherCard'})[c.state];
 if(key)return t(key);
 if(c.state==='verifying')return t('confirmingCard');
 if(c.state==='ready')return fill(c.invited?'sentCard':c.mailFailed?'mailFailedCard':!c.fragment?'missingLinkCard':'fundedCard',c);
 if(c.state==='awaiting_funds')return t(c.balanceUnits>0?'confirmingCard':'waitingCard');
 return t('checkingCard');
}
function nimNotices(){try{const value=JSON.parse(read('wire-nim-notices','[]'));return Array.isArray(value)?value:[];}catch{return [];}}
function dateValue(value){const ms=typeof value==='number'?value:Date.parse(value||'');return Number.isFinite(ms)&&ms>0?ms:null;}
function serverDates(data){
 const dates={};for(const[key,source]of [['createdAt','created'],['sentAt','sent_at'],['claimedAt','claimed_at']]){const value=dateValue(data[source]);if(value)dates[key]=value;}return dates;
}
function appendDates(row,card){
 const list=document.createElement('dl');list.className='wire-card-dates';
 for(const key of ['createdAt','sentAt','claimedAt']){
  const ms=dateValue(card[key]);if(!ms||key==='claimedAt'&&!['paid','claimed'].includes(card.state))continue;
  const item=document.createElement('div'),label=document.createElement('dt'),value=document.createElement('dd'),time=document.createElement('time');
  label.textContent=t(key);time.dateTime=new Date(ms).toISOString();time.textContent=new Intl.DateTimeFormat(lang,{dateStyle:'medium',timeStyle:'short'}).format(ms);value.append(time);item.append(label,value);list.append(item);
 }
 if(list.childElementCount)row.append(list);
}
function renderCards(){
 if(mobileEntryOnly()){$('wire-cards').hidden=true;$('wire-card-notices').hidden=true;return;}
 const panel=$('wire-cards');if(!panel)return;
 panel.hidden=!email; $('wire-cards-title').textContent=t('myCards');
 let all=cards().filter(c=>c.sender===email).map(c=>({...c,nim:false}));
 for(const p of nimCards.values())if(p.sender===email)all.push({...p,nim:true,asset:'NIM',state:p.stage,...serverDates(p),invited:p.stage==='sent',noticePending:nimNotices().some(n=>n.id===p.id&&n.sender===email&&n.noticePending)});
 all=historyEntries(all);
 const list=$('wire-cards-list');list.replaceChildren();
 $('wire-cards-empty').textContent=t('emptyCards');$('wire-cards-empty').hidden=all.length>0;
 for(const c of all){
  const row=document.createElement('article');row.className='wire-card-row';row.dataset.cardId=c.id;
  const amount=document.createElement('strong');amount.textContent=(c.amount||'—')+' '+(c.nim?'NIM · Nimiq':assetNames[c.asset]||'USDC/USDT');
  const heading=document.createElement('div');heading.className='wire-card-heading';heading.append(coinIcon(c.nim?'nim':c.asset),amount);
  const recipient=document.createElement('p');recipient.className='wire-card-recipient';recipient.textContent=c.recipient||'—';
  const state=document.createElement('p');state.className='wire-card-state';state.textContent=cardLabel(c);row.append(heading,recipient,state);appendDates(row,c);
  if(c.nim)appendNimActions(row,c);else appendStableActions(row,c);
  if(c.stale){const stale=document.createElement('p');stale.className='wire-card-stale';stale.textContent=t('staleCard');row.append(stale);}
  list.append(row);
 }
 for(const [id,result]of stableActions)if(result.message&&!all.some(c=>c.id===id)){const note=document.createElement('p');note.className='wire-refund-result';note.setAttribute('role','status');note.textContent=life(result.message);list.append(note);}
 const notices=$('wire-card-notices');notices.replaceChildren();
 for(const c of all.filter(c=>c.invited&&c.noticePending)){
  const banner=document.createElement('section');banner.className='wire-card-banner';
  const message=document.createElement('p');message.setAttribute('role','status');const received=c.nim?c.claim_state==='claimed':c.state==='paid'||Boolean(c.claimedAt);message.textContent=fill(received?'bannerClaimed':'bannerCard',c);const end=received?0:dateValue(c.createdAt||c.created);if(end)message.textContent+=' '+life('validUntil').replace('{date}',new Intl.DateTimeFormat(lang,{dateStyle:'medium',timeStyle:'short'}).format(end+86400000));
  const close=document.createElement('button');close.type='button';close.className='secondary-btn';close.textContent='×';close.setAttribute('aria-label',t('dismissCard'));
  close.addEventListener('click',()=>{if(c.nim)save('wire-nim-notices',JSON.stringify(nimNotices().map(n=>n.id===c.id&&n.sender===email?{...n,noticePending:false}:n)));else update(c.id,{noticePending:false});renderCards();$('wire-cards-title').focus();});
  banner.append(message,close);notices.append(banner);
 }
 notices.hidden=!email||!notices.childElementCount;
}
function appendNimActions(row,p){
 const state=nimActions.get(p.id)||{},actions=document.createElement('div');actions.className='nim-row-actions';
 const button=(label,run)=>{const b=document.createElement('button');b.type='button';b.className='secondary-btn';b.textContent=label;b.disabled=Boolean(state.loading);b.onclick=run;actions.append(b);return b;};
 if(unpaidCard(p)){
  button(ht('retry'),()=>openNimWindow(p));
  button(ht('cancel'),()=>void dashboardCancel(p));
 }
 if(recoverableCard(p)){
  if(state.href){const link=document.createElement('a');link.href=state.href;link.target='_blank';link.rel='noopener noreferrer';link.textContent=ht('open');actions.append(link);}
  else button(ht('recover'),()=>void dashboardRecover(p));
 }
 row.append(actions);
 if(state.notice){const note=document.createElement('p');note.className='nim-row-notice';note.setAttribute('role','status');note.textContent=state.notice;row.append(note);}
}
async function dashboardCancel(p){
 if(nimActions.get(p.id)?.loading||!unpaidCard(p)||!window.confirm(ht('confirm')))return;
 const epoch=cardEpoch;nimActions.set(p.id,{loading:true});renderCards();
 try{
  const result=await cancelUnpaidCard(p,cardCall);if(epoch!==cardEpoch)return;
  nimCards.set(p.id,result.payment);nimActions.set(p.id,{notice:result.received?ht('arrived'):''});
  // A cached editor must not offer payment on the cancelled card.
  nimView?.dismiss();nimCache?.flow.destroy();nimCache=null;
  await syncNimCards();
 }catch(error){if(epoch!==cardEpoch)return;nimActions.set(p.id,{notice:ht(error.message==='cannot_cancel'?'cannot':'failed')});await syncNimCards();}
 finally{if(epoch===cardEpoch){const state=nimActions.get(p.id);if(state)state.loading=false;renderCards();}}
}
async function dashboardRecover(p){
 if(nimActions.get(p.id)?.loading||!recoverableCard(p))return;
 const epoch=cardEpoch;nimActions.set(p.id,{loading:true});renderCards();
 try{
  const result=await cardCall('/api/payments/'+p.id+'/recovery',{});if(epoch!==cardEpoch)return;
  const url=new URL(result.cashlink);if(url.origin!=='https://hub.nimiq.com'||url.pathname!=='/cashlink/'||!/^#[A-Za-z0-9_=-]+$/.test(url.hash))throw Error('invalid_link');
  nimActions.set(p.id,{href:url.href});
 }catch{if(epoch===cardEpoch)nimActions.set(p.id,{notice:ht('failed')});}
 finally{if(epoch===cardEpoch)renderCards();}
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
   Object.assign(c,{sender:owner,amount:String(server.amount||''),asset:server.asset,recipient:String(server.recipient||''),created:Date.parse(server.created)||c.created||0,invited:c.invited===true||server.invited===true},serverDates(server));
   if(newlySent){c.noticePending=true;playSentSound(server.created);}
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
   const previous=cards().find(c=>c.id===id);
   const refundFinished=previous?.refundPending&&st.state==='withdrawn';
   const refundStopped=previous?.refundPending&&st.state==='expired'&&st.recovering===false&&!st.stale&&Date.now()-(previous.refundPendingAt||0)>60000;
   update(id,{sender:owner,state:st.state,balanceUnits:Number(st.balanceUnits??st.balance_units)||0,stale:st.stale===true,checkedAt:Date.now(),...serverDates(st),expiresAt:st.expires_at||null,recovering:st.recovering===true,...(refundFinished||refundStopped?{refundPending:false}:{})});
   if(refundFinished)stableActions.set(id,{message:'refundDone',sent:true});
   if(refundStopped){autoRefunds.add(id);stableActions.set(id,{message:'refundFailed',address:knownRefundAddress(previous)});}
   renderCards();
   let entry=cards().find(c=>c.id===id);
   if(stableFunds(entry)&&!entry.recovering&&!entry.refundPending&&!stableActions.get(id)?.sent&&!autoRefunds.has(id)){const to=knownRefundAddress(entry);if(to&&localRefundEntry(id)){autoRefunds.add(id);await refundStable(entry,to,true);}}
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
function nimPending(p){return historyPending(p)||recoverableCard(p);}
function scheduleNimCards(){
 clearTimeout(nimTimer);nimTimer=null;
 if(email&&nimOn&&!document.hidden&&[...nimCards.values()].some(nimPending))nimTimer=setTimeout(()=>{void syncNimCards();},6000);
}
function syncNimCards(){
 if(nimPromise)return nimPromise;if(!email||!nimOn)return Promise.resolve();const epoch=cardEpoch,owner=email;
 const promise=(async()=>{try{const result=await cardCall('/api/payments');if(epoch!==cardEpoch||owner!==email)return;for(const p of result.payments||[])if(p.sender===owner)nimCards.set(p.id,p);renderCards();}catch{}finally{if(epoch===cardEpoch)scheduleNimCards();}})();
 nimPromise=promise;promise.finally(()=>{if(nimPromise===promise)nimPromise=null;}).catch(()=>{});return promise;
}
function stopCards(){
 cardEpoch++;for(const w of watching.values())clearTimeout(w.timer);watching.clear();clearTimeout(nimTimer);nimTimer=null;
 for(const c of cardControllers)c.abort();cardControllers.clear();ownedPromise=null;nimPromise=null;nimCards.clear();nimActions.clear();stableActions.clear();autoRefunds.clear();
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
 if(nimOn)void syncNimCards();
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
   all.unshift({ id: card.id, fragment: hash, recipient: String(entry?.protectedRecipient||entry?.to||card.recipient||''),amount:String(card.amount||''),asset:entry?.asset||card.asset,sender:email, invited: false, state:'awaiting_funds',created: entry?.at||0,fundingAddress:validRefundAddress(entry?.senderAddress)?entry.senderAddress:'' });
   keep(all);renderCards();watch(card.id);void syncOwnedCards().catch(()=>{});
  } catch { /* not a card of ours */ }
 }
}

// The card library is only fetched where the engine serves it (payments switched on).
function loadCashlink() {
 if (window.PhoenixCashlink) return Promise.resolve();
 return new Promise((ok, no) => { const s = document.createElement('script'); s.src = '/cashlink.js'; s.onload = ok; s.onerror = no; document.head.append(s); });
}


const stablePreviews=new WeakMap();
function stablePreview(win){
 const panel=win.querySelector('.nimiq-payment-panel'),preview=win.querySelector('.nimiq-payment-preview');
 const selected=win.querySelector('.nimiq-payment-assets button[aria-pressed="true"]'),asset=selected&&buttonAsset(selected);
 if(!panel||!preview||!assetNames[asset]||win.classList.contains('wire-coin-picker'))return null;
 if(preview.querySelector('.nimiq-payment-card:not(.wire-local-card)')){win.classList.remove('wire-has-local-preview');return null;}
 const amount=panel.querySelector('input[inputmode="decimal"]'),message=panel.querySelector('input[maxlength="140"]'),recipient=panel.querySelector('input[type="email"]');
 if(!amount||!message||!recipient)return null;
 let state=stablePreviews.get(win);
 if(!state){
  const card=$('wire-nim-template').content.querySelector('#payment-card').cloneNode(true);
  for(const node of [card,...card.querySelectorAll('[id]')]){node.dataset.cardPart=node.id;node.removeAttribute('id');}
  card.hidden=false;card.classList.add('wire-local-card');
  state={card,id:crypto.getRandomValues(new Uint8Array(1))[0].toString(16).padStart(2,'0')};stablePreviews.set(win,state);
  for(const event of ['input','change'])win.addEventListener(event,()=>enhanceStableWindow(win));
 }
 if(!preview.contains(state.card))preview.replaceChildren(state.card);
 win.classList.add('wire-has-local-preview');
 const raw=amount.value.trim(),valid=/^(0|[1-9][0-9]{0,6})(\.[0-9]{1,6})?$/.test(raw)&&Number(raw)>=0.01&&Number(raw)<=1000;
 renderCardPreview(state.card,{language:lang,id:state.id,asset:asset.toUpperCase(),amount:valid?raw:'0',message:message.value,recipient:recipient.value,codeText:flowText(lang,'cardCode')});
 return state.card;
}
function enhanceStableWindow(win){
 if(win.classList.contains('wire-nim-window'))return;
 const activeButton=win.querySelector('.nimiq-payment-assets button[aria-pressed="true"]'),activeAsset=activeButton&&buttonAsset(activeButton);
 // Replace only displayed mirrored artwork; the original files and email generation stay unchanged.
 for(const image of win.querySelectorAll('.wire-coin-icon,.usdt-coin')){
  const mark=assetMark(image.dataset.asset||activeAsset,lang);if(mark){if(image.getAttribute('src')!==mark.src)image.setAttribute('src',mark.src);image.alt=mark.alt;}
 }
 const draft=stablePreview(win);
 const amount=win.querySelector('input[inputmode="decimal"]');
 const host=draft||amount?.closest('label')||win.querySelector('.usdt-pay');
 if(host){let note=win.querySelector('.wire-lifetime-note');if(!note){note=document.createElement('p');note.className='wire-lifetime-note';}if(note.parentElement!==host)host.append(note);const text=life(draft?'cardLife':'life');if(note.textContent!==text)note.textContent=text;}
 const funded=win.querySelector('.usdt-funded');if(!funded)return;
 const selected=win.querySelector('.nimiq-payment-assets button[aria-pressed="true"]'),asset=selected?buttonAsset(selected):'';
 let sent=[];try{sent=JSON.parse(read('phoenixmail.usdt.sent.v2','[]'));}catch{}
 const entry=sent.filter(e=>e.asset===asset).sort((a,b)=>b.at-a.at)[0],c=entry&&cards().find(c=>c.id===entry.card?.id);
 const created=c&&dateValue(c.createdAt||c.created);if(!created)return;
 let expiry=funded.parentElement.querySelector('.wire-card-expiry');if(!expiry){expiry=document.createElement('p');expiry.className='wire-card-expiry';funded.after(expiry);}
 const text=life('validUntil').replace('{date}',new Intl.DateTimeFormat(lang,{dateStyle:'medium',timeStyle:'short'}).format(created+86400000));if(expiry.textContent!==text)expiry.textContent=text;
}

function mountPayments() {
 if (payments) return;
 const addBack = win => {
  if(!win.classList.contains('wire-nim-window'))win.classList.toggle('wire-coin-picker',choosingCoin);
  enhanceStableWindow(win);
  const head = win.querySelector('.nimiq-payment-head');
  if (!head || head.querySelector('.wire-window-back')) return;
  win.classList.toggle('wire-coin-picker',choosingCoin);
  const title=head.querySelector('h3');if(title)title.textContent=choosingCoin?t('chooseCoin'):title.textContent.replace(/^⤢\s*/, '');
  const assets=win.querySelector('.nimiq-payment-assets');if(assets)assets.hidden=!choosingCoin;
  assets?.querySelectorAll('button').forEach(button=>{button.prepend(coinIcon(buttonAsset(button)));});
  if(!choosingCoin&&title){const selected=assets?.querySelector('[aria-pressed="true"]');if(selected){title.textContent=selected.textContent;title.prepend(coinIcon(buttonAsset(selected)));}}
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
 $('pay').addEventListener('click', () => { if (!email||fundingBlocked) return; closePaymentWindow(); if (!usdtOn) openNimWindow(); else{choosingCoin=true;addBack(payments.open('send', 'usdc_polygon'));} });
 // The copied window replaces itself when switching coins; enhance each replacement without editing pm/.
 const windows = new MutationObserver(changes => {
  for (const change of changes) for (const node of change.addedNodes) {
   if (node.nodeType === 1 && node.matches('.nimiq-payment-modal')) node.querySelectorAll('.nimiq-payment-window').forEach(addBack);
  }
 });
 const lifetimeWindows=new MutationObserver(()=>{if(document?.body)document.querySelectorAll('.nimiq-payment-window:not(.wire-nim-window)').forEach(enhanceStableWindow);});
 lifetimeWindows.observe(document.body,{childList:true,subtree:true});
 windows.observe(document.body, { childList: true });
 payments = window.PhoenixNimiqPayments.create({ language: () => lang, account: () => 'wire', senderEmail: () => email, attach, status: ()=>{renderCards();},
  brand: { icon: '/brand/phoenix-emblem.webp', alt: 'Phoenix Wire', first: 'Phoenix', second: 'Wire' }, snapshot: () => null, recipient: () => '', bindOTPRecipient: () => true, openRecoveredClaim: () => false });
}

function showSender(verified, who,admin=false) {
 senderAdmin=verified&&admin===true;
 if(email&&(!verified||refundSender(email)!==refundSender(who)))forgetRefundAddress(email);
 if(!verified)accessExpanded=false;
 if(verified){codeSentTo='';codeUntil=0;paintCodeRequest();}
 if(!verified||email&&email!==who)stopCards();
 if(!verified&&nimCache){const cached=nimCache;nimView?.dismiss();cached.flow.destroy();cached.modal.remove();nimCache=null;resumed=false;}
 email = verified ? who : '';
 $('wire-sender').hidden = verified || !enabled; $('wire-ready').hidden = mobileEntryOnly() || !verified || !enabled;
 $('wire-who').textContent = email;configureAdmin(verified&&admin===true);paintSenderAccess();
 if (verified && enabled && !mobileEntryOnly()) {
  mountPayments();void startCards();
  try { if (nimOn && !resumed && !fundingBlocked && !handoffError && (handoffPayment?.sender.toLowerCase()===email.toLowerCase()||/^[a-f0-9]{48}$/.test(sessionStorage.getItem('wire-pending-payment') || ''))) { resumed = true; openNimWindow(); } } catch { /* tab storage may be unavailable */ }
 }
 renderCards();
}

$('wire-browser-open').addEventListener('click',()=>{
 browserEntry=true;
 try{sessionStorage.setItem('wire-mobile-browser','yes');}catch{}
 showSender(Boolean(email),email,senderAdmin);
 if(!$('wire-sender').hidden)$('wire-email').focus();else if(!inviteGate.hidden)accessLink.focus();
});
$('wire-handoff-open').addEventListener('click',()=>{
 if(cancelledHandoff()){
  const button=$('wire-handoff-open');if(button.disabled)return;button.disabled=true;
  try{window.close();}catch{ /* Some hosts refuse script-initiated closing. The help remains visible. */ }
  setTimeout(()=>{if(!window.closed&&cancelledHandoff()){$('wire-handoff-exit-help').hidden=false;button.disabled=false;}},300);
  return;
 }
 if(!fundingBlocked&&!handoffError)openNimWindow();
});
$('wire-handoff-logout').addEventListener('click',async()=>{
 try{await api('/api/logout',{});const cached=nimCache;nimView?.dismiss();cached?.flow.destroy();nimCache=null;scopedID='';handoffPayment=null;$('wire-handoff-home').hidden=true;$('wire-entry').hidden=false;showSender(false);say(t('handoffExpired'));}
 catch{$('wire-handoff-state').textContent=t('handoffFailed');}
});
$('wire-language').addEventListener('change', e => setLanguage(e.target.value));
$('wire-theme').addEventListener('click', () => { const v = document.documentElement.dataset.theme === 'light-paper' ? 'dark' : 'light'; applyTheme(v); save('wire-theme', v); });
$('wire-palette').addEventListener('change',event=>{applyPalette(event.target.value);save('wire-palette',paletteMode);});
// The server says how long to wait after "too many codes": the page counts it down (m:ss) and re-enables the button by itself.
let waitUntil=0,waitTimer=0;
const clock=seconds=>Math.floor(seconds/60)+':'+String(seconds%60).padStart(2,'0');
function paintWait(){
 clearTimeout(waitTimer);
 const remaining=Math.max(0,Math.ceil((waitUntil-Date.now())/1000));
 if(remaining>0){say(flowText(lang,'waitCodeIn').replace('{time}',clock(remaining)));waitTimer=setTimeout(paintWait,1000);}
 else if(waitUntil){waitUntil=0;say('');}
 paintCodeRequest();
}
function paintCodeRequest(){
 const button=$('wire-send-code'),remaining=Math.max(0,Math.ceil((codeUntil-Date.now())/1000)),waiting=Math.max(0,Math.ceil((waitUntil-Date.now())/1000));
 button.disabled=codeSending||remaining>0||waiting>0;
 button.textContent=codeSending?t('sendingCode'):waiting?t('send')+' ('+clock(waiting)+')':codeSentTo?(remaining?t('codeSentButton')+' ('+remaining+' s)':t('resendCode')):t('send');
 button.dataset.state=codeSending?'sending':codeSentTo?'sent':'idle';
 $('wire-code-delivery').textContent=codeSending?t('sendingCode'):codeSentTo?t('codeSent'):'';
 $('wire-code-delivery').hidden=!codeSending&&!codeSentTo;
 clearTimeout(codeTimer);if(remaining&&!codeSending)codeTimer=setTimeout(paintCodeRequest,1000);
}
$('wire-email').addEventListener('input',()=>{
 if(codeSentTo&&$('wire-email').value.trim()!==codeSentTo){codeSentTo='';codeUntil=0;$('wire-code-box').hidden=true;$('wire-code').value='';paintCodeRequest();}
});
$('wire-send-code').addEventListener('click', async () => {
 if(codeSending||Date.now()<codeUntil||Date.now()<waitUntil)return;
 if (!$('wire-email').reportValidity() || !$('wire-email').value) return;
 senderInviteOnly=false;codeSending=true;$('wire-email').disabled=true;$('wire-verify').disabled=true;paintCodeRequest();
 const controller=new AbortController(),timeout=setTimeout(()=>controller.abort(),30000);
 try {const to=$('wire-email').value.trim();await api('/api/sender/request', { email: to, language: lang },controller.signal);codeSentTo=to;codeUntil=Date.now()+30000;$('wire-code-box').hidden = false;say(t('codeSent'));$('wire-code').focus();}
 catch(error){senderInviteOnly=error.code==='invite_only';codeSentTo='';codeUntil=0;$('wire-code-box').hidden=true;$('wire-code').value='';if(error.code==='wait_before_resend'&&error.retryAfter>0){waitUntil=Date.now()+error.retryAfter*1000;paintWait();}else say(senderInviteOnly?flowText(lang,'inviteOnly'):error.code==='maintenance'?flowText(lang,'maintenance'):error.code==='wait_before_resend'?flowText(lang,'waitCode'):t('fail'));}
 finally{clearTimeout(timeout);codeSending=false;$('wire-email').disabled=false;$('wire-verify').disabled=false;paintCodeRequest();}
});
$('wire-verify').addEventListener('click', async () => {
 try { const r = await api('/api/sender/verify', { code: $('wire-code').value.trim() }); $('wire-code').value = ''; $('wire-code-box').hidden = true; say(''); showSender(true, r.email);void api('/api/session').then(session=>{if(email===session.email)configureAdmin(session.admin===true);}).catch(()=>{}); }
 catch(error) { say(error.code==='maintenance'?flowText(lang,'maintenance'):t('wrong')); }
});
$('wire-logout').addEventListener('click', async () => {
 const button=$('wire-logout');if(button.disabled)return;button.disabled=true;
 const controller=new AbortController(),timeout=setTimeout(()=>controller.abort(),30000);
 try{
  await api('/api/logout', {},controller.signal);closePaymentWindow();showSender(false);
  handoffPayment=null;scopedID='';fundingBlocked=false;$('wire-entry').hidden=false;$('wire-handoff-home').hidden=true;
  codeSentTo='';codeUntil=0;$('wire-email').value='';$('wire-code').value='';$('wire-code-box').hidden=true;paintCodeRequest();say('');$('wire-email').focus();
 }catch{say(t('fail'));}finally{clearTimeout(timeout);button.disabled=false;}
});

const savedPalette=read('wire-palette','');
applyPalette(savedPalette==='live'||palettes.includes(savedPalette)?savedPalette:({orange:'phoenixfire',violet:'aurora',teal:'ocean'}[read('wire-accent','')]||(savedPalette?'phoenixfire':'live')));
applyTheme(read('wire-theme', 'dark'));
setLanguage(preference);
async function refreshGas() {
 try { const g = await api('/api/gas'); $('wire-gas').hidden = !(g.polygon?.available || g.ethereum?.available); } catch { $('wire-gas').hidden = true; }
}
setInterval(() => { if (!document.hidden&&!scopedID) refreshGas(); }, 60000);
(async () => {
 try {
  // Redeem at most once. No retry or persistence of this bearer credential.
  if(incomingToken!==null){
   let result;
   const controller=new AbortController(),timeout=setTimeout(()=>controller.abort(),15000);
   try{result=await api('/api/handoff/redeem',{token:incomingToken},controller.signal);}
   catch(error){
    handoffError=error.code==='session_conflict'?'handoffConflict':error.code==='payment_not_fundable'?'handoffNotFundable':error.status===410?'handoffCardExpired':error.status===404?'handoffExpired':'handoffUnavailable';
    fundingBlocked=error.code==='payment_not_fundable'||error.status===410;
   }
   finally{clearTimeout(timeout);incomingToken=null;}
   if(result){
    const p=result.payment;
    if(p&&/^[a-f0-9]{48}$/.test(p.id)&&(result.scoped===true||result.already_signed_in===true)){
     handoffPayment=p;
     if(['fr','en','es','de','zh'].includes(p.language)){preference=p.language;save('wire-language',preference);setLanguage(preference);}
    }else handoffError='handoffExpired';
   }
  }
  const config = await api('/api/config');inviteOnly=config.invite_only===true;maintenance=config.maintenance===true;paintMaintenance();
  usdtOn = config.mode === 'integrated' && (config.assets || []).some(a => /^USD[CT]_/.test(a));
  nimOn = config.mode === 'integrated' && config.payments_enabled === true && (config.assets || []).includes('NIM');
  enabled = usdtOn || nimOn;
  if (!enabled) { say(t('off')); return; }
  const session = await api('/api/session');
  const scope=session.handoff?.payment_id;
  if(!session.verified&&/^[a-f0-9]{48}$/.test(scope||'')){
   scopedID=scope;usdtOn=false;stopCards();
   if(handoffPayment?.id!==scope){
    const list=await api('/api/payments');handoffPayment=list.payments?.find(p=>p.id===scope)||null;
   }
   $('wire-entry').hidden=true;$('wire-handoff-home').hidden=false;
   $('wire-handoff-state').textContent=handoffError?t(handoffError):t('handoffLimited');
   if(handoffPayment&&!handoffError){
    if(['fr','en','es','de','zh'].includes(handoffPayment.language))setLanguage(handoffPayment.language);
    openNimWindow();
   }
  }else{
   if (usdtOn) await loadCashlink();
   showSender(Boolean(session.verified), session.email,session.admin);refreshGas();
  }
  $('pay').disabled=fundingBlocked;paintHandoff();paintMiniAppSuggestion();
  if(handoffError){say(t(handoffError));if(scopedID)$('wire-handoff-state').textContent=t(handoffError);}
 } catch { say(t('off'));if(scopedID){$('wire-entry').hidden=false;$('wire-sender').hidden=false;} }
})();




