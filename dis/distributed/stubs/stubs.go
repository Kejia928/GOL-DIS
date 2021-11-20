package stubs

var CalculateNewState = "GolWorker.CalculateNewState"
var RunAllTurns = "Broker.RunAllTurns"
var GetNewData = "Broker.GetNewData"

type RequestToWorker struct {
	World [][]byte
	NewWorld [][]byte
	Turns int
	ImageHeight int
	ImageWidth int
	Stop bool
}

type ResponseFromWorker struct {
	NewWorld [][]byte
	Turns int
	AliveNumber int
}

type RequestToBroker struct {
	Params Params
	World [][]byte
	Stop bool
}

type RequestNewData struct {

}

type ResponseFromBroker struct {
	NewWorld [][]byte
	Turn int
	AliveNumber int
}

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}