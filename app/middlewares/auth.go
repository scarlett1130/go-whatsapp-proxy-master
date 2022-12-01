package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"os"
)

func Auth(c *fiber.Ctx) error {
	authKey := c.Query(`auth`)

	if authKey != os.Getenv("API_KEY") {
		return c.SendStatus(401)
	}

	return c.Next()
}
