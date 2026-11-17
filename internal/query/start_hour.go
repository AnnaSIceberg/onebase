package query

import "fmt"

func isStartOfHourCall(tokens []tok, at int) bool {
	return at+1 < len(tokens) && tokens[at].kind == tIdent &&
		lowerFast(tokens[at].val) == "началочаса" && tokens[at+1].kind == tLParen
}

func startOfHourEnd(tokens []tok, at int) int {
	depth := 0
	for i := at + 1; i < len(tokens); i++ {
		switch tokens[i].kind {
		case tLParen:
			depth++
		case tRParen:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func validateStartOfHourCalls(tokens []tok) error {
	for i := range tokens {
		if !isStartOfHourCall(tokens, i) {
			continue
		}
		end := startOfHourEnd(tokens, i)
		if end < 0 {
			return fmt.Errorf("НачалоЧаса: ожидается один аргумент даты")
		}
		args := splitTopLevelArgs(tokens[i+2 : end])
		if len(args) != 1 || len(args[0]) == 0 {
			return fmt.Errorf("НачалоЧаса: ожидается один аргумент даты")
		}
	}
	return nil
}
