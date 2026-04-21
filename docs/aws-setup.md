# AWS S3 Setup Guide

This document is intended for the cloud/DevOps engineer responsible for provisioning the AWS infrastructure that `g-nice-api` relies on for media storage.

---

## Overview

The API uploads user-generated media (profile photos, post images/videos) to a single **S3 bucket** (`g-nice-media`, region `af-south-1`). Media is organised by key prefix:

| Prefix | Usage |
|---|---|
| `posts/{userID}/{uuid}.{ext}` | Post images and videos |
| `avatars/{userID}/{uuid}.{ext}` | User profile pictures |

---

## Step 1 — Create the S3 Bucket

1. Open the [S3 Console](https://s3.console.aws.amazon.com/).
2. Click **Create bucket**.
3. **Bucket name**: `g-nice-media`
4. **Region**: `af-south-1` (Africa — Cape Town)
5. **Object Ownership**: `ACLs disabled` (recommended — use a bucket policy for public read instead of per-object ACLs).
6. **Block Public Access**: Uncheck `Block all public access`. Acknowledge the warning. The bucket policy below restricts what is actually public.
7. Leave all other settings as defaults and create the bucket.

---

## Step 2 — Bucket Policy (Public Read on Media Objects)

Apply the following bucket policy so that uploaded media URLs work in a browser without authentication:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadMedia",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": [
        "arn:aws:s3:::g-nice-media/posts/*",
        "arn:aws:s3:::g-nice-media/avatars/*"
      ]
    }
  ]
}
```

> **How to apply:** S3 Console → `g-nice-media` → Permissions tab → Bucket policy → Edit → paste the JSON above → Save.

---

## Step 3 — CORS Configuration

The frontend needs to load media directly. Apply this CORS rule to the bucket:

```json
[
  {
    "AllowedHeaders": ["*"],
    "AllowedMethods": ["GET", "HEAD"],
    "AllowedOrigins": ["*"],
    "ExposeHeaders": [],
    "MaxAgeSeconds": 3600
  }
]
```

> **How to apply:** S3 Console → `g-nice-media` → Permissions tab → Cross-origin resource sharing (CORS) → Edit → paste the JSON above → Save.

---

## Step 4 — Create an IAM User (Least-Privilege)

The API server authenticates with a **dedicated IAM user** that has _only_ the S3 permissions it needs — no more.

1. Open the [IAM Console](https://console.aws.amazon.com/iam/).
2. Navigate to **Users → Create user**.
3. **User name**: `g-nice-api-s3`
4. **AWS access type**: Programmatic access only (no Console login needed).
5. On the permissions step, choose **Attach policies directly** → **Create inline policy**.
6. Paste the following policy JSON:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowS3MediaOperations",
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject"
      ],
      "Resource": "arn:aws:s3:::g-nice-media/*"
    }
  ]
}
```

7. Name the policy `GNiceApiS3Policy` and create it.
8. Complete the user creation.
9. On the final screen, **download the `.csv` file** or copy the credentials immediately — the secret key is shown only once.

---

## Step 5 — Environment Variables

Populate the following variables in the application's environment (`.env` for local dev, or the secret manager / ECS task definition / EC2 parameter store for production):

| Variable | Where to find it |
|---|---|
| `AWS_ACCESS_KEY_ID` | IAM Console → Users → `g-nice-api-s3` → Security credentials → Access keys |
| `AWS_SECRET_ACCESS_KEY` | Shown once at access key creation — store in a secrets manager |
| `AWS_REGION` | `af-south-1` (fixed) |
| `AWS_S3_BUCKET` | `g-nice-media` (fixed) |

> [!CAUTION]
> Never commit real credentials to source control. Use AWS Secrets Manager, SSM Parameter Store, or your CI/CD platform's secret store for production deployments.

---

## Optional — CloudFront CDN (Recommended for Production)

For lower latency globally and to avoid S3 request costs at scale, put a **CloudFront distribution** in front of the bucket:

1. Create a CloudFront distribution with `g-nice-media.s3.af-south-1.amazonaws.com` as the origin.
2. Set the **Default Cache Behavior** to allow `GET` and `HEAD`.
3. Note the CloudFront domain (e.g. `d1abc.cloudfront.net`).
4. Update the API's URL construction in `internal/storage/s3.go` to use the CloudFront domain instead of the raw S3 URL.

This is optional at launch and can be added without any code changes to the upload flow.
