package game

import (
	"crypto/rand"
	"math/big"
)

// WheelSegment представляет сегмент на колесе фортуны
type WheelSegment struct {
	ID          int     `json:"id"`
	Multiplier  float64 `json:"multiplier"`
	Color       string  `json:"color"`
	Probability float64 `json:"probability"` // 0.0 - 1.0
	Label       string  `json:"label"`
}

// WheelGame представляет игру в колесо фортуны
type WheelGame struct {
	Segments  []WheelSegment `json:"segments"`
	Result    *WheelSegment  `json:"result"`
	SpinAngle float64        `json:"spin_angle"` // Финальный угол для анимации на фронтенде
}

// возвращает стандартную конфигурацию сегментов колеса
func DefaultWheelSegments() []WheelSegment {
	return []WheelSegment{
		{ID: 1, Multiplier: 0.0, Color: "#4a4a4a", Probability: 0.30, Label: "0x"},
		{ID: 2, Multiplier: 0.5, Color: "#e74c3c", Probability: 0.25, Label: "0.5x"},
		{ID: 3, Multiplier: 1.0, Color: "#f39c12", Probability: 0.22, Label: "1x"},
		{ID: 4, Multiplier: 1.5, Color: "#2ecc71", Probability: 0.09, Label: "1.5x"},
		{ID: 5, Multiplier: 2.0, Color: "#3498db", Probability: 0.06, Label: "2x"},
		{ID: 6, Multiplier: 3.0, Color: "#9b59b6", Probability: 0.035, Label: "3x"},
		{ID: 7, Multiplier: 5.0, Color: "#e67e22", Probability: 0.025, Label: "5x"},
		{ID: 8, Multiplier: 10.0, Color: "#f1c40f", Probability: 0.02, Label: "10x"},
	}
}

// создает новую игру на колесе со стандартными сегментами
func NewWheelGame() *WheelGame {
	return &WheelGame{
		Segments: DefaultWheelSegments(),
	}
}

// создает игру на колесе с пользовательскими сегментами
func NewWheelGameWithSegments(segments []WheelSegment) *WheelGame {
	return &WheelGame{
		Segments: segments,
	}
}

// выполняет вращение колеса и возвращает выигрышный сегмент
func (g *WheelGame) Spin() *WheelSegment {
	// Генерируем криптографически безопасное случайное число
	max := big.NewInt(1000000) // точность 0.000001
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		n = big.NewInt(500000)
	}

	random := float64(n.Int64()) / 1000000.0 // 0.0 - 0.999999

	// Находим выигрышный сегмент на основе распределения вероятностей
	cumulative := 0.0
	for i := range g.Segments {
		cumulative += g.Segments[i].Probability
		if random < cumulative {
			g.Result = &g.Segments[i]
			break
		}
	}

	// Запасной вариант: последний сегмент если что-то пошло не так
	if g.Result == nil {
		g.Result = &g.Segments[len(g.Segments)-1]
	}

	// Рассчитываем угол вращения для анимации на фронтенде
	// Каждый сегмент занимает 360/количество_сегментов градусов
	segmentAngle := 360.0 / float64(len(g.Segments))
	baseAngle := float64(g.Result.ID-1) * segmentAngle

	// Добавляем случайное смещение внутри сегмента + несколько полных оборотов
	offsetMax := big.NewInt(int64(segmentAngle * 100))
	offsetN, _ := rand.Int(rand.Reader, offsetMax)
	offset := float64(offsetN.Int64()) / 100.0

	rotations := 5 // Количество полных оборотов для анимации
	g.SpinAngle = float64(rotations*360) + baseAngle + offset

	return g.Result
}

// вычисляет выигрышную сумму для данной ставки
func (g *WheelGame) CalculateWinAmount(bet int64) int64 {
	if g.Result == nil {
		return 0
	}
	return int64(float64(bet) * g.Result.Multiplier)
}

// возвращает детали игры для хранения
func (g *WheelGame) ToDetails() map[string]interface{} {
	result := map[string]interface{}{
		"spin_angle": g.SpinAngle,
	}
	if g.Result != nil {
		result["segment_id"] = g.Result.ID
		result["multiplier"] = g.Result.Multiplier
		result["color"] = g.Result.Color
		result["label"] = g.Result.Label
	}
	return result
}

// вычисляет ожидаемый возврат колеса
func (g *WheelGame) GetExpectedReturn() float64 {
	expected := 0.0
	for _, seg := range g.Segments {
		expected += seg.Probability * seg.Multiplier
	}
	return expected
}