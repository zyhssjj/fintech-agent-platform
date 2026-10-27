package approval

import (
	"encoding/json"

	"gorm.io/datatypes"
)

func mustJSON(value any) datatypes.JSON {
	if value == nil {
		return datatypes.JSON([]byte("{}"))
	}
	data, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(data)
}

func datatypesJSONTo[T any](value datatypes.JSON, target *T) error {
	if len(value) == 0 {
		return nil
	}
	return json.Unmarshal(value, target)
}

func toString(value any) string {
	text, _ := value.(string)
	return text
}

func toStrings(value any) []string {
	items, _ := value.([]string)
	if items == nil {
		return []string{}
	}
	return items
}
