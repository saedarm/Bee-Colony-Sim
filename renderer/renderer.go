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

	prevFoodNectar []float64
	prevFoodActive []bool
	frameCount     int

	ShowTrails   bool
	ShowFoodInfo bool
}

func New() *Renderer {
	r := &Renderer{
		CameraYaw: 0.8, CameraPitch: 0.55, CameraDist: 55.0,
		TargetX: 0, TargetY: 4, TargetZ: 0,
		LowColor: rl.NewColor(30, 130, 60, 255), MidColor: rl.NewColor(230, 200, 80, 255),
		HighColor: rl.NewColor(255, 80, 30, 255), WaterColor: rl.NewColor(25, 50, 90, 255),
		ShowTrails: true, ShowFoodInfo: false,
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
		Position: rl.NewVector3(r.TargetX+cosP*cosY*r.CameraDist, r.TargetY+sinP*r.CameraDist, r.TargetZ+cosP*sinY*r.CameraDist),
		Target:   rl.NewVector3(r.TargetX, r.TargetY, r.TargetZ),
		Up:       rl.NewVector3(0, 1, 0), Fovy: 45, Projection: rl.CameraPerspective,
	}
}

func (r *Renderer) BuildTerrainMesh(t *terrain.Terrain) {
	w, d := t.Width, t.Depth
	halfScale := t.ScaleXZ / 2.0
	triCount := (w - 1) * (d - 1) * 2
	vertCount := triCount * 3
	verts := make([]float32, vertCount*3)
	cols := make([]uint8, vertCount*4)
	norms := make([]float32, vertCount*3)
	vi, ci, ni := 0, 0, 0
	for x := 0; x < w-1; x++ {
		for z := 0; z < d-1; z++ {
			x0 := -halfScale + float32(x)/float32(w-1)*t.ScaleXZ
			x1 := -halfScale + float32(x+1)/float32(w-1)*t.ScaleXZ
			z0 := -halfScale + float32(z)/float32(d-1)*t.ScaleXZ
			z1 := -halfScale + float32(z+1)/float32(d-1)*t.ScaleXZ
			y00, y10 := t.Heights[x][z]*t.ScaleY, t.Heights[x+1][z]*t.ScaleY
			y01, y11 := t.Heights[x][z+1]*t.ScaleY, t.Heights[x+1][z+1]*t.ScaleY
			v0 := [3]float32{x0, y00, z0}
			v1 := [3]float32{x1, y10, z0}
			v2 := [3]float32{x0, y01, z1}
			v3 := [3]float32{x1, y10, z0}
			v4 := [3]float32{x1, y11, z1}
			v5 := [3]float32{x0, y01, z1}
			n1 := compNorm(v0, v1, v2)
			n2 := compNorm(v3, v4, v5)
			for ti, tri := range [2][3][3]float32{{v0, v1, v2}, {v3, v4, v5}} {
				nn := [2][3]float32{n1, n2}
				for _, v := range tri {
					verts[vi], verts[vi+1], verts[vi+2] = v[0], v[1], v[2]
					vi += 3
					h := v[1] / t.ScaleY
					c := r.heightColor(h)
					sh := float32(0.55 + 0.45*float64(nn[ti][1]))
					cols[ci] = cB(float32(c.R) * sh)
					cols[ci+1] = cB(float32(c.G) * sh)
					cols[ci+2] = cB(float32(c.B) * sh)
					cols[ci+3] = 255
					ci += 4
					norms[ni], norms[ni+1], norms[ni+2] = nn[ti][0], nn[ti][1], nn[ti][2]
					ni += 3
				}
			}
		}
	}
	mesh := rl.Mesh{VertexCount: int32(vertCount), TriangleCount: int32(triCount)}
	mesh.Vertices = &verts[0]
	mesh.Normals = &norms[0]
	mesh.Colors = &cols[0]
	rl.UploadMesh(&mesh, false)
	r.TerrainMesh = mesh
	r.TerrainModel = rl.LoadModelFromMesh(mesh)
}

func compNorm(v0, v1, v2 [3]float32) [3]float32 {
	ux, uy, uz := v1[0]-v0[0], v1[1]-v0[1], v1[2]-v0[2]
	vx, vy, vz := v2[0]-v0[0], v2[1]-v0[1], v2[2]-v0[2]
	nx, ny, nz := uy*vz-uz*vy, uz*vx-ux*vz, ux*vy-uy*vx
	l := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
	if l > 0 { return [3]float32{nx / l, ny / l, nz / l} }
	return [3]float32{0, 1, 0}
}

func cB(v float32) uint8 {
	if v < 0 { return 0 }
	if v > 255 { return 255 }
	return uint8(v)
}

func (r *Renderer) heightColor(h float32) rl.Color {
	if h < 0 { h = 0 }
	if h > 1 { h = 1 }
	if h < 0.25 { return lerpC(r.WaterColor, r.LowColor, h/0.25) }
	if h < 0.55 { return lerpC(r.LowColor, r.MidColor, (h-0.25)/0.30) }
	return lerpC(r.MidColor, r.HighColor, (h-0.55)/0.45)
}

func lerpC(a, b rl.Color, t float32) rl.Color {
	if t < 0 { t = 0 }
	if t > 1 { t = 1 }
	return rl.NewColor(uint8(float32(a.R)*(1-t)+float32(b.R)*t), uint8(float32(a.G)*(1-t)+float32(b.G)*t), uint8(float32(a.B)*(1-t)+float32(b.B)*t), 255)
}

func (r *Renderer) UpdateCamera(dt float32) {
	if rl.IsMouseButtonDown(rl.MouseRightButton) {
		d := rl.GetMouseDelta()
		r.CameraYaw += d.X * 0.005
		r.CameraPitch -= d.Y * 0.005
		if r.CameraPitch < 0.1 { r.CameraPitch = 0.1 }
		if r.CameraPitch > 1.45 { r.CameraPitch = 1.45 }
	}
	w := rl.GetMouseWheelMove()
	if w != 0 {
		r.CameraDist -= w * 3
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

func (r *Renderer) DrawTerrain() { rl.DrawModel(r.TerrainModel, rl.NewVector3(0, 0, 0), 1.0, rl.White) }

func (r *Renderer) DrawFoods(foods []*colony.FoodSource, t *terrain.Terrain, abandonLimit int) {
	r.frameCount++
	if r.prevFoodNectar == nil || len(r.prevFoodNectar) != len(foods) {
		r.prevFoodNectar = make([]float64, len(foods))
		r.prevFoodActive = make([]bool, len(foods))
		for i, f := range foods {
			r.prevFoodNectar[i] = f.Nectar
			r.prevFoodActive[i] = f.Active
		}
	}

	for i, food := range foods {
		// Detect exhaustion for burst particles
		if !food.Active && r.prevFoodActive[i] {
			wx, wz := float32(food.X), float32(food.Z)
			wy := t.HeightAt(wx, wz) + 1.5
			r.Particles.Emit(particles.Burst, wx, wy, wz, 25, rl.NewColor(255, 100, 30, 255))
		}
		r.prevFoodActive[i] = food.Active
		if !food.Active {
			r.prevFoodNectar[i] = 0
			continue
		}

		wx, wz := float32(food.X), float32(food.Z)
		wy := t.HeightAt(wx, wz) + 1.0

		// Size scales with fitness quality (normalized roughly)
		fitScale := float32(0.5 + 0.5*(food.Fitness/100.0))
		if fitScale > 1.5 { fitScale = 1.5 }
		if fitScale < 0.4 { fitScale = 0.4 }
		baseSize := (float32(0.6) + float32(food.Nectar)*1.0) * fitScale
		food.PulsePhase += 0.03
		pulse := float32(1.0 + 0.1*math.Sin(float64(food.PulsePhase)))
		size := baseSize * pulse

		// Flash when being worked
		flashMul := uint8(0)
		if food.BeingWorked {
			flashMul = 30
		}

		pos := rl.NewVector3(wx, wy, wz)
		glowPos := rl.NewVector3(wx, t.HeightAt(wx, wz)+0.05, wz)
		glowA := uint8(25 + int(food.Nectar*50))

		switch food.ShapeType {
		case 0:
			rl.DrawCubeV(pos, rl.NewVector3(size, size, size), rl.NewColor(0+flashMul, 255, 220, 220))
			rl.DrawCubeWiresV(pos, rl.NewVector3(size*1.05, size*1.05, size*1.05), rl.NewColor(0, 255, 180, 255))
			rl.DrawCircle3D(glowPos, size*2, rl.NewVector3(1, 0, 0), 90, rl.NewColor(0, 255, 200, glowA))
		case 1:
			drawPyramid(pos, size*1.3, rl.NewColor(255, 220+flashMul, 50, 220))
			rl.DrawCircle3D(glowPos, size*2, rl.NewVector3(1, 0, 0), 90, rl.NewColor(255, 220, 50, glowA))
		case 2:
			drawOctahedron(pos, size*1.1, rl.NewColor(220+flashMul, 80, 255, 220))
			rl.DrawCircle3D(glowPos, size*2, rl.NewVector3(1, 0, 0), 90, rl.NewColor(200, 50, 255, glowA))
		case 3:
			rl.DrawSphere(pos, size*0.7, rl.NewColor(60+flashMul, 180, 255, 200))
			rl.DrawSphereWires(pos, size*0.75, 8, 8, rl.NewColor(100, 220, 255, 255))
			rl.DrawCircle3D(glowPos, size*2, rl.NewVector3(1, 0, 0), 90, rl.NewColor(60, 180, 255, glowA))
		case 4:
			rl.DrawCylinder(pos, size*0.5, size*0.5, size*0.9, 6, rl.NewColor(255, 100+flashMul, 80, 220))
			rl.DrawCylinderWires(pos, size*0.55, size*0.55, size*0.95, 6, rl.NewColor(255, 140, 100, 255))
			rl.DrawCircle3D(glowPos, size*2, rl.NewVector3(1, 0, 0), 90, rl.NewColor(255, 100, 80, glowA))
		}

		// Nectar bar
		barH := float32(food.Nectar) * 2.5
		barPos := rl.NewVector3(wx+size*0.9, wy, wz)
		barColor := rl.NewColor(100, 255, 100, 200)
		if food.Nectar < 0.3 { barColor = rl.NewColor(255, 200, 50, 200) }
		if food.Nectar < 0.1 { barColor = rl.NewColor(255, 60, 60, 200) }
		rl.DrawCube(barPos, 0.15, barH, 0.15, barColor)

		// === DANGER RING: shows how close to abandonment ===
		dangerRatio := colony.AbandonRatio(food, abandonLimit)
		if dangerRatio > 0.3 {
			ringAlpha := uint8(dangerRatio * 180)
			ringR := size*1.8 + dangerRatio*1.5
			ringColor := rl.NewColor(255, uint8(200*(1-dangerRatio)), 0, ringAlpha)
			rl.DrawCircle3D(glowPos, ringR, rl.NewVector3(1, 0, 0), 90, ringColor)
			if dangerRatio > 0.7 {
				// Second pulsing ring
				pulse2 := float32(1.0 + 0.3*math.Sin(float64(food.PulsePhase*3)))
				rl.DrawCircle3D(glowPos, ringR*pulse2, rl.NewVector3(1, 0, 0), 90, rl.NewColor(255, 30, 0, uint8(ringAlpha/2)))
			}
		}

		// === ONLOOKER POPULARITY indicator ===
		if food.OnlookerCount > 0 {
			popSize := float32(food.OnlookerCount) * 0.3
			popPos := rl.NewVector3(wx, wy+size+0.5, wz)
			rl.DrawSphere(popPos, 0.15+popSize*0.1, rl.NewColor(80, 160, 255, uint8(100+food.OnlookerCount*20)))
		}

		// Ambient pollen
		if r.frameCount%15 == 0 && food.Nectar > 0.4 {
			r.Particles.Emit(particles.Pollen, wx, wy+0.5, wz, 1, rl.NewColor(255, 230, 100, 150))
		}

		// Sparkle on nectar collection
		if food.Nectar < r.prevFoodNectar[i]-0.002 {
			r.Particles.Emit(particles.Sparkle, wx, wy+0.3, wz, 4, rl.NewColor(255, 255, 200, 230))
		}

		// Discovery glow
		if food.JustFound {
			r.Particles.Emit(particles.Sparkle, wx, wy, wz, 12, rl.NewColor(100, 255, 100, 255))
		}

		r.prevFoodNectar[i] = food.Nectar
	}
}

func drawPyramid(pos rl.Vector3, size float32, color rl.Color) {
	top := rl.NewVector3(pos.X, pos.Y+size, pos.Z)
	h := size * 0.5
	c := [4]rl.Vector3{{X: pos.X - h, Y: pos.Y, Z: pos.Z - h}, {X: pos.X + h, Y: pos.Y, Z: pos.Z - h}, {X: pos.X + h, Y: pos.Y, Z: pos.Z + h}, {X: pos.X - h, Y: pos.Y, Z: pos.Z + h}}
	w := rl.NewColor(aB(color.R, 40), aB(color.G, 40), aB(color.B, 40), 255)
	for i := 0; i < 4; i++ {
		n := (i + 1) % 4
		rl.DrawTriangle3D(top, c[i], c[n], color)
		rl.DrawLine3D(top, c[i], w)
		rl.DrawLine3D(c[i], c[n], w)
	}
	rl.DrawTriangle3D(c[0], c[2], c[1], color)
	rl.DrawTriangle3D(c[0], c[3], c[2], color)
}

func drawOctahedron(pos rl.Vector3, size float32, color rl.Color) {
	top := rl.NewVector3(pos.X, pos.Y+size, pos.Z)
	bot := rl.NewVector3(pos.X, pos.Y-size, pos.Z)
	h := size * 0.6
	c := [4]rl.Vector3{{X: pos.X - h, Y: pos.Y, Z: pos.Z - h}, {X: pos.X + h, Y: pos.Y, Z: pos.Z - h}, {X: pos.X + h, Y: pos.Y, Z: pos.Z + h}, {X: pos.X - h, Y: pos.Y, Z: pos.Z + h}}
	w := rl.NewColor(aB(color.R, 60), aB(color.G, 60), aB(color.B, 60), 255)
	for i := 0; i < 4; i++ {
		n := (i + 1) % 4
		rl.DrawTriangle3D(top, c[i], c[n], color)
		rl.DrawTriangle3D(bot, c[n], c[i], color)
		rl.DrawLine3D(top, c[i], w)
		rl.DrawLine3D(bot, c[i], w)
		rl.DrawLine3D(c[i], c[n], w)
	}
}

func aB(v uint8, add int) uint8 { r := int(v) + add; if r > 255 { return 255 }; return uint8(r) }

func (r *Renderer) DrawBees(bees []*colony.Bee, t *terrain.Terrain) {
	for _, bee := range bees {
		wx, wz := float32(bee.X), float32(bee.Z)
		wy := t.HeightAt(wx, wz) + 3.0
		wobX := float32(math.Sin(float64(bee.Wobble))) * 0.5
		wobZ := float32(math.Cos(float64(bee.Wobble*1.3))) * 0.5
		wy += float32(math.Sin(float64(bee.Wobble*2.0))) * 0.4
		pos := rl.NewVector3(wx+wobX, wy, wz+wobZ)

		dx := float32(bee.TargetX) - wx
		dz := float32(bee.TargetZ) - wz
		angle := float32(math.Atan2(float64(dz), float64(dx))) * (180.0 / math.Pi)

		var tint, trailColor rl.Color
		switch bee.Role {
		case colony.Employed:
			tint = rl.NewColor(255, 220, 80, 255)
			trailColor = rl.NewColor(255, 200, 0, 100)
		case colony.Onlooker:
			tint = rl.NewColor(120, 180, 255, 255)
			trailColor = rl.NewColor(80, 160, 255, 100)
		case colony.Scout:
			tint = rl.NewColor(255, 100, 100, 255)
			trailColor = rl.NewColor(255, 60, 60, 150)
		}

		scale := rl.NewVector3(2.0, 2.0, 2.0)
		rl.DrawModelEx(r.BeeAssets.BeeModel, pos, rl.NewVector3(0, 1, 0), angle, scale, tint)

		// Glow
		ga := uint8(30)
		if bee.Role == colony.Scout { ga = 55 }
		rl.DrawSphere(pos, 1.2, rl.NewColor(tint.R, tint.G, tint.B, ga))

		// Scout launch burst
		if bee.ScoutLaunch {
			r.Particles.Emit(particles.ScoutTrail, pos.X, pos.Y, pos.Z, 8, rl.NewColor(255, 80, 80, 255))
		}

		// Trail particles
		if r.frameCount%5 == 0 {
			r.Particles.Emit(particles.Trail, pos.X, pos.Y-0.3, pos.Z, 1, trailColor)
		}

		// Scout leaves brighter trail
		if bee.Role == colony.Scout && r.frameCount%2 == 0 {
			r.Particles.Emit(particles.ScoutTrail, pos.X, pos.Y, pos.Z, 1, rl.NewColor(255, 100, 60, 200))
		}

		// Trail line to food target
		if r.ShowTrails && (bee.Role == colony.Employed || bee.Role == colony.Onlooker) && bee.FoodIdx >= 0 {
			ty := t.HeightAt(float32(bee.TargetX), float32(bee.TargetZ)) + 1.0
			tp := rl.NewVector3(float32(bee.TargetX), ty, float32(bee.TargetZ))
			rl.DrawLine3D(pos, tp, rl.NewColor(tint.R, tint.G, tint.B, 35))
		}

		// Arrival sparkle
		if bee.JustArrived && bee.Role == colony.Employed {
			r.Particles.Emit(particles.Sparkle, pos.X, pos.Y, pos.Z, 5, rl.NewColor(255, 220, 100, 200))
		}

		// Shadow
		shY := t.HeightAt(wx+wobX, wz+wobZ) + 0.05
		rl.DrawCircle3D(rl.NewVector3(wx+wobX, shY, wz+wobZ), 0.5, rl.NewVector3(1, 0, 0), 90, rl.NewColor(0, 0, 0, 25))
	}
}

// DrawScoutPaths renders animated arcs for recent scout discoveries
func (r *Renderer) DrawScoutPaths(events []colony.ScoutEvent, t *terrain.Terrain) {
	for _, ev := range events {
		alpha := uint8(255 * (1 - ev.Age/3.0))
		if alpha < 10 { continue }

		fromY := t.HeightAt(float32(ev.FromX), float32(ev.FromZ)) + 2
		toY := t.HeightAt(float32(ev.ToX), float32(ev.ToZ)) + 2

		// Draw arc path
		segments := 15
		for s := 0; s < segments; s++ {
			t1 := float32(s) / float32(segments)
			t2 := float32(s+1) / float32(segments)

			x1 := float32(ev.FromX)*(1-t1) + float32(ev.ToX)*t1
			z1 := float32(ev.FromZ)*(1-t1) + float32(ev.ToZ)*t1
			y1 := fromY*(1-t1) + toY*t1 + float32(math.Sin(float64(t1)*math.Pi))*8 // arc height

			x2 := float32(ev.FromX)*(1-t2) + float32(ev.ToX)*t2
			z2 := float32(ev.FromZ)*(1-t2) + float32(ev.ToZ)*t2
			y2 := fromY*(1-t2) + toY*t2 + float32(math.Sin(float64(t2)*math.Pi))*8

			rl.DrawLine3D(
				rl.NewVector3(x1, y1, z1),
				rl.NewVector3(x2, y2, z2),
				rl.NewColor(255, 80, 40, alpha),
			)
		}
	}
}

// DrawFoodInfoOverlay renders per-food labels when toggled on
func (r *Renderer) DrawFoodInfoOverlay(foods []*colony.FoodSource, t *terrain.Terrain, abandonLimit int) {
	if !r.ShowFoodInfo { return }
	for i, food := range foods {
		if !food.Active { continue }
		wx, wz := float32(food.X), float32(food.Z)
		wy := t.HeightAt(wx, wz) + 4.0

		// Project 3D to 2D
		screenPos := rl.GetWorldToScreen(rl.NewVector3(wx, wy, wz), r.Camera)
		sx, sy := int32(screenPos.X), int32(screenPos.Y)

		// Background
		rl.DrawRectangle(sx-50, sy-10, 100, 50, rl.NewColor(0, 0, 0, 160))

		rl.DrawText(fmt.Sprintf("#%d F:%.1f", i, food.Fitness), sx-45, sy-6, 10, rl.White)
		rl.DrawText(fmt.Sprintf("N:%.0f%% T:%d/%d", food.Nectar*100, food.Trials, abandonLimit), sx-45, sy+8, 10, rl.LightGray)
		rl.DrawText(fmt.Sprintf("Pop:%d", food.OnlookerCount), sx-45, sy+22, 10, rl.NewColor(80, 160, 255, 255))
	}
}

func (r *Renderer) DrawParticles() { r.Particles.Draw(r.Camera) }
func (r *Renderer) UpdateParticles(dt float32) { r.Particles.Update(dt) }

func (r *Renderer) DrawHive(t *terrain.Terrain) {
	hy := t.HeightAt(0, 0) + 0.05
	pos := rl.NewVector3(0, hy, 0)
	rl.DrawCylinder(pos, 2.2, 1.6, 2.5, 8, rl.NewColor(200, 160, 60, 240))
	rl.DrawCylinderWires(pos, 2.2, 1.6, 2.5, 8, rl.NewColor(160, 120, 40, 255))
	roof := rl.NewVector3(0, hy+2.5, 0)
	rl.DrawCylinder(roof, 0.1, 2.5, 1.3, 8, rl.NewColor(180, 140, 40, 240))
	rl.DrawCylinderWires(roof, 0.1, 2.5, 1.3, 8, rl.NewColor(140, 100, 30, 255))
	rl.DrawCube(rl.NewVector3(1.8, hy+0.7, 0), 0.9, 0.7, 0.7, rl.NewColor(60, 40, 20, 255))
	rl.DrawSphere(rl.NewVector3(0, hy+1.5, 0), 3.5, rl.NewColor(255, 200, 50, 12))
}

func (r *Renderer) DrawHUD(c *colony.Colony, presetName, phase string, paused bool, speed float32, tickProgress float32) {
	rl.DrawRectangle(10, 10, 320, 320, rl.NewColor(0, 0, 0, 180))
	rl.DrawRectangleLines(10, 10, 320, 320, rl.NewColor(255, 200, 0, 200))
	y := int32(22)
	sp := int32(24)
	rl.DrawText("BEE COLONY SIMULATOR", 20, y, 20, rl.Gold)
	y += sp + 6
	rl.DrawText(fmt.Sprintf("Preset: %s", presetName), 20, y, 15, rl.White)
	y += sp
	rl.DrawText(fmt.Sprintf("Generation: %d", c.Generation), 20, y, 15, rl.White)
	y += sp
	rl.DrawText(fmt.Sprintf("Phase: %s", phase), 20, y, 15, rl.LightGray)
	y += sp
	rl.DrawText(fmt.Sprintf("Best Fitness: %.2f", c.BestFitness), 20, y, 15, rl.NewColor(100, 255, 100, 255))
	y += sp
	rl.DrawText(fmt.Sprintf("Active: %d  Exhausted: %d  Discovered: %d", countActive(c.Foods), c.FoodsExhausted, c.FoodsDiscovered), 20, y, 13, rl.LightGray)
	y += sp
	rl.DrawText(fmt.Sprintf("Particles: %d", r.Particles.ActiveCount()), 20, y, 13, rl.NewColor(160, 160, 160, 180))
	y += sp

	// Speed and pause
	speedTxt := fmt.Sprintf("Speed: %.1fx", speed)
	if paused { speedTxt = "PAUSED" }
	pauseColor := rl.White
	if paused { pauseColor = rl.NewColor(255, 100, 100, 255) }
	rl.DrawText(speedTxt, 20, y, 16, pauseColor)
	y += sp

	// Tick progress bar
	rl.DrawRectangle(20, y, 280, 8, rl.NewColor(40, 40, 40, 200))
	barW := int32(280 * tickProgress)
	if barW > 280 { barW = 280 }
	rl.DrawRectangle(20, y, barW, 8, rl.NewColor(255, 200, 0, 180))
	y += 16

	// Legend
	rl.DrawCircle(30, y+5, 6, rl.NewColor(255, 200, 0, 255))
	rl.DrawText("Employed", 42, y, 13, rl.NewColor(255, 200, 0, 255))
	rl.DrawCircle(140, y+5, 6, rl.NewColor(80, 160, 255, 255))
	rl.DrawText("Onlooker", 152, y, 13, rl.NewColor(80, 160, 255, 255))
	rl.DrawCircle(250, y+5, 6, rl.NewColor(255, 60, 60, 255))
	rl.DrawText("Scout", 262, y, 13, rl.NewColor(255, 60, 60, 255))

	// Controls
	helpY := int32(rl.GetScreenHeight()) - 30
	rl.DrawRectangle(0, helpY-5, int32(rl.GetScreenWidth()), 35, rl.NewColor(0, 0, 0, 140))
	rl.DrawText("SPACE:pause  +/-:speed  T:trails  I:food info  Right-drag:orbit  Scroll:zoom  WASD:pan  1-5:presets  R:restart",
		15, helpY, 13, rl.NewColor(200, 200, 200, 190))
}

func countActive(foods []*colony.FoodSource) int {
	n := 0
	for _, f := range foods { if f.Active { n++ } }
	return n
}
