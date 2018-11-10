
// content of index.js
const fs = require('fs');
const http = require('http');
const AWS = require('aws-sdk');
const sharp = require('sharp');

const downloadUrlTo = (url, destFile, callback) => {
    var file = fs.createWriteStream(destFile);
    var request = http.get(url, function(response) {
        if (response.statusCode !== 200) {
            return callback('Response status was ' + response.statusCode);
        }
        
        response.pipe(file);
        file.on('finish', function() {
            file.close(callback);
        });
    });
};

const resizeImage = (inFile, outFile, size, callback) => {
    sharp(inFile)
        .resize(size)
        .toFile(outFile, (err, info) => {
            callback(err);
        });
};

const uploadToS3 = (file, s3bucket, s3key, callback) => {
    const s3 = new AWS.S3();
    fs.readFile(file, function (err, data) {
        if (err) {
            return callback(err);
        }
        const params = {Bucket: s3bucket, Key: s3key, Body: data, ACL: "public-read", ContentType: "image/jpeg"};
        s3.putObject(params, function(err, data) {
            if (err) {
                return callback(err);
            }
            console.log("Uploaded " + file + " to: s3://" + s3bucket + s3key);
            callback(null, data);
        });
    });
};

////////////////////////////////////////////////
// http //
//////////

const errResp = (response, err, reason) => {
    const msg = "Error: " + reason + " err=" + err;
    console.log(msg);
    response.end(msg);
}

const readBody = (req, callback) => {
    var data = "";

    req.on('data', function (d) {
        data += d;
    });
    
    req.on('end', function () {
        callback(data);
    });
};

const requestHandler = (request, response) => {
    readBody(request, function(data) {
        let input = JSON.parse(data);
        handleReq(input, function(err, output) {
            if (err) {
                errResp(response, err, "downloadUrl failed for: " + url);
            }
            else {
                let s3url = output.resizeUrl;
                console.log("Generated: " + s3url);
                response.end(JSON.stringify(output));
            }
        });
    });
};

const handleReq = function(input, callback) {
    let url = input.url;
    let size = input.height;
    
    let fname = url.substring(url.lastIndexOf("/")+1).replace(/ /g, "_");
    let outname = new Date().getTime() + "_" + Math.random().toString(36).substring(7) + ".jpg";
    let tmpFile = "/tmp/" + fname;
    let outFile = "/tmp/" + outname;
    let region = "us-west-2";
    let s3bucket = "bitmech-west2";
    let s3key = "resized/" + outname;
    
    downloadUrlTo(url, tmpFile, function(err) {
        if (err) {
            callback(err);
        }
        else {
            let startTime = new Date().getTime();
            console.log("Resizing " + tmpFile + " to size: " + size);
            resizeImage(tmpFile, outFile, size, function(err) {
                fs.unlink(tmpFile, function(err) {
                    if (err) { 
                        console.log("Unable to delete: " + tmpFile + " - " + err);
                    }
                });
                if (err) {
                    callback(err);
                }
                else {
                    let elapsed = new Date().getTime() - startTime;
                    console.log("Uploading to: s3://" + s3bucket + "/" + s3key);
                    uploadToS3(outFile, s3bucket, s3key, function(err, data) {
                        fs.unlink(outFile, function(err) {
                            if (err) { 
                                console.log("Unable to delete: " + outFile + " - " + err);
                            }
                        });
                        if (err) {
                            callback(err);
                        }
                        else {
                            let resizeUrl = "https://s3-" + region + ".amazonaws.com/" + s3bucket + "/" + s3key;
                            console.log("Resized to: " + resizeUrl);
                            callback(null, {
                                "elapsedMillis": elapsed,
                                "maxRSSKB": 0,
                                "impl": "vips-sharp",
                                "origUrl": url,
                                "resizeUrl": resizeUrl
                            });
                        }
                    });
                }
            });
        }
    });
};

////////////////////////////////////////////////
// main //
//////////

if (process.env.LAMBDA_TASK_ROOT) {
    // in aws lambda - export handler
    exports.handler = function(event, context, callback) {
        handleReq(event, callback);
    };
}
else {
    // run web server - not in aws lambda
    const port = 3000;
    const server = http.createServer(requestHandler);
    server.listen(port, (err) => {
        if (err) {
            return console.log('something bad happened', err);
        }
        console.log(`server is listening on ${port}`);
    });
}
