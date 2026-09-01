package gamification

// XPBreakdown discrimina o ganho detalhado de XP obtido pelo jogador.
type XPBreakdown struct {
	BaseXP            int `json:"baseXp"`
	FirstAttemptBonus int `json:"firstAttemptBonus"`
	StreakBonus       int `json:"streakBonus"`
	PhaseBonus        int `json:"phaseBonus"`
	LevelBonus        int `json:"levelBonus"`
	Total             int `json:"total"`
}

// CalculateAnswerXP calcula o XP total e detalhado ganho ao responder um exercício.
func CalculateAnswerXP(rules Rules, isCorrect bool, isFirstAttempt bool, currentStreak int) XPBreakdown {
	if !isCorrect {
		return XPBreakdown{}
	}

	breakdown := XPBreakdown{
		BaseXP: rules.BaseCorrectXP,
	}

	if isFirstAttempt {
		breakdown.FirstAttemptBonus = rules.FirstAttemptBonusXP
	}

	switch currentStreak {
	case rules.Streak5Threshold:
		breakdown.StreakBonus = rules.Streak5BonusXP
	case rules.Streak3Threshold:
		breakdown.StreakBonus = rules.Streak3BonusXP
	}

	breakdown.Total = breakdown.BaseXP + breakdown.FirstAttemptBonus + breakdown.StreakBonus
	return breakdown
}

// CalculatePhaseCompletionXP calcula o bônus concedido ao completar uma fase.
func CalculatePhaseCompletionXP(rules Rules) XPBreakdown {
	return XPBreakdown{
		PhaseBonus: rules.PhaseCompletionBonusXP,
		Total:      rules.PhaseCompletionBonusXP,
	}
}

// CalculateLevelCompletionXP calcula o bônus concedido ao completar um nível.
func CalculateLevelCompletionXP(rules Rules) XPBreakdown {
	return XPBreakdown{
		LevelBonus: rules.LevelCompletionBonusXP,
		Total:      rules.LevelCompletionBonusXP,
	}
}
