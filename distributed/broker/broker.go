package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type Task struct {
	World [][]byte
	NewWorld [][]byte
	Height int
	Width int
}

type Result struct {
	NewWorld [][]byte
	AliveNum int
	Turn int
}

type Broker struct {
	listener net.Listener
	world [][]byte
	aliveNum int
	turn int
	quit bool
}

func CreatWorld(imageHeight, imageWidth int) [][]byte {
	world := make([][]byte, imageHeight)
	for i := range world {
		world[i] = make([]byte, imageWidth)
	}
	return world
}

func GetTaskForEachWorker(wholeTask Task, thread int, tasks []Task) []Task {
	//fmt.Println("thread", thread)
	height := wholeTask.Height/thread
	for t := 0; t < thread; t++ {
		tasks[t].Width = wholeTask.Width
		if t != thread-1 {
			tasks[t].Height = height
			//fmt.Println(tasks[t].Height)
		} else {
			tasks[t].Height = wholeTask.Height-(height*(thread-1))
			//fmt.Println(tasks[t].Height)
		}
		//set suitable newWorld
		for y := 0; y < tasks[t].Height; y++ {
			for x := 0; x < tasks[t].Width; x++ {
				tasks[t].NewWorld[y][x] = wholeTask.NewWorld[y][x]    //do not matter the data inside each NewWorld
			}
		}
		//separate the world
		for y := 0; y <= tasks[t].Height+1; y++ {
			for x := 0; x < tasks[t].Width; x++ {
				if (height*t-1+y) < 0 {
					 tasks[t].World[y][x] = wholeTask.World[wholeTask.Height-1][x]
				} else if (height*t-1+y) > wholeTask.Height-1 {
					 tasks[t].World[y][x] = wholeTask.World[0][x]
				} else {
					 tasks[t].World[y][x] = wholeTask.World[(height*t-1)+y][x]
				}
			}
		}
	}
	return tasks
}

func GetWholeResult(results []Result, wholeResult Result, p stubs.Params) Result {
	Range := p.ImageHeight/p.Threads
	alive := 0
	for thread := 0; thread < p.Threads; thread++ {
		//put data into the new world
		if thread == p.Threads-1 {
			for y := 0; y < p.ImageHeight-(Range*(p.Threads-1)); y++ {
				for x := 0; x < p.ImageWidth; x++ {
					wholeResult.NewWorld[(Range*(p.Threads-1))+y][x] = results[thread].NewWorld[y][x]
				}
			}
		} else {
			for y := 0; y < Range; y++ {
				for x := 0; x < p.ImageWidth; x++ {
					wholeResult.NewWorld[((Range)*thread)+y][x] = results[thread].NewWorld[y][x]
				}
			}
		}
		alive += results[thread].AliveNum
		wholeResult.Turn = results[thread].Turn
	}
	wholeResult.AliveNum = alive
	return wholeResult
}

var done = make(chan bool) //For channel synchronization
var quit = make(chan bool)
var waitGroup sync.WaitGroup

func (b *Broker) RunAllTurns(req stubs.RequestToBroker, res *stubs.ResponseFromBroker) (err error){
	client1, _ := rpc.Dial("tcp", "127.0.0.1:8040")
	client2, _ := rpc.Dial("tcp", "127.0.0.1:8050")
	client3, _ := rpc.Dial("tcp", "127.0.0.1:8060")
	client4, _ := rpc.Dial("tcp", "127.0.0.1:8070")
	client5, _ := rpc.Dial("tcp", "127.0.0.1:8080")
	client6, _ := rpc.Dial("tcp", "127.0.0.1:8090")
	client7, _ := rpc.Dial("tcp", "127.0.0.1:8010")
	client8, _ := rpc.Dial("tcp", "127.0.0.1:8020")
	defer client1.Close()
	defer client2.Close()
	defer client3.Close()
	defer client4.Close()
	defer client5.Close()
	defer client6.Close()
	defer client7.Close()
	defer client8.Close()

	//press "K", stop listen
	if req.Stop {
		request := stubs.RequestToWorker{Stop: true}
		response := new(stubs.ResponseFromWorker)
		client1.Call(stubs.CalculateNewState, request, response)
		client2.Call(stubs.CalculateNewState, request, response)
		client3.Call(stubs.CalculateNewState, request, response)
		client4.Call(stubs.CalculateNewState, request, response)
		client5.Call(stubs.CalculateNewState, request, response)
		client6.Call(stubs.CalculateNewState, request, response)
		client7.Call(stubs.CalculateNewState, request, response)
		client8.Call(stubs.CalculateNewState, request, response)
		b.listener.Close()
		return
	}

	b.turn = 0
	b.aliveNum = 0
	b.world = req.World
	b.quit = false
	// monitor for each server (true means free)
	c1 := make(chan bool, 1)
	c2 := make(chan bool, 1)
	c3 := make(chan bool, 1)
	c4 := make(chan bool, 1)
	c5 := make(chan bool, 1)
	c6 := make(chan bool, 1)
	c7 := make(chan bool, 1)
	c8 := make(chan bool, 1)
	// in the start, make all server all available
	c1 <- true
	c2 <- true
	c3 <- true
	c4 <- true
	c5 <- true
	c6 <- true
	c7 <- true
	c8 <- true
	newWorld := CreatWorld(req.Params.ImageHeight, req.Params.ImageWidth)
	var waitGroup2 sync.WaitGroup
	var mutex sync.Mutex

	// ----- Task ----- //
	WholeTask := Task{
		World:    b.world,
		NewWorld: newWorld,
		Height:   req.Params.ImageHeight,
		Width:    req.Params.ImageWidth,
	}
	tasks := make([]Task, req.Params.Threads)
	// ----- Result ----- //
	wholeResult := Result{
		NewWorld: newWorld,
		AliveNum: b.aliveNum,
		Turn:     b.turn,
	}
	results := make([]Result, req.Params.Threads)
	//Initial the list
	for i := 0; i < req.Params.Threads; i++{
		h := req.Params.ImageHeight/req.Params.Threads
		if i == req.Params.Threads-1 {
			h = req.Params.ImageHeight - ((req.Params.Threads-1) * (req.Params.ImageHeight / req.Params.Threads))
		}
		tasks[i].World = CreatWorld(h+2, req.Params.ImageWidth) //add halo region in the world
		tasks[i].NewWorld = CreatWorld(h, req.Params.ImageWidth)
		results[i].NewWorld = CreatWorld(h, req.Params.ImageWidth)
	}

	//run all turn
	for {
		if b.turn >= req.Params.Turns {
			break
		}
		// initial the whole task
		WholeTask.World = b.world
		// Separate the Task to each worker
		tasks = GetTaskForEachWorker(WholeTask, req.Params.Threads, tasks)

		// ----- Run all thread ----- //
		for t := 0; t < req.Params.Threads; t++ {
			// --- Sent one task to server --- //
			request := stubs.RequestToWorker{
				World:       tasks[t].World,
				NewWorld:    tasks[t].NewWorld,
				Turns:       b.turn,
				ImageHeight: tasks[t].Height,
				ImageWidth:  tasks[t].Width,
				Thread:      t,
				Stop:        false,
			}
			select {
			case <- c1:
				fmt.Println("choose 1")
				waitGroup2.Add(1)
				response := new(stubs.ResponseFromWorker)
				go func() {
					client1.Call(stubs.CalculateNewState, request, response)
					mutex.Lock()
					results[response.Thread].Turn = response.Turns
					results[response.Thread].AliveNum = response.AliveNumber
					results[response.Thread].NewWorld = response.NewWorld
					mutex.Unlock()
					waitGroup2.Done()
					return
				}()
				c1 <- true
				goto ChooseNext
			case <- c2:
				fmt.Println("choose 2")
				waitGroup2.Add(1)
				response := new(stubs.ResponseFromWorker)
				go func() {
					client2.Call(stubs.CalculateNewState, request, response)
					mutex.Lock()
					results[response.Thread].Turn = response.Turns
					results[response.Thread].AliveNum = response.AliveNumber
					results[response.Thread].NewWorld = response.NewWorld
					mutex.Unlock()
					waitGroup2.Done()
					return
				}()
				c2 <- true
				goto ChooseNext
			case <- c3:
				fmt.Println("choose 3")
				waitGroup2.Add(1)
				response := new(stubs.ResponseFromWorker)
				go func() {
					client3.Call(stubs.CalculateNewState, request, response)
					mutex.Lock()
					results[response.Thread].Turn = response.Turns
					results[response.Thread].AliveNum = response.AliveNumber
					results[response.Thread].NewWorld = response.NewWorld
					mutex.Unlock()
					waitGroup2.Done()
					return
				}()
				c3 <- true
				goto ChooseNext
			case <- c4:
				fmt.Println("choose 4")
				waitGroup2.Add(1)
				response := new(stubs.ResponseFromWorker)
				go func() {
					client4.Call(stubs.CalculateNewState, request, response)
					mutex.Lock()
					results[response.Thread].Turn = response.Turns
					results[response.Thread].AliveNum = response.AliveNumber
					results[response.Thread].NewWorld = response.NewWorld
					mutex.Unlock()
					waitGroup2.Done()
					return
				}()
				c4 <- true
				goto ChooseNext
			case <- c5:
				fmt.Println("choose 5")
				waitGroup2.Add(1)
				response := new(stubs.ResponseFromWorker)
				go func() {
					client5.Call(stubs.CalculateNewState, request, response)
					mutex.Lock()
					results[response.Thread].Turn = response.Turns
					results[response.Thread].AliveNum = response.AliveNumber
					results[response.Thread].NewWorld = response.NewWorld
					mutex.Unlock()
					waitGroup2.Done()
					return
				}()
				c5 <- true
				goto ChooseNext
			case <- c6:
				fmt.Println("choose 6")
				waitGroup2.Add(1)
				response := new(stubs.ResponseFromWorker)
				go func() {
					client6.Call(stubs.CalculateNewState, request, response)
					mutex.Lock()
					results[response.Thread].Turn = response.Turns
					results[response.Thread].AliveNum = response.AliveNumber
					results[response.Thread].NewWorld = response.NewWorld
					mutex.Unlock()
					waitGroup2.Done()
					return
				}()
				c6 <- true
				goto ChooseNext
			case <- c7:
				fmt.Println("choose 7")
				waitGroup2.Add(1)
				response := new(stubs.ResponseFromWorker)
				go func() {
					client7.Call(stubs.CalculateNewState, request, response)
					mutex.Lock()
					results[response.Thread].Turn = response.Turns
					results[response.Thread].AliveNum = response.AliveNumber
					results[response.Thread].NewWorld = response.NewWorld
					mutex.Unlock()
					waitGroup2.Done()
					return
				}()
				c7 <- true
				goto ChooseNext
			case <- c8:
				fmt.Println("choose 8")
				waitGroup2.Add(1)
				response := new(stubs.ResponseFromWorker)
				go func() {
					client8.Call(stubs.CalculateNewState, request, response)
					mutex.Lock()
					results[response.Thread].Turn = response.Turns
					results[response.Thread].AliveNum = response.AliveNumber
					results[response.Thread].NewWorld = response.NewWorld
					mutex.Unlock()
					waitGroup2.Done()
					return
				}()
				c8 <- true
				goto ChooseNext
			}
			ChooseNext:
				continue
		}

		// ----- Combine the result and return to distributor -----//
		waitGroup2.Wait()
		wholeResult = GetWholeResult(results, wholeResult, req.Params)
		newWorld = wholeResult.NewWorld
		b.aliveNum = wholeResult.AliveNum
		b.turn = wholeResult.Turn
		b.world = newWorld
		waitGroup.Add(1)
		done <- true
		waitGroup.Wait()
		select {
		case <- quit:
			return
		default:
		}
	}
	return
}

func (b *Broker) GetNewData(req stubs.RequestNewData, res *stubs.ResponseFromBroker) (err error){
	<- done
	res.Turn = b.turn
	res.AliveNumber = b.aliveNum
	res.NewWorld = b.world
	waitGroup.Done()
	return
}

func (b *Broker) Quit(req stubs.RequestQuit, res *stubs.ResponseFromBroker) (err error){
	quit <- req.Quit
	return
}

func main() {
	pAddr := flag.String("port","8030","Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	broker := &Broker{}
	broker.listener, _ = net.Listen("tcp", ":"+*pAddr)
	rpc.Register(broker)
	defer broker.listener.Close()
	rpc.Accept(broker.listener)
}
