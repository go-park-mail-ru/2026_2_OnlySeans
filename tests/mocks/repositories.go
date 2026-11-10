package mocks

import (
	context "context"
	json "encoding/json"
	reflect "reflect"

	domain "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	gomock "go.uber.org/mock/gomock"
)

type MockFilmCreator struct {
	ctrl     *gomock.Controller
	recorder *MockFilmCreatorMockRecorder
	isgomock struct{}
}

type MockFilmCreatorMockRecorder struct {
	mock *MockFilmCreator
}

func NewMockFilmCreator(ctrl *gomock.Controller) *MockFilmCreator {
	mock := &MockFilmCreator{ctrl: ctrl}
	mock.recorder = &MockFilmCreatorMockRecorder{mock}
	return mock
}

func (m *MockFilmCreator) EXPECT() *MockFilmCreatorMockRecorder {
	return m.recorder
}

func (m *MockFilmCreator) CreateFilm(ctx context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateFilm", ctx, input)
	ret0, _ := ret[0].(domain.FilmID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFilmCreatorMockRecorder) CreateFilm(ctx, input any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateFilm", reflect.TypeOf((*MockFilmCreator)(nil).CreateFilm), ctx, input)
}

type MockFilmUpdater struct {
	ctrl     *gomock.Controller
	recorder *MockFilmUpdaterMockRecorder
	isgomock struct{}
}

type MockFilmUpdaterMockRecorder struct {
	mock *MockFilmUpdater
}

func NewMockFilmUpdater(ctrl *gomock.Controller) *MockFilmUpdater {
	mock := &MockFilmUpdater{ctrl: ctrl}
	mock.recorder = &MockFilmUpdaterMockRecorder{mock}
	return mock
}

func (m *MockFilmUpdater) EXPECT() *MockFilmUpdaterMockRecorder {
	return m.recorder
}

func (m *MockFilmUpdater) UpdateFilm(ctx context.Context, id domain.FilmID, input domain.CreateFilmInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateFilm", ctx, id, input)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockFilmUpdaterMockRecorder) UpdateFilm(ctx, id, input any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateFilm", reflect.TypeOf((*MockFilmUpdater)(nil).UpdateFilm), ctx, id, input)
}

type MockFilmDeleter struct {
	ctrl     *gomock.Controller
	recorder *MockFilmDeleterMockRecorder
	isgomock struct{}
}

type MockFilmDeleterMockRecorder struct {
	mock *MockFilmDeleter
}

func NewMockFilmDeleter(ctrl *gomock.Controller) *MockFilmDeleter {
	mock := &MockFilmDeleter{ctrl: ctrl}
	mock.recorder = &MockFilmDeleterMockRecorder{mock}
	return mock
}

func (m *MockFilmDeleter) EXPECT() *MockFilmDeleterMockRecorder {
	return m.recorder
}

func (m *MockFilmDeleter) DeleteFilm(ctx context.Context, id domain.FilmID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFilm", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockFilmDeleterMockRecorder) DeleteFilm(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFilm", reflect.TypeOf((*MockFilmDeleter)(nil).DeleteFilm), ctx, id)
}

type MockFilmRatingRepository struct {
	ctrl     *gomock.Controller
	recorder *MockFilmRatingRepositoryMockRecorder
	isgomock struct{}
}

type MockFilmRatingRepositoryMockRecorder struct {
	mock *MockFilmRatingRepository
}

func NewMockFilmRatingRepository(ctrl *gomock.Controller) *MockFilmRatingRepository {
	mock := &MockFilmRatingRepository{ctrl: ctrl}
	mock.recorder = &MockFilmRatingRepositoryMockRecorder{mock}
	return mock
}

func (m *MockFilmRatingRepository) EXPECT() *MockFilmRatingRepositoryMockRecorder {
	return m.recorder
}

func (m *MockFilmRatingRepository) DeleteFilmRating(ctx context.Context, accountID domain.UserID, filmID domain.FilmID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFilmRating", ctx, accountID, filmID)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockFilmRatingRepositoryMockRecorder) DeleteFilmRating(ctx, accountID, filmID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFilmRating", reflect.TypeOf((*MockFilmRatingRepository)(nil).DeleteFilmRating), ctx, accountID, filmID)
}

func (m *MockFilmRatingRepository) GetFilmRatingSummary(ctx context.Context, filmID domain.FilmID) (domain.FilmRatingSummary, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFilmRatingSummary", ctx, filmID)
	ret0, _ := ret[0].(domain.FilmRatingSummary)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFilmRatingRepositoryMockRecorder) GetFilmRatingSummary(ctx, filmID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFilmRatingSummary", reflect.TypeOf((*MockFilmRatingRepository)(nil).GetFilmRatingSummary), ctx, filmID)
}

func (m *MockFilmRatingRepository) GetFilmRatingsSummaries(ctx context.Context, filmIDs []domain.FilmID) ([]domain.FilmRatingSummary, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFilmRatingsSummaries", ctx, filmIDs)
	ret0, _ := ret[0].([]domain.FilmRatingSummary)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFilmRatingRepositoryMockRecorder) GetFilmRatingsSummaries(ctx, filmIDs any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFilmRatingsSummaries", reflect.TypeOf((*MockFilmRatingRepository)(nil).GetFilmRatingsSummaries), ctx, filmIDs)
}

func (m *MockFilmRatingRepository) SetFilmRating(ctx context.Context, accountID domain.UserID, filmID domain.FilmID, score int) (domain.FilmRating, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetFilmRating", ctx, accountID, filmID, score)
	ret0, _ := ret[0].(domain.FilmRating)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFilmRatingRepositoryMockRecorder) SetFilmRating(ctx, accountID, filmID, score any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetFilmRating", reflect.TypeOf((*MockFilmRatingRepository)(nil).SetFilmRating), ctx, accountID, filmID, score)
}

type MockCollectionFilmRatingRepository struct {
	ctrl     *gomock.Controller
	recorder *MockCollectionFilmRatingRepositoryMockRecorder
	isgomock struct{}
}

type MockCollectionFilmRatingRepositoryMockRecorder struct {
	mock *MockCollectionFilmRatingRepository
}

func NewMockCollectionFilmRatingRepository(ctrl *gomock.Controller) *MockCollectionFilmRatingRepository {
	mock := &MockCollectionFilmRatingRepository{ctrl: ctrl}
	mock.recorder = &MockCollectionFilmRatingRepositoryMockRecorder{mock}
	return mock
}

func (m *MockCollectionFilmRatingRepository) EXPECT() *MockCollectionFilmRatingRepositoryMockRecorder {
	return m.recorder
}

func (m *MockCollectionFilmRatingRepository) GetCollectionFilmRatingSummaries(ctx context.Context, collectionID domain.CollectionID, limit, offset int) ([]domain.FilmRatingSummary, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCollectionFilmRatingSummaries", ctx, collectionID, limit, offset)
	ret0, _ := ret[0].([]domain.FilmRatingSummary)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockCollectionFilmRatingRepositoryMockRecorder) GetCollectionFilmRatingSummaries(ctx, collectionID, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCollectionFilmRatingSummaries", reflect.TypeOf((*MockCollectionFilmRatingRepository)(nil).GetCollectionFilmRatingSummaries), ctx, collectionID, limit, offset)
}

type MockFilmRatingCollectionRepository struct {
	ctrl     *gomock.Controller
	recorder *MockFilmRatingCollectionRepositoryMockRecorder
	isgomock struct{}
}

type MockFilmRatingCollectionRepositoryMockRecorder struct {
	mock *MockFilmRatingCollectionRepository
}

func NewMockFilmRatingCollectionRepository(ctrl *gomock.Controller) *MockFilmRatingCollectionRepository {
	mock := &MockFilmRatingCollectionRepository{ctrl: ctrl}
	mock.recorder = &MockFilmRatingCollectionRepositoryMockRecorder{mock}
	return mock
}

func (m *MockFilmRatingCollectionRepository) EXPECT() *MockFilmRatingCollectionRepositoryMockRecorder {
	return m.recorder
}

func (m *MockFilmRatingCollectionRepository) DeleteFilmRating(ctx context.Context, accountID domain.UserID, filmID domain.FilmID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFilmRating", ctx, accountID, filmID)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockFilmRatingCollectionRepositoryMockRecorder) DeleteFilmRating(ctx, accountID, filmID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFilmRating", reflect.TypeOf((*MockFilmRatingCollectionRepository)(nil).DeleteFilmRating), ctx, accountID, filmID)
}

func (m *MockFilmRatingCollectionRepository) GetCollectionFilmRatingSummaries(ctx context.Context, collectionID domain.CollectionID, limit, offset int) ([]domain.FilmRatingSummary, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCollectionFilmRatingSummaries", ctx, collectionID, limit, offset)
	ret0, _ := ret[0].([]domain.FilmRatingSummary)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockFilmRatingCollectionRepositoryMockRecorder) GetCollectionFilmRatingSummaries(ctx, collectionID, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCollectionFilmRatingSummaries", reflect.TypeOf((*MockFilmRatingCollectionRepository)(nil).GetCollectionFilmRatingSummaries), ctx, collectionID, limit, offset)
}

func (m *MockFilmRatingCollectionRepository) GetFilmRatingSummary(ctx context.Context, filmID domain.FilmID) (domain.FilmRatingSummary, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFilmRatingSummary", ctx, filmID)
	ret0, _ := ret[0].(domain.FilmRatingSummary)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFilmRatingCollectionRepositoryMockRecorder) GetFilmRatingSummary(ctx, filmID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFilmRatingSummary", reflect.TypeOf((*MockFilmRatingCollectionRepository)(nil).GetFilmRatingSummary), ctx, filmID)
}

func (m *MockFilmRatingCollectionRepository) GetFilmRatingsSummaries(ctx context.Context, filmIDs []domain.FilmID) ([]domain.FilmRatingSummary, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFilmRatingsSummaries", ctx, filmIDs)
	ret0, _ := ret[0].([]domain.FilmRatingSummary)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFilmRatingCollectionRepositoryMockRecorder) GetFilmRatingsSummaries(ctx, filmIDs any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFilmRatingsSummaries", reflect.TypeOf((*MockFilmRatingCollectionRepository)(nil).GetFilmRatingsSummaries), ctx, filmIDs)
}

func (m *MockFilmRatingCollectionRepository) SetFilmRating(ctx context.Context, accountID domain.UserID, filmID domain.FilmID, score int) (domain.FilmRating, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetFilmRating", ctx, accountID, filmID, score)
	ret0, _ := ret[0].(domain.FilmRating)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFilmRatingCollectionRepositoryMockRecorder) SetFilmRating(ctx, accountID, filmID, score any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetFilmRating", reflect.TypeOf((*MockFilmRatingCollectionRepository)(nil).SetFilmRating), ctx, accountID, filmID, score)
}

type MockFilmRepository struct {
	ctrl     *gomock.Controller
	recorder *MockFilmRepositoryMockRecorder
	isgomock struct{}
}

type MockFilmRepositoryMockRecorder struct {
	mock *MockFilmRepository
}

func NewMockFilmRepository(ctrl *gomock.Controller) *MockFilmRepository {
	mock := &MockFilmRepository{ctrl: ctrl}
	mock.recorder = &MockFilmRepositoryMockRecorder{mock}
	return mock
}

func (m *MockFilmRepository) EXPECT() *MockFilmRepositoryMockRecorder {
	return m.recorder
}

func (m *MockFilmRepository) GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFilmByID", ctx, id)
	ret0, _ := ret[0].(domain.Film)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFilmRepositoryMockRecorder) GetFilmByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFilmByID", reflect.TypeOf((*MockFilmRepository)(nil).GetFilmByID), ctx, id)
}

func (m *MockFilmRepository) ListFilms(ctx context.Context, limit, offset int) ([]domain.Film, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListFilms", ctx, limit, offset)
	ret0, _ := ret[0].([]domain.Film)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockFilmRepositoryMockRecorder) ListFilms(ctx, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListFilms", reflect.TypeOf((*MockFilmRepository)(nil).ListFilms), ctx, limit, offset)
}

type MockCollectionRepository struct {
	ctrl     *gomock.Controller
	recorder *MockCollectionRepositoryMockRecorder
	isgomock struct{}
}

type MockCollectionRepositoryMockRecorder struct {
	mock *MockCollectionRepository
}

func NewMockCollectionRepository(ctrl *gomock.Controller) *MockCollectionRepository {
	mock := &MockCollectionRepository{ctrl: ctrl}
	mock.recorder = &MockCollectionRepositoryMockRecorder{mock}
	return mock
}

func (m *MockCollectionRepository) EXPECT() *MockCollectionRepositoryMockRecorder {
	return m.recorder
}

func (m *MockCollectionRepository) GetCollectionByID(ctx context.Context, id domain.CollectionID) (domain.Collection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCollectionByID", ctx, id)
	ret0, _ := ret[0].(domain.Collection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockCollectionRepositoryMockRecorder) GetCollectionByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCollectionByID", reflect.TypeOf((*MockCollectionRepository)(nil).GetCollectionByID), ctx, id)
}

func (m *MockCollectionRepository) ListCollectionFilms(ctx context.Context, collectionID domain.CollectionID, limit, offset int) ([]domain.Film, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCollectionFilms", ctx, collectionID, limit, offset)
	ret0, _ := ret[0].([]domain.Film)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockCollectionRepositoryMockRecorder) ListCollectionFilms(ctx, collectionID, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCollectionFilms", reflect.TypeOf((*MockCollectionRepository)(nil).ListCollectionFilms), ctx, collectionID, limit, offset)
}

func (m *MockCollectionRepository) ListCollections(ctx context.Context, limit, offset int) ([]domain.Collection, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCollections", ctx, limit, offset)
	ret0, _ := ret[0].([]domain.Collection)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockCollectionRepositoryMockRecorder) ListCollections(ctx, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCollections", reflect.TypeOf((*MockCollectionRepository)(nil).ListCollections), ctx, limit, offset)
}

type MockFilmsRepository struct {
	ctrl     *gomock.Controller
	recorder *MockFilmsRepositoryMockRecorder
	isgomock struct{}
}

type MockFilmsRepositoryMockRecorder struct {
	mock *MockFilmsRepository
}

func NewMockFilmsRepository(ctrl *gomock.Controller) *MockFilmsRepository {
	mock := &MockFilmsRepository{ctrl: ctrl}
	mock.recorder = &MockFilmsRepositoryMockRecorder{mock}
	return mock
}

func (m *MockFilmsRepository) EXPECT() *MockFilmsRepositoryMockRecorder {
	return m.recorder
}

func (m *MockFilmsRepository) CreateFilm(ctx context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateFilm", ctx, input)
	ret0, _ := ret[0].(domain.FilmID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFilmsRepositoryMockRecorder) CreateFilm(ctx, input any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateFilm", reflect.TypeOf((*MockFilmsRepository)(nil).CreateFilm), ctx, input)
}

func (m *MockFilmsRepository) DeleteFilm(ctx context.Context, id domain.FilmID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFilm", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockFilmsRepositoryMockRecorder) DeleteFilm(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFilm", reflect.TypeOf((*MockFilmsRepository)(nil).DeleteFilm), ctx, id)
}

func (m *MockFilmsRepository) GetCollectionByID(ctx context.Context, id domain.CollectionID) (domain.Collection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCollectionByID", ctx, id)
	ret0, _ := ret[0].(domain.Collection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFilmsRepositoryMockRecorder) GetCollectionByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCollectionByID", reflect.TypeOf((*MockFilmsRepository)(nil).GetCollectionByID), ctx, id)
}

func (m *MockFilmsRepository) GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFilmByID", ctx, id)
	ret0, _ := ret[0].(domain.Film)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFilmsRepositoryMockRecorder) GetFilmByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFilmByID", reflect.TypeOf((*MockFilmsRepository)(nil).GetFilmByID), ctx, id)
}

func (m *MockFilmsRepository) ListCollectionFilms(ctx context.Context, collectionID domain.CollectionID, limit, offset int) ([]domain.Film, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCollectionFilms", ctx, collectionID, limit, offset)
	ret0, _ := ret[0].([]domain.Film)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockFilmsRepositoryMockRecorder) ListCollectionFilms(ctx, collectionID, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCollectionFilms", reflect.TypeOf((*MockFilmsRepository)(nil).ListCollectionFilms), ctx, collectionID, limit, offset)
}

func (m *MockFilmsRepository) ListCollections(ctx context.Context, limit, offset int) ([]domain.Collection, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCollections", ctx, limit, offset)
	ret0, _ := ret[0].([]domain.Collection)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockFilmsRepositoryMockRecorder) ListCollections(ctx, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCollections", reflect.TypeOf((*MockFilmsRepository)(nil).ListCollections), ctx, limit, offset)
}

func (m *MockFilmsRepository) ListFilms(ctx context.Context, limit, offset int) ([]domain.Film, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListFilms", ctx, limit, offset)
	ret0, _ := ret[0].([]domain.Film)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockFilmsRepositoryMockRecorder) ListFilms(ctx, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListFilms", reflect.TypeOf((*MockFilmsRepository)(nil).ListFilms), ctx, limit, offset)
}

func (m *MockFilmsRepository) UpdateFilm(ctx context.Context, id domain.FilmID, input domain.CreateFilmInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateFilm", ctx, id, input)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockFilmsRepositoryMockRecorder) UpdateFilm(ctx, id, input any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateFilm", reflect.TypeOf((*MockFilmsRepository)(nil).UpdateFilm), ctx, id, input)
}

type MockPersonRepository struct {
	ctrl     *gomock.Controller
	recorder *MockPersonRepositoryMockRecorder
	isgomock struct{}
}

type MockPersonRepositoryMockRecorder struct {
	mock *MockPersonRepository
}

func NewMockPersonRepository(ctrl *gomock.Controller) *MockPersonRepository {
	mock := &MockPersonRepository{ctrl: ctrl}
	mock.recorder = &MockPersonRepositoryMockRecorder{mock}
	return mock
}

func (m *MockPersonRepository) EXPECT() *MockPersonRepositoryMockRecorder {
	return m.recorder
}

func (m *MockPersonRepository) Create(ctx context.Context, input domain.PersonInput) (domain.PersonID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, input)
	ret0, _ := ret[0].(domain.PersonID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockPersonRepositoryMockRecorder) Create(ctx, input any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockPersonRepository)(nil).Create), ctx, input)
}

func (m *MockPersonRepository) Delete(ctx context.Context, id domain.PersonID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockPersonRepositoryMockRecorder) Delete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockPersonRepository)(nil).Delete), ctx, id)
}

func (m *MockPersonRepository) GetByID(ctx context.Context, id domain.PersonID) (*domain.Person, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, id)
	ret0, _ := ret[0].(*domain.Person)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockPersonRepositoryMockRecorder) GetByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockPersonRepository)(nil).GetByID), ctx, id)
}

func (m *MockPersonRepository) List(ctx context.Context, limit, offset int) ([]domain.Person, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, limit, offset)
	ret0, _ := ret[0].([]domain.Person)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockPersonRepositoryMockRecorder) List(ctx, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockPersonRepository)(nil).List), ctx, limit, offset)
}

func (m *MockPersonRepository) Update(ctx context.Context, id domain.PersonID, input domain.PersonInput) (*domain.Person, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, id, input)
	ret0, _ := ret[0].(*domain.Person)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockPersonRepositoryMockRecorder) Update(ctx, id, input any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockPersonRepository)(nil).Update), ctx, id, input)
}

type MockReviewRepository struct {
	ctrl     *gomock.Controller
	recorder *MockReviewRepositoryMockRecorder
	isgomock struct{}
}

type MockReviewRepositoryMockRecorder struct {
	mock *MockReviewRepository
}

func NewMockReviewRepository(ctrl *gomock.Controller) *MockReviewRepository {
	mock := &MockReviewRepository{ctrl: ctrl}
	mock.recorder = &MockReviewRepositoryMockRecorder{mock}
	return mock
}

func (m *MockReviewRepository) EXPECT() *MockReviewRepositoryMockRecorder {
	return m.recorder
}

func (m *MockReviewRepository) Create(ctx context.Context, authorID domain.UserID, filmID domain.FilmID, input domain.ReviewInput) (*domain.Review, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, authorID, filmID, input)
	ret0, _ := ret[0].(*domain.Review)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockReviewRepositoryMockRecorder) Create(ctx, authorID, filmID, input any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockReviewRepository)(nil).Create), ctx, authorID, filmID, input)
}

func (m *MockReviewRepository) Delete(ctx context.Context, id domain.ReviewID, deletedBy domain.UserID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id, deletedBy)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockReviewRepositoryMockRecorder) Delete(ctx, id, deletedBy any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockReviewRepository)(nil).Delete), ctx, id, deletedBy)
}

func (m *MockReviewRepository) GetByID(ctx context.Context, id domain.ReviewID) (*domain.Review, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, id)
	ret0, _ := ret[0].(*domain.Review)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockReviewRepositoryMockRecorder) GetByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockReviewRepository)(nil).GetByID), ctx, id)
}

func (m *MockReviewRepository) ListByFilm(ctx context.Context, filmID domain.FilmID, limit, offset int) ([]domain.Review, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListByFilm", ctx, filmID, limit, offset)
	ret0, _ := ret[0].([]domain.Review)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockReviewRepositoryMockRecorder) ListByFilm(ctx, filmID, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListByFilm", reflect.TypeOf((*MockReviewRepository)(nil).ListByFilm), ctx, filmID, limit, offset)
}

func (m *MockReviewRepository) Update(ctx context.Context, id domain.ReviewID, changes domain.ReviewUpdate) (*domain.Review, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, id, changes)
	ret0, _ := ret[0].(*domain.Review)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockReviewRepositoryMockRecorder) Update(ctx, id, changes any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockReviewRepository)(nil).Update), ctx, id, changes)
}

type MockAvatarRepository struct {
	ctrl     *gomock.Controller
	recorder *MockAvatarRepositoryMockRecorder
	isgomock struct{}
}

type MockAvatarRepositoryMockRecorder struct {
	mock *MockAvatarRepository
}

func NewMockAvatarRepository(ctrl *gomock.Controller) *MockAvatarRepository {
	mock := &MockAvatarRepository{ctrl: ctrl}
	mock.recorder = &MockAvatarRepositoryMockRecorder{mock}
	return mock
}

func (m *MockAvatarRepository) EXPECT() *MockAvatarRepositoryMockRecorder {
	return m.recorder
}

func (m *MockAvatarRepository) DeleteAvatar(ctx context.Context, userID domain.UserID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAvatar", ctx, userID)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockAvatarRepositoryMockRecorder) DeleteAvatar(ctx, userID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAvatar", reflect.TypeOf((*MockAvatarRepository)(nil).DeleteAvatar), ctx, userID)
}

func (m *MockAvatarRepository) GetAvatar(ctx context.Context, userID domain.UserID) (*domain.AvatarFile, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAvatar", ctx, userID)
	ret0, _ := ret[0].(*domain.AvatarFile)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockAvatarRepositoryMockRecorder) GetAvatar(ctx, userID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAvatar", reflect.TypeOf((*MockAvatarRepository)(nil).GetAvatar), ctx, userID)
}

func (m *MockAvatarRepository) SaveAvatar(ctx context.Context, userID domain.UserID, file domain.AvatarFile) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveAvatar", ctx, userID, file)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockAvatarRepositoryMockRecorder) SaveAvatar(ctx, userID, file any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveAvatar", reflect.TypeOf((*MockAvatarRepository)(nil).SaveAvatar), ctx, userID, file)
}

type MockPosterRepository struct {
	ctrl     *gomock.Controller
	recorder *MockPosterRepositoryMockRecorder
	isgomock struct{}
}

type MockPosterRepositoryMockRecorder struct {
	mock *MockPosterRepository
}

func NewMockPosterRepository(ctrl *gomock.Controller) *MockPosterRepository {
	mock := &MockPosterRepository{ctrl: ctrl}
	mock.recorder = &MockPosterRepositoryMockRecorder{mock}
	return mock
}

func (m *MockPosterRepository) EXPECT() *MockPosterRepositoryMockRecorder {
	return m.recorder
}

func (m *MockPosterRepository) SavePoster(ctx context.Context, filmID domain.FilmID, userID domain.UserID, poster domain.PosterFile) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SavePoster", ctx, filmID, userID, poster)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockPosterRepositoryMockRecorder) SavePoster(ctx, filmID, userID, poster any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SavePoster", reflect.TypeOf((*MockPosterRepository)(nil).SavePoster), ctx, filmID, userID, poster)
}

type MockLibraryRepository struct {
	ctrl     *gomock.Controller
	recorder *MockLibraryRepositoryMockRecorder
	isgomock struct{}
}

type MockLibraryRepositoryMockRecorder struct {
	mock *MockLibraryRepository
}

func NewMockLibraryRepository(ctrl *gomock.Controller) *MockLibraryRepository {
	mock := &MockLibraryRepository{ctrl: ctrl}
	mock.recorder = &MockLibraryRepositoryMockRecorder{mock}
	return mock
}

func (m *MockLibraryRepository) EXPECT() *MockLibraryRepositoryMockRecorder {
	return m.recorder
}

func (m *MockLibraryRepository) Get(ctx context.Context, user domain.UserID) (domain.Library, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, user)
	ret0, _ := ret[0].(domain.Library)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockLibraryRepositoryMockRecorder) Get(ctx, user any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockLibraryRepository)(nil).Get), ctx, user)
}

func (m *MockLibraryRepository) Preferences(ctx context.Context, user domain.UserID) (json.RawMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Preferences", ctx, user)
	ret0, _ := ret[0].(json.RawMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockLibraryRepositoryMockRecorder) Preferences(ctx, user any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Preferences", reflect.TypeOf((*MockLibraryRepository)(nil).Preferences), ctx, user)
}

func (m *MockLibraryRepository) SavePreference(ctx context.Context, user domain.UserID, key string, value json.RawMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SavePreference", ctx, user, key, value)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockLibraryRepositoryMockRecorder) SavePreference(ctx, user, key, value any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SavePreference", reflect.TypeOf((*MockLibraryRepository)(nil).SavePreference), ctx, user, key, value)
}

func (m *MockLibraryRepository) SetSaved(ctx context.Context, user domain.UserID, list domain.List, film int64, saved bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetSaved", ctx, user, list, film, saved)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockLibraryRepositoryMockRecorder) SetSaved(ctx, user, list, film, saved any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSaved", reflect.TypeOf((*MockLibraryRepository)(nil).SetSaved), ctx, user, list, film, saved)
}

type MockPersonCatalogRepository struct {
	ctrl     *gomock.Controller
	recorder *MockPersonCatalogRepositoryMockRecorder
	isgomock struct{}
}

type MockPersonCatalogRepositoryMockRecorder struct {
	mock *MockPersonCatalogRepository
}

func NewMockPersonCatalogRepository(ctrl *gomock.Controller) *MockPersonCatalogRepository {
	mock := &MockPersonCatalogRepository{ctrl: ctrl}
	mock.recorder = &MockPersonCatalogRepositoryMockRecorder{mock}
	return mock
}

func (m *MockPersonCatalogRepository) EXPECT() *MockPersonCatalogRepositoryMockRecorder {
	return m.recorder
}

func (m *MockPersonCatalogRepository) FilmPersons(arg0 context.Context, arg1 int64) ([]domain.PersonCredit, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FilmPersons", arg0, arg1)
	ret0, _ := ret[0].([]domain.PersonCredit)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockPersonCatalogRepositoryMockRecorder) FilmPersons(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FilmPersons", reflect.TypeOf((*MockPersonCatalogRepository)(nil).FilmPersons), arg0, arg1)
}

func (m *MockPersonCatalogRepository) Get(arg0 context.Context, arg1 int64) (domain.CatalogPerson, []domain.FilmCredit, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(domain.CatalogPerson)
	ret1, _ := ret[1].([]domain.FilmCredit)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockPersonCatalogRepositoryMockRecorder) Get(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockPersonCatalogRepository)(nil).Get), arg0, arg1)
}
