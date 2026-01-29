package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"

	"telegram_webapp/internal/db"
	"telegram_webapp/internal/domain"
	"telegram_webapp/internal/repository"
	"telegram_webapp/internal/service"
)

func main() {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        log.Fatal("DATABASE_URL not set")
    }
    jwtSecret := os.Getenv("JWT_SECRET")
    if jwtSecret == "" {
        log.Fatal("JWT_SECRET not set")
    }

    port := os.Getenv("APP_PORT")
    if port == "" {
        port = "8080"
    }

    pool := db.Connect(dsn)
    defer pool.Close()

    ur := repository.NewUserRepository(pool)
    ctx := context.Background()

    // prepare users
    uA, err := ur.GetByTgID(ctx, 3001)
    if err != nil {
        uA = &domain.User{TgID: 3001, Username: "smokeA", FirstName: "A"}
        if err := ur.Create(ctx, uA); err != nil {
            log.Fatalf("create userA: %v", err)
        }
    }

    uB, err := ur.GetByTgID(ctx, 3002)
    if err != nil {
        uB = &domain.User{TgID: 3002, Username: "smokeB", FirstName: "B"}
        if err := ur.Create(ctx, uB); err != nil {
            log.Fatalf("create userB: %v", err)
        }
    }

    // init jwt and generate tokens
    service.InitJWT()
    tokenA, err := service.GenerateJWT(uA.ID)
    if err != nil {
        log.Fatalf("gen token A: %v", err)
    }
    tokenB, err := service.GenerateJWT(uB.ID)
    if err != nil {
        log.Fatalf("gen token B: %v", err)
    }

    dialer := websocket.DefaultDialer

    // use 127.0.0.1 to prefer IPv4 (avoid resolving to [::1])
    wsURLA := fmt.Sprintf("ws://127.0.0.1:%s/ws?token=%s", port, tokenA)
    wsURLB := fmt.Sprintf("ws://127.0.0.1:%s/ws?token=%s", port, tokenB)

    connA, _, err := dialer.Dial(wsURLA, nil)
    if err != nil {
        log.Fatalf("dial A: %v", err)
    }
    defer connA.Close()

    connB, _, err := dialer.Dial(wsURLB, nil)
    if err != nil {
        log.Fatalf("dial B: %v", err)
    }
    defer connB.Close()

    // wait for start (drain state messages)
    drainUntilStart := func(conn *websocket.Conn) {
        deadline := time.Now().Add(2 * time.Second)
        for time.Now().Before(deadline) {
            conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
            _, msg, err := conn.ReadMessage()
            if err != nil {
                continue
            }
            var obj map[string]any
            _ = json.Unmarshal(msg, &obj)
            if t, ok := obj["type"].(string); ok && t == "start" {
                return
            }
        }
    }

    drainUntilStart(connA)
    drainUntilStart(connB)

    // send moves
    if err := connA.WriteMessage(websocket.TextMessage, []byte(`{"type":"move","value":"rock"}`)); err != nil {
        log.Fatalf("write A: %v", err)
    }
    if err := connB.WriteMessage(websocket.TextMessage, []byte(`{"type":"move","value":"scissors"}`)); err != nil {
        log.Fatalf("write B: %v", err)
    }

    // read results
    readResult := func(conn *websocket.Conn, name string) {
        conn.SetReadDeadline(time.Now().Add(3 * time.Second))
        _, msg, err := conn.ReadMessage()
        if err != nil {
            log.Printf("%s read error: %v", name, err)
            return
        }
        log.Printf("%s got: %s", name, string(msg))
    }

    readResult(connA, "A")
    readResult(connB, "B")

    log.Println("smoke test finished")
}
