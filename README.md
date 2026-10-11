# LoL Remote

Télécommande mobile pour le client League of Legends : accepter les parties, choisir et bannir son champion, gérer ses pages de runes et suivre la draft depuis son téléphone.

Réécriture en Go du script Python original (voir « Historique » plus bas) : un exécutable Windows unique, sans Python ni dépendance à installer.

## Installation

Télécharger `lol-remote.exe` depuis la page [Releases](../../releases) et le lancer (double-clic). Aucune console ne s'ouvre : une icône apparaît dans la zone de notification, avec une fenêtre qui affiche l'adresse et le code PIN à saisir sur le téléphone.

Autoriser l'exécutable dans le pare-feu Windows (réseaux privés) au premier lancement.

## Architecture

```
Téléphone (PWA)  ──HTTP/HTTPS──▶  lol-remote.exe (PC, port 53217)  ──HTTPS local──▶  Client League (LCU API)
```

Le client League expose une API REST non officielle, la **LCU API**, qui n'écoute que sur `127.0.0.1`. Le téléphone ne peut donc pas lui parler directement : `lol-remote.exe` sert de relais.

### Connexion à la LCU

- Le port et le jeton d'authentification du client sont lus directement sur le processus `LeagueClientUx.exe` (ligne de commande, via PowerShell), ce qui fonctionne quel que soit le dossier d'installation.
- En variante, la variable d'environnement `LOL_REMOTE_LOCKFILE` peut pointer vers le `lockfile` du client si on préfère cette méthode.
- Authentification en Basic Auth, utilisateur `riot`. Le certificat du client est auto-signé, donc la vérification SSL est désactivée pour ces requêtes.
- Le port et le mot de passe changent à chaque redémarrage du client ; une erreur 401/403 ou de connexion déclenche une relecture automatique des identifiants.

## Lancement

```
lol-remote.exe
```

Une fenêtre affiche l'adresse et le code PIN au démarrage ; l'icône dans la zone de notification permet de les revoir à tout moment, d'ouvrir un code QR vers l'adresse, ou de quitter.

Le PIN est aléatoire à chaque lancement, sauf si la variable d'environnement `LOL_REMOTE_PIN` est définie. Le port par défaut est `53217` ; s'il est déjà pris, un port libre est choisi automatiquement.

### Accès hors du Wi-Fi (optionnel)

Avec Tailscale sur le PC et le téléphone, `tailscale serve --bg 53217` rend l'app accessible en `https://<pc>.<tailnet>.ts.net`, y compris en 4G (adapter le port si l'app en a choisi un autre).

## Fonctionnalités

**Partie et sélection**
- Écran « Partie trouvée » avec boutons Accepter / Refuser, plus une option d'acceptation automatique.
- Grille des champions avec recherche. Les champions non jouables ou déjà pris sont grisés.
- Survoler ou verrouiller un pick. Phase de ban gérée (bouton « Bannir »).
- Chronomètre de la phase, en rouge sous 10 secondes.
- Aperçu de la draft : picks et hovers alliés (hover en transparence), picks ennemis, bans des deux équipes, rôles. Les pseudos s'affichent seulement quand le client les fournit (Riot les masque en classée).

**Runes**
- Liste des pages, activation en un tap, suppression des pages modifiables.
- Éditeur complet : arbre principal et clé de voûte, 3 lignes principales, arbre secondaire (2 runes sur 2 lignes différentes), 3 lignes de fragments.
- Runes recommandées par le client pour le champion choisi et le rôle assigné. « Appliquer » écrase la page active si elle est modifiable, sinon crée une nouvelle page.

**Web app**
- Manifest et icône générés en PNG au démarrage : l'app s'ajoute à l'écran d'accueil et s'ouvre en plein écran.

**Sur le PC**
- Pas de console : une icône dans la zone de notification, menu « Adresse et code PIN », « Code QR » (ouvre un PNG du QR dans la visionneuse par défaut) et « Quitter ».

## Structure du projet

Architecture en couches (clean architecture) :

| Package | Rôle |
|---|---|
| `internal/league` | Cas d'usage : `Service`, modèles de données, interface `Client` (le port), erreurs métier. Ne dépend de rien d'autre dans ce projet. |
| `internal/lcu` | Adaptateur : implémente `league.Client` par-dessus l'API LCU réelle (HTTPS, découverte des identifiants). |
| `internal/icon` | Génération de l'icône PNG/ICO et du manifest, sans dépendance. |
| `internal/httpapi` | Adaptateur HTTP : routes, JSON, PIN, page mobile embarquée (`page.html`). Ne dépend que de `league` et `icon`. |
| `internal/trayui` | Icône de la zone de notification (Windows uniquement, `//go:build windows`) ; une version de repli console-only compile sur les autres OS pour le développement. |
| `main.go` | Racine de composition : assemble `lcu.Client` → `league.Service` → `httpapi.Server` → `trayui`. |

### Routes du serveur

Toutes les routes `/api/*` exigent l'en-tête `X-Pin`.

| Méthode | Route | Rôle |
|---|---|---|
| GET | `/` | page mobile |
| GET | `/manifest.webmanifest`, `/icon-192.png`, `/icon-512.png` | web app |
| GET | `/icon/<id>` | icône d'un champion (proxy LCU, en cache) |
| GET | `/asset?p=/lol-game-data/assets/...` | proxy des images du client (runes, arbres) |
| GET | `/api/state` | phase, état de la sélection, action du joueur, équipes |
| GET | `/api/champions` | liste `{id, name}` |
| GET | `/api/runes` | pages, page active, arbres et runes |
| GET | `/api/runes/recommended` | runes conseillées pour le champion en cours |
| POST | `/api/select` | `{actionId, championId, lock}` : survole puis verrouille si `lock` |
| POST | `/api/accept`, `/api/decline` | ready check |
| POST | `/api/auto-accept` | `{enabled}` |
| POST | `/api/runes/activate`, `/api/runes/delete` | `{id}` |
| POST | `/api/runes/save` | `{id?, name, primaryStyleId, subStyleId, selectedPerkIds[9]}` |
| POST | `/api/runes/apply` | mêmes champs, sur la page active ou une nouvelle page |

## Compiler depuis les sources

```
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H=windowsgui" -o lol-remote.exe .
```

`-H=windowsgui` évite qu'une console s'ouvre derrière l'icône de la zone de notification. Deux dépendances (sans cgo, donc compatible cross-compilation) : [`getlantern/systray`](https://github.com/getlantern/systray) pour l'icône et [`skip2/go-qrcode`](https://github.com/skip2/go-qrcode) pour le code QR.

## Publier une release

Chaque tag `vX.Y.Z` poussé sur GitHub déclenche la CI (`.github/workflows/release.yml`), qui compile `lol-remote.exe` pour Windows et publie une Release avec le binaire joint.

```
git tag v1.0.0
git push origin v1.0.0
```

## Points d'attention

- La LCU n'est pas une API officielle : les endpoints peuvent changer après un patch.
- `champion-summary.json` contient des variantes de modes spéciaux avec le même nom et un autre ID ; on garde l'ID le plus petit par nom.
- Pendant un ban, la liste des champions autorisés est calculée côté serveur (tous sauf ceux déjà bannis ou verrouillés), car `bannable-champion-ids` n'est pas fiable.
- Sur certaines versions, la modification d'une page (`PUT`) échoue : la page est alors supprimée puis recréée.
- En `http` sur le Wi-Fi, le PIN circule en clair sur le réseau local. Ne pas exposer le port directement sur Internet.

## Historique

Ce projet a démarré comme un script Python unique (`lol_remote.py`, bibliothèque standard uniquement), pour éviter à chaque joueur d'installer Python. La réécriture en Go produit un exécutable natif à la place.
