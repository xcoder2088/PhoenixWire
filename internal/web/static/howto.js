// How-to page: how to send a payment with Phoenix Wire, step by step, with example cards (made-up amounts).
const languages = ['fr', 'en'];
const copy = {
 en: {
  title: 'How it works', back: 'Back to Phoenix Wire', lang: 'Language', auto: 'Automatic', theme: 'Switch appearance', powered: 'Powered by',
  lead: 'Send NIM, USDC or USDT to anyone who has an email address. They receive a card by email and only they can claim it, after a code sent to that address.',
  steps: [
   {h: 'Confirm your email', p: ['Open Phoenix Wire, type your email and tap “Send me a code”. Enter the six-digit code you receive. You only do this once: your email stays confirmed for 24 hours on this device.'],
    img: ['entry', 'The first screen: your email, then a six-digit code.'],
    tip: 'Sending is by invitation unless an administrator has opened access. If sending is not available for you yet, ask the person who runs the service.'},
   {h: 'Make the card', p: ['Fill in the card: the recipient’s email, the coin (NIM, USDC or USDT) and the amount, and a short note if you like (up to 160 characters). The card appears as you type, so you can check it before paying.', 'USDC and USDT amounts are between 1 and 1000.'],
    img: ['form', 'The card form: recipient, coin, amount and note.']},
   {h: 'Pay', p: ['For NIM, pay with Nimpay or the Nimiq Wallet in your browser.', 'For USDC or USDT, make one payment from any wallet: tap “Open in my wallet” or copy the card address. The amount shown already includes the fees, so the recipient pays nothing. Your wallet adds its own network fee. The network needs about 2 minutes to confirm.'],
    tip: 'The key of a USDC/USDT card is kept in the browser where you made it. Do not clear this browser’s data until the card is claimed or taken back.'},
   {h: 'Wire sends the card', p: ['Once the payment is confirmed, Wire emails the card to the recipient. “My cards” shows where each card stands: waiting for payment, sent, claimed, refunded or expired. You can send the card again if the email did not arrive.'],
    cards: [['card-nim', 'NIM'], ['card-usdc', 'USDC'], ['card-usdt', 'USDT']], cardsNote: 'Examples of cards, with made-up amounts.'},
   {h: 'The recipient claims it', p: ['The recipient opens the link in the email and taps “Send my code”. Wire sends a six-digit code to their address, nothing is sent automatically. They enter it, then choose where to receive the money, or open the usual Nimiq payment link.']}
  ],
  know: ['Good to know', [
   'A card is valid for 24 hours. If nobody claims it, an unclaimed NIM card is refunded automatically to the address that paid it. For USDC/USDT, open Phoenix Wire in the same browser and tap “Take my money back”: the relay keeps its fees.',
   'Once a payment is claimed, nobody, including Wire, can reverse it. Check the recipient’s email carefully before you pay.',
   'NIM has no Wire fee beyond the network fee. For USDC/USDT the total you see includes the fees. The details are in the Terms of use.']],
  terms: 'Terms of use'
 },
 fr: {
  title: 'Comment ça marche', back: 'Retour à Phoenix Wire', lang: 'Langue', auto: 'Automatique', theme: 'Changer l’apparence', powered: 'Propulsé par',
  lead: 'Envoyez des NIM, des USDC ou des USDT à toute personne qui a une adresse courriel. Elle reçoit une carte par courriel et elle seule peut l’encaisser, après un code envoyé à cette adresse.',
  steps: [
   {h: 'Confirmez votre courriel', p: ['Ouvrez Phoenix Wire, tapez votre courriel et touchez « M’envoyer un code ». Entrez le code à six chiffres que vous recevez. Vous ne le faites qu’une fois : votre courriel reste confirmé 24 heures sur cet appareil.'],
    img: ['entry', 'Le premier écran : votre courriel, puis un code à six chiffres.'],
    tip: 'L’envoi se fait sur invitation, sauf si un administrateur a ouvert l’accès. Si l’envoi n’est pas encore disponible pour vous, demandez à la personne qui gère le service.'},
   {h: 'Créez la carte', p: ['Remplissez la carte : le courriel du destinataire, la monnaie (NIM, USDC ou USDT), le montant, et un petit mot si vous le souhaitez (160 caractères au plus). La carte se dessine pendant que vous écrivez : vous pouvez la vérifier avant de payer.', 'Pour l’USDC et l’USDT, le montant se situe entre 1 et 1000.'],
    img: ['form', 'Le formulaire de la carte : destinataire, monnaie, montant et petit mot.']},
   {h: 'Payez', p: ['Pour le NIM, payez avec Nimpay ou avec Nimiq Wallet dans votre navigateur.', 'Pour l’USDC ou l’USDT, faites un seul paiement depuis n’importe quel portefeuille : touchez « Ouvrir dans mon portefeuille » ou copiez l’adresse de la carte. Le montant affiché inclut déjà les frais : le destinataire ne paie rien. Votre portefeuille ajoute ses propres frais de réseau. Le réseau met environ 2 minutes à confirmer.'],
    tip: 'La clé d’une carte USDC/USDT est gardée dans le navigateur où vous l’avez créée. N’effacez pas les données de ce navigateur avant que la carte soit réclamée ou reprise.'},
   {h: 'Wire envoie la carte', p: ['Une fois le paiement confirmé, Wire envoie la carte par courriel au destinataire. « Mes cartes » indique où en est chaque carte : en attente de paiement, envoyée, réclamée, remboursée ou expirée. Vous pouvez renvoyer la carte si le courriel n’est pas arrivé.'],
    cards: [['card-nim', 'NIM'], ['card-usdc', 'USDC'], ['card-usdt', 'USDT']], cardsNote: 'Exemples de cartes, avec des montants inventés.'},
   {h: 'Le destinataire l’encaisse', p: ['Le destinataire ouvre le lien du courriel et touche « Envoyer mon code ». Wire envoie un code à six chiffres à son adresse, rien ne part automatiquement. Il l’entre, puis choisit où recevoir l’argent, ou ouvre le lien de paiement Nimiq habituel.']}
  ],
  know: ['Bon à savoir', [
   'Une carte est valable 24 heures. Si personne ne la réclame, une carte NIM est remboursée automatiquement à l’adresse qui l’a payée. Pour l’USDC/USDT, ouvrez Phoenix Wire dans le même navigateur et touchez « Reprendre mon argent » : le relais garde ses frais.',
   'Une fois un paiement réclamé, personne, Wire compris, ne peut l’annuler. Vérifiez bien le courriel du destinataire avant de payer.',
   'Le NIM n’a aucun frais Wire en plus des frais de réseau. Pour l’USDC/USDT, le total affiché inclut les frais. Le détail est dans les Conditions d’utilisation.']],
  terms: 'Conditions d’utilisation'
 }
};
function pick(value, preferences) {
 if (languages.includes(value)) return value;
 for (const p of preferences || []) {const l = String(p).slice(0, 2).toLowerCase();if (languages.includes(l)) return l;}
 return 'en';
}
function el(doc, tag, text, cls) {const n = doc.createElement(tag);if (text !== undefined) n.textContent = text;if (cls) n.className = cls;return n;}
function figure(doc, src, alt, caption) {
 const f = el(doc, 'figure', undefined, 'howto-fig'), img = el(doc, 'img');
 img.src = src;img.alt = alt;img.loading = 'lazy';img.width = 560;f.append(img);
 if (caption) f.append(el(doc, 'figcaption', caption));
 return f;
}
export function renderHowto(doc, lang) {
 const c = copy[lang];
 doc.documentElement.lang = lang;
 doc.title = c.title + ' · Phoenix Wire';
 doc.getElementById('howto-title').textContent = c.title;
 doc.getElementById('howto-lead').textContent = c.lead;
 doc.getElementById('howto-back').textContent = c.back;
 const picker = doc.getElementById('wire-language');
 if (picker) {picker.setAttribute('aria-label', c.lang);picker.querySelector('[value="auto"]').textContent = c.auto;}
 const theme = doc.getElementById('legal-theme');if (theme) theme.textContent = c.theme;
 const powered = doc.querySelector('[data-legal-powered]');if (powered) powered.textContent = c.powered + ' ';
 const blocks = [];
 c.steps.forEach((s, i) => {
  const sec = el(doc, 'section', undefined, 'howto-step'), h = el(doc, 'h2');
  h.append(el(doc, 'span', String(i + 1), 'howto-num'), el(doc, 'span', s.h));
  sec.append(h);
  for (const text of s.p) sec.append(el(doc, 'p', text));
  if (s.img) sec.append(figure(doc, '/howto/' + s.img[0] + '-' + lang + '.webp', s.img[1], s.img[1]));
  if (s.cards) {
   const row = el(doc, 'div', undefined, 'howto-cards');
   for (const [file, name] of s.cards) row.append(figure(doc, '/howto/' + file + '-' + lang + '.webp', name + ' card', name));
   sec.append(row, el(doc, 'p', s.cardsNote, 'howto-note'));
  }
  if (s.tip) sec.append(el(doc, 'p', s.tip, 'howto-tip'));
  blocks.push(sec);
 });
 const know = el(doc, 'section', undefined, 'howto-step'), list = el(doc, 'ul', undefined, 'howto-list');
 know.append(el(doc, 'h2', c.know[0]));
 for (const text of c.know[1]) list.append(el(doc, 'li', text));
 const terms = el(doc, 'a', c.terms);terms.href = '/terms.html?lang=' + lang;
 know.append(list, el(doc, 'p'));know.lastChild.append(terms);
 blocks.push(know);
 doc.getElementById('howto-content').replaceChildren(...blocks);
 for (const [name, label] of [['terms', c.terms]]) {const a = doc.querySelector('footer [data-legal-link="' + name + '"]');if (a) a.textContent = label;}
}
export function mountHowto(doc, win) {
 const read = (key, fallback) => {try {return win.localStorage.getItem(key) || fallback;} catch {return fallback;}};
 const save = (key, value) => {try {win.localStorage.setItem(key, value);} catch {}};
 const picker = doc.getElementById('wire-language');
 const query = new URLSearchParams(win.location.search).get('lang');
 let choice = languages.includes(query) ? query : read('wire-language', 'auto');
 if (choice !== 'auto' && !languages.includes(choice)) choice = 'auto';
 const prefs = win.navigator.languages?.length ? win.navigator.languages : [win.navigator.language];
 const paint = () => {renderHowto(doc, pick(choice, prefs));picker.value = languages.includes(choice) ? choice : 'auto';};
 picker.addEventListener('change', () => {choice = languages.includes(picker.value) ? picker.value : 'auto';save('wire-language', choice);paint();});
 const palette = read('wire-palette', 'phoenixfire');
 doc.documentElement.dataset.palette = ['phoenixfire', 'aurora', 'ocean'].includes(palette) ? palette : 'phoenixfire';
 let theme = read('wire-theme', 'dark') === 'light' ? 'light-paper' : 'dark';
 doc.documentElement.dataset.theme = theme;
 doc.getElementById('legal-theme').addEventListener('click', () => {
  theme = theme === 'dark' ? 'light-paper' : 'dark';doc.documentElement.dataset.theme = theme;save('wire-theme', theme === 'dark' ? 'dark' : 'light');
 });
 paint();
}
if (typeof document !== 'undefined' && typeof window !== 'undefined') mountHowto(document, window);
