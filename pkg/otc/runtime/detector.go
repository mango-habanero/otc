package runtime

import "sort"

// sortByPriority sorts runtimes by priority in descending order (highest first).
// In case of equal priority, runtimes maintain their detection order (stable sort).
func sortByPriority(runtimes []Runtime) {
	sort.SliceStable(runtimes, func(i, j int) bool {
		return runtimes[i].Priority > runtimes[j].Priority
	})
}
