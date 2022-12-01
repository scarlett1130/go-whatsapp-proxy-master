package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/leadstolink/go-whatsapp-proxy/app/controllers"
	"github.com/leadstolink/go-whatsapp-proxy/app/middlewares"
)

func Setup(app *fiber.App, controller *controllers.Controller) {
	app.Use(cors.New())
	app.Use("/api", middlewares.Auth)

	app.Get("/api/user/login", controller.Login)
	app.Get("/api/user/logout", controller.Logout)

	app.Get("/api/system/shutdown", controller.Shutdown)
	app.Get("/api/system/restart", controller.Restart)

	app.Post("/api/message/send", controller.SendMessage)

	app.Get("/api/message/last", controller.LastMessage)

	app.Get("/api/tool/check-number/:number", controller.NumberInfo)
}
