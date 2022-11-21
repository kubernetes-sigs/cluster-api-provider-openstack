/*
Copyright 2022 The Kubernetes Authors.

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

package simulator

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

func getNameFromQuery(query string) (string, error) {
	queryValues, err := getValuesFromQuery(query)
	if err != nil {
		return "", fmt.Errorf("parsing query: %w", err)
	}

	name, ok := queryValues["name"]
	if !ok {
		return "", fmt.Errorf("simulator only supports lookup by name. name not provided")
	}
	if len(queryValues) != 1 {
		keys := []string{}
		for k := range queryValues {
			keys = append(keys, k)
		}
		return "", fmt.Errorf("simulator only supports lookup by name. Additional parameters given: %s", strings.Join(keys, " "))
	}

	return name, nil
}

func getValuesFromQuery(query string) (map[string]string, error) {
	queryParsed, err := url.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("parsing query: %w", err)
	}

	ret := make(map[string]string)

	queryValues := queryParsed.Query()
	for k, v := range queryValues {
		switch {
		case len(v) == 1:
			ret[k] = v[0]
		case len(v) > 1:
			return nil, fmt.Errorf("multiple query values for %s", k)
		}
	}

	return ret, nil
}

func generateUUID() string {
	uuid, err := uuid.NewRandom()
	if err != nil {
		// Don't know why this would fail, but if it did it would be a simulator issue
		panic(err)
	}

	return uuid.String()
}
