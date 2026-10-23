package models

import (
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain/errors"
	"gorm.io/gorm"
)

type Group struct {
	gorm.Model
	AccessType string     `gorm:"type:text;check:access_type IN ('view_only', 'view_edit');default:view_only"`
	DatabaseID int64      `gorm:"not null"`
	Databases  []Database `gorm:"many2many:group_databases;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Users      []User     `gorm:"many2many:user_groups;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (g *Group) BeforeCreate(tx *gorm.DB) (err error) {
	if !g.hasDatabaseIDs() {
		return errors.ErrGroupMustHaveDatabase
	}

	if err := g.areDatabaseIDsValid(tx); err != nil {
		return err
	}

	return nil
}

func (g *Group) hasDatabaseIDs() bool {
	return len(g.Databases) > 0
}

func (g *Group) areDatabaseIDsValid(tx *gorm.DB) (err error) {
	var inputDBIDs []uint

	for _, db := range g.Databases {
		if db.ID != 0 {
			inputDBIDs = append(inputDBIDs, db.ID)
		}
	}

	if len(inputDBIDs) == 0 {
		return errors.ErrDatabaseNotFound
	}

	var countInDB int64
	err = tx.Model(&Database{}).Where("id IN ?", inputDBIDs).Count(&countInDB).Error
	if err != nil {
		return err
	}

	if int(countInDB) != len(inputDBIDs) {
		return errors.ErrDatabaseNotFound
	}

	return nil
}
