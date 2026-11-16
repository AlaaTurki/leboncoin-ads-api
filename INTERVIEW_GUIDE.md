# Guide d'entretien — modifications du test backend

Ce document résume les changements du dépôt `leboncoin-ads-api` et fournit des éléments pour les expliquer en entretien. Il ne remplace pas la lecture du code : l’objectif est de pouvoir justifier les décisions et leurs compromis.

## Présentation courte

> J’ai corrigé la récupération d’annonce, ajouté la suppression propriétaire, enrichi la lecture avec le pseudo du vendeur, exposé le prix en unité monétaire tout en gardant un stockage entier en centimes, rendu l’import atomique et idempotent, puis appliqué le quota glissant avec une protection contre les requêtes concurrentes. Les décisions importantes sont couvertes par des tests et décrites dans l’OpenAPI et le README.

## Les six demandes, une par une

### 1. Corriger `GetAdByID`

**Problème :** le use case transmettait toujours l’identifiant `0` au DAO, quelle que soit la route demandée.

**Changement :** il transmet maintenant l’ID reçu, et les tests attendent l’ID précis au lieu d’accepter n’importe quelle valeur.

**Fichiers :** `ads/internal/cmd/ads-api/usecase/get_ad_by_id.go` et `ads/internal/cmd/ads-api/usecase/get_ad_by_id_test.go`.

**À dire en entretien :** « J’ai corrigé le défaut à la frontière du use case et renforcé le test pour qu’il échoue si le mauvais ID est transmis. »

### 2. Ajouter la suppression d’une annonce

**Changement :** ajout de `DELETE /ads/{adID}`, du use case de suppression et de la méthode DAO PostgreSQL.

La requête SQL filtre à la fois sur l’ID de l’annonce et l’ID du propriétaire. Une annonce inexistante ou appartenant à quelqu’un d’autre produit `404`, afin de ne pas révéler son existence. La suppression réussie retourne `204`.

**Fichiers :** `http/delete_ad.go`, `usecase/delete_ad.go`, `dao/postgres/postgres.go`, la route dans `http/router.go` et leurs tests.

**À dire en entretien :** « L’autorisation est appliquée dans la requête de suppression elle-même, ce qui évite une vérification séparée sujette à une course entre lecture et suppression. »

### 3. Ajouter le pseudo du propriétaire à la lecture

**Changement :** `GET /ads/{adID}` appelle le client utilisateur fourni par le dépôt et ajoute `username` à sa réponse. Un cache mémoire d’une minute limite les appels répétés au service d’identité ; les appels concurrents pour un même utilisateur partagent aussi la recherche en cours.

Si l’utilisateur n’existe plus, l’annonce reste lisible avec un pseudo vide. Si le service d’identité échoue, l’API répond `502`.

**Fichiers :** `http/get_ad.go`, `http/user_cache.go`, `main.go`, `config.go` déjà présent, `docker-compose.yml` et `specs/openapi.yaml`. Le service `/users` lui-même n’a pas été modifié.

**À dire en entretien :** « Le cache est local au processus et volontairement court. Pour plusieurs instances, je pourrais le remplacer par un cache partagé ou accepter quelques appels supplémentaires, selon les exigences de fraîcheur des pseudos. »

### 4. Remplacer `price_cents` par `price`

**Changement :** l’API reçoit et renvoie `price` en unité monétaire sous forme de nombre décimal. La base conserve `price_cents` comme entier `BIGINT` ; les entrées négatives, non finies ou ayant plus de deux décimales sont rejetées.

**Pourquoi garder les centimes en base :** les nombres flottants ne représentent pas exactement toutes les fractions décimales. L’entier en centimes reste la représentation canonique et évite une migration des données.

**Compromis à expliquer :** le renommage du champ est une rupture de contrat pour les clients existants. En production, je planifierais une version d’API ou une période pendant laquelle l’ancien champ est déprécié. Le test applique directement le nouveau contrat.

**Fichiers :** `http/create_ad.go`, `http/get_ad.go`, `http/list_ads.go`, `specs/openapi.yaml` et `README.md`.

### 5. Éviter les doublons quand un import est rejoué

**Changement :** `/ads/import` exige un `Idempotency-Key`. Le lot et ses IDs sont enregistrés dans une même transaction avec une empreinte du contenu. Le propriétaire et la clé identifient une requête : un rejeu identique retourne les mêmes IDs ; la même clé avec un contenu différent retourne `409`.

Un verrou transactionnel PostgreSQL sérialise les tentatives simultanées portant la même clé. En cas d’échec pendant l’insertion, le lot et sa trace d’idempotence sont annulés ensemble.

**Fichiers :** `dao/postgres/postgres.go`, `usecase/create_ad.go`, `http/import_ads.go` et la migration `003_create_ad_import_requests.sql`.

**À dire en entretien :** « Une transaction seule empêche les imports partiels, mais ne couvre pas le cas où la transaction a réussi et où la réponse s’est perdue. La clé d’idempotence couvre aussi ce rejeu après succès. »

**Compromis à discuter :** les clés sont conservées en base. À plus grande échelle, il faudrait définir une durée de rétention et purger les anciennes clés selon le contrat d’idempotence.

### 6. Limiter les créations à cinq sur 24 heures glissantes

**Changement :** le use case ouvre une transaction, prend un verrou consultatif PostgreSQL par propriétaire, compte les événements des dernières 24 heures, puis crée l’annonce et enregistre l’événement avant le commit. Si le quota est atteint, l’API répond `429`.

Les événements sont dans `ad_creation_events`, séparés des annonces : supprimer une annonce ne libère donc pas une place. L’import en lot passe par un chemin distinct et ne consomme pas ce quota.

**Fichiers :** `usecase/create_ad.go`, `dao/dao.go`, `dao/postgres/postgres.go`, la migration `004_create_ad_creation_events.sql` et les tests associés.

**À dire en entretien :** « Le verrou est pris avant le comptage et gardé jusqu’au commit. Deux requêtes simultanées d’un même utilisateur ne peuvent donc pas toutes les deux voir quatre événements puis insérer une cinquième annonce. »

## Autres détails à connaître

- Le mock DAO contenait un test contradictoire : il attendait une erreur alors que le mock était configuré sans erreur. L’assertion a été corrigée.
- `docker-compose.yml` démarre aussi le service d’identité, nécessaire pour remplir `username`.
- Les deux nouvelles migrations sont exécutées automatiquement lors de la première initialisation d’un volume PostgreSQL. Pour un volume existant, le README donne les commandes pour les appliquer une fois.
- Le mécanisme d’authentification fourni par l’exercice accepte un ID utilisateur brut dans `Authorization` ; il ne valide pas de jeton. La suppression vérifie bien le propriétaire associé à cet ID, mais une application réelle doit remplacer ce mécanisme par une identité authentifiée et vérifiée.

## Vérifications effectuées

- `go test ./...` : réussi.
- `docker compose config -q` : réussi.
- Le YAML OpenAPI est analysable.
- Le démarrage réel des conteneurs et les migrations contre un serveur PostgreSQL n’ont pas été exécutés ici. Je les lancerais avant déploiement pour compléter les tests avec la base réelle.

## Questions probables

**Pourquoi ne pas stocker le prix en `float` ?**  
Parce qu’un flottant binaire peut introduire des écarts d’arrondi. L’API expose la forme demandée, mais les centimes entiers restent la source exacte en base.

**Pourquoi `404` quand l’annonce appartient à un autre utilisateur ?**  
Pour éviter de révéler qu’une annonce existe. La suppression ne touche que les lignes dont l’ID et le propriétaire correspondent.

**Une transaction ne suffisait-elle pas pour l’import ?**  
Elle évite les lots partiels, mais pas les doublons si le client rejoue un lot qui a déjà été validé. La clé d’idempotence rend le résultat rejouable.

**Comment le quota tient-il en cas de concurrence ?**  
Le verrou PostgreSQL par utilisateur sérialise les vérifications. Le comptage et l’écriture de l’annonce et de l’événement sont dans la même transaction.

**Quelles améliorations envisager ensuite ?**  
Tester les migrations et les requêtes contre PostgreSQL réelle, définir la rétention des clés d’idempotence et remplacer l’authentification simplifiée de l’exercice par une identité vérifiée.
