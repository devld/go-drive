## Implementing Drive's uploader

In general, drives implemented in JavaScript can be uploaded via `go-drive`'s local uploader, in which case the upload will be relayed through `go-drive`.

If you want to upload directly through the front-end. Then you need to implement a custom uploader.

This script, unlike the Drive script, will be executed in the browser environment and you can use any API supported by the browser.

In this script, you need to define a method, the name of the method doesn't matter.
All variables, code need to be declared in this method (closure).

This method will eventually need to return a [`CustomUploader`](https://github.com/devld/go-drive/blob/d5c3246b68355a76c358c8ea25139b0612f7b7fb/docs/drive-uploaders/types.d.ts#L51-L62).

Once you finish all, then you can copy this js file to `/script-drives`，and name it `<DRIVE_NAME>-uploader.js`.

You can see examples of existing implementations in [`script-drives`](https://github.com/devld/go-drive/tree/master/script-drives).


