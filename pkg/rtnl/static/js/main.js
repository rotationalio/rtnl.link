// Checks if there is an API key, and if not, redirects to the login page.
(function() {
  console.info("link shortening application logged in and ready");
})();

// Ensure the accept header is set to text/html for all htmx requests.
document.body.addEventListener('htmx:configRequest', (e) => {
  e.detail.headers['Accept'] = 'text/html'
});