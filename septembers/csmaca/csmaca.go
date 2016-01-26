package csmaca

import (
	"errors"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/squirrel-land/squirrel"
)

type csmaca struct {
	noDeliveryDistance float64
	interferenceRange  float64

	positionManager squirrel.PositionManager
	buckets         []*leakyBucket // measured by number of nanoseconds used;
	dataRate        float64        // bit data rates in bit/nanosecond

	// MAC layer time in nanoseconds
	difs    int
	sifs    int
	slot    int
	cWindow int // assuming fixed MAC layer contention window

	// MAC layer frame properties in bits
	macFrameMaxBody  int
	macFrameOverhead int
}

func CreateSeptember() squirrel.September {
	ret := new(csmaca)
	ret.dataRate = 54 * 1024 * 1024 * 1e-9 // 54 Mbps
	ret.slot = 9e3                         // 9 microseconds
	ret.sifs = 10e3                        // 10 microseconds
	ret.difs = ret.sifs + 2*ret.slot       // 28 microseconds
	ret.cWindow = 18                       // 18 slots
	ret.macFrameMaxBody = 2312 * 8
	ret.macFrameOverhead = 34
	return ret
}

func (c *csmaca) ParametersHelp() string {
	return `
CSMA/CA is a september that tries to mimic the CSMA/CA process in 802.11. It
delivers packets based on a near 802.11 Ad-hoc model. It considers distance
between nodes and interference, etc..

  "transmission_range": float64, required;
												Maximum transmission range, i.e., the lowest distance
												where packet ratio will be zero.
  "interference_range": float64, required;
												Maximum interference range, normally slighly larger
												than 2x transmission range.
    `
}

func (c *csmaca) Configure(conf *etcd.Node) (err error) {
	if conf == nil {
		err = errors.New("csmaca: conf (*etcd.Node) is nil")
		return
	}

	for _, node := range conf.Nodes {
		if !node.Dir && strings.HasSuffix(node.Key, "transmission_range") {
			c.noDeliveryDistance, err = strconv.ParseFloat(node.Value, 64)
			if err != nil {
				return
			}
		} else if !node.Dir && strings.HasSuffix(node.Key, "interference_range") {
			c.interferenceRange, err = strconv.ParseFloat(node.Value, 64)
			if err != nil {
				return
			}
		}
	}

	if c.noDeliveryDistance <= 0 || c.interferenceRange <= 0 {
		err = errors.New("LowestZeroPacketDeliveryDistance or InterferenceRange is missing from config or not greater than 0")
	}

	return
}

func (c *csmaca) Initialize(positionManager squirrel.PositionManager) {
	c.positionManager = positionManager
	c.buckets = make([]*leakyBucket, positionManager.Capacity())
	for it := range c.buckets {
		c.buckets[it] = NewLeakyBucket(100*1000*1000, time.Millisecond, 1000*1000)
		c.buckets[it].Go()
	}
}

func (c *csmaca) nanosecByData(bytes int) int {
	framebody := int(float64(bytes*8) / c.dataRate)
	frameOverhead := c.macFrameOverhead * (bytes*8/c.macFrameMaxBody + 1)
	return framebody + frameOverhead
}

func (c *csmaca) cw() int {
	return c.slot * c.cWindow / 2
}

func (c *csmaca) nanosecByPacket(packetSize int) int {
	return c.difs + c.cw() + c.nanosecByData(packetSize) + c.sifs + c.nanosecByData(0) // the last nanosecByData(0) is for MAC layer ACK
}

func (c *csmaca) ackIntererence() int {
	return c.difs + c.cw() + c.sifs + c.nanosecByData(0)
}

func (c *csmaca) deliverRate(dest int, dist float64) float64 {
	usage := c.buckets[dest].Usage()
	p_rate := (1-usage)*.1 + .9 // usage transformed from [0, 1] to [.9, 1]
	return p_rate * (1 - math.Pow(dist/c.noDeliveryDistance, 3))
}

func (c *csmaca) SendUnicast(source int, destination int, size int) bool {
	if !(c.positionManager.IsEnabled(source) && c.positionManager.IsEnabled(destination)) {
		return false
	}

	// Go through source bucket
	if !c.buckets[source].In(c.nanosecByPacket(size)) {
		return false
	}

	dist := c.positionManager.Distance(source, destination)

	// Since the packet is out in the air, interference should be put on neighbor nodes
	for _, i := range c.positionManager.Enabled() {
		d1 := c.positionManager.Distance(source, i)
		d2 := c.positionManager.Distance(destination, i)
		if i == destination || i == source {
			continue
		}
		if rand.Float64() < 1-math.Pow(d1/c.interferenceRange, 6) {
			c.buckets[i].In(c.nanosecByPacket(size))
		} else if rand.Float64() < 1-math.Pow(d2/c.interferenceRange, 6) {
			c.buckets[i].In(c.ackIntererence())
		}
	}

	// The packet takes the adventure in the air (fading, etc.)
	if rand.Float64() > c.deliverRate(destination, dist) {
		return false
	}

	// Go through destination bucket
	if !c.buckets[destination].In(c.nanosecByPacket(size)) {
		return false
	}

	// The packet is gonna be delivered!
	return true
}

func (c *csmaca) SendBroadcast(source int, size int, underlying []int) []int {
	if !c.positionManager.IsEnabled(source) {
		return underlying[:0]
	}
	// Go through source bucket
	if !c.buckets[source].In(size) {
		return underlying[:0]
	}

	count := 0
	for _, i := range c.positionManager.Enabled() {
		dist := c.positionManager.Distance(source, i)
		if dist < c.interferenceRange {
			if !c.buckets[i].In(c.nanosecByPacket(size)) {
				// Go through destination bucket. If rejected by the bucket, the
				// broadcasted packet should not be delivered to this node
				continue
			}
		}

		// The packet takes the adventure in the air (fading, etc.). There's still
		// a possibility the packet is not delivered to this node
		if rand.Float64() > c.deliverRate(i, dist) {
			continue
		}

		// The packet is gonna be delivered!
		underlying[count] = i
		count++
	}
	return underlying[:count]
}
