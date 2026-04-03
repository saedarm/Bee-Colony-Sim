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
	Role      BeeRole
	X, Z      float64
	TargetX   float64
	TargetZ   float64
	Fitness   float64
	Trials    int
	Carrying  bool
	CarryX    float64
	CarryZ    float64
	Progress  float32
	WingPhase float32
	Wobble    float32
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
		"Balanced": {NumBees: 30, NumFoods: 10, AbandonLimit: 20, TerrainType: terrain.Rastrigin, Seed: 42},
		"Plenty":   {NumBees: 15, NumFoods: 25, AbandonLimit: 30, TerrainType: terrain.PerlinNoise, Seed: 77},
		"Famine":   {NumBees: 40, NumFoods: 5, AbandonLimit: 10, TerrainType: terrain.Ackley, Seed: 13},
		"Needle":   {NumBees: 25, NumFoods: 15, AbandonLimit: 25, TerrainType: terrain.Rastrigin, Seed: 99},
		"Swarm":    {NumBees: 60, NumFoods: 12, AbandonLimit: 15, TerrainType: terrain.RandomPeaks, Seed: 256},
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

	c.Foods = make([]*FoodSource, cfg.NumFoods)
	for i := 0; i < cfg.NumFoods; i++ {
		c.Foods[i] = c.randomFood()
		c.Foods[i].ShapeType = i % 5
	}

	totalBees := c.NumEmployed + c.NumOnlookers
	c.Bees = make([]*Bee, totalBees)

	for i := 0; i < c.NumEmployed; i++ {
		food := c.Foods[i%len(c.Foods)]
		c.Bees[i] = &Bee{
			Role: Employed, X: rng.Float64()*4 - 2, Z: rng.Float64()*4 - 2,
			TargetX: food.X, TargetZ: food.Z, Fitness: food.Fitness,
			WingPhase: rng.Float32() * math.Pi * 2, Wobble: rng.Float32() * math.Pi * 2,
		}
	}
	for i := c.NumEmployed; i < totalBees; i++ {
		c.Bees[i] = &Bee{
			Role: Onlooker, X: rng.Float64()*4 - 2, Z: rng.Float64()*4 - 2,
			WingPhase: rng.Float32() * math.Pi * 2, Wobble: rng.Float32() * math.Pi * 2,
		}
	}
	return c
}

func (c *Colony) Step() {
	c.Generation++

	for i := 0; i < c.NumEmployed; i++ {
		bee := c.Bees[i]
		bee.Role = Employed
		foodIdx := i % len(c.Foods)
		food := c.Foods[foodIdx]
		if !food.Active {
			continue
		}
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
			food.Nectar -= 0.015
			if food.Nectar < 0 {
				food.Nectar = 0
			}
		} else {
			food.Trials++
		}
		bee.TargetX, bee.TargetZ, bee.Fitness = food.X, food.Z, food.Fitness
		bee.Carrying, bee.CarryX, bee.CarryZ = true, food.X, food.Z
	}

	totalFit := 0.0
	for _, f := range c.Foods {
		if f.Active {
			totalFit += f.Fitness
		}
	}
	for i := c.NumEmployed; i < len(c.Bees); i++ {
		bee := c.Bees[i]
		bee.Role = Onlooker
		if totalFit <= 0 {
			continue
		}
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
			food.Nectar -= 0.01
			if food.Nectar < 0 {
				food.Nectar = 0
			}
		} else {
			food.Trials++
		}
		bee.TargetX, bee.TargetZ, bee.Fitness = food.X, food.Z, food.Fitness
	}

	for i, food := range c.Foods {
		if food.Active && (food.Trials >= c.AbandonLimit || food.Nectar <= 0) {
			food.Active = false
			c.FoodsExhausted++
			nf := c.randomFood()
			nf.ShapeType = food.ShapeType
			c.Foods[i] = nf
			c.FoodsDiscovered++
			beeIdx := i % c.NumEmployed
			if beeIdx < len(c.Bees) {
				b := c.Bees[beeIdx]
				b.Role = Scout
				b.TargetX, b.TargetZ, b.Carrying = nf.X, nf.Z, false
			}
		}
	}

	for _, f := range c.Foods {
		if f.Active && f.Fitness > c.BestFitness {
			c.BestFitness, c.BestX, c.BestZ = f.Fitness, f.X, f.Z
		}
	}
}

func (c *Colony) UpdateAnimation(dt float32) {
	for _, bee := range c.Bees {
		bee.WingPhase += dt * 18.0
		bee.Wobble += dt * 3.5
		speed := float64(dt) * 1.2
		if bee.Role == Scout {
			speed = float64(dt) * 2.5
		}
		bee.X += (bee.TargetX - bee.X) * speed
		bee.Z += (bee.TargetZ - bee.Z) * speed
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

func clamp(v, mn, mx float64) float64 {
	if v < mn {
		return mn
	}
	if v > mx {
		return mx
	}
	return v
}
