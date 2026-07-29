# Plan d'Implémentation : Améliorations Docker

Ce plan décrit les étapes pour implémenter les nouvelles fonctionnalités Docker demandées, notamment la modification transparente des conteneurs existants et la gestion des réseaux et volumes Docker.

## User Review Required

> [!IMPORTANT]
> **Validation requise** : Veuillez lire ce plan et valider l'approche avant que je ne commence l'implémentation. Les modifications de conteneurs impliquent la destruction et la recréation du conteneur, ce qui entraînera une brève interruption de service pour le conteneur concerné.

## Open Questions

> [!WARNING]
> 1. Pour la mise à jour d'un conteneur (`PUT /v2/docker/containers/{id}/update`), souhaitez-vous qu'on puisse modifier *toutes* les configurations (image, ports, variables d'environnement, limites de RAM/CPU), ou seulement certaines propriétés spécifiques (ex: uniquement les variables d'environnement et la RAM) ? Par défaut, l'implémentation prévue prendra la configuration existante, appliquera les modifications demandées, supprimera le conteneur, et le recréera.
> 2. Souhaitez-vous des permissions spécifiques pour la création de réseaux et de volumes (ex: `docker.networks.write`, `docker.volumes.write`), ou doit-on réutiliser `docker.write` ?

## Proposed Changes

---

### Endpoints de Gestion des Conteneurs

#### [MODIFY] [internal/modules/v2/docker/routes.go](file:///c:/Users/iswea/Desktop/ServeoAPI%20V2/internal/modules/v2/docker/routes.go)
- Ajout de la route `PUT /v2/docker/containers/{id}/update` pointant vers `h.UpdateContainer`.

#### [MODIFY] [internal/modules/v2/docker/handler_containers.go](file:///c:/Users/iswea/Desktop/ServeoAPI%20V2/internal/modules/v2/docker/handler_containers.go)
- Ajout du DTO `UpdateContainerRequest` pour spécifier les champs modifiables (RAM, variables d'environnement, ports, etc.).
- Ajout de la méthode `UpdateContainer`.
- Implémentation du flux de mise à jour transparente dans le `Service` :
  1. Inspecter le conteneur existant.
  2. Extraire sa configuration actuelle (Image, Env, Ports, HostConfig).
  3. Appliquer les modifications demandées.
  4. Arrêter et supprimer le conteneur existant (sans supprimer les volumes montés !).
  5. Recréer et démarrer le conteneur avec la nouvelle configuration.

#### [MODIFY] [internal/modules/v2/docker/service.go](file:///c:/Users/iswea/Desktop/ServeoAPI%20V2/internal/modules/v2/docker/service.go)
- Ajout de la méthode `UpdateContainer(ctx, id, req UpdateContainerRequest)` qui encapsulera la logique de recréation.

---

### Endpoints de Réseaux Docker (Networks)

#### [MODIFY] [internal/modules/v2/docker/routes.go](file:///c:/Users/iswea/Desktop/ServeoAPI%20V2/internal/modules/v2/docker/routes.go)
- Ajout des routes pour gérer les réseaux :
  - `GET /v2/docker/networks/`
  - `POST /v2/docker/networks/`
  - `DELETE /v2/docker/networks/{id}`

#### [MODIFY] [internal/modules/v2/docker/handler_networks.go](file:///c:/Users/iswea/Desktop/ServeoAPI%20V2/internal/modules/v2/docker/handler_networks.go)
- Implémentation des handlers `ListNetworks`, `CreateNetwork`, et `DeleteNetwork`.
- Ajout des DTOs correspondants (`CreateNetworkRequest`).

#### [MODIFY] [internal/modules/v2/docker/service.go](file:///c:/Users/iswea/Desktop/ServeoAPI%20V2/internal/modules/v2/docker/service.go)
- Ajout des méthodes `ListNetworks`, `CreateNetwork`, et `DeleteNetwork`.

---

### Endpoints de Volumes Docker (Volumes)

#### [MODIFY] [internal/modules/v2/docker/routes.go](file:///c:/Users/iswea/Desktop/ServeoAPI%20V2/internal/modules/v2/docker/routes.go)
- Ajout des routes pour gérer les volumes :
  - `GET /v2/docker/volumes/`
  - `POST /v2/docker/volumes/`
  - `DELETE /v2/docker/volumes/{name}`

#### [MODIFY] [internal/modules/v2/docker/handler_volumes.go](file:///c:/Users/iswea/Desktop/ServeoAPI%20V2/internal/modules/v2/docker/handler_volumes.go)
- Implémentation des handlers `ListVolumes`, `CreateVolume`, et `DeleteVolume`.
- Ajout des DTOs correspondants (`CreateVolumeRequest`).

#### [MODIFY] [internal/modules/v2/docker/service.go](file:///c:/Users/iswea/Desktop/ServeoAPI%20V2/internal/modules/v2/docker/service.go)
- Ajout des méthodes `ListVolumes`, `CreateVolume`, et `DeleteVolume`.

## Verification Plan

### Automated Tests
- Lancer `go build ./...` et `go test ./...` après les modifications.

### Manual Verification
1. Appeler `PUT /v2/docker/containers/{id}/update` sur un conteneur de test pour modifier sa variable d'environnement, puis inspecter le conteneur pour vérifier que la modification a été prise en compte et que le volume persiste.
2. Lister, créer et supprimer des réseaux et volumes via les nouveaux endpoints et vérifier dans le client Docker (ex: `docker network ls`) que les changements sont répercutés.
