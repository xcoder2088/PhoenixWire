// Wire-only wording for the shared payment window. Keep /pm/ unchanged.
(()=>{
  const words=window.PhoenixUSDTCashlink.WORDS;
  const confirmations={
  "en": "You get back {net} USDT at the chosen address; the relayer’s fees ({fee} USDT) are retained. Continue?",
  "fr": "Vous récupérez {net} USDT à l’adresse choisie ; les frais du relais ({fee} USDT) sont conservés. Continuer ?",
  "es": "Recuperas {net} USDT en la dirección elegida; se retienen las comisiones del relé ({fee} USDT). ¿Continuar?",
  "de": "Du erhältst {net} USDT an der gewählten Adresse zurück; die Gebühren des Relays ({fee} USDT) werden einbehalten. Fortfahren?",
  "zh": "你将在所选地址取回 {net} USDT；中继费用（{fee} USDT）将被保留。继续吗？"
};
  for(const [language,refundConfirm] of Object.entries(confirmations)) words[language].refundConfirm=refundConfirm;
})();
