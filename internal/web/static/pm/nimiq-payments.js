/* MIT. Copyright (c) 2026 xcoder2088. Wallet approval is required for every transfer. */
(function(root){'use strict';
const TYPE='application/vnd.phoenixmail.nimiq-payment+json',NAME='phoenixmail-payment-request.json';
const words={fr:['Envoyer / Demander Nimiq','Demande de paiement','Adresse du bénéficiaire','Montant NIM','Référence (facultative)','Réseau','Réseau principal','Réseau principal','Choisir mon adresse','Créer la carte','Ajouter au courriel','Annuler','Payer via Nimiq','Vérifier le paiement','Paiement confirmé','Transaction envoyée · confirmation en attente','Vérification indisponible','Vérifie le bénéficiaire avant de confirmer dans le wallet.','Demande invalide','Demande non réglée','Paiement annulé ou indisponible','Carte ajoutée au courriel'],en:['Send / Request Nimiq','Payment request','Recipient address','NIM amount','Reference (optional)','Network','Main network','Main network','Choose my address','Create card','Add to email','Cancel','Pay with Nimiq','Check payment','Payment confirmed','Transaction sent · awaiting confirmation','Verification unavailable','Check the recipient before confirming in the wallet.','Invalid request','Unpaid request','Payment cancelled or unavailable','Card added to email'],es:['Enviar / Solicitar Nimiq','Solicitud de pago','Dirección del beneficiario','Monto NIM','Referencia (opcional)','Red','Red principal','Red principal','Elegir mi dirección','Crear tarjeta','Añadir al correo','Cancelar','Pagar con Nimiq','Verificar pago','Pago confirmado','Transacción enviada · pendiente de confirmación','Verificación no disponible','Verifica al beneficiario antes de confirmar en el wallet.','Solicitud inválida','Solicitud sin pagar','Pago cancelado o no disponible','Tarjeta añadida al correo'],de:['Nimiq senden / anfordern','Zahlungsanforderung','Empfängeradresse','NIM-Betrag','Referenz (optional)','Netzwerk','Hauptnetz','Hauptnetz','Meine Adresse wählen','Karte erstellen','Zur E-Mail hinzufügen','Abbrechen','Mit Nimiq bezahlen','Zahlung prüfen','Zahlung bestätigt','Transaktion gesendet · Bestätigung ausstehend','Prüfung nicht verfügbar','Prüfe den Empfänger vor der Bestätigung im Wallet.','Ungültige Anforderung','Unbezahlte Anforderung','Zahlung abgebrochen oder nicht verfügbar','Karte zur E-Mail hinzugefügt'],zh:['发送 / 请求 Nimiq','付款请求','收款人地址','NIM 金额','参考编号（可选）','网络','主网络','主网络','选择我的地址','创建卡片','添加到邮件','取消','通过 Nimiq 付款','检查付款','付款已确认','交易已发送 · 等待确认','无法验证','请在钱包确认前核实收款人。','请求无效','尚未支付的请求','付款已取消或不可用','卡片已添加到邮件']};
const cashWords={fr:['Message (facultatif)','Paiement à recevoir','Verser les NIM et ajouter au courriel','Tu verses le montant maintenant. La personne qui reçoit la carte la scanne et reçoit les NIM dans son portefeuille.','Non protégé : aucun code n’est requis. Toute personne ayant le QR ou le lien peut encaisser les NIM.','NIM à recevoir','Cashlink créé · NIM versés'],en:['Message (optional)','Payment to receive','Send the NIM and add to email','You send the amount now. The person who gets the card scans it and receives the NIM in their wallet.','Non-protected: no code is required. Anyone with the QR or link can claim the NIM.','NIM to receive','Cashlink created · NIM sent'],es:['Mensaje (opcional)','Pago por recibir','Enviar los NIM y añadir al correo','Envías el monto ahora. Quien recibe la tarjeta la escanea y recibe los NIM en su monedero.','Sin protección: no se requiere código. Cualquiera con el QR o el enlace puede reclamar los NIM.','NIM por recibir','Cashlink creado · NIM enviados'],de:['Nachricht (optional)','Zahlung zum Empfangen','NIM senden und zur E-Mail hinzufügen','Du sendest den Betrag jetzt. Wer die Karte erhält, scannt sie und bekommt die NIM in seine Wallet.','Ungeschützt: Kein Code erforderlich. Jeder mit dem QR-Code oder Link kann die NIM einlösen.','NIM zu erhalten','Cashlink erstellt · NIM gesendet'],zh:['留言（可选）','待收款项','发送 NIM 并添加到邮件','你现在发送金额。收到卡片的人扫描后，NIM 会进入其钱包。','未受保护：无需验证码。任何获得二维码或链接的人都可以领取 NIM。','待领取的 NIM','Cashlink 已创建 · NIM 已发送']};
const walletWords={fr:['Mon portefeuille','Tes adresses Nimiq, enregistrées sur cet appareil. Elles apparaissent dans « Choisir mon adresse » pour créer une demande de paiement en un clic.','Aucune adresse enregistrée.','Ajouter une adresse','Nom (facultatif)','Ajouter','Choisir dans Nimiq','Supprimer','Adresse invalide','Autre adresse…','Nom','Adresse Nimiq'],en:['My wallet','Your Nimiq addresses, saved on this device. They appear under “Choose my address” to create a payment request in one click.','No saved address.','Add an address','Name (optional)','Add','Choose in Nimiq','Delete','Invalid address','Another address…','Name','Nimiq address'],es:['Mi monedero','Tus direcciones Nimiq, guardadas en este dispositivo. Aparecen en «Elegir mi dirección» para crear una solicitud de pago con un clic.','Ninguna dirección guardada.','Añadir una dirección','Nombre (opcional)','Añadir','Elegir en Nimiq','Eliminar','Dirección inválida','Otra dirección…','Nombre','Dirección Nimiq'],de:['Meine Wallet','Deine Nimiq-Adressen, auf diesem Gerät gespeichert. Sie erscheinen unter „Meine Adresse wählen“, um mit einem Klick eine Zahlungsanforderung zu erstellen.','Keine gespeicherte Adresse.','Adresse hinzufügen','Name (optional)','Hinzufügen','In Nimiq wählen','Löschen','Ungültige Adresse','Andere Adresse…','Name','Nimiq-Adresse'],zh:['我的钱包','保存在此设备上的 Nimiq 地址。它们会出现在“选择我的地址”中，一键创建付款请求。','没有已保存的地址。','添加地址','名称（可选）','添加','在 Nimiq 中选择','删除','地址无效','其他地址…','名称','Nimiq 地址']};
const sentWords={fr:['Mes envois (cashlinks)','Si la personne dit ne rien avoir reçu, tu vois ici si l’argent a été réclamé, par quelle adresse et quand. Tant que ce n’est pas réclamé, tu peux le récupérer. Le suivi reste sur cet appareil.','Aucun envoi pour l’instant.','À réclamer','Réclamé','Non versé','Vérification…','Récupérer','Non réclamé depuis plus de 24 h','Copier la preuve','Preuve copiée','Réclamé par','Destinataire prévu','Supprimer ce suivi ? Si l’argent n’est pas réclamé, tu perdras le moyen de le récupérer.','Transaction de versement','Transaction de réclamation','Statut','Non réclamé au moment de la vérification','Envoyé le','Adresse du cashlink','Montant','Actualiser'],en:['My sends (cashlinks)','If the person says they received nothing, you can see here whether the money was claimed, by which address and when. As long as it is not claimed, you can take it back. The tracking stays on this device.','No send yet.','To claim','Claimed','Not funded','Checking…','Take back','Unclaimed for more than 24 h','Copy proof','Proof copied','Claimed by','Intended recipient','Delete this tracking? If the money is not claimed, you will lose the way to take it back.','Funding transaction','Claim transaction','Status','Not claimed at the time of the check','Sent on','Cashlink address','Amount','Refresh'],es:['Mis envíos (cashlinks)','Si la persona dice que no recibió nada, aquí ves si el dinero fue reclamado, con qué dirección y cuándo. Mientras no se reclame, puedes recuperarlo. El seguimiento queda en este dispositivo.','Aún no hay envíos.','Por reclamar','Reclamado','No financiado','Verificando…','Recuperar','Sin reclamar desde hace más de 24 h','Copiar prueba','Prueba copiada','Reclamado por','Destinatario previsto','¿Eliminar este seguimiento? Si el dinero no se reclama, perderás la forma de recuperarlo.','Transacción de financiación','Transacción de reclamación','Estado','No reclamado en el momento de la comprobación','Enviado el','Dirección del cashlink','Monto','Actualizar'],de:['Meine Sendungen (Cashlinks)','Sagt die Person, sie habe nichts erhalten, siehst du hier, ob das Geld eingelöst wurde, von welcher Adresse und wann. Solange es nicht eingelöst ist, kannst du es zurückholen. Die Nachverfolgung bleibt auf diesem Gerät.','Noch keine Sendung.','Einzulösen','Eingelöst','Nicht finanziert','Prüfung…','Zurückholen','Seit über 24 h nicht eingelöst','Nachweis kopieren','Nachweis kopiert','Eingelöst von','Vorgesehener Empfänger','Diese Nachverfolgung löschen? Ist das Geld nicht eingelöst, verlierst du die Möglichkeit, es zurückzuholen.','Finanzierungstransaktion','Einlösetransaktion','Status','Zum Zeitpunkt der Prüfung nicht eingelöst','Gesendet am','Cashlink-Adresse','Betrag','Aktualisieren'],zh:['我的发送（Cashlink）','如果对方说没收到，你可以在这里查看款项是否已被领取、由哪个地址领取以及何时领取。未被领取之前，你可以取回。跟踪记录仅保存在此设备上。','暂无发送记录。','待领取','已领取','未注资','检查中…','取回','超过 24 小时未领取','复制凭证','凭证已复制','领取地址','预定收件人','删除此跟踪记录？如果款项尚未被领取，你将无法再取回。','注资交易','领取交易','状态','检查时尚未领取','发送于','Cashlink 地址','金额','刷新']};
const FUNDING=[0,130,128,146,135];
const cardPalettes=[
 ['#deefff','#9fc9eb'], // sky
 ['#ece3ff','#c7b1ee'], // lavender
 ['#dcf5e9','#9bd8bc'], // mint
 ['#ffe2ee','#efaac7'], // blush
 ['#ffe7d8','#eeb999'], // peach
 ['#fff4cc','#e8d181'], // butter
 ['#e6ebff','#b0bdf4'], // periwinkle
 ['#dcf6fa','#90d7e5'], // aqua
 ['#f4e2fa','#d9afea'], // orchid
 ['#ffe2dc','#edaea1'], // coral
 ['#ecf6d8','#bed89a'], // pistachio
 ['#e2f4ef','#9ed5c6']  // seafoam
];
// Older cards omitted pastelV2 and retain their original 6-color mapping.
// New cards carry pastelV2:true, selecting 1 of 12 colors deterministically.
function cardColors(id,pastelV2=false){const count=pastelV2===true?cardPalettes.length:6;return cardPalettes[parseInt(id.slice(0,2),16)%count]}
const compact=s=>String(s).replace(/ /g,'').toUpperCase();
function addressValid(address){const s=compact(address);if(!/^NQ\d{2}[0-9ABCDEFGHJKLMNPQRSTUVXY]{32}$/.test(s)||s==='NQ0700000000000000000000000000000000')return false;let rest=0;for(const c of s.slice(4)+s.slice(0,4)){const n=c>='A'?String(c.charCodeAt(0)-55):c;for(const d of n)rest=(rest*10+Number(d))%97}return rest===1}
const encode=s=>encodeURIComponent(s).replace(/[!'()*]/g,c=>'%'+c.charCodeAt(0).toString(16).toUpperCase());
function validate(r){if(r?.pastelV2!==undefined&&typeof r.pastelV2!=='boolean')throw Error('invalid_request');if(!r||r.version!==1||!/^[a-f0-9]{24}$/.test(r.id)||r.network!=='mainnet'||!addressValid(r.address)||r.address!==compact(r.address)||!Number.isSafeInteger(r.luna)||r.luna<=0)throw Error('invalid_request');const units=BigInt(r.luna),fraction=units%100000n;const amount=String(units/100000n)+(fraction?'.'+String(fraction).padStart(5,'0').replace(/0+$/,''):'');if(r.amount!==amount||typeof r.reference!=='string'||new TextEncoder().encode(r.reference).length>64||/[\r\n]/.test(r.reference)||!r.reference.endsWith('PM-'+r.id))throw Error('invalid_request');const domain='wallet.nimiq.com';if(r.link!==`https://${domain}/#_request/${r.address}/${r.amount}/${encode(r.reference)}_`)throw Error('invalid_request');return r}
function checkoutRequest(r){validate(r);return{appName:'PhoenixMail',recipient:r.address,value:r.luna,extraData:r.reference}}
function validateCash(c){if(c?.pastelV2!==undefined&&typeof c.pastelV2!=='boolean')throw Error('invalid_request');if(!c||c.version!==1||!/^[a-f0-9]{24}$/.test(c.id)||c.network!=='mainnet'||!addressValid(c.address)||c.address!==compact(c.address)||!Number.isSafeInteger(c.luna)||c.luna<=0||typeof c.message!=='string'||!String(c.link).startsWith('https://hub.nimiq.com'+'/cashlink/#'))throw Error('invalid_request');return c}
function cashCheckout(c){validateCash(c);return{appName:'PhoenixMail',recipient:c.address,value:c.luna,extraData:new Uint8Array(FUNDING)}}
function create(o){const l=()=>words[o.language()]||words.en;let closeWindow=null;const busy=new Set(),receipts=new Map();const storage=o.storage||root.localStorage;
 const hubUrl=network=>'https://hub.nimiq.com';
 const hub=network=>new root.HubApi(hubUrl(network));
 // Phones (iPhone/iPad Safari, Android) break the popup flow: the wallet page opens without its request. They use the Hub's redirect flow instead and come back to this page with the answer.
 const mobile=()=>{const n=root.navigator;if(!n)return false;return /iPhone|iPad|iPod|Android/i.test(n.userAgent||'')||(n.platform==='MacIntel'&&n.maxTouchPoints>1)};
 const redirectHub=(network,state)=>new root.HubApi(hubUrl(network),new root.HubApi.RedirectRequestBehavior(undefined,state));
 // PhoenixMail sends `Referrer-Policy: same-origin`, so the browser tells the Hub nothing about where the request comes from and the Hub answers "Invalid request" (it needs the
 // requesting origin, above all for the redirect flow). Only for the length of a Hub request the page allows the origin (and nothing else: no path) to be sent.
 async function withHubReferrer(run){if(typeof document==='undefined'||!document.head)return run();const meta=document.createElement('meta');meta.name='referrer';meta.content='strict-origin';document.head.append(meta);try{return await run()}finally{meta.remove()}}
 const PENDING='phoenixmail.nimiq.pending.v1',COMPOSE='phoenixmail.nimiq.compose.v1';
 // The message being written survives the trip to the wallet: its fields are kept in this TAB's sessionStorage (not localStorage: it is an unsent message) and put back on return.
 const session=()=>{try{return o.session||root.sessionStorage}catch{return null}};
 function keepCompose(){try{const c=o.snapshot?.();if(c)session()?.setItem(COMPOSE,JSON.stringify(c))}catch{}}
  const reopen=m=>{if(typeof document!=='undefined')open(m||'request')};
 function backCompose(){try{const raw=session()?.getItem(COMPOSE);session()?.removeItem(COMPOSE);if(raw)o.restore?.(JSON.parse(raw))}catch{}}
 const setPending=p=>{try{storage.setItem(PENDING,JSON.stringify({...p,at:Date.now()}))}catch{}};
 const clearPending=()=>{try{storage.removeItem(PENDING)}catch{}};
 function toast(text){if(typeof document==='undefined')return;const box=document.createElement('div');box.className='nimiq-payment-toast';box.setAttribute('role','status');box.textContent=text;box.style.cssText='position:fixed;left:50%;bottom:24px;transform:translateX(-50%);max-width:90vw;padding:12px 18px;border-radius:12px;background:#1b2238;color:#fff;border:1px solid #6d5dfc;z-index:2147483647;font:14px system-ui,sans-serif;text-align:center';document.body.append(box);setTimeout(()=>box.remove(),7000)}
 const key=r=>'phoenixmail.nimiq.payment.v1.'+r.network+'.'+r.id;
 function remember(r){if(receipts.has(key(r)))return receipts.get(key(r));try{const data=JSON.parse(storage.getItem(key(r))||'null');return data&&data.address===r.address&&data.luna===r.luna&&data.reference===r.reference?data:null}catch{return null}}
 function save(r,hash){const receipt={address:r.address,luna:r.luna,reference:r.reference,hash};receipts.set(key(r),receipt);try{storage.setItem(key(r),JSON.stringify(receipt))}catch{}}
 async function post(path,data,signal){const response=await (o.fetch||root.fetch)(path,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(data),signal});let result;try{result=await response.json()}catch{throw Error(response.status===404?(o.language()==='fr'?'Fonction Nimiq absente du serveur. Mets PhoenixMail à jour puis redémarre-le.':'Nimiq is missing on the server. Update PhoenixMail and restart it.'):(o.language()==='fr'?'Réponse Nimiq du serveur invalide.':'Invalid Nimiq server response.'))}if(!response.ok)throw Error(result?.error||l()[18]);if(!result||typeof result!=='object')throw Error(l()[18]);return result}
 function accept(r,result){if(!result||!/^[a-fA-F0-9]{64}$/.test(result.hash)||!result.raw||compact(result.raw.recipient)!==r.address||result.raw.value!==r.luna)throw Error(l()[18]);const extra=typeof result.raw.extraData==='string'?result.raw.extraData:new TextDecoder().decode(result.raw.extraData);if(extra!==r.reference)throw Error(l()[18]);save(r,result.hash);return{hash:result.hash}}
 async function pay(r,status){validate(r);if(busy.has(key(r)))return;busy.add(key(r));try{let transaction=remember(r);if(!transaction){if(mobile()){setPending({kind:'checkout',request:r});await withHubReferrer(()=>redirectHub(r.network,{phoenixNimiq:'checkout'}).checkout(checkoutRequest(r)));return}const result=await withHubReferrer(()=>hub(r.network).checkout(checkoutRequest(r)));transaction=accept(r,result);status(l()[15],false)}
  const result=await post('/api/nimiq/payment-status',{request:r,hash:transaction.hash});status(result.confirmed===true?l()[14]:l()[15],result.confirmed===true);return{hash:transaction.hash,confirmed:result.confirmed===true}
 }catch(e){status(remember(r)?l()[16]:l()[20],false);throw e}finally{busy.delete(key(r))}}
 // Keep the preview's PhoenixMail brand/center emblem and layout aligned with
 // the approved SecureClaim card. Only the payment-specific controls differ.
 function cardShell(data,heading,network,extra){
  const box=document.createElement('section');box.className='nimiq-payment-card';
  const colors=extra?.colors||cardColors(data.id,data.pastelV2===true);
  box.style.setProperty('--card-bg',colors[0]);
  box.style.setProperty('--card-accent',colors[1]);
  box.dataset.palette=String(parseInt(data.id.slice(0,2),16)%(data.pastelV2===true?cardPalettes.length:6));
  const header=document.createElement('div');header.className='nimiq-card-head';
  const brand=document.createElement('div');brand.className='nimiq-card-brand';
  const bk=o.brand||{}; // Wire's one change: the app's own emblem and name on the card (PhoenixMail's by default)
  const mark=document.createElement('img');mark.src=bk.icon||'brand/phoenixmail-approved-icon-transparent.webp';mark.alt=bk.alt||'PhoenixMail';
  const name=document.createElement('span');name.className='nimiq-card-name';name.append(bk.first||'Phoenix');const mail=document.createElement('strong');mail.textContent=bk.second||'Mail';name.append(mail);
  brand.append(mark,name);
  header.append(brand);if(network){const badge=document.createElement('span');badge.className='nimiq-payment-network';badge.textContent=network;header.append(badge)}
  const title=document.createElement('h3');title.textContent=(extra?.brand||'Nimiq')+' · '+heading;
  const amount=document.createElement('strong');amount.className='nimiq-payment-amount';amount.textContent=data.amount+(extra?.unit||' NIM');
  const content=document.createElement('div');content.className='nimiq-card-content';
  box.append(header,title,amount,content);
  return{box,content};
 }
 function card(r,qr){
  validate(r);
  const {box,content}=cardShell(r,l()[1],'');
  if(qr){const img=document.createElement('img');img.className='nimiq-payment-qr';img.src='data:image/png;base64,'+qr;img.alt='QR Nimiq';content.append(img)}
  const iqon=document.createElement('img');iqon.className='nimiq-address-iqon nimiq-card-iqon';iqon.alt='';iqon.hidden=true;content.append(iqon);root.PhoenixNimiqIdenticons?.paint(iqon,r.address);
  for(const value of [r.address,r.reference]){const text=document.createElement('p');text.textContent=value;content.append(text)}
  const notice=document.createElement('small');notice.textContent=l()[17];
  const status=document.createElement('output');status.role='status';status.textContent=remember(r)?l()[15]:l()[19];
  const button=document.createElement('button');button.type='button';button.className='secondary-btn';button.textContent=remember(r)?l()[13]:l()[12];
  button.onclick=async()=>{button.disabled=true;try{const result=await pay(r,(text,confirmed)=>{status.textContent=text;box.dataset.confirmed=String(confirmed)});if(result?.confirmed)button.hidden=true}catch{}finally{button.disabled=false;button.textContent=l()[13]}};
  content.append(notice,status,button);
  
  return box;
 }
 const cw=()=>cashWords[o.language()]||cashWords.en;
 function cashCard(c,qr){
  validateCash(c);
  const{box,content}=cardShell(c,cw()[1],'');
  if(qr){const img=document.createElement('img');img.className='nimiq-payment-qr';img.src='data:image/png;base64,'+qr;img.alt='QR Nimiq';content.append(img)}
  const iqon=document.createElement('img');iqon.className='nimiq-address-iqon nimiq-card-iqon';iqon.alt='';iqon.hidden=true;content.append(iqon);root.PhoenixNimiqIdenticons?.paint(iqon,c.address);
  if(c.message){const text=document.createElement('p');text.textContent=c.message;content.append(text)}
  const notice=document.createElement('small');notice.textContent=cw()[4];content.append(notice);
  
  return box;
 }
 // A protected send must never preview a bearer Cashlink QR.
 // The ONLY recipient QR is generated after the backend has encrypted and
 // admitted the funded Cashlink and contains the opaque OTP claim URL.
 function protectedPendingCard(c){
  validateCash(c);
  const words={fr:['Paiement protégé','Le destinataire devra vérifier son code reçu par courriel pour ouvrir le paiement.'],en:['Protected payment','The recipient must verify their emailed code to unlock this payment.'],es:['Pago protegido','El destinatario deberá verificar su código por correo para abrir el pago.'],de:['Geschützte Zahlung','Der Empfänger muss den E-Mail-Code bestätigen, um die Zahlung zu öffnen.'],zh:['受保护的付款','收件人必须验证邮件验证码才能解锁付款。']}[o.language()]||['Protected payment','No redeemable QR until protected.'];
  const {box,content}=cardShell(c,words[0],({fr:'Protégé',en:'Protected',es:'Protegido',de:'Geschützt',zh:'受保护'})[o.language()]||'Protected');
  const iqon=document.createElement('img');iqon.className='nimiq-address-iqon nimiq-card-iqon';iqon.alt='Nimiq Cashlink Nimicon';iqon.hidden=true;content.append(iqon);root.PhoenixNimiqIdenticons?.paint(iqon,c.address);
  const message=document.createElement('p');message.textContent=words[1];content.append(message);
  if(c.message){const note=document.createElement('p');note.textContent=c.message;content.append(note)}
  
  return box;
 }
 function acceptCash(c,result){if(!result||!/^[a-fA-F0-9]{64}$/.test(result.hash)||!result.raw||compact(result.raw.recipient)!==c.address||result.raw.value!==c.luna)throw Error(l()[18]);return result.hash}
 const fundingWords=()=>({
  fr:['Paiement envoyé. Confirmation du réseau en cours…','Paiement confirmé. Protection de la carte en cours…','Vérifier et ajouter au courriel (sans repayer)','La confirmation prend plus de temps. Réessaie avec ce bouton : aucun second paiement ne sera effectué.'],
  en:['Payment sent. Waiting for network confirmation…','Payment confirmed. Protecting the card…','Check and add to email (no second payment)','Confirmation is taking longer. Retry with this button; no second payment will be made.'],
  es:['Pago enviado. Esperando confirmación de la red…','Pago confirmado. Protegiendo la tarjeta…','Verificar y añadir al correo (sin volver a pagar)','La confirmación tarda más. Reintenta con este botón; no se realizará otro pago.'],
  de:['Zahlung gesendet. Netzwerkbestätigung wird abgewartet…','Zahlung bestätigt. Karte wird geschützt…','Prüfen und zur E-Mail hinzufügen (ohne erneut zu zahlen)','Die Bestätigung dauert länger. Erneut versuchen; es wird keine zweite Zahlung ausgeführt.'],
  zh:['付款已发送，正在等待网络确认…','付款已确认，正在保护卡片…','检查并添加到邮件（无需再次付款）','确认需要更长时间。请使用此按钮重试；不会再次付款。']
 }[o.language()]||['Payment sent. Waiting for network confirmation…','Payment confirmed. Protecting the card…','Check and add to email (no second payment)','Confirmation is taking longer. Retry; no second payment will be made.']);
 async function waitCashFunding(c,hash,progress){
  const deadline=Date.now()+60000;
  for(let attempt=0;attempt<30;attempt++){
   progress?.(fundingWords()[0]);
   const controller=new root.AbortController(),timer=root.setTimeout(()=>controller.abort(),Math.min(8000,Math.max(1,deadline-Date.now())));
   try{if((await post('/api/nimiq/cashlink-status',{request:c,hash},controller.signal)).confirmed===true)return true}catch{}finally{root.clearTimeout(timer)}
   if(attempt===29||Date.now()>=deadline)break;
   await new Promise(resolve=>root.setTimeout(resolve,2000));
  }
  return false;
 }
 // The data file of the card holds the claim link: it only goes into the message once the wallet has sent the money.
 async function finishCash(c,attachments,hash,protectedRecipient='',progress){
  recordSent(c,hash,protectedRecipient);
  let confirmed=false;
  if(protectedRecipient)confirmed=await waitCashFunding(c,hash,progress);
  else try{confirmed=(await post('/api/nimiq/cashlink-status',{request:c,hash})).confirmed===true}catch{}
  if(protectedRecipient){
   // Never send the original bearer link if OTP admission fails.
   if(!confirmed)throw Error(fundingWords()[3]);
   progress?.(fundingWords()[1]);
   const protectedCard=await post('/api/nimiq/secureclaim-protect',{request:c,hash,recipient:protectedRecipient});
   if(protectedCard.status!=='otp_protected'||!Array.isArray(protectedCard.attachments)||!protectedCard.request?.claimURL)throw Error('Protected payment unavailable');
   if(root.PhoenixNimiqIdenticons&&protectedCard.request.address){
    const iqon=await root.PhoenixNimiqIdenticons.attachment({id:protectedCard.request.claimId,address:protectedCard.request.address});
    protectedCard.attachments.push(iqon);
   }
   progress?.(fundingWords()[1]);
   if(o.bindOTPRecipient&&o.bindOTPRecipient(protectedRecipient)===false)throw Error('Protected payment recipient mismatch');
   o.attach(protectedCard.attachments);
   markSentProtected(c,protectedCard.request.claimURL);
  }else{o.attach(attachments)}
  o.status?.(cw()[6]);return confirmed
 }
 async function fundCash(c,attachments,protectedRecipient='',progress){validateCash(c);
  // OTP requests are accepted only on Mainnet. Reject BEFORE
  // wallet checkout so a failed admission cannot strand funded coins.
  if(protectedRecipient&&(c.network!=='mainnet'||c.luna>1000*100000))throw Error('The maximum for a protected payment is 1,000 NIM');
  const previous=sentList().find(e=>e.id===c.id);
  if(previous?.hash||previous?.fundingHash)return finishCash(c,attachments,previous.hash||previous.fundingHash,protectedRecipient,progress);
  if(previous?.fundingStarted){
   const state=await post('/api/nimiq/cashlink-state',{network:c.network,address:c.address,luna:c.luna});
   if(state.funded===true&&/^[a-fA-F0-9]{64}$/.test(state.fundingHash||''))return finishCash(c,attachments,state.fundingHash,protectedRecipient,progress);
   throw Error('A wallet checkout was already opened. Check My sends; no second payment was started.');
  }
  recordSent(c,'',protectedRecipient);
  const reserved=sentList(),entry=reserved.find(e=>e.id===c.id);if(!entry)throw Error('Cashlink recovery record unavailable');entry.fundingStarted=true;saveSent(reserved);
  if(mobile()){setPending({kind:'cashlink',request:c,attachments,protectedRecipient});keepCompose();await withHubReferrer(()=>redirectHub(c.network,{phoenixNimiq:'cashlink'}).checkout(cashCheckout(c)));return null}const result=await withHubReferrer(()=>hub(c.network).checkout(cashCheckout(c)));return finishCash(c,attachments,acceptCash(c,result),protectedRecipient,progress)}
 // The saved addresses ("My wallet" in the settings): kept on this device, shown with their Nimiq avatar.
 const WALLET='phoenixmail.nimiq.wallet.v1',ww=()=>walletWords[o.language()]||walletWords.en;
 const savedList=()=>{try{const v=JSON.parse(storage.getItem(WALLET)||'[]');return Array.isArray(v)?v.filter(e=>e&&addressValid(e.address)&&e.network==='mainnet').slice(0,50).map(e=>({address:compact(e.address),network:e.network,label:String(e.label||'').slice(0,40),at:Number(e.at)||0})):[]}catch{return[]}};
 const saveList=list=>{try{storage.setItem(WALLET,JSON.stringify(list.slice(0,50)))}catch{}};
 function addSaved(address,network,label){address=compact(address);if(!addressValid(address)||network!=='mainnet')return false;const list=savedList(),i=list.findIndex(e=>e.address===address&&e.network===network);if(i>=0){if(label&&!list[i].label){list[i].label=label;saveList(list)}}else{list.unshift({address,network,label:String(label||'').slice(0,40),at:Date.now()});saveList(list)}return true}
 const shortAddress=a=>a.slice(0,4)+' '+a.slice(4,8)+' … '+a.slice(-4);
 function iqonImg(address){const img=document.createElement('img');img.className='nimiq-address-iqon nimiq-wallet-iqon';img.alt='';img.hidden=true;root.PhoenixNimiqIdenticons?.paint(img,address);return img}
 let walletHost=null;
 function mountWallet(host){if(host)walletHost=host;host=walletHost;if(!host||typeof document==='undefined')return;const t=ww(),w=l();host.textContent='';host.classList.add('nimiq-wallet-page');
  const head=document.createElement('div');head.className='settings-section-head';const box=document.createElement('div'),h=document.createElement('h3'),p=document.createElement('p');h.textContent=t[0];p.textContent=t[1];box.append(h,p);head.append(box);host.append(head);
  const list=document.createElement('div');list.className='nimiq-wallet-list';const saved=savedList();
  if(!saved.length){const empty=document.createElement('p');empty.className='nimiq-wallet-empty';empty.textContent=t[2];list.append(empty)}
  for(const e of saved){const row=document.createElement('div');row.className='nimiq-wallet-row';const name=document.createElement('input');name.type='text';name.maxLength=40;name.value=e.label;name.placeholder=shortAddress(e.address);name.setAttribute('aria-label',t[10]);name.onchange=()=>{const all=savedList(),i=all.findIndex(x=>x.address===e.address&&x.network===e.network);if(i>=0){all[i].label=name.value.trim().slice(0,40);saveList(all)}};
   const info=document.createElement('div');info.className='nimiq-wallet-info';const addr=document.createElement('small');addr.textContent=e.address.replace(/(.{4})/g,'$1 ').trim();const net=document.createElement('span');net.className='nimiq-payment-network';net.textContent=w[e.network!=='mainnet'?6:7];info.append(name,addr);
   const del=document.createElement('button');del.type='button';del.className='secondary-btn';del.textContent='✕';del.title=t[7];del.setAttribute('aria-label',t[7]);del.onclick=()=>{saveList(savedList().filter(x=>!(x.address===e.address&&x.network===e.network)));mountWallet()};
   row.append(iqonImg(e.address),info,del);list.append(row)}
  host.append(list);
  const form=document.createElement('form');form.className='nimiq-wallet-add';const title=document.createElement('strong');title.textContent=t[3];
  const net=document.createElement('select');net.hidden=true;net.setAttribute('aria-label',w[5]);for(const [v,n]of [['mainnet',w[7]]]){const op=document.createElement('option');op.value=v;op.textContent=n;net.append(op)}
  const addr=document.createElement('input');addr.type='text';addr.autocomplete='off';addr.placeholder='NQ00 0000 0000 0000 0000 0000 0000 0000 0000';addr.setAttribute('aria-label',t[11]);
  const name=document.createElement('input');name.type='text';name.maxLength=40;name.placeholder=t[4];name.setAttribute('aria-label',t[10]);
  const add=document.createElement('button');add.type='submit';add.className='secondary-btn';add.textContent=t[5];const pick=document.createElement('button');pick.type='button';pick.className='secondary-btn';pick.textContent=t[6];const status=document.createElement('output');status.role='status';
  form.onsubmit=e=>{e.preventDefault();if(!addSaved(addr.value,net.value,name.value.trim())){status.textContent=t[8];return}mountWallet()};
  pick.onclick=async()=>{pick.disabled=true;status.textContent='';try{if(mobile()){setPending({kind:'address',network:net.value,account:o.account?.()||'default',mode:'wallet',label:name.value.trim()});await withHubReferrer(()=>redirectHub(net.value,{phoenixNimiq:'address'}).chooseAddress({appName:'PhoenixMail'}));return}const chosen=await withHubReferrer(()=>hub(net.value).chooseAddress({appName:'PhoenixMail'}));if(!chosen||!addSaved(chosen.address,net.value,name.value.trim()))throw Error('state');mountWallet()}catch{status.textContent=w[20]}finally{pick.disabled=false}};
  form.append(title,net,addr,name,add,pick,status);host.append(form);if(usdt){const usdtWallet=document.createElement('section');host.append(usdtWallet);usdt.mountWallet(usdtWallet)}const sent=document.createElement('section');host.append(sent);renderSent(sent);if(usdt){const usdtSent=document.createElement('section');host.append(usdtSent);usdt.mountSent(usdtSent)}}
 // The cashlinks this device made: the sender's safety net. The chain says whether and by whom each was claimed; until then the sender can take the money back with the same link.
 const SENT='phoenixmail.nimiq.sent.v1',sw=()=>sentWords[o.language()]||sentWords.en;
 const sentList=()=>{try{const v=JSON.parse(storage.getItem(SENT)||'[]');return Array.isArray(v)?v.filter(e=>{try{validateCash(e);return true}catch{return false}}).slice(0,50):[]}catch{return[]}};
 const saveSent=list=>{const encoded=JSON.stringify(list.slice(0,50));storage.setItem(SENT,encoded);if(storage.getItem(SENT)!==encoded)throw Error('Cannot save the Cashlink recovery record on this device');};
 function recordSent(c,hash,otpRecipient=''){const old=sentList().find(e=>e.id===c.id);const list=sentList().filter(e=>e.id!==c.id);list.unshift({...(old||{}),...c,hash:hash||old?.hash||'',otpRecipient:otpRecipient||old?.otpRecipient||'',to:String(old?.to||o.recipient?.()||'').slice(0,200),at:old?.at||Date.now()});saveSent(list)}
 function markSentProtected(c,claimURL){const all=sentList(),i=all.findIndex(e=>e.id===c.id);if(i<0)return;all[i].otpProtected=true;all[i].otpClaimURL=claimURL;saveSent(all)}
 const when=ms=>{try{return new Date(ms).toLocaleString(o.language())}catch{return String(ms)}};
 function proofText(e){const t=sw(),lines=['PhoenixMail · Cashlink Nimiq',t[20]+' : '+e.amount+' NIM',t[18]+' : '+when(e.at),t[19]+' : '+e.address];if(e.fundingHash||e.hash)lines.push(t[14]+' : '+(e.fundingHash||e.hash));lines.push(t[16]+' : '+(e.claimed?t[4]+' · '+t[11]+' '+e.claimTo+(e.claimedAt?' · '+when(e.claimedAt):'')+(e.claimHash?' · '+t[15]+' '+e.claimHash:''):t[17]+' ('+when(e.checkedAt||Date.now())+')'));return lines.join('\n')}
 async function refreshSent(host){const now=Date.now();for(const e of sentList()){if(e.claimed||now-(e.checkedAt||0)<30000)continue;try{const st=await post('/api/nimiq/cashlink-state',{network:e.network,address:e.address,luna:e.luna});const all=sentList(),i=all.findIndex(x=>x.id===e.id);if(i>=0){Object.assign(all[i],{funded:st.funded===true,fundingHash:st.fundingHash||all[i].fundingHash||'',claimed:st.claimed===true,claimHash:st.claimHash||'',claimTo:st.claimTo||'',claimedAt:st.claimedAt||0,checkedAt:Date.now()});saveSent(all)}}catch{}}if(host.isConnected)renderSent(host,false)}
 function renderSent(host,check=true){const t=sw();host.textContent='';host.className='nimiq-sent';const head=document.createElement('div');head.className='nimiq-sent-head';const h=document.createElement('h3');h.textContent=t[0];const refresh=document.createElement('button');refresh.type='button';refresh.className='secondary-btn';refresh.textContent=t[21];refresh.onclick=()=>{const all=sentList();all.forEach(e=>{e.checkedAt=0});saveSent(all);refreshSent(host)};head.append(h,refresh);const intro=document.createElement('p');intro.textContent=t[1];host.append(head,intro);
  const list=sentList();if(!list.length){const empty=document.createElement('p');empty.className='nimiq-wallet-empty';empty.textContent=t[2];host.append(empty);return}
  for(const e of list){const late=!e.claimed&&e.funded!==false&&Date.now()-e.at>864e5;const row=document.createElement('div');row.className='nimiq-wallet-row nimiq-sent-row';if(late)row.dataset.late='true';
   const info=document.createElement('div');info.className='nimiq-wallet-info';const title=document.createElement('strong');title.textContent=e.amount+' NIM'+(e.message?' · '+e.message:'');const line=document.createElement('small');line.textContent=when(e.at)+(e.to?' · '+t[12]+' : '+e.to:'');info.append(title,line);
   const state=document.createElement('span');state.className='nimiq-payment-network nimiq-sent-state';state.textContent=e.claimed?t[4]:e.funded===false?t[5]:e.checkedAt?t[3]:t[6];info.append(state);
   if(e.claimed){const by=document.createElement('small');by.textContent=t[11]+' '+(e.claimTo?shortAddress(e.claimTo):'')+(e.claimedAt?' · '+when(e.claimedAt):'');info.append(by)}
   if(late){const warn=document.createElement('small');warn.className='nimiq-sent-late';warn.textContent=t[8];info.append(warn)}
   const actions=document.createElement('div');actions.className='nimiq-sent-actions';
   if(!e.claimed&&e.funded!==false){const take=document.createElement('button');take.type='button';take.className='secondary-btn';take.textContent=t[7];take.onclick=()=>root.open(e.link,'_blank','noopener,noreferrer');actions.append(take)}
   if(e.otpRecipient && (e.hash||e.fundingHash) && e.network==='mainnet' && !e.claimed){
    const retry=document.createElement('button');retry.type='button';retry.className='secondary-btn';
    retry.textContent=({fr:'Terminer la protection (sans repayer)',en:'Finish protection (no second payment)',es:'Completar la protección (sin volver a pagar)',de:'Schutz abschließen (nicht erneut zahlen)',zh:'完成保护（无需再次付款）'}[o.language()]||'Finish protection (no second payment)');
    retry.onclick=async()=>{
      retry.disabled=true;
      try{if(o.openRecoveredClaim && o.openRecoveredClaim(e.otpRecipient)===false)throw Error('The recipient of the message differs from the protected payment recipient');
       await finishCash(e,[],e.hash||e.fundingHash,e.otpRecipient,text=>{retry.textContent=text});o.onRecoveredAttached?.();renderSent(host,false);
      }catch(error){retry.disabled=false;const note=document.createElement('small');note.setAttribute('role','alert');note.textContent=(error?.message||'Protected payment unavailable')+' — No second payment needed.';info.append(note)}
    };actions.append(retry);
   }
   if(e.funded!==false){const copy=document.createElement('button');copy.type='button';copy.className='secondary-btn';copy.textContent=t[9];copy.onclick=async()=>{try{await root.navigator.clipboard.writeText(proofText(e));copy.textContent=t[10]}catch{}};actions.append(copy)}
   const del=document.createElement('button');del.type='button';del.className='secondary-btn';del.textContent='✕';del.title=ww()[7];del.setAttribute('aria-label',ww()[7]);del.onclick=()=>{if(!e.claimed&&e.funded!==false&&root.confirm&&!root.confirm(t[13]))return;saveSent(sentList().filter(x=>x.id!==e.id));renderSent(host,false)};actions.append(del);
   row.append(iqonImg(e.address),info,actions);host.append(row)}
  if(check)refreshSent(host)}
 function receive(host,r,qr){if(!r)return;try{host.append(card(r,qr))}catch{const error=document.createElement('p');error.textContent=l()[18];host.append(error)}}
 function encrypted(host,files,verified){const requests=files.filter(f=>f.type===TYPE);if(!requests.length)return;if(requests.length!==1){receive(host,{});return}const file=requests[0];try{const r=validate(JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(file.data),c=>c.charCodeAt(0)))));const image=files.find(f=>f.name==='phoenixmail-payment-'+r.id+'.png'&&f.type==='image/png');receive(host,r,verified?image?.data:undefined)}catch{receive(host,{})}}
 const usdt=root.PhoenixUSDT?root.PhoenixUSDT.create({o,storage,cardShell}):null;
 function open(mode='request',asset='nim'){
  const w=l(),ui={fr:['Formulaire','Aperçu de la carte','Crée une carte pour voir son aperçu.','Fermer','Déplacer la fenêtre','Modifier'],en:['Form','Card preview','Create a card to preview it.','Close','Move window','Edit'],es:['Formulario','Vista previa','Crea una tarjeta para verla.','Cerrar','Mover ventana','Editar'],de:['Formular','Kartenvorschau','Erstelle eine Karte für die Vorschau.','Schließen','Fenster bewegen','Bearbeiten'],zh:['表单','卡片预览','创建卡片以查看预览。','关闭','移动窗口','编辑']}[o.language()]||['Form','Card preview','Create a card to preview it.','Close','Move window','Edit'];
  closeWindow?.();
  const modal=document.createElement('div');modal.className='nimiq-payment-modal';
  const win=document.createElement('section');win.className='nimiq-payment-window';win.setAttribute('role','dialog');win.setAttribute('aria-label',w[1]);win.dataset.view='form';
  const head=document.createElement('div');head.className='nimiq-payment-head';head.tabIndex=0;head.title=ui[4];
  const title=document.createElement('h3');title.textContent='⤢ '+paymentsLabel();
  const close=document.createElement('button');close.type='button';close.textContent='×';close.setAttribute('aria-label',ui[3]);head.append(title,close);win.append(head);
  let resize=null,observer=null;const dismiss=()=>{if(resize)root.removeEventListener('resize',resize);observer?.disconnect();root.visualViewport?.removeEventListener('resize',resize);root.visualViewport?.removeEventListener('scroll',resize);modal.remove();closeWindow=null;o.button?.focus()};closeWindow=dismiss;close.onclick=dismiss;
  const tabs=document.createElement('div');tabs.className='nimiq-payment-tabs';
  const formTab=document.createElement('button'),previewTab=document.createElement('button');formTab.type=previewTab.type='button';formTab.textContent=ui[0];previewTab.textContent=ui[1];previewTab.disabled=true;tabs.append(formTab,previewTab);win.append(tabs);
  const view=name=>{win.dataset.view=name;formTab.setAttribute('aria-pressed',String(name==='form'));previewTab.setAttribute('aria-pressed',String(name==='preview'))};formTab.onclick=()=>view('form');previewTab.onclick=()=>view('preview');view('form');
  const body=document.createElement('div');body.className='nimiq-payment-body';const panel=document.createElement('form');panel.className='nimiq-payment-panel';const previewHost=document.createElement('section');previewHost.className='nimiq-payment-preview';previewHost.setAttribute('aria-label',ui[1]);const hint=document.createElement('p');hint.textContent=ui[2];previewHost.append(hint);body.append(panel,previewHost);win.append(body);
  const modeLabels={fr:['Demander un paiement','Envoyer NIM'],en:['Request payment','Send NIM'],es:['Solicitar pago','Enviar NIM'],de:['Zahlung anfordern','NIM senden'],zh:['请求付款','发送 NIM']}[o.language()]||['Request payment','Send NIM'];
  const modes=document.createElement('div');modes.className='nimiq-payment-controls';for(const [value,label]of [['request',modeLabels[0]],['send',modeLabels[1]]]){const button=document.createElement('button');button.type='button';button.textContent=label;button.setAttribute('aria-pressed',String(mode===value));button.onclick=()=>{dismiss();open(value,asset)};modes.append(button)}panel.append(modes);
  // NIM or one of the two USDT variants; each choice reopens the window for that asset.
  const assetNames={nim:'NIM',usdt_ethereum:'USDT · Ethereum',usdc_polygon:'USDC · Polygon',usdt_polygon:'USDT · Polygon'},assetRow=document.createElement('div');assetRow.className='nimiq-payment-controls nimiq-payment-assets';
  if(usdt)for(const [value,label] of Object.entries(assetNames)){const button=document.createElement('button');button.type='button';button.textContent=label;button.setAttribute('aria-pressed',String(asset===value));button.onclick=()=>{dismiss();open(mode,value)};assetRow.append(button)}
  if(usdt)panel.insertBefore(assetRow,modes);
  if(usdt&&asset!=='nim'){
   const u=usdt.labels((root.PhoenixUSDT&&root.PhoenixUSDT.CHAINS[asset]||{}).symbol);modes.children[0].textContent=u.modes[0];modes.children[1].textContent=u.modes[1];
   usdt.panel({mode,asset,panel,previewHost,view,previewTab,modal,dismiss,modeButtons:[...modes.querySelectorAll('button'),...assetRow.querySelectorAll('button')]});
   return finish();
  }
  const fields={};for(const [name,label,type,value]of [['network',w[5],'select','mainnet'],['address',w[2],'text',''],['amount',w[3],'text',''],['reference',w[4],'text','']]){const el=document.createElement('label');el.textContent=label;const input=document.createElement(type==='select'?'select':'input');input.name=name;if(type==='select'){for(const [v,t]of [['mainnet',w[7]]]){const option=document.createElement('option');option.value=v;option.textContent=t;input.append(option)}}else{input.type=type;input.autocomplete='off';if(name==='amount'){input.inputMode='decimal';input.placeholder='12.50000'}if(name==='reference')input.maxLength=32;else input.required=true}input.value=value;fields[name]=input;if(name==='network')el.hidden=true;el.append(input);panel.append(el)}
  const addressWrap=document.createElement('div');addressWrap.className='nimiq-address-field';const addressIqon=document.createElement('img');addressIqon.className='nimiq-address-iqon';addressIqon.alt='';addressIqon.hidden=true;fields.address.before(addressWrap);addressWrap.append(addressIqon,fields.address);const updateIqon=()=>root.PhoenixNimiqIdenticons?.paint(addressIqon,fields.address.value);fields.address.addEventListener('input',updateIqon);
  const addressKey=()=>"phoenixmail.nimiq.receive.v1."+(o.account?.()||'default')+'.'+fields.network.value;const fillAddress=()=>{try{fields.address.value=mode==='request'?(storage.getItem(addressKey())||savedList().find(e=>e.network===fields.network.value)?.address||''):''}catch{}};fields.network.onchange=()=>{picker.hidden=true;fillAddress();updateIqon()};fillAddress();updateIqon();
  const choose=document.createElement('button');choose.type='button';choose.className='secondary-btn';choose.textContent=w[8];choose.hidden=mode==='send';panel.append(choose);
  const controls=document.createElement('div');controls.className='nimiq-payment-controls';const submit=document.createElement('button');submit.type='submit';submit.className='secondary-btn';submit.textContent=w[9];const cancel=document.createElement('button');cancel.type='button';cancel.className='secondary-btn';cancel.textContent=w[11];cancel.onclick=dismiss;controls.append(submit,cancel);panel.append(controls);const status=document.createElement('output');status.role='status';panel.append(status);
  if(mode==='send'){fields.address.closest('label').hidden=true;addressWrap.hidden=true;fields.address.required=false;fields.address.value='';fields.reference.parentElement.firstChild.textContent=cw()[0];const how=document.createElement('p');how.className='nimiq-payment-how';how.textContent=cw()[3];modes.after(how)}
  let protectInput=null,protectEmail=null,legacyInput=null,otpReady=false,otpDelivery='';
  if(mode==='send'){
   const captions={
    fr:['Protégé','Courriel du destinataire','SecureClaim et l’envoi MailerSend doivent être configurés pour activer ce paiement.','Code de six chiffres par courriel · 10 minutes · maximum 1 000 NIM.','','Le destinataire doit vérifier son courriel avant de pouvoir réclamer les NIM.'],
    en:['Protected','Recipient email','Configure SecureClaim and MailerSend email delivery to enable this payment.','Six-digit email code · 10 minutes · maximum 1,000 NIM.','','The recipient must verify their email before claiming the NIM.'],
    es:['Protegido','Correo del destinatario','Configura SecureClaim y el envío MailerSend para activar este pago.','Código de seis dígitos · 10 minutos · máximo 1.000 NIM.','','El destinatario debe verificar su correo antes de recibir los NIM.'],
    de:['Geschützt','Empfänger-E-Mail','SecureClaim und MailerSend müssen für diese Zahlung eingerichtet sein.','Sechsstelliger E-Mail-Code · 10 Minuten · maximal 1.000 NIM.','','Der Empfänger muss seine E-Mail vor dem Einlösen der NIM bestätigen.'],
    zh:['受保护','收件人邮箱','请配置 SecureClaim 和 MailerSend 邮件发送以启用此付款。','六位邮件验证码 · 10 分钟 · 上限 1,000 NIM。','','收件人必须验证邮箱后才能领取 NIM。']
   }[o.language()]||['Protected','Recipient email','Configure SecureClaim and MailerSend email delivery.','Six-digit email code · 10 minutes · maximum 1,000 NIM.','','The recipient must verify their email before claiming the NIM.'];
   const row=document.createElement('label');row.className='nimiq-otp-option';
   protectInput=document.createElement('input');protectInput.type='radio';protectInput.name='phoenixmail-payment-method';protectInput.value='otp';protectInput.disabled=true;row.append(protectInput,document.createTextNode(' '+captions[0]));
   const emailRow=document.createElement('label');emailRow.hidden=true;emailRow.textContent=captions[1];
   protectEmail=document.createElement('input');protectEmail.type='email';protectEmail.autocomplete='email';protectEmail.value=o.recipient?.()||'';emailRow.append(protectEmail);
   const otpHint=document.createElement('small');otpHint.className='nimiq-payment-how';otpHint.setAttribute('role','status');otpHint.textContent=captions[2];
   // A plain bearer Cashlink is NEVER the default or a silent fallback.
   // The sender explicitly chooses the classic payment method; OTP never falls back.
   const classicWords={fr:['Non protégé','Toute personne qui possède le lien ou le QR peut réclamer les NIM.'],en:['Non-protected','Anyone with the link or QR can claim the NIM.'],es:['Sin protección','Cualquier persona con el enlace o QR puede recibir los NIM.'],de:['Ungeschützt','Jeder mit dem Link oder QR-Code kann die NIM einlösen.'],zh:['未受保护','任何拥有链接或二维码的人都可以领取 NIM。']}[o.language()]||['Non-protected','Anyone with the link or QR can claim the NIM.'];
   const classicCaption=classicWords[0];
   const classicRow=document.createElement('label');classicRow.className='nimiq-otp-option';
   legacyInput=document.createElement('input');legacyInput.type='radio';legacyInput.name='phoenixmail-payment-method';legacyInput.value='classic';
   const classicText=document.createElement('span'),classicTitle=document.createElement('strong'),classicNote=document.createElement('small');classicTitle.textContent=classicCaption;classicNote.textContent=classicWords[1];classicText.append(classicTitle,classicNote);classicRow.append(legacyInput,classicText);
   panel.insertBefore(row,controls);panel.insertBefore(emailRow,controls);panel.insertBefore(otpHint,controls);panel.insertBefore(classicRow,controls);
   protectInput.onchange=()=>{
    if(protectInput.checked)legacyInput.checked=false;
    emailRow.hidden=!protectInput.checked;protectEmail.required=protectInput.checked;
    otpHint.textContent=protectInput.checked?captions[5]:otpReady?captions[3]:captions[2];
   };
   legacyInput.onchange=()=>{
    if(legacyInput.checked)protectInput.checked=false;
    else if(otpReady&&fields.network.value==='mainnet')protectInput.checked=true;
    protectInput.onchange();
   };
   const updateOTP=()=>{
    protectInput.disabled=!otpReady||fields.network.value!=='mainnet';
    protectInput.checked=!protectInput.disabled&&!legacyInput.checked;
    protectInput.onchange();
    otpHint.textContent=fields.network.value!=='mainnet'?captions[2]:(otpReady?captions[3]:captions[2]);
   };
   fields.network.addEventListener('change',updateOTP);
   (o.fetch||root.fetch)('/api/nimiq/secureclaim-protect',{cache:'no-store'}).then(r=>r.ok?r.json():null).then(v=>{otpReady=v?.enabled===true&&v?.delivery==='email'&&v?.network==='mainnet'&&v?.mainnet===true;otpDelivery=v?.delivery||'';updateOTP()}).catch(()=>{otpReady=false;updateOTP()});
  }
  // Saved addresses of this network are offered first; "Another address…" (or no saved address at all) opens the Nimiq wallet and saves what it returns.
  const picker=document.createElement('div');picker.className='nimiq-address-picker';picker.hidden=true;choose.after(picker);
  const hubChoose=async()=>{choose.disabled=true;try{if(mobile()){setPending({kind:'address',network:fields.network.value,account:o.account?.()||'default',mode});keepCompose();await withHubReferrer(()=>redirectHub(fields.network.value,{phoenixNimiq:'address'}).chooseAddress({appName:'PhoenixMail'}));return}const selected=await withHubReferrer(()=>hub(fields.network.value).chooseAddress({appName:'PhoenixMail'}));fields.address.value=selected.address;updateIqon();addSaved(selected.address,fields.network.value);try{storage.setItem(addressKey(),selected.address)}catch{}}catch{status.textContent=w[20]}finally{choose.disabled=false}};
  choose.onclick=()=>{const list=savedList().filter(e=>e.network===fields.network.value);if(!list.length){picker.hidden=true;return hubChoose()}if(!picker.hidden){picker.hidden=true;return}
   const rows=list.map(e=>{const b=document.createElement('button');b.type='button';b.className='nimiq-address-option';const text=document.createElement('span');const strong=document.createElement('strong');strong.textContent=e.label||shortAddress(e.address);const small=document.createElement('small');small.textContent=shortAddress(e.address);text.append(strong,small);b.append(iqonImg(e.address),text);b.onclick=()=>{fields.address.value=e.address;updateIqon();try{storage.setItem(addressKey(),e.address)}catch{}picker.hidden=true};return b});
   const other=document.createElement('button');other.type='button';other.className='nimiq-address-option nimiq-address-other';other.textContent=ww()[9];other.onclick=()=>{picker.hidden=true;hubChoose()};picker.replaceChildren(...rows,other);picker.hidden=false};
  let paymentLocked=false;
  panel.onsubmit=async event=>{event.preventDefault();if(paymentLocked)return;submit.disabled=true;status.textContent='';try{if(mode==='send'){
   // Fail closed before creating/funding a bearer Cashlink. A missing
   // OTP backend must never silently fall back to an unprotected QR.
   const useOTP=protectInput?.checked===true;
   if(!useOTP&&!legacyInput?.checked){
    status.textContent=({fr:'Envoi bloqué : le paiement protégé est indisponible. Aucun Cashlink n’a été créé. Démarre NimiqSecureClaim ou choisis explicitement un paiement non protégé.',en:'Sending blocked: protected payment unavailable. No Cashlink was created. Start NimiqSecureClaim or explicitly choose a non-protected payment.'}[o.language()]||'Sending blocked: protected payment unavailable. No Cashlink was created.');
    return;
   }
   if(useOTP&&(!otpReady||fields.network.value!=='mainnet'||!protectEmail?.checkValidity())){
    status.textContent='Protected payment service or recipient unavailable';return;
   }
   const otpRecipient=useOTP?protectEmail.value.trim():'';
   if(otpRecipient&&o.bindOTPRecipient&&o.bindOTPRecipient(otpRecipient)===false){
    status.textContent='The protected payment recipient must be the single To address (no Cc/Bcc).';return;
   }
   const made=await post('/api/nimiq/cashlink-request',{network:fields.network.value,amount:fields.amount.value.trim(),message:fields.reference.value.trim()});if(!modal.isConnected)return;validateCash(made.request);if(root.PhoenixNimiqIdenticons){made.attachments.push(await root.PhoenixNimiqIdenticons.attachment(made.request));if(!modal.isConnected)return}const png=made.attachments.find(a=>a.type==='image/png'||a.contentType==='image/png');const shown=useOTP?protectedPendingCard(made.request):cashCard(made.request,png?.data);const fund=document.createElement('button');fund.type='button';fund.className='secondary-btn';fund.textContent=cw()[2];fund.onclick=async()=>{if(fund.disabled)return;fund.disabled=true;notice.textContent='';try{paymentLocked=true;submit.disabled=true;back.disabled=true;
    for(const input of panel.querySelectorAll('input,select'))input.disabled=true;
    for(const button of modes.querySelectorAll('button'))button.disabled=true;
    const recipient=otpRecipient;
    if(useOTP){
     const response=await (o.fetch||root.fetch)('/api/nimiq/secureclaim-protect',{cache:'no-store'}).then(r=>r.ok?r.json():null).catch(()=>null);
     if(response?.enabled!==true||response?.delivery!=='email')throw Error('Protected payment service unavailable — wallet checkout cancelled before payment');
     if(o.bindOTPRecipient&&o.bindOTPRecipient(recipient)===false)throw Error('Protected payment recipient mismatch');
    }else if(!legacyInput?.checked)throw Error('Unprotected Cashlink not acknowledged');
    const done=await fundCash(made.request,made.attachments,recipient,text=>{if(!modal.isConnected)throw Error('Payment saved in My sends');notice.textContent=text;status.textContent=text});if(done===null)return;dismiss()
   }catch(error){notice.textContent=error?.message||w[20];status.textContent=notice.textContent;view('preview')}
   finally{
    fund.disabled=false;
    const saved=sentList().find(e=>e.id===made.request.id);
    paymentLocked=saved?.fundingStarted===true;submit.disabled=paymentLocked;back.disabled=paymentLocked;
    if(paymentLocked)fund.textContent=fundingWords()[2];
    else{for(const input of panel.querySelectorAll('input,select'))input.disabled=false;for(const button of modes.querySelectorAll('button'))button.disabled=false;if(protectInput)protectInput.disabled=!otpReady}
   }};const back=document.createElement('button');back.type='button';back.className='secondary-btn nimiq-payment-edit';back.textContent=ui[5];back.onclick=()=>{view('form');fields.amount.focus()};const notice=document.createElement('output');notice.setAttribute('role','status');notice.setAttribute('aria-live','polite');notice.style.cssText='display:block;padding:12px 4px;line-height:1.5;text-align:center;overflow-wrap:anywhere';shown.append(fund,notice);previewHost.replaceChildren(shown,back);previewTab.disabled=false;view('preview');return}const result=await post('/api/nimiq/payment-request',Object.fromEntries(Object.entries(fields).map(([k,v])=>[k,v.value.trim()])));if(!modal.isConnected)return;validate(result.request);if(root.PhoenixNimiqIdenticons){result.attachments.push(await root.PhoenixNimiqIdenticons.attachment(result.request));if(!modal.isConnected)return}const qr=result.attachments.find(a=>a.type==='image/png');const preview=card(result.request,qr?.data);if(mode==='request'){preview.querySelector('button').remove();const add=document.createElement('button');add.type='button';add.className='secondary-btn';add.textContent=w[10];add.onclick=()=>{o.attach(result.attachments);o.status(w[21]);dismiss()};preview.append(add)}const edit=document.createElement('button');edit.type='button';edit.className='secondary-btn nimiq-payment-edit';edit.textContent=ui[5];edit.onclick=()=>{view('form');fields.amount.focus()};previewHost.replaceChildren(preview,edit);previewTab.disabled=false;view('preview')}catch{status.textContent=w[18]}finally{submit.disabled=paymentLocked}};
  return finish();
  function finish(){
  modal.append(win);document.body.append(modal);
  let drag=null;head.onpointerdown=e=>{if(e.button!==0||e.target.closest('button'))return;const r=win.getBoundingClientRect();drag={x:e.clientX-r.left,y:e.clientY-r.top};head.focus({preventScroll:true});head.setPointerCapture?.(e.pointerId);e.preventDefault()};
  const place=(x,y)=>{const r=win.getBoundingClientRect();win.style.margin='0';win.style.left=Math.max(0,Math.min(innerWidth-r.width,x))+'px';win.style.top=Math.max(0,Math.min(innerHeight-r.height,y))+'px'};
  resize=()=>{if(!win.isConnected)return;if(root.PhoenixViewport){root.PhoenixViewport.center(win);return}const r=win.getBoundingClientRect();place(r.left,r.top)};root.addEventListener('resize',resize);root.visualViewport?.addEventListener('resize',resize);root.visualViewport?.addEventListener('scroll',resize);if(root.ResizeObserver){observer=new root.ResizeObserver(resize);observer.observe(win)};
  head.onpointermove=e=>{if(drag)place(e.clientX-drag.x,e.clientY-drag.y)};for(const event of ['pointerup','pointercancel','lostpointercapture'])head.addEventListener(event,()=>drag=null);
  head.onkeydown=e=>{if(e.target.closest('button'))return;const dx={ArrowLeft:-20,ArrowRight:20}[e.key]||0,dy={ArrowUp:-20,ArrowDown:20}[e.key]||0;if(dx||dy){e.preventDefault();const r=win.getBoundingClientRect();place(r.left+dx,r.top+dy)}};
  win.onkeydown=e=>{if(e.key==='Escape'){e.preventDefault();e.stopPropagation();dismiss()}};head.focus({preventScroll:true});resize();
  return win;
  }
 }

 // Back from the Hub's redirect (phones): finish the payment or keep the chosen address, then say what happened.
 async function resume(){let p=null;try{p=JSON.parse(storage.getItem(PENDING)||'null')}catch{}
  if(!p||!root.HubApi||!root.HubApi.RequestType||!root.HubApi.RedirectRequestBehavior)return;
  if(Date.now()-p.at>30*60*1000||(p.network||p.request?.network)!=='mainnet'){clearPending();return}
  const network=p.network||p.request.network,api=redirectHub(network,{phoenixNimiq:p.kind});
  const cancelled=()=>{clearPending();toast(l()[20]);if(p.kind==='address'&&p.mode==='wallet'){o.openWallet?.();return}if(p.kind==='address'){backCompose();reopen(p.mode)}if(p.kind==='cashlink'){backCompose();reopen('send')}};
  api.on(root.HubApi.RequestType.CHECKOUT,async result=>{clearPending();try{if(p.kind==='cashlink'){validateCash(p.request);const hash=acceptCash(p.request,result);backCompose();const confirmed=await finishCash(p.request,p.attachments,hash,p.protectedRecipient||'',text=>o.status?.(text));toast(cw()[6]+(confirmed?' ✓':''));return}if(p.kind!=='checkout')throw Error('state');validate(p.request);const done=accept(p.request,result);toast(l()[15]);const status=await post('/api/nimiq/payment-status',{request:p.request,hash:done.hash});toast(status.confirmed===true?l()[14]:l()[15])}catch{toast(l()[16])}},cancelled);
  api.on(root.HubApi.RequestType.CHOOSE_ADDRESS,selected=>{clearPending();try{if(p.kind!=='address'||!selected||!addressValid(selected.address))throw Error('state');addSaved(selected.address,p.network,p.label);if(p.mode==='wallet'){toast(ww()[0]+' ✓');o.openWallet?.();return}storage.setItem('phoenixmail.nimiq.receive.v1.'+p.account+'.'+p.network,compact(selected.address));toast(l()[8]+' ✓');backCompose();reopen(p.mode)}catch{toast(l()[20])}},cancelled);
  try{await api.checkRedirectResponse()}catch{}}
 const paymentsLabel=()=>({fr:'Paiements',en:'Payments',es:'Pagos',de:'Zahlungen',zh:'付款'})[o.language()]||'Payments';
 function refresh(){if(!o.button)return;o.button.textContent='';const img=document.createElement('img');img.src='brand/nimiq-logo.svg';img.alt='';const coin=document.createElement('img');coin.src='brand/usdt-coin.png';coin.alt='';const label=document.createElement('span');label.textContent=paymentsLabel();o.button.append(img,coin,label)}if(o.button){refresh();o.button.addEventListener('click',()=>open());root.addEventListener?.('languagechange',refresh)}if(mobile())Promise.resolve().then(resume);return{open,receive,encrypted,pay,card,cashCard,refresh,resume,mountWallet,savedAddresses:savedList,addSaved,fundCash,sentCashlinks:sentList,usdt};
}
root.PhoenixNimiqPayments={create,validate,validateCash,addressValid,checkoutRequest,cashCheckout,cardColors};
})(typeof window!=='undefined'?window:globalThis);
