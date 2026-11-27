package admin

import (
	"context"

	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type UpdateUserRoleInput struct {
	ActorID      int64
	ActorRole    user.Role
	TargetUserID int64
	NewRole      user.Role
}

type UpdateUserRoleUseCase struct {
	userRepo user.Repository
	log      *logrus.Entry
}

func NewUpdateUserRoleUseCase(userRepo user.Repository, log *logrus.Logger) *UpdateUserRoleUseCase {
	return &UpdateUserRoleUseCase{
		userRepo: userRepo,
		log:      log.WithField("usecase", "admin.update_user_role"),
	}
}

func (uc *UpdateUserRoleUseCase) Execute(ctx context.Context, input UpdateUserRoleInput) (*user.User, error) {
	if !input.ActorRole.IsAdmin() {
		return nil, user.ErrForbiddenRoleChange
	}
	if input.TargetUserID <= 0 {
		return nil, user.ErrUserNotFound
	}

	target, err := uc.userRepo.GetByID(ctx, input.TargetUserID)
	if err != nil {
		return nil, err
	}

	if err := target.ChangeModeratorRole(input.ActorID, input.ActorRole, input.NewRole); err != nil {
		uc.log.WithFields(logrus.Fields{
			"actor_id":       input.ActorID,
			"target_user_id": input.TargetUserID,
			"new_role":       input.NewRole,
		}).WithError(err).Warn("change user role rejected")
		return nil, err
	}

	if err := uc.userRepo.UpdateRole(ctx, target.ID, target.Role, target.UpdatedAt); err != nil {
		return nil, err
	}

	uc.log.WithFields(logrus.Fields{
		"actor_id":       input.ActorID,
		"target_user_id": target.ID,
		"new_role":       target.Role,
	}).Info("user role updated")
	return target, nil
}
