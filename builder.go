package main

import (
	"fmt"
	"math"
	"math/rand"

	"strconv"

	svg "github.com/ajstarks/svgo"
	"github.com/gin-gonic/gin"
)

const CANVAS_PADDING = 32
const CANVAS_WIDTH = 1024
const CANVAS_HEIGHT = 1024

func random(from, to int) int {
	if from >= to {
		return from
	}
	return rand.Intn(to-from) + from
}

func Round(f float64) int {
	return int(math.Floor(f + .5))
}

func randcolor() (int, int, int) {
	return rand.Intn(128) + 127, rand.Intn(128) + 127, rand.Intn(128) + 127
}

func lightColor(r, g, b int, k float64) (int, int, int) {
	k = math.Max(float64(k), 0)
	k = math.Min(float64(k), 1)

	r = int(math.Min(float64(r)+float64(r)*k, 255))
	g = int(math.Min(float64(g)+float64(g)*k, 255))
	b = int(math.Min(float64(b)+float64(b)*k, 255))
	fmt.Printf("r:%v g:%v b:%v\n", r, g, b)
	return r, g, b
}

func darkColor(r, g, b int, k float64) (int, int, int) {
	k = math.Max(float64(k), 0)
	k = math.Min(float64(k), 1)
	k = 1 - k
	return Round(float64(r) * k), Round(float64(g) * k), Round(float64(b) * k)
}

func fill(r, g, b int) string {
	return fmt.Sprintf("fill:rgb(%d,%d,%d)", r, g, b)
}

func drawCanvas(c *gin.Context) {
	c.Writer.Header().Set("Content-type", "image/svg+xml")
	canvas := svg.New(c.Writer)
	canvas.Start(CANVAS_WIDTH, CANVAS_HEIGHT)
	canvas.Title("Cubes")
	drawFloor(canvas)
	x, _ := strconv.Atoi(c.Query("x"))
	y, _ := strconv.Atoi(c.Query("y"))
	w, _ := strconv.Atoi(c.Query("w"))
	h, _ := strconv.Atoi(c.Query("h"))
	z, _ := strconv.Atoi(c.Query("z"))

	drawCube(canvas, x, y, w, h, z)

	drawCube(canvas, x+w, y, w, h*2, z)
	drawCube(canvas, x, y+z, w, h*3, z)
	drawCube(canvas, x, y+z*2, w, h*2, z)
	drawCube(canvas, x+w*3, y, w, h*5, z*5)
	drawCube(canvas, x+w*3, y+y*2, w, h*4, z)

	canvas.End()
}

func drawRepo(c *gin.Context) {

}

var m3 = math.Sqrt(3)

func p(i int) int {
	return int(m3*2*float64(i) + 0.5)
}

func drawCube(canvas *svg.SVG, xt, yt, w, h, z int) {
	w = int(w / 4)
	z = int(z / 4)
	y := CANVAS_HEIGHT - int(CANVAS_WIDTH*(1/m3)) - h
	x := CANVAS_WIDTH / 2

	x -= int((m3 / 2) * float64(xt))
	y += int(xt / 2)

	x += int((m3 / 2) * float64(yt))
	y += int(yt / 2)

	r, g, b := randcolor()

	tx := []int{x, x + p(z), x - p(w) + p(z), x - p(w), x}
	ty := []int{y, y + z*2, y + (z+w)*2, y + w*2, y}
	canvas.Polygon(tx, ty, fill(lightColor(r, g, b, 0.1)))

	lx := []int{x - p(w), x - p(w) + p(z), x - p(w) + p(z), x - p(w), x - p(w)}
	ly := []int{y + w*2, y + (z+w)*2, y + (z+w)*2 + h, y + w*2 + h, y + w*2}
	canvas.Polygon(lx, ly, fill(darkColor(r, g, b, 0.2)))

	rx := []int{x + p(z), x + p(z), x - p(w) + p(z), x - p(w) + p(z), x + p(z)}
	ry := []int{y + z*2, y + z*2 + h, y + (z+w)*2 + h, y + (z+w)*2, y + z*2}
	fmt.Printf("x: %+v\ny: %+v\n", rx, ry)
	canvas.Polygon(rx, ry, fill(r, g, b))
}

func drawFloor(canvas *svg.SVG) {
	r, g, b := random(190, 196), random(206, 255), random(60, 72)
	t := 1 / m3
	x := int(CANVAS_WIDTH / 2)
	y := CANVAS_HEIGHT - int(CANVAS_WIDTH*t)

	tx := []int{x, CANVAS_WIDTH, x, 0, x}
	ty := []int{y, y + int(CANVAS_HEIGHT*t/2), CANVAS_HEIGHT, y + int(CANVAS_HEIGHT*t/2), y}
	canvas.Polygon(tx, ty, fill(r, g, b))

}
