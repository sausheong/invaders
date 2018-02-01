package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"os"
	"time"

	"github.com/disintegration/gift"
	termbox "github.com/nsf/termbox-go"
)

// parameters
var windowWidth, windowHeight = 400, 300
var aliensPerRow = 8
var aliensStartCol = 100
var alienSize = 30
var bombProbability = 0.005
var bombSpeed = 10

// sprites
var src = getImage("imgs/sprites.png")
var background = getImage("imgs/bg.png")
var cannonSprite = image.Rect(20, 47, 38, 59)
var cannonExplode = image.Rect(0, 47, 16, 57)
var alien1Sprite = image.Rect(0, 0, 20, 14)
var alien1aSprite = image.Rect(20, 0, 40, 14)
var alien2Sprite = image.Rect(0, 14, 20, 26)
var alien2aSprite = image.Rect(20, 14, 40, 26)
var alien3Sprite = image.Rect(0, 27, 20, 40)
var alien3aSprite = image.Rect(20, 27, 40, 40)
var alienExplode = image.Rect(0, 60, 16, 68)
var beamSprite = image.Rect(20, 60, 22, 65)
var bombSprite = image.Rect(0, 70, 10, 79)

// Sprite represents a sprite in the game
type Sprite struct {
	size     image.Rectangle // the sprite size
	Filter   *gift.GIFT      // normal filter used to draw the sprite
	FilterA  *gift.GIFT      // alternate filter used to draw the sprite
	FilterE  *gift.GIFT      // exploded filter used to draw the sprite
	Position image.Point     // top left position of the sprite
	Status   bool            // alive or dead
	Points   int             // number of points if destroyed
}

var aliens = []Sprite{}
var bombs = []Sprite{}

// sprite for laser cannon
var laserCannon = Sprite{
	size:     cannonSprite,
	Filter:   gift.New(gift.Crop(cannonSprite)),
	FilterE:  gift.New(gift.Crop(cannonExplode)),
	Position: image.Pt(50, 250),
	Status:   true,
}

// sprite for the laser beam
var beam = Sprite{
	size:     beamSprite,
	Filter:   gift.New(gift.Crop(beamSprite)),
	Position: image.Pt(laserCannon.Position.X+7, 250),
	Status:   false,
}

// used for creating alien sprites
func createAlien(x, y int, sprite, alt image.Rectangle, points int) (s Sprite) {
	s = Sprite{
		size:     sprite,
		Filter:   gift.New(gift.Crop(sprite)),
		FilterA:  gift.New(gift.Crop(alt)),
		FilterE:  gift.New(gift.Crop(alienExplode)),
		Position: image.Pt(x, y),
		Status:   true,
		Points:   points,
	}
	return
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	err := termbox.Init()
	if err != nil {
		panic(err)
	}

	// game variables
	loop := 0           // game loop
	beamShot := false   // the instance where the beam is shot
	gameOver := false   // end of game
	alienDirection := 1 // direction where alien is heading
	score := 0          // number of points scored in the game so far

	// poll for keyboard events in another goroutine
	events := make(chan termbox.Event, 1000)
	go func() {
		for {
			events <- termbox.PollEvent()
		}
	}()

	// show the start screen
	startScreen := getImage("imgs/start.png")
	printImage(startScreen)
start:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Ch == 's' || ev.Ch == 'S' {
				break start
			}
			if ev.Ch == 'q' {
				gameOver = true
				break start
			}
		}
	}

	// populate the aliens
	for i := aliensStartCol; i < aliensStartCol+(alienSize*aliensPerRow); i += alienSize {
		aliens = append(aliens, createAlien(i, 30, alien1Sprite, alien1aSprite, 30))
	}
	for i := aliensStartCol; i < aliensStartCol+(30*aliensPerRow); i += alienSize {
		aliens = append(aliens, createAlien(i, 55, alien2Sprite, alien2aSprite, 20))
	}
	for i := aliensStartCol; i < aliensStartCol+(30*aliensPerRow); i += alienSize {
		aliens = append(aliens, createAlien(i, 80, alien3Sprite, alien3aSprite, 10))
	}

	// main game loop
	for !gameOver {
		// if any of the keyboard events are captured
		select {
		case ev := <-events:
			if ev.Type == termbox.EventKey {
				// exit the game
				if ev.Key == termbox.KeyCtrlQ {
					gameOver = true
				}
				if ev.Key == termbox.KeySpace {
					if beam.Status == false {
						beamShot = true
					}
				}
				if ev.Key == termbox.KeyArrowRight {
					laserCannon.Position.X += 10
				}
				if ev.Key == termbox.KeyArrowLeft {
					laserCannon.Position.X -= 10
				}
			}

		default:

		}

		// create background
		dst := image.NewRGBA(image.Rect(0, 0, windowWidth, windowHeight))
		gift.New().Draw(dst, background)

		// process aliens
		for i := 0; i < len(aliens); i++ {
			aliens[i].Position.X = aliens[i].Position.X + 5*alienDirection
			if aliens[i].Status {
				// if alien is hit by a laser beam
				if collide(aliens[i], beam) {
					// draw the explosion
					aliens[i].FilterE.DrawAt(dst, src, image.Pt(aliens[i].Position.X, aliens[i].Position.Y), gift.OverOperator)
					// alien dies, player scores points
					aliens[i].Status = false
					score += aliens[i].Points
					// reset the laser beam
					resetBeam()
				} else {
					// show alternating alients
					if loop%2 == 0 {
						aliens[i].Filter.DrawAt(dst, src, image.Pt(aliens[i].Position.X, aliens[i].Position.Y), gift.OverOperator)
					} else {
						aliens[i].FilterA.DrawAt(dst, src, image.Pt(aliens[i].Position.X, aliens[i].Position.Y), gift.OverOperator)
					}
					// drop torpedoes
					if rand.Float64() < bombProbability {
						dropBomb(aliens[i])
					}
				}
			}
		}

		// draw bombs, if laser cannon is hit, game over
		for i := 0; i < len(bombs); i++ {
			bombs[i].Position.Y = bombs[i].Position.Y + bombSpeed
			bombs[i].Filter.DrawAt(dst, src, image.Pt(bombs[i].Position.X, bombs[i].Position.Y), gift.OverOperator)
			if collide(bombs[i], laserCannon) {
				gameOver = true
				laserCannon.FilterE.DrawAt(dst, src, image.Pt(laserCannon.Position.X, laserCannon.Position.Y), gift.OverOperator)
			}
		}
		// draw the laser cannon unless it's been destroyed
		if !gameOver {
			laserCannon.Filter.DrawAt(dst, src, image.Pt(laserCannon.Position.X, laserCannon.Position.Y), gift.OverOperator)
		}

		// move the aliens back and forth
		if aliens[0].Position.X < alienSize || aliens[aliensPerRow-1].Position.X > windowWidth-(2*alienSize) {
			alienDirection = alienDirection * -1
			for i := 0; i < len(aliens); i++ {
				aliens[i].Position.Y = aliens[i].Position.Y + 10
			}
		}

		// if the beam is shot, place the beam at start of the cannon
		if beamShot {
			beam.Position.X = laserCannon.Position.X + 7
			beam.Status = true
			beamShot = false
		}

		// keep drawing the beam as it moves every loop
		if beam.Status {
			beam.Filter.DrawAt(dst, src, image.Pt(beam.Position.X, beam.Position.Y), gift.OverOperator)
			beam.Position.Y -= 10
		}

		// if the beam leaves the window reset it
		if beam.Position.Y < 0 {
			resetBeam()
		}

		// if the aliens reach the position of the cannon, it's game over!
		if aliens[0].Position.Y > 180 {
			gameOver = true
		}
		printImage(dst)
		// pause a bit before ending the game
		if gameOver {
			time.Sleep(time.Second)
		}
		fmt.Println("\n\nSCORE:", score)
		loop++
	}
	termbox.Close()
	fmt.Println("\nGAME OVER!\nFinal score:", score)
}

func dropBomb(alien Sprite) {
	torpedo := Sprite{
		size:     bombSprite,
		Filter:   gift.New(gift.Crop(bombSprite)),
		Position: image.Pt(alien.Position.X+7, alien.Position.Y),
		Status:   true,
	}

	bombs = append(bombs, torpedo)
}

func resetBeam() {
	beam.Status = false
	beam.Position.Y = 250
}

func collide(s1, s2 Sprite) bool {
	spriteA := image.Rect(s1.Position.X, s1.Position.Y, s1.Position.X+s1.size.Dx(), s1.Position.Y+s1.size.Dy())
	spriteB := image.Rect(s2.Position.X, s2.Position.Y, s2.Position.X+s1.size.Dx(), s2.Position.Y+s1.size.Dy())
	if spriteA.Min.X < spriteB.Max.X && spriteA.Max.X > spriteB.Min.X &&
		spriteA.Min.Y < spriteB.Max.Y && spriteA.Max.Y > spriteB.Min.Y {
		return true
	}
	return false
}

// this only works for iTerm2!
func printImage(img image.Image) {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	imgBase64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	fmt.Printf("\x1b[2;0H\x1b]1337;File=inline=1:%s\a", imgBase64Str)
}

func getImage(filePath string) image.Image {
	imgFile, err := os.Open(filePath)
	defer imgFile.Close()
	if err != nil {
		fmt.Println("Cannot read file:", err)
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		fmt.Println("Cannot decode file:", err)
	}
	return img
}
