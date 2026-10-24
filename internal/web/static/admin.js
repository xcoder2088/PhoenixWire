// Wire access management. The server remains the authority for every action, including locked-open configuration.
const words={
 fr:{title:'Administration',invitationOnly:"Accès sur invitation seulement",open:"Tout le monde peut s'inscrire et envoyer.",closed:"Seuls les expéditeurs inscrits ou autorisés peuvent envoyer. Les cartes déjà envoyées restent ouvertes.",registered:'Expéditeurs inscrits',allowed:'Adresses autorisées',blocked:'Adresses bloquées',allow:'Autoriser',block:'Bloquer',remove:'Retirer',email:'Autoriser une adresse',empty:'Aucune adresse.',confirmOpen:"Ouvrir Phoenix Wire à tout le monde ?",confirmBlock:'Bloquer cette adresse : {email} ?',admin_required:"L'administration est réservée aux administrateurs connectés.",cannot_block_admin:'Cette adresse est celle d’un administrateur et ne peut pas être bloquée.',list_full:'La liste est pleine. Retire une adresse avant de réessayer.',invalid_email:'Vérifie cette adresse courriel.',failed:'Impossible pour le moment. Réessaie.',busy:'Mise à jour…',already:'J’ai déjà un accès'},
 en:{title:'Administration',invitationOnly:"Invitation-only access",open:"Anyone can sign up and send.",closed:"Only registered or allowed senders can send. Cards already sent remain accessible.",registered:'Registered senders',allowed:'Allowed addresses',blocked:'Blocked addresses',allow:'Allow',block:'Block',remove:'Remove',email:'Allow an email address',empty:'No addresses.',confirmOpen:"Open Phoenix Wire to everyone?",confirmBlock:'Block this address: {email}?',admin_required:'Only signed-in administrators can manage access.',cannot_block_admin:'This address belongs to an administrator and cannot be blocked.',list_full:'The list is full. Remove an address before trying again.',invalid_email:'Please check this email address.',failed:'Unavailable right now. Please try again.',busy:'Updating…',already:'I already have access'},
 es:{title:'Administración',invitationOnly:"Acceso solo por invitación",open:"Todo el mundo puede registrarse y enviar.",closed:"Solo pueden enviar los remitentes registrados o autorizados. Las tarjetas ya enviadas siguen accesibles.",registered:'Remitentes registrados',allowed:'Direcciones autorizadas',blocked:'Direcciones bloqueadas',allow:'Autorizar',block:'Bloquear',remove:'Retirar',email:'Autorizar una dirección de correo',empty:'No hay direcciones.',confirmOpen:"¿Abrir Phoenix Wire a todo el mundo?",confirmBlock:'¿Bloquear esta dirección: {email}?',admin_required:'Solo los administradores conectados pueden gestionar el acceso.',cannot_block_admin:'Esta dirección pertenece a un administrador y no se puede bloquear.',list_full:'La lista está llena. Retira una dirección antes de intentarlo de nuevo.',invalid_email:'Comprueba esta dirección de correo.',failed:'No está disponible ahora. Inténtalo de nuevo.',busy:'Actualizando…',already:'Ya tengo acceso'},
 de:{title:'Administration',invitationOnly:"Zugang nur auf Einladung",open:"Alle können sich registrieren und senden.",closed:"Nur registrierte oder zugelassene Absender können senden. Bereits gesendete Karten bleiben zugänglich.",registered:'Registrierte Absender',allowed:'Erlaubte Adressen',blocked:'Gesperrte Adressen',allow:'Erlauben',block:'Sperren',remove:'Entfernen',email:'E-Mail-Adresse erlauben',empty:'Keine Adressen.',confirmOpen:"Phoenix Wire für alle öffnen?",confirmBlock:'Diese Adresse sperren: {email}?',admin_required:'Nur angemeldete Administratoren können den Zugang verwalten.',cannot_block_admin:'Diese Adresse gehört einem Administrator und kann nicht gesperrt werden.',list_full:'Die Liste ist voll. Entferne zuerst eine Adresse.',invalid_email:'Prüfe diese E-Mail-Adresse.',failed:'Derzeit nicht verfügbar. Versuche es erneut.',busy:'Wird aktualisiert…',already:'Ich habe bereits Zugang'},
 zh:{title:'管理',invitationOnly:"仅限受邀用户访问",open:"任何人都可以注册并发送。",closed:"只有已注册或已获授权的发送者可以发送。已发送的卡片仍可打开。",registered:'已注册发送者',allowed:'已授权邮箱',blocked:'已屏蔽邮箱',allow:'允许',block:'屏蔽',remove:'移除',email:'授权一个邮箱',empty:'暂无邮箱。',confirmOpen:"向所有人开放 Phoenix Wire 吗？",confirmBlock:'屏蔽此邮箱：{email}？',admin_required:'只有已登录的管理员才能管理访问权限。',cannot_block_admin:'此邮箱属于管理员，无法屏蔽。',list_full:'列表已满，请先移除一个邮箱再重试。',invalid_email:'请检查此邮箱地址。',failed:'暂时无法操作，请重试。',busy:'正在更新…',already:'我已有访问权限'}
};
const lockedWords={fr:"La configuration du serveur garde l'accès ouvert : cette case ne prend effet qu'une fois cette configuration retirée.",en:'The server configuration keeps access open: this checkbox only takes effect once that configuration is removed.',es:'La configuración del servidor mantiene el acceso abierto: esta casilla solo tendrá efecto cuando se retire esa configuración.',de:'Die Serverkonfiguration hält den Zugang offen: Dieses Kontrollkästchen wirkt erst, wenn diese Konfiguration entfernt wurde.',zh:'服务器配置保持访问开放：只有移除该配置后，此复选框的设置才会生效。'};
for(const language of Object.keys(lockedWords))words[language].lockedOpen=lockedWords[language];
export function adminText(language,key){return (words[language]||words.en)[key]||words.en.failed;}
export function mountAdmin({root=document.body,language=()=>document.documentElement.lang,request,authorized=false,onAccess=()=>{}}){
 if(authorized!==true)return null;
 const document=root.ownerDocument||root,window=document.defaultView;
 const panel=document.createElement('section');panel.id='wire-admin';panel.className='wire-card panel wire-admin-panel';root.append(panel);
 let view=null,busy=false,notice='',closed=false,epoch=0;
 const t=key=>adminText(language(),key),node=(tag,text)=>{const el=document.createElement(tag);if(text)el.textContent=text;return el;};
 function render(){
  if(closed)return;panel.replaceChildren(node('h2',t('title')));
  const status=node('p',busy?t('busy'):notice?t(notice):'');status.id='wire-admin-status';status.setAttribute('role','status');
  if(view){
   const locked=view.locked_open===true;
   const label=node('label',t('invitationOnly')),toggle=node('input');toggle.type='checkbox';toggle.id='wire-admin-open';toggle.setAttribute('role','switch');toggle.checked=view.open===false;toggle.setAttribute('aria-checked',String(toggle.checked));toggle.disabled=busy||locked;
   toggle.addEventListener('change',()=>{if(busy||view.locked_open===true){render();return;}const open=!toggle.checked;if(open&&!window.confirm(t('confirmOpen'))){render();return;}void action('/api/admin/access',{open});});label.prepend(toggle);const accessState=node('p',t(view.open?'open':'closed'));accessState.id='wire-admin-access-state';accessState.setAttribute('role','status');toggle.setAttribute('aria-describedby',accessState.id);panel.append(label,accessState);
   if(locked){const explanation=node('p',t('lockedOpen'));explanation.id='wire-admin-locked-open';explanation.setAttribute('role','status');panel.append(explanation);toggle.setAttribute('aria-describedby',accessState.id+' '+explanation.id);}
   for(const kind of ['registered','allowed','blocked']){
    const section=node('section');section.className='wire-admin-list';section.dataset.list=kind;section.append(node('h3',t(kind)));const list=node('ul');
    for(const address of view[kind]){const row=node('li'),email=node('span',address),button=node('button',t(kind==='blocked'?'allow':kind==='allowed'?'remove':'block'));button.type='button';button.className='secondary-btn';button.disabled=busy||(kind!=='blocked'&&view.admins.some(a=>a.toLowerCase()===address.toLowerCase()));
     button.addEventListener('click',()=>{if(kind!=='blocked'&&!window.confirm(t('confirmBlock').replace('{email}',address)))return;void action('/api/admin/'+(kind==='blocked'?'allow':'block'),{email:address});});row.append(email,button);list.append(row);}
    if(!list.childElementCount)section.append(node('p',t('empty')));else section.append(list);panel.append(section);
   }
   const form=node('form'),labelEmail=node('label',t('email')),input=node('input'),allow=node('button',t('allow'));form.id='wire-admin-allow';input.type='email';input.autocomplete='off';input.required=true;input.maxLength=254;input.disabled=busy;allow.type='submit';allow.className='primary-btn';allow.disabled=busy;labelEmail.append(input);form.append(labelEmail,allow);
   form.addEventListener('submit',event=>{event.preventDefault();if(input.reportValidity())void action('/api/admin/allow',{email:input.value.trim()});});panel.append(form);
  }panel.append(status);
 }
 function accept(result){if(typeof result?.open!=='boolean'||!['allowed','blocked','registered','admins'].every(k=>Array.isArray(result[k])&&result[k].every(v=>typeof v==='string')))throw Error('invalid_response');view=result;onAccess(result.open);}
 async function action(path,data){
  if(busy||closed)return;busy=true;notice='';const current=++epoch;render();
  try{const result=await request(path,data);if(!closed&&current===epoch)accept(result);}
  catch(error){if(!closed&&current===epoch){notice=['admin_required','cannot_block_admin','list_full','invalid_email'].includes(error.code||error.message)?error.code||error.message:'failed';if(notice==='admin_required')view=null;}}
  finally{if(!closed&&current===epoch){busy=false;render();}}
 }
 void action('/api/admin/access');
 return {render,destroy(){closed=true;epoch++;panel.remove();}};
}

