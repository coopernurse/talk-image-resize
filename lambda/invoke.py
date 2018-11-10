#!/usr/bin/env python3

import boto3
import json

orig_urls = [
    "http://bitmech-west2.s3.amazonaws.com/resize-images/1_Wild_Turkey.jpg",
    "http://bitmech-west2.s3.amazonaws.com/resize-images/ContinentalDC-10-30.jpg",
    "http://bitmech-west2.s3.amazonaws.com/resize-images/P1030659.JPG"
]

lam = boto3.client("lambda")
for url in orig_urls:
    d = {"url": url, "height": 300}
    resp = lam.invoke(FunctionName="image-resize",
                      InvocationType="RequestResponse",
                      Payload=json.dumps(d))
    out = json.loads(resp["Payload"].read())
    print("%s to %s (%d ms)" % (url, out["resizeUrl"], out["elapsedMillis"]))
    
