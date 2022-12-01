package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/leadstolink/go-whatsapp-proxy/app/controllers"
	"github.com/leadstolink/go-whatsapp-proxy/app/routes"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	app := fiber.New()

	dbLog := waLog.Stdout("Database", os.Getenv("LOG_LEVEL"), true)

	dbContainer, err := sqlstore.New("sqlite3", "file:whatsappstore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	controller := controllers.NewController(dbContainer)
	defer controller.GetClient().Disconnect()

	routes.Setup(app, controller)

	if os.Getenv("AUTO_LOGIN") == `1` {
		if err := controller.Autologin(); err != nil {
			log.Fatal("Error auto connect WhatsApp")
		}

	}

	if err := app.Listen(fmt.Sprintf(":%s", os.Getenv("PORT"))); err != nil {
		log.Fatal("error starting http server")
	}
}
