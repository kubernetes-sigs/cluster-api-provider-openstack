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
