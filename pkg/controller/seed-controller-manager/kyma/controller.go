package kyma

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/kyma"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	batchv1 "k8s.io/api/batch/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kyma_controller"
)

type reconciler struct {
	log                     *zap.SugaredLogger
	workerNameLabelSelector labels.Selector
	workerName              string
	recorder                record.EventRecorder
	namespace               string
	versions                kubermatic.Versions
	seedClient              ctrlruntimeclient.Client
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	namespace string,
	numWorkers int,
	versions kubermatic.Versions,
) error {

	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %v", err)
	}

	reconciler := &reconciler{
		log:                     log.Named(ControllerName),
		workerNameLabelSelector: workerSelector,
		workerName:              workerName,
		recorder:                mgr.GetEventRecorderFor(ControllerName),
		namespace:               namespace,
		seedClient:              mgr.GetClient(),
		versions:                versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	if err := c.Watch(&source.Kind{Type: &batchv1.Job{}}, controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient())); err != nil {
		return fmt.Errorf("failed to create watcher for jobs: %v", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch for user cluster: %v", err)
	}

	return nil
}

// Reconcile reconciles the kubermatic cluster template instance in the seed cluster
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {

	log := r.log.With("request", request)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster %s: %w", cluster.Name, ctrlruntimeclient.IgnoreNotFound(err))
	}

	err := r.reconcile(ctx, cluster, log)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster, log *zap.SugaredLogger) error {
	if !helper.ClusterConditionHasStatus(cluster, kubermaticv1.ClusterConditionClusterControllerReconcilingSuccess, corev1.ConditionTrue) {
		return nil
	}

	if _, ok := cluster.Labels["kyma"]; ok {
		if helper.ClusterConditionHasStatus(cluster, kubermaticv1.ClusterConditionKymaInstalled, corev1.ConditionTrue) {
			return nil
		}
		kymaInstallerJob := &batchv1.Job{}
		err := r.seedClient.Get(ctx, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.KymaJobInstallationName}, kymaInstallerJob)
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("error while getting the installation job: %v", err)
		}
		if kymaInstallerJob.Status.Succeeded == 1 {
			helper.SetClusterCondition(cluster, r.versions, kubermaticv1.ClusterConditionKymaInstalled, corev1.ConditionTrue, kubermaticv1.ReasonKymaInstallationCompleted, "")
		} else {
			kymaInstallerCreator := []reconciling.NamedJobCreatorGetter{
				kyma.InstallationJobCreator(),
			}
			if err := reconciling.ReconcileJobs(ctx, kymaInstallerCreator, cluster.Status.NamespaceName, r.seedClient); err != nil {
				return err
			}
			helper.SetClusterCondition(cluster, r.versions, kubermaticv1.ClusterConditionKymaInstalled, corev1.ConditionFalse, kubermaticv1.ReasonKymaInstallationInProgress, "")
		}
		return nil
	} else {
		_, condition := helper.GetClusterCondition(cluster, kubermaticv1.ClusterConditionKymaInstalled)
		if condition == nil || condition.Reason == kubermaticv1.ReasonKymaUninstalled {
			return nil
		}
		//TODO: to finish unistallation
	}

	return nil
}
