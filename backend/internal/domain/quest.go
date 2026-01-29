package domain

import "time"

type QuestType string

const (
	QuestTypeDaily   QuestType = "daily"
	QuestTypeWeekly  QuestType = "weekly"
	QuestTypeOneTime QuestType = "one_time"
)

//Тип действия для квеста
type ActionType string

const (
	ActionTypePlay      ActionType = "play"
	ActionTypeWin       ActionType = "win"
	ActionTypeLose      ActionType = "lose"
	ActionTypeSpendGems ActionType = "spend_gems"
	ActionTypeEarnGems  ActionType = "earn_gems"
)

//Шаблон задания
type Quest struct {
	ID          int64      `db:"id" json:"id"`
	QuestType   QuestType  `db:"quest_type" json:"quest_type"`
	Title       string     `db:"title" json:"title"`
	Description string     `db:"description" json:"description"`
	GameType    *string    `db:"game_type" json:"game_type,omitempty"` // 'rps', 'mines', 'any', NULL
	ActionType  ActionType `db:"action_type" json:"action_type"`
	TargetCount int        `db:"target_count" json:"target_count"`
	RewardGems  int64      `db:"reward_gems" json:"reward_gems"`
	RewardCoins int64      `db:"reward_coins" json:"reward_coins"`
	RewardGK    int64      `db:"reward_gk" json:"reward_gk"`
	IsActive    bool       `db:"is_active" json:"is_active"`
	SortOrder   int        `db:"sort_order" json:"sort_order"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

//Прогресс пользователя по заданию
type UserQuest struct {
	ID              int64      `db:"id" json:"id"`
	UserID          int64      `db:"user_id" json:"user_id"`
	QuestID         int64      `db:"quest_id" json:"quest_id"`
	CurrentCount    int        `db:"current_count" json:"current_count"`
	Completed       bool       `db:"completed" json:"completed"`
	RewardClaimed   bool       `db:"reward_claimed" json:"reward_claimed"`
	StartedAt       time.Time  `db:"started_at" json:"started_at"`
	CompletedAt     *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	RewardClaimedAt *time.Time `db:"reward_claimed_at" json:"reward_claimed_at,omitempty"`
	PeriodStart     time.Time  `db:"period_start" json:"period_start"`
}

//Прогресс с деталями квеста для api ответов
type UserQuestWithDetails struct {
	UserQuest
	Quest Quest `json:"quest"`
}

//Проверка истечения срока квеста(у ежедневных сброс в полночь)
func (uq *UserQuest) IsExpired(quest *Quest) bool {
	now := time.Now()
	switch quest.QuestType {
	case QuestTypeDaily:
		return uq.PeriodStart.Day() != now.Day() ||
			uq.PeriodStart.Month() != now.Month() ||
			uq.PeriodStart.Year() != now.Year()
	case QuestTypeWeekly:
		return now.Sub(uq.PeriodStart) >= 7*24*time.Hour
	case QuestTypeOneTime:
		return false
	}
	return false
}

//Проверка можно ли забрать награду
func (uq *UserQuest) CanClaim() bool {
	return uq.Completed && !uq.RewardClaimed
}

//Возвращает прогресс в процентах от 0 до 100
func (uq *UserQuest) Progress(targetCount int) int {
	if targetCount <= 0 {
		return 100
	}
	progress := (uq.CurrentCount * 100) / targetCount
	if progress > 100 {
		return 100
	}
	return progress
}

// QuestReward содержит все награды за выполнение квеста
type QuestReward struct {
	Gems  int64 `json:"gems"`
	Coins int64 `json:"coins"`
	GK    int64 `json:"gk"`
}
