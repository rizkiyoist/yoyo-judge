package server

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"yoyo-judge/library/calc"
)

func OpenDB() *gorm.DB {
	path := os.Getenv("DATABASE_PATH")
	if path == "" {
		path = "yoyojudge.db"
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("failed to open SQLite at %q: %v", path, err)
	}
	// WAL mode: better concurrent read throughput, same durability.
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")
	if err := db.AutoMigrate(
		&DBUser{}, &DBSession{},
		&DBContest{}, &DBDivision{},
		&DBJudgeAssignment{}, &DBPlayer{}, &DBPlayerRawScore{},
	); err != nil {
		log.Fatalf("auto-migrate failed: %v", err)
	}
	return db
}

// clickerJudgeNames and evalJudgeNames mirror frontend/src/api/mock.ts's arrays.
var clickerJudgeNames = [6][2]string{
	{"Paketu", "Dennis"}, {"Levian", "Saputra"}, {"Boris", "Chietra"},
	{"Doni", "Firmansyah"}, {"Eko", "Prasetyo"}, {"Fajar", "Nugroho"},
}
var evalJudgeNames = [6][2]string{
	{"Reynold", "Andika"}, {"Hendra", "Kusumah"}, {"Indah", "Puspitasari"},
	{"Joko", "Susanto"}, {"Kartika", "Dewi"}, {"Lestari", "Wulandari"},
}

// SeedIfEmpty populates the DB with a demo contest the first time it runs.
// Safe to call on every startup — it returns immediately when any user exists.
func SeedIfEmpty(db *gorm.DB) {
	var count int64
	db.Model(&DBUser{}).Count(&count)
	if count > 0 {
		return
	}

	headJudge := DBUser{ID: newID("u"), FirstName: "Galih", LastName: "Kurniawan", Email: "galih@example.com"}
	db.Create(&headJudge)

	var clickerJudges, evalJudges []DBUser
	for i := 0; i < 6; i++ {
		u := DBUser{
			ID:        newID("u"),
			FirstName: clickerJudgeNames[i][0],
			LastName:  clickerJudgeNames[i][1],
			Email:     fmt.Sprintf("%s.a%d@example.com", strings.ToLower(clickerJudgeNames[i][0]), i+1),
		}
		db.Create(&u)
		clickerJudges = append(clickerJudges, u)
	}
	for i := 0; i < 6; i++ {
		u := DBUser{
			ID:        newID("u"),
			FirstName: evalJudgeNames[i][0],
			LastName:  evalJudgeNames[i][1],
			Email:     fmt.Sprintf("%s.b%d@example.com", strings.ToLower(evalJudgeNames[i][0]), i+1),
		}
		db.Create(&u)
		evalJudges = append(evalJudges, u)
	}

	contestID := newID("c")
	db.Create(&DBContest{
		ID: contestID, Name: "Indonesia National Yoyo Championships", Year: 2026,
		OwnerUserID: headJudge.ID, HeadJudgeUserID: headJudge.ID,
		// Hidden: kept out of listAllContests() so it doesn't clutter the
		// (now globally visible) contest list — still reachable directly
		// by ID for reference.
		Hidden: true,
	})

	stagesJSON, _ := json.Marshal([]calc.ScoringStage{calc.StagePrelim, calc.StageFinal})
	divisionID := newID("d")
	db.Create(&DBDivision{
		ID: divisionID, ContestID: contestID, Name: "3A", Stages: string(stagesJSON),
	})

	playerNames := [][2]string{
		{"Taro", "Yamada"}, {"Jane", "Smith"}, {"Budi", "Santoso"}, {"Mei", "Lin"},
		{"Wira", "Kusuma"}, {"Siti", "Aminah"}, {"Chen", "Wei"}, {"Aiko", "Tanaka"},
		{"Diego", "Santos"}, {"Maria", "Garcia"},
	}
	var players []DBPlayer
	for i, n := range playerNames {
		p := DBPlayer{ID: newID("p"), DivisionID: divisionID, Number: i + 1, Name: n[0] + " " + n[1]}
		db.Create(&p)
		players = append(players, p)
	}

	for _, stage := range []calc.ScoringStage{calc.StagePrelim, calc.StageFinal} {
		for i, u := range clickerJudges {
			db.Create(&DBJudgeAssignment{
				ID: newID("a"), ContestID: contestID, DivisionID: divisionID,
				Stage: string(stage), UserID: u.ID, Role: string(RoleClicker), Slot: i + 1,
			})
		}
		for i, u := range evalJudges {
			db.Create(&DBJudgeAssignment{
				ID: newID("a"), ContestID: contestID, DivisionID: divisionID,
				Stage: string(stage), UserID: u.ID, Role: string(RoleEvaluator), Slot: i + 1,
			})
		}
	}

	finalNets := [][6]int{
		{50, 52, 51, 40, 48, 53}, {55, 53, 54, 45, 50, 55}, {40, 41, 39, 35, 38, 42},
		{58, 57, 59, 50, 56, 58}, {45, 44, 46, 42, 43, 45}, {62, 61, 63, 60, 62, 64},
		{38, 37, 39, 36, 38, 40}, {51, 50, 52, 49, 50, 53}, {47, 46, 48, 45, 47, 49},
		{55, 54, 56, 53, 55, 57},
	}

	for i, p := range players {
		clickers := map[int]ClickerInput{}
		for j := 0; j < 6; j++ {
			clickers[j+1] = ClickerInput{Plus: finalNets[i][j], Minus: 0}
		}
		evals := map[int]map[string]float64{}
		for j := 0; j < 6; j++ {
			evals[j+1] = map[string]float64{
				"EXE": 4 + float64(i%2)*0.5,
				"CTL": 4,
				"TDV": 3.5 + float64(j%2)*0.5,
				"SEM": 4,
				"MU1": 4,
				"MU2": 3.5,
				"BDY": 4,
				"SHW": 4 - float64(i%2)*0.5,
			}
		}
		deductions := MajorDeductions{}
		if i == 2 {
			deductions.Stop = 1
		}
		clickersJSON, _ := json.Marshal(clickers)
		evalsJSON, _ := json.Marshal(evals)
		deductionsJSON, _ := json.Marshal(deductions)

		db.Create(&DBPlayerRawScore{
			DivisionID: divisionID, PlayerID: p.ID, Stage: string(calc.StageFinal),
			Clickers: string(clickersJSON), Deductions: string(deductionsJSON), Evals: string(evalsJSON),
		})
		db.Create(&DBPlayerRawScore{
			DivisionID: divisionID, PlayerID: p.ID, Stage: string(calc.StagePrelim),
			Clickers: "{}", Deductions: "{}", Evals: "{}",
		})
	}
}
