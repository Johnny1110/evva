package main

import (
	"log/slog"
	"os"

	config "github.com/johnny1110/evva/configs"
	"github.com/johnny1110/evva/internal/greeter"
	"github.com/johnny1110/evva/internal/logger"
	"github.com/johnny1110/evva/pkg/common"
	"github.com/joho/godotenv"
)

func main() {
	// load param from .env
	_ = godotenv.Load()

	// setup log
	mainAgentID := common.GenUUID()
	logAgent1, _ := logger.OfAgent("", mainAgentID)
	slog.SetDefault(logAgent1)

	name := "World"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	logAgent1.Debug("preparing greeting", "name", name)
	greeting := greeter.Greet(name)
	logAgent1.Info("greeting ready", "greeting", greeting)

	logAgent2, _ := logger.OfAgent(mainAgentID, common.GenUUID())
	logAgent2.Debug("sub agent greeting", "name", name)
	logAgent2.Info("sub agent greeting", "name", name)

	print(config.Get().LogFormat)
}
