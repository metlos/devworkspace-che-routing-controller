package manager

import (
	"context"
	"reflect"
	"testing"

	"github.com/che-incubator/devworkspace-che-operator/apis/che-controller/v1alpha1"
	"github.com/che-incubator/devworkspace-che-operator/pkg/defaults"
	"github.com/che-incubator/devworkspace-che-operator/pkg/sync"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestCreatesObjectsInSingleHost(t *testing.T) {
	managerName := "che"
	ns := "default"
	scheme := createTestScheme()
	ctx := context.TODO()
	cl := fake.NewFakeClientWithScheme(scheme, &v1alpha1.CheManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      managerName,
			Namespace: ns,
		},
		Spec: v1alpha1.CheManagerSpec{
			Host:    "over.the.rainbow",
			Routing: v1alpha1.SingleHost,
		},
	})

	reconciler := CheReconciler{client: cl, scheme: scheme, gateway: CheGateway{client: cl, scheme: scheme}, syncer: sync.New(cl, scheme)}

	_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: managerName, Namespace: ns}})
	if err != nil {
		t.Fatalf("Failed to reconcile che manager with error: %s", err)
	}

	testGatewayObjectsExist(t, ctx, cl, managerName, ns)
}

func TestUpdatesObjectsInSingleHost(t *testing.T) {
	managerName := "che"
	ns := "default"

	scheme := createTestScheme()

	cl := fake.NewFakeClientWithScheme(scheme,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
				Labels: map[string]string{
					"some":                   "label",
					"app.kubernetes.io/name": "not what we expect",
				},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
		},
		&rbac.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
		},
		&rbac.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
		},
		&v1alpha1.CheManager{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
			Spec: v1alpha1.CheManagerSpec{
				Host:    "over.the.rainbow",
				Routing: v1alpha1.SingleHost,
			},
		})

	ctx := context.TODO()

	reconciler := CheReconciler{client: cl, scheme: scheme, gateway: CheGateway{client: cl, scheme: scheme}, syncer: sync.New(cl, scheme)}

	_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: managerName, Namespace: ns}})
	if err != nil {
		t.Fatalf("Failed to reconcile che manager with error: %s", err)
	}

	testGatewayObjectsExist(t, ctx, cl, managerName, ns)

	depl := &appsv1.Deployment{}
	if err = cl.Get(ctx, client.ObjectKey{Name: managerName, Namespace: ns}, depl); err != nil {
		t.Fatalf("Failed to read the che manager deployment that should exist")
	}

	// checking that we got the update we wanted on the labels...
	expectedLabels := defaults.GetLabelsFromNames(managerName, "deployment")
	expectedLabels["some"] = "label"

	if !reflect.DeepEqual(expectedLabels, depl.GetLabels()) {
		t.Errorf("The deployment should have had its labels reset by the reconciler.")
	}
}

func TestDoesntCreateObjectsInMultiHost(t *testing.T) {
	managerName := "che"
	ns := "default"
	scheme := createTestScheme()
	ctx := context.TODO()
	cl := fake.NewFakeClientWithScheme(scheme, &v1alpha1.CheManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      managerName,
			Namespace: ns,
		},
		Spec: v1alpha1.CheManagerSpec{
			Host:    "over.the.rainbow",
			Routing: v1alpha1.MultiHost,
		},
	})

	reconciler := CheReconciler{client: cl, scheme: scheme, gateway: CheGateway{client: cl, scheme: scheme}, syncer: sync.New(cl, scheme)}

	_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: managerName, Namespace: ns}})
	if err != nil {
		t.Fatalf("Failed to reconcile che manager with error: %s", err)
	}

	testGatewayObjectsDontExist(t, ctx, cl, managerName, ns)
}

func TestDeletesObjectsInMultiHost(t *testing.T) {
	managerName := "che"
	ns := "default"

	scheme := createTestScheme()

	cl := fake.NewFakeClientWithScheme(scheme,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
		},
		&rbac.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
		},
		&rbac.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
		},
		&v1alpha1.CheManager{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managerName,
				Namespace: ns,
			},
			Spec: v1alpha1.CheManagerSpec{
				Host:    "over.the.rainbow",
				Routing: v1alpha1.MultiHost,
			},
		})

	ctx := context.TODO()

	reconciler := CheReconciler{client: cl, scheme: scheme, gateway: CheGateway{client: cl, scheme: scheme}, syncer: sync.New(cl, scheme)}

	_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: managerName, Namespace: ns}})
	if err != nil {
		t.Fatalf("Failed to reconcile che manager with error: %s", err)
	}

	testGatewayObjectsDontExist(t, ctx, cl, managerName, ns)
}
