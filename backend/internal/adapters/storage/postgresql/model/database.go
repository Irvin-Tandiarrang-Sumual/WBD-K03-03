package model

type Database struct {
	ID                 uint        `gorm:"primaryKey"`
	Port               uint        `gorm:"not null"`
	Host               string      `gorm:"not null"`
	User               string      `gorm:"not null"`
	Password           string
	Type               string      `gorm:"type:text;check:type IN ('postgresql', 'mysql');default:postgresql"`
	Name               string      `gorm:"unique;not null"`
	LatestExecutionID  uint        `gorm:"not null;default:0"`
	CurrentMigrationID *uint       `gorm:"index"`
	CurrentMigration   *Migration  `gorm:"foreignKey:CurrentMigrationID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	Groups     []Group     `gorm:"many2many:group_databases;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Migrations []Migration `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Drafts     []Draft     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
