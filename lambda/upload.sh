#!/bin/bash -e

func_name="image-resize"
role="arn:aws:iam::454738051317:role/apex_lambda_function"
memory="512"

# build zip
rm -f lambda.zip
docker run -v "$PWD":/var/task lambci/lambda:build-nodejs8.10 npm install
zip -r lambda.zip server.js node_modules

# check for existing lambda function
cmd="create"
set +e
aws lambda get-function --function-name $func_name > /dev/null 2>&1
if [ $? -eq 0 ]; then
    cmd="update"
    echo "update func"
fi
set -e

if [ "$cmd" = "create" ]; then
    aws lambda create-function --function-name "$func_name" \
        --runtime nodejs8.10 \
        --role "$role" \
        --handler "server.handler" \
        --timeout 60 \
        --memory-size "$memory" \
        --zip-file "fileb://lambda.zip"
else
    aws lambda update-function-code \
        --function-name "$func_name" \
        --zip-file "fileb://lambda.zip"
fi
