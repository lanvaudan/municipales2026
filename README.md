# Unis pour Lanvaudan — Site de Campagne 2026

Dépôt du site officiel de la campagne "Unis pour Lanvaudan" pour les élections municipales de 2026.

Approche minimaliste : backend léger en **Go (Gin)**, frontend **HTML/CSS/JS** sans framework, animations GSAP. Un seul binaire, zéro dépendance externe au runtime.

---

## Architecture

### Backend (`src/main.go`)

Un seul fichier. Deux routeurs Gin indépendants dispatché par **sous-domaine** :

| Sous-domaine | Routeur | Rôle |
|---|---|---|
| `municipales2026.*` (ou tout autre) | `buildPublicRouter` | Site public |
| `bureau.*` | `buildBureauRouter` | Interface d'administration |

Le dispatcher lit le header `Host` de chaque requête HTTP — il suffit que le reverse proxy fasse pointer les deux sous-domaines vers le même port.

**Routes publiques** (`municipales2026.unispourlanvaudan.fr`)
- `GET /` — page de campagne (Go template depuis `content.json`)
- `POST /subscribe` — inscription email RGPD ; HTTP 409 si doublon
- `GET /assets/*` — fichiers statiques

**Routes bureau** (`bureau.unispourlanvaudan.fr`)
- `GET /` — redirige vers `/login` ou `/edit`
- `GET /login`, `POST /login` — authentification par mot de passe
- `GET /edit` — éditeur de contenu (Quill.js, protégé)
- `POST /save` — sauvegarde dans `data/content.json` (protégé)
- `GET /logout` — destruction de session

**Sessions :** cookie HTTP-only `bureau_session` (token 32 octets base64, stocké en mémoire via `sync.Map`).

### Frontend (`templates/`)

- `index.html` — page publique. CSS inline intentionnel. Contenu 100 % rendu côté serveur depuis `content.json` (Go template).
- `bureau_login.html` — formulaire de connexion admin.
- `bureau_editor.html` — éditeur admin (Quill.js 1.3.7 CDN, 3 onglets : textes / engagements / équipe).

### Données (`data/`)

| Fichier | Rôle | Versionné |
|---|---|---|
| `subscribers.db` | SQLite — table `subscribers(id, email UNIQUE, created_at)` | Non |
| `content.json` | Contenu live du site (titres, textes, engagements, candidats) | Non |
| `.env` | `BUREAU_PASSWORD=…` | Non |

---

## Structure du dépôt

```text
/
├── assets/         # Fichiers statiques (logo, manifest.json, sw.js)
├── bin/            # Scripts (build.sh, service.sh, verifDB.sh) + binaire compilé
├── data/           # Données runtime (gitignorées)
├── src/main.go     # Code source backend (unique fichier Go)
├── templates/      # Vues HTML
├── go.mod / go.sum
└── README.md
```

---

## Installation & développement

### Prérequis

- Go 1.20+
- CGO activé (requis par `go-sqlite3`) → un compilateur C doit être disponible
  - Linux : `sudo apt install gcc`
  - macOS : `xcode-select --install`

### Lancer en développement

```bash
# Depuis la racine du projet
go run src/main.go
# → http://localhost:8024
```

### Variables d'environnement

| Variable | Défaut | Rôle |
|---|---|---|
| `PORT` | `8024` | Port d'écoute |
| `GIN_MODE` | `release` | Passer `debug` pour les logs Gin complets |
| `BUREAU_PASSWORD` | `changeme` | Mot de passe de l'interface admin (via `data/.env`) |

### Build production

```bash
bash bin/build.sh
# → bin/lanvaudan
```

### Commandes utiles

```bash
# Vérifier le contenu de la base abonnés
bash bin/verifDB.sh

# Exporter les emails pour publipostage
sqlite3 data/subscribers.db "SELECT email FROM subscribers;" > extraction_emails.csv

# Gérer le service systemd
bash bin/service.sh
```

---

## Conventions front-end

- **Couleurs :** utiliser les variables `:root` — `--bg-color`, `--accent-color`, `--text-color`.
- **Animations au scroll :** ajouter la classe `.reveal` à tout élément bloc pour un fade-in GSAP automatique.
- **Candidats :** la liste est rendue côté serveur depuis le champ `candidates` de `content.json`. Éditer via l'interface bureau ou directement dans le JSON — ne pas dupliquer de HTML manuellement.
- **RGPD :** la balise `<p class="rgpd">` dans le formulaire d'inscription ne doit jamais être masquée ou supprimée.

---

## Roadmap

- [ ] Rate-limiting sur `POST /subscribe`
- [ ] Extraire `<style>` vers `/assets/style.css` si `index.html` devient trop volumineux
- [ ] Commande CLI pour purger la table abonnés
- [ ] Convertir les futurs assets image en `.webp`
