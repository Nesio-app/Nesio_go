package items

import (
	"time"

	"github.com/google/uuid"
)

type CreateRequest struct {
	Name             string     `json:"name"`
	Body             *string    `json:"body,omitempty"`
	RoomID           *uuid.UUID `json:"room_id,omitempty"`
	ContainerID      *uuid.UUID `json:"container_id,omitempty"`
	LocationNote     *string    `json:"location_note,omitempty"`
	ExpiryDate       *string    `json:"expiry_date,omitempty"`
	ExpiryRemindDays *int       `json:"expiry_remind_days,omitempty"`
	IsDocument       bool       `json:"is_document,omitempty"`
	DocumentType     *string    `json:"document_type,omitempty"`
	DocumentNumber   *string    `json:"document_number,omitempty"`
	Quantity         *int       `json:"quantity,omitempty"`
	Unit             *string    `json:"unit,omitempty"`
	PrimaryImageURL  *string    `json:"primary_image_url,omitempty"`
	VisualHash       *string    `json:"visual_hash,omitempty"`
	ReminderLabel    *string    `json:"reminder_label,omitempty"`
	Tags             []string   `json:"tags,omitempty"`
}

type AnalyzeResponse struct {
	Extraction           map[string]any   `json:"extraction"`
	Duplicates           []map[string]any `json:"duplicates"`
	VisualHash           string           `json:"visual_hash"`
	SuggestedRoomID      *uuid.UUID       `json:"suggested_room_id,omitempty"`
	SuggestedContainerID *uuid.UUID       `json:"suggested_container_id,omitempty"`
	ImageURL             string           `json:"image_url,omitempty"`
}

type ListResponse struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	Name            string     `db:"name" json:"name"`
	Type            string     `db:"type" json:"type"`
	Body            *string    `db:"body" json:"body,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	RoomID          *uuid.UUID `db:"room_id" json:"room_id,omitempty"`
	ContainerID     *uuid.UUID `db:"container_id" json:"container_id,omitempty"`
	LocationNote    *string    `db:"location_note" json:"location_note,omitempty"`
	RoomName        *string    `db:"room_name" json:"room_name,omitempty"`
	RoomIcon        *string    `db:"room_icon" json:"room_icon,omitempty"`
	ContainerName   *string    `db:"container_name" json:"container_name,omitempty"`
	ContainerIcon   *string    `db:"container_icon" json:"container_icon,omitempty"`
	ExpiryDate      *time.Time `db:"expiry_date" json:"expiry_date,omitempty"`
	IsDocument      bool       `db:"is_document" json:"is_document"`
	DocumentType    *string    `db:"document_type" json:"document_type,omitempty"`
	DocumentNumber  *string    `db:"document_number" json:"document_number,omitempty"`
	Quantity        int        `db:"quantity" json:"quantity"`
	Unit            *string    `db:"unit" json:"unit,omitempty"`
	PrimaryImageURL *string    `db:"primary_image_url" json:"primary_image_url,omitempty"`
	DaysUntilExpiry *int       `db:"days_until_expiry" json:"days_until_expiry,omitempty"`
}

type DuplicateRow struct {
	ID            uuid.UUID `db:"id"`
	Name          string    `db:"name"`
	VisualHash    string    `db:"visual_hash"`
	RoomName      *string   `db:"room_name"`
	ContainerName *string   `db:"container_name"`
}
