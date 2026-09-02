package server

import "time"

type DBUser struct {
	ID        string `gorm:"primaryKey"`
	FirstName string `gorm:"not null"`
	LastName  string `gorm:"not null"`
	Email     string `gorm:"uniqueIndex;not null"`
	GoogleID  string `gorm:"index"`
	CreatedAt time.Time
}

type DBSession struct {
	Token     string    `gorm:"primaryKey"`
	UserID    string    `gorm:"index;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

type DBContest struct {
	ID          string `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Year        int
	OwnerUserID string `gorm:"index;not null"` // permanent creator, never changes
	// HeadJudgeUserID defaults to OwnerUserID (set at creation) but is
	// transferable — see transferHeadJudge. Empty on rows created before
	// this column existed; dbContestToContest falls back to OwnerUserID.
	HeadJudgeUserID string `gorm:"index"`
	// Locked freezes the *entire* contest (all divisions/stages/scores/
	// players/judges) against any write, including by the head judge
	// themselves — only unlocking (head-judge-only) is exempt.
	Locked bool `gorm:"not null;default:false"`
	// Hidden excludes a contest from listAllContests() without deleting
	// it — used for SeedIfEmpty's demo contest so it doesn't clutter the
	// (now globally visible) contest list, while keeping the row around
	// for reference. A hidden contest is still reachable directly by ID.
	Hidden    bool `gorm:"not null;default:false"`
	CreatedAt time.Time
}

type DBDivision struct {
	ID        string `gorm:"primaryKey"`
	ContestID string `gorm:"index;not null"`
	Name      string `gorm:"not null"`
	Stages    string `gorm:"not null"` // JSON: []ScoringStage
	CreatedAt time.Time
}

type DBJudgeAssignment struct {
	ID         string `gorm:"primaryKey"`
	ContestID  string `gorm:"index;not null"`
	DivisionID string `gorm:"index;not null"`
	Stage      string `gorm:"not null"`
	UserID     string `gorm:"index;not null"`
	Role       string `gorm:"not null"`
	Slot       int    `gorm:"not null"`
	CreatedAt  time.Time
}

type DBPlayer struct {
	ID         string `gorm:"primaryKey"`
	DivisionID string `gorm:"index;not null"`
	Number     int
	Name       string `gorm:"not null"`
	CreatedAt  time.Time
}

// DBPlayerRawScore stores one player's judge inputs for one division+stage.
// (DivisionID, PlayerID, Stage) is unique; Clickers/Deductions/Evals are JSON blobs.
type DBPlayerRawScore struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	DivisionID string `gorm:"uniqueIndex:idx_raw_score;not null"`
	PlayerID   string `gorm:"uniqueIndex:idx_raw_score;not null"`
	Stage      string `gorm:"uniqueIndex:idx_raw_score;not null"`
	Clickers   string `gorm:"not null;default:'{}'"`
	Deductions string `gorm:"not null;default:'{}'"`
	Evals      string `gorm:"not null;default:'{}'"`
	UpdatedAt  time.Time
}
