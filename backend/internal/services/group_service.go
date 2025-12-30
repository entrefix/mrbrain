package services

import (
	"fmt"

	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/repository"
)

type GroupService struct {
	groupRepo *repository.GroupRepository
}

func NewGroupService(groupRepo *repository.GroupRepository) *GroupService {
	return &GroupService{
		groupRepo: groupRepo,
	}
}

func (s *GroupService) Create(userID string, req *models.GroupCreateRequest) (*models.Group, error) {
	group := &models.Group{
		UserID:    &userID,
		Name:      req.Name,
		ColorCode: req.ColorCode,
		IsDefault: false,
	}

	if err := s.groupRepo.Create(group); err != nil {
		return nil, err
	}

	return group, nil
}

func (s *GroupService) GetAll(userID string) ([]models.Group, error) {
	return s.groupRepo.GetAllByUserID(userID)
}

func (s *GroupService) GetByID(userID, groupID string) (*models.Group, error) {
	group, err := s.groupRepo.GetByID(groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, nil
	}

	// Allow access if it's a default group or belongs to the user
	if group.IsDefault || (group.UserID != nil && *group.UserID == userID) {
		return group, nil
	}

	return nil, nil
}

func (s *GroupService) Update(userID, groupID string, req *models.GroupUpdateRequest) (*models.Group, error) {
	// Verify ownership (can't update default groups)
	group, err := s.groupRepo.GetByID(groupID)
	if err != nil {
		return nil, err
	}
	if group == nil || group.IsDefault || group.UserID == nil || *group.UserID != userID {
		return nil, fmt.Errorf("group not found or cannot be updated")
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ColorCode != nil {
		updates["color_code"] = *req.ColorCode
	}

	if len(updates) > 0 {
		if err := s.groupRepo.Update(groupID, updates); err != nil {
			return nil, err
		}
	}

	return s.groupRepo.GetByID(groupID)
}

func (s *GroupService) Delete(userID, groupID string) error {
	// Verify ownership (can't delete default groups)
	group, err := s.groupRepo.GetByID(groupID)
	if err != nil {
		return err
	}
	if group == nil || group.IsDefault || group.UserID == nil || *group.UserID != userID {
		return fmt.Errorf("group not found or cannot be deleted")
	}

	return s.groupRepo.Delete(groupID)
}
