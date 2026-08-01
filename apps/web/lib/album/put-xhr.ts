export type PutBinaryArgs = {
  url: string;
  method: string;
  headers?: Record<string, string>;
  body: Blob;
  signal?: AbortSignal;
  onProgress?: (loaded: number, total: number) => void;
};

export function putBinaryXhr(args: PutBinaryArgs): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open(args.method, args.url);
    if (args.headers) {
      for (const [k, v] of Object.entries(args.headers)) {
        try {
          xhr.setRequestHeader(k, v);
        } catch {
          // Some presigners return forbidden headers (host, content-length)
          // that the browser sets itself — skip rather than fail the upload.
        }
      }
    }
    if (args.onProgress) {
      xhr.upload.onprogress = (ev) => {
        if (ev.lengthComputable) args.onProgress!(ev.loaded, ev.total);
      };
    }
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
      } else {
        reject(new Error(`Upload failed (${xhr.status})`));
      }
    };
    xhr.onerror = () => reject(new Error("Network error during upload"));
    xhr.onabort = () =>
      reject(new DOMException("Upload aborted", "AbortError"));

    if (args.signal) {
      if (args.signal.aborted) {
        xhr.abort();
        return;
      }
      args.signal.addEventListener("abort", () => xhr.abort(), { once: true });
    }
    xhr.send(args.body);
  });
}
