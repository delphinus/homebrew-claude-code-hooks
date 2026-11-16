package notify

// maxTitleRunes はサブタイトルに載せるタブタイトルの最大文字数。
// これを超えるタイトルは末尾を省略記号に置き換える。
const maxTitleRunes = 40

// untitled はタブタイトルが空のときの表示。
const untitled = "(無題)"

// truncateRunes shortens s to at most max runes, marking the cut with "…".
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
