import { mountRelays } from './admin-relay.js?v=0.4.39';
// Wire access management. The server authorizes every action. Registered addresses are never listed.
const words = {
  "fr": {
    "maintenance": "Mode maintenance",
    "maintenanceOn": "Le site est fermé : seuls les administrateurs peuvent se connecter. Les cartes déjà envoyées restent ouvertes.",
    "maintenanceOff": "Le site est ouvert (hors maintenance).",
    "confirmMaintenance": "Activer la maintenance ? Plus personne sauf les administrateurs ne pourra se connecter ni créer de carte.",
    "title": "Administration",
    "invitationOnly": "Accès sur invitation seulement",
    "open": "Tout le monde peut s'inscrire et envoyer.",
    "closed": "Seuls les expéditeurs inscrits ou autorisés peuvent envoyer. Les cartes déjà envoyées restent ouvertes.",
    "allowed": "Invités",
    "blocked": "Bloqués",
    "allow": "Inviter",
    "block": "Bloquer",
    "remove": "Retirer",
    "email": "Inviter une adresse",
    "empty": "Aucune adresse.",
    "confirmOpen": "Ouvrir Phoenix Wire à tout le monde ?",
    "confirmBlock": "Bloquer cette adresse : {email} ?",
    "admin_required": "L'administration est réservée aux administrateurs connectés.",
    "cannot_block_admin": "Cette adresse est celle d’un administrateur et ne peut pas être bloquée.",
    "list_full": "La liste est pleine. Retire une adresse avant de réessayer.",
    "invalid_email": "Vérifie cette adresse courriel.",
    "failed": "Impossible pour le moment. Réessaie.",
    "busy": "Mise à jour…",
    "already": "J’ai déjà un accès",
    "lockedOpen": "La configuration du serveur garde l'accès ouvert : cette case ne prend effet qu'une fois cette configuration retirée.",
    "revoke": "Révoquer",
    "blockEmail": "Bloquer une adresse",
    "search": "Rechercher une adresse",
    "noResults": "Aucune adresse correspondante.",
    "invitedCount": "{count} invités",
    "blockedCount": "{count} bloqués",
    "registeredCount": "{count} inscrits",
    "registeredTotal": "{count} expéditeurs inscrits",
    "confirmRemove": "Retirer cette adresse des invités et des bloqués : {email} ?",
    "actionHelp": "Révoquer enlève l’invitation ; Bloquer coupe l’accès, même pour quelqu’un qui a déjà utilisé Wire.",
    "refundHelp": "Après révocation, une personne qui a déjà envoyé des cartes garde l’accès à ses remboursements.",
    "invitedCountOne": "{count} invité",
    "blockedCountOne": "{count} bloqué",
    "registeredCountOne": "{count} inscrit",
    "registeredTotalOne": "{count} expéditeur inscrit"
  },
  "en": {
    "maintenance": "Maintenance mode",
    "maintenanceOn": "The site is closed: only administrators can sign in. Cards already sent stay open.",
    "maintenanceOff": "The site is open (not in maintenance).",
    "confirmMaintenance": "Turn maintenance on? Nobody but administrators will be able to sign in or create a card.",
    "title": "Administration",
    "invitationOnly": "Invitation-only access",
    "open": "Anyone can sign up and send.",
    "closed": "Only registered or allowed senders can send. Cards already sent remain accessible.",
    "allowed": "Invited",
    "blocked": "Blocked",
    "allow": "Invite",
    "block": "Block",
    "remove": "Remove",
    "email": "Invite an email address",
    "empty": "No addresses.",
    "confirmOpen": "Open Phoenix Wire to everyone?",
    "confirmBlock": "Block this address: {email}?",
    "admin_required": "Only signed-in administrators can manage access.",
    "cannot_block_admin": "This address belongs to an administrator and cannot be blocked.",
    "list_full": "The list is full. Remove an address before trying again.",
    "invalid_email": "Please check this email address.",
    "failed": "Unavailable right now. Please try again.",
    "busy": "Updating…",
    "already": "I already have access",
    "lockedOpen": "The server configuration keeps access open: this checkbox only takes effect once that configuration is removed.",
    "revoke": "Revoke",
    "blockEmail": "Block an email address",
    "search": "Search email addresses",
    "noResults": "No matching addresses.",
    "invitedCount": "{count} invited",
    "blockedCount": "{count} blocked",
    "registeredCount": "{count} registered",
    "registeredTotal": "{count} registered senders",
    "confirmRemove": "Remove this address from the invited and blocked lists: {email}?",
    "actionHelp": "Revoke removes the invitation; Block cuts off access, even for someone who has already used Wire.",
    "refundHelp": "After revocation, someone who has already sent cards keeps access to their refunds.",
    "invitedCountOne": "{count} invited",
    "blockedCountOne": "{count} blocked",
    "registeredCountOne": "{count} registered",
    "registeredTotalOne": "{count} registered sender"
  },
  "es": {
    "maintenance": "Modo mantenimiento",
    "maintenanceOn": "El sitio está cerrado: solo los administradores pueden iniciar sesión. Las tarjetas ya enviadas siguen abiertas.",
    "maintenanceOff": "El sitio está abierto (sin mantenimiento).",
    "confirmMaintenance": "¿Activar el mantenimiento? Nadie salvo los administradores podrá iniciar sesión ni crear tarjetas.",
    "title": "Administración",
    "invitationOnly": "Acceso solo por invitación",
    "open": "Todo el mundo puede registrarse y enviar.",
    "closed": "Solo pueden enviar los remitentes registrados o autorizados. Las tarjetas ya enviadas siguen accesibles.",
    "allowed": "Invitados",
    "blocked": "Bloqueados",
    "allow": "Invitar",
    "block": "Bloquear",
    "remove": "Retirar",
    "email": "Invitar una dirección de correo",
    "empty": "No hay direcciones.",
    "confirmOpen": "¿Abrir Phoenix Wire a todo el mundo?",
    "confirmBlock": "¿Bloquear esta dirección: {email}?",
    "admin_required": "Solo los administradores conectados pueden gestionar el acceso.",
    "cannot_block_admin": "Esta dirección pertenece a un administrador y no se puede bloquear.",
    "list_full": "La lista está llena. Retira una dirección antes de intentarlo de nuevo.",
    "invalid_email": "Comprueba esta dirección de correo.",
    "failed": "No está disponible ahora. Inténtalo de nuevo.",
    "busy": "Actualizando…",
    "already": "Ya tengo acceso",
    "lockedOpen": "La configuración del servidor mantiene el acceso abierto: esta casilla solo tendrá efecto cuando se retire esa configuración.",
    "revoke": "Revocar",
    "blockEmail": "Bloquear una dirección de correo",
    "search": "Buscar direcciones de correo",
    "noResults": "No hay direcciones coincidentes.",
    "invitedCount": "{count} invitados",
    "blockedCount": "{count} bloqueados",
    "registeredCount": "{count} registrados",
    "registeredTotal": "{count} remitentes registrados",
    "confirmRemove": "¿Retirar esta dirección de las listas de invitados y bloqueados: {email}?",
    "actionHelp": "Revocar elimina la invitación; Bloquear corta el acceso, incluso a quien ya haya usado Wire.",
    "refundHelp": "Tras revocar la invitación, quien ya haya enviado tarjetas conserva el acceso a sus devoluciones.",
    "invitedCountOne": "{count} invitado",
    "blockedCountOne": "{count} bloqueado",
    "registeredCountOne": "{count} registrado",
    "registeredTotalOne": "{count} remitente registrado"
  },
  "de": {
    "maintenance": "Wartungsmodus",
    "maintenanceOn": "Die Seite ist geschlossen: Nur Administratoren können sich anmelden. Bereits gesendete Karten bleiben offen.",
    "maintenanceOff": "Die Seite ist offen (keine Wartung).",
    "confirmMaintenance": "Wartung einschalten? Nur Administratoren können sich dann anmelden oder Karten erstellen.",
    "title": "Administration",
    "invitationOnly": "Zugang nur auf Einladung",
    "open": "Alle können sich registrieren und senden.",
    "closed": "Nur registrierte oder zugelassene Absender können senden. Bereits gesendete Karten bleiben zugänglich.",
    "allowed": "Eingeladene",
    "blocked": "Gesperrte",
    "allow": "Einladen",
    "block": "Sperren",
    "remove": "Entfernen",
    "email": "Eine E-Mail-Adresse einladen",
    "empty": "Keine Adressen.",
    "confirmOpen": "Phoenix Wire für alle öffnen?",
    "confirmBlock": "Diese Adresse sperren: {email}?",
    "admin_required": "Nur angemeldete Administratoren können den Zugang verwalten.",
    "cannot_block_admin": "Diese Adresse gehört einem Administrator und kann nicht gesperrt werden.",
    "list_full": "Die Liste ist voll. Entferne zuerst eine Adresse.",
    "invalid_email": "Prüfe diese E-Mail-Adresse.",
    "failed": "Derzeit nicht verfügbar. Versuche es erneut.",
    "busy": "Wird aktualisiert…",
    "already": "Ich habe bereits Zugang",
    "lockedOpen": "Die Serverkonfiguration hält den Zugang offen: Dieses Kontrollkästchen wirkt erst, wenn diese Konfiguration entfernt wurde.",
    "revoke": "Widerrufen",
    "blockEmail": "Eine E-Mail-Adresse sperren",
    "search": "E-Mail-Adressen suchen",
    "noResults": "Keine passenden Adressen.",
    "invitedCount": "{count} eingeladen",
    "blockedCount": "{count} gesperrt",
    "registeredCount": "{count} registriert",
    "registeredTotal": "{count} registrierte Absender",
    "confirmRemove": "Diese Adresse aus den Einladungs- und Sperrlisten entfernen: {email}?",
    "actionHelp": "Widerrufen entfernt die Einladung; Sperren beendet den Zugang auch für Personen, die Wire bereits genutzt haben.",
    "refundHelp": "Nach dem Widerruf behalten Personen, die bereits Karten gesendet haben, Zugang zu ihren Rückzahlungen.",
    "invitedCountOne": "{count} eingeladen",
    "blockedCountOne": "{count} gesperrt",
    "registeredCountOne": "{count} registriert",
    "registeredTotalOne": "{count} registrierter Absender"
  },
  "zh": {
    "maintenance": "维护模式",
    "maintenanceOn": "网站已关闭：只有管理员可以登录。已发送的卡片仍可打开。",
    "maintenanceOff": "网站已开放（未维护）。",
    "confirmMaintenance": "开启维护？除管理员外，任何人都无法登录或创建卡片。",
    "title": "管理",
    "invitationOnly": "仅限受邀用户访问",
    "open": "任何人都可以注册并发送。",
    "closed": "只有已注册或已获授权的发送者可以发送。已发送的卡片仍可打开。",
    "allowed": "受邀邮箱",
    "blocked": "已屏蔽邮箱",
    "allow": "邀请",
    "block": "屏蔽",
    "remove": "移除",
    "email": "邀请一个邮箱",
    "empty": "暂无邮箱。",
    "confirmOpen": "向所有人开放 Phoenix Wire 吗？",
    "confirmBlock": "屏蔽此邮箱：{email}？",
    "admin_required": "只有已登录的管理员才能管理访问权限。",
    "cannot_block_admin": "此邮箱属于管理员，无法屏蔽。",
    "list_full": "列表已满，请先移除一个邮箱再重试。",
    "invalid_email": "请检查此邮箱地址。",
    "failed": "暂时无法操作，请重试。",
    "busy": "正在更新…",
    "already": "我已有访问权限",
    "lockedOpen": "服务器配置保持访问开放：只有移除该配置后，此复选框的设置才会生效。",
    "revoke": "撤销邀请",
    "blockEmail": "屏蔽一个邮箱",
    "search": "搜索邮箱地址",
    "noResults": "没有匹配的邮箱。",
    "invitedCount": "{count} 个受邀邮箱",
    "blockedCount": "{count} 个已屏蔽邮箱",
    "registeredCount": "{count} 个已注册发送者",
    "registeredTotal": "{count} 个已注册发送者",
    "confirmRemove": "从受邀和屏蔽列表中移除此邮箱：{email}？",
    "actionHelp": "撤销邀请会移除邀请；屏蔽会切断访问权限，即使对方以前用过 Wire。",
    "refundHelp": "撤销邀请后，曾发送过卡片的用户仍可访问自己的退款。",
    "invitedCountOne": "{count} 个受邀邮箱",
    "blockedCountOne": "{count} 个已屏蔽邮箱",
    "registeredCountOne": "{count} 个已注册发送者",
    "registeredTotalOne": "{count} 个已注册发送者"
  }
};
export function adminText(language,key){
 const copy=Object.hasOwn(words,language)?words[language]:words.en;
 return Object.hasOwn(copy,key)?copy[key]:words.en.failed;
}
export function mountAdmin({root=document.body,language=()=>document.documentElement.lang,request,authorized=false,onAccess=()=>{}}){
 if(authorized!==true)return null;
 const document=root.ownerDocument||root,window=document.defaultView;
 const panel=document.createElement('section');panel.id='wire-admin';panel.className='wire-card panel wire-admin-panel';root.append(panel);
 let view=null,busy=false,notice='',closed=false,epoch=0,query='';
 const relays=mountRelays({document,language,request,confirm:text=>window.confirm(text)});
 const drafts={allow:'',block:''};
 const t=key=>adminText(language(),key);
 const node=(tag,text)=>{const el=document.createElement(tag);if(text!==undefined)el.textContent=text;return el;};
 const count=(key,value)=>t(value===1?key+'One':key).replace('{count}',String(value??'—'));
 const isAdmin=email=>view.admins.some(address=>address.toLowerCase()===email.toLowerCase());
 function addressButton(actionName,address){
  const button=node('button',t(actionName));button.type='button';button.className='secondary-btn';button.dataset.action=actionName;
  button.setAttribute('aria-label',t(actionName)+' '+address);button.setAttribute('aria-describedby','wire-admin-action-help');
  button.disabled=busy||(actionName==='block'&&isAdmin(address));
  button.addEventListener('click',()=>{
   if(busy||closed)return;
   if((actionName==='block'||actionName==='remove')&&!window.confirm(t(actionName==='block'?'confirmBlock':'confirmRemove').replace('{email}',address)))return;
   void action('/api/admin/'+actionName,{email:address});
  });
  return button;
 }
 function renderLists(){
  const needle=query.trim().toLowerCase();
  for(const kind of ['allowed','blocked']){
   const section=panel.querySelector('[data-list="'+kind+'"]');if(!section)continue;
   const list=section.querySelector('ul'),empty=section.querySelector('.wire-admin-empty');list.replaceChildren();
   for(const address of view[kind]){
    if(!address.toLowerCase().includes(needle))continue;
    const row=node('li'),email=node('span',address),buttons=node('div');
    email.className='wire-admin-address';buttons.className='wire-admin-row-actions';
    for(const name of kind==='allowed'?['revoke','block']:['remove','allow'])buttons.append(addressButton(name,address));
    row.append(email,buttons);list.append(row);
   }
   empty.textContent=t(needle?'noResults':'empty');empty.hidden=list.childElementCount>0;
   list.hidden=!list.childElementCount;
  }
 }
 function addressForm(kind){
  const form=node('form'),label=node('label',t(kind==='allow'?'email':'blockEmail')),input=node('input'),submit=node('button',t(kind));
  form.id='wire-admin-'+kind;input.id=form.id+'-email';label.htmlFor=input.id;
  input.type='email';input.autocomplete='off';input.spellcheck=false;input.required=true;input.maxLength=254;input.disabled=busy;input.value=drafts[kind];
  input.addEventListener('input',()=>{drafts[kind]=input.value;});
  submit.type='submit';submit.className='primary-btn';submit.disabled=busy;label.append(input);form.append(label,submit);
  form.addEventListener('submit',event=>{
   event.preventDefault();if(busy||closed||!input.reportValidity())return;
   drafts[kind]=input.value;const email=input.value.trim();
   if(kind==='block'&&!window.confirm(t('confirmBlock').replace('{email}',email)))return;
   void action('/api/admin/'+kind,{email},kind);
  });
  return form;
 }
 function render(){
  if(closed)return;
  panel.replaceChildren(node('h2',t('title')));panel.setAttribute('aria-busy',String(busy));
  const status=node('p',busy?t('busy'):notice?t(notice):'');status.id='wire-admin-status';status.setAttribute('role','status');
  if(view){
   const locked=view.locked_open===true;
   const label=node('label',t('invitationOnly')),toggle=node('input');toggle.type='checkbox';toggle.id='wire-admin-open';toggle.setAttribute('role','switch');toggle.checked=view.open===false;toggle.setAttribute('aria-checked',String(toggle.checked));toggle.disabled=busy||locked;
   toggle.addEventListener('change',()=>{if(busy||view.locked_open===true){render();return;}const open=!toggle.checked;if(open&&!window.confirm(t('confirmOpen'))){render();return;}void action('/api/admin/access',{open});});
   label.className='wire-admin-switch';label.htmlFor=toggle.id;label.prepend(toggle);
   const accessState=node('p',t(view.open?'open':'closed'));accessState.id='wire-admin-access-state';accessState.setAttribute('role','status');toggle.setAttribute('aria-describedby',accessState.id);panel.append(label,accessState);
   if(locked){const explanation=node('p',t('lockedOpen'));explanation.id='wire-admin-locked-open';explanation.setAttribute('role','status');panel.append(explanation);toggle.setAttribute('aria-describedby',accessState.id+' '+explanation.id);}
   const maintenanceLabel=node('label',t('maintenance')),maintenanceToggle=node('input');maintenanceToggle.type='checkbox';maintenanceToggle.id='wire-admin-maintenance';maintenanceToggle.setAttribute('role','switch');maintenanceToggle.checked=view.maintenance===true;maintenanceToggle.setAttribute('aria-checked',String(maintenanceToggle.checked));maintenanceToggle.disabled=busy;
   maintenanceToggle.addEventListener('change',()=>{if(busy)return;const on=maintenanceToggle.checked;if(on&&!window.confirm(t('confirmMaintenance'))){render();return;}void action('/api/admin/maintenance',{on});});
   maintenanceLabel.className='wire-admin-maintenance-control';maintenanceLabel.htmlFor=maintenanceToggle.id;maintenanceLabel.prepend(maintenanceToggle);const maintenanceState=node('p',t(view.maintenance?'maintenanceOn':'maintenanceOff'));maintenanceState.id='wire-admin-maintenance-state';maintenanceState.setAttribute('role','status');maintenanceToggle.setAttribute('aria-describedby',maintenanceState.id);panel.append(maintenanceLabel,maintenanceState);
   const summary=node('p',[count('invitedCount',view.allowed.length),count('blockedCount',view.blocked.length),count('registeredCount',view.registered_count)].join(' · '));
   summary.id='wire-admin-counts';summary.setAttribute('role','status');panel.append(summary);
   summary.setAttribute('aria-label',[count('invitedCount',view.allowed.length),count('blockedCount',view.blocked.length),count('registeredTotal',view.registered_count)].join(' · '));
   const searchLabel=node('label',t('search')),search=node('input');search.id='wire-admin-search';search.type='search';search.autocomplete='off';search.spellcheck=false;search.value=query;search.disabled=busy;
   search.setAttribute('aria-controls','wire-admin-allowed-list wire-admin-blocked-list');searchLabel.htmlFor=search.id;searchLabel.append(search);panel.append(searchLabel);
   search.addEventListener('input',()=>{query=search.value;renderLists();});
   for(const kind of ['allowed','blocked']){
    const section=node('section');section.className='wire-admin-list';section.dataset.list=kind;
    const heading=node('h3',t(kind));heading.id='wire-admin-'+kind+'-title';section.append(heading);
    const list=node('ul');list.id='wire-admin-'+kind+'-list';list.tabIndex=0;list.setAttribute('aria-labelledby',heading.id);list.setAttribute('role','list');
    const empty=node('p');empty.className='wire-admin-empty';empty.setAttribute('role','status');section.append(list,empty,addressForm(kind==='allowed'?'allow':'block'));panel.append(section);
   }
   const help=node('p',t('actionHelp'));help.id='wire-admin-action-help';help.className='wire-admin-help';panel.append(help);
   const refunds=node('p',t('refundHelp'));refunds.className='wire-admin-help';panel.append(refunds);
   renderLists();
   panel.append(relays.node);relays.render();
  }
  panel.append(status);
 }
 function accept(result){
  if(typeof result?.open!=='boolean'||!['allowed','blocked','admins'].every(key=>Array.isArray(result[key])&&result[key].every(value=>typeof value==='string')))throw Error('invalid_response');
  // Never derive a count from, retain, or render the deprecated registered array.
  view={open:result.open,locked_open:result.locked_open===true,allowed:result.allowed,blocked:result.blocked,admins:result.admins,
   registered_count:Number.isSafeInteger(result.registered_count)&&result.registered_count>=0?result.registered_count:null,maintenance:result.maintenance===true};
  onAccess(view.open,view);
 }
 async function action(path,data,formKind){
  if(busy||closed)return;const focusID=panel.contains(document.activeElement)?document.activeElement.id:'';busy=true;notice='';const current=++epoch;render();
  try{const result=await request(path,data);if(!closed&&current===epoch){accept(result);if(formKind)drafts[formKind]='';}}
  catch(error){if(!closed&&current===epoch){const code=error?.code||error?.message;notice=['admin_required','cannot_block_admin','list_full','invalid_email'].includes(code)?code:'failed';if(notice==='admin_required')view=null;}}
  finally{if(!closed&&current===epoch){const restore=focusID&&document.activeElement===document.body;busy=false;render();const target=restore?document.getElementById(focusID):null;if(target&&!target.disabled)target.focus({preventScroll:true});}}
 }
 void action('/api/admin/access');
 return {render,destroy(){closed=true;epoch++;relays.destroy();panel.remove();}};
}
