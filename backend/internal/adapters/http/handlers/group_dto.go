package handlers

import "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"

type DatabaseAccessRequest struct {
	DatabaseID uint   `json:"databaseId" binding:"required"`
	AccessType string `json:"accessType" binding:"required"`
}

type CreateGroupRequest struct {
	Name             string                  `json:"name" binding:"required"`
	LeaderID         uint                    `json:"leaderId" binding:"required"`
	DatabaseAccesses []DatabaseAccessRequest `json:"databaseAccesses" binding:"required"`
}

type UpdateGroupRequest struct {
	Name             *string                  `json:"name"`
	LeaderID         *uint                    `json:"leaderId"`
	DatabaseAccesses *[]DatabaseAccessRequest `json:"databaseAccesses"`
}

type DatabaseAccessResponse struct {
	DatabaseID uint   `json:"databaseId"`
	AccessType string `json:"accessType"`
}

type GroupResponse struct {
	ID               uint                     `json:"id"`
	Name             string                   `json:"name"`
	LeaderID         uint                     `json:"leaderId"`
	MemberIDs        []uint                   `json:"memberIds"`
	DatabaseAccesses []DatabaseAccessResponse `json:"databaseAccesses"`
}

func toDatabaseAccesses(requests []DatabaseAccessRequest) []domain.DatabaseAccess {
	accesses := make([]domain.DatabaseAccess, 0, len(requests))

	for _, request := range requests {
		accesses = append(accesses, domain.DatabaseAccess{
			DatabaseID: request.DatabaseID,
			AccessType: domain.AccessType(request.AccessType),
		})
	}

	return accesses
}

func toGroupResponse(group domain.Group) GroupResponse {
	accesses := make([]DatabaseAccessResponse, 0, len(group.DatabaseAccesses))

	for _, access := range group.DatabaseAccesses {
		accesses = append(accesses, DatabaseAccessResponse{
			DatabaseID: access.DatabaseID,
			AccessType: string(access.AccessType),
		})
	}

	memberIDs := group.MemberIDs
	if memberIDs == nil {
		memberIDs = []uint{}
	}

	return GroupResponse{
		ID:               group.ID,
		Name:             group.Name,
		LeaderID:         group.LeaderID,
		MemberIDs:        memberIDs,
		DatabaseAccesses: accesses,
	}
}
