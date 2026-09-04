package port

import (
	"errors"
	"math"
	"sort"
)

const (
	maxPCAInputDimension = 4096
	maxPCAInputVectors   = 10_000
	pcaPowerIterations   = 64
	pcaEpsilon           = 1e-10
)

type PCAInputVector struct {
	ArticleID int
	Values    []float32
}

type ContentProjectionPoint struct {
	ArticleID int
	X         float64
	Y         float64
}

// CalculateContentPCA2D calculates a deterministic two-dimensional PCA
// projection using only the standard library. Input order does not affect the
// result; eigenvector signs are canonicalized so a redraw does not randomly
// mirror the galaxy after a restart.
func CalculateContentPCA2D(inputs []PCAInputVector) ([]ContentProjectionPoint, error) {
	if len(inputs) == 0 {
		return []ContentProjectionPoint{}, nil
	}
	if len(inputs) > maxPCAInputVectors {
		return nil, errors.New("too many PCA input vectors")
	}
	ordered := append([]PCAInputVector(nil), inputs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ArticleID < ordered[j].ArticleID })
	dimension := len(ordered[0].Values)
	if ordered[0].ArticleID <= 0 || dimension == 0 || dimension > maxPCAInputDimension {
		return nil, errors.New("PCA input dimension or article id is invalid")
	}
	for index := range ordered {
		if ordered[index].ArticleID <= 0 || len(ordered[index].Values) != dimension {
			return nil, errors.New("PCA input vectors must have one dimension and unique positive article ids")
		}
		if index > 0 && ordered[index-1].ArticleID == ordered[index].ArticleID {
			return nil, errors.New("PCA input article ids must be unique")
		}
	}

	data := make([][]float64, len(ordered))
	mean := make([]float64, dimension)
	for rowIndex, input := range ordered {
		row := make([]float64, dimension)
		for column, value := range input.Values {
			converted := float64(value)
			if math.IsNaN(converted) || math.IsInf(converted, 0) {
				return nil, errors.New("PCA input contains a non-finite value")
			}
			row[column] = converted
			mean[column] += converted
		}
		data[rowIndex] = row
	}
	for column := range mean {
		mean[column] /= float64(len(data))
	}
	centered := make([][]float64, len(data))
	variances := make([]float64, dimension)
	for rowIndex, row := range data {
		centered[rowIndex] = make([]float64, dimension)
		for column, value := range row {
			centered[rowIndex][column] = value - mean[column]
			variances[column] += centered[rowIndex][column] * centered[rowIndex][column]
		}
	}

	firstSeed := basisAtMax(variances)
	first := pcaPowerIteration(centered, firstSeed, nil, 0)
	firstLambda := 0.0
	if len(first) != 0 {
		firstLambda = dot(first, covarianceMultiply(centered, first))
	}
	residualVariances := make([]float64, dimension)
	for column := range residualVariances {
		residualVariances[column] = variances[column]
		if len(first) != 0 {
			residualVariances[column] -= firstLambda * first[column] * first[column]
		}
		if residualVariances[column] < 0 && residualVariances[column] > -pcaEpsilon {
			residualVariances[column] = 0
		}
	}
	second := pcaPowerIteration(centered, basisAtMax(residualVariances), first, firstLambda)
	points := make([]ContentProjectionPoint, len(ordered))
	for index, input := range ordered {
		points[index] = ContentProjectionPoint{ArticleID: input.ArticleID}
		if len(first) != 0 {
			points[index].X = dot(centered[index], first)
		}
		if len(second) != 0 {
			points[index].Y = dot(centered[index], second)
		}
	}
	return points, nil
}

func basisAtMax(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}
	index := 0
	for candidate := 1; candidate < len(values); candidate++ {
		if values[candidate] > values[index] {
			index = candidate
		}
	}
	seed := make([]float64, len(values))
	seed[index] = 1
	return seed
}

func covarianceMultiply(data [][]float64, vector []float64) []float64 {
	result := make([]float64, len(vector))
	for _, row := range data {
		weight := dot(row, vector)
		for index, value := range row {
			result[index] += weight * value
		}
	}
	return result
}

func pcaPowerIteration(data [][]float64, seed, deflation []float64, lambda float64) []float64 {
	if len(seed) == 0 {
		return nil
	}
	vector := normalizeVector(seed)
	if len(vector) == 0 {
		return nil
	}
	for iteration := 0; iteration < pcaPowerIterations; iteration++ {
		next := covarianceMultiply(data, vector)
		if len(deflation) != 0 {
			next = subtractScaled(next, deflation, lambda*dot(deflation, vector))
		}
		if len(deflation) != 0 {
			next = subtractScaled(next, deflation, dot(next, deflation))
		}
		next = normalizeVector(next)
		if len(next) == 0 {
			return nil
		}
		if distance(next, vector) < pcaEpsilon {
			vector = next
			break
		}
		vector = next
	}
	return canonicalizeSign(vector)
}

func normalizeVector(value []float64) []float64 {
	length := math.Sqrt(dot(value, value))
	if length <= pcaEpsilon || math.IsNaN(length) || math.IsInf(length, 0) {
		return nil
	}
	result := make([]float64, len(value))
	for index, component := range value {
		result[index] = component / length
	}
	return result
}

func subtractScaled(left, right []float64, scale float64) []float64 {
	result := append([]float64(nil), left...)
	for index := range result {
		result[index] -= scale * right[index]
	}
	return result
}

func canonicalizeSign(value []float64) []float64 {
	result := append([]float64(nil), value...)
	for _, component := range result {
		if math.Abs(component) <= pcaEpsilon {
			continue
		}
		if component < 0 {
			for index := range result {
				result[index] = -result[index]
			}
		}
		break
	}
	return result
}

func dot(left, right []float64) float64 {
	result := 0.0
	for index := range left {
		result += left[index] * right[index]
	}
	return result
}

func distance(left, right []float64) float64 {
	result := 0.0
	for index := range left {
		delta := left[index] - right[index]
		result += delta * delta
	}
	return math.Sqrt(result)
}
