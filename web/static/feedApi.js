// Generic API request helper and specific endpoints
async function apiRequest(path, options = {}, responseType = 'json') {
  const defaultHeaders = {"X-Requested-With": "XMLHttpRequest"};
  options.headers = Object.assign(defaultHeaders, options.headers || {});
  const response = await fetch(path, options);
  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }
  if (responseType === 'json') {
    return response.json();
  } else {
    return response.text();
  }
}

export function fetchFeeds() {
  return apiRequest('/feeds', { method: 'GET' }, 'text');
}

export function fetchChangelog() {
  return apiRequest('/changelog', { method: 'GET' }, 'text');
}

export function addFeed(params) {
  return apiRequest('/add', {
    method: 'POST',
    headers: {"Content-Type": "application/x-www-form-urlencoded"},
    body: new URLSearchParams(params).toString()
  }, 'json');
}

export function removeFeedAPI(feedKey) {
  return apiRequest('/remove', {
    method: 'POST',
    headers: {"Content-Type": "application/x-www-form-urlencoded"},
    body: new URLSearchParams({ feedKey }).toString()
  }, 'json');
}

export function modifyFeed(params) {
  return apiRequest('/modify', {
    method: 'POST',
    headers: {"Content-Type": "application/x-www-form-urlencoded"},
    body: new URLSearchParams(params).toString()
  }, 'json');
}

export function reloadContainer() {
  return apiRequest('/reload', { method: 'POST' }, 'json');
}
