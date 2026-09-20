package bot

import (
	"strconv"

	"github.com/go-telegram/bot/models"

	"lab042.ru/doomsdaycalc/internal/domain"
	"lab042.ru/doomsdaycalc/internal/i18n"
)

// mainKeyboard: persistent reply keyboard under the message input. Buttons map to
// core actions: add/remove money and view status. Labels come from the language
// catalog.
func mainKeyboard(m i18n.Messages) models.ReplyKeyboardMarkup {
	return models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{{Text: m.BtnWithdraw}, {Text: m.BtnAdd}},
			{{Text: m.BtnStatus}},
			{{Text: m.BtnOther}},
		},
		ResizeKeyboard: true, // fit the keyboard to the buttons
	}
}

// tiersKeyboard: the three saving tiers as inline buttons with "dep:min" | "dep:ok" |
// "dep:max" callback data. Not wired to onCallback yet.
// TODO: i need that? maybe for goal editing.
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

// goalsKeyboard: the users goals as inline buttons, one per row. Callback data is
// "<prefix>:<goal id>", so onCallback can tell which goal was picked and for what.
func goalsKeyboard(prefix string, goals []domain.Goal) models.InlineKeyboardMarkup {
	rows := make([][]models.InlineKeyboardButton, 0, len(goals))
	for _, g := range goals {
		rows = append(rows, []models.InlineKeyboardButton{{
			Text:         g.Title,
			CallbackData: prefix + ":" + strconv.FormatInt(g.ID, 10),
		}})
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: rows}
}
