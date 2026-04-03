package renderer

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/saedarm/bee-colony/beemodel"
	"github.com/saedarm/bee-colony/colony"
	"github.com/saedarm/bee-colony/particles"
	"github.com/saedarm/bee-colony/terrain"
)

type Renderer struct {
	TerrainModel rl.Model
	TerrainMesh  rl.Mesh
	Camera       rl.Camera3D
	BeeAssets    *beemodel.Assets
	Particles    *particles.System

	CameraYaw, CameraPitch, CameraDist float32
	TargetX, TargetY, TargetZ          float32

	LowColor, MidColor, HighColor, WaterColor rl.Color

	// Track food states for particle emission
	prevFoodNectar []float64
	frameCount     int
}

func New() *Renderer {
	r := &Renderer{
		CameraYaw:   0.8,
		CameraPitch: 0.55,
		CameraDist:  50.0,
		TargetX:     0, TargetY: 3, TargetZ: 0,
		LowColor:   rl.NewColor(30, 130, 60, 255),
		MidColor:   rl.NewColor(230, 200, 80, 255),
		HighColor:  rl.NewColor(255, 80, 30, 255),
		WaterColor: rl.NewColor(25, 50, 90, 255),
	}
	r.BeeAssets = beemodel.New()
	r.Particles = particles.NewSystem()
	r.rebuildCamera()
	return r
}

func (r *Renderer) Unload() {
	r.BeeAssets.Unload()
	r.Particles.Unload()
}

func (r *Renderer) rebuildCamera() {
	cosP := float32(math.Cos(float64(r.CameraPitch)))
	sinP := float32(math.Sin(float64(r.CameraPitch)))
	cosY := float32(math.Cos(float64(r.CameraYaw)))
	sinY := float32(math.Sin(float64(r.CameraYaw)))
	r.Camera = rl.Camera3D{
		Position: rl.NewVector3(
			r.TargetX+cosP*cosY*r.CameraDist,
			r.TargetY+sinP*r.CameraDist,
			r.TargetZ+cosP*sinY*r.CameraDist,
		),
		Target:     rl.NewVector3(r.TargetX, r.TargetY, r.TargetZ),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       45,
		Projection: rl.CameraPerspective,
	}
}

// BuildTerrainMesh - same as v2
func (r *Renderer) BuildTerrainMesh(t *terrain.Terrain) {
	w := t.Width
	d := t.Depth
	halfScale := t.ScaleXZ / 2.0
	triangleCount := (w - 1) * (d - 1) * 2
	vertexCount := triangleCount * 3
	vertices := make([]float32, vertexCount*3)
	colors := make([]uint8, vertexCount*4)
	normals := make([]float32, vertexCount*3)
	idx, cidx, nidx := 0, 0, 0

	for x := 0; x < w-1; x++ {
		for z := 0; z < d-1; z++ {
			x0 := -halfScale + float32(x)/float32(w-1)*t.ScaleXZ
			x1 := -halfScale + float32(x+1)/float32(w-1)*t.ScaleXZ
			z0 := -halfScale + float32(z)/float32(d-1)*t.ScaleXZ
			z1 := -halfScale + float32(z+1)/float32(d-1)*t.ScaleXZ
			y00 := t.Heights[x][z] * t.ScaleY
			y10 := t.Heights[x+1][z] * t.ScaleY
			y01 := t.Heights[x][z+1] * t.ScaleY
			y11 := t.Heights[x+1][z+1] * t.ScaleY

			v0 := [3]float32{x0, y00, z0}
			v1 := [3]float32{x1, y10, z0}
			v2 := [3]float32{x0, y01, z1}
			v3 := [3]float32{x1, y10, z0}
			v4 := [3]float32{x1, y11, z1}
			v5 := [3]float32{x0, y01, z1}
			n1 := computeNormal(v0, v1, v2)
			n2 := computeNormal(v3, v4, v5)

			tris := [2][3][3]float32{{v0, v1, v2}, {v3, v4, v5}}
			norms := [2][3]float32{n1, n2}
			for ti, tri := range tris {
				for _, v := range tri {
					vertices[idx], vertices[idx+1], vertices[idx+2] = v[0], v[1], v[2]
					idx += 3
					h := v[1] / t.ScaleY
					c := r.heightColor(h)
					shade := float32(0.55 + 0.45*float64(norms[ti][1]))
					colors[cidx] = clampB(float32(c.R) * shade)
					colors[cidx+1] = clampB(float32(c.G) * shade)
					colors[cidx+2] = clampB(float32(c.B) * shade)
					colors[cidx+3] = 255
					cidx += 4
					normals[nidx], normals[nidx+1], normals[nidx+2] = norms[ti][0], norms[ti][1], norms[ti][2]
					nidx += 3
				}
			}
		}
	}

	mesh := rl.Mesh{VertexCount: int32(vertexCount), TriangleCount: int32(triangleCount)}
	mesh.Vertices = &vertices[0]
	mesh.Normals = &normals[0]
	mesh.Colors = &colors[0]
	rl.UploadMesh(&mesh, false)
	r.TerrainMesh = mesh
	r.TerrainModel = rl.LoadModelFromMesh(mesh)
}

func computeNormal(v0, v1, v2 [3]float32) [3]float32 {
	ux, uy, uz := v1[0]-v0[0], v1[1]-v0[1], v1[2]-v0[2]
	vx, vy, vz := v2[0]-v0[0], v2[1]-v0[1], v2[2]-v0[2]
	nx := uy*vz - uz*vy
	ny := uz*vx - ux*vz
	nz := ux*vy - uy*vx
	l := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
	if l > 0 {
		return [3]float32{nx / l, ny / l, nz / l}
	}
	return [3]float32{0, 1, 0}
}

func clampB(v float32) uint8 {
	if v < 0 { return 0 }
	if v > 255 { return 255 }
	return uint8(v)
}

func (r *Renderer) heightColor(h float32) rl.Color {
	if h < 0 { h = 0 }
	if h > 1 { h = 1 }
	if h < 0.25 { return lerpColor(r.WaterColor, r.LowColor, h/0.25) }
	if h < 0.55 { return lerpColor(r.LowColor, r.MidColor, (h-0.25)/0.30) }
	return lerpColor(r.MidColor, r.HighColor, (h-0.55)/0.45)
}

func lerpColor(a, b rl.Color, t float32) rl.Color {
	if t < 0 { t = 0 }
	if t > 1 { t = 1 }
	return rl.NewColor(
		uint8(float32(a.R)*(1-t)+float32(b.R)*t),
		uint8(float32(a.G)*(1-t)+float32(b.G)*t),
		uint8(float32(a.B)*(1-t)+float32(b.B)*t), 255)
}

func (r *Renderer) UpdateCamera(dt float32) {
	if rl.IsMouseButtonDown(rl.MouseRightButton) {
		d := rl.GetMouseDelta()
		r.CameraYaw += d.X * 0.005
		r.CameraPitch -= d.Y * 0.005
		if r.CameraPitch < 0.1 { r.CameraPitch = 0.1 }
		if r.CameraPitch > 1.45 { r.CameraPitch = 1.45 }
	}
	wheel := rl.GetMouseWheelMove()
	if wheel != 0 {
		r.CameraDist -= wheel * 3
		if r.CameraDist < 15 { r.CameraDist = 15 }
		if r.CameraDist > 100 { r.CameraDist = 100 }
	}
	pan := float32(25.0) * dt
	if rl.IsKeyDown(rl.KeyW) { r.TargetZ -= pan }
	if rl.IsKeyDown(rl.KeyS) { r.TargetZ += pan }
	if rl.IsKeyDown(rl.KeyA) { r.TargetX -= pan }
	if rl.IsKeyDown(rl.KeyD) { r.TargetX += pan }
	r.rebuildCamera()
}

func (r *Renderer) DrawTerrain() {
	rl.DrawModel(r.TerrainModel, rl.NewVector3(0, 0, 0), 1.0, rl.White)
}

// DrawFoods - with particle emission for pollen
func (r *Renderer) DrawFoods(foods []*colony.FoodSource, t *terrain.Terrain) {
	r.frameCount++

	// Initialize nectar tracking
	if r.prevFoodNectar == nil || len(r.prevFoodNectar) != len(foods) {
		r.prevFoodNectar = make([]float64, len(foods))
		for i, f := range foods {
			r.prevFoodNectar[i] = f.Nectar
		}
	}

	for i, food := range foods {
		if !food.Active {
			// Check if this was just exhausted
			if r.prevFoodNectar[i] > 0 {
				wx, wz := float32(food.X), float32(food.Z)
				wy := t.HeightAt(wx, wz) + 1.0
				r.Particles.Emit(particles.Burst, wx, wy, wz, 20, rl.NewColor(255, 150, 50, 255))
			}
			r.prevFoodNectar[i] = 0
			continue
		}

		wx := float32(food.X)
		wz := float32(food.Z)
		wy := t.HeightAt(wx, wz) + 1.0

		baseSize := float32(0.8) + float32(food.Nectar)*1.4
		food.PulsePhase += 0.03
		pulse := float32(1.0 + 0.12*math.Sin(float64(food.PulsePhase)))
		size := baseSize * pulse
		pos := rl.NewVector3(wx, wy, wz)
		glowA := uint8(30 + int(food.Nectar*60))
		glowPos := rl.NewVector3(wx, t.HeightAt(wx, wz)+0.05, wz)

		switch food.ShapeType {
		case 0:
			rl.DrawCubeV(pos, rl.NewVector3(size, size, size), rl.NewColor(0, 255, 220, 220))
			rl.DrawCubeWiresV(pos, rl.NewVector3(size*1.05, size*1.05, size*1.05), rl.NewColor(0, 255, 180, 255))
			rl.DrawCircle3D(glowPos, size*2, rl.NewVector3(1, 0, 0), 90, rl.NewColor(0, 255, 200, glowA))
		case 1:
			drawPyramid(pos, size*1.3, rl.NewColor(255, 220, 50, 220))
			rl.DrawCircle3D(glowPos, size*2, rl.NewVector3(1, 0, 0), 90, rl.NewColor(255, 220, 50, glowA))
		case 2:
			drawOctahedron(pos, size*1.1, rl.NewColor(220, 80, 255, 220))
			rl.DrawCircle3D(glowPos, size*2, rl.NewVector3(1, 0, 0), 90, rl.NewColor(200, 50, 255, glowA))
		case 3:
			rl.DrawSphere(pos, size*0.7, rl.NewColor(60, 180, 255, 200))
			rl.DrawSphereWires(pos, size*0.75, 8, 8, rl.NewColor(100, 220, 255, 255))
			rl.DrawCircle3D(glowPos, size*2, rl.NewVector3(1, 0, 0), 90, rl.NewColor(60, 180, 255, glowA))
		case 4:
			rl.DrawCylinder(pos, size*0.5, size*0.5, size*0.9, 6, rl.NewColor(255, 100, 80, 220))
			rl.DrawCylinderWires(pos, size*0.55, size*0.55, size*0.95, 6, rl.NewColor(255, 140, 100, 255))
			rl.DrawCircle3D(glowPos, size*2, rl.NewVector3(1, 0, 0), 90, rl.NewColor(255, 100, 80, glowA))
		}

		// Nectar bar
		barH := float32(food.Nectar) * 2.5
		barPos := rl.NewVector3(wx+size*0.9, wy, wz)
		rl.DrawCube(barPos, 0.15, barH, 0.15, rl.NewColor(100, 255, 100, 200))

		// Ambient pollen particles near food
		if r.frameCount%10 == 0 && food.Nectar > 0.3 {
			r.Particles.Emit(particles.Pollen, wx, wy+0.5, wz, 1, rl.NewColor(255, 230, 100, 150))
		}

		// Detect nectar decrease -> sparkle
		if food.Nectar < r.prevFoodNectar[i]-0.005 {
			r.Particles.Emit(particles.Sparkle, wx, wy+0.3, wz, 3, rl.NewColor(255, 255, 200, 220))
		}
		r.prevFoodNectar[i] = food.Nectar
	}
}

func drawPyramid(pos rl.Vector3, size float32, color rl.Color) {
	top := rl.NewVector3(pos.X, pos.Y+size, pos.Z)
	half := size * 0.5
	c := [4]rl.Vector3{
		{X: pos.X - half, Y: pos.Y, Z: pos.Z - half},
		{X: pos.X + half, Y: pos.Y, Z: pos.Z - half},
		{X: pos.X + half, Y: pos.Y, Z: pos.Z + half},
		{X: pos.X - half, Y: pos.Y, Z: pos.Z + half},
	}
	wire := rl.NewColor(addB(color.R, 40), addB(color.G, 40), addB(color.B, 40), 255)
	for i := 0; i < 4; i++ {
		n := (i + 1) % 4
		rl.DrawTriangle3D(top, c[i], c[n], color)
		rl.DrawLine3D(top, c[i], wire)
		rl.DrawLine3D(c[i], c[n], wire)
	}
	rl.DrawTriangle3D(c[0], c[2], c[1], color)
	rl.DrawTriangle3D(c[0], c[3], c[2], color)
}

func drawOctahedron(pos rl.Vector3, size float32, color rl.Color) {
	top := rl.NewVector3(pos.X, pos.Y+size, pos.Z)
	bot := rl.NewVector3(pos.X, pos.Y-size, pos.Z)
	half := size * 0.6
	c := [4]rl.Vector3{
		{X: pos.X - half, Y: pos.Y, Z: pos.Z - half},
		{X: pos.X + half, Y: pos.Y, Z: pos.Z - half},
		{X: pos.X + half, Y: pos.Y, Z: pos.Z + half},
		{X: pos.X - half, Y: pos.Y, Z: pos.Z + half},
	}
	wire := rl.NewColor(addB(color.R, 60), addB(color.G, 60), addB(color.B, 60), 255)
	for i := 0; i < 4; i++ {
		n := (i + 1) % 4
		rl.DrawTriangle3D(top, c[i], c[n], color)
		rl.DrawTriangle3D(bot, c[n], c[i], color)
		rl.DrawLine3D(top, c[i], wire)
		rl.DrawLine3D(bot, c[i], wire)
		rl.DrawLine3D(c[i], c[n], wire)
	}
}

func addB(v uint8, add int) uint8 {
	r := int(v) + add
	if r > 255 { return 255 }
	return uint8(r)
}

// DrawBees renders using the procedural bee model with DrawModelEx
func (r *Renderer) DrawBees(bees []*colony.Bee, t *terrain.Terrain) {
	for _, bee := range bees {
		wx := float32(bee.X)
		wz := float32(bee.Z)
		wy := t.HeightAt(wx, wz) + 3.0

		wobX := float32(math.Sin(float64(bee.Wobble))) * 0.5
		wobZ := float32(math.Cos(float64(bee.Wobble*1.3))) * 0.5
		wy += float32(math.Sin(float64(bee.Wobble*2.0))) * 0.4

		pos := rl.NewVector3(wx+wobX, wy, wz+wobZ)

		// Face direction of travel
		dx := float32(bee.TargetX) - wx
		dz := float32(bee.TargetZ) - wz
		angle := float32(math.Atan2(float64(dz), float64(dx))) * (180.0 / math.Pi)

		// Tint based on role
		var tint rl.Color
		var trailColor rl.Color
		switch bee.Role {
		case colony.Employed:
			tint = rl.NewColor(255, 220, 80, 255)
			trailColor = rl.NewColor(255, 200, 0, 100)
		case colony.Onlooker:
			tint = rl.NewColor(120, 180, 255, 255)
			trailColor = rl.NewColor(80, 160, 255, 100)
		case colony.Scout:
			tint = rl.NewColor(255, 100, 100, 255)
			trailColor = rl.NewColor(255, 60, 60, 100)
		}

		// Draw the bee model
		scale := rl.NewVector3(2.0, 2.0, 2.0)
		rl.DrawModelEx(r.BeeAssets.BeeModel, pos, rl.NewVector3(0, 1, 0), angle, scale, tint)

		// Role glow
		glowAlpha := uint8(35)
		if bee.Role == colony.Scout {
			glowAlpha = 50
		}
		rl.DrawSphere(pos, 1.2, rl.NewColor(tint.R, tint.G, tint.B, glowAlpha))

		// Trail particles (every few frames)
		if r.frameCount%4 == 0 {
			r.Particles.Emit(particles.Trail, pos.X, pos.Y-0.3, pos.Z, 1, trailColor)
		}

		// Trail line to target
		if bee.Role == colony.Employed || bee.Role == colony.Onlooker {
			ty := t.HeightAt(float32(bee.TargetX), float32(bee.TargetZ)) + 1.0
			tp := rl.NewVector3(float32(bee.TargetX), ty, float32(bee.TargetZ))
			rl.DrawLine3D(pos, tp, rl.NewColor(tint.R, tint.G, tint.B, 30))
		}

		// Shadow
		shY := t.HeightAt(wx+wobX, wz+wobZ) + 0.05
		rl.DrawCircle3D(rl.NewVector3(wx+wobX, shY, wz+wobZ), 0.6, rl.NewVector3(1, 0, 0), 90, rl.NewColor(0, 0, 0, 30))
	}
}

// DrawParticles renders the particle system (call inside BeginMode3D)
func (r *Renderer) DrawParticles() {
	r.Particles.Draw(r.Camera)
}

// UpdateParticles advances particle simulation
func (r *Renderer) UpdateParticles(dt float32) {
	r.Particles.Update(dt)
}

func (r *Renderer) DrawHive(t *terrain.Terrain) {
	hy := t.HeightAt(0, 0) + 0.05
	pos := rl.NewVector3(0, hy, 0)
	rl.DrawCylinder(pos, 2.2, 1.6, 2.5, 8, rl.NewColor(200, 160, 60, 240))
	rl.DrawCylinderWires(pos, 2.2, 1.6, 2.5, 8, rl.NewColor(160, 120, 40, 255))
	roof := rl.NewVector3(0, hy+2.5, 0)
	rl.DrawCylinder(roof, 0.1, 2.5, 1.3, 8, rl.NewColor(180, 140, 40, 240))
	rl.DrawCylinderWires(roof, 0.1, 2.5, 1.3, 8, rl.NewColor(140, 100, 30, 255))
	ent := rl.NewVector3(1.8, hy+0.7, 0)
	rl.DrawCube(ent, 0.9, 0.7, 0.7, rl.NewColor(60, 40, 20, 255))
	rl.DrawSphere(rl.NewVector3(0, hy+1.5, 0), 3.5, rl.NewColor(255, 200, 50, 12))
}

func (r *Renderer) DrawHUD(c *colony.Colony, presetName string, phase string) {
	rl.DrawRectangle(10, 10, 310, 295, rl.NewColor(0, 0, 0, 180))
	rl.DrawRectangleLines(10, 10, 310, 295, rl.NewColor(255, 200, 0, 200))

	y := int32(22)
	sp := int32(26)
	rl.DrawText("BEE COLONY SIMULATOR", 20, y, 20, rl.Gold)
	y += sp + 8
	rl.DrawText(fmt.Sprintf("Preset: %s", presetName), 20, y, 16, rl.White)
	y += sp
	rl.DrawText(fmt.Sprintf("Generation: %d", c.Generation), 20, y, 16, rl.White)
	y += sp
	rl.DrawText(fmt.Sprintf("Phase: %s", phase), 20, y, 16, rl.LightGray)
	y += sp
	rl.DrawText(fmt.Sprintf("Best Fitness: %.4f", c.BestFitness), 20, y, 16, rl.NewColor(100, 255, 100, 255))
	y += sp
	rl.DrawText(fmt.Sprintf("Active Foods: %d", countActive(c.Foods)), 20, y, 16, rl.White)
	y += sp
	rl.DrawText(fmt.Sprintf("Exhausted: %d  Discovered: %d", c.FoodsExhausted, c.FoodsDiscovered), 20, y, 14, rl.LightGray)
	y += sp
	rl.DrawText(fmt.Sprintf("Particles: %d", r.Particles.ActiveCount()), 20, y, 14, rl.NewColor(180, 180, 180, 180))
	y += sp + 8

	rl.DrawCircle(30, y+5, 7, rl.NewColor(255, 200, 0, 255))
	rl.DrawText("Employed", 44, y, 14, rl.NewColor(255, 200, 0, 255))
	rl.DrawCircle(145, y+5, 7, rl.NewColor(80, 160, 255, 255))
	rl.DrawText("Onlooker", 159, y, 14, rl.NewColor(80, 160, 255, 255))
	rl.DrawCircle(260, y+5, 7, rl.NewColor(255, 60, 60, 255))
	rl.DrawText("Scout", 274, y, 14, rl.NewColor(255, 60, 60, 255))

	helpY := int32(rl.GetScreenHeight()) - 30
	rl.DrawRectangle(0, helpY-5, int32(rl.GetScreenWidth()), 35, rl.NewColor(0, 0, 0, 130))
	rl.DrawText("Right-drag: orbit | Scroll: zoom | WASD: pan | 1-5: presets | R: restart | ESC: menu",
		15, helpY, 14, rl.NewColor(200, 200, 200, 180))
}

func countActive(foods []*colony.FoodSource) int {
	n := 0
	for _, f := range foods {
		if f.Active { n++ }
	}
	return n
}
