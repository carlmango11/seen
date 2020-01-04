#!/usr/bin/env python3

# import cv2
import sys

sys.stderr.write("omgomg")
sys.exit(0)

try:
    storage_dir = sys.argv[1]
    id = sys.argv[2]
    sample_hz = int(sys.argv[3])
    print(storage_dir, id, sample_hz)

    file = storage_dir + "incoming/" + id

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

    sys.exit(0)
except Exception as e:
    sys.stderr.write(e)
    sys.exit(1)