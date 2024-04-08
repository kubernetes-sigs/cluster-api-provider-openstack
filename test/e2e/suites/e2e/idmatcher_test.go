/*
Copyright 2024 The Kubernetes Authors.

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

package e2e

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

type idMatcher struct {
	expected interface{}
}

var (
	_ types.GomegaMatcher   = &idMatcher{}
	_ format.GomegaStringer = &idMatcher{}
)

func (m *idMatcher) Match(actual interface{}) (bool, error) {
	v := reflect.ValueOf(m.expected)
	id := v.FieldByName("ID").String()

	matcher := &gstruct.FieldsMatcher{
		Fields: gstruct.Fields{
			"ID": gomega.Equal(id),
		},
		IgnoreExtras: true,
	}

	return matcher.Match(actual)
}

func (m *idMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected:\n%s\nto have the same ID as:\n%s", format.Object(actual, 1), format.Object(m.expected, 1))
}

func (m *idMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected:\n%s\nnot to have the same ID as:\n%s", format.Object(actual, 1), format.Object(m.expected, 1))
}

func (m *idMatcher) GomegaString() string {
	return fmt.Sprintf("ID match for:\n%s", format.Object(m.expected, 1))
}

func IDOf(expected interface{}) types.GomegaMatcher {
	return &idMatcher{expected: expected}
}

func ConsistOfIDs[T any](expected []T) types.GomegaMatcher {
	matchers := make([]types.GomegaMatcher, len(expected))
	for i := range expected {
		matchers[i] = IDOf(expected[i])
	}

	return gomega.ConsistOf(matchers)
}
