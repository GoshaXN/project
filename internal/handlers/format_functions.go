package handlers

import (
	"fmt"
	"project/internal/models"
)

// formatCategory - форматирование категории для вывода
func (h *Handler) formatCategory(category models.Category) string {
	return fmt.Sprintf("Категория: %s (%v) \nОписание: %s\nАктивность: %v\n\n",
		category.Name, category.ID, category.Description, category.IsActive)
}

// formatOrder - форматирование категории для вывода
func (h *Handler) formatOrder(order models.Order) string { //вывод заказа
	users, err := h.userService.SearchUser(fmt.Sprintf("%d", order.UserID))
	if err != nil {
		return fmt.Sprintf("Пользователь: %d не найден", order.UserID)
	}
	//fmt.Printf("users: %v\n", users)
	user := users[0]
	return fmt.Sprintf("Заказ #%d\nПользователь: %s (%d)\nСумма: %.2f\nСтатус: %s\nДата создания: %s\n",
		order.ID, user.FirstName, order.UserID, order.Amount, order.Status, order.CreatedAt.Format("02.01.2006 15:04"))
}

// formatProduct - форматирование категории для вывода
func (h *Handler) formatProduct(product models.Product) string {
	return fmt.Sprintf("ID: %d\nНазвание: %s\nОписание: %s\nЦена: %.2f\nКоличество: %d\nКатегория ID: %d\nВес: %.2f\nВкус: %s\nБренд: %s\nПорций: %d\nАктивен: %v\nСоздан: %s\n\n",
		product.ID, product.Name, product.Description, product.Price, product.Quantity,
		product.Category_id, product.Weight, product.Flavor, product.Brand, product.Servings,
		product.IsActive, product.CreatedAt.Format("02.01.2006 15:04"))
}

// formatUser - форматирование юзера для вывода
func (h *Handler) formatUser(user models.User) string { // вывод юзера
	roleText := "Покупатель"
	if user.Role == "admin" {
		roleText = "admin"
	}
	return fmt.Sprintf("%s: %s(ID=%d)\nТелеграмм: %s (ID=%d)\nИмя: %s\nТелефон: %s\nПочта: %s\nДата регистрации: %s\n",
		roleText, user.Username, user.ID, user.Username, user.TelegramID, user.FirstName,
		user.Phone, user.Email, user.CreatedAt.Format("02.01.2006"))
}

// formatOrderPagination - форматирование пагинации заказов
func (h *Handler) formatOrderPagination(order models.Order) string {
	return fmt.Sprintf("Заказ #%d\nСумма: %.2f руб.\nСтатус: %s\nДата создания: %s\n",
		order.ID, order.Amount, order.Status,
		order.CreatedAt.Format("02.01.2006 15:04"))
}

// formatCart - форматирование заказа последнего
func (h *Handler) formatCart(order *models.Order, items []models.OrderItem) string { //вывод корзины с товарами
	var response string
	if order == nil {
		response = "Нет заказов!"
		return response
	}
	if len(items) == 0 {
		response = fmt.Sprintf("Заказ #%d\n\nПустая корзина", order.ID)
		return response
	}

	total := 0.0
	for _, item := range items {
		sum := item.Price * float64(item.Quantity)
		total += sum
		product, err := h.productService.SearchProduct(fmt.Sprintf("%d", item.ProductID))
		productName := "Товар"
		var flavor string
		if err == nil && product != nil {
			productName = product[0].Name
			flavor = product[0].Flavor
		}

		response += fmt.Sprintf("Товар: %s (%s) %dшт. - %.2f руб.\n",
			productName, flavor, item.Quantity, sum)
	}

	response += fmt.Sprintf("\nОбщая сумма: %.2f руб.", total)
	response += fmt.Sprintf("\nНомер заказа: #%d", order.ID)
	response += fmt.Sprintf("\nСтатус заказа: #%s", order.Status)

	return response
}
