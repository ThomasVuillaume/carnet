# carnet

Générateur statique de carnets de route : transforme un dossier de voyage
(trace GPX + photos) en contenu prêt à publier sur un site statique,
sans JavaScript, sans fuite de localisation sensible.

> Squelette de projet — développement en cours, rien n'est encore implémenté.

## Installation

_À rédiger (S6 : binaires linux/amd64, linux/arm64, darwin/arm64 publiés en release)._

## Utilisation

```text
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

## Intégration Astro

_À rédiger : schéma de content collection (Zod) correspondant exactement au frontmatter produit._

## Note de profilage

_À rédiger : ce que `pprof` a révélé sur le pipeline photos/rendu._

## Développement

### Prérequis

- **Go 1.26 ou supérieur** (`go version`).
- **golangci-lint v2**, compilé avec la même version majeure de Go que celle
  ciblée par `go.mod`. Un binaire construit avec une toolchain plus ancienne
  refuse de démarrer :

  ```sh
  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
  golangci-lint --version   # doit indiquer « built with go1.26 » ou plus récent
  ```

  Vérifiez que `$(go env GOPATH)/bin` figure bien dans votre `PATH`.

### Commandes

```sh
go build ./...          # compilation de tous les paquets
go test -race ./...     # suite complète, détecteur de data races actif
go vet ./...            # analyses statiques de la toolchain
golangci-lint run       # jeu d'analyseurs du projet (doit rester à zéro avertissement)
gofmt -l .              # liste les fichiers mal formatés (silence = tout va bien)
```

La suite de tests doit s'exécuter **entièrement hors ligne** et en moins de
60 secondes. Aucun test n'accède au réseau : les tuiles de test proviennent
d'un `embed.FS`.

### Stratégie de test : tables de cas

Les tests unitaires du projet sont écrits en **table-driven**, la convention
dominante en Go. Le principe : les cas de test sont des **données**, et la
logique de vérification n'est écrite **qu'une seule fois**.

Concrètement, une tranche de structures anonymes décrit les cas, et une boucle
les exécute :

```go
tests := []struct {
    name     string      // identifie le cas dans la sortie de test
    raw      xmlTrkpt    // entrée
    wantSkip bool        // sortie attendue
    wantTime time.Time
}{
    {
        name:     "NaN latitude",
        raw:      xmlTrkpt{Lat: math.NaN(), Lon: 3.858475, Time: stamp},
        wantSkip: true,
    },
    // ... un cas par ligne
}

for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        t.Parallel()
        // vérification écrite une seule fois, valable pour tous les cas
    })
}
```

Trois règles à respecter en ajoutant des tests :

1. **Le premier champ est `name`.** C'est lui qui apparaît dans la sortie en
   cas d'échec (`--- FAIL: TestPointFrom/NaN_latitude`). Sans lui, il faudrait
   compter les itérations pour identifier le cas fautif.
2. **`t.Run` pour chaque cas.** Il crée un sous-test qui réussit ou échoue
   indépendamment des autres, et devient adressable par `-run`.
3. **`t.Parallel()` sur le test parent et sur chaque sous-test.** Les fonctions
   testées ici sont pures ; l'exécution concurrente est donc gratuite et donne
   au détecteur de races de `-race` quelque chose à observer.

Un cas supplémentaire coûte alors une ligne, pas une fonction. C'est l'intérêt
principal du style : le coût marginal d'un cas limite devient assez faible pour
qu'on le couvre réellement. En retour, la table se lit comme la spécification
de la fonction testée.

Les sous-tests sont adressables individuellement, ce qui rend le débogage
ciblé confortable :

```sh
go test ./internal/gpx/ -run 'TestPointFrom' -v          # tous les cas, nommés
go test ./internal/gpx/ -run 'TestPointFrom/NaN' -v      # les cas NaN seulement
```

Le motif passé à `-run` est découpé sur les `/`, une expression régulière par
niveau d'imbrication.

Ce style est **exigé** sur `internal/geo`, `internal/gpx` et `internal/privacy`,
dont les fonctions sont des transformations pures entrée → sortie. Les cas
limites à couvrir systématiquement : trace vide, trace à point unique,
franchissement de l'antiméridien, pôles, et valeurs flottantes non finies
(`NaN` traverse silencieusement toute validation écrite sous forme de bornes,
puisque toute comparaison l'impliquant est fausse).

### Couverture

La couverture doit rester supérieure à 85 % sur `internal/geo`, `internal/gpx`
et `internal/privacy` :

```sh
go test -cover ./internal/geo/ ./internal/gpx/ ./internal/privacy/
```

Pour inspecter les lignes non couvertes d'un paquet :

```sh
go test -coverprofile=cover.out ./internal/gpx/
go tool cover -html=cover.out
```

### Golden files

Les rendus cartographiques sont comparés à des images de référence stockées
dans `testdata/golden/`. Après un changement volontaire du rendu, régénérez-les
puis **relisez le diff visuellement** avant de le valider :

```sh
go test ./internal/render/ -update
```

### À propos du linter

La configuration `.golangci.yml` ne se contente pas des analyseurs par défaut.
Deux ajouts portent des exigences du projet plutôt que des préférences de style :

- **`depguard`** interdit à la compilation les bibliothèques que le projet
  s'est engagé à ne pas utiliser (`go-staticmaps`, `gg`, `gpxgo`). Écrire
  soi-même l'assemblage de tuiles, la rastérisation et le parsing GPX est un
  objectif du projet, pas un accident.
- **`errorlint`** signale un `%v` là où un `%w` est attendu, ainsi que les
  comparaisons `err == cible` qui devraient passer par `errors.Is`. La chaîne
  d'erreurs reste ainsi traversable de bout en bout.

## Licence

MIT — voir [LICENSE](LICENSE).
