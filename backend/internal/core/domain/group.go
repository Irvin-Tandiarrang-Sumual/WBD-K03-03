package domain

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
	DatabaseAccesses []DatabaseAccess
}

type CreateGroupInput struct {
	Name             string
	DatabaseAccesses []DatabaseAccess
}

type UpdateGroupInput struct {
	Name             *string
	DatabaseAccesses *[]DatabaseAccess
}

func (access AccessType) Allows(required AccessType) bool {
	if access == AccessViewEdit {
		return true
	}
	return access == AccessViewOnly && required == AccessViewOnly
}
