package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type GolWorker struct {
	listener net.Listener
}

const alive = 255
const dead = 0

func (w *GolWorker)mod(x, m int) int {
	return (x + m) % m
}

func (w *GolWorker)CalculateNewState(req stubs.RequestToWorker, res *stubs.ResponseFromWorker) (err error) {
	if req.Stop {
		w.listener.Close()
		return
	}

	//fmt.Println("h", req.ImageHeight)
	//fmt.Println("before", req.Turns)
	AliveNum := 0
	for y := 0; y < req.ImageHeight; y++ {
		for x := 0; x < req.ImageWidth; x++ {
			//calculate neighbours
			neighbours := 0
			for i := -1; i <= 1; i++ {
				for j := -1; j <= 1; j++ {
					if i != 0 || j != 0 {
						if req.World[w.mod(y+i, req.ImageHeight)][w.mod(x+j, req.ImageWidth)] == alive {
							neighbours++
						}
					}
				}
			}

			//make new world
			if req.World[y][x] == alive {
				if neighbours < 2 || neighbours > 3 {
					req.NewWorld[y][x] = dead
				} else {
					req.NewWorld[y][x] = alive
					AliveNum++
				}
			} else {
				if neighbours == 3 {
					req.NewWorld[y][x] = alive
					AliveNum++
				} else {
					req.NewWorld[y][x] = dead
				}
			}
		}
	}
	res.Turns = req.Turns + 1
	res.NewWorld = req.NewWorld
	res.AliveNumber = AliveNum
	//fmt.Println("after", res.Turns)
	return
}

func main(){
	pAddr := flag.String("port","8030","Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	golWorker := &GolWorker{}
	golWorker.listener, _ = net.Listen("tcp", ":" + *pAddr)
	rpc.Register(golWorker)
	defer golWorker.listener.Close()
	rpc.Accept(golWorker.listener)
}