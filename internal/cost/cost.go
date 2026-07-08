package cost

type Pricing struct {
	InputCostPer1K  float64
	OutputCostPer1K float64
}

// ModelPricing maps a model name to its pricing
var ModelPricing = map[string]Pricing{
	"gemini-1.5-pro": {
		InputCostPer1K:  0.00125, // $1.25 / 1M tokens
		OutputCostPer1K: 0.00375, // $3.75 / 1M tokens
	},
	"gpt-4o": {
		InputCostPer1K:  0.005,
		OutputCostPer1K: 0.015,
	},
}

// CalculateCost computes the cost of a request in USD.
func CalculateCost(model string, inputTokens, outputTokens int) float64 {
	pricing, exists := ModelPricing[model]
	if !exists {
		return 0.0
	}
	
	inputCost := (float64(inputTokens) / 1000.0) * pricing.InputCostPer1K
	outputCost := (float64(outputTokens) / 1000.0) * pricing.OutputCostPer1K
	
	return inputCost + outputCost
}
