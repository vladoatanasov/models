package csmaca

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/squirrel-land/squirrel"
)

type csmaca struct {
	transmissionRange float64
	interferenceRange float64

	positionManager squirrel.PositionManager
	buckets         []*leakyBucket // measured by number of nanoseconds used;
	dataRate        float64        // bit per nanosecond

	difs time.Duration // nanoseconds
	phy  *phy

	ucastMaxTXAttempts int // max # of transmissions for each frame
}

func CreateSeptember() squirrel.September {
	ret := new(csmaca)
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
  "mac_protocol"      : string, required;
                        The MAC layer protocol to use. Has to be one of:
                        802.11a, 802.11g, 802.11p10MHz, 802.11p20MHz.
  "max_ucast_attempts": int, required;
                        The maximum number of transmissions that a STA can
                        attempt for the same fame. This is for MAC layer
                        unicast retransmissions.
	"data_rate_mbps":     float64, required;
												MAC data rate in Mbps.
    `
}

func (c *csmaca) Configure(conf *etcd.Node) (err error) {
	if conf == nil {
		err = errors.New("csmaca: conf (*etcd.Node) is nil")
		return
	}

	for _, node := range conf.Nodes {
		if !node.Dir && strings.HasSuffix(node.Key, "transmission_range") {
			c.transmissionRange, err = strconv.ParseFloat(node.Value, 64)
			if err != nil {
				return
			}
		} else if !node.Dir && strings.HasSuffix(node.Key, "interference_range") {
			c.interferenceRange, err = strconv.ParseFloat(node.Value, 64)
			if err != nil {
				return
			}
		} else if !node.Dir && strings.HasSuffix(node.Key, "mac_protocol") {
			switch node.Value {
			case "802.11a":
				c.phy = phy80211a
			case "802.11g":
				c.phy = phy80211g
			case "802.11p10MHz":
				c.phy = phy80211p10
			case "802.11p20MHz":
				c.phy = phy80211p20
			default:
				err = errors.New("unknown mac_protocol")
				return
			}
		} else if !node.Dir && strings.HasSuffix(node.Key, "max_ucast_attempts") {
			c.ucastMaxTXAttempts, err = strconv.Atoi(node.Value)
			if err != nil {
				return
			}
		} else if !node.Dir && strings.HasSuffix(node.Key, "data_rate_mbps") {
			c.dataRate, err = strconv.ParseFloat(node.Value, 64)
			if err != nil {
				return
			}
			c.dataRate = c.dataRate * 1024 * 1024 * 1e-9
		}
	}

	var errorParameters []string
	if c.transmissionRange <= 0 {
		errorParameters = append(errorParameters, "transmission_range")
	}
	if c.interferenceRange <= 0 {
		errorParameters = append(errorParameters, "interference_range")
	}
	if c.phy == nil {
		errorParameters = append(errorParameters, "mac_protocol")
	}
	if c.ucastMaxTXAttempts <= 0 {
		errorParameters = append(errorParameters, "max_ucast_attempts")
	}
	if c.dataRate <= 0 {
		errorParameters = append(errorParameters, "data_rate_mbps")
	}

	if len(errorParameters) != 0 {
		err = fmt.Errorf("parameter(s) missing or invalid: %v", errorParameters)
		return
	}

	c.difs = c.phy.sifs + 2*c.phy.slot
	return
}

func (c *csmaca) Initialize(positionManager squirrel.PositionManager) {
	c.positionManager = positionManager
	c.buckets = make([]*leakyBucket, positionManager.Capacity())
	for it := range c.buckets {
		c.buckets[it] = NewLeakyBucket(50*1000*1000, time.Millisecond, 1000*1000)
		c.buckets[it].Go()
	}
}

func (c *csmaca) bo(cw int) time.Duration { // back-off time
	// for statistic purpose, we take the average of contention window time
	return c.phy.slot * time.Duration(cw) / 2
}

func (c *csmaca) durationByBytes(bytes int) time.Duration {
	return time.Duration(float64(bytes*8)/c.dataRate) * time.Nanosecond
}

func (c *csmaca) durationOfDataFrame(payloadSize int) time.Duration {
	frameBytes := payloadSize + 34 // MAC data frame header is 34 bytes
	return c.durationByBytes(frameBytes)
}

func (c *csmaca) durationOfAckFrame() time.Duration {
	return c.phy.sifs + c.durationByBytes(14) // ACK is 14 bytes
}

func (c *csmaca) deliverRate(dest int, dist float64) float64 {
	usage := c.buckets[dest].Usage()
	p_rate := (1-usage)*.1 + .9 // usage transformed from [0, 1] to [.9, 1]
	return p_rate * (1 - math.Pow(dist/c.transmissionRange, 3))
}

func (c *csmaca) SendUnicast(source int, destination int, size int) bool {
	if !(c.positionManager.IsEnabled(source) && c.positionManager.IsEnabled(destination)) {
		return false
	}

	durationFrame := c.durationOfDataFrame(size)
	durationAck := c.durationOfAckFrame()

	usend := func(cw int) bool {

		// Go through source bucket
		if !c.buckets[source].In(int64(c.difs + c.bo(cw) + durationFrame)) {
			return false
		}

		dist := c.positionManager.Distance(source, destination)

		// Since the data frame is out in the air, interference should be put on
		// neighbor nodes of the source node
		for _, i := range c.positionManager.Enabled() {
			if i == source || i == destination {
				// source's bucket is already done;
				// and we consider the destination's bucket later
				continue
			}
			d1 := c.positionManager.Distance(source, i)
			if rand.Float64() < 1-math.Pow(d1/c.interferenceRange, 6) {
				c.buckets[i].In(int64(durationFrame))
			}
		}

		// The data frame takes the adventure in the air (fading, etc.)
		if rand.Float64() > c.deliverRate(destination, dist) {
			return false
		}

		// data frame Go through destination bucket
		if !c.buckets[destination].In(int64(durationFrame)) {
			return false
		}

		// ACK Go through destination bucket
		if !c.buckets[destination].In(int64(durationAck)) {
			return false
		}

		// Since the packet is delivered, ACK should be sent. Interference should be
		// put on neighbor nodes of the destination node
		for _, i := range c.positionManager.Enabled() {
			if i == destination {
				// destination's bucket is already done
				continue
			}
			d2 := c.positionManager.Distance(destination, i)
			if rand.Float64() < 1-math.Pow(d2/c.interferenceRange, 6) {
				c.buckets[i].In(int64(durationAck))
			}
		}

		// The packet is gonna be delivered!
		return true
	}

	for i, cw := 0, c.phy.cwMin; i < c.ucastMaxTXAttempts; i++ {
		if usend(cw) {
			return true
		}
		if cw <= c.phy.cwMax/2 {
			cw = cw*2 - 1
		}
	}
	return false
}

func (c *csmaca) SendBroadcast(source int, size int, underlying []int) []int {
	if !c.positionManager.IsEnabled(source) {
		return underlying[:0]
	}

	durationFrame := c.durationOfDataFrame(size)

	// Go through source bucket
	if !c.buckets[source].In(int64(c.difs + c.bo(c.phy.cwMin) + durationFrame)) {
		return underlying[:0]
	}

	count := 0
	for _, i := range c.positionManager.Enabled() {
		dist := c.positionManager.Distance(source, i)
		if dist < c.transmissionRange {
			// The packet takes the adventure in the air (fading, etc.)
			if rand.Float64() > c.deliverRate(i, dist) {
				continue
			}

			// Go through destination bucket. If rejected by the bucket, the
			// broadcasted packet should not be delivered to this node
			if !c.buckets[i].In(int64(durationFrame)) {
				continue
			}

			// The packet is gonna be delivered!
			underlying[count] = i
			count++
		} else if dist < c.interferenceRange {
			// not in communication range, but still generating interference
			c.buckets[i].In(int64(durationFrame))
		}
	}
	return underlying[:count]
}
