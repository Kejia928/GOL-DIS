package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

func CreatWorld(imageHeight, imageWidth int) [][]byte {
	world := make([][]byte, imageHeight)
	for i := range world {
		world[i] = make([]byte, imageWidth)
	}
	return world
}

type Broker struct {
	listener net.Listener
	world [][]byte
	aliveNum int
	turn int
}

var done = make(chan bool)

func (b *Broker) RunAllTurns(req stubs.RequestToBroker, res *stubs.ResponseFromBroker) (err error){
	client, _ := rpc.Dial("tcp", "127.0.0.1:8030")
	defer client.Close()
	if req.Stop {
		request := stubs.RequestToWorker{Stop: true}
		response := new(stubs.ResponseFromWorker)
		client.Call(stubs.CalculateNewState, request, response)
		b.listener.Close()
		return
	}

	b.turn = 0
	b.aliveNum = 0
	b.world = req.World
	newWorld := CreatWorld(req.Params.ImageHeight, req.Params.ImageWidth)
	for {
		if b.turn >= req.Params.Turns {
			break
		}
		request := stubs.RequestToWorker{
			World:       b.world,
			NewWorld:    newWorld,
			Turns:       b.turn,
			ImageHeight: req.Params.ImageHeight,
			ImageWidth:  req.Params.ImageWidth,
			Stop:        false,
		}
		response := new(stubs.ResponseFromWorker)
		client.Call(stubs.CalculateNewState, request, response)

		newWorld = response.NewWorld
		b.aliveNum = response.AliveNumber
		b.turn = response.Turns
		b.world = newWorld
		fmt.Println("runAllTurn", b.turn)
		done <- true
	}
	return
}

func (b *Broker) GetNewData(req stubs.RequestNewData, res *stubs.ResponseFromBroker) (err error){
	<- done
	res.Turn = b.turn
	fmt.Println("getNewData:", res.Turn)
	res.AliveNumber = b.aliveNum
	res.NewWorld = b.world
	return

}

func main() {
	pAddr := flag.String("port","8040","Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	broker := &Broker{}
	broker.listener, _ = net.Listen("tcp", ":"+*pAddr)
	rpc.Register(broker)
	defer broker.listener.Close()
	rpc.Accept(broker.listener)
}
