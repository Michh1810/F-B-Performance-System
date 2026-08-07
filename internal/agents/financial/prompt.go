package financial

import "fmt"

const systemPrompt = "You are a financial analyst for a restaurant product team. " +
	"Analyze the provided demand forecast and calculate the margin risk and demand outlook " +
	"for the menu item. Be concise and specific."

func buildPrompt(in Input, forecast ForecastResult) string {
	return fmt.Sprintf(
		"Menu item: %s\n\nDemand forecast (next %d days, model: %s, AI adjustment: %s):\n"+
			"- Baseline daily units: %.2f\n"+
			"- Forecasted units: %d\n"+
			"- Forecasted revenue: $%.2f\n"+
			"- Projected profit: $%.2f",
		in.ItemName, forecast.ForecastWindowDays, forecast.Model, forecast.AIAdjustmentStatus,
		forecast.BaselineUnits, forecast.ForecastedUnits,
		float64(forecast.ForecastedRevenueCents)/100, float64(forecast.ProjectedProfitCents)/100,
	)
}
