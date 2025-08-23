package toolsystem

import "context"

type JSONType string

const (
	JSONString JSONType = "string"
	JSONNumber JSONType = "number"
	JSONObject JSONType = "object"
	JSONArray  JSONType = "array"
	JSONBool   JSONType = "boolean"
)

type ArgSpec struct {
	Name        string
	Type        JSONType
	Description string
	Required    bool
}

type ResultSpec struct {
	Name        string
	Type        JSONType
	Description string
}

type ToolSpec struct {
	Name        string
	Version     string
	Description string
	Args        []ArgSpec
	Result      []ResultSpec
	Tags        []string
}

type ToolHandler func(ctx context.Context, args map[string]any) (map[string]any, error)
