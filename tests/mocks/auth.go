package mocks

import (
	context "context"
	reflect "reflect"

	domain "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	gomock "go.uber.org/mock/gomock"
)

type MockPasswordHasher struct {
	ctrl     *gomock.Controller
	recorder *MockPasswordHasherMockRecorder
	isgomock struct{}
}

type MockPasswordHasherMockRecorder struct {
	mock *MockPasswordHasher
}

func NewMockPasswordHasher(ctrl *gomock.Controller) *MockPasswordHasher {
	mock := &MockPasswordHasher{ctrl: ctrl}
	mock.recorder = &MockPasswordHasherMockRecorder{mock}
	return mock
}

func (m *MockPasswordHasher) EXPECT() *MockPasswordHasherMockRecorder {
	return m.recorder
}

func (m *MockPasswordHasher) Hash(password string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Hash", password)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockPasswordHasherMockRecorder) Hash(password any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Hash", reflect.TypeOf((*MockPasswordHasher)(nil).Hash), password)
}

type MockPasswordVerifier struct {
	ctrl     *gomock.Controller
	recorder *MockPasswordVerifierMockRecorder
	isgomock struct{}
}

type MockPasswordVerifierMockRecorder struct {
	mock *MockPasswordVerifier
}

func NewMockPasswordVerifier(ctrl *gomock.Controller) *MockPasswordVerifier {
	mock := &MockPasswordVerifier{ctrl: ctrl}
	mock.recorder = &MockPasswordVerifierMockRecorder{mock}
	return mock
}

func (m *MockPasswordVerifier) EXPECT() *MockPasswordVerifierMockRecorder {
	return m.recorder
}

func (m *MockPasswordVerifier) Verify(password, hash string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Verify", password, hash)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockPasswordVerifierMockRecorder) Verify(password, hash any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Verify", reflect.TypeOf((*MockPasswordVerifier)(nil).Verify), password, hash)
}

type MockSessionStore struct {
	ctrl     *gomock.Controller
	recorder *MockSessionStoreMockRecorder
	isgomock struct{}
}

type MockSessionStoreMockRecorder struct {
	mock *MockSessionStore
}

func NewMockSessionStore(ctrl *gomock.Controller) *MockSessionStore {
	mock := &MockSessionStore{ctrl: ctrl}
	mock.recorder = &MockSessionStoreMockRecorder{mock}
	return mock
}

func (m *MockSessionStore) EXPECT() *MockSessionStoreMockRecorder {
	return m.recorder
}

func (m *MockSessionStore) Create(ctx context.Context, userID domain.UserID) (*domain.Session, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, userID)
	ret0, _ := ret[0].(*domain.Session)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockSessionStoreMockRecorder) Create(ctx, userID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockSessionStore)(nil).Create), ctx, userID)
}

func (m *MockSessionStore) Delete(ctx context.Context, sessionID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, sessionID)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockSessionStoreMockRecorder) Delete(ctx, sessionID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockSessionStore)(nil).Delete), ctx, sessionID)
}

func (m *MockSessionStore) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, sessionID)
	ret0, _ := ret[0].(*domain.Session)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockSessionStoreMockRecorder) Get(ctx, sessionID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockSessionStore)(nil).Get), ctx, sessionID)
}

type MockUserRepository struct {
	ctrl     *gomock.Controller
	recorder *MockUserRepositoryMockRecorder
	isgomock struct{}
}

type MockUserRepositoryMockRecorder struct {
	mock *MockUserRepository
}

func NewMockUserRepository(ctrl *gomock.Controller) *MockUserRepository {
	mock := &MockUserRepository{ctrl: ctrl}
	mock.recorder = &MockUserRepositoryMockRecorder{mock}
	return mock
}

func (m *MockUserRepository) EXPECT() *MockUserRepositoryMockRecorder {
	return m.recorder
}

func (m *MockUserRepository) Create(ctx context.Context, email, username, passwordHash string) (domain.UserID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, email, username, passwordHash)
	ret0, _ := ret[0].(domain.UserID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockUserRepositoryMockRecorder) Create(ctx, email, username, passwordHash any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockUserRepository)(nil).Create), ctx, email, username, passwordHash)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByEmail", ctx, email)
	ret0, _ := ret[0].(*domain.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockUserRepositoryMockRecorder) GetByEmail(ctx, email any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByEmail", reflect.TypeOf((*MockUserRepository)(nil).GetByEmail), ctx, email)
}

func (m *MockUserRepository) GetUser(ctx context.Context, id domain.UserID) (*domain.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUser", ctx, id)
	ret0, _ := ret[0].(*domain.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockUserRepositoryMockRecorder) GetUser(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockUserRepository)(nil).GetUser), ctx, id)
}
