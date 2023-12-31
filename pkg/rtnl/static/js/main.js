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

// Handle clicks to the logout button in the header
document.getElementById("logout").addEventListener("click", function(e) {
  e.preventDefault()
  window.localStorage.removeItem(API_STORAGE_KEY)
  window.location.href = "/login"
  return false;
});

// Checks if there is an API key, and if not, redirects to the login page.
(function() {
  let apikey = APIKey();
  if (!apikey) {
    window.location.href = "/login"
  } else {
    console.info("link shortening application logged in and ready")
  }
})();

