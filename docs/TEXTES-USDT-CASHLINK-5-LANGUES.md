# Textes USDT cashlink — FR / EN / ES / DE / ZH

Date : 22 octobre 2026. Catalogue proposé pour PhoenixMail ; aucune traduction d'application modifiée et aucun code copié dans PhoenixWire.

## Règles de présentation

Toutes les cartes USDT sont protégées. Aucun sélecteur de carte ouverte n'est prévu. Le libellé français est **Protégée**. Les termes techniques du mécanisme de code ne sont pas affichés.

Polygon, Ethereum, USDT, USDT0, PhoenixMail et Phoenix Wire restent des marques/noms non traduits. Le réseau affiché est Polygon ou Ethereum. Les placeholders sont des valeurs de présentation : jamais une clé du lien, un jeton de récupération ou un secret maître.

Format monétaire localisé ; les calculs restent en unités entières à six décimales. {amount} est le montant USDT du destinataire, {reserve} la réserve USDT pour le relayeur, {total} leur somme. Le portefeuille de l'expéditeur ajoute les frais réseau ordinaires de son transfert de financement. « Une seule fois » désigne ce seul transfert de financement, pas une promesse de gratuité pour l'expéditeur.

Ne jamais traduire une annulation applicative par « fonds remboursés ». Après émission d'une signature, une course avec une réclamation reste possible. « Payé » et les notifications de succès exigent une confirmation et une persistance fiables ; un simple appel de co-signature ne déclenche pas de notification de paiement.

Les textes de remise du lien ci-dessous sont des variantes à activer selon la [décision R3](archive/DECISION-R3-LIEN-COURRIEL.md). Ne pas afficher « carte envoyée » après une simple copie ou ouverture d'une feuille de partage.

## Page d'envoi PhoenixMail

| Clé | FR | EN | ES | DE | ZH |
| --- | --- | --- | --- | --- | --- |
| send.title | Envoyer une carte USDT | Send a USDT card | Enviar una tarjeta USDT | USDT-Karte senden | 发送 USDT 卡片 |
| send.protection | Protégée | Protected | Protegida | Geschützt | 受保护 |
| send.protection_help | Le code est envoyé au courriel du destinataire prévu pour cette carte. | The code is sent to the designated recipient's email address. | El código se envía al correo del destinatario indicado. | Der Code wird an die E-Mail-Adresse des angegebenen Empfängers gesendet. | 验证码将发送至指定收件人的邮箱。 |
| send.email | Courriel du destinataire | Recipient's email | Correo del destinatario | E-Mail-Adresse des Empfängers | 收件人邮箱 |
| send.sender | Votre courriel | Your email | Tu correo electrónico | Deine E-Mail-Adresse | 你的邮箱 |
| send.network | Réseau | Network | Red | Netzwerk | 网络 |
| send.amount | Montant pour le destinataire | Amount for the recipient | Importe para el destinatario | Betrag für den Empfänger | 收款人应收金额 |
| send.reserve | Frais de réseau en réserve (USDT) | Network fee reserve (USDT) | Reserva para gastos de red (USDT) | Reserve für Netzwerkgebühren (USDT) | 网络手续费预留（USDT） |
| send.reserve_help | Cette réserve paie le service qui effectue la réclamation. Le destinataire n'a pas à avancer les frais réseau. | This reserve pays the service that submits the claim. The recipient does not need to advance any network fee. | Esta reserva paga al servicio que tramita el cobro. El destinatario no tiene que adelantar gastos de red. | Diese Reserve bezahlt den Dienst, der die Auszahlung übermittelt. Der Empfänger muss keine Netzwerkgebühren vorstrecken. | 此预留费用用于支付提交领取交易的服务，收款人无需垫付网络手续费。 |
| send.total | Total à transférer : {total} USDT | Total to transfer: {total} USDT | Total a transferir: {total} USDT | Zu überweisender Gesamtbetrag: {total} USDT | 转入总额：{total} USDT |
| send.once | Vous payez une seule fois : le montant et la réserve dans un seul transfert USDT. | You pay once: the amount and reserve in one USDT transfer. | Pagas una sola vez: el importe y la reserva en una única transferencia USDT. | Du zahlst einmal: Betrag und Reserve in einer einzigen USDT-Überweisung. | 只需支付一次：通过一笔 USDT 转账支付金额与预留费用。 |
| send.wallet_fee | Votre portefeuille ajoute les frais réseau habituels de ce transfert. | Your wallet adds its usual network fee for this transfer. | Tu cartera añade los gastos de red habituales de esta transferencia. | Deine Wallet berechnet zusätzlich die übliche Netzwerkgebühr für diese Überweisung. | 你的钱包会另行收取本次转账的常规网络手续费。 |
| send.recipient_free | Le destinataire ne paie rien pour réclamer la carte. | The recipient pays nothing to claim the card. | El destinatario no paga nada para cobrar la tarjeta. | Der Empfänger zahlt nichts für die Einlösung der Karte. | 收款人领取卡片无需付费。 |
| send.message | Message au destinataire | Message to the recipient | Mensaje para el destinatario | Nachricht an den Empfänger | 给收件人的留言 |
| send.message_optional | Facultatif, 140 caractères maximum | Optional, up to 140 characters | Opcional, hasta 140 caracteres | Optional, höchstens 140 Zeichen | 可选，最多 140 个字符 |
| send.preview | Aperçu de la carte | Card preview | Vista previa de la tarjeta | Kartenvorschau | 卡片预览 |
| send.create | Créer la carte | Create card | Crear tarjeta | Karte erstellen | 创建卡片 |
| send.confirm | Confirmer le montant et la réserve | Confirm amount and reserve | Confirmar importe y reserva | Betrag und Reserve bestätigen | 确认金额与预留费用 |
| send.scan | Scannez pour payer la carte | Scan to fund the card | Escanea para ingresar fondos | Zum Aufladen der Karte scannen | 扫码为卡片转入资金 |
| send.wallet | Ouvrir mon portefeuille | Open my wallet | Abrir mi cartera | Meine Wallet öffnen | 打开我的钱包 |
| send.network_warning | Envoyez uniquement {asset} sur {chain} à cette adresse. | Send only {asset} on {chain} to this address. | Envía solo {asset} en {chain} a esta dirección. | Sende an diese Adresse nur {asset} über {chain}. | 仅向此地址发送 {chain} 上的 {asset}。 |
| send.awaiting | En attente de paiement | Awaiting payment | Pendiente de pago | Zahlung ausstehend | 等待付款 |
| send.ready | Prêt | Ready | Listo | Bereit | 已就绪 |
| send.paid | Payé | Paid | Pagado | Bezahlt | 已付款 |
| send.cancelled | Annulé | Cancelled | Cancelado | Storniert | 已取消 |
| send.ready_help | La carte est financée. Le destinataire peut la réclamer avec son code courriel. | The card is funded. The recipient can claim it using the emailed code. | La tarjeta tiene fondos. El destinatario puede cobrarla con el código enviado por correo. | Die Karte ist aufgeladen. Der Empfänger kann sie mit dem Code aus der E-Mail einlösen. | 卡片资金已到账，收款人可使用邮件验证码领取。 |
| send.confirming | Confirmation du paiement en cours | Confirming payment | Confirmando el pago | Zahlung wird bestätigt | 正在确认付款 |
| send.unknown | Nous vérifions le paiement. Ne payez pas une deuxième fois. | We are checking the payment. Do not pay again. | Estamos comprobando el pago. No pagues otra vez. | Wir prüfen die Zahlung. Zahle nicht erneut. | 正在核实付款，请勿重复支付。 |
| send.email_action | Envoyer la carte par courriel | Email the card | Enviar la tarjeta por correo | Karte per E-Mail senden | 通过邮件发送卡片 |
| send.sent | Carte envoyée | Card sent | Tarjeta enviada | Karte gesendet | 卡片已发送 |
| send.share | Partager le lien depuis cet appareil | Share the link from this device | Compartir el enlace desde este dispositivo | Link von diesem Gerät teilen | 从此设备分享链接 |
| send.copied | Lien copié. Il n'a pas encore été envoyé. | Link copied. It has not been sent yet. | Enlace copiado. Todavía no se ha enviado. | Link kopiert. Er wurde noch nicht gesendet. | 链接已复制，尚未发送。 |
| send.backup | Sauvegarder les informations de récupération | Save recovery information | Guardar los datos de recuperación | Wiederherstellungsdaten sichern | 保存恢复信息 |
| send.backup_warning | Gardez cette sauvegarde privée : elle permet de demander la récupération de la carte. | Keep this backup private: it allows recovery of the card to be requested. | Guarda esta copia en privado: permite solicitar la recuperación de la tarjeta. | Halte diese Sicherung privat: Sie ermöglicht die Anforderung einer Rückholung der Karte. | 请妥善保管此备份，它可用于申请取回卡片资金。 |
| send.cancel | Annuler la carte | Cancel card | Cancelar tarjeta | Karte stornieren | 取消卡片 |
| send.cancel_help | Annuler ne rembourse pas les fonds. Une réclamation déjà signée peut encore aboutir. | Cancelling does not refund the funds. An already signed claim may still complete. | Cancelar no devuelve los fondos. Un cobro ya firmado aún puede completarse. | Eine Stornierung erstattet kein Guthaben. Eine bereits signierte Einlösung kann noch ausgeführt werden. | 取消不会退回资金，已签名的领取交易仍可能完成。 |
| send.refund | Récupérer les fonds | Recover funds | Recuperar los fondos | Guthaben zurückholen | 取回资金 |
| send.refund_pending | Récupération en cours | Recovery in progress | Recuperación en curso | Rückholung läuft | 正在取回资金 |
| send.refunded | Fonds récupérés | Funds recovered | Fondos recuperados | Guthaben zurückgeholt | 资金已取回 |
| send.invalid_email | Entrez le courriel valide du destinataire. | Enter the recipient's valid email address. | Introduce un correo válido del destinatario. | Gib eine gültige E-Mail-Adresse des Empfängers ein. | 请输入有效的收件人邮箱。 |
| send.invalid_amount | Entrez un montant entre {min} et {max} USDT. | Enter an amount between {min} and {max} USDT. | Introduce un importe entre {min} y {max} USDT. | Gib einen Betrag zwischen {min} und {max} USDT ein. | 请输入 {min} 至 {max} USDT 的金额。 |
| send.invalid_fee | La réserve de frais n'est pas valide. | The fee reserve is invalid. | La reserva para gastos no es válida. | Die Gebührenreserve ist ungültig. | 手续费预留金额无效。 |
| send.unavailable | Réseau indisponible. Réessayez plus tard sans créer une nouvelle carte. | Network unavailable. Try again later without creating a new card. | Red no disponible. Inténtalo más tarde sin crear otra tarjeta. | Netzwerk nicht verfügbar. Versuche es später erneut, ohne eine neue Karte zu erstellen. | 网络不可用，请稍后重试，不要创建新卡片。 |

## Page de réclamation

| Clé | FR | EN | ES | DE | ZH |
| --- | --- | --- | --- | --- | --- |
| claim.title | Recevoir votre paiement | Receive your payment | Recibir tu pago | Deine Zahlung empfangen | 领取你的付款 |
| claim.protection | Carte protégée | Protected card | Tarjeta protegida | Geschützte Karte | 受保护卡片 |
| claim.amount | Vous recevez {amount} USDT sur {chain}. | You receive {amount} USDT on {chain}. | Recibes {amount} USDT en {chain}. | Du erhältst {amount} USDT auf {chain}. | 你将收到 {chain} 上的 {amount} USDT。 |
| claim.free | Vous ne payez rien pour recevoir ce paiement. | You pay nothing to receive this payment. | No pagas nada para recibir este pago. | Du zahlst nichts, um diese Zahlung zu empfangen. | 领取这笔付款无需付费。 |
| claim.code_help | Demandez un code à l'adresse courriel prévue pour cette carte. | Request a code at the email address designated for this card. | Solicita un código en el correo indicado para esta tarjeta. | Fordere einen Code an die für diese Karte vorgesehene E-Mail-Adresse an. | 请向此卡片指定的邮箱申请验证码。 |
| claim.send_code | Recevoir mon code par courriel | Email me a code | Recibir mi código por correo | Code per E-Mail erhalten | 通过邮件接收验证码 |
| claim.code_sent | Code envoyé. Vérifiez aussi les courriels indésirables. | Code sent. Check your spam folder too. | Código enviado. Revisa también el correo no deseado. | Code gesendet. Prüfe auch den Spamordner. | 验证码已发送，请同时检查垃圾邮件。 |
| claim.code | Code reçu par courriel | Code received by email | Código recibido por correo | Code aus der E-Mail | 邮件验证码 |
| claim.verify | Vérifier le code | Verify code | Verificar código | Code bestätigen | 验证验证码 |
| claim.resend | Renvoyer le code | Resend code | Reenviar código | Code erneut senden | 重新发送验证码 |
| claim.resend_wait | Renvoyer dans {seconds} s | Resend in {seconds}s | Reenviar en {seconds} s | Erneut senden in {seconds} s | {seconds} 秒后可重新发送 |
| claim.address | Donnez votre adresse de réception | Enter your receiving address | Indica tu dirección de recepción | Gib deine Empfangsadresse ein | 请输入你的收款地址 |
| claim.address_help | Cette adresse doit accepter {asset} sur {chain}. Aucun frais réseau ne vous sera demandé. | This address must accept {asset} on {chain}. You will not be asked to pay a network fee. | Esta dirección debe admitir {asset} en {chain}. No se te pedirá pagar gastos de red. | Diese Adresse muss {asset} auf {chain} unterstützen. Du musst keine Netzwerkgebühr bezahlen. | 此地址必须支持 {chain} 上的 {asset}，领取时无需支付网络手续费。 |
| claim.confirm_address | Confirmez cette adresse : {address} | Confirm this address: {address} | Confirma esta dirección: {address} | Bestätige diese Adresse: {address} | 请确认此地址：{address} |
| claim.address_final | Après confirmation, la destination de cette carte ne pourra plus être changée. | Once confirmed, this card's destination cannot be changed. | Tras confirmar, no se podrá cambiar el destino de esta tarjeta. | Nach der Bestätigung kann die Zieladresse dieser Karte nicht mehr geändert werden. | 确认后，此卡片的收款地址将无法更改。 |
| claim.confirm | Confirmer et recevoir | Confirm and receive | Confirmar y recibir | Bestätigen und empfangen | 确认并领取 |
| claim.pending | Votre paiement est en cours de confirmation. | Your payment is being confirmed. | Tu pago se está confirmando. | Deine Zahlung wird bestätigt. | 你的付款正在确认中。 |
| claim.success | Paiement reçu sur votre adresse. | Payment received at your address. | Pago recibido en tu dirección. | Zahlung an deiner Adresse eingegangen. | 付款已到达你的地址。 |
| claim.tx | Voir la transaction | View transaction | Ver la transacción | Transaktion ansehen | 查看交易 |
| claim.invalid_code | Code invalide ou expiré. Demandez un nouveau code. | Invalid or expired code. Request a new code. | Código no válido o caducado. Solicita otro código. | Code ungültig oder abgelaufen. Fordere einen neuen Code an. | 验证码无效或已过期，请申请新验证码。 |
| claim.expired | Cette carte a expiré. Contactez l'expéditeur. | This card has expired. Contact the sender. | Esta tarjeta ha caducado. Contacta con el remitente. | Diese Karte ist abgelaufen. Kontaktiere den Absender. | 此卡片已过期，请联系发件人。 |
| claim.cancelled | Cette carte est annulée. Contactez l'expéditeur. | This card is cancelled. Contact the sender. | Esta tarjeta está cancelada. Contacta con el remitente. | Diese Karte wurde storniert. Kontaktiere den Absender. | 此卡片已取消，请联系发件人。 |
| claim.not_funded | Cette carte n'est pas encore financée. Aucun paiement supplémentaire ne vous est demandé. | This card is not funded yet. You are not being asked to make any payment. | Esta tarjeta aún no tiene fondos. No se te pide realizar ningún pago. | Diese Karte ist noch nicht aufgeladen. Du wirst zu keiner Zahlung aufgefordert. | 此卡片资金尚未到账，你无需进行任何付款。 |
| claim.network_unavailable | Réseau indisponible. Réessayez plus tard. | Network unavailable. Try again later. | Red no disponible. Inténtalo más tarde. | Netzwerk nicht verfügbar. Versuche es später erneut. | 网络不可用，请稍后重试。 |
| claim.invalid_address | Vérifiez l'adresse de réception et le réseau choisi. | Check the receiving address and selected network. | Comprueba la dirección de recepción y la red elegida. | Prüfe die Empfangsadresse und das gewählte Netzwerk. | 请检查收款地址与所选网络。 |
| claim.address_set | Une autre adresse est déjà confirmée pour cette carte. | A different address is already confirmed for this card. | Ya hay otra dirección confirmada para esta tarjeta. | Für diese Karte wurde bereits eine andere Adresse bestätigt. | 此卡片已确认使用其他收款地址。 |
| claim.already_paid | Cette carte a déjà été payée. Consultez la transaction. | This card has already been paid. View the transaction. | Esta tarjeta ya se ha pagado. Consulta la transacción. | Diese Karte wurde bereits ausgezahlt. Sieh dir die Transaktion an. | 此卡片已完成付款，请查看交易。 |
| claim.too_many_attempts | Trop de tentatives. Réessayez plus tard. | Too many attempts. Try again later. | Demasiados intentos. Inténtalo más tarde. | Zu viele Versuche. Versuche es später erneut. | 尝试次数过多，请稍后重试。 |
| claim.code_delivery_failed | Le code n'a pas pu être envoyé. Réessayez plus tard. | The code could not be sent. Try again later. | No se pudo enviar el código. Inténtalo más tarde. | Der Code konnte nicht gesendet werden. Versuche es später erneut. | 验证码发送失败，请稍后重试。 |
| claim.unknown | Nous vérifions le résultat. N'envoyez pas d'argent et ne recommencez pas le paiement. | We are checking the outcome. Do not send money or start another payment. | Estamos comprobando el resultado. No envíes dinero ni inicies otro pago. | Wir prüfen das Ergebnis. Sende kein Geld und starte keine weitere Zahlung. | 正在核实结果，请勿转账或重新发起付款。 |
| claim.retry | Vérifier à nouveau | Check again | Comprobar de nuevo | Erneut prüfen | 再次核实 |

## Notifications à l'expéditeur

Utiliser la langue expéditeur enregistrée, pas celle du navigateur du destinataire. Ces messages ne contiennent ni lien complet de réclamation, ni secret de récupération. {status_url} est un lien vers l'espace expéditeur exigeant l'authentification appropriée ; ne pas placer le jeton de récupération dans son URL.

| Clé | FR | EN | ES | DE | ZH |
| --- | --- | --- | --- | --- | --- |
| notify.ready.subject | Votre carte USDT est prête | Your USDT card is ready | Tu tarjeta USDT está lista | Deine USDT-Karte ist bereit | 你的 USDT 卡片已就绪 |
| notify.ready.body | Le financement de votre carte de {amount} USDT sur {chain} est confirmé. Le destinataire peut la réclamer avec son code courriel. | Funding for your {amount} USDT card on {chain} is confirmed. The recipient can claim it using the emailed code. | Los fondos de tu tarjeta de {amount} USDT en {chain} están confirmados. El destinatario puede cobrarla con el código enviado por correo. | Die Einzahlung für deine Karte über {amount} USDT auf {chain} ist bestätigt. Der Empfänger kann sie mit dem Code aus der E-Mail einlösen. | 你在 {chain} 上的 {amount} USDT 卡片资金已确认，收款人可使用邮件验证码领取。 |
| notify.paid.subject | Votre carte USDT a été réclamée | Your USDT card has been claimed | Tu tarjeta USDT se ha cobrado | Deine USDT-Karte wurde eingelöst | 你的 USDT 卡片已被领取 |
| notify.paid.body | Le paiement de {amount} USDT sur {chain} est confirmé à l'adresse choisie par le destinataire. | The payment of {amount} USDT on {chain} is confirmed at the address chosen by the recipient. | El pago de {amount} USDT en {chain} está confirmado en la dirección elegida por el destinatario. | Die Zahlung von {amount} USDT auf {chain} an die vom Empfänger gewählte Adresse ist bestätigt. | {chain} 上的 {amount} USDT 付款已确认到达收款人选择的地址。 |
| notify.refunded.subject | Votre récupération USDT est confirmée | Your USDT recovery is confirmed | La recuperación de tus USDT está confirmada | Deine USDT-Rückholung ist bestätigt | 你的 USDT 取回操作已确认 |
| notify.refunded.body | La récupération de {amount} USDT sur {chain} est confirmée. Consultez le détail de votre envoi. | Recovery of {amount} USDT on {chain} is confirmed. View your payment details. | La recuperación de {amount} USDT en {chain} está confirmada. Consulta los detalles del envío. | Die Rückholung von {amount} USDT auf {chain} ist bestätigt. Sieh dir die Details deiner Sendung an. | {chain} 上的 {amount} USDT 已确认取回，请查看付款记录详情。 |
| notify.details | Voir mon envoi | View my payment | Ver mi envío | Meine Sendung ansehen | 查看我的付款 |
| notify.cancelled.subject | Votre carte USDT est annulée | Your USDT card is cancelled | Tu tarjeta USDT está cancelada | Deine USDT-Karte wurde storniert | 你的 USDT 卡片已取消 |
| notify.cancelled.body | Les nouvelles réclamations de cette carte sont bloquées par le service. Cela ne rembourse pas les fonds et ne révoque pas une transaction déjà signée. | The service blocks new claims for this card. This does not refund funds or revoke an already signed transaction. | El servicio bloquea nuevos cobros de esta tarjeta. Esto no devuelve los fondos ni revoca una transacción ya firmada. | Der Dienst blockiert neue Einlösungen dieser Karte. Das erstattet kein Guthaben und widerruft keine bereits signierte Transaktion. | 服务已阻止此卡片的新领取请求，但这不会退回资金，也不会撤销已签名的交易。 |

## Conditions d'activation

Les promesses de destination finale, d'état payé et de notification fiable sont des critères produit à atteindre, pas une validation du code actuel. Corriger UL-01 à UL-08 de la revue avant d'activer ces textes. Le message « carte prête » décrit le financement ; il ne doit pas ressusciter l'ancien modèle où le destinataire donnait d'abord son adresse avant que l'expéditeur ne paie.

Aucune chaîne, montant, adresse de réception ou résultat de transaction ne doit être déduit d'un texte libre de message. Tous les placeholders proviennent de données validées ; échapper les valeurs dans le courriel et dans la page.
