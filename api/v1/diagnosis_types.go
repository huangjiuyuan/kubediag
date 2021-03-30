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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DiagnosisPending means that the diagnosis has been accepted by the system, but no operation has been started.
	DiagnosisPending DiagnosisPhase = "Pending"
	// DiagnosisRunning means the diagnosis has been bound to a node and one of the operations have been started.
	// At least one operation is still running.
	DiagnosisRunning DiagnosisPhase = "Running"
	// DiagnosisSucceeded means that all operations in some path have voluntarily terminated with a response code
	// of 200, and the system is not going to execute rest operations.
	DiagnosisSucceeded DiagnosisPhase = "Succeeded"
	// DiagnosisFailed means that all paths in the graph have terminated, and at least one operation in each path
	// terminated in a failure.
	DiagnosisFailed DiagnosisPhase = "Failed"
	// DiagnosisUnknown means that for some reason the state of the diagnosis could not be obtained, typically due
	// to an error in communicating with the host of the diagnosis.
	DiagnosisUnknown DiagnosisPhase = "Unknown"

	// DiagnosisAccepted means that the diagnosis has been accepted by kube diagnoser agent.
	DiagnosisAccepted DiagnosisConditionType = "Accepted"
	// DiagnosisComplete means the diagnosis has completed its execution.
	DiagnosisComplete DiagnosisConditionType = "Complete"
)

// DiagnosisSpec defines the desired state of Diagnosis.
type DiagnosisSpec struct {
	// OperationSet is the name of operation set which represents diagnosis pipeline to be executed.
	OperationSet string `json:"operationSet"`
	// One of NodeName and PodReference must be specified.
	// NodeName is a specific node which the diagnosis is on.
	// +optional
	NodeName string `json:"nodeName,omitempty"`
	// PodReference contains details of the target pod.
	// +optional
	PodReference *PodReference `json:"podReference,omitempty"`
	// Parameters is a set of the parameters to be passed to opreations.
	// A key is the node id in referenced operation set, while the value contains parameters for executing operation.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// PodReference contains details of the target pod.
type PodReference struct {
	NamespacedName `json:",inline"`
	// Container specifies name of the target container.
	// +optional
	Container string `json:"container,omitempty"`
}

// NamespacedName represents a kubernetes api resource.
type NamespacedName struct {
	// Namespace specifies the namespace of a kubernetes api resource.
	Namespace string `json:"namespace"`
	// Name specifies the name of a kubernetes api resource.
	Name string `json:"name"`
}

// DiagnosisStatus defines the observed state of Diagnosis.
type DiagnosisStatus struct {
	// Phase is a simple, high-level summary of where the diagnosis is in its lifecycle.
	// The conditions array, the reason and message fields contain more detail about the
	// pod's status.
	// There are five possible phase values:
	//
	// DiagnosisPending: The diagnosis has been accepted by the system, but no operation has been started.
	// DiagnosisRunning: The diagnosis has been bound to a node and one of the operations have been started.
	// At least one operation is still running.
	// DiagnosisSucceeded: All operations in some path have voluntarily terminated with a response code
	// of 200, and the system is not going to execute rest operations.
	// DiagnosisFailed: All paths in the graph have terminated, and at least one operation in each path
	// terminated in a failure.
	// DiagnosisUnknown: For some reason the state of the diagnosis could not be obtained, typically due
	// to an error in communicating with the host of the diagnosis.
	// +optional
	Phase DiagnosisPhase `json:"phase,omitempty"`
	// Conditions contains current service state of diagnosis.
	// +optional
	Conditions []DiagnosisCondition `json:"conditions,omitempty"`
	// StartTime is RFC 3339 date and time at which the object was acknowledged by the system.
	// +optional
	StartTime metav1.Time `json:"startTime,omitempty"`
	// FailedPaths contains all failed paths in diagnosis pipeline.
	// The last node in the path is the one which fails to execute operation.
	// +optional
	FailedPaths []Path `json:"failedPath,omitempty"`
	// SucceededPath is the succeeded paths in diagnosis pipeline.
	// +optional
	SucceededPath Path `json:"succeededPath,omitempty"`
	// OperationResults contains results of operations.
	// A key is the node id in referenced operation set, while the value usually contains detailed information of
	// operation result.
	// +optional
	OperationResults map[string]OperationResult `json:"operationResults,omitempty"`
	// Checkpoint is the checkpoint for resuming unfinished diagnosis.
	// +optional
	Checkpoint *Checkpoint `json:"checkpoint,omitempty"`
}

// DiagnosisCondition contains details for the current condition of this diagnosis.
type DiagnosisCondition struct {
	// Type is the type of the condition.
	Type DiagnosisConditionType `json:"type"`
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// LastTransitionTime specifies last time the condition transitioned from one status
	// to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Reason is a unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Message is a human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// OperationResult contains the result of an operation.
type OperationResult struct {
	// Operation is the name of operation.
	Operation string `json:"operation"`
	// Result is is a blob of information containing result of an executed operation.
	// +optional
	Result *string `json:"result,omitempty"`
}

// Checkpoint is the checkpoint for resuming unfinished diagnosis.
type Checkpoint struct {
	// PathIndex is the index of current path in operation set status.
	PathIndex int `json:"pathIndex"`
	// NodeIndex is the index of current node in path.
	NodeIndex int `json:"nodeIndex"`
}

// DiagnosisPhase is a label for the condition of a diagnosis at the current time.
type DiagnosisPhase string

// DiagnosisConditionType is a valid value for DiagnosisCondition.Type.
type DiagnosisConditionType string

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Diagnosis is the Schema for the diagnoses API.
type Diagnosis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiagnosisSpec   `json:"spec,omitempty"`
	Status DiagnosisStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DiagnosisList contains a list of Diagnosis.
type DiagnosisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Diagnosis `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Diagnosis{}, &DiagnosisList{})
}