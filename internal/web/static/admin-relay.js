// The administrator's view of the gas wallets ("relayers") of the USDC / USDT cards: what each one holds, what the cards earned, and the way to
// take the surplus out. The server decides what stays on a wallet (the minimum the service needs, plus the gas of the cards still alive) and
// refuses every address that is not on the operator's list: this page only asks.
const words = {
  fr: {
    relays: 'Portefeuilles de gaz et gains', refresh: 'Actualiser', loading: 'Chargement…', none: 'Aucun portefeuille de gaz configuré.',
    gasCoin: 'Gaz', kept: 'gardé en réserve', withdrawable: 'retirable', waiting: 'Cartes en attente', fees: 'Frais gagnés', earnedTitle: 'Ce que les cartes ont rapporté',
    payouts: 'versements', earned: 'frais', gas: 'gaz payé', margin: 'marge', withdraw: 'Retirer', to: 'Vers', noAddress: 'Retraits désactivés : aucune adresse autorisée sur le serveur.',
    confirmWithdraw: 'Retirer {amount} {symbol} du portefeuille {chain} vers {to} ? Le portefeuille garde le minimum dont le service a besoin.',
    sent: 'Envoyé', unconfirmed: 'Envoi en cours, à vérifier sur l’explorateur', help: 'Le portefeuille garde toujours le minimum pour que le service continue de fonctionner.',
    nothing_to_withdraw: 'Rien à retirer pour le moment : tout le reste sert de réserve.', withdraw_not_configured: 'Retraits désactivés : aucune adresse autorisée sur le serveur.',
    withdraw_destination_not_allowed: 'Cette adresse n’est pas dans la liste autorisée du serveur.', reserve_required: 'Le gaz de ce retrait entamerait la réserve des cartes en cours.',
    usdt_unavailable: 'USDC et USDT ne sont pas disponibles ici.', failed: 'Impossible pour le moment. Réessaie.', busy: 'En cours…'
  },
  en: {
    relays: 'Gas wallets and earnings', refresh: 'Refresh', loading: 'Loading…', none: 'No gas wallet is set up.',
    gasCoin: 'Gas', kept: 'kept in reserve', withdrawable: 'can be withdrawn', waiting: 'Cards waiting', fees: 'Fees held', earnedTitle: 'What the cards earned',
    payouts: 'payouts', earned: 'fees', gas: 'gas paid', margin: 'margin', withdraw: 'Withdraw', to: 'To', noAddress: 'Withdrawals are off: the server has no authorized address.',
    confirmWithdraw: 'Withdraw {amount} {symbol} from the {chain} wallet to {to}? The wallet keeps the minimum the service needs.',
    sent: 'Sent', unconfirmed: 'Sending, check the explorer', help: 'The wallet always keeps the minimum the service needs to keep working.',
    nothing_to_withdraw: 'Nothing to withdraw right now: the rest is the reserve.', withdraw_not_configured: 'Withdrawals are off: the server has no authorized address.',
    withdraw_destination_not_allowed: 'This address is not on the server’s authorized list.', reserve_required: 'The gas of this withdrawal would eat into the reserve for live cards.',
    usdt_unavailable: 'USDC and USDT are not available here.', failed: 'Unavailable right now. Please try again.', busy: 'Working…'
  },
  es: {
    relays: 'Carteras de gas y ganancias', refresh: 'Actualizar', loading: 'Cargando…', none: 'No hay ninguna cartera de gas configurada.',
    gasCoin: 'Gas', kept: 'guardado en reserva', withdrawable: 'retirable', waiting: 'Tarjetas en espera', fees: 'Comisiones acumuladas', earnedTitle: 'Lo que han ganado las tarjetas',
    payouts: 'pagos', earned: 'comisiones', gas: 'gas pagado', margin: 'margen', withdraw: 'Retirar', to: 'A', noAddress: 'Retiros desactivados: el servidor no tiene ninguna dirección autorizada.',
    confirmWithdraw: '¿Retirar {amount} {symbol} de la cartera {chain} a {to}? La cartera conserva el mínimo que necesita el servicio.',
    sent: 'Enviado', unconfirmed: 'Enviando, comprueba el explorador', help: 'La cartera siempre conserva el mínimo para que el servicio siga funcionando.',
    nothing_to_withdraw: 'No hay nada que retirar ahora: el resto es la reserva.', withdraw_not_configured: 'Retiros desactivados: el servidor no tiene ninguna dirección autorizada.',
    withdraw_destination_not_allowed: 'Esta dirección no está en la lista autorizada del servidor.', reserve_required: 'El gas de este retiro tocaría la reserva de las tarjetas activas.',
    usdt_unavailable: 'USDC y USDT no están disponibles aquí.', failed: 'No está disponible ahora. Inténtalo de nuevo.', busy: 'En curso…'
  },
  de: {
    relays: 'Gas-Wallets und Einnahmen', refresh: 'Aktualisieren', loading: 'Wird geladen…', none: 'Keine Gas-Wallet eingerichtet.',
    gasCoin: 'Gas', kept: 'als Reserve behalten', withdrawable: 'abhebbar', waiting: 'Wartende Karten', fees: 'Gehaltene Gebühren', earnedTitle: 'Was die Karten eingebracht haben',
    payouts: 'Auszahlungen', earned: 'Gebühren', gas: 'Gas bezahlt', margin: 'Marge', withdraw: 'Abheben', to: 'An', noAddress: 'Abhebungen sind aus: Der Server hat keine freigegebene Adresse.',
    confirmWithdraw: '{amount} {symbol} aus der {chain}-Wallet an {to} abheben? Die Wallet behält das Minimum, das der Dienst braucht.',
    sent: 'Gesendet', unconfirmed: 'Wird gesendet, im Explorer prüfen', help: 'Die Wallet behält immer das Minimum, damit der Dienst weiterläuft.',
    nothing_to_withdraw: 'Im Moment nichts abzuheben: der Rest ist die Reserve.', withdraw_not_configured: 'Abhebungen sind aus: Der Server hat keine freigegebene Adresse.',
    withdraw_destination_not_allowed: 'Diese Adresse steht nicht auf der freigegebenen Liste des Servers.', reserve_required: 'Das Gas dieser Abhebung würde die Reserve für laufende Karten angreifen.',
    usdt_unavailable: 'USDC und USDT sind hier nicht verfügbar.', failed: 'Derzeit nicht verfügbar. Versuche es erneut.', busy: 'Läuft…'
  },
  zh: {
    relays: 'Gas 钱包与收益', refresh: '刷新', loading: '加载中…', none: '尚未设置 Gas 钱包。',
    gasCoin: 'Gas', kept: '保留作储备', withdrawable: '可提取', waiting: '等待中的卡片', fees: '已收手续费', earnedTitle: '卡片的收益',
    payouts: '笔支付', earned: '手续费', gas: '已付 Gas', margin: '利润', withdraw: '提取', to: '提取到', noAddress: '提取已关闭：服务器没有授权地址。',
    confirmWithdraw: '从 {chain} 钱包向 {to} 提取 {amount} {symbol}？钱包会保留服务所需的最低额度。',
    sent: '已发送', unconfirmed: '发送中，请在区块浏览器核对', help: '钱包始终保留让服务继续运行的最低额度。',
    nothing_to_withdraw: '目前没有可提取的金额：其余部分是储备。', withdraw_not_configured: '提取已关闭：服务器没有授权地址。',
    withdraw_destination_not_allowed: '此地址不在服务器的授权列表中。', reserve_required: '此次提取的 Gas 会动用进行中卡片的储备。',
    usdt_unavailable: '此处不提供 USDC 和 USDT。', failed: '暂时无法操作，请重试。', busy: '处理中…'
  }
};
export function relayText(language, key) {
  const copy = Object.hasOwn(words, language) ? words[language] : words.en;
  return Object.hasOwn(copy, key) ? copy[key] : words.en.failed;
}
const assetNames = { usdc_polygon: 'USDC · Polygon', usdt_polygon: 'USDT · Polygon', usdt_ethereum: 'USDT · Ethereum' };
const chainNames = { polygon: 'Polygon', ethereum: 'Ethereum' };

// A whole number of the smallest unit as text with `shown` decimals, cut and not rounded (no float: wei do not fit in one).
export function amountText(value, decimals, shown) {
  try {
    const n = BigInt(value), base = 10n ** BigInt(decimals);
    const fraction = (n % base).toString().padStart(decimals, '0').slice(0, shown);
    return String(n / base) + (shown ? '.' + fraction : '');
  } catch { return '—'; }
}

export function mountRelays({ document, language, request, confirm }) {
  const box = document.createElement('details'), summary = document.createElement('summary'), body = document.createElement('div');
  box.id = 'wire-admin-relays'; box.className = 'wire-admin-relays'; body.className = 'wire-admin-relays-body'; box.append(summary, body);
  let data = null, notice = '', busy = false, loaded = false, closed = false, epoch = 0, destination = '';
  const t = key => relayText(language(), key);
  const node = (tag, text) => { const el = document.createElement(tag); if (text !== undefined) el.textContent = text; return el; };

  function accept(result) {
    const strings = list => Array.isArray(list) && list.every(value => typeof value === 'string');
    if (!result || !Array.isArray(result.relayers) || !strings(result.withdraw_to ?? [])) throw Error('invalid_response');
    data = { relayers: result.relayers.filter(r => r && typeof r.chain === 'string'), to: result.withdraw_to ?? [], totals: Array.isArray(result.totals) ? result.totals : [] };
    if (!data.to.includes(destination)) destination = data.to[0] ?? '';
  }
  async function load() {
    if (busy || closed) return;
    busy = true; notice = ''; const current = ++epoch; render();
    try { const result = await request('/api/admin/relayers'); if (!closed && current === epoch) { accept(result); loaded = true; } }
    catch (error) { if (!closed && current === epoch) notice = errorKey(error); }
    finally { if (!closed && current === epoch) { busy = false; render(); } }
  }
  const known = ['nothing_to_withdraw', 'withdraw_not_configured', 'withdraw_destination_not_allowed', 'reserve_required', 'usdt_unavailable'];
  const errorKey = error => { const code = error?.code || error?.message; return known.includes(code) ? code : 'failed'; };

  async function withdraw(relayer, what, symbol, amount) {
    if (busy || closed || !destination) return;
    if (!confirm(t('confirmWithdraw').replace('{amount}', amount).replace('{symbol}', symbol).replace('{chain}', chainNames[relayer.chain] || relayer.chain).replace('{to}', destination))) return;
    busy = true; notice = ''; const current = ++epoch; render();
    let sent = null;
    try { sent = await request('/api/admin/relayers/withdraw', { chain: relayer.chain, what, to: destination }); }
    catch (error) { if (!closed && current === epoch) notice = errorKey(error); }
    if (closed || current !== epoch) return;
    busy = false;
    if (sent) { notice = ''; result = sent; } else result = null;
    render();
    if (sent) void load(); // the balances after the transaction
  }
  let result = null;

  function relayerNode(relayer) {
    const section = node('section'); section.className = 'wire-admin-relay'; section.dataset.chain = relayer.chain;
    const address = typeof relayer.address === 'string' ? relayer.address.slice(0, 6) + '…' + relayer.address.slice(-4) : '';
    section.append(node('h4', (chainNames[relayer.chain] || relayer.chain) + ' · ' + address));
    const symbol = relayer.native_symbol || '';
    section.append(node('p', t('gasCoin') + ' ' + symbol + ': ' + amountText(relayer.native_wei, 18, 4) + ' · ' + amountText(relayer.reserve_wei, 18, 4) + ' ' + t('kept') + ' · ' + amountText(relayer.withdrawable_wei, 18, 4) + ' ' + t('withdrawable')));
    section.append(node('p', t('waiting') + ': ' + (Number.isSafeInteger(relayer.pending_cards) ? relayer.pending_cards : 0)));
    const actions = node('div'); actions.className = 'wire-admin-row-actions';
    const native = amountText(relayer.withdrawable_wei, 18, 4);
    const nativeButton = node('button', t('withdraw') + ' ' + native + ' ' + symbol); nativeButton.type = 'button'; nativeButton.className = 'secondary-btn'; nativeButton.dataset.what = 'native';
    nativeButton.disabled = busy || !destination || !/^[1-9]\d*$/.test(String(relayer.withdrawable_wei));
    nativeButton.addEventListener('click', () => { void withdraw(relayer, 'native', symbol, native); });
    actions.append(nativeButton);
    for (const token of Array.isArray(relayer.tokens) ? relayer.tokens : []) {
      const label = assetNames[token.asset] || token.symbol || token.asset, shown = amountText(token.units, 6, 4);
      section.append(node('p', t('fees') + ' ' + label + ': ' + shown));
      const button = node('button', t('withdraw') + ' ' + shown + ' ' + (token.symbol || '')); button.type = 'button'; button.className = 'secondary-btn'; button.dataset.what = token.asset;
      button.disabled = busy || !destination || !/^[1-9]\d*$/.test(String(token.units));
      button.addEventListener('click', () => { void withdraw(relayer, token.asset, token.symbol || '', shown); });
      actions.append(button);
    }
    section.append(actions);
    return section;
  }

  function render() {
    if (closed) return;
    summary.textContent = t('relays'); body.replaceChildren();
    if (!box.open && !loaded) return;
    const refresh = node('button', t('refresh')); refresh.type = 'button'; refresh.className = 'secondary-btn'; refresh.disabled = busy; refresh.addEventListener('click', () => { void load(); });
    const status = node('p', busy ? t('busy') : notice ? t(notice) : ''); status.setAttribute('role', 'status'); status.className = 'wire-admin-relay-status';
    body.append(refresh, status);
    if (result) {
      const done = node('p', t(result.status === 'sent' ? 'sent' : 'unconfirmed') + ': ' + (typeof result.tx === 'string' ? result.tx : '')); done.className = 'wire-admin-relay-result';
      if (typeof result.explorer === 'string' && result.explorer.startsWith('https://') && /^0x[0-9a-fA-F]{64}$/.test(result.tx || '')) {
        const link = node('a', result.tx); link.href = result.explorer + result.tx; link.target = '_blank'; link.rel = 'noopener noreferrer';
        done.textContent = t(result.status === 'sent' ? 'sent' : 'unconfirmed') + ': '; done.append(link);
      }
      body.append(done);
    }
    if (!data) return;
    if (!data.relayers.length) { body.append(node('p', t('none'))); return; }
    if (!data.to.length) body.append(node('p', t('noAddress')));
    else if (data.to.length === 1) body.append(node('p', t('to') + ': ' + data.to[0]));
    else {
      const label = node('label', t('to')), select = node('select'); select.id = 'wire-admin-relay-to'; select.disabled = busy;
      for (const address of data.to) { const option = node('option', address); option.value = address; option.selected = address === destination; select.append(option); }
      select.addEventListener('change', () => { destination = select.value; });
      label.append(select); body.append(label);
    }
    for (const relayer of data.relayers) body.append(relayerNode(relayer));
    body.append(node('p', t('help')));
    if (data.totals.length) {
      const earned = node('section'); earned.className = 'wire-admin-earned'; earned.append(node('h4', t('earnedTitle')));
      for (const row of data.totals) {
        const net = typeof row.net === 'string' ? row.net : '—';
        earned.append(node('p', (assetNames[row.asset] || row.asset) + ' · ' + (Number.isSafeInteger(row.payouts) ? row.payouts : 0) + ' ' + t('payouts') + ' · ' + t('earned') + ' ' + (row.earned ?? '—') + ' · ' + t('gas') + ' ' + (row.cost ?? '—') + ' · ' + t('margin') + ' ' + net));
      }
      body.append(earned);
    }
  }
  box.addEventListener('toggle', () => { if (box.open && !loaded) void load(); else render(); });
  return { node: box, render, destroy() { closed = true; epoch++; box.remove(); } };
}
