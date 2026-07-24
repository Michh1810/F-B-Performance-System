package financial

import "fmt"

const systemPrompt = "You are a financial analyst for a restaurant product team. " +
	"Analyze the provided POS/sales data and calculate the margin risk and demand outlook " +
	"for the menu item. Be concise and specific."

func buildPrompt(in Input) string {
	return fmt.Sprintf("Menu item: %s\n\nFinancial/POS data:\n%s", in.ItemName, in.FinancialData)
}
