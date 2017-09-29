package goscore

import (
	"encoding/xml"
	"math"
	"sync"
)

// GradientBoostedModel - GradientBoostedModel PMML
type GradientBoostedModel struct {
	Version  string  `xml:"version,attr"`
	Trees    []Node  `xml:"MiningModel>Segmentation>Segment>MiningModel>Segmentation>Segment>TreeModel"`
	Target   target  `xml:"MiningModel>Segmentation>Segment>MiningModel>Targets>Target"`
	Constant float64 `xml:"MiningModel>Segmentation>Segment>MiningModel>Output>OutputField>Apply>Constant"`
}

type target struct {
	XMLName         xml.Name
	RescaleConstant float64 `xml:"rescaleConstant,attr"`
}

// Score - traverses all trees in GradientBoostedModel with features and returns exp(sum) / (1 + exp(sum))
// where sum is sum of trees + rescale constant
func (gbm GradientBoostedModel) Score(features map[string]interface{}) (float64, error) {
	sum := fetchConst(gbm)

	for _, tree := range gbm.Trees {
		score, err := tree.TraverseTree(features)
		if err != nil {
			return -1, err
		}
		sum += score
	}
	return math.Exp(sum) / (1 + math.Exp(sum)), nil
}

type result struct {
	ErrorName error
	Score     float64
}

// ScoreConcurrently - same as Score but concurrent
func (gbm GradientBoostedModel) ScoreConcurrently(features map[string]interface{}) (float64, error) {
	sum := fetchConst(gbm)
	messages := make(chan result, len(gbm.Trees))

	var wg sync.WaitGroup
	wg.Add(len(gbm.Trees))

	for _, tree := range gbm.Trees {
		go func(tree Node, features map[string]interface{}) {
			treeScore, err := tree.TraverseTree(features)

			messages <- result{ErrorName: err, Score: treeScore}
			wg.Done()
		}(tree, features)
	}
	wg.Wait()

	var res result
	for i := 0; i < len(gbm.Trees); i++ {
		res = <-messages
		if res.ErrorName != nil {
			return -1, res.ErrorName
		}
		sum += res.Score
	}
	return math.Exp(sum) / (1 + math.Exp(sum)), nil
}

func fetchConst(gbm GradientBoostedModel) (sum float64) {
	if gbm.Version == "4.2" {
		sum = gbm.Constant
	} else if gbm.Version == "4.3" {
		sum = gbm.Target.RescaleConstant
	}
	return sum
}
