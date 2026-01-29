package main

import (
	"context"
	"log"
	"os"

	"telegram_webapp/internal/db"
	"telegram_webapp/internal/domain"
	"telegram_webapp/internal/repository"
	"telegram_webapp/internal/service"
)

func main() {
	// expects DATABASE_URL env var
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}

	pool := db.Connect(dsn)
	defer pool.Close()

	repo := repository.NewUserRepository(pool)
	ctx := context.Background()

	tgID := int64(1234567890)

	// try to find existing user
	existing, err := repo.GetByTgID(ctx, tgID)
	var u *domain.User
	if err == nil {
		u = existing
		log.Printf("user already exists id=%d\n", u.ID)
	} else {
		u = &domain.User{
			TgID:      tgID,
			Username:  "testuser",
			FirstName: "Tester",
		}

		if err := repo.Create(ctx, u); err != nil {
			log.Fatalf("create user failed: %v", err)
		}

		log.Printf("user created id=%d\n", u.ID)
	}

	// verify read
	u2, err := repo.GetByTgID(ctx, u.TgID)
	if err != nil {
		log.Fatalf("get by tg id failed: %v", err)
	}
	log.Printf("fetched user id=%d username=%s first_name=%s created_at=%v\n", u2.ID, u2.Username, u2.FirstName, u2.CreatedAt)

	// initialize JWT and print token
	service.InitJWT()
	token, err := service.GenerateJWT(u2.ID)
	if err != nil {
		log.Fatalf("failed to generate token: %v", err)
	}
	log.Printf("token=%s\n", token)
}
