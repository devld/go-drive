defineUploader({
  chunkSize: 5 * 1024 * 1024,

  async start(ctx) {
    if (ctx.chunks === 1) return null;

    const res = await ctx.request({
      method: "post",
      url:
        ctx.config.baseURL +
        "/buckets/" +
        ctx.config.bucket +
        "/objects/" +
        ctx.config.encodedKey +
        "/uploads",
      headers: { Authorization: "UpToken " + ctx.config.token },
    });
    return { uploadId: res.data.uploadId };
  },

  async upload(ctx, { blob, seq, session, onProgress }) {
    if (!session) {
      const form = new FormData();
      form.append("key", ctx.config.key);
      form.append("token", ctx.config.token);
      form.append("file", blob, `${Math.random()}`);
      return ctx.request({
        method: "post",
        url: ctx.config.baseURL,
        data: form,
        onUploadProgress: onProgress,
      });
    }

    return ctx.request({
      method: "put",
      url:
        ctx.config.baseURL +
        "/buckets/" +
        ctx.config.bucket +
        "/objects/" +
        ctx.config.encodedKey +
        "/uploads/" +
        session.uploadId +
        "/" +
        (seq + 1),
      headers: {
        Authorization: "UpToken " + ctx.config.token,
        "Content-Type": "application/octet-stream",
      },
      data: blob,
      onUploadProgress: onProgress,
    });
  },

  async complete(ctx, { session, parts }) {
    if (!session) return;
    await ctx.request({
      method: "post",
      url:
        ctx.config.baseURL +
        "/buckets/" +
        ctx.config.bucket +
        "/objects/" +
        ctx.config.encodedKey +
        "/uploads/" +
        session.uploadId,
      headers: { Authorization: "UpToken " + ctx.config.token },
      data: {
        parts: parts.map((e, i) => ({
          partNumber: i + 1,
          etag: e.data.etag,
        })),
      },
    });
  },

  async abort(ctx, { session }) {
    if (!session) return;
    await ctx.request({
      method: "delete",
      url:
        ctx.config.baseURL +
        "/buckets/" +
        ctx.config.bucket +
        "/objects/" +
        ctx.config.encodedKey +
        "/uploads/" +
        session.uploadId,
      headers: { Authorization: "UpToken " + ctx.config.token },
    }).catch(() => undefined);
  },
});
