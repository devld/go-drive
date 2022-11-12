/// <reference path="./types.d.ts"/>

/**
 * @param {UploadFactoryContext} ctx
 * @returns {CustomUploader}
 */
function qiniuUploader(ctx) {
  const chunkSize = 5 * 1024 * 1024; // 5MB
  const chunks = Math.ceil(ctx.task.file.size / chunkSize);

  let res;
  let uploadId;
  let parts;

  const multipartURL =
    ctx.config.baseURL +
    "/buckets/" +
    ctx.config.bucket +
    "/objects/" +
    ctx.config.encodedKey +
    "/uploads";
  const commonHeaders = { Authorization: "UpToken " + ctx.config.token };

  return {
    async prepare() {
      if (chunks === 1) return chunks;

      const res = await ctx.request({
        method: "post",
        url: multipartURL,
        headers: commonHeaders,
      });
      uploadId = res.data.uploadId;
      parts = Array(chunks).fill("");

      return chunks;
    },
    async upload(data, seq, onProgress) {
      if (chunks === 1) {
        const form = new FormData();
        form.append("key", ctx.config.key);
        form.append("token", ctx.config.token);
        form.append("file", data, `${Math.random()}`);

        res = await ctx.request({
          method: "post",
          url: ctx.config.baseURL,
          data: form,
          onUploadProgress: onProgress,
        });
        return res;
      }

      const res = await ctx.request({
        method: "put",
        url: multipartURL + "/" + uploadId + "/" + (seq + 1),
        headers: {
          ...commonHeaders,
          "Content-Type": "application/octet-stream",
        },
        data: data,
        onUploadProgress: onProgress,
      });

      parts[seq] = res.data;
      return res;
    },
    async complete() {
      if (chunks > 1) {
        res = (
          await ctx.request({
            method: "post",
            url: multipartURL + "/" + uploadId,
            headers: commonHeaders,
            data: {
              parts: parts.map((e, i) => ({ partNumber: i + 1, etag: e.etag })),
            },
          })
        ).data;
      }

      await ctx.uploadCallback({ action: "Completed" });
      return res;
    },
    onCleanup() {
      if (res) return;
      ctx
        .request({
          method: "delete",
          url: multipartURL + "/" + uploadId,
          headers: commonHeaders,
        })
        .catch(() => {
          // ignore
        });
    },
    getChunk(seq) {
      return ctx.task.file.slice(seq * chunkSize, (seq + 1) * chunkSize);
    },
  };
}
