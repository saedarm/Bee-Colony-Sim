package terrain

import (
	"math"
)

// Type selects which fitness landscape to generate
type Type int

const (
	Rastrigin Type = iota
	Ackley
	Rosenbrock
	PerlinNoise
	RandomPeaks
)

// Terrain holds the heightmap mesh data and fitness evaluation
type Terrain struct {
	Type       Type
	Width      int // grid subdivisions
	Depth      int // grid subdivisions
	ScaleXZ    float32
	ScaleY     float32
	Heights    [][]float32 // [x][z] normalized 0..1
	MinFitness float64
	MaxFitness float64
	peaks      []peak // for RandomPeaks type
	permTable  [512]int
}

type peak struct {
	X, Z      float64
	Height    float64
	Spread    float64
}

// New creates a terrain with the given type and grid resolution
func New(terrainType Type, gridSize int) *Terrain {
	t := &Terrain{
		Type:    terrainType,
		Width:   gridSize,
		Depth:   gridSize,
		ScaleXZ: 40.0,
		ScaleY:  8.0,
	}

	// Initialize Perlin permutation table
	t.initPerlin()

	// Generate height data
	t.Heights = make([][]float32, gridSize)
	for i := range t.Heights {
		t.Heights[i] = make([]float32, gridSize)
	}

	t.MinFitness = math.MaxFloat64
	t.MaxFitness = -math.MaxFloat64

	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			// Map grid coords to function domain
			fx := mapRange(float64(x), 0, float64(gridSize-1), -5.12, 5.12)
			fz := mapRange(float64(z), 0, float64(gridSize-1), -5.12, 5.12)

			val := t.evaluate(fx, fz)
			if val < t.MinFitness {
				t.MinFitness = val
			}
			if val > t.MaxFitness {
				t.MaxFitness = val
			}

			t.Heights[x][z] = float32(val)
		}
	}

	// Normalize heights to 0..1
	rang := t.MaxFitness - t.MinFitness
	if rang == 0 {
		rang = 1
	}
	for x := 0; x < gridSize; x++ {
		for z := 0; z < gridSize; z++ {
			t.Heights[x][z] = float32((float64(t.Heights[x][z]) - t.MinFitness) / rang)
		}
	}

	return t
}

// Fitness evaluates the fitness function at world coordinates (wx, wz)
// Returns higher values for better food sources (we invert minimization functions)
func (t *Terrain) Fitness(wx, wz float64) float64 {
	// Map world coords to function domain
	halfScale := float64(t.ScaleXZ) / 2.0
	fx := mapRange(wx, -halfScale, halfScale, -5.12, 5.12)
	fz := mapRange(wz, -halfScale, halfScale, -5.12, 5.12)
	raw := t.evaluate(fx, fz)
	// Invert: higher = better food source
	return t.MaxFitness - raw + 0.001
}

// HeightAt returns the interpolated terrain height at world position
func (t *Terrain) HeightAt(wx, wz float32) float32 {
	halfScale := t.ScaleXZ / 2.0
	// Map world coords to grid coords
	gx := mapRange32(wx, -halfScale, halfScale, 0, float32(t.Width-1))
	gz := mapRange32(wz, -halfScale, halfScale, 0, float32(t.Depth-1))

	// Clamp
	if gx < 0 {
		gx = 0
	}
	if gz < 0 {
		gz = 0
	}
	if gx >= float32(t.Width-1) {
		gx = float32(t.Width - 2)
	}
	if gz >= float32(t.Depth-1) {
		gz = float32(t.Depth - 2)
	}

	ix := int(gx)
	iz := int(gz)
	fx := gx - float32(ix)
	fz := gz - float32(iz)

	// Bilinear interpolation
	h00 := t.Heights[ix][iz]
	h10 := t.Heights[ix+1][iz]
	h01 := t.Heights[ix][iz+1]
	h11 := t.Heights[ix+1][iz+1]

	h0 := h00*(1-fx) + h10*fx
	h1 := h01*(1-fx) + h11*fx
	h := h0*(1-fz) + h1*fz

	return h * t.ScaleY
}

// evaluate returns the raw fitness function value
func (t *Terrain) evaluate(x, z float64) float64 {
	switch t.Type {
	case Rastrigin:
		return t.rastrigin(x, z)
	case Ackley:
		return t.ackley(x, z)
	case Rosenbrock:
		return t.rosenbrock(x, z)
	case PerlinNoise:
		return t.perlinTerrain(x, z)
	case RandomPeaks:
		return t.randomPeaksTerrain(x, z)
	default:
		return t.rastrigin(x, z)
	}
}

// Rastrigin: lots of local optima - beautiful spiky landscape
func (t *Terrain) rastrigin(x, z float64) float64 {
	A := 10.0
	return 2*A + (x*x - A*math.Cos(2*math.Pi*x)) + (z*z - A*math.Cos(2*math.Pi*z))
}

// Ackley: one sharp global minimum surrounded by bumpy plateau
func (t *Terrain) ackley(x, z float64) float64 {
	a := 20.0
	b := 0.2
	c := 2 * math.Pi
	sum1 := x*x + z*z
	sum2 := math.Cos(c*x) + math.Cos(c*z)
	return -a*math.Exp(-b*math.Sqrt(sum1/2)) - math.Exp(sum2/2) + a + math.E
}

// Rosenbrock: narrow curved valley
func (t *Terrain) rosenbrock(x, z float64) float64 {
	return 100*(z-x*x)*(z-x*x) + (1-x)*(1-x)
}

// Perlin noise terrain - organic rolling hills
func (t *Terrain) perlinTerrain(x, z float64) float64 {
	// Multi-octave noise for natural-looking terrain
	val := 0.0
	amp := 1.0
	freq := 0.5
	for i := 0; i < 4; i++ {
		val += amp * t.perlin2D(x*freq, z*freq)
		amp *= 0.5
		freq *= 2.0
	}
	// Shift to positive
	return (val + 2.0) * 10.0
}

// Random peaks
func (t *Terrain) randomPeaksTerrain(x, z float64) float64 {
	if len(t.peaks) == 0 {
		// Generate deterministic peaks using simple hash
		t.peaks = []peak{
			{X: 0.0, Z: 0.0, Height: 40, Spread: 1.5},
			{X: 3.0, Z: -2.0, Height: 30, Spread: 1.0},
			{X: -2.5, Z: 3.5, Height: 35, Spread: 1.2},
			{X: -4.0, Z: -3.0, Height: 20, Spread: 0.8},
			{X: 2.0, Z: 4.0, Height: 25, Spread: 1.0},
			{X: 4.5, Z: 1.0, Height: 15, Spread: 0.6},
			{X: -1.0, Z: -4.5, Height: 28, Spread: 1.3},
		}
	}
	val := 0.0
	for _, p := range t.peaks {
		dx := x - p.X
		dz := z - p.Z
		val += p.Height * math.Exp(-(dx*dx+dz*dz)/(2*p.Spread*p.Spread))
	}
	return val
}

// Simple Perlin noise implementation
func (t *Terrain) initPerlin() {
	perm := [256]int{
		151, 160, 137, 91, 90, 15, 131, 13, 201, 95, 96, 53, 194, 233, 7, 225,
		140, 36, 103, 30, 69, 142, 8, 99, 37, 240, 21, 10, 23, 190, 6, 148,
		247, 120, 234, 75, 0, 26, 197, 62, 94, 252, 219, 203, 117, 35, 11, 32,
		57, 177, 33, 88, 237, 149, 56, 87, 174, 20, 125, 136, 171, 168, 68, 175,
		74, 165, 71, 134, 139, 48, 27, 166, 77, 146, 158, 231, 83, 111, 229, 122,
		60, 211, 133, 230, 220, 105, 92, 41, 55, 46, 245, 40, 244, 102, 143, 54,
		65, 25, 63, 161, 1, 216, 80, 73, 209, 76, 132, 187, 208, 89, 18, 169,
		200, 196, 135, 130, 116, 188, 159, 86, 164, 100, 109, 198, 173, 186, 3, 64,
		52, 217, 226, 250, 124, 123, 5, 202, 38, 147, 118, 126, 255, 82, 85, 212,
		207, 206, 59, 227, 47, 16, 58, 17, 182, 189, 28, 42, 223, 183, 170, 213,
		119, 248, 152, 2, 44, 154, 163, 70, 221, 153, 101, 155, 167, 43, 172, 9,
		129, 22, 39, 253, 19, 98, 108, 110, 79, 113, 224, 232, 178, 185, 112, 104,
		218, 246, 97, 228, 251, 34, 242, 193, 238, 210, 144, 12, 191, 179, 162, 241,
		81, 51, 145, 235, 249, 14, 239, 107, 49, 192, 214, 31, 181, 199, 106, 157,
		184, 84, 204, 176, 115, 121, 50, 45, 127, 4, 150, 254, 138, 236, 205, 93,
		222, 114, 67, 29, 24, 72, 243, 141, 128, 195, 78, 66, 215, 61, 156, 180,
	}
	for i := 0; i < 256; i++ {
		t.permTable[i] = perm[i]
		t.permTable[i+256] = perm[i]
	}
}

func (t *Terrain) perlin2D(x, y float64) float64 {
	X := int(math.Floor(x)) & 255
	Y := int(math.Floor(y)) & 255
	x -= math.Floor(x)
	y -= math.Floor(y)
	u := fade(x)
	v := fade(y)
	A := t.permTable[X] + Y
	B := t.permTable[X+1] + Y
	return lerp64(v,
		lerp64(u, grad2D(t.permTable[A], x, y), grad2D(t.permTable[B], x-1, y)),
		lerp64(u, grad2D(t.permTable[A+1], x, y-1), grad2D(t.permTable[B+1], x-1, y-1)),
	)
}

func fade(t float64) float64   { return t * t * t * (t*(t*6-15) + 10) }
func lerp64(t, a, b float64) float64 { return a + t*(b-a) }
func grad2D(hash int, x, y float64) float64 {
	h := hash & 3
	switch h {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	default:
		return -x - y
	}
}

func mapRange(val, inMin, inMax, outMin, outMax float64) float64 {
	return outMin + (val-inMin)*(outMax-outMin)/(inMax-inMin)
}

func mapRange32(val, inMin, inMax, outMin, outMax float32) float32 {
	return outMin + (val-inMin)*(outMax-outMin)/(inMax-inMin)
}
