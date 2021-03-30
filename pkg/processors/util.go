/*
Copyright 2021 The Kube Diagnoser Authors.

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

package processors

import (
	"encoding/json"
)

// DecodeOperationContext unmarshals json encoding into a map[string][]byte, which is the format of operation context.
func DecodeOperationContext(body []byte) (map[string][]byte, error) {
	data := make(map[string][]byte)
	err := json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}