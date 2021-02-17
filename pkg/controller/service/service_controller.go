package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	_ "github.com/lib/pq"
	"github.com/google/uuid"
)

const (
	// TODO: Add a description for the constants
	serviceNameAnnotation  = "buildly.discovery.k8s.io/service_name"
	endpointNameAnnotation = "buildly.discovery.k8s.io/endpoint_name"
	databaseLogicModuleTable = "gateway_logicmodule"

	dbHost     = os.Getenv("DATABASE_HOST")
	dbPort     = os.Getenv("DATABASE_PORT")
	dbUser     = os.Getenv("DATABASE_USER")
	dbPassword = os.Getenv("DATABASE_PASSWORD")
	dbName     = os.Getenv("DATABASE_NAME")
)

var log = logf.Log.WithName("controller_buildly_discovery")

// Add creates a new Service Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileService{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("service-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to resource Service
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileService{}

// ReconcileService reconciles a Service object
type ReconcileService struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Service object and makes changes based on the state read
// and what is in the Service.Spec
func (r *ReconcileService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Service")

	// TODO: Remove this check because the operator should work only in the namespace it's deployed
	if request.Namespace != "walhall-dev-local" {
		return reconcile.Result{}, nil
	}

	// Fetch the Service instance
	instance := &corev1.Service{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return reconcile.Result{}, nil
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return reconcile.Result{}, nil
	}

	// TODO: If the service is being created we add to the database
	// TODO: If the service is being delete we need to remove from the db
	// TODO: If the service is being update we might need to update it the db (https://bit.ly/2m20LNZ)
	// TODO: Check if the connection will be closed after the whole execution

	var endpointName, serviceName string

	// Get the endpoint name from the annotation or use the K8S object name if not defined
	// We recommend users to set at least the endpoint name
	if value, found := instance.Annotations[endpointNameAnnotation]; found {
		endpointName = value
	} else {
		endpointName = instance.Spec.Selector["app"]
	}

	// Get the service name from the annotation or use the K8S object name if not defined
	if value, found := instance.Annotations[serviceNameAnnotation]; found {
		serviceName = value
	} else {
		serviceName = instance.Spec.Selector["app"]
	}

	// Generate UUID and endpoint of service
	moduleUUID := uuid.New().String()
	servicePort := instance.Spec.Ports[0].Port
	endpoint := fmt.Sprintf("http://%s.%s:%d", instance.Name, instance.Namespace, servicePort)

	// Create SQL statement to add service to Buildly instance database
	sqlStatement := fmt.Sprintf("INSERT INTO %s (module_uuid, name, endpoint, endpoint_name) VALUES ($1, $2, $3, $4)", databaseLogicModuleTable)
	_, err = db.Exec(sqlStatement, moduleUUID, serviceName, endpoint, endpointName)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
