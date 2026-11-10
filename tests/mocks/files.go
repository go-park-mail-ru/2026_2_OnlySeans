package mocks

import (
	context "context"
	io "io"
	reflect "reflect"

	domain "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	gomock "go.uber.org/mock/gomock"
)

type MockAvatarStorage struct {
	ctrl     *gomock.Controller
	recorder *MockAvatarStorageMockRecorder
	isgomock struct{}
}

type MockAvatarStorageMockRecorder struct {
	mock *MockAvatarStorage
}

func NewMockAvatarStorage(ctrl *gomock.Controller) *MockAvatarStorage {
	mock := &MockAvatarStorage{ctrl: ctrl}
	mock.recorder = &MockAvatarStorageMockRecorder{mock}
	return mock
}

func (m *MockAvatarStorage) EXPECT() *MockAvatarStorageMockRecorder {
	return m.recorder
}

func (m *MockAvatarStorage) Delete(ctx context.Context, key string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, key)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockAvatarStorageMockRecorder) Delete(ctx, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockAvatarStorage)(nil).Delete), ctx, key)
}

func (m *MockAvatarStorage) Upload(ctx context.Context, key, contentType string, body io.ReadSeeker) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Upload", ctx, key, contentType, body)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockAvatarStorageMockRecorder) Upload(ctx, key, contentType, body any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Upload", reflect.TypeOf((*MockAvatarStorage)(nil).Upload), ctx, key, contentType, body)
}

type MockAvatarWriter struct {
	ctrl     *gomock.Controller
	recorder *MockAvatarWriterMockRecorder
	isgomock struct{}
}

type MockAvatarWriterMockRecorder struct {
	mock *MockAvatarWriter
}

func NewMockAvatarWriter(ctrl *gomock.Controller) *MockAvatarWriter {
	mock := &MockAvatarWriter{ctrl: ctrl}
	mock.recorder = &MockAvatarWriterMockRecorder{mock}
	return mock
}

func (m *MockAvatarWriter) EXPECT() *MockAvatarWriterMockRecorder {
	return m.recorder
}

func (m *MockAvatarWriter) SaveAvatar(ctx context.Context, userID domain.UserID, file domain.AvatarFile) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveAvatar", ctx, userID, file)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockAvatarWriterMockRecorder) SaveAvatar(ctx, userID, file any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveAvatar", reflect.TypeOf((*MockAvatarWriter)(nil).SaveAvatar), ctx, userID, file)
}

type MockAvatarURLStorage struct {
	ctrl     *gomock.Controller
	recorder *MockAvatarURLStorageMockRecorder
	isgomock struct{}
}

type MockAvatarURLStorageMockRecorder struct {
	mock *MockAvatarURLStorage
}

func NewMockAvatarURLStorage(ctrl *gomock.Controller) *MockAvatarURLStorage {
	mock := &MockAvatarURLStorage{ctrl: ctrl}
	mock.recorder = &MockAvatarURLStorageMockRecorder{mock}
	return mock
}

func (m *MockAvatarURLStorage) EXPECT() *MockAvatarURLStorageMockRecorder {
	return m.recorder
}

func (m *MockAvatarURLStorage) GetURL(ctx context.Context, key string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetURL", ctx, key)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockAvatarURLStorageMockRecorder) GetURL(ctx, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetURL", reflect.TypeOf((*MockAvatarURLStorage)(nil).GetURL), ctx, key)
}

type MockPosterStorage struct {
	ctrl     *gomock.Controller
	recorder *MockPosterStorageMockRecorder
	isgomock struct{}
}

type MockPosterStorageMockRecorder struct {
	mock *MockPosterStorage
}

func NewMockPosterStorage(ctrl *gomock.Controller) *MockPosterStorage {
	mock := &MockPosterStorage{ctrl: ctrl}
	mock.recorder = &MockPosterStorageMockRecorder{mock}
	return mock
}

func (m *MockPosterStorage) EXPECT() *MockPosterStorageMockRecorder {
	return m.recorder
}

func (m *MockPosterStorage) Delete(ctx context.Context, key string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, key)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockPosterStorageMockRecorder) Delete(ctx, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockPosterStorage)(nil).Delete), ctx, key)
}

func (m *MockPosterStorage) Upload(ctx context.Context, key, contentType string, body io.ReadSeeker) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Upload", ctx, key, contentType, body)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockPosterStorageMockRecorder) Upload(ctx, key, contentType, body any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Upload", reflect.TypeOf((*MockPosterStorage)(nil).Upload), ctx, key, contentType, body)
}

type MockPosterFilmReader struct {
	ctrl     *gomock.Controller
	recorder *MockPosterFilmReaderMockRecorder
	isgomock struct{}
}

type MockPosterFilmReaderMockRecorder struct {
	mock *MockPosterFilmReader
}

func NewMockPosterFilmReader(ctrl *gomock.Controller) *MockPosterFilmReader {
	mock := &MockPosterFilmReader{ctrl: ctrl}
	mock.recorder = &MockPosterFilmReaderMockRecorder{mock}
	return mock
}

func (m *MockPosterFilmReader) EXPECT() *MockPosterFilmReaderMockRecorder {
	return m.recorder
}

func (m *MockPosterFilmReader) GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFilmByID", ctx, id)
	ret0, _ := ret[0].(domain.Film)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockPosterFilmReaderMockRecorder) GetFilmByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFilmByID", reflect.TypeOf((*MockPosterFilmReader)(nil).GetFilmByID), ctx, id)
}
