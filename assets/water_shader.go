//go:build ignore

//kage:unit pixels

package main

const Pi = 3.141592

// Uniform variables.
var Time float
var Offset vec2
var Colors [4]vec4

var ScreenHeight float

func colorRamp(t float) vec4 {
	if t < 0.25 {
		return mix(Colors[0], Colors[1], t/0.25)
	} else if t < 0.5 {
		return mix(Colors[1], Colors[2], (t-0.25)/0.25)
	} else if t < 0.75 {
		return mix(Colors[2], Colors[3], (t-0.5)/0.25)
	} else {
		return mix(Colors[3], Colors[0], (t-0.75)/0.25)
	}
}

func rotateV(v vec2, theta float) vec2 {
	c := cos(theta)
	s := sin(theta)
	return vec2(v.x*c-v.y*s, v.x*s+v.y*c)
}

func imageSrc0At01(at vec2) vec4 {
	origin0 := imageSrc0Origin()
	imgSize := imageSrc0Size()
	return imageSrc0UnsafeAt(mod(imgSize*at, imgSize) + origin0)
}

func imageSrc1At01(at vec2) vec4 {
	origin0 := imageSrc0Origin()
	imgSize := imageSrc1Size()
	return imageSrc1UnsafeAt(mod(imgSize*at, imgSize) + origin0)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	time := Time * 5
	_ = time

	pos := (dstPos.xy + Offset - imageDstOrigin()) / ScreenHeight

	scale1 := 2.0
	pos1 := pos * vec2(scale1, scale1*3)

	scale2 := 0.7
	pos2 := pos * vec2(scale2, scale2*3)

	rotV1 := pos1 - vec2(0.5, 0.5)
	rotV1 = rotateV(rotV1, rotV1.x+time*0.03)
	rotV1 += vec2(0.5, 0.5)

	rotV2 := pos2 - vec2(0.5, 0.5)
	rotV2 = rotateV(rotV2, rotV2.x-time*0.005)
	rotV2 += vec2(0.5, 0.5)

	waveV1 := pos1
	waveV1.y += cos(pos1.x*Pi-2*Pi) * sin(time*0.1)

	waveV2 := pos2
	waveV2.y += cos(pos2.x*Pi-2*Pi) * sin(time*-0.05)

	c2 := imageSrc1At01((waveV2*0.1 + rotV2*-0.6 + vec2(time*0.01, time*0.0004)))
	c1 := imageSrc0At01((waveV1*0.1 + rotV1*0.2*(0.5+c2.r*0.5) + vec2(time*0.0004, time*0.001)) * 0.3)
	_ = c2

	return colorRamp(mod(c1.r*0.6+time*0.01+c2.r*3, 1))
}
