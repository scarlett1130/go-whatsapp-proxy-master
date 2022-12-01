package controllers

import "github.com/gofiber/fiber/v2"

func (k *Controller) NumberInfo(c *fiber.Ctx) error {
	number := `+` + c.Params(`number`)

	info, _ := k.client.IsOnWhatsApp([]string{number})

	return c.JSON(info)
}
