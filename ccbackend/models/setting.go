package models

import (
	"time"

	"github.com/lib/pq"
)

// SettingType represents the type of setting value
type SettingType string

const (
	SettingTypeBool      SettingType = "bool"
	SettingTypeString    SettingType = "string"
	SettingTypeStringArr SettingType = "stringarr"
)

// SettingKeyDefinition defines a supported setting key with its expected type
type SettingKeyDefinition struct {
	Key  string
	Type SettingType
}

// SupportedSettings is the registry of all supported setting keys with their types
var SupportedSettings = map[string]SettingKeyDefinition{
	"org/onboarding_finished": {
		Key:  "org/onboarding_finished",
		Type: SettingTypeBool,
	},
}

// Setting represents a generic setting with all possible value types
type Setting struct {
	ID             string         `json:"id"                        db:"id"`
	OrganizationID string         `json:"organization_id"           db:"organization_id"`
	ScopeType      string         `json:"scope_type"                db:"scope_type"`
	ScopeID        string         `json:"scope_id"                  db:"scope_id"`
	Key            string         `json:"key"                       db:"key"`
	ValueBoolean   *bool          `json:"value_boolean,omitempty"   db:"value_boolean"`
	ValueString    *string        `json:"value_string,omitempty"    db:"value_string"`
	ValueStringArr pq.StringArray `json:"value_stringarr,omitempty" db:"value_stringarr"`
	CreatedAt      time.Time      `json:"created_at"                db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"                db:"updated_at"`
}

// SettingBool represents a boolean setting
type SettingBool struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ScopeType      string    `json:"scope_type"`
	ScopeID        string    `json:"scope_id"`
	Key            string    `json:"key"`
	Value          bool      `json:"value"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SettingString represents a string setting
type SettingString struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ScopeType      string    `json:"scope_type"`
	ScopeID        string    `json:"scope_id"`
	Key            string    `json:"key"`
	Value          string    `json:"value"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SettingStringArr represents a string array setting
type SettingStringArr struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ScopeType      string    `json:"scope_type"`
	ScopeID        string    `json:"scope_id"`
	Key            string    `json:"key"`
	Value          []string  `json:"value"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ToSettingBool converts a generic Setting to SettingBool
func (s *Setting) ToSettingBool() *SettingBool {
	if s.ValueBoolean == nil {
		return nil
	}
	return &SettingBool{
		ID:             s.ID,
		OrganizationID: s.OrganizationID,
		ScopeType:      s.ScopeType,
		ScopeID:        s.ScopeID,
		Key:            s.Key,
		Value:          *s.ValueBoolean,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

// ToSettingString converts a generic Setting to SettingString
func (s *Setting) ToSettingString() *SettingString {
	if s.ValueString == nil {
		return nil
	}
	return &SettingString{
		ID:             s.ID,
		OrganizationID: s.OrganizationID,
		ScopeType:      s.ScopeType,
		ScopeID:        s.ScopeID,
		Key:            s.Key,
		Value:          *s.ValueString,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

// ToSettingStringArr converts a generic Setting to SettingStringArr
func (s *Setting) ToSettingStringArr() *SettingStringArr {
	if len(s.ValueStringArr) == 0 {
		return nil
	}
	return &SettingStringArr{
		ID:             s.ID,
		OrganizationID: s.OrganizationID,
		ScopeType:      s.ScopeType,
		ScopeID:        s.ScopeID,
		Key:            s.Key,
		Value:          []string(s.ValueStringArr),
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

// GetSettingType returns the type of the setting based on which value field is set
func (s *Setting) GetSettingType() SettingType {
	if s.ValueBoolean != nil {
		return SettingTypeBool
	}
	if s.ValueString != nil {
		return SettingTypeString
	}
	if len(s.ValueStringArr) > 0 {
		return SettingTypeStringArr
	}
	return ""
}
