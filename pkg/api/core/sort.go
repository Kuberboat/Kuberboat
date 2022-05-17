package core

type NodeTimeSlice []*Node

func (nodes NodeTimeSlice) Len() int {
	return len(nodes)
}

func (nodes NodeTimeSlice) Less(i, j int) bool {
	return nodes[i].CreationTimestamp.Before(nodes[j].CreationTimestamp)
}

func (nodes NodeTimeSlice) Swap(i, j int) {
	nodes[i], nodes[j] = nodes[j], nodes[i]
}
