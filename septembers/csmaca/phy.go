package csmaca

import "time"

type phy struct {
	slot     time.Duration
	sifs     time.Duration
	preamble time.Duration

	cwMin int // # slots
	cwMax int // # slots
}

var phyOFDM20 *phy = &phy{
	slot:     9 * time.Microsecond,
	sifs:     16 * time.Microsecond,
	preamble: 16 * time.Microsecond,
	cwMin:    15,
	cwMax:    1023,
}

var phyOFDM10 *phy = &phy{
	slot:     13 * time.Microsecond,
	sifs:     32 * time.Microsecond,
	preamble: 32 * time.Microsecond,
	cwMin:    15,
	cwMax:    1023,
}

var phyOFDM5 *phy = &phy{
	slot:     21 * time.Microsecond,
	sifs:     64 * time.Microsecond,
	preamble: 64 * time.Microsecond,
	cwMin:    15,
	cwMax:    1023,
}

var (
	phy80211a   = phyOFDM20
	phy80211g   = phyOFDM20
	phy80211p10 = phyOFDM10
	phy80211p20 = phyOFDM20
)
