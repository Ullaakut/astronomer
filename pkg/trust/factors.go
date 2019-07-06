package trust

// Factors.
const (
	PrivateContributionFactor  FactorName = "Private contributions"
	ContributionScoreFactor    FactorName = "Weighted contributions"
	IssueContributionFactor    FactorName = "Created issues"
	CommitContributionFactor   FactorName = "Commits authored"
	RepoContributionFactor     FactorName = "Repositories"
	PRContributionFactor       FactorName = "Pull requests"
	PRReviewContributionFactor FactorName = "Code reviews"
	AccountAgeFactor           FactorName = "Account age (days)"
	Overall                    FactorName = "Overall trust"
)

// TODO: If we allow users to choose the year until which
// to scan, references will be needed for each year.
var (
	factors = []FactorName{
		ContributionScoreFactor,
		PrivateContributionFactor,
		IssueContributionFactor,
		CommitContributionFactor,
		RepoContributionFactor,
		PRContributionFactor,
		PRReviewContributionFactor,
		AccountAgeFactor,
	}

	percentiles = []Percentile{"5", "10", "15", "20", "25", "30", "35", "40", "45", "50", "55", "60", "65", "70", "75", "80", "85", "90", "95"}

	// References are based on the average values of values typically
	// found on popular repositories.
	factorReferences = map[FactorName]float64{
		PrivateContributionFactor:  600,
		ContributionScoreFactor:    24000,
		IssueContributionFactor:    20,
		CommitContributionFactor:   370,
		RepoContributionFactor:     30,
		PRContributionFactor:       20,
		PRReviewContributionFactor: 10,
		AccountAgeFactor:           1600,
	}

	percentileReferences = map[Percentile]float64{
		"5":  10,
		"10": 30,
		"15": 65,
		"20": 120,
		"25": 230,
		"30": 310,
		"35": 520,
		"40": 680,
		"45": 990,
		"50": 1450,
		"55": 2150,
		"60": 3230,
		"65": 4870,
		"70": 6480,
		"75": 9830,
		"80": 12020,
		"85": 24110,
		"90": 39970,
		"95": 74670,
	}

	// factorWeights represents the importance of each factor in
	// the calculation of the overall trust factor.
	factorWeights = map[FactorName]int{
		PrivateContributionFactor:  1,
		IssueContributionFactor:    3,
		CommitContributionFactor:   3,
		RepoContributionFactor:     2,
		PRContributionFactor:       2,
		PRReviewContributionFactor: 2,
		ContributionScoreFactor:    8,
		AccountAgeFactor:           2,
	}
)
