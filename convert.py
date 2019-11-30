import cv2
import sys

file = sys.argv[1]
sample_hz = int(sys.argv[2])
print(file, sample_hz)

vidcap = cv2.VideoCapture(file)

total = int(vidcap.get(cv2.CAP_PROP_FRAME_COUNT))
fps = vidcap.get(cv2.CAP_PROP_FPS)

sample_every = int(fps / sample_hz)
print(total, fps, sample_every)

success, image = vidcap.read()
count = 0

while success:
    if count % sample_every == 0:
        cv2.imwrite("out/%d.jpg" % count, image)

    success,image = vidcap.read()
    count += 1