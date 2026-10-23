package models

import "gorm.io/gorm"

type Database struct {
	gorm.Model
	Port     uint   `gorm:"unique;not null"`
	Host     string `gorm:"not null"`
	User     string `gorm:"not null"`
	Password string
	Type     string `gorm:"type:text;check:type IN ('postgresql', 'mysql');default:postgresql"`
	Name     string `gorm:"unique;not null"`

	Groups     []Group     `gorm:"many2many:group_databases;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Migrations []Migration `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}
