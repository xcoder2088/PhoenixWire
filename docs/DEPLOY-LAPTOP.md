# Déployer Phoenix Wire sur le portable (sans the maintainer)

> Le guide pas à pas, avec toutes les commandes à copier, est [DEPLOIEMENT-SIMPLE.md](DEPLOIEMENT-SIMPLE.md). Cette page garde les détails (où est quoi, adresses des relais, transfert des fonds).

Le site `https://wire.phoenixmail.ca` est servi par le service `phoenixwire` du portable (`user@host`),
programme `/opt/phoenixwire/phoenixwire`. Déployer = remplacer ce programme par une version neuve et redémarrer le service.
Le script le fait en une commande. **Il ne touche ni aux clés, ni à la configuration, ni aux données.**

## Déployer une nouvelle version

Sur ton ordinateur, dans le dossier du projet (au même réseau que le portable) :

```bash
cd ~/Desktop/PhoenixWire
git pull
./scripts/deploy-laptop.sh
```

Le script : affiche la version, lance les tests, construit le programme, l'envoie au portable, vérifie son empreinte,
garde l'ancien (`phoenixwire.prev`), redémarre le service et vérifie que le site répond. **Si le service ne démarre pas,
il remet tout seul l'ancienne version.**

Options :

| Commande | Effet |
|---|---|
| `./scripts/deploy-laptop.sh --dry-run` | tests et construction seulement, rien n'est envoyé |
| `./scripts/deploy-laptop.sh --skip-tests` | déploie sans les tests (à éviter) |
| `./scripts/deploy-laptop.sh --rollback` | remet la version précédente |

Il faut Go sur ton ordinateur (Node aussi pour les tests JavaScript ; sans Node ils sont sautés). Pour un autre serveur :
`WIRE_LAPTOP=utilisateur@adresse ./scripts/deploy-laptop.sh`.

### Si ça échoue

- `Permission denied` / `Connection timed out` : tu n'es pas sur le même réseau que le portable, ou ta clé ssh n'est pas chargée.
- Un test échoue : ne déploie pas. Regarde le message, ou demande à the maintainer de corriger.
- Le site ne répond plus après le déploiement : `./scripts/deploy-laptop.sh --rollback`.
- Voir ce que dit le service (sur le portable) : `sudo journalctl -u phoenixwire -n 50 --no-pager`

## Les trois commandes à connaître sur le portable

```bash
systemctl is-active phoenixwire             # doit répondre : active
sudo systemctl restart phoenixwire          # redémarrer
sudo journalctl -u phoenixwire -n 50 --no-pager   # les derniers messages
```

## Ce qu'il y a où (portable)

| Quoi | Où |
|---|---|
| Programme | `/opt/phoenixwire/phoenixwire` (et `.prev` : la version d'avant) |
| Configuration (courriel, clés d'API, réglages) | `/etc/phoenixwire/local.env` (réservé à root, ne jamais l'afficher ni le mettre dans Git) |
| **Clé principale** | `/etc/phoenixwire/master.key` |
| Données (cartes, clés des relais, sessions) | `/var/lib/phoenixwire/` |

La clé principale et les données vont ensemble : **sans elles, les fonds des relais sont perdus.** Une mise à jour du
programme ne les touche jamais.

## Sauvegarde chiffrée (à faire avant de mettre de l'argent sur les relais)

Sur le portable (te demande une phrase secrète deux fois ; garde-la dans ton gestionnaire de mots de passe) :

```bash
(umask 077; sudo tar czf - /etc/phoenixwire /var/lib/phoenixwire | openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -salt -out ~/wire-backup.enc)
```

Vérifier qu'elle se relit (doit lister des fichiers, dont `master.key`) :

```bash
openssl enc -d -aes-256-cbc -pbkdf2 -iter 600000 -in ~/wire-backup.enc | tar tz
```

Puis copier `~/wire-backup.enc` hors du portable (clé USB) et supprimer l'original : `shred -u ~/wire-backup.enc`.

## Les relais de Wire (adresses publiques)

- Polygon (USDC et USDT Polygon), en POL : `0x6a24200f9ce9F10a68e2F901424e6F946919e8EE` (le plancher suit le prix du gaz : de 0,1 à 0,2 POL pour accepter des cartes)
- Ethereum (USDT Ethereum), en ETH : `0xcb641A54c9325B555cA2B7DB53143C9952e6bB85` (le plancher suit le prix du gaz : de 0,0005 à 0,01 ETH ; au gaz actuel, une fraction de milliETH suffit)

Le gaz gratuit pour l'expéditeur est activé par `NIMIQ_CLAIM_USDTLINK_GAS=on` dans `/etc/phoenixwire/local.env`
(limites du moteur : au plus 0,3 POL par carte, au prix du moment, une fois par carte, 2 par heure, 10 par jour ; le frais de la carte rembourse ce don). Après un changement de ce fichier :
`sudo systemctl restart phoenixwire`.

## Transférer les fonds du relais PhoenixMail vers celui de Wire

1. Sur le portable, autoriser les deux adresses ci-dessus comme destinations de retrait du service PhoenixMail :

   ```bash
   sudo cp -p /etc/nimiqsecureclaim/secureclaim.env /etc/nimiqsecureclaim/secureclaim.env.bak-withdraw
   echo "NIMIQ_CLAIM_USDTLINK_WITHDRAW_TO=0x6a24200f9ce9F10a68e2F901424e6F946919e8EE,0xcb641A54c9325B555cA2B7DB53143C9952e6bB85" | sudo tee -a /etc/nimiqsecureclaim/secureclaim.env >/dev/null
   sudo systemctl restart nimiq-secureclaim && sleep 5 && systemctl is-active nimiq-secureclaim
   ```

2. Dans PhoenixMail : enregistrer une passkey (Paramètres → Gérer ma sécurité), puis, dans le panneau admin « Relais de paiement »,
   retirer vers ces adresses (Polygon vers la première, Ethereum vers la seconde). Le panneau laisse la réserve nécessaire
   aux cartes en cours ; chaque retrait demande la passkey.

## Plus tard : vers un serveur Oracle

Quand Wire déménagera sur Oracle : même programme, mêmes fichiers `/etc/phoenixwire` et `/var/lib/phoenixwire` (restaurés
depuis la sauvegarde chiffrée), mêmes adresses de relais. Le script accepte `WIRE_LAPTOP=utilisateur@adresse`.
