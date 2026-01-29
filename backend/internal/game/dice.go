package game

import (
	"crypto/rand"
	"math/big"
)

// представляет одну игру с бросанием кубика (кубик 1-6)
type DiceGame struct {
	Target     int     `json:"target"`      // целевое число (1-6) или индикатор диапазона
	Result     int     `json:"result"`      // результат броска (1-6)
	Mode       string  `json:"mode"`        // "exact", "low" (1-3), "high" (4-6)
	Multiplier float64 `json:"multiplier"`  // множитель выплаты
	Won        bool    `json:"won"`         // выиграл ли игрок
	// устаревшие поля для обратной совместимости
	RollOver   bool    `json:"roll_over,omitempty"`
}

const (
	DiceMinTarget = 1
	DiceMaxTarget = 6
	DiceSides     = 6
	DiceModeExact = "exact"  // угадать точное число
	DiceModeLow   = "low"    // диапазон 1-3
	DiceModeHigh  = "high"   // диапазон 4-6

	DiceMultiplierExact = 5.5  // шанс 1/6 = 5.5x
	DiceMultiplierRange = 1.8  // шанс 1/2 = 1.8x (с учетом маржи казино)
)

// создает новую игру с кубиком с заданными параметрами
func NewDiceGame(target int, mode string) *DiceGame {
	// проверяем режим
	if mode != DiceModeExact && mode != DiceModeLow && mode != DiceModeHigh {
		mode = DiceModeExact // по умолчанию точный режим
	}

	// для режимов диапазона target не используется
	if mode == DiceModeLow || mode == DiceModeHigh {
		target = 0 // не применимо для ставок на диапазон
	} else {
		// ограничиваем target допустимым диапазоном (1-6) для точного режима
		if target < DiceMinTarget {
			target = DiceMinTarget
		}
		if target > DiceMaxTarget {
			target = DiceMaxTarget
		}
	}

	// вычисляем множитель на основе режима
	var multiplier float64
	if mode == DiceModeExact {
		multiplier = DiceMultiplierExact // 5.5x за точное число
	} else {
		multiplier = DiceMultiplierRange // 1.8x за диапазон
	}

	g := &DiceGame{
		Target:     target,
		Mode:       mode,
		Multiplier: multiplier,
	}
	return g
}

// возвращает множитель выплаты на основе режима
func (g *DiceGame) CalculateMultiplier() float64 {
	if g.Mode == DiceModeExact {
		return DiceMultiplierExact
	}
	return DiceMultiplierRange
}

// возвращает вероятность выигрыша (процент)
func (g *DiceGame) WinChance() float64 {
	switch g.Mode {
	case DiceModeExact:
		// 1 из 6 = 16.67%
		return 100.0 / float64(DiceSides)
	case DiceModeLow, DiceModeHigh:
		// 3 из 6 = 50%
		return 50.0
	default:
		return 0
	}
}

// выполняет бросок кубика и возвращает результат (1-6)
func (g *DiceGame) Roll() int {
	// генерируем криптографически безопасное случайное число (1-6)
	max := big.NewInt(DiceSides)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// запасной вариант - никогда не должно происходить
		n = big.NewInt(0)
	}

	g.Result = int(n.Int64()) + 1 // преобразуем 0-5 в 1-6

	// определяем выигрыш/проигрыш на основе режима
	switch g.Mode {
	case DiceModeExact:
		// выигрыш, если точное число совпадает
		g.Won = g.Result == g.Target
	case DiceModeLow:
		// выигрыш, если результат 1, 2 или 3
		g.Won = g.Result >= 1 && g.Result <= 3
	case DiceModeHigh:
		// выигрыш, если результат 4, 5 или 6
		g.Won = g.Result >= 4 && g.Result <= 6
	default:
		g.Won = false
	}

	return g.Result
}

// вычисляет сумму выигрыша для данной ставки
func (g *DiceGame) CalculateWinAmount(bet int64) int64 {
	if !g.Won {
		return 0
	}
	return int64(float64(bet) * g.Multiplier)
}

// возвращает детали игры для хранения
func (g *DiceGame) ToDetails() map[string]interface{} {
	return map[string]interface{}{
		"target":     g.Target,
		"mode":       g.Mode,
		"result":     g.Result,
		"multiplier": g.Multiplier,
		"win_chance": g.WinChance(),
	}
}