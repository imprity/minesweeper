//go:build ignore

//kage:unit pixels

package main

const Pi = 3.141592

// Uniform variables.
var Time float
var Cursor vec2

func ColorRamp(t float) vec4 {
	var colors [5]vec4
	colors[0] = vec4(0.8, 0.8, 0.9, 1)
	colors[1] = vec4(0.4, 0, 0.6, 1)
	colors[2] = vec4(0.4, 1, 0.6, 1)
	colors[3] = vec4(0.9, 0.4, 0.6, 1)
	colors[4] = vec4(0.8, 0.8, 0.9, 1)

	segment := (1.0 / 4.0)

	for i := 0; i < 4; i++ {
		limit := float(i+1) * segment
		if t < limit {
			t = (t - float(i)*segment) / segment
			return mix(colors[i], colors[i+1], t)
		}
	}

	return mix(colors[3], colors[4], (t-0.85)/segment)
}

func rotateV(v vec2, theta float) vec2 {
	c := cos(theta)
	s := sin(theta)
	return vec2(v.x*c-v.y*s, v.x*s+v.y*c)
}

func imageSrc0At01(at vec2) vec4 {
	origin0 := imageSrc0Origin()
	imgSize := imageSrc0Size()
	return imageSrc0At(mod(imgSize*at, imgSize) + origin0)
}

func imageSrc1At01(at vec2) vec4 {
	origin0 := imageSrc0Origin()
	imgSize := imageSrc1Size()
	return imageSrc1At(mod(imgSize*at, imgSize) + origin0)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	time := Time * 0.8
	_ = time

	pos := dstPos.xy / imageDstSize()
	pos *= (Cursor.x / 100)

	//pos.y += cos(pos.x * Pi - 2*Pi) * sin(time * 0.1)
	//pos.x += time * 0.01
	//pos.y -= time * 0.01

	rotV := pos - vec2(0.5, 0.5)
	rotV = rotateV(rotV, rotV.x+time*0.03)
	rotV += vec2(0.5, 0.5)

	waveV := pos
	waveV.y += cos(pos.x*Pi-2*Pi) * sin(time*0.1)

	c1 := imageSrc0At01((waveV*0.1 + rotV*0.2 + vec2(time*0.0004, time*0.001)) * 0.3)
	c2 := imageSrc1At01((waveV*0.1 + rotV*-0.6*(0.8+c1.r*0.2) + vec2(time*0.01, time*0.0004)))
	_ = c2

	return ColorRamp(mod(c1.r*0.6+time*0.01+c2.r*0.2, 1))
	//return ColorRamp(c2.r)
}

/*
func Fragment2(dstPos vec4, srcPos vec2, color vec4) vec4 {
	img0Size := imageSrc0Size()
	img0Origin := imageSrc0Origin()
	_ = img0Origin

	c0 := imageSrc0At(mod(srcPos*vec2(sin(Time*0.001)*0.001+0.9, 0.4)*0.3+vec2(-Time*0.1, Time*0.2), img0Size)*5 + img0Origin)
	_ = c0

	img1Size := imageSrc1Size()
	img1Origin := imageSrc1Origin()
	_ = img1Origin

	c1 := imageSrc1At(mod(srcPos*vec2(sin(Time*0.001)*0.1+0.7, 0.4)*0.2+vec2(Time*-0.2, Time*0.3)*100+c0.r*100, img1Size) + img0Origin)
	_ = c1

	a := mod(c0.r+Time*0.001+c1.r, 1)

	//return vec4(a,a,a,a)
	return ColorRamp(a)
}
*/
