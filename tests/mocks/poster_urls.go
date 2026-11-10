package mocks

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

type MockPosterURLStorage struct {
	ctrl     *gomock.Controller
	recorder *MockPosterURLStorageMockRecorder
	isgomock struct{}
}

type MockPosterURLStorageMockRecorder struct {
	mock *MockPosterURLStorage
}

func NewMockPosterURLStorage(ctrl *gomock.Controller) *MockPosterURLStorage {
	mock := &MockPosterURLStorage{ctrl: ctrl}
	mock.recorder = &MockPosterURLStorageMockRecorder{mock}
	return mock
}

func (m *MockPosterURLStorage) EXPECT() *MockPosterURLStorageMockRecorder {
	return m.recorder
}

func (m *MockPosterURLStorage) GetURL(ctx context.Context, key string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetURL", ctx, key)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockPosterURLStorageMockRecorder) GetURL(ctx, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetURL", reflect.TypeOf((*MockPosterURLStorage)(nil).GetURL), ctx, key)
}
