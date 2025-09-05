package toolsystem

import (
	"context"
	"fmt"
	"time"
)

// Example tool that demonstrates the new tool system usage
func CreateExampleTool() (Tool, error) {
	return NewToolBuilder("get_weather", "1.0.0", "Get weather information for a location").
		AddStringParameter("location", "The location to get weather for", true).
		AddStringParameter("units", "Temperature units", false, "celsius", "fahrenheit").
		AddBooleanParameter("include_forecast", "Include 7-day forecast", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			location, ok := args["location"].(string)
			if !ok {
				return nil, fmt.Errorf("location parameter is required and must be a string")
			}

			units := "celsius"
			if u, exists := args["units"]; exists {
				if unitStr, ok := u.(string); ok {
					units = unitStr
				}
			}

			includeForecast := false
			if f, exists := args["include_forecast"]; exists {
				if forecast, ok := f.(bool); ok {
					includeForecast = forecast
				}
			}

			// Simulate weather API call
			result := map[string]any{
				"location":    location,
				"temperature": 22,
				"units":       units,
				"conditions":  "sunny",
				"timestamp":   time.Now().Format(time.RFC3339),
			}

			if includeForecast {
				result["forecast"] = []map[string]any{
					{"day": "tomorrow", "temperature": 25, "conditions": "partly cloudy"},
					{"day": "day_after", "temperature": 20, "conditions": "rainy"},
				}
			}

			return result, nil
		}).
		AddTags("weather", "api", "utility").
		Build()
}

// Example of registering multiple tools
func RegisterExampleTools(registry Registry) error {
	// Register weather tool
	weatherTool, err := CreateExampleTool()
	if err != nil {
		return fmt.Errorf("failed to create weather tool: %w", err)
	}

	if err := registry.Register(weatherTool); err != nil {
		return fmt.Errorf("failed to register weather tool: %w", err)
	}

	// Register a simple calculator tool
	calcTool, err := NewToolBuilder("calculate", "1.0.0", "Perform basic arithmetic calculations").
		AddNumberParameter("a", "First number", true).
		AddNumberParameter("b", "Second number", true).
		AddStringParameter("operation", "Operation to perform", true, "add", "subtract", "multiply", "divide").
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			a, aOk := args["a"].(float64)
			b, bOk := args["b"].(float64)
			operation, opOk := args["operation"].(string)

			if !aOk || !bOk || !opOk {
				return nil, fmt.Errorf("invalid parameters")
			}

			var result float64
			switch operation {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return nil, fmt.Errorf("division by zero")
				}
				result = a / b
			default:
				return nil, fmt.Errorf("unsupported operation: %s", operation)
			}

			return map[string]any{
				"result":    result,
				"operation": fmt.Sprintf("%v %s %v", a, operation, b),
			}, nil
		}).
		AddTags("math", "utility").
		Build()

	if err != nil {
		return fmt.Errorf("failed to create calculator tool: %w", err)
	}

	if err := registry.Register(calcTool); err != nil {
		return fmt.Errorf("failed to register calculator tool: %w", err)
	}

	return nil
}
