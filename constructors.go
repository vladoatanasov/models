package models

import (
	"github.com/squirrel-land/models/mobilityManagers/grpcUpdatablePositions"
	"github.com/squirrel-land/models/mobilityManagers/interactivePositions"
	"github.com/squirrel-land/models/mobilityManagers/staticDefinedPositions"
	"github.com/squirrel-land/models/mobilityManagers/staticUniformPositions"
	"github.com/squirrel-land/models/septembers/csmaca"
	"github.com/squirrel-land/models/septembers/distanceBased"
	"github.com/squirrel-land/models/septembers/passThrough"
	"github.com/squirrel-land/squirrel"
)

var MobilityManagers = map[string]func() squirrel.MobilityManager{
	"StaticUniformPositions": staticUniformPositions.NewStaticUniformPositions,
	"StaticDefinedPositions": staticDefinedPositions.NewStaticDefinedPositions,
	"InteractivePositions":   interactivePositions.NewInteractivePositions,
	"gRPCUpdatablePositions": grpcUpdatablePositions.NewGRPCUpdatablePositions,
}

var Septembers = map[string]func() squirrel.September{
	"PassThrough":   passThrough.CreateSeptember,
	"DistanceBased": distanceBased.CreateSeptember,
	"CSMA/CA":       csmaca.CreateSeptember,

	/* legacy names */
	"September0th": passThrough.CreateSeptember,
	"September1st": distanceBased.CreateSeptember,
	"September2nd": csmaca.CreateSeptember,
}
