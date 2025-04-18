// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/Azure/ARO-RP/pkg/hive (interfaces: ClusterManager,SyncSetManager)
//
// Generated by this command:
//
//	mockgen -destination=../util/mocks/hive/hive.go github.com/Azure/ARO-RP/pkg/hive ClusterManager,SyncSetManager
//

// Package mock_hive is a generated GoMock package.
package mock_hive

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"

	v10 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/openshift/hive/apis/hive/v1"
	v1alpha1 "github.com/openshift/hive/apis/hiveinternal/v1alpha1"

	api "github.com/Azure/ARO-RP/pkg/api"
)

// MockClusterManager is a mock of ClusterManager interface.
type MockClusterManager struct {
	ctrl     *gomock.Controller
	recorder *MockClusterManagerMockRecorder
	isgomock struct{}
}

// MockClusterManagerMockRecorder is the mock recorder for MockClusterManager.
type MockClusterManagerMockRecorder struct {
	mock *MockClusterManager
}

// NewMockClusterManager creates a new mock instance.
func NewMockClusterManager(ctrl *gomock.Controller) *MockClusterManager {
	mock := &MockClusterManager{ctrl: ctrl}
	mock.recorder = &MockClusterManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClusterManager) EXPECT() *MockClusterManagerMockRecorder {
	return m.recorder
}

// CreateNamespace mocks base method.
func (m *MockClusterManager) CreateNamespace(ctx context.Context, docID string) (*v10.Namespace, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNamespace", ctx, docID)
	ret0, _ := ret[0].(*v10.Namespace)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNamespace indicates an expected call of CreateNamespace.
func (mr *MockClusterManagerMockRecorder) CreateNamespace(ctx, docID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNamespace", reflect.TypeOf((*MockClusterManager)(nil).CreateNamespace), ctx, docID)
}

// CreateOrUpdate mocks base method.
func (m *MockClusterManager) CreateOrUpdate(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOrUpdate", ctx, sub, doc)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateOrUpdate indicates an expected call of CreateOrUpdate.
func (mr *MockClusterManagerMockRecorder) CreateOrUpdate(ctx, sub, doc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOrUpdate", reflect.TypeOf((*MockClusterManager)(nil).CreateOrUpdate), ctx, sub, doc)
}

// Delete mocks base method.
func (m *MockClusterManager) Delete(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, doc)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockClusterManagerMockRecorder) Delete(ctx, doc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockClusterManager)(nil).Delete), ctx, doc)
}

// GetClusterDeployment mocks base method.
func (m *MockClusterManager) GetClusterDeployment(ctx context.Context, doc *api.OpenShiftClusterDocument) (*v1.ClusterDeployment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClusterDeployment", ctx, doc)
	ret0, _ := ret[0].(*v1.ClusterDeployment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClusterDeployment indicates an expected call of GetClusterDeployment.
func (mr *MockClusterManagerMockRecorder) GetClusterDeployment(ctx, doc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClusterDeployment", reflect.TypeOf((*MockClusterManager)(nil).GetClusterDeployment), ctx, doc)
}

// GetClusterSync mocks base method.
func (m *MockClusterManager) GetClusterSync(ctx context.Context, doc *api.OpenShiftClusterDocument) (*v1alpha1.ClusterSync, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClusterSync", ctx, doc)
	ret0, _ := ret[0].(*v1alpha1.ClusterSync)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClusterSync indicates an expected call of GetClusterSync.
func (mr *MockClusterManagerMockRecorder) GetClusterSync(ctx, doc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClusterSync", reflect.TypeOf((*MockClusterManager)(nil).GetClusterSync), ctx, doc)
}

// Install mocks base method.
func (m *MockClusterManager) Install(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, version *api.OpenShiftVersion, customManifests map[string]runtime.Object) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Install", ctx, sub, doc, version, customManifests)
	ret0, _ := ret[0].(error)
	return ret0
}

// Install indicates an expected call of Install.
func (mr *MockClusterManagerMockRecorder) Install(ctx, sub, doc, version, customManifests any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Install", reflect.TypeOf((*MockClusterManager)(nil).Install), ctx, sub, doc, version, customManifests)
}

// IsClusterDeploymentReady mocks base method.
func (m *MockClusterManager) IsClusterDeploymentReady(ctx context.Context, doc *api.OpenShiftClusterDocument) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsClusterDeploymentReady", ctx, doc)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsClusterDeploymentReady indicates an expected call of IsClusterDeploymentReady.
func (mr *MockClusterManagerMockRecorder) IsClusterDeploymentReady(ctx, doc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsClusterDeploymentReady", reflect.TypeOf((*MockClusterManager)(nil).IsClusterDeploymentReady), ctx, doc)
}

// IsClusterInstallationComplete mocks base method.
func (m *MockClusterManager) IsClusterInstallationComplete(ctx context.Context, doc *api.OpenShiftClusterDocument) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsClusterInstallationComplete", ctx, doc)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsClusterInstallationComplete indicates an expected call of IsClusterInstallationComplete.
func (mr *MockClusterManagerMockRecorder) IsClusterInstallationComplete(ctx, doc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsClusterInstallationComplete", reflect.TypeOf((*MockClusterManager)(nil).IsClusterInstallationComplete), ctx, doc)
}

// ResetCorrelationData mocks base method.
func (m *MockClusterManager) ResetCorrelationData(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResetCorrelationData", ctx, doc)
	ret0, _ := ret[0].(error)
	return ret0
}

// ResetCorrelationData indicates an expected call of ResetCorrelationData.
func (mr *MockClusterManagerMockRecorder) ResetCorrelationData(ctx, doc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetCorrelationData", reflect.TypeOf((*MockClusterManager)(nil).ResetCorrelationData), ctx, doc)
}

// MockSyncSetManager is a mock of SyncSetManager interface.
type MockSyncSetManager struct {
	ctrl     *gomock.Controller
	recorder *MockSyncSetManagerMockRecorder
	isgomock struct{}
}

// MockSyncSetManagerMockRecorder is the mock recorder for MockSyncSetManager.
type MockSyncSetManagerMockRecorder struct {
	mock *MockSyncSetManager
}

// NewMockSyncSetManager creates a new mock instance.
func NewMockSyncSetManager(ctrl *gomock.Controller) *MockSyncSetManager {
	mock := &MockSyncSetManager{ctrl: ctrl}
	mock.recorder = &MockSyncSetManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSyncSetManager) EXPECT() *MockSyncSetManagerMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockSyncSetManager) Get(ctx context.Context, namespace, name string, getType reflect.Type) (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, namespace, name, getType)
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockSyncSetManagerMockRecorder) Get(ctx, namespace, name, getType any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockSyncSetManager)(nil).Get), ctx, namespace, name, getType)
}

// List mocks base method.
func (m *MockSyncSetManager) List(ctx context.Context, namespace, label string, listType reflect.Type) (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, namespace, label, listType)
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockSyncSetManagerMockRecorder) List(ctx, namespace, label, listType any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockSyncSetManager)(nil).List), ctx, namespace, label, listType)
}
