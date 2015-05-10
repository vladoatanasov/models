package staticUniformPositions

import (
	"errors"
	"strconv"
	"strings"

	"github.com/coreos/go-etcd/etcd"
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

  "spacing": float64, required;
             Space between nodes.
  "shape":   string, required;
             The shape which positions of nodes should follow; can be one of
             ["Linear"].
    `
}

func (mobilityManager *staticUniformPositions) Configure(conf *etcd.Node) (err error) {
	if conf == nil {
		err = errors.New("StaticUniformPositions: conf (*etcd.Node) is nil")
		return
	}

	var shape string

	for _, node := range conf.Nodes {
		if !node.Dir && strings.HasSuffix(node.Key, "/spacing") {
			mobilityManager.spacing, err = strconv.ParseFloat(node.Value, 64)
			if err != nil {
				return
			}
		} else if !node.Dir && strings.HasSuffix(node.Key, "/shape") {
			shape = node.Value
		}
	}

	if mobilityManager.spacing <= 0 {
		return errors.New("spacing is missing from config or is not greater than 0")
	}
	if shape == "" {
		return errors.New("shape is missing from config")
	}

	switch shape {
	case "Linear":
		mobilityManager.next = staticNextPointLinear
	default:
		return errors.New("unknown shape")
	}

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
