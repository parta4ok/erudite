package port

import (
	"github.com/parta4ok/kvs/auth/internal/entities"
)

//go:generate mockgen -source=command_factory.go -destination=./testdata/command_factory.go -package=testdata
type CommandFactory interface {
	NewIntrospectedCommand(cjwt string) entities.Command
	NewSignInCommand(userName string, password string) entities.Command
	NewAddUserCommand(user *entities.User) entities.Command
	NewDeleteUserCommand(userID string) entities.Command
	NewUpdateUserCommand(user *entities.User) entities.Command
	NewGetLinkedUsersCommand(userID string) entities.Command
	NewAddGroupCommand(title string, linkedID string) entities.Command
	NewGetMentorGroupsCommand(mentorID string) entities.Command
	NewGetUserByIDCommand(userID string) entities.Command
	NewGetGroupTitleByIDCommand(groupID string) entities.Command
	NewDynamicRegisterCommand(userID, provider string) entities.Command
	NewGetAllUsersCommand() entities.Command
	NewGetAllGroupsCommand() entities.Command
}
