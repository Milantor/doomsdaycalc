package bot

import (
	"github.com/go-telegram/bot/models"

	"lab042.ru/doomsdaycalc/internal/i18n"
)

// mainKeyboard: persistent reply keyboard under the message input. Buttons map to
// core actions: add/remove money and view status. Labels come from the language
// catalog.
func mainKeyboard(m i18n.Messages) models.ReplyKeyboardMarkup {
	return models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{{Text: m.BtnAdd}, {Text: m.BtnWithdraw}},
			{{Text: m.BtnStatus}},
		},
		ResizeKeyboard: true, // fit the keyboard to the buttons
	}
}

// tiersKeyboard: the three saving tiers as inline buttons. Callback data uses the
// "dep:" prefix that onCallback parses: "dep:min" | "dep:ok" | "dep:max".
func tiersKeyboard(m i18n.Messages) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: m.BtnTierMin, CallbackData: "dep:min"},
				{Text: m.BtnTierOK, CallbackData: "dep:ok"},
				{Text: m.BtnTierMax, CallbackData: "dep:max"},
			},
		},
	}
}
