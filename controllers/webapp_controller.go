package controllers

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	webappv1alpha1 "github.com/you/webapp-operator/api/v1alpha1"
)

const webAppFinalizer = "webapp.apps.mycompany.com/finalizer"

// WebAppReconciler reconciles a WebApp object
type WebAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// RBAC permissions
// +kubebuilder:rbac:groups=apps.mycompany.com,resources=webapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.mycompany.com,resources=webapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var webapp webappv1alpha1.WebApp
	if err := r.Get(ctx, req.NamespacedName, &webapp); err != nil {
		// NotFound or other
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Set default replicas if not provided
	if webapp.Spec.Replicas == nil {
		var d int32 = 1
		webapp.Spec.Replicas = &d
	}

	// Desired Deployment
	dep := r.desiredDeployment(&webapp)
	if err := controllerutil.SetControllerReference(&webapp, dep, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	var existingDep appsv1.Deployment
	err := r.Get(ctx, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name}, &existingDep)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if client.IgnoreNotFound(err) == nil {
		// create
		if err := r.Create(ctx, dep); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// update if spec changed
		if !reflect.DeepEqual(existingDep.Spec, dep.Spec) {
			existingDep.Spec = dep.Spec
			if err := r.Update(ctx, &existingDep); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Desired Service
	svc := r.desiredService(&webapp)
	if err := controllerutil.SetControllerReference(&webapp, svc, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	var existingSvc corev1.Service
	err = r.Get(ctx, client.ObjectKey{Namespace: svc.Namespace, Name: svc.Name}, &existingSvc)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if client.IgnoreNotFound(err) == nil {
		// create
		if err := r.Create(ctx, svc); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// update (preserve ClusterIP)
		svc.Spec.ClusterIP = existingSvc.Spec.ClusterIP
		if !reflect.DeepEqual(existingSvc.Spec, svc.Spec) {
			existingSvc.Spec = svc.Spec
			if err := r.Update(ctx, &existingSvc); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Update status from deployment
	var updatedDep appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name}, &updatedDep); err != nil {
		return ctrl.Result{}, err
	}
	available := updatedDep.Status.AvailableReplicas
	if webapp.Status.AvailableReplicas != available {
		webapp.Status.AvailableReplicas = available
		if err := r.Status().Update(ctx, &webapp); err != nil {
			logger.Error(err, "failed to update WebApp status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *WebAppReconciler) desiredDeployment(w *webappv1alpha1.WebApp) *appsv1.Deployment {
	labels := map[string]string{"app": w.Name}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-dep",
			Namespace: w.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: w.Spec.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "web",
						Image: w.Spec.Image,
						Ports: []corev1.ContainerPort{{ContainerPort: w.Spec.Port}},
					}},
				},
			},
		},
	}
}

func (r *WebAppReconciler) desiredService(w *webappv1alpha1.WebApp) *corev1.Service {
	labels := map[string]string{"app": w.Name}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name + "-svc",
			Namespace: w.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port:       w.Spec.Port,
				TargetPort: intstr.FromInt(int(w.Spec.Port)),
			}},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func (r *WebAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1alpha1.WebApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
