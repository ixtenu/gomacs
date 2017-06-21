package main

import (
	"fmt"
	"github.com/zyedidia/highlight"
	"strconv"
	"strings"
	"unicode/utf8"
)

func editorScroll(sx, sy int) {
	Global.CurrentB.rx = 0
	if Global.CurrentB.cy < Global.CurrentB.NumRows {
		Global.CurrentB.rx = editorRowCxToRx(Global.CurrentB.Rows[Global.CurrentB.cy])
	}

	if Global.CurrentB.cy < Global.CurrentB.rowoff {
		Global.CurrentB.rowoff = Global.CurrentB.cy
	}
	if Global.CurrentB.cy >= Global.CurrentB.rowoff+sy {
		Global.CurrentB.rowoff = Global.CurrentB.cy - sy + 1
	}
	if Global.CurrentB.rx < Global.CurrentB.coloff {
		Global.CurrentB.coloff = Global.CurrentB.rx
	}
	if Global.CurrentB.rx >= Global.CurrentB.coloff+sx {
		Global.CurrentB.coloff = Global.CurrentB.rx - sx + 1
	}
}

func editorCentreView() {
	rowoff := Global.CurrentB.cy - (Global.CurrentBHeight / 2)
	if rowoff >= 0 {
		Global.CurrentB.rowoff = rowoff
	}
}

func MoveCursor(x, y int) {
	// Initial position of the cursor
	icx, icy := Global.CurrentB.cx, Global.CurrentB.cy
	// Regular cursor movement - most cases
	realline := icy < Global.CurrentB.NumRows && Global.CurrentB.NumRows != 0
	nx, ny := icx+x, icy+y
	if realline && icx <= Global.CurrentB.Rows[icy].Size {
		if x >= 1 {
			_, rs := utf8.DecodeRuneInString(Global.CurrentB.Rows[icy].Data[icx:])
			nx = icx + rs
		} else if x <= -1 {
			_, rs :=
				utf8.DecodeLastRuneInString(Global.CurrentB.Rows[icy].Data[:icx])
			nx = icx - rs
		}
	}
	if nx >= 0 && ny < Global.CurrentB.NumRows && realline && nx <= Global.CurrentB.Rows[icy].Size {
		Global.CurrentB.cx = nx
	}
	if ny >= 0 && ny <= Global.CurrentB.NumRows {
		Global.CurrentB.cy = ny
	}

	// Edge cases
	realline = Global.CurrentB.cy < Global.CurrentB.NumRows && Global.CurrentB.NumRows != 0
	if x < 0 && Global.CurrentB.cy > 0 && icx == 0 {
		// Left at the beginning of a line
		Global.CurrentB.cy--
		MoveCursorToEol()
	} else if realline && y == 0 && icx == Global.CurrentB.Rows[Global.CurrentB.cy].Size && x > 0 {
		// Right at the end of a line
		Global.CurrentB.cy++
		MoveCursorToBol()
	} else if realline && Global.CurrentB.cx > Global.CurrentB.Rows[Global.CurrentB.cy].Size {
		// Snapping to the end of the line when coming from a longer line
		MoveCursorToEol()
	} else if !realline && y == 1 {
		// Moving cursor down to the EOF
		MoveCursorToBol()
	}
}

func MoveCursorToEol() {
	if Global.CurrentB.cy < Global.CurrentB.NumRows {
		Global.CurrentB.cx = Global.CurrentB.Rows[Global.CurrentB.cy].Size
	}
}

func MoveCursorToBol() {
	Global.CurrentB.cx = 0
}

func MovePage(back bool, sy int) {
	for i := 0; i < sy; i++ {
		if back {
			MoveCursor(0, -1)
		} else {
			MoveCursor(0, 1)
		}
	}
}

func MoveCursorBackPage() {
	_, sy := GetScreenSize()
	Global.CurrentB.cy = Global.CurrentB.rowoff
	MovePage(true, sy)
}

func MoveCursorForthPage() {
	_, sy := GetScreenSize()
	Global.CurrentB.cy = Global.CurrentB.rowoff + sy - 1
	if Global.CurrentB.cy > Global.CurrentB.NumRows {
		Global.CurrentB.cy = Global.CurrentB.NumRows
	}
	MovePage(false, sy)
}

// HACK: Go does not have static variables, so these have to go in global state.
var last_match int = -1
var direction int = 1
var saved_hl_line int
var saved_hl highlight.LineMatch = nil

func editorFindCallback(query string, key string) {
	Global.Input = query
	if saved_hl != nil {
		Global.CurrentB.Rows[saved_hl_line].HlMatches = saved_hl
		saved_hl = nil
	}
	if key == "C-s" {
		direction = 1
	} else if key == "C-r" {
		direction = -1
		//If we cancelled or finished...
	} else if key == "C-c" || key == "C-g" || key == "RET" {
		if key == "C-c" || key == "C-g" {
			Global.Input = "Cancelled search."
		}
		//...outta here!
		last_match = -1
		direction = 1
		return
	} else {
		last_match = -1
		direction = 1
	}

	if last_match == -1 {
		direction = 1
	}
	current := last_match
	for range Global.CurrentB.Rows {
		current += direction
		if current == -1 {
			current = Global.CurrentB.NumRows - 1
		} else if current == Global.CurrentB.NumRows {
			current = 0
		}
		row := Global.CurrentB.Rows[current]
		match := strings.Index(row.Render, query)
		if match > -1 {
			last_match = current
			Global.CurrentB.cy = current
			Global.CurrentB.cx = editorRowRxToCx(row, match)
			Global.CurrentB.rowoff = Global.CurrentB.NumRows
			saved_hl_line = current
			saved_hl = make(highlight.LineMatch)
			for k, v := range row.HlMatches {
				saved_hl[k] = v
			}
			var c highlight.Group
			if row.HlMatches != nil {
				row.HlMatches[match] = 255
				ql := len(query)
				for i := 0; i <= match+ql; i++ {
					if i >= match {
						row.HlMatches[i] = 255
					}
					if saved_hl[i] != 0 {
						c = saved_hl[i]
					}
				}
				if ql == 0 {
					row.HlMatches[match] = saved_hl[match]
				} else {
					row.HlMatches[match+ql] = c
				}
			}
			break
		}
	}
}

func editorFind() {
	saved_cx := Global.CurrentB.cx
	saved_cy := Global.CurrentB.cy
	saved_co := Global.CurrentB.coloff
	saved_ro := Global.CurrentB.rowoff

	query := editorPrompt("Search", editorFindCallback)

	if query == "" {
		//Search cancelled, go back to where we were
		Global.CurrentB.cx = saved_cx
		Global.CurrentB.cy = saved_cy
		Global.CurrentB.coloff = saved_co
		Global.CurrentB.rowoff = saved_ro
	}
}

func doQueryReplace() {
	orig := editorPrompt("Find", nil)
	if orig == "" {
		Global.Input = "Can't query-replace with an empty query"
		return
	}
	replace := editorPrompt("Replace "+orig+" with", nil)
	ql := len(orig)
	for cy, row := range Global.CurrentB.Rows {
		match := strings.Index(row.Render, orig)
		if match != -1 {
			Global.CurrentB.cy = cy
			Global.CurrentB.cx = editorRowRxToCx(row, match)
			Global.CurrentB.rowoff = Global.CurrentB.NumRows
			saved_hl_line = cy
			saved_hl = make(highlight.LineMatch)
			for k, v := range row.HlMatches {
				saved_hl[k] = v
			}
			var c highlight.Group
			if row.HlMatches != nil {
				row.HlMatches[match] = 255

				for i := 0; i <= match+ql; i++ {
					if i >= match {
						row.HlMatches[i] = 255
					}
					if saved_hl[i] != 0 {
						c = saved_hl[i]
					}
				}
				if ql == 0 {
					row.HlMatches[match] = saved_hl[match]
				} else {
					row.HlMatches[match+ql] = c
				}
			}
			yes, err := editorYesNoPrompt("Replace with "+replace+"?", false)
			if err != nil {
				row.HlMatches = saved_hl
				return
			} else if yes {
				Global.CurrentB.Dirty = true
				editorAddUndo(false, 0, row.Size, cy, cy, row.Data)
				row.Data = strings.Replace(row.Data, orig, replace, -1)
				row.Size = len(row.Data)
				editorAddUndo(true, 0, row.Size, cy, cy, row.Data)
				editorUpdateRow(row, Global.CurrentB)
			} else {
				row.HlMatches = saved_hl
			}
		}
	}
}

func doReplaceString() {
	orig := editorPrompt("Find", nil)
	if orig == "" {
		Global.Input = "Can't string-replace with an empty query"
		return
	}
	replace := editorPrompt("Replace "+orig+" with", nil)
	matches := 0
	lines := 0
	ql := len(orig)
	nl := len(replace)
	for cy, row := range Global.CurrentB.Rows {
		match := strings.LastIndex(row.Render, orig)
		if match != -1 {
			count := strings.Count(row.Render, orig)
			matches += count
			lines++
			Global.CurrentB.cy = cy
			Global.CurrentB.cx = editorRowRxToCx(row, match+ql-(count*(ql-nl)))
			Global.CurrentB.rowoff = Global.CurrentB.NumRows
			Global.CurrentB.Dirty = true
			editorAddUndo(false, 0, row.Size, cy, cy, row.Data)
			row.Data = strings.Replace(row.Data, orig, replace, -1)
			row.Size = len(row.Data)
			editorAddUndo(true, 0, row.Size, cy, cy, row.Data)
			editorUpdateRow(row, Global.CurrentB)
		}
	}
	if matches > 0 {
		Global.Input = fmt.Sprintf("Replaced %d occurences on %d lines",
			matches, lines)
	} else {
		Global.Input = "No matches found"
	}
}

func gotoLine() {
	line, err := strconv.Atoi(editorPrompt("Go to line", nil))
	if err != nil {
		Global.Input = "Cancelled."
		return
	}
	line--
	if line < 0 {
		line = 0
	} else if line > Global.CurrentB.NumRows {
		line = Global.CurrentB.NumRows
	}
	Global.CurrentB.cy = line
	Global.Input = "Jumping to line " + strconv.Itoa(line+1)
}

func gotoChar() {
	line, err := strconv.Atoi(editorPrompt("Go to char", nil))
	if err != nil {
		Global.Input = "Cancelled."
		return
	}
	if Global.CurrentB.cy == Global.CurrentB.NumRows {
		return
	}
	datalen := len(Global.CurrentB.Rows[Global.CurrentB.cy].Data)
	if line < 0 {
		line = 0
	} else if line >= datalen {
		line = datalen
	}
	Global.CurrentB.cx = line
	Global.Input = "Jumping to char " + strconv.Itoa(line)
}
