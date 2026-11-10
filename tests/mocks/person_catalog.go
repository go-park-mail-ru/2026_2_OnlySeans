package mocks

import (
	context "context"
	reflect "reflect"

	domain "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	gomock "go.uber.org/mock/gomock"
)

type MockCatalogFilmRepository struct {
	ctrl     *gomock.Controller
	recorder *MockCatalogFilmRepositoryMockRecorder
	isgomock struct{}
}

type MockCatalogFilmRepositoryMockRecorder struct {
	mock *MockCatalogFilmRepository
}

func NewMockCatalogFilmRepository(ctrl *gomock.Controller) *MockCatalogFilmRepository {
	mock := &MockCatalogFilmRepository{ctrl: ctrl}
	mock.recorder = &MockCatalogFilmRepositoryMockRecorder{mock}
	return mock
}

func (m *MockCatalogFilmRepository) EXPECT() *MockCatalogFilmRepositoryMockRecorder {
	return m.recorder
}

func (m *MockCatalogFilmRepository) GetFilmByID(arg0 context.Context, arg1 domain.FilmID) (domain.Film, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFilmByID", arg0, arg1)
	ret0, _ := ret[0].(domain.Film)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockCatalogFilmRepositoryMockRecorder) GetFilmByID(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFilmByID", reflect.TypeOf((*MockCatalogFilmRepository)(nil).GetFilmByID), arg0, arg1)
}
