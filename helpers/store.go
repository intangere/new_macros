package helpers

var store = map[string]any{}

func NewOrGetArray(name string) []any {
	if entry, ok := store[name]; ok {
		if arr, ok := entry.([]any); ok {
			return arr
		} else {
			panic("Store key `"+name+"` is not an array!")
		}
	} else {
		arr := []any{}
		store[name] = arr
		return arr
	}
}

func UpdateArray(name string, array []any) {
	store[name] = array
}
