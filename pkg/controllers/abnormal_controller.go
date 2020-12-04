/*
Copyright 2020 The Kube Diagnoser Authors.

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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	diagnosisv1 "netease.com/k8s/kube-diagnoser/api/v1"
	"netease.com/k8s/kube-diagnoser/pkg/util"
)

// AbnormalReconciler reconciles a Abnormal object.
type AbnormalReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	mode                 string
	sourceManagerCh      chan diagnosisv1.Abnormal
	informationManagerCh chan diagnosisv1.Abnormal
	diagnoserChainCh     chan diagnosisv1.Abnormal
	recovererChainCh     chan diagnosisv1.Abnormal
}

// NewAbnormalReconciler creates a new AbnormalReconciler.
func NewAbnormalReconciler(
	cli client.Client,
	log logr.Logger,
	scheme *runtime.Scheme,
	mode string,
	sourceManagerCh chan diagnosisv1.Abnormal,
	informationManagerCh chan diagnosisv1.Abnormal,
	diagnoserChainCh chan diagnosisv1.Abnormal,
	recovererChainCh chan diagnosisv1.Abnormal,
) *AbnormalReconciler {
	return &AbnormalReconciler{
		Client:               cli,
		Log:                  log,
		Scheme:               scheme,
		mode:                 mode,
		sourceManagerCh:      sourceManagerCh,
		informationManagerCh: informationManagerCh,
		diagnoserChainCh:     diagnoserChainCh,
		recovererChainCh:     recovererChainCh,
	}
}

// +kubebuilder:rbac:groups=diagnosis.netease.com,resources=abnormals,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=diagnosis.netease.com,resources=abnormals/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete

// Reconcile synchronizes a Abnormal object according to the phase.
func (r *AbnormalReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("abnormal", req.NamespacedName)

	log.Info("reconciling Abnormal")

	var abnormal diagnosisv1.Abnormal
	if err := r.Get(ctx, req.NamespacedName, &abnormal); err != nil {
		log.Error(err, "unable to fetch Abnormal")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// The master will process an abnormal which has not been accept yet, while the agent will process
	// an abnormal in InformationCollecting, Diagnosing, Recovering phases.
	if r.mode == "master" {
		switch abnormal.Status.Phase {
		case "":
			// Set abnormal NodeName if NodeName is empty and PodReference is not nil.
			if abnormal.Spec.NodeName == "" {
				if abnormal.Spec.PodReference == nil {
					log.Error(fmt.Errorf("nodeName and podReference are both empty"), "ignoring invalid Abnormal")
					return ctrl.Result{}, nil
				}

				var pod corev1.Pod
				if err := r.Get(ctx, client.ObjectKey{
					Name:      abnormal.Spec.PodReference.Name,
					Namespace: abnormal.Spec.PodReference.Namespace,
				}, &pod); err != nil {
					log.Error(err, "unable to fetch Pod")
					return ctrl.Result{}, client.IgnoreNotFound(err)
				}

				abnormal.Spec.NodeName = pod.Spec.NodeName
				if err := r.Update(ctx, &abnormal); err != nil {
					log.Error(err, "unable to update Abnormal")
					return ctrl.Result{}, client.IgnoreNotFound(err)
				}
			}

			err := util.QueueAbnormal(ctx, r.sourceManagerCh, abnormal)
			if err != nil {
				log.Error(err, "failed to send abnormal to source manager queue")
			}
		}
	} else if r.mode == "agent" {
		switch abnormal.Status.Phase {
		case diagnosisv1.InformationCollecting:
			_, condition := util.GetAbnormalCondition(&abnormal.Status, diagnosisv1.InformationCollected)
			if condition != nil {
				log.Info("ignoring Abnormal in phase InformationCollecting with condition InformationCollected")
			} else {
				err := util.QueueAbnormal(ctx, r.informationManagerCh, abnormal)
				if err != nil {
					log.Error(err, "failed to send abnormal to information manager queue")
				}
			}
		case diagnosisv1.AbnormalDiagnosing:
			err := util.QueueAbnormal(ctx, r.diagnoserChainCh, abnormal)
			if err != nil {
				log.Error(err, "failed to send abnormal to diagnoser chain queue")
			}
		case diagnosisv1.AbnormalRecovering:
			err := util.QueueAbnormal(ctx, r.recovererChainCh, abnormal)
			if err != nil {
				log.Error(err, "failed to send abnormal to recoverer chain queue")
			}
		case diagnosisv1.AbnormalSucceeded:
			log.Info("ignoring Abnormal in phase Succeeded")
		case diagnosisv1.AbnormalFailed:
			log.Info("ignoring Abnormal in phase Failed")
		case diagnosisv1.AbnormalUnknown:
			log.Info("ignoring Abnormal in phase Unknown")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager setups AbnormalReconciler with the provided manager.
func (r *AbnormalReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&diagnosisv1.Abnormal{}).
		Complete(r)
}
