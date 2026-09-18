package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	examplev1 "github.com/example/simple-operator/api/v1alpha1"
)

// +kubebuilder:rbac:groups=config.example.com,resources=cvfavorites,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.example.com,resources=cvfavorites/status,verbs=get;update;patch

// CVFavoriteReconciler handles CVFavorite objects by calling the external favorites service
type CVFavoriteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type favoriteRequest struct {
	UserID string `json:"userId"`
	CVID   string `json:"cvId"`
	Action string `json:"action"`
}

func (r *CVFavoriteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cr examplev1.CVFavorite
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// prepare request
	fr := favoriteRequest{UserID: cr.Spec.UserID, CVID: cr.Spec.CVID, Action: cr.Spec.Action}
	body, _ := json.Marshal(fr)

	// call external service
	clientHTTP := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("%s/favorites", cr.Spec.ServiceURL)
	logger.Info("calling favorites service", "url", url, "body", string(body))
	resp, err := clientHTTP.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		cr.Status.Synced = false
		cr.Status.Message = fmt.Sprintf("call failed: %v", err)
		_ = r.Status().Update(ctx, &cr)
		logger.Error(err, "failed to call favorites service")
		return ctrl.Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		cr.Status.Synced = true
		cr.Status.Message = fmt.Sprintf("service responded %d", resp.StatusCode)
	} else {
		cr.Status.Synced = false
		cr.Status.Message = fmt.Sprintf("service error %d", resp.StatusCode)
	}

	if err := r.Status().Update(ctx, &cr); err != nil {
		logger.Error(err, "failed to update CVFavorite status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CVFavoriteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplev1.CVFavorite{}).
		Complete(r)
}
