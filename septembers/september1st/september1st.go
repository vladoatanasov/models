package september1st

import (
	"errors"
	"math"
	"math/rand"
	"strconv"
	"strings"

	"github.com/coreos/go-etcd/etcd"
	"github.com/squirrel-land/squirrel"
)

type september1st struct {
	positionManager    squirrel.PositionManager
	noDeliveryDistance float64
}

func NewSeptember1st() squirrel.September {
	return &september1st{}
}

func (september *september1st) ParametersHelp() string {
	return `September1st delivers packets only based on distance between nodes. It applies
a packet loss (d/D)^4 to each packet, where d is the distance between the two
nodes, and D is the maximum communication range. It does not consider
interference.

  "transmission_range": float64, required;
												Maximum transmission range, i.e., the lowest distance
												where packet delivery ratio will be zero.
    `
}

func (september *september1st) Configure(conf *etcd.Node) (err error) {
	if conf == nil {
		err = errors.New("September1st: conf (*etcd.Node) is nil")
		return
	}

	found := false
	for _, node := range conf.Nodes {
		if !node.Dir && strings.HasSuffix(node.Key, "transmission_range") {
			september.noDeliveryDistance, err = strconv.ParseFloat(node.Value, 64)
			if err != nil {
				return
			}
			found = true
			break
		}
	}
	if !found {
		err = errors.New("transmission_range is missing from config")
	}
	return
}

func (september *september1st) Initialize(positionManager squirrel.PositionManager) {
	september.positionManager = positionManager
}

func (september *september1st) SendUnicast(source int, destination int, size int) bool {
	return september.isToBeDelivered(source, destination)
}

func (september *september1st) SendBroadcast(source int, size int, underlying []int) []int {
	count := 0
	for _, i := range september.positionManager.Enabled() {
		if i != source && september.isToBeDelivered(source, i) {
			underlying[count] = i
			count++
		}
	}
	return underlying[:count]
}

func (september *september1st) isToBeDelivered(id1 int, id2 int) bool {
	if september.positionManager.IsEnabled(id1) && september.positionManager.IsEnabled(id2) {
		dist := september.positionManager.Distance(id1, id2)
		if dist < september.noDeliveryDistance*0.8 {
			return true
		}
		return rand.Float64() > math.Pow(dist/september.noDeliveryDistance, 4)
	} else {
		return false
	}
}
