export const supportedLanguages = Object.freeze(['fr', 'en', 'es', 'de', 'zh']);

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
