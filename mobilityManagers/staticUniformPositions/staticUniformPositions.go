package staticUniformPositions

import (
	"errors"
	"github.com/squirrel-land/squirrel"
)

type staticUniformPositions struct {
	nodes   []*squirrel.Position
	spacing float64
	next    func(*squirrel.Position, float64) *squirrel.Position
}

func NewStaticUniformPositions() squirrel.MobilityManager {
	return &staticUniformPositions{}
}

func (mobilityManager *staticUniformPositions) ParametersHelp() string {
	return `StaticUniformPositions is a mobility manager in which nodes are not mobile.
Nodes are positioned uniformly on a grid map.

  "Spacing": float64, required;
             Space between nodes.
  "Shape":   string, required;
             The shape which positions of nodes should follow; can be one of
             ["Linear"].
    `
}

func (mobilityManager *staticUniformPositions) Configure(config map[string]interface{}) error {
	spacing, ok := config["Spacing"].(float64)
	if ok != true {
		return errors.New("Spacing is missing from config")
	}
	shape, ok := config["Shape"].(string)
	if ok != true {
		return errors.New("Shape is missing from config")
	}
	switch shape {
	case "Linear":
		mobilityManager.next = staticNextPointLinear
	default:
		return errors.New("unknown shape")
	}
	mobilityManager.spacing = spacing
	return nil
}

func (mobilityManager *staticUniformPositions) Initialize(positionManager squirrel.PositionManager) {
	ch := make(chan []int)
	positionManager.RegisterEnabledChanged(ch)
	go func() {
		for {
			enabled := <-ch
			var latest *squirrel.Position
			for _, index := range enabled {
				latest = mobilityManager.next(latest, mobilityManager.spacing)
				positionManager.SetPosition(index, latest)
			}
		}
	}()
}

func staticNextPointLinear(prev *squirrel.Position, spacing float64) *squirrel.Position {
	next := &squirrel.Position{}
	if prev == nil {
		next.X = 0
		next.Y = 0
		next.Height = 0
	} else {
		next.X = prev.X + spacing
		next.Y = prev.Y
		next.Height = prev.Height
	}
	return next
}
