# carnet

Générateur statique de carnets de route : transforme un dossier de voyage
(trace GPX + photos) en contenu prêt à publier sur un site statique,
sans JavaScript, sans fuite de localisation sensible.

> Squelette de projet — développement en cours, rien n'est encore implémenté.

## Installation

_À rédiger (S6 : binaires linux/amd64, linux/arm64, darwin/arm64 publiés en release)._

## Utilisation

```
carnet build <trip-dir> [options]
carnet check <trip-dir> [options]
carnet version
```

_À rédiger._

## Format d'entrée

_À rédiger : structure du dossier de voyage, `carnet.yaml`, `track.gpx`, `photos/`._

## Référence de configuration

_À rédiger : toutes les clés de `carnet.yaml` et leurs valeurs par défaut._

## Modèle de confidentialité et ses limites

_À rédiger : zones d'exclusion, troncature, arrondi, purge EXIF, garde-fou final —
et les limites du dispositif (recoupement entre carnets possible)._

## Dénivelé : méthode de calcul

_À rédiger : lissage par fenêtre glissante et seuil de bruit (aucune méthode n'est canonique)._

## Intégration Astro

_À rédiger : schéma de content collection (Zod) correspondant exactement au frontmatter produit._

## Note de profilage

_À rédiger : ce que `pprof` a révélé sur le pipeline photos/rendu._

## Licence

MIT — voir [LICENSE](LICENSE).
