package main

const (
	privateContributionFactor  factorName = "Private contributions"
	contributionScoreFactor               = "Weighted contributions"
	issueContributionFactor               = "Created issues"
	commitContributionFactor              = "Commits authored"
	repoContributionFactor                = "Repositories"
	prContributionFactor                  = "Pull requests"
	prReviewContributionFactor            = "Code reviews"
	accountAgeFactor                      = "Account age (days)"
	overallTrust                          = "Overall trust"
)

// TODO: If we allow users to choose the year until which
// to scam, references will be needed for each year.
var (
	factors = []factorName{
		contributionScoreFactor,
		privateContributionFactor,
		issueContributionFactor,
		commitContributionFactor,
		repoContributionFactor,
		prContributionFactor,
		prReviewContributionFactor,
		accountAgeFactor,
	}

	percentiles = []float64{5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95}

	factorReferences = map[factorName]float64{
		privateContributionFactor:  600,
		contributionScoreFactor:    24000,
		issueContributionFactor:    20,
		commitContributionFactor:   370,
		repoContributionFactor:     30,
		prContributionFactor:       20,
		prReviewContributionFactor: 10,
		accountAgeFactor:           1600,
	}

	percentileReferences = map[float64]float64{
		5:  38,
		10: 105,
		15: 210,
		20: 320,
		25: 455,
		30: 650,
		35: 1000,
		40: 1500,
		45: 2000,
		50: 3000,
		55: 4000,
		60: 5000,
		65: 6500,
		70: 9000,
		75: 14000,
		80: 20000,
		85: 30000,
		90: 55000,
		95: 110000,
	}

	// factorWeights represents the importance of each factor in
	// the calculation of the overall trust factor.
	factorWeights = map[factorName]int{
		privateContributionFactor:  1,
		issueContributionFactor:    3,
		commitContributionFactor:   3,
		repoContributionFactor:     2,
		prContributionFactor:       2,
		prReviewContributionFactor: 2,
		contributionScoreFactor:    8,
		accountAgeFactor:           2,
	}
)
