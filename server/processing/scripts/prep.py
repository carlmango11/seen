#!/usr/bin/env python3

import os
import cv2
import sys

try:
    file = sys.argv[1]
    output_dir = sys.argv[2]
    sample_hz = int(sys.argv[3])

    os.mkdir(output_dir)

    vidcap = cv2.VideoCapture(file)

    total = int(vidcap.get(cv2.CAP_PROP_FRAME_COUNT))
    fps = vidcap.get(cv2.CAP_PROP_FPS)

    sample_every = int(fps / sample_hz)

    success, image = vidcap.read()
    count = 0

    while success:
        if count % sample_every == 0:
            cv2.imwrite("%s%d.jpg" % (output_dir, count), image)

        success,image = vidcap.read()
        count += 1

    sys.exit(0)
except Exception as e:
    sys.stderr.write(e)
    sys.exit(0) # exit with correct code so that the error message will get picked up
