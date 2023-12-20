// TODO: remove temporary API key and fetch from local storage.
const API_STORAGE_KEY = "rtnlapistoragekey";

function APIKey() {
  return window.localStorage.getItem(API_STORAGE_KEY);
}

// Adds additional headers to the requests made by htmx
document.body.addEventListener('htmx:configRequest', function(e) {
  e.detail.headers['Accept'] = "text/html";

  let apikey = APIKey()
  if (apikey) {
    e.detail.headers['Authorization'] = 'Bearer ' + APIKey();
  }
});