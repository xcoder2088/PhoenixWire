export const legalCopy = {
  "en": {
    "ui": [
      "Terms of use",
      "Privacy",
      "Back to Phoenix Wire",
      "Language",
      "Automatic",
      "Switch appearance",
      "Powered by",
      "Data kept",
      "Purpose",
      "Retention",
      "Operator and contact"
    ],
    "pending": [
      "Operator name",
      "Contact address",
      "Governing law",
      "Minimum age",
      "Prohibited uses"
    ],
    "values": ["Xcoder", "phoenixmails@proton.me", "The laws of Québec, Canada, and the federal laws of Canada that apply there.", "18 years", "Any illegal use: fraud, money laundering, or payments for goods or services that are themselves illegal. Paying for lawful goods or services agreed between the users is allowed."],
    "notice": "Last updated: 24 October 2026.",
    "identity": "Phoenix Wire is operated by the person named below. Questions and requests, including deletion requests, go to the contact address below.",
    "lifetime": [
      "24-hour cards",
      "A sender confirms an email with a six-digit code, funds a NIM, USDC on Polygon, or USDT on Polygon or Ethereum card, and Wire emails a link to the recipient. The recipient confirms their own email with another code, then chooses a receiving address or opens the usual Nimiq payment link.",
      "Wire does not hold your personal wallet. Each card has its own address on the blockchain, the public record of payments.",
      "A card is valid for 24 hours. An unclaimed NIM card is then refunded automatically to the address that funded it. For USDC/USDT, the sender opens Wire and recovers the funds using their browser key. If NIM was funded from an exchange or a custodial wallet, the refund returns to that funding address."
    ],
    "fees": [
      "Fees and refunds",
      "NIM has no Wire fee beyond the network fee. For USDC/USDT, the fee is the card’s network cost when created, with a margin, plus 6% of the amount. The sender sees one total amount to pay. This margin helps absorb changes in network costs.",
      "The relay sends the card’s payment and pays its network cost. A refund retains the full relay fee: the estimated network cost, including its margin, plus 6% of the amount.",
      "Estimates use third-party data: network nodes for network costs, Chainlink for prices, and CoinGecko as a fallback."
    ],
    "risk": [
      "Final payments and risk",
      "Once a payment is claimed, nobody, including Wire, can reverse it. Check the receiving address carefully.",
      "The software is provided as it is, without warranty. It moves real money. A code review is not a security certification."
    ],
    "access": [
      "Sending access",
      "Sending is by invitation unless an administrator opens access. Changing access does not change cards already sent."
    ],
    "rules": ["Rules of use", "The rules below apply to everyone who uses Phoenix Wire. They are listed with the operator and the contact details at the end of this page."],
    "storage": [
      "How Wire handles data",
      "Wire keeps its main state in an encrypted file (AES-256-GCM), with permissions restricted to its owner. Engine files are also private; files holding secrets are encrypted. This is not a guarantee that data can never be accessed."
    ],
    "rows": [
      [
        "Sender email; session token hash",
        "Sender access",
        "Email confirmation code: 15 minutes. Signed-in session: 24 hours. Sessions are deleted on expiry or when the sender is blocked."
      ],
      [
        "NIM cards: sender and recipient emails, amount, message up to 160 characters, language, public card address, stage, dates and decorative image",
        "Card operation, history, email and refund",
        "Finished cards: email addresses and messages are erased 60 days after the card ends, if it holds nothing. Unpaid drafts: 2 days. Cancelled cards: 14 days."
      ],
      [
        "USDC/USDT cards: sender and recipient emails, amount, message, public card address, refund token held by Wire and dates",
        "Card operation, history, email and refund",
        "Erased 60 days after the card ends, if it holds nothing."
      ],
      [
        "Secret-keyed hash of the visitor’s IP address and a counter",
        "Abuse prevention",
        "About 1 hour."
      ],
      [
        "Allowed and blocked email lists",
        "Invitation-only access",
        "Until an administrator removes them."
      ],
      [
        "Record of reminders and notices sent",
        "Avoid sending the same notice twice",
        "Kept with the card."
      ]
    ],
    "engine": [
      "Keys and payment records",
      "The engine keeps each NIM card’s private key, encrypted with its master key, to pay the recipient or refund it. USDC/USDT co-signing and relay keys are derived from that master key.",
      "It also keeps a journal per NIM payout or refund: destination, signed transaction and transaction identifier, without a key. Email verification codes are kept as hashes with their expiry.",
      "The margin ledger records the first eight characters of a card identifier, amounts and network costs per settled USDC/USDT payout. It contains no email or address. No retention period for these engine records is specified here."
    ],
    "others": [
      "Who else sees data",
      [
        [
          "MailerSend",
          "Sender and recipient emails, message, card amount, link and codes, to deliver emails."
        ],
        [
          "Cloudflare",
          "Visitor IP addresses and requests, to connect visitors to the server. Wire uses the forwarded visitor address only for rate limiting, as a hash."
        ],
        [
          "Blockchain nodes",
          "Card and relay addresses queried for balances, funding and network costs."
        ],
        [
          "NIM, Polygon and Ethereum",
          "Payments are public and permanent on their blockchains."
        ],
        [
          "Chainlink; CoinGecko as a fallback",
          "The coin identifier only, to price the network cost in the fee."
        ],
        [
          "Nimiq services, Nimpay and receiving wallets",
          "What you do with them to receive NIM. Their own terms apply."
        ]
      ]
    ],
    "absent": [
      "What Wire does not keep",
      "Wire has no passwords. It does not store the USDC/USDT link key: the key stays in the sender’s browser and recipient’s link. Wire sees the complete claim link once to email it, but does not store it.",
      "Wire does not keep clear-text IP addresses, advertising identifiers or analytics. Email open and click tracking is disabled in every message. This does not mean Cloudflare or MailerSend sees no data."
    ],
    "deletion": [
      "Requesting deletion and its limits",
      "Wire erases a card’s email addresses and message by itself 60 days after the card ends, if the card holds nothing. A card that still holds money keeps its data, because its sender may need to recover it. You can ask for an earlier removal for a card that is over. Payments stay on the blockchain, and backups keep what they held on the day they were made.",
      "Send requests to the contact address below. No response deadline is promised here.",
      "Wire cannot remove blockchain records, MailerSend delivery logs or Cloudflare data. Those providers’ own retention applies. Removing contact data cannot reverse a claimed payment."
    ]
  },
  "fr": {
    "ui": [
      "Conditions d’utilisation",
      "Confidentialité",
      "Retour à Phoenix Wire",
      "Langue",
      "Automatique",
      "Changer d’apparence",
      "Propulsé par",
      "Données gardées",
      "Pourquoi",
      "Durée de conservation",
      "Exploitant et contact"
    ],
    "pending": [
      "Nom de l’exploitant",
      "Adresse de contact",
      "Loi applicable",
      "Âge minimum",
      "Usages interdits"
    ],
    "values": ["Xcoder", "phoenixmails@proton.me", "Les lois du Québec (Canada) et les lois fédérales du Canada qui s’y appliquent.", "18 ans", "Tout usage illégal : fraude, blanchiment d’argent, ou paiement de biens ou de services eux-mêmes illégaux. Payer des biens ou des services légaux convenus entre les utilisateurs est permis."],
    "notice": "Dernière mise à jour : 24 octobre 2026.",
    "identity": "Phoenix Wire est exploité par la personne nommée ci-dessous. Les questions et les demandes, y compris de suppression, vont à l’adresse de contact ci-dessous.",
    "lifetime": [
      "Des cartes de 24 heures",
      "L’expéditeur confirme son courriel avec un code à six chiffres, paie une carte NIM, USDC sur Polygon ou USDT sur Polygon ou Ethereum. Wire envoie un lien au destinataire. Celui-ci confirme son propre courriel avec un autre code, puis choisit une adresse de réception ou ouvre le lien de paiement Nimiq habituel.",
      "Wire ne garde pas ton portefeuille personnel. Chaque carte a sa propre adresse sur la chaîne, le registre public des paiements.",
      "Une carte est valable 24 heures. Une carte NIM non réclamée est ensuite remboursée automatiquement à l’adresse qui l’a financée. Pour l’USDC/USDT, l’expéditeur ouvre Wire et reprend les fonds avec la clé de son navigateur. Si le NIM venait d’une plateforme d’échange ou d’un portefeuille géré par un tiers, le remboursement retourne à cette adresse de financement."
    ],
    "fees": [
      "Frais et remboursements",
      "Pour le NIM, Wire ne prend aucuns frais au-delà des frais du réseau. Pour l’USDC/USDT : le coût réseau de la carte à sa création, avec une marge, plus 6 % du montant. L’expéditeur voit un seul montant total à payer. Cette marge aide à absorber les variations des coûts réseau.",
      "Le relais envoie le paiement de la carte et paie son coût réseau. Lors d’un remboursement, ses frais complets sont conservés : coût réseau estimé, avec sa marge, plus 6 % du montant.",
      "Les estimations utilisent des données de tiers : les nœuds du réseau pour ses coûts, Chainlink pour les prix et CoinGecko en secours."
    ],
    "risk": [
      "Paiements définitifs et risques",
      "Un paiement réclamé est irréversible, y compris pour Wire. Vérifie soigneusement l’adresse de réception.",
      "Le logiciel est fourni tel quel, sans garantie. Il déplace de l’argent réel. Une revue du code n’est pas une certification de sécurité."
    ],
    "access": [
      "Accès à l’envoi",
      "L’envoi se fait sur invitation, sauf si un administrateur ouvre l’accès. Un changement d’accès ne modifie pas les cartes déjà envoyées."
    ],
    "rules": ["Règles d’utilisation", "Les règles ci-dessous s’appliquent à toute personne qui utilise Phoenix Wire. Elles figurent avec l’exploitant et le contact à la fin de cette page."],
    "storage": [
      "Comment Wire traite les données",
      "Wire garde son état principal dans un fichier chiffré (AES-256-GCM), accessible seulement à son propriétaire par les permissions du fichier. Les fichiers du moteur sont aussi privés ; ceux qui gardent des secrets sont chiffrés. Cela ne garantit pas que les données soient inaccessibles en toute circonstance."
    ],
    "rows": [
      [
        "Courriel de l’expéditeur ; empreinte du jeton de session",
        "Accès de l’expéditeur",
        "Code de confirmation : 15 minutes. Session : 24 heures. Sessions supprimées à expiration ou quand l’expéditeur est bloqué."
      ],
      [
        "Cartes NIM : courriels de l’expéditeur et du destinataire, montant, message de 160 caractères maximum, langue, adresse publique de la carte, étape, dates et image décorative",
        "Fonctionnement, historique, courriel et remboursement",
        "Cartes terminées : courriels et messages effacés 60 jours après la fin de la carte, si elle ne contient rien. Brouillons non payés : 2 jours. Cartes annulées : 14 jours."
      ],
      [
        "Cartes USDC/USDT : courriels de l’expéditeur et du destinataire, montant, message, adresse publique de la carte, jeton de remboursement gardé par Wire et dates",
        "Fonctionnement, historique, courriel et remboursement",
        "Effacé 60 jours après la fin de la carte, si elle ne contient rien."
      ],
      [
        "Empreinte de l’adresse IP du visiteur calculée avec un secret, et compteur",
        "Limiter les abus",
        "Environ 1 heure."
      ],
      [
        "Listes de courriels autorisés et bloqués",
        "Gérer l’accès sur invitation",
        "Jusqu’au retrait par un administrateur."
      ],
      [
        "Trace des rappels et avis envoyés",
        "Ne pas envoyer deux fois le même avis",
        "Conservée avec la carte."
      ]
    ],
    "engine": [
      "Clés et traces de paiement",
      "Le moteur garde la clé privée de chaque carte NIM, chiffrée avec sa clé principale, pour payer le destinataire ou rembourser. Les clés de cosignature et de relais USDC/USDT sont dérivées de cette clé principale.",
      "Il garde aussi un journal par paiement ou remboursement NIM : destination, transaction signée et identifiant de transaction, sans clé. Les codes de vérification par courriel sont gardés sous forme d’empreintes avec leur expiration.",
      "Le registre de marge garde les huit premiers caractères de l’identifiant de carte, les montants et les coûts réseau de chaque paiement USDC/USDT finalisé. Aucun courriel ni adresse. Aucune durée de conservation de ces traces du moteur n’est précisée ici."
    ],
    "others": [
      "Qui voit aussi des données",
      [
        [
          "MailerSend",
          "Courriels de l’expéditeur et du destinataire, message, montant, lien et codes, pour envoyer les courriels."
        ],
        [
          "Cloudflare",
          "Adresses IP des visiteurs et requêtes, pour relier les visiteurs au serveur. Wire utilise l’adresse transmise seulement pour limiter les abus, sous forme d’empreinte."
        ],
        [
          "Nœuds des chaînes",
          "Adresses des cartes et du relais interrogées pour les soldes, le financement et les coûts réseau."
        ],
        [
          "NIM, Polygon et Ethereum",
          "Les paiements sur leurs chaînes sont publics et permanents."
        ],
        [
          "Chainlink ; CoinGecko en secours",
          "Seulement l’identifiant de la monnaie, pour estimer le coût réseau dans les frais."
        ],
        [
          "Services Nimiq, Nimpay et portefeuilles de réception",
          "Ce que tu fais avec eux pour recevoir du NIM. Leurs propres conditions s’appliquent."
        ]
      ]
    ],
    "absent": [
      "Ce que Wire ne garde pas",
      "Wire n’a pas de mots de passe. Il ne stocke pas la clé du lien USDC/USDT : elle reste dans le navigateur de l’expéditeur et le lien du destinataire. Wire voit une fois le lien complet pour l’envoyer par courriel, mais ne le stocke pas.",
      "Wire ne garde pas d’adresses IP en clair, d’identifiants publicitaires ni de données d’analyse d’audience. Le suivi des ouvertures et clics est désactivé dans chaque courriel. Cela ne veut pas dire que Cloudflare ou MailerSend ne voient aucune donnée."
    ],
    "deletion": [
      "Demander une suppression et ses limites",
      "Wire efface lui-même les courriels et le message d’une carte 60 jours après sa fin, si elle ne contient rien. Une carte qui contient encore de l’argent garde ses données : son expéditeur peut avoir besoin de la récupérer. Tu peux demander un retrait plus tôt pour une carte terminée. Les paiements restent sur la blockchain, et les sauvegardes gardent ce qu’elles contenaient le jour où elles ont été faites.",
      "Envoie tes demandes à l’adresse de contact ci-dessous. Aucun délai de réponse n’est promis ici.",
      "Wire ne peut pas retirer les données des chaînes, les journaux d’envoi de MailerSend ni les données de Cloudflare. Les durées de ces fournisseurs s’appliquent. Retirer les coordonnées ne peut pas annuler un paiement réclamé."
    ]
  },
  "es": {
    "ui": [
      "Condiciones de uso",
      "Privacidad",
      "Volver a Phoenix Wire",
      "Idioma",
      "Automático",
      "Cambiar apariencia",
      "Con tecnología de",
      "Datos conservados",
      "Finalidad",
      "Conservación",
      "Operador y contacto"
    ],
    "pending": [
      "Nombre del operador",
      "Dirección de contacto",
      "Ley aplicable",
      "Edad mínima",
      "Usos prohibidos"
    ],
    "values": ["Xcoder", "phoenixmails@proton.me", "Las leyes de Quebec (Canadá) y las leyes federales de Canadá aplicables allí.", "18 años", "Cualquier uso ilegal: fraude, blanqueo de capitales o pago de bienes o servicios que sean en sí mismos ilegales. Pagar bienes o servicios legales acordados entre los usuarios está permitido."],
    "notice": "Última actualización: 24 de octubre de 2026.",
    "identity": "Phoenix Wire lo opera la persona indicada abajo. Las preguntas y solicitudes, incluidas las de borrado, van a la dirección de contacto de abajo.",
    "lifetime": [
      "Tarjetas de 24 horas",
      "El remitente confirma su correo con un código de seis dígitos y paga una tarjeta NIM, USDC en Polygon o USDT en Polygon o Ethereum. Wire envía un enlace al destinatario. Este confirma su propio correo con otro código y elige una dirección para recibir o abre el enlace de pago habitual de Nimiq.",
      "Wire no guarda tu monedero personal. Cada tarjeta tiene su propia dirección en la cadena, el registro público de pagos.",
      "La tarjeta vale 24 horas. Una tarjeta NIM sin reclamar se devuelve automáticamente a la dirección que la financió. Para USDC/USDT, el remitente abre Wire y recupera el dinero con la clave de su navegador. Si el NIM llegó desde una plataforma de intercambio o un monedero custodiado, la devolución vuelve a esa dirección de financiación."
    ],
    "fees": [
      "Comisiones y devoluciones",
      "NIM no tiene comisión de Wire aparte de la del propio sistema de pagos. Para USDC/USDT: el coste de red de la tarjeta al crearla, con un margen, más el 6 % del importe. El remitente ve un solo total a pagar. Este margen ayuda a absorber las variaciones de los costes de red.",
      "El servicio de retransmisión envía el pago y paga el coste de red. En una devolución conserva su comisión completa: coste de red estimado, incluido su margen, más el 6 % del importe.",
      "Las estimaciones usan datos de terceros: nodos de la red para sus costes, Chainlink para los precios y CoinGecko como alternativa."
    ],
    "risk": [
      "Pagos definitivos y riesgos",
      "Una vez reclamado el pago, nadie, incluido Wire, puede revertirlo. Comprueba bien la dirección de destino.",
      "El programa se ofrece tal como está, sin garantía. Mueve dinero real. Una revisión del código no es una certificación de seguridad."
    ],
    "access": [
      "Acceso para enviar",
      "El envío es por invitación, salvo que un administrador abra el acceso. Los cambios de acceso no modifican las tarjetas ya enviadas."
    ],
    "rules": ["Normas de uso", "Las normas siguientes se aplican a todas las personas que usan Phoenix Wire. Figuran con el operador y el contacto al final de esta página."],
    "storage": [
      "Cómo trata Wire los datos",
      "Wire guarda su estado principal en un archivo cifrado (AES-256-GCM), con permisos limitados al propietario del archivo. Los archivos del motor también son privados; los que contienen secretos están cifrados. Esto no garantiza que los datos sean inaccesibles en toda circunstancia."
    ],
    "rows": [
      [
        "Correo del remitente; huella del token de sesión",
        "Acceso del remitente",
        "Código de confirmación: 15 minutos. Sesión: 24 horas. Las sesiones se eliminan al caducar o al bloquear al remitente."
      ],
      [
        "Tarjetas NIM: correos del remitente y destinatario, importe, mensaje de hasta 160 caracteres, idioma, dirección pública, estado, fechas e imagen decorativa",
        "Operación, historial, correo y devolución",
        "Tarjetas terminadas: los correos y mensajes se borran 60 días después del fin de la tarjeta, si no contiene nada. Borradores sin pagar: 2 días. Tarjetas canceladas: 14 días."
      ],
      [
        "Tarjetas USDC/USDT: correos del remitente y destinatario, importe, mensaje, dirección pública, token de devolución guardado por Wire y fechas",
        "Operación, historial, correo y devolución",
        "Se borra 60 días después del fin de la tarjeta, si no contiene nada."
      ],
      [
        "Huella de la IP del visitante calculada con un secreto, y contador",
        "Evitar abusos",
        "Aproximadamente 1 hora."
      ],
      [
        "Listas de correos permitidos y bloqueados",
        "Acceso por invitación",
        "Hasta que un administrador los retire."
      ],
      [
        "Registro de recordatorios y avisos enviados",
        "No enviar dos veces el mismo aviso",
        "Con la tarjeta."
      ]
    ],
    "engine": [
      "Claves y registros de pagos",
      "El motor guarda la clave privada de cada tarjeta NIM, cifrada con su clave maestra, para pagar o devolver fondos. Las claves de firma conjunta y retransmisión USDC/USDT se derivan de esa clave maestra.",
      "También guarda un registro por pago o devolución NIM: destino, transacción firmada e identificador de transacción, sin clave. Los códigos por correo se guardan como huellas con su caducidad.",
      "El registro de margen contiene los ocho primeros caracteres del identificador de tarjeta, importes y costes de red por pago USDC/USDT liquidado. No contiene correos ni direcciones. Aquí no se especifica un plazo de conservación de estos registros del motor."
    ],
    "others": [
      "Quién más ve datos",
      [
        [
          "MailerSend",
          "Correos del remitente y destinatario, mensaje, importe, enlace y códigos, para entregar los correos."
        ],
        [
          "Cloudflare",
          "IP de los visitantes y solicitudes, para llegar al servidor. Wire usa la dirección transmitida solo para limitar abusos, como huella."
        ],
        [
          "Nodos de las cadenas",
          "Direcciones de las tarjetas y del servicio de retransmisión consultadas para saldos, financiación y costes de red."
        ],
        [
          "NIM, Polygon y Ethereum",
          "Sus pagos son públicos y permanentes en las cadenas."
        ],
        [
          "Chainlink; CoinGecko como alternativa",
          "Solo el identificador de la moneda, para valorar el coste de red en la comisión."
        ],
        [
          "Servicios Nimiq, Nimpay y monederos de recepción",
          "Lo que haces con ellos para recibir NIM. Se aplican sus propias condiciones."
        ]
      ]
    ],
    "absent": [
      "Lo que Wire no conserva",
      "Wire no tiene contraseñas. No guarda la clave del enlace USDC/USDT: queda en el navegador del remitente y en el enlace del destinatario. Wire ve el enlace completo una vez para enviarlo por correo, pero no lo guarda.",
      "Wire no conserva IP en texto claro, identificadores publicitarios ni datos de analítica. El seguimiento de aperturas y clics está desactivado en todos los correos. Esto no significa que Cloudflare o MailerSend no vean datos."
    ],
    "deletion": [
      "Solicitar borrado y sus límites",
      "Wire borra por sí mismo los correos y el mensaje de una tarjeta 60 días después de su fin, si no contiene nada. Una tarjeta que aún contiene dinero conserva sus datos: su remitente puede necesitar recuperarlo. Puedes pedir una retirada anterior para una tarjeta terminada. Los pagos permanecen en la blockchain, y las copias de seguridad conservan lo que tenían el día en que se hicieron.",
      "Envía tus solicitudes a la dirección de contacto indicada abajo. Aquí no se promete un plazo de respuesta.",
      "Wire no puede borrar datos de las cadenas, registros de entrega de MailerSend ni datos de Cloudflare. Se aplican los plazos de esos proveedores. Borrar datos de contacto no revierte un pago reclamado."
    ]
  },
  "de": {
    "ui": [
      "Nutzungsbedingungen",
      "Datenschutz",
      "Zurück zu Phoenix Wire",
      "Sprache",
      "Automatisch",
      "Darstellung wechseln",
      "Unterstützt von",
      "Gespeicherte Daten",
      "Zweck",
      "Aufbewahrung",
      "Betreiber und Kontakt"
    ],
    "pending": [
      "Name des Betreibers",
      "Kontaktadresse",
      "Anwendbares Recht",
      "Mindestalter",
      "Verbotene Nutzungen"
    ],
    "values": ["Xcoder", "phoenixmails@proton.me", "Das Recht von Québec (Kanada) und das dort geltende Bundesrecht Kanadas.", "18 Jahre", "Jede rechtswidrige Nutzung: Betrug, Geldwäsche oder Zahlungen für Waren oder Dienstleistungen, die selbst rechtswidrig sind. Die Bezahlung rechtmäßiger Waren oder Dienstleistungen, die zwischen den Nutzern vereinbart wurden, ist erlaubt."],
    "notice": "Zuletzt aktualisiert: 24. Oktober 2026.",
    "identity": "Phoenix Wire wird von der unten genannten Person betrieben. Fragen und Anfragen, auch zur Löschung, gehen an die unten stehende Kontaktadresse.",
    "lifetime": [
      "Karten mit 24 Stunden Gültigkeit",
      "Der Absender bestätigt seine E-Mail mit einem sechsstelligen Code und bezahlt eine Karte: NIM, USDC auf Polygon oder USDT auf Polygon oder Ethereum. Wire sendet dem Empfänger einen Link. Dieser bestätigt seine eigene E-Mail mit einem weiteren Code und wählt eine Empfangsadresse oder öffnet den üblichen Nimiq-Zahlungslink.",
      "Wire verwahrt nicht deine persönliche Wallet. Jede Karte hat eine eigene Adresse auf der Blockchain, dem öffentlichen Zahlungsverzeichnis.",
      "Eine Karte ist 24 Stunden gültig. Nicht eingelöste NIM werden danach automatisch an die finanzierende Adresse zurückgezahlt. Bei USDC/USDT öffnet der Absender Wire und holt das Geld mit seinem Browserschlüssel zurück. Kam NIM von einer Börse oder einer fremdverwalteten Wallet, geht die Rückzahlung an diese finanzierende Adresse."
    ],
    "fees": [
      "Gebühren und Rückzahlungen",
      "Bei NIM verlangt Wire keine zusätzliche Gebühr neben der Netzwerkgebühr. Bei USDC/USDT: die Netzwerkkosten der Karte bei Erstellung mit einem Aufschlag plus 6 % des Betrags. Der Absender sieht einen Gesamtbetrag. Dieser Aufschlag hilft, Schwankungen der Netzwerkkosten aufzufangen.",
      "Der Weiterleitungsdienst sendet die Kartenzahlung und bezahlt die Netzwerkkosten. Bei einer Rückzahlung behält er die gesamte Gebühr: geschätzte Netzwerkkosten einschließlich Aufschlag plus 6 % des Betrags.",
      "Die Schätzungen nutzen Daten Dritter: Netzwerkknoten für die Netzwerkkosten, Chainlink für Preise und CoinGecko als Ersatz."
    ],
    "risk": [
      "Endgültige Zahlungen und Risiken",
      "Eine eingelöste Zahlung kann niemand rückgängig machen, auch Wire nicht. Prüfe die Empfangsadresse sorgfältig.",
      "Die Software wird wie vorliegend und ohne Gewährleistung bereitgestellt. Es geht um echtes Geld. Eine Codeprüfung ist keine Sicherheitszertifizierung."
    ],
    "access": [
      "Zugang zum Senden",
      "Senden ist nur auf Einladung möglich, sofern ein Administrator den Zugang nicht öffnet. Änderungen am Zugang ändern bereits versandte Karten nicht."
    ],
    "rules": ["Nutzungsregeln", "Die folgenden Regeln gelten für alle, die Phoenix Wire nutzen. Sie stehen am Ende dieser Seite zusammen mit dem Betreiber und dem Kontakt."],
    "storage": [
      "Wie Wire Daten verarbeitet",
      "Wire speichert seinen Hauptzustand in einer verschlüsselten Datei (AES-256-GCM), deren Zugriffsrechte auf den Dateieigentümer beschränkt sind. Die Dateien der Zahlungssoftware sind ebenfalls privat; Dateien mit Geheimnissen sind verschlüsselt. Dies garantiert keine Unzugänglichkeit unter allen Umständen."
    ],
    "rows": [
      [
        "E-Mail des Absenders; Hash des Sitzungstokens",
        "Absenderzugang",
        "Bestätigungscode: 15 Minuten. Sitzung: 24 Stunden. Sitzungen werden bei Ablauf oder Sperrung des Absenders gelöscht."
      ],
      [
        "NIM-Karten: E-Mails von Absender und Empfänger, Betrag, Nachricht bis 160 Zeichen, Sprache, öffentliche Kartenadresse, Status, Daten und dekoratives Bild",
        "Betrieb, Verlauf, E-Mail und Rückzahlung",
        "Abgeschlossene Karten: E-Mail-Adressen und Nachrichten werden 60 Tage nach dem Ende der Karte gelöscht, wenn sie nichts enthält. Unbezahlte Entwürfe: 2 Tage. Stornierte Karten: 14 Tage."
      ],
      [
        "USDC/USDT-Karten: E-Mails von Absender und Empfänger, Betrag, Nachricht, öffentliche Kartenadresse, von Wire gespeichertes Rückzahlungstoken und Daten",
        "Betrieb, Verlauf, E-Mail und Rückzahlung",
        "Wird 60 Tage nach dem Ende der Karte gelöscht, wenn sie nichts enthält."
      ],
      [
        "Mit einem Geheimnis berechneter Hash der Besucher-IP und Zähler",
        "Missbrauch begrenzen",
        "Etwa 1 Stunde."
      ],
      [
        "Listen erlaubter und gesperrter E-Mails",
        "Einladungszugang",
        "Bis zur Entfernung durch einen Administrator."
      ],
      [
        "Aufzeichnung versandter Erinnerungen und Hinweise",
        "Doppelte Hinweise vermeiden",
        "Zusammen mit der Karte."
      ]
    ],
    "engine": [
      "Schlüssel und Zahlungsaufzeichnungen",
      "Die Zahlungssoftware speichert den privaten Schlüssel jeder NIM-Karte, verschlüsselt mit dem Hauptschlüssel, zur Auszahlung oder Rückzahlung. Die USDC/USDT-Mitzeichnungs- und Weiterleitungsschlüssel werden aus dem Hauptschlüssel abgeleitet.",
      "Sie speichert auch ein Journal je NIM-Auszahlung oder Rückzahlung: Zieladresse, signierte Transaktion und Transaktionskennung, ohne Schlüssel. E-Mail-Codes werden als Hash mit Ablaufzeit gespeichert.",
      "Das Margenverzeichnis enthält die ersten acht Zeichen der Kartenkennung, Beträge und Netzwerkkosten je abgeschlossener USDC/USDT-Auszahlung. Keine E-Mail und keine Adresse. Für diese Aufzeichnungen ist hier keine Aufbewahrungsfrist angegeben."
    ],
    "others": [
      "Wer sonst Daten sieht",
      [
        [
          "MailerSend",
          "Absender- und Empfänger-E-Mails, Nachricht, Betrag, Link und Codes zur E-Mail-Zustellung."
        ],
        [
          "Cloudflare",
          "Besucher-IP-Adressen und Anfragen für den Zugang zum Server. Wire nutzt die weitergeleitete Adresse nur zur Missbrauchsbegrenzung als Hash."
        ],
        [
          "Blockchain-Knoten",
          "Abgefragte Karten- und Weiterleitungsadressen für Guthaben, Finanzierung und Netzwerkkosten."
        ],
        [
          "NIM, Polygon und Ethereum",
          "Zahlungen auf ihren Blockchains sind öffentlich und dauerhaft."
        ],
        [
          "Chainlink; CoinGecko als Ersatz",
          "Nur die Währungskennung zur Berechnung der Netzwerkkosten in der Gebühr."
        ],
        [
          "Nimiq-Dienste, Nimpay und Empfangs-Wallets",
          "Deine dortigen Aktionen zum Empfang von NIM. Deren eigene Bedingungen gelten."
        ]
      ]
    ],
    "absent": [
      "Was Wire nicht speichert",
      "Wire hat keine Passwörter. Der USDC/USDT-Linkschlüssel wird nicht gespeichert: Er bleibt im Browser des Absenders und im Empfängerlink. Wire sieht den vollständigen Einlöse-Link einmal zum E-Mail-Versand, speichert ihn aber nicht.",
      "Wire speichert keine IP-Adressen im Klartext, Werbekennungen oder Analysedaten. Öffnungs- und Klickverfolgung ist in jeder E-Mail deaktiviert. Das bedeutet nicht, dass Cloudflare oder MailerSend keine Daten sehen."
    ],
    "deletion": [
      "Löschung anfragen und ihre Grenzen",
      "Wire löscht die E-Mail-Adressen und die Nachricht einer Karte 60 Tage nach ihrem Ende selbst, wenn sie nichts enthält. Eine Karte, die noch Geld enthält, behält ihre Daten, weil ihr Absender sie vielleicht zurückholen muss. Du kannst für eine beendete Karte eine frühere Entfernung verlangen. Zahlungen bleiben auf der Blockchain, und Sicherungen behalten, was sie am Tag ihrer Erstellung enthielten.",
      "Sende Anfragen an die unten angegebene Kontaktadresse. Eine Antwortfrist wird hier nicht zugesagt.",
      "Wire kann weder Blockchain-Daten noch MailerSend-Zustellprotokolle oder Cloudflare-Daten entfernen. Die Fristen dieser Anbieter gelten. Das Entfernen von Kontaktdaten macht keine eingelöste Zahlung rückgängig."
    ]
  },
  "zh": {
    "ui": [
      "使用条款",
      "隐私说明",
      "返回 Phoenix Wire",
      "语言",
      "自动",
      "切换外观",
      "技术支持：",
      "保存的数据",
      "用途",
      "保存期限",
      "运营方与联系方式"
    ],
    "pending": [
      "运营方名称",
      "联系地址",
      "适用法律",
      "最低年龄",
      "禁止的用途"
    ],
    "values": ["Xcoder", "phoenixmails@proton.me", "魁北克（加拿大）法律及在当地适用的加拿大联邦法律。", "18 岁", "任何非法用途：欺诈、洗钱，或为本身违法的商品或服务付款。为用户之间约定的合法商品或服务付款是允许的。"],
    "notice": "最后更新：2026 年 10 月 24 日。",
    "identity": "Phoenix Wire 由下方所列的个人运营。问题和请求（包括删除请求）请发送至下方的联系地址。",
    "lifetime": [
      "有效期为 24 小时的卡片",
      "发送者用六位验证码确认邮箱，然后为卡片付款：NIM、Polygon 上的 USDC，或 Polygon、Ethereum 上的 USDT。Wire 将链接发给接收者。接收者用另一个验证码确认自己的邮箱，再选择收款地址，或打开通常的 Nimiq 付款链接。",
      "Wire 不保管你的个人钱包。每张卡片在区块链上有独立地址；区块链是公开的付款记录。",
      "卡片有效期为 24 小时。未领取的 NIM 随后自动退回为卡片付款的地址。USDC/USDT 的发送者需打开 Wire，使用浏览器中的密钥取回资金。如果 NIM 来自交易平台或托管钱包，退款会返回该付款地址。"
    ],
    "fees": [
      "费用和退款",
      "NIM 除网络费用外，不收取 Wire 费用。USDC/USDT 的费用为创建卡片时估算的网络费用（含余量），加上金额的 6%。发送者会看到一个应付总额。 这部分余量有助于应对网络费用的波动。",
      "转发服务发送卡片付款并支付网络费用。退款时会保留完整的转发费用：预估网络费用（含余量），加上金额的 6%。",
      "估算使用第三方数据：网络节点提供网络费用数据，Chainlink 提供价格，CoinGecko 作为备用来源。"
    ],
    "risk": [
      "不可撤销的付款和风险",
      "款项一旦被领取，任何人（包括 Wire）都无法撤销。请仔细检查收款地址。",
      "软件按现状提供，不作保证。它处理真实资金。代码审查不等于安全认证。"
    ],
    "access": [
      "发送权限",
      "除非管理员开放权限，否则发送需要邀请。权限变化不会改变已经发送的卡片。"
    ],
    "rules": ["使用规则", "以下规则适用于所有使用 Phoenix Wire 的人。它们与运营方和联系方式一起列在本页末尾。"],
    "storage": [
      "Wire 如何处理数据",
      "Wire 将主要状态保存在加密文件（AES-256-GCM）中，文件权限仅允许其所有者访问。支付引擎文件也属于私有文件；其中含有秘密的文件经过加密。这并不保证数据在任何情况下都无法被访问。"
    ],
    "rows": [
      [
        "发送者邮箱；会话令牌的哈希值",
        "发送者访问",
        "确认码：15 分钟。登录会话：24 小时。会话到期或发送者被封锁时删除。"
      ],
      [
        "NIM 卡片：发送者及接收者邮箱、金额、最多 160 字符的留言、语言、卡片公开地址、状态、日期和装饰图片",
        "卡片运行、历史记录、邮件和退款",
        "已结束的卡片：若卡片中没有资金，邮箱地址和留言会在卡片结束后 60 天删除。未付款草稿：2 天。已取消卡片：14 天。"
      ],
      [
        "USDC/USDT 卡片：发送者及接收者邮箱、金额、留言、卡片公开地址、Wire 保存的退款令牌和日期",
        "卡片运行、历史记录、邮件和退款",
        "若卡片中没有资金，卡片结束后 60 天删除。"
      ],
      [
        "用秘密值计算的访客 IP 地址哈希及计数",
        "防止滥用",
        "大约 1 小时。"
      ],
      [
        "获准和被封锁的邮箱列表",
        "管理邀请权限",
        "直到管理员移除。"
      ],
      [
        "已发送的提醒和通知记录",
        "避免重复发送同一通知",
        "随卡片保存。"
      ]
    ],
    "engine": [
      "密钥和付款记录",
      "支付引擎保存每张 NIM 卡片的私钥，并用主密钥加密，以便向接收者付款或退款。USDC/USDT 的共同签名和转发密钥由主密钥派生。",
      "引擎还为每笔 NIM 付款或退款保存日志：目标地址、已签名的交易和交易标识，不含密钥。邮箱验证码以哈希及到期时间的形式保存。",
      "利润记录包含每笔已结算 USDC/USDT 付款的卡片标识前八个字符、金额和网络费用，不含邮箱或地址。此处未规定这些引擎记录的保存期限。"
    ],
    "others": [
      "其他哪些方能看到数据",
      [
        [
          "MailerSend",
          "发送者及接收者邮箱、留言、金额、链接和验证码，用于投递邮件。"
        ],
        [
          "Cloudflare",
          "访客 IP 地址和请求，用于连接服务器。Wire 仅将转发的访客地址以哈希形式用于限制滥用。"
        ],
        [
          "区块链节点",
          "为查询余额、入金和网络费用而查询的卡片及转发地址。"
        ],
        [
          "NIM、Polygon 和 Ethereum",
          "这些区块链上的付款公开且永久保存。"
        ],
        [
          "Chainlink；备用 CoinGecko",
          "仅币种标识，用于计算费用中的网络成本。"
        ],
        [
          "Nimiq 服务、Nimpay 和收款钱包",
          "你使用它们接收 NIM 时的操作。适用它们自己的条款。"
        ]
      ]
    ],
    "absent": [
      "Wire 不保存什么",
      "Wire 没有密码。不保存 USDC/USDT 链接密钥：密钥留在发送者浏览器和接收者链接中。Wire 为发送邮件会看见一次完整领取链接，但不保存该链接。",
      "Wire 不保存明文 IP 地址、广告标识或分析数据。每封邮件的打开及点击跟踪都已关闭。这不表示 Cloudflare 或 MailerSend 看不到数据。"
    ],
    "deletion": [
      "请求删除及其限制",
      "若卡片中没有资金，Wire 会在卡片结束 60 天后自行删除其邮箱地址和留言。仍有资金的卡片会保留其数据，因为发送者可能需要取回。对于已结束的卡片，你可以要求更早删除。付款永久保留在区块链上，备份保留其制作当天的内容。",
      "请向下方联系地址发送请求。此处不承诺回复期限。",
      "Wire 无法移除区块链记录、MailerSend 投递日志或 Cloudflare 数据。适用这些服务方自己的保存期限。移除联系数据不能撤销已领取的付款。"
    ]
  }
};

// Every unresolved public field is listed here and in PAYMENT-WINDOW.md.
export const ownerFields = {
 terms: ['operator', 'contact', 'law', 'minimum-age', 'prohibited-uses'],
 privacy: ['operator', 'contact']
};
const fieldIds = ['operator', 'contact', 'law', 'minimum-age', 'prohibited-uses'];
const languages = ['fr', 'en', 'es', 'de', 'zh'];
export function legalLanguage(value, preferences = []) {
 if (languages.includes(value)) return value;
 for (const preference of preferences) {
  const short = String(preference).toLowerCase().split(/[-_]/)[0];
  if (languages.includes(short)) return short;
 }
 return 'en';
}
function node(doc, tag, text, className) {
 const element = doc.createElement(tag);
 if (text !== undefined) element.textContent = text;
 if (className) element.className = className;
 return element;
}
export function paintLegalLinks(doc, language) {
 const words = legalCopy[legalLanguage(language)].ui;
 for (const link of doc.querySelectorAll('[data-legal-link]')) {
  const page = link.dataset.legalLink;
  if (page !== 'terms' && page !== 'privacy') continue;
  const label = words[page === 'terms' ? 0 : 1];
  if (link.textContent !== label) link.textContent = label;
  const nav = link.closest('nav'), navLabel = words[0] + ' / ' + words[1];
  if (nav && nav.getAttribute('aria-label') !== navLabel) nav.setAttribute('aria-label', navLabel);
  // Only the chosen language is carried over. Never forward card URLs or fragments.
  const href = '/' + page + '.html?lang=' + legalLanguage(language);
  if (link.getAttribute('href') !== href) link.setAttribute('href', href);
 }
 for (const template of doc.querySelectorAll('template')) paintLegalLinks(template.content, language);
 for (const powered of doc.querySelectorAll('[data-legal-powered]')) {
  const label = words[6] + ' ';
  if (powered.textContent !== label) powered.textContent = label;
 }
}
export function renderLegal(doc, page, language) {
 if (!Object.hasOwn(ownerFields, page)) return;
 const lang = legalLanguage(language), copy = legalCopy[lang], article = doc.getElementById('legal-content');
 if (!article) return;
 doc.documentElement.lang = lang;
 doc.title = copy.ui[page === 'terms' ? 0 : 1] + ' · Phoenix Wire';
 const heading = doc.getElementById('legal-title');
 if (heading) heading.textContent = copy.ui[page === 'terms' ? 0 : 1];
 const notice = doc.getElementById('legal-notice');
 if (notice) notice.textContent = copy.notice;
 const blocks = [];
 function section(parts, id) {
  const block = node(doc, 'section');
  if (id) block.id = id;
  block.append(node(doc, 'h2', parts[0]));
  for (const text of parts.slice(1)) block.append(node(doc, 'p', text));
  blocks.push(block);
 }
 if (page === 'terms') {
  section(copy.lifetime, 'cards'); section(copy.fees, 'fees');
  section(copy.risk, 'risk'); section(copy.access, 'access'); section(copy.rules, 'rules');
 } else {
  section(copy.storage, 'storage');
  const block = node(doc, 'section'), wrap = node(doc, 'div', undefined, 'legal-table-wrap');
  const table = node(doc, 'table'), head = node(doc, 'thead'), row = node(doc, 'tr');
  table.append(node(doc, 'caption', copy.ui[7]));
  for (const text of copy.ui.slice(7, 10)) {const cell = node(doc, 'th', text);cell.scope = 'col';row.append(cell);}
  head.append(row);table.append(head);
  const body = node(doc, 'tbody');
  for (const fields of copy.rows) {
   const row = node(doc, 'tr');
   fields.forEach((text, i) => {const cell = node(doc, i ? 'td' : 'th', text);if (!i) cell.scope = 'row';row.append(cell);});
   body.append(row);
  }
  table.append(body);wrap.append(table);block.append(wrap);blocks.push(block);
  section(copy.engine, 'engine');
  const recipients = node(doc, 'section');recipients.id = 'recipients';recipients.append(node(doc, 'h2', copy.others[0]));
  const list = node(doc, 'dl');
  for (const [name, text] of copy.others[1]) list.append(node(doc, 'dt', name), node(doc, 'dd', text));
  recipients.append(list);blocks.push(recipients);
  section(copy.absent, 'not-kept'); section(copy.deletion, 'deletion');
 }
 section([copy.ui[10], copy.identity], 'owner-decisions');
 const pending = blocks.at(-1); // the closing section: operator and contact, then the rules of use
 for (const id of ownerFields[page]) {
  const value = copy.values[fieldIds.indexOf(id)];
  const p = node(doc, 'p', undefined, 'legal-owner');p.id = 'owner-' + id;p.dataset.ownerField = id;
  let shown = node(doc, 'span', value);
  if (id === 'contact') {shown = node(doc, 'a', value);shown.href = 'mailto:' + value;}
  p.append(node(doc, 'strong', copy.pending[fieldIds.indexOf(id)] + ': '), shown);
  pending.append(p);
 }
 article.replaceChildren(...blocks);
 const back = doc.getElementById('legal-back');if (back) back.textContent = copy.ui[2];
 const picker = doc.getElementById('wire-language');
 if (picker) {picker.setAttribute('aria-label', copy.ui[3]);picker.querySelector('[value="auto"]').textContent = copy.ui[4];}
 const theme = doc.getElementById('legal-theme');if (theme) theme.textContent = copy.ui[5];
 const powered = doc.querySelector('[data-legal-powered]');if (powered) powered.textContent = copy.ui[6] + ' ';
 paintLegalLinks(doc, lang);
}
export function mountLegal(doc, win) {
 const read = (key, fallback) => {try {return win.localStorage.getItem(key) || fallback;} catch {return fallback;}};
 const save = (key, value) => {try {win.localStorage.setItem(key, value);} catch {}};
 const page = doc.body.dataset.legalPage;
 if (Object.hasOwn(ownerFields, page || '')) {
  const picker = doc.getElementById('wire-language');
  const query = new URLSearchParams(win.location.search).get('lang');
  let choice = languages.includes(query) ? query : read('wire-language', 'auto');
  if (choice !== 'auto' && !languages.includes(choice)) choice = 'auto';
  const preferences = win.navigator.languages?.length ? win.navigator.languages : [win.navigator.language];
  function paint() {renderLegal(doc, page, legalLanguage(choice, preferences));picker.value = choice;}
  picker.addEventListener('change', () => {choice = languages.includes(picker.value) ? picker.value : 'auto';save('wire-language', choice);paint();});
  const palette = read('wire-palette', 'phoenixfire');
  doc.documentElement.dataset.palette = ['phoenixfire', 'aurora', 'ocean'].includes(palette) ? palette : 'phoenixfire';
  let theme = read('wire-theme', 'dark') === 'light' ? 'light-paper' : 'dark';
  doc.documentElement.dataset.theme = theme;
  doc.getElementById('legal-theme').addEventListener('click', () => {
   theme = theme === 'dark' ? 'light-paper' : 'dark';doc.documentElement.dataset.theme = theme;save('wire-theme', theme === 'dark' ? 'dark' : 'light');
  });
  paint();
 } else {
  paintLegalLinks(doc, doc.documentElement.lang);
  new win.MutationObserver(records => {
   if (records.some(record => record.type === 'attributes' || [...record.addedNodes].some(child =>
    child.nodeType === 1 && (child.matches('[data-legal-link]') || child.querySelector('[data-legal-link]')))))
    paintLegalLinks(doc, doc.documentElement.lang);
  }).observe(doc.documentElement, {attributes: true, attributeFilter: ['lang'], childList: true, subtree: true});
 }
}
if (typeof document !== 'undefined' && typeof window !== 'undefined') mountLegal(document, window);
