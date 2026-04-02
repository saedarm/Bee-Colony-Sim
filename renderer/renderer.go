package renderer

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/saedarm/bee-colony/colony"
	"github.com/saedarm/bee-colony/terrain"
)

// Renderer handles all visual output
type Renderer struct {
	TerrainModel rl.Model
	TerrainMesh  rl.Mesh
	Camera       rl.Camera3D
	CameraAngle  float32
	CameraSpeed  float32
	AutoOrbit    bool

	// Colors for terrain gradient
	LowColor  rl.Color
	MidColor  rl.Color
	HighColor rl.Color
}

// New creates a renderer with default camera
func New() *Renderer {
	r := &Renderer{
		CameraAngle: 0,
		CameraSpeed: 0.3,
		AutoOrbit:   true,
		LowColor:    rl.NewColor(34, 85, 34, 255),   // dark green valley
		MidColor:    rl.NewColor(180, 160, 80, 255), // golden midlands
		HighColor:   rl.NewColor(255, 100, 50, 255), // orange-red peaks
	}

	r.Camera = rl.Camera3D{
		Position:   rl.NewVector3(30, 25, 30),
		Target:     rl.NewVector3(0, 0, 0),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       45,
		Projection: rl.CameraPerspective,
	}

	return r
}

// BuildTerrainMesh generates a 3D mesh from terrain data
func (r *Renderer) BuildTerrainMesh(t *terrain.Terrain) {
	w := t.Width
	d := t.Depth
	halfScale := t.ScaleXZ / 2.0

	triangleCount := (w - 1) * (d - 1) * 2
	vertexCount := triangleCount * 3

	vertices := make([]float32, vertexCount*3)
	colors := make([]uint8, vertexCount*4)
	normals := make([]float32, vertexCount*3)

	idx := 0
	cidx := 0
	nidx := 0

	for x := 0; x < w-1; x++ {
		for z := 0; z < d-1; z++ {
			// World positions for quad corners
			x0 := -halfScale + float32(x)/float32(w-1)*t.ScaleXZ
			x1 := -halfScale + float32(x+1)/float32(w-1)*t.ScaleXZ
			z0 := -halfScale + float32(z)/float32(d-1)*t.ScaleXZ
			z1 := -halfScale + float32(z+1)/float32(d-1)*t.ScaleXZ

			y00 := t.Heights[x][z] * t.ScaleY
			y10 := t.Heights[x+1][z] * t.ScaleY
			y01 := t.Heights[x][z+1] * t.ScaleY
			y11 := t.Heights[x+1][z+1] * t.ScaleY

			// Triangle 1: (x0,z0), (x1,z0), (x0,z1)
			verts := [6][3]float32{
				{x0, y00, z0}, {x1, y10, z0}, {x0, y01, z1},
				{x1, y10, z0}, {x1, y11, z1}, {x0, y01, z1},
			}

			for _, v := range verts {
				vertices[idx] = v[0]
				vertices[idx+1] = v[1]
				vertices[idx+2] = v[2]
				idx += 3

				// Color based on height
				h := v[1] / t.ScaleY // normalized 0..1
				c := r.heightColor(h)
				colors[cidx] = c.R
				colors[cidx+1] = c.G
				colors[cidx+2] = c.B
				colors[cidx+3] = c.A
				cidx += 4

				// Simple up normal (we could compute proper normals but this looks fine)
				normals[nidx] = 0
				normals[nidx+1] = 1
				normals[nidx+2] = 0
				nidx += 3
			}
		}
	}

	mesh := rl.Mesh{
		VertexCount:   int32(vertexCount),
		TriangleCount: int32(triangleCount),
	}

	// Allocate and copy data
	mesh.Vertices = &vertices[0]
	mesh.Normals = &normals[0]
	mesh.Colors = &colors[0]

	rl.UploadMesh(&mesh, false)

	r.TerrainMesh = mesh
	r.TerrainModel = rl.LoadModelFromMesh(mesh)
}

// heightColor blends between low/mid/high colors based on normalized height
func (r *Renderer) heightColor(h float32) rl.Color {
	if h < 0 {
		h = 0
	}
	if h > 1 {
		h = 1
	}

	if h < 0.5 {
		t := h * 2 // 0..1
		return rl.NewColor(
			uint8(float32(r.LowColor.R)*(1-t)+float32(r.MidColor.R)*t),
			uint8(float32(r.LowColor.G)*(1-t)+float32(r.MidColor.G)*t),
			uint8(float32(r.LowColor.B)*(1-t)+float32(r.MidColor.B)*t),
			255,
		)
	}
	t := (h - 0.5) * 2 // 0..1
	return rl.NewColor(
		uint8(float32(r.MidColor.R)*(1-t)+float32(r.HighColor.R)*t),
		uint8(float32(r.MidColor.G)*(1-t)+float32(r.HighColor.G)*t),
		uint8(float32(r.MidColor.B)*(1-t)+float32(r.HighColor.B)*t),
		255,
	)
}

// UpdateCamera handles auto-orbit and manual camera control
func (r *Renderer) UpdateCamera(dt float32) {
	// Manual camera: right-click drag to orbit, scroll to zoom
	if rl.IsMouseButtonDown(rl.MouseRightButton) {
		delta := rl.GetMouseDelta()
		r.CameraAngle += delta.X * 0.005
		r.AutoOrbit = false
	}

	// Scroll to zoom
	wheel := rl.GetMouseWheelMove()
	if wheel != 0 {
		dist := rl.Vector3Length(r.Camera.Position)
		dist -= wheel * 2
		if dist < 10 {
			dist = 10
		}
		if dist > 80 {
			dist = 80
		}
		r.Camera.Position = rl.NewVector3(
			float32(math.Cos(float64(r.CameraAngle)))*dist,
			r.Camera.Position.Y,
			float32(math.Sin(float64(r.CameraAngle)))*dist,
		)
	}

	// Auto-orbit
	if r.AutoOrbit {
		r.CameraAngle += dt * r.CameraSpeed
	}

	dist := rl.Vector3Length(r.Camera.Position)
	r.Camera.Position = rl.NewVector3(
		float32(math.Cos(float64(r.CameraAngle)))*dist*0.7,
		r.Camera.Position.Y,
		float32(math.Sin(float64(r.CameraAngle)))*dist*0.7,
	)
}

// DrawTerrain renders the terrain mesh
func (r *Renderer) DrawTerrain() {
	rl.DrawModel(r.TerrainModel, rl.NewVector3(0, 0, 0), 1.0, rl.White)
}

// DrawFoods renders food sources as geometric shapes
func (r *Renderer) DrawFoods(foods []*colony.FoodSource, t *terrain.Terrain) {
	for _, food := range foods {
		if !food.Active {
			continue
		}

		wx := float32(food.X)
		wz := float32(food.Z)
		wy := t.HeightAt(wx, wz) + 0.5

		// Size based on nectar remaining
		size := float32(0.3) + float32(food.Nectar)*0.7

		// Pulse animation
		food.PulsePhase += 0.02
		pulse := float32(1.0 + 0.1*math.Sin(float64(food.PulsePhase)))
		size *= pulse

		// Color based on fitness quality (brighter = better)
		alpha := uint8(100 + int(food.Nectar*155))
		pos := rl.NewVector3(wx, wy, wz)

		switch food.ShapeType {
		case 0: // Cube
			rl.DrawCube(pos, size, size, size, rl.NewColor(0, 255, 200, alpha))
			rl.DrawCubeWires(pos, size, size, size, rl.NewColor(0, 200, 160, 255))
		case 1: // Tetrahedron (drawn as a cone with 4 sides approximation)
			drawPyramid(pos, size, rl.NewColor(255, 200, 0, alpha))
		case 2: // Octahedron (two pyramids)
			drawOctahedron(pos, size, rl.NewColor(200, 50, 255, alpha))
		case 3: // Sphere standing in for icosahedron
			rl.DrawSphere(pos, size*0.5, rl.NewColor(50, 200, 255, alpha))
			rl.DrawSphereWires(pos, size*0.5, 6, 6, rl.NewColor(30, 150, 200, 255))
		case 4: // Cylinder standing in for dodecahedron
			rl.DrawCylinder(pos, size*0.3, size*0.5, size, 6, rl.NewColor(255, 100, 100, alpha))
			rl.DrawCylinderWires(pos, size*0.3, size*0.5, size, 6, rl.NewColor(200, 80, 80, 255))
		}
	}
}

// drawPyramid renders a simple 4-sided pyramid
func drawPyramid(pos rl.Vector3, size float32, color rl.Color) {
	top := rl.NewVector3(pos.X, pos.Y+size, pos.Z)
	half := size * 0.5
	corners := [4]rl.Vector3{
		rl.NewVector3(pos.X-half, pos.Y, pos.Z-half),
		rl.NewVector3(pos.X+half, pos.Y, pos.Z-half),
		rl.NewVector3(pos.X+half, pos.Y, pos.Z+half),
		rl.NewVector3(pos.X-half, pos.Y, pos.Z+half),
	}
	for i := 0; i < 4; i++ {
		next := (i + 1) % 4
		rl.DrawTriangle3D(top, corners[i], corners[next], color)
		rl.DrawLine3D(top, corners[i], rl.NewColor(color.R, color.G, color.B, 255))
	}
}

// drawOctahedron renders two pyramids joined at the base
func drawOctahedron(pos rl.Vector3, size float32, color rl.Color) {
	drawPyramid(pos, size, color)
	// Bottom pyramid (inverted)
	bottom := rl.NewVector3(pos.X, pos.Y-size, pos.Z)
	half := size * 0.5
	corners := [4]rl.Vector3{
		rl.NewVector3(pos.X-half, pos.Y, pos.Z-half),
		rl.NewVector3(pos.X+half, pos.Y, pos.Z-half),
		rl.NewVector3(pos.X+half, pos.Y, pos.Z+half),
		rl.NewVector3(pos.X-half, pos.Y, pos.Z+half),
	}
	for i := 0; i < 4; i++ {
		next := (i + 1) % 4
		rl.DrawTriangle3D(bottom, corners[next], corners[i], color)
	}
}

// DrawBees renders all bee agents
func (r *Renderer) DrawBees(bees []*colony.Bee, t *terrain.Terrain) {
	for _, bee := range bees {
		wx := float32(bee.X)
		wz := float32(bee.Z)
		wy := t.HeightAt(wx, wz) + 1.5 // fly above terrain

		// Wobble for organic movement
		wobbleX := float32(math.Sin(float64(bee.Wobble))) * 0.3
		wobbleZ := float32(math.Cos(float64(bee.Wobble*1.3))) * 0.3
		wy += float32(math.Sin(float64(bee.Wobble*2.0))) * 0.2

		pos := rl.NewVector3(wx+wobbleX, wy, wz+wobbleZ)

		// Body color based on role
		var bodyColor rl.Color
		switch bee.Role {
		case colony.Employed:
			bodyColor = rl.NewColor(255, 200, 0, 255) // gold
		case colony.Onlooker:
			bodyColor = rl.NewColor(80, 160, 255, 255) // blue
		case colony.Scout:
			bodyColor = rl.NewColor(255, 60, 60, 255) // red
		}

		// Bee body: ellipsoid approximated as sphere
		rl.DrawSphere(pos, 0.25, bodyColor)

		// Head
		headPos := rl.NewVector3(pos.X+0.2, pos.Y+0.05, pos.Z)
		rl.DrawSphere(headPos, 0.12, rl.NewColor(60, 40, 20, 255))

		// Wings (flapping triangles)
		wingOffset := float32(math.Sin(float64(bee.WingPhase))) * 0.3
		leftWing := rl.NewVector3(pos.X, pos.Y+0.15+wingOffset, pos.Z-0.3)
		rightWing := rl.NewVector3(pos.X, pos.Y+0.15-wingOffset, pos.Z+0.3)
		rl.DrawLine3D(pos, leftWing, rl.NewColor(200, 200, 200, 150))
		rl.DrawLine3D(pos, rightWing, rl.NewColor(200, 200, 200, 150))

		// Trail line to target (faint)
		if bee.Role == colony.Employed || bee.Role == colony.Onlooker {
			targetY := t.HeightAt(float32(bee.TargetX), float32(bee.TargetZ)) + 0.5
			targetPos := rl.NewVector3(float32(bee.TargetX), targetY, float32(bee.TargetZ))
			trailColor := rl.NewColor(bodyColor.R, bodyColor.G, bodyColor.B, 40)
			rl.DrawLine3D(pos, targetPos, trailColor)
		}
	}
}

// DrawHive renders the hive at center
func (r *Renderer) DrawHive(t *terrain.Terrain) {
	hy := t.HeightAt(0, 0) + 0.1
	pos := rl.NewVector3(0, hy, 0)
	rl.DrawCylinder(pos, 1.5, 1.0, 1.5, 8, rl.NewColor(180, 140, 60, 200))
	rl.DrawCylinderWires(pos, 1.5, 1.0, 1.5, 8, rl.NewColor(140, 100, 40, 255))

	// Roof
	roofPos := rl.NewVector3(0, hy+1.5, 0)
	rl.DrawCylinder(roofPos, 0.1, 1.8, 0.8, 8, rl.NewColor(160, 120, 40, 200))
}

// DrawHUD renders the stats overlay
func (r *Renderer) DrawHUD(c *colony.Colony, presetName string, phase string) {
	// Background panel
	rl.DrawRectangle(10, 10, 280, 230, rl.NewColor(0, 0, 0, 160))
	rl.DrawRectangleLines(10, 10, 280, 230, rl.NewColor(255, 200, 0, 200))

	y := int32(20)
	spacing := int32(22)

	rl.DrawText("BEE COLONY SIMULATOR", 20, y, 16, rl.Gold)
	y += spacing + 5

	rl.DrawText(fmt.Sprintf("Preset: %s", presetName), 20, y, 14, rl.White)
	y += spacing

	rl.DrawText(fmt.Sprintf("Generation: %d", c.Generation), 20, y, 14, rl.White)
	y += spacing

	rl.DrawText(fmt.Sprintf("Phase: %s", phase), 20, y, 14, rl.LightGray)
	y += spacing

	rl.DrawText(fmt.Sprintf("Best Fitness: %.4f", c.BestFitness), 20, y, 14, rl.NewColor(100, 255, 100, 255))
	y += spacing

	rl.DrawText(fmt.Sprintf("Active Foods: %d", countActive(c.Foods)), 20, y, 14, rl.White)
	y += spacing

	rl.DrawText(fmt.Sprintf("Exhausted: %d  Discovered: %d", c.FoodsExhausted, c.FoodsDiscovered), 20, y, 14, rl.LightGray)
	y += spacing + 5

	// Legend
	rl.DrawRectangle(20, y, 10, 10, rl.NewColor(255, 200, 0, 255))
	rl.DrawText("Employed", 35, y, 12, rl.NewColor(255, 200, 0, 255))
	rl.DrawRectangle(120, y, 10, 10, rl.NewColor(80, 160, 255, 255))
	rl.DrawText("Onlooker", 135, y, 12, rl.NewColor(80, 160, 255, 255))
	rl.DrawRectangle(220, y, 10, 10, rl.NewColor(255, 60, 60, 255))
	rl.DrawText("Scout", 235, y, 12, rl.NewColor(255, 60, 60, 255))

	// Controls hint
	rl.DrawText("Right-click drag: orbit | Scroll: zoom | R: restart | 1-5: presets", 10, int32(rl.GetScreenHeight())-25, 12, rl.NewColor(200, 200, 200, 150))
}

func countActive(foods []*colony.FoodSource) int {
	n := 0
	for _, f := range foods {
		if f.Active {
			n++
		}
	}
	return n
}
