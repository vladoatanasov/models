package september0th

import (
	"github.com/coreos/go-etcd/etcd"
	"github.com/squirrel-land/squirrel"
)

type september0th struct {
	positionManager squirrel.PositionManager
}

func NewSeptember0th() squirrel.September {
	return &september0th{}
}

func (september *september0th) ParametersHelp() string {
	return `September0th delivers every packet sent into squirrel as long as the src and dst are valid.`
}

func (september *september0th) Configure(conf *etcd.Node) (err error) {
	return nil
}

func (september *september0th) Initialize(positionManager squirrel.PositionManager) {
	september.positionManager = positionManager
}

func (september *september0th) SendUnicast(source int, destination int, size int) bool {
	return september.isToBeDelivered(source, destination)
}

func (september *september0th) SendBroadcast(source int, size int, underlying []int) []int {
	count := 0
	for _, i := range september.positionManager.Enabled() {
		if i != source {
			underlying[count] = i
			count++
		}
	}
	return underlying[:count]
}

func (september *september0th) isToBeDelivered(id1 int, id2 int) bool {
	if september.positionManager.IsEnabled(id1) && september.positionManager.IsEnabled(id2) {
		return true
	} else {
		return false
	}
}
