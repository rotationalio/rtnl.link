const API_STORAGE_KEY = "rtnlapistoragekey";

function APIKey() {
  return window.localStorage.getItem(API_STORAGE_KEY);
}

// Checks if there is an API key, and if not, redirects to the login page.
(function() {
  let apikey = APIKey();
  if (!apikey) {
    window.location.href = "/login"
  } else {
    console.info("link shortening application logged in and ready")
  }
})();

// Adds additional headers to the requests made by htmx
document.body.addEventListener('htmx:configRequest', function(e) {
  e.detail.headers['Accept'] = "text/html";

  let apikey = APIKey()
  if (apikey) {
    e.detail.headers['Authorization'] = 'Bearer ' + APIKey();
  }
});