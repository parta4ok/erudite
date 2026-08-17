package public

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/auth/internal/port"
	"github.com/parta4ok/kvs/auth/pkg/dto"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	httpport "github.com/parta4ok/kvs/toolkit/pkg/port/http"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
	"github.com/pkg/errors"
)

const (
	basePath                = "/auth/v1"
	signinPath              = "/signin"
	addUserPath             = "/add-user"
	deleteUserPath          = "/delete-user"
	updateUserPath          = "/update-user"
	addGroupPath            = "/add-group"
	getMentorGroupsPath     = "/mentor-groups"
	dynamicregistrationPath = "/dynamic-registration"
	listUsersPath           = "/users"
	listGroupsPath          = "/groups"

	right_admin  = "admin"
	right_mentor = "mentor"

	PortType = "http_public_auth"
)

type Server struct {
	factory  port.CommandFactory
	accessor Accessor
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

func (s *Server) setOption(opts ...ServerOption) {
	for _, opt := range opts {
		opt(s)
	}
}

func New(opts ...ServerOption) (*Server, error) {
	serv := &Server{}

	serv.setOption(opts...)

	if serv.factory == nil {
		err := errors.Wrap(entities.ErrInternal, "factory not set")
		slog.Error("factory not set", "error", err.Error())
		return nil, err
	}

	if serv.accessor == nil {
		err := errors.Wrap(entities.ErrInternal, "accessor not set")
		slog.Error("accessor not set", "error", err.Error())
		return nil, err
	}

	return serv, nil
}

func (s *Server) Routes() []httpport.Route {
	return []httpport.Route{
		{
			Method:  http.MethodPost,
			Pattern: basePath + signinPath, Handler: s.Signin,
		},
		{
			Method:  http.MethodPut,
			Pattern: basePath + addUserPath, Handler: s.AddUser,
		},
		{
			Method:  http.MethodPut,
			Pattern: basePath + addGroupPath, Handler: s.AddGroup,
		},
		{
			Method:  http.MethodDelete,
			Pattern: basePath + deleteUserPath + "/{user_id}", Handler: s.DeleteUser,
		},
		{
			Method:  http.MethodPatch,
			Pattern: basePath + updateUserPath + "/{user_id}", Handler: s.UpdateUser,
		},
		{
			Method:  http.MethodGet,
			Pattern: basePath + "/{user_id}" + getMentorGroupsPath, Handler: s.GetMentorGroups,
		},
		{
			Method:  http.MethodPost,
			Pattern: basePath + dynamicregistrationPath, Handler: s.DynamicRegistration,
		},
		{
			Method:  http.MethodGet,
			Pattern: basePath + listUsersPath, Handler: s.GetAllUsers,
		},
		{
			Method:  http.MethodGet,
			Pattern: basePath + listGroupsPath, Handler: s.GetAllGroups,
		},
	}
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
	ctx, span, cancel := tracer.Start(req.Context(), "SigninHandlerSpan")
	defer cancel()

	resp.Header().Set("Content-Type", "application/json")

	var requestDTO dto.SigninRequestDTO
	if err := json.NewDecoder(req.Body).Decode(&requestDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam,
			"decode req body to signinRequestDTO failure: %v", err)
		slog.Error("decode req body to signinRequestDTO failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	res, err := s.factory.NewSignInCommand(requestDTO.Login, requestDTO.Password).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "signin command executing failure")
		slog.Error("signin command executing failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	if res == nil {
		err := errors.Wrap(entities.ErrInternal, "signin command executing completed with nil result")
		slog.Error("signin command executing completed with nil result", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	if !res.Success {
		err := errors.Wrap(entities.ErrInternal, "signin command executing completed with bad status")
		slog.Error("signin command executing completed with bad status", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	signinDTO := &dto.SigninResponseDTO{Token: res.Message}

	data, err := json.Marshal(signinDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal token failure: %v", err)
		slog.Error("marshal token failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusCreated)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error("write data to response failure", "error", err.Error())
		span.SetError(err)
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
	ctx, span, cancel := tracer.Start(req.Context(), "AddUserHandlerSpan")
	defer cancel()

	resp.Header().Set("Content-Type", "application/json")

	if err := s.getValidatedAuthContext(resp, req, []string{right_admin}); err != nil {
		err := errors.Wrap(err, "getValidatedAuthContext")
		slog.Error("getValidatedAuthContext", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	var requestDTO dto.AddUserDTO
	if err := json.NewDecoder(req.Body).Decode(&requestDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam,
			"decode req body to requestDTO failure: %v", err)
		slog.Error("decode req body to requestDTO failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	user, err := entities.NewUser(requestDTO.Username,
		requestDTO.Password,
		requestDTO.FullName,
		requestDTO.Rights,
		requestDTO.Contacts,
		requestDTO.GroupID,
	)
	if err != nil {
		err := errors.Wrap(err, "create base user user failure")
		slog.Error("create base user user failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	addUserResult, err := s.factory.NewAddUserCommand(user).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "add user command failure")
		slog.Error("add user command failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	if !addUserResult.Success {
		err := errors.Wrap(entities.ErrInternal, "add user failure")
		slog.Error("add user failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	responseDTO := &dto.AddUserResponseDTO{
		UserID: addUserResult.Message,
	}

	data, err := json.Marshal(responseDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal response failure: %v", err)
		slog.Error("marshal response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusCreated)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error("write data to response failure", "error", err.Error())
		span.SetError(err)
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
	ctx, span, cancel := tracer.Start(req.Context(), "DeleteUserHandlerSpan")
	defer cancel()

	resp.Header().Set("Content-Type", "application/json")

	if err := s.getValidatedAuthContext(resp, req, []string{right_admin}); err != nil {
		err := errors.Wrap(err, "getValidatedAuthContext")
		slog.Error("getValidatedAuthContext", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	userID := chi.URLParam(req, "user_id")

	if userID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "userID invalid")
		slog.Error("userID invalid", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	deleteUserResult, err := s.factory.NewDeleteUserCommand(userID).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "delete user command failure")
		slog.Error("delete user command failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	if !deleteUserResult.Success {
		err := errors.Wrap(entities.ErrInternal, "delete user failure")
		slog.Error("delete user failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusNoContent)
}

// Update user by ID
//
// @Summary      Update user
// @Description  Update user by ID. Requires admin rights.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Param        user_id path string true "User ID to update"
// @Param        request body dto.UpdateUserDTO true "User credentials and other data"
// @Success      200 {object} dto.UpdateUserResponseDTO "User updated"
// @Failure      400 {object} dto.ErrorDTO "Invalid request parameters"
// @Failure      401 {object} dto.ErrorDTO "Unauthorized"
// @Failure      403 {object} dto.ErrorDTO "Forbidden"
// @Failure      404 {object} dto.ErrorDTO "User not found"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /auth/v1/update-user/{user_id} [patch]
//
//nolint:funlen //ok
func (s *Server) UpdateUser(resp http.ResponseWriter, req *http.Request) {
	slog.Info("UpdateUser started")
	ctx, span, cancel := tracer.Start(req.Context(), "UpdateUserHandlerSpan")
	defer cancel()

	resp.Header().Set("Content-Type", "application/json")

	if err := s.getValidatedAuthContext(resp, req, []string{right_admin}); err != nil {
		err := errors.Wrap(err, "getValidatedAuthContext")
		slog.Error("getValidatedAuthContext", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	userID := chi.URLParam(req, "user_id")

	if userID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "userID invalid")
		slog.Error("userID invalid", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	var updateUserDTO dto.UpdateUserDTO
	if err := json.NewDecoder(req.Body).Decode(&updateUserDTO); err != nil {
		err := errors.Wrap(entities.ErrInvalidParam, "invalid request body")
		slog.Error("invalid request body", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	updateUser := &entities.UserUpdate{
		ID:           userID,
		Username:     updateUserDTO.Username,
		FullName:     updateUserDTO.Fullname,
		PasswordHash: updateUserDTO.Password,
		Rights:       updateUserDTO.Rights,
		Contacts:     updateUserDTO.Contacts,
		GroupID:      updateUserDTO.GroupID,
	}

	updateUserResult, err := s.factory.NewUpdateUserCommand(updateUser).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "update user command exec failure")
		slog.Error("update user command exec failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	if !updateUserResult.Success {
		err := errors.Wrap(entities.ErrInternal, "update user failure")
		slog.Error("update user failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	responseDTO := &dto.UpdateUserResponseDTO{
		UserID: updateUserResult.Message,
	}

	data, err := json.Marshal(responseDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal response failure: %v", err)
		slog.Error("marshal response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusOK)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error("write data to response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}
}

// Add new group
//
// @Summary      Add new group
// @Description  Add new group with title and linked mentor ID. Requires admin rights.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Param        request body dto.AddGroupRequestDTO true "Group data"
// @Success      201 {object} dto.AddGroupResponseDTO "New group created"
// @Failure      400 {object} dto.ErrorDTO "Invalid request parameters"
// @Failure      401 {object} dto.ErrorDTO "Unauthorized"
// @Failure      403 {object} dto.ErrorDTO "Forbidden"
// @Failure      409 {object} dto.ErrorDTO "Conflict"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /auth/v1/add-group [put]
//
//nolint:funlen //ok
func (s *Server) AddGroup(resp http.ResponseWriter, req *http.Request) {
	slog.Info("AddGroup started")
	ctx, span, cancel := tracer.Start(req.Context(), "AddGroupHandlerSpan")
	defer cancel()

	resp.Header().Set("Content-Type", "application/json")

	if err := s.getValidatedAuthContext(resp, req, []string{right_admin}); err != nil {
		err := errors.Wrap(err, "getValidatedAuthContext")
		slog.Error("getValidatedAuthContext", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	var requestDTO dto.AddGroupRequestDTO
	if err := json.NewDecoder(req.Body).Decode(&requestDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam,
			"decode req body to requestDTO failure: %v", err)
		slog.Error("decode req body to requestDTO failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	res, err := s.factory.NewAddGroupCommand(requestDTO.Title, requestDTO.LinkedID).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "add group command exec failure")
		slog.Error("add group command exec failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	if !res.Success {
		err := errors.Wrap(entities.ErrInternal, "add group failure")
		slog.Error("add group failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	responseDTO := &dto.AddGroupResponseDTO{
		GroupID: res.Message,
	}

	data, err := json.Marshal(responseDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal response failure: %v", err)
		slog.Error("marshal response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusCreated)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error("write data to response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}
}

// Get mentor groups
//
// @Summary      Get mentor groups
// @Description  Get all groups managed by mentor with students list. Requires mentor rights.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Param        user_id path string true "Mentor user ID"
// @Success      200 {array} dto.GroupDTO "List of mentor groups with students"
// @Failure      400 {object} dto.ErrorDTO "Invalid request parameters"
// @Failure      401 {object} dto.ErrorDTO "Unauthorized"
// @Failure      403 {object} dto.ErrorDTO "Forbidden"
// @Failure      404 {object} dto.ErrorDTO "User not found"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /auth/v1/{user_id}/mentor-groups [get]
//
//nolint:funlen //ok
func (s *Server) GetMentorGroups(resp http.ResponseWriter, req *http.Request) {
	slog.Info("GetMentorGroups started")

	ctx, span, cancel := tracer.Start(req.Context(), "GetMentorGroupsHandlerSpan")
	defer cancel()

	resp.Header().Set("Content-Type", "application/json")

	if err := s.getValidatedAuthContext(resp, req, []string{right_mentor}); err != nil {
		err := errors.Wrap(err, "getValidatedAuthContext")
		slog.Error("getValidatedAuthContext", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	userID := chi.URLParam(req, "user_id")

	if userID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "userID invalid")
		slog.Error("userID invalid", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	res, err := s.factory.NewGetMentorGroupsCommand(userID).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "get mentor groups command exec failure")
		slog.Error("get mentor groups command exec failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	groups, ok := res.Payload.([]*entities.Group)
	if !ok {
		err := errors.Wrap(entities.ErrInternal, "result assertion failure")
		slog.Error("result assertion failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	groupDTOs := make([]dto.GroupDTO, 0, len(groups))
	for _, group := range groups {
		groupDTO := dto.GroupDTO{
			ID:       group.GetID(),
			Name:     group.GetName(),
			Students: make([]dto.StudentDTO, 0),
		}

		for _, student := range group.GetStudents() {
			studentDTO := dto.StudentDTO{
				ID:       student.GetID(),
				Name:     student.GetName(),
				Fullname: student.GetFullname(),
			}

			groupDTO.Students = append(groupDTO.Students, studentDTO)
		}

		groupDTOs = append(groupDTOs, groupDTO)
	}

	data, err := json.Marshal(groupDTOs)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal response failure: %v", err)
		slog.Error("marshal response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusOK)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error("write data to response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}
}

// Get all users
//
// @Summary      Get all users
// @Description  Get list of all registered users. Requires admin rights.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Success      200 {array} dto.UserDTO "List of all users"
// @Failure      401 {object} dto.ErrorDTO "Unauthorized"
// @Failure      403 {object} dto.ErrorDTO "Forbidden"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /auth/v1/users [get]
//
//nolint:funlen //ok
func (s *Server) GetAllUsers(resp http.ResponseWriter, req *http.Request) {
	slog.Info("GetAllUsers started")

	ctx, span, cancel := tracer.Start(req.Context(), "GetAllUsersHandlerSpan")
	defer cancel()

	resp.Header().Set("Content-Type", "application/json")

	if err := s.getValidatedAuthContext(resp, req, []string{right_admin}); err != nil {
		err := errors.Wrap(err, "getValidatedAuthContext")
		slog.Error("getValidatedAuthContext", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	res, err := s.factory.NewGetAllUsersCommand().Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "get all users command exec failure")
		slog.Error("get all users command exec failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	users, ok := res.Payload.([]*entities.User)
	if !ok {
		err := errors.Wrap(entities.ErrInternal, "result assertion failure")
		slog.Error("result assertion failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	userDTOs := make([]dto.UserDTO, 0, len(users))
	for _, user := range users {
		userDTOs = append(userDTOs, dto.UserDTO{
			ID:       user.ID,
			Username: user.Username,
			FullName: user.FullName,
			Rights:   user.Rights,
			Contacts: user.Contacts,
			GroupID:  user.GroupID,
		})
	}

	data, err := json.Marshal(userDTOs)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal response failure: %v", err)
		slog.Error("marshal response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusOK)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error("write data to response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}
}

// Get all groups
//
// @Summary      Get all groups
// @Description  Get list of all groups with linked mentor and students. Requires admin rights.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Success      200 {array} dto.GroupDTO "List of all groups"
// @Failure      401 {object} dto.ErrorDTO "Unauthorized"
// @Failure      403 {object} dto.ErrorDTO "Forbidden"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /auth/v1/groups [get]
//
//nolint:funlen,dupl //ok
func (s *Server) GetAllGroups(resp http.ResponseWriter, req *http.Request) {
	slog.Info("GetAllGroups started")

	ctx, span, cancel := tracer.Start(req.Context(), "GetAllGroupsHandlerSpan")
	defer cancel()

	resp.Header().Set("Content-Type", "application/json")

	if err := s.getValidatedAuthContext(resp, req, []string{right_admin}); err != nil {
		err := errors.Wrap(err, "getValidatedAuthContext")
		slog.Error("getValidatedAuthContext", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	res, err := s.factory.NewGetAllGroupsCommand().Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "get all groups command exec failure")
		slog.Error("get all groups command exec failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	groups, ok := res.Payload.([]*entities.Group)
	if !ok {
		err := errors.Wrap(entities.ErrInternal, "result assertion failure")
		slog.Error("result assertion failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	groupDTOs := make([]dto.GroupDTO, 0, len(groups))
	for _, group := range groups {
		groupDTO := dto.GroupDTO{
			ID:       group.GetID(),
			Name:     group.GetName(),
			LinkedID: group.GetLinkedID(),
			Students: make([]dto.StudentDTO, 0),
		}

		for _, student := range group.GetStudents() {
			studentDTO := dto.StudentDTO{
				ID:       student.GetID(),
				Name:     student.GetName(),
				Fullname: student.GetFullname(),
			}

			groupDTO.Students = append(groupDTO.Students, studentDTO)
		}

		groupDTOs = append(groupDTOs, groupDTO)
	}

	data, err := json.Marshal(groupDTOs)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal response failure: %v", err)
		slog.Error("marshal response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusOK)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error("write data to response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}
}

func (s *Server) errProcessing(resp http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	errDTO := dto.ErrorDTO{
		StatusCode: statusCode,
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
		slog.Error("marshal failure", "error", err.Error())
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(errDTO.StatusCode)
	resp.Write(errDtoData) //nolint:errcheck,gosec //ok
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
		slog.Error("authoriztion header not set", "error", err.Error())
		return err
	}

	const prefix = "Bearer "
	authorizationData := strings.Split(authHeader, prefix)
	if len(authorizationData) != 2 {
		err := errors.Wrap(entities.ErrForbidden, "authoriztion header invalid")
		slog.Error("authoriztion header invalid", "error", err.Error())
		return err
	}

	jwt := authorizationData[1]
	introspectResult, err := s.factory.NewIntrospectedCommand(jwt).Exec(req.Context())
	if err != nil {
		err := errors.Wrap(err, "inrospection failure")
		slog.Error("inrospection failure", "error", err.Error())
		return err
	}

	if !introspectResult.Success {
		err := errors.Wrap(entities.ErrForbidden, "operation forbidden")
		slog.Error("operation forbidden", "error", err.Error())
		return err
	}

	claims, ok := introspectResult.Payload.(*entities.UserClaims)
	if !ok {
		err := errors.Wrap(entities.ErrForbidden, "assertion of user claims failure")
		slog.Error("assertion of user claims failure", "error", err.Error())
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
		slog.Error("checkUserRights failure", "error", err.Error())
		s.errProcessing(resp, err)
		return err
	}

	return nil
}

// DynamicRegistration initiates dynamic registration by sending a 4-digit code
//
// @Summary      Dynamic registration
// @Description  Generates a 4-digit code and sends it to via the provider (email, telegram, etc.)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.DynamicRegistrationDTO true "User ID and delivery provider"
// @Success      200 "Code sent successfully"
// @Failure      400  {object}  dto.ErrorDTO "Invalid request parameters"
// @Failure      409  {object}  dto.ErrorDTO "Conflict"
// @Failure      500  {object}  dto.ErrorDTO "Internal server error"
// @Router       /auth/v1/dynamic-registration [post]
//
//nolint:funlen //ok
func (s *Server) DynamicRegistration(resp http.ResponseWriter, req *http.Request) {
	slog.Info("DynamicRegistration started")
	ctx, span, cancel := tracer.Start(req.Context(), "DynamicRegistrationHandlerSpan")
	defer cancel()

	resp.Header().Set("Content-Type", "application/json")

	var requestDTO dto.DynamicRegistrationDTO
	if err := json.NewDecoder(req.Body).Decode(&requestDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam,
			"decode req body to DynamicRegistrationDTO failure: %v", err)
		slog.Error("decode req body to DynamicRegistrationDTO failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	res, err := s.factory.NewDynamicRegisterCommand(requestDTO.UserID, requestDTO.Provider).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "dynamic register command executing failure")
		slog.Error("dynamic register command executing failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	if res == nil {
		err := errors.Wrap(entities.ErrInternal,
			"dynamic register command executing completed with nil result")
		slog.Error("dynamic register command executing completed with nil result", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	if !res.Success {
		err := errors.Wrap(entities.ErrInternal,
			"Dynamic register command executing completed with bad status")
		slog.Error("dynamic register command executing completed with bad status", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusOK)
}
