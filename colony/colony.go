package colony

import (
	"math"
	"math/rand"

	"github.com/saedarm/bee-colony/terrain"
)

type BeeRole int

const (
	Employed BeeRole = iota
	Onlooker
	Scout
)

type Bee struct {
	Role         BeeRole
	X, Z         float64
	TargetX      float64
	TargetZ      float64
	Fitness      float64
	FoodIdx      int     // which food source this bee is working
	Carrying     bool
	WingPhase    float32
	Wobble       float32
	ScoutLaunch  bool    // true on the frame this bee became a scout
	JustArrived  bool    // true when bee reaches its target
}

type FoodSource struct {
	X, Z       float64
	Fitness    float64
	Nectar     float64
	MaxNectar  float64
	Trials     int
	ShapeType  int
	Active     bool
	PulsePhase float32
	JustFound  bool    // true on the frame this source was discovered
	BeingWorked bool   // true if bees visited this tick
	OnlookerCount int  // how many onlookers chose this source this tick
}

type Colony struct {
	Bees            []*Bee
	Foods           []*FoodSource
	Terrain         *terrain.Terrain
	BestFitness     float64
	BestX, BestZ    float64
	Generation      int
	NumEmployed     int
	NumOnlookers    int
	AbandonLimit    int
	WorldHalfSize   float64
	FoodsExhausted  int
	FoodsDiscovered int
	HiveX, HiveZ    float64
	rng             *rand.Rand

	// Events for renderer
	ScoutEvents    []ScoutEvent
	ExhaustEvents  []ExhaustEvent
}

type ScoutEvent struct {
	FromX, FromZ float64
	ToX, ToZ     float64
	Age          float32 // seconds since event
}

type ExhaustEvent struct {
	X, Z float64
	Age  float32
}

type Config struct {
	NumBees      int
	NumFoods     int
	AbandonLimit int
	TerrainType  terrain.Type
	Seed         int64
}

func Presets() map[string]Config {
	return map[string]Config{
		"Balanced": {NumBees: 30, NumFoods: 10, AbandonLimit: 40, TerrainType: terrain.Rastrigin, Seed: 42},
		"Plenty":   {NumBees: 15, NumFoods: 20, AbandonLimit: 60, TerrainType: terrain.PerlinNoise, Seed: 77},
		"Famine":   {NumBees: 40, NumFoods: 5, AbandonLimit: 25, TerrainType: terrain.Ackley, Seed: 13},
		"Needle":   {NumBees: 25, NumFoods: 15, AbandonLimit: 50, TerrainType: terrain.Rastrigin, Seed: 99},
		"Swarm":    {NumBees: 60, NumFoods: 12, AbandonLimit: 35, TerrainType: terrain.RandomPeaks, Seed: 256},
	}
}

func NewColony(cfg Config, t *terrain.Terrain) *Colony {
	rng := rand.New(rand.NewSource(cfg.Seed))
	halfSize := float64(t.ScaleXZ) / 2.0 * 0.85

	c := &Colony{
		Terrain:       t,
		NumEmployed:   cfg.NumBees / 2,
		NumOnlookers:  cfg.NumBees / 2,
		AbandonLimit:  cfg.AbandonLimit,
		WorldHalfSize: halfSize,
		rng:           rng,
	}

	// Initialize food with minimum spacing
	c.Foods = make([]*FoodSource, 0, cfg.NumFoods)
	minDist := halfSize * 0.3 // minimum distance between sources
	for i := 0; i < cfg.NumFoods; i++ {
		var food *FoodSource
		for attempts := 0; attempts < 50; attempts++ {
			candidate := c.randomFood()
			if c.isFarEnough(candidate.X, candidate.Z, minDist) {
				food = candidate
				break
			}
		}
		if food == nil {
			food = c.randomFood() // fallback
		}
		food.ShapeType = i % 5
		c.Foods = append(c.Foods, food)
	}

	totalBees := c.NumEmployed + c.NumOnlookers
	c.Bees = make([]*Bee, totalBees)

	for i := 0; i < c.NumEmployed; i++ {
		foodIdx := i % len(c.Foods)
		food := c.Foods[foodIdx]
		c.Bees[i] = &Bee{
			Role: Employed, X: rng.Float64()*6 - 3, Z: rng.Float64()*6 - 3,
			TargetX: food.X, TargetZ: food.Z, Fitness: food.Fitness,
			FoodIdx:   foodIdx,
			WingPhase: rng.Float32() * math.Pi * 2, Wobble: rng.Float32() * math.Pi * 2,
		}
	}
	for i := c.NumEmployed; i < totalBees; i++ {
		c.Bees[i] = &Bee{
			Role: Onlooker, X: rng.Float64()*6 - 3, Z: rng.Float64()*6 - 3,
			FoodIdx:   -1,
			WingPhase: rng.Float32() * math.Pi * 2, Wobble: rng.Float32() * math.Pi * 2,
		}
	}
	return c
}

func (c *Colony) isFarEnough(x, z, minDist float64) bool {
	for _, f := range c.Foods {
		dx := f.X - x
		dz := f.Z - z
		if math.Sqrt(dx*dx+dz*dz) < minDist {
			return false
		}
	}
	return true
}

func (c *Colony) Step() {
	c.Generation++

	// Reset per-tick state
	for _, f := range c.Foods {
		f.BeingWorked = false
		f.OnlookerCount = 0
		f.JustFound = false
	}
	for _, b := range c.Bees {
		b.ScoutLaunch = false
	}

	// === PHASE 1: Employed Bees ===
	for i := 0; i < c.NumEmployed; i++ {
		bee := c.Bees[i]
		bee.Role = Employed
		foodIdx := i % len(c.Foods)
		bee.FoodIdx = foodIdx
		food := c.Foods[foodIdx]
		if !food.Active {
			continue
		}
		food.BeingWorked = true

		k := c.rng.Intn(len(c.Foods))
		for k == foodIdx {
			k = c.rng.Intn(len(c.Foods))
		}
		nb := c.Foods[k]
		phi := c.rng.Float64()*2 - 1
		newX := clamp(food.X+phi*(food.X-nb.X), -c.WorldHalfSize, c.WorldHalfSize)
		newZ := clamp(food.Z+phi*(food.Z-nb.Z), -c.WorldHalfSize, c.WorldHalfSize)
		nf := c.Terrain.Fitness(newX, newZ)

		if nf > food.Fitness {
			food.X, food.Z, food.Fitness, food.Trials = newX, newZ, nf, 0
			food.Nectar -= 0.005 // MUCH slower depletion
			if food.Nectar < 0 {
				food.Nectar = 0
			}
		} else {
			food.Trials++
		}
		bee.TargetX, bee.TargetZ, bee.Fitness = food.X, food.Z, food.Fitness
		bee.Carrying = true
	}

	// === PHASE 2: Onlooker Bees (roulette wheel) ===
	totalFit := 0.0
	for _, f := range c.Foods {
		if f.Active {
			totalFit += f.Fitness
		}
	}
	for i := c.NumEmployed; i < len(c.Bees); i++ {
		bee := c.Bees[i]
		bee.Role = Onlooker
		bee.FoodIdx = -1
		if totalFit <= 0 {
			continue
		}

		// Roulette wheel selection
		r := c.rng.Float64() * totalFit
		cum := 0.0
		sel := 0
		for j, f := range c.Foods {
			if !f.Active {
				continue
			}
			cum += f.Fitness
			if cum >= r {
				sel = j
				break
			}
		}
		food := c.Foods[sel]
		if !food.Active {
			continue
		}
		bee.FoodIdx = sel
		food.OnlookerCount++
		food.BeingWorked = true

		k := c.rng.Intn(len(c.Foods))
		for k == sel && len(c.Foods) > 1 {
			k = c.rng.Intn(len(c.Foods))
		}
		nb := c.Foods[k]
		phi := c.rng.Float64()*2 - 1
		newX := clamp(food.X+phi*(food.X-nb.X), -c.WorldHalfSize, c.WorldHalfSize)
		newZ := clamp(food.Z+phi*(food.Z-nb.Z), -c.WorldHalfSize, c.WorldHalfSize)
		nf := c.Terrain.Fitness(newX, newZ)

		if nf > food.Fitness {
			food.X, food.Z, food.Fitness, food.Trials = newX, newZ, nf, 0
			food.Nectar -= 0.003
			if food.Nectar < 0 {
				food.Nectar = 0
			}
		} else {
			food.Trials++
		}
		bee.TargetX, bee.TargetZ, bee.Fitness = food.X, food.Z, food.Fitness
	}

	// === PHASE 3: Scout Bees ===
	for i, food := range c.Foods {
		if food.Active && (food.Trials >= c.AbandonLimit || food.Nectar <= 0) {
			oldX, oldZ := food.X, food.Z
			food.Active = false
			c.FoodsExhausted++

			// Record exhaust event
			c.ExhaustEvents = append(c.ExhaustEvents, ExhaustEvent{X: oldX, Z: oldZ})

			nf := c.randomFood()
			nf.ShapeType = food.ShapeType
			nf.JustFound = true
			c.Foods[i] = nf
			c.FoodsDiscovered++

			// Record scout event
			c.ScoutEvents = append(c.ScoutEvents, ScoutEvent{
				FromX: oldX, FromZ: oldZ, ToX: nf.X, ToZ: nf.Z,
			})

			beeIdx := i % c.NumEmployed
			if beeIdx < len(c.Bees) {
				b := c.Bees[beeIdx]
				b.Role = Scout
				b.ScoutLaunch = true
				b.TargetX, b.TargetZ = nf.X, nf.Z
				b.FoodIdx = i
				b.Carrying = false
			}
		}
	}

	// Update best
	for _, f := range c.Foods {
		if f.Active && f.Fitness > c.BestFitness {
			c.BestFitness, c.BestX, c.BestZ = f.Fitness, f.X, f.Z
		}
	}
}

// UpdateEvents ages and prunes transient events
func (c *Colony) UpdateEvents(dt float32) {
	// Age scout events
	alive := c.ScoutEvents[:0]
	for i := range c.ScoutEvents {
		c.ScoutEvents[i].Age += dt
		if c.ScoutEvents[i].Age < 3.0 {
			alive = append(alive, c.ScoutEvents[i])
		}
	}
	c.ScoutEvents = alive

	// Age exhaust events
	alive2 := c.ExhaustEvents[:0]
	for i := range c.ExhaustEvents {
		c.ExhaustEvents[i].Age += dt
		if c.ExhaustEvents[i].Age < 2.0 {
			alive2 = append(alive2, c.ExhaustEvents[i])
		}
	}
	c.ExhaustEvents = alive2
}

func (c *Colony) UpdateAnimation(dt float32) {
	for _, bee := range c.Bees {
		bee.WingPhase += dt * 18.0
		bee.Wobble += dt * 3.5
		speed := float64(dt) * 1.0
		if bee.Role == Scout {
			speed = float64(dt) * 2.0
		}
		prevX, prevZ := bee.X, bee.Z
		bee.X += (bee.TargetX - bee.X) * speed
		bee.Z += (bee.TargetZ - bee.Z) * speed

		// Check if just arrived
		dx := bee.TargetX - bee.X
		dz := bee.TargetZ - bee.Z
		dist := math.Sqrt(dx*dx + dz*dz)
		prevDx := bee.TargetX - prevX
		prevDz := bee.TargetZ - prevZ
		prevDist := math.Sqrt(prevDx*prevDx + prevDz*prevDz)
		bee.JustArrived = dist < 1.0 && prevDist >= 1.0
	}
}

func (c *Colony) randomFood() *FoodSource {
	x := (c.rng.Float64()*2 - 1) * c.WorldHalfSize
	z := (c.rng.Float64()*2 - 1) * c.WorldHalfSize
	return &FoodSource{
		X: x, Z: z, Fitness: c.Terrain.Fitness(x, z),
		Nectar: 1.0, MaxNectar: 1.0, Active: true,
	}
}

// AbandonRatio returns how close a food source is to being abandoned (0..1)
func AbandonRatio(food *FoodSource, limit int) float32 {
	if limit <= 0 {
		return 0
	}
	r := float32(food.Trials) / float32(limit)
	if r > 1 {
		return 1
	}
	return r
}

func clamp(v, mn, mx float64) float64 {
	if v < mn { return mn }
	if v > mx { return mx }
	return v
}
