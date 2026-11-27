package model

type Group struct {
	ID        uint       `gorm:"primaryKey"`
	Name      string     `gorm:"uniqueIndex;not null"`
	LeaderID  uint       `gorm:"not null"`
	Leader    User       `gorm:"foreignKey:LeaderID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Databases []Database `gorm:"many2many:group_databases;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Users     []User     `gorm:"many2many:user_groups;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type GroupDatabase struct {
	GroupID    uint   `gorm:"primaryKey"`
	DatabaseID uint   `gorm:"primaryKey"`
	AccessType string `gorm:"type:text;not null;check:access_type IN ('view_only','view_edit')"`
}
