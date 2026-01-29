package ws

//Определяет результат player1 vs player2
func decide(moveA, moveB string) string {
	if moveA == moveB {
		return "draw"
	}

	switch moveA {
	case "rock":
		if moveB == "scissors" {
			return "win"
		}
	case "paper":
		if moveB == "rock" {
			return "win"
		}
	case "scissors":
		if moveB == "paper" {
			return "win"
		}
	}

	return "lose"
}
