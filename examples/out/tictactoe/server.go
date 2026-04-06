package main

import "errors"
import "fmt"
import os "os"
// GameState: TypeDefShapeExpr({cells: Array(String), nextPlayer: String, status: String})
type GameState struct {
	cells      []string `json:"cells"`
	nextPlayer string   `json:"nextPlayer"`
	status     string   `json:"status"`
}
// MoveRequest: TypeDefShapeExpr({state: GameState, row: Int, col: Int})
type MoveRequest struct {
	col   int       `json:"col"`
	row   int       `json:"row"`
	state GameState `json:"state"`
}
// MoveResponse: TypeDefShapeExpr({message: String, state: GameState, ok: Bool})
type MoveResponse struct {
	message string    `json:"message"`
	ok      bool      `json:"ok"`
	state   GameState `json:"state"`
}
// T_CQ83zP8NNan: TypeDefShapeExpr({})
type T_CQ83zP8NNan struct {
}
// T_JTaataA3nDc: TypeDefShapeExpr({})
type T_JTaataA3nDc struct {
}
// T_PNXTj8VxMub: TypeDefShapeExpr({})
type T_PNXTj8VxMub struct {
}
// T_iw8no2aCk8H: TypeDefShapeExpr({})
type T_iw8no2aCk8H struct {
}

func ApplyMove(req MoveRequest) (MoveResponse, error) {
	row := req.row
	if row <= -1 {
		return MoveResponse{state: GameState{nextPlayer: "", status: "", cells: nil}, ok: false, message: ""}, errors.New("assertion failed: Int.GreaterThan(-1)")
	}
	if row >= 3 {
		return MoveResponse{ok: false, message: "", state: GameState{cells: nil, nextPlayer: "", status: ""}}, errors.New("assertion failed: Int.LessThan(3)")
	}
	col := req.col
	if col <= -1 {
		return MoveResponse{ok: false, message: "", state: GameState{cells: nil, nextPlayer: "", status: ""}}, errors.New("assertion failed: Int.GreaterThan(-1)")
	}
	if col >= 3 {
		return MoveResponse{state: GameState{status: "", cells: nil, nextPlayer: ""}, ok: false, message: ""}, errors.New("assertion failed: Int.LessThan(3)")
	}
	idx := cellIndex(row, col)
	cellEmpty := req.state.cells[idx] == ""
	if !cellEmpty {
		return MoveResponse{state: GameState{cells: nil, nextPlayer: "", status: ""}, ok: false, message: ""}, errors.New("assertion failed: Bool.True()")
	}
	next := setCell(cloneCells(req.state.cells), idx, req.state.nextPlayer)
	np := opponent(req.state.nextPlayer)
	return MoveResponse{state: GameState{cells: next, nextPlayer: np, status: req.state.status}, ok: true, message: "ok"}, nil
}
func NewGame() GameState {
	board := emptyBoard()
	return GameState{cells: board, nextPlayer: "X", status: "playing"}
}
func PlayMove(req MoveRequest) (MoveResponse, error) {
	return ApplyMove(req)
}
func boardFull(cells []string) bool {
	for i := 0; i < 9; i++ {
		if cells[i] == "" {
			return false
		}
	}
	return true
}
func cellIndex(row int, col int) int {
	return row*3 + col
}
func cloneCells(cells []string) []string {
	out := emptyBoard()
	copy(out, cells)
	return out
}
func emptyBoard() []string {
	return []string{"", "", "", "", "", "", "", "", ""}
}
func getCell(cells []string, idx int) string {
	return cells[idx]
}
func invalidMove(msg string) error {
	return errors.New(msg)
}
func lineWinner(cells []string, a int, b int, c int) string {
	x0 := cells[a]
	if x0 == "" {
		return ""
	}
	if x0 == cells[b] && x0 == cells[c] {
		return x0
	}
	return ""
}
func main() {
	fmt.Println("=== tic-tac-toe merged-package example ===")
	g := NewGame()
	fmt.Println("initial | next:", g.nextPlayer, "| status:", g.status)
	a0 := getCell(g.cells, 0)
	a1 := getCell(g.cells, 1)
	a2 := getCell(g.cells, 2)
	fmt.Println("top row |", a0, a1, a2)
	move, errMove := PlayMove(MoveRequest{state: g, row: 1, col: 2})
	if errMove != nil {
		fmt.Println(errMove.Error())
		os.Exit(1)
	}
	fmt.Println("after X @ (1,2) | ok:", move.ok, "| msg:", move.message)
	fmt.Println("then    | next:", move.state.nextPlayer, "| status:", move.state.status)
}
func opponent(p string) string {
	if p == "X" {
		return "O"
	}
	return "X"
}
func setCell(cells []string, idx int, val string) []string {
	out := emptyBoard()
	copy(out, cells)
	out[idx] = val
	return out
}
func winnerOrEmpty(cells []string) string {
	lineA := []int{0, 3, 6, 0, 1, 2, 0, 2}
	lineB := []int{1, 4, 7, 3, 4, 5, 4, 4}
	lineC := []int{2, 5, 8, 6, 7, 8, 8, 6}
	for i := 0; i < 8; i++ {
		w := lineWinner(cells, lineA[i], lineB[i], lineC[i])
		if w != "" {
			return w
		}
	}
	return ""
}
