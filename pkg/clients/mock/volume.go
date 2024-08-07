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
// Code generated by MockGen. DO NOT EDIT.
// Source: sigs.k8s.io/cluster-api-provider-openstack/pkg/clients (interfaces: VolumeClient)
//
// Generated by this command:
//
//	mockgen -package mock -destination=volume.go sigs.k8s.io/cluster-api-provider-openstack/pkg/clients VolumeClient
//

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	volumes "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	gomock "go.uber.org/mock/gomock"
)

// MockVolumeClient is a mock of VolumeClient interface.
type MockVolumeClient struct {
	ctrl     *gomock.Controller
	recorder *MockVolumeClientMockRecorder
}

// MockVolumeClientMockRecorder is the mock recorder for MockVolumeClient.
type MockVolumeClientMockRecorder struct {
	mock *MockVolumeClient
}

// NewMockVolumeClient creates a new mock instance.
func NewMockVolumeClient(ctrl *gomock.Controller) *MockVolumeClient {
	mock := &MockVolumeClient{ctrl: ctrl}
	mock.recorder = &MockVolumeClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockVolumeClient) EXPECT() *MockVolumeClientMockRecorder {
	return m.recorder
}

// CreateVolume mocks base method.
func (m *MockVolumeClient) CreateVolume(arg0 volumes.CreateOptsBuilder) (*volumes.Volume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVolume", arg0)
	ret0, _ := ret[0].(*volumes.Volume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVolume indicates an expected call of CreateVolume.
func (mr *MockVolumeClientMockRecorder) CreateVolume(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVolume", reflect.TypeOf((*MockVolumeClient)(nil).CreateVolume), arg0)
}

// DeleteVolume mocks base method.
func (m *MockVolumeClient) DeleteVolume(arg0 string, arg1 volumes.DeleteOptsBuilder) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVolume", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVolume indicates an expected call of DeleteVolume.
func (mr *MockVolumeClientMockRecorder) DeleteVolume(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVolume", reflect.TypeOf((*MockVolumeClient)(nil).DeleteVolume), arg0, arg1)
}

// GetVolume mocks base method.
func (m *MockVolumeClient) GetVolume(arg0 string) (*volumes.Volume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVolume", arg0)
	ret0, _ := ret[0].(*volumes.Volume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVolume indicates an expected call of GetVolume.
func (mr *MockVolumeClientMockRecorder) GetVolume(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVolume", reflect.TypeOf((*MockVolumeClient)(nil).GetVolume), arg0)
}

// ListVolumes mocks base method.
func (m *MockVolumeClient) ListVolumes(arg0 volumes.ListOptsBuilder) ([]volumes.Volume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListVolumes", arg0)
	ret0, _ := ret[0].([]volumes.Volume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListVolumes indicates an expected call of ListVolumes.
func (mr *MockVolumeClientMockRecorder) ListVolumes(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListVolumes", reflect.TypeOf((*MockVolumeClient)(nil).ListVolumes), arg0)
}
