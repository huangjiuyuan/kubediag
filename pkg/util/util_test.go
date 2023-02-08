/*
Copyright 2020 The KubeDiag Authors.

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

package util

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	diagnosisv1 "github.com/kubediag/kubediag/api/v1"
)

func TestUpdateDiagnosisCondition(t *testing.T) {
	diagnosisStatus := diagnosisv1.DiagnosisStatus{
		Conditions: []diagnosisv1.DiagnosisCondition{
			{
				Type:    diagnosisv1.OperationSetNotReady,
				Status:  corev1.ConditionTrue,
				Reason:  "successfully",
				Message: "sync diagnosis successfully",
			},
		},
	}

	tests := []struct {
		status    *diagnosisv1.DiagnosisStatus
		condition diagnosisv1.DiagnosisCondition
		expected  bool
		desc      string
	}{
		{
			status: &diagnosisStatus,
			condition: diagnosisv1.DiagnosisCondition{
				Type:    diagnosisv1.OperationSetNotReady,
				Status:  corev1.ConditionTrue,
				Reason:  "successfully",
				Message: "sync diagnosis successfully",
			},
			expected: false,
			desc:     "all equal, no update",
		},
		{
			status: &diagnosisStatus,
			condition: diagnosisv1.DiagnosisCondition{
				Type:    diagnosisv1.DiagnosisComplete,
				Status:  corev1.ConditionTrue,
				Reason:  "successfully",
				Message: "sync diagnosis successfully",
			},
			expected: true,
			desc:     "not equal Type, should get updated",
		},
		{
			status: &diagnosisStatus,
			condition: diagnosisv1.DiagnosisCondition{
				Type:    diagnosisv1.OperationSetNotReady,
				Status:  corev1.ConditionFalse,
				Reason:  "successfully",
				Message: "sync diagnosis successfully",
			},
			expected: true,
			desc:     "not equal Status, should get updated",
		},
	}

	for _, test := range tests {
		resultStatus := UpdateDiagnosisCondition(test.status, &test.condition)
		assert.Equal(t, test.expected, resultStatus, test.desc)
	}
}

func TestGetDiagnosisCondition(t *testing.T) {
	type expectedStruct struct {
		index     int
		condition *diagnosisv1.DiagnosisCondition
	}

	tests := []struct {
		status   *diagnosisv1.DiagnosisStatus
		condType diagnosisv1.DiagnosisConditionType
		expected expectedStruct
		desc     string
	}{
		{
			status:   nil,
			condType: diagnosisv1.OperationSetNotReady,
			expected: expectedStruct{-1, nil},
			desc:     "status nil, not found",
		},
		{
			status: &diagnosisv1.DiagnosisStatus{
				Conditions: nil,
			},
			condType: diagnosisv1.OperationSetNotReady,
			expected: expectedStruct{-1, nil},
			desc:     "conditions nil, not found",
		},
		{
			status: &diagnosisv1.DiagnosisStatus{
				Conditions: []diagnosisv1.DiagnosisCondition{
					{
						Type:    diagnosisv1.OperationSetNotReady,
						Status:  corev1.ConditionTrue,
						Reason:  "successfully",
						Message: "sync diagnosis successfully",
					},
				},
			},
			condType: diagnosisv1.OperationSetNotReady,
			expected: expectedStruct{0, &diagnosisv1.DiagnosisCondition{
				Type:    diagnosisv1.OperationSetNotReady,
				Status:  corev1.ConditionTrue,
				Reason:  "successfully",
				Message: "sync diagnosis successfully"},
			},
			desc: "condition found",
		},
	}

	for _, test := range tests {
		resultIndex, resultCond := GetDiagnosisCondition(test.status, test.condType)
		assert.Equal(t, test.expected.index, resultIndex, test.desc)
		assert.Equal(t, test.expected.condition, resultCond, test.desc)
	}
}

func TestGetPodUnhealthyReason(t *testing.T) {
	tests := []struct {
		pod      corev1.Pod
		expected string
		desc     string
	}{
		{
			pod:      corev1.Pod{},
			expected: "Unknown",
			desc:     "empty pod",
		},
		{
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Ready: true,
						},
					},
				},
			},
			expected: "Unknown",
			desc:     "ready pod",
		},
		{
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Ready: false,
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "reason1",
								},
							},
						},
					},
				},
			},
			expected: "reason1",
			desc:     "terminated pod",
		},
		{
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Ready: false,
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: "reason2",
								},
							},
						},
					},
				},
			},
			expected: "reason2",
			desc:     "waiting pod",
		},
		{
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Ready: false,
							LastTerminationState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "reason3",
								},
							},
						},
					},
				},
			},
			expected: "reason3",
			desc:     "pod with last termination",
		},
	}

	for _, test := range tests {
		reason := GetPodUnhealthyReason(test.pod)
		assert.Equal(t, test.expected, reason, test.desc)
	}
}

func TestUpdatePodUnhealthyReasonStatistics(t *testing.T) {
	type expectedStruct struct {
		updated               bool
		containerStateReasons map[string]int
	}

	tests := []struct {
		containerStateReasons map[string]int
		reason                string
		expected              expectedStruct
		desc                  string
	}{
		{
			containerStateReasons: map[string]int{},
			reason:                "",
			expected: expectedStruct{
				updated:               false,
				containerStateReasons: map[string]int{},
			},
			desc: "empty reason",
		},
		{
			containerStateReasons: map[string]int{},
			reason:                "reason1",
			expected: expectedStruct{
				updated:               true,
				containerStateReasons: map[string]int{"reason1": 1},
			},
			desc: "new reason added",
		},
		{
			containerStateReasons: map[string]int{"reason1": 1, "reason2": 1},
			reason:                "reason1",
			expected: expectedStruct{
				updated:               true,
				containerStateReasons: map[string]int{"reason1": 2, "reason2": 1},
			},
			desc: "reason updated",
		},
	}

	for _, test := range tests {
		updated := UpdatePodUnhealthyReasonStatistics(test.containerStateReasons, test.reason)
		assert.Equal(t, test.expected.updated, updated, test.desc)
		assert.Equal(t, test.expected.containerStateReasons, test.containerStateReasons, test.desc)
	}
}

func TestIsNodeReady(t *testing.T) {
	tests := []struct {
		node     corev1.Node
		expected bool
		desc     string
	}{
		{
			node:     corev1.Node{},
			expected: false,
			desc:     "node status is empty",
		},
		{
			node: corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: true,
			desc:     "node is ready",
		},
		{
			node: corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			expected: false,
			desc:     "node is not ready",
		},
		{
			node: corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeNetworkUnavailable,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: false,
			desc:     "node is network unavailable",
		},
	}

	for _, test := range tests {
		ready := IsNodeReady(test.node)
		assert.Equal(t, test.expected, ready, test.desc)
	}
}

func TestGetNodeUnhealthyConditionType(t *testing.T) {
	tests := []struct {
		node     corev1.Node
		expected corev1.NodeConditionType
		desc     string
	}{
		{
			node:     corev1.Node{},
			expected: "Unknown",
			desc:     "node status is empty",
		},
		{
			node: corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   "type1",
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: "type1",
			desc:     "unhealthy node",
		},
		{
			node: corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: "Unknown",
			desc:     "healthy node",
		},
	}

	for _, test := range tests {
		conditionType := GetNodeUnhealthyConditionType(test.node)
		assert.Equal(t, test.expected, conditionType, test.desc)
	}
}

func TestFormatURL(t *testing.T) {
	tests := []struct {
		scheme   string
		host     string
		port     string
		path     string
		expected *url.URL
		desc     string
	}{
		{
			scheme: "http",
			host:   "127.0.0.1",
			port:   "8080",
			path:   "/test",
			expected: &url.URL{
				Scheme: "http",
				Host:   "127.0.0.1:8080",
				Path:   "/test",
			},
			desc: "regular url",
		},
	}

	for _, test := range tests {
		resultURL := FormatURL(test.scheme, test.host, test.port, test.path)
		assert.Equal(t, test.expected, resultURL, test.desc)
	}
}

func TestIsDiagnosisNodeNameMatched(t *testing.T) {
	tests := []struct {
		task     diagnosisv1.Task
		node     string
		expected bool
		desc     string
	}{
		{
			task: diagnosisv1.Task{
				Spec: diagnosisv1.TaskSpec{
					NodeName: "",
				},
			},
			node:     "node1",
			expected: false,
			desc:     "empty node name",
		},
		{
			task: diagnosisv1.Task{
				Spec: diagnosisv1.TaskSpec{
					NodeName: "node1",
				},
			},
			node:     "node1",
			expected: true,
			desc:     "node name matched",
		},
	}

	for _, test := range tests {
		matched := IsTaskNodeNameMatched(test.task, test.node)
		assert.Equal(t, test.expected, matched, test.desc)
	}
}

func TestRetrievePodsOnNode(t *testing.T) {
	tests := []struct {
		pods     []corev1.Pod
		nodeName string
		expected []corev1.Pod
		desc     string
	}{
		{
			pods:     []corev1.Pod{},
			nodeName: "node1",
			expected: []corev1.Pod{},
			desc:     "empty slice",
		},
		{
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod1",
					},
					Spec: corev1.PodSpec{
						NodeName: "node1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod2",
					},
					Spec: corev1.PodSpec{
						NodeName: "node2",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod3",
					},
					Spec: corev1.PodSpec{
						NodeName: "node1",
					},
				},
			},
			nodeName: "node1",
			expected: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod1",
					},
					Spec: corev1.PodSpec{
						NodeName: "node1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod3",
					},
					Spec: corev1.PodSpec{
						NodeName: "node1",
					},
				},
			},
			desc: "pods not on provided node removed",
		},
	}

	for _, test := range tests {
		resultPods := RetrievePodsOnNode(test.pods, test.nodeName)
		assert.Equal(t, test.expected, resultPods, test.desc)
	}
}

func TestRetrieveDiagnosesOnNode(t *testing.T) {
	tests := []struct {
		tasks    []diagnosisv1.Task
		nodeName string
		expected []diagnosisv1.Task
		desc     string
	}{
		{
			tasks:    []diagnosisv1.Task{},
			nodeName: "node1",
			expected: []diagnosisv1.Task{},
			desc:     "empty slice",
		},
		{
			tasks: []diagnosisv1.Task{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "diagnosis1",
					},
					Spec: diagnosisv1.TaskSpec{
						NodeName: "node1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "diagnosis2",
					},
					Spec: diagnosisv1.TaskSpec{
						NodeName: "node2",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "diagnosis3",
					},
					Spec: diagnosisv1.TaskSpec{
						NodeName: "node1",
					},
				},
			},
			nodeName: "node1",
			expected: []diagnosisv1.Task{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "diagnosis1",
					},
					Spec: diagnosisv1.TaskSpec{
						NodeName: "node1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "diagnosis3",
					},
					Spec: diagnosisv1.TaskSpec{
						NodeName: "node1",
					},
				},
			},
			desc: "tasks not on provided node removed",
		},
	}

	for _, test := range tests {
		resultDiagnoses := RetrieveTasksOnNode(test.tasks, test.nodeName)
		assert.Equal(t, test.expected, resultDiagnoses, test.desc)
	}
}

func TestScanLastNonEmptyLine(t *testing.T) {
	type expectedStruct struct {
		advance int
		token   []byte
	}

	tests := []struct {
		data     []byte
		eof      bool
		expected expectedStruct
		desc     string
	}{
		{
			data: []byte("test"),
			eof:  false,
			expected: expectedStruct{
				advance: 0,
				token:   nil,
			},
			desc: "request more",
		},
		{
			data: []byte("\n"),
			eof:  false,
			expected: expectedStruct{
				advance: 1,
				token:   nil,
			},
			desc: "only eol",
		},
		{
			data: []byte("\n\r\r\r"),
			eof:  false,
			expected: expectedStruct{
				advance: 4,
				token:   nil,
			},
			desc: "only eol multi",
		},
		{
			data: []byte("\n\rworld"),
			eof:  false,
			expected: expectedStruct{
				advance: 2,
				token:   nil,
			},
			desc: "only eol not eof",
		},
		{
			data: []byte("hello"),
			eof:  true,
			expected: expectedStruct{
				advance: 5,
				token:   []byte("hello"),
			},
			desc: "eof simple",
		},
		{
			data: []byte("hello\n"),
			eof:  true,
			expected: expectedStruct{
				advance: 6,
				token:   []byte("hello"),
			},
			desc: "eof trailing eol 1",
		},
		{
			data: []byte("hello\r\n"),
			eof:  true,
			expected: expectedStruct{
				advance: 7,
				token:   []byte("hello"),
			},
			desc: "eof trailing eol 2",
		},
		{
			data: []byte("hello\r\n\r\n"),
			eof:  true,
			expected: expectedStruct{
				advance: 9,
				token:   []byte("hello"),
			},
			desc: "eof trailing eol 4",
		},
		{
			data: []byte("hello\nworld"),
			eof:  false,
			expected: expectedStruct{
				advance: 6,
				token:   []byte("hello"),
			},
			desc: "one line",
		},
		{
			data: []byte("one\ntwotwo\nthreethreethree\rfourfourfourfour"),
			eof:  false,
			expected: expectedStruct{
				advance: 27,
				token:   []byte("threethreethree"),
			},
			desc: "many lines",
		},
		{
			data: []byte("one\ntwotwo\nthreethreethree\rfourfourfourfour"),
			eof:  true,
			expected: expectedStruct{
				advance: 43,
				token:   []byte("fourfourfourfour"),
			},
			desc: "many lines eof",
		},
		{
			data: []byte("one\ntwotwo\nthreethreethree\rfourfourfourfour\n"),
			eof:  false,
			expected: expectedStruct{
				advance: 44,
				token:   []byte("fourfourfourfour"),
			},
			desc: "many lines eol",
		},
		{
			data: []byte("one\ntwotwo\nthreethreethree\rfourfourfourfour\r\n"),
			eof:  false,
			expected: expectedStruct{
				advance: 45,
				token:   []byte("fourfourfourfour"),
			},
			desc: "many lines multi eol",
		},
	}

	for _, test := range tests {
		advance, token, err := ScanLastNonEmptyLine(test.data, test.eof)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, test.expected.advance, advance, test.desc)
		assert.Equal(t, test.expected.token, token, test.desc)
	}
}

func TestRemoveDuplicateStrings(t *testing.T) {
	tests := []struct {
		slice    []string
		expected []string
		desc     string
	}{
		{
			slice:    nil,
			expected: nil,
			desc:     "nil slice",
		},
		{
			slice:    []string{},
			expected: []string{},
			desc:     "empty slice",
		},
		{
			slice:    []string{"a", "b", "c", "d"},
			expected: []string{"a", "b", "c", "d"},
			desc:     "slice without deplicated occurances",
		},
		{
			slice:    []string{"a", "b", "c", "d", "c", "b"},
			expected: []string{"a", "b", "c", "d"},
			desc:     "duplicated strings",
		},
	}

	for _, test := range tests {
		dedup := RemoveDuplicateStrings(test.slice)
		assert.Equal(t, test.expected, dedup, test.desc)
	}
}
