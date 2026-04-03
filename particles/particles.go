package particles

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type ParticleType int

const (
	Sparkle    ParticleType = iota
	Pollen
	Burst
	Trail
	ScoutTrail // bright fast-fading trail for scout launches
)

const MaxParticles = 1500

type Particle struct {
	Active          bool
	Type            ParticleType
	X, Y, Z        float32
	VX, VY, VZ     float32
	Life, MaxLife   float32
	Size            float32
	Color           rl.Color
	Gravity         float32
}

type System struct {
	Pool     [MaxParticles]Particle
	Textures map[ParticleType]rl.Texture2D
	rng      *rand.Rand
}

func NewSystem() *System {
	s := &System{
		Textures: make(map[ParticleType]rl.Texture2D),
		rng:      rand.New(rand.NewSource(12345)),
	}
	s.Textures[Sparkle] = generateGlowTexture(32, rl.NewColor(255, 255, 255, 255))
	s.Textures[Trail] = generateGlowTexture(16, rl.NewColor(255, 255, 255, 255))
	s.Textures[ScoutTrail] = generateGlowTexture(24, rl.NewColor(255, 255, 255, 255))
	s.Textures[Pollen] = generateFuzzyTexture(24, rl.NewColor(255, 230, 100, 255))
	s.Textures[Burst] = generateStarTexture(32, rl.NewColor(255, 255, 255, 255))
	return s
}

func (s *System) Unload() {
	for _, tex := range s.Textures {
		rl.UnloadTexture(tex)
	}
}

func generateGlowTexture(size int, tint rl.Color) rl.Texture2D {
	img := rl.GenImageColor(size, size, rl.NewColor(0, 0, 0, 0))
	center := float32(size) / 2.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float32(x) - center + 0.5
			dy := float32(y) - center + 0.5
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			norm := dist / center
			if norm > 1 { norm = 1 }
			alpha := uint8(float32(tint.A) * (1 - norm*norm))
			rl.ImageDrawPixel(img, int32(x), int32(y), rl.NewColor(tint.R, tint.G, tint.B, alpha))
		}
	}
	tex := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	return tex
}

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
			if norm > 1 { continue }
			noise := float32(rng.Float64()*0.4 + 0.6)
			alpha := uint8(float32(tint.A) * (1 - norm) * noise)
			rl.ImageDrawPixel(img, int32(x), int32(y), rl.NewColor(tint.R, tint.G, tint.B, alpha))
		}
	}
	tex := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	return tex
}

func generateStarTexture(size int, tint rl.Color) rl.Texture2D {
	img := rl.GenImageColor(size, size, rl.NewColor(0, 0, 0, 0))
	center := float32(size) / 2.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float32(x) - center + 0.5
			dy := float32(y) - center + 0.5
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			norm := dist / center
			if norm > 1 { continue }
			ax := float32(math.Abs(float64(dx)))
			ay := float32(math.Abs(float64(dy)))
			minA := ax
			if ay < minA { minA = ay }
			star := float32(1.0) - minA/center
			if star < 0 { star = 0 }
			alpha := uint8(float32(tint.A) * (1 - norm) * (0.3 + 0.7*star))
			rl.ImageDrawPixel(img, int32(x), int32(y), rl.NewColor(tint.R, tint.G, tint.B, alpha))
		}
	}
	tex := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	return tex
}

func (s *System) Emit(ptype ParticleType, x, y, z float32, count int, color rl.Color) {
	for i := 0; i < count; i++ {
		p := s.findFree()
		if p == nil { return }
		p.Active = true
		p.Type = ptype
		p.X, p.Y, p.Z = x, y, z

		switch ptype {
		case Sparkle:
			p.VX = (s.rng.Float32()*2 - 1) * 0.8
			p.VY = s.rng.Float32() * 2.5
			p.VZ = (s.rng.Float32()*2 - 1) * 0.8
			p.MaxLife = 0.8 + s.rng.Float32()*0.5
			p.Size = 0.35 + s.rng.Float32()*0.3
			p.Gravity = -1.0
			p.Color = color
		case Pollen:
			p.VX = (s.rng.Float32()*2 - 1) * 0.2
			p.VY = 0.1 + s.rng.Float32()*0.3
			p.VZ = (s.rng.Float32()*2 - 1) * 0.2
			p.MaxLife = 2.0 + s.rng.Float32()*1.5
			p.Size = 0.2 + s.rng.Float32()*0.15
			p.Gravity = 0.05
			p.Color = rl.NewColor(255, 230, 100, 180)
		case Burst:
			angle := s.rng.Float32() * math.Pi * 2
			speed := float32(3.0 + s.rng.Float64()*4.0)
			p.VX = float32(math.Cos(float64(angle))) * speed
			p.VY = 2.0 + s.rng.Float32()*4.0
			p.VZ = float32(math.Sin(float64(angle))) * speed
			p.MaxLife = 1.0 + s.rng.Float32()*0.5
			p.Size = 0.6 + s.rng.Float32()*0.5
			p.Gravity = -4.0
			p.Color = color
		case Trail:
			p.VX = (s.rng.Float32()*2 - 1) * 0.08
			p.VY = s.rng.Float32() * 0.15
			p.VZ = (s.rng.Float32()*2 - 1) * 0.08
			p.MaxLife = 0.4 + s.rng.Float32()*0.3
			p.Size = 0.12 + s.rng.Float32()*0.08
			p.Gravity = 0
			p.Color = color
		case ScoutTrail:
			p.VX = (s.rng.Float32()*2 - 1) * 0.3
			p.VY = s.rng.Float32() * 0.5
			p.VZ = (s.rng.Float32()*2 - 1) * 0.3
			p.MaxLife = 0.6 + s.rng.Float32()*0.4
			p.Size = 0.4 + s.rng.Float32()*0.3
			p.Gravity = -0.5
			p.Color = color
		}
		p.Life = p.MaxLife
	}
}

func (s *System) Update(dt float32) {
	for i := range s.Pool {
		p := &s.Pool[i]
		if !p.Active { continue }
		p.Life -= dt
		if p.Life <= 0 { p.Active = false; continue }
		p.VY += p.Gravity * dt
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.Z += p.VZ * dt
		drag := float32(1.0 - 0.5*dt)
		p.VX *= drag
		p.VZ *= drag
	}
}

func (s *System) Draw(camera rl.Camera3D) {
	for i := range s.Pool {
		p := &s.Pool[i]
		if !p.Active { continue }
		lifeRatio := p.Life / p.MaxLife
		alpha := uint8(float32(p.Color.A) * lifeRatio)
		size := p.Size * (0.4 + 0.6*lifeRatio)
		tint := rl.NewColor(p.Color.R, p.Color.G, p.Color.B, alpha)
		pos := rl.NewVector3(p.X, p.Y, p.Z)
		tex := s.Textures[Sparkle]
		if t, ok := s.Textures[p.Type]; ok { tex = t }
		rl.DrawBillboard(camera, tex, pos, size, tint)
	}
}

func (s *System) ActiveCount() int {
	n := 0
	for i := range s.Pool { if s.Pool[i].Active { n++ } }
	return n
}

func (s *System) findFree() *Particle {
	for i := range s.Pool { if !s.Pool[i].Active { return &s.Pool[i] } }
	return nil
}
