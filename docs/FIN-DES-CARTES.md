# La fin d'une carte (24 heures)

Toute carte, NIM, USDC ou USDT, vit **24 heures** à partir de sa création.

| | NIM | USDC / USDT |
|---|---|---|
| Une heure avant la fin | courriel de rappel à l'**expéditeur**, une seule fois, si la carte contient encore l'argent | idem |
| À la fin | Wire **rembourse tout seul** (voir plus bas) | courriel « ta carte a expiré, l'argent est toujours dessus » ; l'expéditeur le reprend avec Phoenix Wire |
| Après la fin | le destinataire ne peut plus ouvrir la carte | idem |

## NIM : remboursement automatique

- Un balayage tourne toutes les 5 minutes (`internal/wire/expiry.go`).
- Il paie le contenu de la carte à l'**adresse qui l'a financée** (l'expéditeur de la transaction de financement, lue sur la chaîne), puis écrit le paiement dans un journal **avant** de l'envoyer. Un second passage ne peut que répéter ce même paiement.
- Il ne touche **pas** une carte vide, ni une carte que le destinataire a déjà ouverte (son lien lui a été remis), ni une carte dont il a commencé le paiement vers une adresse.
- Étapes visibles dans l'historique de l'expéditeur : `refunding` (envoyé, pas encore dans un bloc) puis `refunded`. Une carte `refunded` ne peut plus être ouverte.
- Un NIM n'a pas de frais de service : tout le contenu revient.
- Risque connu : si l'expéditeur a payé depuis un compte d'échange, le remboursement retourne à l'adresse de cet échange.

## USDC / USDT : pourquoi pas automatique

La carte est un coffre à deux clés : celle du lien (dans le navigateur de l'expéditeur, puis dans le courriel du destinataire) et celle de Wire, qui ne signe qu'après le code reçu par courriel. Wire seul ne peut donc pas renvoyer l'argent. L'expéditeur le reprend avec la clé de son navigateur (« Reprendre mon argent »). Le relais **garde son frais** (gaz et 6 %).

## Courriels à l'expéditeur

Trois avis (`internal/cards/notice.go`), en 5 langues, une fois chacun : `reminder` (une heure avant), `refunded` (NIM remboursé), `expired` (USDC/USDT expirée). Le seul lien est celui de Phoenix Wire.
