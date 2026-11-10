package mocks

import (
	context "context"
	reflect "reflect"

	domain "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	gomock "go.uber.org/mock/gomock"
)

type MockPersonalRatingRepository struct {
	ctrl     *gomock.Controller
	recorder *MockPersonalRatingRepositoryMockRecorder
	isgomock struct{}
}

type MockPersonalRatingRepositoryMockRecorder struct {
	mock *MockPersonalRatingRepository
}

func NewMockPersonalRatingRepository(ctrl *gomock.Controller) *MockPersonalRatingRepository {
	mock := &MockPersonalRatingRepository{ctrl: ctrl}
	mock.recorder = &MockPersonalRatingRepositoryMockRecorder{mock}
	return mock
}

func (m *MockPersonalRatingRepository) EXPECT() *MockPersonalRatingRepositoryMockRecorder {
	return m.recorder
}

func (m *MockPersonalRatingRepository) GetMyScore(arg0 context.Context, arg1 domain.UserID, arg2 domain.FilmID) (*int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMyScore", arg0, arg1, arg2)
	ret0, _ := ret[0].(*int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockPersonalRatingRepositoryMockRecorder) GetMyScore(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMyScore", reflect.TypeOf((*MockPersonalRatingRepository)(nil).GetMyScore), arg0, arg1, arg2)
}

type MockRatingFilmLookup struct {
	ctrl     *gomock.Controller
	recorder *MockRatingFilmLookupMockRecorder
	isgomock struct{}
}

type MockRatingFilmLookupMockRecorder struct {
	mock *MockRatingFilmLookup
}

func NewMockRatingFilmLookup(ctrl *gomock.Controller) *MockRatingFilmLookup {
	mock := &MockRatingFilmLookup{ctrl: ctrl}
	mock.recorder = &MockRatingFilmLookupMockRecorder{mock}
	return mock
}

func (m *MockRatingFilmLookup) EXPECT() *MockRatingFilmLookupMockRecorder {
	return m.recorder
}

func (m *MockRatingFilmLookup) GetFilmByID(arg0 context.Context, arg1 domain.FilmID) (domain.Film, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFilmByID", arg0, arg1)
	ret0, _ := ret[0].(domain.Film)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockRatingFilmLookupMockRecorder) GetFilmByID(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFilmByID", reflect.TypeOf((*MockRatingFilmLookup)(nil).GetFilmByID), arg0, arg1)
}
