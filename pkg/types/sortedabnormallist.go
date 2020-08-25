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

package types

import (
	diagnosisv1 "netease.com/k8s/kube-diagnoser/api/v1"
)

// SortedAbnormalListByStartTime contains sorted abnormals by StartTime in ascending order.
// It satisfies sort.Interface by implemeting the following methods:
//
// Len() int
// Less(i, j int) bool
// Swap(i, j int)
type SortedAbnormalListByStartTime []diagnosisv1.Abnormal

// Len is the number of elements in SortedAbnormalListByStartTime.
func (al SortedAbnormalListByStartTime) Len() int {
	return len(al)
}

// Less reports whether the element with index i should sort before the element with index j.
func (al SortedAbnormalListByStartTime) Less(i, j int) bool {
	if i > len(al) || j > len(al) {
		return false
	}

	return al[i].Status.StartTime.Before(&al[j].Status.StartTime)
}

// Swap swaps the elements with indexes i and j.
func (al SortedAbnormalListByStartTime) Swap(i, j int) {
	al[i], al[j] = al[j], al[i]
}