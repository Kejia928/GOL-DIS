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
}

var world [][]byte
var aliveNum int
var turn int

func (b *Broker) RunAllTurns(req stubs.RequestToBroker, res *stubs.ResponseFromBroker) (err error){
	client, _ := rpc.Dial("tcp", "127.0.0.1:8030")
	defer client.Close()

	if req.Stop {
		request := stubs.RequestToWorker{Stop: false}
		response := new(stubs.ResponseFromWorker)
		client.Call(stubs.Worker, request, response)
		b.listener.Close()
		return
	}

	turn = 0
	aliveNum = 0
	world = req.World
	fmt.Println("receiving")
	newWorld := CreatWorld(req.Params.ImageHeight, req.Params.ImageWidth)
	fmt.Println("before", turn)
	for {
		if turn >= req.Params.Turns{
			break
		}
		request := stubs.RequestToWorker{
			World:       world,
			NewWorld:    newWorld,
			Turns:       turn,
			ImageHeight: req.Params.ImageHeight,
			ImageWidth:  req.Params.ImageWidth,
			Stop:        false,
		}
		response := new(stubs.ResponseFromWorker)
		client.Call(stubs.Worker, request, response)

		newWorld = response.NewWorld
		aliveNum = response.AliveNumber
		turn = response.Turns
		world = newWorld
		fmt.Println("after", response.Turns)
		fmt.Println("global", turn)
	}
	return
}

func (b *Broker) GetNewData(req stubs.RequestToBroker, res *stubs.ResponseFromBroker) (err error){

	fmt.Println("should have:", turn)
	res.Turn = turn
	res.AliveNumber = aliveNum
	res.NewWorld = world
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
