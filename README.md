# image resizing

Slides from SeaGL 2018

## viewing slides

Open `slides/index.html` in a web browser.

## running demo

Make sure you have docker installed.

```bash
# build the images
cd app
./build.sh

# run using swarm mode
docker stack deploy -c docker-compose.yml resize

# or run with docker compose
docker-compose up -d
```

Go to: http://127.0.0.1:1323/

## deploying the vips/sharp lambda

I didn't make this code particularly modular and did dumb
things like hardcode the bucket name.  Sorry.  But if you
want to hack on it and change those strings you can 
try deploying it to your AWS account.

```bash
# Export your AWS creds or AWS_PROFILE env var
cd lambda

# edit upload.sh and set your IAM role (needs lambda execute privs)
# sorry, I didn't author a cloudformation template for this..

# run upload.sh to build and create the lambda function
./upload.sh

# invoke the function with some test images
pip3 install --user boto3
./invoke.py
```
