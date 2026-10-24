/* MIT. Copyright (c) 2026 Xcoder. USDT cards (Ethereum, Polygon): the same window as Nimiq, other asset. No wallet connection: the sender pays from any wallet. */
(function(root){'use strict';
const CHAINS={
 usdt_ethereum:{key:'ethereum',symbol:'USDT',name:'Ethereum',label:'USDT · Ethereum',gas:'ETH',explorer:/^https:\/\/etherscan\.io\/tx\/0x[0-9a-fA-F]{64}$/},
 usdc_polygon:{key:'polygon',symbol:'USDC',name:'Polygon',label:'USDC · Polygon',gas:'POL',explorer:/^https:\/\/polygonscan\.com\/tx\/0x[0-9a-fA-F]{64}$/},
 usdt_polygon:{key:'polygon',symbol:'USDT',name:'Polygon',label:'USDT · Polygon',gas:'POL',explorer:/^https:\/\/polygonscan\.com\/tx\/0x[0-9a-fA-F]{64}$/}
};
const WORDS={
 fr:{openSends:'Mes envois USDT',modes:['Demander un paiement','Envoyer USDT'],wallet:'Mon adresse USDT',another:'Autre adresse…',address:'Adresse USDT (0x…)',amount:'Montant (USDT)',reference:'Référence (facultative)',message:'Message (facultatif)',create:'Créer la carte',cancel:'Annuler',
  protectedTitle:'Protégé',protectedNote:'Code de six chiffres par courriel · 10 minutes · maximum 1 000 USDT.',protectedChosen:'Le destinataire doit vérifier son courriel avant de donner son adresse.',recipient:'Courriel du destinataire',openTitle:'Non protégé',openNote:'Toute personne qui possède le lien peut choisir où recevoir les USDT.',
  how:'Tu crées la carte et tu l’ajoutes au courriel. Quand la personne donne l’adresse où recevoir, tu scannes un seul QR code pour payer : ton portefeuille ajoute les frais du réseau, comme pour tout transfert. Elle reçoit exactement le montant.',saveHint:'Enregistre ton adresse USDT dans Paramètres → Mon portefeuille pour la choisir ici en un clic.',
  senderWallet:'Je paierai depuis',anyWallet:'Un autre portefeuille ou une plateforme',senderHint:'Choisis l’adresse depuis laquelle tu paieras : PhoenixMail reconnaît ainsi ton paiement. Un autre portefeuille ou une plateforme d’échange fonctionne aussi.',
  request:'Demande de paiement',scan:'Scannez avec votre portefeuille ou copiez l’adresse.',only:'Envoyez uniquement des USDT sur {chain}.',unpaid:'Demande non réglée',addToEmail:'Ajouter au courriel',added:'Carte ajoutée au courriel.',edit:'Modifier',
  created:'Carte ajoutée au courriel. Quand le destinataire aura donné son adresse, tu seras prévenu par courriel : tu scanneras alors un seul QR code dans Paramètres → Mon portefeuille (Mes envois USDT).',cardTitle:'Votre carte',
  unavailable:'L’envoi d’USDT est indisponible pour le moment.',protectedUnavailable:'Le paiement protégé est indisponible pour le moment. Choisis un paiement non protégé ou réessaie plus tard.',
  errors:{invalid_request:'Demande invalide. Vérifie le montant et le courriel.',usdt_unavailable:'Le service USDT est indisponible. Réessaie plus tard.',too_many_cards:'Trop de cartes créées. Réessaie dans une heure.',chain_unavailable:'Le réseau est inaccessible pour le moment. Réessaie bientôt.',invalid_service_answer:'Réponse du service invalide. Aucune carte n’a été utilisée.',forbidden:'Refusé : ce suivi ne correspond pas à cette carte, ou elle est déjà payée.',claim_not_found:'Carte introuvable.',invalid_address:'Adresse invalide.',claim_expired:'Cette carte a expiré.',card_cancelled:'Cette carte est annulée.'},
  sentTitle:'Mes envois USDT',sentIntro:'Le destinataire donne son adresse, puis tu paies d’un seul scan. Ici tu vois où en est chaque carte. Le suivi reste sur cet appareil.',none:'Aucun envoi USDT pour l’instant.',stAwaiting:'En attente de l’adresse du destinataire',stReady:'Prêt à payer',stPaid:'Payé',stCancelled:'Annulé',stExpired:'Expiré',late:'Le destinataire n’a pas donné d’adresse depuis plus de 24 h',
  payTitle:'Paie {amount} USDT sur {chain}',payHelp:'Scanne avec ton portefeuille. Il envoie exactement {amount} USDT au destinataire et ajoute lui-même les frais du réseau, comme pour tout transfert.',payTo:'Adresse du destinataire',copyAddress:'Copier l’adresse',copyAmount:'Copier le montant',copied:'Copié',openWallet:'Ouvrir dans mon portefeuille',paying:'Paiement en attente… cette page le détecte dès qu’il est confirmé.',
  cancelCard:'Annuler cette carte',cancelConfirm:'Annuler cette carte ? Le destinataire ne pourra plus donner d’adresse.',copyProof:'Copier la preuve',proofCopied:'Preuve copiée',forget:'Supprimer ce suivi ?',view:'Voir la transaction',addCard:'Ajouter la carte au courriel',to:'Destinataire prévu',refresh:'Actualiser',protectedBadge:'Protégé',
  proofTitle:'PhoenixMail · Carte USDT',proofAmount:'Montant',proofDate:'Date',proofCard:'Carte',proofState:'État',proofTx:'Transaction',proofChecked:'vérifié le',
  walletTitle:'Mon portefeuille USDT',walletIntro:'Tes adresses USDT, enregistrées sur cet appareil. Elles reçoivent les demandes de paiement et servent à reconnaître tes paiements. La même adresse fonctionne sur Ethereum et Polygon.',walletEmpty:'Aucune adresse USDT enregistrée.',walletAdd:'Ajouter une adresse USDT',walletName:'Nom (facultatif)',walletAddButton:'Ajouter',walletInvalid:'Adresse invalide : vérifie chaque caractère.',walletDelete:'Supprimer',walletBoth:'Les deux réseaux',walletUse:'Utiliser cette adresse sur',walletOnly:'{n} seulement',walletChain:'Réseau',walletAddress:'Adresse USDT'},
 en:{openSends:'My USDT sends',modes:['Request payment','Send USDT'],wallet:'My USDT address',another:'Another address…',address:'USDT address (0x…)',amount:'Amount (USDT)',reference:'Reference (optional)',message:'Message (optional)',create:'Create card',cancel:'Cancel',
  protectedTitle:'Protected',protectedNote:'Six-digit email code · 10 minutes · maximum 1,000 USDT.',protectedChosen:'The recipient must verify their email before giving their address.',recipient:'Recipient email',openTitle:'Non-protected',openNote:'Anyone with the link can choose where to receive the USDT.',
  how:'You create the card and add it to the email. When the person gives the address to receive at, you scan one QR code to pay: your wallet adds the network fee, as for any transfer. They receive exactly the amount.',saveHint:'Save your USDT address in Settings → My wallet to pick it here in one click.',
  senderWallet:'I will pay from',anyWallet:'Another wallet or an exchange',senderHint:'Choose the address you will pay from: PhoenixMail recognises your payment that way. Another wallet or an exchange works too.',
  request:'Payment request',scan:'Scan with your wallet or copy the address.',only:'Send only USDT on {chain}.',unpaid:'Unpaid request',addToEmail:'Add to email',added:'Card added to the email.',edit:'Edit',
  created:'Card added to the email. When the recipient has given their address you will be told by email: then scan one QR code in Settings → My wallet (My USDT sends).',cardTitle:'Your card',
  unavailable:'USDT sending is not available right now.',protectedUnavailable:'Protected payment is not available right now. Choose a non-protected payment or try again later.',
  errors:{invalid_request:'Invalid request. Check the amount and the email.',usdt_unavailable:'The USDT service is not available. Try again later.',too_many_cards:'Too many cards created. Try again in an hour.',chain_unavailable:'The network is not reachable right now. Try again shortly.',invalid_service_answer:'Invalid answer from the service. No card was used.',forbidden:'Refused: this record does not match this card, or it is already paid.',claim_not_found:'Card not found.',invalid_address:'Invalid address.',claim_expired:'This card has expired.',card_cancelled:'This card is cancelled.'},
  sentTitle:'My USDT sends',sentIntro:'The recipient gives their address, then you pay with one scan. Here you see where each card stands. Tracking stays on this device.',none:'No USDT send yet.',stAwaiting:'Waiting for the recipient’s address',stReady:'Ready to pay',stPaid:'Paid',stCancelled:'Cancelled',stExpired:'Expired',late:'The recipient has not given an address for more than 24 h',
  payTitle:'Pay {amount} USDT on {chain}',payHelp:'Scan with your wallet. It sends exactly {amount} USDT to the recipient and adds the network fee itself, as for any transfer.',payTo:'Recipient address',copyAddress:'Copy address',copyAmount:'Copy amount',copied:'Copied',openWallet:'Open in my wallet',paying:'Waiting for the payment… this page notices it as soon as it is confirmed.',
  cancelCard:'Cancel this card',cancelConfirm:'Cancel this card? The recipient will no longer be able to give an address.',copyProof:'Copy proof',proofCopied:'Proof copied',forget:'Remove this record?',view:'View transaction',addCard:'Add card to email',to:'Intended recipient',refresh:'Refresh',protectedBadge:'Protected',
  proofTitle:'PhoenixMail · USDT card',proofAmount:'Amount',proofDate:'Date',proofCard:'Card',proofState:'State',proofTx:'Transaction',proofChecked:'checked on',
  walletTitle:'My USDT wallet',walletIntro:'Your USDT addresses, saved on this device. They receive payment requests and help recognise your payments. The same address works on Ethereum and Polygon.',walletEmpty:'No USDT address saved.',walletAdd:'Add a USDT address',walletName:'Name (optional)',walletAddButton:'Add',walletInvalid:'Invalid address: check every character.',walletDelete:'Delete',walletBoth:'Both networks',walletUse:'Use this address on',walletOnly:'{n} only',walletChain:'Network',walletAddress:'USDT address'},
 es:{openSends:'Mis envíos USDT',modes:['Solicitar pago','Enviar USDT'],wallet:'Mi dirección USDT',another:'Otra dirección…',address:'Dirección USDT (0x…)',amount:'Importe (USDT)',reference:'Referencia (opcional)',message:'Mensaje (opcional)',create:'Crear tarjeta',cancel:'Cancelar',
  protectedTitle:'Protegido',protectedNote:'Código de seis dígitos por correo · 10 minutos · máximo 1.000 USDT.',protectedChosen:'El destinatario debe verificar su correo antes de dar su dirección.',recipient:'Correo del destinatario',openTitle:'Sin protección',openNote:'Cualquier persona con el enlace puede elegir dónde recibir los USDT.',
  how:'Creas la tarjeta y la añades al correo. Cuando la persona da la dirección donde recibir, escaneas un solo código QR para pagar: tu monedero añade la comisión de red, como en cualquier transferencia. Ella recibe exactamente el importe.',saveHint:'Guarda tu dirección USDT en Ajustes → Mi monedero para elegirla aquí con un clic.',
  senderWallet:'Pagaré desde',anyWallet:'Otro monedero o un exchange',senderHint:'Elige la dirección desde la que pagarás: así PhoenixMail reconoce tu pago. Otro monedero o un exchange también sirve.',
  request:'Solicitud de pago',scan:'Escanea con tu monedero o copia la dirección.',only:'Envía solo USDT en {chain}.',unpaid:'Solicitud sin pagar',addToEmail:'Añadir al correo',added:'Tarjeta añadida al correo.',edit:'Editar',
  created:'Tarjeta añadida al correo. Cuando el destinatario haya dado su dirección te avisaremos por correo: entonces escanearás un solo código QR en Ajustes → Mi monedero (Mis envíos USDT).',cardTitle:'Tu tarjeta',
  unavailable:'El envío de USDT no está disponible ahora.',protectedUnavailable:'El pago protegido no está disponible ahora. Elige un pago sin protección o inténtalo más tarde.',
  errors:{invalid_request:'Solicitud no válida. Comprueba el importe y el correo.',usdt_unavailable:'El servicio USDT no está disponible. Inténtalo más tarde.',too_many_cards:'Demasiadas tarjetas creadas. Inténtalo en una hora.',chain_unavailable:'La red no está disponible ahora. Inténtalo en un momento.',invalid_service_answer:'Respuesta del servicio no válida. No se usó ninguna tarjeta.',forbidden:'Rechazado: este registro no corresponde a esta tarjeta, o ya está pagada.',claim_not_found:'Tarjeta no encontrada.',invalid_address:'Dirección no válida.',claim_expired:'Esta tarjeta ha caducado.',card_cancelled:'Esta tarjeta está cancelada.'},
  sentTitle:'Mis envíos USDT',sentIntro:'El destinatario da su dirección y luego pagas con un solo escaneo. Aquí ves cómo va cada tarjeta. El seguimiento queda en este dispositivo.',none:'Aún no hay envíos USDT.',stAwaiting:'Esperando la dirección del destinatario',stReady:'Listo para pagar',stPaid:'Pagado',stCancelled:'Cancelado',stExpired:'Caducado',late:'El destinatario no ha dado su dirección desde hace más de 24 h',
  payTitle:'Paga {amount} USDT en {chain}',payHelp:'Escanea con tu monedero. Envía exactamente {amount} USDT al destinatario y añade él mismo la comisión de red, como en cualquier transferencia.',payTo:'Dirección del destinatario',copyAddress:'Copiar dirección',copyAmount:'Copiar importe',copied:'Copiado',openWallet:'Abrir en mi monedero',paying:'Esperando el pago… esta página lo detecta en cuanto se confirma.',
  cancelCard:'Cancelar esta tarjeta',cancelConfirm:'¿Cancelar esta tarjeta? El destinatario ya no podrá dar su dirección.',copyProof:'Copiar prueba',proofCopied:'Prueba copiada',forget:'¿Eliminar este registro?',view:'Ver la transacción',addCard:'Añadir la tarjeta al correo',to:'Destinatario previsto',refresh:'Actualizar',protectedBadge:'Protegido',
  proofTitle:'PhoenixMail · Tarjeta USDT',proofAmount:'Importe',proofDate:'Fecha',proofCard:'Tarjeta',proofState:'Estado',proofTx:'Transacción',proofChecked:'comprobado el',
  walletTitle:'Mi monedero USDT',walletIntro:'Tus direcciones USDT, guardadas en este dispositivo. Reciben las solicitudes de pago y ayudan a reconocer tus pagos. La misma dirección sirve en Ethereum y Polygon.',walletEmpty:'Ninguna dirección USDT guardada.',walletAdd:'Añadir una dirección USDT',walletName:'Nombre (opcional)',walletAddButton:'Añadir',walletInvalid:'Dirección no válida: revisa cada carácter.',walletDelete:'Eliminar',walletBoth:'Ambas redes',walletUse:'Usar esta dirección en',walletOnly:'Solo {n}',walletChain:'Red',walletAddress:'Dirección USDT'},
 de:{openSends:'Meine USDT-Sendungen',modes:['Zahlung anfordern','USDT senden'],wallet:'Meine USDT-Adresse',another:'Andere Adresse…',address:'USDT-Adresse (0x…)',amount:'Betrag (USDT)',reference:'Referenz (optional)',message:'Nachricht (optional)',create:'Karte erstellen',cancel:'Abbrechen',
  protectedTitle:'Geschützt',protectedNote:'Sechsstelliger E-Mail-Code · 10 Minuten · maximal 1.000 USDT.',protectedChosen:'Der Empfänger muss seine E-Mail bestätigen, bevor er seine Adresse angibt.',recipient:'E-Mail des Empfängers',openTitle:'Ungeschützt',openNote:'Jeder mit dem Link kann wählen, wohin die USDT gehen.',
  how:'Du erstellst die Karte und fügst sie der E-Mail hinzu. Sobald die Person die Adresse zum Empfangen angibt, scannst du einen einzigen QR-Code zum Bezahlen: Deine Wallet fügt die Netzwerkgebühr hinzu, wie bei jeder Überweisung. Sie erhält genau den Betrag.',saveHint:'Speichere deine USDT-Adresse unter Einstellungen → Meine Wallet, um sie hier mit einem Klick zu wählen.',
  senderWallet:'Ich bezahle von',anyWallet:'Andere Wallet oder Börse',senderHint:'Wähle die Adresse, von der du bezahlst: So erkennt PhoenixMail deine Zahlung. Eine andere Wallet oder eine Börse geht auch.',
  request:'Zahlungsanforderung',scan:'Mit deiner Wallet scannen oder die Adresse kopieren.',only:'Sende nur USDT auf {chain}.',unpaid:'Unbezahlte Anforderung',addToEmail:'Zur E-Mail hinzufügen',added:'Karte zur E-Mail hinzugefügt.',edit:'Bearbeiten',
  created:'Karte zur E-Mail hinzugefügt. Sobald der Empfänger seine Adresse angegeben hat, erhältst du eine E-Mail: Dann scannst du einen QR-Code unter Einstellungen → Meine Wallet (Meine USDT-Sendungen).',cardTitle:'Deine Karte',
  unavailable:'Der USDT-Versand ist gerade nicht verfügbar.',protectedUnavailable:'Die geschützte Zahlung ist gerade nicht verfügbar. Wähle eine ungeschützte Zahlung oder versuche es später erneut.',
  errors:{invalid_request:'Ungültige Anfrage. Prüfe Betrag und E-Mail.',usdt_unavailable:'Der USDT-Dienst ist nicht verfügbar. Versuche es später erneut.',too_many_cards:'Zu viele Karten erstellt. Versuche es in einer Stunde erneut.',chain_unavailable:'Das Netzwerk ist gerade nicht erreichbar. Versuche es gleich noch einmal.',invalid_service_answer:'Ungültige Antwort des Dienstes. Es wurde keine Karte verwendet.',forbidden:'Abgelehnt: Dieser Eintrag gehört nicht zu dieser Karte, oder sie ist bereits bezahlt.',claim_not_found:'Karte nicht gefunden.',invalid_address:'Ungültige Adresse.',claim_expired:'Diese Karte ist abgelaufen.',card_cancelled:'Diese Karte wurde storniert.'},
  sentTitle:'Meine USDT-Sendungen',sentIntro:'Der Empfänger gibt seine Adresse an, dann bezahlst du mit einem Scan. Hier siehst du, wo jede Karte steht. Die Verfolgung bleibt auf diesem Gerät.',none:'Noch keine USDT-Sendung.',stAwaiting:'Wartet auf die Adresse des Empfängers',stReady:'Bereit zum Bezahlen',stPaid:'Bezahlt',stCancelled:'Storniert',stExpired:'Abgelaufen',late:'Der Empfänger hat seit über 24 h keine Adresse angegeben',
  payTitle:'Bezahle {amount} USDT auf {chain}',payHelp:'Scanne mit deiner Wallet. Sie sendet genau {amount} USDT an den Empfänger und fügt die Netzwerkgebühr selbst hinzu, wie bei jeder Überweisung.',payTo:'Adresse des Empfängers',copyAddress:'Adresse kopieren',copyAmount:'Betrag kopieren',copied:'Kopiert',openWallet:'In meiner Wallet öffnen',paying:'Warte auf die Zahlung… diese Seite erkennt sie, sobald sie bestätigt ist.',
  cancelCard:'Diese Karte stornieren',cancelConfirm:'Diese Karte stornieren? Der Empfänger kann dann keine Adresse mehr angeben.',copyProof:'Nachweis kopieren',proofCopied:'Nachweis kopiert',forget:'Diesen Eintrag entfernen?',view:'Transaktion ansehen',addCard:'Karte zur E-Mail hinzufügen',to:'Vorgesehener Empfänger',refresh:'Aktualisieren',protectedBadge:'Geschützt',
  proofTitle:'PhoenixMail · USDT-Karte',proofAmount:'Betrag',proofDate:'Datum',proofCard:'Karte',proofState:'Status',proofTx:'Transaktion',proofChecked:'geprüft am',
  walletTitle:'Meine USDT-Wallet',walletIntro:'Deine USDT-Adressen, auf diesem Gerät gespeichert. Sie empfangen Zahlungsanforderungen und helfen, deine Zahlungen zu erkennen. Dieselbe Adresse funktioniert auf Ethereum und Polygon.',walletEmpty:'Keine USDT-Adresse gespeichert.',walletAdd:'USDT-Adresse hinzufügen',walletName:'Name (optional)',walletAddButton:'Hinzufügen',walletInvalid:'Ungültige Adresse: Prüfe jedes Zeichen.',walletDelete:'Löschen',walletBoth:'Beide Netzwerke',walletUse:'Diese Adresse verwenden auf',walletOnly:'Nur {n}',walletChain:'Netzwerk',walletAddress:'USDT-Adresse'},
 zh:{openSends:'我的 USDT 发送',modes:['请求付款','发送 USDT'],wallet:'我的 USDT 地址',another:'其他地址…',address:'USDT 地址 (0x…)',amount:'金额 (USDT)',reference:'参考编号（可选）',message:'留言（可选）',create:'创建卡片',cancel:'取消',
  protectedTitle:'受保护',protectedNote:'六位邮件验证码 · 10 分钟 · 上限 1,000 USDT。',protectedChosen:'收件人必须先验证邮箱，才能提供他的地址。',recipient:'收件人邮箱',openTitle:'未受保护',openNote:'任何拥有链接的人都可以选择在哪里接收 USDT。',
  how:'你创建卡片并添加到邮件。当对方提供接收地址后，你只需扫描一个二维码付款：你的钱包会像任何转账一样加上网络费用。对方收到的正好是该金额。',saveHint:'在 设置 → 我的钱包 中保存你的 USDT 地址，即可在此一键选择。',
  senderWallet:'我将从此付款',anyWallet:'其他钱包或交易所',senderHint:'选择你要用来付款的地址：PhoenixMail 据此识别你的付款。其他钱包或交易所也可以。',
  request:'付款请求',scan:'用你的钱包扫描，或复制地址。',only:'仅在 {chain} 上发送 USDT。',unpaid:'尚未支付的请求',addToEmail:'添加到邮件',added:'卡片已添加到邮件。',edit:'编辑',
  created:'卡片已添加到邮件。收件人提供地址后，我们会通过邮件通知你：然后在 设置 → 我的钱包（我的 USDT 发送）中扫描一个二维码。',cardTitle:'你的卡片',
  unavailable:'USDT 发送暂时不可用。',protectedUnavailable:'受保护付款暂时不可用。请选择未受保护的付款，或稍后再试。',
  errors:{invalid_request:'请求无效。请检查金额和邮箱。',usdt_unavailable:'USDT 服务不可用，请稍后再试。',too_many_cards:'创建的卡片过多，请一小时后再试。',chain_unavailable:'网络暂时无法访问，请稍后重试。',invalid_service_answer:'服务返回无效。未使用任何卡片。',forbidden:'被拒绝：此记录与该卡片不匹配，或已付款。',claim_not_found:'未找到卡片。',invalid_address:'地址无效。',claim_expired:'此卡片已过期。',card_cancelled:'此卡片已取消。'},
  sentTitle:'我的 USDT 发送',sentIntro:'收件人提供地址后，你扫描一次即可付款。这里显示每张卡片的进度。跟踪记录保存在此设备上。',none:'暂无 USDT 发送记录。',stAwaiting:'等待收件人提供地址',stReady:'可以付款',stPaid:'已付款',stCancelled:'已取消',stExpired:'已过期',late:'收件人超过 24 小时未提供地址',
  payTitle:'在 {chain} 上支付 {amount} USDT',payHelp:'用你的钱包扫描。它会向收件人发送正好 {amount} USDT，并像任何转账一样自行加上网络费用。',payTo:'收件人地址',copyAddress:'复制地址',copyAmount:'复制金额',copied:'已复制',openWallet:'在我的钱包中打开',paying:'等待付款…付款确认后此页面会立即发现。',
  cancelCard:'取消此卡片',cancelConfirm:'取消此卡片？收件人将无法再提供地址。',copyProof:'复制凭证',proofCopied:'凭证已复制',forget:'删除此记录？',view:'查看交易',addCard:'将卡片添加到邮件',to:'预期收件人',refresh:'刷新',protectedBadge:'受保护',
  proofTitle:'PhoenixMail · USDT 卡片',proofAmount:'金额',proofDate:'日期',proofCard:'卡片',proofState:'状态',proofTx:'交易',proofChecked:'检查于',
  walletTitle:'我的 USDT 钱包',walletIntro:'你的 USDT 地址，保存在此设备上。它们用于接收付款请求，并帮助识别你的付款。同一地址可用于 Ethereum 和 Polygon。',walletEmpty:'尚未保存 USDT 地址。',walletAdd:'添加 USDT 地址',walletName:'名称（可选）',walletAddButton:'添加',walletInvalid:'地址无效：请检查每个字符。',walletDelete:'删除',walletBoth:'两个网络',walletUse:'此地址用于',walletOnly:'仅 {n}',walletChain:'网络',walletAddress:'USDT 地址'}
};
// ---- Keccak-256 (only to check EIP-55 address checksums in the browser; mirrors the Go implementation) ----
const M64=(1n<<64n)-1n;
const RC=[0x0000000000000001n,0x0000000000008082n,0x800000000000808an,0x8000000080008000n,0x000000000000808bn,0x0000000080000001n,0x8000000080008081n,0x8000000000008009n,0x000000000000008an,0x0000000000000088n,0x0000000080008009n,0x000000008000000an,0x000000008000808bn,0x800000000000008bn,0x8000000000008089n,0x8000000000008003n,0x8000000000008002n,0x8000000000000080n,0x000000000000800an,0x800000008000000an,0x8000000080008081n,0x8000000000008080n,0x0000000080000001n,0x8000000080008008n];
const ROT=[1,3,6,10,15,21,28,36,45,55,2,14,27,41,56,8,25,43,62,18,39,61,20,44],PI=[10,7,11,17,18,3,5,16,8,21,24,4,15,23,19,13,12,2,20,14,22,9,6,1];
const rotl=(x,n)=>((x<<BigInt(n))|(x>>BigInt(64-n)))&M64;
function keccakF(a){const bc=new Array(5);for(let round=0;round<24;round++){
 for(let i=0;i<5;i++)bc[i]=a[i]^a[i+5]^a[i+10]^a[i+15]^a[i+20];
 for(let i=0;i<5;i++){const t=bc[(i+4)%5]^rotl(bc[(i+1)%5],1);for(let j=0;j<25;j+=5)a[j+i]^=t}
 let t=a[1];for(let i=0;i<24;i++){const j=PI[i];bc[0]=a[j];a[j]=rotl(t,ROT[i]);t=bc[0]}
 for(let j=0;j<25;j+=5){for(let i=0;i<5;i++)bc[i]=a[j+i];for(let i=0;i<5;i++)a[j+i]^=(~bc[(i+1)%5]&M64)&bc[(i+2)%5]}
 a[0]^=RC[round]}}
function keccak256(bytes){const rate=136,state=new Array(25).fill(0n),padded=new Uint8Array(Math.ceil((bytes.length+1)/rate)*rate);padded.set(bytes);padded[bytes.length]^=0x01;padded[padded.length-1]^=0x80;
 for(let off=0;off<padded.length;off+=rate){for(let i=0;i<rate/8;i++){let lane=0n;for(let k=7;k>=0;k--)lane=(lane<<8n)|BigInt(padded[off+i*8+k]);state[i]^=lane}keccakF(state)}
 const out=new Uint8Array(32);for(let i=0;i<4;i++){let lane=state[i];for(let k=0;k<8;k++){out[i*8+k]=Number(lane&0xffn);lane>>=8n}}return out}
const toHex=b=>Array.from(b,x=>x.toString(16).padStart(2,'0')).join('');
function checksum(address){const lower=String(address).slice(2).toLowerCase(),hash=toHex(keccak256(new TextEncoder().encode(lower)));let out='0x';for(let i=0;i<40;i++)out+=(/[a-f]/.test(lower[i])&&parseInt(hash[i],16)>=8)?lower[i].toUpperCase():lower[i];return out}
function addressValid(address){const a=String(address||'').trim();if(!/^0x[0-9a-fA-F]{40}$/.test(a)||/^0x0{40}$/.test(a))return false;const body=a.slice(2);return body===body.toLowerCase()||body===body.toUpperCase()||checksum(a)===a}
const normalize=a=>addressValid(a)?checksum(String(a).trim()):'';
const shortAddress=a=>a.slice(0,6)+' … '+a.slice(-4);
const amountOk=s=>/^(0|[1-9][0-9]{0,6})(\.[0-9]{1,6})?$/.test(String(s).trim())&&Number(s)>=0.01;
function validateCashlink(c){
 if(!c||c.version!==2||!/^[0-9a-f]{32}$/.test(c.id)||!CHAINS[c.asset]||!amountOk(c.amount)||Number(c.amount)>1000||c.protected!==true||typeof c.message!=='string'||c.message.length>140)throw Error('invalid_card');
 let u;try{u=new URL(c.claimURL)}catch{throw Error('invalid_card')}
 const local=['localhost','127.0.0.1','[::1]'].includes(u.hostname);
 if((u.protocol!=='https:'&&!(u.protocol==='http:'&&local))||u.username||(u.pathname!==''&&u.pathname!=='/')||u.search!==''||!/^#c=[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}$/.test(u.hash))throw Error('invalid_card');
 return c}
function validateClaim(c){
 if(c&&c.version===2)return validateCashlink(c);
 if(!c||c.version!==1||!/^[0-9a-f]{48}$/.test(c.id)||!CHAINS[c.asset]||!amountOk(c.amount)||Number(c.amount)>1000||typeof c.protected!=='boolean'||typeof c.message!=='string'||c.message.length>140)throw Error('invalid_card');
 let u;try{u=new URL(c.claimURL)}catch{throw Error('invalid_card')}
 const local=['localhost','127.0.0.1','[::1]'].includes(u.hostname);
 if((u.protocol!=='https:'&&!(u.protocol==='http:'&&local))||u.username||u.hash||(u.pathname!==''&&u.pathname!=='/')||u.search!=='?claim='+c.id)throw Error('invalid_card');
 return c}
function validateRequest(r){
 if(!r||r.version!==1||!/^[0-9a-f]{24}$/.test(r.id)||!CHAINS[r.asset]||!addressValid(r.address)||r.address!==checksum(r.address)||!amountOk(r.amount)||!Number.isSafeInteger(r.units)||typeof r.reference!=='string'||typeof r.link!=='string'||!r.link.startsWith('ethereum:0x'))throw Error('invalid_request');
 return r}
// ethereum:0x… links are only ever used as the target of an "open in my wallet" link, after being rebuilt from validated fields.
const safeWalletLink=l=>/^ethereum:0x[0-9a-fA-F]{40}@(1|137)(\/transfer\?address=0x[0-9a-fA-F]{40}&uint256=\d{1,15}|\?value=\d{1,30})$/.test(String(l))?l:'';
// The words name the stablecoin of the asset in use (USDC on Polygon, USDT elsewhere); a generic list says USDC/USDT.
const sw=(o,sym)=>typeof o==='string'?o.replace(/USDT/g,sym).replace(/PhoenixMail/g,root.PhoenixBrandName||'PhoenixMail'):Array.isArray(o)?o.map(x=>sw(x,sym)):o&&typeof o==='object'&&!(o instanceof RegExp)?Object.fromEntries(Object.entries(o).map(([k,v])=>[k,sw(v,sym)])):o;
const fmt=(s,v)=>String(s).replace(/\{(\w+)\}/g,(m,k)=>v[k]??m);
function el(tag,props,...kids){const n=document.createElement(tag);if(props)for(const[k,v]of Object.entries(props)){if(k==='class')n.className=v;else if(k==='text')n.textContent=v;else if(k in n&&k!=='list')n[k]=v;else n.setAttribute(k,v)}for(const k of kids)if(k!=null)n.append(k);return n}
const EXTRA={
 fr:{recipientMismatch:'Le destinataire doit être l’unique adresse « À » (sans Cc/Cci).',openMessageFirst:'Ouvre d’abord le message auquel ajouter la carte.',invalidAmount:'Montant invalide (0,01 à 1 000 USDT, 6 décimales au maximum).',invalidRequestAmount:'Montant invalide (au moins 0,01 USDT, 6 décimales au maximum).',chooseMethod:'Choisis : protégé ou non protégé.',noAddress:'Choisis ou saisis ton adresse USDT.',checking:'Vérification…',invalidEmail:'Courriel du destinataire invalide.'},
 en:{recipientMismatch:'The recipient must be the single To address (no Cc/Bcc).',openMessageFirst:'Open the message to add the card to first.',invalidAmount:'Invalid amount (0.01 to 1,000 USDT, up to 6 decimals).',invalidRequestAmount:'Invalid amount (at least 0.01 USDT, up to 6 decimals).',chooseMethod:'Choose protected or non-protected.',noAddress:'Choose or enter your USDT address.',checking:'Checking…',invalidEmail:'Invalid recipient email.'},
 es:{recipientMismatch:'El destinatario debe ser la única dirección «Para» (sin Cc/Cco).',openMessageFirst:'Abre primero el mensaje al que añadir la tarjeta.',invalidAmount:'Importe no válido (0,01 a 1.000 USDT, hasta 6 decimales).',invalidRequestAmount:'Importe no válido (al menos 0,01 USDT, hasta 6 decimales).',chooseMethod:'Elige protegido o sin protección.',noAddress:'Elige o introduce tu dirección USDT.',checking:'Comprobando…',invalidEmail:'Correo del destinatario no válido.'},
 de:{recipientMismatch:'Der Empfänger muss die einzige „An“-Adresse sein (ohne Cc/Bcc).',openMessageFirst:'Öffne zuerst die Nachricht, zu der die Karte hinzugefügt werden soll.',invalidAmount:'Ungültiger Betrag (0,01 bis 1.000 USDT, höchstens 6 Dezimalstellen).',invalidRequestAmount:'Ungültiger Betrag (mindestens 0,01 USDT, höchstens 6 Dezimalstellen).',chooseMethod:'Wähle geschützt oder ungeschützt.',noAddress:'Wähle oder gib deine USDT-Adresse ein.',checking:'Wird geprüft…',invalidEmail:'Ungültige E-Mail des Empfängers.'},
 zh:{recipientMismatch:'收件人必须是唯一的“收件人”地址（不能有抄送/密送）。',openMessageFirst:'请先打开要添加卡片的邮件。',invalidAmount:'金额无效（0.01 至 1,000 USDT，最多 6 位小数）。',invalidRequestAmount:'金额无效（至少 0.01 USDT，最多 6 位小数）。',chooseMethod:'请选择受保护或未受保护。',noAddress:'请选择或输入你的 USDT 地址。',checking:'检查中…',invalidEmail:'收件人邮箱无效。'}
};

function create(c){
 const o=c.o,storage=c.storage;
 // Marked Wire hook: only the embedding host supplies these copy overrides.
 const L0=()=>({...(WORDS[o.language()]||WORDS.en),...(root.PhoenixWordOverrides?.[o.language()]||{})}),L=sym=>sw(L0(),sym||'USDC/USDT'),X=()=>EXTRA[o.language()]||EXTRA.en;
 const CL=root.PhoenixUSDTCashlink?root.PhoenixUSDTCashlink.create(o):null;
 const isCL=e=>!!(e&&e.card&&e.card.version===2);
 const unitsText=u=>(Number(u)/1e6).toFixed(6).replace(/0+$/,'').replace(/\.$/,'');
 const WALLET='phoenixmail.usdt.wallet.v1',SENT='phoenixmail.usdt.sent.v2';
 const read=key=>{try{const v=JSON.parse(storage.getItem(key)||'[]');return Array.isArray(v)?v:[]}catch{return[]}};
 const write=(key,list,limit=50)=>{const s=JSON.stringify(list.slice(0,limit));storage.setItem(key,s);if(storage.getItem(key)!==s)throw Error('storage')};
 const when=ms=>{try{return new Date(ms).toLocaleString(o.language())}catch{return String(ms)}};

 // ---- the sender's own USDT addresses ("My wallet" in the settings) ----
 const wallets=()=>read(WALLET).filter(e=>e&&addressValid(e.address)&&['ethereum','polygon'].includes(e.chain)).map(e=>({address:normalize(e.address),chain:e.chain,label:String(e.label||'').slice(0,40),at:Number(e.at)||0}));
 function addWallet(address,chains,label){const a=normalize(address);if(!a)return false;const list=wallets();for(const chain of [...chains].reverse()){const i=list.findIndex(e=>e.address===a&&e.chain===chain);if(i>=0){if(label&&!list[i].label)list[i].label=label}else list.unshift({address:a,chain,label:String(label||'').slice(0,40),at:Date.now()})}try{write(WALLET,list)}catch{return false}return true}
 const removeWallet=(address,chain)=>write(WALLET,wallets().filter(e=>!(e.address===address&&e.chain===chain)));

 // ---- calls to PhoenixMail's server ----
 async function api(path,data){
  const response=await (o.fetch||root.fetch)(path,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(data)});
  let body;try{body=await response.json()}catch{const e=Error('usdt_unavailable');e.code='usdt_unavailable';throw e}
  if(!response.ok||!body||typeof body!=='object'){const e=Error(body?.error||'invalid_request');e.code=body?.error||'invalid_request';throw e}
  return body}
 const errorText=e=>{const code=e?.code||'';const t=L().errors[code]||(CL&&CL.words().errors[code]);return t||(e?.message&&!/^[a-z_]+$/.test(e.message)?e.message:L().errors.usdt_unavailable)};
 let gasOn=false,capsLoaded=false;
 async function capabilities(){try{const r=await (o.fetch||root.fetch)('/api/usdt/status',{cache:'no-store'});const cap=r.ok?await r.json():{};gasOn=cap.gas===true;return cap}catch{return{}}}
 const copy=async(text,button)=>{try{await root.navigator.clipboard.writeText(text);if(button){const old=button.textContent;button.textContent=L().copied;setTimeout(()=>{button.textContent=old},1400)}}catch{}};
  const shell=(data,heading,asset,chip)=>{const made=c.cardShell({id:data.id,amount:data.amount,pastelV2:true},heading,chip||'',{brand:CHAINS[asset].label,unit:' '+CHAINS[asset].symbol});made.box&&made.box.classList&&made.box.classList.add('usdt-card');return made} // the colour comes from the card ID, from the same twelve families as the Nimiq cards;
 const png=b64=>'data:image/png;base64,'+b64;
 const coinFile=a=>CHAINS[a]&&CHAINS[a].symbol==='USDC'?'brand/usdc-coin.png':'brand/usdt-coin.png',coinAlt=a=>CHAINS[a]&&CHAINS[a].symbol==='USDC'?'USD Coin USDC':'Tether USDT';
 const coin=a=>el('img',{class:'usdt-coin',src:coinFile(a),alt:coinAlt(a),width:64,height:64});
 function walletLink(href,label){const safe=safeWalletLink(href);if(!safe)return null;return el('a',{class:'secondary-btn usdt-open-wallet',href:safe,text:label,rel:'noopener noreferrer'})}
 const explorerOf=(chain,hash)=>/^0x[0-9a-fA-F]{64}$/.test(hash||'')?(chain.key==='ethereum'?'https://etherscan.io/tx/':'https://polygonscan.com/tx/')+hash:'';

 // ---- a payment request (the recipient of the email pays the sender's own address) ----
 function requestCard(r,qr){
  validateRequest(r);const t=L(CHAINS[r.asset].symbol),chain=CHAINS[r.asset];
  const{box,content}=shell(r,t.request,r.asset);
  if(qr)content.append(el('img',{class:'nimiq-payment-qr',src:png(qr),alt:'QR USDT'}));
  content.append(coin(r.asset),el('p',{class:'usdt-address',text:r.address}),el('p',{text:t.scan}),el('p',{class:'usdt-only',text:fmt(t.only,{chain:chain.name})}));
  if(r.reference)content.append(el('p',{text:r.reference}));
  content.append(el('small',{text:t.unpaid}));
  return box}

 // ---- sends: what this device made. The card goes in the email at once; the payment comes later, when the recipient has given an address ----
 const TERMINAL=['paid','cancelled','expired','withdrawn','settled_elsewhere'];
 const sentList=()=>read(SENT).filter(e=>{try{validateClaim(e.card);return typeof e.senderToken==='string'&&e.senderToken.length>=20&&e.senderToken.length<=64}catch{return false}}).slice(0,500);
 // A card that may still hold money, or whose refund key is still needed, is NEVER evicted to make room.
 const holdsRights=e=>isCL(e)&&!(['withdrawn','settled_elsewhere'].includes(e.state)||(['paid','cancelled','expired'].includes(e.state)&&e.checkedAt&&!(e.balanceUnits>0)&&!e.recovering));
 const saveSent=list=>{const keep=list.filter(holdsRights),rest=list.filter(e=>!holdsRights(e)).slice(0,Math.max(0,50-keep.length)),all=[...keep,...rest].sort((a,b)=>(b.at||0)-(a.at||0));write(SENT,all,all.length)};
 function updateSent(id,patch){const all=sentList(),i=all.findIndex(e=>e.card.id===id);if(i>=0){Object.assign(all[i],patch);try{saveSent(all)}catch{}}return all[i]}
 function rememberSent(made,recipient,senderAddress){const all=sentList().filter(e=>e.card.id!==made.request.id);all.unshift({card:made.request,asset:made.request.asset,senderToken:made.senderToken,senderAddress:senderAddress||'',protectedRecipient:made.request.protected?recipient:'',to:String(o.recipient?.()||recipient||'').slice(0,200),at:Date.now(),state:'awaiting_address',attached:false,attachments:made.attachments});saveSent(all)}
 function stateLabel(e){const t=L();if(isCL(e)&&CL)return CL.stateLabel(e.state);return({awaiting_address:t.stAwaiting,ready:t.stReady,paid:t.stPaid,cancelled:t.stCancelled,expired:t.stExpired})[e.state]||t.stAwaiting}
 function attachCard(rec){
  if(rec.card.protected&&o.bindOTPRecipient&&o.bindOTPRecipient(rec.protectedRecipient)===false)throw Error(X().recipientMismatch);
  if(!rec.card.protected&&rec.to&&o.openRecoveredClaim&&o.openRecoveredClaim(rec.to)===false)throw Error(X().openMessageFirst);
  o.attach(rec.attachments);updateSent(rec.card.id,{attached:true});o.status?.(L().added)}

 // ---- the window: same shell as Nimiq, USDT form ----
 function panel(ctx){
  const t=L(CHAINS[ctx.asset].symbol),chain=CHAINS[ctx.asset],{mode,panel,previewHost,view,previewTab,modal,dismiss}=ctx;
  const status=el('output',{role:'status'});
  const field=(label,input)=>{const l=el('label',{text:label});l.append(input);return l};
  const amount=el('input',{type:'text',autocomplete:'off',inputMode:'decimal',placeholder:'12.50',required:true});
  const controls=el('div',{class:'nimiq-payment-controls'});
  const submit=el('button',{type:'submit',class:'secondary-btn',text:t.create}),cancel=el('button',{type:'button',class:'secondary-btn',text:t.cancel});cancel.onclick=dismiss;controls.append(submit,cancel);
  const alive=()=>modal.isConnected;
  let addressInput=null,pickSelect=null,reference=null,message=null,protectRadio=null,openRadio=null,emailInput=null,ready={send:false,protected:false},cashlinkOn=false,senderParts=[];
  // A saved address of the sender on this chain, or a typed one; "another wallet or an exchange" means: any wallet.
  const saved=wallets().filter(e=>e.chain===chain.key);
  function addressPicker(labelText,anyLabel){
   addressInput=el('input',{type:'text',autocomplete:'off',spellcheck:false,placeholder:'0x…'});
   const typed=field(t.address,addressInput);
   if(saved.length){
    pickSelect=el('select');for(const e of saved)pickSelect.append(el('option',{value:e.address,text:(e.label?e.label+' · ':'')+shortAddress(e.address)}));pickSelect.append(el('option',{value:'',text:anyLabel}));
    typed.hidden=true;
    return[field(labelText,pickSelect),typed]}
   return[typed,el('small',{class:'nimiq-payment-how',text:t.saveHint})]}

  if(mode==='request'){
   const parts=addressPicker(t.wallet,t.another);
   if(pickSelect)pickSelect.onchange=()=>{parts[1].hidden=pickSelect.value!==''};
   reference=el('input',{type:'text',autocomplete:'off',maxLength:64});
   panel.append(...parts,field(t.amount,amount),field(t.reference,reference));
  }else{
   message=el('input',{type:'text',autocomplete:'off',maxLength:140});
   const parts=addressPicker(t.senderWallet,t.anyWallet);
   // "any wallet": nothing typed is needed, the field stays hidden when saved addresses exist
   if(pickSelect){pickSelect.onchange=()=>{parts[1].hidden=true}}
   const howEl=el('p',{class:'nimiq-payment-how',text:t.how}),hintEl=el('small',{class:'nimiq-payment-how',text:t.senderHint});senderParts=[...parts,hintEl];
   panel.append(howEl,...parts,hintEl,field(t.amount,amount),field(t.message,message));
   const method=(value,title,note)=>{const input=el('input',{type:'radio',name:'phoenixmail-usdt-method',value});const text=el('span'),strong=el('strong',{text:title});text.append(strong,el('small',{text:note}));const row=el('label',{class:'nimiq-otp-option'});row.append(input,text);return{input,row}};
   const p=method('protected',t.protectedTitle,t.protectedNote),n=method('open',t.openTitle,t.openNote);
   protectRadio=p.input;openRadio=n.input;protectRadio.disabled=true;
   const emailRow=el('label',{text:t.recipient});emailRow.hidden=true;emailInput=el('input',{type:'email',autocomplete:'email',value:o.recipient?.()||''});emailRow.append(emailInput);
   const protectedHint=el('small',{class:'nimiq-payment-how',text:t.protectedChosen});protectedHint.hidden=true;
   panel.append(p.row,emailRow,protectedHint,n.row);
   const refreshMethod=()=>{emailRow.hidden=!protectRadio.checked;protectedHint.hidden=!protectRadio.checked;emailInput.required=protectRadio.checked};
   protectRadio.onchange=()=>{if(protectRadio.checked)openRadio.checked=false;refreshMethod()};
   openRadio.onchange=()=>{if(openRadio.checked)protectRadio.checked=false;refreshMethod()};
   capabilities().then(cap=>{ready={send:cap.send===true,protected:cap.protected===true};
    // Cashlink mode: the card holds the money; every card is protected; the sender needs no wallet address to create it.
    if(cap.cashlink===true&&CL&&root.PhoenixCashlink){cashlinkOn=true;ready.send=true;ready.protected=true;n.row.hidden=true;for(const x of senderParts)x.hidden=true;howEl.textContent=CL.words().how}
    protectRadio.disabled=!ready.protected;if(ready.protected&&!openRadio.checked)protectRadio.checked=true;refreshMethod();
    if(!ready.send){status.textContent=t.unavailable;submit.disabled=true}else if(!ready.protected){status.textContent=t.protectedUnavailable}});
  }
  panel.append(controls,status);
  if(mode==='send'&&o.openWallet){const sends=el('button',{type:'button',class:'secondary-btn',text:t.openSends});sends.onclick=()=>{dismiss();o.openWallet()};const extra=el('div',{class:'usdt-panel-actions'});extra.append(sends);panel.append(extra);ctx.actionsRow=extra}

  let locked=false;
  const lock=on=>{locked=on;for(const i of panel.querySelectorAll('input,select'))i.disabled=on;submit.disabled=on;for(const b of ctx.modeButtons)b.disabled=on;if(!on&&protectRadio)protectRadio.disabled=!ready.protected};
  // the address the sender chose: a saved one, a typed one (checked), or none (any wallet)
  const chosenAddress=()=>{if(pickSelect&&pickSelect.value)return{ok:true,value:pickSelect.value};const typed=addressInput.value.trim();if(!typed)return{ok:true,value:''};const n=normalize(typed);return n?{ok:true,value:n,isNew:true}:{ok:false}};

  panel.onsubmit=async event=>{
   event.preventDefault();if(locked)return;status.textContent='';
   try{
    const chosen=chosenAddress();
    if(!chosen.ok){status.textContent=t.walletInvalid;return}
    if(mode==='request'){
     if(!amountOk(amount.value)){status.textContent=X().invalidRequestAmount;return}
     if(!chosen.value){status.textContent=X().noAddress;return}
     submit.disabled=true;
     const made=await api('/api/usdt/payment-request',{asset:ctx.asset,address:chosen.value,amount:amount.value.trim(),reference:reference.value.trim()});if(!alive())return;
     validateRequest(made.request);
     if(chosen.isNew)addWallet(chosen.value,[chain.key],'');
     const qr=(made.attachments.find(a=>a.type==='image/png')||{}).data;
     const preview=requestCard(made.request,qr);
     const add=el('button',{type:'button',class:'secondary-btn',text:t.addToEmail});
     add.onclick=()=>{o.attach(made.attachments);o.status?.(t.added);dismiss()};
     const back=el('button',{type:'button',class:'secondary-btn nimiq-payment-edit',text:t.edit});back.onclick=()=>view('form');
     preview.append(add);previewHost.replaceChildren(preview,back);previewTab.disabled=false;view('preview');
     return}
    // send: create the card and put it in the email; the payment comes later
    if(!amountOk(amount.value)||Number(amount.value)>1000){status.textContent=X().invalidAmount;return}
    if(cashlinkOn){
     const to=emailInput.value.trim();if(!emailInput.checkValidity()||!to){status.textContent=X().invalidEmail;return}
     if(o.bindOTPRecipient&&o.bindOTPRecipient(to)===false){status.textContent=X().recipientMismatch;return}
     lock(true);status.textContent=X().checking;
     const entry=await CL.createCard({asset:ctx.asset,amount:amount.value.trim(),message:message.value.trim(),recipient:to,senderEmail:String(o.senderEmail?.()||'')});if(!alive())return;
     if(entry.asset!==ctx.asset)throw Error('invalid_card');
     validateClaim(entry.card);rememberCashlink(entry);status.textContent='';showCashlink(entry,ctx);return}
    if(!protectRadio.checked&&!openRadio.checked){status.textContent=X().chooseMethod;return}
    const isProtected=protectRadio.checked;let recipient='';
    if(isProtected){recipient=emailInput.value.trim();if(!emailInput.checkValidity()||!recipient){status.textContent=X().invalidEmail;return}
     if(o.bindOTPRecipient&&o.bindOTPRecipient(recipient)===false){status.textContent=X().recipientMismatch;return}}
    lock(true);status.textContent=X().checking;
    const made=await api('/api/usdt/card',{asset:ctx.asset,amount:amount.value.trim(),message:message.value.trim(),protected:isProtected,recipient,senderAddress:chosen.value,senderEmail:String(o.senderEmail?.()||''),lang:o.language()});if(!alive())return;
    validateClaim(made.request);
    if(made.request.asset!==ctx.asset||made.request.protected!==isProtected)throw Error('invalid_card');
    if(chosen.isNew)addWallet(chosen.value,[chain.key],'');
    rememberSent(made,recipient,chosen.value);
    const rec=sentList().find(e=>e.card.id===made.request.id);
    status.textContent='';
    // the card as the recipient will see it, and the way forward
    const{box,content}=shell(rec.card,t.cardTitle,ctx.asset,rec.card.protected?t.protectedTitle:'');
    content.append(coin(rec.asset),el('p',{text:rec.card.protected?t.protectedChosen:t.openNote}));if(rec.card.message)content.append(el('p',{text:rec.card.message}));
    const note=el('output',{role:'status',class:'usdt-note'});content.append(note);
    previewHost.replaceChildren(box);previewTab.disabled=false;view('preview');
    try{attachCard(rec);note.textContent=t.created}catch(e){note.textContent=e?.message||t.errors.usdt_unavailable}
   }catch(e){status.textContent=errorText(e);lock(false)}
   finally{if(mode==='request')submit.disabled=false}
  };
 }

 // ---- settings: My wallet (USDT addresses of the sender) ----
 function mountWallet(host){
  const t=L();host.textContent='';host.classList.add('usdt-wallet');
  const head=el('div',{class:'settings-section-head'}),box=el('div');box.append(el('h3',{text:t.walletTitle}),el('p',{text:t.walletIntro}));head.append(box);host.append(head);
  const list=el('div',{class:'nimiq-wallet-list'}),saved=wallets();
  if(!saved.length)list.append(el('p',{class:'nimiq-wallet-empty',text:t.walletEmpty}));
  for(const e of saved){
   const row=el('div',{class:'nimiq-wallet-row'});
   const name=el('input',{type:'text',maxLength:40,value:e.label,placeholder:shortAddress(e.address)});name.setAttribute('aria-label',t.walletName);
   name.onchange=()=>{const all=wallets(),i=all.findIndex(x=>x.address===e.address&&x.chain===e.chain);if(i>=0){all[i].label=name.value.trim().slice(0,40);try{write(WALLET,all)}catch{}}};
   const info=el('div',{class:'nimiq-wallet-info'});const addr=el('small',{class:'usdt-address',text:e.address});
   info.append(name,addr,el('span',{class:'nimiq-payment-network',text:Object.values(CHAINS).find(x=>x.key===e.chain).label}));
   const del=el('button',{type:'button',class:'secondary-btn',text:'✕',title:t.walletDelete});del.setAttribute('aria-label',t.walletDelete);del.onclick=()=>{removeWallet(e.address,e.chain);mountWallet(host)};
   const icon=el('img',{class:'usdt-coin usdt-coin-small',src:'brand/usdt-coin.png',alt:'',width:40,height:40});
   row.append(icon,info,del);list.append(row)}
  host.append(list);
  const form=el('form',{class:'nimiq-wallet-add'}),title=el('strong',{text:t.walletAdd});
  const addr=el('input',{type:'text',autocomplete:'off',spellcheck:false,placeholder:'0x…'});addr.setAttribute('aria-label',t.walletAddress);
  const name=el('input',{type:'text',maxLength:40,placeholder:t.walletName});name.setAttribute('aria-label',t.walletName);
  const net=el('select',{id:'usdt-wallet-net'});for(const[v,l]of[['both',t.walletBoth],['ethereum',t.walletOnly.replace('{n}','Ethereum')],['polygon',t.walletOnly.replace('{n}','Polygon')]])net.append(el('option',{value:v,text:l}));
  const netRow=el('label',{class:'usdt-wallet-net',for:'usdt-wallet-net'});netRow.append(el('span',{text:t.walletUse}),net); // a visible label: the list alone said nothing
  const add=el('button',{type:'submit',class:'secondary-btn',text:t.walletAddButton}),status=el('output',{role:'status'});
  form.onsubmit=e=>{e.preventDefault();if(!addWallet(addr.value,net.value==='both'?['ethereum','polygon']:[net.value],name.value.trim())){status.textContent=t.walletInvalid;return}mountWallet(host)};
  form.append(title,addr,name,netRow,add,status);host.append(form)}

 // ---- cashlinks: the card holds the money; the sender pays it once, from any wallet ----
 function rememberCashlink(entry){const all=sentList().filter(e=>e.card.id!==entry.card.id);all.unshift(entry);try{saveSent(all)}catch{const e=Error('local_storage');e.code='local_storage';throw e}}
 // What the sender sees until the card is paid: one QR for amount + fee reserve, and a way to get the gas of that one payment.
 function fundingPanel(entry,onChange,watch){
  const t=L(CHAINS[entry.asset].symbol),w=CL.words(CHAINS[entry.asset].symbol),chain=CHAINS[entry.asset],cl=entry.cashlink,pay=cl.pay,coinName=CL.gasCoin(chain.key);
  const inCard=!!(watch&&watch.inCard); // on the pastel card: only what is needed to pay; the rest lives in the window
  const box=el('div',{class:inCard?'usdt-pay usdt-pay-card':'usdt-pay'});
  box.append(el('h4',{text:CL.fmt(w.fundTitle,{amount:pay.amount,chain:chain.name})}),el('p',{text:CL.fmt(w.fundHelp,{fee:unitsText(cl.feeUnits)})}));
  if(pay.qr)box.append(el('img',{class:'nimiq-payment-qr',src:png(pay.qr),alt:'QR'}));
  const to=el('div',{class:'usdt-addressrow'});
  to.append(el('code',{class:'usdt-address',text:inCard?pay.recipient:shortAddress(pay.recipient),title:pay.recipient}));
  const cpA=el('button',{type:'button',class:'secondary-btn',text:t.copyAddress});cpA.onclick=()=>copy(pay.recipient,cpA);to.append(cpA);
  box.append(el('small',{class:'usdt-where',text:w.whereMoney}),el('small',{text:t.payTo}),to);
  if(!inCard){const row=el('div',{class:'usdt-actions'}),cpM=el('button',{type:'button',class:'secondary-btn',text:t.copyAmount});cpM.onclick=()=>copy(pay.amount,cpM);row.append(cpM);const open=walletLink(pay.link,t.openWallet);if(open)row.append(open);box.append(row)}
  // The gas of the sender's payment. In PhoenixMail it is paid by the sender's own wallet (a plain note). Where the free gas is
  // switched on (PhoenixWire) a light shows who pays: green, PhoenixWire (entering a wallet address sends the gas by itself);
  // red, the sender, because free gas is unavailable or because he chose to pay with his own coin (the box under the light).
  if(!gasOn){if(!inCard)box.append(el('small',{class:'usdt-gasnote',text:CL.fmt(w.ownGasNote,{gas:coinName})}))}
  else{
   const gasBox=el('div',{class:'usdt-gas'}),gasNote=el('output',{role:'status'});
   const led=el('span',{class:'usdt-led wait',role:'img'}),ledText=el('strong',{text:''}),ledRow=el('p',{class:'usdt-ledrow'});ledRow.append(led,ledText);
   const own=el('input',{type:'checkbox',id:'usdt-own-'+entry.card.id}),ownRow=el('label',{class:'usdt-own',for:own.id});ownRow.append(own,el('span',{text:CL.fmt(w.ownGas,{gas:coinName})}));ownRow.hidden=true;
   const saved=wallets().filter(e=>e.chain===chain.key);
   const addr=el('input',{type:'text',autocomplete:'off',spellcheck:false,placeholder:w.myAddress,value:saved[0]?saved[0].address:''}),addrRow=el('div',{class:'usdt-gasaddr'});addrRow.hidden=true;
   addrRow.append(el('small',{text:CL.fmt(w.gasAuto,{gas:coinName})}),addr);
   let status={available:false,reason:'wait'},asked='';
   const paint=()=>{const green=status.available&&!own.checked;led.className='usdt-led '+(status.reason==='wait'?'wait':green?'green':'red');led.setAttribute('aria-label',green?'green':'red');
    ledText.textContent=status.reason==='wait'?'':green?w.ledGreen:(own.checked&&status.available?CL.fmt(w.ledOwn,{gas:coinName}):CL.fmt(w.ledRed,{gas:coinName}));
    ownRow.hidden=!status.available;addrRow.hidden=!green};
   const send=async()=>{const a=normalize(addr.value);if(!a||a===asked||own.checked||!status.available)return;asked=a;gasNote.textContent=w.gasSending;
    try{const r=await CL.gas(entry,a);gasNote.textContent=r.already?w.gasAlready:w.gasSent;addWallet(a,[chain.key],'')} // no re-render here: a re-render would ask again
    catch(e){asked='';gasNote.textContent=e?.code==='gas_not_needed'?CL.fmt(w.gasNotNeeded,{gas:coinName}):errorText(e)}};
   own.onchange=()=>{gasNote.textContent='';paint()};addr.oninput=send;addr.onchange=send;
   CL.gasStatus(entry.asset).then(r=>{status={available:r.available===true,reason:String(r.reason||'off')};paint()}).catch(()=>{status={available:false,reason:'off'};paint()});
   gasBox.append(ledRow,ownRow,addrRow,gasNote);box.append(gasBox)}
  const waiting=el('small',{class:'usdt-waiting',text:inCard?'':t.paying}); // on the card, silent until there is something to say
  box.append(waiting);if(!inCard)box.append(el('small',{text:w.keyNote}));
  // On the screen that just made the card: watch it, say when the payment is seen and when the card is funded, offer to close.
  if(watch){let stopped=false;
   // the Close button is there as soon as the payment is seen: nobody has to guess that the window can be closed
   const close=el('button',{type:'button',class:'secondary-btn usdt-close',text:w.close});close.onclick=()=>{watch.dismiss&&watch.dismiss()};
   const funded=()=>{const ok=el('p',{class:'usdt-funded',role:'status',text:'✓ '+CL.fmt(w.fundedMsg,{email:entry.protectedRecipient||entry.to||''})});box.replaceChildren(ok,close);
    // the window closes by itself a moment later: nobody has to guess that it can be closed
    setTimeout(()=>{if(box.isConnected&&watch.dismiss)watch.dismiss()},watch.closeAfter??4000)};
   const tick=async()=>{if(stopped||!box.isConnected)return;
    try{const st=await CL.statusOf(entry.card.id);updateSent(entry.card.id,{state:st.state||entry.state,balanceUnits:st.balanceUnits||0,checkedAt:Date.now()});
     if(['ready','paid','verifying'].includes(st.state)){stopped=true;funded();return}
     if((st.balanceUnits||0)>=entry.cashlink.units+entry.cashlink.feeUnits){waiting.textContent=w.confirmingMsg;if(!close.isConnected)box.append(close)}}catch{}
    if(!stopped)setTimeout(tick,watch.every||6000)};
   setTimeout(tick,watch.first||3000)}
  return box}
 function showCashlink(entry,ctx){
  // Everything is on the one pastel card, like a Nimiq cashlink: the coin, the QR code to pay, the address, the status.
  const t=L(CHAINS[entry.asset].symbol),{previewHost,previewTab,view}=ctx;
  const{box,content}=shell(entry.card,t.cardTitle,entry.asset,t.protectedTitle);
  content.append(coin(entry.asset));if(entry.card.message)content.append(el('p',{text:entry.card.message}));
  content.append(fundingPanel(entry,null,{dismiss:ctx.dismiss,inCard:true}));
  const note=el('output',{role:'status',class:'usdt-note'});content.append(note);
  previewHost.replaceChildren(box);previewTab.disabled=false;view('preview');
  // "Open in my wallet" sits beside "My sends", in the same format, in the form panel
  if(ctx.actionsRow){for(const old of ctx.actionsRow.querySelectorAll('.usdt-open-wallet'))old.remove();const wl=walletLink(entry.cashlink.pay.link,t.openWallet);if(wl)ctx.actionsRow.append(wl)}
  try{attachCard(entry);note.textContent=t.added}catch(e){note.textContent=e?.message||t.errors.usdt_unavailable}}

 // ---- settings: My USDT sends ----
 function proofText(e){const t=L(CHAINS[e.asset].symbol),chain=CHAINS[e.asset];const lines=[t.proofTitle,t.proofAmount+' : '+e.card.amount+' '+chain.label,t.proofDate+' : '+when(e.at),t.proofCard+' : '+e.card.id,t.proofState+' : '+stateLabel(e)];if(e.txHash)lines.push(t.proofTx+' : '+e.txHash);lines.push(t.proofChecked+' '+when(e.checkedAt||Date.now()));return lines.join('\n')}
 let sentTimer=null;
 async function refreshSent(host,force){
  const now=Date.now();
  for(const e of sentList()){
   const watch=isCL(e)&&CL&&(!TERMINAL.includes(e.state)||force||e.balanceUnits>0||e.recovering);if(TERMINAL.includes(e.state)&&!watch)continue;if(!force&&now-(e.checkedAt||0)<20000)continue;
   try{if(isCL(e)&&CL){const s=await CL.statusOf(e.card.id);updateSent(e.card.id,{state:s.state||e.state,balanceUnits:s.balanceUnits||0,recovering:!!s.recovering,txHash:s.txHash||e.txHash||'',txKind:s.txKind||e.txKind||'',stale:!!s.stale,checkedAt:Date.now()})}else{const s=await api('/api/usdt/card-status',{id:e.card.id});updateSent(e.card.id,{state:s.state||e.state,pay:s.pay||null,txHash:s.txHash||e.txHash||'',checkedAt:Date.now()})}}catch{}}
  if(host.isConnected){renderSent(host,false);clearTimeout(sentTimer);if(sentList().some(e=>!TERMINAL.includes(e.state)||(isCL(e)&&(e.balanceUnits>0||e.recovering))))sentTimer=setTimeout(()=>{if(host.isConnected)refreshSent(host,true)},30000)}}
 // The one payment: a USDT transfer straight to the recipient's address. The wallet adds the network fee by itself.
 function payPanel(e){
  const t=L(CHAINS[e.asset].symbol),chain=CHAINS[e.asset],pay=e.pay;
  const box=el('div',{class:'usdt-pay'});
  box.append(el('h4',{text:fmt(t.payTitle,{amount:pay.amount,chain:chain.name})}),el('p',{text:fmt(t.payHelp,{amount:pay.amount})}));
  if(pay.qr)box.append(el('img',{class:'nimiq-payment-qr',src:png(pay.qr),alt:'QR'}));
  const to=el('div',{class:'usdt-addressrow'});
  to.append(el('code',{class:'usdt-address',text:shortAddress(pay.recipient),title:pay.recipient}));
  const cpA=el('button',{type:'button',class:'secondary-btn',text:t.copyAddress});cpA.onclick=()=>copy(pay.recipient,cpA);to.append(cpA);
  const row=el('div',{class:'usdt-actions'}),cpM=el('button',{type:'button',class:'secondary-btn',text:t.copyAmount});cpM.onclick=()=>copy(pay.amount,cpM);row.append(cpM);
  const open=walletLink(pay.link,t.openWallet);if(open)row.append(open);
  box.append(el('small',{text:t.payTo}),to,row,el('small',{class:'usdt-waiting',text:t.paying}));
  return box}
 function renderSent(host,check=true){
  if(CL&&!capsLoaded){capsLoaded=true;capabilities().then(()=>{if(host.isConnected)renderSent(host,false)})} // learn whether the gas light is on
  const t=L();host.textContent='';host.className='nimiq-sent usdt-sent';
  const head=el('div',{class:'nimiq-sent-head'}),refresh=el('button',{type:'button',class:'secondary-btn',text:t.refresh});refresh.onclick=()=>refreshSent(host,true);head.append(el('h3',{text:t.sentTitle}),refresh);host.append(head,el('p',{text:t.sentIntro}));
  if(CL&&sentList().some(isCL)){const ex=el('button',{type:'button',class:'secondary-btn',text:CL.words().exportKeys});ex.onclick=async()=>{try{await root.navigator.clipboard.writeText(JSON.stringify(sentList().filter(isCL).map(e=>({id:e.card.id,asset:e.asset,link:e.card.claimURL,senderToken:e.senderToken,safe:e.cashlink&&e.cashlink.safe})),null,1));ex.textContent=CL.words().keysCopied}catch{}};host.append(ex)}
  const list=sentList();if(!list.length){host.append(el('p',{class:'nimiq-wallet-empty',text:t.none}));return}
  for(const e of list){
   const chain=CHAINS[e.asset],settled=TERMINAL.includes(e.state),late=e.state==='awaiting_address'&&Date.now()-e.at>864e5;
   const row=el('div',{class:'nimiq-wallet-row nimiq-sent-row'});if(late)row.dataset.late='true';
   const info=el('div',{class:'nimiq-wallet-info'});
   info.append(el('strong',{text:e.card.amount+' '+chain.label+(e.card.message?' · '+e.card.message:'')}),el('small',{text:when(e.at)+(e.to?' · '+t.to+' : '+e.to:'')}));
   const state=el('span',{class:'nimiq-payment-network nimiq-sent-state',text:stateLabel(e)});state.dataset.state=e.state;info.append(state);
   if(e.card.protected)info.append(el('span',{class:'nimiq-payment-network',text:t.protectedBadge}));
   const link=explorerOf(chain,e.txHash);if(link)info.append(el('a',{href:link,target:'_blank',rel:'noopener noreferrer',text:t.view+' →'}));
   if(late)info.append(el('small',{class:'nimiq-sent-late',text:t.late}));
   const actions=el('div',{class:'nimiq-sent-actions'}),message=el('small',{role:'alert'});
   if(!e.attached&&!settled){const add=el('button',{type:'button',class:'secondary-btn',text:t.addCard});add.onclick=()=>{try{attachCard(e);renderSent(host,false)}catch(err){message.textContent=err?.message||''}};actions.append(add)}
   if(!settled){const cancel=el('button',{type:'button',class:'secondary-btn',text:t.cancelCard});cancel.onclick=async()=>{if(root.confirm&&!root.confirm(t.cancelConfirm))return;try{const r=isCL(e)&&CL?await CL.cancel(e.card.id,e.senderToken):await api('/api/usdt/card-cancel',{id:e.card.id,senderToken:e.senderToken});updateSent(e.card.id,{state:r.state||'cancelled',pay:null});renderSent(host,false)}catch(err){message.textContent=errorText(err)}};actions.append(cancel)}
   if(e.state==='paid'||e.txHash){const cp=el('button',{type:'button',class:'secondary-btn',text:t.copyProof});cp.onclick=async()=>{try{await root.navigator.clipboard.writeText(proofText(e));cp.textContent=t.proofCopied}catch{}};actions.append(cp)}
   const del=el('button',{type:'button',class:'secondary-btn',text:'✕',title:t.walletDelete});del.setAttribute('aria-label',t.walletDelete);del.onclick=()=>{const risky=isCL(e)&&(holdsRights(e)||e.balanceUnits>0);if((!settled||risky)&&root.confirm&&!root.confirm(risky?CL.words().forgetWarn:t.forget))return;saveSent(sentList().filter(x=>x.card.id!==e.card.id));renderSent(host,false)};actions.append(del);
   info.append(message);
   row.append(el('img',{class:'usdt-coin usdt-coin-small',src:coinFile(e.asset),alt:'',width:40,height:40}),info,actions);
   if(isCL(e)&&CL){
    const w=CL.words(chain.symbol),fee=e.cashlink.feeUnits,bal=e.balanceUnits||0;
    if(e.state==='awaiting_funds')row.append(fundingPanel(e,()=>refreshSent(host,true)));
    if(bal>0)info.append(el('small',{text:CL.fmt(w.balance,{amount:unitsText(bal)})}));
    if(e.recovering)info.append(el('small',{text:w.refundVerifying}));
    if(e.stale)info.append(el('small',{text:w.stale}));
    // Take the money back: any unpaid card that holds more than the fee reserve, or a surplus left after the payout.
    if(bal>fee&&e.state!=='withdrawn'&&!e.recovering){
     const back=el('button',{type:'button',class:'secondary-btn',text:w.refund});
     back.onclick=()=>{const form=el('div',{class:'usdt-refund'}),dest=el('input',{type:'text',autocomplete:'off',spellcheck:false,placeholder:w.refundTo,value:wallets().find(x=>x.chain===chain.key)?.address||''}),go=el('button',{type:'button',class:'secondary-btn',text:w.refund});
      go.onclick=async()=>{const a=normalize(dest.value);if(!a){message.textContent=w.refundAsk;return}go.disabled=true;try{const r=await CL.refund(e,a,{confirm:({units,fee})=>root.confirm(CL.fmt(w.refundConfirm,{net:unitsText(units),fee:unitsText(fee)}))});updateSent(e.card.id,{recovering:true});message.textContent=r.status==='refunded'?w.refundDone:w.refundVerifying;refreshSent(host,true)}catch(err){message.textContent=errorText(err);go.disabled=false}};
      form.append(el('small',{text:w.refundAsk}),dest,go);back.replaceWith(form)};
     actions.append(back)}}
   else if(e.state==='ready'&&e.pay&&e.pay.recipient&&addressValid(e.pay.recipient))row.append(payPanel(e));
   host.append(row)}
  if(check)refreshSent(host,false)}

 return{panel,mountWallet,mountSent:renderSent,wallets,addWallet,sentList,saveSent,holdsRights,labels:sym=>L(sym),requestCard,fundingPanel}
}
root.PhoenixUSDT={create,addressValid,checksum,keccak256,validateClaim,validateRequest,CHAINS,words:WORDS,extra:EXTRA};
})(typeof window!=='undefined'?window:globalThis);
