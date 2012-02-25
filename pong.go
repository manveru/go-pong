package main

import (
  "github.com/banthar/Go-SDL/sdl"
  "math"
  "time"
  "math/rand"
  "flag"
  "fmt"
)

var worldHeight *int = flag.Int("height", 200, "Height of Game")
var worldWidth *int = flag.Int("width", 200, "Width of Game")
var paddleSpeed *int = flag.Int("player-speed", 4, "Speed of player paddle")
var enemySpeed *int = flag.Int("enemy-speed", 4, "Speed of enemy paddle")
var ballSpeed *int = flag.Int("ball-speed", 4, "Speed of the ball")
var showUsage *bool = flag.Bool("help", false, "Show help")

func main() {
  sdlSetup()
  rand.Seed(time.Now().UnixNano())
  sdl.WM_SetCaption("PonGo", "")

  flag.Parse()

  if *showUsage {
    flag.PrintDefaults()
    return
  }

  world := NewWorld(*worldHeight, *worldWidth)

  go world.Run()
  world.HandleEvents()

  defer Quit(world)
}

type Ball struct {
  vector   *Vector2
  velocity *Vector2
  radius   float64
  speed    float64
  color    uint32
}

func NewBall(x, y float64) (ball *Ball) {
  ball = &Ball{
    vector: &Vector2{X: x, Y: y},
    speed:  float64(*ballSpeed),
    radius: 2.0,
    color:  0xffffff,
  }

  velocity := Vector2{X: float64(1 + rand.Intn(5)), Y: float64(rand.Intn(5))}
  ball.velocity = velocity.Normalize().MultiplyNum(ball.speed)
  return
}


func (ball *Ball) Update(world *World) {
  velocity := ball.velocity
  future := ball.vector.Plus(velocity)
  radius := ball.radius

  if velocity.Y < 0 && future.Y <= radius {
    velocity.Y = -velocity.Y
  }

  if velocity.Y > 0 && future.Y >= (float64(world.Height)-radius) {
    velocity.Y = -velocity.Y
  }

  if velocity.X < 0 {
    paddle := world.Paddle
    hit, _ := paddle.Hit(ball.vector, future)

    if hit {
      velocity.X = -velocity.X
      velocity = velocity.Normalize().MultiplyNum(ball.speed)
    } else if future.X <= radius {
      fmt.Println("Enemy scores")
      world.Score.Enemy++
      velocity.X = -velocity.X
    }
  }

  if velocity.X > 0 {
    enemy := world.Enemy
    hit, _ := enemy.Hit(ball.vector, future)

    if hit {
      velocity.X = -velocity.X
      velocity = velocity.Normalize().MultiplyNum(ball.speed)
    } else if future.X >= float64(world.Width) {
      fmt.Println("Player scores")
      world.Score.Paddle++
      velocity.X = -velocity.X
    }
  }

  ball.velocity = velocity
  ball.vector = ball.vector.Plus(velocity)
}

func (ball *Ball) Draw(world *World) {
  world.Screen.FillRect(ball.Rect(), ball.color)
}

func (ball *Ball) Rect() *sdl.Rect {
  size := uint16(ball.radius * 2)
  x := ball.vector.X - ball.radius
  y := ball.vector.Y - ball.radius
  return &sdl.Rect{X: int16(x), Y: int16(y), W: size, H: size}
}

type Paddle struct {
  vector               *Vector2
  target               *Vector2
  height, width, speed float64
  color                uint32
}

func NewPaddle(x, y, w, h float64) *Paddle {
  return &Paddle{
    vector: &Vector2{X: x, Y: y},
    target: &Vector2{X: x, Y: y},
    height: h,
    width:  w,
    color:  0x6666ff,
    speed:  float64(*paddleSpeed),
  }
}

func (paddle *Paddle) Go(x, y float64) {
  paddle.target = &Vector2{X: paddle.vector.X, Y: y}
}

func (paddle *Paddle) Update(world *World) {
  goal := paddle.target.Minus(paddle.vector)

  if goal.Length() > paddle.speed {
    goal = goal.Normalize().MultiplyNum(paddle.speed)
  }

  future := paddle.vector.Plus(goal)
  if future.Y < (paddle.height / 2) {
    return
  }
  if (future.Y + (paddle.height / 2)) > float64(world.Height) {
    return
  }
  paddle.vector = future
}

func (paddle *Paddle) Draw(world *World) {
  world.Screen.FillRect(paddle.Rect(), paddle.color)
}

func (paddle *Paddle) Rect() *sdl.Rect {
  h := paddle.height
  w := paddle.width
  x := paddle.vector.X - float64(w/2)
  y := paddle.vector.Y - float64(h/2)
  return &sdl.Rect{X: int16(x), Y: int16(y), W: uint16(w), H: uint16(h)}
}
func (paddle *Paddle) Hit(past, future *Vector2) (hit bool, place *Vector2) {
  // our front line
  halfHeight := paddle.height / 2
  halfWidth := paddle.width / 2
  x0, y0 := (paddle.vector.X + halfWidth), (paddle.vector.Y - halfHeight)
  x1, y1 := (paddle.vector.X + halfWidth), (paddle.vector.Y + halfHeight)

  return paddle.hitCore(x0, y0, x1, y1, past, future)
}

type Enemy struct {
  Paddle
}

func NewEnemy(x, y, w, h float64) *Enemy {
  return &Enemy{
    Paddle: Paddle{
      width:  w,
      height: h,
      color:  0xff6666,
      speed:  float64(*enemySpeed),
      target: &Vector2{X: 0, Y: 0},
      vector: &Vector2{X: x, Y: y},
    },
  }
}

func (enemy *Enemy) Hit(past, future *Vector2) (hit bool, place *Vector2) {
  // our front line
  halfHeight := enemy.height / 2
  halfWidth := enemy.width / 2
  x0, y0 := (enemy.vector.X - halfWidth), (enemy.vector.Y - halfHeight)
  x1, y1 := (enemy.vector.X - halfWidth), (enemy.vector.Y + halfHeight)

  return enemy.hitCore(x0, y0, x1, y1, past, future)
}

func (paddle *Paddle) hitCore(x0, y0, x1, y1 float64, past, future *Vector2) (hit bool, place *Vector2) {
  // line between past and future
  x2, y2 := past.X, past.Y
  x3, y3 := future.X, future.Y
  d := (x1-x0)*(y3-y2) - (y1-y0)*(x3-x2)

  if math.Abs(d) < 0.001 {
    return
  } // never hit since parallel

  ab := ((y0-y2)*(x3-x2) - (x0-x2)*(y3-y2)) / d

  if ab > 0.0 && ab < 1.0 {
    cd := ((y0-y2)*(x1-x0) - (x0-x2)*(y1-y0)) / d
    if cd > 0.0 && cd < 1.0 {
      linx := x0 + ab*(x1-x0)
      liny := y0 + ab*(y1-y0)
      hit = true
      place = &Vector2{X: linx, Y: liny}
    }
  }

  // no hit
  return
}

func (enemy *Enemy) Update(world *World) {
  if world.Ball.velocity.X > 0 {
    targetY := world.Ball.vector.Y
    targetX := enemy.vector.X
    goal := (&Vector2{X: targetX, Y: targetY}).Minus(enemy.vector)

    if goal.Length() > enemy.speed {
      goal = goal.Normalize().MultiplyNum(enemy.speed)
    }

    enemy.target = goal
  }

  future := enemy.vector.Plus(enemy.target)
  if future.Y < (enemy.height / 2) {
    return
  }
  if (future.Y + (enemy.height / 2)) > float64(world.Height) {
    return
  }
  enemy.vector = future
}

type World struct {
  running       bool
  pause         bool
  Height, Width int
  Screen        *sdl.Surface
  Ball          *Ball
  Paddle        *Paddle
  Enemy         *Enemy
  Score         *Score
}

func NewWorld(height, width int) *World {
  return &World{
    Height:  height,
    Width:   width,
    Screen:  NewSurface(width, height),
    Ball:    NewBall(float64(width)/2, float64(height)/2),
    Paddle:  NewPaddle(5, float64(height)/2, 5, 30),
    Enemy:   NewEnemy(float64(width-5), float64(height)/2, 5, 30),
    Score:   NewScore(),
    running: true,
    pause:   false,
  }
}

func (world *World) HandleEvents() {
  for world.running {
    for ev := sdl.PollEvent(); ev != nil; ev = sdl.PollEvent() {
      switch e := ev.(type) {
      case *sdl.QuitEvent:
        world.running = false
      case *sdl.KeyboardEvent:
        switch sdl.GetKeyName(sdl.Key(e.Keysym.Sym)) {
        case "p":
          world.pause = !world.pause
        case "j":
          world.Paddle.Go(0, world.Paddle.vector.Y+world.Paddle.speed)
        case "k":
          world.Paddle.Go(0, world.Paddle.vector.Y-world.Paddle.speed)
        case "q":
          world.running = false
        }
      case *sdl.MouseMotionEvent:
        world.Paddle.Go(float64(e.X), float64(e.Y))
      }
    }

    sdl.Delay(25)
  }
}

func (world *World) Run() {
  for world.running {
    if !world.pause {
      world.Update()
      world.Draw()
    }
    sdl.Delay(25)
  }
}

func (world *World) Update() {
  world.Ball.Update(world)
  world.Paddle.Update(world)
  world.Enemy.Update(world)
}

func (world *World) Draw() {
  world.Screen.FillRect(nil, 0x0)

  center := &sdl.Rect{X: int16(world.Width/2) - 1, Y: 0, H: uint16(world.Height * 2), W: 2}
  world.Screen.FillRect(center, 0x333333)

  world.Paddle.Draw(world)
  world.Enemy.Draw(world)
  world.Ball.Draw(world)
  world.Score.Draw(world)

  world.Screen.Flip()
}

type Score struct {
  Enemy  int
  Paddle int
  color  *sdl.Color
}

func NewScore() (score *Score) {
  score = &Score{
    color:  &sdl.Color{255, 255, 255, 0},
    Enemy:  0,
    Paddle: 0,
  }
  return score
}

func (score *Score) Draw(world *World) {
  pRect := &sdl.Rect{
    X: int16(world.Paddle.width + world.Paddle.vector.X),
    Y: 3, W: 3, H: 3,
  }

  for p := score.Paddle; p > 0; p-- {
    pRect.X += 6
    world.Screen.FillRect(pRect, 0x6666ff)
  }

  if int(pRect.X) > world.Width {
    fmt.Println("You Win!")
    world.running = false
  }

  eRect := &sdl.Rect{
    X: int16(world.Enemy.vector.X - world.Enemy.width),
    Y: int16(world.Height - 6), W: 3, H: 3,
  }

  for e := score.Enemy; e >= 0; e-- {
    eRect.X -= 6
    world.Screen.FillRect(eRect, 0xff6666)
  }

  if eRect.X <= 0 {
    fmt.Println("You Lose!")
    world.running = false
  }
}

type Vector2 struct {
  X, Y float64
}

func (v *Vector2) Normalize() *Vector2 {
  length := v.Length()
  return &Vector2{X: (v.X / length), Y: (v.Y / length)}
}

func (v *Vector2) MultiplyNum(other float64) *Vector2 {
  return &Vector2{X: (v.X * other), Y: (v.Y * other)}
}

func (v *Vector2) Plus(other *Vector2) *Vector2 {
  return &Vector2{X: (v.X + other.X), Y: (v.Y + other.Y)}
}

func (v *Vector2) Minus(other *Vector2) *Vector2 {
  return &Vector2{X: (v.X - other.X), Y: (v.Y - other.Y)}
}

func (v *Vector2) Length() float64 {
  return math.Sqrt((v.X * v.X) + (v.Y * v.Y))
}

func NewSurface(height int, width int) (surface *sdl.Surface) {
  surface = sdl.SetVideoMode(height, width, 32, 0)
  if surface == nil {
    panic(sdl.GetError())
  }
  return
}

func sdlSetup() (world *World) {
  if sdl.Init(sdl.INIT_EVERYTHING) != 0 {
    panic(sdl.GetError())
  }
  sdl.EnableUNICODE(1)
  sdl.EnableKeyRepeat(25, 25)

  return
}

func Quit(world *World) {
  sdl.Quit()
}
