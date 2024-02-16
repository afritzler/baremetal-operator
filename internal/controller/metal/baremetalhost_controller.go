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

	metalv1alpha1 "github.com/afritzler/baremetal-operator/api/metal/v1alpha1"
	"github.com/afritzler/baremetal-operator/internal/bmc"
	"github.com/go-logr/logr"
	"github.com/stmcginnis/gofish/redfish"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BareMetalHostReconciler reconciles a BareMetalHost object
type BareMetalHostReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	BasicAuth bool
}

//+kubebuilder:rbac:groups=metal.afritzler.github.io,resources=baremetalhosts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=metal.afritzler.github.io,resources=baremetalhosts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=metal.afritzler.github.io,resources=baremetalhosts/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BareMetalHostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	host := &metalv1alpha1.BareMetalHost{}
	if err := r.Get(ctx, req.NamespacedName, host); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return ctrl.Result{}, r.reconcileExists(ctx, log, host)
}

func (r *BareMetalHostReconciler) reconcileExists(ctx context.Context, log logr.Logger, host *metalv1alpha1.BareMetalHost) error {
	if !host.DeletionTimestamp.IsZero() {
		return r.delete(ctx, log, host)
	}
	return r.reconcile(ctx, log, host)
}

func (r *BareMetalHostReconciler) delete(ctx context.Context, log logr.Logger, host *metalv1alpha1.BareMetalHost) error {
	log.V(1).Info("Deleting host")

	log.V(1).Info("Deleted host")
	return nil
}

func (r *BareMetalHostReconciler) reconcile(ctx context.Context, log logr.Logger, host *metalv1alpha1.BareMetalHost) error {
	log.V(1).Info("Reconciling host")

	bmcClient, err := r.createBMCClient(ctx, host)
	if err != nil {
		return fmt.Errorf("failed to create BMC client: %w", err)
	}
	defer bmcClient.GetClient().Logout()

	log.V(1).Info("Updating host status from system information")
	if err := r.updateHostStatusFromSystemInfo(ctx, log, host, bmcClient); err != nil {
		return err
	}
	log.V(1).Info("Updated host status from system information")

	log.V(1).Info("Ensuring host power state")
	if err := r.ensurePowerState(ctx, log, bmcClient, host); err != nil {
		return err
	}
	log.V(1).Info("Ensured host power state")

	log.V(1).Info("Ensuring state transition")
	var oldStatus, newStatus metalv1alpha1.HostState
	if oldStatus, newStatus, err = r.ensureHostStatus(ctx, log, host); err != nil {
		return err
	}
	log.V(1).Info("Host status transitioned", "OldSystemStatus", oldStatus, "NewSystemStatus", newStatus)

	log.V(1).Info("Reconciled host")
	return nil
}

func (r *BareMetalHostReconciler) ensurePowerState(_ context.Context, _ logr.Logger, bmcClient bmc.BMC, host *metalv1alpha1.BareMetalHost) error {
	// TODO: this needs to go into the actual state machine
	if host.Status.State == metalv1alpha1.StateInitial {
		if err := bmcClient.SetPXEBootOnce(host.Spec.SystemID); err != nil {
			return fmt.Errorf("failed to set boot PXE once boot order for host: %w", err)
		}
	}

	if host.Spec.Power == metalv1alpha1.PowerStateOn && host.Status.PowerState == redfish.OffPowerState {
		if err := bmcClient.PowerOn(); err != nil {
			return fmt.Errorf("failed to change power state to %s: %w", metalv1alpha1.PowerStateOn, err)
		}
	}

	if host.Spec.Power == metalv1alpha1.PowerStateOff && host.Status.PowerState == redfish.OnPowerState {
		if err := bmcClient.PowerOff(); err != nil {
			return fmt.Errorf("failed to change power state to %s: %w", metalv1alpha1.PowerStateOff, err)
		}
	}

	return nil
}

func (r *BareMetalHostReconciler) createBMCClient(ctx context.Context, host *metalv1alpha1.BareMetalHost) (bmc.BMC, error) {
	var err error
	var bmcClient bmc.BMC

	switch host.Spec.BMC.Type {
	case metalv1alpha1.BMCTypeRedfishLocal:
		bmcClient, err = bmc.NewRedfishLocalBMC(ctx, host.Spec.SystemID, host.Spec.BMC.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to create redfish local client: %w", err)
		}
	case metalv1alpha1.BMCTypeRedfish:
		bmcSecret := &v1.Secret{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: host.Spec.BMC.SecretRef.Namespace, Name: host.Spec.BMC.SecretRef.Name}, bmcSecret); err != nil {
			return nil, fmt.Errorf("failed to get BMC access secret for host: %w", err)
		}
		username, ok := bmcSecret.Data["username"]
		if !ok {
			return nil, fmt.Errorf("no username provided in BMC access secret")
		}
		password, ok := bmcSecret.Data["password"]
		if !ok {
			return nil, fmt.Errorf("no password provided in BMC access secret")
		}
		bmcClient, err = bmc.NewRedfishBMC(ctx, host.Spec.SystemID, host.Spec.BMC.Address, string(username), string(password), r.BasicAuth)
		if err != nil {
			return nil, fmt.Errorf("failed to create redfish client: %w", err)
		}
	default:
		return nil, fmt.Errorf("BMC type %s is not supported", host.Spec.BMC.Type)
	}
	return bmcClient, nil
}

func (r *BareMetalHostReconciler) determineTargetHostStatus(host *metalv1alpha1.BareMetalHost) metalv1alpha1.HostState {
	switch host.Status.State {
	case metalv1alpha1.StateInitial:
		if r.isHostReady(host) {
			return metalv1alpha1.StateAvailable
		}
	// ... [handle other statuses] ...
	case metalv1alpha1.StateAvailable:
		return metalv1alpha1.StateAvailable
	case metalv1alpha1.StateTainted:
		// TODO: this has to be set later by the controller which is responsible for sanitizing the host
		return metalv1alpha1.StateInitial
	}
	return metalv1alpha1.StateInitial
}

// isHostReady checks if the BareMetalHost is ready to be marked as Available.
func (r *BareMetalHostReconciler) isHostReady(host *metalv1alpha1.BareMetalHost) bool {
	// Implement your readiness checks here. This could involve:
	// - Checking if the hardware is properly initialized.

	// This is a placeholder for the actual logic.
	// Return true if the host is ready, false otherwise.
	return true
}

func (r *BareMetalHostReconciler) ensureHostStatus(ctx context.Context, log logr.Logger, host *metalv1alpha1.BareMetalHost) (oldStatus, newStatus metalv1alpha1.HostState, err error) {
	targetHostState := r.determineTargetHostStatus(host)

	if targetHostState == metalv1alpha1.StateInitial {
		if err := r.initializeHost(ctx, log, host, targetHostState); err != nil {
			return "", "", err
		}
	}

	hostBase := host.DeepCopy()
	host.Status.State = targetHostState

	if host.Spec.ClaimRef != nil {
		host.Status.Phase = metalv1alpha1.PhaseBound
		host.Status.State = metalv1alpha1.StateReserved
	}

	if host.Spec.ClaimRef == nil {
		host.Status.Phase = metalv1alpha1.PhaseUnbound
	}

	log.V(1).Info("Patching host status", "State", host.Status.State, "Phase", host.Status.Phase)
	if err := r.Status().Patch(ctx, host, client.MergeFrom(hostBase)); err != nil {
		return "", "", err
	}
	log.V(1).Info("Patched host status", "State", host.Status.State, "Phase", host.Status.Phase)

	return metalv1alpha1.StateInitial, metalv1alpha1.StateInitial, nil
}

func (r *BareMetalHostReconciler) updateHostStatusFromSystemInfo(ctx context.Context, log logr.Logger, host *metalv1alpha1.BareMetalHost, bmcClient bmc.BMC) error {
	log.V(1).Info("Getting system info")
	info, err := bmcClient.GetSystemInfo()
	if err != nil {
		return fmt.Errorf("failed to get system info: %w", err)
	}
	log.V(1).Info("Retrieved system info")

	hostBase := host.DeepCopy()
	host.Status.Manufacturer = info.Manufacturer
	host.Status.Model = info.Model
	host.Status.Health = info.Status.Health
	host.Status.SystemState = info.Status.State
	host.Status.PowerState = info.PowerState

	log.V(1).Info("Found networkinterfaces for host", "NetworkInterfaces", len(info.NetworkInterfaces))
	updateHostNICsFromSystemInfo(info, host)

	log.V(1).Info("Found processors for host", "Processors", len(info.Processors))
	for _, newProcessor := range info.Processors {
		updated := false
		// Check if this Processor ID already exists in host.State.Processors
		for i, existingProcessor := range host.Status.Processors {
			if newProcessor.ID == existingProcessor.ID {
				// Update existing Processor
				host.Status.Processors[i] = metalv1alpha1.Processor{
					ID:                    newProcessor.ID,
					ProcessorType:         newProcessor.ProcessorType,
					ProcessorArchitecture: newProcessor.ProcessorArchitecture,
					InstructionSet:        newProcessor.InstructionSet,
					Manufacturer:          newProcessor.Manufacturer,
					Model:                 newProcessor.Model,
					MHz:                   newProcessor.MaxSpeedMHz,
					Cores:                 newProcessor.TotalCores,
					Threads:               newProcessor.TotalThreads,
				}
				updated = true
				break
			}
		}
		// If Processor ID was not found in existing list, append it
		if !updated {
			host.Status.Processors = append(host.Status.Processors, metalv1alpha1.Processor{
				ID:                    newProcessor.ID,
				ProcessorType:         newProcessor.ProcessorType,
				ProcessorArchitecture: newProcessor.ProcessorArchitecture,
				InstructionSet:        newProcessor.InstructionSet,
				Manufacturer:          newProcessor.Manufacturer,
				Model:                 newProcessor.Model,
				MHz:                   newProcessor.MaxSpeedMHz,
				Cores:                 newProcessor.TotalCores,
				Threads:               newProcessor.TotalThreads,
			})
		}
	}

	log.V(1).Info("Patching system information in host status")
	if err := r.Status().Patch(ctx, host, client.MergeFrom(hostBase)); err != nil {
		return fmt.Errorf("failed to update host status: %w", err)
	}
	log.V(1).Info("Patched system information in host status")
	return nil
}

func updateHostNICsFromSystemInfo(info bmc.SystemInfo, host *metalv1alpha1.BareMetalHost) {
	for _, newNic := range info.NetworkInterfaces {
		updated := false
		// Check if this NIC ID already exists in host.State.NetworkInterfaces
		for i, existingNic := range host.Status.NetworkInterfaces {
			if newNic.ID == existingNic.ID {
				// Update existing NIC
				host.Status.NetworkInterfaces[i] = metalv1alpha1.NetworkInterface{
					ID:                  newNic.ID,
					MACAddress:          newNic.MACAddress,
					PermanentMACAddress: newNic.PermanentMACAddress,
				}
				updated = true
				break
			}
		}
		// If NIC ID was not found in existing list, append it
		if !updated {
			host.Status.NetworkInterfaces = append(host.Status.NetworkInterfaces, metalv1alpha1.NetworkInterface{
				ID:                  newNic.ID,
				MACAddress:          newNic.MACAddress,
				PermanentMACAddress: newNic.PermanentMACAddress,
			})
		}
	}
}

func (r *BareMetalHostReconciler) initializeHost(ctx context.Context, log logr.Logger, host *metalv1alpha1.BareMetalHost, state metalv1alpha1.HostState) error {
	hostBase := host.DeepCopy()

	if host.Spec.ClaimRef == nil && state == metalv1alpha1.StateInitial {
		host.Spec.Power = metalv1alpha1.PowerStateOff
	}

	log.V(1).Info("Patching host after initialization")
	if err := r.Patch(ctx, host, client.MergeFrom(hostBase)); err != nil {
		return err
	}
	log.V(1).Info("Patched host after initialization")

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BareMetalHostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&metalv1alpha1.BareMetalHost{}).
		Complete(r)
}
