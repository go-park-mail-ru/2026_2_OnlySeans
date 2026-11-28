package onlineconf

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	productrouter "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	AdminBasePath            = "/api/onlineconf/admin"
	InternalBasePath         = "/internal/onlineconf"
	healthPath               = "/healthz"
	mePath                   = AdminBasePath + "/me"
	scopesPath               = AdminBasePath + "/scopes"
	scopePath                = scopesPath + "/{id}"
	configsPath              = AdminBasePath + "/configs"
	configPath               = configsPath + "/{id}"
	flagsPath                = AdminBasePath + "/flags"
	flagPath                 = flagsPath + "/{id}"
	evaluatePath             = flagPath + "/evaluate"
	auditPath                = AdminBasePath + "/audit"
	snapshotPath             = InternalBasePath + "/snapshot"
	contentTypeOptionsHeader = "X-Content-Type-Options"
	noSniffValue             = "nosniff"
	healthyStatus            = "ok"
)

type HandlerUseCases struct {
	Scopes   *usecase.ScopesUseCase
	Configs  *usecase.ConfigsUseCase
	Flags    *usecase.FlagsUseCase
	Evaluate *usecase.EvaluateFlagUseCase
	Audit    *usecase.ListAuditUseCase
}

type Handler struct {
	scopes    *usecase.ScopesUseCase
	configs   *usecase.ConfigsUseCase
	flags     *usecase.FlagsUseCase
	evaluate  *usecase.EvaluateFlagUseCase
	audit     *usecase.ListAuditUseCase
	snapshots usecase.SnapshotProvider
	appLogger logger.Logger
}

func NewRouter(cases HandlerUseCases, middleware *Middleware, snapshots usecase.SnapshotProvider, appLogger logger.Logger) (http.Handler, error) {
	if cases.Scopes == nil || cases.Configs == nil || cases.Flags == nil || cases.Evaluate == nil || cases.Audit == nil {
		return nil, errors.New("onlineconf handler use cases are required")
	}

	if middleware == nil {
		return nil, errors.New("onlineconf middleware is required")
	}

	if snapshots == nil {
		return nil, oc.ErrNilSnapshotProvider
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	h := &Handler{scopes: cases.Scopes, configs: cases.Configs, flags: cases.Flags, evaluate: cases.Evaluate, audit: cases.Audit, snapshots: snapshots, appLogger: appLogger}
	admin := mux.NewRouter()
	admin.NotFoundHandler = http.HandlerFunc(h.notFound)
	admin.MethodNotAllowedHandler = http.HandlerFunc(h.methodNotAllowed)
	admin.HandleFunc(mePath, h.me).Methods(http.MethodGet)
	admin.HandleFunc(scopesPath, h.listScopes).Methods(http.MethodGet)
	admin.HandleFunc(scopesPath, h.createScope).Methods(http.MethodPost)
	admin.HandleFunc(scopePath, h.getScope).Methods(http.MethodGet)
	admin.HandleFunc(scopePath, h.updateScope).Methods(http.MethodPut)
	admin.HandleFunc(scopePath, h.deleteScope).Methods(http.MethodDelete)
	admin.HandleFunc(configsPath, h.listConfigs).Methods(http.MethodGet)
	admin.HandleFunc(configsPath, h.createConfig).Methods(http.MethodPost)
	admin.HandleFunc(configPath, h.getConfig).Methods(http.MethodGet)
	admin.HandleFunc(configPath, h.updateConfig).Methods(http.MethodPut)
	admin.HandleFunc(configPath, h.deleteConfig).Methods(http.MethodDelete)
	admin.HandleFunc(flagsPath, h.listFlags).Methods(http.MethodGet)
	admin.HandleFunc(flagsPath, h.createFlag).Methods(http.MethodPost)
	admin.HandleFunc(flagPath, h.getFlag).Methods(http.MethodGet)
	admin.HandleFunc(flagPath, h.updateFlag).Methods(http.MethodPut)
	admin.HandleFunc(flagPath, h.deleteFlag).Methods(http.MethodDelete)
	admin.HandleFunc(evaluatePath, h.evaluateFlag).Methods(http.MethodPost)
	admin.HandleFunc(auditPath, h.listAudit).Methods(http.MethodGet)

	internal := mux.NewRouter()
	internal.NotFoundHandler = http.HandlerFunc(h.notFound)
	internal.MethodNotAllowedHandler = http.HandlerFunc(h.methodNotAllowed)
	internal.HandleFunc(snapshotPath, h.snapshot).Methods(http.MethodGet)

	root := mux.NewRouter()
	root.NotFoundHandler = http.HandlerFunc(h.notFound)
	root.MethodNotAllowedHandler = http.HandlerFunc(h.methodNotAllowed)
	root.Handle(AdminBasePath, middleware.RequireAdmin(admin))
	root.PathPrefix(AdminBasePath + "/").Handler(middleware.RequireAdmin(admin))
	root.Handle(InternalBasePath, middleware.RequireService(internal))
	root.PathPrefix(InternalBasePath + "/").Handler(middleware.RequireService(internal))
	root.HandleFunc(healthPath, h.health).Methods(http.MethodGet)
	wrapped := productrouter.WithLogging(productrouter.WithRecover(root, appLogger), appLogger)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeOptionsHeader, noSniffValue)
		w.Header().Set(cacheControlHeader, privateNoStore)
		wrapped.ServeHTTP(w, r)
	}), nil
}
