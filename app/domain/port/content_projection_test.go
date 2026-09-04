package port

import "testing"

func TestCalculateContentPCA2DIsStableAcrossInputOrder(t *testing.T) {
	inputs := []PCAInputVector{
		{ArticleID: 3, Values: []float32{3, 1, 0}},
		{ArticleID: 1, Values: []float32{1, 0, 2}},
		{ArticleID: 2, Values: []float32{2, 2, 1}},
	}
	first, err := CalculateContentPCA2D(inputs)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CalculateContentPCA2D([]PCAInputVector{inputs[2], inputs[0], inputs[1]})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != len(second) || first[0].ArticleID != 1 || first[1].ArticleID != 2 || first[2].ArticleID != 3 {
		t.Fatalf("points are not ordered by article id: %#v", first)
	}
	for index := range first {
		if first[index] != second[index] {
			t.Fatalf("order changed point %d: first=%+v second=%+v", index, first[index], second[index])
		}
	}
}

func TestCalculateContentPCA2DRejectsInvalidVectors(t *testing.T) {
	tests := [][]PCAInputVector{
		{{ArticleID: 1, Values: []float32{1}}, {ArticleID: 1, Values: []float32{2}}},
		{{ArticleID: 1, Values: []float32{1}}, {ArticleID: 2, Values: []float32{1, 2}}},
		{{ArticleID: 0, Values: []float32{1}}},
	}
	for index, input := range tests {
		if _, err := CalculateContentPCA2D(input); err == nil {
			t.Fatalf("case %d: invalid PCA input was accepted", index)
		}
	}
}

func TestCalculateContentPCA2DReturnsZeroForConstantData(t *testing.T) {
	points, err := CalculateContentPCA2D([]PCAInputVector{
		{ArticleID: 1, Values: []float32{1, 1}},
		{ArticleID: 2, Values: []float32{1, 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range points {
		if point.X != 0 || point.Y != 0 {
			t.Fatalf("constant data produced non-zero point: %+v", point)
		}
	}
}
