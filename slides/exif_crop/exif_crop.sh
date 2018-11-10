#!/bin/bash -e

# -crop <width>x<height>{+-}<x>{+-}<y>{%}
#
# The width and height give the size of the image that remains after cropping,
# and x and y are offsets that give the location of the top left corner of the
# cropped image with respect to the original image. 
gm convert P1030735.JPG -crop 1000x1600+700+250 cropped_wrong.jpg

# 4000x3000 rotated 270
#   reverse the target height/width (1000x1600 -> 1600x1000)
#   4000-1600-250 => 2150
#   700
gm convert P1030735.JPG -crop 1600x1000+2150+700 cropped_right.jpg

