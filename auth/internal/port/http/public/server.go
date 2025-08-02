package public

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/auth/internal/port"
	"github.com/parta4ok/kvs/auth/pkg/dto"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	"github.com/pkg/errors"
)

const (
	basePath       = "/auth/v1"
	signinPath     = "/signin"
	addUserPath    = "/add-user"
	deleteUserPath = "/delete-user"

	right_admin = "admin"
)

type Server struct {
	router   *chi.Mux
	server   *http.Server
	factory  port.CommandFactory
	accessor Accessor
	cfg      *ServerCfg
}

type ServerCfg struct {
	Port    string
	Timeout time.Duration
}

type ServerOption func(*Server)

func WithFactory(factory port.CommandFactory) ServerOption {
	return func(s *Server) {
		s.factory = factory
	}
}

func WithAccessor(accessor Accessor) ServerOption {
	return func(s *Server) {
		s.accessor = accessor
	}
}

func WithConfig(cfg *ServerCfg) ServerOption {
	return func(s *Server) {
		s.cfg = cfg
	}
}

func (s *Server) setOption(opts ...ServerOption) {
	for _, opt := range opts {
		opt(s)
	}
}

func New(opts ...ServerOption) (*Server, error) {
	r := chi.NewMux()

	serv := &Server{
		router: r,
	}

	serv.setOption(opts...)

	if serv.factory == nil {
		err := errors.Wrap(entities.ErrInternal, "factory not set")
		slog.Error(err.Error())
		return nil, err
	}

	if serv.accessor == nil {
		err := errors.Wrap(entities.ErrInternal, "accessor not set")
		slog.Error(err.Error())
		return nil, err
	}

	if serv.cfg == nil {
		err := errors.Wrap(entities.ErrInvalidParam, "config not set")
		slog.Error(err.Error())
		return nil, err
	}

	if serv.cfg.Port == "" {
		err := errors.Wrap(entities.ErrInternal, "port not set")
		slog.Error(err.Error())
		return nil, err
	}

	return serv, nil
}

func (s *Server) Start() {
	slog.Info("public port started")
	s.registerRoutes()

	s.server = &http.Server{
		Addr:              s.cfg.Port,
		Handler:           s.router,
		ReadHeaderTimeout: s.cfg.Timeout,
		WriteTimeout:      s.cfg.Timeout,
		IdleTimeout:       s.cfg.Timeout,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error(err.Error())
		}
	}()

	<-done

	s.Stop()
}

func (s *Server) Stop() {
	slog.Info("server will be stopping")

	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*2)
	defer cancelFn()

	if err := s.server.Shutdown(ctx); err != nil {
		slog.Error(errors.Wrapf(entities.ErrInternal, "shutdown err: %v", err).Error())
	}

	slog.Info("server stop gracefully")
}

func (s *Server) registerRoutes() {
	s.router.Use(s.timeoutMiddleware)

	s.router.Post(basePath+signinPath, s.Signin)
	s.router.Put(basePath+addUserPath, s.AddUser)
	s.router.Route(basePath, func(r chi.Router) {
		r.Delete(deleteUserPath+"/{user_id}", s.DeleteUser)
	})
}

// Sign in user
//
// @Summary      Sign in
// @Description  Authenticates user with provided credentials and returns JWT token
// @Accept       json
// @Produce      json
// @Param        request body dto.SigninRequestDTO true "User credentials"
// @Success      201  {object}  dto.SigninResponseDTO "JWT created"
// @Failure      400  {object}  dto.ErrorDTO "Invalid request parameters"
// @Failure      401  {object}  dto.ErrorDTO "Unauthorized"
// @Failure      500  {object}  dto.ErrorDTO "Internal server error"
// @Router       /auth/v1/signin [post]
//
//nolint:funlen //ok
func (s *Server) Signin(resp http.ResponseWriter, req *http.Request) {
	slog.Info("Signin started")
	resp.Header().Set("Content-Type", "application/json")

	var requestDTO dto.SigninRequestDTO
	if err := json.NewDecoder(req.Body).Decode(&requestDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam,
			"decode req body to signinRequestDTO failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	command, err := s.factory.NewSignInCommand(req.Context(), requestDTO.Login,
		requestDTO.Password)
	if err != nil {
		err := errors.Wrap(err, "signin command creating failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	res, err := command.Exec()
	if err != nil {
		err := errors.Wrap(err, "signin command executing failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	if res == nil {
		err := errors.Wrap(entities.ErrInternal, "signin command executing completed with nil result")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	if !res.Success {
		err := errors.Wrap(entities.ErrInternal, "signin command executing completed with bad status")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	signinDTO := &dto.SigninResponseDTO{Token: res.Message}

	data, err := json.Marshal(signinDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal token failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusCreated)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}
}

// Add new user
//
// @Summary      Add new user
// @Description  Add new user with selected credentials and other user info
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Param        request body dto.AddUserDTO true "User credentials and other data"
// @Success      201  {object}  dto.AddUserResponseDTO "New user created"
// @Failure      400  {object}  dto.ErrorDTO "Invalid request parameters"
// @Failure      401  {object}  dto.ErrorDTO "Unauthorized"
// @Failure		 403  {object}  dto.ErrorDTO "Forbidden"
// @Failure		 409  {object}  dto.ErrorDTO "Conflict"
// @Failure      500  {object}  dto.ErrorDTO "Internal server error"
// @Router       /auth/v1/add-user [put]
//
//nolint:funlen //ok
func (s *Server) AddUser(resp http.ResponseWriter, req *http.Request) {
	slog.Info("AddUser started")
	resp.Header().Set("Content-Type", "application/json")

	if err := s.getValidatedAuthContext(resp, req, []string{right_admin}); err != nil {
		err := errors.Wrap(err, "getValidatedAuthContext")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	var requestDTO dto.AddUserDTO
	if err := json.NewDecoder(req.Body).Decode(&requestDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam,
			"decode req body to requestDTO failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	addUserCommand, err := s.factory.NewAddUserCommand(req.Context(), requestDTO.Username,
		requestDTO.Password, requestDTO.Rights, requestDTO.Contacts)
	if err != nil {
		err := errors.Wrap(err, "new add user failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	addUserResult, err := addUserCommand.Exec()
	if err != nil {
		err := errors.Wrap(err, "add user command failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	if !addUserResult.Success {
		err := errors.Wrap(entities.ErrInternal, "add user failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	responseDTO := &dto.AddUserResponseDTO{
		UserID: addUserResult.Message,
	}

	data, err := json.Marshal(responseDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal response failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusCreated)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}
}

// Delete user by ID
//
// @Summary      Delete user
// @Description  Delete existing user by ID. Requires admin rights.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Param        user_id path string true "User ID to delete"
// @Success      204 "User successfully deleted"
// @Failure      400 {object} dto.ErrorDTO "Invalid request parameters"
// @Failure      401 {object} dto.ErrorDTO "Unauthorized"
// @Failure      403 {object} dto.ErrorDTO "Forbidden"
// @Failure      404 {object} dto.ErrorDTO "User not found"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /auth/v1/delete-user/{user_id} [delete]
//
//nolint:funlen //ok
func (s *Server) DeleteUser(resp http.ResponseWriter, req *http.Request) {
	slog.Info("DeleteUser started")
	resp.Header().Set("Content-Type", "application/json")

	if err := s.getValidatedAuthContext(resp, req, []string{right_admin}); err != nil {
		err := errors.Wrap(err, "getValidatedAuthContext")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	userID := chi.URLParam(req, "user_id")

	if userID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "userID invalid")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	deleteUserCommand, err := s.factory.NewDeleteUserCommand(req.Context(), userID)
	if err != nil {
		err := errors.Wrap(err, "new delete user failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	deleteUserResult, err := deleteUserCommand.Exec()
	if err != nil {
		err := errors.Wrap(err, "delete user command failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	if !deleteUserResult.Success {
		err := errors.Wrap(entities.ErrInternal, "add user failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusNoContent)
}

func (s *Server) errProcessing(resp http.ResponseWriter, err error) {
	stausCode := http.StatusInternalServerError
	errDTO := dto.ErrorDTO{
		StatusCode: stausCode,
		ErrMsg:     err.Error(),
	}

	switch {
	case errors.Is(err, entities.ErrInvalidParam):
		errDTO.StatusCode = http.StatusBadRequest
	case errors.Is(err, entities.ErrForbidden):
		errDTO.StatusCode = http.StatusForbidden
	case errors.Is(err, entities.ErrNotFound):
		errDTO.StatusCode = http.StatusNotFound
	case errors.Is(err, entities.ErrAlreadyExists):
		errDTO.StatusCode = http.StatusConflict
	}

	errDtoData, err := json.Marshal(&errDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal failure: %v", err)
		slog.Error(err.Error())
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(errDTO.StatusCode)
	resp.Write(errDtoData) //nolint:errcheck,gosec //ok
}

func (s *Server) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), s.cfg.Timeout)
		defer cancel()

		req = req.WithContext(ctx)
		next.ServeHTTP(resp, req)
	})
}

func (s *Server) checkUserRights(ctx context.Context, requiredRights []string) error {
	hasEnoughRights, err := s.accessor.HasPermission(ctx, requiredRights)
	if err != nil {
		return err
	}

	if !hasEnoughRights {
		err := errors.Wrap(entities.ErrForbidden, "user has not enough rights")
		return err
	}

	return nil
}

func (s *Server) getValidatedAuthContext(resp http.ResponseWriter, req *http.Request,
	rights []string) error {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		err := errors.Wrap(entities.ErrForbidden, "authoriztion header not set")
		slog.Error(err.Error())
		return err
	}

	const prefix = "Bearer "
	authorizationData := strings.Split(authHeader, prefix)
	if len(authorizationData) != 2 {
		err := errors.Wrap(entities.ErrForbidden, "authoriztion header invalid")
		slog.Error(err.Error())
		return err
	}

	jwt := authorizationData[1]
	introspectCommand, err := s.factory.NewIntrospectedCommand(req.Context(), jwt)
	if err != nil {
		err := errors.Wrap(err, "new inrospect failure")
		slog.Error(err.Error())
		return err
	}

	introspectResult, err := introspectCommand.Exec()
	if err != nil {
		err := errors.Wrap(err, "inrospection failure")
		slog.Error(err.Error())
		return err
	}

	if !introspectResult.Success {
		err := errors.Wrap(entities.ErrForbidden, "operation forbidden")
		slog.Error(err.Error())
		return err
	}

	claims, ok := introspectResult.Payload.(*entities.UserClaims)
	if !ok {
		err := errors.Wrap(entities.ErrForbidden, "assertion of user claims failure")
		slog.Error(err.Error())
		return err
	}

	ctx := context.WithValue(req.Context(), accessor.UserClaims, &accessor.Claims{
		Username: claims.Username,
		Issuer:   claims.Issuer,
		Subject:  claims.Subject,
		Audience: claims.Audience,
		Rights:   claims.Rights,
	})

	if err := s.checkUserRights(ctx, rights); err != nil {
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return err
	}

	return nil
}
