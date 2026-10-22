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
  USDT0_POLYGON: { unit: 'USDT0', network: 'Polygon', symbol: '₮', decimals: 6 }
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
