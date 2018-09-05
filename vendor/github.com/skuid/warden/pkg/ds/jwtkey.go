package ds

import (
	"time"

	"github.com/skuid/picard"
)

// JWTKey stores verification keys for JWTs
type JWTKey struct {
	Metadata       picard.Metadata `json:"-" picard:"tablename=site_jwt_key"`
	ID             string          `json:"id" picard:"primary_key,column=id"`
	OrganizationID string          `json:"siteId" picard:"multitenancy_key,column=organization_id"`
	PublicKey      string          `json:"publicKey" picard:"column=public_key" validate:"required"`
	CreatedByID    string          `picard:"column=created_by_id,audit=created_by"`
	UpdatedByID    string          `picard:"column=updated_by_id,audit=updated_by"`
	CreatedDate    time.Time       `picard:"column=created_at,audit=created_at"`
	UpdatedDate    time.Time       `picard:"column=updated_at,audit=updated_at"`
}
