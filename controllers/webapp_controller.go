package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	examplev1 "github.com/example/simple-operator/api/v1alpha1"
)

// SimpleConfigReconciler reconciles a SimpleConfig object
type SimpleConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// RBAC: allow managing SimpleConfig (config.example.com) and ConfigMaps
// +kubebuilder:rbac:groups=config.example.com,resources=simpleconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.example.com,resources=simpleconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (r *SimpleConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var sc examplev1.SimpleConfig
	if err := r.Get(ctx, req.NamespacedName, &sc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cmName := sc.Name + "-cm"
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Namespace: sc.Namespace, Name: cmName}, cm)
	if err != nil {
		// not found => create
		newCm := &corev1.ConfigMap{
			ObjectMeta: ctrl.ObjectMeta{Namespace: sc.Namespace, Name: cmName},
			Data:       sc.Spec.Data,
		}
		if err := controllerutil.SetControllerReference(&sc, newCm, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, newCm); err != nil {
			logger.Error(err, "failed to create ConfigMap")
			return ctrl.Result{}, err
		}
		// update status
		sc.Status.ConfigMapName = cmName
		sc.Status.Synced = true
		if err := r.Status().Update(ctx, &sc); err != nil {
			logger.Error(err, "failed to update SimpleConfig status")
		}
		return ctrl.Result{}, nil
	}

	// found -> ensure data matches
	if !equalMaps(cm.Data, sc.Spec.Data) {
		cm.Data = sc.Spec.Data
		if err := r.Update(ctx, cm); err != nil {
			logger.Error(err, "failed to update ConfigMap")
			return ctrl.Result{}, err
		}
	}

	// ensure status
	if sc.Status.ConfigMapName != cmName || sc.Status.Synced != true {
		sc.Status.ConfigMapName = cmName
		sc.Status.Synced = true
		if err := r.Status().Update(ctx, &sc); err != nil {
			logger.Error(err, "failed to update SimpleConfig status")
		}
	}

	return ctrl.Result{}, nil
}

func equalMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func (r *SimpleConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplev1.SimpleConfig{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
