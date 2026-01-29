package repository

import (
	"context"
	"time"

	"telegram_webapp/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type QuestRepository struct {
	db *pgxpool.Pool
}

func NewQuestRepository(db *pgxpool.Pool) *QuestRepository {
	return &QuestRepository{db: db}
}

// возвращает все активные квесты
func (r *QuestRepository) GetActiveQuests(ctx context.Context) ([]*domain.Quest, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, quest_type, title, description, game_type, action_type,
				target_count, reward_gems, COALESCE(reward_coins, 0), COALESCE(reward_gk, 0),
				is_active, sort_order, created_at, updated_at
		 FROM quests
		 WHERE is_active = true
		 ORDER BY sort_order, id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanQuests(rows)
}

// возвращает квесты по типу
func (r *QuestRepository) GetQuestsByType(ctx context.Context, questType domain.QuestType) ([]*domain.Quest, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, quest_type, title, description, game_type, action_type,
				target_count, reward_gems, COALESCE(reward_coins, 0), COALESCE(reward_gk, 0),
				is_active, sort_order, created_at, updated_at
		 FROM quests
		 WHERE is_active = true AND quest_type = $1
		 ORDER BY sort_order, id`,
		questType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanQuests(rows)
}

// возвращает квест по ID
func (r *QuestRepository) GetQuestByID(ctx context.Context, id int64) (*domain.Quest, error) {
	var q domain.Quest
	err := r.db.QueryRow(ctx,
		`SELECT id, quest_type, title, description, game_type, action_type,
				target_count, reward_gems, COALESCE(reward_coins, 0), COALESCE(reward_gk, 0),
				is_active, sort_order, created_at, updated_at
		 FROM quests
		 WHERE id = $1`,
		id,
	).Scan(&q.ID, &q.QuestType, &q.Title, &q.Description, &q.GameType, &q.ActionType,
		&q.TargetCount, &q.RewardGems, &q.RewardCoins, &q.RewardGK, &q.IsActive, &q.SortOrder, &q.CreatedAt, &q.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &q, nil
}

// возвращает прогресс пользователя по квестам
func (r *QuestRepository) GetUserQuests(ctx context.Context, userID int64) ([]*domain.UserQuestWithDetails, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
			uq.id, uq.user_id, uq.quest_id, uq.current_count, uq.completed,
			uq.reward_claimed, uq.started_at, uq.completed_at, uq.reward_claimed_at, uq.period_start,
			q.id, q.quest_type, q.title, q.description, q.game_type, q.action_type,
			q.target_count, q.reward_gems, COALESCE(q.reward_coins, 0), COALESCE(q.reward_gk, 0),
			q.is_active, q.sort_order, q.created_at, q.updated_at
		 FROM user_quests uq
		 JOIN quests q ON uq.quest_id = q.id
		 WHERE uq.user_id = $1 AND q.is_active = true
		 ORDER BY uq.completed, q.sort_order, q.id`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.UserQuestWithDetails
	for rows.Next() {
		var uqd domain.UserQuestWithDetails
		err := rows.Scan(
			&uqd.ID, &uqd.UserID, &uqd.QuestID, &uqd.CurrentCount, &uqd.Completed,
			&uqd.RewardClaimed, &uqd.StartedAt, &uqd.CompletedAt, &uqd.RewardClaimedAt, &uqd.PeriodStart,
			&uqd.Quest.ID, &uqd.Quest.QuestType, &uqd.Quest.Title, &uqd.Quest.Description,
			&uqd.Quest.GameType, &uqd.Quest.ActionType, &uqd.Quest.TargetCount, &uqd.Quest.RewardGems,
			&uqd.Quest.RewardCoins, &uqd.Quest.RewardGK,
			&uqd.Quest.IsActive, &uqd.Quest.SortOrder, &uqd.Quest.CreatedAt, &uqd.Quest.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, &uqd)
	}

	return result, nil
}

// получает или создаёт прогресс пользователя по квесту
func (r *QuestRepository) GetOrCreateUserQuest(ctx context.Context, userID, questID int64, periodStart time.Time) (*domain.UserQuest, error) {
	var uq domain.UserQuest

	// попытка найти существующую запись
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, quest_id, current_count, completed, reward_claimed,
				started_at, completed_at, reward_claimed_at, period_start
		 FROM user_quests
		 WHERE user_id = $1 AND quest_id = $2 AND period_start = $3`,
		userID, questID, periodStart,
	).Scan(&uq.ID, &uq.UserID, &uq.QuestID, &uq.CurrentCount, &uq.Completed,
		&uq.RewardClaimed, &uq.StartedAt, &uq.CompletedAt, &uq.RewardClaimedAt, &uq.PeriodStart)

	if err == nil {
		return &uq, nil
	}

	// создать новую запись
	err = r.db.QueryRow(ctx,
		`INSERT INTO user_quests (user_id, quest_id, period_start)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, quest_id, current_count, completed, reward_claimed,
				   started_at, completed_at, reward_claimed_at, period_start`,
		userID, questID, periodStart,
	).Scan(&uq.ID, &uq.UserID, &uq.QuestID, &uq.CurrentCount, &uq.Completed,
		&uq.RewardClaimed, &uq.StartedAt, &uq.CompletedAt, &uq.RewardClaimedAt, &uq.PeriodStart)

	if err != nil {
		return nil, err
	}

	return &uq, nil
}

// обновляет прогресс квеста
func (r *QuestRepository) UpdateProgress(ctx context.Context, uq *domain.UserQuest) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_quests
		 SET current_count = $1, completed = $2, completed_at = $3
		 WHERE id = $4`,
		uq.CurrentCount, uq.Completed, uq.CompletedAt, uq.ID,
	)
	return err
}

// отмечает награду как полученную и возвращает все награды
func (r *QuestRepository) ClaimReward(ctx context.Context, userQuestID int64) (*domain.QuestReward, error) {
	var reward domain.QuestReward
	now := time.Now()

	err := r.db.QueryRow(ctx,
		`UPDATE user_quests uq
		 SET reward_claimed = true, reward_claimed_at = $1
		 FROM quests q
		 WHERE uq.id = $2
		   AND uq.quest_id = q.id
		   AND uq.completed = true
		   AND uq.reward_claimed = false
		 RETURNING q.reward_gems, COALESCE(q.reward_coins, 0), COALESCE(q.reward_gk, 0)`,
		now, userQuestID,
	).Scan(&reward.Gems, &reward.Coins, &reward.GK)

	if err != nil {
		return nil, err
	}

	return &reward, nil
}

// увеличивает прогресс и проверяет завершение
func (r *QuestRepository) IncrementProgress(ctx context.Context, userID int64, quest *domain.Quest, increment int) error {
	periodStart := r.getPeriodStart(quest.QuestType)

	// получить или создать запись прогресса
	uq, err := r.GetOrCreateUserQuest(ctx, userID, quest.ID, periodStart)
	if err != nil {
		return err
	}

	// если уже выполнено - ничего
	if uq.Completed {
		return nil
	}

	// увеличить прогресс
	uq.CurrentCount += increment

	// проверить завершение
	if uq.CurrentCount >= quest.TargetCount {
		uq.Completed = true
		now := time.Now()
		uq.CompletedAt = &now
	}

	return r.UpdateProgress(ctx, uq)
}

// сбрасывает ежедневные квесты (вызывать по cron)
func (r *QuestRepository) ResetDailyQuests(ctx context.Context) error {
	today := time.Now().Truncate(24 * time.Hour)

	// удалить старые незавершённые daily квесты
	_, err := r.db.Exec(ctx,
		`DELETE FROM user_quests
		 WHERE quest_id IN (SELECT id FROM quests WHERE quest_type = 'daily')
		   AND period_start < $1
		   AND reward_claimed = false`,
		today,
	)
	return err
}

// сбрасывает еженедельные квесты (вызывать по cron)
func (r *QuestRepository) ResetWeeklyQuests(ctx context.Context) error {
	weekAgo := time.Now().AddDate(0, 0, -7)

	_, err := r.db.Exec(ctx,
		`DELETE FROM user_quests
		 WHERE quest_id IN (SELECT id FROM quests WHERE quest_type = 'weekly')
		   AND period_start < $1
		   AND reward_claimed = false`,
		weekAgo,
	)
	return err
}

// возвращает начало текущего периода для типа квеста
func (r *QuestRepository) getPeriodStart(questType domain.QuestType) time.Time {
	now := time.Now()
	switch questType {
	case domain.QuestTypeDaily:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case domain.QuestTypeWeekly:
		// Начало недели (понедельник)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
	case domain.QuestTypeOneTime:
		// Для разовых квестов используется фиксированную дату
		return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return now
}

// Helper для сканирования квестов
func (r *QuestRepository) scanQuests(rows interface{ Next() bool; Scan(dest ...interface{}) error }) ([]*domain.Quest, error) {
	var result []*domain.Quest
	for rows.Next() {
		var q domain.Quest
		err := rows.Scan(&q.ID, &q.QuestType, &q.Title, &q.Description, &q.GameType, &q.ActionType,
			&q.TargetCount, &q.RewardGems, &q.RewardCoins, &q.RewardGK, &q.IsActive, &q.SortOrder, &q.CreatedAt, &q.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, &q)
	}
	return result, nil
}
