package models

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"gorm.io/gorm"
)

type User struct {
	ID uint `gorm:"primaryKey"`

	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
	Role     string `gorm:"type:text;check:role IN ('user', 'admin');default:user"`

	Sessions []Session `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Groups   []Group   `gorm:"many2many:user_groups;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type UserGroup struct {
	UserID  uint `gorm:"primaryKey"`
	GroupID uint `gorm:"primaryKey"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if len(u.Groups) == 0 {
		return domain.ErrUserMustHaveGroup
	}

	inputIDs := make([]uint, 0, len(u.Groups))

	for _, group := range u.Groups {
		if group.ID != 0 {
			inputIDs = append(inputIDs, group.ID)
		}
	}

	var countInDB int64

	if err := tx.
		Model(&Group{}).
		Where("id IN ?", inputIDs).
		Count(&countInDB).Error; err != nil {
		return err
	}

	if int(countInDB) != len(inputIDs) {
		return domain.ErrInvalidGroupsIDs
	}

	return nil
}