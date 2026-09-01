package gamification

// AchievementID identifica unicamente uma conquista do jogo.
type AchievementID string

const (
	AchievementFirstCorrect         AchievementID = "first_correct"
	AchievementStreak3              AchievementID = "streak_3"
	AchievementStreak5              AchievementID = "streak_5"
	AchievementBeginnerComplete     AchievementID = "beginner_complete"
	AchievementIntermediateComplete AchievementID = "intermediate_complete"
	AchievementAdvancedComplete     AchievementID = "advanced_complete"
)

// Achievement descreve uma conquista ganha durante a sessão.
type Achievement struct {
	ID          AchievementID `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Icon        string        `json:"icon"`
}
