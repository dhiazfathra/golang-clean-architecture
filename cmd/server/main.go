package main

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/dhiazfathra/golang-clean-architecture/pkg/module/auth"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/config"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/database"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/eventstore"
	"github.com/dhiazfathra/golang-clean-architecture/pkg/platform/session"
)

func main() {
	cfg := config.MustLoad()
	db := database.MustConnect(cfg.DatabaseURL)
	vk := session.MustConnectValkey(cfg.ValkeyURL)
	es := eventstore.NewPgStore(db)

	sessionStore := session.NewValkeyStore(vk)
	hasher := auth.NewBcryptHasher()
	// userSvc wired in M5; pass nil placeholder until then
	authSvc := auth.NewService(sessionStore, nil, hasher)

	runner := eventstore.NewProjectionRunner(db, es)
	runner.Start(context.Background())

	e := echo.New()
	public := e.Group("")
	protected := e.Group("")
	protected.Use(session.RequireSession(sessionStore))

	auth.RegisterRoutes(public, protected, auth.NewHandler(authSvc))

	e.Logger.Fatal(e.Start(cfg.ListenAddr))
}
