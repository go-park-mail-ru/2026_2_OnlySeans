package mocks

import (
	context "context"
	reflect "reflect"

	domain "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	gomock "go.uber.org/mock/gomock"
)

type MockProfileRepository struct {
	ctrl     *gomock.Controller
	recorder *MockProfileRepositoryMockRecorder
	isgomock struct{}
}

type MockProfileRepositoryMockRecorder struct {
	mock *MockProfileRepository
}

func NewMockProfileRepository(ctrl *gomock.Controller) *MockProfileRepository {
	mock := &MockProfileRepository{ctrl: ctrl}
	mock.recorder = &MockProfileRepositoryMockRecorder{mock}
	return mock
}

func (m *MockProfileRepository) EXPECT() *MockProfileRepositoryMockRecorder {
	return m.recorder
}

func (m *MockProfileRepository) UpdateProfile(arg0 context.Context, arg1 domain.UserID, arg2, arg3 string) (*domain.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateProfile", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*domain.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockProfileRepositoryMockRecorder) UpdateProfile(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateProfile", reflect.TypeOf((*MockProfileRepository)(nil).UpdateProfile), arg0, arg1, arg2, arg3)
}

type MockProfileReader struct {
	ctrl     *gomock.Controller
	recorder *MockProfileReaderMockRecorder
	isgomock struct{}
}

type MockProfileReaderMockRecorder struct {
	mock *MockProfileReader
}

func NewMockProfileReader(ctrl *gomock.Controller) *MockProfileReader {
	mock := &MockProfileReader{ctrl: ctrl}
	mock.recorder = &MockProfileReaderMockRecorder{mock}
	return mock
}

func (m *MockProfileReader) EXPECT() *MockProfileReaderMockRecorder {
	return m.recorder
}

func (m *MockProfileReader) GetProfile(arg0 context.Context, arg1 domain.UserID) (*domain.Profile, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProfile", arg0, arg1)
	ret0, _ := ret[0].(*domain.Profile)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockProfileReaderMockRecorder) GetProfile(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProfile", reflect.TypeOf((*MockProfileReader)(nil).GetProfile), arg0, arg1)
}

type MockProfilePosterStorage struct {
	ctrl     *gomock.Controller
	recorder *MockProfilePosterStorageMockRecorder
	isgomock struct{}
}

type MockProfilePosterStorageMockRecorder struct {
	mock *MockProfilePosterStorage
}

func NewMockProfilePosterStorage(ctrl *gomock.Controller) *MockProfilePosterStorage {
	mock := &MockProfilePosterStorage{ctrl: ctrl}
	mock.recorder = &MockProfilePosterStorageMockRecorder{mock}
	return mock
}

func (m *MockProfilePosterStorage) EXPECT() *MockProfilePosterStorageMockRecorder {
	return m.recorder
}

func (m *MockProfilePosterStorage) GetURL(arg0 context.Context, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetURL", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockProfilePosterStorageMockRecorder) GetURL(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetURL", reflect.TypeOf((*MockProfilePosterStorage)(nil).GetURL), arg0, arg1)
}
