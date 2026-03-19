package config

import "math"

const (
	// ProjectMapContextRatioDefault は ProjectMap に割り当てるデフォルト比率。
	ProjectMapContextRatioDefault = 0.05
	// ProjectMapContextRatioMin は許容する最小比率。
	ProjectMapContextRatioMin = 0.01
	// ProjectMapContextRatioMax は許容する最大比率。
	ProjectMapContextRatioMax = 0.20
)

// NormalizeProjectMapContextRatio は無効な比率をデフォルト値へ正規化する。
func NormalizeProjectMapContextRatio(ratio float64) float64 {
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return ProjectMapContextRatioDefault
	}
	if ratio < ProjectMapContextRatioMin || ratio > ProjectMapContextRatioMax {
		return ProjectMapContextRatioDefault
	}

	return ratio
}
