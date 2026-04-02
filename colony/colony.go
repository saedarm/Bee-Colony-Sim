package colony

import (
	"math"
	"math/rand"

	"github.com/saedarm/bee-colony/terrain"
)

// BeeRole distinguishes the three types of bees in ABC
type BeeRole int

const (
	Employed BeeRole = iota
	Onlooker
	Scout
)

// Bee represents a single bee agent in the colony
type Bee struct {
	Role     BeeRole
	X, Z     float64 // world position
	TargetX  float64 // where this bee is heading
	TargetZ  float64
	Fitness  float64 // fitness of current food source
	Trials   int     // number of failed improvement attempts
	Carrying bool    // true if bee has picked up food
	CarryX   float64 // where it picked up from
	CarryZ   float64

	// Animation state
	Progress float32 // 0..1 interpolation toward target
	WingPhase float32
	Wobble    float32
}

// FoodSource represents a food source on the terrain
type FoodSource struct {
	X, Z       float64
	Fitness    float64
	Nectar     float64 // remaining nectar 0..1
	MaxNectar  float64
	Trials     int
	ShapeType  int     // 0=cube, 1=tetra, 2=octa, 3=icosa, 4=dodeca
	Active     bool
	PulsePhase float32 // for visual pulsing
}

// Colony is the full ABC simulation state
type Colony struct {
	Bees       []*Bee
	Foods      []*FoodSource
	Terrain    *terrain.Terrain
	BestFitness float64
	BestX      float64
	BestZ      float64
	Generation int

	// Parameters
	NumEmployed   int
	NumOnlookers  int
	AbandonLimit  int
	WorldHalfSize float64

	// Stats
	FoodsExhausted int
	FoodsDiscovered int

	// Hive location (center)
	HiveX, HiveZ float64

	rng *rand.Rand
}

// Config holds the setup parameters for a simulation run
type Config struct {
	NumBees      int
	NumFoods     int
	AbandonLimit int
	TerrainType  terrain.Type
	Seed         int64
}

// Presets returns a named set of configurations
func Presets() map[string]Config {
	return map[string]Config{
		"Balanced": {
			NumBees: 30, NumFoods: 10, AbandonLimit: 20,
			TerrainType: terrain.Rastrigin, Seed: 42,
		},
		"Plenty": {
			NumBees: 15, NumFoods: 25, AbandonLimit: 30,
			TerrainType: terrain.PerlinNoise, Seed: 77,
		},
		"Famine": {
			NumBees: 40, NumFoods: 5, AbandonLimit: 10,
			TerrainType: terrain.Ackley, Seed: 13,
		},
		"Needle": {
			NumBees: 25, NumFoods: 15, AbandonLimit: 25,
			TerrainType: terrain.Rastrigin, Seed: 99,
		},
		"Swarm": {
			NumBees: 60, NumFoods: 12, AbandonLimit: 15,
			TerrainType: terrain.RandomPeaks, Seed: 256,
		},
	}
}

// NewColony creates and initializes a colony from config
func NewColony(cfg Config, t *terrain.Terrain) *Colony {
	rng := rand.New(rand.NewSource(cfg.Seed))
	halfSize := float64(t.ScaleXZ) / 2.0 * 0.9 // stay slightly inbounds

	c := &Colony{
		Terrain:       t,
		NumEmployed:   cfg.NumBees / 2,
		NumOnlookers:  cfg.NumBees / 2,
		AbandonLimit:  cfg.AbandonLimit,
		WorldHalfSize: halfSize,
		HiveX:         0,
		HiveZ:         0,
		rng:           rng,
	}

	// Initialize food sources
	c.Foods = make([]*FoodSource, cfg.NumFoods)
	for i := 0; i < cfg.NumFoods; i++ {
		c.Foods[i] = c.randomFood()
		c.Foods[i].ShapeType = i % 5
	}

	// Initialize bees
	totalBees := c.NumEmployed + c.NumOnlookers
	c.Bees = make([]*Bee, totalBees)

	// Employed bees: one per food source (cycle if more bees than food)
	for i := 0; i < c.NumEmployed; i++ {
		foodIdx := i % len(c.Foods)
		food := c.Foods[foodIdx]
		c.Bees[i] = &Bee{
			Role:     Employed,
			X:        c.HiveX,
			Z:        c.HiveZ,
			TargetX:  food.X,
			TargetZ:  food.Z,
			Fitness:  food.Fitness,
			Progress: 0,
			WingPhase: rng.Float32() * math.Pi * 2,
			Wobble:   rng.Float32() * math.Pi * 2,
		}
	}

	// Onlooker bees: start at hive, waiting
	for i := c.NumEmployed; i < totalBees; i++ {
		c.Bees[i] = &Bee{
			Role:     Onlooker,
			X:        c.HiveX,
			Z:        c.HiveZ,
			TargetX:  c.HiveX,
			TargetZ:  c.HiveZ,
			Progress: 0,
			WingPhase: rng.Float32() * math.Pi * 2,
			Wobble:   rng.Float32() * math.Pi * 2,
		}
	}

	return c
}

// Step advances the ABC algorithm by one generation
func (c *Colony) Step() {
	c.Generation++

	// === PHASE 1: Employed Bees ===
	for i := 0; i < c.NumEmployed; i++ {
		bee := c.Bees[i]
		bee.Role = Employed

		// Pick a food source for this employed bee
		foodIdx := i % len(c.Foods)
		food := c.Foods[foodIdx]

		if !food.Active {
			continue
		}

		// Generate neighbor solution: v_ij = x_ij + phi * (x_ij - x_kj)
		k := c.rng.Intn(len(c.Foods))
		for k == foodIdx {
			k = c.rng.Intn(len(c.Foods))
		}
		neighbor := c.Foods[k]

		phi := c.rng.Float64()*2 - 1 // [-1, 1]
		newX := food.X + phi*(food.X-neighbor.X)
		newZ := food.Z + phi*(food.Z-neighbor.Z)

		// Clamp to world bounds
		newX = clamp(newX, -c.WorldHalfSize, c.WorldHalfSize)
		newZ = clamp(newZ, -c.WorldHalfSize, c.WorldHalfSize)

		newFitness := c.Terrain.Fitness(newX, newZ)

		// Greedy selection
		if newFitness > food.Fitness {
			food.X = newX
			food.Z = newZ
			food.Fitness = newFitness
			food.Trials = 0

			// Deplete nectar slightly on successful exploitation
			food.Nectar -= 0.02
			if food.Nectar < 0 {
				food.Nectar = 0
			}
		} else {
			food.Trials++
		}

		// Direct bee toward food
		bee.TargetX = food.X
		bee.TargetZ = food.Z
		bee.Fitness = food.Fitness
		bee.Carrying = true
		bee.CarryX = food.X
		bee.CarryZ = food.Z
	}

	// === PHASE 2: Onlooker Bees (roulette wheel selection) ===
	// Calculate selection probabilities
	totalFitness := 0.0
	for _, f := range c.Foods {
		if f.Active {
			totalFitness += f.Fitness
		}
	}

	for i := c.NumEmployed; i < len(c.Bees); i++ {
		bee := c.Bees[i]
		bee.Role = Onlooker

		if totalFitness <= 0 {
			continue
		}

		// Roulette wheel selection
		r := c.rng.Float64() * totalFitness
		cumulative := 0.0
		selectedIdx := 0
		for j, f := range c.Foods {
			if !f.Active {
				continue
			}
			cumulative += f.Fitness
			if cumulative >= r {
				selectedIdx = j
				break
			}
		}

		food := c.Foods[selectedIdx]
		if !food.Active {
			continue
		}

		// Generate neighbor solution (same as employed phase)
		k := c.rng.Intn(len(c.Foods))
		for k == selectedIdx && len(c.Foods) > 1 {
			k = c.rng.Intn(len(c.Foods))
		}
		neighbor := c.Foods[k]

		phi := c.rng.Float64()*2 - 1
		newX := food.X + phi*(food.X-neighbor.X)
		newZ := food.Z + phi*(food.Z-neighbor.Z)
		newX = clamp(newX, -c.WorldHalfSize, c.WorldHalfSize)
		newZ = clamp(newZ, -c.WorldHalfSize, c.WorldHalfSize)

		newFitness := c.Terrain.Fitness(newX, newZ)

		if newFitness > food.Fitness {
			food.X = newX
			food.Z = newZ
			food.Fitness = newFitness
			food.Trials = 0
			food.Nectar -= 0.01
			if food.Nectar < 0 {
				food.Nectar = 0
			}
		} else {
			food.Trials++
		}

		bee.TargetX = food.X
		bee.TargetZ = food.Z
		bee.Fitness = food.Fitness
	}

	// === PHASE 3: Scout Bees ===
	for i, food := range c.Foods {
		if food.Active && (food.Trials >= c.AbandonLimit || food.Nectar <= 0) {
			// Abandon this food source
			food.Active = false
			c.FoodsExhausted++

			// Replace with new random food source
			newFood := c.randomFood()
			newFood.ShapeType = food.ShapeType
			c.Foods[i] = newFood
			c.FoodsDiscovered++

			// Turn an employed bee into a scout briefly
			beeIdx := i % c.NumEmployed
			if beeIdx < len(c.Bees) {
				bee := c.Bees[beeIdx]
				bee.Role = Scout
				bee.TargetX = newFood.X
				bee.TargetZ = newFood.Z
				bee.Carrying = false
			}
		}
	}

	// === Update best solution ===
	for _, f := range c.Foods {
		if f.Active && f.Fitness > c.BestFitness {
			c.BestFitness = f.Fitness
			c.BestX = f.X
			c.BestZ = f.Z
		}
	}
}

// UpdateAnimation advances bee positions smoothly between ticks
func (c *Colony) UpdateAnimation(dt float32) {
	for _, bee := range c.Bees {
		// Advance wing flapping
		bee.WingPhase += dt * 15.0
		bee.Wobble += dt * 3.0

		// Move bee toward target
		speed := float32(0.8)
		if bee.Role == Scout {
			speed = 1.5 // scouts zip around faster
		}

		bee.Progress += dt * speed
		if bee.Progress > 1.0 {
			bee.Progress = 1.0
		}

		// Interpolate position
		bee.X = lerp(bee.X, float64(bee.TargetX), float64(dt*speed))
		bee.Z = lerp(bee.Z, float64(bee.TargetZ), float64(dt*speed))
	}
}

// randomFood creates a new random food source
func (c *Colony) randomFood() *FoodSource {
	x := (c.rng.Float64()*2 - 1) * c.WorldHalfSize
	z := (c.rng.Float64()*2 - 1) * c.WorldHalfSize
	fitness := c.Terrain.Fitness(x, z)

	return &FoodSource{
		X:         x,
		Z:         z,
		Fitness:   fitness,
		Nectar:    1.0,
		MaxNectar: 1.0,
		Trials:    0,
		Active:    true,
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}
