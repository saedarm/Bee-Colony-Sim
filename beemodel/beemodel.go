package beemodel

import (
	"math"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Assets holds the loaded/generated models for bee rendering
type Assets struct {
	BeeModel  rl.Model
	Loaded    bool
}

// New creates a bee model. It first tries to load from assets/models/bee.glb,
// and if not found, generates a procedural low-poly bee mesh.
func New() *Assets {
	a := &Assets{}

	// Try to load external model first
	if fileExists("assets/models/bee.glb") {
		a.BeeModel = rl.LoadModel("assets/models/bee.glb")
		a.Loaded = true
		return a
	}
	if fileExists("assets/models/bee.obj") {
		a.BeeModel = rl.LoadModel("assets/models/bee.obj")
		a.Loaded = true
		return a
	}

	// Generate procedural bee
	a.BeeModel = generateProceduralBee()
	a.Loaded = true
	return a
}

// Unload frees GPU resources
func (a *Assets) Unload() {
	if a.Loaded {
		rl.UnloadModel(a.BeeModel)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// generateProceduralBee builds a multi-part mesh: abdomen + thorax + head
// all combined into a single mesh with vertex colors
func generateProceduralBee() rl.Model {
	// We'll build the bee from UV spheres of different sizes and colors
	// Abdomen: large, yellow with dark stripes
	// Thorax: medium, yellow
	// Head: small, dark brown

	var allVerts []float32
	var allColors []uint8
	var allNormals []float32

	beeYellow := rl.NewColor(255, 200, 30, 255)
	beeStripe := rl.NewColor(40, 25, 10, 255)
	beeBrown := rl.NewColor(60, 40, 20, 255)
	wingColor := rl.NewColor(200, 210, 240, 120)

	// Abdomen - elongated sphere at rear
	addEllipsoid(&allVerts, &allColors, &allNormals,
		-0.35, 0, 0,    // center offset
		0.45, 0.35, 0.35, // radii
		12, 8,
		beeYellow, beeStripe, true) // striped=true

	// Thorax - rounder sphere in middle
	addEllipsoid(&allVerts, &allColors, &allNormals,
		0.15, 0.02, 0,
		0.28, 0.26, 0.26,
		10, 8,
		beeYellow, beeYellow, false)

	// Head - small dark sphere at front
	addEllipsoid(&allVerts, &allColors, &allNormals,
		0.48, 0.05, 0,
		0.16, 0.14, 0.14,
		8, 6,
		beeBrown, beeBrown, false)

	// Wings - flat quads (two triangles each)
	// Left wing
	addWing(&allVerts, &allColors, &allNormals,
		0.1, 0.28, -0.15,   // base
		-0.15, 0.55, -0.5,  // tip
		0.3, 0.35, -0.05,   // trailing edge
		wingColor)

	// Right wing
	addWing(&allVerts, &allColors, &allNormals,
		0.1, 0.28, 0.15,
		-0.15, 0.55, 0.5,
		0.3, 0.35, 0.05,
		wingColor)

	triCount := len(allVerts) / 9
	vertCount := triCount * 3

	mesh := rl.Mesh{
		VertexCount:   int32(vertCount),
		TriangleCount: int32(triCount),
	}

	mesh.Vertices = &allVerts[0]
	mesh.Colors = &allColors[0]
	mesh.Normals = &allNormals[0]

	rl.UploadMesh(&mesh, false)
	return rl.LoadModelFromMesh(mesh)
}

// addEllipsoid generates a UV sphere with given radii and vertex colors
// If striped=true, alternating latitude bands get the stripe color
func addEllipsoid(verts *[]float32, colors *[]uint8, normals *[]float32,
	cx, cy, cz float32,
	rx, ry, rz float32,
	slices, stacks int,
	baseColor, stripeColor rl.Color,
	striped bool) {

	for i := 0; i < stacks; i++ {
		phi1 := math.Pi * float64(i) / float64(stacks)
		phi2 := math.Pi * float64(i+1) / float64(stacks)

		for j := 0; j < slices; j++ {
			theta1 := 2 * math.Pi * float64(j) / float64(slices)
			theta2 := 2 * math.Pi * float64(j+1) / float64(slices)

			// Four vertices of quad
			p := [4][3]float32{
				spherePoint(cx, cy, cz, rx, ry, rz, phi1, theta1),
				spherePoint(cx, cy, cz, rx, ry, rz, phi1, theta2),
				spherePoint(cx, cy, cz, rx, ry, rz, phi2, theta2),
				spherePoint(cx, cy, cz, rx, ry, rz, phi2, theta1),
			}

			// Color: stripe every other band
			col := baseColor
			if striped && i%2 == 1 {
				col = stripeColor
			}

			// Two triangles
			addTriangle(verts, colors, normals, p[0], p[1], p[2], col)
			addTriangle(verts, colors, normals, p[0], p[2], p[3], col)
		}
	}
}

func spherePoint(cx, cy, cz, rx, ry, rz float32, phi, theta float64) [3]float32 {
	sp := float32(math.Sin(phi))
	cp := float32(math.Cos(phi))
	st := float32(math.Sin(theta))
	ct := float32(math.Cos(theta))
	return [3]float32{
		cx + rx*sp*ct,
		cy + ry*cp,
		cz + rz*sp*st,
	}
}

// addWing adds a flat triangular wing
func addWing(verts *[]float32, colors *[]uint8, normals *[]float32,
	bx, by, bz float32,
	tx, ty, tz float32,
	ex, ey, ez float32,
	col rl.Color) {

	p0 := [3]float32{bx, by, bz}
	p1 := [3]float32{tx, ty, tz}
	p2 := [3]float32{ex, ey, ez}

	// Both sides (front and back face)
	addTriangle(verts, colors, normals, p0, p1, p2, col)
	addTriangle(verts, colors, normals, p0, p2, p1, col) // back face
}

func addTriangle(verts *[]float32, colors *[]uint8, normals *[]float32,
	v0, v1, v2 [3]float32, col rl.Color) {

	// Normal
	ux := v1[0] - v0[0]
	uy := v1[1] - v0[1]
	uz := v1[2] - v0[2]
	vx := v2[0] - v0[0]
	vy := v2[1] - v0[1]
	vz := v2[2] - v0[2]
	nx := uy*vz - uz*vy
	ny := uz*vx - ux*vz
	nz := ux*vy - uy*vx
	l := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
	if l > 0 {
		nx /= l
		ny /= l
		nz /= l
	}

	for _, v := range [3][3]float32{v0, v1, v2} {
		*verts = append(*verts, v[0], v[1], v[2])
		*colors = append(*colors, col.R, col.G, col.B, col.A)
		*normals = append(*normals, nx, ny, nz)
	}
}
