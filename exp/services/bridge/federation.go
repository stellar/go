package main

type FederationResponse struct {
	Address   string `json:"stellar_address"`
	AccountID string `json:"account_id"`
	MemoType  string `json:"id"`
	Memo      string `json:"memo"`
}

func Federation(c *fiber.Ctx) error {
	t := c.Query("type")
	if t != "name" {
		return c.Status(400).JSON(fiber.Map{"error": "Param 'type' must be 'name'."})
	}
	q := c.Query("q")
	if q == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Param 'q' must be provided."})
	}
	domainSuffix := "*" + domain
	if !strings.HasSuffix(q, domainSuffix) {
		return c.Status(404).JSON(fiber.Map{"error": "No accounts found matching query."})
	}
	username := strings.TrimSuffix(q, domainSuffix)
	a := Account{}
	err := db.Model(&a).Where("username = ?", username).First()
	if err == pg.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "No accounts found matching query."})
	}
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(FederationResponse{
		Address:   q,
		AccountID: account,
		MemoType:  "id",
		Memo:      strconv.FormatInt(a.MemoID, 10),
	})
}
