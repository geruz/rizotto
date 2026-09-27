# Files

    rizotto solution add files -download redirect

Installs file storage on any S3 compatible object storage: the file service owning
the metadata, the presigner, and the routes handing out upload and download links.

It **requires** the `user` solution: a file belongs to somebody, and every route
checks that the caller owns what they ask for.

## The bytes never pass through the service

A caller asking to upload gets a presigned link and PUTs the file to the storage
itself. That is not only cheaper — the gateway parses JSON and form bodies but not
multipart, and has no streaming answer, so proxying files would mean new framework
machinery for no gain.

    1. POST /api/v1/files                    announce name, type and size
       -> { file: {...status: "pending"}, uploadUrl, expiresAt }
    2. PUT <uploadUrl>                       the caller sends the bytes to the storage
    3. POST /api/v1/files/{id}/confirm       we HEAD the object and record its real size
       -> { ...status: "ready" }

A file stays `pending` until step 3 finds the bytes, which is what tells a finished
upload from an abandoned one. Nothing else accepts a pending file: asking for its
content answers an error rather than a broken link.

## The routes

    POST   /api/v1/files                  announce a file, get an upload link
    POST   /api/v1/files/{file_id}/confirm  check the bytes arrived
    GET    /api/v1/files                  the files of the caller, newest first
    GET    /api/v1/files/{file_id}        the metadata of one
    GET    /api/v1/files/{file_id}/content  where to read the bytes
    DELETE /api/v1/files/{file_id}        the row and the object

Every one of them is behind `api.AuthArea`, so an unauthenticated caller is
answered 401 before the handler runs.

## The table

    files   id, user_id, object_key UNIQUE, name, content_type, size_bytes, status, ...

`object_key` is generated here — the owner's id, a random name from
`crypto/token.Secure`, and the extension of what was uploaded, reduced to
`[a-z0-9]`. The name the caller sent is stored as data and **never** used as a
path, so it cannot reach out of its prefix, collide with somebody else's file, or
carry anything surprising into the key.

`user_id` carries no foreign key: the users belong to the user service, which owns
its own tables and its own migration history.

## Ownership

Every query is scoped by `user_id`, and a file of somebody else answers **not
found** rather than forbidden — the answer must not confirm that it exists. The
check is in the service, not only in the controller, so a second caller of the same
RPC cannot skip it.

## The size limit

`S3_MAX_UPLOAD_MB` is checked twice, because a presigned link says where the bytes
may go but not how many. The announced size is refused up front; the real size is
read on confirm, and a file that came in larger is deleted from the storage and
refused. Both answer the public code `file_too_large`.

## `-download redirect` or `-download url`

`redirect` answers 302 to the presigned link. Right for a browser: an `<img src>`
or a plain link works with no code at all.

`url` answers `{ url, expiresAt }`. Right for a client that wants to know when the
link dies, or to fetch the bytes itself.

## Configuration

    S3_ENDPOINT             the storage without the bucket
    S3_REGION
    S3_BUCKET
    S3_ACCESS_KEY_ID
    S3_SECRET_ACCESS_KEY
    S3_PATH_STYLE           "true" puts the bucket in the path, which MinIO needs
    S3_LINK_TTL_MINUTES     how long a link lives (default 15)
    S3_MAX_UPLOAD_MB        the largest file allowed (default 25)

The bucket needs a CORS rule allowing PUT and GET from the origin of your
application: the upload link points at the storage, not at this service, so the
browser talks to another origin.

## The presigner

`services/file/storage` signs links with AWS Signature Version 4, in about 250
lines of `crypto/hmac` and no dependency at all. Only **query** signing is
implemented, because a presigned link is all this service ever needs — HEAD and
DELETE are plain requests against a presigned link too, which is why there is one
primitive rather than three.

Its tests pin that the signature covers the method and the key, that it is
reproducible, and that both addressing styles land where they should. The signing
itself was verified against MinIO: PUT, HEAD, GET and DELETE are accepted, a
tampered signature and a download link used to upload are refused with
`SignatureDoesNotMatch`.

## What it deliberately leaves out

No multipart upload, so a file has to fit in one PUT — which is what
`S3_MAX_UPLOAD_MB` is for; anything bigger wants the S3 multipart API and a
different flow. No image processing or thumbnails. No public objects: every read
goes through a link that expires, and making a bucket public is a decision for the
project. Abandoned pending rows are never collected, so a periodic sweep of old
`pending` files is yours to add.
