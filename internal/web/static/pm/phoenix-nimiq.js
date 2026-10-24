/* MIT. Copyright (c) 2026 Xcoder. Wallet keys remain in the Nimiq Keyguard. */
(function(root){
'use strict';
const words={
 fr:['Wallet Nimiq','Connecter le wallet','Authentifier','Authentifié','Le courriel a changé : authentifie à nouveau.','Signature invalide','Signé · wallet inconnu','Signé · wallet reconnu','Wallet différent du contact reconnu','Reconnaître ce wallet','Vérifie cette adresse avec ton contact par un autre moyen avant de la reconnaître.','Wallet mémorisé','Signature en cours…','Connexion annulée ou indisponible','Authentification avec Cci indisponible pour cette version.','Authentification demandée mais non validée. Signe ou désactive Auth.','Authentification Nimiq désactivée pour ce courriel.'],
 en:['Nimiq wallet','Connect wallet','Authenticate','Authenticated','Email changed: authenticate again.','Invalid signature','Signed · unfamiliar wallet','Signed · recognized wallet','Wallet differs from recognized contact','Recognize this wallet','Verify this address with your contact through another channel before recognizing it.','Wallet remembered','Signing…','Connection cancelled or unavailable','Authentication with Bcc is unavailable in this version.','Authentication requested but not validated. Sign or disable Auth.','Nimiq authentication disabled for this email.'],
 es:['Wallet Nimiq','Conectar wallet','Autenticar','Autenticado','El correo cambió: autentica de nuevo.','Firma no válida','Firmado · wallet desconocido','Firmado · wallet reconocido','Wallet diferente del contacto reconocido','Reconocer este wallet','Verifica esta dirección con tu contacto por otro medio antes de reconocerla.','Wallet guardado','Firmando…','Conexión cancelada o no disponible','La autenticación con Cco no está disponible en esta versión.','Autenticación solicitada pero no validada. Firma o desactiva Auth.','Autenticación Nimiq desactivada para este correo.'],
 de:['Nimiq Wallet','Wallet verbinden','Authentifizieren','Authentifiziert','E-Mail geändert: erneut authentifizieren.','Ungültige Signatur','Signiert · unbekanntes Wallet','Signiert · erkanntes Wallet','Wallet weicht vom bekannten Kontakt ab','Dieses Wallet erkennen','Prüfe diese Adresse über einen anderen Kanal mit deinem Kontakt.','Wallet gespeichert','Signatur läuft…','Verbindung abgebrochen oder nicht verfügbar','Authentifizierung mit Bcc ist in dieser Version nicht verfügbar.','Authentifizierung angefordert, aber nicht bestätigt. Signiere oder deaktiviere Auth.','Nimiq-Authentifizierung für diese E-Mail deaktiviert.'],
 zh:['Nimiq 钱包','连接钱包','验证签名','已签名','邮件已更改，请重新签名。','签名无效','已签名 · 未知钱包','已签名 · 已认可钱包','钱包与已认可联系人不同','认可此钱包','认可前，请通过其他渠道与联系人核实此地址。','已记住钱包','正在签名…','连接已取消或不可用','此版本不支持密送邮件签名。','已请求签名验证，但尚未完成。请签名或关闭签名验证。','已为此邮件关闭 Nimiq 签名验证。']
};
const compact={fr:['Signer','Signé'],en:['Sign','Signed'],es:['Firmar','Firmado'],de:['Signieren','Signiert'],zh:['签名','已签名']};
const hex=a=>Array.from(a,b=>b.toString(16).padStart(2,'0')).join('');
const normalize=a=>String(a||'').replace(/\s/g,'').toUpperCase();
function create(o){
 let proof=null,snapshot='',busy=false,required=false,revision=0,signedRevision=-1,lastRender='',walletIcon=null;
 const labels=()=>words[o.language()]||words.en;
 const memoryKey=()=>`phoenixmail.nimiq.wallet.v1.${o.account()}`;
 const pinnedKey=email=>`phoenixmail.nimiq.contact.v1.${String(email).toLowerCase()}`;
 const read=key=>{try{return localStorage.getItem(key)||''}catch{return ''}};
 const write=(key,value)=>{try{localStorage.setItem(key,value)}catch{}};
 const remembered=()=>read(memoryKey());
 const hub=()=>{if(!root.HubApi)throw Error(labels()[13]);return new root.HubApi('https://hub.nimiq.com')};
 async function json(path,data){const r=await fetch(path,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(data)});let d;try{d=await r.json()}catch{throw Error(r.status===404?(o.language()==='fr'?'Fonction Nimiq absente du serveur. Mets PhoenixMail à jour puis redémarre-le.':'Nimiq is missing on the server. Update PhoenixMail and restart it.'):(o.language()==='fr'?'Réponse Nimiq du serveur invalide.':'Invalid Nimiq server response.'))}if(!r.ok)throw Error(d?.error||labels()[5]);if(!d||typeof d!=='object')throw Error(labels()[5]);return d}
 function stamp(){return JSON.stringify(o.draft())}
 function update(){
  const lang=o.language(),l=labels(),wallet=remembered(),canSign=!o.canSign||o.canSign();
  const rendered=[lang,wallet,busy,Boolean(proof),required,canSign].join('\0');if(rendered===lastRender)return;lastRender=rendered;
  if(!walletIcon){walletIcon=document.createElement('img');walletIcon.src='brand/nimiq-logo.svg';walletIcon.alt='';o.walletButton.textContent='';o.walletButton.append(walletIcon,document.createTextNode('Nimiq Auth'))}
  o.walletButton.dataset.associated=String(Boolean(wallet));o.walletButton.ariaLabel=wallet?l[11]:l[1];o.walletButton.title=wallet?l[11]+' · '+wallet:l[1];
  const short=compact[lang]||compact.en;o.authButton.textContent=short[proof?1:0];o.authButton.ariaLabel=proof?l[3]:l[2];o.authButton.disabled=busy||!canSign;o.walletButton.disabled=busy;
  if(o.cancelButton){o.cancelButton.hidden=!required;o.cancelButton.disabled=busy}
 }
 function invalidate(){proof=null;o.onProof(null);o.status(labels()[4],'error');update()}
 function changed(){revision++;if(!proof)return;if(revision!==signedRevision)invalidate()}
 async function connect(){if(busy)return;const key=memoryKey();busy=true;update();try{
  const nonce=crypto.getRandomValues(new Uint8Array(16));const message='PhoenixMail wallet association v1\nNonce: '+hex(nonce);
  const result=await hub().signMessage({appName:'PhoenixMail',message,signer:remembered()||undefined});
  const verification=await json('/api/nimiq/wallet',{wallet:result.signer,publicKey:hex(result.signerPublicKey),signature:hex(result.signature),message});
  if(verification.verified!==true)throw Error(labels()[5]);
  write(key,result.signer);o.status(labels()[11],'idle');
 }catch(e){o.status(e.message||labels()[13],'error')}finally{busy=false;update()}}
 async function authenticate(){if(busy)return;busy=true;required=true;proof=null;o.onProof(null);update();o.status(labels()[12],'idle');try{
  const draft=o.draft(),signingRevision=revision;snapshot=JSON.stringify(draft);
  if(draft.bcc.trim())throw Error(labels()[14]);
  // Hub opens its popup synchronously, then resolves the request argument promise.
  const signer=hub();
  const preparing=json('/api/nimiq/prepare',draft);
  const request=preparing.then(prepared=>({appName:'PhoenixMail',message:prepared.message,signer:remembered()||undefined}));
  // If Hub cannot open, preparation may still reject; keep that rejection handled.
  request.catch(()=>{});
  const signed=await signer.signMessage(request);
  const prepared=await preparing;
  if(revision!==signingRevision||stamp()!==snapshot)throw Error(labels()[4]);
  const candidate={manifest:prepared.manifest,wallet:signed.signer,publicKey:hex(signed.signerPublicKey),signature:hex(signed.signature)};
  const verification=await json('/api/nimiq/prepare',{...draft,proof:candidate});
  if(verification.verified!==true)throw Error(labels()[5]);
  if(revision!==signingRevision||stamp()!==snapshot)throw Error(labels()[4]);
  proof=candidate;signedRevision=revision;write(memoryKey(),candidate.wallet);o.onProof(proof);o.status(labels()[3],'success');
 }catch(e){proof=null;o.onProof(null);o.status(e.message||labels()[13],'error')}finally{busy=false;update()}}
 async function preflight(){if(proof&&stamp()!==snapshot)invalidate();if(busy)throw Error(labels()[12]);if(required&&!proof)throw Error(labels()[15]);if(proof)await json('/api/nimiq/prepare',{...o.draft(),proof})}
 function reset(){proof=null;required=false;snapshot='';o.onProof(null);update()}
 function receive(host,email,status){if(!status)return;const l=labels();const box=document.createElement('div');box.className='nimiq-verification';const image=document.createElement('img');image.src='brand/nimiq-logo.svg';image.alt='Nimiq';box.append(image);const pin=read(pinnedKey(email));const valid=status.state==='valid';box.dataset.state=valid?'valid':'invalid';const text=!valid?l[5]:!pin?l[6]:normalize(pin)===normalize(status.wallet)?l[7]:l[8];const span=document.createElement('span');span.textContent=text;box.append(span);box.title=status.wallet||'';
  if(valid&&normalize(pin)!==normalize(status.wallet)){const button=document.createElement('button');button.type='button';button.textContent=l[9];button.onclick=()=>{if(confirm(l[10]+'\n\n'+status.wallet)){write(pinnedKey(email),status.wallet);box.remove();receive(host,email,status)}};box.append(button)}host.append(box)
 }
 if(o.cancelButton)o.cancelButton.addEventListener('click',()=>{if(busy)return;reset();o.status(labels()[16],'idle')});o.walletButton.addEventListener('click',connect);o.authButton.addEventListener('click',authenticate);update();
 return {connect,authenticate,preflight,reset,changed,receive,refresh:update};
}
root.PhoenixNimiq={create};
})(window);
