# 🗳️ Unis pour Lanvaudan - Site de Campagne 2026

Bienvenue sur le dépôt du site officiel de la campagne "Unis pour Lanvaudan" pour les élections municipales de 2026. 

Ce projet a été conçu selon une approche "Minimalist & Vanilla" : un backend léger et robuste en **Go (Gin)** couplé à un frontend **HTML/CSS/JS** sans framework lourd, mis en valeur par des animations fluides via **GSAP**.

L'objectif est d'offrir une plateforme performante, éco-conçue et facile à déployer, permettant de présenter les engagements de la liste et de récolter les adresses e-mail des sympathisants (dans le strict respect du RGPD).

---

## 🏗️ Architecture du Projet

Le projet est divisé en deux couches simples :

### 1. Backend (Go + Gin)
- **Point d'entrée :** `src/main.go`.
- **Serveur HTTP :** Construit avec le framework web [Gin](https://gin-gonic.com/) pour des performances optimales et un routage ultra-rapide.
- **Base de données :** SQLite locale stockée dans `data/subscribers.db`. Elle est automatiquement provisionnée au premier lancement de l'application (via GORM/SQL natif avec le package `go-sqlite3`).
- **API :** Un endpoint `POST /subscribe` permet de récupérer les adresses e-mail et de gérer les erreurs (doublons, format invalide) en renvoyant du JSON.

### 2. Frontend (Vanilla + GSAP)
- **Templates :** Les pages (notamment `index.html`) sont servies par le moteur de templates HTML intégré à Gin, depuis le dossier `templates/`.
- **Ressources statiques :** Les images, logos et potentiels fichiers tiers (non-CDN) sont placés dans l'arborescence `assets/` (servi publiquement sur la route `/assets`).
- **Animations :** [GreenSock (GSAP)](https://gsap.com/) et le plugin ScrollTrigger sont utilisés pour animer de manière performante l'apparition des éléments lors du défilement.
- **AJAX :** Le formulaire de souscription utilise l'API native JavaScript `fetch()` pour envoyer les données au serveur de manière asynchrone, offrant une expérience fluide avec un système de notifications (Toasts) fait-maison.

---

## 📂 Structure du Répertoire

```text
/
├── assets/             # Fichiers statiques et PWA (logo, manifest, sw.js)
├── bin/                # Scripts de build et exécutables (ex: build.sh)
├── data/               # Données locales (subscribers.db)
├── src/                # Code source du backend Go (main.go)
├── templates/          # Vues HTML/Go templates (ex: index.html)
├── go.mod / go.sum     # Gestion centralisée des dépendances Go
└── README.md           # Ce fichier
```

---

## 🚀 Installation & Lancement (Environnement de Développement)

### Prérequis
- [Go](https://go.dev/dl/) en version 1.20 ou supérieure.
- Un compilateur C (CGO requis pour compiler le driver `go-sqlite3`).
  - *MacOS* : `xcode-select --install`
  - *Linux* : `sudo apt install gcc` (ou équivalent selon la distribution)
  - *Windows* : TDM-GCC ou MinGW.

### Étapes de déploiement local

1. **Se rendre à la racine du projet :**
   ```bash
   cd /Volumes/CLHD/LABS/lanvaudan
   ```

2. **Télécharger et vérifier les dépendances Go :**
   ```bash
   go mod tidy
   ```

3. **Démarrer le serveur HTTP de développement :**
   ```bash
   go run src/main.go
   ```
   > Par défaut, le serveur écoutera sur le port `8080`. L'interface de la landing page est alors accessible sur [http://localhost:8080](http://localhost:8080).

### Réglages (Variables d'environnement)
- `PORT` : Pour forcer le port d'écoute (ex: `PORT=3000 go run src/main.go`).
- `GIN_MODE` : Le code source bascule implicitement Gin en statut "release". En cas de besoin de debug approfondi, passez `GIN_MODE=debug` en variable d'environnement pour réactiver le logger de requêtes complet de Gin.

---

## 🗄️ Modèle de Données (Base de données SQLite)

Le fichier SQLite `subscribers.db` intègre une table unique : `subscribers`.

| Colonne      | Type     | Description                                      |
|--------------|----------|--------------------------------------------------|
| `id`         | INTEGER  | PK (Clé primaire auto-incrémentée)               |
| `email`      | TEXT     | Adresse email de l'abonné (Contrainte: `UNIQUE`) |
| `created_at` | DATETIME | Horodatage d'inscription (CURRENT_TIMESTAMP)     |

*Notes pratiques pour l'équipe technique :* 
- La contrainte `UNIQUE` sur le champ `email` nous protège des inscriptions multiples accidentelles. Lorsque cela se produit, `main.go` intercepte l'erreur du gestionnaire SQLite et retourne une erreur `HTTP 409 Conflict`.
- Il n'existe pas d'interface d'administration `admin/` active dans cette version MVP. Pour exporter la liste de courriels (pour publipostage via Brevo / Mailchimp), ouvrez un terminal et tapez :
  ```bash
  sqlite3 data/subscribers.db "SELECT email FROM subscribers;" > extraction_emails.csv
  ```

---

## 🎨 Lignes Directrices (Contribution Front-End)

Si vous intervenez sur `templates/index.html`, tâchez de respecter ces quelques conventions pour un projet long terme pérenne :

1. **CSS Vanilla Scoped :** Les styles sont intentionnellement encapsulés dans la balise `<style>` du document pour limiter les requêtes initiales. Utilisez logiquement les variables (`--bg-color`, `--accent-color`, `--text-color`) localisées à `:root` pour préserver le contrat design/couleur de la charte.
2. **Animations & Défilement :** Pour qu'un nouveau bloc textuel profite du fondu d'apparition vertical, ajoutez simplement la classe `.reveal` au tag HTML de bloc ciblé. ScrollTrigger gère le reste.
3. **Le Trombinoscope Collaboratif (`#teamBody`) :** La liste des candidats est générée de façon réactive côté client. Si l'équipe évolue, inutile de duper le HTML en brut. Trouvez la constante array `candidates` dans la balise `<script>`, et mettez à jour la configuration JSON.
4. **Conformité RGPD :** Tout changement esthétique du bloc de souscription e-mail ne doit en aucun cas supplanter ou masquer la clause explicative RGPD (`<p class="rgpd">`).

---

## 🗓️ Roadmap & TODO List (Post MVP)

- [ ] Implémenter une forme de *Rate-Limiting* sur la route POST `/subscribe` (protection contre le flood par un robot).
- [ ] Centraliser le bloc `<style>` dans `/assets/style.css` lorsque le fichier `index.html` deviendra structurellement trop massif.
- [ ] Implémenter une commande CLI interne pour purger facilement les données de la table avant lancement final.
- [ ] Compresser toutes les bannières futures importées dans `/assets` au format `.webp`.

---

*Ce projet est construit sur des bases robustes, saines, et autonomes. Faisons notre part pour Lanvaudan. Bon code à toutes et à tous ! 🤝*
