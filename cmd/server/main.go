package main

import (
	"context"
	"log"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/config"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
)

func main() {
	cfg := config.MustLoad()
	db := database.MustConnect(cfg.DatabaseURL)
	_ = db
	es := eventstore.NewPgStore(db)
	_ = es
	runner := eventstore.NewProjectionRunner(db, es)
	runner.Start(context.Background())
	log.Printf("server would start on %s", cfg.ListenAddr)
}
