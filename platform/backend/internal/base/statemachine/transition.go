package statemachine

func canTransition(transitions map[string][]string, from, to string) bool {
	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	for _, state := range allowed {
		if state == to {
			return true
		}
	}
	return false
}
