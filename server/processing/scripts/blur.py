import json
import cv2
import dataclasses


j = """
[{"id":0,"keyFrames":[{"index":2,"x":564,"y":266,"size":20,"frameId":18},{"index":8,"x":566,"y":274,"size":20,"frameId":72},{"index":17,"x":566,"y":293,"size":20,"frameId":153},{"index":20,"x":545,"y":320,"size":50,"frameId":180},{"index":21,"x":482,"y":322,"size":100,"frameId":189},{"index":22,"x":469,"y":357,"size":100,"frameId":198},{"index":23,"x":364,"y":502,"size":120,"frameId":207},{"index":24,"x":303,"y":622,"size":120,"frameId":216}],"colour":"red"}]
"""

gs = json.loads(j)


@dataclasses.dataclass()
class Overlay:
    x: int
    y: int
    size: int


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


out_path = '/Users/carl/Movies/output2.mp4'
img_path = '/Users/carl/Movies/run.mp4'
vid = cv2.VideoCapture(img_path)

fps = vid.get(cv2.CAP_PROP_FPS)

fourcc = cv2.VideoWriter_fourcc('m','p','4','v')
out = cv2.VideoWriter(out_path, fourcc, fps, (int(vid.get(3)), int(vid.get(4))))

c = 0
while True:
    ret, img = vid.read()
    if not ret:
        break

    out.write(process_frame(c, img, gs))
    c += 1

vid.release()
out.release()
print('finished')