// loadtest-seed creates disposable authenticated fixtures only in *_test databases.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"unihub-workshop/internal/config"
	"unihub-workshop/internal/database"
	"unihub-workshop/internal/middleware"
	"unihub-workshop/internal/model"
)

func main() {
	cfg := config.Load()
	if !strings.HasSuffix(cfg.DBName, "_test") {
		log.Fatal("fixtures require a disposable database ending in _test")
	}
	count := 50
	if value := os.Getenv("FIXTURE_USERS"); value != "" {
		var err error
		count, err = strconv.Atoi(value)
		if err != nil || count < 1 || count > 12000 {
			log.Fatal("FIXTURE_USERS must be 1..12000")
		}
	}
	pool := database.NewPostgresPool(cfg)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx)
	hash, err := bcrypt.GenerateFromPassword([]byte("Demo123456!"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	// A new workshop per run lets repeated demo gates register the same users
	// without colliding with tickets from the previous run.
	workshopID := uuid.NewString()
	_, err = tx.Exec(ctx, `INSERT INTO workshops(id,title,description,speaker,room,start_time,end_time,registration_start_time,registration_end_time,capacity,available_seats,status) VALUES($1,'DevOps load test','Disposable CI fixture','Demo','CI',now()+interval '7 days',now()+interval '7 days 2 hours',now()-interval '1 hour',now()+interval '1 day',$2,$2,'PUBLISHED') ON CONFLICT(id) DO NOTHING`, workshopID, count)
	if err != nil {
		log.Fatal(err)
	}
	sessions := make([]map[string]string, 0, count)
	for i := 1; i <= count; i++ {
		studentID := fmt.Sprintf("demo%05d", i)
		var id string
		var version int
		err = tx.QueryRow(ctx, `INSERT INTO users(user_id,password_hash,full_name,email,role) VALUES($1,$2,$3,$4,'STUDENT') ON CONFLICT(user_id) DO UPDATE SET full_name=excluded.full_name RETURNING id,auth_version`, studentID, string(hash), "Demo "+studentID, studentID+"@example.test").Scan(&id, &version)
		if err != nil {
			log.Fatal(err)
		}
		token, err := middleware.GenerateJWT(cfg.AuthSecret, id, model.RoleStudent, version)
		if err != nil {
			log.Fatal(err)
		}
		sessions = append(sessions, map[string]string{"student_id": studentID, "token": token})
	}
	if err = tx.Commit(ctx); err != nil {
		log.Fatal(err)
	}
	payload, err := json.Marshal(map[string]interface{}{"workshop_id": workshopID, "sessions": sessions})
	if err != nil {
		log.Fatal(err)
	}
	file := os.Getenv("FIXTURE_FILE")
	if file == "" {
		file = "/tmp/unihub-sessions.json"
	}
	if err = os.WriteFile(file, payload, 0600); err != nil {
		log.Fatal(err)
	}
	log.Printf("Created %d authenticated fixtures", count)
}
