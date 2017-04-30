package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/creack/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	_ "github.com/lib/pq"
)

// Common errors.
var (
	ErrInvalidType = errors.New("invalid type")
)

// ScanToString returns the string version of the given interface.
// If not a `string` or a `[]byte`, returns ErrInvalidType.
func ScanToString(src interface{}) (string, error) {
	switch s := src.(type) {
	case string:
		return s, nil
	case []byte:
		return string(s), nil
	default:
		return "", ErrInvalidType
	}
}

// TimeMetadata .
type TimeMetadata struct {
	CreatedAt time.Time  `json:"created_at"           db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"           db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// MarshalJSON implements json.Marshaler interface.
func (tm *TimeMetadata) MarshalJSON() ([]byte, error) {
	if tm == nil {
		return nil, nil
	}
	mm := map[string]time.Time{}
	if !tm.CreatedAt.IsZero() {
		mm["created_at"] = tm.CreatedAt
	}
	if !tm.UpdatedAt.IsZero() {
		mm["updated_at"] = tm.UpdatedAt
	}
	if tm.DeletedAt != nil && !tm.DeletedAt.IsZero() {
		mm["deleted_at"] = *tm.DeletedAt
	}
	if len(mm) == 0 {
		return nil, nil
	}
	return json.Marshal(mm)
}

// Scan1 implements sql.Scan interface.
func (tm *TimeMetadata) Scan1(src interface{}) error {
	s, err := ScanToString(src)
	if err != nil {
		return errors.Wrap(err, "invalid type for TimeMetadata scan")
	}
	s = strings.Trim(s, "()")
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return errors.New("invalid count for TimeMetadata scan")
	}
	for i, part := range parts {
		parts[i] = strings.Trim(part, `"`)
	}

	tm.CreatedAt, err = pq.ParseTimestamp(time.UTC, parts[0])
	if err != nil {
		return errors.Wrap(err, "error parsing created_at")
	}
	tm.UpdatedAt, err = pq.ParseTimestamp(time.UTC, parts[1])
	if err != nil {
		return errors.Wrap(err, "error parsing updated_at")
	}
	if parts[2] != "" {
		deletedAt, err := pq.ParseTimestamp(time.UTC, parts[2])
		if err != nil {
			return errors.Wrap(err, "error parsing deleted_at")
		}
		tm.DeletedAt = &deletedAt
	}
	return nil
}

// Metadata .
type Metadata struct {
	Owner        *User `json:"owner,omitempty"`
	TimeMetadata `json:",inline" db:"timemetadata"`
}

// MarshalJSON implements json.Marshaler interface.
func (m Metadata) MarshalJSON() ([]byte, error) {
	mm := map[string]interface{}{}
	if m.Owner != nil {
		mm["owner_id"] = m.Owner.ID
	}
	if !m.TimeMetadata.CreatedAt.IsZero() {
		mm["created_at"] = m.TimeMetadata.CreatedAt
	}
	if !m.TimeMetadata.UpdatedAt.IsZero() {
		mm["updated_at"] = m.TimeMetadata.UpdatedAt
	}
	if m.TimeMetadata.DeletedAt != nil && !m.TimeMetadata.DeletedAt.IsZero() {
		mm["deleted_at"] = m.TimeMetadata.DeletedAt
	}
	return json.Marshal(mm)
}

// Scan1 implements sql.Scan interface.
func (m *Metadata) Scan1(src interface{}) error {
	s, err := ScanToString(src)
	if err != nil {
		return errors.Wrap(err, "invalid type for Metadata scan")
	}
	s = strings.Trim(s, "()")
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return errors.New("invalid count for Metadata scan")
	}
	for i, part := range parts {
		parts[i] = strings.Trim(part, `"`)
	}
	ownerID := uuid.Parse(parts[0])
	if ownerID == nil {
		return errors.New("invalid owner_id for Metadata scan")
	}
	m.Owner = &User{ID: ownerID}

	return m.TimeMetadata.Scan1("(" + strings.Join(parts[1:], ",") + ")")
}

// User .
type User struct {
	ID uuid.UUID `json:"user_id" db:"user_id"`

	Organizations UserOrganizations `json:"organization_memberships,omitempty" db:"organization_memberships"`
	Teams         []UserTeam        `json:"team_memberships,omitempty"         db:"team_memberships"`
	PaymentPlan   *PaymentPlan      `json:"payment_plan,omitempty"             db:"payment_plan"`

	Metadata Metadata `json:"metadata" db:"metadata"`
}

// UserOrganizations .
type UserOrganizations []UserOrganization

// Scan implement sql.Scanner interface.
func (uos *UserOrganizations) Scan(src interface{}) error {
	var strArray pq.StringArray

	if err := strArray.Scan(src); err != nil {
		return errors.Wrap(err, "error parsing db result into string array")
	}

	for _, elem := range strArray {
		uo := UserOrganization{}
		if err := uo.Scan(elem); err != nil {
			return errors.Wrap(err, "error parsing db result element into user organization")
		}
		*uos = append(*uos, uo)
	}
	return nil
}

// UserOrganization .
type UserOrganization struct {
	UserID         uuid.UUID `json:"user_id"         db:"user_id"`
	OrganizationID uuid.UUID `json:"organization_id" db:"organization_id"`

	Role string `json:"role" db:"role"`

	Metadata Metadata `json:"metadata" db:"metadata"`
}

// Scan impements sql.Scanner interface.
func (uo *UserOrganization) Scan(src interface{}) error {
	s, err := ScanToString(src)
	if err != nil {
		return errors.Wrap(err, "invalid type for UserOrganization scan")
	}

	parts := strings.Split(strings.Trim(s, "()"), ",")

	uo.UserID = uuid.Parse(parts[0])
	uo.OrganizationID = uuid.Parse(parts[1])
	uo.Role = parts[2]

	if uo.UserID == nil {
		return errors.New("invalid user_id")
	}
	if uo.OrganizationID == nil {
		return errors.New("invalid organization_id")
	}
	if uo.Role == "" {
		return errors.New("invalid user_role")
	}
	tm := "(" + strings.Join(parts[3:7], ",") + ")"
	if err := uo.Metadata.Scan1(tm); err != nil {
		return errors.Wrap(err, "error scan TimeMetadata for UserOrganization")
	}

	return nil
}

// Organization .
type Organization struct {
	ID uuid.UUID `json:"organization_id" db:"organization_id"`

	Users       []*UserOrganization `json:"users"                  db:"users"`
	Teams       []*Team             `json:"teams,omitempty"        db:"teams"`
	PaymentPlan *PaymentPlan        `json:"payment_plan,omitempty" db:"payment_plan"`

	Metadata Metadata `json:"metadata" db:"metadata"`
}

// UserTeam .
type UserTeam struct {
	UserID         uuid.UUID `json:"user_id"         db:"user_id"`
	OrganizationID uuid.UUID `json:"organization_id" db:"organization_id"`

	Role string `json:"role" db:"role"`

	Metadata Metadata `json:"metadata"`
}

// Team .
type Team struct {
	ID uuid.UUID `json:"team_id" db:"team_id"`

	Organization *Organization `json:"organization,omitempty" db:"organization"`
	Users        []*UserTeam   `json:"users"                  db:"users"`

	Name     string `json:"name"     db:"name"`
	Capacity int    `json:"capacity" db:"capacity"` // Maximum number of users in the team. 0 = no limit.

	Metadata Metadata `json:"metadata"`
}

// PaymentPlan .
type PaymentPlan struct {
	ID uuid.UUID `json:"payment_plan_id" db:"payment_plan_id"`

	Name     string  `json:"name"     db:"name"`
	Cost     float64 `json:"cost"     db:"cost"`
	Currency string  `json:"currency" db:"currency"`
	Term     string  `json:"term"     db:"term"` // Term of the payment plan. "Yearly", "Monthly", etc..

	Metadata `json:",inline" db:"metadata"`
}

func test(ctx context.Context) error {
	db, err := sqlx.ConnectContext(ctx, "postgres", "postgres://postgres@192.168.99.100:5432/test?sslmode=disable")
	if err != nil {
		return errors.Wrap(err, "error connect to db")
	}

	const queryGetUser = `
SELECT
  u.user_id,
  u.owner_id     AS "metadata.owner.user_id",
  array_agg(uoj) AS "organization_memberships",
  u.created_at   AS "metadata.timemetadata.created_at",
  u.updated_at   AS "metadata.timemetadata.updated_at",
  u.deleted_at   AS "metadata.timemetadata.deleted_at"
FROM users u
LEFT JOIN user_organization_join uoj
  USING (user_id)
WHERE u.user_id = uuid_nil()
GROUP BY u.user_id
`

	u := User{}
	if err := db.GetContext(ctx, &u, queryGetUser); err != nil {
		return errors.Wrap(err, "error get user")
	}

	enc := json.NewEncoder(os.Stdout)

	enc.SetIndent("", "    ")
	_ = enc.Encode(u)

	const queryInsertUser = `
INSERT INTO users (
  user_id,
  owner_id
) VALUES (
  ?,
  ?
)
`
	u.ID = uuid.NewRandom()
	query := db.Rebind(queryInsertUser)
	if _, err := db.ExecContext(ctx, query,
		u.ID,
		u.Metadata.Owner.ID,
	); err != nil {
		return errors.Wrap(err, "error insert user")
	}

	return nil
}

func main() {
	if err := test(context.Background()); err != nil {
		log.Fatal(err)
	}
}
