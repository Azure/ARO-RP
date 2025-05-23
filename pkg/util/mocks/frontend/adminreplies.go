// Code generated by MockGen. DO NOT EDIT.
// Source: adminreplies.go
//
// Generated by this command:
//
//	mockgen -source adminreplies.go -destination=../util/mocks/frontend/adminreplies.go github.com/Azure/ARO-RP/pkg/frontend StreamResponder
//

// Package mock_frontend is a generated GoMock package.
package mock_frontend

import (
	io "io"
	http "net/http"
	reflect "reflect"

	logrus "github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"
)

// MockStreamResponder is a mock of StreamResponder interface.
type MockStreamResponder struct {
	ctrl     *gomock.Controller
	recorder *MockStreamResponderMockRecorder
	isgomock struct{}
}

// MockStreamResponderMockRecorder is the mock recorder for MockStreamResponder.
type MockStreamResponderMockRecorder struct {
	mock *MockStreamResponder
}

// NewMockStreamResponder creates a new mock instance.
func NewMockStreamResponder(ctrl *gomock.Controller) *MockStreamResponder {
	mock := &MockStreamResponder{ctrl: ctrl}
	mock.recorder = &MockStreamResponderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamResponder) EXPECT() *MockStreamResponderMockRecorder {
	return m.recorder
}

// AdminReplyStream mocks base method.
func (m *MockStreamResponder) AdminReplyStream(log *logrus.Entry, w http.ResponseWriter, header http.Header, reader io.Reader, err error) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AdminReplyStream", log, w, header, reader, err)
}

// AdminReplyStream indicates an expected call of AdminReplyStream.
func (mr *MockStreamResponderMockRecorder) AdminReplyStream(log, w, header, reader, err any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AdminReplyStream", reflect.TypeOf((*MockStreamResponder)(nil).AdminReplyStream), log, w, header, reader, err)
}

// ReplyStream mocks base method.
func (m *MockStreamResponder) ReplyStream(log *logrus.Entry, w http.ResponseWriter, header http.Header, reader io.Reader, err error) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ReplyStream", log, w, header, reader, err)
}

// ReplyStream indicates an expected call of ReplyStream.
func (mr *MockStreamResponderMockRecorder) ReplyStream(log, w, header, reader, err any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReplyStream", reflect.TypeOf((*MockStreamResponder)(nil).ReplyStream), log, w, header, reader, err)
}
