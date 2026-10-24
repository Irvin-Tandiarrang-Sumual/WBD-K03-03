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
	ID uint
	Name string
	DatabaseAccesses []DatabaseAccess
}
