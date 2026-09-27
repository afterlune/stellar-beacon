package api

import "encoding/json"

func mapResponseDTO[T any](source any) (T, error) {
	var result T
	encoded, err := json.Marshal(source)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(encoded, &result); err != nil {
		return result, err
	}
	return result, nil
}
