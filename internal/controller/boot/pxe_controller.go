/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package boot

import (
	"context"
	"fmt"

	bootv1alpha1 "github.com/afritzler/baremetal-operator/api/boot/v1alpha1"
	"github.com/afritzler/baremetal-operator/api/metal/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/onmetal/controller-utils/clientutils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	pxeConfigFieldOwner = client.FieldOwner("boot.afritzler.github.io/pxe-controller")
)

// PXEReconciler reconciles a PXE object
type PXEReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	PXEServiceNamespace string
}

//+kubebuilder:rbac:groups=boot.afritzler.github.io,resources=pxes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=boot.afritzler.github.io,resources=pxes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=boot.afritzler.github.io,resources=pxes/finalizers,verbs=update
//+kubebuilder:rbac:groups=metal.afritzler.github.io,resources=baremetalhostclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=metal.afritzler.github.io,resources=baremetalhostclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=metal.afritzler.github.io,resources=baremetalhosts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=metal.afritzler.github.io,resources=baremetalhosts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PXEReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	pxeConfig := &bootv1alpha1.PXE{}
	if err := r.Get(ctx, req.NamespacedName, pxeConfig); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconcileExists(ctx, log, pxeConfig)
}

func (r *PXEReconciler) reconcileExists(ctx context.Context, log logr.Logger, config *bootv1alpha1.PXE) (ctrl.Result, error) {
	if !config.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, config)
	}
	return r.reconcile(ctx, log, config)
}

func (r *PXEReconciler) delete(ctx context.Context, _ logr.Logger, pxeConfig *bootv1alpha1.PXE) (ctrl.Result, error) {
	pxeSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.PXEServiceNamespace,
			Name:      fmt.Sprintf("ipxe-%s", pxeConfig.Spec.SystemUUID),
		},
	}
	if err := r.Delete(ctx, pxeSecret); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove PXE secret: %w", err)
	}

	if _, err := clientutils.PatchEnsureNoFinalizer(ctx, r.Client, pxeConfig, bootv1alpha1.PXEFinalizer); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PXEReconciler) reconcile(ctx context.Context, log logr.Logger, pxeConfig *bootv1alpha1.PXE) (ctrl.Result, error) {
	log.V(1).Info("Reconciling PXE configuration")

	log.V(1).Info("Ensuring finalizer")
	if modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, pxeConfig, bootv1alpha1.PXEFinalizer); err != nil || modified {
		return ctrl.Result{}, err
	}

	hostClaim := &v1alpha1.BareMetalHostClaim{}
	ignitionSecret := &v1.Secret{}
	if pxeConfig.Spec.BareMetalHostClaimRef.Name != "" {
		if err := r.Get(ctx, client.ObjectKey{Namespace: pxeConfig.Namespace, Name: pxeConfig.Spec.BareMetalHostClaimRef.Name}, hostClaim); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Get(ctx, client.ObjectKey{Namespace: pxeConfig.Namespace, Name: pxeConfig.Spec.IgnitionRef.Name}, ignitionSecret); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// nothing to do as there is not Igniton
		return ctrl.Result{}, nil
	}

	pxeSecret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.PXEServiceNamespace,
			Name:      fmt.Sprintf("ipxe-%s", pxeConfig.Spec.SystemUUID),
		},
		Data: ignitionSecret.Data,
	}

	if err := r.Patch(ctx, pxeSecret, client.Apply, pxeConfigFieldOwner); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed applying PXE secret in %s namespace: %w", r.PXEServiceNamespace, err)
	}

	pxeConfigBase := pxeConfig.DeepCopy()
	pxeConfig.Status.State = bootv1alpha1.PXEStateReady
	if err := r.Status().Patch(ctx, pxeConfig, client.MergeFrom(pxeConfigBase)); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PXEReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bootv1alpha1.PXE{}).
		Complete(r)
}
