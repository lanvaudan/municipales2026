#!/bin/bash
# Script de compilation pour le backend Go

# S'assurer qu'on se place à la racine du projet (le parent du dossier bin)
cd "$(dirname "$0")/.."

# Nom du binaire compilé
APP_NAME="lanvaudan"

echo "⚙️ Compilation de l'application Go..."

# Compilation (Le code source est dans le dossier src/)
go build -o "bin/$APP_NAME" src/main.go

if [ $? -eq 0 ]; then
    echo "✅ Succès ! L'exécutable a été généré dans le dossier bin/$APP_NAME"
    echo "🚀 Pour le lancer en environnement de prod : cd .. && ./bin/$APP_NAME"
else
    echo "❌ Erreur lors de la compilation."
    exit 1
fi
