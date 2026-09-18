# SimpleConfig Operator — Très simple exemple (Go)

Objectif
--------
Ce projet illustre un opérateur Kubernetes minimal et pédagogique. Il montre comment définir un CRD "SimpleConfig" (group: config.example.com) et un controller très simple qui transforme la Spec du CR en un ConfigMap Kubernetes.

Cas réel et utile :
- Permet à une équipe de stocker des configurations applicatives dans des Custom Resources (ex: config déclarative dans Git) et laisser l'opérateur créer/mettre à jour un ConfigMap que les pods consomment.

Fichiers importants
-------------------
- go.mod — module et dépendances
- main.go — démarre le manager controller-runtime
- api/v1alpha1/groupversion_info.go — enregistre le GroupVersion
- api/v1alpha1/webapp_types.go — définit SimpleConfig (Spec: data map, Status: configMapName + synced)
- controllers/webapp_controller.go — logique minimale : crée/maintient un ConfigMap avec les données de spec
- config/crd/bases/apps.mycompany.com_webapps.yaml — CRD à appliquer (fichier remplacé par SimpleConfig CRD)
- config/samples/apps_v1alpha1_webapp.yaml — exemple de CR

Comment utiliser
----------------
Prérequis : Go, kubectl et un cluster accessible (minikube/kind).

1) Installer les dépendances :
   cd <repo>/crd
   go mod tidy

2) Appliquer la CRD :
   kubectl apply -f config/crd/bases/apps.mycompany.com_webapps.yaml

3) Lancer l'opérateur localement :
   go run main.go

4) Créer un SimpleConfig :
   kubectl apply -f config/samples/apps_v1alpha1_webapp.yaml

5) Vérifier :
   kubectl get simpleconfig -A
   kubectl get configmap example-config-cm -n default -o yaml

Explication simple du controller
--------------------------------
- Lorsqu'un SimpleConfig est créé, le controller crée un ConfigMap nommé <simpleconfig-name>-cm contenant les paires key/value spécifiées dans spec.data.
- Si le SimpleConfig est mis à jour, l'opérateur met à jour le ConfigMap pour refléter spec.data.
- Le controller met à jour status.configMapName et status.synced pour indiquer que l'objet est appliqué.

Pourquoi c'est simple et pédagogique
----------------------------------
- Le controller ne gère qu'une seule ressource Kubernetes (ConfigMap) et transforme directement spec -> ConfigMap.
- Pas de Deployment/Service, pas de logique complexe : idéal pour comprendre le pattern reconcile.

Prochaines étapes possibles
--------------------------
- Ajouter tests unitaires avec fake client.
- Ajouter RBAC manifests et déployer l'opérateur en cluster.
- Étendre pour générer Secrets (chiffrés) ou gérer versioning.



Objectif
--------
Ce dépôt contient un exemple minimal et pédagogique d'un opérateur Kubernetes écrit en Go. L'objectif est d'illustrer les concepts essentiels :
- Définir un CustomResourceDefinition (CRD) pour une ressource WebApp.
- Implémenter les types Go (Spec/Status) pour cette ressource.
- Écrire un controller (reconciler) qui synchronise l'état désiré (Spec) avec l'état observé (objects Kubernetes : Deployment et Service).
- Montrer l'usage de ownerReferences, status subresource, et des bonnes pratiques d'idempotence.

Structure du projet
-------------------
Les fichiers créés dans /Users/ahmedhachicha/crd :

- [go.mod](/Users/ahmedhachicha/crd/go.mod) — module et dépendances Go.
- [main.go](/Users/ahmedhachicha/crd/main.go) — création du manager controller-runtime et enregistrement du controller.
- [api/v1alpha1/groupversion_info.go](/Users/ahmedhachicha/crd/api/v1alpha1/groupversion_info.go) — enregistrement du GroupVersion et des types dans le scheme.
- [api/v1alpha1/webapp_types.go](/Users/ahmedhachicha/crd/api/v1alpha1/webapp_types.go) — définition du CustomResource WebApp (Spec & Status) avec marqueurs kubebuilder.
- [controllers/webapp_controller.go](/Users/ahmedhachicha/crd/controllers/webapp_controller.go) — la logique du Reconciler (création/mise à jour du Deployment et Service, mise à jour du status).
- [config/crd/bases/apps.mycompany.com_webapps.yaml](/Users/ahmedhachicha/crd/config/crd/bases/apps.mycompany.com_webapps.yaml) — CRD (apiextensions.k8s.io/v1) à appliquer sur le cluster.
- [config/samples/apps_v1alpha1_webapp.yaml](/Users/ahmedhachicha/crd/config/samples/apps_v1alpha1_webapp.yaml) — exemple de CustomResource pour tester.

Comment lancer et tester
------------------------
Prérequis : un cluster Kubernetes accessible via kubeconfig (minikube, kind, ou cluster réel) et Go installé.

1. Se placer dans le projet :
   cd /Users/ahmedhachicha/crd

2. Installer les dépendances :
   go mod tidy

3. Installer la CRD sur le cluster :
   kubectl apply -f config/crd/bases/apps.mycompany.com_webapps.yaml

4. Lancer l'opérateur localement (utilise kubeconfig courant) :
   go run main.go
   (ou : go build -o webapp-operator && ./webapp-operator)

5. Appliquer l'exemple de CustomResource :
   kubectl apply -f config/samples/apps_v1alpha1_webapp.yaml

6. Vérifier les ressources :
   kubectl get webapps -A
   kubectl get webapp example-webapp -n default -o yaml
   kubectl get deploy,svc -n default

7. Nettoyage :
   kubectl delete -f config/samples/apps_v1alpha1_webapp.yaml

Explication du code et des concepts clés
---------------------------------------
1) Types API (api/v1alpha1/webapp_types.go)
- Spec : définit ce que l'utilisateur souhaite (replicas, image, port).
  - Replicas est un *pointer* (*int32) pour pouvoir détecter l'absence de valeur et fournir une valeur par défaut.
- Status : champs mis à jour par l'opérateur pour refléter l'état observé (availableReplicas).
- Marqueurs kubebuilder (+kubebuilder:...) aident à générer la CRD et ajouter des colonnes imprimables.

2) CRD (config/crd/...) :
- Fichier apiextensions.k8s.io/v1 qui déclare le schema (openAPIV3Schema) pour la validation côté API server.
- Subresource status activé pour permettre la mise à jour du champ status séparément.

3) main.go : manager et scheme
- Le scheme enregistre les types core (Deployment, Service) et le type WebApp afin que le manager et le client sachent sérialiser/désérialiser ces objets.
- Le manager fournit le client et l'environnement d'exécution pour les controllers.

4) Controller / Reconciler (controllers/webapp_controller.go)

Flux principal du Reconcile :
- Récupère l'objet WebApp demandé (r.Get).
- Définit des valeurs par défaut (ex : replicas = 1 si nil).
- Construit l'objet désiré (desiredDeployment, desiredService) à partir de la Spec du CR.
- Définit l'owner reference (SetControllerReference) pour que le Deployment et le Service soient liés au WebApp : cela permet au garbage collector Kubernetes de supprimer automatiquement les enfants lors de la suppression du CR.
- Lit s'il existe déjà un Deployment/Service avec le nom attendu :
  - S'il n'existe pas, le crée.
  - S'il existe, compare la Spec et met à jour si nécessaire (pattern idempotent).
  - Pour le Service, préserve ClusterIP existant (champ immuable) lors des mises à jour.
- Récupère le Deployment mis à jour et lit availableReplicas. Si différente du status enregistré, met à jour webapp.Status via r.Status().Update.

Points importants et bonnes pratiques illustrées :
- Idempotence : le reconcile peut être appelé de façon répétée ; les opérations doivent être sûres et aboutir au même état désiré sans effets de bord.
- Separation spec/status : spec = souhait, status = observé. Le controller écrit le status via l'API status subresource pour refléter l'état réel.
- OwnerReferences : essentiels pour la gestion du cycle de vie des objets enfants.
- RBAC : les commentaires +kubebuilder:rbac servent à générer les règles nécessaires. Si on déploie manuellement, il faudra fournir un Role/ClusterRole avec les droits listés.
- Préserver ClusterIP : ClusterIP est immuable — lors d'un update on doit le préserver depuis l'objet existant.

Suggestions pour aller plus loin
------------------------------
- Ajouter un finalizer pour gérer un cleanup ordonné lors de la suppression du CR (utile si l'opérateur gère des ressources externes).
- Écrire des tests unitaires pour le reconcile avec le fake client de controller-runtime.
- Faire des tests d'intégration avec envtest (controller-runtime envtest) ou déployer sur kind et exécuter des tests end-to-end.
- Générer les manifests (RBAC, Kustomize) automatiquement via controller-gen / kubebuilder markers et ajouter targets make.
- Packager l'opérateur dans une image Docker et déployer dans le cluster.

Si tu veux que j'ajoute :
- un exemple de finalizer dans le controller,
- un test unitaire pour Reconcile,
- les manifests RBAC/ServiceAccount/Kustomize pour déployer l'opérateur,
- ou le Dockerfile + pipeline de build — dis-moi lequel et je l'ajoute.


Licence et notes
----------------
Exemple pédagogique : code fourni « tel quel » pour apprentissage. Ne pas utiliser en production sans audits et améliorations (gestion des erreurs, retries backoff, metrics, observabilité, etc.).



