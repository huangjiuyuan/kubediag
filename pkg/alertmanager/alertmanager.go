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

package alertmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/alertmanager/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	diagnosisv1 "netease.com/k8s/kube-diagnoser/api/v1"
	"netease.com/k8s/kube-diagnoser/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Alertmanager can handle valid post alerts requests.
type Alertmanager interface {
	// Handler handles http requests.
	Handler(http.ResponseWriter, *http.Request)
}

// alertmanager manages prometheus alerts received by kube diagnoser.
type alertmanager struct {
	// Context carries values across API boundaries.
	context.Context
	// Logger represents the ability to log messages.
	logr.Logger

	// client knows how to perform CRUD operations on Kubernetes objects.
	client client.Client
	// repeatInterval specifies how long to wait before sending a notification again if it has already
	// been sent successfully for an alert.
	repeatInterval time.Duration
	// firingAlertSet contains all alerts fired by alertmanager.
	firingAlertSet map[uint64]time.Time
}

// NewAlertmanager creates a new Alertmanager.
func NewAlertmanager(
	ctx context.Context,
	logger logr.Logger,
	cli client.Client,
	repeatInterval time.Duration,
) Alertmanager {
	firingAlertSet := make(map[uint64]time.Time)

	return &alertmanager{
		Context:        ctx,
		Logger:         logger,
		client:         cli,
		repeatInterval: repeatInterval,
		firingAlertSet: firingAlertSet,
	}
}

// Handler handles http requests for sending prometheus alerts.
func (am *alertmanager) Handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			am.Error(err, "unable to read request body")
			http.Error(w, fmt.Sprintf("unable to read request body: %v", err), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var alerts []*types.Alert
		err = json.Unmarshal(body, &alerts)
		if err != nil {
			am.Error(err, "failed to unmarshal request body")
			http.Error(w, fmt.Sprintf("failed to unmarshal request body: %v", err), http.StatusInternalServerError)
			return
		}

		for _, alert := range alerts {
			// Skip if the alert is resolved.
			if alert.Resolved() {
				continue
			}

			// Skip alerts if the repeat interval has not been passed.
			fingerprint := alert.Fingerprint()
			now := time.Now()
			lastFiring, ok := am.firingAlertSet[uint64(fingerprint)]
			if ok && lastFiring.After(now.Add(-am.repeatInterval)) {
				continue
			}

			am.Info("starting to handling prometheus alert", "alert", alert)

			// Create abnormal according to the prometheus alert.
			name := fmt.Sprintf("%s.%s.%s", util.PrometheusAlertGeneratedAbnormalPrefix, strings.ToLower(alert.Name()), alert.Fingerprint().String()[:7])
			namespace := util.DefautlNamespace
			abnormal := diagnosisv1.Abnormal{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: diagnosisv1.AbnormalSpec{
					Source:   diagnosisv1.CustomSource,
					NodeName: "my-node",
				},
			}
			if err := am.client.Create(am, &abnormal); err != nil {
				if !apierrors.IsAlreadyExists(err) {
					am.Error(err, "unable to create abnormal")
					http.Error(w, fmt.Sprintf("unable to create abnormal: %v", err), http.StatusInternalServerError)
					return
				}
			}

			// Update alert fired time if the abnormal is created successfully.
			am.firingAlertSet[uint64(fingerprint)] = now
		}

		w.Write([]byte("OK"))
	default:
		http.Error(w, fmt.Sprintf("method %s is not supported", r.Method), http.StatusMethodNotAllowed)
	}
}
