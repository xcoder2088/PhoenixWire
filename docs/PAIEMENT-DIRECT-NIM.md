# Recevoir une carte NIM directement dans un portefeuille

Jusqu'ici, une carte NIM s'ouvrait par un lien Hub (`hub.nimiq.com/cashlink/#…`) : le destinataire devait passer par le Hub.
Nouveau : après le code reçu par courriel, il choisit **où** recevoir l'argent.

| Choix | Ce qui se passe |
|---|---|
| **Recevoir dans Nimiq Pay** (mini-app) ou **dans un portefeuille** (adresse collée) | Le serveur signe une seule transaction qui envoie tout le contenu de la carte à cette adresse. La clé de la carte ne sort jamais du serveur. |
| **Ouvrir avec le Hub** (comme avant) | `release` : le lien Hub est renvoyé, rien d'autre ne change. |

Les deux se font avec le même code, mais **un seul choix par carte** : dès qu'un paiement direct a commencé, le lien Hub est refusé
(`409 payout_already_started`), et inversement un code déjà utilisé pour le Hub ne sert plus (`409`).

## Contrat d'API (passerelle Wire, `POST`, sans session)

`POST /api/claims/<id>/payout` — corps `{"code":"123456","to":"NQ28 CQ7G …"}`

- Première fois : le code est obligatoire. Casse et espaces de l'adresse n'ont pas d'importance.
- Ensuite, **même adresse, sans code** : renvoie l'état et, si le nœud a perdu la transaction, la renvoie telle quelle.
  Utile pour reprendre après une coupure réseau ou un rechargement de la page.
- Une autre adresse que la première : `409 payout_already_started`. Jamais de second paiement.

`POST /api/claims/<id>/payout_status` — corps `{}` : où en est le paiement (`404 payout_not_started` s'il n'y en a pas).

Réponse (200) : `{"status","hash","to","value","confirmations"}`, `value` en luna (1 NIM = 100 000).

| `status` | Sens | Affichage |
|---|---|---|
| `payout_pending` | transaction envoyée, pas encore dans un bloc | « Envoi en cours… » ; redemander `payout_status` toutes les 3 s |
| `payout_confirmed` | dans un bloc, exécutée | « Reçu ! » |
| `payout_failed` | le réseau a refusé l'exécution (rare) | « Un problème est survenu », écrire au support ; l'argent reste dans la carte |

Erreurs : `400 invalid_destination` (adresse invalide, adresse brûlée, ou adresse de la carte elle-même), `403 invalid_or_expired_code`,
`409 cashlink_not_confirmed_or_invalid` (le code **n'est pas** consommé), `409 payout_already_started`, `503 chain_unavailable` /
`payout_unavailable` (réessayer).

## Pour l'interface (the maintainer)

1. Après la saisie du code, **avant** d'appeler `release` : proposer le choix « Recevoir dans Nimiq Pay » / « Recevoir dans un autre
   portefeuille » / « Ouvrir avec le Hub ». Le code ne doit être envoyé qu'au moment du choix (il ne sert qu'une fois).
2. Dans le navigateur interne de Nimiq Pay (`window.nimiq` présent) : `await window.nimiq.listAccounts()` renvoie les adresses du
   portefeuille ; prendre la première (ou laisser choisir s'il y en a plusieurs), puis `payout` avec `to`.
3. Ailleurs : champ « adresse Nimiq » (validation souple côté page, la vraie validation est sur le serveur).
4. Garder `id` + adresse choisie en mémoire de la page ; au rechargement, `payout_status` dit si un paiement est déjà en route.
5. Textes en FR/EN/ES/DE/ZH ; jamais « Mainnet » ni « OTP ».

## Côté serveur

- `engine/secureclaim/internal/nimiq/tx.go` : transaction « basic » du réseau (format validé contre le nœud : 139 octets).
- `engine/secureclaim/service/cashlink_payout.go` : l'action, le journal et le suivi.
- Journal : `<état>/payouts/<id>.json` (droits 0600, **aucun secret** : destination, transaction signée, hash). Écrit **avant** l'envoi ;
  c'est lui qui garantit « une seule destination, une seule transaction ». À sauvegarder avec le reste de l'état.
- La carte envoie tout son solde (frais 0). Si la transaction n'a pas été incluse au bout de ~7000 blocs et que la carte contient encore
  l'argent, elle est resignée pour **la même** adresse.
- Changement de principe : le moteur ne se contentait pas de garder la clé, il peut maintenant **signer** — uniquement ce paiement, uniquement
  après le code du courriel.
