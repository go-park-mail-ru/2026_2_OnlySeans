package persons_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	persondelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	personusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/gorilla/mux"
)

// Тестовый логгер. Ничего не записывает.
type handlerTestLogger struct{}

func (l *handlerTestLogger) Info(string, map[string]any)  {}
func (l *handlerTestLogger) Debug(string, map[string]any) {}
func (l *handlerTestLogger) Warn(string, map[string]any)  {}
func (l *handlerTestLogger) Error(string, map[string]any) {}

// Тестовый репозиторий.
type handlerTestRepository struct {
	createFn func(context.Context, domain.PersonInput) (domain.PersonID, error)
	getFn    func(context.Context, domain.PersonID) (*domain.Person, error)
	listFn   func(context.Context, int, int) ([]domain.Person, int, error)
	updateFn func(context.Context, domain.PersonID, domain.PersonInput) (*domain.Person, error)
	deleteFn func(context.Context, domain.PersonID) error
}

func (r *handlerTestRepository) Create(
	ctx context.Context,
	input domain.PersonInput,
) (domain.PersonID, error) {
	if r.createFn != nil {
		return r.createFn(ctx, input)
	}
	return 0, nil
}

func (r *handlerTestRepository) GetByID(
	ctx context.Context,
	id domain.PersonID,
) (*domain.Person, error) {
	if r.getFn != nil {
		return r.getFn(ctx, id)
	}
	return nil, nil
}

func (r *handlerTestRepository) List(
	ctx context.Context,
	limit, offset int,
) ([]domain.Person, int, error) {
	if r.listFn != nil {
		return r.listFn(ctx, limit, offset)
	}
	return nil, 0, nil
}

func (r *handlerTestRepository) Update(
	ctx context.Context,
	id domain.PersonID,
	input domain.PersonInput,
) (*domain.Person, error) {
	if r.updateFn != nil {
		return r.updateFn(ctx, id, input)
	}
	return nil, nil
}

func (r *handlerTestRepository) Delete(
	ctx context.Context,
	id domain.PersonID,
) error {
	if r.deleteFn != nil {
		return r.deleteFn(ctx, id)
	}
	return nil
}

// Создаём хэндлер со всеми необходимыми usecase.
func newHandlerTest(t *testing.T, repo domain.PersonRepository) *persondelivery.Handler {
	t.Helper()

	appLogger := logger.Logger(&handlerTestLogger{})

	create, err := personusecase.NewCreatePersonUseCase(repo, appLogger)
	if err != nil {
		t.Fatalf("failed to create create usecase: %v", err)
	}

	get, err := personusecase.NewGetPersonUseCase(repo, appLogger)
	if err != nil {
		t.Fatalf("failed to create get usecase: %v", err)
	}

	list, err := personusecase.NewListPersonsUseCase(repo, appLogger)
	if err != nil {
		t.Fatalf("failed to create list usecase: %v", err)
	}

	update, err := personusecase.NewUpdatePersonUseCase(repo, appLogger)
	if err != nil {
		t.Fatalf("failed to create update usecase: %v", err)
	}

	deletePerson, err := personusecase.NewDeletePersonUseCase(repo, appLogger)
	if err != nil {
		t.Fatalf("failed to create delete usecase: %v", err)
	}

	handler, err := persondelivery.NewHandler(persondelivery.HandlerUseCases{
		CreatePerson: create,
		GetPerson:    get,
		ListPersons:  list,
		UpdatePerson: update,
		DeletePerson: deletePerson,
	}, appLogger)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	return handler
}

func newHandlerTestPerson() *domain.Person {
	now := time.Now()

	return &domain.Person{
		ID:        1,
		FirstName: "Ada",
		LastName:  "Lovelace",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newHandlerTestRequest(method, url, body string) *http.Request {
	return httptest.NewRequest(method, url, strings.NewReader(body))
}

// Устанавливаем ID в запросе, как это делает gorilla/mux.
func withPersonID(r *http.Request, id string) *http.Request {
	return mux.SetURLVars(r, map[string]string{
		"id": id,
	})
}

// Проверяем HTTP-статус и наличие нужного текста в ответе.
func checkHandlerResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	wantStatus int,
	wantBody string,
) {
	t.Helper()

	if recorder.Code != wantStatus {
		t.Errorf(
			"expected status %d, got %d; body: %s",
			wantStatus,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if wantBody != "" && !strings.Contains(recorder.Body.String(), wantBody) {
		t.Errorf(
			"expected response body to contain %q, got %s",
			wantBody,
			recorder.Body.String(),
		)
	}
}

// CREATE

func TestCreatePersonHandler(t *testing.T) {
	repoErr := errors.New("database error")

	tests := []struct {
		name       string
		body       string
		createFn   func(context.Context, domain.PersonInput) (domain.PersonID, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			body: `{
				"first_name": "Ada",
				"last_name": "Lovelace",
				"birth_date": "1990-01-01"
			}`,
			createFn: func(_ context.Context, input domain.PersonInput) (domain.PersonID, error) {
				return 1, nil
			},
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":1}`,
		},
		{
			name:       "invalid JSON",
			body:       `{"first_name":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "unknown JSON field",
			body: `{
				"first_name": "Ada",
				"last_name": "Lovelace",
				"unknown": "value"
			}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid birth date",
			body: `{
				"first_name": "Ada",
				"last_name": "Lovelace",
				"birth_date": "wrong-date"
			}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid first name",
			body: `{
				"first_name": "",
				"last_name": "Lovelace"
			}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "repository error",
			body: `{
				"first_name": "Ada",
				"last_name": "Lovelace"
			}`,
			createFn: func(_ context.Context, _ domain.PersonInput) (domain.PersonID, error) {
				return 0, repoErr
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &handlerTestRepository{
				createFn: tt.createFn,
			}

			handler := newHandlerTest(t, repo)

			request := newHandlerTestRequest(
				http.MethodPost,
				"/api/persons",
				tt.body,
			)
			recorder := httptest.NewRecorder()

			handler.CreatePerson(recorder, request)

			checkHandlerResponse(t, recorder, tt.wantStatus, tt.wantBody)
			if tt.wantStatus == http.StatusCreated {
				if strings.TrimSpace(recorder.Body.String()) != tt.wantBody {
					t.Errorf("unexpected creation response: %s", recorder.Body.String())
				}
			}
		})
	}
}

// GET

func TestGetPersonHandler(t *testing.T) {
	repoErr := errors.New("database error")

	tests := []struct {
		name       string
		id         string
		getFn      func(context.Context, domain.PersonID) (*domain.Person, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			id:   "1",
			getFn: func(_ context.Context, id domain.PersonID) (*domain.Person, error) {
				return newHandlerTestPerson(), nil
			},
			wantStatus: http.StatusOK,
			wantBody:   `"first_name":"Ada"`,
		},
		{
			name:       "invalid ID",
			id:         "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero ID",
			id:         "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "person not found",
			id:   "1",
			getFn: func(_ context.Context, _ domain.PersonID) (*domain.Person, error) {
				return nil, domain.ErrPersonNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "repository error",
			id:   "1",
			getFn: func(_ context.Context, _ domain.PersonID) (*domain.Person, error) {
				return nil, repoErr
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &handlerTestRepository{
				getFn: tt.getFn,
			}

			handler := newHandlerTest(t, repo)

			request := newHandlerTestRequest(
				http.MethodGet,
				"/api/persons/"+tt.id,
				"",
			)
			request = withPersonID(request, tt.id)

			recorder := httptest.NewRecorder()

			handler.GetPerson(recorder, request)

			checkHandlerResponse(t, recorder, tt.wantStatus, tt.wantBody)
		})
	}
}

// LIST

func TestListPersonsHandler(t *testing.T) {
	repoErr := errors.New("database error")

	tests := []struct {
		name       string
		url        string
		listFn     func(context.Context, int, int) ([]domain.Person, int, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			url:  "/api/persons?limit=10&offset=5",
			listFn: func(_ context.Context, limit, offset int) ([]domain.Person, int, error) {
				if limit != 10 || offset != 5 {
					t.Errorf("expected limit=10 and offset=5, got limit=%d offset=%d", limit, offset)
				}

				return []domain.Person{*newHandlerTestPerson()}, 1, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   `"total":1`,
		},
		{
			name: "default pagination",
			url:  "/api/persons",
			listFn: func(_ context.Context, limit, offset int) ([]domain.Person, int, error) {
				if limit != 20 || offset != 0 {
					t.Errorf("expected limit=20 and offset=0, got limit=%d offset=%d", limit, offset)
				}

				return []domain.Person{}, 0, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   `"limit":20`,
		},
		{
			name:       "invalid limit",
			url:        "/api/persons?limit=abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "limit is too large",
			url:        "/api/persons?limit=101",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid offset",
			url:        "/api/persons?offset=-1",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "repository error",
			url:  "/api/persons",
			listFn: func(_ context.Context, _, _ int) ([]domain.Person, int, error) {
				return nil, 0, repoErr
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &handlerTestRepository{
				listFn: tt.listFn,
			}

			handler := newHandlerTest(t, repo)

			request := newHandlerTestRequest(
				http.MethodGet,
				tt.url,
				"",
			)
			recorder := httptest.NewRecorder()

			handler.ListPersons(recorder, request)

			checkHandlerResponse(t, recorder, tt.wantStatus, tt.wantBody)
		})
	}
}

// UPDATE

func TestUpdatePersonHandler(t *testing.T) {
	repoErr := errors.New("database error")

	tests := []struct {
		name       string
		id         string
		body       string
		updateFn   func(context.Context, domain.PersonID, domain.PersonInput) (*domain.Person, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			id:   "1",
			body: `{
				"first_name": "Grace",
				"last_name": "Hopper",
				"birth_date": "1990-01-01"
			}`,
			updateFn: func(_ context.Context, id domain.PersonID, input domain.PersonInput) (*domain.Person, error) {
				return &domain.Person{
					ID:        id,
					FirstName: input.FirstName,
					LastName:  input.LastName,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   `"first_name":"Grace"`,
		},
		{
			name:       "invalid ID",
			id:         "abc",
			body:       `{"first_name":"Grace","last_name":"Hopper"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			id:         "1",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid birth date",
			id:   "1",
			body: `{
				"first_name": "Grace",
				"last_name": "Hopper",
				"birth_date": "wrong-date"
			}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "person not found",
			id:   "1",
			body: `{"first_name":"Grace","last_name":"Hopper"}`,
			updateFn: func(_ context.Context, _ domain.PersonID, _ domain.PersonInput) (*domain.Person, error) {
				return nil, domain.ErrPersonNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "repository error",
			id:   "1",
			body: `{"first_name":"Grace","last_name":"Hopper"}`,
			updateFn: func(_ context.Context, _ domain.PersonID, _ domain.PersonInput) (*domain.Person, error) {
				return nil, repoErr
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &handlerTestRepository{
				updateFn: tt.updateFn,
			}

			handler := newHandlerTest(t, repo)

			request := newHandlerTestRequest(
				http.MethodPut,
				"/api/persons/"+tt.id,
				tt.body,
			)
			request = withPersonID(request, tt.id)

			recorder := httptest.NewRecorder()

			handler.UpdatePerson(recorder, request)

			checkHandlerResponse(t, recorder, tt.wantStatus, tt.wantBody)
		})
	}
}

// DELETE

func TestDeletePersonHandler(t *testing.T) {
	repoErr := errors.New("database error")

	tests := []struct {
		name       string
		id         string
		deleteFn   func(context.Context, domain.PersonID) error
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			id:   "1",
			deleteFn: func(_ context.Context, _ domain.PersonID) error {
				return nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid ID",
			id:         "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "person not found",
			id:   "1",
			deleteFn: func(_ context.Context, _ domain.PersonID) error {
				return domain.ErrPersonNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "repository error",
			id:   "1",
			deleteFn: func(_ context.Context, _ domain.PersonID) error {
				return repoErr
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &handlerTestRepository{
				deleteFn: tt.deleteFn,
			}

			handler := newHandlerTest(t, repo)

			request := newHandlerTestRequest(
				http.MethodDelete,
				"/api/persons/"+tt.id,
				"",
			)
			request = withPersonID(request, tt.id)

			recorder := httptest.NewRecorder()

			handler.DeletePerson(recorder, request)

			checkHandlerResponse(t, recorder, tt.wantStatus, tt.wantBody)
		})
	}
}
