package mocks

import (
	context "context"
	reflect "reflect"

	domain "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	gomock "go.uber.org/mock/gomock"
)

type MockAuthenticator struct {
	ctrl     *gomock.Controller
	recorder *MockAuthenticatorMockRecorder
	isgomock struct{}
}

type MockAuthenticatorMockRecorder struct {
	mock *MockAuthenticator
}

func NewMockAuthenticator(ctrl *gomock.Controller) *MockAuthenticator {
	mock := &MockAuthenticator{ctrl: ctrl}
	mock.recorder = &MockAuthenticatorMockRecorder{mock}
	return mock
}

func (m *MockAuthenticator) EXPECT() *MockAuthenticatorMockRecorder {
	return m.recorder
}

func (m *MockAuthenticator) Authenticate(ctx context.Context, sessionID string) (domain.UserID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Authenticate", ctx, sessionID)
	ret0, _ := ret[0].(domain.UserID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockAuthenticatorMockRecorder) Authenticate(ctx, sessionID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Authenticate", reflect.TypeOf((*MockAuthenticator)(nil).Authenticate), ctx, sessionID)
}

type MockCatalogEditor struct {
	ctrl     *gomock.Controller
	recorder *MockCatalogEditorMockRecorder
	isgomock struct{}
}

type MockCatalogEditorMockRecorder struct {
	mock *MockCatalogEditor
}

func NewMockCatalogEditor(ctrl *gomock.Controller) *MockCatalogEditor {
	mock := &MockCatalogEditor{ctrl: ctrl}
	mock.recorder = &MockCatalogEditorMockRecorder{mock}
	return mock
}

func (m *MockCatalogEditor) EXPECT() *MockCatalogEditorMockRecorder {
	return m.recorder
}

func (m *MockCatalogEditor) CanEditPersons(arg0 context.Context, arg1 domain.UserID) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CanEditPersons", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockCatalogEditorMockRecorder) CanEditPersons(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CanEditPersons", reflect.TypeOf((*MockCatalogEditor)(nil).CanEditPersons), arg0, arg1)
}
