let videos = $state([]);
let totalCount = $state(0);
let loading = $state(false);

export function getVideos() { return videos; }
export function getTotalCount() { return totalCount; }
export function getLoading() { return loading; }

export function setVideos(list, total) {
  videos = list;
  totalCount = total;
}

export function setLoading(val) {
  loading = val;
}
