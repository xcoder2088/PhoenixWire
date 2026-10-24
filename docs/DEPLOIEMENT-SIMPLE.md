# Déployer Phoenix Wire sur le portable : le guide simple

Le site `https://wire.phoenixmail.ca` tourne sur le portable (`UTILISATEUR@PORTABLE`). **Déployer = envoyer une nouvelle version du programme et le redémarrer.** Le script fait tout. Il ne touche ni aux clés, ni à la configuration, ni aux données.

## 0. Une seule fois : dire au script où déployer

Le dépôt ne contient aucune adresse personnelle. Crée ce fichier (droits privés) avec **ton** portable et **ton** site :

```bash
mkdir -p ~/.config/phoenixwire && printf 'WIRE_LAPTOP=UTILISATEUR@PORTABLE\nWIRE_SITE=https://ton-site.example\n' > ~/.config/phoenixwire/deploy.env && chmod 600 ~/.config/phoenixwire/deploy.env
```

(`UTILISATEUR@PORTABLE` : le compte ssh et l'adresse de la machine qui sert Wire.) Sans ce fichier, le script refuse de partir et le dit.

## 1. Déployer (le cas normal)

Sur ton ordinateur, dans un terminal :

```bash
cd ~/chemin/vers/PhoenixWire
```

```bash
git pull
```

```bash
./scripts/deploy-laptop.sh
```

Le script : lance les tests, construit le programme, l'envoie au portable, le redémarre, puis **vérifie que le site répond et que chaque fichier de la page est servi**.

À la fin, tu dois voir :

```
phoenixwire: active
...
Done: <numéro de version> is live.
```

**Si une question « Continue? [y/N] » apparaît** (il y a des changements non enregistrés dans le dossier) : réponds `y` seulement si tu sais pourquoi, sinon `N`.

**Si un test échoue** : le script s'arrête avant d'envoyer quoi que ce soit. Rien n'est cassé. Copie le message d'erreur et ouvre un ticket (ou demande au mainteneur).

## 2. Vérifier que ça marche

```bash
curl -s https://wire.phoenixmail.ca/api/config
```

Tu dois voir `"payments_enabled":true`. Puis ouvre `https://wire.phoenixmail.ca` dans le navigateur (Ctrl+Maj+R pour forcer le rechargement) : la page s'affiche, sans zone vide.

Pour voir ce que fait le service (sur le portable) :

```bash
sudo journalctl -u phoenixwire -n 50 --no-pager
```

## 3. Revenir en arrière si ça va mal

Remet la version précédente (le script la garde toujours) :

```bash
./scripts/deploy-laptop.sh --rollback
```

## 4. Options du script

| Commande | Effet |
|---|---|
| `./scripts/deploy-laptop.sh --dry-run` | tests et construction seulement, rien n'est envoyé |
| `./scripts/deploy-laptop.sh --skip-tests` | déploie sans les tests (à éviter) |
| `./scripts/deploy-laptop.sh --rollback` | remet la version précédente |

## 5. Se connecter au portable

```bash
ssh UTILISATEUR@PORTABLE
```

Trois commandes utiles une fois connecté :

```bash
systemctl is-active phoenixwire
```
(doit répondre `active`)

```bash
sudo systemctl restart phoenixwire
```
(redémarre le service ; attends 6 secondes)

```bash
sudo journalctl -u phoenixwire -n 50 --no-pager
```
(les 50 derniers messages)

## 6. Changer un réglage (fichier de configuration)

La configuration est dans `/etc/phoenixwire/local.env` (réservée à root, **ne jamais l'afficher ni la mettre dans Git**). Toujours faire une copie avant :

```bash
sudo cp -p /etc/phoenixwire/local.env /etc/phoenixwire/local.env.bak
```

Ajouter un réglage (exemple : un administrateur de plus) :

```bash
echo "PHOENIXWIRE_ADMINS=adresse1@exemple.com,adresse2@exemple.com" | sudo tee -a /etc/phoenixwire/local.env >/dev/null
```

Puis redémarrer :

```bash
sudo systemctl restart phoenixwire && sleep 6 && systemctl is-active phoenixwire
```

Réglages courants :

| Réglage | Effet |
|---|---|
| `PHOENIXWIRE_ADMINS=a@x.com,b@y.com` | ces adresses sont administrateurs (panneau « Administration ») |
| `PHOENIXWIRE_SENDERS=a@x.com,b@y.com` | ces adresses peuvent envoyer, en plus des inscrits |
| `PHOENIXWIRE_SENDERS=open` | tout le monde peut envoyer (à éviter : l'interrupteur du panneau est plus sûr) |
| `NIMIQ_CLAIM_USDTLINK_FEE_BPS=700` | frais de service à 7 % (600 = 6 %) |
| `NIMIQ_CLAIM_USDTLINK_FEE_MARGIN=2` | marge sur le gaz (1,5 par défaut) |
| `NIMIQ_CLAIM_USDTLINK_COINGECKO=off` | ne pas interroger CoinGecko pour le prix |

Si le service ne repart pas après un changement : remets la copie.

```bash
sudo cp -p /etc/phoenixwire/local.env.bak /etc/phoenixwire/local.env && sudo systemctl restart phoenixwire
```

## 7. Sauvegarde chiffrée (à refaire de temps en temps)

Sur le portable. La commande demande ta phrase secrète deux fois (rien ne s'affiche en la tapant) :

```bash
read -rs -p "Phrase secrète : " P; echo; read -rs -p "Retape-la : " Q; echo; if [ "$P" = "$Q" ] && [ -n "$P" ]; then (umask 077; sudo tar czf - /etc/phoenixwire /var/lib/phoenixwire 2>/dev/null | P="$P" openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -salt -pass env:P -out ~/wire-backup.enc) && echo "Sauvegarde créée." ; else echo "Les deux phrases ne correspondent pas."; fi; unset P Q
```

Vérifier qu'elle se relit (doit lister des fichiers, dont `etc/phoenixwire/master.key`) :

```bash
read -rs -p "Phrase secrète : " P; echo; P="$P" openssl enc -d -aes-256-cbc -pbkdf2 -iter 600000 -pass env:P -in ~/wire-backup.enc | tar tz | head -20; unset P
```

Ensuite : copie `~/wire-backup.enc` hors du portable (autre ordinateur, stockage en ligne), puis supprime l'original :

```bash
shred -u ~/wire-backup.enc
```

**Sans la clé principale et les données, les fonds des relais sont perdus.** Garde la phrase secrète dans ton gestionnaire de mots de passe, séparée du fichier.

## 8. Où sont les choses (portable)

| Quoi | Où |
|---|---|
| Programme | `/opt/phoenixwire/phoenixwire` (et `.prev` : la version d'avant) |
| Configuration | `/etc/phoenixwire/local.env` |
| **Clé principale** | `/etc/phoenixwire/master.key` |
| Données (cartes, clés des relais, journaux) | `/var/lib/phoenixwire/` |
| Registre de marge (gains et coûts) | `/var/lib/phoenixwire/engine/usdtlink/ledger.jsonl` |

Lire le registre de marge (une ligne par paiement) :

```bash
sudo journalctl -u phoenixwire --no-pager | grep "ledger"
```

## 9. Le site de présentation (phoenixmail.ca) : déploiement séparé

Ce n'est pas le même programme : ce sont des fichiers statiques du dépôt PhoenixMail (`website/public/`), copiés dans `/var/www/phoenixmail/public/` du portable. Le mainteneur le fait ; en cas de besoin, copie `index.html`, `site.css`, `site.js` (et les images ajoutées) dans ce dossier avec `sudo install -o root -g root -m 644`, après avoir gardé une copie dans un dossier `.backup-<date>`. Cloudflare garde l'ancienne page jusqu'à 4 heures : *Caching → Purge Everything* pour tout voir tout de suite.

## 10. Compiler soi-même

Le script compile pour toi. À la main, depuis le dossier du projet :

```bash
CGO_ENABLED=0 go build -trimpath -o phoenixwire ./cmd/phoenixwire
```

Le projet impose le compilateur **Go 1.26.10 ou plus récent** (ligne `toolchain` de `go.mod`) : si ton Go est plus ancien, il télécharge tout seul le bon. C'est important : la bibliothèque standard de Go reçoit des correctifs de sécurité (surtout le TLS), et un programme compilé avec une version ancienne les garde en défaut.

Vérifier avec quel Go un programme a été compilé, puis s'il a des failles connues :

```bash
go version -m ./phoenixwire | head -1
```

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest -mode=binary ./phoenixwire
```

La deuxième commande doit répondre `No vulnerabilities found.` À refaire de temps en temps, et à chaque nouvelle version de Go (nouveau `toolchain` dans `go.mod`, puis déploiement). Le script affiche `built with: go…` à chaque déploiement.

## 11. Problèmes courants

| Symptôme | Cause probable | Quoi faire |
|---|---|---|
| `Permission denied` / `Connection timed out` | pas sur le même réseau que le portable, ou clé ssh non chargée | se brancher au bon réseau ; `ssh-add` |
| Page vide après déploiement | un fichier de la page n'est pas servi | le script l'affiche (`MISSING ...`) ; sinon `--rollback` puis prévenir le mainteneur |
| `phoenixwire` pas `active` | erreur de configuration ou de permissions | `sudo journalctl -u phoenixwire -n 50 --no-pager`, puis remettre la copie de la configuration |
| Un test échoue | changement incomplet | ne pas déployer ; envoyer le message au mainteneur |
