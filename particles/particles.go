package particles

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ParticleType determines visual behavior
type ParticleType int

const (
	Sparkle  ParticleType = iota // nectar collection sparkles
	Pollen                       // drifting pollen near food
	Burst                        // explosion when food exhausted
	Trail                        // bee flight trail
)

const MaxParticles = 1000

// Particle is a single visual particle
type Particle struct {
	Active   bool
	Type     ParticleType
	X, Y, Z  float32
	VX, VY, VZ float32
	Life     float32 // remaining life 0..1
	MaxLife  float32
	Size     float32
	Color    rl.Color
	Gravity  float32
}

// System manages particle pool and textures
type System struct {
	Pool     [MaxParticles]Particle
	Textures map[ParticleType]rl.Texture2D
	rng      *rand.Rand
}

// NewSystem creates the particle system and generates procedural textures
func NewSystem() *System {
	s := &System{
		Textures: make(map[ParticleType]rl.Texture2D),
		rng:      rand.New(rand.NewSource(12345)),
	}

	// Generate soft glow circle texture (for sparkles and trails)
	s.Textures[Sparkle] = generateGlowTexture(32, rl.NewColor(255, 255, 255, 255))
	s.Textures[Trail] = generateGlowTexture(16, rl.NewColor(255, 255, 255, 255))

	// Fuzzy pollen texture
	s.Textures[Pollen] = generateFuzzyTexture(24, rl.NewColor(255, 230, 100, 255))

	// Star burst
	s.Textures[Burst] = generateStarTexture(32, rl.NewColor(255, 255, 255, 255))

	return s
}

// Unload frees GPU textures
func (s *System) Unload() {
	for _, tex := range s.Textures {
		rl.UnloadTexture(tex)
	}
}

// generateGlowTexture creates a soft radial falloff circle
func generateGlowTexture(size int, tint rl.Color) rl.Texture2D {
	img := rl.GenImageColor(size, size, rl.NewColor(0, 0, 0, 0))
	center := float32(size) / 2.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float32(x) - center + 0.5
			dy := float32(y) - center + 0.5
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			norm := dist / center
			if norm > 1 {
				norm = 1
			}
			// Smooth falloff
			alpha := uint8(float32(tint.A) * (1 - norm*norm))
			rl.ImageDrawPixel(img, int32(x), int32(y), rl.NewColor(tint.R, tint.G, tint.B, alpha))
		}
	}
	tex := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	return tex
}

// generateFuzzyTexture creates a noisy soft circle
func generateFuzzyTexture(size int, tint rl.Color) rl.Texture2D {
	img := rl.GenImageColor(size, size, rl.NewColor(0, 0, 0, 0))
	center := float32(size) / 2.0
	rng := rand.New(rand.NewSource(777))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float32(x) - center + 0.5
			dy := float32(y) - center + 0.5
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			norm := dist / center
			if norm > 1 {
				continue
			}
			noise := float32(rng.Float64()*0.4 + 0.6)
			alpha := uint8(float32(tint.A) * (1 - norm) * noise)
			rl.ImageDrawPixel(img, int32(x), int32(y), rl.NewColor(tint.R, tint.G, tint.B, alpha))
		}
	}
	tex := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	return tex
}

// generateStarTexture creates a 4-pointed star shape
func generateStarTexture(size int, tint rl.Color) rl.Texture2D {
	img := rl.GenImageColor(size, size, rl.NewColor(0, 0, 0, 0))
	center := float32(size) / 2.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float32(x) - center + 0.5
			dy := float32(y) - center + 0.5
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			norm := dist / center
			if norm > 1 {
				continue
			}

			// Star shape: brighter along axes
			ax := float32(math.Abs(float64(dx)))
			ay := float32(math.Abs(float64(dy)))
			minAx := ax
			if ay < minAx {
				minAx = ay
			}
			starFactor := float32(1.0) - minAx/center
			if starFactor < 0 {
				starFactor = 0
			}

			alpha := uint8(float32(tint.A) * (1 - norm) * (0.3 + 0.7*starFactor))
			rl.ImageDrawPixel(img, int32(x), int32(y), rl.NewColor(tint.R, tint.G, tint.B, alpha))
		}
	}
	tex := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	return tex
}

// Emit spawns particles at a location
func (s *System) Emit(ptype ParticleType, x, y, z float32, count int, color rl.Color) {
	for i := 0; i < count; i++ {
		p := s.findFree()
		if p == nil {
			return // pool exhausted
		}

		p.Active = true
		p.Type = ptype
		p.X = x
		p.Y = y
		p.Z = z

		switch ptype {
		case Sparkle:
			spread := float32(0.8)
			p.VX = (s.rng.Float32()*2 - 1) * spread
			p.VY = s.rng.Float32() * 2.0
			p.VZ = (s.rng.Float32()*2 - 1) * spread
			p.MaxLife = 0.6 + s.rng.Float32()*0.4
			p.Size = 0.3 + s.rng.Float32()*0.3
			p.Gravity = -0.5
			p.Color = color

		case Pollen:
			p.VX = (s.rng.Float32()*2 - 1) * 0.3
			p.VY = s.rng.Float32() * 0.5
			p.VZ = (s.rng.Float32()*2 - 1) * 0.3
			p.MaxLife = 1.5 + s.rng.Float32()*1.0
			p.Size = 0.2 + s.rng.Float32()*0.2
			p.Gravity = 0.1 // slowly rises
			p.Color = rl.NewColor(255, 230, 100, 200)

		case Burst:
			angle := s.rng.Float32() * math.Pi * 2
			speed := 2.0 + s.rng.Float32()*3.0
			p.VX = float32(math.Cos(float64(angle))) * speed
			p.VY = 1.0 + s.rng.Float32()*3.0
			p.VZ = float32(math.Sin(float64(angle))) * speed
			p.MaxLife = 0.8 + s.rng.Float32()*0.5
			p.Size = 0.5 + s.rng.Float32()*0.5
			p.Gravity = -3.0
			p.Color = color

		case Trail:
			p.VX = (s.rng.Float32()*2 - 1) * 0.1
			p.VY = s.rng.Float32() * 0.2
			p.VZ = (s.rng.Float32()*2 - 1) * 0.1
			p.MaxLife = 0.3 + s.rng.Float32()*0.2
			p.Size = 0.15 + s.rng.Float32()*0.1
			p.Gravity = 0
			p.Color = color
		}

		p.Life = p.MaxLife
	}
}

// Update advances all active particles
func (s *System) Update(dt float32) {
	for i := range s.Pool {
		p := &s.Pool[i]
		if !p.Active {
			continue
		}

		p.Life -= dt
		if p.Life <= 0 {
			p.Active = false
			continue
		}

		p.VY += p.Gravity * dt
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.Z += p.VZ * dt

		// Slow down over time
		drag := float32(1.0 - 0.5*dt)
		p.VX *= drag
		p.VZ *= drag
	}
}

// Draw renders all active particles as billboards
func (s *System) Draw(camera rl.Camera3D) {
	for i := range s.Pool {
		p := &s.Pool[i]
		if !p.Active {
			continue
		}

		lifeRatio := p.Life / p.MaxLife
		alpha := uint8(float32(p.Color.A) * lifeRatio)
		size := p.Size * (0.5 + 0.5*lifeRatio) // shrink as it dies

		tint := rl.NewColor(p.Color.R, p.Color.G, p.Color.B, alpha)
		pos := rl.NewVector3(p.X, p.Y, p.Z)

		tex, ok := s.Textures[p.Type]
		if !ok {
			tex = s.Textures[Sparkle]
		}

		rl.DrawBillboard(camera, tex, pos, size, tint)
	}
}

// ActiveCount returns how many particles are alive
func (s *System) ActiveCount() int {
	n := 0
	for i := range s.Pool {
		if s.Pool[i].Active {
			n++
		}
	}
	return n
}

func (s *System) findFree() *Particle {
	for i := range s.Pool {
		if !s.Pool[i].Active {
			return &s.Pool[i]
		}
	}
	return nil
}
