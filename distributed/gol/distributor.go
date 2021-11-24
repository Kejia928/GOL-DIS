package gol

import (
	"fmt"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	KeyPressed <-chan rune
}

func CreatWorld(imageHeight, imageWidth int) [][]byte {
	world := make([][]byte, imageHeight)
	for i := range world {
		world[i] = make([]byte, imageWidth)
	}
	return world
}

func LoadWorld(p Params, c distributorChannels) [][]byte {
	world := CreatWorld(p.ImageHeight, p.ImageWidth)
	//read file
	inputImageName := fmt.Sprintf("%vx%v", p.ImageHeight, p.ImageWidth)
	c.ioFilename <- inputImageName
	//store the world
	c.ioCommand <- ioInput
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			input := <-c.ioInput
			world[y][x] = input
		}
	}
	return world
}

func CalculateAliveCells(p Params, world [][]byte) []util.Cell {
	var aliveCells []util.Cell
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] == 255 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
}

func OutPutImage(p Params, world [][]byte, filename string, c distributorChannels) {
	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			outPut := world[y][x]
			c.ioOutput <- outPut
		}
	}
}

func GetFlippedCell(p Params, world, newWorld [][]byte) []util.Cell {
	var FlippedCell []util.Cell
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] != newWorld [y][x] {
				FlippedCell = append(FlippedCell, util.Cell{X: x, Y: y})
			}
		}
	}
	return FlippedCell
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	//RPC Client
	client, _ := rpc.Dial("tcp", "127.0.0.1:8030")
	defer client.Close()

	// Create a 2D slice and store the world.
	world := LoadWorld(p, c)
	newWorld := CreatWorld(p.ImageHeight, p.ImageWidth)

	tickerFinished := make(chan bool)
	Execute := make(chan bool)
	pause := false
	mutex := &sync.Mutex{}
	aliveNum := 0
	turn := 0

	//Ticker and Keyboard control
	go func() {
		for {
			//Ticker : every two second out put the alive cells
			ticker := time.NewTicker(2 * time.Second)
			select {
			case key := <- c.KeyPressed:
				switch key {
				case 's':
					filename := fmt.Sprintf("%vx%vx%v-%s", p.ImageHeight, p.ImageWidth, turn, "Press-s")
					OutPutImage(p, world, filename, c)
					c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: filename}
				case 'q':
					pause = true
					request := stubs.RequestQuit{Quit: true}
					response := new(stubs.ResponseFromBroker)
					client.Call(stubs.Quit, request, response)
					c.events <- StateChange{CompletedTurns: turn, NewState: Quitting}
					filename := fmt.Sprintf("%vx%vx%v-%s", p.ImageHeight, p.ImageWidth, turn, "Press-q")
					OutPutImage(p, world, filename, c)
					c.ioCommand <- ioCheckIdle
					<- c.ioIdle
					c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: filename}
					close(c.events)
				case 'p':
					if pause {
						Execute <- true
						pause = false
						fmt.Println("Continuing")
						c.events <- StateChange{CompletedTurns: turn, NewState: Executing}
					} else {
						pause = true
						c.events <- StateChange{CompletedTurns: turn, NewState: Paused}
					}
				case 'k':
					pause = true
					request := stubs.RequestToBroker{Stop: true}
					response := new(stubs.ResponseFromBroker)
					client.Call(stubs.RunAllTurns, request, response)
					fmt.Println("Server Stop!")
					c.events <- StateChange{CompletedTurns: turn, NewState: Quitting}
					filename := fmt.Sprintf("%vx%vx%v-%s", p.ImageHeight, p.ImageWidth, turn, "Press-k")
					OutPutImage(p, world, filename, c)
					c.ioCommand <- ioCheckIdle
					<- c.ioIdle
					c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: filename}
					close(c.events)
				}
			case <-ticker.C:
				mutex.Lock()
				if !pause {
					c.events <- AliveCellsCount{CompletedTurns: turn, CellsCount: aliveNum}
				}
				mutex.Unlock()
			case <-tickerFinished:
				ticker.Stop()
				return
			}
		}
	}()

	//sent started world to broker
	request := stubs.RequestToBroker{World: world, Params: stubs.Params(p), Stop: false}
	response := new(stubs.ResponseFromBroker)
	go client.Call(stubs.RunAllTurns, request, response)

	//update data from broker
	for {
		if turn >= p.Turns {
			break
		}
		req := stubs.RequestNewData{}
		res := new(stubs.ResponseFromBroker)
		client.Call(stubs.GetNewData, req, res)

		// get response from golWorker
		newWorld = res.NewWorld

		//keyboard control
		if pause {
			<- Execute
		}

		//SDL Live View
		var cells []util.Cell
		if turn == 0 {
			cells = CalculateAliveCells(p, newWorld)
		} else {
			cells = GetFlippedCell(p, world, newWorld)
		}
		for cell := 0; cell < len(cells); cell++ {
			c.events <- CellFlipped{CompletedTurns: turn, Cell: cells[cell]}
		}

		world = newWorld
		mutex.Lock()
		c.events <- TurnComplete{CompletedTurns: turn}
		turn = res.Turn
		aliveNum = res.AliveNumber
		mutex.Unlock()
	}

	tickerFinished <- true
	// Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          CalculateAliveCells(p, world),
	}

	//out put .pgm image
	outImageName := fmt.Sprintf("%vx%vx%v-%v", p.ImageHeight, p.ImageWidth, turn, p.Threads)
	OutPutImage(p, world, outImageName, c)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
