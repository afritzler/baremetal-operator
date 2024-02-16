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

package metal

import (
	"context"
	"fmt"

	"github.com/afritzler/baremetal-operator/api/boot/v1alpha1"
	metalv1alpha1 "github.com/afritzler/baremetal-operator/api/metal/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/onmetal/controller-utils/clientutils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	bareMetalClaimFieldOwner = client.FieldOwner("metal.afritzler.github.io/hostclaim-controller")
)

// BareMetalHostClaimReconciler reconciles a BareMetalHostClaim object
type BareMetalHostClaimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.afritzler.github.io,resources=baremetalhostclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.afritzler.github.io,resources=baremetalhostclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.afritzler.github.io,resources=baremetalhostclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=core.afritzler.github.io,resources=baremetalhosts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.afritzler.github.io,resources=baremetalhosts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.afritzler.github.io,resources=baremetalhosts/finalizers,verbs=update
//+kubebuilder:rbac:groups=boot.afritzler.github.io,resources=pxes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=boot.afritzler.github.io,resources=pxes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=boot.afritzler.github.io,resources=pxes/finalizers,verbs=update
//+kubebuilder:rbac:groups=boot.afritzler.github.io,resources=dhcps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=boot.afritzler.github.io,resources=dhcps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=boot.afritzler.github.io,resources=dhcps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BareMetalHostClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	claim := &metalv1alpha1.BareMetalHostClaim{}
	if err := r.Get(ctx, req.NamespacedName, claim); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconcileExists(ctx, log, claim)
}

func (r *BareMetalHostClaimReconciler) reconcileExists(ctx context.Context, log logr.Logger, claim *metalv1alpha1.BareMetalHostClaim) (ctrl.Result, error) {
	if !claim.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, claim)
	}
	return r.reconcile(ctx, log, claim)
}

func (r *BareMetalHostClaimReconciler) delete(ctx context.Context, log logr.Logger, claim *metalv1alpha1.BareMetalHostClaim) (ctrl.Result, error) {
	log.V(1).Info("Deleting host claim")
	host := &metalv1alpha1.BareMetalHost{}
	if err := r.Get(ctx, types.NamespacedName{Name: claim.Spec.BareMetalHostRef.Name}, host); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get host for claim: %w", err)
	}

	log.V(1).Info("Removing claimRef on host", "Host", host.Name)
	hostBase := host.DeepCopy()
	host.Spec.ClaimRef = nil
	if err := r.Patch(ctx, host, client.MergeFrom(hostBase)); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove claimRef from host: %w", err)
	}
	log.V(1).Info("Removed claimRef on host", "Host", host.Name)

	log.V(1).Info("Patching host status", "Host", host.Name, "HostState", host.Status.State)
	host.Status.State = metalv1alpha1.StateTainted
	if err := r.Status().Patch(ctx, host, client.MergeFrom(hostBase)); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to patch host status: %w", err)
	}
	log.V(1).Info("Patched host status", "Host", host.Name, "HostState", host.Status.State)

	log.V(1).Info("Removing finalizer on host", "Host", host.Name)
	if _, err := clientutils.PatchEnsureNoFinalizer(ctx, r.Client, host, metalv1alpha1.BareMetalHostClaimFinalizer); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove finalizer from host: %w", err)
	}
	if _, err := clientutils.PatchEnsureNoFinalizer(ctx, r.Client, claim, metalv1alpha1.BareMetalHostClaimFinalizer); err != nil {
		return ctrl.Result{}, err
	}
	log.V(1).Info("Removed finalizer on host", "Host", host.Name)

	log.V(1).Info("Deleted host claim")

	return ctrl.Result{}, nil
}

func (r *BareMetalHostClaimReconciler) reconcile(ctx context.Context, log logr.Logger, claim *metalv1alpha1.BareMetalHostClaim) (ctrl.Result, error) {
	log.V(1).Info("Ensuring finalizer")
	if modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, claim, metalv1alpha1.BareMetalHostClaimFinalizer); err != nil || modified {
		return ctrl.Result{}, err
	}
	host := &metalv1alpha1.BareMetalHost{}
	if err := r.Get(ctx, types.NamespacedName{Name: claim.Spec.BareMetalHostRef.Name}, host); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get host for claim: %w", err)
	}
	if host.Spec.ClaimRef != nil && host.Spec.ClaimRef.UID != claim.UID {
		return ctrl.Result{}, fmt.Errorf("failed to claim host %s as it is already in claimed by somebody else", host.Name)
	}
	if modified, err := clientutils.PatchEnsureFinalizer(ctx, r.Client, host, metalv1alpha1.BareMetalHostClaimFinalizer); err != nil || modified {
		return ctrl.Result{}, err
	}
	log.V(1).Info("Ensured finalizer")

	log.V(1).Info("Apply PXE configuration")
	// TODO: we should wait until the PXE configuration is ready
	if err := r.applyPXEConfiguration(ctx, log, claim, host); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply PXE configuration: %w", err)
	}
	log.V(1).Info("Applied PXE configuration")

	log.V(1).Info("Apply DHCP configuration")
	// TODO: we should wait until the DHCP configuration is ready
	if err := r.applyDHCPConfiguration(ctx, log, claim); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply DHCP configuration: %w", err)
	}
	log.V(1).Info("Applied DHCP configuration")

	if host.Spec.ClaimRef == nil {
		log.V(1).Info("Applying claimRef on host", "Host", host.Name)
		hostBase := host.DeepCopy()
		host.Spec.ClaimRef = &v1.ObjectReference{
			Kind:      "BareMetalHostClaim",
			Namespace: claim.Namespace,
			Name:      claim.Name,
			UID:       claim.UID,
		}
		if err := r.Patch(ctx, host, client.MergeFrom(hostBase)); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to patch claimRef on host %s: %w", host.Name, err)
		}
		log.V(1).Info("Applied claimRef on host", "Host", host.Name)
	}

	if host.Spec.ClaimRef != nil {
		log.V(1).Info("Ensure power state")
		// only power on machine if the PXE configuration is ready
		if claim.Spec.Power == metalv1alpha1.PowerStateOn && claim.Spec.IgnitionRef != nil {
			pxeConfig := &v1alpha1.PXE{}
			if err := r.Get(ctx, client.ObjectKey{Namespace: claim.Namespace, Name: claim.Name}, pxeConfig); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to get PXE configuration for claim: %w", err)
			}
			if pxeConfig.Status.State == v1alpha1.PXEStateReady {
				hostBase := host.DeepCopy()
				host.Spec.Power = claim.Spec.Power
				if err := r.Patch(ctx, host, client.MergeFrom(hostBase)); err != nil {
					return ctrl.Result{}, fmt.Errorf("faield to patch the power status on host %s: %w", host.Name, err)
				}
			}
		} else {
			hostBase := host.DeepCopy()
			host.Spec.Power = claim.Spec.Power
			if err := r.Patch(ctx, host, client.MergeFrom(hostBase)); err != nil {
				return ctrl.Result{}, fmt.Errorf("faield to patch the power status on host %s: %w", host.Name, err)
			}
		}
		log.V(1).Info("Ensured power state")
	}

	claimBase := claim.DeepCopy()
	claim.Status.Phase = metalv1alpha1.PhaseBound
	if err := r.Status().Patch(ctx, claim, client.MergeFrom(claimBase)); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BareMetalHostClaimReconciler) applyPXEConfiguration(ctx context.Context, _ logr.Logger, claim *metalv1alpha1.BareMetalHostClaim, host *metalv1alpha1.BareMetalHost) error {
	pxe := &v1alpha1.PXE{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PXE",
			APIVersion: "boot.afritzler.github.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: claim.Namespace,
			Name:      claim.Name,
		},
		Spec: v1alpha1.PXESpec{
			BareMetalHostClaimRef: v1.LocalObjectReference{Name: claim.Name},
			IgnitionRef:           claim.Spec.IgnitionRef,
			Image:                 claim.Spec.Image,
			FooUUID:               host.Spec.FooUUID,
		},
	}

	if err := controllerutil.SetOwnerReference(claim, pxe, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on PXE configuration: %w", err)
	}

	if err := r.Patch(ctx, pxe, client.Apply, bareMetalClaimFieldOwner, client.ForceOwnership); err != nil {
		return fmt.Errorf("failed to apply PXE configuration: %w", err)
	}

	pxeBase := pxe.DeepCopy()
	pxe.Status.State = v1alpha1.PXEStateCreated
	if err := r.Status().Patch(ctx, pxe, client.MergeFrom(pxeBase)); err != nil {
		return fmt.Errorf("failed to apply PXE configuration: %w", err)
	}

	return nil
}

func (r *BareMetalHostClaimReconciler) applyDHCPConfiguration(ctx context.Context, _ logr.Logger, claim *metalv1alpha1.BareMetalHostClaim) error {
	dhcp := &v1alpha1.DHCP{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DHCP",
			APIVersion: "boot.afritzler.github.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: claim.Namespace,
			Name:      claim.Name,
		},
		Spec: v1alpha1.DHCPSpec{
			BareMetalHostRef: claim.Spec.BareMetalHostRef,
		},
	}

	if err := controllerutil.SetOwnerReference(claim, dhcp, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on DHCP configuration: %w", err)
	}

	if err := r.Patch(ctx, dhcp, client.Apply, bareMetalClaimFieldOwner, client.ForceOwnership); err != nil {
		return fmt.Errorf("failed to apply DHCP configuration: %w", err)
	}

	dhcpBase := dhcp.DeepCopy()
	dhcp.Status.State = v1alpha1.DHCPStateCreated
	if err := r.Status().Patch(ctx, dhcp, client.MergeFrom(dhcpBase)); err != nil {
		return fmt.Errorf("failed to patch DHCP configuration status: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BareMetalHostClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&metalv1alpha1.BareMetalHostClaim{}).
		Owns(&v1alpha1.PXE{}).
		Owns(&v1alpha1.DHCP{}).
		// TODO: watch setup for ignition secret
		Watches(&metalv1alpha1.BareMetalHost{}, r.enqueueBareMetalHostClaimsByRefs()).
		Complete(r)
}

func (r *BareMetalHostClaimReconciler) enqueueBareMetalHostClaimsByRefs() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []reconcile.Request {
		log := ctrl.LoggerFrom(ctx)

		host := object.(*metalv1alpha1.BareMetalHost)
		var req []reconcile.Request
		claimList := &metalv1alpha1.BareMetalHostClaimList{}
		if err := r.List(ctx, claimList); err != nil {
			log.Error(err, "failed to list host claims")
			return nil
		}
		for _, claim := range claimList.Items {
			if claim.Spec.BareMetalHostRef.Name == host.Name {
				req = append(req, reconcile.Request{
					NamespacedName: types.NamespacedName{Namespace: claim.Namespace, Name: claim.Name},
				})
				return req
			}
		}

		return req
	})
}
