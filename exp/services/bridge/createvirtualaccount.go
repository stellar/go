package main

type CreateVirtualAccountRequest struct {
	Username string `json:"username"`
}

type CreateVirtualAccountResponse struct {
	Address string `json:"address"`
}

func CreateVirtualAccount(c *fiber.Ctx) error {
	req := &CreateVirtualAccountRequest{}
	err := c.BodyParser(&req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	a := Account{Username: req.Username}
	_, err = db.Model(&a).Insert()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(CreateVirtualAccountResponse{
		Address: a.Username + "*" + domain,
	})
}
