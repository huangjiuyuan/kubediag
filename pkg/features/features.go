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

package features

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"k8s.io/component-base/featuregate"
)

const (
	// Alertmanager can handle valid post alerts requests.
	//
	// Mode: master
	// Owner: @huangjiuyuan
	// Alpha: 0.1.5
	Alertmanager featuregate.Feature = "Alertmanager"
	// Eventer generates abnormals from kubernetes events.
	//
	// Mode: master
	// Owner: @huangjiuyuan
	// Alpha: 0.1.5
	Eventer featuregate.Feature = "Eventer"
	// ClusterHealthEvaluator evaluates the health status of kubernetes cluster.
	//
	// Mode: master
	// Owner: @huangjiuyuan
	// Alpha: 0.1.5
	ClusterHealthEvaluator featuregate.Feature = "ClusterHealthEvaluator"

	// PodCollector manages information of all pods on the node.
	//
	// Mode: agent
	// Owner: @huangjiuyuan
	// Alpha: 0.1.5
	PodCollector featuregate.Feature = "PodCollector"
	// ContainerCollector manages information of all containers on the node.
	//
	// Mode: agent
	// Owner: @huangjiuyuan
	// Alpha: 0.1.5
	ContainerCollector featuregate.Feature = "ContainerCollector"
	// ProcessCollector manages information of all processes on the node.
	//
	// Mode: agent
	// Owner: @huangjiuyuan
	// Alpha: 0.1.5
	ProcessCollector featuregate.Feature = "ProcessCollector"
	// FileStatusCollector manages information that finding status of files.
	//
	// Mode: agent
	// Owner: @huangjiuyuan
	// Alpha: 0.1.5
	FileStatusCollector featuregate.Feature = "FileStatusCollector"
	// SystemdCollector manages information of systemd on the node.
	//
	// Mode: agent
	// Owner: @huangjiuyuan
	// Alpha: 0.1.5
	SystemdCollector featuregate.Feature = "SystemdCollector"
	// PodDiskUsageDiagnoser manages diagnosis that finding disk usage of pods.
	//
	// Mode: agent
	// Owner: @huangjiuyuan
	// Alpha: 0.1.5
	PodDiskUsageDiagnoser featuregate.Feature = "PodDiskUsageDiagnoser"
	// TerminatingPodDiagnoser manages diagnosis on terminating pods.
	//
	// Mode: agent
	// Owner: @huangjiuyuan
	// Alpha: 0.1.5
	TerminatingPodDiagnoser featuregate.Feature = "TerminatingPodDiagnoser"
	// SignalRecoverer manages recovery that sending signal to processes.
	//
	// Mode: agent
	// Owner: @huangjiuyuan
	// Alpha: 0.1.5
	SignalRecoverer featuregate.Feature = "SignalRecoverer"
)

var defaultKubeDiagnoserFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	Alertmanager:            {Default: true, PreRelease: featuregate.Alpha},
	Eventer:                 {Default: false, PreRelease: featuregate.Alpha},
	ClusterHealthEvaluator:  {Default: true, PreRelease: featuregate.Alpha},
	PodCollector:            {Default: true, PreRelease: featuregate.Alpha},
	ContainerCollector:      {Default: true, PreRelease: featuregate.Alpha},
	ProcessCollector:        {Default: true, PreRelease: featuregate.Alpha},
	FileStatusCollector:     {Default: true, PreRelease: featuregate.Alpha},
	SystemdCollector:        {Default: true, PreRelease: featuregate.Alpha},
	PodDiskUsageDiagnoser:   {Default: true, PreRelease: featuregate.Alpha},
	TerminatingPodDiagnoser: {Default: true, PreRelease: featuregate.Alpha},
	SignalRecoverer:         {Default: true, PreRelease: featuregate.Alpha},
}

// KubeDiagnoserFeatureGate indicates whether a given feature is enabled or not and stores flag gates for known features.
type KubeDiagnoserFeatureGate interface {
	// Enabled returns true if the key is enabled.
	Enabled(featuregate.Feature) bool
	// KnownFeatures returns a slice of strings describing the known features.
	KnownFeatures() []string
	// SetFromMap stores flag gates for known features from a map[string]bool or returns an error.
	SetFromMap(map[string]bool) error
}

// kubeDiagnoserFeatureGate manages features of kube diagnoser.
type kubeDiagnoserFeatureGate struct {
	// lock guards writes to known and enabled.
	lock sync.Mutex
	// known holds a map[featuregate.Feature]featuregate.FeatureSpec.
	known *atomic.Value
	// enabled holds a map[featuregate.Feature]bool.
	enabled *atomic.Value
}

// NewFeatureGate creates a new KubeDiagnoserFeatureGate.
func NewFeatureGate() KubeDiagnoserFeatureGate {
	// Set default known features.
	knownMap := make(map[featuregate.Feature]featuregate.FeatureSpec)
	for key, value := range defaultKubeDiagnoserFeatureGates {
		knownMap[key] = value
	}
	known := new(atomic.Value)
	known.Store(knownMap)

	// Set default enabled features.
	enabledMap := make(map[featuregate.Feature]bool)
	for key, value := range defaultKubeDiagnoserFeatureGates {
		enabledMap[key] = value.Default
	}
	enabled := new(atomic.Value)
	enabled.Store(enabledMap)

	return &kubeDiagnoserFeatureGate{
		known:   known,
		enabled: enabled,
	}
}

// Enabled returns true if the key is enabled.
func (kf *kubeDiagnoserFeatureGate) Enabled(key featuregate.Feature) bool {
	if value, ok := kf.enabled.Load().(map[featuregate.Feature]bool)[key]; ok {
		return value
	}
	if value, ok := kf.known.Load().(map[featuregate.Feature]featuregate.FeatureSpec)[key]; ok {
		return value.Default
	}

	return false
}

// KnownFeatures returns a slice of strings describing the known features.
// Deprecated and GA features are hidden from the list.
func (kf *kubeDiagnoserFeatureGate) KnownFeatures() []string {
	var known []string
	for key, value := range kf.known.Load().(map[featuregate.Feature]featuregate.FeatureSpec) {
		if value.PreRelease == featuregate.GA || value.PreRelease == featuregate.Deprecated {
			continue
		}
		known = append(known, fmt.Sprintf("%s=true|false (%s - default=%t)", key, value.PreRelease, value.Default))
	}
	sort.Strings(known)

	return known
}

// SetFromMap stores flag gates for known features from a map[string]bool or returns an error.
func (kf *kubeDiagnoserFeatureGate) SetFromMap(featureMap map[string]bool) error {
	kf.lock.Lock()
	defer kf.lock.Unlock()

	// Copy existing state.
	knownMap := make(map[featuregate.Feature]featuregate.FeatureSpec)
	for key, value := range kf.known.Load().(map[featuregate.Feature]featuregate.FeatureSpec) {
		knownMap[key] = value
	}
	enabledMap := make(map[featuregate.Feature]bool)
	for key, value := range kf.enabled.Load().(map[featuregate.Feature]bool) {
		enabledMap[key] = value
	}

	// Set flag gates for known features from a map[string]bool.
	for key, value := range featureMap {
		key := featuregate.Feature(key)
		featureSpec, ok := knownMap[key]
		if !ok {
			return fmt.Errorf("unrecognized feature gate: %s", key)
		}
		if featureSpec.LockToDefault && featureSpec.Default != value {
			return fmt.Errorf("cannot set feature gate %v to %v, feature is locked to %v", key, value, featureSpec.Default)
		}
		enabledMap[key] = value
	}

	// Persist changes.
	kf.known.Store(knownMap)
	kf.enabled.Store(enabledMap)

	return nil
}