package models

import (
	"github.com/squirrel-land/models/common"
	"github.com/squirrel-land/models/mobilityManagers/interactivePositions"
	"github.com/squirrel-land/models/mobilityManagers/staticDefinedPositions"
	"github.com/squirrel-land/models/mobilityManagers/staticUniformPositions"
	"github.com/squirrel-land/models/septembers/september0th"
	"github.com/squirrel-land/models/septembers/september1st"
	"github.com/squirrel-land/models/septembers/september2nd"
)

var MobilityManagers = map[string]func() common.MobilityManager{
	"StaticUniformPositions": staticUniformPositions.NewStaticUniformPositions,
	"StaticDefinedPositions": staticDefinedPositions.NewStaticDefinedPositions,
	"InteractivePositions":   interactivePositions.NewInteractivePositions,
}

var Septembers = map[string]func() common.September{
	"September0th": september0th.NewSeptember0th,
	"September1st": september1st.NewSeptember1st,
	"September2nd": september2nd.NewSeptember2nd,
}
