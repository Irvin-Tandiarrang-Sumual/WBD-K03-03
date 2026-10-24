package models

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"gorm.io/gorm"
)

type User struct {
	ID uint `gorm:"primaryKey"`

	Username string `gorm:"unique;not null"`
	Password string `gorm:"notnull"`
	Role     string `gorm:"type:text;check:role IN ('user', 'admin');default:user"`

	Sessions []Session `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Groups   []Group   `gorm:"many2many:user_groups;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.hasGroupIDs() {
		return domain.ErrUserMustHaveGroup
	}

	return u.areGroupIDsValid(tx)

}

func (u *User) hasGroupIDs() bool {
	return len(u.Groups) == 0
}

func (u *User) areGroupIDsValid(tx *gorm.DB) (err error) {
	var inputIDs []uint
	for _, g := range u.Groups {
		if g.ID != 0 {
			inputIDs = append(inputIDs, g.ID)
		}
	}

	var countInDB int64
	err = tx.Model(&Group{}).Where("id IN ?", inputIDs).Count(&countInDB).Error
	if err != nil {
		return err
	}

	if int(countInDB) != len(inputIDs) {
		return domain.ErrInvalidGroupsIDs
	}

	return nil
}
