import {cardWords,cardPalettes} from './card-mirror.js?v=0.3.0';
export const supportedLanguages = Object.freeze(['fr', 'en', 'es', 'de', 'zh']);

export function cardPresentation(language,id='00') {
 const words=cardWords[language]||cardWords.en;
 const first=/^[a-f0-9]{2}/i.test(id)?parseInt(id.slice(0,2),16):0;
 const [background,border]=cardPalettes[first%cardPalettes.length];
 return {heading:words[0],action:words[1],locked:words[2],protected:words[3],background,border};
}

export function paintCard(element,language,id='00') {
 const p=cardPresentation(language,id);
 element.style.setProperty('--card-bg',p.background);
 element.style.setProperty('--card-border',p.border);
 return p;
}

// Match PhoenixMail's locale precedence, including regional tags and English fallback.
export function resolveLanguage(preference, languages) {
  if (supportedLanguages.includes(preference)) return preference;
  for (const locale of (Array.isArray(languages) ? languages : [languages])) {
    const language = String(locale || '').toLowerCase().split(/[-_]/)[0];
    if (supportedLanguages.includes(language)) return language;
  }
  return 'en';
}

export const assets = Object.freeze({
  NIM: { unit: 'NIM', network: 'Nimiq', symbol: 'N', decimals: 5 },
  USDT_ETHEREUM: { unit: 'USDT', network: 'Ethereum', symbol: '₮', decimals: 6 },
  USDC_POLYGON: { unit: 'USDC', network: 'Polygon', symbol: '$', decimals: 6 },
  USDT_POLYGON: { unit: 'USDT', network: 'Polygon', symbol: '₮', decimals: 6 }
});

export function addressValid(address) {
  const compact=String(address||'').replace(/ /g,'').toUpperCase();
  if(!/^NQ\d{2}[0-9ABCDEFGHJKLMNPQRSTUVXY]{32}$/.test(compact))return false;
  let rest=0;
  for(const c of compact.slice(4)+compact.slice(0,4)){
    const digits=c>='A'?String(c.charCodeAt(0)-55):c;
    for(const digit of digits)rest=(rest*10+Number(digit))%97;
  }
  return rest===1;
}

// Preview formatting only. Real transfers must use server-validated integer base units.
export function previewAmount(value, asset) {
  if (!Object.hasOwn(assets, asset)) throw new Error('invalid_asset');
  const currency = assets[asset];
  if (!currency) throw new Error('invalid_asset');
  const amount = String(value).trim().replace(',', '.');
  if (!new RegExp(`^\\d{1,12}(?:\\.\\d{1,${currency.decimals}})?$`).test(amount) || !/[1-9]/.test(amount)) {
    throw new Error('invalid_amount');
  }
  return amount.replace(/^0+(?=\d)/, '');
}


// Shared sender-history policy. Creation dates come from the server, never polling time.
export function nimFunded(p) {
 return ['funded','delivery_pending','delivery_failed','sent'].includes(p.stage) ||
  ['cancelled','expired'].includes(p.stage) && p.claim_state==='unclaimed';
}
export function unpaidCard(p,now=Date.now()) {
 const created=Date.parse(p.created||'');
 return ['ready','funding_started','preparing'].includes(p.stage) && !p.claim_state &&
  Number.isFinite(created) && created<=now && now<created+86400000;
}
export function recoverableCard(p) { return !['refunding','refunded'].includes(p.stage)&&Boolean(p.address)&&nimFunded(p)&&p.claim_state==='unclaimed'; }
export function historyEntries(entries,now=Date.now()) {
 const created=p=>Number(p.createdAt)|| (typeof p.created==='number'?p.created:Date.parse(p.created||'')) || 0;
 const alive=entries.filter(p=>!(p.stage==='cancelled'||p.state==='cancelled') || p.nim!==false&&recoverableCard(p));
 const recent=alive.filter(p=>created(p)>0&&created(p)<=now&&(now<created(p)+86400000||['refunding','refunded'].includes(p.stage))).sort((a,b)=>created(b)-created(a)).slice(0,3);
 const older=alive.filter(p=>created(p)>0&&now>=created(p)+86400000&&(p.nim===false?p.state==='expired'&&Number(p.balanceUnits)>0:recoverableCard(p)));
 return [...recent,...older].sort((a,b)=>created(b)-created(a));
}
export function historyPending(p,now=Date.now()) {
 if(p.stage==='refunding')return true;
 if(p.stage==='refunded')return false;
 const created=Date.parse(p.created||'');
 return Number.isFinite(created) && now<created+86400000 && p.claim_state!=='claimed';
}
export const senderHistoryWords={
 fr:{refunding:"Remboursement en cours…",refunded:"Remboursée automatiquement",received:'Reçu',claiming:'Envoi en cours…',unclaimed:'Envoyée, pas encore reçue',unpaid:'Paiement non reçu',checking:'Vérification en cours…',cancel:'Annuler la carte',confirm:'Annuler cette carte non payée ?',arrived:'Le paiement est bien arrivé',retry:'Réessayer le paiement',recover:'Récupérer les fonds',open:'Ouvrir la récupération',failed:'Impossible pour le moment, réessayez.',cannot:'Cette carte ne peut plus être annulée.',details:'Détails techniques'},
 en:{refunding:"Refund in progress…",refunded:"Automatically refunded",received:'Received',claiming:'Sending…',unclaimed:'Sent, not yet received',unpaid:'Payment not received',checking:'Checking…',cancel:'Cancel card',confirm:'Cancel this unpaid card?',arrived:'The payment has arrived',retry:'Retry payment',recover:'Recover funds',open:'Open recovery',failed:'Unavailable right now, please try again.',cannot:'This card can no longer be cancelled.',details:'Technical details'},
 es:{refunding:"Reembolso en curso…",refunded:"Reembolsada automáticamente",received:'Recibida',claiming:'Enviando…',unclaimed:'Enviada, aún no recibida',unpaid:'Pago no recibido',checking:'Comprobando…',cancel:'Cancelar la tarjeta',confirm:'¿Cancelar esta tarjeta sin pagar?',arrived:'El pago ha llegado',retry:'Reintentar el pago',recover:'Recuperar fondos',open:'Abrir recuperación',failed:'No disponible ahora, inténtalo de nuevo.',cannot:'Esta tarjeta ya no se puede cancelar.',details:'Detalles técnicos'},
 de:{refunding:"Rückerstattung läuft…",refunded:"Automatisch zurückerstattet",received:'Empfangen',claiming:'Wird gesendet…',unclaimed:'Gesendet, noch nicht empfangen',unpaid:'Zahlung nicht eingegangen',checking:'Wird geprüft…',cancel:'Karte stornieren',confirm:'Diese unbezahlte Karte stornieren?',arrived:'Die Zahlung ist eingegangen',retry:'Zahlung erneut versuchen',recover:'Guthaben zurückholen',open:'Rückholung öffnen',failed:'Derzeit nicht verfügbar, bitte erneut versuchen.',cannot:'Diese Karte kann nicht mehr storniert werden.',details:'Technische Details'},
 zh:{refunding:"正在退款…",refunded:"已自动退款",received:'已收到',claiming:'正在发送…',unclaimed:'已发送，尚未收到',unpaid:'尚未收到付款',checking:'正在检查…',cancel:'取消卡片',confirm:'取消这张尚未付款的卡片？',arrived:'付款已到账',retry:'重试付款',recover:'取回资金',open:'打开资金恢复',failed:'暂时无法使用，请重试。',cannot:'此卡片已无法取消。',details:'技术详情'}
};
export function senderHistoryText(language,key) {return (senderHistoryWords[language]||senderHistoryWords.en)[key];}
export function nimHistoryLabel(p,language,now=Date.now()) {
 const key=['refunding','refunded'].includes(p.stage)?p.stage:p.claim_state==='claimed'?'received':p.claim_state==='claiming'?'claiming':nimFunded(p)&&p.claim_state==='unclaimed'?'unclaimed':unpaidCard(p,now)?'unpaid':'checking';
 return senderHistoryText(language,key);
}
export async function cancelUnpaidCard(p,request) {
 try{return {...await request('/api/payments/'+p.id+'/cancel',{}),received:false};}
 catch(error){
  if(error.message!=='already_funded')throw error;
  let result;
  try{result=await request('/api/payments/'+p.id+'/confirm',{});}
  catch{result={payment:{...p,stage:'funded'}};}
  return {...result,received:true};
 }
}
