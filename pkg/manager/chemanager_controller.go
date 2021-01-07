//
// Copyright (c) 2019-2020 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

package manager

import (
	"context"

	"github.com/che-incubator/devworkspace-che-operator/apis/che-controller/v1alpha1"
	"github.com/che-incubator/devworkspace-che-operator/pkg/defaults"
	"github.com/che-incubator/devworkspace-che-operator/pkg/infrastructure"
	"github.com/che-incubator/devworkspace-che-operator/pkg/sync"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log = ctrl.Log.WithName("che")

	routeDiffOpts = cmp.Options{
		cmpopts.IgnoreFields(routev1.Route{}, "TypeMeta", "ObjectMeta", "Status"),
		cmpopts.IgnoreFields(routev1.RouteSpec{}, "WildcardPolicy"),
	}

	ingressDiffOpts = cmp.Options{
		cmpopts.IgnoreFields(v1beta1.Ingress{}, "TypeMeta", "ObjectMeta", "Status"),
	}
)

type CheReconciler struct {
	client  client.Client
	scheme  *runtime.Scheme
	gateway CheGateway
	syncer  sync.Syncer
}

func (r *CheReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.client = mgr.GetClient()
	r.scheme = mgr.GetScheme()
	r.gateway.client = mgr.GetClient()
	r.gateway.scheme = mgr.GetScheme()
	r.syncer = sync.New(r.client, r.scheme)

	bld := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CheManager{}).
		Owns(&corev1.Service{}).
		Owns(&v1beta1.Ingress{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbac.Role{}).
		Owns(&rbac.RoleBinding{})
	if infrastructure.Current.Type == infrastructure.OpenShift {
		bld.Owns(&routev1.Route{})
	}
	return bld.Complete(r)
}

func (r *CheReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	// make sure we've checked we're in a valid state
	current := &v1alpha1.CheManager{}
	err := r.client.Get(ctx, req.NamespacedName, current)
	if err != nil {
		if errors.IsNotFound(err) {
			// Ok, our current router disappeared...
			return ctrl.Result{}, nil
		}
		// other error - let's requeue
		return ctrl.Result{}, err
	}

	if current.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.finalize(current)
	}

	var anyChanged, changed bool

	if changed, err = r.reconcileGateway(ctx, current); err != nil {
		return ctrl.Result{}, err
	}
	anyChanged = anyChanged || changed

	if infrastructure.Current.Type == infrastructure.OpenShift {
		if changed, err = r.reconcileRoute(ctx, current); err != nil {
			return ctrl.Result{}, err
		}
		anyChanged = anyChanged || changed
	} else {
		if changed, err = r.reconcileIngress(ctx, current); err != nil {
			return ctrl.Result{}, err
		}
		anyChanged = anyChanged || changed
	}

	return r.updateStatus(ctx, current, anyChanged)
}

func (r *CheReconciler) updateStatus(ctx context.Context, manager *v1alpha1.CheManager, changed bool) (ctrl.Result, error) {
	currentPhase := manager.Status.GatewayPhase

	if manager.Spec.Routing == v1alpha1.MultiHost {
		manager.Status.GatewayPhase = v1alpha1.GatewayPhaseInactive
	} else if changed {
		manager.Status.GatewayPhase = v1alpha1.GatewayPhaseInitializing
	} else {
		manager.Status.GatewayPhase = v1alpha1.GatewayPhaseEstablished
	}

	if currentPhase != manager.Status.GatewayPhase {
		return ctrl.Result{Requeue: true}, r.client.Status().Update(ctx, manager)
	}

	return ctrl.Result{Requeue: currentPhase == v1alpha1.GatewayPhaseInitializing}, nil
}

func (r *CheReconciler) finalize(router *v1alpha1.CheManager) error {
	// implement if needed
	return nil
}

func (r *CheReconciler) reconcileGateway(ctx context.Context, manager *v1alpha1.CheManager) (bool, error) {
	var changed bool
	var err error

	if manager.Spec.Routing == v1alpha1.SingleHost {
		changed, err = r.gateway.Sync(ctx, manager)
	} else {
		changed, err = true, r.gateway.Delete(ctx, manager)
	}

	return changed, err
}

func (r *CheReconciler) reconcileRoute(ctx context.Context, manager *v1alpha1.CheManager) (bool, error) {
	route := getRouteSpec(manager)
	var changed bool
	var err error

	if manager.Spec.Routing == v1alpha1.SingleHost {
		changed, err = r.syncer.Sync(ctx, manager, route, routeDiffOpts)
	} else {
		changed, err = true, r.syncer.Delete(ctx, route)
	}

	return changed, err
}

func (r *CheReconciler) reconcileIngress(ctx context.Context, manager *v1alpha1.CheManager) (bool, error) {
	ingress := getIngressSpec(manager)
	var changed bool
	var err error

	if manager.Spec.Routing == v1alpha1.SingleHost {
		changed, err = r.syncer.Sync(ctx, manager, ingress, ingressDiffOpts)
	} else {
		changed, err = true, r.syncer.Delete(ctx, ingress)
	}

	return changed, err
}

func getRouteSpec(manager *v1alpha1.CheManager) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      manager.Name,
			Namespace: manager.Namespace,
			Labels:    defaults.GetLabelsForComponent(manager, "external-access"),
		},
		Spec: routev1.RouteSpec{
			Host: manager.Spec.Host,
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: GetGatewayServiceName(manager),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt(GatewayPort),
			},
			TLS: &routev1.TLSConfig{
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
				Termination:                   routev1.TLSTerminationEdge,
			},
		},
	}
}

func getIngressSpec(manager *v1alpha1.CheManager) *v1beta1.Ingress {
	pathType := v1beta1.PathTypeImplementationSpecific
	return &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      manager.Name,
			Namespace: manager.Namespace,
			Labels:    defaults.GetLabelsForComponent(manager, "external-access"),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                       "nginx",
				"nginx.ingress.kubernetes.io/proxy-read-timeout":    "3600",
				"nginx.ingress.kubernetes.io/proxy-connect-timeout": "3600",
				// "nginx.ingress.kubernetes.io/ssl-redirect":          "true", - do we need this?
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: manager.Spec.Host,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: v1beta1.IngressBackend{
										ServiceName: GetGatewayServiceName(manager),
										ServicePort: intstr.FromInt(GatewayPort),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
