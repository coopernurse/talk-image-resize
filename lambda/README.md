# Create Thumbnail using Sharp JS Library 

The code here creates a resized copy of the image you upload to an s3 bucket. It triggers on an event in s3 after an upload to a specified bucket and adds the new resized image to another bucket/location in s3.

The example uses:
uploads-bucket (upload location), and creates a resized copy in another bucket (resized-bucket).

## Install

* ```npm install``` to get the pre-requisites

* then zip everything (including the node modules)

* Add the lambda function and then setup IAM policy to upload, and run lambda etc.

* Set ObjectCreated(All) for S3 in select Lambda (function uploaded)

* Upload the zip to Lambda and test

## Update Permissions

### Sample Cors:

```
<?xml version="1.0" encoding="UTF-8"?>
<CORSConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
    <CORSRule>
        <AllowedOrigin>*</AllowedOrigin>
        <AllowedMethod>GET</AllowedMethod>
    </CORSRule>
    <CORSRule>
        <AllowedOrigin>http://localhost:4200</AllowedOrigin>
        <AllowedMethod>GET</AllowedMethod>
        <AllowedMethod>PUT</AllowedMethod>
        <AllowedMethod>POST</AllowedMethod>
        <AllowedMethod>DELETE</AllowedMethod>
        <AllowedHeader>*</AllowedHeader>
    </CORSRule>
</CORSConfiguration>
```

### Sample Bucket Permissions:

```
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "AddPerm",
			"Effect": "Allow",
			"Principal": "*",
			"Action": "s3:GetObject",
			"Resource": "arn:aws:s3:::ecms/*"
		},
		{
			"Sid": "DelegateS3Access",
			"Effect": "Allow",
			"Principal": {
				"AWS": "arn:aws:iam::786462021999:root"
			},
			"Action": "s3:*",
			"Resource": [
				"arn:aws:s3:::ecms/*",
				"arn:aws:s3:::ecms"
			]
		}
	]
}
```

### Sample Resized Permissions:

```
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "AddPerm",
			"Effect": "Allow",
			"Principal": "*",
			"Action": "s3:GetObject",
			"Resource": "arn:aws:s3:::ecmsresized/*"
		},
		{
			"Sid": "DelegateS3Access",
			"Effect": "Allow",
			"Principal": {
				"AWS": "arn:aws:iam::786462021999:root"
			},
			"Action": "s3:*",
			"Resource": "arn:aws:s3:::ecmsresized/*"
		}
	]
}
```
