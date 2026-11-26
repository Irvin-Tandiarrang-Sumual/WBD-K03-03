package domain

const (
	MaxGroupNameLength = 255
)

type AccessType string

const (
	AccessViewOnly AccessType = "view_only"
	AccessViewEdit AccessType = "view_edit"
)

type DatabaseAccess struct {
	DatabaseID uint
	AccessType AccessType
}

type Group struct {
	ID               uint
	Name             string
	LeaderID         uint
	DatabaseAccesses []DatabaseAccess
}

type CreateGroupInput struct {
	Name             string
	LeaderID         uint
	DatabaseAccesses []DatabaseAccess
}

type UpdateGroupInput struct {
	Name             *string
	LeaderID         *uint
	DatabaseAccesses *[]DatabaseAccess
}

func (access AccessType) Allows(required AccessType) bool {
	if access == AccessViewEdit {
		return true
	}
	return access == AccessViewOnly && required == AccessViewOnly
}
