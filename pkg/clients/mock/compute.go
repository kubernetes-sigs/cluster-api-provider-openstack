/*
Copyright 2023 The Kubernetes Authors.

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

// Code generated by MockGen. DO NOT EDIT.
// Source: sigs.k8s.io/cluster-api-provider-openstack/pkg/clients (interfaces: ComputeClient)

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	attachinterfaces "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/extensions/attachinterfaces"
	availabilityzones "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/extensions/availabilityzones"
	servergroups "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/extensions/servergroups"
	flavors "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	servers "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	clients "sigs.k8s.io/cluster-api-provider-openstack/pkg/clients"
)

// MockComputeClient is a mock of ComputeClient interface.
type MockComputeClient struct {
	ctrl     *gomock.Controller
	recorder *MockComputeClientMockRecorder
}

// MockComputeClientMockRecorder is the mock recorder for MockComputeClient.
type MockComputeClientMockRecorder struct {
	mock *MockComputeClient
}

// NewMockComputeClient creates a new mock instance.
func NewMockComputeClient(ctrl *gomock.Controller) *MockComputeClient {
	mock := &MockComputeClient{ctrl: ctrl}
	mock.recorder = &MockComputeClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockComputeClient) EXPECT() *MockComputeClientMockRecorder {
	return m.recorder
}

// CreateServer mocks base method.
func (m *MockComputeClient) CreateServer(arg0 servers.CreateOptsBuilder) (*clients.ServerExt, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateServer", arg0)
	ret0, _ := ret[0].(*clients.ServerExt)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateServer indicates an expected call of CreateServer.
func (mr *MockComputeClientMockRecorder) CreateServer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateServer", reflect.TypeOf((*MockComputeClient)(nil).CreateServer), arg0)
}

// DeleteAttachedInterface mocks base method.
func (m *MockComputeClient) DeleteAttachedInterface(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAttachedInterface", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAttachedInterface indicates an expected call of DeleteAttachedInterface.
func (mr *MockComputeClientMockRecorder) DeleteAttachedInterface(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAttachedInterface", reflect.TypeOf((*MockComputeClient)(nil).DeleteAttachedInterface), arg0, arg1)
}

// DeleteServer mocks base method.
func (m *MockComputeClient) DeleteServer(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteServer", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteServer indicates an expected call of DeleteServer.
func (mr *MockComputeClientMockRecorder) DeleteServer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteServer", reflect.TypeOf((*MockComputeClient)(nil).DeleteServer), arg0)
}

// GetFlavorFromName mocks base method.
func (m *MockComputeClient) GetFlavorFromName(arg0 string) (*flavors.Flavor, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFlavorFromName", arg0)
	ret0, _ := ret[0].(*flavors.Flavor)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFlavorFromName indicates an expected call of GetFlavorFromName.
func (mr *MockComputeClientMockRecorder) GetFlavorFromName(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFlavorFromName", reflect.TypeOf((*MockComputeClient)(nil).GetFlavorFromName), arg0)
}

// GetServer mocks base method.
func (m *MockComputeClient) GetServer(arg0 string) (*clients.ServerExt, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServer", arg0)
	ret0, _ := ret[0].(*clients.ServerExt)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetServer indicates an expected call of GetServer.
func (mr *MockComputeClientMockRecorder) GetServer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServer", reflect.TypeOf((*MockComputeClient)(nil).GetServer), arg0)
}

// ListAttachedInterfaces mocks base method.
func (m *MockComputeClient) ListAttachedInterfaces(arg0 string) ([]attachinterfaces.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAttachedInterfaces", arg0)
	ret0, _ := ret[0].([]attachinterfaces.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAttachedInterfaces indicates an expected call of ListAttachedInterfaces.
func (mr *MockComputeClientMockRecorder) ListAttachedInterfaces(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAttachedInterfaces", reflect.TypeOf((*MockComputeClient)(nil).ListAttachedInterfaces), arg0)
}

// ListAvailabilityZones mocks base method.
func (m *MockComputeClient) ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAvailabilityZones")
	ret0, _ := ret[0].([]availabilityzones.AvailabilityZone)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAvailabilityZones indicates an expected call of ListAvailabilityZones.
func (mr *MockComputeClientMockRecorder) ListAvailabilityZones() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAvailabilityZones", reflect.TypeOf((*MockComputeClient)(nil).ListAvailabilityZones))
}

// ListServerGroups mocks base method.
func (m *MockComputeClient) ListServerGroups() ([]servergroups.ServerGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListServerGroups")
	ret0, _ := ret[0].([]servergroups.ServerGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListServerGroups indicates an expected call of ListServerGroups.
func (mr *MockComputeClientMockRecorder) ListServerGroups() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListServerGroups", reflect.TypeOf((*MockComputeClient)(nil).ListServerGroups))
}

// ListServers mocks base method.
func (m *MockComputeClient) ListServers(arg0 servers.ListOptsBuilder) ([]clients.ServerExt, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListServers", arg0)
	ret0, _ := ret[0].([]clients.ServerExt)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListServers indicates an expected call of ListServers.
func (mr *MockComputeClientMockRecorder) ListServers(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListServers", reflect.TypeOf((*MockComputeClient)(nil).ListServers), arg0)
}
