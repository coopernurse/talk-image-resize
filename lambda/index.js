//CreateThumbnail

// dependencies
var async = require('async');
var AWS = require('aws-sdk');
var sharp = require('sharp');
var util = require('util');

// constants
var MAX_WIDTH  = 100;
var MAX_HEIGHT = 100;

// get reference to S3 client 
var s3 = new AWS.S3();
 
exports.handler = function(event, context) {
	// Read options from the event.
	console.log("Reading options from event:\n", util.inspect(event, {depth: 5}));
	var srcBucket = event.Records[0].s3.bucket.name;
	// Object key may have spaces or unicode non-ASCII characters.
    var srcKey    =  decodeURIComponent(event.Records[0].s3.object.key.replace(/\+/g, " "));  
	var dstBucket = srcBucket.replace(/uploads/g, "resized");
	var dstKey    = "resized-" + srcKey;

	// Sanity check: validate that source and destination are different buckets.
	if (srcBucket == dstBucket) {
		console.error("Destination bucket must not match source bucket.");
		return;
	}

	// Infer the image type.
	var typeMatch = srcKey.match(/\.([^.]*)$/);
	if (!typeMatch) {
		console.error('unable to infer image type for key ' + srcKey);
		return;
	}
	var imageType = typeMatch[1];
	if (imageType != "jpg" && imageType != "png" && imageType != "jpeg") {
		console.log('skipping non-image ' + srcKey);
		return;
	}

	// Download the image from S3, transform, and upload to a different S3 bucket.
	async.waterfall([
		function download(next) {
			// Download the image from S3 into a buffer.
			s3.getObject({
					Bucket: srcBucket,
					Key: srcKey
				},
				next);
		},
		function transform(response, next) {
			sharp(response.Body)
				.resize(MAX_WIDTH, MAX_HEIGHT, {
					kernel: sharp.kernel.nearest
				})
				.crop(sharp.strategy.attention)
				//.crop(sharp.strategy.entropy)
				// .background('white')
				// .embed()
				.toFormat(imageType)
				.toBuffer()
				.then(function(outputBuffer) {
				  // outputBuffer contains JPEG image data no wider than 200 pixels and no higher
				  // than 200 pixels regardless of the inputBuffer image dimensions
				  next(null, response.ContentType, outputBuffer);
				});			
		},
		function upload(contentType, data, next) {
			// Stream the transformed image to a different S3 bucket.
			s3.putObject({
					Bucket: dstBucket,
					Key: dstKey,
					Body: data,
					ContentType: contentType
				},
				next);
			}
		], function (err) {
			if (err) {
				console.error(
					'Unable to resize ' + srcBucket + '/' + srcKey +
					' and upload to ' + dstBucket + '/' + dstKey +
					' due to an error: ' + err
				);
			} else {
				console.log(
					'Successfully resized ' + srcBucket + '/' + srcKey +
					' and uploaded to ' + dstBucket + '/' + dstKey
				);
			}

			context.done();
		}
	);
};