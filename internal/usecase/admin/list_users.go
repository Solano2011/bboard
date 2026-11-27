package admin

import (
	"context"

	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type ListUsersInput struct {
	ActorRole user.Role
	Search    string
	Limit     int
	Offset    int
}

type ListUsersUseCase struct {
	userRepo user.Repository
	log      *logrus.Entry
}

func NewListUsersUseCase(userRepo user.Repository, log *logrus.Logger) *ListUsersUseCase {
	return &ListUsersUseCase{
		userRepo: userRepo,
		log:      log.WithField("usecase", "admin.list_users"),
	}
}

func (uc *ListUsersUseCase) Execute(ctx context.Context, input ListUsersInput) ([]*user.User, error) {
	if !input.ActorRole.IsAdmin() {
		return nil, user.ErrForbiddenRoleChange
	}

	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	return uc.userRepo.List(ctx, input.Search, limit, offset)
}
