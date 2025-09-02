package bot

import "github.com/NastyaGoryachaya/crypto-rate-service/internal/ports/errcode"

func translateBotError(code errcode.Code) string {
	switch code {
	case errcode.NotFoundCoins:
		return "Валюта не найдена"
	case errcode.NotFoundPrices:
		return "Данные о цене не найдены"
	default:
		return "Внутренняя ошибка сервиса, попробуйте позже"
	}
}
