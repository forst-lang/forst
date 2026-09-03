package main

func brookEdges() [][]string {
	return [][]string{[]string{"a", "b"}, []string{"c", "d"}}
}
func main() {
	rows := brookEdges()
	println(len(rows))
	println(len(rows[0]))
	println(rows[0][0])
}
