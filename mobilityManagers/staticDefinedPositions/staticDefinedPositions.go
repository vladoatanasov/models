package staticDefinedPositions

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/coreos/go-etcd/etcd"
	"github.com/squirrel-land/squirrel"
)

type staticDefinedPositions struct {
	positions [][3]float64
}

func NewStaticDefinedPositions() squirrel.MobilityManager {
	return &staticDefinedPositions{}
}

func (mobilityManager *staticDefinedPositions) ParametersHelp() string {
	return ``
}

func (mobilityManager *staticDefinedPositions) Configure(conf *etcd.Node) (err error) {
	if conf == nil {
		err = errors.New("StaticDefinedPositions: conf (*etcd.Node) is nil")
		return
	}

	onePosition := [3]float64{0, 0, 0}
	mobilityManager.positions = append(mobilityManager.positions, onePosition)
	for _, node := range conf.Nodes {
		if node.Dir && strings.HasSuffix(node.Key, "/positions") {
			for _, position := range conf.Nodes {
				var xs, ys, zs bool
				i := len(mobilityManager.positions) - 1
				for _, e := range position.Nodes {
					if !e.Dir && strings.HasSuffix(e.Key, "/x") {
						mobilityManager.positions[i][0], err = strconv.ParseFloat(e.Value, 64)
						if err != nil {
							err = fmt.Errorf("Parsing position value [%s] error: %s\n", e.Key, err.Error())
							return
						}
						xs = true
					} else if !e.Dir && strings.HasSuffix(e.Key, "/y") {
						mobilityManager.positions[i][1], err = strconv.ParseFloat(e.Value, 64)
						if err != nil {
							err = fmt.Errorf("Parsing position value [%s] error: %s\n", e.Key, err.Error())
							return
						}
						ys = true
					} else if !e.Dir && strings.HasSuffix(e.Key, "/z") {
						mobilityManager.positions[i][2], err = strconv.ParseFloat(e.Value, 64)
						if err != nil {
							err = fmt.Errorf("Parsing position value [%s] error: %s\n", e.Key, err.Error())
							return
						}
						zs = true
					}
				}
				if xs && ys && zs {
					mobilityManager.positions = append(mobilityManager.positions, onePosition)
				}
			}
		}
	}
	return nil
}

func (mobilityManager *staticDefinedPositions) Initialize(positionManager squirrel.PositionManager) {
	ch := make(chan []int)
	positionManager.RegisterEnabledChanged(ch)
	go func() {
		for {
			enabled := <-ch
			for i, index := range enabled {
				if i < len(mobilityManager.positions) {
					p := mobilityManager.positions[i]
					positionManager.Set(index, p[0], p[1], p[2])
				}
			}
		}
	}()
}
