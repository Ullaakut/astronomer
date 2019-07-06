package main

const (
	privateContributionFactor  factorName = "Private contributions"
	contributionScoreFactor    factorName = "Weighted contributions"
	issueContributionFactor    factorName = "Created issues"
	commitContributionFactor   factorName = "Commits authored"
	repoContributionFactor     factorName = "Repositories"
	prContributionFactor       factorName = "Pull requests"
	prReviewContributionFactor factorName = "Code reviews"
	accountAgeFactor           factorName = "Account age (days)"
	overallTrust               factorName = "Overall trust"
)

// TODO: If we allow users to choose the year until which
// to scan, references will be needed for each year.
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

	// References are based on the average values of values typically
	// found on popular repositories.
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
		5:  10,
		10: 30,
		15: 65,
		20: 120,
		25: 230,
		30: 310,
		35: 520,
		40: 680,
		45: 990,
		50: 1450,
		55: 2150,
		60: 3230,
		65: 4870,
		70: 6480,
		75: 9830,
		80: 12020,
		85: 24110,
		90: 39970,
		95: 74670,
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
