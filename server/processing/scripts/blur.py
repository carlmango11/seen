#!/usr/bin/env python3

import json
import sys
import cv2
import dataclasses
import base64

autoblur = False
faceCascade = cv2.CascadeClassifier(cv2.data.haarcascades + "haarcascade_frontalface_default.xml")

in_file = sys.argv[1]
out_file = sys.argv[2]

try:
	guide_json = base64.b64decode(sys.argv[3])
	gs = json.loads(guide_json)
except IndexError:
	# No guide -> AutoBlur
	autoblur = True

@dataclasses.dataclass()
class Overlay:
	x: int
	y: int
	size: int

def autoblur(frame):
	# Detect faces in the image
	gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)

	faces = faceCascade.detectMultiScale(gray,
		scaleFactor=1.1,
		minNeighbors=5,
		minSize=(30, 30)
	)

	# Prob duplicating with find_and_blur
	for (x, y, w, h) in faces:
		sub_face = frame[y:y + h, x:x + w]
		# apply a gaussian blur on this new recangle image
		sub_face = cv2.GaussianBlur(sub_face,(23, 23), 30)
		# merge this blurry rectangle to our final image
		frame[y:y + sub_face.shape[0], x:x + sub_face.shape[1]] = sub_face

	return frame

def find_and_blur(x, y, size, img):
	roi_color = img[y:y + size, x:x + size]
	blur = cv2.GaussianBlur(roi_color, (101,101), 0)
	img[y:y + size, x:x + size] = blur

	return img


def find_overlays(guides, c):
	overlays = []

	for g in guides:
		for i, kf in enumerate(g["keyFrames"]):
			if c == kf["frameId"]:
				overlays.append(to_overlay(kf))

			if c == 0:
				break

			if kf["frameId"] > c:
				overlays.append(midOverlay(c, g["keyFrames"][i-1], g["keyFrames"][i]))
				break

	return overlays


def to_overlay(kf) -> Overlay:
	return Overlay(kf["x"], kf["y"], kf["size"])


def midOverlay(c, a, b):
	o = c - a["frameId"]
	scalar = o / (b["frameId"] - a["frameId"])

	moveX = (b["x"] - a["x"]) * scalar
	moveY = (b["y"] - a["y"]) * scalar
	moveSize = (b["size"] - a["size"]) * scalar

	return Overlay(int(a["x"] + moveX), int(a["y"] + moveY), int(a["size"] + moveSize))


def process_frame(c, img, guides):
	overlays = find_overlays(guides, c)

	for ol in overlays:
		img = find_and_blur(ol.x, ol.y, ol.size, img)

	return img


try:
	vid = cv2.VideoCapture(in_file)

	fps = vid.get(cv2.CAP_PROP_FPS)

	fourcc = cv2.VideoWriter_fourcc('m','p','4','v')
	out = cv2.VideoWriter(out_file, fourcc, fps, (int(vid.get(3)), int(vid.get(4))))

	c = 0
	while True:
		ret, img = vid.read()
		if not ret:
			break

		if autoblur:
			out.write(autoblur(img))
		else:
			out.write(process_frame(c, img, gs))
		c += 1

	vid.release()
	out.release()

	sys.exit(0)
except Exception as e:
	sys.stderr.write(e)
	sys.exit(0) # exit with correct code so that the error message will get picked up
