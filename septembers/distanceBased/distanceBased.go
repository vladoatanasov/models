package distanceBased

import (
	"errors"
	"math"
	"math/rand"
	"strconv"
	"strings"

	"github.com/coreos/go-etcd/etcd"
	"github.com/squirrel-land/squirrel"
)

type distanceBased struct {
	positionManager    squirrel.PositionManager
	noDeliveryDistance float64
}

func CreateSeptember() squirrel.September {
	return &distanceBased{}
}

func (d *distanceBased) ParametersHelp() string {
	return `DistanceBased is a september that delivers packets only based on distance
between nodes. It applies a packet loss (d/D)^4 to each packet, where d is the
distance between the two nodes, and D is the maximum communication range. It
does not consider interference.

  "transmission_range": float64, required;
												Maximum transmission range, i.e., the lowest distance
												where packet delivery ratio will be zero.
    `
}

func (d *distanceBased) Configure(conf *etcd.Node) (err error) {
	if conf == nil {
		err = errors.New("DistanceBased: conf (*etcd.Node) is nil")
		return
	}

	found := false
	for _, node := range conf.Nodes {
		if !node.Dir && strings.HasSuffix(node.Key, "transmission_range") {
			d.noDeliveryDistance, err = strconv.ParseFloat(node.Value, 64)
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

func (d *distanceBased) Initialize(positionManager squirrel.PositionManager) {
	d.positionManager = positionManager
}

func (d *distanceBased) SendUnicast(source int, destination int, size int) bool {
	return d.isToBeDelivered(source, destination)
}

func (d *distanceBased) SendBroadcast(source int, size int, underlying []int) []int {
	count := 0
	for _, i := range d.positionManager.Enabled() {
		if i != source && d.isToBeDelivered(source, i) {
			underlying[count] = i
			count++
		}
	}
	return underlying[:count]
}

func (d *distanceBased) isToBeDelivered(id1 int, id2 int) bool {
	if d.positionManager.IsEnabled(id1) && d.positionManager.IsEnabled(id2) {
		dist := d.positionManager.Distance(id1, id2)
		if dist < d.noDeliveryDistance*0.8 {
			return true
		}
		return rand.Float64() > math.Pow(dist/d.noDeliveryDistance, 4)
	} else {
		return false
	}
}
