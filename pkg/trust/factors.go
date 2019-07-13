package trust

// All of the factors taken into account by Astronomer
// and shown in the report.
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
		PrivateContributionFactor:  300,
		ContributionScoreFactor:    18000,
		IssueContributionFactor:    18,
		CommitContributionFactor:   370,
		RepoContributionFactor:     25,
		PRContributionFactor:       20,
		PRReviewContributionFactor: 7,
		AccountAgeFactor:           1600,
	}

	percentileReferences = map[Percentile]float64{
		"5":  6,
		"10": 18,
		"15": 34,
		"20": 76,
		"25": 142,
		"30": 232,
		"35": 396,
		"40": 435,
		"45": 625,
		"50": 1005,
		"55": 1490,
		"60": 2230,
		"65": 3680,
		"70": 5100,
		"75": 7850,
		"80": 9230,
		"85": 17800,
		"90": 28495,
		"95": 51230,
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
